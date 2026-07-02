---
name: wails-v3-getscreen-primary-resize
description: wails v3 alpha.60 桌面端获取屏幕尺寸 + 按屏幕比例重置主窗口尺寸的可用模式(GetScreen 走 InvokeSync 需窗口 ready,resize 应在 startupAsync 协程里调而非 NewApp 阻塞阶段)
metadata:
  type: project
---

# wails v3 alpha.60 GetScreen + SetSize 桌面端自适应

alpha.60 的 `Window.GetScreen() (*Screen, error)` + `Window.SetSize(w,h int)` + `Window.SetMinSize(w,h int)`
都是公开接口,组合起来实现"按屏幕 DIP 宽高 N% 自动设主窗口尺寸"完全可行。

## 关键事实

1. **`Screen.Size` / `Screen.Bounds` 都是 DIP 宽度**,与 `WebviewWindowOptions.Width/Height`
   是同一坐标系 — **不需要再手动除 `ScaleFactor`**。这是为什么 `Screen.Size.Width` 拿到的
   数字可以直接当窗口宽度的参考使用。

2. **`GetScreen()` 内部走 `InvokeSyncWithResultAndError`**,要求窗口 Native 端已经注册
   **且 Wails 主循环 ready**(主循环没起来时 InvokeSync 消息无人消费,会卡死/超时)。
   - **不能在 `NewApp` 阻塞阶段调**(Wails 主循环还没起来)。
   - **不能在 startupAsync 协程的 sleep 之前调**(协程已起但主循环尚未消费消息)。
   - 正确做法:resize 在 `Run()` 之后的 `startupAsync` 协程里 `time.Sleep(500ms)` **之后**。

3. **场景里 "autoResize=true + start_minimized=true" 的次序敏感**:
   - **正确**:先 resize(在 sleep 后)再 start_minimized 检查。窗口尺寸调整发生在任何可见化之前。

4. **`Screen.Size.Width` / `Screen.Size.Height` 在某些平台(Mac 多屏场景)可能为 0** →
   兜底用 `Screen.Bounds`。两者都为 0 时记 warning,不动窗口,由 NewApp 的兜底 Width/Height 顶住。

5. **高度按屏高比例算,不是从 W × ratio 推**。用户场景从宽 80% × 高 16:10 升到宽高均 90%,
   视觉上更接近最大化但不挡任务栏。alpha.60 有 `InitialPosition: WindowCentered` 字段,
   但**没有"按屏比例自动算 Size"的内建字段**,仍需走 SetSize 异步路径。

## 模板代码

参考 `desktop/wails_app.go:resizePrimaryToScreenRatio` 与 `startupAsync`:

```go
const (
    defaultPrimaryWidthRatio  = 0.9
    defaultPrimaryHeightRatio = 0.9
    minPrimaryWidthRatio      = 0.6
    minPrimarySizeFloorWidth  = 960
    minPrimarySizeFloorHeight = 600
)

// 屏幕宽高各自按比例算(不再从 W × ratio 推出 H) — 用户要求宽度 90%+ 高度 90%。
func (a *App) resizePrimaryToScreenRatio(widthRatio, heightRatio float64) {
    w := a.app.Window.Current()
    if w == nil { return }
    screen, err := w.GetScreen()
    if err != nil || screen == nil {
        log.Printf("GetScreen failed: %v, fallback", err); return
    }
    screenW := screen.Size.Width
    if screenW <= 0 { screenW = screen.Bounds.Width }
    screenH := screen.Size.Height
    if screenH <= 0 { screenH = screen.Bounds.Height }
    if screenW <= 0 || screenH <= 0 { return }

    newW := int(math.Round(float64(screenW) * widthRatio))
    newH := int(math.Round(float64(screenH) * heightRatio))
    minW := int(math.Round(float64(screenW) * minPrimaryWidthRatio))
    if minW < minPrimarySizeFloorWidth { minW = minPrimarySizeFloorWidth }
    minH := int(math.Round(float64(screenH) * minPrimaryWidthRatio))
    if minH < minPrimarySizeFloorHeight { minH = minPrimarySizeFloorHeight }

    w.SetSize(newW, newH)
    w.SetMinSize(minW, minH)
}

// 必须在 startupAsync 协程里跑,**在 500ms sleep 之后**,先于 start_minimized:
go func() {
    time.Sleep(500 * time.Millisecond)  // 等 Wails 主循环 ready
    if autoResize { a.resizePrimaryToScreenRatio(0.9, 0.9) }
    // ...
    if startMinimized { /* w.Hide() */ }
}()
```

## 调用顺序的铁律

- **resize 必须在 `time.Sleep(500ms)` 之后,绝不能在 sleep 之前**。
  之前调试发现 sleep 前调会"无效" — 因为 alpha.60 的 `GetScreen()` 走
  `InvokeSyncWithResultAndError`,主循环没起来时消息无人消费,
  会卡死/超时,resize 静默失败,窗口还是兜底的 1280×800。
- **resize 又必须在 start_minimized 检查之前** — 否则 start_minimized=true 时
  窗口可能先一帧以 1280 闪现再被 Hide。
- 所以正确顺序是:**sleep → resize → start_minimized 检查**。中间 500ms 闪烁可接受。

## How to apply

- 新加桌面端"按屏幕自适应尺寸 / 位置 / 居中打开"功能时,直接复用本模式。
- 用 Wails 给的 `GetScreen()` / `SetSize()` / `SetMinSize()` 即可,alpha.60 全部已 stable
  (虽然带 alpha 后缀,但窗口控制 API 是稳定的)。
- 不要在 `NewApp` 阻塞阶段调 `GetScreen`,否则会卡。需要异步,且要在 sleep 之后。

## `AppConfig.AutoSizeByScreen` 模式

Wails v3 alpha.60 没有"窗口尺寸记忆"的内建开关。
要兼顾"按屏幕比例自适应"与"调用方显式给固定尺寸",用一个 `AutoSizeByScreen bool` 字段做开关:

- main.go 默认不传 Width/Height → `AutoSizeByScreen` 走默认 true 路径
- main.go 显式给 `Width=1600, Height=1000` → `NewApp` 自动把 `AutoSizeByScreen=false`,
  本次启动走固定尺寸;同时要把 `autoResize bool` 透传到 `App` 让 startupAsync 知道
