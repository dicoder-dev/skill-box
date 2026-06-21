package maps

// =============================================================================
// 地图服务接口与数据类型定义
// 定义了统一的地图服务接口IMapService和各 Provider 需要实现的接口
// 同时定义了所有请求和响应的数据结构
// =============================================================================

import "context"

type ProviderName string

// =============================================================================
// 提供商名称常量
// 支持的地图服务提供商
// =============================================================================

const (
	MapNameBaidu    ProviderName = "baidu"    // 百度地图
	MapNameAmap     ProviderName = "amap"     // 高德地图
	MapNameGoogle   ProviderName = "google"   // Google Maps
	MapNameTianditu ProviderName = "tianditu" // 天地图
	MapNameHuawei   ProviderName = "huawei"   // 华为地图
)

// =============================================================================
// 操作类型常量
// 定义地图服务支持的各种操作类型
// =============================================================================

type OperationType string

const (
	OperationPlaceSearch        OperationType = "place_search"          // POI地点搜索
	OperationDrivingRoute       OperationType = "driving_route"        // 驾车路线规划
	OperationTransitRoute       OperationType = "transit_route"        // 公交路线规划
	OperationGeocoding         OperationType = "geocoding"           // 地理编码（地址转坐标）
	OperationReverseGeocoding   OperationType = "reverse_geocoding"   // 逆地理编码（坐标转地址）
	OperationAdministrative   OperationType = "administrative"     // 行政区划查询
)

// =============================================================================
// 出行模式常量
// 路线规划时支持的出行方式
// =============================================================================

type TravelMode string

const (
	TravelModeDriving TravelMode = "driving" // 驾车
	TravelModeTransit TravelMode = "transit" // 公交/公共交通
	TravelModeWalking TravelMode = "walking" // 步行
)

// =============================================================================
// 请求选项结构体
// 用于指定请求的特定选项，如提供商偏好、国内/国外等
// =============================================================================

// RequestOptions 请求选项，用于指定请求的偏好配置
// CountryCode: 国家代码，如"CN"表示中国
// IsDomestic: 是否为国内请求，nil表示自动判断
// Provider: 偏好使用的地图提供商
type RequestOptions struct {
	Provider    ProviderName `json:"provider"`     // 偏好使用的地图提供商
	CountryCode string       `json:"country_code"` // 国家代码
	IsDomestic  *bool        `json:"is_domestic"`  // 是否为国内请求
}

// IMapService 地图服务接口
// 定义了所有地图服务需要实现的接口方法
// 使用此接口可以统一调用不同地图提供商的服务的
type IMapService interface {
	// RoutePlanning 线路规划
	// 根据起终点坐标计算驾车或公交路线
	// 参数：RoutePlanningRequest，包含起终点、策略等信息
	// 返回：RoutePlanningResponse，包含路线距离、耗时、详细步骤等
	RoutePlanning(ctx context.Context, req *RoutePlanningRequest) (*RoutePlanningResponse, error)

	// PlaceSearch POI地点搜索
	// 根据关键字搜索地点或POI
	// 参数：PlaceSearchRequest，包含搜索关键字���城市、位置等信息
	// 返回：PlaceSearchResponse，包含搜索结果列表
	PlaceSearch(ctx context.Context, req *PlaceSearchRequest) (*PlaceSearchResponse, error)

	// AdministrativeRegionQuery 行政区划查询
	// 根据位置或关键字查询行政区划信息
	// 参数：AdministrativeRegionRequest
	// 返回：AdministrativeRegionResponse
	AdministrativeRegionQuery(ctx context.Context, req *AdministrativeRegionRequest) (*AdministrativeRegionResponse, error)

	// Geocoding 地理编码
	// 将地址转换为坐标（经纬度）
	// 参数：GeocodingRequest，包含地址信息
	// 返回：GeocodingResponse，包含转换后的坐标
	Geocoding(ctx context.Context, req *GeocodingRequest) (*GeocodingResponse, error)

	// ReverseGeocoding 逆地理编码
	// 将坐标（经纬度）转换为地址
	// 参数：ReverseGeocodingRequest，包含坐标信息
	// 返回：ReverseGeocodingResponse，包含解析后的地址信息
	ReverseGeocoding(ctx context.Context, req *ReverseGeocodingRequest) (*ReverseGeocodingResponse, error)
}

// Provider 地图平台适配器接口
// 继承自IMapService，并添加了Name和IsAvailable方法
// 每个具体的地图提供商（如百度、高德等）都需要实现此接口
type Provider interface {
	IMapService
	Name() ProviderName       // 返回提供商名称
	IsAvailable() bool        // 检查提供商是否可用（是否配置了API密钥）
}

// =============================================================================
// 路线规划相关数据结构
// =============================================================================

// RoutePlanningRequest 线路规划请求
// Origin: 起点坐标，格式：经度,纬度（如"116.403988,39.914266"）
// Destination: 终点坐标，格式同上
// Strategy: 路线策略，可选值：
//   - fastest: 最快路线（默认）
//   - shortest: 最短路线
//   - avoid_highways: 避开高速
// Waypoints: 途经点，多个点用|分隔，格式：经度,纬度|经度,纬度
// Mode: 出行方式，driving/transit/walking
// City: 公交规划时优先使用的城市
type RoutePlanningRequest struct {
	Origin      string         `json:"origin"`        // 起点坐标，格式：经度,纬度
	Destination string         `json:"destination"`  // 终点坐标，格式：经度,纬度
	Strategy    string         `json:"strategy"`     // 路线策略：fastest(最快), shortest(最短), avoid_highways(避开高速)
	Waypoints   string         `json:"waypoints"`   // 途经点，多个点用|分隔，格式：经度,纬度|经度,纬度
	Mode        TravelMode     `json:"mode"`        // 出行方式：driving/transit/walking
	City        string         `json:"city"`        // 公交规划优先使用的城市
	Options     RequestOptions `json:"options"`    // 请求选项
}

// RoutePlanningResponse 线路规划响应
// Status: 响应状态，"ok"表示成功
// Message: 状态消息
// Provider: 返回结果的地图提供商
// Distance: 总距离（单位：米）
// Duration: 总耗时（单位：秒）
// Routes: 规划出的路线列表
type RoutePlanningResponse struct {
	Status      string      `json:"status"`      // 响应状态
	Message     string      `json:"message"`     // 状态消息
	Provider    string      `json:"provider"`    // 返回结果的地图提供商
	Distance    float64     `json:"distance"`   // 总距离（单位：米）
	Duration    int64       `json:"duration"`    // 总耗时（单位：秒）
	Routes      []Route     `json:"routes"`      // 规划出的路线列表
	RawResponse interface{} `json:"raw_response,omitempty"` // 原始响应数据
}

// Route 单条路线信息
// Distance: 此段距离（单位：米）
// Duration: 此段耗时（单位：秒）
// Steps: 路线步骤列表
// Polyline: 路线编码（用于在地图上绘制）
// Summary: 路线摘要信息
type Route struct {
	Distance float64     `json:"distance"` // 此段距离（单位：米）
	Duration int64       `json:"duration"` // 此段耗时（单位：秒）
	Steps    []RouteStep `json:"steps"`    // 路线步骤列表
	Polyline string      `json:"polyline"` // 路线编码
	Summary  string      `json:"summary"` // 路线摘要信息
}

// RouteStep 路线步骤
// Instruction: 导航指示（如"右转进入中关村大街"）
// Distance: 此步骤距离（单位：米）
// Duration: 此步骤耗时（单位：秒）
// Polyline: 此步骤的路线编码
type RouteStep struct {
	Instruction string  `json:"instruction"` // 导航指示
	Distance    float64 `json:"distance"`   // 此步骤距离（单位：米）
	Duration    int64   `json:"duration"`   // 此步骤耗时（单位：秒）
	Polyline    string  `json:"polyline"`   // 此步骤的路线编码
}

// =============================================================================
// POI地点搜索相关数据结构
// =============================================================================

// PlaceSearchRequest POI地点搜索请求
// Query: 搜索关键字
// Location: 中心点坐标，用于附近搜索
// Radius: 搜索半径（单位：米）
// PageSize: 每页返回数量
// PageIndex: 页码索引（从1开始）
// Types: POI类型过滤
// City: 搜索城市
type PlaceSearchRequest struct {
	Query     string         `json:"query"`      // 搜索关键字
	Location  string         `json:"location"`   // 中心点坐标，用于附近搜索
	Radius    int            `json:"radius"`    // 搜索半径（单位：米）
	PageSize  int            `json:"page_size"` // 每页返回数量
	PageIndex int            `json:"page_index"` // 页码索引（从1开始）
	Types     string         `json:"types"`     // POI类型过滤
	City      string         `json:"city"`       // 搜索城市
	Options   RequestOptions `json:"options"`  // 请求选项
}

// PlaceSearchResponse POI地点搜索响应
type PlaceSearchResponse struct {
	Status      string      `json:"status"`      // 响应状态
	Message     string      `json:"message"`     // 状态消息
	Provider    string      `json:"provider"`    // 返回结果的地图提供商
	Total       int         `json:"total"`       // 总结果数
	Places      []Place     `json:"places"`     // 地点列表
	RawResponse interface{} `json:"raw_response,omitempty"` // 原始响应数据
}

// Place 单个地点信息
type Place struct {
	ID             string  `json:"id"`              // 地点ID
	Name           string  `json:"name"`            // 地点名称
	Address        string  `json:"address"`         // 详细地址
	Location       string  `json:"location"`        // 坐标，格式：经度,纬度
	Distance       float64 `json:"distance"`       // 距离（单位：米）
	PoiType        string  `json:"poi_type"`        // POI类型
	TypeCode       string  `json:"type_code"`       // 类型代码
	Type           string  `json:"type"`            // 类型名称
	Phone          string  `json:"phone"`          // 电话
	Rating         float64 `json:"rating"`          // 评分
	PhotoURL       string  `json:"photo_url"`      // 照片URL
	BusinessHours  string  `json:"business_hours"` // 营业时间
	Province      string  `json:"province"`        // 省份
	ProvinceCode  string  `json:"province_code"`  // 省份代码
	City          string  `json:"city"`            // 城市
	CityCode      string  `json:"city_code"`       // 城市代码
	District      string  `json:"district"`       // 区县
	DistrictCode  string  `json:"district_code"`  // 区县代码
}

// =============================================================================
// 行政区划查询相关数据结构
// =============================================================================

// AdministrativeRegionRequest 行政区划查询请求
// Location: 位置坐标，用于查找所在行政区划
// Level: 行政区划级别（如province、city、district）
// Keyword: 关键字，用于按名称搜索行政区划
type AdministrativeRegionRequest struct {
	Location string         `json:"location"` // 位置坐标，用于查找所在行政区划
	Level    string         `json:"level"`   // 行政区划级别（如province、city、district）
	Keyword  string         `json:"keyword"` // 关键字，用于按名称搜索行政区划
	Options  RequestOptions `json:"options"` // 请求选项
}

// AdministrativeRegionResponse 行政区划查询响应
type AdministrativeRegionResponse struct {
	Status      string                 `json:"status"`      // 响应状态
	Message     string                 `json:"message"`     // 状态消息
	Provider    string                 `json:"provider"`    // 返回结果的地图提供商
	Regions     []AdministrativeRegion `json:"regions"`    // 行政区划列表
	RawResponse interface{}            `json:"raw_response,omitempty"` // 原始响应数据
}

// AdministrativeRegion 单个行政区划信息
type AdministrativeRegion struct {
	Code     string `json:"code"`     // 行政区划代码
	Name     string `json:"name"`     // 行政区划名称
	Level    string `json:"level"`     // 级别（province/city/district等）
	Parent   string `json:"parent"`   // 父级代码
	Location string `json:"location"` // 中心点坐标
	Boundary string `json:"boundary"` // 边界坐标（用于绘制区域）
}

// =============================================================================
// 地理编码相关数据结构
// =============================================================================

// GeocodingRequest 地理编码请求（地址转坐标）
// Address: 地址字符串
// City: 城市名称（用于提高精度）
type GeocodingRequest struct {
	Address string         `json:"address"` // 地址字符串
	City    string         `json:"city"`    // 城市名称（用于提高精度）
	Options RequestOptions `json:"options"` // 请求选项
}

// GeocodingResponse 地理编码响应
type GeocodingResponse struct {
	Status      string            `json:"status"`      // 响应状态
	Message     string             `json:"message"`     // 状态消息
	Provider    string             `json:"provider"`    // 返回结果的地图提供商
	Locations   []GeocodingResult `json:"locations"`   // 解析结果列表
	RawResponse interface{}       `json:"raw_response,omitempty"` // 原始响应数据
}

// GeocodingResult 单个地理编码结果
type GeocodingResult struct {
	Latitude  float64 `json:"latitude"`  // 纬度
	Longitude float64 `json:"longitude"` // 经度
	Precise   bool    `json:"precise"`   // 是否精确匹配
	Level     string  `json:"level"`     // 匹配级别
	Province  string  `json:"province"`  // 省份
	City      string  `json:"city"`      // 城市
	District  string  `json:"district"`  // 区县
	Address   string  `json:"address"`   // 完整地址
}

// =============================================================================
// 逆地理编码相关数据结构
// =============================================================================

// ReverseGeocodingRequest 逆地��编��请求（坐标转地址）
// Location: 坐标，格式：经度,纬度
// Radius: 搜索半径（单位：米）
type ReverseGeocodingRequest struct {
	Location string         `json:"location"` // 坐标，格式：经度,纬度
	Radius   int            `json:"radius"`   // 搜索半径（单位：米）
	Options  RequestOptions `json:"options"`  // 请求选项
}

// ReverseGeocodingResponse 逆地理编码响应
type ReverseGeocodingResponse struct {
	Status      string                 `json:"status"`      // 响应状态
	Message     string                 `json:"message"`     // 状态消息
	Provider    string                 `json:"provider"`    // 返回结果的地图提供商
	Result      ReverseGeocodingResult `json:"result"`      // 解析结果
	RawResponse interface{}            `json:"raw_response,omitempty"` // 原始响应数据
}

// ReverseGeocodingResult 单个逆地理编码结果
type ReverseGeocodingResult struct {
	FormattedAddress string  `json:"formatted_address"` // 格式化地址
	Country          string  `json:"country"`            // 国家
	Province         string  `json:"province"`          // 省份
	City             string  `json:"city"`             // 城市
	District         string  `json:"district"`          // 区县
	Township         string  `json:"township"`        // 乡镇/街道
	Street           string  `json:"street"`           // 道路
	StreetNumber     string  `json:"street_number"`    // 门牌号
	Latitude         float64 `json:"latitude"`         // 纬度
	Longitude        float64 `json:"longitude"`        // 经度
}