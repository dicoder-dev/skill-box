package maps

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// 强制实现IMapService接口
var _ IMapService = (*MapBaidu)(nil)

type MapBaidu struct {
	ak string
}

func NewMapBaidu(ak string) *MapBaidu {
	return &MapBaidu{
		ak: ak, //百度地图API密钥
	}
}

// AdministrativeRegionQuery implements IMapService.
func (m *MapBaidu) AdministrativeRegionQuery(ctx context.Context, req *AdministrativeRegionRequest) (*AdministrativeRegionResponse, error) {
	panic("unimplemented")
}

// PlaceSearch implements IMapService.
func (m *MapBaidu) PlaceSearch(ctx context.Context, req *PlaceSearchRequest) (*PlaceSearchResponse, error) {
	panic("unimplemented")
}

// RoutePlanning implements IMapService.
func (m *MapBaidu) RoutePlanning(ctx context.Context, req *RoutePlanningRequest) (*RoutePlanningResponse, error) {
	panic("unimplemented")
}

// Geocoding 地理编码（地址转坐标）
// 百度地图 API 文档: https://lbsyun.baidu.com/index.php?title=webapi/guide/webservice-geocoding
func (m *MapBaidu) Geocoding(ctx context.Context, req *GeocodingRequest) (*GeocodingResponse, error) {
	if req.Address == "" {
		return nil, fmt.Errorf("地址不能为空")
	}

	// 构建请求 URL
	apiURL := "https://api.map.baidu.com/geocoding/v3/"
	params := url.Values{}
	params.Set("address", req.Address)
	params.Set("output", "json")
	params.Set("ak", m.ak)

	if req.City != "" {
		params.Set("city", req.City)
	}

	fullURL := apiURL + "?" + params.Encode()

	// 创建 HTTP 请求
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	httpReq, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 发送请求
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 解析 JSON 响应
	var baiduResp struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
		Result  struct {
			Location struct {
				Lng       float64 `json:"lng"`
				Lat       float64 `json:"lat"`
			} `json:"location"`
			Precise     int    `json:"precise"`      // 1:精确匹配, 0:非精确匹配
			Confidence  int    `json:"confidence"`   // 可信度
			Comprehension int  `json:"comprehension"` // 地址理解程度
			Level       string `json:"level"`        // 地址类型
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &baiduResp); err != nil {
		return nil, fmt.Errorf("解析 JSON 失败: %w, 响应: %s", err, string(body))
	}

	// 检查状态码
	if baiduResp.Status != 0 {
		return &GeocodingResponse{
			Status:  "error",
			Message: fmt.Sprintf("百度地图 API 错误: %d - %s", baiduResp.Status, baiduResp.Message),
		}, nil
	}

	// 构造响应
	response := &GeocodingResponse{
		Status:  "ok",
		Message: "success",
		Locations: []GeocodingResult{
			{
				Latitude:  baiduResp.Result.Location.Lat,
				Longitude: baiduResp.Result.Location.Lng,
				Precise:   baiduResp.Result.Precise == 1,
				Level:     baiduResp.Result.Level,
			},
		},
	}

	return response, nil
}
