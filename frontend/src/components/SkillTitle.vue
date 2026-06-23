<script setup>
// SkillTitle - 显示 SKILL.md 的元信息(description / title)。
//
// 用法:
//   <SkillTitle :source-path="f.source_path" :fetcher="fetchTitle" />
//
// props.fetcher 接收 sourcePath 异步返回 { title, description };
// 优先展示 description(更有信息量),空时回落到 title(首行 # 标题),
// 两者都没有就不渲染,避免占位造成抖动。
//
// 后端实现走 platform.fs.readText(source_path + '/SKILL.md'),
// 前端用 parseSkillMeta 抽 frontmatter.description + body 首个 # 标题。

import { ref, watch, onMounted } from 'vue'

const props = defineProps({
  sourcePath: { type: String, required: true },
  fetcher: { type: Function, required: true },
})

const text = ref('')
let reqId = 0

async function load() {
  const myId = ++reqId
  try {
    const meta = await props.fetcher(props.sourcePath)
    if (myId !== reqId) return
    // description 优先,空时回落到 title(# 首行)
    text.value = (meta?.description || meta?.title || '').trim()
  } catch (_) {
    if (myId === reqId) text.value = ''
  }
}

onMounted(load)
watch(() => props.sourcePath, load)
</script>

<template>
  <div v-if="text" class="skill-title">
    <span class="skill-title-label" :title="text">{{ text }}</span>
  </div>
</template>

<style scoped>
.skill-title {
  margin: 2px 0 4px;
  font-size: 12px;
  line-height: 1.5;
  color: var(--text-dim);
  /* 允许 2 行,避免一行截断太短看不全 */
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  word-break: break-word;
}

.skill-title-label {
  display: inline;
}
</style>
