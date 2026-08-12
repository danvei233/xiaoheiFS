/**
 * 回归测试：模拟登录 token 刷新后不得被误认成真实登录（H4 / P2-70 / m053 H2）。
 *
 * 背景：旧实现 `loginToken: localStorage.getItem(STORAGE_KEY)` 从 token 推断，
 *   模拟登录（impersonation）写入的 token 刷新后被当成真实登录，
 *   `cart.mergeGuestCart` 会把设备级游客购物车并进被模拟用户的服务端账号。
 *
 * 修复：显式持久化会话种类 `user_token_kind`（'real' | 'impersonated'）；
 *   `readStoredAuthSession` 只有 'real' 才恢复 loginToken，其余一律为空
 *   （fail-closed）。模拟登录建立/刷新/退出都不能合并。
 *
 * 运行：`node --test src/stores/__tests__/authSessionKind.regression.test.mjs`
 */
import { bundleEntry } from '../../../tests/_bundle.mjs'
import assert from 'node:assert/strict'

// P1-21 刷新模块含动态 import('axios')；这些测试只验证 auth store 的会话
// 种类/合并语义，不需要真实刷新链路，stub 掉避免 esbuild 拉 axios。
const refreshStub = {
  name: 'stub-refresh',
  setup(build) {
    build.onResolve({ filter: /^@\/utils\/auth\/refresh$/ }, () => ({
      path: 'refresh-stub',
      namespace: 'refresh'
    }))
    build.onLoad({ filter: /.*/, namespace: 'refresh' }, () => ({
      contents: `
        export const currentSessionEpoch = () => 0
        export const bumpSessionEpoch = () => {}
        export const storeRefreshToken = (realm, token) => {
          if (!globalThis.__refreshSlot) globalThis.__refreshSlot = {}
          globalThis.__refreshSlot[realm] = token || ''
        }
        export const clearRefreshToken = (realm) => {
          if (globalThis.__refreshSlot) delete globalThis.__refreshSlot[realm]
        }
        export const refreshSession = async () => null
      `,
      loader: 'js'
    }))
  }
}

// 阻断 services/user → services/http → element-plus 的级联。
const userStub = {
  name: 'stub-user-service',
  setup(build) {
    build.onResolve({ filter: /^@\/services\/user$/ }, () => ({
      path: 'user-service-stub',
      namespace: 'user-service'
    }))
    build.onLoad({ filter: /.*/, namespace: 'user-service' }, () => ({
      contents: `
        const noop = async () => ({ data: {} })
        export const userLogin = noop
        export const userLogout = noop
        export const getMe = async () => ({ data: null })
        export const updateMe = noop
        export const addCartItem = noop
        export const clearCart = noop
        export const deleteCartItem = noop
        export const listCart = noop
        export const updateCartItem = noop
        export const listVps = noop
        export const getVpsDetail = noop
        export const getVpsPanelUrl = noop
        export const refreshVps = noop
        export const getDashboard = noop
        export const listOrders = noop
        export const getWallet = noop
        export const getRealNameStatus = noop
      `,
      loader: 'js'
    }))
  }
}

// adminAuth store 静态 import @/services/admin（adminLogin/getAdminProfile），
// stub 掉避免级联 http → element-plus。
const adminServiceStub = {
  name: 'stub-admin-service',
  setup(build) {
    build.onResolve({ filter: /^@\/services\/admin$/ }, () => ({
      path: 'admin-service-stub',
      namespace: 'admin-service'
    }))
    build.onLoad({ filter: /.*/, namespace: 'admin-service' }, () => ({
      contents: `
        const noop = async () => ({ data: {} })
        export const adminLogin = noop
        export const adminLogout = noop
        export const getAdminProfile = noop
      `,
      loader: 'js'
    }))
  }
}

// auth.applyImpersonatedToken 现在会动态 import @/stores/cart（带 element-plus
// 的 ElMessage 链），测试里把它 stub 掉，只验证 resetForAccountSwitch 被调用。
const cartStub = {
  name: 'stub-cart',
  setup(build) {
    build.onResolve({ filter: /^@\/stores\/cart$/ }, () => ({
      path: 'cart-stub',
      namespace: 'cart'
    }))
    build.onLoad({ filter: /.*/, namespace: 'cart' }, () => ({
      contents: `
        let resetCalls = 0
        export const __resetCalls = () => resetCalls
        export const useCartStore = () => ({
          resetForAccountSwitch: () => { resetCalls += 1 }
        })
      `,
      loader: 'js'
    }))
  }
}

const store = new Map()
globalThis.localStorage = {
  getItem: (k) => store.get(k) ?? null,
  setItem: (k, v) => store.set(k, String(v)),
  removeItem: (k) => store.delete(k)
}
globalThis.__refreshSlot = {}

const mod = await bundleEntry(
  `export { useAuthStore, readStoredAuthSession } from '@/stores/auth'
   export { useAdminAuthStore } from '@/stores/adminAuth'
   export { createPinia, setActivePinia } from 'pinia'`,
  import.meta.url,
  { plugins: [refreshStub, userStub, cartStub, adminServiceStub] }
)

const seed = (token, kind) => {
  store.clear()
  if (token !== undefined) store.set('user_token', token)
  if (kind !== undefined) store.set('user_token_kind', kind)
}

// ── 纯函数：readStoredAuthSession 的刷新还原规则 ────────────────────────
seed('jwt-token', 'real')
assert.deepEqual(
  mod.readStoredAuthSession(),
  {
    token: 'jwt-token',
    loginToken: 'jwt-token',
    kind: 'real'
  },
  '真实登录刷新后恢复 loginToken'
)

seed('jwt-token', 'impersonated')
assert.deepEqual(
  mod.readStoredAuthSession(),
  {
    token: 'jwt-token',
    loginToken: '',
    kind: 'impersonated'
  },
  '模拟登录刷新后 loginToken 必须为空'
)

seed('jwt-token', undefined)
assert.equal(mod.readStoredAuthSession().loginToken, '', '旧版本无 kind 键 → fail-closed 不恢复')

seed('jwt-token', 'weird')
assert.equal(mod.readStoredAuthSession().loginToken, '', '未知 kind → fail-closed 不恢复')

seed(undefined, 'real')
assert.deepEqual(
  mod.readStoredAuthSession(),
  { token: '', loginToken: '', kind: '' },
  '无 token 即空会话'
)

// ── store 刷新还原（reload 场景）────────────────────────────────────────
// 模拟登录的 token 在 localStorage，kind=impersonated → 实例化后 loginToken 为空。
seed('impersonated-token', 'impersonated')
mod.setActivePinia(mod.createPinia())
const impersonatedStore = mod.useAuthStore()
assert.equal(impersonatedStore.token, 'impersonated-token', 'token 仍被还原')
assert.equal(impersonatedStore.loginToken, '', '模拟登录刷新后 loginToken 为空')
assert.equal(impersonatedStore.sessionKind, 'impersonated')
// cart 合并门禁：loginToken !== token → 不合并
assert.notEqual(impersonatedStore.loginToken, impersonatedStore.token)

// 真实登录刷新 → loginToken === token → 合并门禁放行。
seed('real-token', 'real')
mod.setActivePinia(mod.createPinia())
const realStore = mod.useAuthStore()
assert.equal(realStore.loginToken, 'real-token', '真实登录刷新后恢复 loginToken')
assert.equal(realStore.loginToken, realStore.token, '真实登录合并门禁放行')

// ── applyImpersonatedToken：建立/清除时都不合并 ─────────────────────────
mod.setActivePinia(mod.createPinia())
const fresh = mod.useAuthStore()
fresh.applyImpersonatedToken('mock-user-token')
assert.equal(fresh.token, 'mock-user-token')
assert.equal(fresh.loginToken, '', '模拟登录建立时 loginToken 保持空')
assert.equal(store.get('user_token_kind'), 'impersonated', 'kind 持久化为 impersonated')

// ── applyImpersonatedToken：建立/清除时都不合并 ─────────────────────────

fresh.logout()
assert.equal(fresh.loginToken, '', '登出后 loginToken 清空')
assert.equal(store.has('user_token_kind'), false, '登出清除 kind 键')

// ── P1-21（绕过）：applyImpersonatedToken 必须清掉真实登录残留的 refresh_token ──
globalThis.__refreshSlot = { user: 'real-user-rt' }
fresh.applyImpersonatedToken('mock-user-token-2')
assert.equal(
  globalThis.__refreshSlot['user'],
  undefined,
  '模拟登录清空 user refresh_token，防止 doRefresh 用残留真实 rt 升级会话（P1-21）'
)
assert.equal(store.get('user_token_kind'), 'impersonated', 'kind 仍为 impersonated')

// ── P1-21（admin MFA）：completeMfa 的 refresh 槽必须存 refresh_token ──────────
globalThis.__refreshSlot = {}
mod.setActivePinia(mod.createPinia())
const adminStore = mod.useAdminAuthStore()
adminStore.completeMfa('access-token', 'refresh-token')
assert.equal(store.get('admin_token'), 'access-token', 'admin access token 写入 admin_token')
assert.equal(
  globalThis.__refreshSlot['admin'],
  'refresh-token',
  'admin 刷新槽存 refresh_token，而不是 access_token（P1-21）'
)

console.log('✅ authSessionKind.regression.test.mjs —— 模拟登录刷新不合并断言通过')
