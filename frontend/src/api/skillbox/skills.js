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

export function createSkill(payload) {
  return http.post('/api/skillbox/skills/create', payload)
}

export function updateSkill(payload) {
  return http.post('/api/skillbox/skills/update', payload)
}

export function deleteSkill(payload) {
  return http.post('/api/skillbox/skills/delete', payload)
}
