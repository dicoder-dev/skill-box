package skilladapter

import (
	"fmt"
	"sort"
	"sync"
)

// Registry adapter 注册表:toolID -> Adapter。
//
// 启动时由各 adapter 子包在自己的 init() 里调用 defaultRegistry.Register;
// 调用方通过 Get / MustGet 取出。v1 不做热插拔,所有 adapter 编译期已知。
type Registry struct {
	mu sync.RWMutex
	m  map[string]Adapter
}

var defaultRegistry = &Registry{m: make(map[string]Adapter)}

// DefaultRegistry 返回全局默认 registry 指针(供 Applier 等"忘了注入"的场景兜底)。
// 不要在 adapter 业务代码里使用 — 业务侧应通过 Service.WithAdapterRegistry 注入;
// 这里只暴露给那些"生产路径就是用默认"的初始化器,避免 nil panic。
func DefaultRegistry() *Registry { return defaultRegistry }

// Register 把 adapter 注册到默认 registry。同名重复注册会 panic(早期暴露重复实现)。
func Register(a Adapter) { defaultRegistry.Register(a) }

// Get 取出 toolID 对应的 adapter;不存在返回 (nil, false)。
func Get(toolID string) (Adapter, bool) { return defaultRegistry.Get(toolID) }

// MustGet 同 Get,缺失时 panic。用于编译期已知一定存在的代码路径。
func MustGet(toolID string) Adapter { return defaultRegistry.MustGet(toolID) }

// All 返回已注册 adapter 的列表(按 ToolID 排序,保证调用方顺序稳定)。
func All() []Adapter { return defaultRegistry.All() }

// Register 把 adapter 注册到当前 registry。同名重复注册会 panic。
func (r *Registry) Register(a Adapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.m == nil {
		r.m = make(map[string]Adapter)
	}
	if _, exists := r.m[a.ToolID()]; exists {
		panic(fmt.Sprintf("skilladapter: duplicate registration for %q", a.ToolID()))
	}
	r.m[a.ToolID()] = a
}

// Get 取出 toolID 对应的 adapter;不存在返回 (nil, false)。
func (r *Registry) Get(toolID string) (Adapter, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.m[toolID]
	return a, ok
}

// MustGet 同 Get,缺失时 panic。
func (r *Registry) MustGet(toolID string) Adapter {
	a, ok := r.Get(toolID)
	if !ok {
		panic(fmt.Sprintf("skilladapter: adapter %q not registered", toolID))
	}
	return a
}

// All 返回已注册 adapter 的列表(按 ToolID 排序)。
func (r *Registry) All() []Adapter {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Adapter, 0, len(r.m))
	for _, a := range r.m {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ToolID() < out[j].ToolID() })
	return out
}

// Reload 用给定的 adapter 列表整体替换 registry 的内容。
//
// 2026-06-30 二改:工具元数据从 yaml 改 DB 后,业务层(stool Service)改完
// 工具需要重新加载 — 用 Reload(adapters) 整体替换一次,包括删除用户
// 在 UI 删掉的工具。比逐个 Add/Remove 简单,且保证"最终状态就是给定列表"。
//
// 失败语义:无错误(纯内存替换);tool_id 重复由调用方在 spec 加载阶段
// (LoadAllFromDB) 兜底,这里不去重。
func (r *Registry) Reload(adapters []Adapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.m = make(map[string]Adapter, len(adapters))
	for _, a := range adapters {
		r.m[a.ToolID()] = a
	}
}
