// GlobalHotKeyManager 全局快捷键句柄(跨平台抽象)。
//
// Register / Unregister 由 globalhotkey_darwin.go (cgo Carbon) 或
// globalhotkey_other.go (stub) 各自实现。
package desktop

import "sync"

// GlobalHotKeyManager 全局快捷键句柄。
type GlobalHotKeyManager struct {
	mu        sync.Mutex
	installed bool
}

// NewGlobalHotKeyManager 构造句柄(由平台特定文件实现)。
var NewGlobalHotKeyManager func() *GlobalHotKeyManager
