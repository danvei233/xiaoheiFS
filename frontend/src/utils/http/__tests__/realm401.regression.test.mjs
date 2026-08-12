/**
 * 回归测试：401 处理按 realm 分离 + 错误消息兼容后端字段（H6 / P1-21 / P2-73）。
 *
 * 背景：旧实现 401 防抖是跨 realm 共用的单一布尔，`unauthorizedUrl` 只在
 *   HTTP 401 分支设置，业务 code=401（HTTP 200）会读到上次请求残留的 URL，
 *   用户与管理员同时 401 时可能只登出一个或登错域。
 *
 * 目标契约：
 *  - `realmFromUrl` 从**当前 response 的 config.url** 推导域（/admin/api→admin，
 *    其余→user），禁止从旧全局 URL 推断；
 *  - 业务错误消息兼容后端 `{msg}` / `{error}` / `{message}`；
 *  - `handleError` 同样兼容 {error}/{message}。
 *
 * 运行：`node --test src/utils/http/__tests__/realm401.regression.test.mjs`
 */
import { bundleEntry } from '../../../../tests/_bundle.mjs'
import assert from 'node:assert/strict'

const localesStub = {
  name: 'stub-locales',
  setup(build) {
    build.onResolve({ filter: /^@\/locales$/ }, () => ({ path: 'locales-stub', namespace: 'stub' }))
    build.onLoad({ filter: /.*/, namespace: 'stub' }, () => ({
      contents: `export const $t = (key) => key; export default { global: { te: () => true } }`,
      loader: 'js'
    }))
  }
}
const storesStub = {
  name: 'stub-stores',
  setup(build) {
    build.onResolve({ filter: /^@\/stores\/auth$/ }, () => ({
      path: 'auth-store-stub',
      namespace: 'auth-store'
    }))
    build.onResolve({ filter: /^@\/stores\/adminAuth$/ }, () => ({
      path: 'admin-store-stub',
      namespace: 'admin-store'
    }))
    build.onLoad({ filter: /.*/, namespace: 'auth-store' }, () => ({
      contents: `export const useAuthStore = () => ({ token: '', logout: () => {} })`,
      loader: 'js'
    }))
    build.onLoad({ filter: /.*/, namespace: 'admin-store' }, () => ({
      contents: `export const useAdminAuthStore = () => ({ token: '', logout: () => {} })`,
      loader: 'js'
    }))
    // 阻断 logOut 里动态 import('@/router/session') 拉起的整条路由链。
    build.onResolve({ filter: /^@\/router\/session$/ }, () => ({
      path: 'session-stub',
      namespace: 'session-stub'
    }))
    build.onLoad({ filter: /.*/, namespace: 'session-stub' }, () => ({
      contents: `export const clearSession = () => {}`,
      loader: 'js'
    }))
    // web_auth_surface F5：logOut 现在也会动态 import('@/services/navigate')
    // 跳回登录页；阻断它以免拉起整条 router 链（.vue loader 未配置）。
    build.onResolve({ filter: /^@\/services\/navigate$/ }, () => ({
      path: 'navigate-stub',
      namespace: 'navigate-stub'
    }))
    build.onLoad({ filter: /.*/, namespace: 'navigate-stub' }, () => ({
      contents: `export const navigateReplace = async () => {}`,
      loader: 'js'
    }))
  }
}

const { realmFromUrl, resolveBusinessErrorMessage } = await bundleEntry(
  `export { realmFromUrl, resolveBusinessErrorMessage } from '@/utils/http/index'`,
  import.meta.url,
  { plugins: [localesStub, storesStub] }
)

// web_auth_surface F5：模板 axios 栈用户域 401 也必须清 worktab 并跳回登录页，
// 与业务栈 services/http 行为一致（不能只清 token 停在僵死页面）。
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import path from 'node:path'
const __dirname = path.dirname(fileURLToPath(import.meta.url))
const httpSource = readFileSync(path.resolve(__dirname, '../index.ts'), 'utf8')
assert.ok(httpSource.includes("clearSession('console'"), 'utils/http 用户域 401 必须清 worktab')
assert.ok(
  httpSource.includes("import('@/services/navigate')"),
  'utils/http 用户域 401 必须跳转登录页'
)
// web_auth_surface F5：管理域 401（富文本上传走本模板栈）也必须跳回管理登录页，
// 不能停在"会话已清但页面僵死"状态。
assert.ok(httpSource.includes("clearSession('admin'"), 'utils/http 管理域 401 必须清 admin 会话')
assert.ok(
  httpSource.includes('navigateReplace(`/${currentAdminPath()}/login?redirect='),
  'utils/http 管理域 401 必须跳回 /<adminPath>/login'
)
assert.ok(
  httpSource.includes('isOnAdminPath()'),
  'utils/http 管理域 401 跳转必须仅在管理后台路径内触发'
)

// ── realm 推导：从当前 response 的 config.url，不依赖全局状态 ───────────
assert.equal(realmFromUrl('/admin/api/v1/users'), 'admin', '/admin/api 前缀 → admin 域')
assert.equal(realmFromUrl('/api/v1/orders'), 'user', '/api 前缀 → user 域')
assert.equal(realmFromUrl('/api/v1/orders/1/events'), 'user', 'SSE 事件流 → user 域')
assert.equal(realmFromUrl('/uploads/a.png'), 'user', '非 admin 前缀一律 user 域')
assert.equal(realmFromUrl(''), 'user', '空 URL 回退 user 域（fail-safe）')
assert.equal(realmFromUrl(undefined), 'user', 'undefined URL 回退 user 域（fail-safe）')

// ── 业务错误消息字段兼容（utils/http）───────────────────────────────────
assert.equal(resolveBusinessErrorMessage({ msg: '余额不足' }), '余额不足', '{msg} 优先')
assert.equal(resolveBusinessErrorMessage({ error: '余额不足' }), '余额不足', '{error} 兼容')
assert.equal(resolveBusinessErrorMessage({ message: '余额不足' }), '余额不足', '{message} 兼容')
assert.equal(
  resolveBusinessErrorMessage({ msg: 'm', error: 'e', message: 'x' }),
  'm',
  'msg 最高优先'
)
assert.equal(resolveBusinessErrorMessage({}), '', '空对象无消息')
assert.equal(resolveBusinessErrorMessage({ msg: 123 }), '', '非字符串消息忽略')
assert.equal(resolveBusinessErrorMessage(undefined), '', 'undefined 无消息')
assert.equal(resolveBusinessErrorMessage(null), '', 'null 无消息')

console.log('✅ realm401.regression.test.mjs —— realm 分离 + 错误字段兼容断言通过')
