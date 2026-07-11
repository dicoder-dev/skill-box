// skillbox/onboarding.js - 首次 Onboarding 域的 HTTP 客户端。
//
// 后端路径:
//   GET  /api/skillbox/onboarding/status
//   POST /api/skillbox/onboarding/scan
//   POST /api/skillbox/onboarding/import        - 消费 scan 缓存导入
//   POST /api/skillbox/onboarding/import-local   - 从本地文件夹/压缩包路径导入(JSON,桌面端)
//   POST /api/skillbox/onboarding/import-zip-bytes - 从压缩包字节流导入(octet-stream,Web 端)
//   GET  /api/skillbox/onboarding/global-skills  - 列出 ~/.agents/skills 候选(2026-07-10 增)
//   POST /api/skillbox/onboarding/import-global-paths - 按 source_path 批量导入(2026-07-10 增)
//
// 2026-07-11 改:压缩包支持扩展为 zip / tar / tar.gz / tgz / tar.bz2 / tbz2 / tar.xz / txz,
// 端点名字保留历史命名(/import-zip-bytes),payload 用 archive-bytes 而非 zip-bytes 命名,
// 旧前端兼容:runOnboardingImportZipBytes 留作 alias 指向新实现。

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
// 2026-07-11 注:mode='zip_path' 是历史命名,实际支持任意压缩包格式(见 LocalImportPanel ACCEPT_EXTS)。
export function runOnboardingImportLocal(payload) {
  return http.post('/api/skillbox/onboarding/import-local', payload)
}

// runOnboardingImportArchiveBytes Web 端走 octet-stream:把 File 转 ArrayBuffer 后 POST。
// 这里用 fetch 直传,不通过 http 客户端(避免拦截器把 body 序列化成 JSON)。
export async function runOnboardingImportArchiveBytes(arrayBuffer) {
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

// 向后兼容 alias:旧名字指向新实现,避免破坏外部调用方。
export const runOnboardingImportZipBytes = runOnboardingImportArchiveBytes

// 2026-07-10 增:全局目录检索 — 列出 ~/.agents/skills 下所有候选 skill。
// 响应 data 形如:{root: string, exists: bool, items: GlobalSkillCandidate[]}
// - root: 实际扫描根(磁盘绝对路径),前端展示用
// - exists: 目录是否存在;不存在时 items=[]
export function getOnboardingGlobalSkills() {
  return http.get('/api/skillbox/onboarding/global-skills')
}

// 2026-07-10 增:全局目录批量导入 — 把用户在候选列表里勾选的 source_path 批量落地。
// 响应与 /import-local 同构(LocalImportResult),前端可共用结果渲染。
export function runOnboardingImportGlobalPaths(paths = []) {
  return http.post('/api/skillbox/onboarding/import-global-paths', { paths })
}