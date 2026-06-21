// skillbox/onboarding.js - 首次 Onboarding 域的 HTTP 客户端。
//
// 后端路径:
//   GET  /api/skillbox/onboarding/status
//   POST /api/skillbox/onboarding/scan
//   POST /api/skillbox/onboarding/import

import { http } from '@/core/utils/requests'

export function getOnboardingStatus() {
  return http.get('/api/skillbox/onboarding/status')
}

export function runOnboardingScan() {
  return http.post('/api/skillbox/onboarding/scan', {})
}

export function runOnboardingImport(items = []) {
  return http.post('/api/skillbox/onboarding/import', { items })
}
