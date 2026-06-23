# ONBOARDING.md — 新会话冷启动清单

> **你是刚被唤醒的 Claude,上下文为零。** 这份文件告诉你:
> 1. 现在该按什么顺序读什么
> 2. 读完应当掌握哪些事实
> 3. 用户给任务时,你该怎么响应
>
> 这份文件 **只读一次** —— 读完按下面"完成确认"清单逐项打钩,即可开始正常工作。

---

## 0. 第一步(强制):读根 `CLAUDE.md`

```
Read CLAUDE.md
```

读完你应该知道:

- 项目一句话(Wails v3 + Gin + Vue 3 双形态)
- 协作铁律(删除/强推必须确认;其他无需确认;中文沟通;称呼"靓仔";注释简体中文)
- 必读文档表(何时读什么)

---

## 1. 第二步:加载长期记忆

```
Read docs/agent/memory/MEMORY.md
```

然后按 MEMORY.md 的索引,**只 `Read` 当前会话真正需要的那几份**:

| 场景 | 读这些 |
| --- | --- |
| 默认(任何会话) | `user_profile.md` + `feedback_communication.md` + `feedback_safety.md` |
| 用户提到外部系统 / 链接 | 加读 `reference_external.md` |
| 用户问"项目进度 / 在做什么" | 加读 `project_state.md` |
| 用户提到工具偏好(图片 / 搜索) | 加读 `feedback_tools.md` |
| 准备提交代码时 | 加读 `feedback_auto_commit.md` |

**不要**一次性 `Read` 全部 memory 文件。

---

## 2. 第三步:确认上下文边界

在动手前,先在心里回答这几个问题:

| 问题 | 答案来源 |
| --- | --- |
| 我在哪个项目?根目录在哪? | `pwd` 或 CLAUDE.md |
| 用户这次想干什么? | 当前消息 |
| 这是闲聊还是具体任务? | 当前消息 |
| 有没有相关的最近 task 文件? | `ls docs/agent/task/`,**按月份目录(如 `2026-06/`)只看最近 3 份** |

---

## 3. 分支:用户给了什么?

### 分支 A:用户给了具体任务

1. `Read docs/agent/task/README.md` —— 学会任务文件的结构 + **月份目录**约定
2. `ls docs/agent/task/YYYY-MM/`(本月的目录)—— 看最近 5 份文件名,**挑主题最相关的 1-2 份** 读
3. 按 `task/_template.md` **立刻建一份本次任务文件**:
   - 完整路径:`docs/agent/task/YYYY-MM/YYYY-MM-DD_<主题>.md`
   - 月份目录不存在时先 `mkdir -p`
   - 填"需求 / 任务列表 / 执行进度"前三节
4. 改代码前,按 CLAUDE.md 必读表加载对应 `docs/agent/project/*.md`
5. 开始干

### 分支 B:用户在做项目探索("这个项目是什么")

1. `Read docs/agent/project/README.md` —— 看 AI 视角的项目规则目录
2. `Read docs/project/README.md` —— 看人视角的项目业务文档目录
3. `Read docs/project/项目架构.md` —— 详细架构
4. 不需要读 memory(不需要偏好,只需要事实)
5. 回答时直接给结论,不要复述读过的内容

### 分支 C:用户只是闲聊 / 一句话问题

- **也要建任务文件**(`docs/agent/task/YYYY-MM/YYYY-MM-DD_<主题>.md`),
  但用最简版:用户原话 + Claude 答复 + 任何决定/约定,不需要"问题与方案 / 总结"等大节
- 一两句话答完
- 闲聊的价值在于保留上下文,几个月后回看能记起"当时我们聊过什么 / 定下了什么"
- **唯一例外**:纯粹的"XX 文件在哪" / "YY 命令怎么用"这类查询性闲聊,可以省略任务文件

---

## 4. 干活时的强制纪律

| 触发 | 动作 |
| --- | --- |
| 改 Go 业务代码 | 先 `Read docs/agent/project/conventions.md` |
| 改 Vue 业务代码 | 先 `Read docs/agent/project/conventions.md` + `tech_stack.md` |
| 改启动流程 / 配置 | 先 `Read docs/agent/project/architecture.md` |
| 提 PR / commit | 先 `Read docs/agent/project/workflow.md` |
| 用户透露新偏好 / 纠正你 | 立即更新对应 `feedback_*.md`,含 why |
| 用户透露项目状态变化 | 立即更新 `project_state.md` |
| 任务结束 | 填 task 文件"总结"小节,跑"提炼清单";确认文件在 `docs/agent/task/YYYY-MM/` 下 |
| 完成一个功能点 / 修复 | 按 `feedback_auto_commit.md` 自主 commit(**不再用 hook**) |

---

## 5. 完成确认(读完这份文件后,逐项打钩)

读完 ONBOARDING.md 之后,你应当能在心里回答:

- [ ] 我知道项目是什么(Wails + Gin + Vue 3 双形态)
- [ ] 我知道根 `CLAUDE.md` 在哪、内容是什么
- [ ] 我知道 memory 在哪、按什么规则加载
- [ ] 我知道 task 文件怎么命名、怎么写
- [ ] 我知道 project/ 下哪份文件管什么
- [ ] 我知道什么操作必须确认、什么可以直接做
- [ ] 我知道用户称呼、沟通语言、注释语言
- [ ] 我知道任务结束后该提炼什么到哪个文件
- [ ] 我知道完成功能点后要自主 commit(不靠 hook)

8 项全部打钩 → **可以开始工作了**。

如果有任何一项打不上钩,**停下来 `Read` 对应文件补齐**,不要猜。