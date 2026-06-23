# Memory 索引(Claude 默认扫描)

> **必读入口**。每个会话开头 Claude 应当读这份 MEMORY.md 来加载历史上下文。
> 不要一次性把这里列的文件都 `Read` 进来,**按需读**。

## User(用户画像)

| 文件 | 用途 | 何时读 |
| --- | --- | --- |
| [user_profile.md](./user_profile.md) | 用户是谁、什么角色、技术栈背景 | 任何会话开头 |

## Feedback(行为反馈)

| 文件 | 用途 | 何时读 |
| --- | --- | --- |
| [feedback_communication.md](./feedback_communication.md) | 沟通偏好(语言 / 称呼 / 节奏) | 任何会话开头 |
| [feedback_safety.md](./feedback_safety.md) | 高风险操作必须确认 | 涉及删除 / 推送 / 强推 |

## Project(项目状态)

| 文件 | 用途 | 何时读 |
| --- | --- | --- |
| [project_state.md](./project_state.md) | 当前在做什么 / 关键决策 / 期限 | 任何会话开头 |

## Reference(外部资源指针)

| 文件 | 用途 | 何时读 |
| --- | --- | --- |
| [reference_external.md](./reference_external.md) | 外部系统 / 文档 / 看板链接 | 用户提到外部资源时 |

## 维护规则

### 何时新增 / 更新

| 触发场景 | 操作 |
| --- | --- |
| 用户透露新身份 / 角色 / 偏好 | 更新对应 `user_*.md` / `feedback_*.md` |
| 用户纠正 Claude 的做法 | 新增 / 更新 `feedback_*.md`,**必须包含 why** |
| 项目阶段 / 目标变化 | 更新 `project_state.md` |
| 用户提到外部系统 / 文档地址 | 更新 `reference_*.md` |
| Claude 自己推导出的代码事实 | **不要写进 memory**(读代码即可) |

### 命名约定

- `user_<主题>.md` / `feedback_<主题>.md` / `project_<主题>.md` / `reference_<主题>.md`
- 一个文件一个主题;主题稳定后不再开新文件
- 不写"日常笔记 / 当前任务进度"(那些放 `docs/agent/task/`)

### 文件体例

每条 memory 文件:

```markdown
---
name: 简短名称
description: 一句话定位(用于判断何时相关)
type: user | feedback | project | reference
---

正文…

# feedback / project 类型必须含:
**Why:**(为什么)
**How to apply:**(何时 / 在哪触发)
```

### 不要写进 memory

- 代码结构 / 命名规范 / 目录树(读代码即可)
- git 历史 / 谁改了什么(`git log` 权威)
- 调试方案 / 修复 recipe(修复在代码里)
- 临时任务细节(放 `task/`)
- CLAUDE.md 里已有的内容