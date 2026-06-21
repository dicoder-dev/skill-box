// skillbox/market.js - 三方市场域的 HTTP 客户端。
//
// 后端路径:
//   GET  /api/skillbox/market/sources
//   GET  /api/skillbox/market/skills?source_id=&keyword=&page=&size=
//   POST /api/skillbox/market/refresh
//   POST /api/skillbox/market/install

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

export function installMarketSkill(payload) {
  return http.post('/api/skillbox/market/install', payload)
}
