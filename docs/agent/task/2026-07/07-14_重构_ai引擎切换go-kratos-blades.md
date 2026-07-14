# aiengine 全量切换到 go-kratos/blades

**日期:** 2026-07-14
**状态:** 已完成

## 1. 需求

用户决定把项目自研的 AI 引擎(`api-server/internal/aiengine/`)全量替换为
`github.com/go-kratos/blades` 官方框架。

理由:
- 自研 openai.go / anthropic.go 用 `net/http` + `bufio` 手写 SSE,维护成本高
- 加新协议(Gemini / Cohere / 本地模型)要再写一套
- 没有 Agent / Chain 抽象,业务侧想做"先审文档再翻译"这种多步流水线得自己包

用户已确认"项目未上线,选 A 方案全量切换,不需要兼容旧接口"。

## 2. 决策

| 决策点 | 选择 |
|---|---|
| 切换范围 | A:全量切到 blades |
| 对外接口 | 全切到 blades 原生类型 `*Message` / `ModelProvider` / `Generator` |
| Preset 怎么办 | 仍用 `Preset` struct,内部走 `blades.UserMessage/SystemMessage` 构造 |
| 前端 SSE 协议 | 不动,后端做 `*ModelResponse` → `{kind,text,usage}` 协议转换 |
| API key 来源 | 仍走 settings KV(`ai:<name>:api_key`),构造 model 时注入 |
| TestConnection | 保留,改用 blades `model.NewStreaming` 探针 |
| Chain/Agent 示例 | 写 `example/chain.go`,演示 "safety_check → translate_skill" 的 `flow.NewSequentialAgent`,不挂 controller |

## 3. 实施步骤

### 3.1 拉依赖

- `go get github.com/go-kratos/blades@latest` → v0.5.0
- `go get github.com/go-kratos/blades/contrib/openai@latest` → v0.3.0
- `go get github.com/go-kratos/blades/contrib/anthropic@latest` → v0.3.0
- 间接依赖:openai-go/v3、anthropic-sdk-go、jsonschema-go、blades/kit

### 3.2 删旧 aiengine

删除:
- `aiengine/openai.go` / `anthropic.go` / `manager.go` / `types.go` / `aiengine_test.go`

新建:
- `aiengine/engine.go` — Engine(替代 Manager,选 provider + 拼 key + 出 `blades.ModelProvider`)
- `aiengine/adapter.go` — `registerDefaults` 把 KindOpenAI / KindAnthropic / KindOpenAICom 注册到 blades 工厂
- `aiengine/preset.go` — Preset 迁到 `blades.UserMessage` / `SystemMessage`
- `aiengine/example/chain.go` — `flow.NewSequentialAgent` 示例
- `aiengine/engine_test.go` — Engine + 流的单测

### 3.3 改 6 个调用方

| 文件 | 改动 |
|---|---|
| `caiprovider/chat_stream.a.go` | `Messages` 改 `[]*blades.Message`,for 循环 `stream.Recv()` → `for resp, err := range chat.Stream`,SSE 输出协议不变 |
| `caiprovider/{list_presets,create,update,delete,get,set_key,list_providers,test_provider}.a.go` | `sai.NewManager` → `sai.NewEngine`,`mgr` 改名 `eng` |
| `sai/ai.s.go` | `manager` 字段 → `engine`;`Chat` 返回 `*ChatResult` 含 `iter.Seq2[*blades.ModelResponse, error]`;`TestConnection` 用 `blades.UserMessage` 探针 |
| `sai/ai.s_test.go` | 重写,移除 `aiengine.ChatRequest/StreamEvent/RoleUser` 等旧类型引用,改用 `*blades.Message` |
| `skilltester/ai_walker.go` | `Build` 闭包返回 `blades.ModelProvider`,`RunAIWalk` 用 `prov.NewStreaming` 替代 `prov.Chat` + channel |
| `sskilltest/skilltest.s.go` | `mgr` → `eng`,`NewManagerForTester` → `NewEngineForTester`,`buildForAI` 现场取 settings key 喂给 `BuildFromConfig` |
| `sskilltest/skilltest.s_test.go` | 重写,移除 `NewManagerForTester` 引用 |
| `cskilltest/run_skill_test.a.go` | `NewManagerForTester` → `NewEngineForTester`,删 `aiengine` import |

### 3.4 测试

- `go build ./...` 干净通过
- `go test ./internal/aiengine/... ./internal/skilltester/... ./internal/gapi/service/ai/... ./internal/gapi/service/skilltester/... ./internal/skillapp/...` 全绿
- `pkg/task` 在 `go test ./...` 时 600s 超时,但它跑的是 robfig/cron 自循环,跟本次改造无关,不动

## 4. API 形态对照

| 旧 aiengine | 新 blades |
|---|---|
| `aiengine.Provider` interface | `blades.ModelProvider` interface |
| `aiengine.ChatRequest{Provider, Model, Messages, ...}` | `blades.ModelRequest{Messages, Instruction, ...}` |
| `aiengine.Message{Role, Content}` | `blades.Message{Role, Parts: []Part{TextPart{Text}}}` |
| `aiengine.StreamEvent{Kind, Text, Err, Usage}` | 由 controller 现场映射:`Message.Text()` → chunk,`Status==Completed` → done |
| `Provider.Chat(ctx, req, key, chan<- StreamEvent) error` | `ModelProvider.NewStreaming(ctx, req) Generator[*ModelResponse, error]` |
| `aiengine.Manager` | `aiengine.Engine` |
| `Manager.Build(row) (Provider, key, error)` | `Engine.Build(row) (ModelProvider, key, error)` |

## 5. 关键设计点

- **API key 在 NewModel 时注入**:`openai.NewModel(model, openai.Config{APIKey, BaseURL})` 一次给定,后续 `NewStreaming` 不再传。
- **流式收尾靠 `Status==Completed`**:blades 的 generator 在 `choiceToResponse(... StatusCompleted)` 写一帧后关闭;之前我们的 fakeProvider 也按这个语义测。
- **SSE 协议保留**:前端零改动,controller 在 `writeSSEChunk/Error/Done` 三个函数里把 `*blades.ModelResponse` 转 `{kind, text, err, usage}`。
- **Preset 占位符**:`{var}` 替换,缺失保留原样,跟旧版一致。
- **Engine 与 SecretStore 解耦**:第三方 / 单测可以通过 `Engine.Register(kind, ModelFactory)` 注入自己的 provider;`secretAdapterForTester` 复用 settings。

## 6. 后续可扩展

- **Chain / Agent 业务化**:`example/chain.go` 已搭好架子,业务侧按需把多步流水线挂到 controller(比如 `POST /api/skillbox/ai/chain/safety-then-translate`)
- **Tool 调用**:blades 有 `tools.Tool` 抽象,加上后 AI 能调用本地工具(读文件 / 跑脚本 / 操作 store 等)
- **Memory / Session**:`blades.Session` + `blades.NewRunner(agent, ...)` 提供多轮记忆,适合做"AI 助手"长期对话场景
- **多模态**:blades 内置 `FilePart` / `DataPart`,后续可以加图片/语音支持

## 7. 验证

- `cd api-server && go build ./...` 干净通过
- 相关单测全绿
- 桌面端手测待 `wails3 dev` 时跑(翻译 Skill / preset 列表 / 测试 provider / AI 走查)

## 8. 风险

- `blades.WithAPIKey` 这种 option 不存在(已确认),API key 在 NewModel 时注入
- 7 个 caiprovider controller 在 stash 期间被回滚成旧版本,已通过 `git rm` + 重新 Write 全部修正;最终 diff 干净
- 流式 done 帧:blades 的 `Status==Completed` 写完最后帧后会 yield `nil` 关闭 generator,controller 检测到这个再写 `data: [DONE]\n\n`
