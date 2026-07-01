// store/market.js - 三方市场域的 Pinia store(2026-07-01 改造:全走 API)。
//
// 策略(2026-07-01):完全弃用本地缓存。
//   - listMarketSkillsRemote:走 adapter.Discover + in-memory 分页,每次都打三方源
//   - skillhub:走 /api/skills?keyword= 搜索语义
//   - skills.sh:走 50 页 /api/audits + substring(API 无 search 参数)
//   - installed 标记仍走本地 store(扫 ~/.skill-box,与 API 数据正交)
//   - 旧缓存端点 listMarketSkillsWithInstalled / refreshSource 后端仍保留
//     (CLI/调试用),前端不再调用
//
// 用法:
//   import { useMarketStore } from '@/core/store/market'
//   const store = useMarketStore()
//   await store.loadSources()
//   await store.loadSkills()    // 每次都打远端

import { defineStore } from 'pinia'
import {
  listSources,
  listMarketSkillsRemote,
  pullMarketSkillV2,
  listMarketSourcesAggregated,
  updateMarketSource,
} from '@/api/skillbox/market'
import { listProjects } from '@/api/skillbox/projects'

export const useMarketStore = defineStore('market', {
  state: () => ({
    // 源
    sources: [], // [{ id, name, type, enabled, config_json }]
    activeSourceId: 0, // 0 = "全部源" 聚合视图

    // 列表 — 当前页(market_skills 缓存 + installed 标记)
    skills: [], // 完整结构 + installed bool
    installed: {}, // name -> bool
    total: 0,
    page: 1,
    size: 20,
    keyword: '',
    showInstalledOnly: false, // "只看未拉取" 开关

    // 项目(供 scope=project 选项)
    projects: [], // [{ id, name, alias, root_path }]

    // 拉取
    pulling: false,
    lastPullResult: null, // PullV2Result
    lastError: '',

    // 状态
    // 2026-07-01 简化:每次都打远端,只剩 loading 单 flag(取代旧 refreshing)。
    loading: false,
  }),
  getters: {
    activeSource(state) {
      if (state.activeSourceId === 0) return null
      return state.sources.find((s) => s.id === state.activeSourceId) || null
    },
    totalPages(state) {
      return Math.max(1, Math.ceil(state.total / state.size))
    },
  },
  actions: {
    // --- 源 ---
    async loadSources() {
      try {
        const res = await listSources()
        this.sources = res.items || []
        if (this.sources.length > 0 && !this.activeSourceId) {
          // 默认进第一个源(更易引导用户点刷新)
          this.activeSourceId = this.sources[0].id
        }
      } catch (e) {
        this.lastError = e?.message || String(e)
        throw e
      }
    },

    async loadSourcesAggregated() {
      try {
        const res = await listMarketSourcesAggregated()
        // items 仍为 MarketSource 列表;附加字段单独 map
        this.sources = res.items || []
        this._skillCount = res.skill_count || {}
        this._lastFetchedAt = res.last_fetched_at || {}
      } catch (e) {
        this.lastError = e?.message || String(e)
        throw e
      }
    },

    // --- 项目 ---
    async loadProjects() {
      try {
        const res = await listProjects({ page: 1, size: 200 })
        this.projects = res.list || res.items || []
      } catch (e) {
        this.projects = []
      }
    },

    // --- 列表(2026-07-01 改:走纯远端) ---
    async loadSkills() {
      this.loading = true
      this.lastError = ''
      try {
        // listMarketSkillsRemote:走 adapter.Discover,完全不读本地缓存,
        // 响应永远是三方源最新数据;keyword 透传到三方源(skillhub 走真实搜索语义,
        // skills.sh 走 substring,因为该 API 无 search 参数)。
        const res = await listMarketSkillsRemote({
          source_id: this.activeSourceId,
          keyword: this.keyword,
          page: this.page,
          size: this.size,
        })
        // 注入 installed bool 到每个 item
        const installedMap = res.installed || {}
        this.installed = installedMap
        this.skills = (res.items || []).map((it) => ({
          ...it,
          installed: !!installedMap[it.name],
        }))
        this.total = res.total || 0
        // 过滤"只看未拉取"
        if (this.showInstalledOnly) {
          this.skills = this.skills.filter((s) => !s.installed)
          this.total = this.skills.length
        }
      } catch (e) {
        this.lastError = e?.message || String(e)
        throw e
      } finally {
        this.loading = false
      }
    },

    // --- 拉取(v2) ---
    // 2026-07-01 改名:install → pull。语义对齐"从三方源拉取 skill 到 skill-box"。
    async pull({ sourceId, remoteId, scope, projectId, tools, finalName, groupPath }) {
      this.pulling = true
      this.lastError = ''
      try {
        const res = await pullMarketSkillV2({
          source_id: sourceId,
          remote_id: remoteId,
          scope: scope || 'global',
          project_id: projectId || 0,
          tools: tools || [],
          final_name: finalName || '',
          group_path: groupPath || '',
        })
        this.lastPullResult = res
        // 拉取后立刻刷新 installed 标记
        if (res?.name) {
          this.installed[res.name] = true
        }
        return res
      } catch (e) {
        this.lastError = e?.message || String(e)
        throw e
      } finally {
        this.pulling = false
      }
    },

    // install 旧名 alias(2026-07-01 deprecated),新代码请用 pull。
    // 行为完全等价,留作向后兼容。
    async install(payload) {
      return this.pull(payload)
    },

    // --- 源管理 ---
    async updateSource(sourceId, payload) {
      const res = await updateMarketSource(sourceId, payload)
      // 同步更新本地 sources 列表
      const idx = this.sources.findIndex((s) => s.id === sourceId)
      if (idx >= 0 && res) {
        this.sources[idx] = { ...this.sources[idx], ...res }
      }
      return res
    },

    // --- 切换 ---
    setSourceActive(id) {
      this.activeSourceId = id
      this.page = 1
    },

    setKeyword(kw) {
      this.keyword = kw
      this.page = 1
    },

    toggleShowInstalledOnly() {
      this.showInstalledOnly = !this.showInstalledOnly
      this.page = 1
    },
  },
})
