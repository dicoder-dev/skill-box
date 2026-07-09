// skillbox/market.js - 三方市场域的 HTTP 客户端(2026-07-09 增)。
//
// 后端路径:
//   POST /api/skillbox/market/install-from-input
//
// 上一代前端(2026-07-01 前的 v1/v2 市场方案)有一整套 /api/skillbox/market/sources
// /list-skills 等端点的客户端,2026-07-09 v3 后 MarketView 改为卡片 + 跳浏览器,
// 那些客户端已被删除。新加 install-from-input 是 v4 输入框 → 后端下载的核心接口。

import { http } from '@/core/utils/requests'

/**
 * 用户输入框一键拉取到本地 skill-box store。
 *
 * @param {Object} payload
 * @param {string} payload.source_hint - 当前 tab source_type("skillhub" / "skillssh");空 = auto
 * @param {string} payload.input       - 用户原文(slug / URL / owner/repo@skill / GitHub URL)
 * @param {string} [payload.scope]     - "global"(默认) / "project"
 * @param {number} [payload.project_id] - scope=project 时必填
 *
 * @returns {Promise<{
 *   source_type: string,
 *   source_name: string,
 *   remote_id: string,
 *   resolved_url: string,
 *   skill_name: string,
 *   skill_version: string,
 *   scope: string,
 * }>}
 */
export function installFromInput(payload) {
  return http.post('/api/skillbox/market/install-from-input', payload)
}