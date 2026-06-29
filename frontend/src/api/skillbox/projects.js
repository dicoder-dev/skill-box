// skillbox/projects.js - Project 域的 HTTP 客户端。
//
// 后端路径:
//   GET    /api/skillbox/projects?page=&size=&keyword=
//   GET    /api/skillbox/projects/get?id=
//   POST   /api/skillbox/projects/create
//   POST   /api/skillbox/projects/update
//   POST   /api/skillbox/projects/delete
//   GET    /api/skillbox/projects/scan?project_id=N  - 扫描项目被哪些工具 / skill 引用

import { http } from '@/core/utils/requests'

export function listProjects(params = {}) {
  return http.get('/api/skillbox/projects', params)
}

export function getProject(id) {
  return http.get('/api/skillbox/projects/get', { id })
}

export function createProject(payload) {
  return http.post('/api/skillbox/projects/create', payload)
}

export function updateProject(payload) {
  return http.post('/api/skillbox/projects/update', payload)
}

export function deleteProject(id) {
  return http.post('/api/skillbox/projects/delete', { id })
}

// scanProject 扫描指定项目被哪些工具 / skill 引用。
// 纯读接口,每次请求服务端都会重扫磁盘(不读 DB),前端按 project 缓存避免重复调用。
export function scanProject(projectId) {
  return http.get('/api/skillbox/projects/scan', { project_id: projectId })
}
