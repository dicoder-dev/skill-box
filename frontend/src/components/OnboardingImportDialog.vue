<script setup>
// OnboardingImportDialog - 弹窗形态的"导入技能"。
//
// 把 OnboardingView 的内容塞进 Modal 弹窗,提供给 SkillsView 的"导入"按钮使用。
// 关闭弹窗 / 完成导入都会通知父组件。
//
// 事件:
//   - update:modelValue:标准 v-model 开关
//   - imported:导入完成(无论成功或部分失败),payload 是 importResult

import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Icon } from '@iconify/vue'
import Modal from '@/components/Modal.vue'
import OnboardingView from '@/views/OnboardingView.vue'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
})
const emit = defineEmits(['update:modelValue', 'imported'])

const { t } = useI18n()

const innerRef = ref(null)

function close() {
  emit('update:modelValue', false)
}

function onDone(result) {
  emit('imported', result)
  // 完成后弹窗可以自动关掉,也可以让用户看到统计后自己关
  // 这里选择:不自动关,让用户点"再扫一次"继续,或点关闭
  // 但 reset() 在用户点"再扫一次"时调用
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
      <Icon icon="mdi:tray-arrow-down" width="18" height="18" />
    </template>
    <OnboardingView ref="innerRef" @done="onDone" />
  </Modal>
</template>
