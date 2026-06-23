<script setup>
// SkillTitle - 显示 SKILL.md 的首行 # 标题。
//
// 用法:
//   <SkillTitle :source-path="f.source_path" :fetcher="fetchTitle" />
//
// props.fetcher 接收 sourcePath 异步返回 title 字符串(后端读文件+前端解析);
// fetcher 内部走 platform.fs.readText 抽象,失败返空串。
//
// 标题为空时不渲染(<> 避免占位造成抖动),由调用方决定留白。

import { ref, watch, onMounted } from 'vue'

const props = defineProps({
  sourcePath: { type: String, required: true },
  fetcher: { type: Function, required: true },
})

const title = ref('')
const loading = ref(false)
let reqId = 0

async function load() {
  const myId = ++reqId
  loading.value = true
  try {
    const t = await props.fetcher(props.sourcePath)
    // 防止过期请求覆盖新值
    if (myId === reqId) title.value = t || ''
  } finally {
    if (myId === reqId) loading.value = false
  }
}

onMounted(load)
watch(() => props.sourcePath, load)
</script>

<template>
  <div v-if="title" class="skill-title">
    <span class="skill-title-label">{{ title }}</span>
  </div>
</template>

<style scoped>
.skill-title {
  margin: 2px 0 4px;
  font-size: 12px;
  line-height: 1.4;
  color: var(--text-dim);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.skill-title-label {
  /* 标题是从 SKILL.md 第一行取的描述,可能很长 → 截断 */
  display: inline-block;
  max-width: 100%;
}
</style>
