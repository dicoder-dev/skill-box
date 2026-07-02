<script setup>
// ToolIcon - 工具图标统一渲染组件。
//
// 优先级:
//   1) icon_file 非空 → 用 <img src="/api/files/tool-icons/<name>" alt />
//      (前端 vite dev 时直接给绝对 URL,后端 /api/files/tool-icons/ 提供二进制)
//   2) icon_file 为空 → 用 <Icon icon="mdi:..." /> (Iconify 离线优先)
//
// 设计原因:
//   - 用户上传的图标存盘于 ~/.skill-box/tool-icons/<name>,通过后端静态路由拉;
//     命中"自定义图标"概念,长得很品牌化(品牌 logo / 用户自己的 PNG)。
//   - 兜底用 Iconify 是 mdi: 字符串。所有内置 / 迁移期老数据都还有 mdi_icon,
//     前端 mdi 解析即可。
//
// Props:
//   - tool: ToolView(view 字段集)
//     必须字段:{mdi_icon, icon_file}
//   - size:  渲染尺寸(像素),默认 22 与原 <Icon :icon=:width="22"/> 一致
//
// 注意:本组件只渲染图标本身,不负责"主图标右侧 + 文字"那种卡片头部布局 —
// 那些留在 call site 用 flex / grid 自己拼。

import { computed } from 'vue'
import { Icon } from '@iconify/vue'

const props = defineProps({
  tool: { type: Object, required: true },
  size: { type: Number, default: 22 },
})

// imageUrl:当 icon_file 存在时拼成后端静态服务的绝对 URL
// baseURL 由 main.js 写入 window.__APP_CONFIG__,Web 模式 = '',Desktop 模式 = 127.0.0.1:port
function resolveBaseURL() {
  if (typeof window === 'undefined') return ''
  const cfg = window.__APP_CONFIG__
  if (cfg && typeof cfg.baseURL === 'string') return cfg.baseURL.replace(/\/$/, '')
  // 兜底:用当前 host(裸 web 模式 web 跑在同源,这里不需要前缀)
  if (window.location) {
    return `${window.location.protocol}//${window.location.host}`
  }
  return ''
}

const iconSrc = computed(() => {
  if (!props.tool || !props.tool.icon_file) return ''
  const base = resolveBaseURL()
  return `${base}/api/files/tool-icons/${props.tool.icon_file}`
})

const mdiIcon = computed(() => {
  if (!props.tool) return 'mdi:cog-outline'
  return props.tool.mdi_icon || 'mdi:cog-outline'
})
</script>

<template>
  <img
    v-if="iconSrc"
    :src="iconSrc"
    :width="size"
    :height="size"
    :alt="tool?.display_name || tool?.tool_id || 'tool icon'"
    class="tool-icon-img"
  />
  <Icon
    v-else
    :icon="mdiIcon"
    :width="size"
    :height="size"
  />
</template>

<style scoped>
.tool-icon-img {
  display: inline-block;
  /* SVG / PNG 在小尺寸下默认插值模糊,object-fit 强制 fit */
  object-fit: contain;
  vertical-align: middle;
  /* 防止父级背景污染:浅色 SVG 边缘 */
  background: transparent;
}
</style>
