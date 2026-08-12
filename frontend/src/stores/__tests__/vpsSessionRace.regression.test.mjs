/**
 * 回归测试：登出/会话切换后，在途请求的旧账号响应不得写回 store
 * （web_data_races F7）。
 *
 * 背景：`logout()` 同步清 token、异步 reset vps/dashboard；若旧账号的
 *   `listVps`/`fetchUserDashboard` 在途，登出后响应晚到会把旧账号数据重新
 *   填回 store，新账号登录后可能被旧响应覆盖（跨账号残留）。
 *
 * 修复：vps/dashboard store 在请求发起时捕获 `auth.token`，响应到达时若
 *   token 已变化则丢弃结果、不写回。
 *
 * 运行：`node --test src/stores/__tests__/vpsSessionRace.regression.test.mjs`
 */
import { bundleEntry } from '../../../tests/_bundle.mjs'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

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

const userStub = {
  name: 'stub-user-service',
  setup(build) {
    build.onResolve({ filter: /^@\/services\/user$/ }, () => ({
      path: 'user-service-stub',
      namespace: 'user-service'
    }))
    build.onLoad({ filter: /.*/, namespace: 'user-service' }, () => ({
      contents: `
        export const listVps = async () => {
          await globalThis.__vpsGate
          return { data: { items: globalThis.__vpsFixture ?? [] } }
        }
        export const getVpsDetail = async (id) => {
          await globalThis.__detailGate
          const fixture = globalThis.__vpsDetailFixture ?? {}
          return { data: fixture[String(id)] ?? null }
        }
        const noop = async () => ({ data: {} })
        export const getVpsPanelUrl = noop
        export const refreshVps = noop
        export const userLogin = noop
        export const userLogout = noop
        export const getMe = async () => ({ data: null })
        export const updateMe = noop
        export const getDashboard = noop
        export const listOrders = noop
        export const getWallet = noop
        export const getRealNameStatus = noop
        export const addCartItem = noop
        export const clearCart = noop
        export const deleteCartItem = noop
        export const listCart = async () => ({ data: { items: [] } })
        export const updateCartItem = noop
      `,
      loader: 'js'
    }))
  }
}

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
// 手动控制 listVps 何时返回；每个场景换新的 gate。
globalThis.__vpsFixture = []
globalThis.__vpsGate = Promise.resolve()
globalThis.__vpsDetailFixture = {}
globalThis.__detailGate = Promise.resolve()

const mod = await bundleEntry(
  `export { useVpsStore } from '@/stores/vps'
   export { useAuthStore } from '@/stores/auth'
   export { resolveVpsIp } from '@/utils/vpsAccess'
   export { createPinia, setActivePinia } from 'pinia'`,
  import.meta.url,
  { plugins: [refreshStub, userStub, elStub] }
)

mod.setActivePinia(mod.createPinia())
const auth = mod.useAuthStore()
const vps = mod.useVpsStore()

// ── 场景 1：旧账号在途，登出后响应晚到 → 丢弃，不写回 ─────────────────
let releaseA
globalThis.__vpsGate = new Promise((resolve) => (releaseA = resolve))
globalThis.__vpsFixture = [{ id: 1, name: 'old-account-vps' }]
auth.token = 'userA'
const pendingFetch = vps.fetchVps() // 在途，等待 gate

auth.logout() // 登出：token 清空
assert.equal(auth.token, '', '登出后 token 为空')

releaseA() // 旧账号响应此时才返回
await pendingFetch
await new Promise((resolve) => setTimeout(resolve, 0))
assert.equal(vps.items.length, 0, '登出后在途响应不得写回旧账号数据')

// ── 场景 2：A 在途，切换为 B 后 A 响应晚到 → 不覆盖 B ─────────────────
globalThis.__vpsGate = new Promise((resolve) => (releaseA = resolve))
globalThis.__vpsFixture = [{ id: 1, name: 'old-account-vps' }]
auth.token = 'userA'
const pendingA = vps.fetchVps() // A 在途

// 切换为 B：B 发起自己的请求（gate 已释放，立即返回）
auth.token = 'userB'
auth.loginToken = 'userB'
auth.sessionKind = 'real'
globalThis.__vpsGate = Promise.resolve()
globalThis.__vpsFixture = [{ id: 2, name: 'userB-vps' }]
await vps.fetchVps()
assert.equal(vps.items.length, 1, 'B 自己的 VPS 已加载')
assert.equal(vps.items[0].id, 2, '展示 B 的 VPS')

releaseA() // A 的晚到响应返回，不得覆盖 B
await pendingA
await new Promise((resolve) => setTimeout(resolve, 0))
assert.equal(vps.items[0].id, 2, 'A 的晚到响应不得覆盖 B 的数据')

// ── 场景 3：同账号内跨 VPS 导航，请求在途时 current 不得残留上一台 ───────
// web_state_isolation F1：vps store 的 current 是单槽，/console/vps/1 →
// /console/vps/2 时若 VPS2 响应未到，模板会渲染 VPS1 的明文 access_info。
// 修复后 fetchDetail 在目标 id 变化时立即清空 current。
auth.token = 'userC'
auth.loginToken = 'userC'
auth.sessionKind = 'real'
globalThis.__vpsDetailFixture = {
  1: { id: 1, name: 'vps-1', access_info: { os_password: 'secret-1' } },
  2: { id: 2, name: 'vps-2', access_info: { os_password: 'secret-2' } }
}
globalThis.__detailGate = Promise.resolve()

// 先加载 VPS1，current 为 VPS1（此时 gate 已释放，立即返回）
await vps.fetchDetail(1)
assert.equal(vps.current?.id, 1, 'VPS1 详情加载完成')
assert.equal(vps.current.access_info.os_password, 'secret-1')

// 置慢 gate：发起 VPS2 请求，在途时 current 必须已清空（不残留 VPS1 明文凭据）
let releaseDetail
globalThis.__detailGate = new Promise((resolve) => (releaseDetail = resolve))
const pendingDetail2 = vps.fetchDetail(2)
await new Promise((resolve) => setTimeout(resolve, 0))
assert.equal(vps.current, null, 'VPS2 请求在途时不得残留 VPS1 的明文凭据')

releaseDetail() // VPS2 响应落地
await pendingDetail2
assert.equal(vps.current?.id, 2, 'VPS2 响应后 current 为 VPS2')

// ── 场景 4：VPS2 请求失败时 current 不得残留 VPS1 ──────────────────────
globalThis.__detailGate = Promise.resolve()
globalThis.__vpsDetailFixture = { 2: null } // VPS2 不存在 → 后端 404 → data null
await vps.fetchDetail(2)
assert.equal(vps.current, null, '目标 VPS 请求失败后 current 为空，不残留上一台凭据')

// ── 场景 5：慢响应下旧 VPS1 请求后到，不得覆盖已加载的 VPS2 ──────────────
globalThis.__vpsDetailFixture = {
  1: { id: 1, name: 'vps-1', access_info: { os_password: 'secret-1' } },
  2: { id: 2, name: 'vps-2', access_info: { os_password: 'secret-2' } }
}
let releaseStale1
globalThis.__detailGate = new Promise((resolve) => (releaseStale1 = resolve))
const staleVps1 = vps.fetchDetail(1) // VPS1 在途（慢）

await new Promise((resolve) => setTimeout(resolve, 0))
globalThis.__detailGate = Promise.resolve()
await vps.fetchDetail(2) // VPS2 立即返回
assert.equal(vps.current?.id, 2, 'VPS2 已加载')

releaseStale1() // 旧的 VPS1 响应此时才返回
await staleVps1
await new Promise((resolve) => setTimeout(resolve, 0))
assert.equal(vps.current?.id, 2, '慢响应下旧的 VPS1 不得覆盖 VPS2（id 不匹配被丢弃）')

// ── 场景 6：VPS 列表 IP 解析优先 row.ip，再回退 access_info（P2-4 跨层）──
assert.equal(mod.resolveVpsIp({ ip: '1.2.3.4' }), '1.2.3.4', '顶层 row.ip 优先')
assert.equal(
  mod.resolveVpsIp({ ip: '1.2.3.4', access_info: { remote_ip: '9.9.9.9' } }),
  '1.2.3.4',
  'row.ip 覆盖 access_info.remote_ip'
)
assert.equal(
  mod.resolveVpsIp({ access_info: { remote_ip: '9.9.9.9' } }),
  '9.9.9.9',
  'access_info.remote_ip 回退'
)
assert.equal(
  mod.resolveVpsIp({ access_info_json: '{"remote_ip":"8.8.8.8"}' }),
  '8.8.8.8',
  'access_info_json 字符串回退'
)
assert.equal(
  mod.resolveVpsIp({ access_info: null, AccessInfo: null }),
  '-',
  'access_info 为 null 不崩溃'
)
assert.equal(mod.resolveVpsIp({}), '-', '无 IP 显示占位符')

// ── 场景 7：页面本地 credentials 请求必须同时可取消并校验请求代次 ─────
const detailSource = readFileSync(
  new URL('../../views/console/vps/detail.vue', import.meta.url),
  'utf8'
)
assert.match(detailSource, /credentialsRequestVersion/, 'credentials 使用单调请求代次')
assert.match(
  detailSource,
  /credentialsAbortController\?\.abort\(\)/,
  '路由切换取消旧 credentials 请求'
)
assert.match(
  detailSource,
  /requestedId !== String\(route\.params\.id \|\| ''\)/,
  '响应写回前核对当前 VPS id'
)

console.log('✅ vpsSessionRace.regression.test.mjs —— 会话切换/跨 VPS 在途响应隔离断言通过')
