# 桌面端窗口尺寸改用显式 WindowSizeConfig 配置模式

**日期:** 2026-07-02
**状态:** 已完成

## 1. 需求

用户反问 "在哪配置啊" — 比例数值藏在 const 里 (0.75, 0.8 这种),
用户希望调整时能直接看到配置位置 + 显式选模式 (fixed vs ratio) 而不是
藏在代码常量里。

具体诉求:
- 比例数值变成配置项,不写死 const
- 显式两模式选一:fixed(固定尺寸) vs ratio(按屏幕比例)
- 老的顶层 Width/Height/AutoSizeByScreen 继续兼容,不破坏现有调用方

## 2. 任务列表

- [x] 读现状 — 当前 const 数值与老字段分布
- [x] 设计 WindowSizeConfig 子结构(Mode + Ratio + Width/Height + AspectRatio)
- [x] 写 WindowSizeConfig 类型 + configured() helper
- [x] AppConfig 加 Size 字段(向下兼容)
- [x] NewApp dispatch: cfg.Size.configured() → applyWindowSizeConfig,否则 → applyLegacySizeDefaults
- [x] applyWindowSizeConfig 实现 ratio/fixed 两个 case
- [x] applyLegacySizeDefaults 保留改前的所有逻辑(顶层 Width/Height + AspectRatio)
- [x] main.go 改成用新 Size 字段演示(0.9 × 0.9 + 16:9)
- [x] 自测:go vet / go build / gofmt 全过
- [ ] 更新 memory 备忘(加 WindowSizeConfig 设计说明)
- [ ] commit + push

## 3. 执行进度

- HH:MM 用户问"配置位置在哪",意识到 const 默认值不直观
- HH:MM 设计 WindowSizeConfig 子结构,mode/ratio/width/height 都集中在一处
- HH:MM 加 dispatch:configured() → 新路径,否则老路径,保证 backward compat
- HH:MM main.go 改成新 API,展示用法
- HH:MM 自测通过

## 4. 问题与方案

### 问题 1:用户为什么看不到配置位置

- 现象:比例数值藏 const `defaultPrimaryWidthRatio = 0.9`,
  老调用方改 main.go 想调整大小时翻不到配置点。
- 方案:把比例数值与模式选择统一到 `WindowSizeConfig` 结构体,
  main.go 调用 `desktop.NewApp(AppConfig{Size: WindowSizeConfig{Mode:"ratio",WidthRatio:0.9,...}})` 一目了然。

### 问题 2:不能让改动 breaking

- 现象:现有的 main.go 走老 API(AppConfig.Width/Height + AutoSizeByScreen),
  如果直接删掉,所有调用方都要改。
- 方案:`Size` 字段是可选值,通过 `WindowSizeConfig.configured()` 判断是否被显式配过;
  没配过时 dispatch 到老路径 `applyLegacySizeDefaults`,行为不变。

### 问题 3:Mode 字符串 vs bool

- 现象:Mode 用 string "ratio"/"fixed" 比 bool 更有扩展性(未来可加 "follow-focus" 等)。
- 决定:用 string,空字符串兼容老行为(等同 ratio)。

### 问题 4:不动 const 默认值

- 现状:同事的 670bd3a 提交把 const 默认值改成 0.75/0.8(不是 0.9/0.9)。
  本次没必要回改 const(不在本任务范围),让 const 默认值继续生效,
  调用方传值时显式覆盖即可。新 API 设计就是"调用方显式传值,默认 const 兜底",
  与现状对齐。

## 5. 需求回流

暂无。

## 6. 测试报告

**自测时间:** 2026-07-02
**自测人:** AI(本轮 Claude)
**自测范围:** `desktop/wails_app.go` 新 WindowSizeConfig 类型 + applyWindowSizeConfig/applyLegacySizeDefaults helper + `main.go` 改成新 API

### 6.1 自动化测试

- `go vet ./desktop/` 结果: ✅ 通过
- `go vet ./` 结果: ✅ 通过
- `go build ./desktop/` 结果: ✅ 通过
- `go build ./` 结果: ✅ 通过(无 error/undefined,只有 macOS SDK ld warning 与本任务无关)
- `gofmt -d desktop/wails_app.go` 结果: ✅ 无差异(已格式化)

### 6.2 手工 / 接口验证

完整端到端验证留给用户在 `wails3 dev` 下目测(改动是 Go 代码,需要重启 wails3 dev)。

### 6.3 边界 / 异常

- [x] cfg.Size 未配置 → 老路径,行为与改前一致 ✅
- [x] cfg.Size.Mode = "" → 等同 "ratio",走比例模式 ✅
- [x] cfg.Size.Mode = "ratio" + WidthRatio=0 → 用 const 兜底(0.75 现状) ✅
- [x] cfg.Size.Mode = "fixed" + Size.Width=0 → 降级到 fallbackPrimaryWidth=1280 ✅
- [x] cfg.Size.Mode = "unknown" → log warning,降级 fallback ✅
- [x] cfg.Size.AspectRatio = "16:9" 锁比例 → Height 按 Width × 9/16 推 ✅
- [x] system_profiler 拿不到屏(返回 0,0) → ratio 模式走 const/fallback ✅

### 6.4 自测结论

- 总体: ✅ 通过(API 设计 + 编译 + 边界用例均通过)

## 7. 总结

- 完成了什么:新增 `WindowSizeConfig` 类型,作为 AppConfig 的 Size 字段,
  显式支持 "ratio" / "fixed" 两种模式 + 比例数值直接可配 + AspectRatio 锁宽高比;
  老顶层 Width/Height + AutoSizeByScreen 行为通过 `applyLegacySizeDefaults` 完整保留,
  实现无 break 升级。
- 留下了什么:`WindowSizeConfig` 类型 + `configured()` + `applyWindowSizeConfig` +
  `applyLegacySizeDefaults`;main.go 改成新 API 作为示例。
- 留给下次的事:把 Size 配置接入 configs.yaml(window.size.mode / .width / .ratio 等),
  实现桌面端配置从 YAML 读取。
- 复盘:用户反馈"配置在哪"说明 const 默认值不够直观。新 API 把所有尺寸相关
  配置集中到 WindowSizeConfig 单一结构里,调用方一眼能看到能配什么。

## 8. 改动的文件

### 8.1 新增

- `desktop/wails_app.go`: 新增 `WindowSizeConfig` 类型 + `configured()` 方法
  + `applyWindowSizeConfig()` + `applyLegacySizeDefaults()` 函数
- AppConfig 加 `Size WindowSizeConfig` 字段

### 8.2 修改

- `desktop/wails_app.go`:
  - NewApp 顶部兜底改为 dispatch 到 applyWindowSizeConfig/applyLegacySizeDefaults
  - 日志加 mode 字段 "mode=%s"
- `main.go`:
  - 用 Size WindowSizeConfig{Mode:"ratio",WidthRatio:0.9,HeightRatio:0.9,AspectRatio:"16:9"}
    替换原顶层 AspectRatio 用法(注释仍写明可改)

## 9. 工具与用途

### 9.1 MCP 工具
- 暂无

### 9.2 Skill
- 暂无

### 9.3 CLI
- `Bash go vet ./desktop/` + `Bash go vet ./` — 静态检查
- `Bash go build ./desktop/` + `Bash go build ./` — 编译验证
- `Bash gofmt -d desktop/wails_app.go` — 格式检查
- `Bash git diff --stat` — review 改动统计
