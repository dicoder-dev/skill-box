package maps

// =============================================================================
// 高德地图服务提供商实现
// 提供对高德地图API的封装，实现Provider接口
// 支持：地点搜索、线路规划、地理编码、逆地理编码
// API文档：https://lbs.amap.com/api/webservice/guide/create-project/get-started
// =============================================================================

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// 确保MapAmap实现了Provider接口
var _ Provider = (*MapAmap)(nil)

// =============================================================================
// MapAmap 结构体定义
// =============================================================================

// MapAmap 高德地图服务提供商
type MapAmap struct {
	baseProvider // 嵌入基础Provider，提供通用功能
}

// NewMapAmap 创建高德地图提供商实例
// 参数说明：
//   - cfg: 高德地图配置，包含API密钥等
//   - timeout: HTTP请求超时时间
//
// 返回值：高德地图Provider实例
func NewMapAmap(cfg ProviderConfig, timeout time.Duration) *MapAmap {
	return &MapAmap{
		baseProvider: newBaseProvider(MapNameAmap, cfg, timeout, "https://restapi.amap.com"),
	}
}

// =============================================================================
// 地点搜索实现
// =============================================================================

// PlaceSearch 实现Provider接口的地点搜索方法
// 使用高德地图Place Text API进行POI搜索
// 请求参数：
//   - Query: 搜索关键字
//   - City: 搜索城市
//   - Location/Radius: 周边搜索参数
//   - Types: POI类型过滤
//   - PageSize/PageIndex: 分页参数
func (m *MapAmap) PlaceSearch(ctx context.Context, req *PlaceSearchRequest) (*PlaceSearchResponse, error) {
	// 构建请求参数
	params := url.Values{}
	params.Set("key", m.config.Key)
	params.Set("keywords", req.Query)
	params.Set("offset", strconv.Itoa(defaultPageSize(req.PageSize)))
	params.Set("page", strconv.Itoa(defaultPageIndex(req.PageIndex)))
	if req.City != "" {
		params.Set("city", req.City)
	}
	if req.Types != "" {
		params.Set("types", req.Types)
	}
	if req.Location != "" {
		params.Set("location", req.Location)
	}
	if req.Radius > 0 {
		params.Set("radius", strconv.Itoa(defaultRadius(req.Radius)))
	}

	// 定义响应结构
	var resp struct {
		Status string `json:"status"`
		Info   string `json:"info"`
		Count  string `json:"count"`
		POIs   []struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			Address  string `json:"address"`
			Location string `json:"location"`
			Type     string `json:"type"`
			Tel      any    `json:"tel"`
			Business struct {
				Rating string `json:"rating"`
			} `json:"business"`
			Pname    string `json:"pname"`
			Cityname string `json:"cityname"`
			Adname   string `json:"adname"`
		} `json:"pois"`
	}

	// 发送请求
	if err := m.getJSON(ctx, "/v3/place/text", params, &resp); err != nil {
		return nil, err
	}
	if resp.Status != "1" {
		return nil, fmt.Errorf("高德地点搜索失败: %s", resp.Info)
	}

	// 转换结果
	total, _ := strconv.Atoi(resp.Count)
	out := &PlaceSearchResponse{
		Status:      "ok",
		Message:     "success",
		Provider:    string(m.Name()),
		Total:       total,
		RawResponse: resp,
	}
	for _, item := range resp.POIs {
		out.Places = append(out.Places, Place{
			ID:       item.ID,
			Name:     item.Name,
			Address:  item.Address,
			Location: item.Location,
			Type:     item.Type,
			Phone:    anyToString(item.Tel),
			Province: item.Pname,
			City:     item.Cityname,
			District: item.Adname,
		})
	}
	return out, nil
}

// =============================================================================
// 线路规划实现
// =============================================================================

// RoutePlanning 实现Provider接口的线路规划方法
// 使用高德地图Direction API进行驾车/公交路线规划
// 请求参数：
//   - Origin: 起点坐标
//   - Destination: 终点坐标
//   - Mode: 出行方式（driving/transit）
//   - Strategy: 路线策略
//   - Waypoints: 途经点
//   - City: 公交规划时使用的城市
func (m *MapAmap) RoutePlanning(ctx context.Context, req *RoutePlanningRequest) (*RoutePlanningResponse, error) {
	// 构建请求参数
	params := url.Values{}
	params.Set("key", m.config.Key)
	params.Set("origin", req.Origin)
	params.Set("destination", req.Destination)
	mode := mustMode(req)

	path := "/v3/direction/driving"
	switch mode {
	case TravelModeDriving:
		// 驾车路线规划
		if req.Strategy != "" {
			params.Set("strategy", amapStrategyCode(req.Strategy))
		}
		if req.Waypoints != "" {
			params.Set("waypoints", req.Waypoints)
		}
	case TravelModeTransit:
		// 公交路线规划
		path = "/v3/direction/transit/integrated"
		if req.City != "" {
			params.Set("city", req.City)
		}
	default:
		return nil, newUnsupportedResponse(m.Name(), string(mode))
	}

	// 定义响应结构
	var resp struct {
		Status string `json:"status"`
		Info   string `json:"info"`
		Route  struct {
			Paths []struct {
				Distance string `json:"distance"`
				Duration string `json:"duration"`
				Steps    []struct {
					Instruction string `json:"instruction"`
					Distance    string `json:"distance"`
					Duration    string `json:"duration"`
					Polyline    string `json:"polyline"`
				} `json:"steps"`
			} `json:"paths"`
			Transits []struct {
				Distance string `json:"distance"`
				Duration string `json:"duration"`
				Segments []struct {
					Walking struct {
						Distance string `json:"distance"`
						Duration string `json:"duration"`
						Steps    []struct {
							Instruction string `json:"instruction"`
							Polyline    string `json:"polyline"`
						} `json:"steps"`
					} `json:"walking"`
				} `json:"segments"`
			} `json:"transits"`
		} `json:"route"`
	}

	// 发送请求
	if err := m.getJSON(ctx, path, params, &resp); err != nil {
		return nil, err
	}
	if resp.Status != "1" {
		return nil, fmt.Errorf("高德路线规划失败: %s", resp.Info)
	}

	// 转换结果
	out := &RoutePlanningResponse{
		Status:      "ok",
		Message:     "success",
		Provider:    string(m.Name()),
		RawResponse: resp,
	}

	// 处理驾车路线
	if mode == TravelModeDriving {
		for _, item := range resp.Route.Paths {
			route := Route{
				Distance: parseFloat(item.Distance),
				Duration: parseInt64(item.Duration),
			}
			for _, step := range item.Steps {
				route.Steps = append(route.Steps, RouteStep{
					Instruction: step.Instruction,
					Distance:    parseFloat(step.Distance),
					Duration:    parseInt64(step.Duration),
					Polyline:    step.Polyline,
				})
			}
			if out.Distance == 0 {
				out.Distance = route.Distance
				out.Duration = route.Duration
			}
			out.Routes = append(out.Routes, route)
		}
		return out, nil
	}

	// 处理公交路线
	for _, item := range resp.Route.Transits {
		route := Route{
			Distance: parseFloat(item.Distance),
			Duration: parseInt64(item.Duration),
			Summary:  "transit",
		}
		for _, segment := range item.Segments {
			for _, step := range segment.Walking.Steps {
				route.Steps = append(route.Steps, RouteStep{
					Instruction: step.Instruction,
					Polyline:    step.Polyline,
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

// amapStrategyCode 将策略名称转换为高德API所需的代码
//   - "shortest" -> "2": 最短路线
//   - "avoid_highways" -> "5": 避开高速
//   - 其他 -> "0": 最快路线（默认）
func amapStrategyCode(strategy string) string {
	switch strings.TrimSpace(strategy) {
	case "shortest":
		return "2"
	case "avoid_highways":
		return "5"
	default:
		return "0"
	}
}

// =============================================================================
// 行政区划查询实现
// =============================================================================

// AdministrativeRegionQuery 实现Provider接口的行政区划查询方法
// 高德地图不支持行政区划查询，返回错误
func (m *MapAmap) AdministrativeRegionQuery(ctx context.Context, req *AdministrativeRegionRequest) (*AdministrativeRegionResponse, error) {
	return nil, newUnsupportedResponse(m.Name(), "行政区域查询")
}

// =============================================================================
// 地理编码实现
// =============================================================================

// Geocoding 实现Provider接口的地理编码方法（地址转坐标）
// 使用高德地图Geocoding API
func (m *MapAmap) Geocoding(ctx context.Context, req *GeocodingRequest) (*GeocodingResponse, error) {
	// 构建请求参数
	params := url.Values{}
	params.Set("key", m.config.Key)
	params.Set("address", req.Address)
	if req.City != "" {
		params.Set("city", req.City)
	}

	// 定义响应结构
	var resp struct {
		Status   string `json:"status"`
		Info     string `json:"info"`
		Geocodes []struct {
			Location         string `json:"location"`
			Level            string `json:"level"`
			Province         string `json:"province"`
			City             any    `json:"city"`
			District         any    `json:"district"`
			FormattedAddress string `json:"formatted_address"`
		} `json:"geocodes"`
	}

	// 发送请求
	if err := m.getJSON(ctx, "/v3/geocode/geo", params, &resp); err != nil {
		return nil, err
	}
	if resp.Status != "1" {
		return nil, fmt.Errorf("高德地理编码失败: %s", resp.Info)
	}

	// 转换结果
	out := &GeocodingResponse{
		Status:      "ok",
		Message:     "success",
		Provider:    string(m.Name()),
		RawResponse: resp,
	}
	for _, item := range resp.Geocodes {
		lng, lat, err := parseCoordinatePair(item.Location)
		if err != nil {
			continue
		}
		out.Locations = append(out.Locations, GeocodingResult{
			Latitude:  lat,
			Longitude: lng,
			Precise:   true,
			Level:     item.Level,
			Province:  item.Province,
			City:      anyToString(item.City),
			District:  anyToString(item.District),
			Address:   item.FormattedAddress,
		})
	}
	return out, nil
}

// =============================================================================
// 逆地理编码实现
// =============================================================================

// ReverseGeocoding 实现Provider接口的逆地理编码方法（坐标转地址）
// 使用高德地图ReGeocoding API
func (m *MapAmap) ReverseGeocoding(ctx context.Context, req *ReverseGeocodingRequest) (*ReverseGeocodingResponse, error) {
	// 构建请求参数
	params := url.Values{}
	params.Set("key", m.config.Key)
	params.Set("location", req.Location)
	params.Set("extensions", "base")
	if req.Radius > 0 {
		params.Set("radius", strconv.Itoa(req.Radius))
	}

	// 定义响应结构
	var resp struct {
		Status    string `json:"status"`
		Info      string `json:"info"`
		Regeocode struct {
			FormattedAddress any `json:"formatted_address"`
			AddressComponent struct {
				Country      any `json:"country"`
				Province     any `json:"province"`
				City         any `json:"city"`
				District     any `json:"district"`
				Township     any `json:"township"`
				StreetNumber struct {
					Street any `json:"street"`
					Number any `json:"number"`
				} `json:"streetNumber"`
			} `json:"addressComponent"`
		} `json:"regeocode"`
	}

	// 发送请求
	if err := m.getJSON(ctx, "/v3/geocode/regeo", params, &resp); err != nil {
		return nil, err
	}
	if resp.Status != "1" {
		return nil, fmt.Errorf("高德逆地理编码失败: %s", resp.Info)
	}

	// 解析坐标
	lng, lat, err := parseCoordinatePair(req.Location)
	if err != nil {
		return nil, err
	}

	// 返回结果
	return &ReverseGeocodingResponse{
		Status:      "ok",
		Message:     "success",
		Provider:    string(m.Name()),
		RawResponse: resp,
		Result: ReverseGeocodingResult{
			FormattedAddress: anyToString(resp.Regeocode.FormattedAddress),
			Country:          anyToString(resp.Regeocode.AddressComponent.Country),
			Province:         anyToString(resp.Regeocode.AddressComponent.Province),
			City:             anyToString(resp.Regeocode.AddressComponent.City),
			District:         anyToString(resp.Regeocode.AddressComponent.District),
			Township:         anyToString(resp.Regeocode.AddressComponent.Township),
			Street:           anyToString(resp.Regeocode.AddressComponent.StreetNumber.Street),
			StreetNumber:     anyToString(resp.Regeocode.AddressComponent.StreetNumber.Number),
			Longitude:        lng,
			Latitude:         lat,
		},
	}, nil
}
