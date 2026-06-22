//go:build darwin

// macOS 真全局快捷键实现。
//
// 用 Carbon RegisterEventHotKey 注册一个 OS 级 hotkey,无需窗口聚焦即可响应。
// Carbon 的 EventHandler 跑在 main CFRunLoop 上,跟 Wails 主循环兼容(都跑
// 在 NSApplication 主线程)。handler 通过 export 的 goHotKeyFired 把事件
// 派发回 Go 侧,Go 侧再调用户注册的 callback。
//
// 限制:
//   - macOS 13+ RegisterEventHotKey 需要应用被授予 "辅助功能 (Accessibility)"
//     权限;否则 RegisterEventHotKey 静默失败(回调不触发)。用户首次需要去
//     "系统设置 → 隐私与安全 → 辅助功能"勾选 skill-box。
//   - Carbon 是 deprecated API(自 macOS 10.8),但仍然是注册全局热键的唯一
//     公开稳定 API。SwiftUI / AppKit 没有原生 hotkey 接口。
//   - 本文件只暴露 *GlobalHotKeyManager + combo="Cmd+Shift+S" 一组绑定;
//     V2 扩展需要动态 combo 解析(modifiers + keycode 拆分)。
package desktop

/*
#cgo CFLAGS: -x objective-c -Wno-deprecated-declarations
#cgo LDFLAGS: -framework Carbon -framework Cocoa
#import <Carbon/Carbon.h>

// 注册一个全局热键。
// mods: cmdKey / shiftKey / optionKey / controlKey 任意组合
// keycode: virtual key code(US 键盘布局)
// 返回:非 0 = 注册成功(ref id),0 = 失败(权限/参数错误)
static UInt32 gHotKeyID = 0;
static EventHandlerRef gHandlerRef = NULL;
extern void goHotKeyFired(uint32_t id);

static OSStatus hotKeyHandler(EventHandlerCallRef next, EventRef event, void *userData) {
    EventHotKeyID hkID;
    GetEventParameter(event, kEventParamDirectObject, typeEventHotKeyID,
        NULL, sizeof(hkID), NULL, &hkID);
    if (hkID.id == (UInt32)userData) {
        goHotKeyFired((uint32_t)hkID.id);
    }
    return noErr;
}

static int registerHotKey(UInt32 mods, UInt32 keycode) {
    if (gHandlerRef != NULL) return -1;
    EventTypeSpec evtType;
    evtType.eventClass = kEventClassKeyboard;
    evtType.eventKind = kEventHotKeyPressed;
    InstallApplicationEventHandler(&hotKeyHandler, 1, &evtType, (void *)gHotKeyID, &gHandlerRef);
    EventHotKeyID hkID;
    hkID.signature = 'htk1';
    hkID.id = gHotKeyID;
    EventHotKeyRef ref = NULL;
    OSStatus s = RegisterEventHotKey(keycode, mods, hkID, GetApplicationEventTarget(), 0, &ref);
    if (s != noErr) return 0;
    return 1;
}

static void unregisterHotKey() {
    // Carbon 没有 "unregister all",需要保存 EventHotKeyRef。
    // 当前实现只支持单 hotkey,重启 app 才解绑 — V2 重构。
    if (gHandlerRef != NULL) {
        RemoveEventHandler(gHandlerRef);
        gHandlerRef = NULL;
    }
}
*/
import "C"

import (
	"fmt"
	"sync"
)

// globalHotKeyManager 全局 hotkey 句柄,单例。
var globalHotKeyManager = &globalHotKey{
	handlers: make(map[string]func()),
}

// globalHotKey 内部状态,只通过 globalHotKeyManager 访问。
type globalHotKey struct {
	mu       sync.Mutex
	handlers map[string]func()
	installed bool
}

// fire 派发 hotkey 事件到对应 combo 的 handler。export 给 C。
//
//go:export goHotKeyFired
func fire(id uint32) {
	if id != 1 {
		return
	}
	globalHotKeyManager.mu.Lock()
	h := globalHotKeyManager.handlers["Cmd+Shift+S"]
	globalHotKeyManager.mu.Unlock()
	if h != nil {
		go h() // 异步执行,避免阻塞 CFRunLoop
	}
}

// Register 注册一个全局快捷键。
// 当前 V1 仅支持 combo="Cmd+Shift+S",keycode=kVK_ANSI_S、mods=cmdKey|shiftKey。
// 其它 combo 返回 error,后续 V2 扩展。
func (g *GlobalHotKeyManager) Register(combo string, h func()) error {
	if g == nil || h == nil {
		return fmt.Errorf("globalhotkey: invalid args")
	}
	if combo != "Cmd+Shift+S" {
		return fmt.Errorf("globalhotkey: combo %q not supported (only Cmd+Shift+S in V1)", combo)
	}
	g.mu.Lock()
	g.handlers[combo] = h
	installed := g.installed
	g.installed = true
	g.mu.Unlock()
	if installed {
		return nil
	}
	const mods = (1 << 8) | (1 << 9) // cmdKey=256, shiftKey=512
	const keycode = 1                 // kVK_ANSI_S = 1
	rc := C.registerHotKey(C.uint(mods), C.uint(keycode))
	if rc != 1 {
		return fmt.Errorf("globalhotkey: RegisterEventHotKey failed (likely missing Accessibility entitlement)")
	}
	return nil
}

// Unregister 取消绑定。当前 V1 不支持单 combo 解除,要清需重启 app。
func (g *GlobalHotKeyManager) Unregister(combo string) error {
	if g == nil {
		return nil
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.handlers, combo)
	if len(g.handlers) == 0 {
		C.unregisterHotKey()
		g.installed = false
	}
	return nil
}
