<!--
  IconPark 通用图标组件

  替换原 @iconify/vue 的 <Icon> 用法。
  - icon: 接受 mdi 风格字符串 (mdi:xxx) 或 iconpark 组件名 (Close / Loading ...)
    内部自动通过 MDI_TO_ICONPARK 映射到 iconpark 组件,保证业务侧零侵入切换。
  - size / width / height: 透传给 svg size,默认 16。
  - theme: outline | filled | two-tone | multi-color,默认 outline。
  - fill: 颜色字符串,默认 currentColor 跟随父级文字色。
  - 其余 class 透传到根 <svg>,便于复用 .toast-icon / .ctx-item-icon 等已有类名。

  使用方式:
    <IconPark icon="mdi:loading" />                              // SFC 模板(首选)
    import IconPark from '@/components/IconPark.vue'             // 业务 import(默认)
    // 注:不再兼容 `import { Icon }` 写法,业务侧改用 default import。模板里标签名仍是 <IconPark />。

  Why:
    项目统一图标库规范要求所有图标走 iconpark,见 docs/agent/memory/fe-icon-library.md。
    本组件不引入 emoji,不支持第三方 @iconify 远端 API,完全离线打包,适配 wails3 webview。

  注意:Vue 3 SFC 不支持同时 `<script>` 块 + `<script setup>` 块共存时让 named export
  和 template 都正确解析(模板解析依赖单一 default export)。要兼容 named import,
  必须放在单独的 .js shim 文件 — 业务侧直接用 default import 即可,见 IconPark.vue 注释。
-->
<script setup>
import { computed } from 'vue'
import * as IconParkAll from '@icon-park/vue-next'
import { MDI_TO_ICONPARK, NOT_FOUND_ICON } from '@/core/icons/iconparkMap.js'

const props = defineProps({
  icon:        { type: String, required: true },
  size:        { type: [Number, String], default: 16 },
  width:       { type: [Number, String], default: null },
  height:      { type: [Number, String], default: null },
  theme:       { type: String, default: 'outline' },
  fill:        { type: [String, Array], default: () => ['currentColor'] },
  strokeWidth: { type: Number, default: 4 },
  spin:        { type: Boolean, default: false },
})

// mdi:loading → Loading;其余 mdi:xxx → 映射表;非 mdi → 原名作 iconpark 组件名
const componentName = computed(() => {
  const raw = props.icon
  if (!raw) return NOT_FOUND_ICON
  if (raw.startsWith('mdi:')) {
    const key = raw.slice(4)
    return MDI_TO_ICONPARK[key] || NOT_FOUND_ICON
  }
  return raw
})

const resolvedSize = computed(() => props.width ?? props.size)
const IconComponent = computed(
  () => IconParkAll[componentName.value] || IconParkAll[NOT_FOUND_ICON]
)
</script>

<template>
  <component
    :is="IconComponent"
    :size="resolvedSize"
    :theme="theme"
    :fill="fill"
    :stroke-width="strokeWidth"
    :spin="spin || undefined"
  />
</template>
