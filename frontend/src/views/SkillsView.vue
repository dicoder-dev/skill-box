<script setup>
// SkillsView - 技能首页(左右布局)
//
// 左侧:技能列表(顶部"新建 / 导入"按钮 + 搜索框 + 列表项,选中态高亮)
// 右侧:选中 skill 的详情
//   - 顶部 toolbar:技能名 + 版本 + 源徽章;右侧操作图标(测试 / 打标签 / 在文件夹打开 / 删除,hover 显示文字)
//   - scope chips:多选,默认"全局"必选;其他取自 listProjects
//   - 标签列表(横向 chips)
//   - 下方渲染 SKILL.md 的 body(markdown 简单自渲染)

import { ref, reactive, computed, onMounted, watch, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { Icon } from '@iconify/vue'
import { listSkills, getSkill, createSkill, updateSkill, deleteSkill, getSkillScopeStatus, applySkill, listApplies, undoApply, forceUndoApply } from '@/api/skillbox/skills'
import { listProjects } from '@/api/skillbox/projects'
import { runSkillTest } from '@/api/skillbox/skill_test'
import { createTag, listTags, deleteTag, diffTag, rollbackTag } from '@/api/skillbox/tags'
import AIPanel from '@/components/AIPanel.vue'
import Modal from '@/components/Modal.vue'
import { renderMarkdown } from '@/core/utils/markdown.js'
import { platform } from '@/platform'
import OnboardingImportDialog from '@/components/OnboardingImportDialog.vue'
import { useToastStore } from '@/core/store/toast'

const { t } = useI18n()

// ====== 列表 + 选中态 ======
const keyword = ref('')
const loading = ref(false)
const error = ref('')
const items = ref([])
const total = ref(0)
const page = ref(1)
const size = 200
const selectedKey = ref(null) // 选中项的 key 字符串

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
    // 2026-06-26 改:在 updateSkill 之前先快照已启用的 (tool, scope, project_id) 组合,
    // 因为 updateSkill 成功后要遍历它们调 apply,把新内容同步拷贝到磁盘。
    // 快照必须在 updateSkill 之前,避免 apply 链路重入(虽然 updateSkill 不会改 scopeHits)
    const targetSkill = { ...current.value }
    const existingApplies = scopeHits.value
      .filter((h) => h.exists)
      .map((h) => ({ tool_id: h.tool_id, scope: h.scope, project_id: h.project_id || 0 }))
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
    // 2026-06-26 改:遍历已启用的 (tool, scope, project) 组合,逐个调 apply 让磁盘上的
    // 副本同步到新内容。后端 apply 本身就是"覆盖"语义(走 PreSnapshot + 原子写),
    // 所以前端只要"再调一次 apply"就完成同步。
    // 失败不阻断(主保存已成功),统一在末尾弹 toast 汇总。
    if (existingApplies.length) {
      const failList = []
      for (const a of existingApplies) {
        try {
          await applySkill({
            name: targetSkill.name,
            scope: a.scope,
            project_id: a.project_id,
            tools: [a.tool_id],
          })
        } catch (e) { failList.push({ tool: a.tool_id, scope: a.scope, project_id: a.project_id, msg: e?.message || String(e) }) }
      }
      if (failList.length) {
        toast.error(t('skills.editor.syncPartialFailed', {
          ok: existingApplies.length - failList.length,
          total: existingApplies.length,
        }))
      } else {
        toast.success(t('skills.editor.syncAllSuccess', { n: existingApplies.length }))
      }
    } else {
      // 没在已启用命中 → 提示"未同步到任何工具/项目",用户知道主保存成功但磁盘上还没生效
      toast.info(t('skills.editor.syncNone', { name: targetSkill.name }))
    }
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

// ====== Scope 两级展示(2026-06-24 改:不再可写,纯只读展示后端实时扫描结果) ======
// 旧版"勾选全局/项目 → 写回 scope 字段"的设计,被后端"直接读文件系统"方案替代。
// 现在只展示当前 skill 在 (tool, scope, project) 笛卡尔积下的实际存在情况:
//   - 工具行:5 个编程工具 chip,数字徽章 = 该工具下有几处命中
//   - 作用域行:全局 + 各项目 chip,chip 内角标列出哪些工具里有命中
// 不再写库、不再触发 updateSkill;用户要变更生效位置直接通过本地文件操作。
const scopeTools = ref([])        // [{tool_id, display_name, icon}]
const scopeProjects = ref([])     // [{id, name, alias, root_path}]
const scopeHits = ref([])         // [{tool_id, scope, project_id, project_label, path, exists, is_system}]
const scopeLoading = ref(false)
const scopeError = ref('')

// 2026-06-25 改:工具行 chip 改成"单选切换器",作用域 chip 只对当前选中工具生效。
// 未选中工具时,作用域 chip 置灰不可点,提示"先选工具"。
const selectedToolID = ref(null)  // 当前选中的 tool_id;null = 未选

// 2026-06-25 二改:工具 chip 点击后,后端正在重拉 scopeStatus 时,
// 在工具 chip 上显示 spinner 反馈用户"我正在同步磁盘状态"。
const syncingToolID = ref(null)   // 同步中的 tool_id;null = 未同步

// 2026-06-25 增:成功启用/停用后,被操作的 (scope, project_id) 短暂高亮 2s
// 用于让用户眼睛锁定刚操作的 chip。值是 key('global' | 'p:<id>')。
const flashTargetKey = ref(null)
let _flashTimer = null
function flashTarget(key) {
  flashTargetKey.value = key
  if (_flashTimer) clearTimeout(_flashTimer)
  _flashTimer = setTimeout(() => { flashTargetKey.value = null }, 2000)
}

// 全局 toast
const toast = useToastStore()

// 工具名 → 显示名(优先用后端 tools 数组;缺省时退化到 tool_id 本身)
const toolDisplay = computed(() => {
  const m = {}
  for (const t of scopeTools.value) m[t.tool_id] = t.display_name || t.tool_id
  return m
})

// 工具名 → 图标(后端 icon 字段已废弃为空,前端按 tool_id 映射 mdi)
const TOOL_ICON_MAP = {
  codex: 'mdi:console',
  claude: 'mdi:robot-outline',
  opencode: 'mdi:code-tags',
  cursor: 'mdi:cursor-default-click-outline',
  trae: 'mdi:leaf',
}
function toolIcon(toolID) { return TOOL_ICON_MAP[toolID] || 'mdi:puzzle-outline' }
function toolShort(toolID) {
  // 短名:codex/claude/opencode/cursor/trae 直接用 id,首字母大写
  if (!toolID) return '?'
  return toolID.charAt(0).toUpperCase() + toolID.slice(1)
}

// 命中聚合(后端按路径逐条返回,前端按 (scope, project_id) 聚合成"一个 chip")
//
// key 规则:
//   - global:'global'
//   - project:'p:<id>'
// value: { key, scope, project_id, project_label, hits: [...], existsCount }
const scopeTargets = computed(() => {
  const map = new Map()
  for (const h of scopeHits.value) {
    const key = h.scope === 'global' ? 'global' : `p:${h.project_id}`
    if (!map.has(key)) {
      map.set(key, {
        key,
        scope: h.scope,
        project_id: h.project_id || 0,
        project_label: h.project_label || (h.scope === 'global' ? t('skills.list.scopeGlobalChip') : ''),
        hits: [],
        existsCount: 0,
      })
    }
    const e = map.get(key)
    e.hits.push(h)
    if (h.exists) e.existsCount++
  }
  // 全局放最前,其余项目按 project_id 升序
  const list = Array.from(map.values())
  list.sort((a, b) => {
    if (a.scope !== b.scope) return a.scope === 'global' ? -1 : 1
    return a.project_id - b.project_id
  })
  return list
})

// 工具聚合:每个 tool_id 对应 { tool_id, display, icon, hitCount, hasHit }
const scopeToolSummary = computed(() => {
  const out = []
  for (const t of scopeTools.value) {
    const hits = scopeHits.value.filter((h) => h.tool_id === t.tool_id)
    const hitCount = hits.filter((h) => h.exists).length
    out.push({
      tool_id: t.tool_id,
      display: t.display_name || t.tool_id,
      icon: toolIcon(t.tool_id),
      hitCount,
      hasHit: hitCount > 0,
    })
  }
  return out
})

// 2026-06-25 删:isScopeTargetDisabled 不再用,未选工具时点 chip 改弹 toast 提示。
function selectedToolHitExists(target) {
  if (!selectedToolID.value) return false
  const h = target.hits.find((x) => x.tool_id === selectedToolID.value)
  return !!(h && h.exists)
}
function selectedToolBusy(target) {
  if (!selectedToolID.value) return false
  return target.hits.some((h) => h.tool_id === selectedToolID.value && isBusy(h.tool_id, h.scope, h.project_id))
}

// 2026-06-25 二改:加 silent 选项。
//   - silent=false(默认):切换 scopeLoading,模板 v-if 会让整段 scope 区替换成 spinner;
//     适合"切 skill / 首次加载",需要先展示骨架再填数据。
//   - silent=true:不切 scopeLoading,保留旧 chip 视觉只更新 scopeHits;
//     适合"选工具时重拉同步",用户已能看到 chip,只是要后台刷新。
// silent 模式失败:不弹全屏 error 段(避免盖住 chip),把错误塞进 scopeError 静默记录
// (tool-level scope 本身就是只读镜像,失败不会阻断操作)。
async function loadScopeStatus({ silent = false } = {}) {
  if (!current.value) return
  if (!silent) scopeLoading.value = true
  scopeError.value = ''
  try {
    const resp = await getSkillScopeStatus({
      name: current.value.name,
      version: current.value.version,
    })
    scopeTools.value = resp?.tools || []
    scopeProjects.value = resp?.projects || []
    scopeHits.value = resp?.hits || []
    // 2026-06-25:加载完成后,如果之前选中的工具不在新工具列表里,清空选中
    if (selectedToolID.value && !scopeTools.value.some((t) => t.tool_id === selectedToolID.value)) {
      selectedToolID.value = null
    }
  } catch (e) {
    scopeError.value = e?.message || String(e)
    if (!silent) {
      scopeTools.value = []
      scopeProjects.value = []
      scopeHits.value = []
    }
    selectedToolID.value = null
  } finally {
    if (!silent) scopeLoading.value = false
  }
}

// ====== scope chip 点击行为(2026-06-24 新增) ======
// 行为:未生效 chip → 调 apply(同名已存在时弹确认框让用户选覆盖);
// 已生效 chip → 调 unapply(弹 danger 确认框二次确认)。
// 进度反馈:用 busyKey 标记当前操作的 (tool_id, scope, project_id),
// 在 chip 上显示 spinner 避免重复点击。
const busyKey = ref('') // 形如 "claude|global|0",空表示无操作中

function busyKeyFor(toolID, scope, projectID) {
  return `${toolID}|${scope}|${projectID || 0}`
}

function isBusy(toolID, scope, projectID) {
  return busyKey.value === busyKeyFor(toolID, scope, projectID)
}

// 工具 chip 行:点击行为 — 切换"选中工具"(单选)
// 2026-06-25 改:不再触发批量启用/停用,仅做"工具选择器";后续作用域 chip 的
// 启用/停用都基于 selectedToolID 做单条操作。
// 2026-06-25 二改:切到某工具时,调一次 getSkillScopeStatus 完整重拉,
// 把该工具在所有 (全局 + 各项目) 路径的 SKILL.md 存在状态同步到 UI;
// 这样用户从外部 cp 文件后,选工具就能立刻看到状态变化。
async function handleToolChipClick(toolSummary) {
  // 单选切换:同一工具再点 = 取消;不同工具 = 切换
  if (selectedToolID.value === toolSummary.tool_id) {
    selectedToolID.value = null
    return
  }
  selectedToolID.value = toolSummary.tool_id
  // 同步重拉 scopeStatus,把磁盘最新状态反映到 scopeHits
  // 全量重扫后,selectedToolHitExists(tg) 会基于新数据重新计算 chip 态
  // 用 silent:不切 scopeLoading,保留旧 chip 视觉,只静默更新 scopeHits
  syncingToolID.value = toolSummary.tool_id
  try {
    await loadScopeStatus({ silent: true })
  } finally {
    syncingToolID.value = null
  }
}

// 作用域 chip 行:点击行为 — 仅对 selectedToolID 做单条启用/停用
// 2026-06-25 改:从"全工具批量"改为"对当前选中工具做单条操作"。
// - 未选工具:直接 return(模板已 disabled,这里再做防御)
// - 选中工具在该 (scope, project) 下不存在命中 → 启用
// - 选中工具在该 (scope, project) 下已存在命中 → 停用
// doApplyOne / doUnapplyOne 内部已经包含 loadScopeStatus + toast + flash,
//
// 这里不再重复刷新。
async function handleScopeChipClick(target) {
  if (!current.value) return
  if (!selectedToolID.value) {
    // 2026-06-25 改:不再 return,给用户一个明确提示,告诉他要点上面工具行。
    toast.info(t('skills.list.scopeSelectToolFirst'))
    return
  }
  const targetTool = selectedToolID.value
  const targetHit = target.hits.find((h) => h.tool_id === targetTool)
  const toolLabel = toolDisplay.value[targetTool] || targetTool
  if (targetHit && targetHit.exists) {
    // 已生效 → 停用单条
    const ok = await openConfirm({
      title: t('skills.list.unapplyConfirmTitle'),
      message: t('skills.list.unapplyConfirmMessage', {
        name: current.value.name,
        tool: toolLabel,
        scope: target.project_label,
      }),
      confirmText: t('common.delete'),
      variant: 'danger',
    })
    if (!ok) return
    await doUnapplyOne(targetHit)
    return
  }
  // 未生效 → 启用单条
  // 若后端未返回该 (scope, project) 的占位记录(从未写入过),需要构造一条
  // 不存在的 hit 用于 doApplyOne
  const fakeHit = targetHit || {
    tool_id: targetTool,
    scope: target.scope,
    project_id: target.project_id || 0,
    exists: false,
  }
  const ok = await openConfirm({
    title: t('skills.list.applyConfirmTitle'),
    message: t('skills.list.applyConfirmMessage', {
      name: current.value.name,
      tool: toolLabel,
      scope: target.project_label,
    }),
    confirmText: t('common.confirm'),
  })
  if (!ok) return
  await doApplyOne(fakeHit)
}

// doApplyOne 启用单个 (tool, scope, project) 组合。
//
// 后端是 cskillapply.ApplySkill:入参 { scope, project_id, name, tools: [toolID] },
// 同名已存在时由 skillapp 内部走 PreSnapshot + 原子覆盖,所以前端不用单独弹覆盖确认。
//
// 2026-06-25 改:成功后弹 toast + 闪 chip,失败弹 error toast。
// 顺序:先 await apply → await loadScopeStatus 刷新磁盘状态 → 再 toast + flash,
// 这样 toast/flash 出现时 chip 已经处于"已生效"选中态,语义对齐。
async function doApplyOne(h) {
  busyKey.value = busyKeyFor(h.tool_id, h.scope, h.project_id)
  // 2026-06-25 二改:锁住 apply 时的 skill 元数据,避免 await 期间用户切到别的 skill
  // 导致后续 patch 找错列表项。
  const targetSkill = current.value ? { ...current.value } : null
  try {
    await applySkill({
      name: targetSkill.name,
      scope: h.scope,
      project_id: h.project_id || 0,
      tools: [h.tool_id],
    })
    await loadScopeStatus()
    // 2026-06-25 三改:用确定性增量 patch(锁定 targetSkill + 显式 toolId/scope/op),
    // 不依赖 scopeHits 推算,避免 await 期间 current 变化或 scopeHits 被污染。
    patchAppliedTools(targetSkill, h.tool_id, h.scope, 'add')
    const targetKey = h.scope === 'global' ? 'global' : `p:${h.project_id}`
    flashTarget(targetKey)
    const toolLabel = toolDisplay.value[h.tool_id] || h.tool_id
    toast.success(t('skills.list.applySuccess', {
      path: `${toolLabel} · ${h.scope === 'global' ? t('skills.list.scopeGlobalChip') : `#${h.project_id}`}`,
    }))
  } catch (e) {
    toast.error(t('skills.list.applyFailed', { msg: e?.message || String(e) }))
    scopeError.value = t('skills.list.applyFailed', { msg: e?.message || String(e) })
  } finally {
    busyKey.value = ''
  }
}

// doUnapplyOne 停用单个 (tool, scope, project) 组合。
//
// 后端用 skillapp 的 apply/undo 机制(走 PreSnapshot 还原或删目标文件),
// 但 undo 是按 apply_id 撤销,所以前端先 listApplies 找最近一条未撤销的 apply_id。
// 没找到就报错(用户应该是从外部把目录删了,不走 skillbox undo)。
//
// 2026-06-25 改:成功/失败都用 toast 反馈。toast/flash 在 loadScopeStatus 之后,
//
// 保证 flash 那 2s 内 chip 已经是"已停用"态(从 chip-active → chip-muted)。
async function doUnapplyOne(h) {
  busyKey.value = busyKeyFor(h.tool_id, h.scope, h.project_id)
  // 2026-06-25 二改:锁住 undo 时的 skill 元数据(同上,避免 await 期间 current 被切走)
  const targetSkill = current.value ? { ...current.value } : null
  try {
    const list = await listApplies({
      scope: h.scope,
      name: targetSkill.name,
      tool: h.tool_id,
      status: 'applied',
      page: 1,
      size: 1, // 找最近一条即可
    })
    const last = list?.items?.[0]
    if (!last) {
      // 2026-06-25 增:DB 没记录(用户手动 cp / 外部安装,scope-status 命中但
      // skill_applies 表里没行)→ 走 force-undo 走 scope-status 删磁盘 +
      // 插占位 rolled_back 行。project_id 这里 h 可能没带(0),后端会从
      // h.scope=project 时从项目列表里按 id 定位,project_id=0 + scope=project
      // 找不到会返 404。
      await forceUndoApply({
        scope: h.scope,
        project_id: h.project_id || 0,
        name: targetSkill.name,
        tool: h.tool_id,
      })
      await loadScopeStatus()
      patchAppliedTools(targetSkill, h.tool_id, h.scope, 'remove')
      const targetKey = h.scope === 'global' ? 'global' : `p:${h.project_id}`
      flashTarget(targetKey)
      const toolLabel = toolDisplay.value[h.tool_id] || h.tool_id
      toast.success(t('skills.list.unapplySuccess', {
        path: `${toolLabel} · ${h.scope === 'global' ? t('skills.list.scopeGlobalChip') : `#${h.project_id}`}`,
      }))
      return
    }
    await undoApply({ apply_id: last.id })
    // 注:后端 SkillApply entity 主键 json tag 是 "id",不是 "apply_id"
    await loadScopeStatus()
    // 2026-06-25 三改:用确定性增量 patch(锁定 targetSkill + 显式 toolId/scope/op)
    patchAppliedTools(targetSkill, h.tool_id, h.scope, 'remove')
    const targetKey = h.scope === 'global' ? 'global' : `p:${h.project_id}`
    flashTarget(targetKey)
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

// AI 侧栏
const aiOpen = ref(false)
function toggleAI() { aiOpen.value = !aiOpen.value }

// 2026-06-25 二改:skillKey 改为只取 name(后端 listSkills 不返回 scope/project_id,
// 之前用 scope|project_id|name|version 会因为 scope/project_id 都是 undefined,
// 所有 item 的 key 都一样,导致 findIndex 总是命中 idx=0,splice 时把第一行
// 列表项错误覆盖)。
// store layout 是 <StoreRoot>/<name>/SKILL.md,name 在 storeRoot 里唯一,version 只
// 是 SKILL.md frontmatter metadata,不影响列表项定位。
function skillKey(p) {
  if (!p) return ''
  return p.name || ''
}

// 2026-06-25 增 + 二改:apply/unapply 成功后,用锁定的 toolId + scope 局部更新
// items 里指定 skill 的 applied_tools 字段。
//
// 设计: 不依赖 scopeHits 推算(避免 await 期间 scopeHits 被 loadCurrent 切走
// 污染后,推算成别的 skill 的 applied_tools 写到目标列表项)。改用确定性增量:
//   - apply  global: add toolId 到 applied_tools(去重)
//   - apply  非 global: 不动
//   - unapply global: remove toolId
//   - unapply 非 global: 不动
// 2026-06-25 三改:接受显式 targetSkill / toolId / scope 参数(由 caller 锁定),
// 避免在 await 期间 current 变化导致找错列表项。
function patchAppliedTools(targetSkill, toolId, scope, op) {
  if (!targetSkill || !toolId || scope !== 'global') return // 只 global 改列表项
  const idx = items.value.findIndex((x) => skillKey(x) === skillKey(targetSkill))
  if (idx < 0) return
  const cur = items.value[idx]
  const curSet = new Set(cur.applied_tools || [])
  if (op === 'add') curSet.add(toolId)
  else if (op === 'remove') curSet.delete(toolId)
  items.value.splice(idx, 1, { ...cur, applied_tools: Array.from(curSet) })
}

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
  error.value = ''
  try {
    const resp = await listSkills({
      keyword: keyword.value || undefined,
      page: page.value,
      size,
    })
    items.value = resp?.items || []
    total.value = resp?.total || 0
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
  // 2026-06-25 二改:去掉切走前的 syncLocalAppliedTools。
  // 旧版用旧 scopeHits 推算旧 current 的 applied_tools,但旧 scopeHits 可能是 []
  // (loadScopeStatus 还没跑完),会把列表项清空。改用"只在 mutation 路径 sync"。
  // 2026-06-25:切 skill 时清掉"工具选中"和 scope 状态,避免把旧 skill 的选择带过来
  selectedToolID.value = null
  scopeHits.value = []
  scopeTools.value = []
  scopeProjects.value = []
  scopeError.value = ''
  try {
    const full = await getSkill({
      scope: row.scope,
      project_id: row.project_id,
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
    current.value = { ...row, _full: full }
    // 同步拉一次 tag 列表,让详情区"标签"chip 有数据
    try {
      const out = await listTags({ skill_id: row.id })
      currentTagList.value = out?.items || []
    } catch (_) { currentTagList.value = [] }
    // 拉 scope 实时状态(工具/作用域两级展示)
    await loadScopeStatus()
    // 2026-06-25 二改:去掉切到后的 syncLocalAppliedTools(同上,新 scopeHits 推算的
    // 新 current 列表项值和后端 reload 时注入的值应一致;切到新 skill 也会在
    // 该 skill 上的 doApplyOne/doUnapplyOne 路径 sync,不会漏同步)。
  } catch (e) {
    currentError.value = e?.message || String(e)
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

// 选中列表项
function selectItem(row) {
  // 切换 skill 时清掉内联编辑态,避免把旧 skill 的 editBody 带到新 skill
  if (editing.value) cancelInlineEdit()
  selectedKey.value = skillKey(row)
  loadCurrent(row)
}

// 监听选中 key 变化(支持按 Enter 在搜索结果中跳转)
watch(selectedKey, (k) => {
  if (!k) return
  const row = items.value.find((x) => skillKey(x) === k)
  if (row) loadCurrent(row)
})

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
const renderedHtml = computed(() => renderMarkdown(currentBody.value))

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
async function copySourcePath() {
  if (!current.value) return
  const sp = current.value._full?.canonical?.source_path
    || current.value._full?.source_path
    || ''
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
      const failList = []
      for (const tid of draft.applyTools) {
        try {
          await applySkill({
            name: draft.name,
            scope: draft.scope,
            project_id: draft.project_id || 0,
            tools: [tid],
          })
        } catch (e) { failList.push({ tool: tid, msg: e?.message || String(e) }) }
      }
      if (failList.length) {
        toast.error(t('skills.editor.applyPartialFailed', {
          ok: draft.applyTools.length - failList.length,
          total: draft.applyTools.length,
          fails: failList.map((f) => f.tool).join(', '),
        }))
      } else {
        toast.success(t('skills.editor.applyAllSuccess', { n: draft.applyTools.length }))
      }
    }
    editorOpen.value = false
    await reload()
    // 选回刚保存的
    const row = items.value.find((x) => x.name === payload.name && x.version === payload.version)
    if (row) selectItem(row)
  } catch (e) { error.value = e?.message || String(e) }
}

// ====== 删除 ======
async function removeCurrent() {
  if (!current.value) return
  const row = current.value
  const ok = await openConfirm({
    title: t('common.delete'),
    message: t('skills.list.confirmDelete', { name: row.name, version: row.version }),
    variant: 'danger',
    confirmText: t('common.delete'),
  })
  if (!ok) return
  try {
    await deleteSkill({ scope: row.scope, project_id: row.project_id, name: row.name, version: row.version })
    if (editing.value) cancelInlineEdit()
    current.value = null
    selectedKey.value = null
    await reload()
  } catch (e) { error.value = e?.message || String(e) }
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

onMounted(() => {
  reload()
})
</script>

<template>
  <div class="skills-layout">
    <!-- 左侧:技能列表 -->
    <aside class="skills-pane">
      <!-- 顶部操作栏 -->
      <div class="left-topbar">
        <button class="left-action" :title="t('skills.list.btnNewSkillTitle')" @click="startNew">
          <Icon icon="mdi:plus" width="16" height="16" />
          <span>{{ t('skills.list.btnNewSkill') }}</span>
        </button>
        <button class="left-action" :title="t('skills.list.btnImportSkillTitle')" @click="goOnboarding">
          <Icon icon="mdi:tray-arrow-down" width="16" height="16" />
          <span>{{ t('skills.list.btnImportSkill') }}</span>
        </button>
      </div>

      <!-- 搜索框 -->
      <div class="left-search">
        <Icon icon="mdi:magnify" width="14" height="14" class="search-icon" />
        <input
          v-model="keyword"
          :placeholder="t('skills.searchPlaceholder')"
          class="search-input"
          :title="t('skills.list.searchTitle')"
          @keyup.enter="onSearchEnter"
        />
      </div>

      <p v-if="error" class="left-error">
        <Icon icon="mdi:alert-circle-outline" width="12" height="12" />
        {{ error }}
      </p>

      <!-- 列表 -->
      <ul class="skill-list" role="listbox" :aria-label="t('skills.title')">
        <li
          v-for="(p, i) in filteredItems"
          :key="p.name"
          :ref="(el) => { if (el) listRefs[i] = el }"
          tabindex="0"
          role="option"
          :aria-selected="selectedKey === skillKey(p)"
          :class="['skill-item', { 'skill-item-active': selectedKey === skillKey(p) }]"
          @click="selectItem(p)"
          @keyup.enter="selectItem(p)"
        >
          <span class="skill-item-bar"></span>
          <div class="skill-item-main">
            <div class="skill-item-head">
              <span class="skill-item-name">{{ p.name }}</span>
              <span class="skill-item-version">@{{ p.version }}</span>
            </div>
            <div class="skill-item-meta">
              <!-- 2026-06-25 改:LOCAL 标签换成"已全局应用工具"列表(来自 list 接口的 applied_tools) -->
              <span
                v-for="tid in (p.applied_tools || [])"
                :key="tid"
                class="skill-item-tool-chip"
                :title="t('skills.list.appliedGlobal', { tool: toolDisplay[tid] || tid })"
              >
                <Icon :icon="toolIcon(tid)" width="11" height="11" />
                <span>{{ toolShort(tid) }}</span>
              </span>
            </div>
          </div>
        </li>
      </ul>

      <div v-if="!loading && !filteredItems.length" class="skill-list-empty">
        <Icon icon="mdi:book-open-variant" width="28" height="28" />
        <p>{{ t('skills.list.emptyTitle') }}</p>
        <p class="hint">{{ t('skills.list.emptyHint') }}</p>
      </div>

      <div v-if="loading" class="skill-list-loading">
        <span class="spinner"></span>
        <span>{{ t('common.processing') }}</span>
      </div>

      <!-- 翻页 -->
      <footer v-if="totalPages > 1" class="left-pager">
        <button :disabled="page <= 1" @click="gotoPage(page - 1)">
          <Icon icon="mdi:chevron-left" width="12" height="12" />
          {{ t('common.prev') }}
        </button>
        <span>{{ page }} / {{ totalPages }}</span>
        <button :disabled="page >= totalPages" @click="gotoPage(page + 1)">
          {{ t('common.next') }}
          <Icon icon="mdi:chevron-right" width="12" height="12" />
        </button>
      </footer>
    </aside>

    <!-- 右侧:技能详情 -->
    <section class="detail-pane">
      <!-- 空状态 -->
      <div v-if="!current" class="detail-empty">
        <Icon icon="mdi:cursor-default-click-outline" width="40" height="40" />
        <p class="empty-title">{{ t('skills.list.selectToView') }}</p>
      </div>

      <template v-else>
        <!-- 顶部 toolbar -->
        <header class="detail-toolbar">
          <div class="detail-title-block">
            <div class="detail-title-row">
              <h1 class="detail-name">{{ current.name }}</h1>
              <code
                class="detail-version"
                role="button"
                tabindex="0"
                :title="t('skills.tag.titlePrefix')"
                @click="openTagDialog"
                @keyup.enter="openTagDialog"
              >@{{ current.version }}</code>
              <span :class="['badge', current.source === 'market' ? 'blue' : 'gray']">{{ current.source || 'local' }}</span>
              <!-- 2026-06-25 改:编辑按钮从 detail-body 顶部搬到 detail-title-row 右侧(在 LOCAL 徽章之后)
                   2026-06-26 二改:编辑态时,同位置显示"取消/保存"实心按钮(替换 ghost-link) -->
              <div class="detail-title-actions">
                <button
                  v-if="!editing"
                  class="ghost-link"
                  :title="t('common.edit')"
                  @click="startInlineEdit"
                >
                  <Icon icon="mdi:pencil" width="12" height="12" />
                  {{ t('common.edit') }}
                </button>
                <!-- 编辑态:同位置显示 取消 / 保存 实心按钮 -->
                <template v-else>
                  <button
                    class="title-action-btn title-action-cancel"
                    :disabled="editSaving"
                    @click="cancelInlineEdit"
                  >
                    <Icon icon="mdi:close" width="13" height="13" />
                    {{ t('common.cancel') }}
                  </button>
                  <button
                    class="title-action-btn title-action-save"
                    :disabled="editSaving"
                    @click="saveInlineEdit"
                  >
                    <span v-if="editSaving" class="spinner spinner-sm"></span>
                    <Icon v-else icon="mdi:content-save" width="13" height="13" />
                    {{ editSaving ? t('common.processing') : t('common.save') }}
                  </button>
                </template>
              </div>
            </div>

            <!-- 2026-06-25 改:description 下方接触发词行内展示(查看态保留在 title-block 紧凑展示) -->
            <p v-if="!editing && currentMeta.description" class="detail-desc">{{ currentMeta.description }}</p>

            <!-- 2026-06-25 改:触发词行内展示,在 description 下方(查看态) -->
            <div v-if="!editing && (currentMeta.triggers || []).length" class="detail-triggers-row">
              <span class="triggers-label">{{ t('skills.editor.triggers') }}</span>
              <span class="meta-text">{{ (currentMeta.triggers || []).join('、') }}</span>
            </div>
          </div>

          <div class="detail-actions">
            <!-- 2026-06-26 改:把原生 :title 换成 :data-tip,触发全局 CSS 自定义 tooltip(0 延迟),
                 否则浏览器原生 title 会有 ~1s 延迟,体感很慢 -->
            <button
              class="icon-btn"
              :data-tip="t('skills.list.tooltipTest')"
              :aria-label="t('skills.list.tooltipTest')"
              :disabled="testing"
              @click="triggerTest"
            >
              <span v-if="testing" class="spinner spinner-sm"></span>
              <Icon v-else icon="mdi:test-tube" width="16" height="16" />
            </button>
            <button
              class="icon-btn"
              :data-tip="t('skills.list.tooltipTag')"
              :aria-label="t('skills.list.tooltipTag')"
              @click="openTagDialog"
            >
              <Icon icon="mdi:tag-outline" width="16" height="16" />
            </button>
            <button
              class="icon-btn"
              :data-tip="t('skills.list.tooltipOpenFolder')"
              :aria-label="t('skills.list.tooltipOpenFolder')"
              @click="openInFolder"
            >
              <Icon icon="mdi:folder-outline" width="16" height="16" />
            </button>
            <button
              class="icon-btn"
              :data-tip="t('skills.list.copyPath')"
              :aria-label="t('skills.list.copyPath')"
              @click="copySourcePath"
            >
              <Icon icon="mdi:content-copy" width="16" height="16" />
            </button>
            <button
              class="icon-btn"
              :data-tip="t('skills.list.tooltipDelete')"
              :aria-label="t('skills.list.tooltipDelete')"
              @click="removeCurrent"
            >
              <Icon icon="mdi:delete" width="16" height="16" />
            </button>
            <button
              class="icon-btn ai-btn"
              :data-tip="aiOpen ? t('skills.btnAiClose') : t('skills.btnAiOpen')"
              :aria-label="aiOpen ? t('skills.btnAiClose') : t('skills.btnAiOpen')"
              @click="toggleAI"
            >
              <Icon :icon="aiOpen ? 'mdi:robot' : 'mdi:robot-outline'" width="16" height="16" />
            </button>
          </div>
        </header>

        <!-- 2026-06-26 新增:编辑态的描述/触发词 编辑区移到 toolbar 外,
             变成 detail-pane 下的独立 section,跟其他 detail-section 一样占满整页宽度
             (放在 toolbar 内会被 detail-actions(右侧 6 个图标按钮)挤掉 35% 宽度) -->
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
            <Icon icon="mdi:alert-circle-outline" width="12" height="12" />
            {{ editError }}
          </p>
        </section>

        <p v-if="openError" class="message message-error">
          <Icon icon="mdi:alert-circle-outline" width="12" height="12" />
          {{ openError }}
        </p>

        <!-- scope 两级(2026-06-24 改:只读,展示实时扫描结果;2026-06-26 改:编辑态隐藏) -->
        <section v-if="!editing" class="detail-section">
          <header class="section-header">
            <h3>
              <Icon icon="mdi:earth" width="14" height="14" />
              {{ t('skills.list.scopeLabel') }}
            </h3>
            <span v-if="!scopeLoading && scopeHits.length" class="muted small-hint">
              {{ t('skills.list.scopeHitCount', { n: scopeHits.filter((h) => h.exists).length }) }}
            </span>
          </header>

          <p v-if="scopeLoading" class="section-loading">
            <span class="spinner spinner-sm"></span>
            <span class="muted">…</span>
          </p>
          <p v-else-if="scopeError" class="message message-error">
            <Icon icon="mdi:alert-circle-outline" width="12" height="12" />
            {{ scopeError }}
          </p>

          <template v-else>
            <!-- 第一行:工具(5 个)— 单选切换器(2026-06-25 改)
                 视觉态:
                   - 命中(主色填充) = 该工具有生效记录
                   - 选中(蓝色边框) = 用户当前正在为这个工具选作用域
                   - 命中 + 选中 = 主色填充 + 蓝色加粗边框
                   - 未命中 + 未选中 = 虚线 muted -->
            <div class="scope-row">
              <span class="scope-row-label">{{ t('skills.list.scopeToolsRow') }}</span>
              <div class="chip-row">
                <button
                  v-for="t in scopeToolSummary"
                  :key="t.tool_id"
                  type="button"
                  :class="[
                    'chip', 'chip-tool',
                    t.hasHit ? 'chip-active' : 'chip-muted',
                    selectedToolID === t.tool_id ? 'chip-tool-selected' : '',
                    syncingToolID === t.tool_id ? 'chip-tool-syncing' : '',
                  ]"
                  :title="t.hasHit
                    ? `${t.display}: ${t.hitCount} 处生效`
                    : `${t.display}: 0 处生效`"
                  @click="handleToolChipClick(t)"
                >
                  <span
                    v-if="syncingToolID === t.tool_id"
                    class="spinner spinner-sm chip-spinner"
                  ></span>
                  <Icon v-else :icon="t.icon" width="12" height="12" />
                  <span>{{ toolShort(t.tool_id) }}</span>
                  <span v-if="t.hitCount > 0" class="chip-count">{{ t.hitCount }}</span>
                </button>
                <span v-if="selectedToolID" class="chip-tool-selected-hint muted">
                  {{ t('skills.list.scopeToolSelected', { tool: toolDisplay[selectedToolID] || selectedToolID }) }}
                </span>
              </div>
            </div>

            <!-- 第二行:作用域(全局 + 各项目)— 仅当选中工具后才显示(2026-06-26 改)
                 视觉态:
                   - 选中工具在该 chip 内已生效 → 蓝色 active
                   - 选中工具在该 chip 内未生效 → muted(虚线) -->
            <div v-if="selectedToolID" class="scope-row">
              <span class="scope-row-label">{{ t('skills.list.scopeTargetsRow') }}</span>
              <div class="chip-row">
                <button
                  v-for="tg in scopeTargets"
                  :key="tg.key"
                  type="button"
                  :class="[
                    'chip', 'chip-scope-target',
                    selectedToolHitExists(tg) ? 'chip-active' : 'chip-muted',
                    !selectedToolID ? 'chip-target-no-tool' : '',
                    selectedToolBusy(tg) ? 'chip-busy' : '',
                    flashTargetKey === tg.key ? 'chip-flash' : '',
                  ]"
                  :title="selectedToolHitExists(tg) ? t('skills.list.unapplyConfirmTitle') : t('skills.list.applyConfirmTitle')"
                  @click="handleScopeChipClick(tg)"
                >
                  <span
                    v-if="selectedToolBusy(tg)"
                    class="spinner spinner-sm chip-spinner"
                  ></span>
                  <Icon
                    v-else
                    :icon="tg.scope === 'global' ? 'mdi:earth' : 'mdi:folder-outline'"
                    width="12"
                    height="12"
                  />
                  <span>{{ tg.project_label }}</span>
                  <span v-if="selectedToolHitExists(tg)" class="chip-mini-list">
                    <Icon
                      :icon="toolIcon(selectedToolID)"
                      width="10"
                      height="10"
                      class="chip-mini-icon"
                    />
                  </span>
                </button>
                <span v-if="!scopeTargets.length" class="chip-empty muted">
                  {{ t('skills.list.scopeEmpty') }}
                </span>
              </div>
            </div>
          </template>
        </section>

        <!-- 标签 section:2026-06-25 改 — 不再展示 chip 列表,改为点击版本号弹出标签弹窗。
             section 本身保留,只显示一行说明 + "管理"按钮占位也行,但用户只要求"不打 tag 部分显示"。
             直接整段删掉,标签入口只剩顶栏的 tag-outline 按钮和 detail-version 点击。 -->

        <!-- 2026-06-25 改:触发词已搬到 description 下方行内展示,触发词 + 更新时间 独立 section 删除。
             更新时间挪到 detail-toolbar 标题行右侧,作为次要信息展示。 -->

        <!-- 正文 -->
        <section class="detail-section detail-body">
          <header class="section-header">
            <h3>
              <Icon :icon="editing ? 'mdi:pencil-box-outline' : 'mdi:text-box-outline'" width="14" height="14" />
              {{ editing ? t('skills.list.bodyEditing') : t('skills.list.bodyTitle') }}
            </h3>
            <!-- 2026-06-26 改:"取消/保存"已搬到 detail-title-row 右侧(替换"编辑"按钮位置),
                 这里不再放操作按钮,保持 section-header 干净 -->
          </header>

          <p v-if="editError" class="message message-error">
            <Icon icon="mdi:alert-circle-outline" width="12" height="12" />
            {{ editError }}
          </p>

          <!-- 编辑态:内联 textarea(Markdown 原文) -->
          <textarea
            v-if="editing"
            v-model="editBody"
            class="md-editor"
            spellcheck="false"
            :placeholder="t('skills.list.bodyEmpty')"
          ></textarea>

          <!-- 查看态:渲染 -->
          <template v-else>
            <div v-if="currentLoading" class="detail-loading">
              <span class="spinner"></span>
              <span>{{ t('common.processing') }}</span>
            </div>
            <p v-else-if="currentError" class="message message-error">
              <Icon icon="mdi:alert-circle-outline" width="12" height="12" />
              {{ currentError }}
            </p>
            <div v-else-if="currentBody" class="md-body" v-html="renderedHtml"></div>
            <p v-else class="section-empty">{{ t('skills.list.bodyEmpty') }}</p>
          </template>
        </section>
      </template>
    </section>

    <!-- AI 侧栏 -->
    <AIPanel v-if="aiOpen" :context-text="currentSkillMd" @apply="onAIApply" />

    <!-- Tag 弹窗 -->
    <Modal
      v-model="tagOpen"
      size="xl"
      :title="current ? t('skills.tag.titlePrefix') + ' — ' + current.name + '@' + current.version : t('skills.tag.titlePrefix')"
    >
      <template #title-icon>
        <Icon icon="mdi:tag-outline" width="18" height="18" />
      </template>

      <p v-if="tagMessage" class="message message-success">
        <Icon icon="mdi:check-circle-outline" width="14" height="14" />
        {{ tagMessage }}
      </p>
      <p v-if="tagError" class="message message-error">
        <Icon icon="mdi:alert-circle-outline" width="14" height="14" />
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
        <Icon icon="mdi:arrow-right" width="14" height="14" class="diff-arrow" />
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
        <Icon icon="mdi:tag-off-outline" width="36" height="36" />
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
        <Icon icon="mdi:test-tube" width="18" height="18" />
      </template>

      <div :class="['test-status-row', `test-status-${lastTest?.run?.status || 'errored'}`]">
        <span v-if="lastTest?.run" class="test-status-badge">{{ lastTest.run.status }}</span>
        <p v-if="testError" class="message message-error" style="margin: 0">
          <Icon icon="mdi:alert-circle-outline" width="14" height="14" />
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
        <Icon :icon="editingKey ? 'mdi:pencil' : 'mdi:plus'" width="18" height="18" />
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
                <Icon icon="mdi:earth" width="13" height="13" />
                {{ t('skills.editor.scopeGlobal') }}
              </button>
              <button
                type="button"
                :class="['seg-btn', draft.scope === 'project' ? 'seg-active' : '']"
                :disabled="!!editingKey"
                @click="draft.scope = 'project'"
              >
                <Icon icon="mdi:folder-outline" width="13" height="13" />
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
              <Icon :icon="toolIcon(tool.tool_id)" width="12" height="12" />
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
          <textarea v-model="draft.body" rows="14" class="code"></textarea>
        </div>

        <p v-if="error" class="message message-error" style="margin: 0 0 12px">
          <Icon icon="mdi:alert-circle-outline" width="14" height="14" />
          {{ error }}
        </p>
      </form>
      <template #footer>
        <button type="button" class="ghost" @click="editorOpen = false">
          <Icon icon="mdi:close" width="14" height="14" />
          {{ t('common.cancel') }}
        </button>
        <button type="button" class="primary" @click="submit">
          <Icon :icon="editingKey ? 'mdi:content-save' : 'mdi:plus'" width="14" height="14" />
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

.skill-list-empty,
.skill-list-loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 32px 16px;
  color: var(--text-faint);
  text-align: center;
}

.skill-list-empty .hint {
  font-size: 12px;
  color: var(--text-faint);
  margin: 0;
}

.skill-list-loading {
  flex-direction: row;
  font-size: 12px;
  color: var(--text-dim);
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
  position: sticky;
  top: 0;
  z-index: 5;
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding: 16px 20px;
  background: var(--bg-card);
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}

.detail-title-block {
  min-width: 0;
  flex: 1;
}

.detail-title-row {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  min-width: 0;
}

/* 2026-06-25 新增:标题行右侧"编辑"按钮容器
   2026-06-26 改:编辑态时,容器内显示"取消/保存"实心按钮(替换 ghost-link) */
.detail-title-actions {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin-left: auto; /* 顶到最右 */
}

/* 2026-06-26 新增:标题行右侧的"取消/保存"实心按钮 */
.title-action-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 28px;
  padding: 0 10px;
  font-size: 12px;
  font-weight: 500;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all 0.12s ease;
  border: 1px solid transparent;
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

.detail-name {
  font-size: 18px;
  font-weight: 700;
  color: var(--text);
  margin: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 100%;
}

.detail-version {
  font-size: 12px;
  color: var(--text-dim);
  font-family: 'JetBrains Mono', monospace;
  background: var(--primary-dim);
  padding: 2px 6px;
  border-radius: 4px;
  cursor: pointer;
  transition: background-color 0.12s ease, color 0.12s ease;
}
/* 2026-06-25 改:点击版本号弹出 tag 弹窗,所以 .detail-version 视觉上是可点的 */
.detail-version:hover {
  background: var(--primary);
  color: var(--bg-card);
}
.detail-version:focus-visible {
  outline: 2px solid var(--accent-blue);
  outline-offset: 1px;
}

.detail-desc {
  margin: 6px 0 0;
  font-size: 13px;
  color: var(--text-dim);
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

/* 2026-06-25 新增:编辑态的 description 编辑框
   2026-06-26 改:边框变细(默认 1px 淡边 + 焦点 1px 实线 + 主色 box-shadow,去掉 3px 厚光晕),
   不固定 width,跟父容器(.detail-title-block,flex:1)自适应占满整页宽度 */
.desc-editor {
  margin: 6px 0 0;
  display: block;
  width: 100%;
  /* 2026-06-26 改:精确锁 2 行高(13 × 1.5 × 2 + 16 padding + 2 border = 57),
     与 rows="2" 对齐;min-height 留作下限保护,内容多了自动扩 */
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

/* 2026-06-26 新增:编辑态的描述/触发词 编辑区独立 section
   跟其他 detail-section 一样 padding,内部 editor-field-full 占满 */
.detail-edit-fields {
  display: flex;
  flex-direction: column;
  gap: 12px;
  background: var(--bg-subtle); /* 跟 toolbar 区分,提示"这块在编辑" */
}
.detail-edit-fields .editor-field-full {
  gap: 6px;
}

/* 2026-06-25 新增:触发词行内展示 — 在 description 下方 */
.detail-triggers-row {
  margin: 8px 0 0;
  display: flex;
  align-items: flex-start;
  gap: 8px;
  font-size: 13px;
  line-height: 1.5;
  color: var(--text);
}
.detail-triggers-row.editing {
  align-items: flex-start;
  flex-direction: column;
  gap: 4px;
}
.triggers-label {
  flex-shrink: 0;
  font-size: 11px;
  color: var(--text-faint);
  text-transform: uppercase;
  letter-spacing: 0.3px;
  padding-top: 3px;
}

.detail-actions {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}

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

/* 让 danger hover 提示删除样式 - 用 :nth-last-child 单独标红 */
.detail-actions .icon-btn[aria-label="删除"]:hover:not(:disabled) {
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
   Scope 两级布局(2026-06-24)
   - 第一行:工具(5 个)— 命中用 chip-active,未命中用 chip-muted
   - 第二行:作用域(全局/各项目)— chip-active 标志有命中,
     chip-mini-list 内显示命中工具的 mdi 图标
   ============================================ */
.scope-row {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  margin-bottom: 8px;
}
.scope-row:last-child { margin-bottom: 0; }
.scope-row-label {
  flex: 0 0 64px;
  font-size: 12px;
  color: var(--text-faint);
  font-weight: 500;
  padding-top: 6px;
  letter-spacing: 0.3px;
}

/* 工具行 chip:active 用主色(黑),未命中用 muted(2026-06-25 改:不降透明度,正常显示就好) */
.chip-tool {
  cursor: pointer;
  background: var(--bg-card);
  color: var(--text);
  border-color: var(--border);
  border-style: dashed; /* 未命中虚线边框,有命中时 active 覆盖回 solid */
  font-family: inherit;
  position: relative;
}
.chip-tool.chip-muted:hover { background: var(--bg-hover); color: var(--text); }
/* 2026-06-26 改:工具行 chip 命中态改用蓝色(var(--accent-blue)),不再用黑色,
   与作用域行 chip-active 保持一致,整体配色更协调 */
.chip-tool.chip-active {
  background: var(--accent-blue);
  color: #fff;
  border-color: var(--accent-blue);
  border-style: solid;
}
.chip-tool.chip-active:hover { background: var(--accent-blue); color: #fff; }

/* 2026-06-26 改:工具 chip 命中 + 选中叠加态 — 蓝色背景 + 蓝色边框(与作用域行一致) */
.chip-tool.chip-active.chip-tool-selected {
  background: var(--accent-blue);
  color: #fff;
  border-color: var(--accent-blue);
}

/* 2026-06-25 新增:工具 chip "已选中"(单选切换器)态
   - 2026-06-26 改:用主色(黑)边框区分"操作选中",不再用蓝色,
     避免与"已启用"色(蓝填充)混淆,用户一眼能区分"我正在为哪个工具操作"
   - 2026-06-26 二改:边框从 2px+1px shadow 收成 1px(双层用 box-shadow 模拟),
     不再显粗;颜色从黑改紫色(var(--accent-violet)),与已启用蓝 / 未选灰 三色清晰区分 */
.chip-tool.chip-tool-selected {
  border-color: var(--accent-violet);
  border-width: 1px;
  /* 1px 双层边框:外层 inset 1px,内层 outset 0.5px 累加视觉为 1.5px,
     避免 border-width 切换导致尺寸跳动 */
  border-style: solid;
  box-shadow: inset 0 0 0 1px var(--accent-violet);
}
.chip-tool.chip-tool-selected .chip-count {
  background: var(--accent-violet-bg);
  color: var(--accent-violet);
}
.chip-tool.chip-active.chip-tool-selected .chip-count {
  background: rgba(255, 255, 255, 0.18);
  color: #fff;
}

/* 工具行尾部提示:当前已选工具 */
.chip-tool-selected-hint {
  font-size: 11px;
  padding-left: 4px;
}

/* 2026-06-25 二改:工具 chip 正在同步磁盘(后端重拉 scopeStatus) */
.chip-tool-syncing {
  cursor: wait;
  opacity: 0.85;
}

.chip-count {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 16px;
  height: 16px;
  padding: 0 4px;
  margin-left: 2px;
  font-size: 10px;
  font-weight: 700;
  background: var(--bg-card);
  color: var(--text);
  border-radius: 8px;
}
.chip-tool.chip-active .chip-count {
  background: var(--bg-card);
  color: var(--text);
}

/* 作用域行 chip:active 蓝色,未命中 muted(2026-06-25 改:不降透明度,正常显示) */
.chip-scope-target {
  cursor: pointer;
  font-family: inherit;
}
.chip-scope-target.chip-muted {
  background: var(--bg-card);
  color: var(--text);
  border-color: var(--border);
  border-style: dashed;
}
.chip-scope-target.chip-muted:hover { background: var(--bg-hover); color: var(--text); }
.chip-scope-target.chip-active {
  background: var(--accent-blue-bg);
  color: var(--accent-blue);
  border-color: var(--accent-blue-border);
  border-style: solid;
}
.chip-scope-target.chip-active:hover {
  background: var(--accent-blue-bg);
  color: var(--accent-blue);
}

/* 2026-06-25 改:未选工具时生效位置 chip 不再 disabled,只是视觉稍弱以提示用户"先选工具" */
.chip-scope-target.chip-target-no-tool {
  opacity: 0.85;
}
.chip-scope-target.chip-target-no-tool:hover {
  background: var(--bg-hover);
  color: var(--text);
}

/* busy 状态 — 操作中,弱化视觉,显示 spinner */
.chip-busy {
  cursor: wait !important;
  opacity: 0.6 !important;
  pointer-events: none;
}
.chip-spinner {
  width: 10px;
  height: 10px;
  border-width: 1.5px;
}

/* 2026-06-25 新增:操作成功后的脉冲高亮,2s 内让用户眼睛锁定刚操作的 chip */
@keyframes chipFlash {
  0%   { box-shadow: 0 0 0 3px var(--accent-blue); transform: scale(1); }
  20%  { box-shadow: 0 0 0 4px var(--accent-blue); transform: scale(1.04); }
  100% { box-shadow: 0 0 0 0 transparent; transform: scale(1); }
}
.chip-flash {
  animation: chipFlash 1.6s ease-out;
}

.chip-mini-list {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  margin-left: 4px;
  padding-left: 6px;
  border-left: 1px solid var(--accent-blue-border);
}
.chip-mini-icon {
  color: var(--accent-blue);
  opacity: 0.85;
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

.detail-body {
  padding-bottom: 24px;
  /* 占满 .detail-pane 剩余高度,让 .md-editor 能 flex:1 自适应填满 */
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.md-body {
  font-size: 13.5px;
  line-height: 1.7;
  color: var(--text);
  word-wrap: break-word;
}

.md-body :deep(h1),
.md-body :deep(h2),
.md-body :deep(h3) {
  margin: 16px 0 8px;
  font-weight: 600;
  color: var(--text);
}
.md-body :deep(h1) { font-size: 18px; }
.md-body :deep(h2) { font-size: 16px; }
.md-body :deep(h3) { font-size: 14px; }

.md-body :deep(p) { margin: 8px 0; }
.md-body :deep(ul),
.md-body :deep(ol) { margin: 8px 0 8px 20px; padding: 0; }
.md-body :deep(li) { margin: 2px 0; }

.md-body :deep(code) {
  font-family: 'JetBrains Mono', monospace;
  font-size: 0.9em;
  background: var(--primary-dim);
  padding: 1px 5px;
  border-radius: 4px;
}

.md-body :deep(pre) {
  margin: 10px 0;
  padding: 12px 14px;
  background: var(--bg-subtle);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  overflow-x: auto;
  font-family: 'JetBrains Mono', monospace;
  font-size: 12.5px;
  line-height: 1.6;
}

.md-body :deep(pre code) {
  background: transparent;
  padding: 0;
  font-size: inherit;
}

.md-body :deep(a) {
  color: var(--accent-blue);
  text-decoration: underline;
  text-underline-offset: 2px;
}

.md-body :deep(blockquote) {
  margin: 8px 0;
  padding: 6px 12px;
  border-left: 3px solid var(--border);
  color: var(--text-dim);
  background: var(--bg-subtle);
  border-radius: 0 var(--radius-sm) var(--radius-sm) 0;
}

.md-body :deep(hr) {
  border: none;
  border-top: 1px solid var(--border);
  margin: 14px 0;
}

.md-body :deep(table) {
  border-collapse: collapse;
  margin: 10px 0;
  font-size: 12.5px;
}
.md-body :deep(th),
.md-body :deep(td) {
  border: 1px solid var(--border);
  padding: 6px 10px;
  text-align: left;
}
.md-body :deep(th) { background: var(--bg-subtle); font-weight: 600; }

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
.editor-field-full textarea { min-height: 100px; }

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
