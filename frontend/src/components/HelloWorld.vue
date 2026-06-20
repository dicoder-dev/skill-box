<script setup>
import { ref, onMounted } from 'vue'
import { storeToRefs } from 'pinia'
import { http } from '@/core/utils/requests'
import { platform } from '../platform/index.js'
import { useAppStore } from '@/store/app.js'
import { dlog, derr, enableDebug, isDebug } from '@/core/utils/debug.js'

defineProps({
  msg: String,
})

const store = useAppStore()
const { isDesktop: storeIsDesktop, runMode, needAuth, appName, baseURL, deployLabel } = storeToRefs(store)

const name = ref('')
const healthStatus = ref('checking…')
const serverPort = ref('-')

async function doPing() {
  dlog('doPing clicked')
  try {
    const data = await http.get('/api/health')
    healthStatus.value = `${data.status} (ts=${data.ts})`
    dlog('doPing ok', data)
  } catch (e) {
    healthStatus.value = `error: ${e.message}`
    derr('doPing failed', e)
  }
}

async function doListUsers() {
  dlog('doListUsers clicked')
  try {
    const data = await http.post('/api/sys_user/search', { page: 1, size: 5 })
    dlog('doListUsers ok', data)
    return data
  } catch (e) {
    derr('doListUsers failed', e)
    return { error: e.message }
  }
}

function toggleDebug() {
  enableDebug()
  dlog('debug toggled, enabled=', isDebug())
}

onMounted(async () => {
  dlog('HelloWorld mounted; store=', {
    runMode: runMode.value,
    needAuth: needAuth.value,
    isDesktop: storeIsDesktop.value,
    baseURL: baseURL.value,
  })
  await doPing()
  if (storeIsDesktop.value) {
    try { serverPort.value = String(await platform.app.getServerPort()) } catch (_) {}
  } else {
    serverPort.value = '(web 模式,无本地端口)'
  }
})
</script>

<template>
  <h1>{{ msg }}</h1>

  <div class="result">双部署:Web + 桌面端,业务统一走 HTTP。</div>

  <div class="card">
    <p>部署形态:<b>{{ deployLabel }}</b>(runMode={{ runMode }}, needAuth={{ needAuth }})</p>
    <p>应用名:<b>{{ appName || '(未配置)' }}</b></p>
    <p>本地后端端口:<b>{{ serverPort }}</b></p>
    <p>baseURL:<code>{{ baseURL || '(空=同源)' }}</code></p>
    <p>健康检查:<b>{{ healthStatus }}</b></p>
    <div class="input-box">
      <input aria-label="input" class="input" v-model="name" type="text" autocomplete="off"/>
      <button class="btn" @click="doPing">Ping 后端</button>
      <button class="btn" @click="toggleDebug" v-if="!isDebug()">开 Debug 日志</button>
    </div>
  </div>

  <div class="footer">
    <p>HTTP 接口样板,业务继续按 controller 写,Gin 路由自动注册。</p>
  </div>
</template>
