* 前端界面设计时，用户偏好简洁一点的界面，并且很反感AI感很强的界面 -20260623更新
* Claude Code 用户日常用的 skill 在 `~/.claude/skills/` 下(以 symlink 形式存在,目标在 `~/.agents/skills/` 等),不是 `~/.claude/plugins/marketplaces/`。adapter 必须同时扫这两个根,否则会漏掉用户的日常 skill。-20260623
* Claude / Codex adapter 不仅要扫 user 根,还要扫 system 根(plugins/marketplaces / .system / .curated),但 importer 必须用 category=user|system 区分。BaseAdapter.SystemPaths + IsSystemPath 是约定,新增工具时记得声明。-20260623
* 桌面端请求/响应日志在 `~/.skill-box/logs/2026-06/06-DD-request.txt`(每请求一段,含响应体)。bug 复现不出来时第一件事是翻这个文件,不要瞎猜。-20260623
* skillstore 是 `(StoreRoot, scope, name, version)` 唯一索引的物理存储,ListNames 只看 name 不看 version,前端做"已存在"判断用 name 即可。-20260623
* 跨 module 共享代码不能放在 `ginp-api/internal/`(被 Go internal 规则拒),要用 root module 的 `pkg/` 平面(如 `pkg/fsutil`)。-20260624
* Onboarding 入口在 phase2 直接展示扫描结果,跳过 phase1 status 步骤。状态信息改由 App.vue 顶栏 toolsReady/total 徽章提供,不要再回退到 phase1 表格。-20260624
* Onboarding 重复检测分两层:① 客户端 store 已存在(name 匹配,不分 version)→ 置灰 + "已存在"标签;② 跨工具同名互斥(同 name + version 不同 tool_id)→ 选一个后另一个自动取消。selectExclusiveByName 集中处理。-20260624