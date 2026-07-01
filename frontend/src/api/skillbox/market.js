// skillbox/market.js - 三方市场域的 HTTP 客户端。
//
// 后端路径(2026-07-01 改:术语 install → pull,新增 skills-remote 端点):
//   GET  /api/skillbox/market/sources                          (旧)
//   GET  /api/skillbox/market/skills?source_id=&keyword=...    (旧)
//   POST /api/skillbox/market/refresh                          (旧)
//   POST /api/skillbox/market/install                          (旧,deprecated,只写盘不 apply)
//   POST /api/skillbox/market/install-v2                       (新,写盘+apply 一站式)
//   GET  /api/skillbox/market/skills-with-installed            (旧,读本地缓存)
//   GET  /api/skillbox/market/skills-remote                    (新,纯远端,不读缓存)
//   GET  /api/skillbox/market/sources/aggregated               (旧,源 + skill_count + last_fetched_at)
//   POST /api/skillbox/market/sources/:id/update               (新,局部更新 enabled/config_json)
//
// 2026-07-01 改造:全走 API。前端统一调 listMarketSkillsRemote,旧缓存端点保留。

import { http } from '@/core/utils/requests'

// 2026-07-01 改:listMarketSkillsRemote 超时对齐后端 90s。
// 后端 ctx 超时 90s(见 list_skills_remote.a.go),前端 15s 默认太短会被 fetch
// 先杀,看到的就是"网络异常"而不是后端真实返回。统一 90s 让两端窗口一致,
// 用户感知到的就是"远端比较慢"而不是前端假死/后端未响应。
// 90s 来自 skillhub 去 maxDiscoverItems 上限后,翻页到 total 全部拉完的实测上限。
const MARKET_REMOTE_TIMEOUT_MS = 90_000

export function listSources() {
  return http.get('/api/skillbox/market/sources')
}

export function listMarketSkills(params = {}) {
  return http.get('/api/skillbox/market/skills', params)
}

// 触发三方源刷新(2026-07-01 改:支持 keyword 透传到三方源搜索)。
// opts.keyword: 空 = 拉全量目录;非空 = 三方源搜索语义。
export function refreshSource(sourceId, opts = {}) {
  return http.post('/api/skillbox/market/refresh', {
    source_id: sourceId,
    keyword: opts.keyword || '',
  })
}

// 旧 install 端点(2026-06-30 标 deprecated):只写盘不 apply。生产请改用 pullMarketSkillV2。
export function installMarketSkill(payload) {
  return http.post('/api/skillbox/market/install', payload)
}

// --- 2026-06-30 增 ---

// 带 installed 标记的列表(2026-06-30 增)。响应多 installed map。
export function listMarketSkillsWithInstalled(params = {}) {
  return http.get('/api/skillbox/market/skills-with-installed', params)
}

// 纯远端列表(2026-07-01 增):不读本地缓存,每次打三方源。
// 返回 schema 与 listMarketSkillsWithInstalled 一致(沿用 installed map)。
// skillhub 走 /api/skills?keyword= 真实搜索语义;
// skills.sh 走 50 页 /api/audits + substring(API 无搜索参数)。
//
// 2026-07-01 改:显式传 45s timeout,覆盖 http 客户端默认 15s。
// 见 MARKET_REMOTE_TIMEOUT_MS 注释 — 对齐后端 ctx 超时窗口。
export function listMarketSkillsRemote(params = {}) {
  return http.get('/api/skillbox/market/skills-remote', params, {
    timeout: MARKET_REMOTE_TIMEOUT_MS,
  })
}

// 一键拉取:写盘 + apply(2026-07-01 改名:installMarketSkillV2 → pullMarketSkillV2)。
// payload: { source_id, remote_id, scope, project_id, tools, final_name }
export function pullMarketSkillV2(payload) {
  return http.post('/api/skillbox/market/install-v2', payload)
}

// installMarketSkillV2 旧名 alias(2026-07-01 deprecated),新代码请用 pullMarketSkillV2。
export const installMarketSkillV2 = pullMarketSkillV2

// 源聚合列表(2026-06-30 增):每个源带 skill_count / last_fetched_at。
export function listMarketSourcesAggregated() {
  return http.get('/api/skillbox/market/sources/aggregated')
}

// 局部更新源(2026-06-30 增):支持 enabled / config_json 单独改。
// payload: { enabled?: boolean, config_json?: string }
export function updateMarketSource(sourceId, payload) {
  return http.post(`/api/skillbox/market/sources/${sourceId}/update`, payload)
}
