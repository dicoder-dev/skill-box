// skillbox/skills.js - Skill 域的 HTTP 客户端。
//
// 后端路径:
//   GET    /api/skillbox/skills?scope=&project_id=&keyword=&page=&size=
//   GET    /api/skillbox/skills/get?scope=&project_id=&name=&version=&full=
//   POST   /api/skillbox/skills/create
//   POST   /api/skillbox/skills/update
//   POST   /api/skillbox/skills/delete

import { http } from '@/core/utils/requests'

export function listSkills(params = {}) {
  return http.get('/api/skillbox/skills', params)
}

export function getSkill(params) {
  return http.get('/api/skillbox/skills/get', { ...params, full: params.full ? 1 : undefined })
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
