package ratelimiter

import (
	"sync"
)

// ConfigRefreshCallback 配置刷新回调函数类型
type ConfigRefreshCallback func()

// ConfigManager 配置管理器
type ConfigManager struct {
	callbacks []ConfigRefreshCallback
	mutex     sync.RWMutex
}

var globalConfigManager = &ConfigManager{
	callbacks: make([]ConfigRefreshCallback, 0),
}

// RegisterRefreshCallback 注册配置刷新回调
func RegisterRefreshCallback(callback ConfigRefreshCallback) {
	globalConfigManager.mutex.Lock()
	defer globalConfigManager.mutex.Unlock()

	globalConfigManager.callbacks = append(globalConfigManager.callbacks, callback)
}

// NotifyConfigRefresh 通知配置刷新
func NotifyConfigRefresh() {
	globalConfigManager.mutex.RLock()
	callbacks := make([]ConfigRefreshCallback, len(globalConfigManager.callbacks))
	copy(callbacks, globalConfigManager.callbacks)
	globalConfigManager.mutex.RUnlock()

	// 执行所有回调
	for _, callback := range callbacks {
		if callback != nil {
			callback()
		}
	}
}
