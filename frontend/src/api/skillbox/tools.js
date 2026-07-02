// skillbox/tools.js - 工具元数据(tools + tool_paths)域 HTTP 客户端。
//
// 后端路径(2026-06-30 上线,替代原 internal/skilladapter/toolspecs/specs/*.yaml):
//   GET  /api/skillbox/tools                       - 列出所有工具(含 paths)
//   POST /api/skillbox/tools/create                - 新建用户工具(is_system 强制 false)
//   POST /api/skillbox/tools/update                - 改工具元数据;paths 非 null = 整组覆盖
//   POST /api/skillbox/tools/delete                - 删工具(系统工具不可删)
//   POST /api/skillbox/tools/paths/add             - 给工具追加一条 path
//   POST /api/skillbox/tools/paths/delete          - 按 id 删一条 path
//   POST /api/skillbox/tools/reload                - 把 DB 重新拉到 skilladapter.Registry
//   POST /api/skillbox/tools/upload-icon           - 上传工具自定义图标(multipart)
//   GET  /api/files/tool-icons/<name>              - 静态服务用户/seed 的图标
//
// 业务规则(后端兜底,前端不重复校验,只做"友好提示"):
//   - is_system=true:tool_id / is_system 不可改,整行不可删;其他字段可改
//   - is_system=false:全部可改 + 可删
//   - maturity ∈ stable | experimental | deprecated
//   - icon_file 和 mdi_icon 至少要有一个非空;两者可同时存在(icon_file 优先)
//   - paths 每条:scope ∈ global|project;category ∈ user|system;path 非空
//
// 字段命名严格跟后端 RequestCreateTool / RequestUpdateTool 一致(snake_case),
// 不在前端做"驼峰 ↔ 蛇形"互转,降低心智负担。

import { http } from '@/core/utils/requests'

/**
 * 列出所有 AI 编程工具元数据(含每条 path)。
 * 返回:Promise<{ items: ToolView[] }>
 *   ToolView = { id, tool_id, display_name, mdi_icon, icon_file, maturity, note,
 *                is_system, enabled, sort_order,
 *                paths: [{ scope, category, path, path_order }],
 *                created_at, updated_at }
 */
export function listTools() {
  return http.get('/api/skillbox/tools')
}

/**
 * 新建一个用户工具(is_system 后端强制 false)。
 * @param {object} payload - 字段同后端 RequestCreateTool:
 *   { tool_id, display_name, mdi_icon, icon_file, maturity, note, enabled, sort_order,
 *     paths: [{ scope, category, path, path_order }] }
 *   paths 字段可省略 / 空数组;icon_file 和 mdi_icon 至少有一个非空。
 * @returns {Promise<ToolView>} 新建的工具视图(含后端分配的 id)。
 */
export function createTool(payload) {
  return http.post('/api/skillbox/tools/create', payload)
}

/**
 * 改一个工具的元数据。空值表示"不改";paths 非 null = 整组覆盖。
 * @param {object} payload - 字段同后端 RequestUpdateTool:
 *   { tool_id (locator, 不可改),
 *     display_name?, mdi_icon?, icon_file?, maturity?, note?, enabled?, sort_order?,
 *     paths?: [{ scope, category, path, path_order }] }
 * @returns {Promise<ToolView>} 更新后的工具视图。
 */
export function updateTool(payload) {
  return http.post('/api/skillbox/tools/update', payload)
}

/**
 * 删一个用户工具。系统工具(is_system=true)会被后端 400 拒绝。
 * @param {string} tool_id 工具 canonical ID。
 */
export function deleteTool(tool_id) {
  return http.post('/api/skillbox/tools/delete', { tool_id })
}

/**
 * 给工具追加一条 path(不覆盖现有);改完建议再调 reloadTools() 生效。
 * @param {object} payload - { tool_id, scope, category, path, path_order }
 * @returns {Promise<ToolPath>} 新建的 path 视图(含后端分配的 id)。
 */
export function addToolPath(payload) {
  return http.post('/api/skillbox/tools/paths/add', payload)
}

/**
 * 按主键 id 删一条 path。
 * @param {number} path_id - e_tool_path.id 主键。
 */
export function deleteToolPath(path_id) {
  return http.post('/api/skillbox/tools/paths/delete', { path_id })
}

/**
 * 把 DB 重新加载到 skilladapter.DefaultRegistry,让 adapter 立刻反映新数据。
 * 前端改完 /create /update /delete /paths.* 后调用一次即可。
 */
export function reloadTools() {
  return http.post('/api/skillbox/tools/reload', {})
}

/**
 * 上传工具自定义图标。
 * @param {File} file - HTML <input type="file"> 选中的图片文件
 * @returns {Promise<{name:string,url:string}>}
 *   - name: basename,如 "claude_1719300123.png";前端再把它写到 tool.icon_file
 *   - url: 服务地址,如 "/api/files/tool-icons/claude_1719300123.png"
 */
export function uploadToolIcon(file) {
  const fd = new FormData()
  fd.append('file', file)
  return http.post('/api/skillbox/tools/upload-icon', fd, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
}
