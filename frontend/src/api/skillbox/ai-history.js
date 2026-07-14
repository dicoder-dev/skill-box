// api/skillbox/ai-history.js - AI 历史对话 HTTP 客户端(2026-07-14 增)
//
// 与 chatStream 区分:
//   - chatStream 走裸 fetch + SSE;
//   - 历史对话走普通 http 拦截器(POST/GET 即可)。
//
// 请求字段全部小写 snake_case 与后端 controller 对齐。
import { http } from '@/core/utils/requests'

// SaveHistory POST /api/skillbox/ai/history/save
// source_path: 磁盘绝对路径(skill 在 store.root 下,带 SKILL.md)
// items: HistoryItem[];空数组 = 清空服务端历史
export function saveHistory(payload) {
  return http.post('/api/skillbox/ai/history/save', payload)
}

// ListHistory GET /api/skillbox/ai/history/list?source_path=<...>
// 不存在时返回 { version: 1, items: [] }。
export function listHistory(params) {
  return http.get('/api/skillbox/ai/history/list', params)
}
