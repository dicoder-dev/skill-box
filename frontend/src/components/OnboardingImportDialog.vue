<script setup>
// OnboardingImportDialog - 弹窗形态的"导入技能"。
//
// 2026-07-01 改:顶部加 tab,两种入口并列:
//   - 「扫描工具」:走 OnboardingView(扫已装编程工具目录,勾选导入)
//   - 「从本地导入」:走 LocalImportPanel(选本地文件夹/zip,直接落地)
//
// 事件:
//   - update:modelValue:标准 v-model 开关
//   - imported:任一路径导入完成(payload 是 importResult);父组件收到后
//     可选择刷新列表或继续在弹窗内查看结果。

import { ref, provide } from 'vue'
import { useI18n } from 'vue-i18n'
import IconPark from '@/components/IconPark.vue'
import Modal from '@/components/Modal.vue'
import OnboardingView from '@/views/OnboardingView.vue'
import LocalImportPanel from '@/components/LocalImportPanel.vue'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
})
const emit = defineEmits(['update:modelValue', 'imported'])

const { t } = useI18n()

// 当前激活的 tab:'scan' | 'local'
const tab = ref('scan')

function close() {
  emit('update:modelValue', false)
}

// 2026-07-07 改:不再只在用户点"完成"按钮时才走 imported,
// 而是子面板完成导入即通知父组件,刷新首页。
// 两条路径(扫工具 / 本地导入)都通过这个事件通知父组件。
// 父组件(SkillsView)一般用 importOpen=false 关闭弹窗,然后 reload() 列表。
// 用 ref 缓存"上次已上报 result 的指纹",防止同一结果重复通知页面刷新。
//   - 指纹 = JSON 字符串,简单可靠。
//   - 切换 tab / 再点"再导一次" 重置时也需要清掉,避免旧指纹锁住新结果。
let lastEmittedSig = ''
function tryEmitImported(result) {
  if (!result) return
  const okCount = Number(result.ok || 0)
  const hasOk = (Array.isArray(result.results) && result.results.some((r) => r && r.ok))
    || okCount > 0
  if (!hasOk) return
  const sig = `${tab.value}|${okCount}|${result.failed || 0}|${JSON.stringify(result.results || [])}`
  if (sig === lastEmittedSig) return
  lastEmittedSig = sig
  emit('imported', result)
  // 不自动关,让用户看到统计结果再点关闭或「再导一次」/「完成」按钮。
}
function resetEmittedSig() {
  lastEmittedSig = ''
}
// 2026-07-07 增:把通知能力 provide 给子组件,让 OnboardingView / LocalImportPanel
// 在得到后端响应那一刻就主动通知父组件,而不是依赖用户点"完成"按钮。
// 注意:SkillsView 的侧栏(用作导入入口) + 项目侧弹窗不同入口都共用这个组件,
// provide 只对当前树生效,新建实例互不干扰。
provide('notifyImportDone', tryEmitImported)
provide('resetImportDoneSig', resetEmittedSig)

// 兼容旧 path:子组件 @done 事件流仍然保留(点击"完成"按钮)。
// 注意:此时 tryEmitImported 已经把同一 result 报过一次,lastEmittedSig
// 会自动去重,避免重复 emit('imported') 触发页面多次 reload。
function onDone(result) {
  tryEmitImported(result)
}
</script>

<template>
  <Modal
    :model-value="modelValue"
    size="full"
    :title="t('onboarding.title')"
    @update:model-value="(v) => emit('update:modelValue', v)"
  >
    <template #title-icon>
      <IconPark icon="mdi:tray-arrow-down" width="18" height="18" />
    </template>

    <!-- 顶部 tab 切换 -->
    <div class="oid-tabs" role="tablist">
      <button
        type="button"
        role="tab"
        :aria-selected="tab === 'scan'"
        :class="['oid-tab', { active: tab === 'scan' }]"
        @click="tab = 'scan'"
      >
        <IconPark icon="mdi:magnify-scan" width="14" height="14" />
        {{ t('onboarding.tabs.scan') }}
      </button>
      <button
        type="button"
        role="tab"
        :aria-selected="tab === 'local'"
        :class="['oid-tab', { active: tab === 'local' }]"
        @click="tab = 'local'"
      >
        <IconPark icon="mdi:folder-download-outline" width="14" height="14" />
        {{ t('onboarding.tabs.local') }}
      </button>
    </div>

    <!-- tab 内容(单选渲染) -->
    <OnboardingView v-if="tab === 'scan'" @done="onDone" />
    <LocalImportPanel v-else @done="onDone" />
  </Modal>
</template>

<style scoped>
.oid-tabs {
  display: flex;
  gap: 4px;
  margin-bottom: 18px;
  border-bottom: 1px solid var(--border);
}

.oid-tab {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 9px 16px;
  background: transparent;
  border: none;
  border-bottom: 2px solid transparent;
  margin-bottom: -1px;
  color: var(--text-dim);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  white-space: nowrap;
  transition: color 0.15s ease, border-color 0.15s ease;
}

.oid-tab:hover:not(.active) {
  color: var(--text);
}

.oid-tab.active {
  color: var(--accent-blue);
  border-bottom-color: var(--accent-blue);
}
</style>