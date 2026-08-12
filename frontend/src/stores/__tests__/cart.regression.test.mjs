/**
 * 回归测试：模拟登录（impersonation）切换账号时购物车必须清空残留并加载
 * 被模拟用户的真实服务端车（web_security F3 / web_perf F2）。
 *
 * 背景：
 *  - `applyImpersonatedToken` 只换 token 不清购物车 → 上一个账号的
 *    `serverItems` 残留给被模拟用户（跨账号数据泄露）。
 *  - 模拟登录（loginToken !== token）下 `mergeGuestCart` 早退，而旧
 *    `fetchCart` 在 `guestItems` 非空时只走 merge 分支 → 被模拟用户的服务端
 *    车永远不加载，展示上一账号残留或空车。
 *
 * 修复：
 *  - `applyImpersonatedToken` 触发 `resetForAccountSwitch()`（清 serverItems/
 *    guestItems）；cart 的 token watcher 在非真实登录下也清 serverItems。
 *  - `fetchCart` 对非 merge-eligible 会话直接 `loadServerCart()`。
 *
 * 运行：`node --test src/stores/__tests__/cart.regression.test.mjs`
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
        export const storeRefreshToken = () => {}
        export const clearRefreshToken = () => {}
        export const refreshSession = async () => null
      `,
      loader: 'js'
    }))
  }
}

// 阻断 services/user → services/http → element-plus 的级联；listCart 从
// globalThis.__cartFixture 读当前"服务端购物车"，让测试可注入每个账号的车。
const userStub = {
  name: 'stub-user-service',
  setup(build) {
    build.onResolve({ filter: /^@\/services\/user$/ }, () => ({
      path: 'user-service-stub',
      namespace: 'user-service'
    }))
    build.onLoad({ filter: /.*/, namespace: 'user-service' }, () => ({
      contents: `
        export const listCart = async () => ({
          data: { items: globalThis.__cartFixture?.items ?? [] }
        })
        const noop = async () => ({ data: {} })
        export const userLogin = noop
        export const userLogout = noop
        export const getMe = async () => ({ data: null })
        export const updateMe = noop
        export const addCartItem = noop
        export const clearCart = noop
        export const deleteCartItem = noop
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

// cart.ts 顶层 import ElMessage；stub 掉避免打包整个 element-plus。
const elStub = {
  name: 'stub-element-plus',
  setup(build) {
    build.onResolve({ filter: /^element-plus$/ }, () => ({ path: 'el-stub', namespace: 'el' }))
    build.onLoad({ filter: /.*/, namespace: 'el' }, () => ({
      contents: `export const ElMessage = { warning: () => {}, error: () => {}, success: () => {} }`,
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
globalThis.__cartFixture = { items: [] }

const mod = await bundleEntry(
  `export { useCartStore } from '@/stores/cart'
   export { useAuthStore } from '@/stores/auth'
   export { createPinia, setActivePinia } from 'pinia'`,
  import.meta.url,
  { plugins: [refreshStub, userStub, elStub] }
)

const flush = () => new Promise((resolve) => setTimeout(resolve, 0))

mod.setActivePinia(mod.createPinia())
const auth = mod.useAuthStore()
const cart = mod.useCartStore()

// ── 场景 1：管理员真实登录并加载自己的服务端车 ─────────────────────────
auth.token = 'admin-token'
auth.loginToken = 'admin-token'
auth.sessionKind = 'real'
await flush()
globalThis.__cartFixture = { items: [{ id: 1, package_id: 10, qty: 1, amount: 100 }] }
await cart.fetchCart()
assert.equal(cart.serverItems.length, 1, '管理员自己的服务端车已加载')
assert.equal(cart.serverItems[0].id, 1)

// ── 场景 2：模拟登录切换账号 → 清空上一账号残留 ───────────────────────
auth.applyImpersonatedToken('userB-token')
assert.equal(auth.loginToken, '', '模拟登录 loginToken 为空')
await flush() // watcher 触发
await flush() // applyImpersonatedToken 动态 import 的 reset 落地
assert.equal(cart.serverItems.length, 0, '模拟登录切换后清空上一账号服务端购物车（F3）')

// ── 场景 3：模拟登录下 fetchCart 加载被模拟用户的服务端车 ──────────────
// 即使设备级游客车非空，也不并入、不早退（F2）。
cart.guestItems = [{ id: 'g1', package_id: 99, qty: 2, amount: 50, spec: {}, source: 'guest' }]
globalThis.__cartFixture = { items: [{ id: 5, package_id: 55, qty: 1, amount: 888 }] }
await cart.fetchCart()
assert.equal(cart.serverItems.length, 1, '被模拟用户的服务端车被加载')
assert.equal(cart.serverItems[0].id, 5, '加载的是当前 token 的服务端车')
assert.equal(cart.items.length, 1, 'items 只展示服务端车，游客车不得并入')
assert.equal(cart.items[0].id, 5, '展示的是被模拟用户的车，而非上一账号')

// ── 场景 4：连续模拟用户 A→B，B 不看到 A 的车 ─────────────────────────
globalThis.__cartFixture = { items: [{ id: 10, package_id: 100, qty: 1, amount: 10 }] }
await cart.fetchCart()
assert.equal(cart.serverItems[0].id, 10, '用户 A 的服务端车')
auth.applyImpersonatedToken('userC-token')
await flush()
await flush()
assert.equal(cart.serverItems.length, 0, '切换到 B 前清空 A 的车')
globalThis.__cartFixture = { items: [{ id: 20, package_id: 200, qty: 1, amount: 20 }] }
await cart.fetchCart()
assert.equal(cart.serverItems[0].id, 20, 'B 只看到自己的车')
assert.ok(!cart.serverItems.some((item) => item.id === 10), 'B 的视图不含 A 的条目')

// ── resetForAccountSwitch 直接行为 ─────────────────────────────────────
cart.serverItems = [{ id: 1, package_id: 1, qty: 1, amount: 1, spec: {}, source: 'server' }]
cart.guestItems = [{ id: 'g', package_id: 2, qty: 1, amount: 1, spec: {}, source: 'guest' }]
cart.resetForAccountSwitch()
assert.equal(cart.serverItems.length, 0, 'resetForAccountSwitch 清空 serverItems')
assert.equal(cart.guestItems.length, 0, 'resetForAccountSwitch 清空 guestItems')

console.log('✅ cart.regression.test.mjs —— 模拟登录购物车隔离断言通过')
