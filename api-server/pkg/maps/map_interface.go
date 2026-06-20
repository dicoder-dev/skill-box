package maps

import (
	"context"
)

// IMapService 地图服务接口
type IMapService interface {
	// RoutePlanning 线路规划
	RoutePlanning(ctx context.Context, req *RoutePlanningRequest) (*RoutePlanningResponse, error)

	// PlaceSearch 地点检索
	PlaceSearch(ctx context.Context, req *PlaceSearchRequest) (*PlaceSearchResponse, error)

	// AdministrativeRegionQuery 行政区域查询
	AdministrativeRegionQuery(ctx context.Context, req *AdministrativeRegionRequest) (*AdministrativeRegionResponse, error)

	// Geocoding 地理编码（地址转坐标）
	Geocoding(ctx context.Context, req *GeocodingRequest) (*GeocodingResponse, error)
}

// RoutePlanningRequest 线路规划请求
type RoutePlanningRequest struct {
	Origin      string `json:"origin"`       // 起点坐标，格式：经度,纬度
	Destination string `json:"destination"`  // 终点坐标，格式：经度,纬度
	//交通方式：car(汽车), bike(自行车), walk(步行)
	Strategy    string `json:"strategy"`     // 路线策略：fastest(最快), shortest(最短), avoid_highways(避开高速)
	Waypoints   string `json:"waypoints"`    // 途经点，多个点用|分隔，格式：经度,纬度|经度,纬度
}

// RoutePlanningResponse 线路规划响应
type RoutePlanningResponse struct {
	Status     string      `json:"status"`      // 状态码
	Message    string      `json:"message"`     // 状态信息
	Distance   float64     `json:"distance"`    // 总距离(米)
	Duration   int64       `json:"duration"`    // 总耗时(秒)
	Routes     []Route     `json:"routes"`      // 路线信息
}

// Route 路线信息
type Route struct {
	Distance   float64     `json:"distance"`   // 距离(米)
	Duration   int64       `json:"duration"`   // 耗时(秒)
	Steps      []RouteStep `json:"steps"`      // 路线步骤
	Polyline   string      `json:"polyline"`   // 路线坐标串
}

// RouteStep 路线步骤
type RouteStep struct {
	Instruction string  `json:"instruction"` // 导航指令
	Distance    float64 `json:"distance"`    // 距离(米)
	Duration    int64   `json:"duration"`    // 耗时(秒)
	Polyline    string  `json:"polyline"`    // 坐标串
}

// PlaceSearchRequest 地点检索请求
type PlaceSearchRequest struct {
	Query     string  `json:"query"`      // 搜索关键词
	Location  string  `json:"location"`   // 中心点坐标，格式：经度,纬度
	Radius    int     `json:"radius"`     // 搜索半径(米)，默认5000
	PageSize  int     `json:"page_size"`  // 每页数量，默认20
	PageIndex int     `json:"page_index"` // 页码，从1开始
	Types     string  `json:"types"`      // 地点类型，多个用|分隔
	City      string  `json:"city"`       // 城市限制
}

// PlaceSearchResponse 地点检索响应
type PlaceSearchResponse struct {
	Status    string  `json:"status"`     // 状态码
	Message   string  `json:"message"`    // 状态信息
	Total     int     `json:"total"`      // 总数量
	Places    []Place `json:"places"`     // 地点列表
}

// Place 地点信息
type Place struct {
	ID          string  `json:"id"`           // 地点ID
	Name        string  `json:"name"`         // 地点名称
	Address     string  `json:"address"`      // 详细地址
	Location    string  `json:"location"`     // 坐标，格式：经度,纬度
	Distance    float64 `json:"distance"`     // 距离中心点距离(米)
	Type        string  `json:"type"`         // 地点类型
	Phone       string  `json:"phone"`        // 电话
	Rating      float64 `json:"rating"`       // 评分
	PhotoURL    string  `json:"photo_url"`    // 照片URL
	BusinessHours string `json:"business_hours"` // 营业时间
}

// AdministrativeRegionRequest 行政区域查询请求
type AdministrativeRegionRequest struct {
	Location string `json:"location"` // 坐标，格式：经度,纬度
	Level    string `json:"level"`    // 查询级别：country(国家), province(省份), city(城市), district(区县)
}

// AdministrativeRegionResponse 行政区域查询响应
type AdministrativeRegionResponse struct {
	Status   string           `json:"status"`   // 状态码
	Message  string           `json:"message"`  // 状态信息
	Regions  []AdministrativeRegion `json:"regions"` // 行政区域信息
}

// AdministrativeRegion 行政区域信息
type AdministrativeRegion struct {
	Code     string `json:"code"`      // 行政区划代码
	Name     string `json:"name"`      // 行政区划名称
	Level    string `json:"level"`     // 级别：country, province, city, district
	Parent   string `json:"parent"`    // 上级行政区划代码
	Location string `json:"location"`  // 中心点坐标，格式：经度,纬度
	Boundary string `json:"boundary"`  // 边界坐标串
}

// GeocodingRequest 地理编码请求（地址转坐标）
type GeocodingRequest struct {
	Address string `json:"address"` // 地址关键词，如"北京天安门"
	City    string `json:"city"`    // 城市限制，如"北京市"
}

// GeocodingResponse 地理编码响应
type GeocodingResponse struct {
	Status    string            `json:"status"`     // 状态码
	Message   string            `json:"message"`    // 状态信息
	Locations []GeocodingResult `json:"locations"`  // 地理编码结果列表
}

// GeocodingResult 地理编码结果
type GeocodingResult struct {
	Latitude  float64 `json:"latitude"`   // 纬度
	Longitude float64 `json:"longitude"`  // 经度
	Precise   bool    `json:"precise"`    // 位置的附加信息，是否精确匹配
	Level     string  `json:"level"`      // 地址类型，如"道路"、"门牌号"、"POI"等
	Province  string  `json:"province"`   // 省份
	City      string  `json:"city"`       // 城市
	District  string  `json:"district"`   // 区县
}
