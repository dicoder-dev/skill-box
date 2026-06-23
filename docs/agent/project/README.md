# docs/agent/project — AI 工作流要遵守的项目规则

> 这些是 **给 Claude 看的"项目规则"**,不是给人看的业务文档。
> 业务文档在 `docs/project/`(项目架构/需求规划/进度/开发报告)。
> 本目录的差别:只装那些"Claude 改代码时必须遵守的规则",与具体业务无关。

## 目录索引

| 文件 | 定位 | 何时读 |
| --- | --- | --- |
| [README.md](./README.md) | 本文件,目录索引 + 何时读什么 | 第一次进入 |
| [architecture.md](./architecture.md) | 架构关键信息(Claude 视角,精简版) | 改后端/前端代码前 |
| [tech_stack.md](./tech_stack.md) | 技术栈 + 版本约束 | 装依赖 / 升级版本前 |
| [conventions.md](./conventions.md) | 命名 / 目录 / 文件规范 | 任何写代码场景 |
| [workflow.md](./workflow.md) | 开发 / 提 PR / 验证流程 | 提交前 |

## 何时读什么的强约束

- 改 **任意 Go 文件** → 必须先 `@conventions.md`,再按需读 `architecture.md`
- 改 **任意 Vue 文件** → 必须先 `@conventions.md`,再按需读 `tech_stack.md`
- 写 **commit / PR** → 必须先 `@workflow.md`
- 第一次进入项目(新会话 + 用户没指定子目录)→ 全部按需加载一次,缓存到当前会话

## 与 `docs/project/` 的边界

| 维度 | `docs/project/` | `docs/agent/project/` |
| --- | --- | --- |
| 读者 | 项目内外的工程师 | Claude / Claude Code |
| 内容 | 项目是什么 / 怎么搭 / 怎么演进 | Claude 改代码必须遵守的规则 |
| 维护 | 人工维护,跟业务一起演进 | Claude 维护,按 Claude 视角精简 |
| 体例 | 详细、含代码示例、可读 | 列表式、规则化、便于机器消费 |

两者 **不重复**:`docs/project/项目架构.md` 写"为什么这样设计",
`docs/agent/project/architecture.md` 只写"Claude 改这里时要知道的关键节点"。