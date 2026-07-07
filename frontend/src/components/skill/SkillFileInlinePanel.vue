<script setup>
// SkillFileInlinePanel - 首页右侧主区域:目录树 + 预览/编辑
//
// 2026-07-04 改 v2:替换原来的 detail-body(SKILL.md 单独渲染区),
// 把整个右侧详情区换成"左目录树 + 右预览/编辑"两栏布局:
//   - 左侧 200px:技能包全文件树(含 SKILL.md)
//   - 右侧:文件预览/编辑(代码走 Monaco,markdown 也走 Monaco 不再单独渲染)
//   - SKILL.md 也走 Monaco 编辑(用 markdown language,统一编辑器风格)
//   - 支持编辑保存(updateSkill)
//
// 2026-07-04 改 v3:SKILL.md 的 frontmatter 不直接显示在编辑器里,
// 在面板顶部右侧加一个 [info] 图标,点击后弹窗显示完整的 frontmatter
// (name / version / description / triggers / author / license / depends_on / target_tools)。

import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import IconPark from '@/components/IconPark.vue'
import Modal from '@/components/Modal.vue'
import FileTreeView from './FileTreeView.vue'
import CodeViewer from './CodeViewer.vue'
// 2026-07-07 改:scope 区从 SkillsView 搬到本组件,自管 loadScopeStatus / apply / undo。
// 选中 skill 变更时,本组件内部重拉 scope-status,自身维护折叠态。
import { updateSkill, getStoreInfo, getSkillScopeStatus, applySkill, listApplies, undoApply, forceUndoApply } from '@/api/skillbox/skills'
import { inspectApplyResult, formatFailedDetail } from '@/api/skillbox/apply_result.js'
import { useToastStore } from '@/core/store/toast'
import { useAppStore } from '@/core/store/app'

const { t } = useI18n()
const toast = useToastStore()
const appStore = useAppStore()

const props = defineProps({
  // 技能包文件列表 [{path, content}] - 来自后端 canonical.files
  files: { type: Array, default: () => [] },
  // 当前选中的 skill:{ name, version, scope, project_id, source, group_path, canonical }
  skill: { type: Object, default: () => ({}) },
})

const emit = defineEmits(['saved'])

// 当前选中的文件
const selectedFile = ref(null)
const selectedKey = ref('')  // 用于 FileTreeView 的 selectedPath

// 2026-07-05 改:编辑模式从组件级 ref 改为按 path 索引的 map。
// 旧实现切文件就 reset view,导致用户编辑 A.md → 切到 B.md → 切回 A.md,
// A.md 又进入 view,跟"每个文件独立记忆"的直觉不一致。
// 2026-07-06 改:key 不能只用 path,跨 skill 切换时多个 skill 都有 SKILL.md,
// path 相同但语义不同 → 会串。改成 "<skillName>/<path>" 做 key。
// skillName 为空时(初始未加载),只用 path 也行,因为那时只可能有一个文件。
const editModeMap = reactive({})
function modeKey(skillName, path) {
  if (!path) return ''
  // skillName 可能为空:首屏 / 未加载完成时只用 path 也安全(只有一个 skill)
  return skillName ? `${skillName}/${path}` : path
}
function getMode(skillName, path) {
  const k = modeKey(skillName, path)
  if (!k) return 'view'
  return editModeMap[k] || 'view'
}
function setMode(skillName, path, m) {
  const k = modeKey(skillName, path)
  if (!k) return
  editModeMap[k] = m
}

// 监听 props.files 变化,更新 selectedFile
// 2026-07-04 修(Commit 8+):保存代码文件后,父组件 onDrawerSaved 会 reload 整个 skill,
// props.files 重新赋值,这个 watch 会触发。旧版总是 fallback 到 SKILL.md,
// 导致用户编辑了 examples/foo.py 点保存 → 跳回 SKILL.md,体验很糟。
// 修复:files 变化时优先保留 selectedKey(用户正在编辑的文件),找不到再 fallback SKILL.md。
//
// 2026-07-06 修:跨 skill 切换时,两个 skill 都可能有 SKILL.md,path 相同;
// 旧代码 "selectedFile.path === prev 时不替换" 导致保留旧 selectedFile 引用,
// selectedFile.content 仍是上一个 skill 的内容 → isDirty 用旧 orig 比新 current
// → 误判为 dirty。
// 新策略:files 引用变了 / 当前 skill.name 变了 → 强制用新 files 里的 found 对象
// 替换 selectedFile(即便 path 相同,也是不同 skill 的同名文件,内容不一样)。
watch(
  () => [props.files, props.skill?.name],
  () => {
    const files = props.files
    if (!files || !files.length) {
      selectedFile.value = null
      selectedKey.value = ''
      localFiles.clear()
      dirtyPaths.value = new Set()
      return
    }
    // 优先用 selectedKey 在新 files 里找;找到就用 found 替换 selectedFile
    // (不再做 "path 相等就保留旧 selectedFile" 的优化,跨 skill 切换必须替换)
    const prev = selectedKey.value
    const target = (prev && files.find((f) => f.path === prev))
      || files.find((f) => f.path === 'SKILL.md')
      || files[0]
    selectedFile.value = target
    selectedKey.value = target?.path || ''
  },
  { immediate: true },
)

function onSelectFile(file) {
  // 2026-07-05 改:不再强制 reset editMode,改用 editModeMap 按 path 独立记忆。
  // 用户在 A.md 点编辑 → 切到 B.md(B 默认 view) → 切回 A.md(仍 edit),
  // localFiles 也是按 path 隔离,Monaco/Tiptap 内容都还在。
  selectedFile.value = file
  selectedKey.value = file.path
}

// 编辑态独立副本
const localFiles = reactive(new Map())
const dirtyPaths = ref(new Set())

watch(
  () => [props.files],
  () => {
    localFiles.clear()
    // 2026-07-04 改:localFiles 存"Monaco 看到的内容"——
    // SKILL.md 存 body(剥 frontmatter),其它文件存原文。
    // 与 displayContent / isDirty 的语义保持一致,避免永远 dirty。
    for (const f of props.files || []) {
      const c = f.content || ''
      const stored = f.path === 'SKILL.md' ? splitSkillMd(c).body : c
      localFiles.set(f.path, stored)
    }
    dirtyPaths.value = new Set()
  },
  { immediate: true, deep: true },
)

const isReadOnly = computed(() => false)  // v2 改:所有文件都可编辑
const currentContent = computed(() => {
  const path = selectedFile.value?.path
  if (!path) return ''
  return localFiles.has(path) ? localFiles.get(path) : (selectedFile.value?.content || '')
})

// 2026-07-04 增:SKILL.md 在 Monaco 里**不显示 frontmatter 区域**(用户反馈太干扰)。
// 策略:
//   - Monaco 看到的内容 = body(去掉开头 --- 块)
//   - localFiles / selectedFile.content 始终存完整 SKILL.md 原文
//   - 保存时用 rebuildSkillMd 把 frontmatter + 编辑后 body 重新拼回
//   - 顶部加 [i] frontmatter 弹窗,告诉用户这些元数据存在但不在编辑器里
function splitSkillMd(text) {
  if (!text) return { frontmatter: '', body: '' }
  // 匹配开头 --- 到下一个 --- 的 frontmatter 块(允许末尾有空行)
  const m = text.match(/^---\s*\n([\s\S]*?)\n---\s*\n?/)
  if (!m) return { frontmatter: '', body: text }
  return {
    frontmatter: m[0],         // 含 --- 包裹的完整块
    body: text.slice(m[0].length),  // frontmatter 之后的内容
  }
}

// 给 Monaco 显示用的内容(SKILL.md 去掉 frontmatter,其它文件原样)
const displayContent = computed(() => {
  if (!selectedFile.value) return ''
  if (selectedFile.value.path === 'SKILL.md') {
    return splitSkillMd(currentContent.value).body
  }
  return currentContent.value
})

// 2026-07-05 增:当前选中文件的 mode,模板里用,内部切换也用。
// 等价于 editModeMap[selectedFile.path],但包成 computed 触发响应式更新。
const currentMode = computed(() => getMode(props.skill?.name, selectedFile.value?.path || ''))

const isDirty = computed(() => {
  const path = selectedFile.value?.path
  if (!path) return false
  if (!localFiles.has(path)) return false
  // 2026-07-04 改:SKILL.md 时比较 body(同 displayContent 的逻辑)
  const current = localFiles.get(path) || ''
  const origFull = selectedFile.value?.content || ''
  const orig = path === 'SKILL.md' ? splitSkillMd(origFull).body : origFull
  return current !== orig
})

const fileSize = computed(() => (currentContent.value || '').length)

function onContentChange(v) {
  const path = selectedFile.value?.path
  if (!path) return
  // 2026-07-04 改:SKILL.md 时,Monaco 拿到的是 body,原文件含 frontmatter,
  // 不能直接比 localFiles 跟 selectedFile.content(永远不等 → 永远 dirty)。
  // 统一存 localFiles = "Monaco 看到的内容"(SKILL.md 是 body,其它文件是原文)。
  // orig(用于 dirty 判定)同步剥 frontmatter。
  localFiles.set(path, v || '')
  const origFull = selectedFile.value?.content || ''
  const orig = path === 'SKILL.md' ? splitSkillMd(origFull).body : origFull
  const s = new Set(dirtyPaths.value)
  if ((v || '') !== orig) s.add(path)
  else s.delete(path)
  dirtyPaths.value = s
}

function onDirtyChange(d) {
  const path = selectedFile.value?.path
  if (!path) return
  const s = new Set(dirtyPaths.value)
  if (d) s.add(path)
  else s.delete(path)
  dirtyPaths.value = s
}

// 保存当前文件
const saving = ref(false)
async function saveCurrent() {
  if (!selectedFile.value) return
  saving.value = true
  try {
    const path = selectedFile.value.path
    let newContent = localFiles.get(path) || ''

    // 2026-07-04 改:SKILL.md 保存时,把 Monaco 编辑的 body + 原 frontmatter 拼回。
    // 否则保存的就是"剥离 frontmatter 的 body",磁盘文件就丢了元数据。
    if (path === 'SKILL.md') {
      const orig = selectedFile.value?.content || ''
      const { frontmatter } = splitSkillMd(orig)
      // 如果原文件有 frontmatter,拼回去;如果用户把 frontmatter 全删了,新文件也不加(尊重用户)
      if (frontmatter) {
        newContent = frontmatter + (newContent.startsWith('\n') ? '' : '\n') + newContent
      }
    }

    const updatedFiles = (props.files || []).map((f) =>
      f.path === path ? { ...f, content: newContent } : f,
    )
    await updateSkill({
      scope: props.skill.scope,
      project_id: props.skill.project_id,
      name: props.skill.name,
      version: props.skill.version,
      source: props.skill.source || 'local',
      manifest: props.skill.canonical?.manifest || {
        name: props.skill.name,
        version: props.skill.version,
      },
      files: updatedFiles,
    })
    selectedFile.value = { ...selectedFile.value, content: newContent }
    const s = new Set(dirtyPaths.value)
    s.delete(path)
    dirtyPaths.value = s
    // 2026-07-04 改:保存成功后自动切回渲染模式(用户编辑目的已达到,
    // 切回 view 让他们确认结果,也避免一直占着 Monaco 实例)
    // 2026-07-05 改:按 path 独立记忆,所以只重置当前文件的 mode
    setMode(props.skill?.name, path, 'view')
    emit('saved', { path, content: newContent })
    toast.success(t('skills.fileBrowser.saved', { path }))
  } catch (e) {
    toast.error(t('skills.fileBrowser.saveFailed', { msg: e?.message || e }))
  } finally {
    saving.value = false
  }
}

function resetCurrent() {
  if (!selectedFile.value) return
  const path = selectedFile.value.path
  const origFull = selectedFile.value.content || ''
  // 2026-07-04 改:SKILL.md 时 Monaco 看到的是 body,reset 也要把 body 写回 localFiles
  // 否则 onContentChange 比对会判 dirty(完整 vs body 永远不等)
  const orig = path === 'SKILL.md' ? splitSkillMd(origFull).body : origFull
  localFiles.set(path, orig)
  onContentChange(orig)
  const s = new Set(dirtyPaths.value)
  s.delete(path)
  dirtyPaths.value = s
}

// store_root(用于"在文件夹打开")
const storeRoot = ref('')
async function fetchStoreRoot() {
  if (storeRoot.value) return
  try {
    const info = await getStoreInfo()
    storeRoot.value = info?.store_root || ''
  } catch (_) { storeRoot.value = '' }
}

// ====== 2026-07-07 改:scope 区从 SkillsView 搬到本组件 ======
// 数据源:后端 getSkillScopeStatus(name, version) 返回的实时磁盘状态
// 形态:scopeTools(工具元数据列表) + scopeHits(扁平命中数组)
// 视图形态:scopeGroupByTool — 以 tool 为父级分组,展开后列出该工具的生效位置
//
// 折叠态 scopeCollapsed:
//   - null  = 全部展开(首次进入 / 切 skill 的初始态)
//   - Set   = 用户手动收起的 tool_id 集合(被收起的会一直保持折叠,
//     直到用户再次展开,或者切到另一个 skill 重新置 null)
const scopeTools = ref([])
const scopeHits = ref([])
const scopeLoading = ref(false)
const scopeError = ref('')
const scopeCollapsed = ref(null)

// 当前操作中的 (tool, scope, project),用于在 target 上显示 spinner 防重复点
const busyKey = ref('')
function busyKeyFor(toolID, scope, projectID) {
  return `${toolID}|${scope}|${projectID || 0}`
}
function isScopeTargetBusy(group, target) {
  return busyKey.value === busyKeyFor(group.tool_id, target.scope, target.project_id)
}

// 工具 id 短名(首字母大写,跟 SkillsView 旧 chip 行为一致)
function toolShort(toolID) {
  if (!toolID) return '?'
  return toolID.charAt(0).toUpperCase() + toolID.slice(1)
}
// 工具 icon:优先用 scopeTools[].icon,缺省时退化到 puzzle
function toolIcon(toolID) {
  const t = scopeTools.value.find((x) => x.tool_id === toolID)
  return t?.icon || 'mdi:puzzle-outline'
}

// 以 tool 为父级聚合 + 子项按 (scope, project) 聚合
// 每个 group.targets 的子项是"该 tool 下唯一一条 (scope, project) 记录",
// exists 直接来自该 tool 在该位置的命中。
const scopeGroupByTool = computed(() => {
  const out = []
  for (const t of scopeTools.value) {
    const toolHits = scopeHits.value.filter((h) => h.tool_id === t.tool_id)
    // 二次聚合:同 (scope, project_id) 理论上只一条,但保留兜底(取 exists=true 的那条)
    const map = new Map()
    for (const h of toolHits) {
      const key = h.scope === 'global' ? 'global' : `p:${h.project_id}`
      if (!map.has(key)) {
        map.set(key, {
          key,
          scope: h.scope,
          project_id: h.project_id || 0,
          project_label: h.project_label || (h.scope === 'global' ? t('skills.list.scopeGlobalChip') : ''),
          exists: !!h.exists,
          path: h.path,
        })
      } else if (h.exists) {
        map.get(key).exists = true
      }
    }
    const targets = Array.from(map.values())
    targets.sort((a, b) => {
      if (a.scope !== b.scope) return a.scope === 'global' ? -1 : 1
      return a.project_id - b.project_id
    })
    out.push({
      tool_id: t.tool_id,
      display: t.display_name || t.tool_id,
      icon: toolIcon(t.tool_id),
      hitCount: toolHits.filter((h) => h.exists).length,
      hasHit: toolHits.some((h) => h.exists),
      targets,
    })
  }
  return out
})

function isScopeToolCollapsed(toolID) {
  if (!scopeCollapsed.value) return false
  return scopeCollapsed.value.has(toolID)
}
function toggleScopeTool(toolID) {
  const cur = scopeCollapsed.value || new Set()
  const next = new Set(cur)
  if (next.has(toolID)) next.delete(toolID)
  else next.add(toolID)
  // 全部收起时回到 null(等价"全展开"的初始态,避免出现"全折叠"歧义)
  scopeCollapsed.value = next.size === scopeTools.value.length ? null : next
}

async function loadScopeStatus() {
  const sk = props.skill
  if (!sk || !sk.name) return
  scopeLoading.value = true
  scopeError.value = ''
  try {
    const resp = await getSkillScopeStatus({
      name: sk.name,
      version: sk.version,
    })
    scopeTools.value = resp?.tools || []
    scopeHits.value = resp?.hits || []
  } catch (e) {
    scopeError.value = e?.message || String(e)
  } finally {
    scopeLoading.value = false
  }
}

// 监听选中 skill 变化 → 重拉 scope-status + 重置折叠态(null = 全展开)
watch(
  () => [props.skill?.name, props.skill?.version],
  () => {
    if (!props.skill?.name) return
    scopeCollapsed.value = null
    loadScopeStatus()
  },
)

// 2026-07-07 增:外部(如 SkillsView 的 onDrawerSaved、跨页 skills:refresh 事件)触发
// 本组件重新拉 scope-status,这样用户在 SkillsView 改完 SKILL.md 保存后,
// 立即能看到 scope 列表刷新。InlinePanel 自身 apply/undo 完成后也会调本方法。
// 通过 window event 通信,避免 props 链路过深。
function onScopeRefresh() {
  loadScopeStatus()
}

onMounted(() => {
  fetchStoreRoot()
  if (props.skill?.name) loadScopeStatus()
  // 监听跨页 / 跨组件的 scope 重拉事件
  window.addEventListener('skillbox:scope-refresh', onScopeRefresh)
})
onUnmounted(() => {
  window.removeEventListener('skillbox:scope-refresh', onScopeRefresh)
})

// doApplyOne — 启用单个 (tool, scope, project) 组合
// 2026-07-07 改:从 SkillsView 搬过来。Service.Apply 即便单 tool 失败也返 200,
// 用 inspectApplyResult 读 res.all_ok / res.partial_failure 才能区分
// 真正成功 vs 后端把失败静默吞掉的场景。
async function doApplyOne(h) {
  busyKey.value = busyKeyFor(h.tool_id, h.scope, h.project_id)
  const targetSkill = props.skill && props.skill.name
    ? { name: props.skill.name, version: props.skill.version }
    : null
  try {
    const res = await applySkill({
      name: targetSkill.name,
      scope: h.scope,
      project_id: h.project_id || 0,
      tools: [h.tool_id],
    })
    await loadScopeStatus()
    const toolLabel = toolDisplay.value[h.tool_id] || h.tool_id
    const ins = inspectApplyResult(res)
    if (ins.allOk) {
      toast.success(t('skills.list.applySuccess', {
        path: `${toolLabel} · ${h.scope === 'global' ? t('skills.list.scopeGlobalChip') : `#${h.project_id}`}`,
      }))
    } else {
      const detail = formatFailedDetail(ins.failedItems)
      toast.error(t('skills.apply.partialFailed', {
        ok: (res?.applies?.length || 0) - ins.failedItems.length,
        total: res?.applies?.length || 0,
        detail,
      }), 6000)
      scopeError.value = detail
    }
  } catch (e) {
    toast.error(t('skills.list.applyFailed', { msg: e?.message || String(e) }))
    scopeError.value = t('skills.list.applyFailed', { msg: e?.message || String(e) })
  } finally {
    busyKey.value = ''
  }
}

// doUnapplyOne — 停用单个 (tool, scope, project) 组合
// 2026-07-07 改:从 SkillsView 搬过来。DB 没记录时走 force-undo 走 scope-status
// 删磁盘 + 插占位 rolled_back 行(用户手动 cp / 外部安装,scope-status 命中但
// skill_applies 表里没行)。
async function doUnapplyOne(h) {
  busyKey.value = busyKeyFor(h.tool_id, h.scope, h.project_id)
  const targetSkill = props.skill && props.skill.name
    ? { name: props.skill.name, version: props.skill.version }
    : null
  try {
    const list = await listApplies({
      scope: h.scope,
      name: targetSkill.name,
      tool: h.tool_id,
      status: 'applied',
      page: 1,
      size: 1,
    })
    const last = list?.items?.[0]
    if (!last) {
      await forceUndoApply({
        scope: h.scope,
        project_id: h.project_id || 0,
        name: targetSkill.name,
        tool: h.tool_id,
      })
      await loadScopeStatus()
      const toolLabel = toolDisplay.value[h.tool_id] || h.tool_id
      toast.success(t('skills.list.unapplySuccess', {
        path: `${toolLabel} · ${h.scope === 'global' ? t('skills.list.scopeGlobalChip') : `#${h.project_id}`}`,
      }))
      return
    }
    await undoApply({ apply_id: last.id })
    await loadScopeStatus()
    const toolLabel = toolDisplay.value[h.tool_id] || h.tool_id
    toast.success(t('skills.list.unapplySuccess', {
      path: `${toolLabel} · ${h.scope === 'global' ? t('skills.list.scopeGlobalChip') : `#${h.project_id}`}`,
    }))
  } catch (e) {
    toast.error(t('skills.list.unapplyFailed', { msg: e?.message || String(e) }))
    scopeError.value = t('skills.list.unapplyFailed', { msg: e?.message || String(e) })
  } finally {
    busyKey.value = ''
  }
}

// 工具显示名(优先用后端 tools 数组;缺省时退化到 toolShort 短名)
const toolDisplay = computed(() => {
  const m = {}
  for (const t of scopeTools.value) m[t.tool_id] = t.display_name || toolShort(t.tool_id)
  return m
})

// 点击生效位置 chip:已生效 → 停用,未生效 → 启用
async function handleScopeGroupClick(group, target) {
  if (!props.skill?.name) return
  const sk = props.skill
  const fakeHit = {
    tool_id: group.tool_id,
    scope: target.scope,
    project_id: target.project_id || 0,
    exists: !!target.exists,
    path: target.path,
  }
  if (target.exists) {
    const ok = await openConfirm({
      title: t('skills.list.unapplyConfirmTitle'),
      message: t('skills.list.unapplyConfirmMessage', {
        name: sk.name,
        tool: group.display,
        scope: target.project_label,
      }),
      confirmText: t('common.delete'),
      variant: 'danger',
    })
    if (!ok) return
    await doUnapplyOne(fakeHit)
    return
  }
  const ok = await openConfirm({
    title: t('skills.list.applyConfirmTitle'),
    message: t('skills.list.applyConfirmMessage', {
      name: sk.name,
      tool: group.display,
      scope: target.project_label,
    }),
    confirmText: t('common.confirm'),
  })
  if (!ok) return
  await doApplyOne(fakeHit)
}

// ====== 通用确认弹窗(reuse SkillsView openConfirm 思路,但本组件不依赖 SkillsView) ======
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
onMounted(fetchStoreRoot)

const skillRelPath = computed(() => {
  const gp = props.skill.group_path || ''
  return gp ? `${gp}/${props.skill.name || ''}` : (props.skill.name || '')
})

// ====== Frontmatter 弹窗 ======
// 2026-07-04 增:从 SKILL.md 文件内容解析 YAML frontmatter,弹窗展示。
// 不在 Monaco 里直接显示 frontmatter(让用户专心编辑正文)。
//
// 简易解析:不引 js-yaml 依赖(打包又 +30KB),自己写一个最小解析器,
// 只支持扁平的 key: value 和 key: [array] 语法(skillbox manifest 实际只用这些)。
const fmOpen = ref(false)

// 从 SKILL.md 文件内容里抽 frontmatter 块
function parseFrontmatter(text) {
  if (!text) return {}
  const m = text.match(/^---\s*\n([\s\S]*?)\n---/)
  if (!m) return {}
  const block = m[1]
  const out = {}
  // 每行格式:key: value  或  key: [a, b]
  for (const line of block.split('\n')) {
    const kv = line.match(/^([a-zA-Z_][\w]*)\s*:\s*(.*)$/)
    if (!kv) continue
    const key = kv[1]
    let v = kv[2].trim()
    // 数组:[a, b] → 拆
    if (v.startsWith('[') && v.endsWith(']')) {
      v = v.slice(1, -1).split(',').map((s) => {
        let x = s.trim()
        // 去掉外层引号
        if ((x.startsWith('"') && x.endsWith('"')) || (x.startsWith("'") && x.endsWith("'"))) {
          x = x.slice(1, -1)
        }
        return x
      }).filter(Boolean)
    } else if ((v.startsWith('"') && v.endsWith('"')) || (v.startsWith("'") && v.endsWith("'"))) {
      v = v.slice(1, -1)
    }
    out[key] = v
  }
  return out
}

const frontmatter = computed(() => {
  const md = (props.files || []).find((f) => f.path === 'SKILL.md')
  return parseFrontmatter(md?.content || '')
})

const hasFrontmatter = computed(() => Object.keys(frontmatter.value).length > 0)

// 展示用的 key 顺序(常用在前)
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
  // 其它 key 追加
  for (const k of Object.keys(fm)) {
    if (!FM_KEY_ORDER.includes(k)) ordered.push([k, fm[k]])
  }
  return ordered
})

function openFrontmatter() { fmOpen.value = true }
function closeFrontmatter() { fmOpen.value = false }
</script>

<template>
  <div class="sfip">
    <header class="sfip-header">
      <div class="sfip-title-block">
        <IconPark icon="mdi:folder-multiple-outline" width="16" height="16" />
        <span class="sfip-name">{{ skill?.name || '' }}<span v-if="skill?.version" class="sfip-version">@{{ skill.version }}</span></span>
        <span class="sfip-count">{{ (files || []).length }} files</span>
      </div>
      <!-- 2026-07-04 增:SKILL.md frontmatter 弹窗按钮(只读展示,不影响编辑)
           frontmatter 里有 name / version / triggers / description 等元数据,
           单独看比混在 markdown 正文里更清晰。 -->
      <button
        v-if="hasFrontmatter"
        class="sfip-fm-btn"
        :title="'查看 frontmatter'"
        :aria-label="'查看 frontmatter'"
        @click="openFrontmatter"
      >
        <IconPark icon="mdi:information-outline" width="15" height="15" />
      </button>
    </header>

    <div class="sfip-body">
      <!-- 左:作用域(以工具为父级,展开后竖向列出生效位置) + 文件树 上下分段 -->
      <nav class="sfip-left">
        <!-- 2026-07-07 改:作用域区从 SkillsView 搬到本组件,固定在目录树顶部 -->
        <section v-if="!scopeLoading && scopeGroupByTool.length" class="sfip-scope">
          <header class="sfip-scope-header">
            <IconPark icon="mdi:earth" width="13" height="13" />
            <span>{{ t('skills.list.scopeLabel') }}</span>
          </header>
          <ul class="sfip-scope-list">
            <li
              v-for="group in scopeGroupByTool"
              :key="group.tool_id"
              class="sfip-scope-group"
            >
              <button
                type="button"
                class="sfip-scope-row"
                :title="group.display"
                @click="toggleScopeTool(group.tool_id)"
              >
                <IconPark
                  :icon="isScopeToolCollapsed(group.tool_id) ? 'mdi:chevron-right' : 'mdi:chevron-down'"
                  width="12"
                  height="12"
                  class="sfip-scope-chevron"
                />
                <IconPark :icon="group.icon" width="12" height="12" />
                <span class="sfip-scope-row-name">{{ group.display }}</span>
                <span v-if="group.hitCount > 0" class="sfip-scope-row-count">{{ group.hitCount }}</span>
              </button>
              <ul v-if="!isScopeToolCollapsed(group.tool_id)" class="sfip-scope-targets">
                <li v-for="target in group.targets" :key="target.key">
                  <button
                    type="button"
                    :class="['sfip-scope-target', target.exists ? 'sfip-scope-target-active' : '']"
                    :title="target.exists ? t('skills.list.unapplyConfirmTitle') : t('skills.list.applyConfirmTitle')"
                    :disabled="!!busyKey"
                    @click="handleScopeGroupClick(group, target)"
                  >
                    <span
                      v-if="isScopeTargetBusy(group, target)"
                      class="sfip-spinner sfip-spinner-xs"
                    ></span>
                    <IconPark
                      v-else
                      :icon="target.scope === 'global' ? 'mdi:earth' : 'mdi:folder-outline'"
                      width="11"
                      height="11"
                    />
                    <span class="sfip-scope-target-name">{{ target.project_label }}</span>
                  </button>
                </li>
                <li v-if="!group.targets.length" class="sfip-scope-empty">
                  {{ t('skills.list.scopeEmpty') }}
                </li>
              </ul>
            </li>
          </ul>
          <p v-if="scopeError" class="sfip-scope-error">
            <IconPark icon="mdi:alert-circle-outline" width="11" height="11" />
            {{ scopeError }}
          </p>
        </section>
        <p v-else-if="scopeLoading" class="sfip-scope-loading">
          <span class="sfip-spinner sfip-spinner-xs"></span>
        </p>

        <!-- 文件树(目录树) -->
        <div class="sfip-tree-wrap">
          <FileTreeView
            v-if="(files || []).length"
            :files="files"
            :initial-selected-path="selectedKey"
            :dirty-paths="dirtyPaths"
            @select-file="onSelectFile"
          />
          <p v-else class="sfip-tree-empty">该技能包为空</p>
        </div>
      </nav>

      <!-- 右:文件预览/编辑 -->
      <main class="sfip-viewer">
        <header class="sfip-viewer-header">
          <span class="sfip-viewer-path">{{ selectedFile?.path || t('skills.fileBrowser.noFile') }}</span>
          <span v-if="selectedFile?.path" class="sfip-viewer-size">{{ fileSize }} B</span>
          <!-- 2026-07-04 增:编辑模式切换按钮(默认 view,点击进 edit,再点回 view)
               放在文件大小右侧,与 dirty 提示和保存按钮同一行
               2026-07-05 改:按当前文件的 mode 显示,模式存到 editModeMap[path]
               实现每个文件独立记忆 -->
          <button
            v-if="selectedFile?.path && currentMode === 'view'"
            class="sfip-mode-btn"
            :title="'编辑'"
            :aria-label="'编辑'"
            @click="setMode(props.skill?.name, selectedFile.path, 'edit')"
          >
            <IconPark icon="mdi:pencil-outline" width="14" height="14" />
          </button>
          <button
            v-else-if="selectedFile?.path && currentMode === 'edit'"
            class="sfip-mode-btn sfip-mode-btn-active"
            :title="'返回预览'"
            :aria-label="'返回预览'"
            @click="setMode(props.skill?.name, selectedFile.path, 'view')"
          >
            <IconPark icon="mdi:eye-outline" width="14" height="14" />
          </button>
          <span v-if="isDirty" class="sfip-viewer-dirty">● {{ t('skills.fileBrowser.unsavedShort') }}</span>
          <button
            v-if="isDirty"
            class="sfip-btn"
            :disabled="saving"
            @click="resetCurrent"
          >{{ t('skills.fileBrowser.discard') }}</button>
          <button
            v-if="isDirty"
            class="sfip-btn sfip-btn-primary"
            :disabled="saving"
            @click="saveCurrent"
          >
            <span v-if="saving" class="sfip-spinner"></span>
            <IconPark v-else icon="mdi:content-save" width="13" height="13" />
            {{ saving ? t('skills.fileBrowser.saving') : t('skills.fileBrowser.save') }}
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
          @update:content="onContentChange"
          @dirty-change="onDirtyChange"
        />
        <div v-else class="sfip-empty">
          <IconPark icon="mdi:file-outline" width="48" height="48" />
          <p>{{ t('skills.fileBrowser.pickOne') }}</p>
        </div>
      </main>
    </div>

    <!-- 2026-07-07 增:Confirm 弹窗(本组件 scope enable/disable 用) -->
    <Modal
      v-model="confirmOpen"
      size="sm"
      :title="confirmOpts.title"
      :close-on-mask="false"
    >
      <p class="sfip-confirm-message">{{ confirmOpts.message }}</p>
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

    <!-- 2026-07-04 增:Frontmatter 弹窗(SKILL.md 元数据,只读展示) -->
    <Modal v-model="fmOpen" size="md" :title="`${skill?.name || ''} · frontmatter`">
      <div class="sfip-fm">
        <p class="sfip-fm-hint">SKILL.md 文件头部的元数据,主入口信息从这里来。</p>
        <table class="sfip-fm-table">
          <tbody>
            <tr v-for="[k, v] in frontmatterEntries" :key="k">
              <th>{{ k }}</th>
              <td>
                <template v-if="Array.isArray(v)">
                  <span v-for="(item, i) in v" :key="i" class="sfip-fm-chip">{{ item }}</span>
                  <span v-if="!v.length" class="sfip-fm-empty">[]</span>
                </template>
                <template v-else>
                  <span class="sfip-fm-value">{{ v || '""' }}</span>
                </template>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
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
.sfip-header {
  display: flex;
  align-items: center;
  padding: 10px 16px;
  border-bottom: 1px solid var(--border);
  background: var(--bg-card);
  flex-shrink: 0;
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
.sfip-count {
  color: var(--text-faint);
  font-size: 11px;
  padding: 1px 6px;
  background: var(--bg-subtle);
  border-radius: 999px;
}
.sfip-fm-btn {
  background: transparent;
  border: 1px solid var(--border);
  color: var(--text-faint);
  width: 26px;
  height: 26px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  cursor: pointer;
  margin-left: auto;
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
}
/* 2026-07-07 改:左栏改成"作用域 + 文件树"上下两段,作用域固定在顶部,
   文件树占剩余空间并可滚动;总宽度沿用原 200px(与 FileTreeView 一致) */
.sfip-left {
  width: 200px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  min-height: 0;
  border-right: 1px solid var(--border);
  background: var(--bg-subtle);
}
.sfip-tree-wrap {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 8px 10px;
}
.sfip-tree-empty {
  color: var(--text-faint);
  font-size: 12px;
  padding: 12px 8px;
  margin: 0;
}

/* 2026-07-07 增:scope 区(以工具为父级分组,展开后竖向列出生效位置) */
.sfip-scope {
  border-bottom: 1px solid var(--border);
  padding: 8px 6px 6px;
  background: var(--bg-card);
  flex-shrink: 0;
  max-height: 55%;
  overflow-y: auto;
}
.sfip-scope-header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 0 6px 6px;
  font-size: 11px;
  color: var(--text-faint);
  text-transform: uppercase;
  letter-spacing: 0.4px;
  font-weight: 600;
}
.sfip-scope-list { list-style: none; margin: 0; padding: 0; }
.sfip-scope-group { margin: 0; }
.sfip-scope-row {
  display: flex;
  align-items: center;
  gap: 6px;
  width: 100%;
  padding: 4px 6px;
  background: transparent;
  border: 0;
  font-family: inherit;
  font-size: 12px;
  color: var(--text);
  cursor: pointer;
  border-radius: 4px;
}
.sfip-scope-row:hover { background: var(--bg-hover); }
.sfip-scope-chevron { color: var(--text-faint); flex-shrink: 0; }
.sfip-scope-row-name {
  flex: 1;
  text-align: left;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.sfip-scope-row-count {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 16px;
  height: 14px;
  padding: 0 4px;
  font-size: 10px;
  font-weight: 700;
  background: var(--accent-blue);
  color: white;
  border-radius: 7px;
  flex-shrink: 0;
}
.sfip-scope-targets {
  list-style: none;
  margin: 2px 0 4px;
  padding: 0 0 0 14px;
}
.sfip-scope-target {
  display: flex;
  align-items: center;
  gap: 6px;
  width: 100%;
  padding: 3px 6px;
  background: transparent;
  border: 1px solid transparent;
  font-family: inherit;
  font-size: 11.5px;
  color: var(--text-dim);
  cursor: pointer;
  border-radius: 4px;
}
.sfip-scope-target:hover:not(:disabled) { background: var(--bg-hover); color: var(--text); }
.sfip-scope-target:disabled { cursor: wait; opacity: 0.7; }
.sfip-scope-target-active {
  background: var(--accent-blue-bg);
  color: var(--accent-blue);
  border-color: var(--accent-blue-border);
}
.sfip-scope-target-active:hover:not(:disabled) {
  background: var(--accent-blue-bg);
  color: var(--accent-blue);
}
.sfip-scope-target-name {
  flex: 1;
  text-align: left;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.sfip-scope-empty {
  font-size: 11px;
  color: var(--text-faint);
  padding: 2px 6px 4px;
  font-style: italic;
}
.sfip-scope-error {
  display: flex;
  align-items: center;
  gap: 4px;
  margin: 4px 6px 0;
  padding: 4px 6px;
  background: var(--danger-dim);
  color: var(--danger);
  font-size: 11px;
  border-radius: 4px;
}
.sfip-scope-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 8px;
  margin: 0;
  border-bottom: 1px solid var(--border);
}
.sfip-spinner-xs { width: 10px; height: 10px; border-width: 1.5px; }
.sfip-confirm-message {
  margin: 0;
  font-size: 14px;
  line-height: 1.6;
  color: var(--text);
  white-space: pre-line;
}
.sfip-viewer {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  background: var(--bg-card);
}
.sfip-viewer-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 14px;
  border-bottom: 1px solid var(--border);
  color: var(--text-dim);
  font-size: 12px;
  background: var(--bg-card);
  flex-shrink: 0;
}
.sfip-viewer-path {
  flex: 1;
  font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.sfip-viewer-size {
  color: var(--text-faint);
  font-size: 11px;
}
.sfip-viewer-dirty {
  color: var(--accent-amber, #d97706);
  font-weight: 500;
}
/* 2026-07-04 增:view/edit 模式切换按钮 */
.sfip-mode-btn {
  background: transparent;
  border: 1px solid var(--border);
  color: var(--text-faint);
  width: 26px;
  height: 26px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  cursor: pointer;
  transition: background 120ms ease, color 120ms ease, border-color 120ms ease;
}
.sfip-mode-btn:hover {
  background: var(--bg-hover);
  color: var(--accent-blue);
  border-color: var(--accent-blue);
}
.sfip-mode-btn-active {
  background: var(--accent-blue);
  color: white;
  border-color: var(--accent-blue);
}
.sfip-mode-btn-active:hover {
  background: var(--accent-blue);
  color: white;
  filter: brightness(1.1);
}
.sfip-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  border: 1px solid var(--border);
  background: var(--bg-card);
  color: var(--text-dim);
  border-radius: 6px;
  font-size: 12px;
  cursor: pointer;
  transition: background 120ms ease, color 120ms ease;
}
.sfip-btn:hover:not(:disabled) {
  background: var(--bg-hover);
  color: var(--text);
}
.sfip-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.sfip-btn-primary {
  background: var(--accent-blue);
  color: white;
  border-color: var(--accent-blue);
}
.sfip-btn-primary:hover:not(:disabled) {
  filter: brightness(1.1);
  color: white;
}
.sfip-spinner {
  display: inline-block;
  width: 12px;
  height: 12px;
  border: 2px solid currentColor;
  border-top-color: transparent;
  border-radius: 50%;
  animation: sfip-spin 0.8s linear infinite;
}
@keyframes sfip-spin { to { transform: rotate(360deg); } }
.sfip-empty {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: var(--text-faint);
}

/* 2026-07-04 增:Frontmatter 弹窗内容样式 */
.sfip-fm {
  font-size: 13px;
}
.sfip-fm-hint {
  color: var(--text-faint);
  font-size: 12px;
  margin: 0 0 14px;
}
.sfip-fm-table {
  width: 100%;
  border-collapse: collapse;
}
.sfip-fm-table th,
.sfip-fm-table td {
  text-align: left;
  padding: 8px 10px;
  border-bottom: 1px solid var(--border);
  vertical-align: top;
}
.sfip-fm-table th {
  width: 130px;
  color: var(--text-dim);
  font-weight: 500;
  font-size: 12px;
}
.sfip-fm-table td {
  color: var(--text);
  font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
  font-size: 12.5px;
  word-break: break-all;
}
.sfip-fm-table tr:last-child th,
.sfip-fm-table tr:last-child td {
  border-bottom: none;
}
.sfip-fm-value {
  white-space: pre-wrap;
}
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
  font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
}
.sfip-fm-empty {
  color: var(--text-faint);
  font-style: italic;
}
</style>