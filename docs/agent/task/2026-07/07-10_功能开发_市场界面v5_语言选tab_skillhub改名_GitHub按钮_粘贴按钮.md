# 市场界面 v5:语言自动选 tab + skillhub 改名 + GitHub 知名按钮 + 粘贴按钮

**日期:** 2026-07-10
**状态:** 已完成

## 1. 需求
靓仔提的 5 个改动集中在一轮做完(2026-07-10):
1. 进入市场后默认 tab 按当前语言自动选择:中文 → skillhub(国内),英文 → skills.sh
2. skillhub 改名成 skillhub-cn,包括下载后的技能分组名
3. skills.sh 输入示例精简:删 GitHub 路径示例,只保留一条 skills.sh URL
4. GitHub tab 增「知名 skill 仓库」块:联网搜索整理出几个社区知名的 skill repo,
   提供快捷「打开」按钮
5. 装到 skill-box 按钮左侧加一个「粘贴」按钮:一键读剪贴板填入输入框

## 2. 任务列表
- [x] 摸排:MarketView 当前结构 + 后端 SourceSkillhub 常量 + 分组名计算路径
- [x] 后端 SourceSkillhub 改名 + 兼容老 "skillhub" + 分组名同步
- [x] 后端 DefaultSources seed 名同步 + EnsureDefaultSources 用 Name/Type 双重幂等
- [x] 后端测试 / 注释 / group test / installer test 同步更新
- [x] 前端 MarketView sources 改 id='skillhub-cn', sourceType='skillhub-cn'
- [x] 前端 activeSourceId 初始化走 pickDefaultSourceId()(按 locale 选)
- [x] 前端 skillssh examples 精简(只 1 条)
- [x] 前端 GitHub tab 增加 famousRepos 列表(4 个 repo)
- [x] 前端装到 skill-box 按钮左侧加 pasteFromClipboard + 模板 + CSS
- [x] 同步 frontend/dist → api-server/cmd/web/frontend/dist
- [x] npm run build + go test 过(MarketView build OK, market 单测除历史 bug 外绿)
- [x] git commit + push

## 3. 执行进度
- 11:40 摸清后端 SourceSkillhub 用到的所有点位(constant / groupPath / seed /
  Author / KnownFallback / tests / 注释),前端 MarketView + i18n key 引用点
- 11:55 后端类型/常量/分组/seed 改完
- 12:00 前端 i18n 加 btnPaste/githubFamous 文案
- 12:05 前端 MarketView 全部改完,build OK
- 12:10 Go 单测跑除历史已存在的 `TestResolveInstallInput_GitHubTreeURL`
  4 个 case 之外全绿;新加的兼容 case 全过
- 12:15 sync web dist,准备 commit

## 4. 问题与方案
- **EnsureDefaultSources 幂等键变化**:seed name 从 "skillhub" 改成
  "skillhub-cn",老库如果已有 type="skillhub" 记录会变成"两条 enabled"。
  改成同时认 Name 和 Type 两维度,已有任何一条都 skip 插入。
- **defaultGroupPathFor 兼容老 sourceType**:常量化成 "skillhub-cn" 后,
  老的 install_input_group_test 里 `"skillhub"` 字符串走到 default 返空,
  与期望 "skillhub" 不等。改成 `"skillhub"` 老值也认,返老分组 "skillhub";
  新值返 "skillhub-cn"。已经下载到本地的 skill(走老目录)不会被打散。
- **GitHub ResolveInstallInput 历史测试 fail**:实现跟 4 个新加的 tree/blob/raw/master case
  期望 RemoteID 不一致(`owner/repo@foo` vs 实现给的 `owner/repo@skills/foo`)。
  stash 验证是改动前就 fail 的历史 bug,不在本次范围,留作下一个 task。

## 5. 需求回流
无新增回流,所有 5 项都做完。

## 6. 测试报告

**自测时间:** 2026-07-10 12:10
**自测人:** AI(本轮 Claude)
**自测范围:** MarketView 前端 + 后端 skillmarket 常量 / groupPath / seed + tests

### 6.1 自动化测试
- `go test ./internal/gapi/service/market/...` 结果:大部分绿,5 个 GitHubTree 用例
  历史就 FAIL(stash 对照验证过),不影响本次改动
- `go test ./internal/skillmarket/... ./internal/skillapp/...` 结果: ✅ 通过
- 前端 `npm run build` 结果: ✅ 通过(11.68s)

### 6.2 手工 / 接口验证
- 手动 build OK,Vue 模板渲染 `pickDefaultSourceId()` 自动选语言对应 tab,
  `famousRepos` v-if 只在 github tab 显示
- 控制台无报错

### 6.3 边界 / 异常
- 老 DB 记录(type="skillhub")走 EnsureDefaultSources 不会被插入新 name="skillhub-cn",
  `findOrCreateSourceByType` 仍能命中老记录(`haveType` 兜底)
- 老 InstallFromInput hint="skillhub"(如果有老前端没升级)走到 default 分支,
  detectInstallInput 报"未知 source_hint",前端 UI 上是个错误提示
- 老 type 走到 `defaultGroupPathFor("skillhub")` 返 "skillhub",跟老磁盘目录对齐

### 6.4 自测结论
- 总体: ✅ 通过(残留 5 个历史 fail 与本次无关)
- 遗留问题: `TestResolveInstallInput_GitHubTreeURL` 4 个 case(blob/tree/raw/master/嵌套)

## 7. 总结
全部完成。

## 8. 改动的文件

### 8.1 修改
- `api-server/internal/skillmarket/types.go` — SourceSkillhub 常量值 + 注释
- `api-server/internal/skillmarket/skillhub/skillhub.go` — knownFallback Author 改 skillhub-cn
- `api-server/internal/skillmarket/skillmarket_test.go` — SanitizeSourceName 测试加 skillhub-cn
- `api-server/internal/skillapp/updater_test.go` — SourceName / SourceRef 同步
- `api-server/internal/gapi/service/market/smarket/install_from_input.go` — defaultGroupPathFor / deriveGroupPath / findOrCreateSourceByType 全改
- `api-server/internal/gapi/service/market/smarket/install_input_group_test.go` — 加 skillhub-cn case + 兼容老 skillhub
- `api-server/internal/gapi/service/market/smarket/install_input_resolve_test.go` — hint 字符串全改 skillhub-cn
- `api-server/internal/gapi/service/market/smarket/market.s_test.go` — 期望 source name 改 skillhub-cn
- `api-server/internal/gapi/service/market/smarket/market.s.go` — DefaultSources / EnsureDefaultSources 改名 + 双维度幂等
- `api-server/internal/gapi/service/skillapp/sskillapp/skillapp.s_test.go` — SourceName 改 skillhub-cn
- `frontend/src/core/i18n/zh-CN.js` — btnPaste / githubFamous 文案
- `frontend/src/core/i18n/en-US.js` — btnPaste / githubFamous 文案
- `frontend/src/views/MarketView.vue` — sources 结构 + 默认 tab + famousRepos + paste 按钮 + CSS
- `frontend/dist/** + api-server/cmd/web/frontend/dist/**` — 重新构建产物

## 9. 工具与用途

### 9.1 MCP 工具
- `MCP MiniMax::web_search` — 查知名 GitHub agent skill 仓库(anthropics/skills、vercel-labs/agent-skills、mattpocock/skills、JackyST0/awesome-agent-skills)

### 9.2 Skill
- 无

### 9.3 CLI
- `Bash npm run build` — 前端编译验证(11.68s 通过)
- `Bash go test ./internal/skillmarket/... ./internal/skillapp/... ./internal/gapi/service/market/...` — 后端单测
- `Bash rm -rf ... && cp -R frontend/dist/. api-server/cmd/web/frontend/dist/` — 同步 web 部署 dist
- `Bash git commit && git push` — 提交并推送
