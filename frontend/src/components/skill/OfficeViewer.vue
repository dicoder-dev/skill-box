<script setup>
// OfficeViewer - 技能包内 office 文档在线预览
//
// 2026-07-08 增:用 @vue-office 套件支持 docx / pdf / xlsx / xls / pptx 在线预览。
// 替代 CodeViewer 之前把 pdf 归到二进制兜底的体验。
//
// 数据流:后端 getSkill full=true → canonical.files[i].content 是 string(Go 默认 utf-8
// 把所有字节读成 string)。前端拿到 string 需要转 ArrayBuffer 给 vue-office(它内部
// 解析 zip / 二进制格式)。
//
// 关键点:Vue props 传 ArrayBuffer 时,Vue 会自动包一层 reactive proxy,而 vue-office
// 内部用 instanceof ArrayBuffer 判断。所以这里要么直接传 ArrayBuffer 不包 ref,
// 要么用 markRaw/shallowRef。我们用 shallowRef + 直接 .value 写,绕过 proxy。

import { computed, ref, shallowRef, watch, defineAsyncComponent } from 'vue'
// 2026-07-08 改:vue-office 4 个组件用 defineAsyncComponent 动态 import,
// 不进 CodeViewer 主 chunk。pdfjs-dist 单包 ~ 1MB,docx 的 mammoth 类解析器
// 也 ~ 800KB,excel 用 xlsx 解析 ~ 500KB,4 个全静态 import 直接撑爆主 bundle。
// 改成按 kind 动态加载,首次打开对应 office 文档时才下载对应 chunk。
const VueOfficeDocx = defineAsyncComponent(() => import('@vue-office/docx'))
const VueOfficeExcel = defineAsyncComponent(() => import('@vue-office/excel'))
const VueOfficePdf = defineAsyncComponent(() => import('@vue-office/pdf'))
const VueOfficePptx = defineAsyncComponent(() => import('@vue-office/pptx'))
// css 静态 import 跟组件进同一 chunk;动态 css 在 vue3 里支持不稳,这里接受样式
// 跟第一个动态 import 的组件同批次加载。
import '@vue-office/docx/lib/index.css'
import '@vue-office/excel/lib/index.css'

const props = defineProps({
  kind: { type: String, default: '' },        // 'docx' | 'excel' | 'pdf' | 'pptx'
  content: { type: String, default: '' },      // 文件内容 string(后端返回)
})

// shallowRef:ArrayBuffer 不能被 deep reactive 包,会破坏 vue-office 内部 instanceof 判断
const arrayBuffer = shallowRef(null)
const parseError = ref('')

// string → ArrayBuffer:每个 charCode 占 2 字节(JS UTF-16) → Uint8Array → .buffer
// 2026-07-08 改:用 TextEncoder 走 UTF-8 编码更准(中文/二进制字节不会被拆)。
// 但 office 文件本身就是二进制,后端按字节读 string 再 UTF-8 encode 等价于还原原始字节,
// 所以这个路径稳定。
const encodeError = ref('')
function rebuildBuffer() {
  parseError.value = ''
  encodeError.value = ''
  if (!props.content) {
    arrayBuffer.value = null
    return
  }
  try {
    const encoder = new TextEncoder()
    const u8 = encoder.encode(props.content)
    // 复制到一个独立的 ArrayBuffer(避免 u8.buffer 的整段共享)
    const ab = new ArrayBuffer(u8.byteLength)
    new Uint8Array(ab).set(u8)
    arrayBuffer.value = ab
  } catch (e) {
    encodeError.value = e?.message || String(e)
    arrayBuffer.value = null
  }
}
watch(() => props.content, rebuildBuffer, { immediate: true })

const Component = computed(() => {
  switch (props.kind) {
    case 'docx': return VueOfficeDocx
    case 'excel': return VueOfficeExcel
    case 'pdf': return VueOfficePdf
    case 'pptx': return VueOfficePptx
    default: return null
  }
})

function onError(e) {
  parseError.value = e?.message || String(e)
}
</script>

<template>
  <div class="ov-host">
    <component
      v-if="Component && arrayBuffer && !encodeError"
      :is="Component"
      :src="arrayBuffer"
      class="ov-component"
      @error="onError"
    />
    <div v-else-if="encodeError || parseError" class="ov-error">
      <p>文档解析失败: {{ encodeError || parseError }}</p>
      <p class="ov-error-hint">文件可能已损坏,或在 wails3 webview 下二进制 bytes 还原不完整。请用"在文件夹中打开"在原生应用查看。</p>
    </div>
    <div v-else class="ov-loading">
      <span class="ov-spinner" />
    </div>
  </div>
</template>

<style scoped>
.ov-host {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: auto;
  background: #f8fafc;
}
.ov-component {
  flex: 1;
  min-height: 0;
}
.ov-error {
  padding: 32px 20px;
  text-align: center;
  color: var(--danger, #dc2626);
  font-size: 13px;
}
.ov-error-hint {
  margin-top: 8px;
  font-size: 12px;
  color: var(--text-faint, #94a3b8);
}
.ov-loading {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 60px;
}
.ov-spinner {
  width: 18px;
  height: 18px;
  border: 2px solid var(--border, #e2e8f0);
  border-top-color: var(--accent-blue, #2563eb);
  border-radius: 50%;
  animation: ov-spin 0.8s linear infinite;
}
@keyframes ov-spin { to { transform: rotate(360deg); } }
</style>