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
    load, createGroup, deleteGroup, moveSkill,
    toggleCollapse, setSelected, setDropTarget,
    // helpers(供外部乐观更新)
    removeSkillByPath, removeGroupByPath, moveSkillInTree,
  }
})
