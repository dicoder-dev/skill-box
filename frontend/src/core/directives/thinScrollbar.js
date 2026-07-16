// core/directives/thinScrollbar.js
//
// v-thin-scrollbar:自绘 1px 极细滚动条指令
//
// 背景:
//   macOS Chrome / Safari 滚动条轨道最小 11px(scrollbar-width:thin 极限),
//   CSS 无法继续压缩。要做 1px 必须隐藏原生轨道 + 自绘 thumb。
//
// 行为:
//   - mounted: 给元素隐藏原生滚动条(scrollbar-width:none + ::-webkit-scrollbar display:none),
//     创建两个绝对定位的 .tsb-thumb 元素(垂直 + 水平),挂到元素右/下边缘
//   - 监听元素 scroll + ResizeObserver,实时更新 thumb 位置 + 高度/宽度比例
//   - hover 时 thumb 加深颜色(可选,目前保持始终同色 35% accent 蓝)
//   - 卸载时移除监听 + 移除 thumb DOM
//
// 用法:
//   <div class="tree-container" v-thin-scrollbar>...</div>
//
// 适用场景:垂直/水平均可;同一元素支持两者;嵌套容器各自独立。
//
// 不做:
//   - 不支持 RTL(从右往左)
//   - 不支持双向 thumb 拖动(目前只读 thumb;原生滚动条已隐藏,用户仍可鼠标滚轮/拖动内容)
//   - 不做平滑滚动(原生 behavior 即可)

let _directiveInstalled = false

// 注入全局 CSS:对任何 .tsb-host 元素隐藏原生滚动条
function _ensureStyle() {
  if (typeof document === 'undefined') return
  if (document.getElementById('__thin_scrollbar_style__')) return
  const style = document.createElement('style')
  style.id = '__thin_scrollbar_style__'
  style.textContent = `
    .tsb-host { scrollbar-width: none; -ms-overflow-style: none; }
    .tsb-host::-webkit-scrollbar { display: none; width: 0; height: 0; }
    /* 桌面软件惯例:任何地方都不显示滚动条,只有 hover 容器时才浮现 */
    .tsb-thumb {
      position: absolute;
      background: color-mix(in srgb, var(--accent-blue, #3b82f6) 40%, transparent);
      border-radius: 999px;
      pointer-events: none;
      transition: background 150ms ease, opacity 150ms ease;
      z-index: 5;
      opacity: 0;
    }
    .tsb-thumb-v { top: 0; right: 0; width: 1px; height: 0; }
    .tsb-thumb-h { left: 0; bottom: 0; height: 1px; width: 0; }
    .tsb-host:hover .tsb-thumb {
      opacity: 1;
      background: color-mix(in srgb, var(--accent-blue, #3b82f6) 70%, transparent);
    }
  `
  document.head.appendChild(style)
}

function _createThumb(axis) {
  const el = document.createElement('div')
  el.className = axis === 'v' ? 'tsb-thumb tsb-thumb-v' : 'tsb-thumb tsb-thumb-h'
  return el
}

function _updateThumb(host, vThumb, hThumb) {
  // 垂直 thumb
  const sh = host.scrollHeight, ch = host.clientHeight, st = host.scrollTop
  const hasV = sh > ch + 1
  if (hasV) {
    const ratio = ch / sh
    const h = Math.max(20, ch * ratio)
    vThumb.style.height = h + 'px'
    vThumb.style.top = (st / (sh - ch)) * (ch - h) + 'px'
    vThumb.style.display = 'block'
  } else {
    vThumb.style.display = 'none'
  }
  // 水平 thumb
  const sw = host.scrollWidth, cw = host.clientWidth, sl = host.scrollLeft
  const hasH = sw > cw + 1
  if (hasH) {
    const ratio = cw / sw
    const w = Math.max(20, cw * ratio)
    hThumb.style.width = w + 'px'
    hThumb.style.left = (sl / (sw - cw)) * (cw - w) + 'px'
    hThumb.style.display = 'block'
  } else {
    hThumb.style.display = 'none'
  }
}

function _bind(el, binding) {
  // 不重复绑定
  if (el.__tsb_bound__) return
  el.__tsb_bound__ = true
  _ensureStyle()
  // 确保容器可定位(thumb 绝对定位需要非 static)
  const cs = getComputedStyle(el)
  if (cs.position === 'static') el.style.position = 'relative'
  el.classList.add('tsb-host')

  const vThumb = _createThumb('v')
  const hThumb = _createThumb('h')
  el.appendChild(vThumb)
  el.appendChild(hThumb)

  const update = () => _updateThumb(el, vThumb, hThumb)
  el.__tsb_update__ = update
  el.addEventListener('scroll', update, { passive: true })
  // 内容尺寸变化(子节点增删/图片加载/窗口尺寸)需重算
  let ro = null
  if (typeof ResizeObserver !== 'undefined') {
    ro = new ResizeObserver(update)
    ro.observe(el)
    // 监听内容容器(el 的第一个子元素作为 scroll content)
    if (el.firstElementChild) ro.observe(el.firstElementChild)
  }
  el.__tsb_ro__ = ro
  // 初次计算
  update()
}

function _unbind(el) {
  if (!el.__tsb_bound__) return
  el.__tsb_bound__ = false
  el.removeEventListener('scroll', el.__tsb_update__)
  if (el.__tsb_ro__) {
    el.__tsb_ro__.disconnect()
    el.__tsb_ro__ = null
  }
  // 移除 thumb
  el.querySelectorAll('.tsb-thumb').forEach(t => t.remove())
  el.classList.remove('tsb-host')
  el.__tsb_update__ = null
}

export const thinScrollbarDirective = {
  mounted(el, binding) { _bind(el, binding) },
  updated(el, binding) {
    // 视图更新后(子元素增删)重新计算
    if (el.__tsb_update__) el.__tsb_update__()
  },
  unmounted(el) { _unbind(el) },
}

// 自动注册到 Vue app(v-main.js 可选调用)
export function installThinScrollbar(app) {
  if (_directiveInstalled) return
  _directiveInstalled = true
  app.directive('thin-scrollbar', thinScrollbarDirective)
}

// 全局暴露,方便在 main.js 注册
export default thinScrollbarDirective