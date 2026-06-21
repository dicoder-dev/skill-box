package maps

// =============================================================================
// Google地图服务提供商实现
// 提供对Google Maps API的封装，实现Provider接口
// 支持：地点搜索、线路规划、地理编码、逆地理编码、行政区划查询
// API文档：https://developers.google.com/maps/documentation
// =============================================================================

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// 确保MapGoogle实现了Provider接口
var _ Provider = (*MapGoogle)(nil)

// =============================================================================
// MapGoogle 结构体定义
// =============================================================================

// MapGoogle Google地图服务提供商
type MapGoogle struct {
	baseProvider // 嵌入基础Provider，提供通用功能
}

// NewMapGoogle 创建Google地图提供商实例
// 参数说明：
//   - cfg: Google地图配置，包含API密钥等
//   - timeout: HTTP请求超时时间
//
// 返回值：Google地图Provider实例
func NewMapGoogle(cfg ProviderConfig, timeout time.Duration) *MapGoogle {
	return &MapGoogle{
		baseProvider: newBaseProvider(MapNameGoogle, cfg, timeout, "https://maps.googleapis.com"),
	}
}

// =============================================================================
// 地点搜索实现
// =============================================================================

// PlaceSearch 实现Provider接口的地点搜索方法
// 使用Google Places API进行POI搜索
// 请求参数：
//   - Query: 搜索关键字
//   - Location: 中心点坐标（用于附近搜索）
//   - Radius: 搜索半径
//   - PageSize: 每页返回数量
func (m *MapGoogle) PlaceSearch(ctx context.Context, req *PlaceSearchRequest) (*PlaceSearchResponse, error) {
	// 构建请求参数
	params := url.Values{}
	params.Set("key", m.config.Key)
	params.Set("query", req.Query)
	if req.Location != "" {
		lng, lat, err := parseCoordinatePair(req.Location)
		if err == nil {
			params.Set("location", fmt.Sprintf("%f,%f", lat, lng))
		}
	}
	if req.Radius > 0 {
		params.Set("radius", strconv.Itoa(defaultRadius(req.Radius)))
	}
	if m.config.Language != "" {
		params.Set("language", m.config.Language)
	}

	// 定义响应结构
	var resp struct {
		Status  string `json:"status"`
		Results []struct {
			Name             string   `json:"name"`
			FormattedAddress string   `json:"formatted_address"`
			PlaceID          string   `json:"place_id"`
			Types            []string `json:"types"`
			Geometry         struct {
				Location struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"location"`
			} `json:"geometry"`
		} `json:"results"`
		ErrorMessage string `json:"error_message"`
	}

	// 发送请求
	if err := m.getJSON(ctx, "/maps/api/place/textsearch/json", params, &resp); err != nil {
		return nil, err
	}
	if resp.Status != "OK" && resp.Status != "ZERO_RESULTS" {
		return nil, fmt.Errorf("Google 地点搜索失败: %s", firstNotEmpty(resp.ErrorMessage, resp.Status))
	}

	// 转换结果
	out := &PlaceSearchResponse{
		Status:      "ok",
		Message:     "success",
		Provider:    string(m.Name()),
		Total:       len(resp.Results),
		RawResponse: resp,
	}
	for _, item := range resp.Results {
		out.Places = append(out.Places, Place{
			ID:       item.PlaceID,
			Name:     item.Name,
			Address:  item.FormattedAddress,
			Location: fmt.Sprintf("%f,%f", item.Geometry.Location.Lng, item.Geometry.Location.Lat),
			Type:     strings.Join(item.Types, ","),
		})
	}
	return out, nil
}

// =============================================================================
// 线路规划实现
// =============================================================================

// RoutePlanning 实现Provider接口的线路规划方法
// 使用Google Directions API进行驾车/公交/步行路线规划
// 请求参数：
//   - Origin: 起点坐标或地址
//   - Destination: 终点坐标或地址
//   - Mode: 出行方式（driving/transit/walking）
//   - Waypoints: 途经点
func (m *MapGoogle) RoutePlanning(ctx context.Context, req *RoutePlanningRequest) (*RoutePlanningResponse, error) {
	// 构建请求参数
	params := url.Values{}
	params.Set("key", m.config.Key)
	params.Set("origin", req.Origin)
	params.Set("destination", req.Destination)
	mode := mustMode(req)
	params.Set("mode", string(mode))

	// 定义响应结构
	var resp struct {
		Status       string `json:"status"`
		ErrorMessage string `json:"error_message"`
		Routes       []struct {
			Summary          string `json:"summary"`
			OverviewPolyline struct {
				Points string `json:"points"`
			} `json:"overview_polyline"`
			Legs []struct {
				Distance struct {
					Value int64 `json:"value"`
				} `json:"distance"`
				Duration struct {
					Value int64 `json:"value"`
				} `json:"duration"`
				Steps []struct {
					HTMLInstructions string `json:"html_instructions"`
					Distance         struct {
						Value int64 `json:"value"`
					} `json:"distance"`
					Duration struct {
						Value int64 `json:"value"`
					} `json:"duration"`
					Polyline struct {
						Points string `json:"points"`
					} `json:"polyline"`
				} `json:"steps"`
			} `json:"legs"`
		} `json:"routes"`
	}

	// 发送请求
	if err := m.getJSON(ctx, "/maps/api/directions/json", params, &resp); err != nil {
		return nil, err
	}
	if resp.Status != "OK" && resp.Status != "ZERO_RESULTS" {
		return nil, fmt.Errorf("Google 路线规划失败: %s", firstNotEmpty(resp.ErrorMessage, resp.Status))
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
			Polyline: item.OverviewPolyline.Points,
			Summary:  item.Summary,
		}
		for _, leg := range item.Legs {
			route.Distance += float64(leg.Distance.Value)
			route.Duration += leg.Duration.Value
			for _, step := range leg.Steps {
				route.Steps = append(route.Steps, RouteStep{
					Instruction: step.HTMLInstructions,
					Distance:    float64(step.Distance.Value),
					Duration:    step.Duration.Value,
					Polyline:    step.Polyline.Points,
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

// =============================================================================
// 行政区划查询实现
// =============================================================================

// AdministrativeRegionQuery 实现Provider接口的行政区划查询方法
// 通过逆地理编码获取坐标所在行政区划信息
func (m *MapGoogle) AdministrativeRegionQuery(ctx context.Context, req *AdministrativeRegionRequest) (*AdministrativeRegionResponse, error) {
	if req.Location == "" {
		return nil, fmt.Errorf("location 不能为空")
	}
	// 调用逆地理编码
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
// 使用Google Geocoding API
func (m *MapGoogle) Geocoding(ctx context.Context, req *GeocodingRequest) (*GeocodingResponse, error) {
	// 构建请求参数
	params := url.Values{}
	params.Set("key", m.config.Key)
	params.Set("address", req.Address)

	// 定义响应结构
	var resp struct {
		Status       string `json:"status"`
		ErrorMessage string `json:"error_message"`
		Results      []struct {
			FormattedAddress string `json:"formatted_address"`
			Geometry         struct {
				Location struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"geometry"`
			} `json:"geometry"`
			Types             []string `json:"types"`
			AddressComponents []struct {
				LongName string   `json:"long_name"`
				Types    []string `json:"types"`
			} `json:"address_components"`
		} `json:"results"`
	}

	// 发送请求
	if err := m.getJSON(ctx, "/maps/api/geocode/json", params, &resp); err != nil {
		return nil, err
	}
	if resp.Status != "OK" && resp.Status != "ZERO_RESULTS" {
		return nil, fmt.Errorf("Google 地理编码失败: %s", firstNotEmpty(resp.ErrorMessage, resp.Status))
	}

	// 转换结果
	out := &GeocodingResponse{
		Status:      "ok",
		Message:     "success",
		Provider:    string(m.Name()),
		RawResponse: resp,
	}
	for _, item := range resp.Results {
		out.Locations = append(out.Locations, GeocodingResult{
			Latitude:  item.Geometry.Location.Lat,
			Longitude: item.Geometry.Location.Lng,
			Precise:   true,
			Level:     strings.Join(item.Types, ","),
			Province:  findGoogleComponent(item.AddressComponents, "administrative_area_level_1"),
			City:      findGoogleComponent(item.AddressComponents, "locality"),
			District:  findGoogleComponent(item.AddressComponents, "administrative_area_level_2"),
			Address:   item.FormattedAddress,
		})
	}
	return out, nil
}

// findGoogleComponent 在地址组件中查找指定类型的值
// Google返回的地址组件有多种类型，如：
//   - administrative_area_level_1: 省/州
//   - locality: 城市
//   - administrative_area_level_2: 区县
//   - route: 道路
//   - street_number: 门牌号
func findGoogleComponent(components []struct {
	LongName string   `json:"long_name"`
	Types    []string `json:"types"`
}, target string) string {
	for _, item := range components {
		for _, itemType := range item.Types {
			if itemType == target {
				return item.LongName
			}
		}
	}
	return ""
}

// =============================================================================
// 逆地理编码实现
// =============================================================================

// ReverseGeocoding 实现Provider接口的逆地理编码方法（坐标转地址）
// 使用Google Geocoding API
func (m *MapGoogle) ReverseGeocoding(ctx context.Context, req *ReverseGeocodingRequest) (*ReverseGeocodingResponse, error) {
	// 解析坐标
	lng, lat, err := parseCoordinatePair(req.Location)
	if err != nil {
		return nil, err
	}

	// 构建请求参数
	params := url.Values{}
	params.Set("key", m.config.Key)
	params.Set("latlng", fmt.Sprintf("%f,%f", lat, lng))

	// 定义响应结构
	var resp struct {
		Status       string `json:"status"`
		ErrorMessage string `json:"error_message"`
		Results      []struct {
			FormattedAddress  string `json:"formatted_address"`
			AddressComponents []struct {
				LongName string   `json:"long_name"`
				Types    []string `json:"types"`
			} `json:"address_components"`
		} `json:"results"`
	}

	// 发送请求
	if err := m.getJSON(ctx, "/maps/api/geocode/json", params, &resp); err != nil {
		return nil, err
	}
	if resp.Status != "OK" && resp.Status != "ZERO_RESULTS" {
		return nil, fmt.Errorf("Google 逆地理编码失败: %s", firstNotEmpty(resp.ErrorMessage, resp.Status))
	}

	// 解析结果
	result := ReverseGeocodingResult{
		Latitude:  lat,
		Longitude: lng,
	}
	if len(resp.Results) > 0 {
		item := resp.Results[0]
		result.FormattedAddress = item.FormattedAddress
		result.Country = findGoogleComponent(item.AddressComponents, "country")
		result.Province = findGoogleComponent(item.AddressComponents, "administrative_area_level_1")
		result.City = findGoogleComponent(item.AddressComponents, "locality")
		result.District = firstNotEmpty(
			findGoogleComponent(item.AddressComponents, "administrative_area_level_2"),
			findGoogleComponent(item.AddressComponents, "sublocality"),
		)
		result.Street = findGoogleComponent(item.AddressComponents, "route")
		result.StreetNumber = findGoogleComponent(item.AddressComponents, "street_number")
	}

	return &ReverseGeocodingResponse{
		Status:      "ok",
		Message:     "success",
		Provider:    string(m.Name()),
		Result:      result,
		RawResponse: resp,
	}, nil
}
