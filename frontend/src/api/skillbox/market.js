// skillbox/market.js - 三方市场域的 HTTP 客户端。
//
// 后端路径(2026-06-30 增 4 个新端点):
//   GET  /api/skillbox/market/sources                          (旧)
//   GET  /api/skillbox/market/skills?source_id=&keyword=...    (旧)
//   POST /api/skillbox/market/refresh                          (旧)
//   POST /api/skillbox/market/install                          (旧,deprecated,只写盘不 apply)
//   POST /api/skillbox/market/install-v2                       (新,写盘+apply 一站式)
//   GET  /api/skillbox/market/skills-with-installed            (新,带 installed 标记)
//   GET  /api/skillbox/market/sources/aggregated               (新,源 + skill_count + last_fetched_at)
//   POST /api/skillbox/market/sources/:id/update               (新,局部更新 enabled/config_json)

import { http } from '@/core/utils/requests'

export function listSources() {
  return http.get('/api/skillbox/market/sources')
}

export function listMarketSkills(params = {}) {
  return http.get('/api/skillbox/market/skills', params)
}

export function refreshSource(sourceId) {
  return http.post('/api/skillbox/market/refresh', { source_id: sourceId })
}

// 旧 install 端点(2026-06-30 标 deprecated):只写盘不 apply。生产请改用 installMarketSkillV2。
export function installMarketSkill(payload) {
  return http.post('/api/skillbox/market/install', payload)
}

// --- 2026-06-30 增 ---

// 带 installed 标记的列表(2026-06-30 增)。响应多 installed map。
export function listMarketSkillsWithInstalled(params = {}) {
  return http.get('/api/skillbox/market/skills-with-installed', params)
}

// 一键安装:写盘 + apply(2026-06-30 增)。
// payload: { source_id, remote_id, scope, project_id, tools, final_name }
export function installMarketSkillV2(payload) {
  return http.post('/api/skillbox/market/install-v2', payload)
}

// 源聚合列表(2026-06-30 增):每个源带 skill_count / last_fetched_at。
export function listMarketSourcesAggregated() {
  return http.get('/api/skillbox/market/sources/aggregated')
}

// 局部更新源(2026-06-30 增):支持 enabled / config_json 单独改。
// payload: { enabled?: boolean, config_json?: string }
export function updateMarketSource(sourceId, payload) {
  return http.post(`/api/skillbox/market/sources/${sourceId}/update`, payload)
}
