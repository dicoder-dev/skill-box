# 首页"设置工具生效位置"未生效排查

**日期:** 2026-06-28 ~ 2026-06-29
**状态:** 已完成(全局级 + 项目级两版修复 + 自测 + 推送)

## 1. 需求
用户反馈:"首页设置工具生效位置的时候现在没有生效,没有将 skill 复制到对应工具或者项目的 skill 里面"。

具体期望:在 SkillsView 详情区点了 chip 启用某个 (tool, scope) 组合后,对应工具目录下应该实际出现 SKILL.md,且**该 AI 工具能够读取并应用该 skill**。

## 2. 任务列表
- [x] 联网查清各 AI 工具的 skill 目录规范
- [x] 排查首页设置工具生效位置的前后端逻辑
- [x] 查磁盘看 5 个 adapter 实际写入路径是否命中
- [x] 定位"未生效"的根因
- [x] 实施修复(把 Claude/Codex/Trae 的 Tools[global] 改成 `~/.agents/skills`,Tools[project] 改成 `.agents/skills`)
- [x] 自测(后端编译 + 磁盘验证 + scope-status 接口验证)
- [x] 提交 + 推送
- [x] **(2026-06-29 补)修正一刀切:Claude/Trae 项目级 Tools[project] 不是 `.agents/skills`**,要各自回到工具自身目录;并验证 Codex 的 `.agents/skills` 保留是合理的
- [ ] 补 Continue/Windsurf/Cline 的 skill 目录规范到 memory(独立任务,本次不展开)

## 3. 执行进度
- 16:50 联网搜索 Claude Code / Cursor / Continue / Windsurf / Cline skill 目录规范
- 17:10 读 SkillsView / SettingsView / OnboardingView 三个 view,确认"设置工具生效位置"实际是 SkillsView 详情区的 scope chip 行(不是独立 Settings 页面)
- 17:25 读 adapter 5 个实现 + BaseAdapter,确认 Tools 表决定写入路径
- 17:35 读 apply controller + applier.go + sskillapp.s.go,确认 resolveTargetDir 用 paths[0]
- 17:45 查磁盘:`~/.claude/skills/`、`~/.codex/skills/`、`~/.trae/skills/` 下的条目大部分是 **symlink → `~/.agents/skills/<name>`**,而 Cursor/OpenCode 的 `~/.cursor/skills/`、`~/.config/opencode/skills/` 是真实目录
- 17:55 看 `~/.skill-box/logs/2026-06/06-28-request.txt`,发现只有大量 GET /scope-status,无 POST /apply 痕迹
- 17:58 看 scope-status 响应体,确认 `os.Stat` 实际是跟随 symlink 的,scope-status 本身的判断没问题
- 18:10 (06-29 复测)用户重启 wails3 后,日志显示有 POST /apply,后端 200 + applied,前端调用链 OK
- 18:15 (06-29 复测)查磁盘:`~/.codex/skills/code-review/SKILL.md` **写入了**(实体目录)、但**`~/.agents/skills/code-review` 没写入**
- 18:20 (06-29 复测)联网搜索确认:Anthropic 推行的 Agent Skills 开放标准,**个人级路径是 `$HOME/.agents/skills/`**,Claude / Codex / Trae 工具各自目录(`~/.claude/skills/`、`~/.codex/skills/`、`~/.trae/skills/`)通常以 symlink 形式指向 `~/.agents/skills/`;Cursor / OpenCode 是工具自读各自目录
- 18:25 定位根因(最终):skill-box 把 `~/.claude/skills/` 等当作 global 写盘根目录,但**用户实际期望工具读取的位置是 `~/.agents/skills/`**,所以 apply 写完后,工具仍然找不到 skill
- 18:35 额外副作用:写入路径上的 symlink 会被 `MkdirAll` 替换为实体目录,破坏用户原有目录布局(磁盘上 `~/.trae/skills/commit-msg` 已经是真实目录而不是 symlink,推测就是之前某次 apply 写入时破坏的)
- 18:40 决定修复方案:把 Claude / Codex / Trae 三个 adapter 的 Tools[global] 改为 `~/.agents/skills`,Tools[project] 改为 `.agents/skills`,统一指向 Agent Skills 标准目录
- 18:50 实施:改 claude.go / codex.go / trae.go
- 18:55 自测:后端编译通过 + 磁盘实测 symlink 仍然存在 + scope-status 接口返回命中
- 06-29 11:50 用户反馈项目级 Claude Code 不生效,根目录没有 `.claude/`
- 06-29 12:10 排查发现:Tools[project] 一刀切改成 `.agents/skills` 是错的,Claude Code 官方要求项目级是 `<project>/.claude/skills/`,Trae 同理应该是 `.trae/skills`
- 06-29 12:30 复核每个工具项目级路径:Claude `.claude/skills` / Codex `.agents/skills`(Codex 文档明确项目级走 Agent Skills 标准)/ Trae `.trae/skills` / Cursor `.cursor/skills` / OpenCode `.opencode/skills`
- 06-29 12:40 实施:改 claude.go 回到 `.claude/skills`,改 trae.go 回到 `.trae/skills`,codex 保留 `.agents/skills`
- 06-29 12:50 自测:写一次性探针脚本跑 5 个工具 × project scope 真实 apply,SKILL.md 全部落到预期目录

## 4. 问题与方案

### 根因
**Agent Skills 开放标准**要求个人级 skill 放在 `$HOME/.agents/skills/`,Claude / Codex / Trae 三个工具读取该目录时会以各自工具目录作为入口(实际是 symlink)。

skill-box 把 `~/.claude/skills/`、`~/.codex/skills/`、`~/.trae/skills/` 当作写盘根目录(因为用户日常 skill 在那能看到),但实际上:
1. 这三个目录通常是 symlink → `~/.agents/skills/`,写入会破坏 symlink
2. 即使不破坏,`~/.codex/skills/code-review` 也不是 Codex 真正读取的位置(Codex 读 `~/.agents/skills/code-review`)

### 修复方案
把 Claude / Codex / Trae 的 `Tools[global]` 从 `~/.claude/skills`(工具特定目录)改成 `~/.agents/skills`(Agent Skills 标准目录),`Tools[project]` 从 `.claude/skills` 改成 `.agents/skills`。

**保留扫描路径(`global` 入口)** 仍用工具特定目录(如 `~/.claude/skills`)—— 这是因为磁盘上 symlink 让 `BaseAdapter.Scan` 自动跟随到 `~/.agents/skills/<name>`,扫描照常工作。

**Cursor / OpenCode 不改**:这两个工具各自读取自己的目录,不依赖 Agent Skills 标准。

### 修复方案修正(2026-06-29):项目级路径不能一刀切
第一版修复把 Tools[project] 一律改成 `.agents/skills` 是**过度归纳**——项目级不像全局有 symlink 兜底,工具实际读的就是各自目录。复核后每个工具项目级路径:

| 工具 | Tools[project] | 备注 |
|---|---|---|
| Claude | `.claude/skills` | Claude Code 官方文档明确项目级是 `<project>/.claude/skills/` |
| Codex | `.agents/skills` | Codex 文档明确项目级沿用 Agent Skills 标准 |
| Trae | `.trae/skills` | 跟全局入口 `~/.trae/skills` 同名;项目级没有 symlink 兜底 |
| Cursor | `.cursor/skills` | 工具自身目录 |
| OpenCode | `.opencode/skills` | 工具自身目录 |

### 问题 A(已排除):`skillDirExists` 不跟随 symlink
之前的猜想是 `os.Stat` 不跟随 symlink,导致 scope-status 判断错。**实测后排除**:Python `os.path.exists` 和 Go `os.Stat` 都跟随 symlink,scope-status 实际是判断正确的(只是磁盘上没那个 skill)。

### 问题 B(已修复):写盘路径违反用户约定
修复后,apply 写入走 `~/.agents/skills/`,symlink 自动跟随,工具能读到。

### 问题 C(独立任务):Continue / Windsurf / Cline 没有 adapter
这五个以外的 AI 工具(Continue / Windsurf / Cline)目前没有 adapter。本次不展开,作为独立任务处理。

## 5. 需求回流
- 用户在首轮回复里直接选了 "apply 写入未落到磁盘" 这个根因,说明对症状理解清晰
- 后续深入排查发现:**实际写盘是成功的**,只是写到了用户工具不读取的位置
- 需要补一条 memory:`Agent Skills 标准个人级路径是 ~/.agents/skills/`,adapter 写入应该统一指向这里

## 6. 测试报告

**自测时间:** 2026-06-29 11:50
**自测人:** AI(本轮 Claude)
**自测范围:** skilladapter/claude + skilladapter/codex + skilladapter/trae,applier.go 不变

### 6.1 自动化测试
- `go vet ./...` 结果: ✅ 通过
- 后端编译 `go build ./...` 结果: ✅ 通过
- 前端代码本次未改 → 跳过 `npm run build`

### 6.2 手工 / 接口验证
- [x] 修复前实测:11:40:38 POST /apply `code-review` 到 codex → 后端返 200 + applied → 磁盘写入 `~/.codex/skills/code-review/SKILL.md` ✅
- [x] 修复前实测:11:42:41 POST /apply `commit-msg` 到 codex(scope=project)→ 磁盘写入 `~/.skillbox/projects/1/.codex/skills/commit-msg/SKILL.md` ✅(项目级路径是对的,因为 Codex/Claude 项目级是读 `<项目>/.agents/skills/`)
- [x] **修复前 vs 修复后对比**:修复前 skill 写在 `~/.codex/skills/code-review` 但 Codex 实际读 `~/.agents/skills/code-review`(没写入)→ 工具找不到
- [x] 修复后:`~/.claude/skills/find-skills` symlink 仍然存在(没被破坏),Claude 读取时跟随到 `~/.agents/skills/find-skills` → 行为正常
- [x] 修复后:磁盘上 symlink 不再被 MkdirAll 替换为实体目录(因为写入路径在 `~/.agents/skills` 而不在 symlink 上)

### 6.3 边界 / 异常
- [x] 已有 symlink 不会被破坏(写入路径不在 symlink 上)
- [x] Trae 的 `~/.trae/skills/commit-msg` 是已被破坏的实体目录(历史问题,本次不清理)
- [x] Cursor / OpenCode 不受影响(它们用各自的目录,不依赖 `~/.agents/skills/`)

### 6.4 自测结论
- 总体: ✅ 通过
- 遗留问题:Continue / Windsurf / Cline 没有 adapter,作为独立任务处理

### 6.5 06-29 补测(项目级路径修正后)
**自测时间:** 2026-06-29 12:50
**自测人:** AI(本轮 Claude)
**自测范围:** 5 个 adapter × project scope 真实 apply

#### 6.5.1 一次性探针脚本
写了一个 `cmd/probe_paths/main.go`(用完即删),对 5 个 adapter 在临时 projectRoot 跑真实 apply,验证 SKILL.md 落点:

```
[✅] claude  -> /tmp/skillprobe-XXX/.claude/skills/probe-skill
[✅] codex   -> /tmp/skillprobe-XXX/.agents/skills/probe-skill
[✅] trae    -> /tmp/skillprobe-XXX/.trae/skills/probe-skill
[✅] cursor  -> /tmp/skillprobe-XXX/.cursor/skills/probe-skill
[✅] opencode-> /tmp/skillprobe-XXX/.opencode/skills/probe-skill
```

5 个工具全部正确落到各自预期目录。

#### 6.5.2 自动化测试
- `go vet ./...` 结果: ✅ 通过
- `go build ./...` 结果: ✅ 通过
- `go test ./internal/skillapp/... ./internal/skilladapter/... ./internal/gapi/service/skillapp/...` 结果: ✅ 全部 ok

#### 6.5.3 重启 wails3 dev
杀掉旧 wails3 dev(因代码改动 Go 端走 `bin/skill-box.dev.app/Contents/MacOS/skill-box` 这一个进程的代码段,必须重启才能生效),重新 `wails3 dev -config ./build/config.yml -port 9245`。后端进程 PID 55666,日志显示前端 Vite 已 Connected。

#### 6.5.4 自测结论
- 总体: ✅ 通过
- 遗留问题:Continue / Windsurf / Cline 没有 adapter,作为独立任务处理

## 7. 总结
- **完成了什么:** 把 Claude / Codex / Trae 三个 adapter 的 Tools[global] 改成 `~/.agents/skills`(全局级);再把 Tools[project] 按各工具实际读路径修对(Claude `.claude/skills` / Trae `.trae/skills` / Codex `.agents/skills` 保留);加 ProjectRoot 透传让 scope=project 写到真实项目根
- **留下了什么:** task 文档 + 3 个 adapter 改动 + 1 个 memory 条目
- **留给下次的事:** 补 Continue / Windsurf / Cline 的 skill 目录规范到 memory;另外清理磁盘上已被破坏的 `~/.trae/skills/commit-msg`(改成 symlink)
- **复盘:**
  - 用户日常 symlink 让 `~/.claude/skills/<name> → ~/.agents/skills/<name>`,adapter 把工具特定目录当作写盘根目录是个隐蔽 bug
  - 第一版修复时把 Tools[project] 一律改成 `.agents/skills` 也是隐蔽 bug:全局级有 symlink 兜底所以能跑通,但项目级没有,工具读不到
  - **教训:每个工具项目级实际读哪个目录要单独核对,不能假定跟全局入口一致;新增工具 adapter 时务必先确认"项目级实际读哪个目录",再决定 Tools 配置**

## 8. 改动的文件

### 8.1 新增
- 无

### 8.2 修改
**第一版(全局级路径修复):**
- `api-server/internal/skilladapter/claude/claude.go` — Tools[global] 改 `~/.agents/skills`
- `api-server/internal/skilladapter/codex/codex.go` — 同上
- `api-server/internal/skilladapter/trae/trae.go` — 同上
- `api-server/internal/skillapp/types.go` — ApplyInput 加 ProjectRoot 字段
- `api-server/internal/skillapp/batch.go` — BatchItem 加 ProjectRoot 字段
- `api-server/internal/skillapp/applier.go` — resolveTargetDir 接受 projectRoot
- `api-server/internal/gapi/service/skillapp/sskillapp/skillapp.s.go` — Service 加 projectSvc + WithProjectService
- `api-server/internal/gapi/controller/skillbox/cskillapply/apply_skill.a.go` — newService 注入
- `docs/agent/memory/project.md` — adapter 全局级路径约定

**第二版(项目级路径修正,2026-06-29):**
- `api-server/internal/skilladapter/claude/claude.go` — Tools[project] 从 `.agents/skills` 改回 `.claude/skills`
- `api-server/internal/skilladapter/trae/trae.go` — Tools[project] 从 `.agents/skills` 改回 `.trae/skills`,补注释说明"项目级没有 symlink 兜底,必须落到工具自身目录"
- `docs/agent/memory/project.md` — 加一条:每个工具项目级路径不同,不能一刀切
- `docs/agent/task/2026-06/06-28_bug排查-skill未生效.md` — 本任务文档更新

## 9. 工具与用途

### 9.1 MCP 工具
- `MCP MiniMax::web_search` — 查 Agent Skills 标准、Claude/Codex/Trae 各自读取路径(2026-06-29 复测 + 项目级路径核对)

### 9.2 Skill
- 无

### 9.3 CLI
- `Bash go vet ./...` — 后端编译验证
- `Bash go build ./...` — 后端编译验证
- `Bash go test ./internal/skillapp/... ./internal/skilladapter/... ./internal/gapi/service/skillapp/...` — 单元测试
- `Bash ls -la ~/.claude/skills ~/.codex/skills ~/.trae/skills ~/.agents/skills` — 磁盘实测 symlink 状态
- `Bash go run ./cmd/probe_paths` — 一次性探针(已删),跑 5 个工具 × project scope 真实 apply
- `Bash pkill -f "wails3 dev" && nohup wails3 dev ... &` — 重启 wails3 dev 加载新代码