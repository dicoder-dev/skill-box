<script setup>
// SkillFileInlinePanel - 首页右侧主区域:目录树 + 预览/编辑
//
// 2026-07-07 v4 重写:完全脱离 vue-i18n,template 内不调 t(),所有展示文本
// 直接用常量字符串写死。原因:vue-i18n 9 + legacy:false + useI18n 在 v-if 懒挂载
// 子组件里,t 会被 Proxy 包装,render function 抛 "t is not a function" 把整段
// component update 弄崩。其他故障(scope 区一直转圈、[i] 信息按钮首次点击
// 无效)均是同一根因的副作用。重写后组件内 0 个 t() 调用。
//
// 二次保险:
//   - onErrorCaptured 兜底 render 错,降级到安全 UI
//   - safeReload 允许用户点"重试"清错误状态
//
// 拆分:
//   - scope 区迁移到 <SkillScopePanel> 子组件,本组件只传 props
//   - Frontmatter / CodeViewer / FileTreeView 已是独立子组件,不重写

import { computed, nextTick, onMounted, onUnmounted, onUpdated, reactive, ref, onErrorCaptured } from 'vue'
import { plainT } from '@/core/i18n/index.js'
import IconPark from '@/components/IconPark.vue'
import Modal from '@/components/Modal.vue'
import ContextMenu from '@/components/ContextMenu.vue'
import FileTreeView from './FileTreeView.vue'
import CodeViewer from './CodeViewer.vue'
import SkillScopePanel from './SkillScopePanel.vue'
import { updateSkill, createSkill, getStoreInfo } from '@/api/skillbox/skills'
// 2026-07-13 改:右侧面板状态改用 useRightPanelMode(原 useMdOutlineVisible 不再使用,
// import 也去掉)。CodeViewer 内也不再 import 这个 composable。
import { useRightPanelMode } from '@/core/composables/useRightPanelMode'
import { useResizablePanel } from '@/core/composables/useResizablePanel'
import { useToastStore } from '@/core/store/toast'

// 2026-07-07 临时调试:桌面端 webview 缓存导致浏览器拉到旧 chunk,
// 用 console 时间戳确认这次是否拿到新版本。
// 用户在桌面端启用 devtools (wails3 dev 默认开 Cmd+Opt+I) 看 console 输出。
console.log('[SkillFileInlinePanel v6] loaded at', new Date().toISOString(), 'no-watch import, no-console')

// 2026-07-08 增:edit mode 切换诊断日志开关 —— 用户反馈"切到其他 skill 默认是
// 编辑模式"那个 bug,临时打开才能看清 setMode / clearEditingState /
// _syncSelectedFile 的实际触发时序。确认 bug 修好后改成 false 即可停。
const DEBUG_EDIT_MODE = true
const dlog = (...args) => { if (DEBUG_EDIT_MODE) console.log(...args) }

// 2026-07-13 改:为 SkillFileInlinePanel 提供一个**绕开 vue-i18n** 的 t()
// 函数,直接复用 core/i18n/plainT()(messages 对象兜底读,带当前 locale 解析,
// 失败回退 zh-CN,再失败返 key 字符串;支持 {n} / {name} 等简单占位符)。
//
// 为什么不用 useI18n():SkillFileInlinePanel 在 v-if 懒挂载 / 切 skill 重建
// 场景下,useI18n 暴露的 t 经常被 Proxy 包装成 ProxyObject,render function
// 拿它当函数调用 → "t is not a function" 把整段 component update 弄崩。
// plainT 走 messages[key] 直接取字符串,绝不会抛 — 这是同 v4 重写时一样的
// 兜底策略,只是从"完全硬编码"升级到"走 i18n key"。
function t(key, values) {
  return plainT(key, values)
}

// ===== 文案常量(i18n key 形式)=====
const LABEL_NO_FILE = 'skills.fileBrowser.noFile'
const LABEL_EDIT = 'common.edit'
const LABEL_PICK = 'skills.fileBrowser.pickOneToBrowse'
const LABEL_DIRTY = 'skills.fileBrowser.modifiedShort'
const LABEL_DISCARD = 'skills.fileBrowser.discardChanges'
const LABEL_SAVE = 'common.save'
const LABEL_SAVING = 'skills.fileBrowser.saving'
const LABEL_FILES = 'skills.fileBrowser.files'
const LABEL_FRONTMATTER_TITLE = 'skills.fileBrowser.viewFrontmatter'
const LABEL_RENDER_ERROR_TITLE = 'skills.fileBrowser.renderError'
const LABEL_RETRY = 'common.retry'

// 2026-07-10 增:大纲面板显隐(全局状态,localStorage 持久化,跨文件保留)。
// CodeViewer 内部大纲渲染也读同一个 composable 状态,这里顶栏按钮和大纲
// header 内的 toggle 是同一份状态,两边都能控制。
const LABEL_OUTLINE_SHOW = 'skills.fileBrowser.showOutline'
const LABEL_OUTLINE_HIDE = 'skills.fileBrowser.hideOutline'
// 2026-07-13 改 v3:右侧面板由两个独立按钮控制,不再走单一 mode 状态:
//   - 大纲按钮:点击切换 outline 面板(仅对 md 文件显示,且 view 模式 + 有标题)
//   - AI 按钮:点击切换 AI 面板(所有文件 + view 模式)
// 两个面板互斥(点开一个会自动隐藏另一个),而不是 3 态 toggle,
// 这样两个按钮始终可见且语义独立,符合用户「大纲 / AI 想看哪个点哪个」的直觉。
const {
  mode: rightMode, aiActive, outlineActive,
  showAI, showOutline, hidePanel,
} = useRightPanelMode()

// 2026-07-13 改 v3:点击大纲按钮 → 显示大纲(若已在显示则隐藏);
// 点击 AI 按钮 → 显示 AI(若已在显示则隐藏)。
// 同一时刻只能有一个面板打开,互斥逻辑封装在这里。
function onClickOutline() {
  if (outlineActive.value) hidePanel()
  else showOutline()
}
function onClickAi() {
  if (aiActive.value) hidePanel()
  else showAI()
}

// 2026-07-13 增:AI 按钮文案 + 大纲按钮文案常量(template 里直接读,避开 i18n Proxy 坑)
const LABEL_AI_OPEN = '打开 AI 助手'
const LABEL_AI_CLOSE = '关闭 AI 助手'

// 2026-07-13 增:CodeViewer 内 AI aside emit 转发。
//   - onAiApplySkill(text):SKILL.md 路径 → 透传到父级 SkillsView.onAIApply(updateSkill)
//   - onAiApplyLocal(text):非 SKILL.md 路径 → 内部直接写 localFiles + 标 dirty,
//     由用户手动点保存(Ctrl+S / Cmd+S 或保存按钮)落盘,与手编编辑体验一致。
//   - onSetRightPanelMode(m):统一处理 CodeViewer emit 的 update:right-panel-mode
//     (outline aside 收起按钮 / AI aside 关闭按钮 / 切换大纲按钮)。
function onAiApplySkill(text) {
  emit('ai-apply-skill', text)
  // 2026-07-13 增 v3:应用后 CodeViewer 闪烁,让用户感知文件已变
  flashApplied()
}

function onAiApplyLocal(text) {
  const path = selectedFile.value?.path
  if (!path || path === 'SKILL.md') return
  localFiles.set(path, text || '')
  // 触发 dirty(走与手编编辑一致的路径)
  const s = new Set(dirtyPaths.value)
  s.add(path)
  dirtyPaths.value = s
  toast.success('AI 输出已写入,请点保存(Ctrl+S / Cmd+S)落盘')
  // 2026-07-13 增 v3:应用后 CodeViewer 闪烁,即使文件还没落盘也让用户看到内容变了
  flashApplied()
}

function onSetRightPanelMode(m) {
  if (m === 'outline') showOutline()
  else if (m === 'ai') showAI()
  else hidePanel()
}

// 2026-07-11 增:文件树右键菜单 + 文件/目录 CRUD 弹窗文案(组件内 0 t(),统一常量)
const LABEL_CTX_NEW_FILE = 'skills.fileBrowser.ctxNewFile'
const LABEL_CTX_NEW_DIR = 'skills.fileBrowser.ctxNewFolder'
const LABEL_CTX_RENAME_FILE = 'skills.fileBrowser.ctxRenameFile'
const LABEL_CTX_RENAME_FOLDER = 'skills.fileBrowser.ctxRenameFolder'
const LABEL_CTX_DELETE_FILE = 'skills.fileBrowser.ctxDeleteFile'
const LABEL_CTX_DELETE_FOLDER = 'skills.fileBrowser.ctxDeleteFolder'
const LABEL_CTX_OPEN_FOLDER = 'skills.fileBrowser.openInExplorer'
const LABEL_NEW_FILE_PROMPT = 'skills.fileBrowser.newFileTitle'
const LABEL_NEW_DIR_PROMPT = 'skills.fileBrowser.newFolderTitle'
const LABEL_RENAME_FILE_PROMPT = 'skills.fileBrowser.renameFileTitle'
const LABEL_RENAME_FOLDER_PROMPT = 'skills.fileBrowser.renameFolderTitle'
const LABEL_DELETE_FILE_PROMPT = 'skills.fileBrowser.deleteFileTitle'
const LABEL_DELETE_FILE_CONFIRM = 'skills.fileBrowser.deleteFileConfirm'

// 2026-07-12 增:触发词改为可选(对应"可选 + 空态"两个文案)
const LABEL_TRIGGERS_OPTIONAL = 'common.optional'
const LABEL_TRIGGERS_EMPTY_HINT = 'skills.editor.triggersEmptyHint'

const toast = useToastStore()

const props = defineProps({
  files: { type: Array, default: () => [] },
  // 2026-07-14 v2 增:磁盘绝对 source_path,由 SkillsView 传过来,沿 prop 链给到 AIRightPanel。
  // 让 AIRightPanel 能把 AI 历史写到正确的 .skill-box/history/ 目录,
  // 不再误用文件相对路径(filePath)。
  sourcePath: { type: String, default: '' },
  skill: { type: Object, default: () => ({}) },
})

const emit = defineEmits(['saved', 'ai-apply-skill'])

// 2026-07-13 增 v3:AI 应用成功后,CodeViewer 根容器黄色边框闪烁 1.5s 让用户看到文件被改了。
// 每次应用 +1,CodeViewer 通过动画响应变化;1.5s 后我们把 applyFlash 重置回 0,
// 这样下一次应用能再次触发 +1(从 0 变 1,1 变 2 这种数字变化才能让 Vue watch 检测到)。
const applyFlash = ref(0)
let _flashTimer = null
function flashApplied() {
  applyFlash.value += 1
  if (_flashTimer) clearTimeout(_flashTimer)
  _flashTimer = setTimeout(() => { applyFlash.value = 0 }, 1600)
}

// ====== File selection state ======
const selectedFile = ref(null)
const selectedKey = ref('')
const localFiles = reactive(new Map())
const dirtyPaths = ref(new Set())

// 2026-07-08 一刀切:不再用 reactive map 缓存每文件 mode,改用单个
// `currentEditingPath` ref 表达当前正在编辑哪个 path(skillName|path)。
// 任何"打开/切换"流程(进入组件 / 选文件 / 选 skill)都自动回到 view,
// 只有用户主动点"编辑"按钮才进入 edit;并且只能编辑一个文件,切走 /
// 放弃 / 保存 自动退出编辑态。彻底消除"残留 edit 模式"的所有可能路径。
//
// reset/save 的 emit 时间窗锁:resetLock 解决"reset/save 后 Tiptap 飞行中
// 异步 emit 把刚重置的 localFiles 又写回最新内容";enterEditGuard 解决
// "刚点编辑后 Tiptap/Monaco 初始化 emit 跟 orig 末尾空白不一致 → 误判 dirty"。
const currentEditingPath = ref('') // `${skillName}|${path}` 或 ''
let resetLockUntil = 0
let enterEditGuardUntil = 0
function editingKey(skillName, path) {
  if (!path) return ''
  return skillName ? `${skillName}|${path}` : path
}
function getCurrentMode(skillName, path) {
  const k = editingKey(skillName, path)
  if (!k) return 'view'
  // 一刀切:当前"正在编辑的 path"必须 === 当前选中 path,否则一律 view
  // —— 任何切文件 / 切 skill 会把 selectedFile 改掉,selectFile 里会先清
  // currentEditingPath,所以"当前选中"≠"正在编辑"基本只发生在 stale 帧。
  return currentEditingPath.value === k ? 'edit' : 'view'
}
function setMode(skillName, path, m) {
  const k = editingKey(skillName, path)
  if (!k) return
  const prev = currentEditingPath.value === k ? 'edit' : 'view'
  if (m === 'edit') {
    // 进编辑:写入唯一 currentEditingPath(覆盖式,单 editing 状态)
    currentEditingPath.value = k
    // Tiptap/Monaco 初始化 emit 的尾随空白差异(归一化后跟 orig 相等)
    // 不应判 dirty。同一 setMode 时打 enterEditGuard 时间窗,期间 emit 仅
    // 同步 localFiles 不算 dirty(避免"刚点编辑就显示保存按钮")。
    enterEditGuardUntil = Date.now() + 80
    if (path && dirtyPaths.value.has(path)) {
      const s = new Set(dirtyPaths.value)
      s.delete(path)
      dirtyPaths.value = s
    }
    dlog('[sfip setMode → edit]', { skillName, path, k, prev })
  } else {
    // 退出编辑:如果退的是当前正在编辑的那个 → 清掉。
    if (currentEditingPath.value === k) {
      currentEditingPath.value = ''
    }
    dlog('[sfip setMode → view]', { skillName, path, k, prev })
  }
}

// 一刀切版"清空"函数:不再有 module 级残留,只需要把当前编辑态清掉。
// 所有调用点(selectItem / onSelectFile / onDiscardDrop / onDiscardSave /
// saveCurrent 成功后 / resetCurrent / 子组件初始化)统一调这一个。
function clearEditingState() {
  dlog('[sfip clearEditingState] before=', currentEditingPath.value)
  currentEditingPath.value = ''
  dirtyPaths.value = new Set()
  resetLockUntil = 0
  enterEditGuardUntil = 0
}

function splitSkillMd(text) {
  if (!text) return { frontmatter: '', body: text }
  const m = text.match(/^---\s*\n([\s\S]*?)\n---\s*\n?/)
  if (!m) return { frontmatter: '', body: text }
  return { frontmatter: m[0], body: text.slice(m[0].length) }
}

// 选中文件 → localFiles 填充,响应 props.files 变化
// 2026-07-07 改 v6:不依赖 vue 的 watch(esm cache 缺 watch 函数,
// webview 拿到的 chunk 里 ReferenceError: Can't find variable: watch),
// 改用 onUpdated + 手动依赖追踪 — 每次父组件 patch 后重新检查 props。
let _lastFilesRef = null
let _lastSkillName = null
let _lastSkillVersion = null
function _syncSelectedFile() {
  const sk = props.skill
  const files = props.files
  const curFilesRef = files
  const curName = sk?.name
  const curVersion = sk?.version
  if (curFilesRef === _lastFilesRef && curName === _lastSkillName && curVersion === _lastSkillVersion) return
  // 2026-07-08 一刀切:切 skill(name/version 不同)时主动清掉编辑态 + dirty,
  // 防止任何 stale 帧在 onUpdated 跑完之前展示残留 edit。配合 currentEditingPath
  // 单点 ref,这里清完就一定回到 view,不会再有 module-level map 残留。
  const skillSwitched = curName !== _lastSkillName || curVersion !== _lastSkillVersion
  if (skillSwitched) {
    dlog('[sfip _syncSelectedFile] skillSwitched', { from: _lastSkillName, to: curName, wasEditing: currentEditingPath.value })
    currentEditingPath.value = ''
    dirtyPaths.value = new Set()
  }
  _lastFilesRef = curFilesRef
  _lastSkillName = curName
  _lastSkillVersion = curVersion
  if (!files || !files.length) {
    selectedFile.value = null
    selectedKey.value = ''
    localFiles.clear()
    dirtyPaths.value = new Set()
    return
  }
  const prev = selectedKey.value
  const target = (prev && files.find((f) => f.path === prev))
    || files.find((f) => f.path === 'SKILL.md')
    || files[0]
  // 2026-07-08 一刀切:_syncSelectedFile 重置 selectedFile 时,如果之前
  // currentEditingPath 还在编辑某个 path,跟新 selectedFile 不一致则清掉。
  // 但因为切 skill 已经清过,这里只需要"切文件"的兜底 —— 实际上组件是
  // :key="selectedFile.path" 重建的,currentEditingPath 由调用方(父级
  // selectItem 等)主动清掉更稳。这里**不再做**额外清理,保持单一来源。
  selectedFile.value = target
  selectedKey.value = target?.path || ''
}
function _syncLocalFiles() {
  const sk = props.skill
  const curFilesRef = props.files
  const curName = sk?.name
  if (curFilesRef === _lastFilesRef && curName === _lastSkillName) return
  // 跟 _syncSelectedFile 共享判断,省一次比较
  _lastFilesRef = curFilesRef
  _lastSkillName = curName
  localFiles.clear()
  for (const f of props.files || []) {
    const c = f.content || ''
    const stored = f.path === 'SKILL.md' ? splitSkillMd(c).body : c
    localFiles.set(f.path, stored)
  }
  dirtyPaths.value = new Set()
}
onUpdated(() => {
  _syncSelectedFile()
  _syncLocalFiles()
})
// 首次同步在 onMounted 里跑一次
// ===== 目录树面板宽度拖拽（写 CSS 变量 --sfip-left-w 到 .sfip-body）=====
const sfipBodyEl = ref(null)
const {
  width: sfipLeftWidth,
  dragging: sfipLeftDragging,
  startDrag: onSfipLeftStartDrag,
  reset: resetSfipLeftWidth,
  sync: syncSfipLeftWidth,
} = useResizablePanel({
  target: 'css-var',
  direction: 'right',
  storageKey: 'sfip-left-w',
  defaultWidth: 280,
  min: 180,
  max: 500,
  cssVar: '--sfip-left-w',
  scopeEl: sfipBodyEl,
})

onMounted(() => {
  _syncSelectedFile()
  _syncLocalFiles()
  fetchStoreRoot()
  // 2026-07-13 增:注册全局 Ctrl+S / Cmd+S 快捷键保存。
  // 挂 window 而非组件根 div,这样焦点在 Monaco/Tiptap 编辑器内部时也能捕获。
  // 仅在编辑模式或当前文件 dirty 时触发,避免空保存(同工具栏"保存"按钮的可见条件)。
  // macOS 上 metaKey(Cmd)等价 ctrlKey;其它平台 ctrlKey。输入法组合中不抢键。
  window.addEventListener('keydown', onKeyDown)
  // 目录树宽度拖拽初始化(setup 时 sfipLeftEl 未挂载,这里再写一次)
  syncSfipLeftWidth()
})

// 2026-07-07 改:切换文件前也走 dirty 检查。
// 用户改完 SKILL.md 后点其他文件(目录树里)→ 弹三选项。
// dirty 检查按"任一文件 dirty"算,不只是当前选中文件。
//
// 2026-07-07 修:file 参数从 FileTreeView emit 出来,FileTreeView.buildTree
// 输出的 file 对象只有 {name, path, size},**没有 content**。如果直接
// selectedFile.value = file → displayContent 算 selectedFile.value.content
// 是 undefined → 空白。修法:从 props.files 里 find 出含 content 的原始对象。
async function onSelectFile(file) {
  if (!file || !file.path) return
  if (file.path === selectedKey.value) return
  const verdict = await ensureCleanBeforeSwitch()
  if (verdict === 'cancel') return
  // 2026-07-08 改:proceed 后双保险,确保切到新文件前编辑态干净。
  if (verdict === 'proceed') clearEditingState()
  // 从 props.files 里拿完整的 {path, content} 对象(包含 content)
  const full = props.files.find((f) => f.path === file.path) || file
  selectedFile.value = full
  selectedKey.value = full.path
}

// 2026-07-07 增:切换前 dirty 检查 + 询问逻辑。
// 父组件在切换 skill / 切换文件前调 ensureCleanBeforeSwitch(),有 dirty 弹
// 三选项(保存 / 放弃 / 取消),等用户决策后再决定是否继续切换。
// 设计目标:无论保存/放弃,dirty 状态都要被清掉(用户的核心诉求),编辑态也
// 回到 view(避免切换后新文件继承 edit 模式)。
//
// 返回值约定(给父调用方):
//   'proceed'  → 允许切换(用户已保存或放弃)
//   'cancel'   → 不切换(用户取消)
//   'busy'     → 当前正在另一个保存流程中,父应该也阻断切换
const _discardResolve = ref(null)
const discardOpen = ref(false)
const discardFilePath = ref('')
const discardFileName = ref('')

async function ensureCleanBeforeSwitch() {
  // 2026-07-07 改:用最新的 dirtyPaths 计算 — computed isDirty 只反映当前选中文件,
  // 这里要"任一文件 dirty 都算脏"。
  if (!dirtyPaths.value || dirtyPaths.value.size === 0) {
    // 2026-07-08 改:即使没 dirty,如果当前文件处于 edit 模式,也要清掉(切走后不该继承 edit)。
    // 这是用户反馈"打开其他 skill 默认是处于编辑状态"那个 bug 的修法。
    // 之前只调 clearEditingState(单文件),现在改 clearEditingState(全部 edit 记录 + dirty),
    // 彻底确保切走之后新 skill 不会命中任何残留 edit 引用。
    clearEditingState()
    return 'proceed'
  }
  // 拿第一个 dirty 文件(多文件 dirty 时也只问一次,统一处理)
  const firstDirty = Array.from(dirtyPaths.value)[0]
  discardFilePath.value = firstDirty
  discardFileName.value = (firstDirty || '').split('/').pop() || firstDirty
  discardOpen.value = true
  return new Promise((resolve) => { _discardResolve.value = resolve })
}

async function onDiscardSave() {
  discardOpen.value = false
  // 保存当前 dirty 文件(只保存第一个 dirty;实际 InlinePanel 里只支持单文件保存,
  // 这里就按现有 saveCurrent 走)
  const r = _discardResolve.value
  _discardResolve.value = null
  if (!r) return
  // 把 selectedFile 切到 dirty 文件再保存
  const target = props.files.find((f) => f.path === discardFilePath.value)
  if (target) {
    selectedFile.value = target
    selectedKey.value = target.path
  }
  try {
    await saveCurrent()
  } catch (_) { /* saveCurrent 内部已 toast */ }
  // 2026-07-08 改:saveCurrent 成功后清所有 edit 态(不只当前一个)。
  // 旧版 clearEditingState 只清 selectedFile 的 mode,如果组件内有
  // 其他 path 也残留,仍会留存。一刀切版就是全清。
  clearEditingState()
  r('proceed')
}

function onDiscardDrop() {
  discardOpen.value = false
  const r = _discardResolve.value
  _discardResolve.value = null
  if (!r) return
  // 2026-07-08 改:放弃后清所有 edit 态(不只当前一个)—— 之前只 clearEditingState
  // 是漏的根因,残留会影响后续 onUpdated 看到旧 mode。
  clearEditingState()
  r('proceed')
}

function onDiscardCancel() {
  discardOpen.value = false
  const r = _discardResolve.value
  _discardResolve.value = null
  if (!r) return
  r('cancel')
}

function resetAllDirty() {
  // 把 dirty 文件的 localFiles 同步回原 content
  for (const dirtyPath of dirtyPaths.value) {
    const f = props.files.find((x) => x.path === dirtyPath)
    if (!f) continue
    const orig = dirtyPath === 'SKILL.md' ? splitSkillMd(f.content || '').body : (f.content || '')
    localFiles.set(dirtyPath, orig)
  }
  dirtyPaths.value = new Set()
}

const currentContent = computed(() => {
  const path = selectedFile.value?.path
  if (!path) return ''
  return localFiles.has(path) ? localFiles.get(path) : (selectedFile.value?.content || '')
})

const displayContent = computed(() => {
  if (!selectedFile.value) return ''
  if (selectedFile.value.path === 'SKILL.md') {
    return splitSkillMd(currentContent.value).body
  }
  return currentContent.value
})

// 2026-07-12 增:当前选中文件是否是 Markdown 文件(用于控制"大纲"按钮显隐)。
// 规则:路径以 .md 结尾即视为 Markdown(覆盖 SKILL.md / *.md / notes.MD)。
// toLowerCase() 让 .MD / .Md 也能命中(Windows / macOS 文件名大小写不一)。
// 没选中文件时返 false,模板里短路不会渲染大纲按钮。
const isMarkdownFile = computed(() => {
  const p = selectedFile.value?.path || ''
  return p.toLowerCase().endsWith('.md')
})

const currentMode = computed(() => {
  const skillName = props.skill?.name
  const path = selectedFile.value?.path || ''
  const m = getCurrentMode(skillName, path)
  dlog('[sfip currentMode]', { skillName, path, mode: m, editing: currentEditingPath.value })
  return m
})

const isDirty = computed(() => {
  const path = selectedFile.value?.path
  if (!path) return false
  if (!localFiles.has(path)) return false
  const current = localFiles.get(path) || ''
  const origFull = selectedFile.value?.content || ''
  const orig = path === 'SKILL.md' ? splitSkillMd(origFull).body : origFull
  // 2026-07-08 改 v2:跟 onContentChange 内的 dirty 判断保持一致,走"去掉末尾
  // 空白后比对"的归一化逻辑 —— 应对编辑器初始化时 emit 的尾随空白差异
  // (Tiptap 标准化、Monaco createModel 触发 onDidChangeContent 等),只有真正
  // 内容变更才视为 dirty。
  const normTail = (s) => String(s || '').replace(/\s+$/g, '')
  return normTail(current) !== normTail(orig)
})

const fileSize = computed(() => (currentContent.value || '').length)

function onContentChange(v) {
  // 2026-07-08 增:重置锁窗口内的 emit 丢弃,避免 Tiptap 飞行中的异步 update 把
  // 已经重置的 localFiles 重新写回用户的最新内容(详见 resetCurrent 注释)。
  if (Date.now() < resetLockUntil) return
  const path = selectedFile.value?.path
  if (!path) return
  const origFull = selectedFile.value?.content || ''
  const orig = path === 'SKILL.md' ? splitSkillMd(origFull).body : origFull
  // 2026-07-08 改 v2:原来用 setMode 时打的 80ms enterEditGuardUntil 时间窗不可靠
  // —— Tiptap/Monaco 初始化 emit 时机不可控(异步加载 worker + nextTick + ...
  // 都可能滞后超过 80ms),导致"刚进编辑模式就因 emit 的内容跟 orig 不完全等价
  // 而被判 dirty"。Tiptap 初始化会规范化末尾换行 / 空白;Monaco createModel
  // 写完整字符串时会触发一次 onDidChangeContent。
  //
  // 改用**内容归一化比对**:把 emit 回来的 v 跟 orig 都做一次"取最后一行尾随空白
  // 归一化"再比,只要去掉末尾 normalize 差异后内容相等,就不算 dirty。这是
  // 编辑器初始化 emit 的本质特征,不影响用户真实输入(用户改中间任何字符 v
  // 跟 orig 都不会归一化到相等)。
  const normalizeTail = (s) => String(s || '').replace(/\s+$/g, '')
  const v0 = Date.now() < enterEditGuardUntil
  if (v0) {
    // 锁窗内仍同步 localFiles(让编辑器状态对得上),但 dirtyPaths 不算 dirty
    localFiles.set(path, v || '')
    return
  }
  localFiles.set(path, v || '')
  if (normalizeTail(v) === normalizeTail(orig)) {
    // 初始化 emit 的尾巴:不更新 dirtyPaths
    if (dirtyPaths.value.has(path)) {
      const s = new Set(dirtyPaths.value)
      s.delete(path)
      dirtyPaths.value = s
    }
    return
  }
  // v 跟 orig 不只是"尾随空白"差异,真正的内容变更 → 标 dirty
  const s = new Set(dirtyPaths.value)
  s.add(path)
  dirtyPaths.value = s
}

function onDirtyChange(d) { /* no-op:用 localFiles 即时算 */ }

// ===== Frontmatter =====
const frontmatter = computed(() => {
  const md = (props.files || []).find((f) => f.path === 'SKILL.md')
  return parseFrontmatter(md?.content || '')
})

function parseFrontmatter(text) {
  if (!text) return {}
  const m = text.match(/^---\s*\n([\s\S]*?)\n---/)
  if (!m) return {}
  const block = m[1]
  const out = {}
  for (const line of block.split('\n')) {
    const kv = line.match(/^([a-zA-Z_][\w]*)\s*:\s*(.*)$/)
    if (!kv) continue
    const key = kv[1]
    let v = kv[2].trim()
    if (v.startsWith('[') && v.endsWith(']')) {
      v = v.slice(1, -1).split(',').map((s) => {
        let x = s.trim()
        if ((x.startsWith('"') && x.endsWith('"')) || (x.startsWith("\'") && x.endsWith("\'"))) {
          x = x.slice(1, -1)
        }
        return x
      }).filter(Boolean)
    } else if ((v.startsWith('"') && v.endsWith('"')) || (v.startsWith("\'") && v.endsWith("\'"))) {
      v = v.slice(1, -1)
    }
    out[key] = v
  }
  return out
}

const hasFrontmatter = computed(() => Object.keys(frontmatter.value).length > 0)
// 2026-07-12 增:技能版本号 — 用于详情区 title 行的 v1.0.0 灰色 badge 显示。
// 数据源(按优先级):
//   1. props.skill.canonical.manifest.version(getSkill full=true 时的权威值)
//   2. props.skill.version(SkillsView enriched 时塞进来的兜底值)
//   3. props.skill.skill_meta.version(list 原始 row 里的版本号)
// 三层兜底避免版本号丢失 → 否则 badge 会回退显示 source = LOCAL。
const skillVersion = computed(() => {
  const sk = props.skill || {}
  return String(
    sk?.canonical?.manifest?.version
    || sk?.version
    || sk?.skill_meta?.version
    || ''
  ).trim()
})
// 2026-07-12 增:技能简介 — 详情区顶部技能名下方那行灰色小字。
// 数据来源:props.skill.canonical.manifest.description(后端 store 在
// getSkill 时把 manifest 挂在 canonical.manifest 上);fallback 到
// currentMeta.description(预留路径,正常情况下前者已覆盖)。trim 后空串
// 就当无 description,模板里 v-if 不渲染那一行,避免出现孤立的占位。
const skillDescription = computed(() => {
  const sk = props.skill || {}
  const desc = sk?.canonical?.manifest?.description
    || sk?.description
    || ''
  return String(desc || '').trim()
})
const fmOpen = ref(false)
// 2026-07-10 改:openFrontmatter 现在只读 — 弹窗是"信息"按钮(原"班级按钮"),只展示
// frontmatter 内容(走回原只读表格)。"编辑"按钮(铅笔)改触发 editFmOpen 表单弹窗。
function openFrontmatter() { fmOpen.value = true }
function closeFrontmatter() { fmOpen.value = false }

// 2026-07-10 增:frontmatter 表单编辑弹窗(独立开关,跟只读 fmOpen 共存但互斥)。
// 由 SkillsView 传下来的 #name-actions 槽里的"编辑"按钮触发,复用 fmForm /
// saveFrontmatterForm,写回走 updateSkill。
const editFmOpen = ref(false)
function openFrontmatterEditor() {
  // 跟 openFrontmatter 一样的字段初始化逻辑
  fmFormError.value = ''
  fmFormSaving.value = false
  const fm = frontmatter.value
  fmForm.name = fm.name || skill.value?.name || ''
  fmForm.version = fm.version || skill.value?.version || ''
  fmForm.description = typeof fm.description === 'string' ? fm.description : ''
  fmForm.author = typeof fm.author === 'string' ? fm.author : ''
  fmForm.license = typeof fm.license === 'string' ? fm.license : ''
  const trg = Array.isArray(fm.triggers) ? fm.triggers : []
  fmForm.triggers = trg.map((s) => String(s || '')).filter((s) => s !== '')
  editFmOpen.value = true
}
function closeFrontmatterEditor() { editFmOpen.value = false }

// 2026-07-10 增:用表单弹窗做"新建 skill"时的字段初始化入口(由父级 startNew 调用)。
// 跟 openFrontmatterEditor 区别:openAsNew 忽略当前 skill 的 frontmatter,
// 用父级传进来的 initial 值(name/version/description/triggers 可选),让用户在
// 表单弹窗里继续编辑。保存时走同一条 saveFrontmatterForm 链路,但因为
// 走 createSkill 而不是 updateSkill,父级要传 isNew=true + 必要 scope 等参数。
// 这里只是把弹窗字段填好,scope 等 createSkill 需要的元数据由父级在调用 saveNewSkill
// 时组装 payload 传入。后端 payload 通过 emit('create-skill', payload) 传出去。
const newSkillInitial = ref(null) // null = 编辑模式,非 null = 新建模式 {name,version,description,triggers}
function openAsNew(initial = {}) {
  fmFormError.value = ''
  fmFormSaving.value = false
  fmForm.name = initial.name || ''
  fmForm.version = initial.version || '0.1.0'
  fmForm.description = initial.description || ''
  fmForm.author = initial.author || ''
  fmForm.license = initial.license || ''
  fmForm.triggers = Array.isArray(initial.triggers)
    ? initial.triggers.map((s) => String(s || '')).filter(Boolean)
    : []
  newSkillInitial.value = initial
  editFmOpen.value = true
}

// 2026-07-10 增:frontmatter 表单编辑态。
// fmForm 跟原 frontmatter computed 是分离的(避免直接改 reactive 引用),
// 保存时调用 saveFrontmatterForm() 把 fmForm 序列化写回 SKILL.md,
// 通过 updateSkill 走标准链路。
const fmForm = reactive({
  name: '',
  version: '',
  description: '',
  author: '',
  license: '',
  triggers: [],
})
const fmFormError = ref('')
const fmFormSaving = ref(false)

function addTrigger() {
  fmForm.triggers.push('')
}
function removeTrigger(idx) {
  fmForm.triggers.splice(idx, 1)
}

function normalizeFmTriggers() {
  // 去空 / 去重复 / trim,顺序保留(用户编辑顺序)
  const seen = new Set()
  const out = []
  for (const raw of fmForm.triggers || []) {
    const v = String(raw || '').trim()
    if (!v) continue
    if (seen.has(v)) continue
    seen.add(v)
    out.push(v)
  }
  return out
}

async function saveFrontmatterForm() {
  fmFormError.value = ''
  // 校验:name 必填,version 必填,description 至少 1 字符
  const name = String(fmForm.name || '').trim()
  const version = String(fmForm.version || '').trim()
  const desc = String(fmForm.description || '').trim()
  if (!name) { fmFormError.value = t('skills.editor.errNameEmpty'); return }
  if (!version) { fmFormError.value = t('skills.editor.errVersionEmpty'); return }
  if (!desc) { fmFormError.value = t('skills.editor.errDescriptionEmpty'); return }
  const triggers = normalizeFmTriggers()
  // 2026-07-12 改:触发词改为可选,不再要求至少 1 个。
  // - fmDict.triggers 处已经只在 triggers.length>0 时写入,所以空数组不会在
  //   SKILL.md frontmatter 里出现 triggers 这一行;
  // - manifest.triggers 仍透传(后端 RenderSkillMD 同理按数组长度决定是否输出)。
  // 这里不做早返回,任何残余校验留给 name/version/description(它们才是真必填)。

  fmFormSaving.value = true
  try {
    // 拼新的 fm 字典(按 FM_KEY_ORDER 顺序保持稳定,空字段不写)
    const fmDict = {}
    if (name) fmDict.name = name
    if (version) fmDict.version = version
    if (desc) fmDict.description = desc
    if (fmForm.author && String(fmForm.author).trim()) fmDict.author = String(fmForm.author).trim()
    if (fmForm.license && String(fmForm.license).trim()) fmDict.license = String(fmForm.license).trim()
    if (triggers.length) fmDict.triggers = triggers
    // 保留原 frontmatter 里其他字段(group_path / source / source_ref / depends_on / target_tools 等)
    // 新建模式没有 oldFm,跳过保留逻辑。
    const oldFm = newSkillInitial.value ? {} : frontmatter.value
    for (const k of Object.keys(oldFm)) {
      if (k in fmDict) continue
      if (k === 'name' || k === 'version' || k === 'description' || k === 'triggers') continue
      if (k === 'author' || k === 'license') continue
      fmDict[k] = oldFm[k]
    }
    // 2026-07-10 改:SKILL.md 不再拼 frontmatter 围栏。frontmatter 走 manifest 字段
    // 给后端,后端 RenderSkillMD 会按 manifest 重渲一份完整 SKILL.md。
    // 这里只取 body(编辑模式 = 当前 localFiles['SKILL.md'];新建模式 = 空)。
    let body = ''
    if (!newSkillInitial.value) {
      const path = selectedFile.value?.path
      body = path === 'SKILL.md'
        ? (localFiles.get('SKILL.md') || splitSkillMd(props.files.find((f) => f.path === 'SKILL.md')?.content || '').body)
        : ''
    }
    const newMd = body || ''

    if (newSkillInitial.value) {
      // ===== 新建模式 =====
      // 走 createSkill。scope / project_id / group_path 等元数据由父级
      // 在 openAsNew 调用时塞到 newSkillInitial 里(后续如果用户改 scope 范围,
      // 父级负责弹出 scope 选择面板 — 当前简化为只传 scope='global')。
      const init = newSkillInitial.value
      const payload = {
        scope: init.scope || 'global',
        project_id: init.project_id || 0,
        name,
        version,
        source: init.source || 'local',
        group_path: init.group_path || '',
        manifest: {
          name,
          version,
          description: desc,
          // 2026-07-12 改:触发词可选,空数组不写入 manifest,避免后端 RenderSkillMD
          // 产出空的 `triggers: []` 行;改用展开运算符按需附加。
          ...(triggers.length ? { triggers } : {}),
          author: fmDict.author || '',
          license: fmDict.license || '',
        },
        files: [{ path: 'SKILL.md', content: newMd }],
      }
      const created = await createSkill(payload)
      // 弹窗关闭 + 清掉新建态
      editFmOpen.value = false
      newSkillInitial.value = null
      // 通知父级:新建完成 + 新 skill 的 path(后端会回填 path)
      emit('created', {
        payload,
        response: created,
        name,
        version,
      })
    } else {
      // ===== 编辑模式 =====
      const sk = skill.value
      if (!sk || !sk.name) {
        fmFormError.value = t('skills.fileBrowser.noSkillSelected')
        fmFormSaving.value = false
        return
      }
      const incomingFiles = (props.files || []).map((f) => {
        if (!f || !f.path) return null
        if (f.path === 'SKILL.md') return { path: 'SKILL.md', content: newMd }
        return { path: f.path, content: f.content || '' }
      }).filter(Boolean)
      await updateSkill({
        scope: sk.scope || 'global',
        project_id: sk.project_id || 0,
        name: sk.name,
        version: sk.version,
        source: sk.source || 'local',
        manifest: {
          name,
          version,
          description: desc,
          // 2026-07-12 改:触发词可选,空数组不写入 manifest。
          ...(triggers.length ? { triggers } : {}),
          author: fmDict.author || '',
          license: fmDict.license || '',
        },
        files: incomingFiles,
      })
      // 本地同步:localFiles['SKILL.md'] 设为新 body(去掉 frontmatter 的部分)
      localFiles.set('SKILL.md', splitSkillMd(newMd).body)
      // 清掉 dirty(刚保存)
      const s = new Set(dirtyPaths.value)
      s.delete('SKILL.md')
      dirtyPaths.value = s
      // 弹窗关掉,emit saved 让父级刷新 currentFiles / listSkills
      editFmOpen.value = false
      emit('saved', { path: 'SKILL.md', content: newMd })
    }
  } catch (e) {
    fmFormError.value = e?.message || String(e)
  } finally {
    fmFormSaving.value = false
  }
}

const FM_KEY_ORDER = [
  'name', 'version', 'description', 'triggers',
  'author', 'license', 'depends_on', 'target_tools',
  'group_path', 'source', 'source_ref',
]
const frontmatterEntries = computed(() => {
  const fm = frontmatter.value
  const ordered = []
  for (const k of FM_KEY_ORDER) {
    if (k in fm) ordered.push([k, fm[k]])
  }
  for (const k of Object.keys(fm)) {
    if (!FM_KEY_ORDER.includes(k)) ordered.push([k, fm[k]])
  }
  return ordered
})

// ===== store root =====
const storeRoot = ref('')
async function fetchStoreRoot() {
  try {
    const info = await getStoreInfo()
    storeRoot.value = info?.store_root || ''
  } catch (_) { storeRoot.value = '' }
}

const skillRelPath = computed(() => {
  const gp = props.skill.group_path || ''
  return gp ? `${gp}/${props.skill.name || ''}` : (props.skill.name || '')
})

// ===== Save / Discard =====
const saving = ref(false)
const saveError = ref('')
// 2026-07-13 增:全局 Ctrl+S / Cmd+S 触发保存。
// 触发条件:编辑模式或当前文件 dirty(与工具栏"保存"按钮 v-if 一致)。
// macOS Cmd+S(metaKey),其它平台 Ctrl+S(ctrlKey)。输入法组合中不抢。
function onKeyDown(e) {
  if (e?.isComposing) return
  const isSaveCombo = (e.key === 's' || e.key === 'S') && (e.metaKey || e.ctrlKey) && !e.altKey
  if (!isSaveCombo) return
  // 当前未选中文件:不抢键(让浏览器自己处理,虽然桌面 webview 也没默认行为)
  if (!selectedFile.value?.path) return
  // 不在编辑模式、也没 dirty → 让浏览器默认行为(虽然理论上也不会保存网页)
  if (!isDirty.value && currentMode.value !== 'edit') return
  e.preventDefault()
  // 防止保存中重复触发(连按多次 Ctrl+S 只跑一次保存)
  if (saving.value) return
  // 异步触发,避免阻塞 keydown handler
  saveCurrent()
}
async function saveCurrent() {
  const path = selectedFile.value?.path
  if (!path) return
  const sk = props.skill
  if (!sk || !sk.name) {
    saveError.value = t('skills.fileBrowser.noSkillSelected')
    return
  }
  saving.value = true
  saveError.value = ''
  try {
    // 2026-07-08 改:后端 store.Save 是"原子全量覆盖"语义(SKILL.md 用 manifest 重渲,
    // 其它 files 走 c.Files 全量写到 tmp 目录再 rename)— 必须由 caller 拼出完整的
    // files 数组,否则就会丢文件。旧实现只 send 当前 dirty 文件 → 其他文件
    // 走 tmp 写不到 → os.RemoveAll 把原目录删了 → 那些文件就消失了。
    // 这里拿 props.files 当骨架,dirty 行用 localFiles 最新值,未 dirty 行用
    // 原 content 透传,确保发出去跟磁盘原本一致。
    const incomingFiles = (props.files || []).map((f) => {
      if (!f || !f.path) return null
      // SKILL.md 比较特殊:后端会用 RenderSkillMD(c.Manifest) 强制重写,
      // 这里送不送 content 都会被覆盖;但为了语义清晰,dirty 时仍按 localFiles 的 body 重拼
      if (dirtyPaths.value.has(f.path)) {
        if (f.path === 'SKILL.md') {
          const localBody = localFiles.get('SKILL.md') || ''
          return { path: 'SKILL.md', content: rebuildSkillMdFromBody(localBody) }
        }
        return { path: f.path, content: localFiles.get(f.path) || '' }
      }
      return { path: f.path, content: f.content || '' }
    }).filter(Boolean)
    // 兜底:极端情况 props.files 为空(还没回填)→ 退化为老行为 + 弹警告
    if (incomingFiles.length === 0) {
      const fallback = path === 'SKILL.md'
        ? { path: 'SKILL.md', content: rebuildSkillMd() }
        : { path, content: localFiles.get(path) || '' }
      incomingFiles.push(fallback)
      saveError.value = t('skills.fileBrowser.incompleteFilesWarning')
    }
    await updateSkill({
      scope: sk.scope || 'global',
      project_id: sk.project_id || 0,
      name: sk.name,
      version: sk.version,
      source: sk.source || 'local',
      manifest: sk.canonical?.manifest || {
        name: sk.name, version: sk.version,
      },
      files: incomingFiles,
    })
    const s = new Set(dirtyPaths.value)
    s.delete(path)
    dirtyPaths.value = s
    // 同步 clean 所有"刚被发出去"的文件(包括 SKILL.md 重建后的新内容)
    for (const f of incomingFiles) {
      const stored = f.path === 'SKILL.md' ? splitSkillMd(f.content || '').body : (f.content || '')
      localFiles.set(f.path, stored)
      s.delete(f.path)
    }
    dirtyPaths.value = s
    const savedContent = path === 'SKILL.md' ? rebuildSkillMd() : (localFiles.get(path) || '')
    // 2026-07-08 改:保存成功后退出编辑态,跟 resetCurrent 一致。"放弃/保存"
    // 按钮自动消失,工具栏恢复显示"编辑"铅笔图标。
    setMode(props.skill?.name, path, 'view')
    emit('saved', { path, content: savedContent })
  } catch (e) {
    saveError.value = e?.message || String(e)
    toast.error(t('skills.fileBrowser.saveFailed', { msg: saveError.value }))
  } finally {
    saving.value = false
  }
}

// 2026-07-10 改:跟 rebuildSkillMd 配套的"从 body 反推完整 SKILL.md"的工具。
// 原本职责:复用已有 frontmatter + 拼 body 得完整字符串。
// 现状:SKILL.md 不再携带 frontmatter 围栏(完全交给 manifest 字段 + 后端
// RenderSkillMD 重渲),这里只返回 body,frontmatter 透传给调用方走 manifest。
function rebuildSkillMdFromBody(body) {
  return body || ''
}

// 2026-07-10 改:SKILL.md 只返 body。frontmatter 走 manifest 字段给后端,
// 后端 RenderSkillMD 会按 manifest 重渲一份完整 SKILL.md(含干净的 frontmatter)。
function rebuildSkillMd() {
  const path = selectedFile.value?.path
  return path === 'SKILL.md' ? (localFiles.get('SKILL.md') || '') : ''
}

function resetCurrent() {
  const path = selectedFile.value?.path
  if (!path) return
  const origFull = selectedFile.value?.content || ''
  const orig = path === 'SKILL.md' ? splitSkillMd(origFull).body : origFull
  // 2026-07-08 修:放弃修改要点 2 次才生效的根因 — Tiptap 编辑器的 onUpdate 是
  // 防抖异步(且 Markdown→HTML→Markdown 转换后字符串跟编辑器实时内容不完全等价),
  // 用户点"放弃修改"瞬间,可能还有一帧 onUpdate 在飞行中:resetCurrent 把 localFiles
  // 重置回 orig 后,Tiptap 那一帧 emit 触发 onContentChange 再次写回用户的最新内容,
  // dirtyPaths 重新被加上,看起来像"放弃失败"。py/json 等 Monaco 走同步
  // onDidChangeContent,没有这个异步窗口,所以一次就生效。
  //
  // 修法:resetCurrent 时打 resetLock 标记,接下来 80ms 内 CodeViewer 传回来的
  // update:content 全部丢弃(避免被 Tiptap 飞行中那一帧覆盖)。Monaco 走同样逻辑,
  // 保证两个编辑器的重置行为统一。
  resetLockUntil = Date.now() + 80
  localFiles.set(path, orig)
  const s = new Set(dirtyPaths.value)
  s.delete(path)
  dirtyPaths.value = s
  // 2026-07-08 改:放弃成功后退出编辑态,回到 view 模式。这样"放弃/保存"按钮
  // 自动消失(由 v-if="currentMode === 'edit' || isDirty" 控制),工具栏恢复
  // 显示"编辑"铅笔图标;CodeViewer 内部从 Monaco/Tiptap 切回 hljs 只读渲染。
  setMode(props.skill?.name, path, 'view')
}

// ====== ErrorBoundary (render error 兜底) ======
const renderError = ref(null)
function safeReload() {
  renderError.value = null
  fetchStoreRoot()
}
onErrorCaptured((err) => {
  console.error('[SkillFileInlinePanel captured]', err)
  renderError.value = err?.message || String(err)
  return false
})

// 2026-07-12 增:简介 hover 自定义快速 tooltip 状态。
// 不依赖浏览器 native title(默认延迟 0.5~1s,等得人抓狂),自己用
// mouseenter/mouseleave + setTimeout 控制 hover 浮层显示。
// - 150ms 延迟避免快速划过去误触发
// - leave 时 clearTimeout 防"显示→立刻被取消"
// - tip 用 Teleport 挂到 body 下 + position:fixed + getBoundingClientRect
//   算视口坐标,避免被父级 overflow / transform / position:relative
//   改变 containing block 导致 tip 错位("左上角显示且不完整"那个 bug)。
// - tipStyle 动态设 top / left / maxWidth,最大宽度 480px 但受视口约束。
const sfipDescEl = ref(null)
const descTipShow = ref(false)
const tipStyle = reactive({ top: '0px', left: '0px', maxWidth: '480px' })
let descTipTimer = 0
function positionTip() {
  const el = sfipDescEl.value
  if (!el) return
  const r = el.getBoundingClientRect()
  // 默认放在 desc 底部下方 6px,左对齐
  let top = r.bottom + 6
  let left = r.left
  // 视口宽度限制:不让 tip 越过右边界 16px
  const vw = window.innerWidth || document.documentElement.clientWidth
  const tipMaxW = Math.min(480, vw - 32)
  if (left + tipMaxW > vw - 16) {
    left = Math.max(16, vw - 16 - tipMaxW)
  }
  // 顶部溢出保护:如果 desc 下方空间不够,改放上方
  const vh = window.innerHeight || document.documentElement.clientHeight
  if (top + 80 > vh) {
    top = Math.max(8, r.top - 6)
    // 动态给个 [data-placement] 让 CSS 切换箭头方向(暂未实现箭头,留接口)
    el.dataset.tipPlacement = 'top'
  } else {
    el.dataset.tipPlacement = 'bottom'
  }
  tipStyle.top = `${Math.round(top)}px`
  tipStyle.left = `${Math.round(left)}px`
  tipStyle.maxWidth = `${tipMaxW}px`
}
function onDescEnter() {
  if (descTipTimer) clearTimeout(descTipTimer)
  descTipTimer = setTimeout(() => {
    descTipShow.value = true
    // nextTick 等 Teleport 真正把 tip 渲染出来后再算坐标
    nextTick(positionTip)
  }, 150)
}
function onDescLeave() {
  if (descTipTimer) { clearTimeout(descTipTimer); descTipTimer = 0 }
  descTipShow.value = false
}
onUnmounted(() => {
  // 组件卸载时清掉残留 timer,避免在已销毁组件上回调
  if (descTipTimer) { clearTimeout(descTipTimer); descTipTimer = 0 }
  // 2026-07-13 增:卸载快捷键监听,避免 InlinePanel :key 重 mount 时旧实例残留
  window.removeEventListener('keydown', onKeyDown)
})

// =====================================================================
// 2026-07-11 增:文件树右键菜单 + 文件/目录 CRUD
// 需求:按位置区分右键菜单
//   - 文档(文件)节点:重命名 + 删除
//   - 分组(目录)节点:新建文件
//   - 根区域(树空白):新建文件夹 + 新建文件
// 设计:所有 CRUD 走 updateSkill({...files: incomingFiles}) 全量提交,后端
//   store.Save 是覆盖式 — 这跟现有"保存一个文件"的链路完全一致,不需要
//   新增后端端点。
//   唯一不变量: SKILL.md 由后端按 manifest 重新渲染,前端不能从 files 数组
//   里删 SKILL.md(后端会自己生成)。新增/重命名/删除文件时,其他文件原样保留。
// =====================================================================

// 右键菜单单例
const ctxMenu = reactive({ open: false, x: 0, y: 0, items: [] })
function closeCtxMenu() {
  ctxMenu.open = false
  ctxMenu.items = []
}

// 文件节点右键:重命名 / 删除
function onCtxFile({ file, event }) {
  ctxMenu.x = event.clientX
  ctxMenu.y = event.clientY
  ctxMenu.items = [
    {
      key: 'rename-file',
      label: t(LABEL_CTX_RENAME_FILE),
      icon: 'mdi:rename-outline',
      onClick: () => openRenameFileDialog(file),
    },
    { divided: true, key: 'div-1', label: '' },
    {
      key: 'delete-file',
      label: t(LABEL_CTX_DELETE_FILE),
      icon: 'mdi:delete',
      danger: true,
      onClick: () => openDeleteFileDialog(file),
    },
  ]
  ctxMenu.open = true
}

// 目录节点右键:新建文件 + 重命名 + 在文件浏览器中打开
// 2026-07-11 改:目录右键只支持 1 项的简化版用户反馈"操作太少",
// 加上重命名(对目录名最后一段做修改)和在文件浏览器中打开
// (直接定位到磁盘目录)。
function onCtxFolder({ dir, event }) {
  ctxMenu.x = event.clientX
  ctxMenu.y = event.clientY
  const dirPath = dir.path || ''
  ctxMenu.items = [
    {
      key: 'new-file',
      label: t(LABEL_CTX_NEW_FILE),
      icon: 'mdi:file-document-plus-outline',
      onClick: () => openNewFileDialog(dirPath),
    },
    { divided: true, key: 'div-1', label: '' },
    {
      key: 'rename-folder',
      label: t(LABEL_CTX_RENAME_FOLDER),
      icon: 'mdi:rename-outline',
      onClick: () => openRenameFolderDialog(dir),
    },
    {
      key: 'open-folder',
      label: t(LABEL_CTX_OPEN_FOLDER),
      icon: 'mdi:folder-outline',
      onClick: () => openFolderInExplorer(dirPath),
    },
    { divided: true, key: 'div-2', label: '' },
    {
      // 2026-07-12 增:删除文件夹 — 复用现有 deleteFile 弹窗(支持 kind='dir')
      key: 'delete-folder',
      label: t(LABEL_CTX_DELETE_FOLDER),
      icon: 'mdi:folder-remove-outline',
      danger: true,
      onClick: () => openDeleteFolderDialog(dir),
    },
  ]
  ctxMenu.open = true
}

// 根区域右键:新建文件夹 / 新建文件 / 在文件浏览器中打开
// 2026-07-12 改:加"在文件浏览器中打开"项(走 openFolderInExplorer('') =
// 打开 skill 根目录)。跟目录节点右键的"在文件浏览器中打开"复用同一项,
// 用户反馈右键空白处缺这个入口,导致想直接定位磁盘目录还得先建个空文件夹
// 然后右键它,体验割裂。
function onCtxRoot({ event }) {
  // 2026-07-11 增:诊断日志 — 确认 root context menu 事件是否到达 InlinePanel
  console.log('[InlinePanel] onCtxRoot fired at', event?.clientX, event?.clientY)
  ctxMenu.x = event.clientX
  ctxMenu.y = event.clientY
  ctxMenu.items = [
    {
      key: 'new-dir',
      label: t(LABEL_CTX_NEW_DIR),
      icon: 'mdi:folder-plus-outline',
      onClick: () => openNewFileDialog('', { kind: 'dir' }),
    },
    {
      key: 'new-file',
      label: t(LABEL_CTX_NEW_FILE),
      icon: 'mdi:file-document-plus-outline',
      onClick: () => openNewFileDialog(''),
    },
    { divided: true, key: 'div-root-1', label: '' },
    {
      key: 'open-folder',
      label: t(LABEL_CTX_OPEN_FOLDER),
      icon: 'mdi:folder-outline',
      onClick: () => openFolderInExplorer(''),
    },
  ]
  ctxMenu.open = true
}

// === 新建文件 / 新建目录 弹窗 ===
// 入参: dirPath(父目录的相对路径;空 = 根), opts.kind = 'file' | 'dir'
const newFileOpen = ref(false)
const newFileDirPath = ref('')
const newFileKind = ref('file') // 'file' | 'dir'
const newFileInput = ref('')
const newFileError = ref('')
const newFileBusy = ref(false)

function openNewFileDialog(dirPath, opts = {}) {
  newFileDirPath.value = dirPath || ''
  newFileKind.value = opts.kind || 'file'
  newFileInput.value = ''
  newFileError.value = ''
  newFileOpen.value = true
}
function closeNewFileDialog() {
  if (newFileBusy.value) return
  newFileOpen.value = false
}

// 文件名 / 目录名校验:不允许 '/',不允许空,不允许 '..',不允许 'SKILL.md'
// (SKILL.md 由 manifest 重新生成,前端不能直接创建)。
function validateFsName(name, kind) {
  const v = (name || '').trim()
  if (!v) return t('skills.fileBrowser.validation.nameRequired')
  if (v.includes('/') || v.includes('\\')) return t('skills.fileBrowser.validation.invalidSeparator')
  if (v === '.' || v === '..') return t('skills.fileBrowser.validation.invalidDotName')
  if (kind === 'file' && v === 'SKILL.md') return t('skills.fileBrowser.validation.invalidSKILL')
  return ''
}
async function submitNewFile() {
  if (newFileBusy.value) return
  const name = (newFileInput.value || '').trim()
  const err = validateFsName(name, newFileKind.value)
  if (err) { newFileError.value = err; return }
  const parent = newFileDirPath.value || ''
  const fullPath = parent ? `${parent}/${name}` : name
  // 重复检测:同路径文件已存在 → 拒
  const existing = (props.files || []).find((f) => f.path === fullPath)
  if (existing) {
    newFileError.value = t('skills.fileBrowser.validation.duplicateName')
    return
  }
  newFileBusy.value = true
  try {
    if (newFileKind.value === 'dir') {
      // 2026-07-11 改:不写占位文件到磁盘(用户反馈"目录里默认有个文件"很奇怪)。
      // 走 createGroup 端点(已有 — /api/skillbox/skills/group/create),只创建
      // 物理目录,files[] 不变。后续 loadFromDir 走 listEmptyDirs 补 .skillbox-placeholder
      // 占位条目,前端 buildTree 走 BUSINESS_PLACEHOLDERS 白名单让目录显示。
      const { createGroup: apiCreateGroup } = await import('@/api/skillbox/skills')
      // group_path 是 skill_root 内部的相对路径(<group_path> + <name>)
      // 例如 skill_name = 'aa' 时,group_path = 'aa/<dir_name>'。
      // 根目录下新建:group_path = '<dir_name>',store 走 CreateGroupDir。
      const sk = props.skill || {}
      const skillName = sk.name || ''
      const groupPath = skillName
        ? `${skillName}/${fullPath}`
        : fullPath
      await apiCreateGroup({ group_path: groupPath })
      newFileOpen.value = false
      // 通知父级 reload — 触发 listEmptyDirs 重新扫描
      emit('saved', { path: '__dir-created__', content: '' })
    } else {
      // 文件走原有 updateSkill 链路(需要把新文件加进 files[])
      await persistFiles([
        ...(props.files || []),
        { path: fullPath, content: '' },
      ])
      newFileOpen.value = false
      // 选中新创建的文件
      const created = (props.files || []).find((f) => f.path === fullPath)
      if (created) selectFileByPath(fullPath)
    }
    toast.success(newFileKind.value === 'dir'
      ? t('skills.fileBrowser.createdDir', { name })
      : t('skills.fileBrowser.createdFile', { name }))
  } catch (e) {
    newFileError.value = e?.message || String(e)
  } finally {
    newFileBusy.value = false
  }
}

// === 重命名文件 弹窗 ===
const renameFileOpen = ref(false)
const renameFileOldPath = ref('')
const renameFileOldName = ref('')
const renameFileInput = ref('')
const renameFileError = ref('')
const renameFileBusy = ref(false)

// === 重命名文件夹 弹窗(2026-07-11 增)===
const renameFolderOpen = ref(false)
const renameFolderOldPath = ref('')
const renameFolderOldName = ref('')
const renameFolderInput = ref('')
const renameFolderError = ref('')
const renameFolderBusy = ref(false)

function openRenameFileDialog(file) {
  if (!file || !file.path) return
  const seg = file.path.split('/').pop() || ''
  renameFileOldPath.value = file.path
  renameFileOldName.value = seg
  renameFileInput.value = seg
  renameFileError.value = ''
  renameFileOpen.value = true
}
function closeRenameFileDialog() {
  if (renameFileBusy.value) return
  renameFileOpen.value = false
}
async function submitRenameFile() {
  if (renameFileBusy.value) return
  const newName = (renameFileInput.value || '').trim()
  const err = validateFsName(newName, 'file')
  if (err) { renameFileError.value = err; return }
  if (newName === renameFileOldName.value) { renameFileOpen.value = false; return }
  const parent = renameFileOldPath.value.includes('/')
    ? renameFileOldPath.value.slice(0, renameFileOldPath.value.lastIndexOf('/'))
    : ''
  const newPath = parent ? `${parent}/${newName}` : newName
  // 重复检测
  const dup = (props.files || []).find((f) => f.path === newPath && f.path !== renameFileOldPath.value)
  if (dup) { renameFileError.value = t('skills.fileBrowser.validation.duplicateFile'); return }
  renameFileBusy.value = true
  try {
    const next = (props.files || []).map((f) =>
      f.path === renameFileOldPath.value ? { ...f, path: newPath } : f
    )
    await persistFiles(next)
    renameFileOpen.value = false
    selectFileByPath(newPath)
    toast.success(t('skills.fileBrowser.renamed', { name: newName }))
  } catch (e) {
    renameFileError.value = e?.message || String(e)
  } finally {
    renameFileBusy.value = false
  }
}

// === 重命名文件夹 弹窗 实现(2026-07-11 增)===
function closeRenameFolderDialog() {
  if (renameFolderBusy.value) return
  renameFolderOpen.value = false
}
async function submitRenameFolder() {
  if (renameFolderBusy.value) return
  const newName = (renameFolderInput.value || '').trim()
  // 复用文件名校验(目录名跟文件名规则一样,不允许 / 和 ..,允许 SKILL.md 是文件专属)
  const err = validateFsName(newName, 'dir')
  if (err) { renameFolderError.value = err; return }
  if (newName === renameFolderOldName.value) { renameFolderOpen.value = false; return }
  const parent = renameFolderOldPath.value.includes('/')
    ? renameFolderOldPath.value.slice(0, renameFolderOldPath.value.lastIndexOf('/'))
    : ''
  const newPath = parent ? `${parent}/${newName}` : newName
  renameFolderBusy.value = true
  try {
    // 把所有 <oldDir>/ 前缀的 file.path 改成 <newDir>/
    // 2026-07-12 改:不再 filter 删占位条目 — 上一版的 filter 会把 cc/.skillbox-placeholder
    // 整条删掉,后端 Save 是全量覆盖式,占位条目一删磁盘上的 cc 目录就永久
    // 消失(用户报告"重命名 cc→dd 后两个都没了")。这里改成保留占位条目
    // 走 prefix 替换 — cc/.skillbox-placeholder → dd/.skillbox-placeholder,
    // 后端 Save 把它转成 mkdir cc/ + mkdir dd/(在 aa/ 下),cc 旧目录会被
    // RemoveAll 清掉、dd 新目录由 MkdirAll 建出,两个空目录都保留。
    const oldPrefix = renameFolderOldPath.value + '/'
    const next = (props.files || []).map((f) => {
      if (!f || !f.path) return f
      if (f.path === renameFolderOldPath.value) {
        return { ...f, path: newPath + '/' + f.path.slice(oldPrefix.length) }
      }
      if (f.path.startsWith(oldPrefix)) {
        return { ...f, path: newPath + '/' + f.path.slice(oldPrefix.length) }
      }
      return f
    })
    await persistFiles(next)
    renameFolderOpen.value = false
    toast.success(t('skills.fileBrowser.renamed', { name: newName }))
  } catch (e) {
    renameFolderError.value = e?.message || String(e)
  } finally {
    renameFolderBusy.value = false
  }
}

// === 删除文件 弹窗 ===
const deleteFileOpen = ref(false)
const deleteFileTarget = ref(null) // { path, name, kind: 'file'|'dir', childCount? }
const deleteFileBusy = ref(false)

function openDeleteFileDialog(file) {
  if (!file || !file.path) return
  deleteFileTarget.value = {
    path: file.path,
    name: file.path.split('/').pop(),
    kind: 'file',
  }
  deleteFileOpen.value = true
}
function openDeleteFolderDialog(dir) {
  if (!dir || !dir.path) return
  // 统计子文件数(用于弹窗展示"包含 N 个文件")
  const prefix = dir.path + '/'
  const childPaths = (props.files || []).filter((f) => f.path.startsWith(prefix)).map((f) => f.path)
  deleteFileTarget.value = {
    path: dir.path,
    name: dir.path.split('/').pop(),
    kind: 'dir',
    childCount: childPaths.length,
    childPaths,
  }
  deleteFileOpen.value = true
}
function closeDeleteFileDialog() {
  if (deleteFileBusy.value) return
  deleteFileOpen.value = false
  deleteFileTarget.value = null
}
async function submitDeleteFile() {
  if (deleteFileBusy.value || !deleteFileTarget.value) return
  const target = deleteFileTarget.value
  deleteFileBusy.value = true
  try {
    let next = props.files || []
    // 2026-07-12 增:显式告诉后端"我要删这些路径",避免 Save 阶段 WalkDir
    // 把已删的目录/文件从原 dir 复制回 tmp 复活(只有后端 Save 收到这个
    // 列表,才会真正物理删除对应路径)。
    const deletedPaths = [target.path]
    if (target.kind === 'file') {
      // 保护 SKILL.md(由后端按 manifest 重建,前端不能"删")
      if (target.path === 'SKILL.md') {
        toast.error(t('skills.fileBrowser.validation.invalidSKILL'))
        deleteFileOpen.value = false
        deleteFileTarget.value = null
        return
      }
      next = next.filter((f) => f.path !== target.path)
    } else {
      // 目录:删所有以 dir.path/ 为前缀的文件
      const prefix = target.path + '/'
      next = next.filter((f) => f.path !== target.path && !f.path.startsWith(prefix))
    }
    await persistFiles(next, deletedPaths)
    deleteFileOpen.value = false
    deleteFileTarget.value = null
    // 当前选中的文件被删了 → 切到 SKILL.md
    if (selectedFile.value && selectedFile.value.path === target.path) {
      selectFileByPath('SKILL.md')
    }
    toast.success(t('skills.fileBrowser.deletedItem', { name: target.name }))
  } catch (e) {
    toast.error(e?.message || String(e))
  } finally {
    deleteFileBusy.value = false
  }
}

// 共享:把 files 数组 updateSkill 持久化,并更新 localFiles 镜像。
// 复用现有 saveCurrent 的链路,只是不重渲 SKILL.md(本组件不维护 manifest)。
//
// 2026-07-12 增:deletedPaths(可选)是前端"明确删除"路径列表,会附加到
// updateSkill payload 的 deleted_paths 字段;后端 Save 收到后会在
// WalkDir 复制阶段跳过这些路径,让物理删除真正落地。不传等价于 nil,
// 走原有 "保留前端不知道的文件" 逻辑(向后兼容普通保存 / 重命名等场景)。
async function persistFiles(files, deletedPaths) {
  const sk = props.skill || {}
  // 2026-07-12 改:不再剥 .skillbox-placeholder 占位条目 — 这些条目是
  // 空目录标识,后端 Save 会用它们 mkdir 对应的空目录(2026-07-12 改
  // store.Save 处理)。剥了之后磁盘上空目录永久丢失 — 用户报告"重命名
  // 空目录后连旧目录都没了"就是因为这个 filter + 后端全量覆盖 Save
  // 双重作用。
  const stripped = files || []
  const payload = {
    scope: sk.scope || 'global',
    project_id: sk.project_id || 0,
    name: sk.name,
    version: sk.version,
    source: sk.source || 'local',
    manifest: sk.canonical?.manifest || {
      name: sk.name, version: sk.version,
    },
    files: stripped,
  }
  if (deletedPaths && deletedPaths.length) {
    payload.deleted_paths = [...deletedPaths]
  }
  await updateSkill(payload)
  // 同步本地缓存 — 用 stripped 跟后端保持一致,避免占位条目留缓存里被下次
  // 编辑循环回带出。
  localFiles.clear()
  for (const f of stripped) {
    if (f.path === 'SKILL.md') {
      const { body } = splitSkillMd(f.content || '')
      localFiles.set(f.path, body)
    } else {
      localFiles.set(f.path, f.content || '')
    }
  }
  // 通知父级刷新(让 SkillsView 重新拉 files 列表,清掉 dirtyPaths 等)
  emit('saved', { path: '__files-changed__', content: '' })
}

function selectFileByPath(path) {
  const f = (props.files || []).find((x) => x.path === path)
  if (f) onSelectFile(f)
}

// 2026-07-11 增:目录节点的 3 个新操作函数
// 1) 重命名目录:复用重命名文件弹窗的样式,只把"最后一段"改成新名,父路径不变。
//    走 updateSkill 链路:把所有 <dir>/ 前缀的 file.path 替换成 <newDir>/
function openRenameFolderDialog(dir) {
  if (!dir || !dir.path) return
  const seg = dir.path.split('/').pop() || ''
  renameFolderOldPath.value = dir.path
  renameFolderOldName.value = seg
  renameFolderInput.value = seg
  renameFolderError.value = ''
  renameFolderOpen.value = true
}
// 2) 在文件浏览器中打开:platform.fs.reveal(物理目录绝对路径)
//
// 2026-07-12 改:之前只看 sk.canonical?.source_dir / source_path,如果当前 skill 的
// canonical 是 nil 或字段缺失,就直接弹"缺少 source_dir"错误。用户反馈右键
// 根区域空白 → 在文件夹中打开 100% 失败,即使 store 里有这个 skill。
//
// 兜底链(从优先到兜底):
//   1. sk.canonical?.source_path(后端 get_skill?full=true 时返;Canonical.SourceDir
//      是 json:"-" 不导出,所以这里读不到)
//   2. getStoreInfo().store_root + sk.group_path + name(skill 在 store 内的物理根)
//   3. 如果 store_root 也拿不到 → 真正失败,弹"无法定位"
//
// 拼路径时把 dirPath(目录右键时带的相对路径)拼到 store 内的物理目录上,
// 跟 SkillsView.openGroupInFolder(走 store_root + 相对路径) 行为一致。
async function openFolderInExplorer(dirPath) {
  const sk = props.skill || {}
  let srcDir = sk.canonical?.source_path || ''
  if (!srcDir) {
    // 兜底:走 store-info 拿 store 物理根,再拼上 skill 的 group_path + name
    try {
      const info = await getStoreInfo()
      const root = info?.store_root || ''
      if (root) {
        const gp = sk.group_path || ''
        const nm = sk.name || ''
        srcDir = gp ? `${root}/${gp}/${nm}` : `${root}/${nm}`
      }
    } catch (_) {
      // ignore
    }
  }
  if (!srcDir) {
    toast.error(t('skills.fileBrowser.sourceDirMissing'))
    return
  }
  const abs = dirPath ? `${srcDir}/${dirPath}` : srcDir
  try {
    const r = await platform.fs.reveal(abs)
    if (r && r.ok === false && r.fallbackUrl) {
      platform.platform.openExternal(r.fallbackUrl)
    }
  } catch (e) {
    toast.error(t('common.openFailed', { msg: e?.message || String(e) }))
  }
}

onUnmounted(() => {})

// 2026-07-07 增:暴露给父组件的方法。
// 父在切换 skill / 切换文件前调 ensureCleanBeforeSwitch(),有 dirty 时弹
// 三选项弹窗,等用户决策返回 'proceed' / 'cancel' 后再决定是否继续。
// resetDirtyNow 是父级强制清 dirty 的兜底(例如删除 skill 流程不需要询问)。
// 2026-07-10 增:openFrontmatterEditor 暴露给父级,让父级的"编辑"按钮 /
// "新建"按钮都能直接打开 InlinePanel 内部的 frontmatter 表单弹窗。
defineExpose({
  ensureCleanBeforeSwitch,
  resetAllDirty,
  isAnyDirty: () => dirtyPaths.value.size > 0,
  clearEditingState,
  openFrontmatterEditor,
  // 2026-07-10 增:openAsNew 复用表单弹窗做"新建 skill"(空字段),
  // 父级(SkillsView.startNew)传 (name, version, description, triggers)
  // 进来,弹窗打开后用户继续填(触发词可编辑)。
  openAsNew,
})
</script>

<template>
  <!-- 渲染异常时的降级 UI -->
  <div v-if="renderError" class="sfip-error">
    <IconPark icon="mdi:alert-circle-outline" width="22" height="22" />
    <h4>{{ t(LABEL_RENDER_ERROR_TITLE) }}</h4>
    <p class="sfip-error-msg">{{ renderError }}</p>
    <button class="primary sm" @click="safeReload">
      <IconPark icon="mdi:refresh" width="14" height="14" />
      {{ t(LABEL_RETRY) }}
    </button>
  </div>
  <div v-else class="sfip">
    <header class="sfip-header">
      <div class="sfip-title-block">
        <IconPark icon="FileCabinet" width="16" height="16" />
        <!-- 2026-07-12 改:把"标题行"(name + version + 文件数 + source 徽标 +
             name-actions 编辑槽)和"描述行"(skillDescription)竖向堆在一列里。

             .sfip-name-row 是标题行,所有标题相关元素都在同一横排:
             name + @version + N files + LOCAL/market 徽标 + 编辑按钮,
             用户期望这堆"标题属性"在一起;
             .sfip-desc 单独下一行,灰色小字,只在有简介时渲染。

             整体仍占据 .sfip-title-block 横排中的"标题位",width 自适应
             (min-width:0 防止把 stack 撑到右侧);.sfip-actions 仍
             margin-left:auto 推到 .sfip-header 最右。 -->
        <div class="sfip-title-stack">
          <div class="sfip-name-row">
            <span class="sfip-name">{{ skill?.name || '' }}</span>
            <span v-if="skillVersion" class="sfip-version">@{{ skillVersion }}</span>
            <span class="sfip-count">{{ (files || []).length }} {{ t(LABEL_FILES, { n: (files || []).length }) }}</span>
            <!-- 2026-07-12 改:badge 之前显示 v{version} 跟 @version 重复;用户反馈
                 "版本号重复显示了,编辑前面的版本号就要显示了",去掉 v{version} badge。
                 留 name-actions 编辑按钮组,无中间 badge 占位,标题行更紧凑。 -->
            <span class="sfip-name-actions">
              <slot name="name-actions" />
            </span>
          </div>
          <!-- 2026-07-12 改:简介单行截断 + mouseenter/mouseleave 自定义快速
               tooltip(150ms 延迟)。
               - 不挂 native :title(1s 延迟 + 浏览器渲染太慢)
               - tip 用 <Teleport to="body"> + position:fixed + 视口坐标,
                 避免被 .sfip-header / .sfip-title-block 的 overflow /
                 transform / position 影响导致"tip 在左上角显示且不完整"。
               - positionTip 用 getBoundingClientRect 算 desc 真实视口位置,
                 desc 下方空间不足时自动改放上方。 -->
          <span
            v-if="skillDescription"
            ref="sfipDescEl"
            class="sfip-desc"
            @mouseenter="onDescEnter"
            @mouseleave="onDescLeave"
          >{{ skillDescription }}</span>
        </div>
      </div>
      <Teleport to="body">
        <div
          v-if="descTipShow && skillDescription"
          class="sfip-desc-tip"
          :style="{ top: tipStyle.top, left: tipStyle.left, maxWidth: tipStyle.maxWidth }"
        >{{ skillDescription }}</div>
      </Teleport>
      <div class="sfip-actions">
        <slot name="actions" />
      </div>
      <button
        v-if="hasFrontmatter"
        class="sfip-fm-btn"
        :data-tip="t(LABEL_FRONTMATTER_TITLE)"
        :aria-label="t(LABEL_FRONTMATTER_TITLE)"
        @click="openFrontmatter"
      >
        <IconPark icon="Info" width="15" height="15" />
      </button>
    </header>

    <div ref="sfipBodyEl" class="sfip-body">
      <nav class="sfip-left">
        <!-- 2026-07-07 改 v3:作用域区移到文件树底部。
             旧版:作用域在顶部 → 用户第一眼看到的是 scope,文件树被挤。
             新版:文件树在上(占主要空间),作用域在底部(辅助信息,默认折叠,
             用户主动展开才看得到生效位置)。 -->
        <div v-thin-scrollbar class="sfip-tree-wrap">
          <!-- 2026-07-07 增:文件树加标题栏,跟 .ssp-scope-header 风格一致 -->
          <header class="sfip-tree-header">
            <!-- 2026-07-08 改:PascalCase 直传 FileCabinet(避免 mdi 映射兜底导致的"看不见"
                 现象)。多文件柜图标跟"skill 目录树"语义贴合(文件夹集合)。 -->
            <IconPark icon="FileCabinet" width="13" height="13" />
            <span>{{ t('skills.fileBrowser.skillDirectory') }}</span>
            <span class="sfip-tree-header-count">{{ (files || []).length }} {{ t('common.count') }}</span>
          </header>
          <FileTreeView
            v-if="(files || []).length"
            :files="files"
            :initial-selected-path="selectedKey"
            :dirty-paths="dirtyPaths"
            @select-file="onSelectFile"
            @context-menu-file="onCtxFile"
            @context-menu-folder="onCtxFolder"
            @context-menu-root="onCtxRoot"
          />
        </div>
        <SkillScopePanel :skill="skill" />
      </nav>

      <!-- 拖拽把手:拖右边界改变目录树宽度(双击重置) -->
      <div
        class="sfip-resizer"
        :class="{ 'sfip-resizer-dragging': sfipLeftDragging }"
        role="separator"
        aria-orientation="vertical"
        :aria-valuenow="sfipLeftWidth"
        aria-valuemin="180"
        aria-valuemax="500"
        title="拖动调整宽度(双击重置)"
        @mousedown="onSfipLeftStartDrag"
        @dblclick="resetSfipLeftWidth"
      />

      <main class="sfip-viewer">
        <header class="sfip-viewer-header">
          <!-- 2026-07-08 改:不再显示 selectedFile.path + fileSize,
               CodeViewer 内部 cv-text-toolbar 已经显示 language 标签,
               顶部 path + size 是冗余信息(用户从左边文件树知道当前选的是哪个)。
               只保留右侧的操作按钮(铅笔/眼睛 + dirty + 放弃/保存)。 -->
          <span class="sfip-viewer-spacer" />
          <button
            v-if="selectedFile?.path && currentMode === 'view'"
            class="sfip-mode-btn"
            :data-tip="t(LABEL_EDIT)"
            :aria-label="t(LABEL_EDIT)"
            @click="setMode(props.skill?.name, selectedFile.path, 'edit')"
          >
            <IconPark icon="Edit" width="14" height="14" />
          </button>
          <!-- 2026-07-13 改 v3:大纲 + AI 两个按钮并存,各自独立控制右侧面板。
               两个按钮始终可见(各自 v-if 条件),点击哪个就显示哪个面板,
               再点同一个按钮会隐藏(同 toggle 语义)。不再三态互玩"消失"。
               - 大纲按钮:仅对 md 文件显示(原始 outline 行为,view 模式)
               - AI 按钮:所有文件 + view 模式(用户决策) -->
          <!-- 大纲按钮:恢复原 ListView 图标,显示条件:md + view(没有标题时点开会进空态,
               用户至少能看到"该文件无标题大纲"的反馈) -->
          <button
            v-if="selectedFile?.path && currentMode === 'view' && isMarkdownFile"
            class="sfip-mode-btn"
            :data-tip="outlineActive ? t(LABEL_OUTLINE_HIDE) : t(LABEL_OUTLINE_SHOW)"
            :aria-label="outlineActive ? t(LABEL_OUTLINE_HIDE) : t(LABEL_OUTLINE_SHOW)"
            :class="{ 'sfip-mode-btn-active': outlineActive }"
            @click="onClickOutline"
          >
            <IconPark icon="ListView" width="14" height="14" />
          </button>
          <!-- AI 按钮:view 模式下始终可见 -->
          <button
            v-if="selectedFile?.path && currentMode === 'view'"
            class="sfip-mode-btn"
            :data-tip="aiActive ? LABEL_AI_CLOSE : LABEL_AI_OPEN"
            :aria-label="aiActive ? LABEL_AI_CLOSE : LABEL_AI_OPEN"
            :class="{ 'sfip-mode-btn-active': aiActive }"
            @click="onClickAi"
          >
            <IconPark :icon="aiActive ? 'mdi:robot' : 'mdi:robot-outline'" width="14" height="14" />
          </button>
          <!-- 2026-07-08 改:删掉"返回预览"按钮(原 mode=edit 分支)。
               用户决定编辑后只能一直编辑,通过"放弃修改"或"保存"按钮离开编辑态。
               避免中间态视觉混乱 — 编辑完直接保存,不要再预览一遍。 -->
          <!-- 2026-07-08 改 v2:两个按钮显示策略分开 —
               "放弃修改" 始终在编辑态下显示(currentMode === 'edit' || isDirty),
                用户没改东西也能放弃(回到 view 模式,等于"取消编辑")。
               "保存" 只在 isDirty 时显示,避免空保存(没改任何东西调 saveCurrent
                是浪费一次 HTTP)。同时 dirty 标签 ● 未保存 也只在 isDirty 时显示。 -->
          <span v-if="isDirty" class="sfip-viewer-dirty">{{ t(LABEL_DIRTY) }}</span>
          <button
            v-if="currentMode === 'edit' || isDirty"
            class="sfip-btn"
            :disabled="saving"
            :data-tip="t(LABEL_DISCARD)"
            :aria-label="t(LABEL_DISCARD)"
            @click="resetCurrent"
          >{{ t(LABEL_DISCARD) }}</button>
          <button
            v-if="isDirty"
            class="sfip-btn sfip-btn-primary"
            :disabled="saving"
            :data-tip="saving ? t(LABEL_SAVING) : t(LABEL_SAVE)"
            :aria-label="t(LABEL_SAVE)"
            @click="saveCurrent"
          >
            <span v-if="saving" class="sfip-spinner"></span>
            <IconPark v-else icon="Save" width="13" height="13" />
            {{ saving ? t(LABEL_SAVING) : t(LABEL_SAVE) }}
          </button>
        </header>
        <CodeViewer
          v-if="selectedFile?.path"
          :key="selectedFile.path"
          :path="selectedFile.path"
          :content="displayContent"
          :mode="currentMode"
          :store-root="storeRoot"
          :skill-rel-path="skillRelPath"
          :source-path="sourcePath"
          :right-panel-mode="rightMode"
          :apply-flash="applyFlash"
          @update:content="onContentChange"
          @dirty-change="onDirtyChange"
          @update:right-panel-mode="onSetRightPanelMode"
          @ai-apply-skill="onAiApplySkill"
          @ai-apply-local="onAiApplyLocal"
        />
        <div v-else class="sfip-empty">
          <IconPark icon="mdi:file-outline" width="48" height="48" />
          <p>{{ t(LABEL_PICK) }}</p>
        </div>
      </main>
    </div>

    <!-- 2026-07-07 修:Modal 必须用 v-model 绑 modelValue(组件内部 watch modelValue 控制渲染),
         旧版用 v-if="fmOpen" + @close="closeFrontmatter" 看似能调,但组件内部 <div v-if="modelValue">
         modelValue 始终是 undefined,所以 mask 永远不渲染 → 弹窗不出现。
         2026-07-10 改:Info 按钮(原"班级按钮")只读弹窗 — 展示 frontmatter 内容,
         不允许编辑。编辑动作由顶栏铅笔按钮触发另一个 editFmOpen 表单弹窗。 -->
    <Modal
      v-model="fmOpen"
      size="md"
      :title="(skill?.name || '') + ' · frontmatter'"
      @close="closeFrontmatter"
    >
      <div class="sfip-fm-body">
        <table v-if="frontmatterEntries.length" class="sfip-fm-table">
          <tbody>
            <tr v-for="[k, v] in frontmatterEntries" :key="k">
              <th>{{ k }}</th>
              <td>
                <template v-if="Array.isArray(v)">
                  <span v-for="(x, i) in v" :key="i" class="sfip-fm-chip">{{ x }}</span>
                </template>
                <template v-else>{{ v }}</template>
              </td>
            </tr>
          </tbody>
        </table>
        <p v-else class="sfip-fm-empty">{{ t('skills.fileBrowser.noFrontmatter') }}</p>
      </div>
      <template #footer>
        <button class="primary" @click="closeFrontmatter">{{ t('common.close') }}</button>
      </template>
    </Modal>

    <!-- 2026-07-10 增:frontmatter 编辑表单弹窗(独立 Modal,跟只读 fmOpen 互斥)。
         由 InlinePanel 顶栏 #name-actions 槽里的"编辑"铅笔按钮触发。
         字段:name/version/description/author/license + triggers 动态列表,
         保存走 saveFrontmatterForm → updateSkill 链路。 -->
    <Modal
      v-model="editFmOpen"
      size="md"
      :title="(newSkillInitial ? t('skills.editor.frontmatterDialogTitle') : t('skills.editor.frontmatterDialogTitle'))"
      :close-on-mask="!fmFormSaving"
      @close="closeFrontmatterEditor"
    >
      <div class="sfip-fm-body">
        <div class="sfip-fm-form">
          <div class="sfip-fm-row">
            <label class="sfip-fm-label">name</label>
            <input
              v-model="fmForm.name"
              class="sfip-fm-input"
              :disabled="fmFormSaving"
              placeholder="review-pr"
              spellcheck="false"
            />
          </div>
          <div class="sfip-fm-row">
            <label class="sfip-fm-label">version</label>
            <input
              v-model="fmForm.version"
              class="sfip-fm-input"
              :disabled="fmFormSaving"
              placeholder="0.1.0"
              spellcheck="false"
            />
          </div>
          <div class="sfip-fm-row">
            <label class="sfip-fm-label">description</label>
            <textarea
              v-model="fmForm.description"
              class="sfip-fm-textarea"
              :disabled="fmFormSaving"
              rows="2"
              spellcheck="false"
              :placeholder="t('skills.editor.descriptionRequiredPlaceholder')"
            />
          </div>
          <div class="sfip-fm-row sfip-fm-row-author">
            <label class="sfip-fm-label">author</label>
            <input
              v-model="fmForm.author"
              class="sfip-fm-input"
              :disabled="fmFormSaving"
              :placeholder="t('common.optional')"
              spellcheck="false"
            />
          </div>
          <div class="sfip-fm-row sfip-fm-row-license">
            <label class="sfip-fm-label">license</label>
            <input
              v-model="fmForm.license"
              class="sfip-fm-input"
              :disabled="fmFormSaving"
              :placeholder="t('skills.editor.licensePlaceholder')"
              spellcheck="false"
            />
          </div>
          <div class="sfip-fm-row sfip-fm-row-triggers">
            <label class="sfip-fm-label">
              triggers
              <span class="sfip-fm-label-hint">{{ t('skills.editor.triggersList') }}</span>
              <!-- 2026-07-12 改:触发词改为可选,label 加 "(可选)" 角标 -->
              <span class="sfip-fm-label-badge">{{ LABEL_TRIGGERS_OPTIONAL }}</span>
            </label>
            <div class="sfip-fm-triggers-list">
              <!-- 2026-07-12 增:触发词为可选,空数组时显示一行虚线占位文案 -->
              <div v-if="!fmForm.triggers.length" class="sfip-fm-trigger-empty">
                <IconPark icon="mdi:lightbulb-on-outline" width="14" height="14" />
                <span>{{ t(LABEL_TRIGGERS_EMPTY_HINT) }}</span>
              </div>
              <div
                v-for="(_, idx) in fmForm.triggers"
                :key="`trg-${idx}`"
                class="sfip-fm-trigger-row"
              >
                <input
                  v-model="fmForm.triggers[idx]"
                  class="sfip-fm-input sfip-fm-trigger-input"
                  :disabled="fmFormSaving"
                  :placeholder="t('skills.editor.triggerPlaceholder', { idx: idx + 1 })"
                  spellcheck="false"
                />
                <button
                  type="button"
                  class="sfip-fm-trigger-del"
                  :disabled="fmFormSaving"
                  :title="t('skills.editor.deleteTrigger', { idx: idx + 1 })"
                  :aria-label="t('skills.editor.deleteTrigger', { idx: idx + 1 })"
                  @click="removeTrigger(idx)"
                >
                  <IconPark icon="mdi:close" width="13" height="13" />
                </button>
              </div>
              <button
                type="button"
                class="sfip-fm-trigger-add"
                :disabled="fmFormSaving"
                @click="addTrigger"
              >
                <IconPark icon="mdi:plus" width="13" height="13" />
                {{ t('skills.editor.addTrigger') }}
              </button>
            </div>
          </div>
        </div>
        <p v-if="fmFormError" class="message message-error sfip-fm-err">
          <IconPark icon="mdi:alert-circle-outline" width="12" height="12" />
          {{ fmFormError }}
        </p>
      </div>
      <template #footer>
        <button type="button" class="ghost" :disabled="fmFormSaving" @click="closeFrontmatterEditor">
          <IconPark icon="mdi:close" width="13" height="13" />
          {{ t('common.cancel') }}
        </button>
        <button
          type="button"
          class="primary"
          :disabled="fmFormSaving"
          @click="saveFrontmatterForm"
        >
          <span v-if="fmFormSaving" class="sfip-spinner"></span>
          <IconPark v-else icon="mdi:content-save" width="13" height="13" />
          {{ fmFormSaving ? t('skills.editor.saving') : t('common.save') }}
        </button>
      </template>
    </Modal>

    <!-- 2026-07-07 增:切换前的 dirty 询问弹窗(三选项:保存/放弃/取消)。
         ensureCleanBeforeSwitch() 返回的 Promise 由 _discardResolve 接住,
         等用户点击对应按钮再 resolve。 -->
    <Modal
      v-model="discardOpen"
      size="sm"
      :title="t('skills.fileBrowser.modifiedTitleDirty')"
      :close-on-mask="false"
    >
      <p class="sfip-discard-msg">
        文件 <code>{{ discardFileName }}</code> {{ t('skills.fileBrowser.discardPrompt') }}
      </p>
      <ul class="sfip-discard-tips">
        <li><strong>{{ t('common.save') }}</strong>:{{ t('skills.fileBrowser.discardSaveHint') }}</li>
        <li><strong>{{ t('skills.fileBrowser.discardChanges') }}</strong>:{{ t('skills.fileBrowser.discardDropHint') }}</li>
        <li><strong>{{ t('common.cancel') }}</strong>:{{ t('skills.fileBrowser.discardCancelHint') }}</li>
      </ul>
      <template #footer>
        <button type="button" class="ghost" @click="onDiscardCancel">{{ t('common.cancel') }}</button>
        <button type="button" class="danger" @click="onDiscardDrop">{{ t('skills.fileBrowser.discardChanges') }}</button>
        <button type="button" class="primary" @click="onDiscardSave">{{ t('skills.fileBrowser.saveChanges') }}</button>
      </template>
    </Modal>

    <!-- 2026-07-11 增:文件树右键菜单(单例) + 4 个 Modal 弹窗。
         复用 SkillsView 同一份 ContextMenu 组件,Modal 也用同一份。
         新建/重命名/删除文件/目录都走 updateSkill(payload.files) 链路,
         由 persistFiles() 统一处理(写盘 + 同步 localFiles + emit('saved'))。 -->

    <!-- 右键菜单 -->
    <ContextMenu
      v-if="ctxMenu.open"
      :x="ctxMenu.x"
      :y="ctxMenu.y"
      :items="ctxMenu.items"
      @close="closeCtxMenu"
    />

    <!-- 新建文件 / 新建目录 弹窗 -->
    <Modal v-model="newFileOpen" size="sm" :close-on-mask="!newFileBusy">
      <template #header>
        <h3 class="modal-title">
          <IconPark
            :icon="newFileKind === 'dir' ? 'mdi:folder-plus-outline' : 'mdi:file-document-plus-outline'"
            width="18" height="18"
          />
          {{ newFileKind === 'dir' ? t(LABEL_CTX_NEW_DIR) : t(LABEL_CTX_NEW_FILE) }}
        </h3>
      </template>
      <div class="editor-field-full">
        <p class="muted small-hint">
          {{ newFileKind === 'dir' ? t(LABEL_NEW_DIR_PROMPT) : t(LABEL_NEW_FILE_PROMPT) }}
        </p>
        <input
          v-model="newFileInput"
          class="group-input"
          :placeholder="newFileKind === 'dir' ? t('skills.fileBrowser.newFolderPlaceholder') : t('skills.fileBrowser.newFilePlaceholder')"
          :disabled="newFileBusy"
          autofocus
          @keyup.enter="submitNewFile"
        />
        <p v-if="newFileDirPath" class="muted small-hint">
          <code>{{ newFileDirPath }}/<span style="color: var(--text)">{{ newFileInput || '...' }}</span></code>
        </p>
        <p v-else class="muted small-hint">
          <code>/<span style="color: var(--text)">{{ newFileInput || '...' }}</span></code>
        </p>
        <p v-if="newFileError" class="message message-error" style="margin: 8px 0 0">
          <IconPark icon="mdi:alert-circle-outline" width="12" height="12" />
          {{ newFileError }}
        </p>
      </div>
      <template #footer>
        <button type="button" class="ghost" :disabled="newFileBusy" @click="closeNewFileDialog">{{ t('common.cancel') }}</button>
        <button
          type="button"
          class="primary"
          :disabled="newFileBusy || !newFileInput.trim()"
          @click="submitNewFile"
        >
          <span v-if="newFileBusy" class="spinner spinner-sm"></span>
          <IconPark v-else icon="mdi:check" width="14" height="14" />
          {{ t('common.confirm') }}
        </button>
      </template>
    </Modal>

    <!-- 重命名文件 弹窗 -->
    <Modal v-model="renameFileOpen" size="sm" :close-on-mask="!renameFileBusy">
      <template #header>
        <h3 class="modal-title">
          <IconPark icon="mdi:rename-outline" width="18" height="18" />
          {{ t(LABEL_RENAME_FILE_PROMPT) }}
        </h3>
      </template>
      <div class="editor-field-full">
        <input
          v-model="renameFileInput"
          class="group-input"
          :placeholder="renameFileOldName"
          :disabled="renameFileBusy"
          autofocus
          @keyup.enter="submitRenameFile"
        />
        <p v-if="renameFileOldPath.includes('/')" class="muted small-hint">
          <code>{{ renameFileOldPath.slice(0, renameFileOldPath.lastIndexOf('/')) }}/<span style="color: var(--text)">{{ renameFileInput || '...' }}</span></code>
        </p>
        <p v-else class="muted small-hint">
          <code>/<span style="color: var(--text)">{{ renameFileInput || '...' }}</span></code>
        </p>
        <p v-if="renameFileError" class="message message-error" style="margin: 8px 0 0">
          <IconPark icon="mdi:alert-circle-outline" width="12" height="12" />
          {{ renameFileError }}
        </p>
      </div>
      <template #footer>
        <button type="button" class="ghost" :disabled="renameFileBusy" @click="closeRenameFileDialog">{{ t('common.cancel') }}</button>
        <button
          type="button"
          class="primary"
          :disabled="renameFileBusy || !renameFileInput.trim() || renameFileInput.trim() === renameFileOldName"
          @click="submitRenameFile"
        >
          <span v-if="renameFileBusy" class="spinner spinner-sm"></span>
          <IconPark v-else icon="mdi:check" width="14" height="14" />
          {{ t('common.save') }}
        </button>
      </template>
    </Modal>

    <!-- 2026-07-11 增:重命名文件夹 弹窗(跟重命名文件同款样式) -->
    <Modal v-model="renameFolderOpen" size="sm" :close-on-mask="!renameFolderBusy">
      <template #header>
        <h3 class="modal-title">
          <IconPark icon="mdi:rename-outline" width="18" height="18" />
          {{ t(LABEL_RENAME_FOLDER_PROMPT) }}
        </h3>
      </template>
      <div class="editor-field-full">
        <input
          v-model="renameFolderInput"
          class="group-input"
          :placeholder="renameFolderOldName"
          :disabled="renameFolderBusy"
          autofocus
          @keyup.enter="submitRenameFolder"
        />
        <p v-if="renameFolderOldPath.includes('/')" class="muted small-hint">
          <code>{{ renameFolderOldPath.slice(0, renameFolderOldPath.lastIndexOf('/')) }}/<span style="color: var(--text)">{{ renameFolderInput || '...' }}</span></code>
        </p>
        <p v-else class="muted small-hint">
          <code>/<span style="color: var(--text)">{{ renameFolderInput || '...' }}</span></code>
        </p>
        <p v-if="renameFolderError" class="message message-error" style="margin: 8px 0 0">
          <IconPark icon="mdi:alert-circle-outline" width="12" height="12" />
          {{ renameFolderError }}
        </p>
      </div>
      <template #footer>
        <button type="button" class="ghost" :disabled="renameFolderBusy" @click="closeRenameFolderDialog">{{ t('common.cancel') }}</button>
        <button
          type="button"
          class="primary"
          :disabled="renameFolderBusy || !renameFolderInput.trim() || renameFolderInput.trim() === renameFolderOldName"
          @click="submitRenameFolder"
        >
          <span v-if="renameFolderBusy" class="spinner spinner-sm"></span>
          <IconPark v-else icon="mdi:check" width="14" height="14" />
          {{ t('common.save') }}
        </button>
      </template>
    </Modal>

    <!-- 删除文件 / 目录 确认弹窗(复用) -->
    <Modal v-model="deleteFileOpen" size="sm" :close-on-mask="!deleteFileBusy">
      <template #header>
        <h3 class="modal-title">
          <IconPark
            :icon="deleteFileTarget?.kind === 'dir' ? 'mdi:folder-remove-outline' : 'mdi:delete'"
            width="18" height="18"
          />
          {{ t(LABEL_DELETE_FILE_PROMPT) }}
        </h3>
      </template>
      <p class="confirm-message">
        <template v-if="deleteFileTarget?.kind === 'dir'">
          {{ t('skills.fileBrowser.deleteFolderConfirm', { name: deleteFileTarget.name }) }}
          <template v-if="deleteFileTarget.childCount > 0">
            <br /><br />
            {{ t('skills.fileBrowser.deleteFolderChildrenWarning', { n: deleteFileTarget.childCount }) }}
          </template>
        </template>
        <template v-else>
          {{ t(LABEL_DELETE_FILE_CONFIRM, { name: deleteFileTarget?.name || '' }) }}
        </template>
      </p>
      <template #footer>
        <button type="button" class="ghost" :disabled="deleteFileBusy" @click="closeDeleteFileDialog">{{ t('common.cancel') }}</button>
        <button type="button" class="danger" :disabled="deleteFileBusy" @click="submitDeleteFile">
          <span v-if="deleteFileBusy" class="spinner spinner-sm"></span>
          <IconPark v-else icon="mdi:delete" width="14" height="14" />
          {{ t('common.delete') }}
        </button>
      </template>
    </Modal>
  </div>
</template>

<style scoped>
.sfip {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  background: var(--bg-card);
}
.sfip-error {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  height: 100%;
  padding: 40px 24px;
  background: var(--bg-card);
  color: var(--text-dim);
  text-align: center;
}
.sfip-error h4 { margin: 0; font-size: 15px; color: var(--text); }
.sfip-error-msg {
  margin: 0;
  max-width: 480px;
  font-size: 12px;
  color: var(--danger);
  font-family: ui-monospace, SFMono-Regular, monospace;
  word-break: break-all;
}
.sfip-header {
  display: flex;
  align-items: center;
  padding: 10px 16px;
  border-bottom: 1px solid var(--border);
  background: var(--bg-card);
  flex-shrink: 0;
  gap: 6px;
}
.sfip-title-block {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: var(--text-dim);
}
.sfip-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
}
.sfip-version {
  color: var(--text-faint);
  font-weight: 400;
  margin-left: 2px;
}
/* 2026-07-12 增:名称 + 简介的竖向堆叠容器。display:flex column 让简介
   紧贴名称行下方。min-width:0 防止 name-row 的 count 把 stack 撑到右侧。 */
.sfip-title-stack {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  min-width: 0;
  gap: 2px;
}
/* 2026-07-12 改:名称行现在容纳 name + version + count + source badge +
   name-actions 编辑按钮,横向 flex + baseline 对齐;name-actions
   用 margin-left:auto 把自己推到 stack 内最右(后续 .sfip-actions 仍
   整体 margin-left:auto 推到 .sfip-header 最右)。 */
.sfip-name-row {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  flex-wrap: wrap;
}
.sfip-name-actions {
  display: inline-flex;
  gap: 6px;
  margin-left: auto;
}
/* 2026-07-12 增:技能简介小字,放在 .sfip-name 正下方一行。
   灰色(--text-faint)+ 12px,-webkit-line-clamp:1 单行截断
   (避免超长换行撑爆顶栏,过长交给 hover 看全)。
   cursor:text(普通文本光标)避免某些平台 (macOS webkit) 渲染 help
   问号图标 — 用户反馈"图标可能不存在",直接用文本光标最稳。
   position:relative 让自定义 .sfip-desc-tip 浮层能基于自己定位。 */
.sfip-desc {
  display: -webkit-box;
  -webkit-line-clamp: 1;
  -webkit-box-orient: vertical;
  overflow: hidden;
  word-break: break-word;
  font-size: 12px;
  line-height: 1.4;
  color: var(--text-faint);
  max-width: 100%;
  cursor: text;
  position: relative;
}
/* 2026-07-12 改:tip 浮层黑底白字样式 — 用户期望 tooltip 是深色卡片,
   比浅底 + 边框更显眼易读(系统级 native title 也是黑底白字,统一风格)。
   浮层背景直接写 #1f2937 (slate-800),文字 #f3f4f6 (gray-100),
   边框用半透明白色提亮阴影。 */
.sfip-desc-tip {
  position: fixed;
  display: block;
  padding: 8px 12px;
  font-size: 12px;
  line-height: 1.5;
  color: #f3f4f6;
  background: #1f2937;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 6px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.45);
  white-space: normal;
  word-break: break-word;
  cursor: text;
  z-index: 100;
  max-height: 60vh;
  overflow-y: auto;
}
.sfip-count {
  color: var(--text-faint);
  font-size: 11px;
  padding: 1px 6px;
  background: var(--bg-subtle);
  border-radius: 999px;
}
.sfip-name-actions { display: inline-flex; gap: 6px; margin-left: auto; }
.sfip-actions {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin-left: auto;
  flex-shrink: 0;
}
.sfip-actions :deep(.icon-btn) {
  background: transparent;
  border: 1px solid var(--border);
  color: var(--text-faint);
  width: 28px;
  height: 26px;
  flex: 0 0 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  cursor: pointer;
  padding: 0;
  overflow: hidden;
  transition: background 120ms ease, color 120ms ease, border-color 120ms ease;
}
.sfip-actions :deep(.icon-btn) > * { flex: 0 0 auto; }
.sfip-actions :deep(.icon-btn:hover:not(:disabled)) {
  background: var(--bg-hover);
  color: var(--accent-blue);
  border-color: var(--accent-blue);
}
.sfip-fm-btn {
  background: transparent;
  border: 1px solid var(--border);
  color: var(--text-faint);
  width: 28px;
  height: 26px;
  flex: 0 0 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  cursor: pointer;
  padding: 0;
  box-sizing: border-box;
  overflow: hidden;
  transition: background 120ms ease, color 120ms ease, border-color 120ms ease;
}
.sfip-fm-btn:hover {
  background: var(--bg-hover);
  color: var(--accent-blue);
  border-color: var(--accent-blue);
}
.sfip-body {
  display: flex;
  flex: 1;
  min-height: 0;
  position: relative;
  /* 修横向滚动条根因:显式收紧,避免子级撑出横向滚动条 */
  overflow: hidden;
  min-width: 0;
}
.sfip-left {
  width: var(--sfip-left-w, 280px);
  flex-shrink: 0;
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: var(--bg);
  min-width: 0;
}

/* 拖拽把手:绝对定位到目录树右边界(不占 flex 宽度,避免撑出横向溢出)。
   命中 8px,视觉细线 hover/拖拽时显蓝色。 */
.sfip-resizer {
  position: absolute;
  top: 0;
  bottom: 0;
  left: var(--sfip-left-w, 280px);
  width: 8px;
  transform: translateX(-4px);
  cursor: col-resize;
  background: transparent;
  z-index: 3;
  user-select: none;
}
.sfip-resizer::after {
  content: '';
  position: absolute;
  top: 0;
  bottom: 0;
  left: 3px;
  width: 2px;
  background: transparent;
  transition: background 120ms ease;
}
.sfip-resizer:hover::after,
.sfip-resizer-dragging::after {
  background: var(--accent-blue);
}
.sfip-tree-wrap {
  flex: 1;
  min-height: 0;
  overflow: auto;
}

/* 2026-07-07 增:文件树标题,跟作用域 .ssp-scope-header 风格一致 */
.sfip-tree-header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  font-size: 12px;
  font-weight: 600;
  color: var(--text-dim);
  letter-spacing: 0.04em;
  text-transform: uppercase;
  border-bottom: 1px solid var(--border);
  background: var(--bg-subtle);
  position: sticky;
  top: 0;
  z-index: 1;
}
.sfip-tree-header-count {
  margin-left: auto;
  font-size: 10px;
  font-weight: 500;
  letter-spacing: 0;
  text-transform: none;
  color: var(--text-faint);
  padding: 1px 6px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: 999px;
}

/* 2026-07-07 改 v3:作用域区移到文件树底部,文件树占满主要高度,
   作用域用 flex-shrink:0 固定在底部一块,ScopePanel 内部自己 max-height:50% 兜底。 */
.sfip-viewer {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.sfip-viewer-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 16px;
  border-bottom: 1px solid var(--border);
  background: var(--bg-subtle);
  font-size: 12px;
  color: var(--text-dim);
  flex-shrink: 0;
  /* 兜底:工具栏按钮较多时不撑破容器,避免详情底部出现横向滚动条 */
  min-width: 0;
  overflow: hidden;
}
.sfip-viewer-path {
  font-family: ui-monospace, SFMono-Regular, monospace;
  color: var(--text);
  flex: 1;
}
.sfip-viewer-size { color: var(--text-faint); }
/* 2026-07-08 增:替代 .sfip-viewer-path 的弹性占位,让右侧操作按钮右对齐 */
.sfip-viewer-spacer { flex: 1; }
.sfip-viewer-dirty {
  color: var(--accent-amber, #b8860b);
  font-size: 11px;
  margin-right: 4px;
}
.sfip-mode-btn,
.sfip-btn {
  background: transparent;
  border: 1px solid var(--border);
  color: var(--text-dim);
  padding: 3px 10px;
  font-size: 12px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.sfip-mode-btn:hover,
.sfip-btn:hover:not(:disabled) {
  background: var(--bg-hover);
  color: var(--text);
  border-color: var(--text-faint);
}
.sfip-mode-btn-active,
.sfip-btn-primary {
  background: var(--accent-blue-bg);
  color: var(--accent-blue);
  border-color: var(--accent-blue-border);
}
.sfip-empty {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: var(--text-faint);
  font-size: 13px;
}
.sfip-spinner {
  display: inline-block;
  width: 12px;
  height: 12px;
  border: 2px solid var(--border);
  border-top-color: var(--accent-blue);
  border-radius: 50%;
  animation: sfip-spin 0.8s linear infinite;
}
@keyframes sfip-spin {
  to { transform: rotate(360deg); }
}
.sfip-fm-body {
  font-size: 13px;
  color: var(--text);
}

/* 2026-07-10 改:frontmatter 弹窗表单模式 — 原只读表格替换为可编辑输入框。
   name/version/description/author/license 为单行/多行输入,triggers 为列表
   动态增删(每行一个 input + 删除按钮)。 */
.sfip-fm-form {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.sfip-fm-row {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.sfip-fm-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-dim);
  font-family: ui-monospace, SFMono-Regular, monospace;
  display: flex;
  align-items: center;
  gap: 8px;
}
.sfip-fm-label-hint {
  font-size: 11px;
  font-weight: 400;
  color: var(--text-faint);
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
}
/* 2026-07-12 增:触发词可选角标(仿照 description "必填" 红色) */
.sfip-fm-label-badge {
  font-size: 10px;
  font-weight: 500;
  padding: 1px 6px;
  border-radius: 999px;
  background: var(--bg-subtle);
  color: var(--text-faint);
  border: 1px solid var(--border);
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
}
.sfip-fm-input,
.sfip-fm-textarea {
  font-size: 13px;
  padding: 6px 10px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg);
  color: var(--text);
  font-family: inherit;
  outline: none;
  transition: border-color 0.12s;
  box-sizing: border-box;
  width: 100%;
}
.sfip-fm-textarea {
  resize: vertical;
  min-height: 48px;
  line-height: 1.5;
}
.sfip-fm-input:hover,
.sfip-fm-textarea:hover {
  border-color: var(--text-faint);
}
.sfip-fm-input:focus,
.sfip-fm-textarea:focus {
  border-color: var(--accent-blue);
  box-shadow: 0 0 0 2px var(--accent-blue-bg);
}
.sfip-fm-input:disabled,
.sfip-fm-textarea:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
.sfip-fm-triggers-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 8px;
  border: 1px dashed var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-subtle);
}
/* 2026-07-12 增:触发词空态占位(虚线 hint + 提示文案) */
.sfip-fm-trigger-empty {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 8px;
  font-size: 12px;
  color: var(--text-faint);
  background: var(--bg);
  border: 1px dashed var(--border);
  border-radius: var(--radius-sm);
}
.sfip-fm-trigger-empty span {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
}
.sfip-fm-trigger-row {
  display: flex;
  align-items: center;
  gap: 6px;
}
.sfip-fm-trigger-input {
  flex: 1 1 auto;
  min-width: 0;
}
.sfip-fm-trigger-del {
  flex: 0 0 auto;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 28px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg);
  color: var(--text-faint);
  cursor: pointer;
  transition: all 0.12s;
  box-sizing: border-box;
}
.sfip-fm-trigger-del:hover:not(:disabled) {
  border-color: var(--accent-red, #ef4444);
  color: var(--accent-red, #ef4444);
  background: var(--bg-hover);
}
.sfip-fm-trigger-del:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.sfip-fm-trigger-add {
  align-self: flex-start;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 5px 10px;
  font-size: 12px;
  color: var(--accent-blue);
  background: transparent;
  border: 1px dashed var(--accent-blue-border, var(--border));
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all 0.12s;
}
.sfip-fm-trigger-add:hover:not(:disabled) {
  border-style: solid;
  background: var(--accent-blue-bg);
}
.sfip-fm-trigger-add:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.sfip-fm-err {
  margin-top: 12px;
}

.sfip-fm-table {
  width: 100%;
  border-collapse: collapse;
}
.sfip-fm-table th,
.sfip-fm-table td {
  padding: 6px 12px;
  border-bottom: 1px solid var(--border);
  text-align: left;
  vertical-align: top;
}
.sfip-fm-table th {
  width: 120px;
  font-weight: 600;
  color: var(--text-dim);
  font-family: ui-monospace, SFMono-Regular, monospace;
  font-size: 12px;
}
.sfip-fm-table td { color: var(--text); }
.sfip-fm-table tr:last-child th,
.sfip-fm-table tr:last-child td { border-bottom: none; }
.sfip-fm-value { white-space: pre-wrap; }
.sfip-fm-chip {
  display: inline-block;
  margin-right: 4px;
  margin-bottom: 2px;
  padding: 1px 8px;
  font-size: 11px;
  color: var(--accent-blue);
  background: var(--bg-subtle);
  border: 1px solid var(--border);
  border-radius: 999px;
  font-family: ui-monospace, SFMono-Regular, monospace;
}
.sfip-fm-empty {
  color: var(--text-faint);
  font-style: italic;
}

/* 2026-07-07 增:dirty 询问弹窗样式 */
.sfip-discard-msg {
  margin: 0 0 12px;
  font-size: 13px;
  line-height: 1.6;
  color: var(--text);
}
.sfip-discard-msg code {
  font-family: ui-monospace, SFMono-Regular, monospace;
  padding: 1px 6px;
  background: var(--bg-subtle);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  font-size: 12px;
}
.sfip-discard-tips {
  margin: 0;
  padding-left: 18px;
  font-size: 12px;
  line-height: 1.8;
  color: var(--text-dim);
}
.sfip-discard-tips strong {
  color: var(--text);
  font-weight: 600;
}
</style>
