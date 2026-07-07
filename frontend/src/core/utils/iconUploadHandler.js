// 工具编辑弹窗 — 自定义图标上传兜底监听
//
// 2026-07-03 v4 修复:把 document-level 'change' 监听从 ToolsView 的 onMounted
// 挪到 main.js bootstrap,确保应用启动就注册,不依赖任何组件 lifecycle / patch。
//
// 根因(再次回顾):
// - v1 (label for + 屏幕外 input) → WKWebView+Modal Teleport 下 click 被静默吞
// - v2 (button + JS ref.click()) → Vue patch 阶段丢失 button onClick + input ref,
//                                   整条 form 链 DOM 上 __vnode 全空,Vue 内部状态机
//                                   异常导致 onMounted handler 也未实际注册
// - v3 (label 包裹 input + onMounted 装 document change listener) → 同样依赖 onMounted
//                                                                  在异常 patch 树上未触发
// - v4 (独立模块,main.js bootstrap 立即注册) → 完全不依赖 ToolsView 的 lifecycle / patch
//
// 工作机制:document 装 capture-phase 'change' 监听,匹配 .icon-upload-input
// 元素(class 标记来自 ToolsView 模板里的 input)。文件选择器触发后,change 事件冒泡
// 到 document,这里统一处理。
//
// 注意:
// - listener 一次注册,应用生命周期内有效
// - 用 class 匹配而不是 id,避免与模板 ref 冲突
// - 处理函数需能访问 tools store + i18n,所以这里只发 CustomEvent,处理函数放 ToolsView 里
//   (ToolsView 通过 window 监听 CustomEvent,见 ToolsView 的 onIconUploadChangeFromGlobal)
import { uploadToolIcon } from '@/api/skillbox/tools'

const ICON_UPLOAD_INPUT_CLASS = 'icon-upload-input'
let bound = false

/**
 * 实际处理文件选择结果。
 * 工具 store 必须由调用方在初始化时注入(因为 store 依赖 pinia 实例,模块顶层拿不到)。
 *
 * @param {object} ctx { tools, t, toast }  store + i18n + toast
 */
export function bindIconUploadHandler(ctx) {
  if (bound) return
  bound = true
  document.addEventListener(
    'change',
    async (e) => {
      const el = e.target
      if (!el || !(el instanceof HTMLInputElement)) return
      if (el.type !== 'file') return
      if (!el.classList.contains(ICON_UPLOAD_INPUT_CLASS)) return
      await handleIconUpload(el, ctx)
    },
    true,
  )
}

async function handleIconUpload(input, ctx) {
  const file = input.files && input.files[0]
  if (!file) {
    input.value = ''
    return
  }
  const { tools, t, toast, uploadingToolFlag } = ctx
  uploadingToolFlag.value = true
  try {
    const res = await uploadToolIcon(file)
    if (res && res.name) {
      tools.form.icon_file = res.name
      toast.success(t('tools.uploadIconOk'))
    }
  } catch (err) {
    toast.error(t('tools.uploadIconFailed', { msg: err?.message || err }))
  } finally {
    uploadingToolFlag.value = false
    // 重置 input,允许选同一文件(浏览器同一 file 不会触发 change 事件)
    input.value = ''
  }
}
