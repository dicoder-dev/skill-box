package maps

// =============================================================================
// 天地图服务提供商实现
// 提供对天地图API的封装，实现Provider接口
// 支持：地点搜索、线路规划、地理编码、逆地理编码、行政区划查询
// API文档：https://wiki.tianditu.gov.cn
// =============================================================================

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// 确保MapTianditu实现了Provider接口
var _ Provider = (*MapTianditu)(nil)

// =============================================================================
// MapTianditu 结构体定义
// =============================================================================

// MapTianditu 天地图服务提供商
type MapTianditu struct {
	baseProvider // 嵌入基础Provider，提供通用功能
}

// NewMapTianditu 创建天地图提供商实例
// 参数说明：
//   - cfg: 天地图配置，包含API密钥等
//   - timeout: HTTP请求超时时间
//
// 返回值：天地图Provider实例
func NewMapTianditu(cfg ProviderConfig, timeout time.Duration) *MapTianditu {
	return &MapTianditu{
		baseProvider: newBaseProvider(MapNameTianditu, cfg, timeout, "https://api.tianditu.gov.cn"),
	}
}

// =============================================================================
// 地点搜索实现
// =============================================================================

// PlaceSearch 实现Provider接口的地点搜索方法
// 使用天地图V2 POI搜索API
// 请求参数：
//   - Query: 搜索关键字
//   - City: 搜索城市
//   - Location/Radius: 周边搜索参数
//   - PageSize/PageIndex: 分页参数
func (m *MapTianditu) PlaceSearch(ctx context.Context, req *PlaceSearchRequest) (*PlaceSearchResponse, error) {
	// 构建请求参数（天地图使用POST请求）
	post := map[string]interface{}{
		"keyWord": req.Query,
		"start":   defaultPageIndex(req.PageIndex) - 1,
		"count":   defaultPageSize(req.PageSize),
		"show":    2,
	}
	if req.Location != "" {
		// 周边搜索模式
		post["pointLonlat"] = req.Location
		post["queryRadius"] = defaultRadius(req.Radius)
		post["queryType"] = 3
		post["level"] = 12
	} else {
		// 天地图普通搜索需要提供视野范围，这里默认使用中国范围进行兼容。
		post["queryType"] = 1
		post["mapBound"] = "73.00000,3.00000,135.00000,54.00000"
		post["level"] = 5
		if req.City != "" {
			// 文档未提供直接的 city 参数，拼接到关键字中作为兼容策略。
			post["keyWord"] = strings.TrimSpace(req.City + " " + req.Query)
		}
	}

	postStr, _ := json.Marshal(post)
	params := url.Values{}
	params.Set("postStr", string(postStr))
	params.Set("type", "query")
	params.Set("tk", m.config.Key)

	// 定义响应结构
	var resp struct {
		Count      int `json:"count"`
		ResultType int `json:"resultType"`
		Pois       []struct {
			Name         string `json:"name"`
			Address      string `json:"address"`
			Lonlat       string `json:"lonlat"`
			Phone        string `json:"phone"`
			PoiType      any    `json:"poiType"`
			TypeCode     string `json:"typeCode"`
			TypeName     string `json:"typeName"`
			Province     string `json:"province"`
			ProvinceCode string `json:"provinceCode"`
			City         string `json:"city"`
			CityCode     string `json:"cityCode"`
			County       string `json:"county"`
			CountyCode   string `json:"countyCode"`
			HotPointID   string `json:"hotPointID"`
		} `json:"pois"`
		Status struct {
			InfoCode int    `json:"infocode"`
			Desc     string `json:"cndesc"`
		} `json:"status"`
	}

	// 发送请求
	if err := m.getJSON(ctx, "/v2/search", params, &resp); err != nil {
		return nil, err
	}
	if resp.Status.InfoCode != 1000 && resp.Status.InfoCode != 0 {
		return nil, fmt.Errorf("天地图地点搜索失败: %s", resp.Status.Desc)
	}

	// 转换结果
	out := &PlaceSearchResponse{
		Status:      "ok",
		Message:     "success",
		Provider:    string(m.Name()),
		Total:       resp.Count,
		RawResponse: resp,
	}
	for _, item := range resp.Pois {
		out.Places = append(out.Places, Place{
			ID:           item.HotPointID,
			Name:         item.Name,
			Address:      item.Address,
			Location:     item.Lonlat,
			PoiType:      strings.TrimSpace(anyToString(item.PoiType)),
			TypeCode:     item.TypeCode,
			Phone:        item.Phone,
			Type:         item.TypeName,
			Province:     item.Province,
			ProvinceCode: item.ProvinceCode,
			City:         item.City,
			CityCode:     item.CityCode,
			District:     item.County,
			DistrictCode: item.CountyCode,
		})
	}
	return out, nil
}

// =============================================================================
// 线路规划实现
// =============================================================================

// RoutePlanning 实现Provider接口的线路规划方法
// 使用天地图驾车/公交路线规划API
// 请求参数：
//   - Origin: 起点坐标
//   - Destination: 终点坐标
//   - Mode: 出行方式（driving/transit）
//   - Strategy: 路线策略
//   - Waypoints: 途经点
func (m *MapTianditu) RoutePlanning(ctx context.Context, req *RoutePlanningRequest) (*RoutePlanningResponse, error) {
	mode := mustMode(req)
	path := "/drive"
	requestType := "search"
	if mode == TravelModeTransit {
		path = "/transit"
		requestType = "busline"
	}
	if mode != TravelModeDriving && mode != TravelModeTransit {
		return nil, newUnsupportedResponse(m.Name(), string(mode))
	}

	// 构建请求参数
	post := map[string]interface{}{
		"orig":  req.Origin,
		"dest":  req.Destination,
		"style": tiandituDriveStyle(req.Strategy),
	}
	if req.Waypoints != "" {
		post["mid"] = strings.ReplaceAll(req.Waypoints, "|", ";")
	}
	if mode == TravelModeTransit {
		post = map[string]interface{}{
			"startposition": req.Origin,
			"endposition":   req.Destination,
			"linetype":      tiandituTransitLineType(req.Strategy),
		}
	}
	postStr, _ := json.Marshal(post)
	params := url.Values{}
	params.Set("postStr", string(postStr))
	params.Set("type", requestType)
	params.Set("tk", m.config.Key)

	// 调用对应的解析方法
	if mode == TravelModeDriving {
		return m.parseDrivingRoute(ctx, path, params)
	}
	return m.parseTransitRoute(ctx, path, params)
}

// tiandituDriveStyle 将策略名称转换为天地图API所需的代码
//   - "shortest" -> "1": 最短路线
//   - "avoid_highways" -> "2": 避开高速
//   - 其他 -> "0": 推荐路线（默认）
func tiandituDriveStyle(strategy string) string {
	switch strings.TrimSpace(strategy) {
	case "shortest":
		return "1"
	case "avoid_highways":
		return "2"
	default:
		return "0"
	}
}

// tiandituTransitLineType 将策略名称转换为天地图公交规划线路类型
//   - "shortest" -> "2": 最少换乘
//   - "avoid_subway" -> "8": 避开地铁
//   - "less_walking" -> "4": 少步行
//   - 其他 -> "1": 推荐（默认）
func tiandituTransitLineType(strategy string) string {
	switch strings.TrimSpace(strategy) {
	case "shortest":
		return "2"
	case "avoid_subway":
		return "8"
	case "less_walking":
		return "4"
	default:
		return "1"
	}
}

// =============================================================================
// 行政区划查询实现
// =============================================================================

// AdministrativeRegionQuery 实现Provider接口的行政区划查询方法
// 使用天地图行政区划查询API
// 注意：仅支持按关键字查询，不支持按坐标查询
func (m *MapTianditu) AdministrativeRegionQuery(ctx context.Context, req *AdministrativeRegionRequest) (*AdministrativeRegionResponse, error) {
	if strings.TrimSpace(req.Keyword) == "" {
		return nil, newUnsupportedResponse(m.Name(), "仅按关键字的行政区划查询")
	}

	// 构建请求参数
	post := map[string]interface{}{
		"queryType": 12,
		"start":     0,
		"count":     10,
		"specify":   req.Keyword,
	}
	postStr, _ := json.Marshal(post)
	params := url.Values{}
	params.Set("postStr", string(postStr))
	params.Set("type", "query")
	params.Set("tk", m.config.Key)

	// 定义响应结构
	var resp struct {
		Area []struct {
			Name      string `json:"name"`
			AdminCode string `json:"adminCode"`
			Level     string `json:"level"`
			Lonlat    string `json:"lonlat"`
			Bound     string `json:"bound"`
		} `json:"area"`
		Status struct {
			InfoCode int    `json:"infocode"`
			Desc     string `json:"cndesc"`
		} `json:"status"`
	}

	// 发送请求
	if err := m.getJSON(ctx, "/v2/search", params, &resp); err != nil {
		return nil, err
	}
	if resp.Status.InfoCode != 1000 && resp.Status.InfoCode != 0 {
		return nil, fmt.Errorf("天地图行政区划查询失败: %s", resp.Status.Desc)
	}

	// 转换结果
	out := &AdministrativeRegionResponse{
		Status:      "ok",
		Message:     "success",
		Provider:    string(m.Name()),
		RawResponse: resp,
	}
	for _, item := range resp.Area {
		out.Regions = append(out.Regions, AdministrativeRegion{
			Code:     item.AdminCode,
			Name:     item.Name,
			Level:    item.Level,
			Location: item.Lonlat,
			Boundary: item.Bound,
		})
	}
	return out, nil
}

// =============================================================================
// 地理编码实现
// =============================================================================

// Geocoding 实现Provider接口的地理编码方法（地址转坐标）
// 使用天地图Geocoder API
func (m *MapTianditu) Geocoding(ctx context.Context, req *GeocodingRequest) (*GeocodingResponse, error) {
	// 构建请求参数
	ds, _ := json.Marshal(map[string]interface{}{
		"keyWord": strings.TrimSpace(strings.TrimSpace(req.City) + strings.TrimSpace(req.Address)),
	})
	params := url.Values{}
	params.Set("ds", string(ds))
	params.Set("tk", m.config.Key)

	// 定义响应结构
	var resp struct {
		Location struct {
			Lon string `json:"lon"`
			Lat string `json:"lat"`
		} `json:"location"`
		AddressComponent struct {
			Address  string `json:"address"`
			Province string `json:"province"`
			City     string `json:"city"`
			District string `json:"county"`
		} `json:"addressComponent"`
		Status string `json:"status"`
		Msg    string `json:"msg"`
	}

	// 发送请求
	if err := m.getJSON(ctx, "/geocoder", params, &resp); err != nil {
		return nil, err
	}
	if resp.Status != "0" || resp.Location.Lon == "" || resp.Location.Lat == "" {
		return nil, fmt.Errorf("天地图地理编码失败: %s", resp.Msg)
	}

	// 解析坐标
	lng, _ := strconv.ParseFloat(resp.Location.Lon, 64)
	lat, _ := strconv.ParseFloat(resp.Location.Lat, 64)
	return &GeocodingResponse{
		Status:      "ok",
		Message:     "success",
		Provider:    string(m.Name()),
		RawResponse: resp,
		Locations: []GeocodingResult{
			{
				Latitude:  lat,
				Longitude: lng,
				Precise:   true,
				Address:   resp.AddressComponent.Address,
				Province:  resp.AddressComponent.Province,
				City:      resp.AddressComponent.City,
				District:  resp.AddressComponent.District,
			},
		},
	}, nil
}

// =============================================================================
// 逆地理编码实现
// =============================================================================

// ReverseGeocoding 实现Provider接口的逆地理编码方法（坐标转地址）
// 使用天地图Geocoder API
func (m *MapTianditu) ReverseGeocoding(ctx context.Context, req *ReverseGeocodingRequest) (*ReverseGeocodingResponse, error) {
	// 解析坐标
	lng, lat, err := parseCoordinatePair(req.Location)
	if err != nil {
		return nil, err
	}

	// 构建请求参数
	params := url.Values{}
	params.Set("postStr", fmt.Sprintf("{\"lon\":%f,\"lat\":%f,\"ver\":1}", lng, lat))
	params.Set("type", "geocode")
	params.Set("tk", m.config.Key)

	// 定义响应结构
	var resp struct {
		Result struct {
			FormattedAddress string `json:"formatted_address"`
			AddressComponent struct {
				Country  string `json:"country"`
				Province string `json:"province"`
				City     string `json:"city"`
				County   string `json:"county"`
				Town     string `json:"town"`
				Road     string `json:"road"`
			} `json:"addressComponent"`
		} `json:"result"`
		Msg string `json:"msg"`
	}

	// 发送请求
	if err := m.getJSON(ctx, "/geocoder", params, &resp); err != nil {
		return nil, err
	}
	if resp.Msg != "ok" && resp.Msg != "OK" && resp.Result.FormattedAddress == "" {
		return nil, fmt.Errorf("天地图逆地理编码失败: %s", resp.Msg)
	}
	return &ReverseGeocodingResponse{
		Status:      "ok",
		Message:     "success",
		Provider:    string(m.Name()),
		RawResponse: resp,
		Result: ReverseGeocodingResult{
			FormattedAddress: resp.Result.FormattedAddress,
			Country:          resp.Result.AddressComponent.Country,
			Province:         resp.Result.AddressComponent.Province,
			City:             resp.Result.AddressComponent.City,
			District:         resp.Result.AddressComponent.County,
			Township:         resp.Result.AddressComponent.Town,
			Street:           resp.Result.AddressComponent.Road,
			Longitude:        lng,
			Latitude:         lat,
		},
	}, nil
}

// =============================================================================
// 内部解析方法
// =============================================================================

// parseDrivingRoute 解析驾车路线响应
// 天地图驾车路线返回XML格���，���要特殊解析
func (m *MapTianditu) parseDrivingRoute(ctx context.Context, path string, params url.Values) (*RoutePlanningResponse, error) {
	// 获取响应字节数据
	body, err := m.getBytes(ctx, path, params)
	if err != nil {
		return nil, err
	}

	// 解析XML响应
	var resp struct {
		XMLName  xml.Name `xml:"result"`
		Distance string   `xml:"distance"`
		Duration string   `xml:"duration"`
		Simple   struct {
			Items []struct {
				Strguide       string `xml:"strguide"`
				StreetLatLon   string `xml:"streetLatLon"`
				StreetDistance string `xml:"streetDistance"`
			} `xml:"item"`
		} `xml:"simple"`
	}
	if err := xml.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("解析天地图驾车规划响应失败: %w, body=%s", err, string(body))
	}

	// 组装路线数据
	route := Route{
		Distance: parseFloat(resp.Distance),
		Duration: int64(parseFloat(resp.Duration)),
	}
	for _, item := range resp.Simple.Items {
		route.Steps = append(route.Steps, RouteStep{
			Instruction: item.Strguide,
			Distance:    parseFloat(item.StreetDistance),
			Polyline:    item.StreetLatLon,
		})
	}

	return &RoutePlanningResponse{
		Status:      "ok",
		Message:     "success",
		Provider:    string(m.Name()),
		Distance:    route.Distance,
		Duration:    route.Duration,
		Routes:      []Route{route},
		RawResponse: resp,
	}, nil
}

// parseTransitRoute 解析公交路线响应
// 天地图公交路线返回JSON格式
func (m *MapTianditu) parseTransitRoute(ctx context.Context, path string, params url.Values) (*RoutePlanningResponse, error) {
	// 定义响应结构
	var resp struct {
		ResultCode int `json:"resultCode"`
		Results    []struct {
			LineType int `json:"lineType"`
			Lines    []struct {
				LineName string `json:"lineName"`
				Segments []struct {
					SegmentType int `json:"segmentType"`
					SegmentLine []struct {
						SegmentName     string  `json:"segmentName"`
						Direction       string  `json:"direction"`
						LineName        string  `json:"lineName"`
						LinePoint       string  `json:"linePoint"`
						SegmentDistance float64 `json:"segmentDistance"`
						SegmentTime     int64   `json:"segmentTime"`
					} `json:"segmentLine"`
				} `json:"segments"`
			} `json:"lines"`
		} `json:"results"`
	}

	// 发送请求
	if err := m.getJSON(ctx, path, params, &resp); err != nil {
		return nil, err
	}
	if resp.ResultCode != 0 && resp.ResultCode != 5 {
		return nil, fmt.Errorf("天地图公交规划失败: resultCode=%d", resp.ResultCode)
	}

	// 转换结果
	out := &RoutePlanningResponse{
		Status:      "ok",
		Message:     "success",
		Provider:    string(m.Name()),
		RawResponse: resp,
	}
	for _, result := range resp.Results {
		for _, line := range result.Lines {
			route := Route{Summary: line.LineName}
			for _, segment := range line.Segments {
				for _, item := range segment.SegmentLine {
					route.Distance += item.SegmentDistance
					route.Duration += item.SegmentTime
					route.Steps = append(route.Steps, RouteStep{
						Instruction: firstNotEmpty(item.Direction, item.LineName, item.SegmentName),
						Distance:    item.SegmentDistance,
						Duration:    item.SegmentTime,
						Polyline:    item.LinePoint,
					})
				}
			}
			if out.Distance == 0 {
				out.Distance = route.Distance
				out.Duration = route.Duration
			}
			out.Routes = append(out.Routes, route)
		}
	}
	return out, nil
}
