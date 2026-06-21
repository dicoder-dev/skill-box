package maps

// =============================================================================
// 华为地图服务提供商实现
// 提供对华为地图API的封装，实现Provider接口
// 支持：地点搜索、驾车路线规划、地理编码、逆地理编码、行政区划查询
// API文档：https://developer.huawei.com/consumer/cn/doc/development/ Maps-Guides/dev-guide-3
// =============================================================================

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// 确保MapHuawei实现了Provider接口
var _ Provider = (*MapHuawei)(nil)

// =============================================================================
// MapHuawei 结构体定义
// =============================================================================

// MapHuawei 华为地图服务提供商
// 华为地图使用两个不同的API域：
//   - siteapi.cloud.huawei.com: 地点搜索、地理编码、逆地理编码
//   - mapapi.cloud.huawei.com: 路线规划
type MapHuawei struct {
	baseProvider               // 基础Provider（用于地点搜索API）
	routeProvider baseProvider // 路线规划Provider
}

// NewMapHuawei 创建华为地图提供商实例
// 参数说明：
//   - cfg: 华为地图配置，包含API密钥等
//   - timeout: HTTP请求超时时间
//
// 返回值：华为地图Provider实例
func NewMapHuawei(cfg ProviderConfig, timeout time.Duration) *MapHuawei {
	// 创建地点搜索API的Provider
	siteProvider := newBaseProvider(MapNameHuawei, cfg, timeout, "https://siteapi.cloud.huawei.com")

	// 创建路线规划API的Provider
	routeCfg := cfg
	if strings.TrimSpace(routeCfg.BaseURL) == "" {
		routeCfg.BaseURL = "https://mapapi.cloud.huawei.com"
	} else {
		// 替换域名
		routeCfg.BaseURL = strings.Replace(routeCfg.BaseURL, "siteapi.cloud.huawei.com", "mapapi.cloud.huawei.com", 1)
	}

	return &MapHuawei{
		baseProvider:  siteProvider,
		routeProvider: newBaseProvider(MapNameHuawei, routeCfg, timeout, "https://mapapi.cloud.huawei.com"),
	}
}

// =============================================================================
// 地点搜索实现
// =============================================================================

// PlaceSearch 实现Provider接口的地点搜索方法
// 使用华为地图Site Service的文本搜索API
// 请求参数：
//   - Query: 搜索关键字
//   - City: 搜索城市（拼接到关键字中提高精度）
//   - Location/Radius: 周边搜索参数
//   - Types: POI类型过滤
//   - Options.CountryCode: 国家代码
//   - PageSize/PageIndex: 分页参数
func (m *MapHuawei) PlaceSearch(ctx context.Context, req *PlaceSearchRequest) (*PlaceSearchResponse, error) {
	// 预处理查询关键字
	query := req.Query
	if req.City != "" {
		// 华为地图将城市拼接到关键字前来提高搜索精度
		query = strings.TrimSpace(req.City + " " + req.Query)
	}

	// 构建请求载荷
	payload := map[string]interface{}{
		"query":     query,
		"pageSize":  defaultHuaweiPageSize(req.PageSize),
		"pageIndex": defaultPageIndex(req.PageIndex),
	}
	if req.Location != "" {
		// 周边搜索模式
		location, err := huaweiLocation(req.Location)
		if err != nil {
			return nil, err
		}
		payload["location"] = location
		payload["radius"] = defaultRadius(req.Radius)
	}
	if req.Types != "" {
		payload["hwPoiType"] = strings.ToUpper(strings.TrimSpace(req.Types))
	}
	if req.Options.CountryCode != "" {
		payload["countryCode"] = strings.ToUpper(strings.TrimSpace(req.Options.CountryCode))
	}
	if m.config.Language != "" {
		payload["language"] = m.config.Language
	}

	// 发送请求
	var resp huaweiSiteResponse
	if err := m.postSiteJSON(ctx, "/mapApi/v1/siteService/searchByText", payload, &resp); err != nil {
		return nil, err
	}
	// 检查响应状态
	if !resp.ok() {
		return nil, fmt.Errorf("华为地图地点搜索失败: %s", resp.message())
	}

	// 转换结果
	out := &PlaceSearchResponse{
		Status:      "ok",
		Message:     "success",
		Provider:    string(m.Name()),
		Total:       resp.TotalCount,
		RawResponse: resp,
	}
	// 如果TotalCount为0，则使用实际返回结果数量
	sites := resp.siteItems()
	if out.Total == 0 {
		out.Total = len(sites)
	}
	for _, item := range sites {
		out.Places = append(out.Places, Place{
			ID:            item.SiteID,
			Name:          item.Name,
			Address:       item.FormatAddress,
			Location:      item.locationString(),
			Distance:      item.Distance,
			Type:          strings.Join(firstNonEmptySlice(item.POI.HwPoiTypes, item.POI.PoiTypes), ","),
			Phone:         firstNotEmpty(item.POI.Phone, item.POI.InternationalPhone),
			Rating:        parseFloat(fmt.Sprint(item.POI.Rating)),
			BusinessHours: strings.Join(item.POI.OpeningHours, ";"),
			Province:      item.Address.State,
			City:          item.Address.City,
			District:      item.Address.District,
		})
	}
	return out, nil
}

// HuaweiPlaceQueryRequest 华为地点搜索/补齐专用请求。
// Query 为关键字；Location 格式为 "经度,纬度"，例如北京中心点 "116.4074,39.9042"。
type HuaweiPlaceQueryRequest struct {
	Query       string   `json:"query"`
	Location    string   `json:"location"`
	Radius      int      `json:"radius"`
	CountryCode string   `json:"countryCode"`
	Language    string   `json:"language"`
	PageIndex   int      `json:"pageIndex"`
	PageSize    int      `json:"pageSize"`
	HwPoiType   string   `json:"hwPoiType"`
	PoiTypes    []string `json:"poiTypes"`
}

// HuaweiProvider 华为地图专属能力接口。
// 不放入通用 Provider，避免要求其他地图厂商实现华为特有能力。
type HuaweiProvider interface {
	TextSearchRaw(ctx context.Context, req *HuaweiPlaceQueryRequest) (*HuaweiSiteResponse, error)
	PlaceAutocompleteRaw(ctx context.Context, req *HuaweiPlaceQueryRequest) (*HuaweiSiteResponse, error)
}

// TextSearchRaw 调用华为关键字地点搜索，返回华为原始结构。
func (m *MapHuawei) TextSearchRaw(ctx context.Context, req *HuaweiPlaceQueryRequest) (*HuaweiSiteResponse, error) {
	resp, err := m.huaweiSiteQuery(ctx, "/mapApi/v1/siteService/searchByText", req)
	if err != nil {
		return nil, err
	}
	if !resp.ok() {
		return nil, fmt.Errorf("华为地图地点搜索失败: %s", resp.message())
	}
	return resp, nil
}

// PlaceAutocompleteRaw 调用华为地点补齐/建议接口，返回华为原始结构。
func (m *MapHuawei) PlaceAutocompleteRaw(ctx context.Context, req *HuaweiPlaceQueryRequest) (*HuaweiSiteResponse, error) {
	resp, err := m.huaweiSiteQuery(ctx, "/mapApi/v1/siteService/querySuggestion", req)
	if err != nil {
		return nil, err
	}
	if !resp.ok() {
		return nil, fmt.Errorf("华为地图地点补齐失败: %s", resp.message())
	}
	return resp, nil
}

func (m *MapHuawei) huaweiSiteQuery(ctx context.Context, path string, req *HuaweiPlaceQueryRequest) (*HuaweiSiteResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("华为地点查询参数不能为空")
	}
	// 华为 Site API 多数参数是可选的，这里只把非空参数放入 payload。
	// 这样同一个方法既能服务“地点补齐”，也能服务“关键字搜索”，避免为两个接口维护两套请求构造逻辑。
	payload := map[string]interface{}{
		"query": strings.TrimSpace(req.Query),
	}
	if strings.TrimSpace(req.Location) != "" {
		location, err := huaweiLocation(req.Location)
		if err != nil {
			return nil, err
		}
		payload["location"] = location
		payload["radius"] = defaultRadius(req.Radius)
	}
	if req.CountryCode != "" {
		payload["countryCode"] = strings.ToUpper(strings.TrimSpace(req.CountryCode))
	}
	if req.Language != "" {
		payload["language"] = strings.TrimSpace(req.Language)
	} else if m.config.Language != "" {
		payload["language"] = m.config.Language
	}
	if req.PageIndex > 0 {
		payload["pageIndex"] = defaultPageIndex(req.PageIndex)
	}
	if req.PageSize > 0 {
		payload["pageSize"] = defaultHuaweiPageSize(req.PageSize)
	}
	if req.HwPoiType != "" {
		payload["hwPoiType"] = strings.ToUpper(strings.TrimSpace(req.HwPoiType))
	}
	if len(req.PoiTypes) > 0 {
		payload["poiTypes"] = req.PoiTypes
	}

	var resp huaweiSiteResponse
	if err := m.postSiteJSON(ctx, path, payload, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// =============================================================================
// 线路规划实现
// =============================================================================

// RoutePlanning 实现Provider接口的线路规划方法
// 使用华为地图Route Service的驾车路线规划API
// 注意：华为地图仅支持驾车路线规划
// 请求参数：
//   - Origin: 起点坐标
//   - Destination: 终点坐标
//   - Strategy: 路线策略
//   - Waypoints: 途经点
func (m *MapHuawei) RoutePlanning(ctx context.Context, req *RoutePlanningRequest) (*RoutePlanningResponse, error) {
	// 检查出行模式（华为仅支持驾车）
	if mustMode(req) != TravelModeDriving {
		return nil, newUnsupportedResponse(m.Name(), string(mustMode(req)))
	}

	// 解析起点坐标
	origin, err := huaweiLocation(req.Origin)
	if err != nil {
		return nil, err
	}
	// 解析终点坐标
	destination, err := huaweiLocation(req.Destination)
	if err != nil {
		return nil, err
	}

	// 构建请求载荷
	payload := map[string]interface{}{
		"origin":      origin,
		"destination": destination,
	}
	if req.Strategy != "" {
		payload["strategy"] = huaweiDrivingStrategy(req.Strategy)
	}
	if req.Waypoints != "" {
		waypoints, err := huaweiWaypoints(req.Waypoints)
		if err != nil {
			return nil, err
		}
		payload["waypoints"] = waypoints
	}

	// 发送请求
	var resp huaweiRouteResponse
	if err := m.postRouteJSON(ctx, "/mapApi/v1/routeService/driving", payload, &resp); err != nil {
		return nil, err
	}
	if !resp.ok() {
		return nil, fmt.Errorf("华为地图驾车路线规划失败: %s", resp.message())
	}

	// 转换结果
	out := &RoutePlanningResponse{
		Status:      "ok",
		Message:     "success",
		Provider:    string(m.Name()),
		RawResponse: resp,
	}
	for _, item := range resp.Routes {
		route := Route{
			Distance: item.PathsDistance(),
			Duration: item.PathsDuration(),
			Polyline: huaweiValueString(item.Polyline),
			Summary:  item.Summary,
		}
		// 兼容旧版字段
		if route.Distance == 0 {
			route.Distance = item.Distance
		}
		if route.Duration == 0 {
			route.Duration = item.Duration
		}
		// 解析步骤
		for _, path := range item.Paths {
			for _, step := range path.Steps {
				route.Steps = append(route.Steps, RouteStep{
					Instruction: step.Instruction,
					Distance:    step.Distance,
					Duration:    step.Duration,
					Polyline:    huaweiValueString(step.Polyline),
				})
			}
		}
		if out.Distance == 0 {
			out.Distance = route.Distance
			out.Duration = route.Duration
		}
		out.Routes = append(out.Routes, route)
	}
	return out, nil
}

// huaweiDrivingStrategy 将策略名称转换为华为API所需的代码
//   - "shortest" -> 1: 最短路线
//   - "avoid_highways" -> 2: 避开高速
//   - 其他 -> 0: 推荐路线（默认）
func huaweiDrivingStrategy(strategy string) int {
	switch strings.TrimSpace(strategy) {
	case "shortest":
		return 1
	case "avoid_highways":
		return 2
	default:
		return 0
	}
}

// =============================================================================
// 行政区划查询实现
// =============================================================================

// AdministrativeRegionQuery 实现Provider接口的行政区划查询方法
// 通过逆地理编码获取坐标所在行政区划信息
func (m *MapHuawei) AdministrativeRegionQuery(ctx context.Context, req *AdministrativeRegionRequest) (*AdministrativeRegionResponse, error) {
	if req.Location == "" {
		return nil, fmt.Errorf("location 不能为空")
	}
	// 调用逆地理编码获取行政区划
	regeo, err := m.ReverseGeocoding(ctx, &ReverseGeocodingRequest{
		Location: req.Location,
		Options:  req.Options,
	})
	if err != nil {
		return nil, err
	}
	return &AdministrativeRegionResponse{
		Status:      "ok",
		Message:     "success",
		Provider:    string(m.Name()),
		RawResponse: regeo.RawResponse,
		Regions: []AdministrativeRegion{
			{
				Name:     firstNotEmpty(regeo.Result.District, regeo.Result.City, regeo.Result.Province, regeo.Result.Country),
				Level:    req.Level,
				Location: req.Location,
			},
		},
	}, nil
}

// =============================================================================
// 地理编码实现
// =============================================================================

// Geocoding 实现Provider接口的地理编码方法（地址转坐标）
// 使用华为地图Site Service的Geocode API
func (m *MapHuawei) Geocoding(ctx context.Context, req *GeocodingRequest) (*GeocodingResponse, error) {
	// 构建请求载荷
	payload := map[string]interface{}{
		"address": strings.TrimSpace(req.Address),
	}
	if req.City != "" {
		payload["city"] = req.City
	}
	if m.config.Language != "" {
		payload["language"] = m.config.Language
	}

	// 发送请求
	var resp huaweiSiteResponse
	if err := m.postSiteJSON(ctx, "/mapApi/v1/siteService/geocode", payload, &resp); err != nil {
		return nil, err
	}
	if !resp.ok() {
		return nil, fmt.Errorf("华为地图地理编码失败: %s", resp.message())
	}

	// 转换结果
	out := &GeocodingResponse{
		Status:      "ok",
		Message:     "success",
		Provider:    string(m.Name()),
		RawResponse: resp,
	}
	for _, item := range resp.siteItems() {
		out.Locations = append(out.Locations, GeocodingResult{
			Latitude:  item.Location.Lat,
			Longitude: item.Location.Lng,
			Precise:   true,
			Province:  item.Address.State,
			City:      item.Address.City,
			District:  item.Address.District,
			Address:   firstNotEmpty(item.FormatAddress, item.Name, req.Address),
		})
	}
	return out, nil
}

// =============================================================================
// 逆地理编码实现
// =============================================================================

// ReverseGeocoding 实现Provider接口的逆地理编码方法（坐标转地址）
// 使用华为地图Site Service的Reverse Geocode API
func (m *MapHuawei) ReverseGeocoding(ctx context.Context, req *ReverseGeocodingRequest) (*ReverseGeocodingResponse, error) {
	// 解析坐标
	location, err := huaweiLocation(req.Location)
	if err != nil {
		return nil, err
	}

	// 构建请求载荷
	payload := map[string]interface{}{
		"location": location,
	}
	if req.Radius > 0 {
		payload["radius"] = req.Radius
	}
	if m.config.Language != "" {
		payload["language"] = m.config.Language
	}

	// 发送请求
	var resp huaweiSiteResponse
	if err := m.postSiteJSON(ctx, "/mapApi/v1/siteService/reverseGeocode", payload, &resp); err != nil {
		return nil, err
	}
	if !resp.ok() {
		return nil, fmt.Errorf("华为地图逆地理编码失败: %s", resp.message())
	}

	// 解析结果
	result := ReverseGeocodingResult{
		Latitude:  location["lat"],
		Longitude: location["lng"],
	}
	if sites := resp.siteItems(); len(sites) > 0 {
		site := sites[0]
		result.FormattedAddress = site.FormatAddress
		result.Country = site.Address.Country
		result.Province = site.Address.State
		result.City = site.Address.City
		result.District = site.Address.District
		result.Street = site.Address.Street
		result.StreetNumber = site.Address.StreetNumber
		// 如果有精确坐标则使用返回的坐标
		if site.Location.Lat != 0 || site.Location.Lng != 0 {
			result.Latitude = site.Location.Lat
			result.Longitude = site.Location.Lng
		}
	}

	return &ReverseGeocodingResponse{
		Status:      "ok",
		Message:     "success",
		Provider:    string(m.Name()),
		Result:      result,
		RawResponse: resp,
	}, nil
}

// =============================================================================
// HTTP请求方法
// =============================================================================

// postSiteJSON 发送POST请求到Site API
// 华为地图地点搜索、地理编码、逆地理编码使用siteapi.cloud.huawei.com
func (m *MapHuawei) postSiteJSON(ctx context.Context, path string, payload interface{}, out interface{}) error {
	params := url.Values{}
	params.Set("key", m.config.Key)
	return m.postJSON(ctx, path, params, payload, out)
}

// postRouteJSON 发送POST请求到Route API
// 华为地图路线规划使用mapapi.cloud.huawei.com
func (m *MapHuawei) postRouteJSON(ctx context.Context, path string, payload interface{}, out interface{}) error {
	params := url.Values{}
	params.Set("key", m.config.Key)
	return m.routeProvider.postJSON(ctx, path, params, payload, out)
}

// =============================================================================
// 数据结构定义
// =============================================================================

// huaweiSiteResponse Site API通用响应结构
type huaweiSiteResponse struct {
	ReturnCode  string       `json:"returnCode"`  // 返回码
	ReturnDesc  string       `json:"returnDesc"`  // 返回描述
	TotalCount  int          `json:"totalCount"`  // 总数量
	Sites       []huaweiSite `json:"sites"`       // 地点列表
	Predictions []huaweiSite `json:"predictions"` // 兼容补齐/建议类返回
	Suggestions []huaweiSite `json:"suggestions"` // 兼容补齐/建议类返回
}

type HuaweiSiteResponse = huaweiSiteResponse

// ok 检查响应是否成功
func (r huaweiSiteResponse) ok() bool {
	return r.ReturnCode == "" || r.ReturnCode == "0"
}

// message 获取响应消息
func (r huaweiSiteResponse) message() string {
	return firstNotEmpty(r.ReturnDesc, r.ReturnCode)
}

func (r huaweiSiteResponse) siteItems() []huaweiSite {
	// 华为不同地点接口返回的列表字段不完全一致：
	// 搜索接口常见为 sites，补齐/建议接口可能是 predictions 或 suggestions。
	// 这里统一归一化，避免上层业务关心具体接口差异。
	switch {
	case len(r.Sites) > 0:
		return r.Sites
	case len(r.Predictions) > 0:
		return r.Predictions
	case len(r.Suggestions) > 0:
		return r.Suggestions
	default:
		return nil
	}
}

// SiteItems 返回华为地点列表，兼容搜索、补齐、建议类接口的不同字段名。
func (r huaweiSiteResponse) SiteItems() []HuaweiSite {
	items := r.siteItems()
	out := make([]HuaweiSite, 0, len(items))
	for _, item := range items {
		out = append(out, item)
	}
	return out
}

// huaweiSite 地点信息结构
type huaweiSite struct {
	SiteID        string        `json:"siteId"`        // 地点ID
	Name          string        `json:"name"`          // 地点名称
	FormatAddress string        `json:"formatAddress"` // 格式化地址
	Location      huaweiPoint   `json:"location"`      // 坐标
	Address       huaweiAddress `json:"address"`       // 地址组件
	POI           huaweiPOI     `json:"poi"`           // POI信息
	Distance      float64       `json:"distance"`      // 距离
}

type HuaweiSite = huaweiSite

// locationString 将坐标转换为字符串格式
func (s huaweiSite) locationString() string {
	if s.Location.Lng == 0 && s.Location.Lat == 0 {
		return ""
	}
	return fmt.Sprintf("%f,%f", s.Location.Lng, s.Location.Lat)
}

// huaweiPoint 坐标结构
type huaweiPoint struct {
	Lat float64 `json:"lat"` // 纬度
	Lng float64 `json:"lng"` // 经度
}

type HuaweiPoint = huaweiPoint

// huaweiAddress 地址组件结构
type huaweiAddress struct {
	Country      string `json:"country"`      // 国家
	State        string `json:"state"`        // 省/州
	City         string `json:"city"`         // 城市
	District     string `json:"district"`     // 区县
	Street       string `json:"street"`       // 道路
	StreetNumber string `json:"streetNumber"` // 门牌号
}

type HuaweiAddress = huaweiAddress

// huaweiPOI POI信息结构
type huaweiPOI struct {
	PoiTypes           []string `json:"poiTypes"`           // POI类型列表
	HwPoiTypes         []string `json:"hwPoiTypes"`         // 华为POI类型列表
	Phone              string   `json:"phone"`              // 电话
	InternationalPhone string   `json:"internationalPhone"` // 国际电话
	Rating             any      `json:"rating"`             // 评分
	OpeningHours       []string `json:"openingHours"`       // 营业时间
}

type HuaweiPOI = huaweiPOI

// huaweiRouteResponse 路线规划响应结构
type huaweiRouteResponse struct {
	ReturnCode string        `json:"returnCode"` // 返回码
	ReturnDesc string        `json:"returnDesc"` // 返回描述
	Routes     []huaweiRoute `json:"routes"`     // 路线列表
}

// ok 检查响应是否成功
func (r huaweiRouteResponse) ok() bool {
	return r.ReturnCode == "" || r.ReturnCode == "0"
}

// message 获取响应消息
func (r huaweiRouteResponse) message() string {
	return firstNotEmpty(r.ReturnDesc, r.ReturnCode)
}

// huaweiRoute 路线信息结构
type huaweiRoute struct {
	Distance float64      `json:"distance"` // 总距离
	Duration int64        `json:"duration"` // 总耗时
	Summary  string       `json:"summary"`  // 路线摘要
	Polyline any          `json:"polyline"` // 路线编码
	Paths    []huaweiPath `json:"paths"`    // 路径列表
}

// PathsDistance 计算所有路径的总距离
func (r huaweiRoute) PathsDistance() float64 {
	var distance float64
	for _, path := range r.Paths {
		distance += path.Distance
	}
	return distance
}

// PathsDuration 计算所有路径的总耗时
func (r huaweiRoute) PathsDuration() int64 {
	var duration int64
	for _, path := range r.Paths {
		duration += path.Duration
	}
	return duration
}

// huaweiPath 路径信息结构
type huaweiPath struct {
	Distance float64      `json:"distance"` // 此段距离
	Duration int64        `json:"duration"` // 此段耗时
	Steps    []huaweiStep `json:"steps"`    // 步骤列表
}

// huaweiStep 步骤信息结构
type huaweiStep struct {
	Instruction string  `json:"instruction"` // 导航指示
	Distance    float64 `json:"distance"`    // 此步骤距离
	Duration    int64   `json:"duration"`    // 此步骤耗时
	Polyline    any     `json:"polyline"`    // 路线编码
}

// =============================================================================
// 辅助函数
// =============================================================================

// huaweiLocation 将字符串坐标转换为华为API所需的地图对象格式
// 输入格式："经度,纬度" 或 "纬度,经度"
// 输出格式：{"lng": 经度, "lat": 纬度}
func huaweiLocation(value string) (map[string]float64, error) {
	lng, lat, err := parseCoordinatePair(value)
	if err != nil {
		return nil, err
	}
	return map[string]float64{
		"lng": lng,
		"lat": lat,
	}, nil
}

// huaweiWaypoints 将字符串格式的途经点转换为华为API所需的数组格式
// 输入格式："经度1,纬度1|经度2,纬度2"
// 输出格式：[{"lng": 经度1, "lat": 纬度1}, {"lng": 经度2, "lat": 纬度2}]
func huaweiWaypoints(value string) ([]map[string]float64, error) {
	parts := strings.Split(value, "|")
	points := make([]map[string]float64, 0, len(parts))
	for _, part := range parts {
		point, err := huaweiLocation(part)
		if err != nil {
			return nil, err
		}
		points = append(points, point)
	}
	return points, nil
}

// defaultHuaweiPageSize 获取华为API默认的分页大小（最大20）
func defaultHuaweiPageSize(size int) int {
	if size <= 0 {
		return 20
	}
	if size > 20 {
		return 20
	}
	return size
}

// firstNonEmptySlice 返回第一个非空切片
func firstNonEmptySlice(values ...[]string) []string {
	for _, value := range values {
		if len(value) > 0 {
			return value
		}
	}
	return nil
}

// huaweiValueString 将任意值转换为字符串
func huaweiValueString(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	default:
		return fmt.Sprint(v)
	}
}
