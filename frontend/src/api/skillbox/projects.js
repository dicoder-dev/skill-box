// skillbox/projects.js - Project 域的 HTTP 客户端。
//
// 后端路径:
//   GET    /api/skillbox/projects?page=&size=&keyword=
//   GET    /api/skillbox/projects/get?id=
//   POST   /api/skillbox/projects/create
//   POST   /api/skillbox/projects/update
//   POST   /api/skillbox/projects/delete

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
