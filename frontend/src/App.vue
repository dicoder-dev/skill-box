<script setup>
import { ref, onMounted } from 'vue'
import ProjectsView from './views/ProjectsView.vue'
import SkillsView from './views/SkillsView.vue'
import MarketView from './views/MarketView.vue'
import OnboardingView from './views/OnboardingView.vue'
import AuditView from './views/AuditView.vue'
import { listSkills } from '@/api/skillbox/skills'
import { listProjects } from '@/api/skillbox/projects'
import { getOnboardingStatus } from '@/api/skillbox/onboarding'

const tab = ref('skills')

// 顶部统计
const stats = ref({
  skills: 0,
  projects: 0,
  toolsReady: 0,
  toolsTotal: 0,
})
const backendOK = ref(false)
const lastHealth = ref('')

async function refreshStats() {
  try {
    const [skillRes, projRes, obRes] = await Promise.all([
      listSkills({ page: 1, size: 1 }).catch(() => ({ total: 0 })),
      listProjects({ page: 1, size: 1 }).catch(() => ({ total: 0 })),
      getOnboardingStatus().catch(() => ({ adapters: [] })),
    ])
    stats.value.skills = skillRes?.total || 0
    stats.value.projects = projRes?.total || 0
    const adapters = obRes?.adapters || []
    stats.value.toolsTotal = adapters.length
    stats.value.toolsReady = adapters.filter((a) => a.global_ok).length
    backendOK.value = true
    lastHealth.value = new Date().toLocaleTimeString()
  } catch (_) {
    backendOK.value = false
  }
}

onMounted(refreshStats)

// 侧栏配置
const navItems = [
  { key: 'skills',      label: 'Skills',      desc: '浏览 / 编辑 / 测试',  icon: '📚' },
  { key: 'projects',    label: 'Projects',    desc: '项目根 / scope 绑定',  icon: '📁' },
  { key: 'market',      label: 'Market',      desc: '三方 skill 市场',      icon: '🛒' },
  { key: 'onboarding',  label: 'Onboarding',  desc: '首次扫描 / 导入',      icon: '🧭' },
  { key: 'audit',       label: 'Audit',       desc: '操作日志 / 审计',      icon: '📜' },
]

function switchTab(k) {
  tab.value = k
  // 切到某个 tab 时,如果该 tab 有副作用数据加载就触发
  if (k === 'onboarding' || k === 'audit' || k === 'skills') refreshStats()
}
</script>

<template>
  <div class="app">
    <aside class="sidebar">
      <div class="brand">
        <div class="logo">📦</div>
        <div class="brand-text">
          <div class="brand-name">Skill Box</div>
          <div class="brand-sub">AI 工具 skill 统一管理</div>
        </div>
      </div>

      <nav class="nav">
        <button
          v-for="n in navItems"
          :key="n.key"
          :class="['nav-item', { active: tab === n.key }]"
          @click="switchTab(n.key)"
        >
          <span class="nav-icon">{{ n.icon }}</span>
          <span class="nav-text">
            <span class="nav-label">{{ n.label }}</span>
            <span class="nav-desc">{{ n.desc }}</span>
          </span>
        </button>
      </nav>

      <div class="sidebar-foot">
        <div class="health" :class="{ ok: backendOK, err: !backendOK }">
          <span class="dot"></span>
          <span class="health-text">
            {{ backendOK ? '后端已连接' : '后端断开' }}
            <small v-if="lastHealth">· {{ lastHealth }}</small>
          </span>
        </div>
        <button class="refresh-btn" @click="refreshStats" title="刷新统计">↻</button>
      </div>
    </aside>

    <main class="main">
      <header class="topbar">
        <div class="crumbs">
          <span class="crumb-tag">Skill Box</span>
          <span class="crumb-sep">/</span>
          <span class="crumb-cur">{{ navItems.find((x) => x.key === tab)?.label }}</span>
        </div>
        <div class="badges">
          <span class="badge" title="当前 store 里 skill 总数">
            <span class="bd-dot bd-blue"></span>Skills <b>{{ stats.skills }}</b>
          </span>
          <span class="badge" title="已登记的项目数">
            <span class="bd-dot bd-purple"></span>Projects <b>{{ stats.projects }}</b>
          </span>
          <span class="badge" title="已检测到的工具 adapter">
            <span class="bd-dot bd-green"></span>Tools <b>{{ stats.toolsReady }} / {{ stats.toolsTotal }}</b>
          </span>
        </div>
      </header>

      <div class="content">
        <ProjectsView v-if="tab === 'projects'" />
        <SkillsView v-else-if="tab === 'skills'" />
        <MarketView v-else-if="tab === 'market'" />
        <OnboardingView v-else-if="tab === 'onboarding'" />
        <AuditView v-else />
      </div>
    </main>
  </div>
</template>

<style>
:root {
  --bg: #f5f7fa;
  --bg-card: #ffffff;
  --bg-sidebar: #1f2937;
  --bg-sidebar-active: #2563eb;
  --border: #e5e7eb;
  --text: #1f2937;
  --text-dim: #6b7280;
  --text-faint: #9ca3af;
  --primary: #2563eb;
  --primary-dim: #dbeafe;
  --success: #059669;
  --success-dim: #d1fae5;
  --warning: #d97706;
  --warning-dim: #fef3c7;
  --danger: #dc2626;
  --danger-dim: #fee2e2;
  --radius: 6px;
  --radius-sm: 4px;
  --shadow: 0 1px 2px rgba(0, 0, 0, 0.04);
  --shadow-card: 0 1px 3px rgba(0, 0, 0, 0.06);
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
  font-size: 14px;
  line-height: 1.5;
  color: var(--text);
  background: var(--bg);
}
html, body, #app { margin: 0; padding: 0; min-height: 100%; background: var(--bg); }

* { box-sizing: border-box; }

/* 通用表单元素 */
input, select, textarea, button {
  font-family: inherit;
  font-size: 14px;
  color: var(--text);
}
input, select, textarea {
  padding: 6px 10px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: #fff;
  outline: none;
  transition: border-color 0.15s, box-shadow 0.15s;
}
input:focus, select:focus, textarea:focus {
  border-color: var(--primary);
  box-shadow: 0 0 0 2px var(--primary-dim);
}
textarea { resize: vertical; font-family: inherit; }
textarea.code, input.code { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 13px; }
button {
  padding: 6px 12px;
  border: 1px solid var(--border);
  background: #fff;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s;
}
button:hover:not(:disabled) { background: #f9fafb; border-color: #d1d5db; }
button:disabled { opacity: 0.45; cursor: not-allowed; }
button.primary {
  background: var(--primary);
  color: #fff;
  border-color: var(--primary);
}
button.primary:hover:not(:disabled) { background: #1d4ed8; border-color: #1d4ed8; }
button.danger {
  background: var(--danger);
  color: #fff;
  border-color: var(--danger);
}
button.danger:hover:not(:disabled) { background: #b91c1c; border-color: #b91c1c; }
button.ghost { background: transparent; border-color: transparent; }
button.ghost:hover:not(:disabled) { background: #f3f4f6; }
button.link {
  background: transparent;
  border: none;
  padding: 4px 6px;
  color: var(--primary);
  border-radius: var(--radius-sm);
}
button.link:hover:not(:disabled) { background: var(--primary-dim); }
button.link.danger { color: var(--danger); }
button.link.danger:hover:not(:disabled) { background: var(--danger-dim); }
button.sm { padding: 3px 8px; font-size: 12px; }

/* 通用工具 */
.muted { color: var(--text-dim); }
.error { color: var(--danger); }
.success { color: var(--success); }
code { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 0.9em; background: #f3f4f6; padding: 1px 5px; border-radius: 3px; color: #374151; }

/* Layout */
.app { display: flex; min-height: 100vh; }

/* 侧栏 */
.sidebar {
  width: 240px;
  background: var(--bg-sidebar);
  color: #d1d5db;
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
}
.brand { padding: 18px 16px 14px; display: flex; align-items: center; gap: 10px; border-bottom: 1px solid #374151; }
.logo {
  width: 36px; height: 36px;
  background: linear-gradient(135deg, #2563eb, #7c3aed);
  border-radius: 8px; display: flex; align-items: center; justify-content: center;
  font-size: 20px;
}
.brand-name { color: #fff; font-weight: 600; font-size: 15px; }
.brand-sub { color: #9ca3af; font-size: 11px; }

.nav { flex: 1; padding: 10px 8px; display: flex; flex-direction: column; gap: 2px; }
.nav-item {
  display: flex; align-items: center; gap: 10px;
  padding: 9px 10px; border-radius: var(--radius-sm);
  background: transparent; border: none; color: #d1d5db; text-align: left;
  width: 100%; transition: background 0.15s;
}
.nav-item:hover:not(.active) { background: rgba(255,255,255,0.06); }
.nav-item.active { background: var(--bg-sidebar-active); color: #fff; }
.nav-icon { font-size: 18px; flex-shrink: 0; }
.nav-text { display: flex; flex-direction: column; min-width: 0; }
.nav-label { font-size: 13px; font-weight: 500; }
.nav-desc { font-size: 11px; color: #9ca3af; }
.nav-item.active .nav-desc { color: #c7d2fe; }

.sidebar-foot {
  padding: 12px 16px; border-top: 1px solid #374151;
  display: flex; align-items: center; gap: 8px;
}
.health { display: flex; align-items: center; gap: 6px; flex: 1; min-width: 0; }
.health .dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; }
.health.ok .dot { background: #34d399; box-shadow: 0 0 0 2px rgba(52, 211, 153, 0.2); }
.health.err .dot { background: #f87171; box-shadow: 0 0 0 2px rgba(248, 113, 113, 0.2); }
.health-text { font-size: 11px; color: #9ca3af; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.health-text small { font-size: 10px; }
.refresh-btn {
  background: transparent; border: 1px solid #374151; color: #d1d5db;
  width: 26px; height: 26px; padding: 0; border-radius: var(--radius-sm);
  display: flex; align-items: center; justify-content: center;
}
.refresh-btn:hover:not(:disabled) { background: rgba(255,255,255,0.08); border-color: #4b5563; }

/* 顶栏 */
.main { flex: 1; display: flex; flex-direction: column; min-width: 0; }
.topbar {
  background: #fff; border-bottom: 1px solid var(--border);
  padding: 10px 20px; display: flex; align-items: center; justify-content: space-between;
  gap: 16px;
}
.crumbs { display: flex; align-items: center; gap: 6px; font-size: 13px; }
.crumb-tag {
  background: linear-gradient(135deg, #2563eb, #7c3aed);
  color: #fff; padding: 2px 8px; border-radius: 10px; font-size: 11px; font-weight: 500;
}
.crumb-sep { color: var(--text-faint); }
.crumb-cur { color: var(--text); font-weight: 500; }
.badges { display: flex; gap: 8px; }
.badge {
  display: inline-flex; align-items: center; gap: 5px;
  padding: 3px 9px; background: #f3f4f6; border-radius: 12px;
  font-size: 12px; color: var(--text-dim);
}
.badge b { color: var(--text); font-weight: 600; }
.bd-dot { width: 6px; height: 6px; border-radius: 50%; }
.bd-blue { background: var(--primary); }
.bd-purple { background: #7c3aed; }
.bd-green { background: var(--success); }

.content { flex: 1; padding: 20px; overflow: auto; }

/* 通用卡片/面板/空态/loading(各 view 用) */
.card {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  box-shadow: var(--shadow);
  padding: 16px;
  margin-bottom: 14px;
}
.card h3 { margin: 0 0 12px; font-size: 14px; color: var(--text); display: flex; align-items: center; gap: 8px; }
.card .card-sub { font-size: 12px; color: var(--text-dim); font-weight: normal; }
.empty-state {
  padding: 40px 20px; text-align: center; color: var(--text-faint);
  background: #fafbfc; border: 1px dashed var(--border); border-radius: var(--radius);
}
.empty-state .empty-icon { font-size: 32px; display: block; margin-bottom: 8px; }
.spinner {
  display: inline-block; width: 14px; height: 14px;
  border: 2px solid var(--primary-dim); border-top-color: var(--primary);
  border-radius: 50%; animation: spin 0.8s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }
</style>
