---
name: ai-history-sourcepath-prop-drilling
description: AI 历史保存的关键修复,4 层 prop drilling source_path;v1 误用 filePath 后端 404 是 silent bug
metadata:
  node_type: memory
  type: feedback
  originSessionId: 67d311e2-2466-4c13-8f82-d0a186f25f9f
---
**坑**:AIRightPanel 接的 `filePath` 是文件相对路径(如 `"SKILL.md"`),**不是**磁盘绝对路径 `source_path`。但 store 用作 key + 传给后端校验 → resolveSkillDirBySourcePath 永久 404,而前端 `flushBackend catch(_){}` 静默吞掉错,**用户看不见历史保存**。

**修复路径**:`SkillsView → SkillFileInlinePanel → CodeViewer(2 处 AIRightPanel)→ AIRightPanel` 每层加 `sourcePath` prop。

```js
// SkillsView
const currentSkillSourcePath = computed(() =>
  current.value?._full?.canonical?.source_path
  || current.value?._full?.source_path || ''
)

// AIRightPanel
const props = defineProps({ sourcePath: { type: String, default: '' }, ... })
watch(() => props.sourcePath, (sp) => ai.setCurrentSource(sp || ''), { immediate: true })
```

**判断口诀**:
- prop 名 `filePath` / `path` / `relPath` → 大概率是文件相对路径,**别**当 source 用
- prop 名 `sourcePath` / `source` / `absPath` → 大概率是磁盘绝对路径,可作 source_path
- `current._full?.canonical?.source_path || current._full?.source_path` 是项目里最常见的"绝对 source_path"兜底链(get_skill?full=true 时返)

**How to apply**:
- 新组件接"绝对 source_path"时,**显式 prop 命名 `sourcePath` / `absPath`**,与 filePath 区分
- 任何"前端 → 后端校验 source 在 store.root 下"链路,都走这 4 层 prop drilling,**别**让 controller 自己再发一次 GET 反查(waste round trip,且 filePath 不是唯一 key)
- catch (_) 静默吞错是 silent failure 大忌;任何"看似可失败但不致命"的调用,要么显式 toast 要么打 log
