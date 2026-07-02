// core/store/tools.js - 工具元数据域 Pinia store。
//
// 2026-07-01 新建。对应后端 7 个 ctool 接口(2026-06-30 上线)。
// 集中管理:
//   - items                工具列表(原始后端数据)
//   - filter               客户端过滤态(keyword + source: all|system|user)
//   - loading / reloading / saving / removing  独立 loading
//   - form                 新建 / 编辑 弹窗的临时表单态(reactive,直接 v-model)
//   - confirmOpen / confirmTarget 单一确认弹窗
//
// 业务规则(后端 stool 服务兜底,前端只做"友好提示"避免误用):
//   - is_system=true:tool_id 不可改 + 行不可删;其他字段(display_name / mdi_icon /
//     maturity / note / enabled / sort_order / paths)可改
//   - is_system=false:全部可改 + 可删
//   - paths 非 null = 整组覆盖(update 语义)
//
// 用法:
//   import { useToolsStore } from '@/core/store/tools'
//   const tools = useToolsStore()
//   await tools.load()
//   tools.setKeyword('claude')
//   tools.openCreate()

import { defineStore } from 'pinia'
import {
  listTools,
  createTool,
  updateTool,
  deleteTool,
  reloadTools,
} from '@/api/skillbox/tools'

// 业务常量:maturity 可选值
const ALLOWED_MATURITY = ['stable', 'experimental', 'deprecated']
const ALLOWED_SCOPE = ['global', 'project']
const ALLOWED_CATEGORY = ['user', 'system']

// 空表单模板(避免多处重复 + 保证 reset 时字段不会跨弹窗泄漏)
function emptyForm() {
  return {
    tool_id: '',
    display_name: '',
    mdi_icon: 'mdi:',
    icon_file: '', // 新增:自定义图标文件名(basename);非空时优先于 mdi_icon
    maturity: 'stable',
    note: '',
    enabled: true,
    sort_order: 0,
    paths: [], // [{ scope: 'global'|'project', category: 'user'|'system', path: string, path_order: number }]
  }
}

// 单条 path 行模板(行内 "添加路径" 用)
function emptyPathRow(order = 0) {
  return { scope: 'global', category: 'user', path: '', path_order: order }
}

export const useToolsStore = defineStore('tools', {
  state: () => ({
    // --- 列表 ---
    items: [], // 原始 ToolView[]

    // --- 过滤(客户端,纯前端) ---
    filter: {
      keyword: '',
      source: 'all', // 'all' | 'system' | 'user'
    },

    // --- loading 各自分开(避免一个 spinner 压所有操作) ---
    loading: false, // list 加载中
    reloading: false, // /tools/reload 加载中
    saving: false, // create / update 中
    removing: false, // delete 中

    // --- 错误 ---
    error: '',

    // --- 新建 / 编辑 弹窗(全应用只有一个,formMode 决定 create / edit) ---
    formOpen: false,
    formMode: 'create', // 'create' | 'edit'
    form: emptyForm(),
    editingToolId: '', // 编辑时记录原 tool_id(locator,只读)
    editingIsSystem: false, // 编辑时记录原始 is_system(决定 tool_id 是否禁用)

    // --- 删除确认(单一确认弹窗) ---
    confirmOpen: false,
    confirmTarget: null, // ToolView | null
  }),

  getters: {
    /** 全量工具数(原始列表,不过滤)。 */
    totalCount: (s) => s.items.length,

    /** 系统工具数量。 */
    systemCount: (s) => s.items.filter((x) => x.is_system).length,

    /** 用户工具数量(is_system = false)。 */
    userCount: (s) => s.items.filter((x) => !x.is_system).length,

    /**
     * 客户端过滤后的列表(view 直接 v-for 这个,view 不再二次过滤)。
     * 匹配规则:
     *   - keyword(忽略大小写)在 display_name 或 tool_id 中任一包含
     *   - source:
     *       'all'    → 不过滤
     *       'system' → 仅 is_system=true
     *       'user'   → 仅 is_system=false
     */
    filteredItems: (s) => {
      const kw = (s.filter.keyword || '').trim().toLowerCase()
      return s.items.filter((x) => {
        if (s.filter.source === 'system' && !x.is_system) return false
        if (s.filter.source === 'user' && x.is_system) return false
        if (!kw) return true
        return (
          (x.display_name || '').toLowerCase().includes(kw) ||
          (x.tool_id || '').toLowerCase().includes(kw)
        )
      })
    },
  },

  actions: {
    // ─── 列表 ──────────────────────────────────────────────────────

    /** 拉取工具列表,失败写 error。 */
    async load() {
      this.loading = true
      this.error = ''
      try {
        const res = await listTools()
        this.items = res?.items || []
      } catch (e) {
        this.error = e?.message || String(e)
        throw e
      } finally {
        this.loading = false
      }
    },

    /** 重新加载 skilladapter.Registry(让已存在的改动对 adapter 立即生效)。 */
    async reloadRegistry() {
      this.reloading = true
      try {
        await reloadTools()
      } catch (e) {
        this.error = e?.message || String(e)
        throw e
      } finally {
        this.reloading = false
      }
    },

    // ─── 过滤(纯客户端) ───────────────────────────────────────────

    setKeyword(kw) {
      this.filter.keyword = kw || ''
    },

    setSource(src) {
      if (src !== 'all' && src !== 'system' && src !== 'user') return
      this.filter.source = src
    },

    // ─── 新建 / 编辑 弹窗 ─────────────────────────────────────────

    /**
     * 打开新建弹窗。form 一律 reset,避免上次编辑数据残留。
     */
    openCreate() {
      this.formMode = 'create'
      this.form = emptyForm()
      this.editingToolId = ''
      this.editingIsSystem = false
      this.formOpen = true
    },

    /**
     * 打开编辑弹窗。tool 表单数据 prefill;记录 editingToolId 用作 update locator。
     * @param {ToolView} t
     */
    openEdit(t) {
      this.formMode = 'edit'
      this.form = {
        tool_id: t.tool_id || '',
        display_name: t.display_name || '',
        mdi_icon: t.mdi_icon || 'mdi:',
        icon_file: t.icon_file || '',
        // 后端如果没设 maturity 返 "" → 前端默认 stable,提交时也允许 ""
        maturity: t.maturity || 'stable',
        note: t.note || '',
        enabled: !!t.enabled,
        sort_order: typeof t.sort_order === 'number' ? t.sort_order : 0,
        paths: (t.paths || []).map((p) => ({
          scope: p.scope,
          category: p.category,
          path: p.path,
          path_order: typeof p.path_order === 'number' ? p.path_order : 0,
        })),
      }
      this.editingToolId = t.tool_id
      this.editingIsSystem = !!t.is_system
      this.formOpen = true
    },

    /** 关闭弹窗(不重置 form,view 切换进出不影响)。 */
    closeForm() {
      this.formOpen = false
    },

    /** form.paths 追加一行(取当前 length 作为默认 path_order)。 */
    addPathRow() {
      this.form.paths.push(emptyPathRow(this.form.paths.length))
    },

    /**
     * 删一行 path,顺手重排 path_order 让列表保持 0..N-1 连续。
     * @param {number} i 行下标
     */
    removePathRow(i) {
      this.form.paths.splice(i, 1)
      this.form.paths.forEach((p, idx) => {
        p.path_order = idx
      })
    },

    /**
     * 提交表单(create / update 统一入口)。
     * 校验在前端做一遍(避免明显错误打后端),后端仍有兜底校验。
     * 成功后:重拉列表 + reloadRegistry(让 adapter 立刻生效)。
     */
    async submitForm() {
      const err = this.validateForm()
      if (err) {
        this.error = err
        throw new Error(err)
      }
      this.error = ''
      this.saving = true
      try {
        const payload = this.buildPayload()
        if (this.formMode === 'create') {
          await createTool(payload)
        } else {
          // update 语义:只把"变化字段"带过去;paths 也总带,后端非 null = 整组覆盖
          await updateTool({ tool_id: this.editingToolId, ...payload })
        }
        this.formOpen = false
        await this.load()
        // reload 是 best-effort:失败也不影响主保存(数据已落 DB,后端启动时也会自动 Reload)
        try {
          await this.reloadRegistry()
        } catch (_) {
          // ignore:列表已刷新,reload 留给后端后续启动
        }
      } catch (e) {
        this.error = e?.message || String(e)
        throw e
      } finally {
        this.saving = false
      }
    },

    /**
     * 切换 enabled。立刻调 update + reload,UI 同步通过 await load 刷新卡片。
     * @param {ToolView} t
     */
    async toggleEnabled(t) {
      if (!t || !t.tool_id) return
      this.saving = true
      this.error = ''
      try {
        await updateTool({ tool_id: t.tool_id, enabled: !t.enabled })
        await this.load()
        try {
          await this.reloadRegistry()
        } catch (_) {
          // ignore
        }
      } catch (e) {
        this.error = e?.message || String(e)
        throw e
      } finally {
        this.saving = false
      }
    },

    // ─── 删除确认 ────────────────────────────────────────────────

    askDelete(t) {
      this.confirmTarget = t
      this.confirmOpen = true
    },

    cancelDelete() {
      this.confirmOpen = false
      this.confirmTarget = null
    },

    async confirmDelete() {
      const t = this.confirmTarget
      if (!t) return
      // 双保险:即便 UI 漏了 v-if,系统工具也走不到这一步
      if (t.is_system) {
        this.error = '系统工具不可删'
        this.cancelDelete()
        return
      }
      this.removing = true
      this.error = ''
      try {
        await deleteTool(t.tool_id)
        this.confirmOpen = false
        this.confirmTarget = null
        await this.load()
        try {
          await this.reloadRegistry()
        } catch (_) {
          // ignore
        }
      } catch (e) {
        this.error = e?.message || String(e)
        throw e
      } finally {
        this.removing = false
      }
    },

    // ─── 内部:校验 ────────────────────────────────────────────────

    /**
     * 校验表单,返 null 表示 OK,否则返第一条错误信息。
     * 注意:系统工具的 tool_id 在 view 层 :disabled,这里不强校验。
     */
    validateForm() {
      const f = this.form
      if (this.formMode === 'create' && !String(f.tool_id || '').trim()) {
        return 'tool_id 不能为空'
      }
      if (!String(f.display_name || '').trim()) {
        return 'display_name 不能为空'
      }
      const mdi = String(f.mdi_icon || '').trim()
      const ico = String(f.icon_file || '').trim()
      // mdi_icon 和 icon_file 至少要有一个非空;mdi_icon 必须以 mdi: 开头
      if (!mdi && !ico) {
        return 'mdi_icon / custom icon 不能同时为空'
      }
      if (mdi && !mdi.startsWith('mdi:')) {
        return 'mdi_icon 必须以 mdi: 开头'
      }
      if (f.maturity && !ALLOWED_MATURITY.includes(f.maturity)) {
        return `maturity 必须是 ${ALLOWED_MATURITY.join('/')}`
      }
      for (let i = 0; i < f.paths.length; i++) {
        const p = f.paths[i]
        if (!String(p.path || '').trim()) {
          return `paths[${i}].path 不能为空`
        }
        if (!ALLOWED_SCOPE.includes(p.scope)) {
          return `paths[${i}].scope 必须是 global/project`
        }
        if (!ALLOWED_CATEGORY.includes(p.category)) {
          return `paths[${i}].category 必须是 user/system`
        }
      }
      return null
    },

    /**
     * 把当前表单打包成 update / create 用的 payload。
     * update 语义:paths 字段总是带上(非 null = 整组覆盖,后端会清空再写);
     * mdi_icon 和 icon_file 也总是带上,后端会对比处理(都是空就报错)。
     */
    buildPayload() {
      const f = this.form
      return {
        display_name: f.display_name.trim(),
        mdi_icon: f.mdi_icon.trim(),
        icon_file: f.icon_file.trim(),
        maturity: f.maturity || 'stable',
        note: f.note || '',
        enabled: !!f.enabled,
        sort_order: Number(f.sort_order) || 0,
        paths: f.paths.map((p, idx) => ({
          scope: p.scope,
          category: p.category,
          path: String(p.path || '').trim(),
          path_order: typeof p.path_order === 'number' ? p.path_order : idx,
        })),
        // create 模式才带 tool_id(edit 模式从 editingToolId 取)
        ...(this.formMode === 'create' ? { tool_id: f.tool_id.trim() } : {}),
      }
    },
  },
})
