<script setup>
import { ref, onMounted } from 'vue'
import { http } from '@/core/utils/requests'
import { platform } from '../platform/index.js'

defineProps({
  msg: String,
})

const name = ref('')
const healthStatus = ref('checking…')
const serverPort = ref('-')

async function doPing() {
  try {
    const data = await http.get('/api/health')
    healthStatus.value = `${data.status} (ts=${data.ts})`
  } catch (e) {
    healthStatus.value = `error: ${e.message}`
  }
}

async function doListUsers() {
  try {
    const data = await http.post('/api/sys_user/search', { page: 1, size: 5 })
    return data
  } catch (e) {
    return { error: e.message }
  }
}

onMounted(async () => {
  await doPing()
  if (platform.isDesktop) {
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
    <p>部署形态:<b>{{ platform.isDesktop ? '桌面端' : 'Web' }}</b></p>
    <p>本地后端端口:<b>{{ serverPort }}</b></p>
    <p>健康检查:<b>{{ healthStatus }}</b></p>
    <div class="input-box">
      <input aria-label="input" class="input" v-model="name" type="text" autocomplete="off"/>
      <button class="btn" @click="doPing">Ping 后端</button>
    </div>
  </div>

  <div class="footer">
    <p>HTTP 接口样板,业务继续按 controller 写,Gin 路由自动注册。</p>
  </div>
</template>
