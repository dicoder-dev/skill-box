---
name: Claude 自主提交代码
description: 功能点完成时由 Claude 自行判断并 commit,不再使用 PostToolUse hook
type: feedback
---

## 规则

每次完成或修复完一个功能点(用户认可的"完成"),Claude 必须:

1. **自行分析改动**:`git status` + `git diff` 看改了哪些文件
2. **自行生成 commit 信息**:中文祈使句,简洁,体现区域(后端 / 前端 / build / 文档)
3. **自行 `git add <具体文件>` + `git commit`**:不要 `git add -A`,逐个点名相关文件
4. **不强推、不 amend、不 --no-verify**:遵守 `feedback_safety.md` 的红线
5. **告知用户**:commit 完成后简短告知用户提交了什么

## 何时提交(完成判定标准)

满足以下任一条件即可提交:

- 用户明确说"完成 / 好了 / 提交吧 / commit 吧"
- 一个完整功能点跑通(测试通过 / 端到端验过)
- 一个 bug 修复完成(根因修复 + 验证)
- 一个独立的文档 / 配置改动完成
- 一个独立的小重构完成

**不**满足时(不该提交):

- 改动还没跑测试 / 没验证
- 改到一半,中途临时中断
- 用户只是问问题,没让做改动
- 改动尚未"成片"(只剩一两行收尾)

## 提交信息风格(沿用仓库历史)

```
<区域>: <简短中文祈使句>
```

示例(从近期 commit 提炼):

- `修复接口样式`
- `迁移 ginp 改动`
- `web: 同步 embed 目录`
- `docs: 补 ONBOARDING.md`
- `agent: 关闭 PostToolUse hook,改由 Claude 自主提交`

**不**强制 conventional-commit 前缀;区域可省略。

## Why

用户已明确指示:git 提交不再用 hook,改由 Claude 自主分析。

之前用 `PostToolUse` hook 的问题:

- **时机过细**:`Edit/Write/MultiEdit` 每次都触发,会把一次完整功能切成 N 个小提交
- **信息不准**:hook 看的是单次工具调用的 diff,没人分析"这是不是完成了一个功能点"
- **粒度混乱**:有时一半代码一次提交、有时一个完整功能一次提交,历史不可读

让 Claude 自主判断可以解决这三点:只在"完成"时提交、commit 信息是分析后的总结、粒度按功能点而非按工具调用。

## How to apply

任何时候,Claude 完成功能/修复后:

1. `git status` 看有哪些文件变动
2. `git diff <file>` 看每个文件的实质改动(必要时)
3. 决定 commit 信息(中文祈使句 + 区域)
4. `git add <file1> <file2> ...` —— **不要 `git add -A`**,逐个点名
5. `git commit -m "..."`(用 HEREDOC,带 Co-Authored-By)
6. `git status` 确认提交成功
7. 简短告知用户

如果改动跨多个功能点,分多个 commit(每个 commit 一个功能点)。

## 与 `docs/agent/task/` 的关系

- task 文件的"总结"小节里**记一笔** "已 commit `<hash>`"
- 不在 commit 信息里提 task 文件路径(那属于协作元数据,不该污染代码提交历史)