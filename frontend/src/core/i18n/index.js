// core/i18n — vue-i18n 9 composition API 接入。
//
// 命名空间:app / nav / common / skills / projects / market / onboarding / audit / ai
// 设计:
//   - 语言包按需引入:zh-CN / en-US 同步加载(总大小 < 30 KB,不分块)
//   - 默认 zh-CN;通过 localStorage('skillbox.lang') 记忆;浏览器 navigator.language 兜底
//   - 全局 $t 由 vue-i18n 注入;SFC 用 <i18n-t> 包裹插值文本
//   - 后端可注入默认语言(由 App.vue 接收 __APP_RUNTIME__?.lang),这里 main.js 之前覆盖
//   - 不暴露业务侧切换 UI(P1);当前默认写死 zh-CN,等 Settings 里加切换器再扩
import { createI18n } from 'vue-i18n'
import zhCN from './zh-CN.js'
import enUS from './en-US.js'

const STORAGE_KEY = 'skillbox.lang'

// 解析默认语言:localStorage > 浏览器 > zh-CN
function detectLocale() {
  try {
    const saved = localStorage.getItem(STORAGE_KEY)
    if (saved === 'zh-CN' || saved === 'en-US') return saved
  } catch (_) { /* 隐私模式可能抛 */ }
  const nav = (typeof navigator !== 'undefined' && navigator.language) || ''
  if (nav.toLowerCase().startsWith('zh')) return 'zh-CN'
  return 'zh-CN' // 默认中文,符合国内用户基线
}

const i18n = createI18n({
  legacy: false, // composition API 模式,必须显式 false
  globalInjection: true,
  locale: detectLocale(),
  fallbackLocale: 'zh-CN',
  messages: {
    'zh-CN': zhCN,
    'en-US': enUS,
  },
})

// 切换语言并持久化。组件里 import { setLocale } 调即可。
export function setLocale(loc) {
  if (loc !== 'zh-CN' && loc !== 'en-US') return
  i18n.global.locale.value = loc
  try { localStorage.setItem(STORAGE_KEY, loc) } catch (_) { /* 隐私模式 */ }
}

// 当前语言(响应式:组件里 const { locale } = useI18n() 也能拿到)
export function getLocale() {
  return i18n.global.locale.value
}

export default i18n