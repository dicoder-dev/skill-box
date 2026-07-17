// skillbox/market.js - 三方市场域的 HTTP 客户端(2026-07-09 增)。
//
// 后端路径:
//   POST /api/skillbox/market/install-from-input
//   响应 200:成功
//   响应 409:同名 skill 已存在(响应体含 conflict_existing_version / conflict_existing_path,
//         前端根据 conflict_mode=overwrite/rename 重试或弹 Modal)

import { http } from '@/core/utils/requests'

/**
 * 用户输入框一键拉取到本地 skill-box store。
 *
 * @param {Object} payload
 * @param {string} payload.source_hint - 当前 tab source_type("skillhub" / "skillssh" / "github")
 * @param {string} payload.input       - 用户原文(只接受 URL)
 * @param {string} [payload.scope]     - "global"(默认) / "project"
 * @param {number} [payload.project_id] - scope=project 时必填
 * @param {string} [payload.conflict_mode] - "" / "prompt"(默认) / "overwrite" / "rename"
 * @param {string} [payload.rename_to] - conflict_mode=rename 时的目标名
 * @param {string} [payload.group_path] - 目标分组路径(2026-07-18 增),空 = 走默认派生
 *
 * @returns {Promise<{
 *   source_type, source_name, remote_id, resolved_url,
 *   skill_name, skill_version, scope, group_path,
 *   conflict_existing_version?, conflict_existing_path?
 * }>}
 *
 * 错误:
 *   400 - 输入格式不识别
 *   404 - 源未注册
 *   409 - 同名 skill 冲突(响应体里 err 含 conflict 字段)
 *
 * 2026-07-10 改:timeout 提到 130s。后端 controller ctx 120s,留 10s 余量;
 * codeload zipball 实测 3.8MB 下载慢网络下 40s+,极端情况更久。
 */
export function installFromInput(payload, options = {}) {
  return http.post('/api/skillbox/market/install-from-input', payload, {
    timeout: 130000,
    ...options,
  })
}

/**
 * 检查同名 skill 是否已存在(轻量,只在第一次「装到 skill-box」点击时用)。
 *
 * 实际上后端 InstallFromInput 默认会做冲突检查返 409,前端可以省去这一步,
 * 统一在 handleInstall catch 409 时弹 Modal。但保留这个方法以便前端在某些场景
 * (如实时显示「本地已有同名」) 提前知道。
 */
export function checkSkillExists(name) {
  return http.get('/api/skillbox/skills/get', { name })
}