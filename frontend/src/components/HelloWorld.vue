<script setup>
import { ref, onMounted } from 'vue'
import { storeToRefs } from 'pinia'
import { http } from '@/core/utils/requests'
import { platform } from '../platform/index.js'
import { useAppStore } from '@/core/store/app.js'
import { dlog, derr, enableDebug, isDebug } from '@/core/utils/debug.js'

defineProps({
  msg: String,
})

const store = useAppStore()
const {
  isDesktop: storeIsDesktop,
  runMode,
  needAuth,
  appName,
  baseURL,
  isWeb,
  authEnabled,
  deployLabel,
} = storeToRefs(store)

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

  <div class="state-card">
    <h3>应用 store 状态</h3>
    <div class="state-section-title">state</div>
    <div class="state-grid">
      <span class="k">runMode</span><span class="v">{{ runMode }}</span>
      <span class="k">needAuth</span><span class="v">{{ needAuth }}</span>
      <span class="k">appName</span><span class="v">{{ appName || '(未配置)' }}</span>
      <span class="k">isDesktop</span><span class="v">{{ storeIsDesktop }}</span>
      <span class="k">baseURL</span><span class="v code">{{ baseURL || '(空=同源)' }}</span>
    </div>
    <div class="state-section-title">getters</div>
    <div class="state-grid">
      <span class="k">isWeb</span><span class="v">{{ isWeb }}</span>
      <span class="k">authEnabled</span><span class="v">{{ authEnabled }}</span>
      <span class="k">deployLabel</span><span class="v">{{ deployLabel }}</span>
    </div>
  </div>

  <div class="footer">
    <p>HTTP 接口样板,业务继续按 controller 写,Gin 路由自动注册。</p>
  </div>
</template>

<style scoped>
.state-card {
  margin: 1rem auto;
  padding: 0.9rem 1.1rem;
  max-width: 720px;
  text-align: left;
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.03);
}
.state-card h3 {
  font-size: 1.05em;
  line-height: 1.2;
  margin: 0 0 0.5rem;
  text-align: center;
}
.state-section-title {
  font-size: 0.75em;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  opacity: 0.6;
  margin: 0.4rem 0 0.25rem;
}
.state-grid {
  display: grid;
  grid-template-columns: max-content 1fr;
  column-gap: 1rem;
  row-gap: 0.2rem;
  font-size: 0.9em;
}
.state-grid .k {
  opacity: 0.65;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}
.state-grid .v {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  word-break: break-all;
}
.state-grid .v.code {
  color: #646cff;
}
@media (prefers-color-scheme: light) {
  .state-card {
    border-color: rgba(0, 0, 0, 0.1);
    background: rgba(0, 0, 0, 0.02);
  }
  .state-grid .v.code {
    color: #535bf2;
  }
}
</style>
