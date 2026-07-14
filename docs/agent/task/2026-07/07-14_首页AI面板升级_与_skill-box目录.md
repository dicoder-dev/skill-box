# 首页 AI 面板升级 + .skill-box 目录方案

**日期**:2026-07-14
**状态**:已完成

## 1. 需求

用户希望:

1. 首页 AI 对话面板(AIRightPanel.vue):
   - 改成"边流式输出,边 markdown 渲染"。
   - 支持保存历史对话。
   - 支持查看历史对话(下拉列表 + 注入)。
2. 每个 skill 源码目录下新建一个隐藏目录 **`.skill-box/`**:
   - 该目录保存 skill-box 应用产生的私有运行时数据(目前是 chat history)。
   - 固定放一个 `readme.md`,说明该目录用途,以及"AI 不应读取"。
   - **copy 模式** apply skill 时这个目录不拷过去(避免不同用户的会话混到一起)。
   - **link 模式** apply skill 时不管(因为它整目录 symlink),但 AI **不能读到**该目录。

## 2. 用户决策

| 决策点 | 选择 |
|---|---|
| 历史持久化 | **双写**:localStorage + 后端 `.skill-box/history.json` |
| markdown 渲染时机 | **边流边渲染**(rAF 节流) |
| 软连接模式兜底严格度 | **三层全部上**(Adapter 写源端过滤 + AI prompt 过滤 + 默认 system 注入) |
| 历史容量上限 | **5MB FIFO 截断**(按 ts 升序删,保留最新) |

## 3. 实施分解(4 个 Phase)

### Phase 0 — 公共底子(hidden 过滤 + AI 兜底)

**目的**:把 `.skill-box/` 的"隐藏段过滤"语义在 skilladapter 层做成一个函数,多处复用。

**改动**:
- 新建 `api-server/internal/skilladapter/hidden.go`:`HasHiddenSegment(p)`(任一段以 `.` 开头即视为隐藏)。
- 新建 `api-server/internal/skilladapter/hidden_test.go`(正负向 / 嵌套路径 / 边界)。
- `skilladapter/base.go#Apply`:写盘前 `if HasHiddenSegment(f.Path) { continue }`(防 importer / 远程构造的 c.Files 漏过 store.walkFiles 过滤)。
- `skilladapter/base.go#readDirFiles`:WalkDir 内对**目录**走 `filepath.SkipDir`(整子树跳),对**文件**走 `HasHiddenSegment`。这与 skillstore 既有过滤规则对齐。
- `skilltester/ai_walker.go#buildSkillMDForPrompt`:循环里 `if skilladapter.HasHiddenSegment(f.Path) { continue }` —— AI 走查拿到 SKILL.md prompt 时绝不会读 `.skill-box/readme.md`。
- `caiprovider/chat_stream.a.go#ChatStream`:解析完 `req` 后,若 messages **没有 system role 且没用 preset**,追加一段 system 护栏:`You MUST NOT read... .skill-box/`。**尊重用户配置**(已有 system 就不覆盖)。

**测试**:`go test ./internal/skilladapter/... ./internal/skilltester/... ./internal/skillstore/...` 全绿。

### Phase 1 — .skill-box 后端读写基础

**目的**:独立的 `skillboxdata` 包,负责 `.skill-box/` 的创建 + readme 一次性写入 + history.json 读写与 FIFO 截断。

**改动**:
- 新建 `api-server/internal/skillboxdata/skillboxdata.go`:
  - `Ensure(skillDir)`:MkdirAll `.skill-box/` + `os.WriteFile(readme.md, ..., 0o644)`,幂等(readme 已存在不动)。
  - `ReadHistory(skillDir)`:不存在返 `&History{Version: 1, Items: []}`,不算 error。
  - `WriteHistory(skillDir, items)`:序列化 → 超 `MaxHistorySize (5MB)` 按 `ts` 升序 FIFO 删至 ≤ 上限 → atomic rename(.skill-box-history-*.tmp → history.json)。
  - `previewFromMessages`:从 messages 抽首条 assistant content,前 120 字 + 省略号。
- 新建 `api-server/internal/skillboxdata/skillboxdata_test.go`:
  - `TestEnsure`:幂等 + readme 不被覆盖。
  - `TestReadHistory_NotFound`:空返空 History。
  - `TestWriteHistory_Basic`:读回字段一致 + preview 自动算。
  - `TestWriteHistory_Truncate`:30 条 × 300KB → 截断后 file size ≤ 5MB + 至少剩 1 条。
  - `TestPreviewFromMessages`:边界用例。
  - `TestDir`:路径拼接。

**测试**:`go test ./internal/skillboxdata/...` 6 PASS,build 干净。

### Phase 2 — AI 历史 controller + service

**目的**:暴露两个 HTTP 接口,前端 AI 历史能保到后端 + 从后端取回。

**改动**:
- 扩 `api-server/internal/gapi/service/ai/sai/ai.s.go`:
  - 加 import:`os`,`path/filepath`,`skillboxdata`,`skillstore`。
  - 加 `ErrEmptySourcePath` / `ErrSourcePathNotInStore`。
  - 加私有方法 `resolveSkillDirBySourcePath(sourcePath)`:校验 source_path 在 `skillstore.root` 之下 + 含 `SKILL.md`,再 `EvalSymlinks` 拿真实路径(避免 symlink 链)。
  - 加 `(s *Service) SaveHistory(sourcePath, items)`:Ensure 后 WriteHistory。
  - 加 `(s *Service) ListHistory(sourcePath)`:ReadHistory 直接返。
- 新建 `api-server/internal/gapi/controller/skillbox/caisession/save_history.a.go`:
  - `POST /api/skillbox/ai/history/save`
  - body `{source_path, items}`,空 source_path 400 / 不在 store 404 / 其它 500。
- 新建 `api-server/internal/gapi/controller/skillbox/caisession/list_history.a.go`:
  - `GET /api/skillbox/ai/history/list?source_path=...`
  - 返 `{version, items}`。`HistoryItemView` 把 `messages` 作为 `json.RawMessage` 透传(前端按需 parse)。

**测试**:`go build ./...` 干净,所有相关包测试全绿。

### Phase 3 — 前端持久化 + markdown 渲染 + 历史 UI

**目的**:AIRightPanel 数据源切到 Pinia store,改成 store 主导,加历史按钮 + 历史 Modal,markdown 渲染按"流式 <pre> → 结束 v-html"切换。

**改动**:
- 新建 `frontend/src/api/skillbox/ai-history.js`:`saveHistory` / `listHistory`(走 `http`,不 SSE)。
- 新建 `frontend/src/core/store/ai.js`(Pinia,options syntax):
  - state:`sessions{[sourcePath]:{items,updatedAt}}` / `currentSourcePath` / `saving` / `historyDialogOpen` / `historyItems` / `loadingList`。
  - actions:`hydrate` / `persistLocal` / `setCurrentSource` / `pushUser` / `pushAssistantPlaceholder` / `patchAssistant(id, patch)` / `setMessageApplied` / `setMessageRejected` / `clearCurrent` / `loadFromBackend` / `pickHistoryItem(item)` / `_append` / `_scheduleBackendSave`(800ms debounce) / `flushBackend`。
  - 双写:localStorage 即时同步;后端 800ms 防抖。
- 新建 `frontend/src/components/ai/AIHistoryDialog.vue`(Modal 列表,复用 `components/Modal.vue`,v-model 控显示;列表项 title/preview/ts/provider+model,点击 → emit pick)。
- 改 `frontend/src/components/ai/AIRightPanel.vue`:
  - 删除:`sessionStorage` 持久化 / `loadSession` / `persistSession` / 组件内 `messages ref` 直接写。
  - 新:数据源 `messages = computed(() => ai.currentMessages)`。
  - `onMounted`:`ai.hydrate()` + `ai.setCurrentSource(props.filePath)`。
  - `sendMessage`:`ai.pushUser(safeText)` + `ai.pushAssistantPlaceholder()`;流式期 `ai.patchAssistant(id, {content, status:'streaming'})`,retry 中 `ai.patchAssistant(id, {content:'', pending:true, retrying:true})`,结束 `ai.patchAssistant(id, {needsApply,content,reason,canApply,..., status:'done'})`,error `status:'error'`,truncated 同上。
  - `applyMessage` / `rejectMessage`:走 `ai.setMessageApplied/SetMessageRejected`。
  - `clearHistory`:`ai.clearCurrent()`。
  - `watch filePath`:`ai.setCurrentSource(new)`(不再 `persistSession`+`loadSession`)。
  - 模板:
    - 头部新增"历史对话"按钮(`mdi:history`,`disabled="!ai.hasSession"`)。
    - AI 消息渲染分四态:
      - `status==='sending'||'streaming'||retrying` → `<pre>` + 光标(避免 markdown-it 每帧 parse 抖动);
      - 否则 → `<div class="airp-md-body" v-html="renderContent(m)" />`;
      - reason 同理。
    - 末尾加 `<AIHistoryDialog v-model="ai.historyDialogOpen" :items :loading @pick="ai.pickHistoryItem" />`。
  - CSS:新增 `.airp-md-body`(覆盖 mono 字体 + line-height 1.55 + 段落间距 + 行内 code 样式 + 列表 padding + 外链用 `.md-external-link`)。
  - markdown 渲染函数 `renderContent(m)` / `renderReason(m)`:调 `renderMarkdownView`,流式期返空串(<pre> 已经接管)。
- 补 `frontend/src/core/i18n/{zh-CN,en-US}.js`:`skills.aiPanel.history` / `historyDialog.{title,loading,empty}`。

**测试**:`cd frontend && npm run build` 通过。

## 4. 关键不变式(必须长期保持)

- `.skill-box/` **永远不进 c.Files**:store.walkFiles / readDirFiles / BaseAdapter.Apply 三层过滤都生效。
- `.skill-box/readme.md` **永远不喂 AI**:ai_walker.go 循环里过滤 + chat_stream.a.go 默认 system 护栏两层兜底。
- link 模式下历史写入仍能用:后端用 `c.SourceDir`(已 EvalSymlinks)拼路径,写源端,与 targetDir 是否为 symlink 解耦。
- 双写失败不阻塞:localStorage 写盘失败和后端 800ms 防抖写盘失败均静默,UI 不显示错误。

## 5. 改动的文件

后端(7 个):
- `api-server/internal/skilladapter/hidden.go`(新)
- `api-server/internal/skilladapter/hidden_test.go`(新)
- `api-server/internal/skilladapter/base.go`(改:Apply 写盘过滤 + readDirFiles WalkDir 过滤)
- `api-server/internal/skilltester/ai_walker.go`(改:buildSkillMDForPrompt 过滤)
- `api-server/internal/gapi/controller/skillbox/caiprovider/chat_stream.a.go`(改:system 护栏)
- `api-server/internal/skillboxdata/skillboxdata.go`(新)
- `api-server/internal/skillboxdata/skillboxdata_test.go`(新)
- `api-server/internal/gapi/service/ai/sai/ai.s.go`(改:加 SaveHistory / ListHistory)
- `api-server/internal/gapi/controller/skillbox/caisession/save_history.a.go`(新)
- `api-server/internal/gapi/controller/skillbox/caisession/list_history.a.go`(新)

前端(5 个):
- `frontend/src/api/skillbox/ai-history.js`(新)
- `frontend/src/core/store/ai.js`(新)
- `frontend/src/components/ai/AIHistoryDialog.vue`(新)
- `frontend/src/components/ai/AIRightPanel.vue`(改:数据切到 store + markdown 渲染 + 历史按钮)
- `frontend/src/core/i18n/zh-CN.js`(改)
- `frontend/src/core/i18n/en-US.js`(改)

memory / docs:
- `docs/agent/memory/hidden-segment-filter.md`(新)
- `docs/agent/memory/MEMORY.md`(追加索引)
- `docs/agent/task/2026-07/07-14_首页AI面板升级_与_skill-box目录.md`(本文)

## 6. 验证

- `cd api-server && go build ./...` 干净
- `go test ./internal/skilladapter/... ./internal/skillboxdata/... ./internal/skillstore/... ./internal/skilltester/... ./internal/gapi/service/ai/... ./internal/gapi/service/skilltester/...` 全绿
- `cd frontend && npm run build` 干净

**手测场景(留给 `wails3 dev` 阶段)**:
1. importer 外部 zip 含 `.skill-box/history.json` → 文件树不出现该目录;apply (copy) 后目标端无 `.skill-box/`。
2. apply (symlink) → targetDir 是 symlink;点 AI 面板的"历史"按钮 → 后端从源端读 + 本地 localStorage 同时显示列表。
3. AI 走查 → model SDK 日志 / 抓包:`messages` 里无 `.skill-box/readme.md` 内容。
4. AI 流式输出 markdown:`/skillbox/ai/chat` 返回的代码块 / 行内 code / 列表 / 链接在前端正确渲染。
5. 关浏览器 → 重开 → 历史按钮点开 → 历史仍在。
6. 同一 skill 切换不同文件 → 历史按 sourcePath 维度隔离,切回仍有旧历史。

## 7. 风险与回退

1. **`HasHiddenSegment` 误杀**:连 `.github/` / `.vscode/` 也跳;项目已有同规则,仅扩到还没过滤的位置(`readDirFiles`)。回退:仅关掉 `ai_walker.go` 那一行,后端其它保留。
2. **5MB 截断误删**:FIFO 仅在超限时按 ts 升序删一条;用户不感知。回退:把 `MaxHistorySize` 调到 50MB,或禁掉截断直接返错。
3. **chat_stream system 注入被覆盖**:用户带显式 system role 时**不**注入。回退:移除 chat_stream.a.go 的 system 注入,只留 ai_walker + Apply 两层。
4. **后端写盘失败**:controller 返 4xx/5xx;前端 `flushBackend` 静默,UI 不阻塞。
5. **边流 markdown 卡顿**:50KB 上限 + rAF 节流;若卡则降级为"流式 <pre>,点消息才展开 markdown"。
6. **link 模式 PII**:核心前提是 AI 喂 prompt 永远走 `buildSkillMDForPrompt`(已过滤)+ chat_stream system 护栏;若以后放开"AI 直接读目录",必须显式再过滤 `.skill-box/`,不能依赖 walkFiles。

## 8. 提交

每个 Phase 一次 commit,commit + push 全程自动。

## 9. 后续

- 5MB 截断:目前只是 FIFO 删,可考虑后续加"按 token 数 / 按 model"精细化。
- `flushBackend` 失败时应让用户手动重试(目前静默)。
- 历史条目可加"导出 JSON / 分享链接"。
- 多模态:blades 内置 `FilePart` / `DataPart`,后续可支持图片输入 + 历史里存图。
