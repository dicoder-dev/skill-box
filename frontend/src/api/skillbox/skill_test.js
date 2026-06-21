// skillbox/skill_test.js - Skill 测试器 HTTP 客户端。
//
// 后端路径:
//   POST /api/skillbox/skills/test/run
//   GET  /api/skillbox/skills/test/list?skill_id=&page=&size=
//   GET  /api/skillbox/skills/test/get?id=

import { http } from '@/core/utils/requests'

export function runSkillTest(payload) {
  return http.post('/api/skillbox/skills/test/run', payload)
}

export function listSkillTests(params = {}) {
  return http.get('/api/skillbox/skills/test/list', params)
}

export function getSkillTest(id) {
  return http.get('/api/skillbox/skills/test/get', { id })
}
