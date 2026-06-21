package maps

// =============================================================================
// 百度地图服务提供商实现
// 提供对百度地图API的封装，实现Provider接口
// 支持：地点搜索、线路规划、地理编码、逆地理编码、行政区划查询
// API文档：https://lbsyun.baidu.com/index.php?title=webapi
// =============================================================================

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// 确保MapBaidu实现了Provider接口
var _ Provider = (*MapBaidu)(nil)

// =============================================================================
// MapBaidu 结构体定义
// =============================================================================

// MapBaidu 百度地图服务提供商
type MapBaidu struct {
	baseProvider // 嵌入基础Provider，提供通用功能
}

// NewMapBaidu 创建百度地图提供商实例
// 参数说明：
//   - cfg: 百度地图配置，包含API密钥等
//   - timeout: HTTP请求超时时间
//
// 返回值：百度地图Provider实例
func NewMapBaidu(cfg ProviderConfig, timeout time.Duration) *MapBaidu {
	return &MapBaidu{
		baseProvider: newBaseProvider(MapNameBaidu, cfg, timeout, "https://api.map.baidu.com"),
	}
}

// =============================================================================
// 地点搜索实现
// =============================================================================

// PlaceSearch 实现Provider接口的地点搜索方法
// 使用百度地图Place API v2进行POI搜索
// 请求参数：
//   - Query: 搜索关键字
//   - City: 搜索城市
//   - Location: 中心点坐标（用于附近搜索）
//   - Radius: 搜索半径
//   - PageSize/PageIndex: 分页参数
func (m *MapBaidu) PlaceSearch(ctx context.Context, req *PlaceSearchRequest) (*PlaceSearchResponse, error) {
	// 构建请求参数
	params := url.Values{}
	params.Set("query", req.Query)
	params.Set("output", "json")
	params.Set("ak", m.config.Key)
	params.Set("page_size", strconv.Itoa(defaultPageSize(req.PageSize)))
	params.Set("page_num", strconv.Itoa(defaultPageIndex(req.PageIndex)-1))
	if req.City != "" {
		params.Set("region", req.City)
	}
	if req.Location != "" {
		params.Set("location", req.Location)
		params.Set("radius", strconv.Itoa(defaultRadius(req.Radius)))
	}

	// 定义响应结构
	var resp struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
		Total   int    `json:"total"`
		Results []struct {
			Name      string `json:"name"`
			Address   string `json:"address"`
			Province  string `json:"province"`
			City      string `json:"city"`
			Area      string `json:"area"`
			Telephone string `json:"telephone"`
			UID       string `json:"uid"`
			Location  struct {
				Lng float64 `json:"lng"`
				Lat float64 `json:"lat"`
			} `json:"location"`
		} `json:"results"`
	}

	// 发送请求
	if err := m.getJSON(ctx, "/place/v2/search", params, &resp); err != nil {
		return nil, err
	}
	// 检查响应状态
	if resp.Status != 0 {
		return nil, fmt.Errorf("百度地点搜索失败: %s", resp.Message)
	}

	// 转换结果
	result := &PlaceSearchResponse{
		Status:      "ok",
		Message:     "success",
		Provider:    string(m.Name()),
		Total:       resp.Total,
		RawResponse: resp,
	}
	for _, item := range resp.Results {
		result.Places = append(result.Places, Place{
			ID:       item.UID,
			Name:     item.Name,
			Address:  item.Address,
			Location: fmt.Sprintf("%f,%f", item.Location.Lng, item.Location.Lat),
			Phone:    item.Telephone,
			Province: item.Province,
			City:     item.City,
			District: item.Area,
		})
	}
	return result, nil
}

// =============================================================================
// 线路规划实现
// =============================================================================

// RoutePlanning 实现Provider接口的线路规划方法
// 使用百度地图Direction Lite API进行驾车/公交路线规划
// 请求参数：
//   - Origin: 起点坐标
//   - Destination: 终点坐标
//   - Mode: 出行方式（driving/transit）
//   - Strategy: 路线策略
//   - Waypoints: 途经点
func (m *MapBaidu) RoutePlanning(ctx context.Context, req *RoutePlanningRequest) (*RoutePlanningResponse, error) {
	mode := mustMode(req)
	path := "/directionlite/v1/driving"

	// 根据出行模式选择API路径
	switch mode {
	case TravelModeDriving:
		path = "/directionlite/v1/driving"
	case TravelModeTransit:
		path = "/directionlite/v1/transit"
	default:
		return nil, newUnsupportedResponse(m.Name(), string(mode))
	}

	// 构建请求参数
	params := url.Values{}
	params.Set("origin", ensureLatLng(req.Origin))
	params.Set("destination", ensureLatLng(req.Destination))
	params.Set("ak", m.config.Key)
	if req.City != "" {
		params.Set("region", req.City)
	}
	if req.Waypoints != "" && mode == TravelModeDriving {
		params.Set("waypoints", req.Waypoints)
	}

	// 定义响应结构
	var resp struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
		Result  struct {
			Routes []struct {
				Distance int64             `json:"distance"`
				Duration int64             `json:"duration"`
				Steps    []json.RawMessage `json:"steps"`
			} `json:"routes"`
		} `json:"result"`
	}

	// 发送请求
	if err := m.getJSON(ctx, path, params, &resp); err != nil {
		return nil, err
	}
	if resp.Status != 0 {
		return nil, fmt.Errorf("百度路线规划失败: %s", resp.Message)
	}

	// 转换结果
	out := &RoutePlanningResponse{
		Status:      "ok",
		Message:     "success",
		Provider:    string(m.Name()),
		RawResponse: resp,
	}
	for _, item := range resp.Result.Routes {
		route := Route{
			Distance: float64(item.Distance),
			Duration: item.Duration,
		}
		// 解析路线步骤
		for _, raw := range item.Steps {
			var decoded any
			if err := json.Unmarshal(raw, &decoded); err != nil {
				continue
			}
			switch v := decoded.(type) {
			case []any:
				for _, elem := range v {
					if stepMap, ok := elem.(map[string]any); ok {
						appendBaiduStep(&route, stepMap)
					}
				}
			case map[string]any:
				appendBaiduStep(&route, v)
			}
		}
		// 保存第一条路线作为主路线
		if out.Distance == 0 {
			out.Distance = route.Distance
			out.Duration = route.Duration
		}
		out.Routes = append(out.Routes, route)
	}
	return out, nil
}

// ensureLatLng 确保坐标格式为纬度,经度（百度API要求）
func ensureLatLng(value string) string {
	parts := strings.Split(value, ",")
	if len(parts) != 2 {
		return value
	}
	lat := strings.TrimSpace(parts[1])
	lng := strings.TrimSpace(parts[0])
	return fmt.Sprintf("%s,%s", lat, lng)
}

// appendBaiduStep 将百度返回的步骤添加到路线中
func appendBaiduStep(route *Route, step map[string]any) {
	route.Steps = append(route.Steps, RouteStep{
		Instruction: anyToString(step["instruction"]),
		Distance:    parseFloat(fmt.Sprint(step["distance"])),
		Duration:    int64(parseFloat(fmt.Sprint(step["duration"]))),
		Polyline:    anyToString(step["path"]),
	})
}

// =============================================================================
// 行政区划查询实现
// =============================================================================

// AdministrativeRegionQuery 实现Provider接口的行政区划查询方法
// 通过逆地理编码获取坐标所在行政区划信息
func (m *MapBaidu) AdministrativeRegionQuery(ctx context.Context, req *AdministrativeRegionRequest) (*AdministrativeRegionResponse, error) {
	if req.Location == "" {
		return nil, fmt.Errorf("location 不能为空")
	}
	// 调用逆地理编码获取行政区划
	resp, err := m.ReverseGeocoding(ctx, &ReverseGeocodingRequest{
		Location: req.Location,
		Options:  req.Options,
	})
	if err != nil {
		return nil, err
	}
	// 组装行政区划结果
	region := AdministrativeRegion{
		Name:     firstNotEmpty(resp.Result.District, resp.Result.City, resp.Result.Province),
		Level:    req.Level,
		Location: req.Location,
	}
	return &AdministrativeRegionResponse{
		Status:      resp.Status,
		Message:     resp.Message,
		Provider:    resp.Provider,
		Regions:     []AdministrativeRegion{region},
		RawResponse: resp.RawResponse,
	}, nil
}

// =============================================================================
// 地理编码实现
// =============================================================================

// Geocoding 实现Provider接口的地理编码方法（地址转坐标）
// 使用百度地图Geocoding API v3
func (m *MapBaidu) Geocoding(ctx context.Context, req *GeocodingRequest) (*GeocodingResponse, error) {
	if req.Address == "" {
		return nil, fmt.Errorf("地址不能为空")
	}

	// 构建请求参数
	params := url.Values{}
	params.Set("address", req.Address)
	params.Set("output", "json")
	params.Set("ak", m.config.Key)
	if req.City != "" {
		params.Set("city", req.City)
	}

	// 定义响应结构
	var resp struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
		Result  struct {
			Location struct {
				Lng float64 `json:"lng"`
				Lat float64 `json:"lat"`
			} `json:"location"`
			Precise int    `json:"precise"`
			Level   string `json:"level"`
		} `json:"result"`
	}

	// 发送请求
	if err := m.getJSON(ctx, "/geocoding/v3/", params, &resp); err != nil {
		return nil, err
	}
	if resp.Status != 0 {
		return &GeocodingResponse{
			Status:      "error",
			Message:     fmt.Sprintf("百度地图 API 错误: %d - %s", resp.Status, resp.Message),
			Provider:    string(m.Name()),
			RawResponse: resp,
		}, nil
	}

	// 返回结果
	return &GeocodingResponse{
		Status:      "ok",
		Message:     "success",
		Provider:    string(m.Name()),
		RawResponse: resp,
		Locations: []GeocodingResult{
			{
				Latitude:  resp.Result.Location.Lat,
				Longitude: resp.Result.Location.Lng,
				Precise:   resp.Result.Precise == 1,
				Level:     resp.Result.Level,
				Address:   req.Address,
				City:      req.City,
			},
		},
	}, nil
}

// =============================================================================
// 逆地理编码实现
// =============================================================================

// ReverseGeocoding 实现Provider接口的逆地理编码方法（坐标转地址）
// 使用百度地图Reverse Geocoding API v3
func (m *MapBaidu) ReverseGeocoding(ctx context.Context, req *ReverseGeocodingRequest) (*ReverseGeocodingResponse, error) {
	if strings.TrimSpace(req.Location) == "" {
		return nil, fmt.Errorf("location 不能为空")
	}

	// 构建请求参数
	params := url.Values{}
	params.Set("location", req.Location)
	params.Set("output", "json")
	params.Set("ak", m.config.Key)
	params.Set("extensions_poi", "0")

	// 定义响应结构
	var resp struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
		Result  struct {
			FormattedAddress string `json:"formatted_address"`
			AddressComponent struct {
				Country      string `json:"country"`
				Province     string `json:"province"`
				City         string `json:"city"`
				District     string `json:"district"`
				Town         string `json:"town"`
				Street       string `json:"street"`
				StreetNumber string `json:"street_number"`
			} `json:"addressComponent"`
		} `json:"result"`
	}

	// 发送请求
	if err := m.getJSON(ctx, "/reverse_geocoding/v3/", params, &resp); err != nil {
		return nil, err
	}
	if resp.Status != 0 {
		return nil, fmt.Errorf("百度逆地理编码失败: %s", resp.Message)
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
			FormattedAddress: resp.Result.FormattedAddress,
			Country:          resp.Result.AddressComponent.Country,
			Province:         resp.Result.AddressComponent.Province,
			City:             resp.Result.AddressComponent.City,
			District:         resp.Result.AddressComponent.District,
			Township:         resp.Result.AddressComponent.Town,
			Street:           resp.Result.AddressComponent.Street,
			StreetNumber:     resp.Result.AddressComponent.StreetNumber,
			Longitude:        lng,
			Latitude:         lat,
		},
	}, nil
}

// =============================================================================
// 辅助函数
// =============================================================================

// firstNotEmpty 返回第一个非空字符串
func firstNotEmpty(values ...string) string {
	for _, item := range values {
		if strings.TrimSpace(item) != "" {
			return item
		}
	}
	return ""
}
