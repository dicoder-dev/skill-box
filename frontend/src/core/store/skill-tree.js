// core/store/skill-tree.js - 技能树形 store
//
// 2026-06-29 增:为支持首页 skill 列表的多级分组 / 拖拽 / 右键菜单,集中管理
// 树形状态、展开折叠、搜索展开联动、CRUD 编排。
//
// 设计要点:
//   - 树数据来自后端 GET /api/skillbox/skills 的 `tree` 字段(嵌套 TreeNode 数组)
//   - 扁平化从树派生(每次 tree 变化重算),供搜索过滤 + 详情跳转用
//   - 折叠态 / drop 目标态 是 UI 临时态,放在 store 里跨组件共享
//   - CRUD 操作后 reload 整棵树(简单可靠,树规模通常 < 200 节点)
//
// 用法:
//   import { useSkillTreeStore } from '@/core/store/skill-tree'
//   const tree = useSkillTreeStore()
//   await tree.load({ keyword: 'react' })
//   await tree.createGroup('frontend/react')
//   await tree.moveSkill({ src: 'a/b', name: 'use-cache', dst: 'c/d' })
//   await tree.deleteGroup('frontend', { cascade: true })

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import {
  listSkills as apiListSkills,
  createGroup as apiCreateGroup,
  deleteGroup as apiDeleteGroup,
  moveSkill as apiMoveSkill,
  moveGroup as apiMoveGroup,
  renameGroup as apiRenameGroup,
  getStoreInfo as apiGetStoreInfo,
} from '@/api/skillbox/skills'

// 一个 TreeNode 的最小形态(对应后端 skillstore.TreeNode)
// {
//   name, path, is_group,
//   children?: TreeNode[],
//   skill_meta?: { name, version, description, triggers, applied_tools }
// }

export const useSkillTreeStore = defineStore('skill-tree', () => {
  // 状态
  const tree = ref([]) // 顶层 TreeNode 数组
  const loading = ref(false)
  const error = ref('')
  const keyword = ref('')
  // 折叠态(用 Set 记录所有当前折叠的 path)
  const collapsedPaths = ref(new Set())
  // 拖拽中:当前 drop 目标(高亮)
  const dropTargetPath = ref('')
  // 选中 skill 的 path(供详情区联动)
  const selectedPath = ref('')
  // 2026-07-09 增:待选中的 skill name(由 MarketView 等外部组件设置)
  // 解决:MarketView 装好跳 skills tab 时,SkillsView 可能还没 mount,
  // 事件就丢了。这里存个"待选清单",SkillsView mount 后 + list 加载完时检查一次。
  const pendingSelectName = ref('')

  // 2026-07-12 增:选中态跨 tab 持久化。
  // 原 selectedPath 是 pinia 的内存 ref,切 tab 时 SkillsView 整体
  // unmount → 销毁(组件级 selectedKey / current 一起没)→ 回到
  // skills tab 时走到 reload 的"自动选第一个"分支,而不是恢复用户
  // 上次选中的 skill。这里把最后一次有效选中 path 同步写一份到
  // localStorage,SkillsView 重新 mount + reload 时优先按这个 path
  // 找回 row,失败再 fallback 到自动选第一个。
  // key 加 storeId 前缀,未来如果引入多 store 不串数据;
  // 失败静默(无痕模式 / 存储满)不阻断 UI。
  const STORAGE_KEY = 'skillbox:skill-tree:last-selected-path'
  function readPersistedSelected() {
    try {
      return localStorage.getItem(STORAGE_KEY) || ''
    } catch (_) {
      return ''
    }
  }
  function writePersistedSelected(path) {
    try {
      if (path) localStorage.setItem(STORAGE_KEY, path)
      else localStorage.removeItem(STORAGE_KEY)
    } catch (_) { /* 静默失败:无痕模式 / 存储满都不阻断 UI */ }
  }
  // 初始化时从 localStorage 恢复一次,确保 store 重新创建(pinia 热重载、
  // 浏览器刷新)后第一次读 selectedPath 就有值,而不是空。
  // 注意:此处写 selectedPath.value 不会触发持久化(下面 setSelected 里
  // 再写等于双写,直接跳过避免重复 IO)。
  const _initialLastSelected = readPersistedSelected()
  if (_initialLastSelected) selectedPath.value = _initialLastSelected
  const storeRoot = ref('')
  const storeRootLoaded = ref(false)

  // 派生:扁平化(只取 skill 叶子),按 group_path 排序后,按 name 排序
  const flatItems = computed(() => {
    const out = []
    const walk = (nodes) => {
      for (const n of nodes || []) {
        if (!n.is_group) {
          out.push(n)
        } else {
          walk(n.children)
        }
      }
    }
    walk(tree.value)
    return out
  })

  // 派生:总 skill 数(供 badge / 统计)
  const totalSkills = computed(() => flatItems.value.length)

  // 2026-07-08 增:首页默认打开第一个技能 — 找"根目录下第一个 skill 叶子";
  // 若根目录下没有 skill,则找第一个 group,递归进入 group 取第一个 skill。
  // 都找不到返回 null,前端空状态提示用户新建。
  // 顺序策略:跳过 group 节点直接找 !is_group,与后端 sortTreeNodes 的
  // "(IsGroup desc, Name asc)" 排序一致 — 根下第一个 skill = tree 中第一个
  // is_group=false 节点(同组内按字典序);若不存在,递归进第一个 group。
  function findFirstSelectableNode() {
    const nodes = tree.value || []
    // 第一步:根目录下找第一个 skill 叶子
    for (const n of nodes) {
      if (!n.is_group) return n
    }
    // 第二步:根目录下没有 skill,找第一个 group,递归取第一个 skill
    const walk = (list) => {
      for (const n of list || []) {
        if (!n.is_group) return n
        if (n.is_group && n.children) {
          const inner = walk(n.children)
          if (inner) return inner
        }
      }
      return null
    }
    for (const n of nodes) {
      if (n.is_group && n.children) {
        const found = walk(n.children)
        if (found) return found
      }
    }
    return null
  }

  // 工具:从 tree 移除一个 skill 节点(乐观更新,失败时 reload)
  function removeSkillByPath(path) {
    const removeIn = (nodes) => {
      for (let i = 0; i < nodes.length; i++) {
        const n = nodes[i]
        if (!n.is_group && n.path === path) {
          nodes.splice(i, 1)
          return true
        }
        if (n.is_group && n.children) {
          if (removeIn(n.children)) return true
        }
      }
      return false
    }
    removeIn(tree.value)
  }

  // 工具:从 tree 移除一个分组节点
  function removeGroupByPath(path) {
    const idx = tree.value.findIndex((n) => n.is_group && n.path === path)
    if (idx >= 0) {
      tree.value.splice(idx, 1)
      return
    }
    // 嵌套分组:递归删
    const removeIn = (nodes) => {
      for (let i = 0; i < nodes.length; i++) {
        const n = nodes[i]
        if (n.is_group) {
          if (n.path === path) {
            nodes.splice(i, 1)
            return true
          }
          if (n.children && removeIn(n.children)) return true
        }
      }
      return false
    }
    removeIn(tree.value)
  }

  // 工具:把一个 skill 节点从 src 移到 dst group 的 children
  function moveSkillInTree(srcPath, dstGroupPath, skillNode) {
    removeSkillByPath(srcPath)
    const insertTo = (nodes) => {
      for (const n of nodes) {
        if (n.is_group && n.path === dstGroupPath) {
          if (!n.children) n.children = []
          n.children.push(skillNode)
          return true
        }
        if (n.is_group && n.children && insertTo(n.children)) return true
      }
      return false
    }
    // dst 是根(空 path)→ 直接 push 到顶层
    if (!dstGroupPath) {
      tree.value.push(skillNode)
      return
    }
    insertTo(tree.value)
  }

  // ====== 加载 ======

  async function load({ keyword: kw } = {}) {
    loading.value = true
    error.value = ''
    try {
      if (typeof kw === 'string') keyword.value = kw
      // 2026-07-03 增:首次 load 时预热 storeRoot(后续 reload 不再重拉,
      // 避免每个关键字切换都多发一次请求)。失败时打 console.warn,不影响 tree 加载。
      if (!storeRootLoaded.value) {
        try {
          const info = await apiGetStoreInfo()
          storeRoot.value = info?.store_root || ''
        } catch (e) {
          console.warn('[skill-tree] getStoreInfo failed:', e?.message || e)
        } finally {
          storeRootLoaded.value = true
        }
      }
      const resp = await apiListSkills({ keyword: keyword.value || undefined, page: 1, size: 1000 })
      tree.value = resp?.tree || []
      // 2026-07-10 改:首页分组默认折叠。
      // 原因:用户反馈首次进入首页时所有 group 处于展开状态,树又长又乱。
      // 这里在 load 完成且非搜索模式下,把 tree 中所有 group path 收集到 collapsedPaths,
      // 实现"默认全折叠"。Set 的 add 是幂等的,reload/手动展开过的 group 也会被覆盖折叠,
      // 这是用户主动选择的产品策略(刷新即重置为折叠初始态),不算 bug。
      // 搜索时走 autoExpandMatchedPaths 自动展开匹配路径,不受本逻辑影响。
      if (!keyword.value) {
        collapseAllGroups()
      }
      // 搜索时:自动展开匹配路径(让结果可见)
      if (keyword.value) {
        autoExpandMatchedPaths()
      }
      // 2026-07-16 增:load 后,如果之前已有选中节点,把它祖先分组展开。
      // 场景:用户点击右侧 ScopePanel 的"全局 Agent"开关 → 派发
      // skillbox:scope-refresh → SkillsView.onScopeChange 调 skillTree.load
      // → 非搜索态下 collapseAllGroups 整体折叠,导致当前选中的 skill 节点
      // 被埋进折叠组里,看起来"左侧目录被关掉了"。这里复用 expandAncestorsOfPath,
      // 把选中节点的所有祖先 group 从 collapsedPaths 移除,保证选中节点可见。
      // 注意:放在 collapseAllGroups / autoExpandMatchedPaths 之后,优先遵循
      // 用户显式的"全部折叠"语义失败兜底(被展开的只是选中节点祖先,不是全部);
      // 搜索态下 selectedPath 通常已被清空(用户切了关键字),if 直接跳过。
      // expandAncestorsOfPath 内部还会校验目标在 tree 里确实是叶子 skill,
      // 找不到或路径异常时静默返回,不会乱展开。
      if (selectedPath.value) {
        expandAncestorsOfPath(selectedPath.value)
      }
      // 2026-07-16 增:load 后,如果之前已有选中节点,把它祖先分组展开。
      // 场景:用户点击右侧 ScopePanel 的"全局 Agent"开关 → 派发
      // skillbox:scope-refresh → SkillsView.onScopeChange 调 skillTree.load
      // → 非搜索态下 collapseAllGroups 整体折叠,导致当前选中的 skill 节点
      // 被埋进折叠组里,看起来"左侧目录被关掉了"。这里复用 expandAncestorsOfPath,
      // 把选中节点的所有祖先 group 从 collapsedPaths 移除,保证选中节点可见。
      // 注意:放在 collapseAllGroups / autoExpandMatchedPaths 之后,优先遵循
      // 用户显式的"全部折叠"语义失败兜底(被展开的只是选中节点祖先,不是全部);
      // 搜索态下 selectedPath 通常已被清空(用户切了关键字),if 直接跳过。
      // expandAncestorsOfPath 内部还会校验目标在 tree 里确实是叶子 skill,
      // 找不到或路径异常时静默返回,不会乱展开。
    } catch (e) {
      error.value = e?.message || String(e)
    } finally {
      loading.value = false
    }
  }

  // 2026-07-10 增:把 tree 中所有 group path 全部加入 collapsedPaths(默认折叠)。
  // 跟 autoExpandMatchedPaths 互为反向操作 — 一个展开匹配组,一个折叠全部组。
  // 抽出独立函数,便于未来 toggle 默认行为的开关(比如加个 "默认展开" 设置项)。
  function collapseAllGroups() {
    const paths = new Set()
    const collectGroupPaths = (node, out) => {
      if (!node.is_group) return
      out.add(node.path)
      for (const c of node.children || []) collectGroupPaths(c, out)
    }
    for (const n of tree.value || []) collectGroupPaths(n, paths)
    // 整体替换(触发响应式),而不是逐个 add
    collapsedPaths.value = new Set(paths)
  }

  // 自动展开所有包含匹配 skill 的分组(搜索时用)
  function autoExpandMatchedPaths() {
    const paths = new Set()
    const walk = (nodes) => {
      for (const n of nodes || []) {
        if (n.is_group) {
          if (n.children?.some((c) => !c.is_group || c.children?.length)) {
            // 该分组有子树,逐层收集 path
            collectGroupPaths(n, paths)
          }
          walk(n.children)
        }
      }
    }
    // 收集所有有 skill 后代的分组 path
    const collectGroupPaths = (node, out) => {
      if (!node.is_group) return
      const hasSkill = (n) => !n.is_group || (n.children && n.children.some(hasSkill))
      if (node.children?.some(hasSkill)) {
        out.add(node.path)
        for (const c of node.children || []) collectGroupPaths(c, out)
      }
    }
    walk(tree.value)
    // 把这些 path 从 collapsed 集合中移除(展开)
    for (const p of paths) collapsedPaths.value.delete(p)
    // 触发响应式更新
    collapsedPaths.value = new Set(collapsedPaths.value)
  }

  // ====== 分组操作 ======

  async function createGroup(groupPath) {
    try {
      const resp = await apiCreateGroup({ group_path: groupPath })
      const norm = resp?.group_path || groupPath
      await load({ keyword: keyword.value })
      return { ok: true, group_path: norm }
    } catch (e) {
      return { ok: false, error: e?.message || String(e) }
    }
  }

  // deleteGroup 删除分组(可级联)。opts.cascade=true 时同时删子树。
  // 失败时回传 deleted_skill_paths(后端在 cascade=false 非空时返回 409 + 列表)
  async function deleteGroup(groupPath, { cascade = false } = {}) {
    try {
      await apiDeleteGroup({ group_path: groupPath, cascade })
      await load({ keyword: keyword.value })
      return { ok: true, deleted_skill_paths: [] }
    } catch (e) {
      // 业务错误(后端返 409 业务码或带 deleted_skill_paths)
      const data = e?.response?.data || e?.data
      if (data?.need_cascade && Array.isArray(data?.deleted_skill_paths)) {
        return { ok: false, need_cascade: true, deleted_skill_paths: data.deleted_skill_paths }
      }
      return { ok: false, error: e?.message || String(e) }
    }
  }

  // ====== 移动 ======

  async function moveSkill({ srcPath, srcGroupPath, name, dstGroupPath }) {
    try {
      await apiMoveSkill({
        src_group_path: srcGroupPath,
        dst_group_path: dstGroupPath,
        name,
      })
      await load({ keyword: keyword.value })
      return { ok: true }
    } catch (e) {
      return { ok: false, error: e?.message || String(e) }
    }
  }

  // 移动整个分组到另一分组下(子路径)。
  // 2026-06-29 增:之前用 moveSkill + name='__group__' sentinel 临时绕过,
  // 后端会返 not found 409;现在走独立 moveGroup 接口。
  async function moveGroup({ srcGroupPath, dstGroupPath }) {
    if (!srcGroupPath) return { ok: false, error: 'src group path is empty' }
    try {
      await apiMoveGroup({
        src_group_path: srcGroupPath,
        dst_group_path: dstGroupPath,
      })
      await load({ keyword: keyword.value })
      return { ok: true }
    } catch (e) {
      return { ok: false, error: e?.message || String(e) }
    }
  }

  // renameGroup 重命名分组(只改最后一段,父路径不变)。
  // 后端返回 new_group_path;前端用乐观更新改 tree 内对应节点的 path/name + 子树所有 path 前缀,
  // 失败时整体 reload 回滚。
  async function renameGroup({ srcGroupPath, newName }) {
    if (!srcGroupPath || !newName) return { ok: false, error: 'empty params' }
    const oldBase = srcGroupPath.split('/').pop()
    if (oldBase === newName) {
      // 同名,后端会幂等返 OK;前端不动 state
      return { ok: true, new_group_path: srcGroupPath }
    }
    // 乐观更新:把 srcGroupPath 在 tree 内所有出现的位置改掉(节点自身 + 子树所有 path)
    // 失败时会 reload 回滚,先快照 oldPaths 用于回滚
    const oldPaths = collectAllPathsUnderGroup(tree.value, srcGroupPath)
    applyGroupRenameInTree(srcGroupPath, newName)
    try {
      const resp = await apiRenameGroup({ src_group_path: srcGroupPath, new_name: newName })
      const norm = resp?.new_group_path || `${pathDirname(srcGroupPath)}/${newName}`
      // 同步把 state 里的 selectedPath / collapsedPaths / dropTargetPath 里的旧前缀换新
      rewriteGroupPathRefs(srcGroupPath, norm)
      return { ok: true, new_group_path: norm }
    } catch (e) {
      // 回滚:把乐观更新改回去
      revertGroupRenameInTree(oldPaths)
      const status = e?.response?.status
      const data = e?.response?.data || e?.data
      const code = data?.code
      if (status === 409 || code === 'target_exists') {
        return { ok: false, code: 'target_exists', error: data?.error || 'target already exists' }
      }
      if (status === 404) {
        return { ok: false, code: 'not_found', error: data?.error || 'source not found' }
      }
      return { ok: false, error: e?.message || String(e) }
    }
  }

  // 工具:收集 srcGroupPath 分组子树里所有旧 path(供回滚用)
  function collectAllPathsUnderGroup(nodes, groupPath) {
    const out = []
    const walk = (ns, parentPath) => {
      for (const n of ns || []) {
        const full = parentPath ? `${parentPath}/${n.name}` : n.name
        if (n.path === groupPath) {
          // 命中目标分组 → 整子树 dump
          dumpSubtree(n, full, out)
          continue
        }
        if (n.is_group) walk(n.children, full)
      }
    }
    walk(nodes, '')
    return out
  }
  function dumpSubtree(n, parentPath, out) {
    out.push({ oldPath: n.path, oldName: n.name, parentPath })
    for (const c of n.children || []) {
      dumpSubtree(c, `${parentPath}/${c.name}`, out)
    }
  }

  // 工具:把 srcGroupPath → newName 的整组子树 path/name 在 tree 内重写
  function applyGroupRenameInTree(srcGroupPath, newName) {
    const walk = (nodes, parentPath) => {
      for (const n of nodes || []) {
        const full = parentPath ? `${parentPath}/${n.name}` : n.name
        if (n.path === srcGroupPath) {
          // 改自身
          n.name = newName
          n.path = parentPath ? `${parentPath}/${newName}` : newName
          // 改子树所有 path
          rewriteSubtreePaths(n, n.path)
          return true
        }
        if (n.is_group) {
          if (walk(n.children, full)) return true
        }
      }
      return false
    }
    walk(tree.value, '')
  }
  function rewriteSubtreePaths(n, newParentPath) {
    if (!n.children) return
    for (const c of n.children) {
      c.path = `${newParentPath}/${c.name}`
      if (c.is_group) rewriteSubtreePaths(c, c.path)
    }
  }

  // 工具:把 rollback 用的旧 path/name 写回 tree
  function revertGroupRenameInTree(oldPaths) {
    // 找到目标分组(原 srcGroupPath 所在位置)用新 path 找,然后把子树恢复
    // 简单策略:重新 load(避免复杂的 tree 重写)
    load({ keyword: keyword.value }).catch(() => {})
  }

  // 工具:把 selectedPath / collapsedPaths / dropTargetPath 里的旧分组前缀换成新
  function rewriteGroupPathRefs(oldGroupPath, newGroupPath) {
    const replace = (p) => {
      if (!p) return p
      if (p === oldGroupPath) return newGroupPath
      if (p.startsWith(oldGroupPath + '/')) return newGroupPath + p.slice(oldGroupPath.length)
      return p
    }
    if (selectedPath.value) selectedPath.value = replace(selectedPath.value)
    if (dropTargetPath.value) dropTargetPath.value = replace(dropTargetPath.value)
    const newCollapsed = new Set()
    for (const p of collapsedPaths.value) newCollapsed.add(replace(p))
    collapsedPaths.value = newCollapsed
  }

  function pathDirname(p) {
    if (!p) return ''
    const i = p.lastIndexOf('/')
    return i < 0 ? '' : p.slice(0, i)
  }

  // ====== 折叠 / 选中 ======

  // 2026-07-12 增:把目标 skill 路径的所有祖先分组从 collapsedPaths 里删除,
  // 让左侧树"展开到该 skill 可见"。背景:store.load 在非搜索态会默认全
  // 折叠(load → collapseAllGroups),用户切到其他 tab 再切回 skills 时,
  // 即使 store 残留 selectedPath,分组还是折叠的,看不到当前选中行,
  // 看起来像"选中态丢了"。这里在选中态恢复后(以及 selectItem 内)
  // 调一次,把祖先分组全部展开。
  // 命中条件:tree 里能找到对应 path 的 skill,且它是叶子节点(非 group)。
  // 否则不展开(避免误展开)。path 为空或找不到都直接返回,不影响主流程。
  function expandAncestorsOfPath(skillPath) {
    if (!skillPath) return
    // 把 "frontend/react/use-cache" → ["frontend", "frontend/react"]
    const segs = String(skillPath).split('/').filter(Boolean)
    if (segs.length < 2) return // 没有分组祖先(直接在根)
    const ancestors = []
    for (let i = 1; i < segs.length; i++) {
      ancestors.push(segs.slice(0, i).join('/'))
    }
    // 校验目标 path 在 tree 里确实是 skill 叶子,避免展开错的分组
    const existsInTree = (nodes, target) => {
      for (const n of nodes || []) {
        if (!n.is_group && n.path === target) return true
        if (n.is_group && n.children && existsInTree(n.children, target)) return true
      }
      return false
    }
    if (!existsInTree(tree.value, skillPath)) return
    let changed = false
    for (const p of ancestors) {
      if (collapsedPaths.value.has(p)) {
        collapsedPaths.value.delete(p)
        changed = true
      }
    }
    if (changed) collapsedPaths.value = new Set(collapsedPaths.value)
  }

  function toggleCollapse(path) {
    if (collapsedPaths.value.has(path)) {
      collapsedPaths.value.delete(path)
    } else {
      collapsedPaths.value.add(path)
    }
    // 触发响应式
    collapsedPaths.value = new Set(collapsedPaths.value)
  }

  function setSelected(path) {
    selectedPath.value = path || ''
    // 2026-07-12 增:每次有效选中都同步落盘一次。空值(清空选中)
    // 也走 removeItem,避免上次选中的 skill 被删后仍残留脏值。
    writePersistedSelected(selectedPath.value)
  }

  // 2026-07-12 增:清空选中态(组件级 selectedKey 清空 / 删除当前 skill /
  // 验证选中失败时调),同步清盘上的持久化值。
  function clearSelected() {
    selectedPath.value = ''
    writePersistedSelected('')
  }

  function setDropTarget(path) {
    dropTargetPath.value = path || ''
  }

  // 2026-07-09 增:外部组件(MarketView)设置"待选 skill name"。
  // SkillsView 在 onMounted + reload 完成后检查并消费(consume 一次即清空)。
  function setPendingSelectName(name) {
    pendingSelectName.value = String(name || '')
  }
  function consumePendingSelectName() {
    const v = pendingSelectName.value
    pendingSelectName.value = ''
    return v
  }

  return {
    // state
    tree, loading, error, keyword, collapsedPaths, dropTargetPath, selectedPath,
    storeRoot, storeRootLoaded, pendingSelectName,
    // getters
    flatItems, totalSkills,
    // actions
    load, createGroup, deleteGroup, moveSkill, moveGroup, renameGroup,
    toggleCollapse, setSelected, clearSelected, setDropTarget, setPendingSelectName, consumePendingSelectName,
    // 2026-07-10 增:折叠/展开所有分组的批量操作
    collapseAllGroups,
    // 2026-07-12 增:把指定 skill 路径的所有祖先分组展开,配合 selectedPath
    // 跨 tab 持久化,保证回到 skills tab 时当前选中的 skill 在左侧树里可见。
    expandAncestorsOfPath,
    // helpers(供外部乐观更新)
    removeSkillByPath, removeGroupByPath, moveSkillInTree,
    // 2026-07-08 增:首页默认打开第一个技能 — 根下首个 skill 叶子,fallback 到首个 group 内的首个 skill
    findFirstSelectableNode,
  }
})
