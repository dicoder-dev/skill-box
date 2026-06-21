// skillbox/ai.js - AI 域 HTTP 客户端(SSE 除外,SSE 走裸 fetch 解析 stream)。
//
// 后端路径:
//   GET    /api/skillbox/ai/providers
//   GET    /api/skillbox/ai/providers/get?id=
//   POST   /api/skillbox/ai/providers/create
//   POST   /api/skillbox/ai/providers/update
//   POST   /api/skillbox/ai/providers/delete
//   POST   /api/skillbox/ai/providers/key
//   GET    /api/skillbox/ai/presets
//   POST   /api/skillbox/ai/chat              (SSE;不经过 http 客户端)

import { http } from '@/core/utils/requests'

export function listProviders() {
  return http.get('/api/skillbox/ai/providers')
}

export function getProvider(id) {
  return http.get('/api/skillbox/ai/providers/get', { id })
}

export function createProvider(payload) {
  return http.post('/api/skillbox/ai/providers/create', payload)
}

export function updateProvider(payload) {
  return http.post('/api/skillbox/ai/providers/update', payload)
}

export function deleteProvider(id) {
  return http.post('/api/skillbox/ai/providers/delete', { id })
}

export function setProviderKey(name, key) {
  return http.post('/api/skillbox/ai/providers/key', { name, key })
}

export function listPresets() {
  return http.get('/api/skillbox/ai/presets')
}

/**
 * 解析 SSE 流的工具:逐行读 chunks,以 \n\n 分帧。
 * 每个回调拿到一个 data 字符串(去掉 "data: " 前缀);DONE 标记触达时返回。
 *
 * @param {Response} resp        fetch 的 Response
 * @param {object}   handlers    { onEvent(data), onDone(), onError(err) }
 * @returns {Promise<void>}
 */
export async function parseSSE(resp, { onEvent, onDone, onError }) {
  if (!resp.body || !resp.body.getReader) {
    onError && onError(new Error('stream not supported'))
    return
  }
  const reader = resp.body.getReader()
  const decoder = new TextDecoder('utf-8')
  let buf = ''
  try {
    while (true) {
      const { value, done } = await reader.read()
      if (done) break
      buf += decoder.decode(value, { stream: true })
      let idx
      // 拆 \n\n 帧(允许 \r\n\r\n)
      while ((idx = buf.indexOf('\n\n')) >= 0) {
        const frame = buf.slice(0, idx)
        buf = buf.slice(idx + 2)
        processFrame(frame, onEvent, onDone, onError)
        if (resp.body._done) return
      }
    }
    // 收尾:流结束了但 buffer 里有残余
    if (buf.trim().length) {
      processFrame(buf, onEvent, onDone, onError)
    }
  } catch (e) {
    onError && onError(e)
  } finally {
    reader.releaseLock()
  }
}

function processFrame(frame, onEvent, onDone, onError) {
  const lines = frame.split('\n')
  const dataLines = []
  for (const line of lines) {
    if (!line) continue
    if (line.startsWith(':')) continue // 心跳 / 注释
    if (line.startsWith('data:')) {
      dataLines.push(line.slice(5).trimStart())
    }
  }
  if (!dataLines.length) return
  const data = dataLines.join('\n')
  if (data === '[DONE]') {
    onDone && onDone()
    return
  }
  try {
    const obj = JSON.parse(data)
    onEvent && onEvent(obj)
  } catch (e) {
    onError && onError(new Error(`bad sse frame: ${data}`))
  }
}

/**
 * 发一次 AI 对话,返回 { abort }。
 * 调用方:
 *   const { abort } = chatStream({ provider, model, messages }, {
 *     onEvent: (ev) => { ... }, onDone: () => {...}, onError: (e) => {...}
 *   })
 *
 * 协议见 backend chat_stream.a.go。
 * @returns {Promise<{ abort: () => void }>}
 */
export async function chatStream(body, { onEvent, onDone, onError, onOpen } = {}) {
  const ctrl = new AbortController()
  let resp
  try {
    resp = await fetch('/api/skillbox/ai/chat', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
      signal: ctrl.signal,
    })
  } catch (e) {
    onError && onError(e)
    return { abort: () => ctrl.abort() }
  }
  if (!resp.ok) {
    let msg = `http ${resp.status}`
    try {
      const j = await resp.json()
      msg = j.error || msg
    } catch (_) {}
    onError && onError(new Error(msg))
    return { abort: () => ctrl.abort() }
  }
  if (onOpen) onOpen()
  await parseSSE(resp, {
    onEvent,
    onDone: () => { resp.body && (resp.body._done = true); onDone && onDone() },
    onError,
  })
  return { abort: () => ctrl.abort() }
}
