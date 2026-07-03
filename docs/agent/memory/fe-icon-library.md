---
name: fe-icon-library
description: 项目统一图标库规范(iconpark),所有图标走 @icon-park/vue-next + 自封 IconPark 组件,mdi 字符串走 MDI_TO_ICONPARK 映射,禁用 emoji 和 @iconify 在线
metadata:
  type: project
---

# 前端图标库规范(iconpark)

项目所有图标统一走 **[IconPark](https://iconpark.oceanengine.com/official)**,由字节开源的 5300+ SVG 图标库,
Vue3 包名 `@icon-park/vue-next`。**禁止**使用 emoji、iconfont、@iconify 在线 API 拉取图标。

## 选型理由

1. **完全离线打包**:`@icon-park/vue-next` 是本地 Vue SFC 组件,不依赖任何 CDN,与 [[wails3-webview-iconify-network]] 一致。
2. **图标量大**:5300+ 图标覆盖 UI 各类场景(工具 / 文件夹 / 状态 / 箭头 / 媒体 等),与本项目后台工具/技能管理场景高度契合。
3. **统一视觉风格**:线性描边一致,色彩主题可配 (outline / filled / two-tone / multi-color)。
4. **2026-07-03 决定**:替代原 mdi: 图标方案,落地封装在 `IconPark.vue` + `iconparkMap.js`。

## 接入位置

- **包**:`frontend/package.json` → `"@icon-park/vue-next": "^1.4.2"`
- **全局样式**:`frontend/src/main.js` 顶部 `import '@icon-park/vue-next/styles/index.css'`
- **打包**:`vite.config.js` 的 `build.rollupOptions.output.manualChunks["vendor-iconpark"]` 把 iconpark 拆到独立 chunk(2.6MB / gzip 397KB),业务主 chunk 不被撑大。

## 通用组件:`<IconPark />`

`frontend/src/components/IconPark.vue` 封装组件,**支持三种用法**:

```vue
<!-- 1. 业务侧用 mdi 字符串(老习惯) -->
<IconPark icon="mdi:loading" size="24" />

<!-- 2. 直接给 iconpark 组件名(PascalCase) -->
<IconPark icon="Close" size="18" />

<!-- 3. 兼容旧 import 写法 -->
import { Icon } from '@/components/IconPark.vue'
<Icon icon="mdi:check" />
```

组件 props:`icon` / `size` / `width` / `height` / `theme` / `fill` / `strokeWidth` / `spin`,
size 缺省 16,fill 缺省 `['currentColor']`(跟随父级文字色)。

## mdi → iconpark 映射表

`frontend/src/core/icons/iconparkMap.js` 维护 `MDI_TO_ICONPARK`,把 mdi 后缀字符串
(如 `loading`、`check-circle-outline`)映射到 iconpark 组件名(如 `Loading`、`CheckOne`)。

**维护规则**:

- 加新映射 → 直接在表里增一行,`key` 是 mdi 后半段,`value` 是 iconpark PascalCase 组件名。
- 选图原则:outline 主题优先(与原 mdi 一致);多个候选选语义最贴的;找不到完全对应的选最相近语义。
- 找不到对应时走兜底 `NOT_FOUND_ICON = 'Help'`(问号占位,比方块更可读)。
- 加完跑一遍 `grep -rEoh "mdi:[a-z0-9-]+" src` 看是否还有遗漏未映射的 mdi 名。

## 命名约定(iconpark 内部)

iconpark 的图标分 One/Two/Three/Four 等变体(尺寸 / 复杂度),选择顺序:

1. 优先 `X`(`Close`):最常用描边版
2. 兼容老 mdi 习惯时用 `XOne`(`CheckOne` 对应 mdi check-circle)
3. 实心 / 多色用 `filled` / `multi-color` theme,不要混名字

## 反模式(禁止)

- ❌ **不要**在业务代码直接 `import { Close } from '@icon-park/vue-next'` 写死具体组件,会破坏映射层一致性。统一走 `<IconPark icon="..." />`。
- ❌ **不要**用 emoji(项目记忆里 `avoid-violet-as-primary-color.md` 同源要求,AI 感强)。
- ❌ **不要**用 iconfont / Material Icons / @iconify 在线图标,违反 webview 离线原则。
- ❌ **不要**自己写 inline SVG 图标(尺寸 / 颜色与 iconpark 不一致),有需要就找 iconpark 里最贴近的。

## How to apply

- 新加视图 / 组件 → 全部图标走 `<IconPark icon="mdi:xxx" />`,mdi 名查映射表。
- 找图标 → 先去 [iconpark 官网](https://iconpark.oceanengine.com/official) 搜,选中的图标右侧复制 PascalCase 组件名。
- 映射表没收录的 mdi 名 → 在 `iconparkMap.js` 加映射,跑一遍 `npm run build` 验证。
- 后续若改图标库 → 改 `IconPark.vue` + `iconparkMap.js` 两处即可,业务侧零改动。
