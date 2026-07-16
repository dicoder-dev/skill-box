# 自绘 1px 极细滚动条 v-thin-scrollbar

2026-07-16 首页滚动条「再细一点」诉求时落地。

## 背景:macOS Chrome 轨道硬下限
macOS Chrome / Safari / WKWebView 的 `::-webkit-scrollbar { width: 1px }` 只能让 **thumb 缩到 1px**,但 **轨道(layout)仍占 ~15px**(macOS 系统级 overlay 滚动条)。
Chrome 121+ 标准 `scrollbar-width: thin` 能收到 **11px**(macOS 硬编码下限),CSS 无论如何压不下去。

## 解法
新建 `core/directives/thinScrollbar.js`,Vue 指令 `v-thin-scrollbar`:

1. mounted: 给元素隐藏原生滚动条(`scrollbar-width:none` + `::-webkit-scrollbar { display: none }`),挂全局 `.tsb-thumb` 样式
2. 注入两个绝对定位 thumb DOM(`.tsb-thumb-v` / `.tsb-thumb-h`)到容器右/下边缘
3. `scroll` + `ResizeObserver` 实时算 thumb 位置(top/left) + 高度/宽度比例
4. unmounted: 清理监听 + 移除 thumb DOM

## 视觉
```css
.tsb-thumb {
  position: absolute;
  background: color-mix(in srgb, var(--accent-blue) 40%, transparent);
  border-radius: 999px;
  pointer-events: none;
  z-index: 5;
  transition: background 150ms ease;
}
.tsb-thumb-v { top: 0; right: 0; width: 1px; }
.tsb-thumb-h { left: 0; bottom: 0; height: 1px; }
.tsb-host:hover .tsb-thumb { background: color-mix(in srgb, var(--accent-blue) 70%, transparent); }
```

## 用法
main.js 全局注册 `installThinScrollbar(app)`,容器加 `v-thin-scrollbar`:
```vue
<div v-thin-scrollbar class="tree-container">...</div>
```

## 踩过的坑
- `.tsb-host` 必须 `position: relative`(指令自动补),否则 thumb 绝对定位找不到锚点
- 容器需要 `overflow: auto` 才能滚动,指令不会自动加
- 初始 scroll/resize 时机:mounted 后立即算一次,后续由 observer 接管
- 不重复绑定:用 `el.__tsb_bound__` 防 unmount 前重 mounted 重复挂 DOM
- scrollbar-width: none + ::-webkit-scrollbar display:none 缺一不可(Chrome / Firefox 都要)

## 替代方案对比(避免再次调研)
- 第三方库(`simplebar` / `overlay-scrollbars`):功能全但体积大,本项目没必要
- `scrollbar-width: thin`:11px 下限,达不到 1px 要求
- 系统偏好(显示滚动条 → 基于鼠标):macOS 端能让 webkit 自定义生效,但依赖用户改设置,不可控