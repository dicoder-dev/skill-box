// skillbox/git.js - 技能仓库版本管理 HTTP 客户端(2026-07-17 增)。
//
// 后端路径(cskillversion 包):
//   GET  /api/skillbox/git/config         读远端配置
//   POST /api/skillbox/git/config         写远端配置 + token 文件
//   GET  /api/skillbox/git/status         仓库状态 + push 队列
//   POST /api/skillbox/git/init           PlainInit
//   GET  /api/skillbox/git/log?limit=N    commit 历史
//   GET  /api/skillbox/git/log/:hash      单 commit 详情
//   GET  /api/skillbox/git/diff?from=A&to=B
//   POST /api/skillbox/git/checkout       reset 工作区
//   POST /api/skillbox/git/push           手动 push
//   POST /api/skillbox/git/discard        丢弃未提交改动
//
// 2026-07-17 设计:
//   - 远端配置 / token 与 configs.Skillbox.Git 同步,设完后即可生效
//   - push 走 store.Save 自动触发;手动 push 是给"重试失败"用
//   - 任何写操作失败仅记录,store.Save 不回滚(业务写盘已成功)

import { http } from '@/core/utils/requests'

/**
 * 读 Git 远端配置。
 * @returns {Promise<{
 *   remote_url: string,
 *   branch: string,
 *   has_token: boolean,
 *   user_name: string,
 *   user_email: string
 * }>}
 */
export function getGitConfig() {
  return http.get('/api/skillbox/git/config')
}

/**
 * 写 Git 远端配置 + token。
 * @param {Object} payload
 * @param {string} [payload.remote_url] - HTTPS URL;留空 = 不修改
 * @param {string} [payload.brranch]    - 留空 = 默认 "main"
 * @param {string} [payload.token]      - GitHub PAT;留空 = 不修改
 * @param {string} [payload.token_file] - 可选;留空 = 默认 ~/.skill-box/.git_token
 * @param {string} [payload.user_name]
 * @param {string} [payload.user_email]
 */
export function saveGitConfig(payload) {
  return http.post('/api/skillbox/git/config', payload)
}

/**
 * 读仓库状态:initialized / HEAD hash / working_clean / pending_push / last_push_error。
 * @returns {Promise<{
 *   initialized: boolean,
 *   branch?: string,
 *   remote_url?: string,
 *   remote_branch?: string,
 *   head_hash?: string,
 *   head_short?: string,
 *   head_message?: string,
 *   working_clean: boolean,
 *   ahead?: number,
 *   behind?: number,
 *   has_token: boolean,
 *   pending_push: number,
 *   last_push_error?: string
 * }>}
 */
export function getGitStatus() {
  return http.get('/api/skillbox/git/status')
}

/**
 * 初始化本地仓库(PlainInit ~/.skill-box/skills);已 init 时 noop。
 */
export function initGit() {
  return http.post('/api/skillbox/git/init')
}

/**
 * 列 commit 历史(最新在前)。
 * @param {number} [limit=50]
 * @returns {Promise<{
 *   items: Array<{
 *     hash: string, short: string, author: string, email: string,
 *     message: string, when: string, files?: string[]
 *   }>,
 *   total: number
 * }>}
 */
export function getGitLog(limit = 50, path = '') {
  return http.get('/api/skillbox/git/log', { limit, path })
}

/**
 * 单 commit 详情。
 * @param {string} hash - 支持短 hash
 */
export function getGitCommit(hash) {
  return http.get(`/api/skillbox/git/log/${encodeURIComponent(hash)}`)
}

/**
 * 两个 commit 之间的 unified diff。
 * @param {string} from
 * @param {string} to
 * @returns {Promise<{diff: string}>}
 */
export function getGitDiff(from, to) {
  return http.get('/api/skillbox/git/diff', { from, to })
}

/**
 * 把工作区 reset 到指定 commit(hard 模式)。
 * 工作区有未提交改动时后端返 409。
 * @param {string} hash
 */
export function checkoutGit(hash) {
  return http.post('/api/skillbox/git/checkout', { hash })
}

/**
 * 手动 push(给"重试失败"用)。
 * 未配置 remote 返 400。
 */
export function pushGit() {
  return http.post('/api/skillbox/git/push')
}

/**
 * 手动 pull(从远端拉取,fast-forward only)。
 * 工作区有未提交改动时后端返 409,前端弹提示让用户先 commit/discard。
 */
export function pullGit() {
  return http.post('/api/skillbox/git/pull')
}

/**
 * 丢弃工作区未提交改动(等价 git reset --hard HEAD)。
 */
export function discardGit() {
  return http.post('/api/skillbox/git/discard')
}