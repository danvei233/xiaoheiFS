export interface SseMessage {
  id?: string
  event?: string
  data: string
}

export interface SseOptions {
  headers?: Record<string, string>
  /**
   * P1-21：访问令牌 getter，每次重连时重新读取，使 SSE 与业务请求共享同一个
   * 授权生命周期——access token 被单飞刷新轮换后，SSE 重连自动使用新 token。
   */
  getToken?: () => string
  onMessage: (msg: SseMessage) => void
  onError?: (error: unknown) => void
  retryMs?: number
  maxRetryMs?: number
}

/**
 * SSE 分帧分隔符：覆盖 LF（\n\n）、CRLF（\r\n\r\n）与裸 CR（\r\r）。
 * 后端 `sse/broker.go` 以 \n\n 分帧；网关/代理可能改写为 CRLF，纯 split('\n\n')
 * 会把整条流当成一个大块。m051 F7。
 */
const FRAME_RE = /\r\n\r\n|\r\r|\n\n/

/**
 * 终端错误状态码：这些状态重连也无法恢复，必须停止退避重试（m051 F6）。
 * 5xx 仍按既有指数退避继续重试。
 */
const TERMINAL_STATUSES = new Set([401, 403, 404])

/**
 * 判定某个 URL 是否被允许携带 Authorization（纯函数，便于回归测试）。
 *
 * 允许两类地址：
 *  - 同源 / 相对路径（P1-21 防护的本意：token 不外泄到无关第三方主机）；
 *  - 应用自己的 API 源（`VITE_API_BASE` / `VITE_API_URL` 配置的绝对后端）——
 *    SSE URL 由调用方用同一个变量拼接（`orders/detail.vue`、`probe/detail.vue`），
 *    当 SPA 托管在 CDN、后端在另一域时，剥掉 token 会让整个 SSE 永远 401
 *    停在终端状态（web_perf F1）。自家 API 源视为可信任，保留 Authorization。
 * 其余跨源地址一律拒绝（fail-closed）。
 */
export function isAllowedAuthOrigin(url: string, origin: string, apiBase: string): boolean {
  try {
    const resolved = new URL(url, origin)
    if (resolved.origin === new URL(origin).origin) return true
    if (apiBase) {
      try {
        const apiOrigin = new URL(apiBase, origin).origin
        if (resolved.origin === apiOrigin) return true
      } catch {
        // 配置的 API 地址解析失败时按不允许处理（fail-closed）。
      }
    }
    return false
  } catch {
    return false
  }
}

/** 运行期决定：当前 URL 是否允许带 Authorization。 */
function isSameOriginUrl(url: string): boolean {
  const apiBase = import.meta.env.VITE_API_BASE || import.meta.env.VITE_API_URL || ''
  return isAllowedAuthOrigin(url, window.location.origin, apiBase)
}

const parseEventChunk = (chunk: string) => {
  const lines = chunk.split('\n')
  const payload: SseMessage = { data: '' }
  lines.forEach((line) => {
    // CRLF 分帧下每行尾部可能残留 \r，统一 trim。
    const content = line.replace(/\r$/, '')
    if (content.startsWith('id:')) {
      payload.id = content.replace('id:', '').trim()
    } else if (content.startsWith('event:')) {
      payload.event = content.replace('event:', '').trim()
    } else if (content.startsWith('data:')) {
      const data = content.replace('data:', '').trim()
      payload.data = payload.data ? `${payload.data}\n${data}` : data
    }
  })
  return payload.data ? payload : null
}

export const createSseConnection = (url: string, options: SseOptions) => {
  let closed = false
  let controller: AbortController | null = null
  let lastEventId = ''
  let retryMs = options.retryMs ?? 1500
  const maxRetryMs = options.maxRetryMs ?? 8000

  const connect = async () => {
    if (closed) return
    controller?.abort()
    controller = new AbortController()
    try {
      const headers: Record<string, string> = {
        Accept: 'text/event-stream',
        ...options.headers
      }
      // P1-21：每次重连重新读取 access token，刷新轮换后 SSE 跟随新 token。
      if (options.getToken) {
        const live = options.getToken()
        if (live) headers['Authorization'] = `Bearer ${live}`
      }
      // 跨源 URL 一律剥掉 Authorization（fail-closed）。
      if (!isSameOriginUrl(url)) {
        delete headers['Authorization']
        delete headers.authorization
      }
      if (lastEventId) {
        headers['Last-Event-ID'] = lastEventId
      }
      const res = await fetch(url, { headers, signal: controller.signal })
      if (!res.ok || !res.body) {
        if (TERMINAL_STATUSES.has(res.status)) {
          // 终端错误：只上报一次 onError，然后彻底停止，不再排定重连。
          closed = true
          options.onError?.(new Error(`SSE failed: ${res.status}`))
          return
        }
        throw new Error(`SSE failed: ${res.status}`)
      }
      retryMs = options.retryMs ?? 1500
      const reader = res.body.getReader()
      const decoder = new TextDecoder('utf-8')
      let buffer = ''
      while (!closed) {
        const { value, done } = await reader.read()
        if (done) break
        buffer += decoder.decode(value, { stream: true })
        const parts = buffer.split(FRAME_RE)
        buffer = parts.pop() || ''
        parts.forEach((chunk) => {
          const msg = parseEventChunk(chunk)
          if (msg) {
            if (msg.id) {
              lastEventId = msg.id
            }
            options.onMessage(msg)
          }
        })
      }
    } catch (error) {
      if (!closed) {
        options.onError?.(error)
        retryMs = Math.min(retryMs * 1.4, maxRetryMs)
      }
    } finally {
      if (!closed) {
        setTimeout(connect, retryMs)
      }
    }
  }

  connect()

  return {
    close: () => {
      closed = true
      controller?.abort()
    }
  }
}
