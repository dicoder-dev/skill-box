// skillbox/tags.js - Tag / Diff / Rollback 的 HTTP 客户端。
//
// 后端路径:
//   POST /api/skillbox/skills/tags/create     - 打 tag(固化当前文件)
//   GET  /api/skillbox/skills/tags/list       - 列 skill 的 tag
//   POST /api/skillbox/skills/tags/delete     - 删 tag(包括 file_snapshots)
//   GET  /api/skillbox/skills/tags/diff       - 两视图 diff(0 = current)
//   POST /api/skillbox/skills/tags/rollback   - 回滚到 tag(自动打 _pre_rollback)

import { http } from '@/core/utils/requests'

export function createTag(payload) {
  return http.post('/api/skillbox/skills/tags/create', payload)
}

export function listTags(params = {}) {
  return http.get('/api/skillbox/skills/tags/list', params)
}

export function deleteTag(payload) {
  return http.post('/api/skillbox/skills/tags/delete', payload)
}

export function diffTag(params = {}) {
  return http.get('/api/skillbox/skills/tags/diff', params)
}

export function rollbackTag(payload) {
  return http.post('/api/skillbox/skills/tags/rollback', payload)
}
