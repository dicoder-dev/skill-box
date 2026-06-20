package maps

import (
	"errors"
	"sync"
)

// 地图服务注册中心与初始化
//
// 目标：
// 1) 通过 InitMaps 一次性初始化各个地图服务的单例
// 2) 通过 GetMapService("baidu") 等按名称获取 IMapService
// 3) 也可在运行时通过 RegisterMapService 动态注册替换

const (
	// MapNameBaidu 百度地图服务名称
	MapNameBaidu = "baidu"
)

var (
	mapRegistryMu sync.RWMutex
	mapInstances   = make(map[string]IMapService)
)

// MapConfig 通用初始化配置
// 约定：key 为地图名称，例如 "baidu"
// value 为对应的初始化配置字典，例如 {"ak": "xxxxx"}
type MapConfig struct {
	BaiduAk string
}

// InitMaps 使用配置初始化并注册各个地图服务的单例。
// 示例：
// InitMaps(MapConfig{
//     BaiduAk: "your_baidu_ak",
// })
func InitMaps(cfg *MapConfig) error {
	if cfg == nil {
		return nil
	}

	// 初始化百度地图
	if cfg.BaiduAk != "" {
		 RegisterMapService(MapNameBaidu, NewMapBaidu(cfg.BaiduAk))
	}


	return nil
}

// RegisterMapService 注册/覆盖一个地图服务单例
func RegisterMapService(name string, svc IMapService) {
	mapRegistryMu.Lock()
	mapInstances[name] = svc
	mapRegistryMu.Unlock()
}

// GetMapInstance 获取已注册的地图服务单例
func GetMapInstance(name string) (IMapService, error) {
	mapRegistryMu.RLock()
	svc, ok := mapInstances[name]
	mapRegistryMu.RUnlock()
	if !ok || svc == nil {
		return nil, errors.New("地图服务未初始化: " + name)
	}
	return svc, nil
}
