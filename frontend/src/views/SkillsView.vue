<script setup>
// SkillsView - 技能首页(左右布局)
//
// 左侧:技能分组树(顶部"新建 / 导入"按钮 + 搜索框 + 树形列表,支持右键 + 拖拽)
// 右侧:选中 skill 的详情
//   - 顶部 toolbar:技能名 + 版本 + 源徽章;右侧操作图标(测试 / 打标签 / 在文件夹打开 / 删除,hover 显示文字)
//   - scope chips:多选,默认"全局"必选;其他取自 listProjects
//   - 标签列表(横向 chips)
//   - 下方渲染 SKILL.md 的 body(markdown 简单自渲染)
//
// 2026-06-29 改:左侧从扁平列表升级为多级分组树,新增右键菜单 + 拖拽 + 级联删除。
// 详情区(右侧 / 弹窗 / 编辑器)逻辑保持不变,只从"通过 name 定位"改为"通过 path 定位"。

import { ref, reactive, computed, onMounted, onUnmounted, onUpdated, nextTick, inject } from 'vue'
import { useI18n } from 'vue-i18n'
import IconPark from '@/components/IconPark.vue'
import { listSkills, getSkill, createSkill, updateSkill, deleteSkill, forceUndoApply, createGroup as apiCreateGroup, deleteGroup as apiDeleteGroup } from '@/api/skillbox/skills'
import { listProjects } from '@/api/skillbox/projects'
import { runSkillTest } from '@/api/skillbox/skill_test'
import { createTag, listTags, deleteTag, diffTag, rollbackTag } from '@/api/skillbox/tags'
// 2026-07-03 增:apply / batch 响应的统一判定工具,把 Service.Apply 宽容路径
// (逐 tool 失败不阻断但仍返 200)下的部分失败显式标出,前端弹 partial_failed toast。
import { inspectApplyResult, formatFailedDetail } from '@/api/skillbox/apply_result.js'
import AIPanel from '@/components/AIPanel.vue'
import Modal from '@/components/Modal.vue'
import RichTextEditor from '@/components/RichTextEditor.vue'
import ContextMenu from '@/components/ContextMenu.vue'
import TreeNode from '@/components/TreeNode.vue'
// 2026-07-04 增:首页技能文件浏览器(抽屉)。Commit 1 只做"目录树 + 纯文本预览",
// 后续 commit 加 Monaco / 编辑 / 保存。
import SkillFileDrawer from '@/components/skill/SkillFileDrawer.vue'
// 2026-07-04 改:抽屉改内联面板,直接放在正文右侧。
import SkillFileInlinePanel from '@/components/skill/SkillFileInlinePanel.vue'
// 2026-06-27 改:详情预览区改用 markdown-it + highlight.js 渲染(支持 GFM / 代码高亮)。
// 编辑态给 Tiptap 喂 HTML 那条路仍用自研 renderMarkdown,在 RichTextEditor 内部独立 import。
import { renderMarkdownView } from '@/core/utils/markdown_view.js'
// 2026-07-04 增:md-body 内 .md-external-link 点击统一走 platform.openExternal(Commit 2)。
import { handleExternalClick } from '@/core/utils/external_link.js'
import 'highlight.js/styles/github.css'
// 2026-07-08 增:GitHub README 同款排版(h1-h6 标题层级 / 列表 / 引用 / 表格 / 任务列表
// 全套标准化样式)。9KB,CodeViewer 在 md-body 上加 markdown-body 类启用。
// 站点主题色 / 行内 code / 代码块深底仍由下面 50 条 .md-body :deep() 自定义覆盖。
import 'github-markdown-css/github-markdown.css'
import { platform } from '@/platform'
import OnboardingImportDialog from '@/components/OnboardingImportDialog.vue'
import { useToastStore } from '@/core/store/toast'
// 2026-06-29 增:skill 树形 store — 集中管理 tree / 选中 / 折叠 / drop 目标
import { useSkillTreeStore } from '@/core/store/skill-tree'
// 2026-07-03 增:tools store — 给 TreeNode 注入工具元数据,
  // 让首页 skill 树 chip 前的图标用真 logo (icon_file 优先 + mdi 兜底),
  // 而不是硬编码的 mdi 字符串。
import { useToolsStore } from '@/core/store/tools'

const { t } = useI18n()

// 2026-06-29 增:树形 store — 左侧 UI 的真相源(tree / 选中 / 折叠 / drop 目标)。
// 详情区 / 编辑器等仍用扁平 items(从 store.flatItems 派生),保持兼容性。
const skillTree = useSkillTreeStore()

// ====== 列表 + 选中态 ======
const keyword = ref('')
const loading = ref(false)
const error = ref('')
// items 现在从 store.flatItems 派生(扁平列表,供详情区等使用)
const items = computed(() => skillTree.flatItems || [])
const total = ref(0)
const page = ref(1)
const size = 200
const selectedKey = ref(null) // 选中项的 key 字符串(= skill path)

// 当前选中的 skill 详情
const current = ref(null)         // 完整 skill 详情(loadSkill 后填充)
const currentMd = ref('')         // 原始 SKILL.md 全文
const currentBody = ref('')       // extractBody 后的正文
const currentMeta = reactive({ description: '', triggers: [] })
const currentTagList = ref([])    // 当前 skill 的 tag 列表
const currentLoading = ref(false)
const currentError = ref('')

// 内联编辑(2026-06-25 三改:同时编辑 description + 触发词 + 正文)
const editing = ref(false)            // 是否处于内联编辑态
const editBody = ref('')              // 编辑器内的 body 文本
const editDescription = ref('')       // 编辑器内的 description 文本
const editTriggersText = ref('')      // 编辑器内的触发词(逗号分隔,2026-06-26 改:默认逗号)
const editError = ref('')             // 校验错误
const editSaving = ref(false)         // 保存中

function startInlineEdit() {
  if (!current.value) return
  editBody.value = currentBody.value || ''
  editDescription.value = currentMeta.description || ''
  // 触发词编辑态:把数组转成"逗号分隔"的纯文本,用户改完再 split 回去
  // 2026-06-26 改:默认用逗号作为分隔符(换行作为兜底也支持)
  editTriggersText.value = (currentMeta.triggers || []).join(', ')
  editError.value = ''
  editing.value = true
}
function cancelInlineEdit() {
  editing.value = false
  editBody.value = ''
  editDescription.value = ''
  editTriggersText.value = ''
  editError.value = ''
}
async function saveInlineEdit() {
  if (!current.value) return
  editError.value = ''
  // 触发词:从文本 split 成数组,过滤空字符串
  // 2026-06-26 改:默认以逗号分隔(换行也支持作为兜底,用户复制粘贴多行也能用)
  const newTriggers = (editTriggersText.value || '')
    .split(/[,\n]/)
    .map((s) => s.trim())
    .filter(Boolean)
  const newDescription = (editDescription.value || '').trim()
  editSaving.value = true
  try {
    // 2026-07-07 改:scope 区从 SkillsView 搬到 SkillFileInlinePanel 后,
    // 这里的 apply 回放逻辑删除 — InlinePanel 自管 scope,会监听 skillbox:scope-refresh
    // 事件拉取最新状态。SKILL.md 写盘到 home store 后,下次 enabled tool 重新
    // 加载时(symlink 模式)自然拿到新内容;copy 模式需用户在 InlinePanel 手动
    // 重启用工具(预期行为,避免自动回放踩到 forceUndoApply 等副作用)。
    const targetSkill = { ...current.value }
    // 先同步到 currentMeta(用户视角的"立刻反馈")
    currentMeta.description = newDescription
    currentMeta.triggers = newTriggers
    // 重新拼 SKILL.md(保留 frontmatter,替换 description/triggers/body)
    const newMd = rebuildSkillMd(editBody.value, newTriggers, newDescription)
    await updateSkill({
      scope: targetSkill.scope,
      project_id: targetSkill.project_id,
      name: targetSkill.name,
      version: targetSkill.version,
      source: targetSkill.source || 'local',
      manifest: {
        name: targetSkill.name,
        version: targetSkill.version,
        description: newDescription,
        triggers: newTriggers,
      },
      files: [{ path: 'SKILL.md', content: newMd }],
    })
    currentMd.value = newMd
    currentBody.value = extractBody(newMd)
    editing.value = false
    // 2026-07-07 改:scope 区从 SkillsView 搬到 SkillFileInlinePanel 后,这里不再
    // 遍历 enabled scope 调 apply(InlinePanel 自管 scope-status,磁盘副本同步
    // 由用户在工具列表手动重启用,或下次 enabled tool 重新加载时自然拿到新内容)。
    // 触发 InlinePanel 重拉一次 scope,这样刚保存的 SKILL.md 元数据如果改了
    // tool/scope 关联也能及时反映。
    window.dispatchEvent(new CustomEvent('skillbox:scope-refresh'))
    toast.success(t('skills.editor.saveOk', { name: targetSkill.name }))
  } catch (e) {
    editError.value = e?.message || String(e)
  } finally {
    editSaving.value = false
  }
}

// 用现有 frontmatter 重新拼一份 SKILL.md
// newBody: 必填,新正文
// newTriggers: 可选,不传则保留 currentMeta.triggers
// newDescription: 可选,不传则保留 currentMeta.description
function rebuildSkillMd(newBody, newTriggers, newDescription) {
  const fm = {
    name: current.value?.name || '',
    version: current.value?.version || '',
    description: newDescription !== undefined ? newDescription : (currentMeta.description || ''),
    triggers: newTriggers !== undefined ? newTriggers : (currentMeta.triggers || []),
  }
  const yaml = Object.entries(fm)
    .map(([k, v]) => Array.isArray(v)
      ? `${k}: [${v.map((x) => JSON.stringify(x)).join(', ')}]`
      : `${k}: ${JSON.stringify(v)}`)
    .join('\n')
  return `---\n${yaml}\n---\n\n${newBody || ''}\n`
}

// 全局 toast
const toast = useToastStore()

// 2026-07-03 增:工具元数据 store,给 TreeNode 注入 logo。
// load() 在 onMounted 启动时已经由 ToolsView 跑过一次,但 SkillsView 进入时可能 store 还没填,
// 这里兜底:首次进入时调一次 load()。
const toolsStore = useToolsStore()
const toolsById = computed(() => {
  const m = {}
  for (const t of toolsStore.items || []) {
    if (t && t.tool_id) m[t.tool_id] = t
  }
  return m
})

// 2026-07-07 改:toolIcon 不再依赖 scopeTools(scope 已搬走),改用 toolsById 查
// toolsStore,缺省时退化到 puzzle outline。toolsStore 在 onMounted 时 load,
// 编辑器弹窗(APPLY_TOOLS)直接读这里取 icon 即可,不会触发额外请求。
function toolIcon(toolID) {
  const t = toolsById.value[toolID]
  return t?.icon || 'mdi:puzzle-outline'
}
// AI 侧栏
const aiOpen = ref(false)
function toggleAI() { aiOpen.value = !aiOpen.value }

// 2026-07-04 改:文件浏览器从抽屉改成内联面板,直接放正文右侧,不再需要 fileDrawerOpen。
// currentFiles 仍保留(供 SkillFileInlinePanel 用)。
const currentFiles = ref([])

// 2026-07-07 改 v2:不依赖 vue 的 watch(wails webview ESM chunk 偶发 ReferenceError: watch,
// 跟 SkillFileInlinePanel v6 修复同源)。改用 onUpdated + 手动引用比较。
let _lastFullRef = null
let _lastFilesRef2 = null
function _syncCurrentFiles() {
  const full = current.value?._full
  const files = full?.canonical?.files
  // 任一引用变化都触发同步(全量替换 currentFiles,避免深层引用比较)
  if (full !== _lastFullRef || files !== _lastFilesRef2) {
    _lastFullRef = full
    _lastFilesRef2 = files
    currentFiles.value = (files || []).map((f) => ({ ...f }))
  }
}
onUpdated(_syncCurrentFiles)
// 2026-07-07 改 v2 补充:onUpdated 在首次 patch 之前不触发,
// 首次进页面 currentFiles 应该立刻同步一次。
onMounted(_syncCurrentFiles)

// 2026-07-04 增(Commit 4):抽屉内文件保存后,主区同步。
//   - 如果改的是 SKILL.md,同步刷新 currentMd / currentBody,主区预览实时更新
//   - 重新拉一次 detail(确保 files 与磁盘一致)
//   - 派发 skillbox:scope-refresh 让 SkillFileInlinePanel 重拉 scope-status
//     (避免组件内重复调 getSkillScopeStatus:这里用事件统一驱动)
async function onDrawerSaved({ path, content }) {
  if (path === 'SKILL.md') {
    currentMd.value = content
    currentBody.value = extractBody(content)
  }
  // 重新拉一次详情(让 currentFiles 与磁盘同步)
  const row = items.value.find((x) => skillKey(x) === selectedKey.value)
  if (row) {
    try {
      await loadCurrent(row)
    } catch (_) { /* 忽略,主区渲染靠 currentMd/currentBody 即可 */ }
  }
  // 2026-07-07 改:不再在 parent 做 enabled scope 的 apply 回放 — scope 区已搬到
  // SkillFileInlinePanel,InlinePanel 内部自管 scope-status,这里只派发
  // skillbox:scope-refresh 事件通知它重拉(InlinePanel 在 onMounted 监听该事件)。
  window.dispatchEvent(new CustomEvent('skillbox:scope-refresh'))
}

// 2026-06-25 二改:skillKey 改为只取 name(后端 listSkills 不返回 scope/project_id,
// 之前用 scope|project_id|name|version 会因为 scope/project_id 都是 undefined,
// 所有 item 的 key 都一样,导致 findIndex 总是命中 idx=0,splice 时把第一行
// 列表项错误覆盖)。
// store layout 是 <StoreRoot>/<name>/SKILL.md,name 在 storeRoot 里唯一,version 只
// 是 SKILL.md frontmatter metadata,不影响列表项定位。
// 2026-06-29 改:skillKey 改用 path,避免同 name 跨分组时撞 key。
// store layout 是 <StoreRoot>/<group>/<name>/SKILL.md,path 在 store 内唯一。
function skillKey(p) {
  if (!p) return ''
  return p.path || p.name || ''
}

// 2026-07-07 改:scope 区搬到 SkillFileInlinePanel 后,apply/unapply 全部在
// InlinePanel 内部完成,这里不再需要 patch 列表项 — InlinePanel 完成 apply/undo
// 后会 dispatch 'skillbox:scope-refresh',parent 只需 reload 列表(store 内部会
// 重新同步 applied_tools)。原 patchAppliedTools 整段删除。

// AI 输入的上下文 = 当前 skill 的 body
const currentSkillMd = computed(() => currentBody.value || '')
function onAIApply(text) {
  const m = text.match(/^---\n[\s\S]*?\n---\n?([\s\S]*)$/)
  currentBody.value = m ? m[1].trim() : text.trim()
  // 同时把 frontmatter 部分也同步到 currentMeta(若 AI 给了完整 frontmatter)
  const fm = text.match(/^---\n([\s\S]*?)\n---/)
  if (fm) {
    try {
      // 极简 frontmatter 解析:description / triggers
      const block = fm[1]
      const desc = block.match(/description:\s*(.+)/)?.[1]?.replace(/^["']|["']$/g, '')
      const trg = block.match(/triggers:\s*\[([^\]]*)\]/)?.[1]
        ?.split(',').map(s => s.trim().replace(/^["']|["']$/g, '')).filter(Boolean)
      if (desc) currentMeta.description = desc
      if (trg) currentMeta.triggers = trg
    } catch (_) { /* 忽略 AI 输出非标准 frontmatter */ }
  }
}

// ====== 数据加载 ======
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / size)))

async function reload() {
  loading.value = true
  // 2026-06-29 改:reload 走 store.load,自动管 tree / flatItems / 折叠态 / 搜索展开
  loading.value = true
  error.value = ''
  try {
    await skillTree.load({ keyword: keyword.value || undefined })
    // total 从 store 的 flatItems.length 派生(兼容旧字段)
    total.value = skillTree.totalSkills
  } catch (e) {
    error.value = e?.message || String(e)
  } finally {
    loading.value = false
  }
}

async function loadProjects() {
  // 旧版用于"作用域可选项",新版 scope-status 接口自带 projects 字段;保留空函数避免调用方报错。
}

async function loadCurrent(row) {
  if (!row) return
  currentLoading.value = true
  currentError.value = ''
  // 2026-07-07 改:scope 区已搬到 SkillFileInlinePanel,parent 不再需要清旧 scope
  // 状态 — InlinePanel 通过 watch props.skill 变化自动重拉 scope-status 并
  // 重置折叠态(null = 全展开)。这里把旧版"清 selectedToolID / scopeHits /
  // scopeTools / scopeProjects / scopeError"那段删掉。
  try {
    // 2026-06-29 改:用 path 传(支持多级分组);name 是叶子短名(后端 SplitPath 兜底)
    const full = await getSkill({
      path: row.path || row.name,
      name: row.name,
      version: row.version,
      full: true,
    })
    const c = full?.canonical?.manifest || {}
    const files = full?.canonical?.files || []
    const md = files.find((f) => f.path === 'SKILL.md')?.content || ''
    currentMd.value = md
    currentBody.value = extractBody(md)
    currentMeta.description = c.description || ''
    currentMeta.triggers = c.triggers || []
    // 2026-06-29 改:在 row 上回填后端给的 path(可能规范化过) + 写回 store 选中态
    const finalPath = full?.path || row.path || row.name
    const enriched = { ...row, _full: full, path: finalPath, group_path: full?.group_path || row.group_path }
    current.value = enriched
    skillTree.setSelected(finalPath)
    // 同步拉一次 tag 列表,让详情区"标签"chip 有数据
    try {
      const out = await listTags({ scope: 'global', name: row.name })
      currentTagList.value = out?.items || []
    } catch (_) { currentTagList.value = [] }
    // 2026-07-07 改:scope-status 拉到工作由 SkillFileInlinePanel 自管
    // (通过 watch props.skill 触发 + scope-refresh 事件驱动),parent 不再调用
    // loadScopeStatus。
  } catch (e) {
    // 2026-07-05 增:识别后端 corrupted_file 错误(磁盘 SKILL.md 损坏),
    // 给用户弹清晰的"需手动修复"提示,而不是单纯的"网络/服务错误"。
    // 后端 controller 在 cskill.get_skill 返 422 + {code: 'corrupted_file', hint}
    // HttpError 实例带 data 字段,所以可以这样识别。
    const isCorrupted = e?.status === 422 && e?.data?.code === 'corrupted_file'
    if (isCorrupted) {
      const hint = e.data?.hint || '磁盘上的 SKILL.md 包含非 UTF-8 字节,可能已损坏。'
      currentError.value = hint
      toast.error(t('skills.fileBrowser.corruptedHint', { name: row.name, hint }), 8000)
    } else {
      currentError.value = e?.message || String(e)
    }
    current.value = { ...row }
    currentMd.value = ''
    currentBody.value = ''
  } finally {
    currentLoading.value = ''
  }
}

function extractBody(skillmd) {
  const m = skillmd.match(/^---\n[\s\S]*?\n---\n?([\s\S]*)$/)
  return m ? m[1].trim() : skillmd
}

// 2026-07-07 增:SkillFileInlinePanel 的 ref,父级在切换 skill/file 前调
// ensureCleanBeforeSwitch() 触发 dirty 询问弹窗(组件内部自管)。
const inlinePanelRef = ref(null)

// 选中列表项
async function selectItem(row) {
  // 切换 skill 时清掉内联编辑态,避免把旧 skill 的 editBody 带到新 skill
  if (editing.value) cancelInlineEdit()
  // 2026-07-07 增:切换前 dirty 询问 — InlinePanel 弹"保存/放弃/取消",等用户决策
  if (inlinePanelRef.value && typeof inlinePanelRef.value.ensureCleanBeforeSwitch === 'function') {
    const verdict = await inlinePanelRef.value.ensureCleanBeforeSwitch()
    if (verdict === 'cancel') return
  }
  selectedKey.value = skillKey(row)
  loadCurrent(row)
}

// 监听选中 key 变化(支持按 Enter 在搜索结果中跳转)
// 2026-07-07 改 v2:不依赖 vue 的 watch(wails webview ESM chunk 偶发 ReferenceError: watch,
// 跟 SkillFileInlinePanel v6 修复同源)。改用 onUpdated + 手动比较。
let _lastSelectedKey = null
function _syncSelectedKey() {
  const k = selectedKey.value
  if (k === _lastSelectedKey) return
  _lastSelectedKey = k
  if (!k) return
  const row = items.value.find((x) => skillKey(x) === k)
  if (row) loadCurrent(row)
}
onUpdated(_syncSelectedKey)

// ====== 搜索 / 翻页 ======
function onSearchEnter() {
  page.value = 1
  reload()
}
function gotoPage(p) {
  if (p < 1 || p > totalPages.value) return
  page.value = p
  reload()
}

// 过滤后的列表(本地关键字过滤,后端只做弱匹配;本地二次过滤可让选中更稳定)
const filteredItems = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  if (!kw) return items.value
  return items.value.filter((x) =>
    (x.name || '').toLowerCase().includes(kw) ||
    (x.version || '').toLowerCase().includes(kw))
})

// ====== 渲染后的 markdown HTML ======
const renderedHtml = computed(() => renderMarkdownView(currentBody.value))

// 2026-07-04 增:md-body 内 .md-external-link 链接统一走 platform.openExternal,
// 桌面端 webview 不会在内部打开,Web 端走 window.open(Commit 2)。
function onMdClick(e) {
  handleExternalClick(e)
}

// ====== Tag 弹窗 ======
const tagOpen = ref(false)
const tagList = ref([])
const tagLoading = ref(false)
const tagError = ref('')
const tagMessage = ref('')
const newTagName = ref('')
const newTagMessage = ref('')
const diffResult = ref(null)
const diffLeftTagID = ref(0)
const diffRightTagID = ref(0)
const rolling = ref(false)

async function openTagDialog() {
  if (!current.value) return
  tagOpen.value = true
  tagList.value = []
  diffResult.value = null
  newTagName.value = ''
  newTagMessage.value = ''
  await loadTagList()
}
async function loadTagList() {
  if (!current.value) return
  tagLoading.value = true
  tagError.value = ''
  try {
    const out = await listTags({ scope: current.value.scope, name: current.value.name })
    tagList.value = out?.items || []
    currentTagList.value = tagList.value
  } catch (e) { tagError.value = e?.message || String(e) }
  finally { tagLoading.value = false }
}
async function doCreateTag() {
  if (!current.value) { tagError.value = t('skills.tag.selectFirst'); return }
  if (!newTagName.value.trim()) { tagError.value = t('skills.tag.emptyName'); return }
  tagLoading.value = true
  tagError.value = ''
  try {
    await createTag({
      scope: current.value.scope,
      project_id: current.value.project_id,
      name: current.value.name,
      tag: newTagName.value.trim(),
      message: newTagMessage.value,
    })
    newTagName.value = ''
    newTagMessage.value = ''
    tagMessage.value = t('skills.tag.msgCreated')
    await loadTagList()
  } catch (e) { tagError.value = e?.message || String(e) }
  finally { tagLoading.value = false }
}
async function doDeleteTag(tagID) {
  const ok = await openConfirm({
    title: t('common.delete'),
    message: t('skills.tag.confirmDelete', { id: tagID }),
    variant: 'danger',
    confirmText: t('common.delete'),
  })
  if (!ok) return
  try {
    await deleteTag({ tag_id: tagID })
    tagMessage.value = t('skills.tag.msgDeleted', { id: tagID })
    await loadTagList()
  } catch (e) { tagError.value = e?.message || String(e) }
}
async function doDiff(leftID, rightID) {
  if (!current.value) { tagError.value = t('skills.tag.selectFirst'); return }
  try {
    const out = await diffTag({ scope: current.value.scope, name: current.value.name, left_tag_id: leftID || 0, right_tag_id: rightID || 0 })
    diffResult.value = out
    diffLeftTagID.value = leftID
    diffRightTagID.value = rightID
  } catch (e) { tagError.value = e?.message || String(e) }
}
async function doRollback(tagID) {
  const ok = await openConfirm({
    title: t('skills.tag.rollbackTo'),
    message: t('skills.tag.confirmRollback', { id: tagID }),
    confirmText: t('skills.tag.rollbackTo'),
    variant: 'danger',
  })
  if (!ok) return
  rolling.value = true
  tagError.value = ''
  try {
    const out = await rollbackTag({ tag_id: tagID })
    tagMessage.value = t('skills.tag.msgRolledBack', { pre: out.pre_rollback_tag, files: out.files_restored })
    diffResult.value = null
    await reload()
    const row = items.value.find((x) => skillKey(x) === selectedKey.value)
    if (row) await loadCurrent(row)
    await loadTagList()
  } catch (e) { tagError.value = e?.message || String(e) }
  finally { rolling.value = false }
}

// 标签 chip 列表(取自 currentTagList,与弹窗共用)
const currentTags = computed(() => currentTagList.value || [])

// ====== 测试弹窗 ======
const testOpen = ref(false)
const testing = ref(false)
const testError = ref('')
const lastTest = ref(null)
async function triggerTest() {
  if (!current.value) return
  const ok = await openConfirm({
    title: t('skills.test.title'),
    message: t('skills.test.confirmRun', { name: current.value.name, version: current.value.version }),
    confirmText: t('skills.list.btnTest'),
  })
  if (!ok) return
  testOpen.value = true
  testing.value = true
  testError.value = ''
  lastTest.value = null
  try {
    const out = await runSkillTest({
      scope: current.value.scope,
      project_id: current.value.project_id,
      name: current.value.name,
      version: current.value.version,
      trigger: 'manual',
    })
    lastTest.value = out
  } catch (e) { testError.value = e?.message || String(e) }
  finally { testing.value = false }
}

// ====== 在文件夹打开 ======
const openError = ref('')
async function openInFolder() {
  if (!current.value) return
  openError.value = ''
  try {
    // 优先用 getSkill 返回的 source_path
    const sp = current.value._full?.canonical?.source_path
      || current.value._full?.source_path
      || ''
    if (!sp) { openError.value = 'no source path'; return }
    // 桌面端用 platform.fs.reveal;Web 端也是同一个实现
    const r = await platform.fs.reveal(sp)
    if (r && r.ok === false && r.fallbackUrl) {
      // Web 端兜底:打开 file://
      platform.platform.openExternal(r.fallbackUrl)
    }
  } catch (e) {
    openError.value = t('skills.list.openFailed', { msg: e?.message || String(e) })
  }
}

// ====== 复制路径 ======
// 2026-07-04 改:从工具栏图标按钮搬到右键菜单(node 参数版),
// 兼容 toolbar(current 走 _full)和右键(node)两条调用链。
// 路径优先用 current._full.canonical.source_path(详情区后端给的真实物理路径),
// 否则 storeRoot + node.path 拼出绝对路径,跟 openSkillInFolder 逻辑保持一致。
async function copySourcePath(node) {
  let sp = ''
  if (current.value && (!node || current.value.name === (node.skill_meta?.name || node.name))) {
    sp = current.value._full?.canonical?.source_path
      || current.value._full?.source_path
      || ''
  }
  if (!sp && node) {
    const root = skillTree.storeRoot || ''
    sp = root && node.path ? `${root}/${node.path}` : (node.path || '')
  }
  if (!sp) return
  try {
    await platform.platform.setClipboardText(sp)
  } catch (_) {
    try { await navigator.clipboard.writeText(sp) } catch (_) {}
  }
}

// ====== 新建 / 编辑(简化版:用弹窗) ======
const editorOpen = ref(false)
const draft = reactive({
  scope: 'global', project_id: 0, name: '', version: '0.1.0',
  description: '', triggersText: '', body: '',
  applyTools: [], // 2026-06-26:新建时勾选的"适用工具"列表
})
const editingKey = ref(null)
// 2026-06-26:弹窗内需要的项目列表(用于"项目作用域"下拉)
const editorProjects = ref([])
const editorProjectsLoading = ref(false)

function startNew() {
  Object.assign(draft, {
    scope: 'global', project_id: 0, name: '', version: '0.1.0',
    description: '', triggersText: '', body: '',
    applyTools: [],
  })
  editingKey.value = null
  error.value = ''
  editorOpen.value = true
  // 弹窗打开时拉一次项目列表(scope=project 才需要,但提前拉好)
  loadEditorProjects()
}

// 拉弹窗内需要的项目列表(全量,简单场景;支持搜索过滤)
async function loadEditorProjects(keyword = '') {
  editorProjectsLoading.value = true
  try {
    const out = await listProjects({ keyword: keyword || undefined, page: 1, size: 200 })
    editorProjects.value = out?.items || []
    // 默认选中第一个项目(若 draft.project_id == 0)
    if (draft.project_id === 0 && editorProjects.value.length) {
      draft.project_id = editorProjects.value[0].id || 0
    }
  } catch (_) { editorProjects.value = [] }
  finally { editorProjectsLoading.value = false }
}

// 工具列表(写死 5 个,跟 scope-status 工具行保持一致;不查后端)
const APPLY_TOOLS = [
  { tool_id: 'codex', display: 'Codex' },
  { tool_id: 'claude', display: 'Claude' },
  { tool_id: 'opencode', display: 'OpenCode' },
  { tool_id: 'cursor', display: 'Cursor' },
  { tool_id: 'trae', display: 'Trae' },
]
function toggleApplyTool(toolID) {
  const i = draft.applyTools.indexOf(toolID)
  if (i >= 0) draft.applyTools.splice(i, 1)
  else draft.applyTools.push(toolID)
}
function isApplyToolChecked(toolID) {
  return draft.applyTools.includes(toolID)
}

function buildSkillMd() {
  const triggers = draft.triggersText.split(/[\n,]/).map((s) => s.trim()).filter(Boolean)
  const m = {
    name: draft.name, version: draft.version,
    description: draft.description, triggers,
  }
  const yaml = Object.entries(m)
    .map(([k, v]) => Array.isArray(v) ? `${k}: [${v.map((x) => JSON.stringify(x)).join(', ')}]` : `${k}: ${JSON.stringify(v)}`)
    .join('\n')
  return `---\n${yaml}\n---\n\n${draft.body || ''}\n`
}
async function submit() {
  error.value = ''
  if (!draft.name.trim()) { error.value = t('skills.editor.errNameEmpty'); return }
  if (draft.description.trim().length < 10) { error.value = t('skills.editor.errDescShort'); return }
  const triggers = draft.triggersText.split(/[\n,]/).map((s) => s.trim()).filter(Boolean)
  if (triggers.length === 0) { error.value = t('skills.editor.errTriggersEmpty'); return }
  // 2026-06-26 增:作用域=project 时必须选具体项目
  if (draft.scope === 'project' && !draft.project_id) {
    error.value = t('skills.editor.errProjectRequired')
    return
  }
  const payload = {
    scope: draft.scope, project_id: draft.project_id,
    name: draft.name, version: draft.version,
    source: 'local',
    manifest: { name: draft.name, version: draft.version, description: draft.description, triggers },
    files: [{ path: 'SKILL.md', content: buildSkillMd() }],
  }
  try {
    if (editingKey.value) await updateSkill(payload)
    else await createSkill(payload)
    // 2026-06-26 增:创建/更新后,遍历勾选的工具调 apply 让 skill 在目标工具生效
    // 失败不阻断保存(apply 是"额外"动作),但要弹 toast 提示
    if (draft.applyTools.length) {
      // 2026-07-03 改:同 doApplyOne —— Service.Apply 宽容路径下 HTTP 200 也可能
      // 落盘失败,要 inspectApplyResult 读 res.partial_failure 显式判定。
      const failItems = []
      for (const tid of draft.applyTools) {
        try {
          const res = await applySkill({
            name: draft.name,
            scope: draft.scope,
            project_id: draft.project_id || 0,
            tools: [tid],
          })
          const ins = inspectApplyResult(res)
          if (!ins.allOk) {
            failItems.push(...ins.failedItems.map((f) => ({ tool: f.tool || tid, msg: f.error })))
          }
        } catch (e) {
          failItems.push({ tool: tid, msg: e?.message || String(e) })
        }
      }
      if (failItems.length) {
        const detail = formatFailedDetail(failItems.map((f) => ({ tool: f.tool, error: f.msg })))
        toast.error(t('skills.apply.partialFailed', {
          ok: draft.applyTools.length - failItems.length,
          total: draft.applyTools.length,
          detail,
        }), 6000)
      } else {
        toast.success(t('skills.editor.applyAllSuccess', { n: draft.applyTools.length }))
      }
    }
    editorOpen.value = false
    await reload()
    // 选回刚保存的(用 path 匹配,支持多级分组下同名 skill)
    const expectedPath = (payload.manifest?.group_path ? payload.manifest.group_path + '/' : '') + payload.name
    const row = items.value.find((x) => x.path === expectedPath) || items.value.find((x) => x.name === payload.name && x.version === payload.version)
    if (row) selectItem(row)
  } catch (e) { error.value = e?.message || String(e) }
}

// ====== 删除 ======
// 2026-06-29 改:删除链路 — 弹窗带 cascade 复选框(默认勾选);
// 勾选时,删完 skillbox 库内的副本后,循环拉 scope-status 拿到所有 (tool, scope, project)
// 命中,再循环调 forceUndoApply 同步清理工具目录(后端 service 已支持按 (scope, project, name, tool)
// 删磁盘副本,无需新加批量接口)。

// 工具目录清理的"幂等批量"实现:对每个 hit 调一次 forceUndoApply,失败聚合到 failList。
// 不会因为某个 hit 失败就中断。
async function cleanupToolDirs(name, version) {
  if (!name) return { okCount: 0, failCount: 0, fails: [] }
  let statusResp
  try {
    statusResp = await getSkillScopeStatus({ name, version: version || '' })
  } catch (e) {
    return { okCount: 0, failCount: 0, fails: [{ tool: '*', msg: e?.message || String(e) }] }
  }
  const hits = (statusResp?.hits || []).filter((h) => h.exists)
  if (hits.length === 0) return { okCount: 0, failCount: 0, fails: [] }
  const fails = []
  let okCount = 0
  for (const h of hits) {
    try {
      await forceUndoApply({
        scope: h.scope,
        project_id: h.project_id || 0,
        name,
        tool: h.tool_id,
      })
      okCount++
    } catch (e) {
      fails.push({ tool: h.tool_id, scope: h.scope, msg: e?.message || String(e) })
    }
  }
  return { okCount, failCount: fails.length, fails }
}

// 弹窗 state(reuse openConfirm,但加一个 cascade 复选框)
// 2026-06-29 改:用专门的 DeleteConfirm 组件,比 openConfirm 多一个 cascade 选项
const deleteOpen = ref(false)
const deleteTarget = ref(null) // { kind: 'skill' | 'group', name, path, version?, deletedSkillPaths? }
const deleteCascade = ref(true)
const deleteBusy = ref(false)

function openDeleteSkill(row) {
  deleteTarget.value = { kind: 'skill', name: row.name, path: row.path || row.name, version: row.version }
  deleteCascade.value = true
  deleteOpen.value = true
}
function openDeleteGroup(node) {
  // node 是 TreeNode;path 是分组相对 root 的路径(可空 = 根)
  // 提前从 tree 递归收集子树所有 skill path(给弹窗里"包含 N 个 skill"提示用)
  const collected = collectSkillPathsUnder(node)
  deleteTarget.value = {
    kind: 'group',
    name: node.name,
    path: node.path || '',
    deletedSkillPaths: collected,
  }
  deleteCascade.value = true
  deleteOpen.value = true
}

// 递归收集分组节点下所有 skill 路径(只取叶子,不含分组路径)
function collectSkillPathsUnder(node) {
  if (!node) return []
  if (!node.is_group) return [node.path]
  const out = []
  const walk = (n) => {
    if (!n.is_group) out.push(n.path)
    else for (const c of n.children || []) walk(c)
  }
  walk(node)
  return out
}

function closeDelete() {
  if (deleteBusy.value) return
  deleteOpen.value = false
  deleteTarget.value = null
}

async function confirmDelete() {
  if (!deleteTarget.value || deleteBusy.value) return
  const target = deleteTarget.value
  // 2026-07-07 增:删 skill 前,如果 InlinePanel 有 dirty,直接清掉(不弹询问 —
  // 删 skill 时文件都一起没,留 dirty 编辑无意义)。
  if (target.kind === 'skill' && inlinePanelRef.value?.isAnyDirty?.()) {
    inlinePanelRef.value.resetAllDirty()
  }
  const cascade = !!deleteCascade.value
  deleteBusy.value = true
  try {
    if (target.kind === 'skill') {
      await deleteSkill({ path: target.path, name: target.name })
      if (editing.value) cancelInlineEdit()
      current.value = null
      selectedKey.value = null
      // 同步工具目录
      if (cascade) {
        const r = await cleanupToolDirs(target.name, target.version)
        if (r.failCount > 0) {
          toast.error(t('skills.list.skillCascadePartial', {
            n: r.failCount,
            detail: r.fails.slice(0, 3).map((f) => `${f.tool}:${f.msg}`).join('; '),
          }))
        } else if (r.okCount > 0) {
          toast.success(t('skills.list.skillCascadeOk', { name: target.name, n: r.okCount }))
        } else {
          toast.success(t('skills.list.skillCascadeSkipped', { name: target.name }))
        }
      }
    } else if (target.kind === 'group') {
      // 先调 delete group;若 cascade=false 且非空,后端返 409,前端再 cascade=true 复调
      const r = await skillTree.deleteGroup(target.path, { cascade })
      if (!r.ok && r.need_cascade) {
        // 显示给用户:包含 N 个 skill,确认级联?
        toast.info(t('skills.list.groupDeleteConfirmCascade', { n: r.deleted_skill_paths?.length || 0 }))
        // 直接复调 cascade=true(用户已勾选)— 如果还是 fail 弹错
        const r2 = await skillTree.deleteGroup(target.path, { cascade: true })
        if (!r2.ok) {
          toast.error(t('skills.list.groupDeleteFailed', { msg: r2.error }))
          return
        }
        // 级联成功后,同步删每个 skill 的工具目录
        if (cascade) {
          let totalCleaned = 0
          let totalFailed = 0
          const failSummary = []
          for (const sp of r2.deleted_skill_paths || []) {
            const name = sp.split('/').pop()
            const cr = await cleanupToolDirs(name, '')
            totalCleaned += cr.okCount
            totalFailed += cr.failCount
            for (const f of cr.fails) failSummary.push(`${name}@${f.tool}:${f.msg}`)
          }
          if (totalFailed > 0) {
            toast.error(t('skills.list.skillCascadePartial', {
              n: totalFailed,
              detail: failSummary.slice(0, 3).join('; '),
            }))
          } else {
            toast.success(t('skills.list.skillCascadeOk', { name: target.name, n: totalCleaned }))
          }
        } else {
          toast.success(t('common.delete'))
        }
      } else if (!r.ok) {
        toast.error(t('skills.list.groupDeleteFailed', { msg: r.error }))
        return
      } else {
        toast.success(t('common.delete'))
      }
    }
    deleteOpen.value = false
    await reload()
    // 删完如果 current 还在(理论上不会),清掉
    if (target.kind === 'skill' && current.value?.path === target.path) {
      current.value = null
      selectedKey.value = null
    }
  } catch (e) {
    toast.error(target.kind === 'skill'
      ? t('skills.list.skillDeleteFailed', { msg: e?.message || String(e) })
      : t('skills.list.groupDeleteFailed', { msg: e?.message || String(e) }))
  } finally {
    deleteBusy.value = false
  }
}

// 兼容旧 removeCurrent(详情区右上角的删除按钮)— 改为弹我们的新弹窗
async function removeCurrent() {
  if (!current.value) return
  openDeleteSkill({
    name: current.value.name,
    path: current.value.path || current.value.name,
    version: current.value.version,
  })
}

// ====== 通用确认弹窗 ======
const confirmOpen = ref(false)
const confirmOpts = reactive({
  title: '', message: '', confirmText: '', cancelText: '', variant: 'default', resolve: null,
})
function openConfirm(opts) {
  confirmOpts.title = opts.title || t('common.confirm')
  confirmOpts.message = opts.message || ''
  confirmOpts.confirmText = opts.confirmText || t('common.confirm')
  confirmOpts.cancelText = opts.cancelText || t('common.cancel')
  confirmOpts.variant = opts.variant || 'default'
  confirmOpen.value = true
  return new Promise((resolve) => { confirmOpts.resolve = resolve })
}
function resolveConfirm(ok) {
  if (confirmOpts.resolve) confirmOpts.resolve(ok)
  confirmOpen.value = false
}

// 跳转 Onboarding(以弹窗形式打开)
function goOnboarding() {
  importOpen.value = true
}

// 列表项键盘可达性
const listRefs = ref([])
function focusItem(i) {
  const el = listRefs.value[i]
  if (el) { el.focus() }
}

// 导入弹窗
const importOpen = ref(false)
function openImport() { importOpen.value = true }
function onImported() {
  // 导入完成后,刷新列表
  reload()
}

// ====== 2026-06-29 增:右键菜单 state + 菜单项构造 + 处理函数 ======

// 右键菜单单例(整个 SkillsView 内只有一个)
const ctxMenu = reactive({
  open: false,
  x: 0,
  y: 0,
  items: [],
})
function closeCtxMenu() {
  ctxMenu.open = false
  ctxMenu.items = []
}

// skill 右键:删除 / 打 tag / 在文件夹打开
function onSkillContextMenu({ node, event }) {
  ctxMenu.x = event.clientX
  ctxMenu.y = event.clientY
  ctxMenu.items = [
    {
      key: 'open-folder',
      label: t('skills.list.ctxOpenFolder'),
      icon: 'mdi:folder-outline',
      onClick: () => openSkillInFolder(node),
    },
    {
      key: 'copy-path',
      label: t('skills.list.ctxCopyPath'),
      icon: 'mdi:content-copy',
      onClick: () => copySourcePath(node),
    },
    {
      key: 'tag',
      label: t('skills.list.ctxTag'),
      icon: 'mdi:tag-outline',
      onClick: () => openSkillTagDialog(node),
    },
    { divided: true, key: 'div-1', label: '' },
    {
      key: 'delete',
      label: t('skills.list.ctxDelete'),
      icon: 'mdi:delete',
      danger: true,
      onClick: () => openDeleteSkill({
        name: node.skill_meta?.name || node.name,
        path: node.path,
        version: node.skill_meta?.version,
      }),
    },
  ]
  ctxMenu.open = true
}

// 分组右键:重命名 / 在文件夹打开 / 删除
// 2026-07-03 改:首页分组只支持单级,移除"新建子分组"项。
function onGroupContextMenu({ node, event }) {
  ctxMenu.x = event.clientX
  ctxMenu.y = event.clientY
  const groupPath = node.path || ''
  ctxMenu.items = [
    {
      key: 'rename',
      label: t('skills.list.ctxRename'),
      icon: 'mdi:rename-outline',
      onClick: () => openRenameGroupDialog(node),
    },
    {
      key: 'open-folder',
      label: t('skills.list.ctxOpenFolder'),
      icon: 'mdi:folder-outline',
      onClick: () => openGroupInFolder(groupPath),
    },
    { divided: true, key: 'div-1', label: '' },
    {
      key: 'delete-group',
      label: t('skills.list.ctxDeleteGroup'),
      icon: 'mdi:folder-remove-outline',
      danger: true,
      onClick: () => openDeleteGroup(node),
    },
  ]
  ctxMenu.open = true
}

// 根区域(树空白处)右键:新建分组
function onRootContextMenu({ event }) {
  ctxMenu.x = event.clientX
  ctxMenu.y = event.clientY
  ctxMenu.items = [
    {
      key: 'new-group',
      label: t('skills.list.ctxNewGroup'),
      icon: 'mdi:folder-plus-outline',
      onClick: () => openNewGroupDialog(''),
    },
  ]
  ctxMenu.open = true
}

// ====== 新建分组弹窗 =====
const newGroupOpen = ref(false)
const newGroupPath = ref('')
const newGroupInput = ref('')
const newGroupBusy = ref(false)
function openNewGroupDialog(parentPath) {
  newGroupPath.value = parentPath || ''
  newGroupInput.value = ''
  newGroupOpen.value = true
}
function closeNewGroupDialog() {
  if (newGroupBusy.value) return
  newGroupOpen.value = false
}
async function submitNewGroup() {
  if (newGroupBusy.value) return
  const seg = (newGroupInput.value || '').trim()
  if (!seg) return
  const fullPath = newGroupPath.value ? `${newGroupPath.value}/${seg}` : seg
  newGroupBusy.value = true
  try {
    const r = await skillTree.createGroup(fullPath)
    if (!r.ok) {
      // 非法(可能含 .. 或含非法字符)— 走 i18n 兜底
      const msg = (r.error || '').toLowerCase().includes('invalid') || (r.error || '').includes('..')
        ? t('skills.list.groupInvalid')
        : r.error
      toast.error(t('skills.list.groupCreateFailed', { msg }))
      return
    }
    newGroupOpen.value = false
    await reload()
  } finally {
    newGroupBusy.value = false
  }
}

// ====== 2026-06-29 增:重命名分组弹窗 ======
const renameGroupOpen = ref(false)
const renameGroupOldPath = ref('')
const renameGroupOldName = ref('')
const renameGroupInput = ref('')
const renameGroupBusy = ref(false)
const renameGroupError = ref('')

// 取路径的父段(供弹窗预览 "frontend/<input>")
function pathDirname(p) {
  if (!p) return ''
  const i = p.lastIndexOf('/')
  return i < 0 ? '' : p.slice(0, i)
}

function openRenameGroupDialog(node) {
  // 根目录(空 path)不允许重命名 — 它没有 "最后一段"
  if (!node || !node.path) {
    toast.error(t('skills.list.groupRenameNotFound'))
    return
  }
  renameGroupOldPath.value = node.path
  renameGroupOldName.value = node.name
  renameGroupInput.value = node.name
  renameGroupError.value = ''
  renameGroupOpen.value = true
}
function closeRenameGroupDialog() {
  if (renameGroupBusy.value) return
  renameGroupOpen.value = false
}
async function submitRenameGroup() {
  if (renameGroupBusy.value) return
  const seg = (renameGroupInput.value || '').trim()
  if (!seg) {
    renameGroupError.value = t('skills.list.groupInvalid')
    return
  }
  // 本地预校验:只允许 a-z0-9-_,避免送后端再被拒
  if (!/^[a-z0-9_-]+$/.test(seg)) {
    renameGroupError.value = t('skills.list.groupInvalid')
    return
  }
  renameGroupBusy.value = true
  renameGroupError.value = ''
  try {
    const r = await skillTree.renameGroup({
      srcGroupPath: renameGroupOldPath.value,
      newName: seg,
    })
    if (!r.ok) {
      if (r.code === 'target_exists') {
        renameGroupError.value = t('skills.list.groupRenameConflict')
      } else if (r.code === 'not_found') {
        renameGroupError.value = t('skills.list.groupRenameNotFound')
      } else {
        const msg = (r.error || '').toLowerCase().includes('invalid') || (r.error || '').includes('..')
          ? t('skills.list.groupInvalid')
          : r.error
        renameGroupError.value = msg
      }
      return
    }
    renameGroupOpen.value = false
    const newName = r.new_group_path?.split('/').pop() || seg
    toast.success(t('skills.list.groupRenameOk', { name: newName }))
    // 乐观更新已经改过 tree,这里不强制 reload(避免选中态被刷掉)
  } finally {
    renameGroupBusy.value = false
  }
}

// ====== skill 在文件夹打开 ======
async function openSkillInFolder(node) {
  // 用 store 数据构造路径;store 内 path 是 "<group>/<name>",这里拼到 root 上
  // 详情区 current._full.canonical.source_path 是后端给的真实物理路径,优先用它
  let sp = ''
  if (current.value && current.value.name === (node.skill_meta?.name || node.name)) {
    sp = current.value._full?.canonical?.source_path || current.value._full?.source_path || ''
  }
  if (!sp) {
    // 2026-07-03 改:用 storeRoot + node.path(相对路径)拼绝对路径。
    // 旧版 ${skillTree.tree?.length ? '' : ''}${node.path} 三元两段都是空串,
    // 实际只传了相对路径,后端 fsutil.Reveal 收到非绝对路径 → os.Stat 失败 500。
    const root = skillTree.storeRoot || ''
    if (root && node.path) {
      sp = `${root}/${node.path}`
    } else {
      // storeRoot 未拉取(罕见)— 走旧逻辑兜底,至少不丢调用
      sp = node.path || ''
    }
  }
  if (!sp) {
    toast.error(t('skills.list.skillOpenFolderFailed', { msg: 'no source path' }))
    return
  }
  try {
    const r = await platform.fs.reveal(sp)
    if (r && r.ok === false && r.fallbackUrl) {
      platform.platform.openExternal(r.fallbackUrl)
    }
    toast.success(t('skills.list.skillOpenFolderOk'))
  } catch (e) {
    toast.error(t('skills.list.skillOpenFolderFailed', { msg: e?.message || String(e) }))
  }
}

async function openGroupInFolder(groupPath) {
  // 2026-07-03 改:用 storeRoot + groupPath 拼绝对路径。groupPath 为空时
  // 直接 reveal storeRoot(根区域)。storeRoot 未拉取时走旧逻辑兜底。
  const root = skillTree.storeRoot || ''
  let abs
  if (root) {
    abs = groupPath ? `${root}/${groupPath}` : root
  } else {
    abs = groupPath || '.'
  }
  try {
    const r = await platform.fs.reveal(abs)
    if (r && r.ok === false && r.fallbackUrl) {
      platform.platform.openExternal(r.fallbackUrl)
    }
  } catch (e) {
    toast.error(t('skills.list.skillOpenFolderFailed', { msg: e?.message || String(e) }))
  }
}

// ====== skill 打 tag(复用详情区右上角 tag 弹窗) ======
async function openSkillTagDialog(node) {
  const name = node.skill_meta?.name || node.name
  const path = node.path || name
  const version = node.skill_meta?.version
  // 直接调列表选中 + openTagDialog
  const row = items.value.find((x) => x.path === path)
  if (!row) {
    // 节点不在 store items 里(罕见)— 临时构造一个伪 row
    selectItem({ name, path, version })
  } else {
    selectItem(row)
  }
  // 等 current 切到再开 tag 弹窗(下一帧)
  await nextTick()
  openTagDialog()
}

// ====== 拖拽处理 ======
// 2026-06-30 改:drop 路由**完全统一到 .tree-container 一个 handler**。
// 之前用"TreeNode 内部 @drop emit + .tree-container 兜底"双 handler,DOM 冒泡
// + stopPropagation 互相打架,5 次 commit 没修好。这次换思路:
//   - TreeNode 内部不再 bind drop,只在 .tree-row 上加 :data-node-path(给
//     elementsFromPoint 用)
//   - .tree-container 一个 drop handler,drop 时用 document.elementsFromPoint
//     实时拿到"鼠标下最顶层的 .tree-row",读 data-node-path 得目标 group path
//   - 任何路径(具体 group / 容器空白处 = 根)都走同一个 onTreeDrop 入口
//
// 这样不管 TreeNode 怎么递归、stopPropagation 怎么搞,真相源唯一。

// 单一 drop 入口接收的 payload:
//   { targetPath: string, source: { type, path, name }, event: DragEvent }
// targetPath: 目标 group 的 path(空字符串 = 根)
async function onTreeDrop({ targetPath, source }) {
  if (!source) return
  if (source.type === 'skill') {
    if (!source.path) return
    const srcGroup = source.path.split('/').slice(0, -1).join('/')
    if (srcGroup === targetPath) {
      toast.info(t('skills.list.moveSameGroup'))
      return
    }
    const r = await skillTree.moveSkill({
      srcPath: source.path,
      srcGroupPath: srcGroup,
      name: source.path.split('/').pop(),
      dstGroupPath: targetPath,
    })
    if (!r.ok) {
      const msg = (r.error || '').includes('already exists')
        ? t('skills.list.moveTargetExists')
        : t('skills.list.moveFailed', { msg: r.error })
      toast.error(msg)
      return
    }
    toast.success(t('common.confirm'))
  } else if (source.type === 'group') {
    // 同位置:跳过(顶层 group 拖到根,或嵌套 group 拖到自身父级)
    if (source.path === targetPath) {
      toast.info(t('skills.list.moveSameGroup'))
      return
    }
    if (isGroupDescendant(targetPath, source.path)) {
      // 目标分组在源分组内部(把 aa 拖到 aa/bb 等)
      toast.error(t('skills.list.moveIntoDescendant'))
      return
    }
    // 顶层 group "挪到根" 是 no-op
    if (targetPath === '' && !source.path.includes('/')) {
      toast.info(t('skills.list.alreadyAtRoot'))
      return
    }
    const r = await skillTree.moveGroup({
      srcGroupPath: source.path,
      dstGroupPath: targetPath,
    })
    if (!r.ok) {
      toast.error(t('skills.list.moveFailed', { msg: r.error }))
      return
    }
    toast.success(t('common.confirm'))
  }
}

// isGroupDescendant:判断 child 是不是 parent 的子孙分组。
// 例:("aa/bb", "aa") = true,("aa", "aa/bb") = false,("aa", "aa") = false(同层不算)。
// 用 / 分隔段比较,避免 "aa-x" 被误判为 "aa" 的子孙。
function isGroupDescendant(child, parent) {
  if (!child || !parent) return false
  if (child === parent) return false
  return child.startsWith(parent + '/')
}

// ====== .tree-container 单一 drop / dragover / dragleave ======

// 用 document.elementsFromPoint 找"鼠标下最顶层的 .tree-row",
// 读它的 data-node-path 属性,得到目标 group path。
// 返回 '' 表示鼠标在容器空白处 = 拖到根。
function detectTargetGroupPath(x, y) {
  // elementsFromPoint 在 Vue/transition 动画中可能短暂返回 [],用 || [] 兜底
  const els = (typeof document !== 'undefined' && document.elementsFromPoint)
    ? document.elementsFromPoint(x, y) || []
    : []
  for (const el of els) {
    // 跳过 .tree-container 自身(它的 dataset.nodePath 可能是 undefined,
    // 但我们要的是它内部的 .tree-row,继续往下找)
    if (el.dataset && el.dataset.nodePath !== undefined) {
      return el.dataset.nodePath
    }
  }
  return ''
}

function onContainerDragOver(e) {
  // 只对 skillbox 内部节点拖动做高亮(避免外部文件拖入时误高亮)
  if (!e.dataTransfer?.types.includes('application/x-skillbox-node')) {
    e.dataTransfer.dropEffect = 'none'
    return
  }
  e.preventDefault()  // 必须 preventDefault 才能让 drop 事件触发
  e.dataTransfer.dropEffect = 'move'

  // 用 elementsFromPoint 实时算目标 group path,有变化才更新 store(避免响应式风暴)
  const targetPath = detectTargetGroupPath(e.clientX, e.clientY)
  if (targetPath !== lastHighlightedPath.value) {
    skillTree.setDropTarget(targetPath)
    rootDropHover.value = targetPath === ''  // 容器空白处 = 显示"放到根"提示
    lastHighlightedPath.value = targetPath
  }
}

function onContainerDrop(e) {
  e.preventDefault()
  const raw = e.dataTransfer?.getData('application/x-skillbox-node')
  // 清视觉态
  skillTree.setDropTarget('')
  rootDropHover.value = false
  lastHighlightedPath.value = ''
  if (!raw) return
  let source
  try {
    source = JSON.parse(raw)
  } catch (_) {
    return
  }
  const targetPath = detectTargetGroupPath(e.clientX, e.clientY)
  onTreeDrop({ targetPath, source, event: e })
}

function onContainerDragLeave(e) {
  // dragleave 在鼠标进入子元素时也会触发(relatedTarget 进入子节点),
  // 只有"鼠标真正离开容器"才是 e.target === e.currentTarget。
  if (e.target === e.currentTarget) {
    skillTree.setDropTarget('')
    rootDropHover.value = false
    lastHighlightedPath.value = ''
  }
}

// module-level 缓存上次高亮的 path,避免 dragover 高频触发时频繁 setDropTarget
// 引发响应式风暴。
const lastHighlightedPath = ref('')
const rootDropHover = ref(false)

// 2026-07-03 增:跨页通知 —— 监听 SettingsView 在 migrate 完成后 emit 的
// 'skills:refresh' 事件,触发 loadScopeStatus({ silent: true }) 静默重拉
// 当前选中 skill 的 scope-status,让用户切回首页时立刻看到新的磁盘形态
// (copy → symlink 或反向)而无需手动点 chip 触发 silent 刷新。
//
// 2026-07-07 改:scope 区搬到 SkillFileInlinePanel,parent 不再直接调
// loadScopeStatus — 改成 dispatch 'skillbox:scope-refresh' 事件,InlinePanel
// 内部监听这个事件后自己 loadScopeStatus。这样保持单一真相源。
//
// appBus 由 App.vue 行 22-39 provide;window event 兜底兼容 web 端
// (无 inject 上下文)和未来跨 webview 场景。
const appBus = inject('appBus', null)
function onSkillsRefresh() {
  // 仅在已选 skill 时刷新;未选时由 reload 在下次切 skill 时自然覆盖。
  if (current.value) {
    window.dispatchEvent(new CustomEvent('skillbox:scope-refresh'))
  }
}

onMounted(() => {
  reload()
  appBus?.on?.('skills:refresh', onSkillsRefresh)
  // 兜底:与 MarketView 跳 tab 的兼容写法对齐(行 119 dispatchEvent)
  window.addEventListener('skillbox:skills-refresh', onSkillsRefresh)
  // 2026-07-03 增:首次进入时拉工具列表,让 TreeNode 的 chip icon 有数据。
  // 如果用户先进 ToolsView 再进首页,这里 load() 会复用 store.items 缓存,
  // 实际不发请求(items.length > 0)。
  if (!toolsStore.items || toolsStore.items.length === 0) {
    toolsStore.load().catch(() => { /* 失败也不影响 skill 列表渲染 */ })
  }
})

onUnmounted(() => {
  appBus?.off?.('skills:refresh', onSkillsRefresh)
  window.removeEventListener('skillbox:skills-refresh', onSkillsRefresh)
})
</script>

<template>
  <div class="skills-layout">
    <!-- 左侧:技能列表 -->
    <aside class="skills-pane">
      <!-- 顶部操作栏 -->
      <div class="left-topbar">
        <button class="left-action" :title="t('skills.list.btnNewSkillTitle')" @click="startNew">
          <IconPark icon="mdi:plus" width="16" height="16" />
          <span>{{ t('skills.list.btnNewSkill') }}</span>
        </button>
        <button class="left-action" :title="t('skills.list.btnImportSkillTitle')" @click="goOnboarding">
          <IconPark icon="mdi:tray-arrow-down" width="16" height="16" />
          <span>{{ t('skills.list.btnImportSkill') }}</span>
        </button>
      </div>

      <!-- 搜索框 -->
      <div class="left-search">
        <IconPark icon="mdi:magnify" width="14" height="14" class="search-icon" />
        <input
          v-model="keyword"
          :placeholder="t('skills.searchPlaceholder')"
          class="search-input"
          :title="t('skills.list.searchTitle')"
          @keyup.enter="onSearchEnter"
        />
      </div>

      <p v-if="error" class="left-error">
        <IconPark icon="mdi:alert-circle-outline" width="12" height="12" />
        {{ error }}
      </p>

      <!-- 2026-06-30 改:拖拽路由**完全统一到 .tree-container 一个 handler**。
           TreeNode 内部不再 bind drop(只 bind @dragstart + :data-node-path)。
           .tree-container 单一接管:
             - @dragover  用 elementsFromPoint 实时高亮目标 group
             - @drop     单一入口,读 source + targetPath 走 onTreeDrop
             - @dragleave  鼠标真正离开容器时清视觉态(target===currentTarget 判定)
           之前 5 次 commit 反复在 TreeNode 内部绑 drop 修各种 bug,这次彻底放弃,
           一个容器一个 handler,真相源唯一。 -->
      <div
        class="tree-container"
        :class="{ 'tree-container-drag-over': rootDropHover }"
        :data-drop-text="t('skills.list.dropToRoot')"
        role="tree"
        :aria-label="t('skills.title')"
        @contextmenu.prevent="onRootContextMenu({ event: $event })"
        @dragover="onContainerDragOver"
        @drop="onContainerDrop"
        @dragleave="onContainerDragLeave"
      >
        <div v-if="loading" class="tree-loading">
          <span class="spinner"></span>
          <span>{{ t('skills.list.loadingTree') }}</span>
        </div>
        <TreeNode
          v-else
          :nodes="skillTree.tree"
          :selected-path="selectedKey"
          :collapsed-paths="skillTree.collapsedPaths"
          :drop-target-path="skillTree.dropTargetPath"
          :depth="0"
          :tools-by-id="toolsById"
          @select-skill="selectItem"
          @context-menu-skill="onSkillContextMenu"
          @context-menu-group="onGroupContextMenu"
          @context-menu-root="onRootContextMenu"
          @toggle-collapse="(p) => skillTree.toggleCollapse(p)"
        />
        <div v-if="!loading && !skillTree.totalSkills" class="tree-empty-hint">
          <p>{{ t('skills.list.emptyTitle') }}</p>
          <p>{{ t('skills.list.treeRootHint') }}</p>
        </div>
      </div>

      <!-- 翻页(暂时禁用,树形不分页) -->
      <footer v-if="false" class="left-pager">
        <button :disabled="page <= 1" @click="gotoPage(page - 1)">
          <IconPark icon="mdi:chevron-left" width="12" height="12" />
          {{ t('common.prev') }}
        </button>
        <span>{{ page }} / {{ totalPages }}</span>
        <button :disabled="page >= totalPages" @click="gotoPage(page + 1)">
          {{ t('common.next') }}
          <IconPark icon="mdi:chevron-right" width="12" height="12" />
        </button>
      </footer>
    </aside>

    <!-- 右侧:技能详情 -->
    <section class="detail-pane">
      <!-- 空状态 -->
      <div v-if="!current" class="detail-empty">
        <IconPark icon="mdi:cursor-default-click-outline" width="40" height="40" />
        <p class="empty-title">{{ t('skills.list.selectToView') }}</p>
      </div>

      <template v-else>


        <!-- 2026-06-26 新增:编辑态的描述/触发词 编辑区移到 toolbar 外,
             变成 detail-pane 下的独立 section,跟其他 detail-section 一样占满整页宽度
             (放在 toolbar 内会被原来的 detail-actions(6 个图标按钮)挤掉 35% 宽度) -->
        <section v-if="editing" class="detail-section detail-edit-fields">
          <div class="editor-field-full">
            <label>{{ t('skills.editor.description') }} <small>({{ t('skills.editor.descriptionHint') }})</small></label>
            <textarea
              v-model="editDescription"
              class="desc-editor"
              rows="2"
              spellcheck="false"
              :placeholder="t('skills.editor.descriptionHint')"
              :disabled="editSaving"
            ></textarea>
          </div>
          <div class="editor-field-full">
            <label>{{ t('skills.editor.triggers') }} <small>({{ t('skills.editor.triggersHint') }})</small></label>
            <textarea
              v-model="editTriggersText"
              class="triggers-editor"
              rows="1"
              spellcheck="false"
              :placeholder="t('skills.editor.triggersHintPlaceholder')"
              :disabled="editSaving"
            ></textarea>
          </div>
          <p v-if="editError" class="message message-error">
            <IconPark icon="mdi:alert-circle-outline" width="12" height="12" />
            {{ editError }}
          </p>
        </section>

        <p v-if="openError" class="message message-error">
          <IconPark icon="mdi:alert-circle-outline" width="12" height="12" />
          {{ openError }}
        </p>

        <!-- 2026-07-07 改:scope 区整段删除,已搬到 SkillFileInlinePanel 左栏顶部。
             旧版"工具行 + 作用域行"两行 chip 现在改为 InlinePanel 内部的
             "以工具为父级分组,展开后竖向列出生效位置"折叠树。 -->


        <!-- 标签 section:2026-06-25 改 — 不再展示 chip 列表,改为点击版本号弹出标签弹窗。
             section 本身保留,只显示一行说明 + "管理"按钮占位也行,但用户只要求"不打 tag 部分显示"。
             直接整段删掉,标签入口只剩顶栏的 tag-outline 按钮和 detail-version 点击。 -->

        <!-- 2026-06-25 改:触发词已搬到 description 下方行内展示,触发词 + 更新时间 独立 section 删除。
             更新时间挪到 detail-toolbar 标题行右侧,作为次要信息展示。 -->

        <!-- 2026-07-04 改 v2:正文区直接换成 SkillFileInlinePanel(目录树 + 预览/编辑),
             不再单独渲染 SKILL.md。SKILL.md 现在是文件树里的一个文件,
             点开就在右侧预览/编辑。 -->
        <section class="detail-section detail-body">
          <SkillFileInlinePanel
            v-if="current && currentFiles.length"
            ref="inlinePanelRef"
            :files="currentFiles"
            :skill="{
              name: current.name,
              version: current.version,
              scope: current.scope,
              project_id: current.project_id,
              source: current.source,
              group_path: current.group_path,
              canonical: current._full?.canonical,
            }"
            @saved="onDrawerSaved"
          >
            <!-- 2026-07-07 改 v3:把"编辑 / 取消 / 保存"按钮搬到 InlinePanel 面包屑行内,
                 跟 [i] 信息按钮同一栏、[i] 左侧依次排列(渲染顺序:name-actions → actions → [i])。
                 这些按钮控制当前 skill 的内联编辑态,state 在 SkillsView 侧,slot 形式透传。 -->
            <template #name-actions>
              <button
                v-if="!editing"
                class="ghost-link sfip-name-action-edit"
                :title="t('common.edit')"
                @click="startInlineEdit"
              >
                <IconPark icon="mdi:pencil" width="12" height="12" />
                {{ t('common.edit') }}
              </button>
              <template v-else>
                <button
                  class="title-action-btn title-action-cancel"
                  :disabled="editSaving"
                  @click="cancelInlineEdit"
                >
                  <IconPark icon="mdi:close" width="12" height="12" />
                  {{ t('common.cancel') }}
                </button>
                <button
                  class="title-action-btn title-action-save"
                  :disabled="editSaving"
                  @click="saveInlineEdit"
                >
                  <span v-if="editSaving" class="spinner spinner-sm"></span>
                  <IconPark v-else icon="mdi:content-save" width="12" height="12" />
                  {{ editSaving ? t('common.processing') : t('common.save') }}
                </button>
              </template>
            </template>
            <!-- 2026-07-07 改:右上角 5 个图标操作搬到这里,跟 [i] 信息按钮同一栏,
                 顺序 [测试 | 标签 | 在文件夹打开 | 删除 | AI] 在 [i] 左侧依次排列。
                 数据流向不变,@click / :data-tip / :disabled 都保留原语义。 -->
            <template #actions>
              <button
                class="icon-btn"
                :data-tip="t('skills.list.tooltipTest')"
                :aria-label="t('skills.list.tooltipTest')"
                :disabled="testing"
                @click="triggerTest"
              >
                <span v-if="testing" class="spinner spinner-sm"></span>
                <IconPark v-else icon="mdi:test-tube" width="15" height="15" />
              </button>
              <button
                class="icon-btn"
                :data-tip="t('skills.list.tooltipTag')"
                :aria-label="t('skills.list.tooltipTag')"
                @click="openTagDialog"
              >
                <IconPark icon="mdi:tag-outline" width="15" height="15" />
              </button>
              <button
                class="icon-btn"
                :data-tip="t('skills.list.tooltipOpenFolder')"
                :aria-label="t('skills.list.tooltipOpenFolder')"
                @click="openInFolder"
              >
                <IconPark icon="mdi:folder-outline" width="15" height="15" />
              </button>
              <button
                class="icon-btn"
                :data-tip="t('skills.list.tooltipDelete')"
                :aria-label="t('skills.list.tooltipDelete')"
                @click="removeCurrent"
              >
                <IconPark icon="mdi:delete" width="15" height="15" />
              </button>
              <button
                class="icon-btn ai-btn"
                :data-tip="aiOpen ? t('skills.btnAiClose') : t('skills.btnAiOpen')"
                :aria-label="aiOpen ? t('skills.btnAiClose') : t('skills.btnAiOpen')"
                @click="toggleAI"
              >
                <IconPark :icon="aiOpen ? 'mdi:robot' : 'mdi:robot-outline'" width="15" height="15" />
              </button>
            </template>
          </SkillFileInlinePanel>
          <p v-else-if="!currentFiles.length && !currentLoading" class="section-empty">{{ t('skills.list.bodyEmpty') }}</p>
        </section>
      </template>
    </section>

    <!-- AI 侧栏 -->
    <AIPanel v-if="aiOpen" :context-text="currentSkillMd" @apply="onAIApply" />

    <!-- 2026-07-04 改:文件浏览器改成正文右侧内联面板(不再用抽屉),挂载点已合并到 detail-body-split 里。 -->
    <!-- (旧) <SkillFileDrawer v-model="fileDrawerOpen" ... /> -->

    <!-- Tag 弹窗 -->
    <Modal
      v-model="tagOpen"
      size="xl"
      :title="current ? t('skills.tag.titlePrefix') + ' — ' + current.name + '@' + current.version : t('skills.tag.titlePrefix')"
    >
      <template #title-icon>
        <IconPark icon="mdi:tag-outline" width="18" height="18" />
      </template>

      <p v-if="tagMessage" class="message message-success">
        <IconPark icon="mdi:check-circle-outline" width="14" height="14" />
        {{ tagMessage }}
      </p>
      <p v-if="tagError" class="message message-error">
        <IconPark icon="mdi:alert-circle-outline" width="14" height="14" />
        {{ tagError }}
      </p>

      <div class="tag-create">
        <input v-model="newTagName" :placeholder="t('skills.tag.createPlaceholder')" class="tag-input" />
        <input v-model="newTagMessage" :placeholder="t('skills.tag.msgPlaceholder')" class="tag-input" />
        <button class="primary" :disabled="tagLoading" @click="doCreateTag">
          {{ tagLoading ? t('common.processing') : t('skills.tag.btnCreate') }}
        </button>
      </div>

      <div v-if="tagList.length" class="tag-actions">
        <span class="diff-label">{{ t('skills.tag.diff') }}:</span>
        <select v-model="diffLeftTagID">
          <option :value="0">{{ t('skills.tag.current') }}</option>
          <option v-for="tg in tagList" :key="tg.tag_id || tg.ID || tg.id" :value="tg.tag_id || tg.ID || tg.id">
            {{ tg.tag }} ({{ (tg.created_at || '').slice(0, 16) }}){{ tg.is_implicit ? t('skills.tag.implicit') : '' }}
          </option>
        </select>
        <IconPark icon="mdi:arrow-right" width="14" height="14" class="diff-arrow" />
        <select v-model="diffRightTagID">
          <option :value="0">{{ t('skills.tag.current') }}</option>
          <option v-for="tg in tagList" :key="tg.tag_id || tg.ID || tg.id" :value="tg.tag_id || tg.ID || tg.id">
            {{ tg.tag }} ({{ (tg.created_at || '').slice(0, 16) }}){{ tg.is_implicit ? t('skills.tag.implicit') : '' }}
          </option>
        </select>
        <button @click="doDiff(diffLeftTagID, diffRightTagID)">{{ t('skills.tag.seeDiff') }}</button>
        <button @click="doDiff(0, 0)">{{ t('skills.tag.clear') }}</button>
      </div>

      <ul v-if="tagList.length" class="tag-list">
        <li v-for="tg in tagList" :key="tg.tag_id || tg.ID || tg.id" :class="{ 'tag-implicit': tg.is_implicit }">
          <span class="tag-id">#{{ tg.tag_id || tg.ID || tg.id }}</span>
          <span class="tag-name"><code>{{ tg.tag }}</code></span>
          <span class="tag-msg">{{ tg.message || t('common.dash') }}</span>
          <span class="tag-time">{{ (tg.created_at || '').slice(0, 19) }}</span>
          <button class="link" @click="doDiff(tg.tag_id || tg.ID || tg.id, 0)">{{ t('skills.tag.vsCurrent') }}</button>
          <button class="link" :disabled="rolling" @click="doRollback(tg.tag_id || tg.ID || tg.id)">
            {{ rolling ? t('skills.tag.rollingBack') : t('skills.tag.rollbackTo') }}
          </button>
          <button class="link danger" @click="doDeleteTag(tg.tag_id || tg.ID || tg.id)">{{ t('common.delete') }}</button>
        </li>
      </ul>

      <div v-else-if="!tagLoading" class="empty-state empty-state-sm">
        <IconPark icon="mdi:tag-off-outline" width="36" height="36" />
        <p class="empty-title">{{ t('common.dash') }}</p>
      </div>

      <div v-if="diffResult" class="diff-panel">
        <header class="diff-header">
          <h4>{{ t('skills.tag.resultTitle') }}</h4>
          <div class="diff-stats">
            <span class="stat stat-added">+{{ t('skills.tag.added', { n: diffResult.added }) }}</span>
            <span class="stat stat-removed">-{{ t('skills.tag.removed', { n: diffResult.removed }) }}</span>
            <span class="stat stat-modified">~{{ t('skills.tag.modified', { n: diffResult.modified }) }}</span>
            <span class="stat stat-unchanged">={{ t('skills.tag.unchanged', { n: diffResult.unchanged }) }}</span>
          </div>
        </header>
        <div v-for="f in diffResult.files" :key="f.path" :class="['diff-file', `diff-kind-${f.kind}`]">
          <div class="diff-file-header">
            <span class="diff-file-kind">{{ f.kind }}</span>
            <code class="diff-file-path">{{ f.path }}</code>
          </div>
          <pre v-if="f.lines?.length" class="diff-content"><span v-for="(l, i) in f.lines" :key="i" :class="`diff-line diff-line-${l.kind}`"><span class="diff-line-no">{{ l.left_no || '' }}|{{ l.right_no || '' }}</span>{{ l.text }}
</span></pre>
        </div>
      </div>
    </Modal>

    <!-- 测试结果弹窗 -->
    <Modal
      v-model="testOpen"
      size="lg"
      :title="t('skills.test.title')"
    >
      <template #title-icon>
        <IconPark icon="mdi:test-tube" width="18" height="18" />
      </template>

      <div :class="['test-status-row', `test-status-${lastTest?.run?.status || 'errored'}`]">
        <span v-if="lastTest?.run" class="test-status-badge">{{ lastTest.run.status }}</span>
        <p v-if="testError" class="message message-error" style="margin: 0">
          <IconPark icon="mdi:alert-circle-outline" width="14" height="14" />
          {{ t('skills.test.errPrefix') }} {{ testError }}
        </p>
        <p v-else-if="lastTest?.run?.summary" class="test-summary">{{ lastTest.run.summary }}</p>
      </div>

      <ul v-if="lastTest?.results?.length" class="test-list">
        <li v-for="r in lastTest.results" :key="r.id || r.ID" :class="`test-check test-check-${r.status}`">
          <span class="test-check-name">{{ r.check }}</span>
          <span class="test-check-status" :class="`status-${r.status}`">{{ r.status }}</span>
          <span class="test-check-msg">{{ r.message }}</span>
        </li>
      </ul>

      <details v-for="r in lastTest?.results || []" :key="`d-${r.id || r.ID}`" class="test-detail">
        <summary>{{ r.check }} detail</summary>
        <pre>{{ r.detail }}</pre>
      </details>

      <div v-if="testing" class="test-loading">
        <span class="spinner"></span>
        <span>{{ t('common.processing') }}</span>
      </div>
    </Modal>

    <!-- 编辑弹窗 -->
    <Modal
      v-model="editorOpen"
      size="xl"
      :title="editingKey ? t('skills.editor.titleEdit') : t('skills.editor.titleNew')"
    >
      <template #title-icon>
        <IconPark :icon="editingKey ? 'mdi:pencil' : 'mdi:plus'" width="18" height="18" />
      </template>
      <form class="editor-form" @submit.prevent="submit">
        <div v-if="editingKey" class="editor-hint-bar">
          <code>{{ editingKey.name }}@{{ editingKey.version }}</code>
        </div>
        <!-- 2026-06-26 改:基础元数据两列(名称 / 版本) -->
        <div class="editor-grid editor-grid-2">
          <div class="editor-field">
            <label>{{ t('skills.editor.name') }}</label>
            <input v-model="draft.name" :placeholder="t('skills.editor.nameHint')" :disabled="!!editingKey" />
          </div>
          <div class="editor-field">
            <label>{{ t('skills.editor.version') }}</label>
            <input v-model="draft.version" :placeholder="t('skills.editor.versionHint')" :disabled="!!editingKey" />
          </div>
        </div>

        <!-- 2026-06-26 改:作用域区改为开关式(全局/项目)+ 项目下拉,更直观 -->
        <div class="editor-field-full">
          <label>{{ t('skills.editor.scope') }}</label>
          <div class="scope-toggle-row">
            <div class="segmented">
              <button
                type="button"
                :class="['seg-btn', draft.scope === 'global' ? 'seg-active' : '']"
                :disabled="!!editingKey"
                @click="draft.scope = 'global'"
              >
                <IconPark icon="mdi:earth" width="13" height="13" />
                {{ t('skills.editor.scopeGlobal') }}
              </button>
              <button
                type="button"
                :class="['seg-btn', draft.scope === 'project' ? 'seg-active' : '']"
                :disabled="!!editingKey"
                @click="draft.scope = 'project'"
              >
                <IconPark icon="mdi:folder-outline" width="13" height="13" />
                {{ t('skills.editor.scopeProject') }}
              </button>
            </div>
            <select
              v-if="draft.scope === 'project'"
              v-model.number="draft.project_id"
              class="project-select"
              :disabled="!!editingKey"
            >
              <option :value="0" disabled>{{ t('skills.editor.projectPick') }}</option>
              <option v-for="p in editorProjects" :key="p.id" :value="p.id">
                {{ p.alias || p.name }}<span v-if="p.alias && p.name"> · {{ p.name }}</span>
              </option>
            </select>
            <span v-else class="muted small-hint">{{ t('skills.editor.scopeGlobalHint') }}</span>
          </div>
        </div>

        <!-- 2026-06-26 新增:适用工具多选 — 提交后自动在勾选的工具上 enable -->
        <div class="editor-field-full">
          <label>{{ t('skills.editor.applyTools') }} <small>({{ t('skills.editor.applyToolsHint') }})</small></label>
          <div class="chip-row apply-tools-row">
            <button
              v-for="tool in APPLY_TOOLS"
              :key="tool.tool_id"
              type="button"
              :class="['chip', 'chip-tool-pick', isApplyToolChecked(tool.tool_id) ? 'chip-active' : '']"
              :title="tool.display"
              @click="toggleApplyTool(tool.tool_id)"
            >
              <IconPark :icon="toolIcon(tool.tool_id)" width="12" height="12" />
              <span>{{ tool.display }}</span>
            </button>
            <span v-if="!draft.applyTools.length" class="chip-empty muted">
              {{ t('skills.editor.applyToolsNone') }}
            </span>
            <span v-else class="chip-tool-selected-hint muted">
              {{ t('skills.editor.applyToolsSelected', { n: draft.applyTools.length }) }}
            </span>
          </div>
        </div>

        <div class="editor-field-full">
          <label>{{ t('skills.editor.description') }} <small>({{ t('skills.editor.descriptionHint') }})</small></label>
          <textarea v-model="draft.description" rows="2"></textarea>
        </div>

        <div class="editor-field-full">
          <label>{{ t('skills.editor.triggers') }} <small>({{ t('skills.editor.triggersHint') }})</small></label>
          <textarea v-model="draft.triggersText" rows="1" :placeholder="t('skills.editor.triggersHintPlaceholder')"></textarea>
        </div>

        <div class="editor-field-full">
          <label>{{ t('skills.editor.body') }}</label>
          <!-- 2026-06-27 改:新建/编辑弹窗 body 也用 Tiptap 所见即所得编辑器(与首页保持一致) -->
          <RichTextEditor
            v-model="draft.body"
            :placeholder="t('skills.list.bodyEmpty')"
            min-height="280px"
          />
        </div>

        <p v-if="error" class="message message-error" style="margin: 0 0 12px">
          <IconPark icon="mdi:alert-circle-outline" width="14" height="14" />
          {{ error }}
        </p>
      </form>
      <template #footer>
        <button type="button" class="ghost" @click="editorOpen = false">
          <IconPark icon="mdi:close" width="14" height="14" />
          {{ t('common.cancel') }}
        </button>
        <button type="button" class="primary" @click="submit">
          <IconPark :icon="editingKey ? 'mdi:content-save' : 'mdi:plus'" width="14" height="14" />
          {{ editingKey ? t('common.save') : t('common.create') }}
        </button>
      </template>
    </Modal>

    <!-- 通用确认弹窗 -->
    <Modal
      v-model="confirmOpen"
      size="sm"
      :title="confirmOpts.title"
      :close-on-mask="false"
    >
      <p class="confirm-message">{{ confirmOpts.message }}</p>
      <template #footer>
        <button type="button" class="ghost" @click="resolveConfirm(false)">
          {{ confirmOpts.cancelText }}
        </button>
        <button
          type="button"
          :class="confirmOpts.variant === 'danger' ? 'danger' : 'primary'"
          @click="resolveConfirm(true)"
        >
          {{ confirmOpts.confirmText }}
        </button>
      </template>
    </Modal>

    <!-- 导入技能 弹窗 -->
    <OnboardingImportDialog v-model="importOpen" @imported="onImported" />

    <!-- 2026-06-29 增:右键菜单(单例 portal) -->
    <ContextMenu
      v-if="ctxMenu.open"
      :x="ctxMenu.x"
      :y="ctxMenu.y"
      :items="ctxMenu.items"
      @close="closeCtxMenu"
    />

    <!-- 2026-06-29 增:新建分组 弹窗 -->
    <Modal v-model="newGroupOpen" size="sm" :title="t('skills.list.ctxNewGroup')" :close-on-mask="false">
      <template #title-icon>
        <IconPark icon="mdi:folder-plus-outline" width="18" height="18" />
      </template>
      <form class="new-group-form" @submit.prevent="submitNewGroup">
        <div class="editor-field-full">
          <label>{{ t('skills.list.groupNamePrompt') }}</label>
          <input
            v-model="newGroupInput"
            class="group-input"
            :placeholder="newGroupPath ? t('skills.list.groupNamePromptSub') : t('skills.list.groupNamePrompt')"
            :disabled="newGroupBusy"
            autofocus
          />
          <p v-if="newGroupPath" class="muted small-hint">
            {{ t('common.create') }} → <code>{{ newGroupPath }}/<span style="color: var(--text)">{{ newGroupInput || '...' }}</span></code>
          </p>
        </div>
      </form>
      <template #footer>
        <button type="button" class="ghost" :disabled="newGroupBusy" @click="closeNewGroupDialog">
          {{ t('common.cancel') }}
        </button>
        <button type="button" class="primary" :disabled="newGroupBusy || !newGroupInput.trim()" @click="submitNewGroup">
          <span v-if="newGroupBusy" class="spinner spinner-sm"></span>
          <IconPark v-else icon="mdi:check" width="14" height="14" />
          {{ t('common.create') }}
        </button>
      </template>
    </Modal>

    <!-- 2026-06-29 增:重命名分组 弹窗 -->
    <Modal v-model="renameGroupOpen" size="sm" :close-on-mask="!renameGroupBusy">
      <template #header>
        <h3 class="modal-title">
          <IconPark icon="mdi:rename-outline" width="18" height="18" />
          {{ t('skills.list.groupRenamePrompt', { name: renameGroupOldName }) }}
        </h3>
      </template>
      <form class="new-group-form" @submit.prevent="submitRenameGroup">
        <div class="editor-field-full">
          <input
            v-model="renameGroupInput"
            class="group-input"
            :placeholder="renameGroupOldName"
            :disabled="renameGroupBusy"
            autofocus
            @keyup.enter="submitRenameGroup"
          />
          <p class="muted small-hint">
            {{ t('skills.list.groupRenameHint') }}
          </p>
          <p v-if="renameGroupOldPath" class="muted small-hint">
            <code>{{ pathDirname(renameGroupOldPath) || '/' }}/<span style="color: var(--text)">{{ renameGroupInput || '...' }}</span></code>
          </p>
          <p v-if="renameGroupError" class="message message-error" style="margin: 8px 0 0">
            <IconPark icon="mdi:alert-circle-outline" width="12" height="12" />
            {{ renameGroupError }}
          </p>
        </div>
      </form>
      <template #footer>
        <button type="button" class="ghost" :disabled="renameGroupBusy" @click="closeRenameGroupDialog">
          {{ t('common.cancel') }}
        </button>
        <button
          type="button"
          class="primary"
          :disabled="renameGroupBusy || !renameGroupInput.trim() || renameGroupInput.trim() === renameGroupOldName"
          @click="submitRenameGroup"
        >
          <span v-if="renameGroupBusy" class="spinner spinner-sm"></span>
          <IconPark v-else icon="mdi:check" width="14" height="14" />
          {{ t('common.save') }}
        </button>
      </template>
    </Modal>

    <!-- 2026-06-29 增:删除确认弹窗(skill / 分组复用,带 cascade 复选框) -->
    <Modal
      v-model="deleteOpen"
      size="sm"
      :title="deleteTarget?.kind === 'group' ? t('skills.list.ctxDeleteGroup') : t('common.delete')"
      :close-on-mask="!deleteBusy"
    >
      <template #title-icon>
        <IconPark
          :icon="deleteTarget?.kind === 'group' ? 'mdi:folder-remove-outline' : 'mdi:delete'"
          width="18"
          height="18"
        />
      </template>
      <p class="confirm-message">
        <template v-if="deleteTarget?.kind === 'group'">
          {{ t('skills.list.groupDeleteConfirm', { name: deleteTarget.name }) }}
          <template v-if="(deleteTarget.deletedSkillPaths || []).length > 0">
            <br /><br />
            {{ t('skills.list.groupDeleteConfirmCascade', { n: deleteTarget.deletedSkillPaths.length }) }}
          </template>
        </template>
        <template v-else>
          {{ t('skills.list.skillDeleteConfirm', { name: deleteTarget?.name }) }}
        </template>
      </p>
      <label class="cascade-check">
        <input v-model="deleteCascade" type="checkbox" :disabled="deleteBusy" />
        <span>
          <template v-if="deleteTarget?.kind === 'group'">
            {{ t('skills.list.groupDeleteCascadeHint') }}
          </template>
          <template v-else>
            {{ t('skills.list.skillDeleteCascadeHint') }}
          </template>
        </span>
      </label>
      <template #footer>
        <button type="button" class="ghost" :disabled="deleteBusy" @click="closeDelete">
          {{ t('common.cancel') }}
        </button>
        <button type="button" class="danger" :disabled="deleteBusy" @click="confirmDelete">
          <span v-if="deleteBusy" class="spinner spinner-sm"></span>
          <IconPark v-else icon="mdi:delete" width="14" height="14" />
          {{ t('common.delete') }}
        </button>
      </template>
    </Modal>
  </div>
</template>

<style scoped>
.skills-layout {
  display: grid;
  grid-template-columns: 320px minmax(0, 1fr);
  grid-template-rows: minmax(0, 1fr);
  grid-auto-rows: minmax(0, 1fr);
  gap: 0;
  /* 取一屏高度 - 顶栏(topbar py-3 + 内容 ≈ 46px) - content-area 上下 padding(20+20)。
     88 是保守值,小屏可能略多出滚动条,大屏留白;不影响功能。
     内部 grid row 用 1fr,所以两栏等高并各自 overflow 滚。 */
  height: calc(100vh - 88px);
  min-height: 0;
  color: var(--text);
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  overflow: hidden;
}

/* grid 子项显式 min-height:0,否则 grid item 默认 min-height:auto
   会被 .detail-pane 的子内容撑大,父级 overflow 失效 */
.skills-pane,
.detail-pane {
  min-height: 0;
}

/* ============================================
   左侧 - 技能列表面板
   ============================================ */
.skills-pane {
  display: flex;
  flex-direction: column;
  min-height: 0;
  background: var(--bg-card);
  border-right: 1px solid var(--border);
}

.left-topbar {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
  padding: 12px;
  border-bottom: 1px solid var(--border);
  background: var(--bg-card);
  flex-shrink: 0;
}

.left-action {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  height: 32px;
  padding: 0 10px;
  font-size: 13px;
  font-weight: 500;
  background: var(--bg-card);
  border: 1px solid var(--border);
  color: var(--text);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all 0.15s ease;
}

.left-action:hover {
  background: var(--bg-hover);
  border-color: var(--text-faint);
}

.left-search {
  position: relative;
  padding: 10px 12px;
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}

.left-search .search-icon {
  position: absolute;
  left: 22px;
  top: 50%;
  transform: translateY(-50%);
  color: var(--text-faint);
  pointer-events: none;
}

.left-search .search-input {
  width: 100%;
  height: 32px;
  padding-left: 30px;
  font-size: 13px;
  background: var(--bg-card);
}

.left-error {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 0;
  padding: 8px 12px;
  background: var(--danger-dim);
  color: var(--danger);
  font-size: 12px;
}

.skill-list {
  list-style: none;
  margin: 0;
  padding: 10px;
  overflow-y: auto;
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 8px; /* 卡片之间留间距 */
}

/* 2026-06-25 改:列表项改卡片样式(圆角 + hover 浮起 + 选中蓝色边框) */
.skill-item {
  position: relative;
  display: flex;
  align-items: center;
  gap: 0;
  padding: 0;
  cursor: pointer;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  transition: border-color 0.15s ease, transform 0.15s ease, box-shadow 0.15s ease;
  outline: none;
  overflow: hidden;
}

/* 2026-06-25 三改:hover 不要灰色背景,只浮起 + 边框变深 */
.skill-item:hover {
  border-color: var(--text-faint);
  transform: translateY(-1px);
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.06);
}
.skill-item:focus-visible {
  border-color: var(--accent-blue);
  box-shadow: 0 0 0 2px var(--accent-blue-bg);
}
/* 2026-06-25 三改:选中态简约 — 只留蓝色边框,去掉 box-shadow 和浅灰背景 */
.skill-item-active {
  background: var(--bg-card);
  border-color: var(--accent-blue);
}
.skill-item-active:hover {
  background: var(--bg-card);
  border-color: var(--accent-blue);
}

/* 2026-06-25 四改:未选中时色条透明不显示,选中时用主色(不用蓝色) */
.skill-item-bar {
  flex-shrink: 0;
  width: 3px;
  align-self: stretch;
  background: transparent;
  margin-right: 0;
  transition: background-color 0.15s ease;
}

.skill-item-active .skill-item-bar { background: var(--primary); }

.skill-item-main {
  flex: 1;
  min-width: 0;
  padding: 10px 12px 10px 10px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.skill-item-head {
  display: flex;
  align-items: baseline;
  gap: 6px;
  min-width: 0;
}

.skill-item-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
  min-width: 0;
}

.skill-item-version {
  font-size: 11px;
  color: var(--text-faint);
  font-family: 'JetBrains Mono', monospace;
  flex-shrink: 0;
}

.skill-item-meta {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.skill-item-tool-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 6px;
  border-radius: 999px;
  background: var(--bg-subtle);
  color: var(--text-dim);
  border: 1px solid var(--border);
  font-size: 11px;
  line-height: 1;
}
.skill-item-tool-chip:hover {
  background: var(--bg-hover);
  color: var(--text);
}

.badge.gray {
  background: var(--bg-subtle);
  color: var(--text-dim);
  border: 1px solid var(--border);
}

/* 2026-06-29 改:左侧从 .skill-list(扁平卡片)换为 .tree-container(树形) */
.tree-container {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 8px 6px 12px;
  /* 2026-06-29 增:容器空白处可右键,给个 cursor 提示 */
  cursor: default;
  /* 2026-06-29 增:根拖入高亮 — 容器整体变蓝底色 + 虚线边框 + 文字"放到根"。
     触发条件:用户拖动 skillbox 节点(.tree-container-drag-over 类)。
     用 position:relative 让 ::after 文字能绝对定位。 */
  position: relative;
  transition: background-color 0.12s ease, border-color 0.12s ease;
  border: 1px dashed transparent;
  border-radius: var(--radius);
}
/* 2026-06-29 增:拖入根区域时高亮整个容器 + 显示"放到根"文字 */
.tree-container-drag-over {
  background: var(--accent-blue-bg);
  border-color: var(--accent-blue);
}
.tree-container-drag-over::before {
  /* 2026-06-29 改:用 attr(data-drop-text) 拿 i18n 文案,支持中英文切换。
     content 用 attr() 在主流浏览器都支持。 */
  content: attr(data-drop-text);
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  padding: 8px 18px;
  font-size: 13px;
  font-weight: 600;
  color: var(--accent-blue);
  background: var(--bg-card);
  border: 1px solid var(--accent-blue);
  border-radius: var(--radius);
  pointer-events: none;
  z-index: 5;
  white-space: nowrap;
  box-shadow: 0 2px 8px var(--accent-blue-bg);
}
/* 2026-06-29 增:树底部留白 + 微弱提示文字,告诉用户"这里能右键新建分组"。
   只在树为空时显示(避免常态视觉噪音) */
.tree-container::after {
  content: '';
  display: block;
  min-height: 60px;
}
.tree-empty-hint {
  padding: 24px 16px;
  text-align: center;
  color: var(--text-faint);
  font-size: 12px;
  font-style: italic;
  user-select: none;
  pointer-events: none; /* 不拦截右键,事件冒泡到 .tree-container */
}
.tree-loading,
.tree-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 32px 16px;
  color: var(--text-faint);
  text-align: center;
}
.tree-empty .hint {
  font-size: 12px;
  color: var(--text-faint);
  margin: 0;
}
.tree-loading {
  flex-direction: row;
  font-size: 12px;
  color: var(--text-dim);
}

/* 2026-06-29 增:新建分组表单 */
.new-group-form {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.group-input {
  width: 100%;
  height: 32px;
  padding: 0 10px;
  font-size: 13px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  outline: none;
  font-family: inherit;
}
.group-input:focus {
  border-color: var(--text);
  box-shadow: 0 0 0 3px var(--primary-dim);
}

/* 2026-06-29 增:删除确认弹窗的 cascade 复选框 */
.cascade-check {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  margin-top: 12px;
  padding: 10px 12px;
  background: var(--bg-subtle);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  font-size: 13px;
  color: var(--text);
  cursor: pointer;
  user-select: none;
}
.cascade-check input[type="checkbox"] {
  margin-top: 2px;
  flex-shrink: 0;
  width: 14px;
  height: 14px;
  cursor: pointer;
}
.cascade-check input[type="checkbox"]:disabled {
  cursor: not-allowed;
}
.cascade-check span {
  flex: 1;
  min-width: 0;
  line-height: 1.5;
}

.left-pager {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 12px;
  border-top: 1px solid var(--border);
  background: var(--bg-card);
  font-size: 12px;
  color: var(--text-dim);
  flex-shrink: 0;
}

.left-pager button {
  padding: 4px 8px;
  font-size: 12px;
  display: inline-flex;
  align-items: center;
  gap: 2px;
}

/* ============================================
   右侧 - 详情面板
   ============================================ */
.detail-pane {
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow-y: auto;
  background: var(--bg);
}

.detail-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  flex: 1;
  color: var(--text-faint);
  padding: 60px 20px;
}

.detail-empty .empty-title {
  margin: 0;
  font-size: 14px;
  color: var(--text-dim);
}

.detail-toolbar {
  /* 2026-07-07 改:detail-toolbar 简化为单行。
     旧版 .detail-title-block / .detail-name / .detail-version / .detail-desc-row /
     .detail-triggers-row / .triggers-label / .scope-card / .scope-row / .chip-tool* /
     .chip-scope-target* / .chip-flash / .chip-busy / .chip-mini-* 整段删除。
     顶部 toolbar 现在只放一个 .detail-toolbar-name(面包屑) + .detail-actions(5 个图标按钮)。
     toolbar 整体 padding 收紧,vertical center,贴合"轻量级"风格。 */
  position: sticky;
  top: 0;
  z-index: 5;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 16px;
  min-height: 40px;
  background: var(--bg-card);
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}

.detail-toolbar-name {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  flex: 1;
  font-size: 13px;
  color: var(--text);
}
.detail-toolbar-name-text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-weight: 500;
}
.detail-toolbar-name .muted {
  color: var(--text-faint);
  font-weight: 400;
  margin-left: 2px;
}
.detail-toolbar-source {
  font-size: 10px;
  padding: 1px 6px;
  font-weight: 500;
  text-transform: lowercase;
  letter-spacing: 0.2px;
}
.detail-toolbar-edit {
  margin-left: 8px;
  height: 26px;
  font-size: 11px;
  padding: 0 8px;
}

/* 2026-06-26 新增:标题行右侧的"取消/保存"实心按钮(沿用,2026-07-07 改放进 detail-toolbar-name 行内) */
.title-action-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 26px;
  padding: 0 10px;
  font-size: 12px;
  font-weight: 500;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all 0.12s ease;
  border: 1px solid transparent;
  margin-left: 6px;
}
.title-action-btn:disabled { opacity: 0.5; cursor: not-allowed; }

.title-action-cancel {
  background: var(--bg-card);
  color: var(--text-dim);
  border-color: var(--border);
}
.title-action-cancel:hover:not(:disabled) {
  background: var(--bg-hover);
  color: var(--text);
  border-color: var(--text-faint);
}

.title-action-save {
  background: var(--text);
  color: var(--bg-card);
  border-color: var(--text);
}
.title-action-save:hover:not(:disabled) {
  background: var(--primary-dim);
  color: var(--text);
  border-color: var(--text);
}

/* 2026-06-25 新增:编辑态的 description 编辑框
   2026-06-26 改:边框变细(默认 1px 淡边 + 焦点 1px 实线 + 主色 box-shadow,去掉 3px 厚光晕),
   不固定 width,跟父容器自适应占满整页宽度 */
.desc-editor {
  margin: 6px 0 0;
  display: block;
  width: 100%;
  height: 57px;
  min-height: 57px;
  padding: 8px 10px;
  font-family: inherit;
  font-size: 13px;
  line-height: 1.5;
  color: var(--text);
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  outline: none;
  resize: vertical;
  transition: border-color 0.12s ease, box-shadow 0.12s ease;
}
.desc-editor:hover { border-color: var(--text-faint); }
.desc-editor:focus {
  border-color: var(--text-faint);
  box-shadow: 0 0 0 1px var(--text-faint);
}
.desc-editor:disabled { opacity: 0.6; cursor: not-allowed; }

/* 2026-06-26 新增:编辑态的描述/触发词 编辑区独立 section */
.detail-edit-fields {
  display: flex;
  flex-direction: column;
  gap: 12px;
  background: var(--bg-subtle);
}
.detail-edit-fields .editor-field-full {
  gap: 6px;
}

/* 2026-07-07 改:.detail-actions 已删(5 个图标按钮搬到 SkillFileInlinePanel 的 #actions slot)。
   .icon-btn 仍保留(其它位置,如 Tag 弹窗按钮,可能仍用得上),仅去掉它的全局 variant 默认样式。
   注意:.icon-btn 的尺寸/间距/hover 仍由 SkillFileInlinePanel 内 .sfip-actions :deep(.icon-btn) 接管。 */

.icon-btn {
  width: 32px;
  height: 32px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  background: transparent;
  border: 1px solid transparent;
  color: var(--text-dim);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all 0.12s ease;
}

.icon-btn:hover:not(:disabled) {
  background: var(--bg-hover);
  color: var(--text);
  border-color: var(--border);
}

.icon-btn:disabled { opacity: 0.5; cursor: not-allowed; }

.icon-btn.ai-btn { color: var(--accent-blue); }
.icon-btn.ai-btn:hover { background: var(--accent-blue-bg); border-color: var(--accent-blue-border); }

/* 2026-07-07 改:删除按钮 danger hover 样式从旧的 .detail-actions 选择器下移出,
   改成全局 .icon-btn[aria-label="删除"] 兜底(SkillFileInlinePanel 内 #actions
   也会匹配上)。不依赖父级 .detail-actions 容器。 */
.icon-btn[aria-label="删除"]:hover:not(:disabled) {
  background: var(--danger-dim);
  color: var(--danger);
  border-color: var(--danger);
}

.spinner-sm {
  width: 12px;
  height: 12px;
  border-width: 2px;
}

.detail-section {
  padding: 14px 20px;
  border-bottom: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  gap: 8px;
}

/* 2026-06-27 改:作用域区做成独立卡片 — 白底 + 圆角 + 上下 margin,
/* 2026-07-07 改:scope-card 整段样式删除(scope 区已搬到 SkillFileInlinePanel)。
   detail-section 的通用 padding 还在,scope 不再需要单独的卡片样式。 */

.detail-section.detail-meta-row {
  flex-direction: row;
  flex-wrap: wrap;
  align-items: flex-start;
  gap: 24px;
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.section-header h3 {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin: 0;
  font-size: 12px;
  font-weight: 600;
  color: var(--text-dim);
  text-transform: uppercase;
  letter-spacing: 0.4px;
}

.ghost-link {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 8px;
  font-size: 12px;
  background: transparent;
  border: 1px solid transparent;
  color: var(--text-dim);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all 0.12s ease;
}

.ghost-link:hover { background: var(--bg-hover); color: var(--text); border-color: var(--border); }
.ghost-link:disabled { opacity: 0.5; cursor: not-allowed; }
.ghost-link.primary-link { color: var(--primary); }
.ghost-link.primary-link:hover { background: var(--primary-dim); }

.body-actions { display: inline-flex; align-items: center; gap: 4px; }

.md-editor {
  display: block;
  /* 自适应高度:在 .detail-body (flex:1) 内填满剩余空间;内容少时至少 320px */
  flex: 1;
  width: 100%;
  min-height: 320px;
  padding: 12px 14px;
  font-family: 'JetBrains Mono', 'Fira Code', ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 13px;
  line-height: 1.6;
  color: var(--text);
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  outline: none;
  /* 自适应高度时不需要手动 resize(用户拖拽会破坏自适应),禁止 */
  resize: none;
  transition: border-color 0.12s ease, box-shadow 0.12s ease;
}

.md-editor:focus {
  border-color: var(--text);
  box-shadow: 0 0 0 3px var(--primary-dim);
}

.chip-row {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
}

.chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  font-size: 12px;
  font-weight: 500;
  background: var(--bg-card);
  color: var(--text-dim);
  border: 1px solid var(--border);
  border-radius: var(--radius-full);
  cursor: pointer;
  transition: all 0.12s ease;
  user-select: none;
}

.chip:hover { background: var(--bg-hover); color: var(--text); }
.chip-active {
  background: var(--text);
  color: var(--bg-card);
  border-color: var(--text);
}
.chip-active:hover { background: var(--text); color: var(--bg-card); }

.chip-global.chip-active { background: var(--accent-blue); border-color: var(--accent-blue); color: #fff; }
.chip-project.chip-active { background: var(--accent-violet); border-color: var(--accent-violet); color: #fff; }
.chip-tag { cursor: default; }
.chip-tag:hover { background: var(--bg-card); color: var(--text-dim); }
.chip-trigger { background: var(--accent-amber-bg); color: var(--accent-amber); border-color: var(--accent-amber-border); }
.chip-trigger:hover { background: var(--accent-amber-bg); color: var(--accent-amber); }

.chip-empty {
  font-size: 12px;
  color: var(--text-faint);
}

/* ============================================
   2026-07-07 改:Scope 两级布局(2026-06-24 旧版)整段删除。
   旧 .scope-row / .scope-row-label / .chip-tool / .chip-scope-target* /
   .chip-busy / .chip-flash / .chip-mini-* / .chip-count / .chip-spinner /
   .chip-tool-selected / .chip-tool-syncing 全部不再使用。
   新的 scope 区在 SkillFileInlinePanel 内部,样式走 .sfip-scope*。
   ============================================ */

/* 编辑器弹窗"适用工具"chip 行尾提示(保留,2026-06-26 引入,跟 .chip-tool-pick 配合用) */
.chip-tool-selected-hint {
  font-size: 11px;
  padding-left: 4px;
}

.section-loading {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin: 0;
  font-size: 12px;
}
.small-hint { font-size: 11px; }

.section-empty {
  margin: 0;
  font-size: 12px;
  color: var(--text-faint);
}

.meta-block { display: flex; flex-direction: column; gap: 6px; min-width: 0; }
.meta-block-time { min-width: 180px; }
.meta-label { font-size: 11px; color: var(--text-faint); text-transform: uppercase; letter-spacing: 0.3px; }
.meta-value { font-size: 12px; color: var(--text-dim); font-family: 'JetBrains Mono', monospace; }
/* 2026-06-25 改:触发词从独立 section 改到 description 下方行内展示,纯文本一行,顿号分隔 */
.meta-text {
  font-size: 13px;
  color: var(--text);
  line-height: 1.5;
  word-break: break-word;
}
/* 2026-06-25 改:编辑态触发词变成 textarea(行内),可同步编辑
   2026-06-26 改:边框变细(去掉 3px 厚光晕,改用 1px 淡边 + 焦点 1px 主色细线),
   不固定 width,跟父容器占满 */
.triggers-editor {
  display: block;
  width: 100%;
  /* 2026-06-26 改:用 height 精确锁初始 1 行高(行高 20 + padding 16 + border 2 = 38),
     内容多了浏览器自动扩(配合 box-sizing:border-box)。min-height 留作下限保护 */
  height: 38px;
  min-height: 38px;
  padding: 8px 10px;
  font-family: 'JetBrains Mono', monospace;
  font-size: 12.5px;
  line-height: 1.6;
  color: var(--text);
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  outline: none;
  resize: vertical;
  transition: border-color 0.12s ease, box-shadow 0.12s ease;
}
.triggers-editor:hover { border-color: var(--text-faint); }
.triggers-editor:focus {
  border-color: var(--text-faint);
  box-shadow: 0 0 0 1px var(--text-faint);
}
.triggers-editor:disabled { opacity: 0.6; cursor: not-allowed; }

/* 2026-07-04 改 v2:detail-body 直接被 SkillFileInlinePanel 占满,
   不再是两栏(SKILL.md 单独 + 文件浏览器内联),目录树已经在 InlinePanel 里。 */
.detail-body {
  padding: 0;
  border-top: 1px solid var(--border);
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.md-body {
  /* 2026-07-07 改:字体 12.5px → 14.5px,正文偏小阅读吃力,统一放大一档。
     标题层级同步上调,代码块字体保持 13px(代码密度天然需要更紧凑)。 */
  font-size: 14.5px;
  line-height: 1.7;
  color: var(--text);
  word-wrap: break-word;
}

/* 标题 — 主色 + 左侧色条 */
.md-body :deep(h1),
.md-body :deep(h2),
.md-body :deep(h3),
.md-body :deep(h4) {
  margin: 24px 0 12px;
  font-weight: 600;
  color: var(--text);
  line-height: 1.4;
}
.md-body :deep(h1) {
  font-size: 24px;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--border);
}
.md-body :deep(h2) { font-size: 20px; }
.md-body :deep(h3) { font-size: 17px; }
.md-body :deep(h4) { font-size: 15px; color: var(--text-muted, #6b7280); }

/* 段落 */
.md-body :deep(p) { margin: 10px 0; }

/* 列表 */
.md-body :deep(ul),
.md-body :deep(ol) {
  margin: 10px 0 10px 4px;
  padding-left: 22px;
}
.md-body :deep(li) {
  margin: 4px 0;
  padding-left: 4px;
}
.md-body :deep(li > p) { margin: 4px 0; }

/* 引用 — 左侧色条 + 浅底 */
.md-body :deep(blockquote) {
  margin: 12px 0;
  padding: 8px 14px;
  border-left: 3px solid var(--accent-blue);
  background: var(--bg-subtle, rgba(59, 130, 246, 0.06));
  color: var(--text-muted, #4b5563);
  border-radius: 0 4px 4px 0;
}
.md-body :deep(blockquote p) { margin: 4px 0; }

/* 链接 */
.md-body :deep(a) {
  color: var(--accent-blue);
  text-decoration: none;
  border-bottom: 1px solid transparent;
  transition: border-color 0.12s, color 0.12s;
}
.md-body :deep(a:hover) {
  border-bottom-color: var(--accent-blue);
}

/* 水平线 */
.md-body :deep(hr) {
  border: none;
  height: 1px;
  background: var(--border);
  margin: 20px 0;
}

/* 行内 code — 深色徽章背景(蓝紫渐变)+ 白字,跟代码块深底风格统一 */
.md-body :deep(code) {
  font-family: 'JetBrains Mono', ui-monospace, 'SFMono-Regular', monospace;
  font-size: 0.88em;
  background: linear-gradient(180deg, #1e293b 0%, #0f172a 100%);
  color: #e2e8f0;
  padding: 1px 7px;
  border-radius: 4px;
  border: 1px solid #334155;
  font-weight: 500;
}
/* 行内 code 内嵌 token 配色 — hljs 在 markdown-it 的 code 选项里不会跑行内
   (只有 fence 块才 highlight),所以行内 code 不会有 .hljs-* span,这一组
   选择器保持占位,即使不命中也不影响。 */

/* 代码块(highlight.js 输出 <pre class="hljs"><code class="hljs language-xxx">)
   2026-07-08 改:深色底(GitHub dark 同款 #0d1117)+ 浅字 + 顶部语言徽章。
   旧版 #f6f8fa 浅底在 SkillViewer cv-md-wrap 同区域底色(亮色)对不上,
   跟 CodeViewer 黑底也割裂。统一用深底,跟站点整体视觉一致。 */
.md-body :deep(pre.hljs) {
  position: relative;
  margin: 14px 0;
  padding: 14px 16px;
  background: #0d1117;
  border: 1px solid #30363d;
  border-radius: var(--radius-sm, 6px);
  overflow-x: auto;
  font-family: 'JetBrains Mono', ui-monospace, 'SFMono-Regular', monospace;
  font-size: 12.5px;
  line-height: 1.65;
  color: #c9d1d9;
}
/* 语言徽章 — 深色风格 */
.md-body :deep(pre.hljs code[class*="language-"])::before {
  content: attr(class);
  position: absolute;
  top: 6px;
  right: 8px;
  font-size: 10px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: #8b949e;
  background: #161b22;
  border: 1px solid #30363d;
  padding: 1px 6px;
  border-radius: 3px;
  font-family: 'JetBrains Mono', monospace;
  pointer-events: none;
}
/* 提取语言名(从 class 里) */
.md-body :deep(pre.hljs code.language-javascript)::before,
.md-body :deep(pre.hljs code.language-js)::before { content: 'JS'; }
.md-body :deep(pre.hljs code.language-typescript)::before,
.md-body :deep(pre.hljs code.language-ts)::before { content: 'TS'; }
.md-body :deep(pre.hljs code.language-python)::before,
.md-body :deep(pre.hljs code.language-py)::before { content: 'PY'; }
.md-body :deep(pre.hljs code.language-go)::before { content: 'GO'; }
.md-body :deep(pre.hljs code.language-bash)::before,
.md-body :deep(pre.hljs code.language-sh)::before,
.md-body :deep(pre.hljs code.language-shell)::before { content: 'SH'; }
.md-body :deep(pre.hljs code.language-json)::before { content: 'JSON'; }
.md-body :deep(pre.hljs code.language-yaml)::before,
.md-body :deep(pre.hljs code.language-yml)::before { content: 'YAML'; }
.md-body :deep(pre.hljs code.language-sql)::before { content: 'SQL'; }
.md-body :deep(pre.hljs code.language-html)::before,
.md-body :deep(pre.hljs code.language-xml)::before { content: 'HTML'; }
.md-body :deep(pre.hljs code.language-css)::before { content: 'CSS'; }
.md-body :deep(pre.hljs code.language-markdown)::before,
.md-body :deep(pre.hljs code.language-md)::before { content: 'MD'; }
/* 检测到但未列入上方的语言(比如 hljs 自动识别) */
.md-body :deep(pre.hljs code:not([class*="language-"]))::before {
  content: 'TXT';
}

.md-body :deep(pre code) {
  background: transparent;
  padding: 0;
  border: none;
  border-radius: 0;
  color: inherit;
  font-size: inherit;
}

/* 表格 — 带边框 + 表头底色 + 斑马纹 */
.md-body :deep(table) {
  border-collapse: collapse;
  margin: 14px 0;
  width: 100%;
  font-size: 13px;
  border: 1px solid var(--border);
  border-radius: 4px;
  overflow: hidden;
}
.md-body :deep(thead) { background: var(--bg-subtle, #f6f8fa); }
.md-body :deep(th) {
  text-align: left;
  padding: 8px 12px;
  font-weight: 600;
  border-bottom: 2px solid var(--border);
  color: var(--text);
}
.md-body :deep(td) {
  padding: 8px 12px;
  border-bottom: 1px solid var(--border);
  border-top: none;
}
.md-body :deep(tbody tr:last-child td) { border-bottom: none; }
.md-body :deep(tbody tr:nth-child(even)) { background: var(--bg-subtle, rgba(0, 0, 0, 0.02)); }

/* 任务列表(GFM) */
.md-body :deep(input[type="checkbox"]) {
  margin-right: 6px;
  vertical-align: middle;
}

/* 图片 */
.md-body :deep(img) {
  max-width: 100%;
  height: auto;
  border-radius: 4px;
  display: block;
  margin: 10px 0;
  border: 1px solid var(--border);
}

.detail-loading {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--text-dim);
  font-size: 13px;
  padding: 12px 0;
}

.message {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  border-radius: var(--radius-sm);
  font-size: 12px;
  margin: 0;
}
.message-success { background: var(--success-dim); color: var(--success); }
.message-error { background: var(--danger-dim); color: var(--danger); }

/* ============================================
   Tag 弹窗(沿用原样)
   ============================================ */
.tag-create {
  display: flex;
  gap: 10px;
  margin-bottom: 16px;
}
.tag-input { flex: 1; }
.tag-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 16px;
  font-size: 13px;
  flex-wrap: wrap;
}
.diff-label { color: var(--text-dim); font-weight: 500; }
.diff-arrow { color: var(--text-faint); }

.tag-list {
  list-style: none;
  padding: 0;
  margin: 0;
  border-top: 1px dashed var(--border);
}
.tag-list li {
  display: grid;
  grid-template-columns: 50px 160px 1fr 160px auto auto auto;
  gap: 10px;
  align-items: center;
  padding: 10px 0;
  border-bottom: 1px dashed var(--border);
  font-size: 13px;
}
.tag-list li.tag-implicit {
  background: var(--bg-subtle);
  margin: 0 -20px;
  padding: 10px 20px;
  border-radius: var(--radius-sm);
  border: 1px dashed var(--border);
  border-bottom: 1px dashed var(--border);
}
.tag-id { font-family: 'JetBrains Mono', monospace; color: var(--text-faint); }
.tag-name code { background: var(--primary-dim); color: var(--text); }
.tag-msg { color: var(--text-dim); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.tag-time { color: var(--text-faint); font-size: 11px; }

.diff-panel {
  margin-top: 20px;
  padding: 16px;
  background: var(--bg-subtle);
  border: 1px solid var(--border);
  border-radius: var(--radius);
}
.diff-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
  flex-wrap: wrap;
  gap: 12px;
}
.diff-header h4 { margin: 0; font-size: 14px; color: var(--text); }
.diff-stats { display: flex; gap: 8px; }
.stat { padding: 2px 8px; border-radius: 4px; font-size: 11px; font-weight: 600; }
.stat-added { background: var(--success-dim); color: var(--success); }
.stat-removed { background: var(--danger-dim); color: var(--danger); }
.stat-modified { background: var(--warning-dim); color: var(--warning); }
.stat-unchanged { background: var(--bg-card); color: var(--text-dim); }

.diff-file { margin: 8px 0; border: 1px solid var(--border); border-radius: 6px; overflow: hidden; }
.diff-kind-added .diff-file-header { background: var(--bg-subtle); border-left: 3px solid var(--success); }
.diff-kind-removed .diff-file-header { background: var(--bg-subtle); border-left: 3px solid var(--danger); }
.diff-kind-modified .diff-file-header { background: var(--bg-subtle); border-left: 3px solid var(--warning); }
.diff-kind-unchanged .diff-file-header { background: var(--bg-card); }
.diff-file-header { display: flex; align-items: center; gap: 10px; padding: 8px 12px; }
.diff-file-kind { font-size: 11px; padding: 2px 6px; border-radius: 4px; background: var(--bg-card); color: var(--text-dim); text-transform: uppercase; font-weight: 600; }
.diff-file-path { font-size: 12px; color: var(--text); }
.diff-content { padding: 8px 12px; margin: 0; font-family: 'JetBrains Mono', monospace; font-size: 12px; line-height: 1.6; background: var(--bg-card); max-height: 300px; overflow: auto; white-space: pre; }
.diff-line { display: block; }
.diff-line-added { background: var(--bg-subtle); color: var(--text); border-left: 3px solid var(--success); }
.diff-line-removed { background: var(--bg-subtle); color: var(--text-dim); border-left: 3px solid var(--danger); text-decoration: line-through; }
.diff-line-context { color: var(--text-dim); }
.diff-line-no { display: inline-block; min-width: 40px; padding-right: 10px; color: var(--text-faint); user-select: none; }

/* ============================================
   测试 / 编辑 / 确认弹窗(沿用原样)
   ============================================ */
.test-status-row {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
  flex-wrap: wrap;
}
.test-status-row.test-status-passed { color: var(--success); }
.test-status-row.test-status-failed { color: var(--danger); }
.test-status-row.test-status-errored { color: var(--warning); }
.test-status-badge { padding: 3px 10px; border-radius: var(--radius-full); font-size: 11px; font-weight: 600; text-transform: uppercase; background: var(--text); color: var(--bg-card); }
.test-summary { color: var(--text-dim); font-size: 13px; margin: 0; flex: 1; min-width: 0; }
.test-list { list-style: none; padding: 0; margin: 0; }
.test-list li { display: grid; grid-template-columns: 140px 90px 1fr; gap: 12px; padding: 8px 0; border-bottom: 1px dashed var(--border); font-size: 13px; align-items: center; }
.test-check-name { font-family: 'JetBrains Mono', monospace; color: var(--text); }
.test-check-status { padding: 2px 8px; border-radius: var(--radius-full); font-size: 11px; font-weight: 600; text-align: center; }
.status-passed { background: var(--success-dim); color: var(--success); }
.status-failed { background: var(--danger-dim); color: var(--danger); }
.status-errored { background: var(--warning-dim); color: var(--warning); }
.status-skipped { background: var(--bg-subtle); color: var(--text-dim); }
.test-check-msg { color: var(--text-dim); }
.test-detail { margin-top: 8px; }
.test-detail summary { cursor: pointer; font-size: 12px; color: var(--text-dim); padding: 4px 0; }
.test-detail pre { background: var(--bg-subtle); padding: 12px; border-radius: var(--radius-sm); font-size: 11px; max-height: 200px; overflow: auto; margin: 8px 0 0; }
.test-loading { display: flex; align-items: center; gap: 10px; padding: 16px 0; color: var(--text-dim); }

.editor-form { display: flex; flex-direction: column; gap: 14px; }
.editor-hint-bar { background: var(--bg-subtle); border: 1px solid var(--border); border-radius: var(--radius-sm); padding: 8px 12px; font-size: 12px; color: var(--text-dim); }
.editor-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 14px; }
/* 2026-06-26 改:弹窗内的元数据两列(name / version)固定 2 列布局 */
.editor-grid-2 { grid-template-columns: 1fr 1fr; }
.editor-field, .editor-field-full { display: flex; flex-direction: column; gap: 6px; }
.editor-field-full small { color: var(--text-faint); }
.editor-field label, .editor-field-full label { font-size: 12px; font-weight: 500; color: var(--text-dim); }
/* 2026-06-26 改:.editor-field-full textarea 默认 100px 最小高度只对"正文"等大段内容生效,
   描述/触发词的 textarea(.desc-editor / .triggers-editor)已经用 height 精确锁了
   1/2 行行高,这条全局 min-height 不能再把它们拉高 */
.editor-field-full > textarea:not(.desc-editor):not(.triggers-editor) { min-height: 100px; }

/* 2026-06-26 新增:作用域开关(全局/项目) */
.scope-toggle-row {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}
.segmented {
  display: inline-flex;
  align-items: stretch;
  background: var(--bg-subtle);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 2px;
  gap: 2px;
}
.seg-btn {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  height: 30px;
  padding: 0 12px;
  font-size: 12.5px;
  font-weight: 500;
  color: var(--text-dim);
  background: transparent;
  border: 1px solid transparent;
  border-radius: calc(var(--radius-sm) - 2px);
  cursor: pointer;
  transition: all 0.12s ease;
}
.seg-btn:hover:not(:disabled) { color: var(--text); }
.seg-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.seg-btn.seg-active {
  background: var(--bg-card);
  color: var(--text);
  border-color: var(--border);
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.04);
}
.seg-btn.seg-active:hover { color: var(--text); }

.project-select {
  flex: 1;
  min-width: 180px;
  max-width: 360px;
  height: 30px;
  padding: 0 10px;
  font-size: 13px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--text);
  outline: none;
  transition: border-color 0.12s ease, box-shadow 0.12s ease;
}
.project-select:focus {
  border-color: var(--text);
  box-shadow: 0 0 0 3px var(--primary-dim);
}
.project-select:disabled { opacity: 0.6; cursor: not-allowed; }

/* 2026-06-26 新增:适用工具多选 chip(借用 scope chip-tool 风格,差异化在 active 色) */
.apply-tools-row {
  padding: 4px 0;
}
.chip-tool-pick {
  cursor: pointer;
  background: var(--bg-card);
  color: var(--text);
  border-color: var(--border);
  border-style: solid;
  font-family: inherit;
}
.chip-tool-pick.chip-active {
  background: var(--text);
  color: var(--bg-card);
  border-color: var(--text);
  border-style: solid;
}
.chip-tool-pick.chip-active:hover {
  background: var(--text);
  color: var(--bg-card);
}
.chip-tool-pick:not(.chip-active):hover {
  background: var(--bg-hover);
  color: var(--text);
  border-color: var(--text-faint);
}

.confirm-message { margin: 0; font-size: 14px; line-height: 1.6; color: var(--text); white-space: pre-line; }

.empty-state { padding: 48px 24px; text-align: center; color: var(--text-faint); background: var(--bg-subtle); border: 1px dashed var(--border); border-radius: var(--radius); }
.empty-state-sm { padding: 24px 16px; }

/* ============================================
   响应式
   ============================================ */
@media (max-width: 900px) {
  .skills-layout { grid-template-columns: 280px minmax(0, 1fr); }
}

@media (max-width: 720px) {
  .skills-layout {
    grid-template-columns: 1fr;
    grid-template-rows: 240px minmax(0, 1fr);
  }
  .skills-pane { border-right: none; border-bottom: 1px solid var(--border); }
  .scope-row { flex-direction: column; gap: 4px; }
  .scope-row-label { flex: none; padding-top: 0; }
}
</style>
