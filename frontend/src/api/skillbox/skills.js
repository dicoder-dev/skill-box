// skillbox/skills.js - Skill 域的 HTTP 客户端。
//
// 后端路径:
//   GET    /api/skillbox/skills?scope=&project_id=&keyword=&page=&size=
//   GET    /api/skillbox/skills/get?scope=&project_id=&name=&version=&full=
//   POST   /api/skillbox/skills/create
//   POST   /api/skillbox/skills/update
//   POST   /api/skillbox/skills/delete

//   GET    /api/skillbox/skills/scope-status?name=&version=

import { http } from '@/core/utils/requests'

export function listSkills(params = {}) {
  return http.get('/api/skillbox/skills', params)
}

export function getSkill(params) {
  return http.get('/api/skillbox/skills/get', { ...params, full: params.full ? 1 : undefined })
}

/**
 * 实时扫描所有 adapter 路径,返回某 skill 在 (tool, scope, project) 笛卡尔积下
 * 哪些位置真实存在 SKILL.md。纯文件系统检查,无 DB 写入。
 * 响应: { name, version, tools: [...], projects: [...], hits: [...] }
 */
export function getSkillScopeStatus(params) {
  return http.get('/api/skillbox/skills/scope-status', params)
}

/**
 * 把 skillbox 库里的 skill 复制到目标工具的 (scope, project) 位置。
 * 入参: { scope, project_id, name, tools: [toolID] }
 * 响应: { name, version, applies: [{tool, target_path, status, apply_id, ...}], all_ok }
 * 注意:同名已存在时直接覆盖(走 skillapp 内置的 PreSnapshot + 原子写)。
 * 来源:api-server/internal/gapi/controller/skillbox/cskillapply/apply_skill.a.go
 */
export function applySkill(payload) {
  return http.post('/api/skillbox/skills/apply', payload)
}

/**
 * 列出 skill 的 apply 历史,用于在 unapply 时找到最近一条未撤销的 apply 行。
 * 入参: { scope, name, tool, status(可选 'applied'/'rolled_back'), page, size }
 * 响应: { items: [{id, tool, scope, project_id, name, target_path, status, ...}], total, ... }
 * 注:行主键 json 字段是 "id"(不是 "apply_id"),前端用 last.id 取出来再调 undoApply。
 */
export function listApplies(params) {
  return http.get('/api/skillbox/skills/apply/list', params)
}

/**
 * 撤销一条 apply(按 apply_id);恢复 PreSnapshot 或删除目标文件。
 * 入参: { apply_id }
 * 来源:api-server/internal/gapi/controller/skillbox/cskillapply/undo_skill.a.go
 */
export function undoApply(payload) {
  return http.post('/api/skillbox/skills/apply/undo', payload)
}

export function createSkill(payload) {
  return http.post('/api/skillbox/skills/create', payload)
}

export function updateSkill(payload) {
  return http.post('/api/skillbox/skills/update', payload)
}

export function deleteSkill(payload) {
  return http.post('/api/skillbox/skills/delete', payload)
}
