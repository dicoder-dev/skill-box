<script setup>
// InstallProgress - 三方导入 4 阶段进度条(2026-07-18 增)。
//
// 封装自 MarketView.vue 原来的内联进度条(.install-progress 块 + advanceProgress
// 平滑动画 + markFailed fail hint 精确定位),做成可复用组件。
// 4 阶段:resolve(15%) → download(60%) → extract(85%) → write(100%)。
// 平滑动画:每阶段 setInterval 30ms / 600ms,模拟真实节奏,跟 MarketView 同款体感。
//
// 父组件用 expose 出来的方法控制进度:
//   <InstallProgress ref="progRef" />
//   progRef.value.reset()
//   progRef.value.advance('resolve')   // → 'download' → 'extract' → 'write'
//   progRef.value.markFailed()         // 任意阶段出错时,保留 stage + 标 fail
//   progRef.value.markDone()           // 成功后,改 stage='done' + 100%
//
// 父组件要监听进度状态,可以把 progressStage / progressPercent 通过 watch 转成
// 自己的状态,或者直接读组件实例 .stage / .percent (ref 暴露的内部状态)。

import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import IconPark from '@/components/IconPark.vue'

const { t } = useI18n()

// 内部状态
const stage = ref('')           // '' | 'resolve' | 'download' | 'extract' | 'write' | 'done' | 'fail'
const percent = ref(0)          // 0-100
const lastFailedStage = ref('') // fail 时记住"卡哪步"

// 阶段目标百分比 — 跟 MarketView STAGE_TARGETS 同款
const STAGE_TARGETS = {
  resolve: 15,
  download: 60,
  extract: 85,
  write: 100,
}

let timer = null

// 当前阶段图标
const stageIcon = computed(() => {
  if (stage.value === 'done') return 'mdi:check-circle'
  if (stage.value === 'fail') return 'mdi:alert-circle'
  return 'mdi:loading'
})
const stageIconSpin = computed(() => {
  return stage.value !== 'done' && stage.value !== 'fail'
})

// 当前阶段文案 + 提示
const stageLabel = computed(() => t(`market.progress.${stage.value}`))
const stageHint = computed(() => {
  if (stage.value === 'done') return t('market.progress.hintDone')
  if (stage.value === 'fail') {
    const k = lastFailedStage.value
      ? k.charAt(0).toUpperCase() + k.slice(1)
      : 'Unknown'
    return t(`market.progress.hintFail${k}`)
  }
  if (!stage.value) return ''
  return t(`market.progress.hint${stage.value.charAt(0).toUpperCase() + stage.value.slice(1)}`)
})

// 暴露给父组件的方法
function reset() {
  if (timer) {
    clearInterval(timer)
    timer = null
  }
  stage.value = ''
  percent.value = 0
  lastFailedStage.value = ''
}

// 平滑推进进度到目标值。固定 600ms,跟 MarketView 同款体感。
function advance(targetStage) {
  // 2026-07-18 改:跟 MarketView 同款 — advance 接收 stage 字符串。
  // 但本组件同时支持无参 advance()(沿用上一次 stage,主要用于「下载中」状态下
  // 不停 setInterval 拉 0% → 100% 的循环;本次不实现,只支持显式 stage)。
  const t = STAGE_TARGETS[targetStage]
  if (t == null) return
  stage.value = targetStage
  const start = percent.value
  const dur = 600
  const startedAt = Date.now()
  if (timer) clearInterval(timer)
  timer = setInterval(() => {
    const elapsed = Date.now() - startedAt
    if (elapsed >= dur) {
      percent.value = t
      clearInterval(timer)
      timer = null
      return
    }
    const k = elapsed / dur
    percent.value = Math.round(start + (t - start) * k)
  }, 30)
}

function markFailed() {
  if (stage.value && stage.value !== 'done' && stage.value !== 'fail') {
    lastFailedStage.value = stage.value
  }
  stage.value = 'fail'
}

function markDone() {
  stage.value = 'done'
  percent.value = 100
  if (timer) {
    clearInterval(timer)
    timer = null
  }
}

defineExpose({ reset, advance, markFailed, markDone, stage, percent })
</script>

<template>
  <!-- 仅在 stage 非空时显示,跟 MarketView .install-progress 行为一致 -->
  <div v-if="stage" class="ip">
    <div class="ip-row">
      <span class="ip-label">
        <IconPark
          :icon="stageIcon"
          width="14"
          height="14"
          :spin="stageIconSpin"
        />
        {{ stageLabel }}
      </span>
      <span class="ip-percent">{{ percent }}%</span>
    </div>
    <div class="ip-track">
      <div class="ip-fill" :style="{ width: percent + '%' }"></div>
    </div>
    <p
      v-if="stage && stage !== 'done' && stage !== 'fail'"
      class="ip-hint"
    >
      <IconPark icon="mdi:chevron-right" width="12" height="12" />
      {{ stageHint }}
    </p>
    <p v-else-if="stage === 'done'" class="ip-hint ip-hint-done">
      <IconPark icon="mdi:check-circle" width="12" height="12" />
      {{ stageHint }}
    </p>
    <p v-else class="ip-hint ip-hint-fail">
      <IconPark icon="mdi:alert-circle" width="12" height="12" />
      {{ stageHint }}
    </p>
  </div>
</template>

<style scoped>
.ip {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px 14px;
  background: color-mix(in srgb, var(--accent) 5%, var(--bg-card, transparent));
  border: 1px solid color-mix(in srgb, var(--accent) 15%, var(--border, #2a2a2a));
  border-radius: 6px;
}
.ip-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.ip-label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 500;
  color: var(--text, #eee);
}
.ip-percent {
  font-family: 'JetBrains Mono', monospace;
  font-size: 12px;
  color: var(--accent, #3b82f6);
  font-weight: 600;
}
.ip-track {
  width: 100%;
  height: 3px;
  background: color-mix(in srgb, var(--accent, #3b82f6) 10%, var(--bg-card, transparent));
  border-radius: 999px;
  overflow: hidden;
}
.ip-fill {
  height: 100%;
  background: color-mix(in srgb, var(--accent, #3b82f6) 55%, transparent);
  transition: width 0.15s linear;
  border-radius: 999px;
}
.ip-hint {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin: 0;
  font-size: 11px;
  color: var(--text-dim, #999);
  line-height: 1.4;
  font-family: 'JetBrains Mono', monospace;
}
.ip-hint-done {
  color: var(--accent, #10b981);
  font-weight: 500;
}
.ip-hint-fail {
  color: #b91c1c;
  font-weight: 500;
}
:global(html.dark) .ip-hint-fail {
  color: #fca5a5;
}
</style>
