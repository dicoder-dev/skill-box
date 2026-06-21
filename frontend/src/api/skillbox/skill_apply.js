// skillbox/skill_apply.js - Skill 应用 / 撤销 / 更新检测 HTTP 客户端。
//
// 后端路径:
//   POST /api/skillbox/skills/apply          - 单 skill 落 1~N 工具
//   POST /api/skillbox/skills/apply/batch    - 批量 (skill × tool 笛卡尔积)
//   POST /api/skillbox/skills/apply/undo     - 撤销一条 apply
//   GET  /api/skillbox/skills/apply/list     - 列 apply 历史
//   GET  /api/skillbox/skills/updates        - 对比本地 vs 三方市场,返回可更新列表

import { http } from '@/core/utils/requests'

export function applySkill(payload) {
  return http.post('/api/skillbox/skills/apply', payload)
}

export function applyBatch(payload) {
  return http.post('/api/skillbox/skills/apply/batch', payload)
}

export function undoApply(payload) {
  return http.post('/api/skillbox/skills/apply/undo', payload)
}

export function listApplies(params = {}) {
  return http.get('/api/skillbox/skills/apply/list', params)
}

export function checkUpdates(params = {}) {
  return http.get('/api/skillbox/skills/updates', params)
}
