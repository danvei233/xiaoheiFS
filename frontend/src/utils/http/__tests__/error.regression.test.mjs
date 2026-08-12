/**
 * 回归测试：HTTP 错误消息优先级（verify-r2 R-9 / m054 L5）。
 *
 * 背景：m054 L5 修复"服务端 msg 被丢弃"时，把 fallback 改成
 *   `data.msg || error.message || getErrorMessage(status)`，导致服务端
 *   无 msg 的 502/504（nginx HTML、后端遗漏 msg）显示 axios 英文原文
 *   "Request failed with status code 502"，取代了修复前的本地化文案。
 *
 * 目标优先级：
 *   1. 服务端 data.msg（业务明细透传）
 *   2. 按状态码的本地化文案（无 msg 的 502 → httpMsg.badGateway）
 *   3. axios 原始 message
 *   4. 通用兜底
 *
 * 运行：`node --test src/utils/http/__tests__/error.regression.test.mjs`
 */
import { bundleEntry } from '../../../../tests/_bundle.mjs'
import assert from 'node:assert/strict'

// 注入可控的 $t，避免加载整条 i18n 链后断言脆。
// resolveErrorMessage 的第三个参数之后是 `t`（默认 $t）。
// 这里把 error.ts 依赖的 @/locales 换成桩模块，阻断到 router/views 的级联。
const localesStub = {
  name: 'stub-locales',
  setup(build) {
    build.onResolve({ filter: /^@\/locales$/ }, () => ({ path: 'locales-stub', namespace: 'stub' }))
    build.onLoad({ filter: /.*/, namespace: 'stub' }, () => ({
      contents: `export const $t = (key) => (typeof key === 'string' && key.startsWith('httpMsg.') ? 'L:' + key : key)`,
      loader: 'js'
    }))
  }
}
const { resolveErrorMessage, HttpError, handleError } = await bundleEntry(
  `export { resolveErrorMessage, HttpError, handleError } from '@/utils/http/error'`,
  import.meta.url,
  { plugins: [localesStub] }
)

const t = (key) => `L:${key}`

// ── 1. 服务端 msg 优先 ────────────────────────────────────────────────
assert.equal(
  resolveErrorMessage('余额不足', 500, 'Request failed with status code 500', t),
  '余额不足',
  '服务端 msg 应优先透传'
)

// ── 2. 无 msg 时按状态码本地化（R-9 核心）──────────────────────────────
assert.equal(
  resolveErrorMessage(undefined, 502, 'Request failed with status code 502', t),
  'L:httpMsg.badGateway',
  '无 msg 的 502 应回退到本地化 badGateway，而非 axios 英文原文'
)
assert.equal(
  resolveErrorMessage(undefined, 504, 'timeout of 20000ms exceeded', t),
  'L:httpMsg.gatewayTimeout',
  '无 msg 的 504 应回退到本地化 gatewayTimeout'
)
assert.equal(
  resolveErrorMessage(undefined, 401, 'Request failed with status code 401', t),
  'L:httpMsg.unauthorized',
  '无 msg 的 401 应回退到本地化 unauthorized'
)

// ── 3. 网络错误（无状态码、无 msg）→ axios message ────────────────────
assert.equal(
  resolveErrorMessage(undefined, undefined, 'Network Error', t),
  'Network Error',
  '无状态码时保留 axios 网络错误信息'
)

// ── 4. 全部缺失 → 通用兜底 ─────────────────────────────────────────────
assert.equal(
  resolveErrorMessage(undefined, undefined, undefined, t),
  'L:httpMsg.requestFailed',
  '全部缺失时回退通用文案'
)

// ── handleError：取消的请求标记 canceled 且不再把 msg 当主文案 ──────────
let thrown = null
try {
  handleError({ code: 'ERR_CANCELED', message: 'canceled' })
} catch (error) {
  thrown = error
}
assert.ok(thrown instanceof HttpError, '取消请求应抛 HttpError')
assert.equal(thrown.canceled, true, '取消请求应标记 canceled')
assert.equal(thrown.code, 400, '取消请求状态码为 ApiStatus.error')

// ── handleError：无 msg 的 502 走本地化文案 ────────────────────────────
thrown = null
try {
  handleError({
    response: { status: 502, data: '<html>bad gateway</html>' },
    message: 'Request failed with status code 502'
  })
} catch (error) {
  thrown = error
}
assert.ok(thrown instanceof HttpError)
// $t 默认走真实 i18n；这里只验证 message 不是 axios 英文原文。
assert.notEqual(
  thrown.message,
  'Request failed with status code 502',
  '无 msg 的 502 不应显示 axios 英文原文'
)

// ── handleError：兼容后端 {error} / {message} 字段（H6）─────────────────
thrown = null
try {
  handleError({ response: { status: 400, data: { error: '余额不足' } }, message: 'Request failed' })
} catch (error) {
  thrown = error
}
assert.equal(thrown.message, '余额不足', '{error} 字段透传')

thrown = null
try {
  handleError({
    response: { status: 400, data: { message: '服务不可用' } },
    message: 'Request failed'
  })
} catch (error) {
  thrown = error
}
assert.equal(thrown.message, '服务不可用', '{message} 字段透传')

thrown = null
try {
  handleError({
    response: { status: 400, data: { msg: '优先', error: '次之' } },
    message: 'Request failed'
  })
} catch (error) {
  thrown = error
}
assert.equal(thrown.message, '优先', '{msg} 字段优先级最高')

console.log('✅ error.regression.test.mjs —— R-9 消息优先级断言通过')
