package skillmarket

import (
	"fmt"
	"sort"
	"sync"
)

// Registry adapter 注册表:sourceID -> MarketAdapter。
//
// 启动时由各 adapter 子包在自己的 init() 里调用 defaultRegistry.Register;
// 调用方通过 Get / MustGet 取出。
type Registry struct {
	mu sync.RWMutex
	m  map[string]MarketAdapter
}

var defaultRegistry = &Registry{m: make(map[string]MarketAdapter)}

// Register 注册到默认 registry。
func Register(a MarketAdapter) { defaultRegistry.Register(a) }

// Get 取出。
func Get(sourceID string) (MarketAdapter, bool) { return defaultRegistry.Get(sourceID) }

// MustGet Get 的 panic 版。
func MustGet(sourceID string) MarketAdapter { return defaultRegistry.MustGet(sourceID) }

// All 已注册 adapter 列表(按 SourceID 排序)。
func All() []MarketAdapter { return defaultRegistry.All() }

// Register 注册到指定 registry。
func (r *Registry) Register(a MarketAdapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.m == nil {
		r.m = make(map[string]MarketAdapter)
	}
	if _, exists := r.m[a.SourceID()]; exists {
		panic(fmt.Sprintf("skillmarket: duplicate registration for %q", a.SourceID()))
	}
	r.m[a.SourceID()] = a
}

// Get 取出。
func (r *Registry) Get(sourceID string) (MarketAdapter, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.m[sourceID]
	return a, ok
}

// MustGet panic 版。
func (r *Registry) MustGet(sourceID string) MarketAdapter {
	a, ok := r.Get(sourceID)
	if !ok {
		panic(fmt.Sprintf("skillmarket: adapter %q not registered", sourceID))
	}
	return a
}

// All 全部。
func (r *Registry) All() []MarketAdapter {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]MarketAdapter, 0, len(r.m))
	for _, a := range r.m {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SourceID() < out[j].SourceID() })
	return out
}
