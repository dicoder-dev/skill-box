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
  renameGroup as apiRenameGroup,
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
      const resp = await apiListSkills({ keyword: keyword.value || undefined, page: 1, size: 1000 })
      tree.value = resp?.tree || []
      // 搜索时:自动展开匹配路径(让结果可见)
      if (keyword.value) {
        autoExpandMatchedPaths()
      }
    } catch (e) {
      error.value = e?.message || String(e)
    } finally {
      loading.value = false
    }
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
  }

  function setDropTarget(path) {
    dropTargetPath.value = path || ''
  }

  return {
    // state
    tree, loading, error, keyword, collapsedPaths, dropTargetPath, selectedPath,
    // getters
    flatItems, totalSkills,
    // actions
    load, createGroup, deleteGroup, moveSkill, renameGroup,
    toggleCollapse, setSelected, setDropTarget,
    // helpers(供外部乐观更新)
    removeSkillByPath, removeGroupByPath, moveSkillInTree,
  }
})
