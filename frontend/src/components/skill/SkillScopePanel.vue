<script setup>
// SkillScopePanel - 作用域(生效位置)展示面板
//
// 从 SkillFileInlinePanel 拆出来独立组件,2026-07-07 v4 重写。
// 独立组件的好处:
//   1. 即使本组件内部的 t() 出现 ProxyObject 异常,也只影响自身 sub-tree
//   2. main InlinePanel 不会因此崩掉(render function 拿不到)
//   3. 作用域逻辑独立维护(loadScopeStatus / apply / undo / fold)
//
// 文案策略:跟 InlinePanel 同 — 全部常量字符串,不调 t(),避免 i18n Proxy 报错。

import { ref, computed, onMounted, onUpdated, onUnmounted, onErrorCaptured } from 'vue'
import IconPark from '@/components/IconPark.vue'
import Modal from '@/components/Modal.vue'
import ToolIcon from '@/components/ToolIcon.vue'
import GitSyncPanel from '@/components/skill/GitSyncPanel.vue'
import { useToastStore } from '@/core/store/toast'
import { useToolsStore } from '@/core/store/tools'
import { getSkillScopeStatus, applySkill, listApplies, undoApply, forceUndoApply, toggleGlobalAgent, getSkill, getStoreInfo } from '@/api/skillbox/skills'
import { inspectApplyResult, formatFailedDetail } from '@/api/skillbox/apply_result.js'
import { listProjects } from '@/api/skillbox/projects'
import { plainT } from '@/core/i18n/index.js'
// 2026-07-12 增:folder 按钮需要 platform.fs.reveal / platform.openExternal,
// 显式 import 而非依赖 wails3 全局变量(裸名 platform 在 Web 部署 / Vite dev
// server 上找不到 → 报 "Can't find variable: platform",跟 SkillFileInlinePanel
// / CodeViewer 等老代码用裸名是同一个坑,这里按 platform/index.js 的命名导出
// 显式 import,跨平台稳)。
import { platform } from '@/platform'

const props = defineProps({
  skill: { type: Object, default: () => ({}) },
})

const toast = useToastStore()
// 2026-07-07 改:作用域区图标 fallback — 后端 scope-status 返回的 tool 元数据不一定带 icon_file,
// 优先用 store 里全量工具的 ToolView(自定义图标 + mdi_icon 完整配置)。
const toolsStore = useToolsStore()
const toolsById = computed(() => {
  const m = {}
  for (const t of toolsStore.items || []) {
    if (t && t.tool_id) m[t.tool_id] = t
  }
  return m
})
function findTool(toolID) {
  // 先在 toolsStore 全量工具表里查(有完整 icon_file / mdi_icon / display_name)
  const fromStore = toolsById.value[toolID]
  if (fromStore) return fromStore
  // 兜底:scope-status 返回的轻量 tool 元数据
  const fromScope = scopeTools.value.find((x) => x.tool_id === toolID)
  return fromScope || null
}
function toolDisplay(toolID) {
  const t = findTool(toolID)
  if (!t) return toolShort(toolID)
  return t.display_name || t.display || t.tool_id || toolShort(toolID)
}

// 2026-07-13 改:SkillScopePanel 文案走 i18n key(原本是 LABEL_* 硬编码常量,
// 跟 SkillFileInlinePanel 同款策略 — 不用 useI18n,直接 plainT 兜底,
// 避免 v-if 懒挂载 + wails webview 下的 ProxyObject 异常)。
// 同样复用 t(key, values) 包装 plainT,跟 SkillsView / SkillFileInlinePanel
// 保持一致。
function t(key, values) {
  return plainT(key, values)
}
const LABEL_SCOPE = 'skills.scope.title'
const LABEL_GLOBAL = 'skills.scope.global'
// 2026-07-16 改:target 显示真实项目名,不再硬拼 LABEL_PROJECT_PREFIX + id 文本。
// 用户反馈"项目 #1 / #2 / #3"是语言标识,正确的应该是从 projects 数据表读取
// 用户在项目页配置的 name(走 loadProjectsMap → projectsById)。
const LABEL_EMPTY = 'skills.scope.empty'
const LABEL_LOADING = 'skills.scope.loading'
const LABEL_ENABLE = 'skills.scope.enable'
const LABEL_DISABLE = 'skills.scope.disable'
const LABEL_TITLE_ERROR = 'skills.scope.loadError'
const LABEL_RETRY = 'skills.scope.retry'
// 2026-07-12 增:顶部"全局 Agent Skill" tag 文案常量(跟 TreeNode 卡片上的 tag 一致)。
// 用户反馈把"全局 Agent"补全成"全局 Agent Skill"——更明确指向"这是一个全局
// Agent 的技能",跟普通工具作用域区分。
const LABEL_GLOBAL_AGENT = 'skills.scope.globalAgent'
const LABEL_GLOBAL_AGENT_TIP = 'skills.scope.globalAgentTip'
// 2026-07-12 增:tag 右侧两个图标按钮的 tip 文案。
// - info 按钮 = 弹窗提示全局 Agent 的说明 + 列出适配的工具
// - folder 按钮 = 直接打开 ~/.agents/skills/ 共享池根目录
const LABEL_GLOBAL_AGENT_INFO_TIP = 'skills.scope.globalAgentInfoTip'
const LABEL_GLOBAL_AGENT_FOLDER_TIP = 'skills.scope.globalAgentFolderTip'

// 2026-07-12 增:info 弹窗内文案(常量字符串,不依赖 i18n,跟组件其它 LABEL_* 一致)。
// 2026-07-12 改:联网搜索后改写"适配全局 Agent 的工具"清单为固定值 ——
// 不要动态过滤 toolsStore(那边字段可能跟用户实际安装的工具不一致,
// 也覆盖不到未在 store 注册的工具)。固定清单基于公开资料总结:
//
//   ✅ 支持 ~/.agents/skills/ 的工具:
//     - GitHub Copilot(VS Code Agent Skills 文档明确)
//     - Antigravity(Google,antigravity.google/docs/skills)
//     - Claude Code(项目级 .agents/skills/ 共享标准)
//     - Codex CLI(沿用开放标准)
//     - Qwen Code(已支持 Skills 标准)
//     - Cursor(项目级 .cursor/skills/,个人级尚未确认 .agents 路径)
//
//   ⚠️ 待确认/暂不支持个人级 ~/.agents/skills/ 的工具:
//     - Claude Code 个人级实际是 ~/.claude/skills/(非 .agents/skills/)
//     - Trae — 文档未明确支持 .agents 路径
//     - Cline — 文档未提及 .agents/skills
//     - Cursor 个人级可能走 ~/.cursor/skills/
//
// 写死而不是走 store 过滤,是产品决定的"事实快照",跟后端 resolver 也
// 解耦(后端只看磁盘 .agents/skills/<name>/SKILL.md 存不存在)。
const LABEL_INFO_TITLE = 'skills.scope.globalAgentInfoTitle'
const LABEL_INFO_DESC = 'skills.scope.globalAgentInfoDesc'
const LABEL_INFO_TOOL_TITLE = 'skills.scope.globalAgentCompatibleToolsTitle'
const LABEL_INFO_TOOL_SUPPORTED = 'skills.scope.globalAgentSupported'
const LABEL_INFO_TOOL_PARTIAL = 'skills.scope.globalAgentPartial'
const LABEL_INFO_TOOL_EMPTY = 'skills.scope.globalAgentEmpty'

const scopeTools = ref([])
const scopeHits = ref([])
const scopeLoading = ref(false)
const scopeError = ref('')
const scopeCollapsed = ref(null)
// 2026-07-07 增:作用域区整体可展开/收起(标题栏点击切换)。
// true = 整体收起(只看到标题);false = 展开(看到工具列表)。
// 2026-07-07 改 v2:默认 false — 用户进首页就应该能看到作用域生效位置,
// 不需要先点开"问号图标"。sectionCollapsed 控制整块可见性。
const sectionCollapsed = ref(false)
function toggleSection() {
  sectionCollapsed.value = !sectionCollapsed.value
}

const busyKey = ref('')
function busyKeyFor(toolID, scope, projectID) {
  return `${toolID}|${scope}|${projectID || 0}`
}
function isScopeTargetBusy(group, target) {
  return busyKey.value === busyKeyFor(group.tool_id, target.scope, target.project_id)
}

function toolShort(toolID) {
  if (!toolID) return '?'
  return toolID.charAt(0).toUpperCase() + toolID.slice(1)
}
// 2026-07-16 改:project 作用域显示真实项目名(从 projects 数据表读取),
// 不再硬拼 i18n 文本「项目 #N」。用户在项目页配置的 name 是事实来源;
// 拿不到(name 缺失 / 接口未拉 / 项目被删)再退回 project_id,保证至少可读。
const projectsById = ref({})
let projectsLoadedAt = 0
async function loadProjectsMap(force = false) {
  // 缓存策略:首次加载或 5 分钟以上 / force=true 才重拉,避免每次切 skill 都打接口。
  const FRESH_MS = 5 * 60 * 1000
  if (!force && projectsLoadedAt && Date.now() - projectsLoadedAt < FRESH_MS) {
    return
  }
  try {
    const out = await listProjects({ page: 1, size: 500 })
    const items = out?.items || []
    const m = {}
    for (const p of items) {
      if (p && p.id !== undefined && p.id !== null) m[p.id] = p
    }
    projectsById.value = m
    projectsLoadedAt = Date.now()
  } catch (_) {
    // 静默失败:targetLabel 拿不到 name 时仍可退回 project_id 兜底,
    // 避免接口 5xx 时整个作用域区空白。
  }
}
function projectLabel(projectID) {
  if (!projectID) return ''
  const p = projectsById.value[projectID]
  if (p && (p.name || p.alias)) return p.name || p.alias
  return `${projectID}`
}
function targetLabel(target) {
  if (!target) return ''
  if (target.scope === 'global') return t(LABEL_GLOBAL)
  return projectLabel(target.project_id)
}

const scopeGroupByTool = computed(() => {
  const out = []
  for (const tool of scopeTools.value) {
    const toolHits = scopeHits.value.filter((h) => h.tool_id === tool.tool_id)
    const map = new Map()
    for (const h of toolHits) {
      const key = h.scope === 'global' ? 'global' : `p:${h.project_id}`
      if (!map.has(key)) {
        map.set(key, {
          key,
          scope: h.scope,
          project_id: h.project_id || 0,
          project_label: targetLabel({ scope: h.scope, project_id: h.project_id }),
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
      tool_id: tool.tool_id,
      display: toolDisplay(tool.tool_id),
      // 不在 group 上挂 icon 字段了,template 用 ToolIcon + findTool(tool_id) 直接取
      hitCount: toolHits.filter((h) => h.exists).length,
      hasHit: toolHits.some((h) => h.exists),
      // 2026-07-08 增:工具是否在全局作用域启用。template 据此在数量标签前显示"全局"chip。
      hasGlobal: toolHits.some((h) => h.scope === 'global' && h.exists),
      targets,
    })
  }
  // 2026-07-07 改:作用域区工具排序 — 按 hitCount(存在命中数)降序,越多越前面;
  // 同数按 tool_id 字母序兜底,保证排序稳定。
  out.sort((a, b) => {
    if (b.hitCount !== a.hitCount) return b.hitCount - a.hitCount
    return String(a.tool_id).localeCompare(String(b.tool_id))
  })
  return out
})

function isCollapsed(toolID) {
  if (!scopeCollapsed.value) return false
  return scopeCollapsed.value.has(toolID)
}
function toggle(toolID) {
  const cur = scopeCollapsed.value || new Set()
  const next = new Set(cur)
  if (next.has(toolID)) next.delete(toolID)
  else next.add(toolID)
  // 2026-07-07 修:全展开时(null)删除最后一个折叠项 → 重新回到全展开(null);
  // 全折叠(所有 tool 都折叠)时,保留 Set(否则所有 group 同时展开太挤)。
  const allCollapsed = next.size === scopeTools.value.length && scopeTools.value.length > 0
  scopeCollapsed.value = next.size === 0 ? null : (allCollapsed ? next : next)
}

// 2026-07-07 改 v3:默认折叠全部工具,用户主动点开才展开。
// 旧版 scopeCollapsed = null → isCollapsed 返 false → 全展开,信息密度太高。
function resetCollapsed() {
  if (scopeTools.value && scopeTools.value.length) {
    scopeCollapsed.value = new Set(scopeTools.value.map((t) => t.tool_id))
  } else {
    scopeCollapsed.value = null
  }
}

async function loadScope({ resetCollapsed: shouldReset } = {}) {
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
    // 2026-07-07 改 v4:
    //   - resetCollapsed=true  → 切 skill(初次进入)时调用,默认折叠全部
    //   - resetCollapsed=false → apply/undo / scope-refresh 事件后调用,保留用户展开态
    if (shouldReset) resetCollapsed()
  } catch (e) {
    scopeError.value = e?.message || String(e)
  } finally {
    scopeLoading.value = false
  }
}

// 2026-07-07 改 v6:不依赖 vue 的 watch(同 SkillFileInlinePanel 修复理由),
// 改用 onUpdated + 手动引用比较。
let _lastSkillName = null
let _lastSkillVersion = null
function _syncWatch() {
  const sk = props.skill
  const curName = sk?.name
  const curVersion = sk?.version
  if (curName === _lastSkillName && curVersion === _lastSkillVersion) return
  _lastSkillName = curName
  _lastSkillVersion = curVersion
  if (!curName) return
  // 2026-07-07 改 v4:切 skill 必须 reset 折叠态,旧折叠对该 skill 的 tool_id 集合无意义。
  // loadScope 内部根据参数决定是否 reset。
  loadScope({ resetCollapsed: true })
  // 2026-07-12 增:切 skill 也要重查"全局 Agent"状态,旧 skill 的 is_global_agent
  // 不能继承到新 skill。同时清掉 toggleLoading(避免新 skill 状态被旧 toggle 的
  // loading 卡住)。
  loadGlobalAgentStatus({ reset: true })
}
onUpdated(_syncWatch)

// 2026-07-12 增:全局 Agent 状态(顶部 tag 的"选中态"=该 skill 同步到
// ~/.agents/skills/ 全局目录下)。
//
// 数据来源:后端 get_skill 接口返回的 is_global_agent / global_source_path 字段
// (后端走 skillstore.ResolveGlobalSourcePath 实时 stat ~/.agents/skills/<name>/
// SKILL.md 是否存在,跟 buildTreeNode 注入到 TreeNode 的 source_path 是同一个
// 判定函数 —— 避免出现"列表说不是全局,详情说是全局"的割裂)。
//
// 切 skill / 收到 scope-refresh 事件(用户保存了 SKILL.md)时重查;toggle 完
// 成后立刻用后端返回值覆盖本地状态,无需再发请求。
const isGlobalAgent = ref(false)
const globalAgentLoading = ref(false)
const toggleAgentBusy = ref(false)
async function loadGlobalAgentStatus({ reset } = {}) {
  const sk = props.skill
  if (!sk || !sk.name) {
    isGlobalAgent.value = false
    return
  }
  // 切 skill 时立刻重置本地状态,避免旧 skill 的全局 Agent tag 在新 skill
  // 加载完成前闪一下。
  if (reset) {
    isGlobalAgent.value = false
  }
  globalAgentLoading.value = true
  try {
    const r = await getSkill({
      name: sk.name,
      version: sk.version || '',
      path: sk.group_path ? `${sk.group_path}/${sk.name}` : sk.name,
    })
    isGlobalAgent.value = !!r?.is_global_agent
  } catch (_) {
    // 静默失败:tag 默认 false,不影响主流程
    isGlobalAgent.value = false
  } finally {
    globalAgentLoading.value = false
  }
}

async function onToggleGlobalAgentClick() {
  if (toggleAgentBusy.value) return
  const sk = props.skill
  if (!sk || !sk.name) return
  toggleAgentBusy.value = true
  const next = !isGlobalAgent.value
  try {
    await toggleGlobalAgent({
      name: sk.name,
      version: sk.version || '',
      // 2026-07-12 增:多级分组支持 —— 老代码只传 name,后端 store.Load(name)
      // 只命中根下直接子目录,分组下的 skill 必报 ErrNotFound。
      // 现在走 group_path + name,跟 store.LoadByPath 对齐。
      group_path: sk.group_path || '',
      enabled: next,
    })
    isGlobalAgent.value = next
    // 通知左侧 skill 卡片列表 reload —— store.ListTree 是实时检测
    // ~/.agents/skills/<name>/ 物理文件,新镜像写盘后下次拉取即可看到 tag。
    // 走 dispatchEvent('skillbox:scope-refresh') 让 SkillsView / ScopePanel 内部
    // 各自刷新(loadScope 自己不重查 isGlobalAgent,这里再单独补一次)。
    window.dispatchEvent(new CustomEvent('skillbox:scope-refresh'))
    // 顺便 reload 自身 scope-status(用户可能希望看到工具列表里某些 tool 的
    // applied_tools 发生变化 — 但其实跟全局 Agent tag 无关,这里保留以防万一)。
    loadScope()
    // 再主动重查一次 is_global_agent 后端权威值(写盘后端 store 可能要 stat
    // 真实路径,跟前端期望严格一致)。
    await loadGlobalAgentStatus()
    toast.success(next
      ? t('skills.scope.globalAgentEnabled', { name: sk.name })
      : t('skills.scope.globalAgentDisabled', { name: sk.name }))
  } catch (e) {
    toast.error(t('skills.scope.globalAgentToggleFailed', { msg: e?.message || e }))
  } finally {
    toggleAgentBusy.value = false
  }
}

function onScopeRefresh() {
  // 2026-07-07 改 v4:外部 scope-refresh(用户保存 SKILL.md / 重拉数据等)
  // 不应该把用户的折叠状态清掉。apply/undo 完成后自己显式调 loadScope() 也是保留态。
  loadScope()
  // 2026-07-12 增:同步重查全局 Agent 状态,确保用户保存 SKILL.md 后 tag 仍是最新值。
  loadGlobalAgentStatus()
  // 2026-07-16 增:项目页新建/重命名/删除项目也会派发 scope-refresh,
  // 强制重拉 projectsById 让 target 显示跟项目数据表对齐。
  loadProjectsMap(true)
}

// 2026-07-12 增:info 弹窗 — 列出"哪些工具适配了 ~/.agents/skills/"。
//
// 2026-07-12 改:从动态 toolsStore 过滤改为固定清单(用户反馈"目前来说是
// 固定的")。理由:
//   - toolsStore 是后端返回的工具元数据表,跟"哪些工具真的支持 .agents/skills
//     个人级路径"是两回事 —— store 里有的工具不一定支持这条路径,反之亦然
//   - 联网搜索结果显示当前生态对 ~/.agents/skills/ 的支持矩阵尚未稳定,
//     走动态过滤会误导用户(显示一堆"已支持"但其实只支持其他路径)
//   - 固定清单是产品文档级别的"事实快照",跟后端 store 解耦,后续有官方
//     声明变化时统一改这一个常量即可
//
// 工具对象结构:
//   - name:    显示名(用 mdi: 字段给 ToolIcon 渲染图标,没图标就 fallback)
//   - mdi:     mdi icon 名
//   - status:  'supported' | 'partial' | 'unsupported' 三种状态
//   - note:    备注 key(走 i18n,skills.scope.toolNotes.*)
const GLOBAL_AGENT_SUPPORTED_TOOLS = [
  {
    name: 'GitHub Copilot',
    mdi: 'mdi:github',
    status: 'supported',
    noteKey: 'skills.scope.toolNotes.vscode',
  },
  {
    name: 'Antigravity',
    mdi: 'mdi:rocket-launch-outline',
    status: 'supported',
    noteKey: 'skills.scope.toolNotes.antigravity',
  },
  {
    name: 'Claude Code',
    mdi: 'mdi:anthropic',
    status: 'partial',
    noteKey: 'skills.scope.toolNotes.claude',
  },
  {
    name: 'Codex CLI',
    mdi: 'mdi:console-line',
    status: 'supported',
    noteKey: 'skills.scope.toolNotes.codex',
  },
  {
    name: 'Qwen Code',
    mdi: 'mdi:language-python',
    status: 'supported',
    noteKey: 'skills.scope.toolNotes.qwen',
  },
  {
    name: 'Cursor',
    mdi: 'mdi:cursor-default-click-outline',
    status: 'partial',
    noteKey: 'skills.scope.toolNotes.cursor',
  },
  {
    name: 'Trae IDE',
    mdi: 'mdi:application-outline',
    status: 'unsupported',
    noteKey: 'skills.scope.toolNotes.opencode',
  },
  {
    name: 'Cline',
    mdi: 'mdi:robot-outline',
    status: 'unsupported',
    noteKey: 'skills.scope.toolNotes.other',
  },
]
const globalAgentInfoOpen = ref(false)
// 显示顺序:supported → partial → unsupported;同状态按字母序稳定排序。
const globalAgentTools = computed(() => {
  const out = [...GLOBAL_AGENT_SUPPORTED_TOOLS]
  const order = { supported: 0, partial: 1, unsupported: 2 }
  out.sort((a, b) => {
    const oa = order[a.status] ?? 9
    const ob = order[b.status] ?? 9
    if (oa !== ob) return oa - ob
    return String(a.name).localeCompare(String(b.name))
  })
  return out
})
function openGlobalAgentInfo() {
  globalAgentInfoOpen.value = true
}
function closeGlobalAgentInfo() {
  globalAgentInfoOpen.value = false
}

// 2026-07-12 增:folder 按钮 — 直接打开 ~/.agents/skills/ 共享池根目录。
//
// 路径来源:后端 get_store_info 接口在 2026-07-12 加了 global_agent_root 字段
// (home + .agents/skills,跨平台一致)。前端不传 ~ 缩写(wails desktop 端
// fs.reveal 不一定做 shell 展开),直接用绝对路径,跟详情区 openInFolder 行为对齐。
// 缓存策略:首次点击时拉一次,后续直接复用 ref,避免重复 HTTP。
const globalAgentRootPath = ref('')
async function ensureGlobalAgentRoot() {
  if (globalAgentRootPath.value) return globalAgentRootPath.value
  try {
    const info = await getStoreInfo()
    globalAgentRootPath.value = info?.global_agent_root || ''
  } catch (_) {
    globalAgentRootPath.value = ''
  }
  return globalAgentRootPath.value
}

async function openGlobalAgentFolder() {
  const p = await ensureGlobalAgentRoot()
  if (!p) {
    toast.error(t('skills.scope.globalAgentPathFailed'))
    return
  }
  try {
    // 2026-07-12 修:platform 已经显式 import,直接 platform.fs.reveal / platform.openExternal,
    // 不要再 platform.platform.openExternal(嵌套访问失败)。
    if (platform?.fs?.reveal) {
      const r = await platform.fs.reveal(p)
      // 桌面端 fs.reveal 成功 → return true / 对象;Web 部署无 hook → 抛 501 带 fallback_url,
      // platform 内部已经把 fallbackUrl 解到 e.data.fallback_url 并 return {ok:false, fallbackUrl},
      // 这里再调 platform.openExternal 走浏览器兜底。
      if (r && r.ok === false && r.fallbackUrl) {
        if (platform?.openExternal) {
          platform.openExternal(r.fallbackUrl)
        } else {
          toast.info(t('skills.scope.globalAgentDirToast', { url: r.fallbackUrl }))
        }
      }
      return
    }
    toast.info(t('skills.scope.dirToast', { path: p }))
  } catch (e) {
    toast.error(t('skills.scope.openFolderFailed', { msg: e?.message || e }))
  }
}

onMounted(() => {
  if (props.skill?.name) {
    loadScope({ resetCollapsed: true })
    loadGlobalAgentStatus({ reset: true })
  }
  // 2026-07-16 增:作用域区每个 target 都要展示 project name(替代硬拼的
  // 「项目 #N」)。listProjects 全量拉一次后缓存进 projectsById,后续 5 分钟内
  // 复用;用户在项目页新建/重命名/删除项目 → SkillsView 收到 scope-refresh
  // 事件也会强制刷新一次,保证本组件的展示跟项目数据表实时一致。
  loadProjectsMap()
  window.addEventListener('skillbox:scope-refresh', onScopeRefresh)
})
// 2026-07-08 增:旧版漏 onUnmounted 清理 listener,导致 ScopePanel 实例重建时
// (InlinePanel 走 :key 重 mount)旧的 window listener 滞留,后续 dispatchEvent
// 会触发所有还活着的 instance 各发一次 scope-status。证据:用户 apply
// code-review 后日志里出现 canvas-design 的 scope-status 请求,实际 UI 上
// canvas-design 不在视野内,是上一个 ScopePanel 实例残留监听造成的幽灵请求。
onUnmounted(() => {
  window.removeEventListener('skillbox:scope-refresh', onScopeRefresh)
})

// ===== Apply / Undo =====
async function doApplyOne(target, group) {
  busyKey.value = busyKeyFor(group.tool_id, target.scope, target.project_id)
  const sk = props.skill
  try {
    const res = await applySkill({
      name: sk.name,
      scope: target.scope,
      project_id: target.project_id || 0,
      tools: [group.tool_id],
    })
    await loadScope()
    // 2026-07-08 修:apply 完成后必须 dispatch 事件,让 SkillsView 重新拉
    // skill 树 — 树节点的 applied_tools 字段不重算的话,左侧 chip 永远显示
    // 旧状态(用户禁用某 tool 后,左侧 chip 仍显示该 tool)。
    // 注释里早就写了"apply 完成后 dispatch skillbox:scope-refresh",
    // 但实际代码漏掉了,这次补上。
    window.dispatchEvent(new CustomEvent('skillbox:scope-refresh'))
    const ins = inspectApplyResult(res)
    if (ins.allOk) {
      toast.success(t('skills.scope.enableSuccess', { tool: group.display, scope: targetLabel(target) }))
    } else {
      const detail = formatFailedDetail(ins.failedItems)
      toast.error(t('skills.scope.partialFailed') + ': ' + detail, 6000)
      scopeError.value = detail
    }
  } catch (e) {
    toast.error(t('skills.scope.enableFailed', { msg: e?.message || e }))
  } finally {
    busyKey.value = ''
  }
}

async function doUnapplyOne(target, group) {
  busyKey.value = busyKeyFor(group.tool_id, target.scope, target.project_id)
  const sk = props.skill
  try {
    const list = await listApplies({
      scope: target.scope,
      name: sk.name,
      tool: group.tool_id,
      status: 'applied',
      page: 1,
      size: 1,
    })
    const last = list?.items?.[0]
    if (!last) {
      await forceUndoApply({
        scope: target.scope,
        project_id: target.project_id || 0,
        name: sk.name,
        tool: group.tool_id,
      })
    } else {
      await undoApply({ apply_id: last.id })
    }
    await loadScope()
    // 2026-07-08 修:同 doApplyOne,unapply 后也必须 dispatch,左侧 chip 才会同步消失。
    window.dispatchEvent(new CustomEvent('skillbox:scope-refresh'))
    toast.success(t('skills.scope.disableSuccess', { tool: group.display, scope: targetLabel(target) }))
  } catch (e) {
    toast.error(t('skills.scope.disableFailed', { msg: e?.message || e }))
  } finally {
    busyKey.value = ''
  }
}

async function handleClick(group, target) {
  if (busyKey.value) return
  if (target.exists) {
    // 2026-07-07 改:用自管 Modal 替代 window.confirm。
    // 旧版 window.confirm 在 wails desktop webview 内被默认禁用/拦截,
    // 用户点完"生效位置"按钮后 confirm 默默返回 false → 流程中断,
    // 用户感受就是"点了没反应"。
    confirmAction.value = {
      kind: 'unapply',
      title: t(LABEL_DISABLE),
      message: t('skills.scope.disableConfirm', { tool: group.display, scope: targetLabel(target), name: props.skill.name }),
      group, target,
    }
  } else {
    confirmAction.value = {
      kind: 'apply',
      title: t(LABEL_ENABLE),
      message: t('skills.scope.enableConfirm', { name: props.skill.name, tool: group.display, scope: targetLabel(target) }),
      group, target,
    }
  }
}

// 2026-07-07 增:自管确认弹窗状态(替代 window.confirm,适配 wails webview)
const confirmAction = ref(null)
const confirmOpen = computed({
  get: () => !!confirmAction.value,
  set: (v) => { if (!v) confirmAction.value = null },
})
async function onConfirmYes() {
  const a = confirmAction.value
  confirmAction.value = null
  if (!a) return
  if (a.kind === 'apply') await doApplyOne(a.target, a.group)
  else await doUnapplyOne(a.target, a.group)
}
function onConfirmNo() {
  confirmAction.value = null
}

// ====== ErrorBoundary 兜底(本组件独立 render,出错只影响自己) ======
const localError = ref(null)
function safeReload() { localError.value = null; loadScope() }
// 2026-07-07 改 v2:必须 return false 阻止错误继续冒泡到父组件
// (父 SkillFileInlinePanel 也有 onErrorCaptured,如果不 return false,会被父级再次捕获,
// 显示成父组件的"加载出错"覆盖页,而不是 scope 区自己的降级 UI)。
onErrorCaptured((err) => {
  console.error('[SkillScopePanel captured]', err)
  localError.value = err?.message || String(err)
  return false
})
</script>

<template>
  <!-- 2026-07-17 增:Git 同步面板(go-git 版本管理)。始终在最顶部展示,
       无论作用域状态如何,用户都能看到仓库 init / push 状态。
       错误降级时不展示(避免噪音);否则无论 localError / scopeLoading 都先渲染。
       2026-07-17 bugfix:之前写成 v-else-if="!localError",把作用域区 v-else-if 链
       截胡,正常状态下作用域区被跳过 — 这里改成独立 v-if 块,作用域区按原条件渲染。 -->
  <GitSyncPanel v-if="!localError" />
  <!-- 错误降级 UI -->
  <div v-if="localError" class="ssp-error">
    <IconPark icon="mdi:alert-circle-outline" width="14" height="14" />
    <span>{{ t(LABEL_TITLE_ERROR) }}: {{ localError }}</span>
    <button class="link" @click="safeReload">{{ t(LABEL_RETRY) }}</button>
  </div>
  <!-- 2026-07-08 改:作用域区高度锁死策略
       - sectionCollapsed=true(整体收起):只显示 header 一行,高度由内容决定,约 36px
       - sectionCollapsed=false(整体展开):面板高度锁成固定值(SECTION_EXPANDED_HEIGHT),
        内部 list 用 overflow:auto 滚动;无论用户点击几个工具展开其下 targets,
        面板自身高度永远不变,只内部滚动,不再把 .sfip-tree-wrap 挤压缩小
       - 实现:.ssp-scope 切换 .is-expanded class,展开态下设 height + flex-basis 固定,
        .ssp-scope-list 设 max-height + overflow-y auto 内部滚动 -->
  <section
    v-else-if="!scopeLoading && (scopeGroupByTool?.length || 0)"
    :class="['ssp-scope', { 'is-expanded': !sectionCollapsed }]"
  >
    <!-- 2026-07-07 改:作用域标题栏改成可点击 button,点击切换整体展开/收起。
         右侧加 chevron 图标提示状态。sectionCollapsed = true 时只显示标题,
         隐藏 .ssp-scope-list;收起态下 max-height 收紧避免占太多空间。 -->
    <button
      type="button"
      class="ssp-scope-header ssp-scope-header-toggle"
      :aria-expanded="!sectionCollapsed"
      @click="toggleSection"
    >
      <!-- 2026-07-08 改:用 Local(图标位置 marker)替代原 mdi:help-circle-outline(Help)。
           "作用域" 语义核心是"这个 skill 在哪些工具/位置上生效",位置/坐标定位图标比问号更贴切。
           用 PascalCase 直传 iconpark 组件名,避免 mdi 映射兜底导致 fallback 到问号。 -->
      <IconPark icon="Local" width="13" height="13" />
      <span>{{ t(LABEL_SCOPE) }}</span>
      <span class="ssp-scope-header-count">{{ t('skills.scope.toolCountShort', { n: scopeGroupByTool.length }) }}</span>
      <!-- 2026-07-07 改:展开收起箭头 chevron-up/down → plus/minus。
           plus = 当前是合上点开后展开,minus = 当前是展开点合上。
           跟 TreeNode/FileTreeNode 同步。 -->
      <IconPark
        :icon="sectionCollapsed ? 'mdi:plus' : 'mdi:minus'"
        width="13"
        height="13"
        class="ssp-scope-header-chevron"
      />
    </button>
    <ul v-if="!sectionCollapsed" class="ssp-scope-list">
      <!-- 2026-07-12 增 v2:作用域区首位"全局 Agent Skill" 行。
           跟左侧 skill 卡片"全局 Agent" tag 同源(后端 getSkill 返回的 is_global_agent
           + 走 skillstore.ResolveGlobalSourcePath 实时 stat ~/.agents/skills/<name>/
           SKILL.md),用户点 tag 切换后会立刻刷新工具列表(scope-refresh event)。
           设计要点(2026-07-12 用户反馈增):
             - 永远排在第一个(放在 v-for 上方,跟其他工具互不影响)
             - 没有展开/折叠(没有 targets 列表)
             - 整行 = 左侧 toggle button(胶囊)+ 右侧 2 个图标按钮(info / folder)
             - 选中态 = emerald 浅底+边框+主色字,未选中=透明+灰文字+无边框
             - 行下方一道横杠把 tag 区跟下方工具列表视觉分隔
             - info 按钮弹窗列出"哪些工具适配了 ~/.agents/skills/"(从 toolsStore 拿)
             - folder 按钮调 platform.fs.reveal 打开 ~/.agents/skills/ 共享池根目录 -->
      <li class="ssp-scope-group ssp-scope-group-global">
        <div class="ssp-global-agent-row">
          <button
            type="button"
            :class="['ssp-global-agent-tag', { 'ssp-global-agent-tag-active': isGlobalAgent }]"
            :disabled="toggleAgentBusy || globalAgentLoading"
            @click="onToggleGlobalAgentClick"
          >
            <span
              v-if="toggleAgentBusy"
              class="ssp-spinner ssp-spinner-xs ssp-global-agent-spinner"
            ></span>
            <IconPark
              v-else
              icon="mdi:earth"
              width="11"
              height="11"
              class="ssp-global-agent-icon"
            />
            <span class="ssp-global-agent-label">{{ t(LABEL_GLOBAL_AGENT) }}</span>
          </button>
          <!-- 2026-07-12 增:tag 右侧两个图标按钮 — info 提示 / folder 打开目录。
               跟 tag 之间留 4px 间距,跟工具列表的 chevron 视觉权重一致。
               点击事件用 stop 防止冒泡触发外层 tag(嵌套 button 不允许,
               实际上这里不是嵌套 — tag 和图标按钮是 div 里的两个并列 button,
               用 stop 是防御性)。
               命中平台:desktop (wails3) 走 platform.fs.reveal 物理打开;
               Web 端兜底走 platform.openExternal fallback。 -->
          <button
            type="button"
            class="ssp-global-agent-icon-btn ssp-global-agent-icon-btn--info"
            @click.stop="openGlobalAgentInfo"
          >
            <IconPark icon="mdi:information-outline" width="11" height="11" />
          </button>
          <button
            type="button"
            class="ssp-global-agent-icon-btn"
            @click.stop="openGlobalAgentFolder"
          >
            <IconPark icon="mdi:folder-outline" width="11" height="11" />
          </button>
        </div>
        <!-- 2026-07-12 增:tag 行下方一根横杠,跟下方工具列表视觉分组。
             全局 Agent 跟"普通工具作用域"是两种语义不同的"位置",用分隔线
             强调边界;横杠颜色用 var(--border) 弱化,跟列表内 group 间距一致。 -->
        <hr class="ssp-global-agent-divider" />
      </li>
      <li
        v-for="group in (scopeGroupByTool || [])"
        :key="group.tool_id"
        class="ssp-scope-group"
      >
        <button
          type="button"
          class="ssp-scope-row"
          :data-tip="group.display"
          @click="toggle(group.tool_id)"
        >
          <IconPark
            :icon="isCollapsed(group.tool_id) ? 'mdi:plus' : 'mdi:minus'"
            width="12"
            height="12"
            class="ssp-scope-chevron"
          />
          <!-- 2026-07-07 改:用 ToolIcon 渲染真图标(icon_file 优先 + mdi 兜底),
               findTool 优先查 toolsStore 全量工具表;store 没数据时降级到
               scopeTools(后端 scope-status 返回的轻量工具元数据)。 -->
          <ToolIcon
            :tool="findTool(group.tool_id) || { mdi_icon: 'mdi:puzzle-outline' }"
            :size="13"
            class="ssp-scope-tool-icon"
          />
          <span class="ssp-scope-row-name">{{ group.display }}</span>
          <!-- 2026-07-08 增:工具启用了全局时,在数量标签前显示"全局"chip,
               让用户一眼看出"这个 skill 是生效在全局还是项目级"。 -->
          <span v-if="group.hasGlobal" class="ssp-scope-row-global" :title="t('skills.scope.globalEnabledTip')">{{ t(LABEL_GLOBAL) }}</span>
          <span v-if="group.hitCount > 0" class="ssp-scope-row-count">{{ group.hitCount }}</span>
        </button>
        <ul v-if="!isCollapsed(group.tool_id)" class="ssp-scope-targets">
          <li v-for="target in group.targets" :key="target.key">
            <button
              type="button"
              :class="['ssp-scope-target', target.exists ? 'ssp-scope-target-active' : '']"
              :title="target.exists ? t(LABEL_DISABLE) : t(LABEL_ENABLE)"
              :disabled="!!busyKey"
              @click="handleClick(group, target)"
            >
              <span
                v-if="isScopeTargetBusy(group, target)"
                class="ssp-spinner ssp-spinner-xs"
              ></span>
              <IconPark
                v-else
                :icon="target.scope === 'global' ? 'mdi:earth' : 'mdi:folder-outline'"
                width="11"
                height="11"
              />
              <span class="ssp-scope-target-name">{{ targetLabel(target) }}</span>
            </button>
          </li>
          <li v-if="!group.targets.length" class="ssp-scope-empty">
            {{ t(LABEL_EMPTY) }}
          </li>
        </ul>
      </li>
    </ul>
    <p v-if="scopeError" class="ssp-scope-error">
      <IconPark icon="mdi:alert-circle-outline" width="11" height="11" />
      {{ scopeError }}
    </p>
  </section>
  <p v-else-if="scopeLoading" class="ssp-scope-loading">
    <span class="ssp-spinner ssp-spinner-xs"></span>
  </p>
  <p v-else class="ssp-scope-empty-tip">{{ t(LABEL_EMPTY) }}</p>

  <!-- 2026-07-07 增:自管确认弹窗,替代 window.confirm。
       wails desktop webview 默认禁用 window.confirm,直接调确认会被静默拒绝。 -->
  <Modal
    v-model="confirmOpen"
    size="sm"
    :title="confirmAction?.title || ''"
    :close-on-mask="false"
    @close="onConfirmNo"
  >
    <p class="ssp-confirm-msg">{{ confirmAction?.message || '' }}</p>
    <template #footer>
      <button type="button" class="ghost" @click="onConfirmNo">{{ t('common.cancel') }}</button>
      <button type="button" class="primary" @click="onConfirmYes">{{ t('common.confirm') }}</button>
    </template>
  </Modal>

  <!-- 2026-07-12 增:全局 Agent 信息弹窗 — 列出适配 ~/.agents/skills/ 的工具。
       跟 confirm 弹窗分两个独立 Modal 组件实例,各自管理 v-model。内容:
       - 顶部描述(全局 Agent 共享池语义)
       - 中间"适配工具"标题 + 工具列表(固定清单,带状态 chip + 备注)
       - 状态:supported(绿) / partial(琥珀) / unsupported(灰)
       - 每行:工具图标 + 名称 + 状态 chip + 备注文字 -->
  <Modal
    v-model="globalAgentInfoOpen"
    size="md"
    :title="t(LABEL_INFO_TITLE)"
    @close="closeGlobalAgentInfo"
  >
    <div class="ssp-info-body">
      <p class="ssp-info-desc">{{ t(LABEL_INFO_DESC) }}</p>
      <h4 class="ssp-info-tool-title">{{ t(LABEL_INFO_TOOL_TITLE) }}</h4>
      <ul v-if="globalAgentTools.length" class="ssp-info-tool-list">
        <li
          v-for="t in globalAgentTools"
          :key="t.name"
          :class="['ssp-info-tool-item', `ssp-info-tool-${t.status}`]"
        >
          <IconPark :icon="t.mdi" width="14" height="14" class="ssp-info-tool-icon" />
          <span class="ssp-info-tool-name">{{ t.name }}</span>
          <span
            v-if="t.status === 'supported'"
            class="ssp-info-tool-badge ssp-info-tool-badge-supported"
          >{{ t(LABEL_INFO_TOOL_SUPPORTED) }}</span>
          <span
            v-else-if="t.status === 'partial'"
            class="ssp-info-tool-badge ssp-info-tool-badge-partial"
          >{{ t(LABEL_INFO_TOOL_PARTIAL) }}</span>
          <span
            v-else
            class="ssp-info-tool-badge ssp-info-tool-badge-unsupported"
          >—</span>
          <span class="ssp-info-tool-note">{{ t.noteKey ? t(t.noteKey) : '' }}</span>
        </li>
      </ul>
      <p v-else class="ssp-info-tool-empty">{{ t(LABEL_INFO_TOOL_EMPTY) }}</p>
    </div>
    <template #footer>
      <button type="button" class="primary" @click="closeGlobalAgentInfo">{{ t('common.close') }}</button>
    </template>
  </Modal>
</template>

<style scoped>
.ssp-scope {
  border-bottom: 1px solid var(--border);
  /* 2026-07-10 改 v7:用户反馈"作用于区域面板"主体要白底,只保留 header 灰底做层级区分。
     之前 v6 改回 var(--bg-subtle) 是为了跟 file tree 视觉对齐,现在按用户最新需求反转:
     外层 panel(包含 body 列表)改用 var(--bg-card) 纯白,header 仍用 var(--bg-subtle) 灰,
     header 自身带 border-bottom 自然分隔 header 与 body 区域。 */
  background: var(--bg-card);
  /* 2026-07-07 改 v4:作用域区移到文件树底部,不要再 max-height:50%(占满左栏下半),
     让它作为底部一块自然收缩,文件树占主空间。 */
  flex-shrink: 0;
  /* 2026-07-07 修:必须显式 width:100% + max-width:100% + box-sizing,
     否则 .ssp-scope-list/.ssp-scope-row 在 flex 子项里按内容撑开,
     把左栏宽度顶出去超出界面。 */
  width: 100%;
  max-width: 100%;
  box-sizing: border-box;
  min-width: 0;
  /* 2026-07-08 改:折叠态(sectionCollapsed=true)下只显示 header,
     高度由内容自然撑开,不设固定高度,避免空占空间 */
}
/* 2026-07-08 增:展开态下高度锁死 —— 用户反馈"点击工具展开 targets 时
   此面板高度不能变"。实现方式:
   - panel 自身 height + flex-basis 锁成 SECTION_EXPANDED_HEIGHT(300px),
     不论里面 tool 列表展开几个 targets,panel 占用的 flex 空间恒定
   - .ssp-scope-list(工具组列表)内部 max-height + overflow-y auto,
     内容溢出走内部滚动,不会再撑开 panel
   这样保证展开状态下 .sfip-tree-wrap(文件树)的高度不被压缩 */
.ssp-scope.is-expanded {
  height: 300px;
  flex: 0 0 300px;
  display: flex;
  flex-direction: column;
}
.ssp-scope-header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  font-size: 12px;
  font-weight: 600;
  /* 2026-07-07 改 v6:作用域标题色跟 sfip-tree-header 完全一致 var(--text-dim),
     图标 svg 强制 currentColor 继承,避免 IconPark svg 默认 fill 跟文字对比过大
     看上去像"白色"。 */
  color: var(--text-dim);
  letter-spacing: 0.04em;
  text-transform: uppercase;
}
.ssp-scope-header :deep(svg) { color: inherit; fill: currentColor; }
/* 2026-07-07 改 v6:标题内 IconPark svg 显式 currentColor,跟文字色统一 */
.ssp-scope-header > :deep(svg),
.ssp-scope-header svg { color: inherit; fill: currentColor; }
/* 2026-07-07 改:作用域标题栏变成可点击 button 后的样式 reset。
   跟 .sfip-tree-header 同款风格(uppercase + sticky + bg-subtle + border-bottom)。
   收起态(sectionCollapsed=true)下整块只占标题一行,不再 max-height:45%,自然收缩。 */
.ssp-scope-header-toggle {
  width: 100%;
  /* 2026-07-07 改 v6:作用域标题栏跟面板同底 var(--bg-subtle),hover 时 var(--bg-hover)
     加深。 */
  background: var(--bg-subtle);
  border: none;
  border-bottom: 1px solid var(--border);
  border-top: 1px solid var(--border);
  cursor: pointer;
  text-align: left;
  font: inherit;
  text-transform: uppercase;
  transition: background 0.12s ease;
}
.ssp-scope-header-toggle:hover { background: var(--bg-hover); }
.ssp-scope-header-count {
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
.ssp-scope-header-chevron {
  margin-left: 4px;
  color: var(--text-faint);
}
.ssp-scope-list {
  list-style: none;
  margin: 0;
  padding: 0 0 6px;
  /* 2026-07-07 修:列表容器限制宽度,长工具名/路径不撑出父级 */
  width: 100%;
  max-width: 100%;
  min-width: 0;
  box-sizing: border-box;
  /* 2026-07-08 改:展开态下,list 占 panel 减 header 的剩余高度,
     内部工具组展开后溢出走 overflow-y auto,不再撑开外层 panel */
  flex: 1 1 auto;
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
  /* 2026-07-08 增:作用域面板内滚动条 —— 跟随全局隐藏 */
  scrollbar-width: none;
}
.ssp-scope-list::-webkit-scrollbar {
  display: none;
  width: 0;
  height: 0;
}
.ssp-scope-list::-webkit-scrollbar-track,
.ssp-scope-list::-webkit-scrollbar-thumb {
  background: transparent;
}
.ssp-scope-group {
  padding: 0;
  min-width: 0;
}
.ssp-scope-row {
  width: 100%;
  max-width: 100%;
  min-width: 0;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 12px;
  background: transparent;
  border: none;
  color: var(--text);
  font: inherit;
  font-size: 13px;
  cursor: pointer;
  text-align: left;
  box-sizing: border-box;
}
.ssp-scope-row:hover { background: var(--bg-hover); }
.ssp-scope-chevron { color: var(--text-faint); flex-shrink: 0; }
.ssp-scope-tool-icon {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 13px;
  height: 13px;
}
.ssp-scope-row-name {
  flex: 1;
  font-weight: 500;
  /* 2026-07-07 修:长工具名截断,否则"启用"按钮会被挤出 */
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}
.ssp-scope-row-count {
  font-size: 11px;
  padding: 1px 6px;
  background: var(--accent-blue-bg);
  color: var(--accent-blue);
  border-radius: 999px;
  flex-shrink: 0;
}
/* 2026-07-08 增:工具启用全局时的"全局"chip。
   2026-07-08 改 v2:跟 .ssp-scope-row-count 视觉一致(同色同字号同圆角),
   之前用实心蓝底+白字跟数量徽章视觉打架,看着像两个不同元素;
   用户反馈希望保持同款 — 区分靠位置(数量标签在前)+ 文案("全局" vs 数字)。 */
.ssp-scope-row-global {
  font-size: 11px;
  padding: 1px 6px;
  background: var(--accent-blue-bg);
  color: var(--accent-blue);
  border-radius: 999px;
  flex-shrink: 0;
  font-weight: 500;
}
.ssp-scope-targets {
  list-style: none;
  margin: 0;
  padding: 0 0 4px 26px;
}
.ssp-scope-target {
  width: 100%;
  max-width: 100%;
  min-width: 0;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 3px 10px;
  margin: 1px 0;
  background: transparent;
  border: 1px dashed var(--border);
  color: var(--text-dim);
  font: inherit;
  font-size: 12px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  text-align: left;
  box-sizing: border-box;
}
.ssp-scope-target-active {
  background: var(--accent-blue-bg);
  color: var(--accent-blue);
  border-style: solid;
  border-color: var(--accent-blue-border);
}
.ssp-scope-target:hover:not(:disabled) {
  background: var(--bg-hover);
  color: var(--text);
}
.ssp-scope-target:disabled { opacity: 0.5; cursor: not-allowed; }
.ssp-scope-target-name {
  flex: 1;
  /* 2026-07-07 修:长路径/项目名截断 */
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}
.ssp-scope-empty,
.ssp-scope-empty-tip {
  font-size: 11px;
  color: var(--text-faint);
  font-style: italic;
  padding: 4px 14px;
  margin: 0;
}
.ssp-scope-error {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 0;
  padding: 6px 12px;
  background: var(--danger-dim);
  color: var(--danger);
  font-size: 12px;
}
.ssp-scope-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 14px;
  border-bottom: 1px solid var(--border);
  color: var(--text-faint);
}
.ssp-spinner {
  display: inline-block;
  width: 12px;
  height: 12px;
  border: 2px solid var(--border);
  border-top-color: var(--accent-blue);
  border-radius: 50%;
  animation: ssp-spin 0.8s linear infinite;
}
.ssp-spinner-xs { width: 12px; height: 12px; }
@keyframes ssp-spin { to { transform: rotate(360deg); } }
.ssp-error {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  background: var(--danger-dim);
  color: var(--danger);
  font-size: 12px;
  border-bottom: 1px solid var(--border);
}
.ssp-error .link {
  background: transparent;
  border: none;
  color: var(--accent-blue);
  text-decoration: underline;
  cursor: pointer;
  font-size: 12px;
  margin-left: auto;
}

/* 2026-07-07 增:自管确认弹窗文案 */
/* 2026-07-12 改 v2:全局 Agent 改成纯 tag 胶囊样式(2026-07-12 用户反馈)。
   设计目标:
     - 选中 = emerald 浅底 + 边框 + 主色字(跟 TreeNode .tree-skill-badge-global-agent
       同款色相,视觉一致)
     - 未选中 = 完全透明背景 + 无边框 + 灰文字(跟工具列表的"未启用"状态
       视觉接近,只是一个轻量的标签)
   实现:
     - 胶囊 padding 1px 8px(比原 row padding 小很多,贴合 tag 形态)
     - 整行可点击 — cursor:pointer,hover 时未选中态显示淡灰底提示可点
     - 选中态 hover 加深一档(emerald-bg → emerald-border),强化反馈
     - 字体 11px(比 row 文字小一档,贴合 tag 视觉权重)
     - 切换中 toggleAgentBusy 时显示 spinner 替代图标 */
.ssp-global-agent-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin: 4px 12px 6px;
  padding: 1px 8px;
  border-radius: 999px;
  background: transparent;
  /* 2026-07-12 改:未选中态边框用 var(--border) 普通灰边(不要再 transparent,
     用户反馈"边框要保留"——保留边让 tag 始终像可识别的标签,不被当成 inline
     文字流走);文字色用 var(--text-dim) 普通灰(不再 --text-faint 置灰,
     让 tag 视觉权重跟其它 pill 同档)。 */
  border: 1px solid var(--border);
  color: var(--text-dim);
  font-size: 11px;
  font-weight: 500;
  line-height: 1.5;
  white-space: nowrap;
  cursor: pointer;
  user-select: none;
  transition: background 0.12s ease, color 0.12s ease, border-color 0.12s ease;
}
.ssp-global-agent-tag:hover:not(:disabled) {
  background: var(--bg-hover);
  color: var(--text);
  border-color: var(--text-faint);
}
/* 选中态:emerald 三件套 + 字色加深 */
.ssp-global-agent-tag-active {
  background: var(--accent-emerald-bg);
  border-color: var(--accent-emerald-border);
  color: var(--accent-emerald);
  font-weight: 600;
}
.ssp-global-agent-tag-active:hover:not(:disabled) {
  /* 选中态 hover 时把边框加深一档,告诉用户"还能再点切换",而不是淡化。
     背景保持 emerald-bg(不能加深太多,亮色底反复叠会发灰)。 */
  border-color: var(--accent-emerald);
  background: var(--accent-emerald-bg);
}
.ssp-global-agent-tag:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
.ssp-global-agent-icon { color: inherit; flex-shrink: 0; }
.ssp-global-agent-spinner { flex-shrink: 0; }

/* 2026-07-12 增 v2:整行 row 容器 — 包住左侧 tag + 右侧两个图标按钮。
   display:flex + align-items:center 让 tag 和图标按钮在同一基线水平排列,
   跟工具行 .ssp-scope-row 视觉对齐。
   2026-07-16 改:用户反馈右侧空白过多。两个图标按钮靠右贴边 →
   info 按钮 margin-left:auto 推到 row 最右,folder 紧跟其后 4px 间距。
   margin 跟旧版一致(左右 12px),保证 tag 的胶囊视觉位置不变。 */
.ssp-global-agent-row {
  display: flex;
  align-items: center;
  gap: 4px;
  margin: 4px 12px 4px;
}
/* 2026-07-12 增:tag 右侧两个图标按钮 — 跟工具行 chevron 同尺寸(11~12px),
   视觉权重轻,不抢 tag 的视觉中心。
   - 未选中态:透明背景 + 灰文字 + 透明边框,hover 时淡灰底
   - 选中态:跟 tag 同色系(emerald),跟 tag 整体协调(可视为 tag 的附属)
   - flex:0 0 auto + box-sizing:border-box + 显式 width/height,避免 icon
     把 flex 父级撑开触发横向滚动(参考 fe-sfip-header-hscroll) */
.ssp-global-agent-icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 auto;
  box-sizing: border-box;
  width: 20px;
  height: 18px;
  padding: 0;
  background: transparent;
  border: 1px solid transparent;
  border-radius: var(--radius-sm);
  color: var(--text-faint);
  cursor: pointer;
  overflow: hidden;
  transition: background 0.12s ease, color 0.12s ease, border-color 0.12s ease;
}
/* 2026-07-16 增:info 按钮是 row 最左侧 → 用 margin-left:auto 推到右端,
   folder 按钮跟在 info 之后 4px(gap),整体靠右贴边,去掉原"tag 在左、
   居中、右侧大段空白"的视觉断层。 */
.ssp-global-agent-icon-btn--info {
  margin-left: auto;
}
.ssp-global-agent-icon-btn:hover {
  background: var(--bg-hover);
  color: var(--text-dim);
}
/* 当 tag 处于选中态时(图标按钮跟 emerald tag 视觉对齐):
   图标颜色淡 emerald,hover 加深,跟 tag hover 的 emerald-border 一致 */
.ssp-global-agent-row:has(.ssp-global-agent-tag-active) .ssp-global-agent-icon-btn {
  color: var(--accent-emerald);
  opacity: 0.85;
}
.ssp-global-agent-row:has(.ssp-global-agent-tag-active) .ssp-global-agent-icon-btn:hover {
  background: var(--accent-emerald-bg);
  color: var(--accent-emerald);
  border-color: var(--accent-emerald-border);
  opacity: 1;
}

/* 2026-07-12 增 v2:tag 行下方的横杠,跟下方工具列表视觉分组。
   跟旧版 border-bottom 区别:这里改用 hr + 显式 margin,占据独立一行,
   视觉上像"标题区 / 内容区"的分割,而不是 group 之间的小间隙。
   颜色用 var(--border) 弱化(跟文件树 header 底边一致)。 */
.ssp-global-agent-divider {
  margin: 4px 12px 6px;
  border: 0;
  border-top: 1px solid var(--border);
  height: 0;
  background: transparent;
}
/* 去掉旧的 .ssp-scope-group-global margin-bottom — 现在由 divider 主导分隔 */
.ssp-scope-group-global {
  margin-bottom: 0;
}

/* 2026-07-12 增:全局 Agent info 弹窗样式。
   - 描述:跟普通段落同色,line-height 1.6 便于阅读
   - 工具标题:h4 字号 12px(比正文小),字色 dim,letter-spacing 跟其他
     section header 对齐
   - 工具列表:flex wrap 让 chip 在窄屏自然换行
   - 单个 chip:浅 emerald 底 + 翠绿字 + icon + 名称,跟"全局 Agent Skill"
     主题色一致,用户一眼看出"这些是适配工具" */
.ssp-info-body {
  font-size: 13px;
  color: var(--text);
}
.ssp-info-desc {
  margin: 0 0 16px;
  line-height: 1.6;
  color: var(--text-dim);
}
.ssp-info-tool-title {
  margin: 0 0 8px;
  font-size: 12px;
  font-weight: 600;
  color: var(--text-dim);
  letter-spacing: 0.04em;
  text-transform: uppercase;
}
.ssp-info-tool-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
/* 2026-07-12 改:从 chip 胶囊改成行列表,每行 = 图标 + 名称 + 状态 badge + 备注。
   理由:chip 胶囊在状态"部分支持/不支持"时无法区分,改成 row 列表让
   用户能横向对比 + 看备注。flex 布局,icon + name + badge 左对齐,
   note 占剩余宽度自动换行。 */
.ssp-info-tool-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  background: var(--bg-subtle);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  font-size: 12px;
  line-height: 1.5;
  flex-wrap: wrap;
}
.ssp-info-tool-icon {
  flex-shrink: 0;
  color: var(--text-dim);
  width: 14px;
  height: 14px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}
.ssp-info-tool-name {
  font-weight: 600;
  color: var(--text);
  flex-shrink: 0;
}
/* 状态 badge:三种颜色对应支持程度
   - supported: emerald(跟 tag 选中态同色系,统一"全局 Agent"主题)
   - partial: amber(琥珀,提示"能用但有限制")
   - unsupported: gray(灰,提示"暂不支持") */
.ssp-info-tool-badge {
  flex-shrink: 0;
  font-size: 10px;
  font-weight: 600;
  padding: 1px 6px;
  border-radius: 999px;
  border: 1px solid transparent;
  white-space: nowrap;
}
.ssp-info-tool-badge-supported {
  background: var(--accent-emerald-bg);
  color: var(--accent-emerald);
  border-color: var(--accent-emerald-border);
}
.ssp-info-tool-badge-partial {
  background: rgba(245, 158, 11, 0.1);
  color: #b45309;
  border-color: rgba(245, 158, 11, 0.3);
}
.ssp-info-tool-badge-unsupported {
  background: var(--bg);
  color: var(--text-faint);
  border-color: var(--border);
}
.ssp-info-tool-note {
  flex: 1 1 100%;
  font-size: 11px;
  color: var(--text-faint);
  margin-top: 2px;
  word-break: break-word;
}
.ssp-info-tool-empty {
  margin: 0;
  font-size: 12px;
  color: var(--text-faint);
  font-style: italic;
}

.ssp-confirm-msg {
  margin: 0;
  font-size: 13px;
  line-height: 1.6;
  color: var(--text);
}
</style>
