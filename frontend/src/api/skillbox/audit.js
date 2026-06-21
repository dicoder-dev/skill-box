// skillbox/audit.js - 审计日志域的 HTTP 客户端(第 10 步)。
//
// 后端路径(待实现):
//   GET  /api/skillbox/audit/logs
//   GET  /api/skillbox/audit/stats

import { http } from '@/core/utils/requests'

export function listAuditLogs(params = {}) {
  return http.get('/api/skillbox/audit/logs', params)
}

export function getAuditStats() {
  return http.get('/api/skillbox/audit/stats')
}
