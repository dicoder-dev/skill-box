---
name: wails-v3-getscreen-primary-resize
description: wails v3 alpha.60 桌面端获取屏幕尺寸 + 按屏幕比例重置主窗口尺寸的可用模式(GetScreen 走 InvokeSync 需窗口 ready,resize 应在 startupAsync 协程里调而非 NewApp 阻塞阶段)
metadata:
  type: project
---

# wails v3 alpha.60 GetScreen + SetSize 桌面端自适应

alpha.60 的 `Window.GetScreen() (*Screen, error)` + `Window.SetSize(w,h int)` + `Window.SetMinSize(w,h int)`
都是公开接口,组合起来实现"按屏幕 DIP 宽度 N% 自动设主窗口尺寸"完全可行。

## 关键事实

1. **`Screen.Size` / `Screen.Bounds` 都是 DIP 宽度**,与 `WebviewWindowOptions.Width/Height`
   是同一坐标系 — **不需要再手动除 `ScaleFactor`**。这是为什么 `Screen.Size.Width` 拿到的
   数字可以直接当窗口宽度的参考使用。

2. **`GetScreen()` 内部走 `InvokeSyncWithResultAndError`**,要求窗口 Native 端已经注册。
   **不能在 `NewApp` 阻塞阶段调**(Wails 主循环还没起来),会卡住或者拿不到值。
   正确做法是把 resize 逻辑放到 `Run()` 之后的 `startupAsync` 协程里(已有 `time.Sleep(500ms)`)。

3. **场景里 "autoResize=true + start_minimized=true" 的次序敏感**:
   - 如果先做完 start_minimized 检查再 resize,start_minimized 路径里的 `w.Hide()`
     会调用,但由于协程并发,可能窗口已经显示 1 帧再被隐藏 → 视觉上闪一下。
   - **正确**:先 resize 再 start_minimized。窗口尺寸调整发生在任何可见化之前。

4. **`Screen.Size.Width` 在某些平台(Mac 多屏场景)可能为 0** → 兜底用 `Screen.Bounds.Width`。
   两者都为 0 时记 warning,不动窗口,由 NewApp 的兜底 Width/Height 顶住。

## 模板代码

参考 `desktop/wails_app.go:resizePrimaryToScreenRatio` 与 `startupAsync`:

```go
const (
    defaultPrimaryWidthRatio  = 0.8
    defaultPrimaryHeightRatio = 0.625 // 16:10 = 10/16
    minPrimaryWidthRatio      = 0.6
    minPrimarySizeFloorWidth  = 960
    minPrimarySizeFloorHeight = 600
)

func (a *App) resizePrimaryToScreenRatio(widthRatio float64) {
    w := a.app.Window.Current()
    if w == nil { return }
    screen, err := w.GetScreen()
    if err != nil || screen == nil {
        log.Printf("GetScreen failed: %v, fallback", err); return
    }
    screenW := screen.Size.Width
    if screenW <= 0 { screenW = screen.Bounds.Width }
    if screenW <= 0 { return }

    newW := int(math.Round(float64(screenW) * widthRatio))
    newH := int(math.Round(float64(newW) * defaultPrimaryHeightRatio))
    minW := int(math.Round(float64(screenW) * minPrimaryWidthRatio))
    if minW < minPrimarySizeFloorWidth { minW = minPrimarySizeFloorWidth }
    minH := int(math.Round(float64(minW) * defaultPrimaryHeightRatio))
    if minH < minPrimarySizeFloorHeight { minH = minPrimarySizeFloorHeight }

    w.SetSize(newW, newH)
    w.SetMinSize(minW, minH)
}

// 必须在 startupAsync 协程里跑,先于 start_minimized:
go func() {
    if autoResize { a.resizePrimaryToScreenRatio(defaultPrimaryWidthRatio) }
    time.Sleep(500 * time.Millisecond)
    // ...
    if startMinimized { /* w.Hide() */ }
}()
```

## `AppConfig.AutoSizeByScreen` 模式

Wails v3 alpha.60 没有"窗口尺寸记忆"的内建开关。
要兼顾"按屏幕比例自适应"与"调用方显式给固定尺寸",用一个 `AutoSizeByScreen bool` 字段做开关:

- main.go 默认不传 Width/Height → `AutoSizeByScreen` 走默认 true 路径
- main.go 显式给 `Width=1600, Height=1000` → `NewApp` 自动把 `AutoSizeByScreen=false`,
  本次启动走固定尺寸;同时要把 `autoResize bool` 透传到 `App` 让 startupAsync 知道

## How to apply

- 新加桌面端"按屏幕自适应尺寸 / 位置 / 居中打开"功能时,直接复用本模式。
- 用 Wails 给的 `GetScreen()` / `SetSize()` / `SetMinSize()` 即可,alpha.60 全部已 stable
  (虽然带 alpha 后缀,但窗口控制 API 是稳定的)。
- 不要在 `NewApp` 阻塞阶段调 `GetScreen`,否则会卡。需要异步。
