// skillbox/onboarding.js - 首次 Onboarding 域的 HTTP 客户端。
//
// 后端路径:
//   GET  /api/skillbox/onboarding/status
//   POST /api/skillbox/onboarding/scan
//   POST /api/skillbox/onboarding/import        - 消费 scan 缓存导入
//   POST /api/skillbox/onboarding/import-local   - 从本地文件夹/zip 路径导入(JSON,桌面端)
//   POST /api/skillbox/onboarding/import-zip-bytes - 从 zip 字节流导入(octet-stream,Web 端)

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

// runOnboardingImportLocal 桌面端走 JSON 入参:mode=folder | zip_path + 绝对路径。
export function runOnboardingImportLocal(payload) {
  return http.post('/api/skillbox/onboarding/import-local', payload)
}

// runOnboardingImportZipBytes Web 端走 octet-stream:把 File 转 ArrayBuffer 后 POST。
// 这里用 fetch 直传,不通过 http 客户端(避免拦截器把 body 序列化成 JSON)。
export async function runOnboardingImportZipBytes(arrayBuffer) {
  const r = await fetch('/api/skillbox/onboarding/import-zip-bytes', {
    method: 'POST',
    headers: { 'Content-Type': 'application/octet-stream' },
    body: arrayBuffer,
    credentials: 'same-origin',
  })
  if (!r.ok) {
    // 后端错误格式:{error: "..."},尝试解析
    let msg = `HTTP ${r.status}`
    try {
      const data = await r.json()
      if (data?.error) msg = data.error
    } catch (_) { /* ignore */ }
    throw new Error(msg)
  }
  return await r.json()
}
