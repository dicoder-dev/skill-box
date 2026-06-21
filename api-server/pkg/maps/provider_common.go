package maps

// =============================================================================
// 地图服务配置与管理器
// 定义地图提供商的配置结构和全局管理器
// 负责提供商的注册、初始化和路由选择
// =============================================================================

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// =============================================================================
// 配置结构体
// =============================================================================

// ProviderConfig 单个地图提供商的配置
// Key: API密钥（必需）
// Secret: API密钥的密钥（部分提供商需要）
// BaseURL: API基础URL（可选，用于覆盖默认URL）
// Language: 默认语言设置
// Region: 默认区域设置
type ProviderConfig struct {
	Key      string // API密钥（必需）
	Secret   string // API密钥的密钥
	BaseURL  string // API基础URL（可选）
	Language string // 默认语言设置
	Region   string // 默认区域设置
}

// RoutingConfig 路由配置
// 定义不同操作类型使用的默认提供商
type RoutingConfig struct {
	DefaultProvider          ProviderName // 默认提供商
	DomesticProvider         ProviderName // 国内提供商
	InternationalProvider    ProviderName // 国际提供商
	PlaceSearchProvider      ProviderName // POI搜索提供商
	DrivingRouteProvider     ProviderName // 驾车路线规划提供商
	TransitRouteProvider     ProviderName // 公交路线规划提供商
	GeocodingProvider        ProviderName // 地理编码提供商
	ReverseGeocodingProvider ProviderName // 逆地理编码提供商
	AdministrativeProvider   ProviderName // 行政区划查询提供商
	DomesticCountryCodes     []string     // 国内国家代码列表
}

// MapConfig 地图服务完整配置
// Timeout: HTTP请求超时时间
// 各提供商的配置
// Routing: 路由配置
type MapConfig struct {
	Timeout  time.Duration // HTTP请求超时时间
	Baidu    ProviderConfig // 百度地图配置
	Amap     ProviderConfig // 高德地图配置
	Google   ProviderConfig // Google地图配置
	Tianditu ProviderConfig // 天地图配置
	Huawei   ProviderConfig // 华为地图配置
	Routing  RoutingConfig // 路由配置

	// 为兼容旧代码保留
	BaiduAk string // 旧版百度地图AK
}

// =============================================================================
// Manager管理器
// 负责管理所有注册的地图提供商
// =============================================================================

// Manager 地图服务管理器
// 管理所有注册的地图提供商，并提供路由选择功能
type Manager struct {
	mu        sync.RWMutex // 读写锁，保证并发安全
	providers map[ProviderName]Provider // 已注册的提供商映射
	routing   RoutingConfig // 路由配置
}

var (
	globalManager *Manager // 全局单例管理器
	managerMu     sync.RWMutex // 全局管理器的锁
)

// =============================================================================
// 初始化与管理函数
// =============================================================================

// InitMaps 初始化全局地图服务
// 是项目启动时必须调用的入口函数
// 参数cfg: 地图服务配置，如果为nil则使用默认配置
func InitMaps(cfg *MapConfig) error {
	manager, err := NewManager(cfg)
	if err != nil {
		return err
	}

	managerMu.Lock()
	globalManager = manager
	managerMu.Unlock()
	return nil
}

// NewManager 创建新的地图服务管理器
// 参数cfg: 地图服务配置
// 返回值：Manager实例和可能的错误
func NewManager(cfg *MapConfig) (*Manager, error) {
	if cfg == nil {
		cfg = &MapConfig{}
	}

	// 规范化配置参数
	normalizeMapConfig(cfg)

	// 创建管理器实例
	manager := &Manager{
		providers: make(map[ProviderName]Provider),
		routing:   cfg.Routing,
	}

	// 注册函数：将提供商注册到管理器
	register := func(provider Provider) {
		if provider != nil {
			manager.providers[provider.Name()] = provider
		}
	}

	// 注册各个地图提供商
	register(NewMapBaidu(cfg.Baidu, cfg.Timeout))
	register(NewMapAmap(cfg.Amap, cfg.Timeout))
	register(NewMapGoogle(cfg.Google, cfg.Timeout))
	register(NewMapTianditu(cfg.Tianditu, cfg.Timeout))
	register(NewMapHuawei(cfg.Huawei, cfg.Timeout))

	return manager, nil
}

// normalizeMapConfig 规范化地图配置
// 设置默认值，确保所有配置项都有有效的值
func normalizeMapConfig(cfg *MapConfig) {
	// 设置默认超时时间
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}

	// 兼容旧版百度AK配置
	if cfg.Baidu.Key == "" && cfg.BaiduAk != "" {
		cfg.Baidu.Key = cfg.BaiduAk
	}

	// 设置默认国内国家代码
	if len(cfg.Routing.DomesticCountryCodes) == 0 {
		cfg.Routing.DomesticCountryCodes = []string{"CN", "CHN", "CHINA", "中国", "中华人民共和国"}
	}

	// 设置默认提供商
	if cfg.Routing.DefaultProvider == "" {
		cfg.Routing.DefaultProvider = MapNameTianditu
	}
	if cfg.Routing.DomesticProvider == "" {
		cfg.Routing.DomesticProvider = MapNameTianditu
	}
	if cfg.Routing.InternationalProvider == "" {
		cfg.Routing.InternationalProvider = MapNameGoogle
	}
}

// GetManager 获取全局管理器实例
func GetManager() *Manager {
	managerMu.RLock()
	defer managerMu.RUnlock()
	return globalManager
}

// =============================================================================
// 提供商注册与管理方法
// =============================================================================

// RegisterProvider 注册地图提供商
// 参数provider: 要注册的提供商实例
func (m *Manager) RegisterProvider(provider Provider) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.providers[provider.Name()] = provider
}

// RegisterMapService 注册地图服务（支持旧版接口）
// 参数name: 提供商名称
// 参数svc: 地图服务实例
func RegisterMapService(name string, svc IMapService) {
	manager := ensureManager()
	if provider, ok := svc.(Provider); ok {
		manager.RegisterProvider(provider)
		return
	}

	// 如果服务不是Provider类型，则使用适配器包装
	manager.RegisterProvider(&legacyProviderAdapter{name: ProviderName(name), svc: svc})
}

// GetMapInstance 获取地图服务实例
// 参数name: 提供商名称
// 返回值：IMapService接口和可能的错误
func GetMapInstance(name ProviderName) (IMapService, error) {
	manager := GetManager()
	if manager == nil {
		return nil, errors.New("地图服务未初始化")
	}

	manager.mu.RLock()
	defer manager.mu.RUnlock()

	provider, ok := manager.providers[name]
	if !ok || provider == nil {
		return nil, errors.New("地图服务未初始化: " + string(name))
	}
	if !provider.IsAvailable() {
		return nil, fmt.Errorf("地图服务未配置 key: %s", name)
	}
	return provider, nil
}

// GetProvider 获取地图提供商
// 参数name: 提供商名称
// 返回值：Provider接口和可能的错误
func (m *Manager) GetProvider(name ProviderName) (Provider, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	provider, ok := m.providers[name]
	if !ok || provider == nil {
		return nil, fmt.Errorf("地图服务未初始化: %s", name)
	}
	if !provider.IsAvailable() {
		return nil, fmt.Errorf("地图服务不可用: %s", name)
	}
	return provider, nil
}

// =============================================================================
// 路由选择方法
// 根据操作类型和选项自动选择合适的提供商
// =============================================================================

// ResolveProvider 根据操作类型和请求选项解析出合适的提供商
// 参数operation: 操作类型（如PlaceSearch、Geocoding等）
// 参数opts: 请求选项
// 返回值：Provider接口和可能的错误
func (m *Manager) ResolveProvider(operation OperationType, opts RequestOptions) (Provider, error) {
	// 1. 如果明确指定了提供商，直接返回
	if opts.Provider != "" {
		return m.GetProvider(opts.Provider)
	}

	// 2. 根据操作类型选择对应的默认提供商
	if provider := m.routingProviderForOperation(operation); provider != "" {
		if svc, err := m.GetProvider(provider); err == nil {
			return svc, nil
		}
	}

	// 3. 根据区域（国内/国际）选择提供商
	if provider := m.routingProviderForRegion(opts); provider != "" {
		if svc, err := m.GetProvider(provider); err == nil {
			return svc, nil
		}
	}

	// 4. 使用默认提供商
	if m.routing.DefaultProvider != "" {
		if svc, err := m.GetProvider(m.routing.DefaultProvider); err == nil {
			return svc, nil
		}
	}

	return nil, errors.New("没有可用的地图服务提供商")
}

// PlaceSearch POI地点搜索（通过Manager代理）
// 自动选择合适的提供商并调用其PlaceSearch方法
func (m *Manager) PlaceSearch(ctx context.Context, req *PlaceSearchRequest) (*PlaceSearchResponse, error) {
	provider, err := m.ResolveProvider(OperationPlaceSearch, req.Options)
	if err != nil {
		return nil, err
	}
	return provider.PlaceSearch(ctx, req)
}

// RoutePlanning 线路规划（通过Manager代理）
// 自动选择合适的提供商并调用其RoutePlanning方法
func (m *Manager) RoutePlanning(ctx context.Context, req *RoutePlanningRequest) (*RoutePlanningResponse, error) {
	op := OperationDrivingRoute
	if req.Mode == TravelModeTransit {
		op = OperationTransitRoute
	}
	provider, err := m.ResolveProvider(op, req.Options)
	if err != nil {
		return nil, err
	}
	return provider.RoutePlanning(ctx, req)
}

// Geocoding 地理编码（通过Manager代理）
// 自动选择合适的提供商并调用其Geocoding方法
func (m *Manager) Geocoding(ctx context.Context, req *GeocodingRequest) (*GeocodingResponse, error) {
	provider, err := m.ResolveProvider(OperationGeocoding, req.Options)
	if err != nil {
		return nil, err
	}
	return provider.Geocoding(ctx, req)
}

// ReverseGeocoding 逆地理编码（通过Manager代理）
// 自动选择合适的提供商并调用其ReverseGeocoding方法
func (m *Manager) ReverseGeocoding(ctx context.Context, req *ReverseGeocodingRequest) (*ReverseGeocodingResponse, error) {
	provider, err := m.ResolveProvider(OperationReverseGeocoding, req.Options)
	if err != nil {
		return nil, err
	}
	return provider.ReverseGeocoding(ctx, req)
}

// AdministrativeRegionQuery 行政区划查询（通过Manager代理）
// 自动选择合适的提供商并调用其AdministrativeRegionQuery方法
func (m *Manager) AdministrativeRegionQuery(ctx context.Context, req *AdministrativeRegionRequest) (*AdministrativeRegionResponse, error) {
	provider, err := m.ResolveProvider(OperationAdministrative, req.Options)
	if err != nil {
		return nil, err
	}
	return provider.AdministrativeRegionQuery(ctx, req)
}

// =============================================================================
// 内部��助��法
// =============================================================================

// routingProviderForOperation 根据操作类型获取对应的默认提供商
func (m *Manager) routingProviderForOperation(operation OperationType) ProviderName {
	switch operation {
	case OperationPlaceSearch:
		return m.routing.PlaceSearchProvider
	case OperationDrivingRoute:
		return m.routing.DrivingRouteProvider
	case OperationTransitRoute:
		return m.routing.TransitRouteProvider
	case OperationGeocoding:
		return m.routing.GeocodingProvider
	case OperationReverseGeocoding:
		return m.routing.ReverseGeocodingProvider
	case OperationAdministrative:
		return m.routing.AdministrativeProvider
	default:
		return ""
	}
}

// routingProviderForRegion 根据区域（国内/国际）获取对应的默认提供商
func (m *Manager) routingProviderForRegion(opts RequestOptions) ProviderName {
	isDomestic := false
	if opts.IsDomestic != nil {
		// 如果明确指定了国内/国际
		isDomestic = *opts.IsDomestic
	} else if opts.CountryCode != "" {
		// 根据国家代码判断
		target := strings.ToUpper(strings.TrimSpace(opts.CountryCode))
		for _, item := range m.routing.DomesticCountryCodes {
			if strings.ToUpper(strings.TrimSpace(item)) == target {
				isDomestic = true
				break
			}
		}
	}

	if isDomestic {
		return m.routing.DomesticProvider
	}
	return m.routing.InternationalProvider
}

// ensureManager 确保管理器已初始化
func ensureManager() *Manager {
	manager := GetManager()
	if manager != nil {
		return manager
	}

	// 如果未初始化，创建一个空的管理器
	manager = &Manager{
		providers: make(map[ProviderName]Provider),
	}

	managerMu.Lock()
	if globalManager == nil {
		globalManager = manager
	}
	manager = globalManager
	managerMu.Unlock()
	return manager
}

// =============================================================================
// 旧版Provider适配器
// 用于兼容实现IMapService但未实现Provider接口的服务
// =============================================================================

// legacyProviderAdapter 旧版服务适配器
// 将实现了IMapService但未实现Provider接口的服务适配为Provider
type legacyProviderAdapter struct {
	name ProviderName // 提供商名称
	svc  IMapService // 实际的服务实现
}

func (l *legacyProviderAdapter) Name() ProviderName { return l.name }
func (l *legacyProviderAdapter) IsAvailable() bool  { return l.svc != nil }

// RoutePlanning 实现Provider接口的RoutePlanning方法
func (l *legacyProviderAdapter) RoutePlanning(ctx context.Context, req *RoutePlanningRequest) (*RoutePlanningResponse, error) {
	return l.svc.RoutePlanning(ctx, req)
}

// PlaceSearch 实现Provider接口的PlaceSearch方法
func (l *legacyProviderAdapter) PlaceSearch(ctx context.Context, req *PlaceSearchRequest) (*PlaceSearchResponse, error) {
	return l.svc.PlaceSearch(ctx, req)
}

// AdministrativeRegionQuery 实现Provider接口的AdministrativeRegionQuery方法
func (l *legacyProviderAdapter) AdministrativeRegionQuery(ctx context.Context, req *AdministrativeRegionRequest) (*AdministrativeRegionResponse, error) {
	return l.svc.AdministrativeRegionQuery(ctx, req)
}

// Geocoding 实现Provider接口的Geocoding方法
func (l *legacyProviderAdapter) Geocoding(ctx context.Context, req *GeocodingRequest) (*GeocodingResponse, error) {
	return l.svc.Geocoding(ctx, req)
}

// ReverseGeocoding 实现Provider接口的ReverseGeocoding方法
func (l *legacyProviderAdapter) ReverseGeocoding(ctx context.Context, req *ReverseGeocodingRequest) (*ReverseGeocodingResponse, error) {
	return l.svc.ReverseGeocoding(ctx, req)
}