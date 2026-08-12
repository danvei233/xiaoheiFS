/**
 * 回归测试：utils/http token 注入决策（m054 F3 / P2-73）。
 *
 * 背景：模板 axios 的请求拦截器原来读 `useUserStore().accessToken`，该值从不
 * 被填充 → 富文本上传永远不带认证头；且绝对 URL 也会被硬塞 token，可能外泄 JWT。
 *
 * 目标契约（resolveAuthorizationHeader 纯函数）：
 *  - `/api/*` → 用户 token；
 *  - `/admin/api/*` → 管理端 token；
 *  - 绝对 URL / 协议相对 → 不携带；
 *  - 无对应 token → 不携带。
 *
 * 运行：`node --test src/utils/http/__tests__/tokenInjection.regression.test.mjs`
 */
import { bundleEntry } from '../../../../tests/_bundle.mjs'
import assert from 'node:assert/strict'

const localesStub = {
  name: 'stub-locales',
  setup(build) {
    build.onResolve({ filter: /^@\/locales$/ }, () => ({ path: 'locales-stub', namespace: 'stub' }))
    build.onLoad({ filter: /.*/, namespace: 'stub' }, () => ({
      contents: `export const $t = (key) => key`,
      loader: 'js'
    }))
  }
}
const storesStub = {
  name: 'stub-stores',
  setup(build) {
    build.onResolve({ filter: /^@\/stores\/auth$/ }, () => ({
      path: 'auth-stub',
      namespace: 'auth-stub'
    }))
    build.onResolve({ filter: /^@\/stores\/adminAuth$/ }, () => ({
      path: 'admin-stub',
      namespace: 'admin-stub'
    }))
    build.onLoad({ filter: /.*/, namespace: 'auth-stub' }, () => ({
      contents: `export const useAuthStore = () => ({ token: '' })`,
      loader: 'js'
    }))
    build.onLoad({ filter: /.*/, namespace: 'admin-stub' }, () => ({
      contents: `export const useAdminAuthStore = () => ({ token: '' })`,
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
    // logOut 用户域 401 还会动态 import('@/services/navigate')（round-4），
    // 它会拉起整条路由链（.vue 无 loader）——同样 stub 掉。
    build.onResolve({ filter: /^@\/services\/navigate$/ }, () => ({
      path: 'navigate-stub',
      namespace: 'navigate-stub'
    }))
    build.onLoad({ filter: /.*/, namespace: 'navigate-stub' }, () => ({
      contents: `export const navigateReplace = () => {}`,
      loader: 'js'
    }))
  }
}

const { resolveAuthorizationHeader } = await bundleEntry(
  `export { resolveAuthorizationHeader } from '@/utils/http/index'`,
  import.meta.url,
  { plugins: [localesStub, storesStub] }
)

// ── 用户域 ─────────────────────────────────────────────────────────────
assert.equal(
  resolveAuthorizationHeader('/api/v1/orders/1', 'user-jwt', 'admin-jwt'),
  'Bearer user-jwt',
  '/api 前缀用用户 token'
)

// ── 管理域 ─────────────────────────────────────────────────────────────
assert.equal(
  resolveAuthorizationHeader('/admin/api/v1/users', 'user-jwt', 'admin-jwt'),
  'Bearer admin-jwt',
  '/admin/api 前缀用管理端 token'
)

// ── 绝对 URL / 协议相对：不携带（防止 JWT 外泄第三方）───────────────────
assert.equal(
  resolveAuthorizationHeader('https://evil.example.com/upload', 'user-jwt', 'admin-jwt'),
  null,
  '绝对 URL 不携带 token'
)
assert.equal(
  resolveAuthorizationHeader('//evil.example.com/upload', 'user-jwt', 'admin-jwt'),
  null,
  '协议相对 URL 不携带 token'
)
assert.equal(
  resolveAuthorizationHeader('javascript:alert(1)', 'user-jwt', 'admin-jwt'),
  null,
  '非 http 协议不携带 token'
)

// ── 无对应 token：不携带 ───────────────────────────────────────────────
assert.equal(
  resolveAuthorizationHeader('/api/v1/me', '', 'admin-jwt'),
  null,
  '用户 token 为空时不带'
)
assert.equal(
  resolveAuthorizationHeader('/admin/api/v1/me', 'user-jwt', ''),
  null,
  '管理 token 为空时不带'
)

// ── 其他站内路径不带（无前缀）───────────────────────────────────────────
assert.equal(
  resolveAuthorizationHeader('/uploads/a.png', 'user-jwt', 'admin-jwt'),
  null,
  '非 API 路径不携带'
)

console.log('✅ tokenInjection.regression.test.mjs —— token 注入决策断言通过')
