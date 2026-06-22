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
//   - 本文件只暴露 combo="Cmd+Shift+S" 一组绑定;
//     V2 扩展需要动态 combo 解析(modifiers + keycode 拆分)。
package desktop

/*
#cgo CFLAGS: -x objective-c -Wno-deprecated-declarations
#cgo LDFLAGS: -framework Carbon -framework Cocoa
#import <Carbon/Carbon.h>

// 注册一个全局热键。
// mods: cmdKey / shiftKey / optionKey / controlKey 任意组合
// keycode: virtual key code(US 键盘布局)
// 返回:1 = 注册成功,0 = 失败(权限/参数错误)
static int gInstalled = 0;
extern void goHotKeyFired(uint32_t id);

static OSStatus hotKeyHandler(EventHandlerCallRef next, EventRef event, void *userData) {
    EventHotKeyID hkID;
    GetEventParameter(event, kEventParamDirectObject, typeEventHotKeyID,
        NULL, sizeof(hkID), NULL, &hkID);
    goHotKeyFired((uint32_t)hkID.id);
    return noErr;
}

static int registerHotKey(UInt32 mods, UInt32 keycode) {
    if (gInstalled) return 1;
    EventTypeSpec evtType;
    evtType.eventClass = kEventClassKeyboard;
    evtType.eventKind = kEventHotKeyPressed;
    InstallApplicationEventHandler(&hotKeyHandler, 1, &evtType, NULL, NULL);
    EventHotKeyID hkID;
    hkID.signature = 'htk1';
    hkID.id = 1;
    EventHotKeyRef ref = NULL;
    OSStatus s = RegisterEventHotKey(keycode, mods, hkID, GetApplicationEventTarget(), 0, &ref);
    if (s != noErr) return 0;
    gInstalled = 1;
    return 1;
}
*/
import "C"

import (
	"fmt"
	"sync"
)

// 平台特定存储(handler + mu),跨 NewGlobalHotKeyManager 共享。
var (
	ghkMu       sync.Mutex
	ghkHandlers = make(map[string]func())
)

func init() {
	NewGlobalHotKeyManager = func() *GlobalHotKeyManager {
		return &GlobalHotKeyManager{}
	}
}

// goHotKeyFired 派发 hotkey 事件。export 给 C(从 C 侧 hotKeyHandler 调用)。
//
//export goHotKeyFired
func goHotKeyFired(id uint32) {
	if id != 1 {
		return
	}
	ghkMu.Lock()
	h := ghkHandlers["Cmd+Shift+S"]
	ghkMu.Unlock()
	if h != nil {
		go h()
	}
}

// Register 注册一个全局快捷键。
// 当前 V1 仅支持 combo="Cmd+Shift+S"(macOS)。
// 其它 combo 返回 error,后续 V2 扩展。
func (g *GlobalHotKeyManager) Register(combo string, h func()) error {
	if g == nil || h == nil {
		return fmt.Errorf("globalhotkey: invalid args")
	}
	if combo != "Cmd+Shift+S" {
		return fmt.Errorf("globalhotkey: combo %q not supported (only Cmd+Shift+S in V1)", combo)
	}
	ghkMu.Lock()
	ghkHandlers[combo] = h
	installed := g.installed
	g.installed = true
	ghkMu.Unlock()
	if installed {
		return nil
	}
	// cmdKey=256(0x100),shiftKey=512(0x200);kVK_ANSI_S=1
	const mods = 0x100 | 0x200
	const keycode = 1
	rc := C.registerHotKey(C.uint(mods), C.uint(keycode))
	if rc != 1 {
		g.installed = false
		return fmt.Errorf("globalhotkey: RegisterEventHotKey failed (likely missing Accessibility entitlement)")
	}
	return nil
}

// Unregister 当前 V1 不支持单 combo 解除,要清需重启 app。
func (g *GlobalHotKeyManager) Unregister(combo string) error {
	return nil
}
