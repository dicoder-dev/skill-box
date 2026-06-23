# docs/agent — Claude Agent 协作约定

> 这套目录是 **给 Claude / Claude Code 用的"协作上下文"**,不是给人看的业务文档。
> `docs/project/` 是项目本身的业务文档;`docs/agent/` 是 AI 工作流。

## 一句话原则

> **根 `CLAUDE.md` 只放极简入口和"何时读什么";** 真正的大块内容按需在 `docs/` 下,
> 让 Claude 只在需要时按 `@docs/agent/xxx` 显式加载,**避免一次性吞下整个仓库**。

## 目录结构

```
docs/
├── project/                        # 项目本身的业务文档(给人看的)
│   ├── README.md                   # 文档索引 + 维护约定
│   ├── 项目架构.md                 # 当前实现的架构说明
│   ├── 需求规划.md                 # 设计文档(决策 + 规范)
│   ├── 进度.md                     # 状态文档(做了 / 待做)
│   └── 开发报告.md                 # 过程文档(踩坑 / 测试 / 回流)
│
└── agent/                          # AI 协作上下文(给 Claude 看的)
    ├── README.md                   # 本文件 —— 分层契约
    ├── ONBOARDING.md               # 新会话冷启动清单(第一份必读)
    │
    ├── memory/                     # 跨会话长期记忆
    │   ├── MEMORY.md               # 索引(必读,Claude 默认会扫这个)
    │   ├── user_*.md               # 用户画像(角色 / 偏好 / 知识背景)
    │   ├── feedback_*.md           # 行为反馈(做过的 / 别再做的 + why)
    │   ├── project_*.md            # 项目状态(目标 / 期限 / 在做什么)
    │   └── reference_*.md          # 外部资源指针(看板 / 文档 / 平台)
    │
    ├── task/                       # 每个对话 / 任务的过程文件
    │   ├── README.md               # 任务文件命名/结构规范
    │   ├── _template.md            # 任务文件模板
    │   └── YYYY-MM-DD_<主题>.md    # 单次任务的过程记录
    │
    └── project/                    # AI 工作流要遵守的"项目规则"
        ├── README.md               # 规则总入口
        ├── architecture.md         # 项目架构(Claude 视角的关键信息)
        ├── conventions.md          # 命名 / 目录 / 文件规范
        ├── workflow.md             # 开发 / 提交 / 验证流程
        └── tech_stack.md           # 技术栈与版本约束
```

## 按需加载策略

| 场景                              | Claude 应该读什么                                              |
| --------------------------------- | -------------------------------------------------------------- |
| 用户开头打招呼 / 无具体任务       | `docs/agent/memory/MEMORY.md`(拿历史偏好 + 用户画像)          |
| 用户给一个具体任务                | `docs/agent/task/README.md` + 同主题最近的 task 文件           |
| 改 Go 后端代码                    | `docs/agent/project/conventions.md` + `architecture.md`        |
| 改 Vue 前端代码                   | `docs/agent/project/conventions.md` + `tech_stack.md`          |
| 提 PR / 改 commit                 | `docs/agent/project/workflow.md`                               |
| 用户问"为什么之前那样做"          | `docs/agent/memory/feedback_*.md` + `docs/agent/task/` 同主题  |
| 第一次进入项目                    | 全部 `docs/agent/project/*` + `MEMORY.md`                      |

## 强制规则(给 Claude 自己看的)

1. **不要把所有 `docs/` 一次塞进上下文**。需要什么,显式 `@文件路径` 或 `Read` 指定。
2. **根 `CLAUDE.md` 是入口,不是说明书**。任何超过 200 行的内容必须搬到 `docs/` 下。
3. **新增任务 → 在 `docs/agent/task/` 下建一份 `<日期>_<主题>.md`**,
   任务结束后在文件里追加 `## 总结` 小节,不要新建独立的"总结文件"。
4. **学到新东西 → 立刻决定落点**:
   - 改的是 Claude 的行为 → `memory/feedback_*.md`
   - 改的是用户信息 → `memory/user_*.md`
   - 改的是项目本身状态 → `memory/project_*.md` 或 `docs/project/`
   - 只是当前任务的事 → 留在 `task/` 文件里,不外溢
5. **业务文档(`docs/project/`)和 AI 上下文(`docs/agent/`)不要混**。
   业务文档是人类维护的产品/技术文档;AI 上下文是机器读的协作上下文。