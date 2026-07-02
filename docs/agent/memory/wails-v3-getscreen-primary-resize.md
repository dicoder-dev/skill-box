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

- 新加桌面端"按屏幕自适应尺寸 / 位置 / 居中打开"功能时,**优先用 macOS
  `system_profiler SPDisplaysDataType` 同步取屏幕尺寸**;不要依赖 wails 的 GetScreen / SetSize。
- 实测证明:alpha.60 的 `Window.SetSize()` 即便在 startupAsync 协程里 sleep 后调,
  也不生效(用户报告"两次都不生效")。**wails 的 alpha API 在时序敏感场景下并不可靠**。
- 同步路径实测可用:`exec.Command("system_profiler", "SPDisplaysDataType").Output()`
  1-3 秒返回,正则 `Resolution:\s+(\d+)\s+x\s+(\d+)` 拿宽高。直接灌进
  `WebviewWindowOptions.Width/Height`,让窗口天生就是大尺寸。
- Retina 缩放下 Resolution 给的是 DIP 值(跟 `WebviewWindowOptions.Width` 同单位),
  不用再除 ScaleFactor。多屏时只取第一个匹配(主屏)。
- 跨平台注意:Windows / Linux 桌面包目前不需要这套(本项目桌面端只发布 darwin/windows,
  但目前主战场是 macOS 开发)。如果未来要适配 windows,需要切换到 win32 EnumDisplayMonitors。

## `WindowSizeConfig` 配置模式(2026-07-02 增)

`AppConfig.Size` 是显式的窗口尺寸配置入口,替代早期散落在 const + 顶层字段
(`Width/Height/AutoSizeByScreen/AspectRatio`)的隐式逻辑。所有尺寸相关参数集中在一处:

```go
app := desktop.NewApp(desktop.AppConfig{
    Size: desktop.WindowSizeConfig{
        Mode:        desktop.WindowSizeModeRatio, // 而非字符串 "ratio"
        WidthRatio:  0.9,
        HeightRatio: 0.9,
        MinWidth:    960,
        MinHeight:   540,
        // AspectRatio: "16:9",  // 可选,锁宽高比
    },
}, backend)
```

调用方通过 `WindowSizeConfig.configured()` 判断是否被显式配置:
- **已配置**:走 `applyWindowSizeConfig`,按 Mode 派发 ratio / fixed 算法。
- **未配置**:回落到 `applyLegacySizeDefaults`,沿用改前的顶层 Width/Height 等字段
  行为,**完全向后兼容**。

两种模式选择规则:
- `Mode == desktop.WindowSizeModeFixed`(常量 `"fixed"`):窗口 = `Size.Width × Size.Height`,不随屏幕变(打包场景)。
- `Mode == desktop.WindowSizeModeRatio`(常量 `"ratio"`)或 `""`:窗口 = 屏幕 × WidthRatio / 屏幕 × HeightRatio,
  留 0 走 const 默认值。配 `AspectRatio="16:9"` 时高度按宽度反推,
  任何屏幕下都锁 16:9。

**重要:Mode 字符串必须用常量 `WindowSizeModeRatio` / `WindowSizeModeFixed`,
不要直接写 `"ratio"` / `"fixed"` 字面值**。常量集中在 `desktop/wails_app.go` 顶部,
switch 与默认行为都在 desktop 包内部完成。AspectRatio 仍是自由字符串("16:9" 这种),
因为它是比例值不是枚举。

未来加新模式(如 `WindowSizeModeFollowFocus`)只需在 const 块加一行,不用改 API 形状。

## `AppConfig.AutoSizeByScreen` 模式(向下兼容路径)

Wails v3 alpha.60 没有"窗口尺寸记忆"的内建开关。
早期版本用 `AppConfig.AutoSizeByScreen bool` 区分两种行为;**新代码优先用
`WindowSizeConfig`**(见上),老字段保留向下兼容:

- main.go 不传 Width/Height → 老 `AutoSizeByScreen` 走默认 true 路径
- main.go 显式给 `Width=1600, Height=1000` → 老 `NewApp` 自动把
  `AutoSizeByScreen=false`,本次启动走固定尺寸;同时要把 `autoResize bool`
  透传到 `App` 让 startupAsync 知道
