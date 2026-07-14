---
name: hidden-segment-filter
description: skill-box 隐藏目录段过滤(.skill-box/、.git/ 等),Phase 0 统一用 HasHiddenSegment
metadata:
  node_type: memory
  type: feedback
  originSessionId: 67d311e2-2466-4c13-8f82-d0a186f25f9f
---
**规则**:任一段以 `.` 开头 → 视为隐藏。

**实现位置**:`api-server/internal/skilladapter/hidden.go#HasHiddenSegment`。

**生效点**(2026-07-14 起):
1. `skilladapter/base.go#Apply`(copy 模式写盘前过滤)
2. `skilladapter/base.go#readDirFiles`(importer 加载时,WalkDir 整目录 SkipDir + 文件级兜底)
3. `skilltester/ai_walker.go#buildSkillMDForPrompt`(AI prompt 拼装前过滤,防 PII)
4. `caiprovider/chat_stream.a.go`(追加 system 护栏,告诉 AI 不要主动读 `.skill-box/`)

**已天然过滤**:`skillstore.walkFiles` + `skillstore.listEmptyDirs`(本来就是这规则,`walkFiles` 一行 return nil)。

**为什么用同一函数**:`walkFiles` 是逐文件 seg 切,我们用同一个 `HasHiddenSegment` 串起来,避免三处分叉。

**为什么不放 `pkg/fsutil`**:`skilladapter` 是语义最合适的归属,放通用 fsutil 反而过于通用。

**How to apply**:
- 任何"把目录内容暴露给 AI 或写到目标工具"的新代码路径,先调 `skilladapter.HasHiddenSegment`
- 不要再写第二份"任何以 . 开头就跳"的逻辑 — 找 `HasHiddenSegment`
- `.skill-box/` 是 skill-box 自有运行时目录,目标工具(Claude / Codex / Trae / OpenCode)不需要它
