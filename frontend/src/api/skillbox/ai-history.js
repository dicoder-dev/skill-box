// api/skillbox/ai-history.js - AI 历史对话 HTTP 客户端(2026-07-14 v2)
//
// 与 chatStream 区分:
//   - chatStream 走裸 fetch + SSE;
//   - 历史对话走普通 http 拦截器(POST/GET 即可)。
//
// v2(2026-07-14):一个对话 = 一份 .skill-box/history/<conv-id>.json。
//   - listHistory 返 metadata-only
//   - getHistory 拉单条 messages
//   - saveHistory 单条 upsert
//   - deleteHistory 按 conv_id 删
//
// 字段全部小写 snake_case 与后端 controller 对齐。
import { http } from '@/core/utils/requests'

// SaveHistory POST /api/skillbox/ai/history/save
// 单条 upsert;body { source_path, item: HistoryItem }
export function saveHistory(payload) {
  return http.post('/api/skillbox/ai/history/save', payload)
}

// ListHistory GET /api/skillbox/ai/history/list?source_path=...
// 返 { items: ConvMeta[] };ConvMeta 是 metadata-only,不返 messages
export function listHistory(params) {
  return http.get('/api/skillbox/ai/history/list', params)
}

// GetHistory GET /api/skillbox/ai/history/get?source_path=...&conv_id=...
// 返 { item: HistoryItem };不存在返 404
export function getHistory(params) {
  return http.get('/api/skillbox/ai/history/get', params)
}

// DeleteHistory POST /api/skillbox/ai/history/delete  (ginp 未实现 DELETE,走 POST)
// 返 { ok: true };不存在幂等也 ok
export function deleteHistory(params) {
  return http.post('/api/skillbox/ai/history/delete', params)
}
