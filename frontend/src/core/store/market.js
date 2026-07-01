// store/market.js - 三方市场域的 Pinia store。
//
// 2026-06-30 新建。集中管理:
//   - sources        已注册源(skillhub / skills.sh / 自定义)
//   - activeSourceId 当前选中的源 id(0 = 聚合视图)
//   - skills         当前页列表(market_skills 缓存 + installed 标记)
//   - projects       项目列表(供 scope=project 选项用)
//   - pullDialog     控制"拉取"弹窗的开关
//
// 2026-07-01 改造:刷新策略 — 仅缓存为空时自动拉,搜索统一走"已拉数据 substring 过滤"。
//   - 进页面 / 切 tab:缓存为空(shouldAutoRefresh())才拉全量,否则只 loadSkills
//   - Enter / 搜索按钮:都走 setKeyword + loadSkills,keyword 在 DB 走 LIKE(等价于已拉数据 substring 过滤)
//   - "搜索按钮" vs "Enter" 区别:搜索按钮显式 force=true 强制拉全量最新
//   - 用户想强制重新拉:点搜索按钮(force=true);默认行为是缓存优先
//
// 用法:
//   import { useMarketStore } from '@/core/store/market'
//   const store = useMarketStore()
//   await store.loadSources()
//   await store.loadSkills()

import { defineStore } from 'pinia'
import {
  listSources,
  listMarketSkillsWithInstalled,
  refreshSource,
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
    // 2026-07-01 改:刷新 loading 统一用单 refreshing flag。
    // 进入页面自动拉、tab 切换自动拉、点搜索按钮打三方源搜索,均共用此 flag。
    refreshing: false, // 唯一的刷新 loading 标记
    lastRefresh: null, // { source_id, pulled_count, inserted, updated, finished_at, error }

    // 列表
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
    // 2026-07-01 增:当前源在 DB 缓存(market_skills)里是否有数据。
    // 用 total 判断而不是 skills 数组,因为 skills 只是当前页(分页 20 条),
    // 而 total 是后端查的"该源下缓存总数"。
    hasCachedData(state) {
      return (state.total || 0) > 0
    },
    // 2026-07-01 增:判断当前源是否需要自动拉全量。
    //   - 强制:用户切源 / 切 keyword 后想看最新数据
    //   - 自动:仅当缓存为空时才拉,避免每次进入都打远端
    shouldAutoRefresh(state) {
      // 缓存非空 = 跳过自动拉(用户可手动点搜索按钮 force 刷新)
      if ((state.total || 0) > 0) return false
      // 缓存为空 = 首次进入,需要拉
      return true
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

    // --- 列表 ---
    async loadSkills() {
      this.loading = true
      this.lastError = ''
      try {
        const res = await listMarketSkillsWithInstalled({
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

    // 2026-07-01 改:keyword 透传到三方源。行为:
    //   - opts.force=true:忽略缓存状态,强制拉全量
    //   - opts.force=false(默认):仅当缓存为空(shouldAutoRefresh)时才拉,
    //                            否则走 loadSkills 读缓存即可
    //   - opts.keyword 为空:拉全量目录
    //   - opts.keyword 非空:走三方源搜索语义(skillhub 走 /api/skills?keyword=,
    //                       skills.sh 走 audits API + substring 过滤)
    async refreshActive(opts = {}) {
      if (!this.activeSourceId) return false
      const keyword = (opts.keyword ?? this.keyword ?? '').trim()
      // 非强制模式:缓存非空就跳过拉三方源
      if (!opts.force && (this.total || 0) > 0) {
        // 仍然 reload 一下当前页(让 keyword 改动立即生效)
        await this.loadSkills()
        return false
      }
      if (this.refreshing) return false // 已在刷,丢弃新请求(简单防抖)
      this.refreshing = true
      this.lastError = ''
      try {
        const res = await refreshSource(this.activeSourceId, { keyword })
        this.lastRefresh = res
        this.page = 1
        await this.loadSkills()
        return true
      } catch (e) {
        this.lastError = e?.message || String(e)
        throw e
      } finally {
        this.refreshing = false
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
      this.lastRefresh = null
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
