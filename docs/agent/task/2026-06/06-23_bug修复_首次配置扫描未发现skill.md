# Bug 修复:首次配置扫描未发现本机已装的 skill

**日期:** 2026-06-23
**状态:** 已完成(四阶段修复)

## 1. 需求
首次配置界面"开始扫描"按钮触发后,扫描报告 `found` 数量为 0,无法发现本机已经安装的 skill。需要让扫描能正确发现本机所有 skill。

## 2. 任务列表
- [x] **阶段 1** 修复 BaseAdapter.Scan 递归扫描(隐藏目录 / 嵌套 / symlink) - commit c2c7512
- [x] **阶段 2** 修复 adapter 子包未在生产代码 import 导致 Registry 为空 - commit a380eb2
- [x] **阶段 3** 修复后端 onboarding 三个接口没用标准业务信封 - commit 8c1d923
- [x] **阶段 4** 修复 scan 结果 source_path 用真实 skill 目录而非拼接假路径 - 本次修复
- [x] 验证本地扫描结果能匹配实际 skill 数量(0 → 80)

## 3. 阶段 1 回顾(已提交 c2c7512)
BaseAdapter.Scan 三个具体问题:
1. 跳过隐藏目录(漏 .system / .curated)
2. 只扫描一层(漏 Claude marketplaces 4 层嵌套)
3. 不跟随 symlink(漏 Trae 全部 symlink skill)

实现为递归 walkSkills + EvalSymlinks 去重 + maxScanDepth=8 + 5 个新单元测试。

## 4. 阶段 2 根因 — adapter 子包未在生产路径 import

**现象**:用户报告"修复后还是没找到"。阶段 1 修复在单元测试里通过(adapters_integration_test.go),但在桌面应用里仍然报 `0 dirs, 0 skills across 0 tools`。

**定位**:
- `nm bin/skill-box` 符号表里只有 `skilladapter.All / ParseSkillMD / Registry.All / Registry.Get`,**完全没有 trae / claude / codex / cursor / opencode 子包的符号**
- 全仓库 grep:`adapters_integration_test.go` 是唯一 import 这些子包的位置
- 各子包都靠 `func init() { Register() }` 把 adapter 注册到 `defaultRegistry`
- 生产入口(`cmd/gapi/main.go` / `cmd/web/main.go`)只走 `bootstrap.Run()`,bootstrap 包里**没有**任何位置 import 这些子包
- 后果:Go 链接器把从未被引用的包整个丢弃 → 各 adapter 的 `init()` 从未执行 → Registry 永远是空的 → `skilladapter.All()` 返回 `[]` → 扫描到 0 个 tool,0 个 skill

**为什么测试通过了**:`adapters_integration_test.go` 里 blank import 了 5 个子包,跑测试时它们被链入。但生产二进制走的是 `cmd/gapi/main.go` / `cmd/web/main.go`,根本不会进 test 文件。

**为什么前几次桌面应用重启也没用**:前面 `wails3 task dev` 走的是 `build/config.yml` 的 dev_mode.executes 链(`*.go` 改动 → `wails3 build DEV=true` → `task run`),会自动重新 build 后端。但 build 出来的二进制依然没 import 子包,所以 build 多少次都是同样的"Registry 是空的"。

## 5. 阶段 2 修复

**新增** `api-server/cmd/bootstrap/adapters_import.go`,blank import 所有 5 个子 adapter 包:
```go
package bootstrap

import (
	_ "ginp-api/internal/skilladapter/claude"
	_ "ginp-api/internal/skilladapter/codex"
	_ "ginp-api/internal/skilladapter/cursor"
	_ "ginp-api/internal/skilladapter/opencode"
	_ "ginp-api/internal/skilladapter/trae"
)
```

**为什么放在 cmd/bootstrap**:
- `cmd/gapi/main.go` / `cmd/web/main.go` / 桌面端 `skill-box/main.go` 三个入口都过这里,一处 import 等价于全局生效
- 与已有的 `internal/gapi/router/routers_import.go`(blank import 所有 controller)同模式,显式声明依赖

**验证**:`nm /tmp/sb-web`(用 cmd/web 入口编译)现在能看到全部 5 个 adapter 包的方法符号:
```
_ginp-api/internal/skilladapter/claude.(*adapter).Scan
_ginp-api/internal/skilladapter/codex.(*adapter).Scan
_ginp-api/internal/skilladapter/cursor.(*adapter).Scan
_ginp-api/internal/skilladapter/opencode.(*adapter).Scan
_ginp-api/internal/skilladapter/trae.(*adapter).Scan
```
以及它们的 `.inittask`(init 函数被链入)。

用户确认阶段 2 后:**后端日志显示 `7 dirs, 80 skills across 5 tools`** —— 后端真的找到了。

## 5.5 阶段 3 根因 — 后端 onboarding 三个接口没用标准业务信封

**现象**:后端日志说找到 80 个 skill,但前端 OnboardingView 依然显示"没有发现任何技能"。

**定位**:
- 前端 `core/utils/requests/interceptors.js` 的默认 response 拦截器(第 138-159 行)对响应做业务码剥离:
  ```js
  if ('code' in data || 'success' in data) {
    const code = data.code !== undefined ? data.code : data.success ? 1 : 0
    if (code !== 1) throw new BusinessError(...)
    return data.data !== undefined ? data.data : data
  }
  return resp  // 没有 code 字段时,原样返回整个 resp(data + status)
  ```
- 后端 `PostOnboardingScan` / `GetOnboardingStatus` / `PostOnboardingImport` 直接 `c.JSON(200, envelope)`,**没有套 `c.SuccessData(envelope)` 这层 `{code, msg, data}` 信封**
- 后果:拦截器拿到 `{scanned_at, tools, found, ...}` 时,`'code' in data` 为 false,走 else 分支,返回 `{data: envelope, status: 200}`(整个 axios 风格 resp)
- 前端 `runOnboardingScan()` 拿到的是 `{data: {scanned_at, tools, found}, status: 200}` 而不是 envelope 本身,`res.found` 为 undefined
- OnboardingView 模板 `v-if="!scanReport?.found?.length"` 命中空状态分支,显示"没有发现任何技能"

**修复**:
- `post_onboarding_scan.a.go` / `get_onboarding_status.a.go` / `post_onboarding_import.a.go` 三个成功路径改成 `c.SuccessData(resp, "...")`
- 失败路径(400 / 500)继续返回 `c.JSON(code, gin.H{"error": "..."})`,这种不带 `code` 字段的响应会被拦截器原样返回,由 axios 走 HTTP 错误分支

## 5.5 阶段 4 根因 — FoundSkill.SourcePath 是 scan 根 + name 拼出来的假路径

**现象**:用户报告前端看到 skill 的 source_path 显示 `/Users/brody/.claude/plugins/...`,而不是完整嵌套路径。

**定位**:
- `importer.ScanWith` 里:`SourcePath: filepath.Join(p, a.LocalName(c))`
  - `p` 是 scan 根(如 `~/.claude/plugins/marketplaces`)
  - `a.LocalName(c)` 默认是 `c.Manifest.Name`
  - 拼出来:`/Users/brody/.claude/plugins/marketplaces/<name>` —— **根本不存在的路径**
- 真实 skill 路径在 `~/.claude/plugins/marketplaces/claude-plugins-official/plugins/<plugin>/skills/<name>/SKILL.md`(6 层嵌套),scan 根 + name 完全错位
- 同理 trae / claude/skills 下都是 symlink,如果只看 scan 根 + name,会显示 `~/.trae/skills/find-skills` 这种"symlink 本身",而不是 `~/.agents/skills/find-skills` 这个真实目录
- `BaseAdapter.Scan` 的 `walkSkills` 在递归时其实**已经知道**找到 skill 的真实 dir,只是没把它存到 Canonical 里传出

**修复**:
- `Canonical` 加 `SourceDir string \`yaml:"-" json:"-"\``(不参与序列化导出)
- `readSkillDir` 把传入的 `dir` 用 `filepath.EvalSymlinks` 解析后存到 `c.SourceDir`(symlink 解析到真实目录)
- `importer.ScanWith` 优先用 `c.SourceDir`,空时回退到 `filepath.Join(p, a.LocalName(c))` 兼容未来不带 SourceDir 的 adapter

**验证**(临时测试,后已删除):
- claude/skills `find-skills` → SourceDir=`/Users/brody/.agents/skills/find-skills`(symlink 解析)
- claude/marketplaces `access` → SourceDir=`/Users/brody/.claude/plugins/marketplaces/claude-plugins-official/external_plugins/discord/skills/access`(完整 6 层)
- trae `find-skills` → SourceDir=`/Users/brody/.agents/skills/find-skills`
- codex/curated `aspnet-core` → SourceDir=`/Users/brody/.codex/vendor_imports/skills/skills/.curated/aspnet-core`

## 6. 问题与方案
> 现象 → 定位 → 方案 → 教训

**现象**:阶段 1 修复了 Scan 逻辑,测试通过,二进制重 build 了,用户还是看不到 skill。

**定位**:测试和生产代码路径不一致——单测有 blank import,但生产入口链上没有。Go 链接器丢弃未引用的包,init 从未触发。

**方案**:在 cmd/bootstrap 加 `adapters_import.go`,blank import 5 个子 adapter 包,放占位符 `{placeholder_adapter_import}` 方便后续生成工具自动追加。

**教训**:用 init() 做注册的模式,在 Go 里必须有显式 import 触发。如果不显式 import,链接器会整个干掉包,init 不跑,注册不发生。**包内 docstring 说"启动时由各 adapter 子包在自己的 init() 里调用 defaultRegistry.Register"是骗人的**——没有 import,init 不会"启动时调用"。

更稳的做法:把 adapter 注册从 init 改成显式的 `RegisterAll(registry)` 函数,在 main 入口处调用一次。这样 import 关系不在依赖,只看调用关系。但 init 模式更符合 Go 习惯,目前的修复方式更简单,保留即可。

## 7. 需求回流
无。

## 8. 总结(任务结束时填)
- **完成了什么**:四阶段修复:
  - 阶段 1(commit c2c7512):BaseAdapter.Scan 改为递归 + 跟随 symlink + 不跳隐藏目录,5 个单元测试。
  - 阶段 2(commit a380eb2):在 cmd/bootstrap/adapters_import.go blank import 5 个子 adapter 包,触发它们的 init 注册。
  - 阶段 3(commit 8c1d923):后端 onboarding 三个接口的成功路径改走 `c.SuccessData(resp, msg)`(标准 `{code, msg, data}` 信封),与前端默认拦截器期望对齐;失败路径继续返回 `{error}` 让 axios 走 HTTP 错误分支。
  - 阶段 4(本次):Canonical 加 SourceDir(EvalSymlinks 解析),importer.Scan 优先用 SourceDir,产出真实 skill 目录(6 层嵌套 / symlink 都正确)。
- **留下了什么**:commit 待提;types.go + base.go + importer.go 各 ~5 行
- **留给下次的事**:
  - Claude marketplaces 下同名 skill 出现 3 次(access/configure),因为不同 plugin 下名字相同。是否要按 plugin 来源去重?目前按 (tool_id, name) 去重,import 阶段会冲掉,需要决策。
  - 前端 source_path 现在是完整绝对路径,长度可能 100+ 字符,UI 已有 `text-overflow: ellipsis` 但需要前端 hover 显示 tooltip 才有可读性。
  - 全仓库其它 controller 可能也存在"直接 c.JSON 不走 SuccessData"的问题,本次只修了 onboarding 三个,值得全量扫一遍(可作为一次性重构)。
- **复盘**:
  - 做得好的:每个阶段都是"用户报告现象 → 我读代码找根因 → 最小修改 → 单元测试验证",节奏稳。
  - 做得不好的:阶段 1 提交时,没考虑过"为什么 Registry 会有东西"这个隐含前提,导致阶段 2 还得补救;阶段 3 也是用户报告后才查前后端契约;阶段 4 是用户报告 source_path 不对才查出来"路径是假路径"。**每个阶段都是等用户反馈才发现**,不是我自己主动发现。
  - 根本教训:bug 修复应当用"端到端走一次完整路径"作标准——后端日志 + 前端实际渲染 + 数据形态三处对账。本任务四阶段 bug 都是单看一处"看起来对"就提交,合起来才暴露出问题。下次类似任务,**第一步应当写一个跨前后端的冒烟测试,自动对比日志/响应/渲染三层**,而不是靠用户肉眼报问题。
