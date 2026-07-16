<script setup>
// UpdatePanel - "软件更新" 卡片的内容块。
//
// 状态机:
//   idle / checking / upToDate / available / downloading / pendingRestart / failed / incomparable
//
// 桌面端显示:当前版本 + 渠道 + 检查按钮 + 状态文字 + 进度条 + 立即升级按钮
// Web 端显示:同上,但"立即升级"按钮变成"去下载页"(调 platform.platform.openExternal)

import { computed } from 'vue'
import { useUpdateStore } from '@/core/store/update.js'

const update = useUpdateStore()

const versionText = computed(() => {
  const v = update.localVersion || '未知'
  return `v${v}`
})

const channelText = computed(() => update.channel || 'stable')

function onCheck() {
  update.check()
}

function onInstall() {
  update.download()
}

function onReset() {
  update.reset()
}
</script>

<template>
  <div class="update-panel">
    <!-- 版本信息 + 当前渠道 -->
    <div class="update-meta">
      <div class="update-version">
        <span class="meta-label">当前版本</span>
        <span class="meta-value">{{ versionText }}</span>
      </div>
      <div class="update-channel">
        <span class="meta-label">渠道</span>
        <span class="meta-value">{{ channelText }}</span>
      </div>
    </div>

    <!-- 状态文字 -->
    <div :class="['update-status', `update-status-${update.state}`]">
      <span class="status-text">{{ update.message }}</span>
      <span v-if="update.error" class="status-error">{{ update.error }}</span>
    </div>

    <!-- 进度条 -->
    <div v-if="update.state === 'downloading'" class="update-progress">
      <div class="update-progress-bar" :style="{ width: `${update.progress}%` }"></div>
    </div>

    <!-- 操作按钮区 -->
    <div class="update-actions">
      <button
        v-if="update.state === 'idle' || update.state === 'upToDate' || update.state === 'incomparable' || update.state === 'failed'"
        class="primary"
        :disabled="update.state === 'checking'"
        @click="onCheck"
      >
        {{ update.state === 'checking' ? '检查中...' : '检查更新' }}
      </button>

      <!-- 桌面端:available / mustUpdate 显示"立即升级" -->
      <button
        v-if="update.isDesktop && (update.state === 'available' || update.state === 'mustUpdate')"
        class="primary"
        @click="onInstall"
      >
        立即升级并重启
      </button>

      <!-- Web 端:available 显示"去下载页"外链 -->
      <button
        v-if="!update.isDesktop && update.state === 'available'"
        class="primary"
        @click="onInstall"
      >
        去下载页
      </button>

      <!-- pendingRestart:父进程即将退出,等重启就好 -->
      <button
        v-if="update.state === 'pendingRestart'"
        class="muted"
        disabled
      >
        应用将在重启后完成更新
      </button>

      <!-- 失败时给"重试 / 重置" -->
      <button
        v-if="update.state === 'failed' || update.state === 'incomparable'"
        class="muted"
        @click="onReset"
      >
        重置
      </button>
    </div>

    <!-- 升级说明 changelog -->
    <div v-if="update.notes && (update.state === 'available' || update.state === 'mustUpdate' || update.state === 'pendingRestart')" class="update-notes">
      <div class="notes-title">更新说明</div>
      <pre class="notes-body">{{ update.notes }}</pre>
    </div>
  </div>
</template>

<style scoped>
.update-panel {
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 4px 0;
}

.update-meta {
  display: flex;
  gap: 24px;
  font-size: 13px;
  color: var(--text-dim);
}

.meta-label {
  font-size: 12px;
  margin-right: 6px;
  color: var(--text-faint);
}

.meta-value {
  font-family: 'JetBrains Mono', monospace;
  color: var(--text);
}

.update-status {
  padding: 12px 14px;
  border-radius: var(--radius-sm);
  font-size: 13px;
  background: var(--bg-subtle);
  border: 1px solid var(--border);
}

.update-status-checking,
.update-status-downloading {
  border-color: var(--primary);
  color: var(--primary);
}

.update-status-available,
.update-status-pendingRestart,
.update-status-mustUpdate {
  background: var(--primary-dim, #e0f2fe);
  border-color: var(--primary);
  color: var(--primary);
}

.update-status-failed {
  border-color: var(--danger);
  background: var(--danger-dim, #fee2e2);
  color: var(--danger);
}

.update-status-upToDate {
  border-color: var(--success);
  color: var(--success);
}

.status-text {
  font-weight: 500;
}

.status-error {
  display: block;
  margin-top: 4px;
  font-size: 12px;
  color: var(--danger);
}

.update-progress {
  height: 3px;
  background: var(--bg-subtle);
  border-radius: 999px;
  overflow: hidden;
}

.update-progress-bar {
  height: 100%;
  /* 颜色淡化:主色降到 55% 透明度,视觉更柔和 */
  background: color-mix(in srgb, var(--primary) 55%, transparent);
  border-radius: 999px;
  transition: width 0.3s ease;
}

.update-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.update-notes {
  background: var(--bg-subtle);
  border-radius: var(--radius-sm);
  padding: 12px 14px;
  border: 1px solid var(--border);
}

.notes-title {
  font-size: 12px;
  color: var(--text-dim);
  margin-bottom: 6px;
}

.notes-body {
  margin: 0;
  font-family: 'JetBrains Mono', monospace;
  font-size: 12px;
  white-space: pre-wrap;
  word-break: break-word;
  color: var(--text);
}

/* 按钮风格复刻 SettingsView.applyMode 那一段,不引紫色 */
button.primary {
  display: inline-flex;
  align-items: center;
  height: 32px;
  padding: 0 14px;
  font-size: 13px;
  font-weight: 500;
  color: var(--primary-contrast, #fff);
  background: var(--primary);
  border: 1px solid var(--primary);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all 0.15s ease;
}

button.primary:hover:not(:disabled) {
  filter: brightness(1.05);
}

button.primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

button.muted {
  display: inline-flex;
  align-items: center;
  height: 32px;
  padding: 0 14px;
  font-size: 13px;
  background: transparent;
  color: var(--text-dim);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  cursor: pointer;
}

button.muted:disabled {
  cursor: not-allowed;
  color: var(--primary);
}
</style>
