/**
 * 回归测试：SSE 客户端（P1-21 / m051 F6+F7 + token 外发防护）。
 *
 * 背景：
 *  - 旧实现任何非 ok 响应都会无限指数退避重连（含 401/403/404 这类永远
 *    不会恢复的状态），并且 header 由调用方原样透传——若 URL 指向外部主机，
 *    Authorization 会外泄。
 *  - 旧实现只按 '\n\n' 分帧，网关改写成 CRLF（\r\n\r\n）时整条流被当成
 *    一个大块，消息全丢。
 *
 * 运行：`node --test src/services/__tests__/sse.regression.test.mjs`
 */
import { bundleEntry } from '../../../tests/_bundle.mjs'
import assert from 'node:assert/strict'

// sse.ts 在运行时读 window.location.origin，这里在 import 前准备好桩。
globalThis.window = { location: { origin: 'https://cloud.example.com' } }

const { createSseConnection } = await bundleEntry(
  `export { createSseConnection } from '@/services/sse'`,
  import.meta.url
)

// ── 0. isAllowedAuthOrigin 纯函数（web_perf F1 自家 API 源白名单）──────
const { isAllowedAuthOrigin } = await bundleEntry(
  `export { isAllowedAuthOrigin } from '@/services/sse'`,
  import.meta.url
)
{
  const origin = 'https://cloud.example.com'
  // 同源放行
  assert.equal(isAllowedAuthOrigin('/api/v1/orders/1/events', origin, ''), true, '相对路径放行')
  assert.equal(isAllowedAuthOrigin('https://cloud.example.com/x', origin, ''), true, '同源放行')
  // 配置的 API 源放行（绝对后端部署）
  assert.equal(
    isAllowedAuthOrigin(
      'https://api.example.com/api/v1/orders/1/events',
      origin,
      'https://api.example.com'
    ),
    true,
    'VITE_API_BASE 配置的自家 API 源放行'
  )
  assert.equal(
    isAllowedAuthOrigin(
      'https://api.example.com/api/v1/orders/1/events',
      origin,
      'https://api.example.com/base'
    ),
    true,
    '带路径的 VITE_API_BASE 也按 origin 匹配'
  )
  // 无关第三方跨源拒绝
  assert.equal(
    isAllowedAuthOrigin('https://evil.example.com/stream', origin, 'https://api.example.com'),
    false,
    '第三方跨源拒绝'
  )
  assert.equal(
    isAllowedAuthOrigin('https://evil.example.com/stream', origin, ''),
    false,
    '无配置时跨源拒绝'
  )
  // 畸形绝对 URL fail-closed（`new URL` 抛错）
  assert.equal(isAllowedAuthOrigin('https://[::1', origin, ''), false, '畸形 URL 拒绝')
}

const textStream = (text) =>
  new ReadableStream({
    start(controller) {
      controller.enqueue(new TextEncoder().encode(text))
      controller.close()
    }
  })

// ── 1. CRLF 分帧（m051 F7）────────────────────────────────────────────
{
  const frames = []
  let resolveDone
  const done = new Promise((resolve) => (resolveDone = resolve))
  globalThis.fetch = async () => ({
    ok: true,
    status: 200,
    body: textStream('id:1\ndata: alpha\r\n\r\nid:2\ndata: beta\r\n\r\n')
  })
  const conn = createSseConnection('/api/v1/orders/1/events', {
    onMessage: (msg) => {
      frames.push(msg)
      if (frames.length === 2) resolveDone()
    }
  })
  await done
  conn.close()
  assert.equal(frames.length, 2, 'CRLF 分帧应解析出两条消息')
  assert.equal(frames[0].data, 'alpha')
  assert.equal(frames[1].data, 'beta')
}

// ── 2. LF 分帧不回归（既有 \n\n 行为）──────────────────────────────────
{
  const frames = []
  let resolveDone
  const done = new Promise((resolve) => (resolveDone = resolve))
  globalThis.fetch = async () => ({
    ok: true,
    status: 200,
    body: textStream('data: one\n\ndata: two\n\n')
  })
  const conn = createSseConnection('/api/v1/orders/1/events', {
    onMessage: (msg) => {
      frames.push(msg)
      if (frames.length === 2) resolveDone()
    }
  })
  await done
  conn.close()
  assert.equal(frames.length, 2, 'LF 分帧行为应保持不变')
}

// ── 3. 401 终端错误：只请求一次、只上报一次 onError、不再重连 ───────────
{
  let fetchCount = 0
  let errorCount = 0
  globalThis.fetch = async () => {
    fetchCount += 1
    return { ok: false, status: 401, body: null }
  }
  const conn = createSseConnection('/api/v1/orders/1/events', {
    onMessage: () => {},
    onError: () => {
      errorCount += 1
    },
    retryMs: 20
  })
  // 留出足够时间窗口证明没有重连
  await new Promise((resolve) => setTimeout(resolve, 120))
  conn.close()
  assert.equal(fetchCount, 1, '401 后不应再次发起请求')
  assert.equal(errorCount, 1, '401 只上报一次 onError')
}

// ── 4. 5xx 仍按退避重试（既有设计保留）─────────────────────────────────
{
  let fetchCount = 0
  globalThis.fetch = async () => {
    fetchCount += 1
    return { ok: false, status: 503, body: null }
  }
  const conn = createSseConnection('/api/v1/orders/1/events', {
    onMessage: () => {},
    retryMs: 10,
    maxRetryMs: 20
  })
  await new Promise((resolve) => setTimeout(resolve, 120))
  conn.close()
  assert.ok(fetchCount >= 2, `5xx 应持续重试，实际请求 ${fetchCount} 次`)
}

// ── 5. 跨源 URL 剥掉 Authorization（P1-21 token 外泄防护）──────────────
{
  let seenHeaders = null
  globalThis.fetch = async (url, init) => {
    seenHeaders = init.headers
    return { ok: true, status: 200, body: textStream('data: hi\n\n') }
  }
  const conn = createSseConnection('https://evil.example.com/stream', {
    headers: { Authorization: 'Bearer user-jwt' },
    onMessage: () => {},
    retryMs: 10
  })
  await new Promise((resolve) => setTimeout(resolve, 50))
  conn.close()
  assert.ok(seenHeaders, 'fetch 应被调用')
  assert.equal(seenHeaders['Authorization'], undefined, '跨源请求不得携带 Authorization')
  assert.equal(seenHeaders.authorization, undefined)
}

// ── 6. 同源 URL 保留 Authorization ────────────────────────────────────
{
  let seenHeaders = null
  globalThis.fetch = async (url, init) => {
    seenHeaders = init.headers
    return { ok: true, status: 200, body: textStream('data: hi\n\n') }
  }
  const conn = createSseConnection('/api/v1/orders/1/events', {
    headers: { Authorization: 'Bearer user-jwt' },
    onMessage: () => {},
    retryMs: 10
  })
  await new Promise((resolve) => setTimeout(resolve, 50))
  conn.close()
  assert.equal(seenHeaders['Authorization'], 'Bearer user-jwt', '同源请求保留 Authorization')
}

// ── 7. VITE_API_BASE 配置的绝对后端跨源：保留 Authorization（web_perf F1）─
{
  // 用带 VITE_API_BASE 的环境重新打包 createSseConnection。
  const { createSseConnection: apiSse } = await bundleEntry(
    `export { createSseConnection } from '@/services/sse'`,
    import.meta.url,
    { define: { 'import.meta.env': JSON.stringify({ VITE_API_BASE: 'https://api.example.com' }) } }
  )
  let seenHeaders = null
  globalThis.fetch = async (url, init) => {
    seenHeaders = init.headers
    return { ok: true, status: 200, body: textStream('data: hi\n\n') }
  }
  const conn = apiSse('https://api.example.com/api/v1/orders/1/events', {
    headers: { Authorization: 'Bearer user-jwt' },
    onMessage: () => {},
    retryMs: 10
  })
  await new Promise((resolve) => setTimeout(resolve, 50))
  conn.close()
  assert.equal(
    seenHeaders['Authorization'],
    'Bearer user-jwt',
    'VITE_API_BASE 指向的自家 API 源必须保留 Authorization'
  )
}

console.log('✅ sse.regression.test.mjs —— SSE 终端停止/分帧/token 防护/API 源白名单断言通过')
