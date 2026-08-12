/**
 * 回归测试：services/http（第一套业务 Axios）401 处理按 realm 分离（H6）。
 *
 * 背景：旧实现 `unauthorizedHandling` 是单一跨 realm 变量——用户 401 处理在途
 *   时管理员 401 进入会把它顶掉，之后同 realm 的第二个 401 失去防抖，出现
 *   重复登出/弹窗；两个域同时 401 还可能只处理一个或登错域。
 *
 * 目标契约：
 *  - 用户域 401 → 只登出用户 + 重定向一次；同域 1s 内第二个 401 被静默；
 *  - 用户域 401 在途时，管理域 401 不受影响（各自独立登出）；
 *  - 登录请求 401 不触发登出；
 *  - 错误消息兼容后端 `{msg}`（500 用 msg 而不是 axios 英文原文）；
 *  - 调用方取消的请求不弹 toast、不记错误。
 *
 * 运行：`node --test src/services/__tests__/httpRealm401.regression.test.mjs`
 */
import { bundleEntry } from '../../../tests/_bundle.mjs'
import assert from 'node:assert/strict'

// ── 观测桩（数组由 globalThis 桩引用，断言通过 globalThis.__* 读取）───
let navigateCalls = []
let errorMessages = []
let notificationMessages = []

const epStub = {
  name: 'stub-element-plus',
  setup(build) {
    build.onResolve({ filter: /^element-plus$/ }, () => ({ path: 'ep-stub', namespace: 'ep' }))
    build.onLoad({ filter: /.*/, namespace: 'ep' }, () => ({
      contents: `
        export const ElMessage = {
          error: (m) => { globalThis.__epMessages.push(m) },
          success: () => {},
          warning: () => {}
        }
        export const ElMessageBox = { confirm: () => Promise.resolve() }
        export const ElNotification = {
          error: ({ title, message }) => { globalThis.__epNotifications.push(message) }
        }
      `,
      loader: 'js'
    }))
  }
}
const storesStub = {
  name: 'stub-stores',
  setup(build) {
    build.onResolve({ filter: /^@\/stores\/auth$/ }, () => ({
      path: 'auth-stub',
      namespace: 'auth'
    }))
    build.onLoad({ filter: /.*/, namespace: 'auth' }, () => ({
      contents: `
        export const useAuthStore = () => ({
          token: 'user-token',
          logout: () => { globalThis.__userLogoutCalls++ }
        })
      `,
      loader: 'js'
    }))
    build.onResolve({ filter: /^@\/stores\/adminAuth$/ }, () => ({
      path: 'admin-stub',
      namespace: 'admin'
    }))
    build.onLoad({ filter: /.*/, namespace: 'admin' }, () => ({
      contents: `
        export const useAdminAuthStore = () => ({
          token: 'admin-token',
          logout: () => { globalThis.__adminLogoutCalls++ },
          setMfaGateState: () => { globalThis.__setMfaGateCalls++ }
        })
      `,
      loader: 'js'
    }))
  }
}
const navigateStub = {
  name: 'stub-navigate',
  setup(build) {
    build.onResolve({ filter: /^@\/services\/navigate$/ }, () => ({
      path: 'navigate-stub',
      namespace: 'nav'
    }))
    build.onLoad({ filter: /.*/, namespace: 'nav' }, () => ({
      contents: `
        export const navigateReplace = async (p) => { globalThis.__navigateCalls.push(p) }
        export const navigatePush = async () => {}
      `,
      loader: 'js'
    }))
  }
}
const sessionStub = {
  name: 'stub-session',
  setup(build) {
    build.onResolve({ filter: /^@\/router\/session$/ }, () => ({
      path: 'session-stub',
      namespace: 'session'
    }))
    build.onLoad({ filter: /.*/, namespace: 'session' }, () => ({
      contents: `export const clearSession = () => { globalThis.__clearSessionCalls++ }`,
      loader: 'js'
    }))
  }
}
// 观测 refreshSession 调用次数/返回值，用于验证 _retried 标记后不再刷新重试。
const refreshStub = {
  name: 'stub-refresh',
  setup(build) {
    build.onResolve({ filter: /^@\/utils\/auth\/refresh$/ }, () => ({
      path: 'refresh-stub',
      namespace: 'refresh'
    }))
    build.onLoad({ filter: /.*/, namespace: 'refresh' }, () => ({
      contents: `
        export const refreshSession = async () => {
          globalThis.__refreshCalls++
          return globalThis.__refreshResult ?? null
        }
      `,
      loader: 'js'
    }))
  }
}

globalThis.window = { location: { pathname: '/console/orders', search: '', hash: '' } }
const storage = new Map()
globalThis.localStorage = {
  getItem: (k) => storage.get(k) ?? null,
  setItem: (k, v) => storage.set(k, String(v)),
  removeItem: (k) => storage.delete(k)
}
globalThis.__epMessages = errorMessages
globalThis.__epNotifications = notificationMessages
globalThis.__userLogoutCalls = 0
globalThis.__adminLogoutCalls = 0
globalThis.__clearSessionCalls = 0
globalThis.__navigateCalls = navigateCalls
globalThis.__setMfaGateCalls = 0
globalThis.__refreshCalls = 0
globalThis.__refreshResult = null

const mod = await bundleEntry(`export { http } from '@/services/http'`, import.meta.url, {
  plugins: [epStub, storesStub, navigateStub, sessionStub, refreshStub]
})

// 拿到响应拦截器的 rejected handler（axios 内部结构）。
const rejected = mod.http.interceptors.response.handlers[0].rejected

const mkError = ({ status, url, body, code, message, signalAborted = false }) => ({
  response: status ? { status, data: body ?? {}, config: { url } } : undefined,
  config: { url, signal: { aborted: signalAborted } },
  code: code ?? undefined,
  message: message ?? 'Request failed with status code ' + (status ?? 0)
})
const reset = () => {
  // 就地清空，避免与 globalThis 桩引用脱节（重新赋值会让桩指向旧数组）。
  navigateCalls.length = 0
  errorMessages.length = 0
  notificationMessages.length = 0
  globalThis.__userLogoutCalls = 0
  globalThis.__adminLogoutCalls = 0
  globalThis.__clearSessionCalls = 0
  globalThis.__refreshCalls = 0
}
/** 等 debounce（1s）窗口过期，避免上个场景的在途标记串扰。 */
const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms))
const settle = async () => {
  await sleep(1100)
}

// ── 1. 用户域 401：登出用户 + 清 worktab + 重定向，只处理一次 ─────────────
reset()
await rejected(mkError({ status: 401, url: '/api/v1/orders' })).catch(() => {})
assert.equal(globalThis.__userLogoutCalls, 1, '用户 401 登出用户一次')
assert.equal(globalThis.__adminLogoutCalls, 0, '用户 401 不登出管理员')
assert.equal(
  globalThis.__clearSessionCalls,
  1,
  '用户 401 清 worktab/keep-alive（web_auth_surface F2）'
)
assert.equal(globalThis.__navigateCalls.length, 1, '用户 401 重定向到登录页')
assert.match(globalThis.__navigateCalls[0], /^\/login\?redirect=/, '重定向带 redirect')
assert.equal(errorMessages.length, 1, '用户 401 弹一次错')

await settle()
// ── 2. 同域 1s 内第二个 401 被静默 ──────────────────────────────────────
reset()
await rejected(mkError({ status: 401, url: '/api/v1/orders/1/events' })).catch(() => {})
assert.equal(globalThis.__userLogoutCalls, 1, '第一次用户 401 触发登出')
await rejected(mkError({ status: 401, url: '/api/v1/cart' })).catch(() => {})
assert.equal(globalThis.__userLogoutCalls, 1, '同域第二个 401 不重复登出')
assert.equal(errorMessages.length, 1, '同域第二个 401 不重复弹窗')

await settle()
// ── 3. 用户域在途时管理域 401 不受影响（各自独立登出）───────────────────
reset()
await rejected(mkError({ status: 401, url: '/api/v1/orders' })).catch(() => {})
assert.equal(globalThis.__userLogoutCalls, 1, '用户 401 登出用户')
// 把当前地址切到后台域，管理 401 才会重定向到后台登录页。
globalThis.window.location.pathname = '/admin/users'
storage.set('admin_path_cache', 'admin')
await rejected(mkError({ status: 401, url: '/admin/api/v1/users' })).catch(() => {})
assert.equal(globalThis.__adminLogoutCalls, 1, '管理域 401 独立登出管理员（不被用户域防抖顶掉）')
assert.equal(globalThis.__clearSessionCalls, 2, '用户 401 清 worktab + 管理域 401 清动态路由/权限')
assert.equal(globalThis.__navigateCalls.length, 2, '用户 401 + 管理 401 各重定向一次')
assert.match(
  globalThis.__navigateCalls[globalThis.__navigateCalls.length - 1],
  /^\/admin\/login\?redirect=/,
  '管理域 401 重定向到 admin 登录页'
)
// 用户域防抖仍生效
globalThis.window.location.pathname = '/console/orders'
await rejected(mkError({ status: 401, url: '/api/v1/vps' })).catch(() => {})
assert.equal(globalThis.__userLogoutCalls, 1, '用户域防抖未被管理域破坏')

await settle()
// ── 4. 登录请求 401 不触发登出 ──────────────────────────────────────────
reset()
await rejected(mkError({ status: 401, url: '/api/v1/auth/login' })).catch(() => {})
assert.equal(globalThis.__userLogoutCalls, 0, '登录请求 401 不登出')
assert.equal(errorMessages.length, 0, '登录请求 401 不弹窗')

await settle()
// ── 5. 错误消息兼容 {msg}（500 → ElNotification 用 msg 而非 axios 原文）──
reset()
await rejected(
  mkError({ status: 500, url: '/api/v1/orders', body: { msg: '服务端内部错误' } })
).catch(() => {})
assert.equal(notificationMessages.length, 1, '5xx 弹通知')
assert.equal(notificationMessages[0], '服务端内部错误', '5xx 消息取 {msg}')

await settle()
// ── 6. 调用方取消的请求：不弹 toast、不记错误 ───────────────────────────
reset()
await rejected(mkError({ status: 401, url: '/api/v1/orders', code: 'ERR_CANCELED' })).catch(
  () => {}
)
assert.equal(globalThis.__userLogoutCalls, 0, '取消请求不登出')
assert.equal(errorMessages.length, 0, '取消请求不弹窗')
assert.equal(globalThis.__navigateCalls.length, 0, '取消请求不重定向')

await settle()
// ── 7. _retried 已标记的 401：不再刷新重试，直接登出（P1-21 重试上限）───
reset()
globalThis.__refreshResult = 'fresh-token' // 模拟刷新永远成功
globalThis.__refreshCalls = 0
const retriedError = mkError({ status: 401, url: '/api/v1/vps' })
retriedError.config._retried = true
await rejected(retriedError).catch(() => {})
assert.equal(globalThis.__refreshCalls, 0, '_retried 401 不再尝试刷新，杜绝刷新-重放死循环')
assert.equal(globalThis.__userLogoutCalls, 1, '_retried 401 直接登出')
assert.equal(errorMessages.length, 1, '_retried 401 弹一次鉴权失败')
globalThis.__refreshResult = null

console.log('✅ httpRealm401.regression.test.mjs —— services/http realm 分离 401 断言通过')
