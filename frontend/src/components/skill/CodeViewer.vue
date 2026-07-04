<script setup>
// CodeViewer - 技能包内单文件预览/编辑器
//
// 三种渲染分支:
//   1. Markdown(.md / .markdown)→ renderMarkdownView 渲染(只读预览)
//   2. 纯文本 / 代码(.py / .js / .json / ... 其它所有文本)→ 纯文本展示(Commit 1 实现)
//      后续 Commit 3 接 Monaco 只读,Commit 4 加编辑
//   3. 二进制(.png / .jpg / .pdf / .zip / ...)→ 兜底"不支持预览" + "在文件夹打开"
//
// Commit 2 实现:markdown 渲染 + 纯文本 + 二进制兜底(简化版,Commit 5 加 i18n + 大文件提示)。
// 不实现 Monaco(Commit 3) / 编辑(Commit 4)。
//
// 2026-07-04 增:首页技能文件浏览器(Commit 2)。

import { computed } from 'vue'
import IconPark from '@/components/IconPark.vue'
import { renderMarkdownView } from '@/core/utils/markdown_view.js'
import { handleExternalClick } from '@/core/utils/external_link.js'
import { platform } from '@/platform'

const props = defineProps({
  // 文件相对路径
  path: { type: String, default: '' },
  // 文件内容
  content: { type: String, default: '' },
})

const emit = defineEmits([])

// 文件后缀
const ext = computed(() => {
  if (!props.path) return ''
  const idx = props.path.lastIndexOf('.')
  return idx >= 0 ? props.path.slice(idx + 1).toLowerCase() : ''
})

const isMarkdown = computed(() => ['md', 'markdown'].includes(ext.value))

// 二进制扩展名(图片 / pdf / 压缩包)
const BINARY_EXTS = [
  'png', 'jpg', 'jpeg', 'gif', 'webp', 'svg', 'bmp', 'ico',
  'pdf',
  'zip', 'tar', 'gz', 'tgz', '7z', 'rar',
]
const isBinary = computed(() => BINARY_EXTS.includes(ext.value))

// 大文件阈值(展示提示用,实际不加载 Monaco 是 Commit 3 之后)
const LARGE_FILE_BYTES = 1024 * 1024
const isLarge = computed(() => (props.content || '').length > LARGE_FILE_BYTES)

const fileName = computed(() => {
  if (!props.path) return ''
  return props.path.slice(props.path.lastIndexOf('/') + 1)
})

// markdown 渲染(共用 SkillsView 主区的渲染器,行为一致)
const renderedMd = computed(() => isMarkdown.value ? renderMarkdownView(props.content || '') : '')

// markdown 容器点击委托
function onMdClick(e) {
  handleExternalClick(e)
}

// "在文件夹打开"按钮
async function openInFolder() {
  // 拼绝对路径:store_root + skill.group_path + skill.name + / + path
  // 当前组件不持有 store_root,改由父级注入;先用最简实现 - 调 platform.fs.reveal
  // 让后端按 path 自动拼绝对路径(后端 fsutil 接受任意路径)
  if (!props.path) return
  try {
    // fsutil.reveal 接受完整绝对路径;父组件(SkillFileDrawer)持有 skill 信息,
    // 此处 emit 给它去拼绝对路径
    // 这里先做"noop 按钮"作为占位,Commit 4 编辑保存时一并接入"在文件夹打开"
  } catch (e) {
    /* 静默 */
  }
}
</script>

<template>
  <div class="code-viewer">
    <!-- 二进制兜底 -->
    <div v-if="isBinary" class="cv-binary">
      <IconPark icon="mdi:file-document-outline" width="56" height="56" />
      <p class="cv-binary-title">{{ fileName }}</p>
      <p class="cv-binary-hint">二进制文件(.{{ ext }})不支持在线预览</p>
      <p class="cv-binary-hint">Commit 5 将支持"在文件夹打开"</p>
    </div>

    <!-- Markdown 渲染 -->
    <div
      v-else-if="isMarkdown"
      class="cv-md md-body"
      v-html="renderedMd"
      @click="onMdClick"
    />

    <!-- 大文件提示(Commit 5 加 "在文件夹打开") -->
    <div v-else-if="isLarge" class="cv-large">
      <IconPark icon="mdi:file-alert-outline" width="56" height="56" />
      <p class="cv-large-title">{{ fileName }}</p>
      <p class="cv-large-hint">文件过大({{ Math.round((content || '').length / 1024) }} KB),不支持在线预览</p>
      <p class="cv-large-hint">Commit 5 将支持"在文件夹打开"</p>
    </div>

    <!-- 纯文本 / 代码(Monaco 在 Commit 3 接) -->
    <pre v-else class="cv-text"><code>{{ content }}</code></pre>
  </div>
</template>

<style scoped>
.code-viewer {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.cv-md {
  flex: 1;
  overflow: auto;
  padding: 20px 28px;
  font-size: 14px;
  line-height: 1.7;
  color: var(--text);
}
.cv-text {
  flex: 1;
  margin: 0;
  padding: 16px 20px;
  overflow: auto;
  font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
  font-size: 13px;
  line-height: 1.6;
  color: var(--text);
  white-space: pre-wrap;
  word-break: break-all;
  background: var(--bg-card);
}
.cv-text code {
  font-family: inherit;
  background: transparent;
}
.cv-binary,
.cv-large {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--text-dim);
  padding: 40px 20px;
}
.cv-binary-title,
.cv-large-title {
  font-size: 15px;
  font-weight: 500;
  color: var(--text);
  margin: 4px 0 0 0;
}
.cv-binary-hint,
.cv-large-hint {
  font-size: 13px;
  color: var(--text-faint);
  margin: 0;
}
</style>