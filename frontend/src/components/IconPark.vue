<!--
  IconPark 通用图标组件

  替换原 @iconify/vue 的 <Icon> 用法。
  - icon: 接受 mdi 风格字符串 (mdi:xxx) 或 iconpark 组件名 (Close / Loading ...)
    内部自动通过 MDI_TO_ICONPARK 映射到 iconpark 组件,保证业务侧零侵入切换。
  - size / width / height: 透传给 svg size,默认 16。
  - theme: outline | filled | two-tone | multi-color,默认 outline。
  - fill: 颜色字符串,默认 currentColor 跟随父级文字色。
  - 其余 class 透传到根 <svg>,便于复用 .toast-icon / .ctx-item-icon 等已有类名。

  使用方式(三种都可):
    <IconPark icon="mdi:loading" />
    <IconPark icon="Close" />
    import { Icon } from '@/components/IconPark.vue'  // <-- 兼容旧 import 写法

  Why:
    项目统一图标库规范要求所有图标走 iconpark,见 docs/agent/memory/fe-icon-library.md。
    本组件不引入 emoji,不支持第三方 @iconify 远端 API,完全离线打包,适配 wails3 webview。
-->
<script>
import { computed as _computed } from 'vue'
import * as IconParkAll from '@icon-park/vue-next'
import { MDI_TO_ICONPARK, NOT_FOUND_ICON } from '@/core/icons/iconparkMap.js'

// default 导出:供 SFC 模板用 <IconPark ... /> (直接用 default)
const IconPark = {
  name: 'IconPark',
  props: {
    icon:        { type: String, required: true },
    size:        { type: [Number, String], default: 16 },
    width:       { type: [Number, String], default: null },
    height:      { type: [Number, String], default: null },
    theme:       { type: String, default: 'outline' },
    fill:        { type: [String, Array], default: () => ['currentColor'] },
    strokeWidth: { type: Number, default: 4 },
    spin:        { type: Boolean, default: false },
  },
  setup(props) {
    const componentName = _computed(() => {
      const raw = props.icon
      if (!raw) return NOT_FOUND_ICON
      if (raw.startsWith('mdi:')) {
        const key = raw.slice(4)
        return MDI_TO_ICONPARK[key] || NOT_FOUND_ICON
      }
      return raw
    })
    const resolvedSize = _computed(() => props.width ?? props.size)
    const resolvedHeight = _computed(() => props.height ?? props.width ?? props.size)
    const IconComponent = _computed(
      () => IconParkAll[componentName.value] || IconParkAll[NOT_FOUND_ICON]
    )
    return { componentName, resolvedSize, resolvedHeight, IconComponent }
  },
}
export default IconPark
// named export:兼容旧代码 `import { Icon } from '@/components/IconPark.vue'`
export const Icon = IconPark
</script>

<script setup>
// 让 SFC 也能用 <IconPark /> 直接渲染(因为 default 导出已存在,这里不再导出)
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
