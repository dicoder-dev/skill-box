# 拖拽把手用绝对定位,不要当 grid/flex 子项

2026-07-16 首页三面板(技能列表/目录树/大纲)加拖拽改宽时踩的坑。

## 现象
把手作为 grid/flex 子项插入,会**参与布局宽度分配**,引发两类问题:
1. **grid**: 把手写 `grid-column:1/2` 会显式占掉第一列,导致 `.skills-pane` 被自动放置算法挤到第二列,`.detail-pane` 反落第一列 —— 整个左右颠倒。
2. **flex**: 把手写 `width:6px; margin:0 -3px` 当 flex 子项,负 margin 只抵消位置偏移,自身 6px 仍占布局宽度,子项总宽 = 面板+把手+viewer 超出容器,`scrollWidth > clientWidth` 撑出横向溢出。

## 正解
把手一律 `position:absolute` 脱离布局流,锚定到边界:
- 父容器加 `position:relative`
- 左侧面板(向右拖): `left: var(--面板宽变量); transform: translateX(-4px)`
- 右侧面板(向左拖): `right: var(--面板宽变量); transform: translateX(4px)`
- 命中区 `width:8px` 透明,视觉细线用 `::after` 2px + hover/dragging 显 `--accent-blue`
- 面板宽度统一用 CSS 变量(如 `--sfip-left-w`),把手 left/right 直接引用同一变量,拖动时 composable 只改变量,把手位置自动跟随

## 配套
- composable: `core/composables/useResizablePanel.js`,target 用 `'css-var'`(写变量) 统一三处;`'grid-col'` 也走 `setProperty(cssVar)`;持久化键 `skillbox:panel:<key>`
- 宽度写 CSS 变量而非 el.style.width,这样绝对定位把手能靠 `left/right: var()` 联动
- setup 时 DOM 未挂载,onMounted 调一次 `sync()` 落地初始宽度

## 附带发现:详情底部横向滚动条真凶
不是拖拽引入的,是 `.sfip-viewer-header`(工具栏)按钮多时 flex 撑破,冒泡到 `.sfip-body`。
兜底: header 加 `min-width:0; overflow:hidden`;并让 `.detail-pane` `overflow-y:auto` → `overflow:hidden`,滚动下沉到子容器。
(与旧记忆 fe-sfip-header-hscroll 同源,那次改按钮 flex:0,这次加 header overflow 兜底。)
