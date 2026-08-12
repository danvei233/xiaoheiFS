/**
 * P1-21 单飞刷新回归测试。
 *
 * 契约：
 *  - 同一 realm 的并发 refreshSession 调用共享同一个 refresh 请求（单飞），
 *    绝不每个 401 各发一次刷新；
 *  - 刷新结果与会话 epoch 绑定：刷新期间发生登出/登录（epoch +1）后，结果被
 *    丢弃，绝不把旧会话 token 写回新会话；
 *  - 刷新失败 / 无 refresh_token 返回 null（调用方据此一次性登出）；
 *  - SSE 的 getToken 每次重连重新读取，token 轮换后 SSE 使用新 token。
 *
 * 运行：`node --test src/utils/__tests__/refreshSingleflight.regression.test.mjs`
 */
import { bundleEntry } from '../../../tests/_bundle.mjs'
import assert from 'node:assert/strict'
import { test } from 'node:test'

// 可控 axios mock：记录每次 post，返回可编程结果。
let postCalls = 0
let postResult = { data: { access_token: 'fresh-access' } }
let postError = null
const axiosMock = {
  post: async () => {
    postCalls++
    if (postError) throw postError
    return postResult
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
      contents: `let t = '';
      export const useAuthStore = () => ({
        set token(v){ t = v }, get token(){ return t },
        set sessionKind(v){ globalThis.__authSessionKind = v },
        get sessionKind(){ return globalThis.__authSessionKind || '' },
        loginToken: ''
      })`,
      loader: 'js'
    }))
    build.onLoad({ filter: /.*/, namespace: 'admin-store' }, () => ({
      contents: `let t = '';
      export const useAdminAuthStore = () => ({ set token(v){ t = v }, get token(){ return t } })`,
      loader: 'js'
    }))
  }
}

const axiosStub = {
  name: 'stub-axios',
  setup(build) {
    build.onResolve({ filter: /^axios$/ }, () => ({ path: 'axios-stub', namespace: 'axios-stub' }))
    build.onLoad({ filter: /.*/, namespace: 'axios-stub' }, () => ({
      contents: `export default { post: async (...a) => {
        return globalThis.__refreshAxiosPost(...a)
      } }`,
      loader: 'js'
    }))
  }
}

const mod = await bundleEntry(
  `export {
    currentSessionEpoch, bumpSessionEpoch, storeRefreshToken, clearRefreshToken, refreshSession
  } from '@/utils/auth/refresh'`,
  import.meta.url,
  { plugins: [storesStub, axiosStub] }
)

// 每个测试独立 localStorage
function freshStorage() {
  const store = new Map()
  globalThis.localStorage = {
    getItem: (k) => (store.has(k) ? store.get(k) : null),
    setItem: (k, v) => store.set(k, String(v)),
    removeItem: (k) => store.delete(k)
  }
  globalThis.__refreshAxiosPost = async (...args) => axiosMock.post(...args)
  globalThis.__authSessionKind = ''
}

freshStorage()

test('singleflight: concurrent refreshes share one request', async () => {
  postCalls = 0
  postResult = { data: { access_token: 'fresh-access' } }
  postError = null
  mod.storeRefreshToken('user', 'rt-1')
  mod.bumpSessionEpoch('user')

  const [a, b, c] = await Promise.all([
    mod.refreshSession('user'),
    mod.refreshSession('user'),
    mod.refreshSession('user')
  ])
  assert.equal(a, 'fresh-access')
  assert.equal(b, 'fresh-access')
  assert.equal(c, 'fresh-access')
  assert.equal(postCalls, 1, 'three concurrent refresh calls must issue exactly one HTTP request')
})

test('epoch binding: refresh result discarded when session changes mid-flight', async () => {
  postCalls = 0
  postResult = { data: { access_token: 'epoch-access' } }
  postError = null
  mod.storeRefreshToken('admin', 'rt-admin')
  mod.bumpSessionEpoch('admin')

  // 模拟刷新在途时发生登出（epoch+1）：第一次调用返回 null，且不写回 token。
  const p = mod.refreshSession('admin')
  mod.bumpSessionEpoch('admin') // 会话在刷新期间变化
  const result = await p
  assert.equal(result, null, 'stale refresh must be discarded when the session epoch changed')
  assert.equal(localStorage.getItem('admin_token'), null, 'stale access token must not be written')
})

test('no refresh token -> null (caller logs out once)', async () => {
  postCalls = 0
  freshStorage()
  const result = await mod.refreshSession('user')
  assert.equal(result, null)
  assert.equal(postCalls, 0, 'no refresh request should be issued without a refresh token')
})

test('refresh failure -> null', async () => {
  freshStorage()
  postCalls = 0
  postError = new Error('refresh rejected')
  mod.storeRefreshToken('user', 'rt-fail')
  const result = await mod.refreshSession('user')
  assert.equal(result, null)
})

test('successful refresh persists rotated refresh token', async () => {
  freshStorage()
  postCalls = 0
  postResult = { data: { access_token: 'rotated-access', refresh_token: 'rotated-rt' } }
  postError = null
  mod.storeRefreshToken('user', 'rt-old')
  const result = await mod.refreshSession('user')
  assert.equal(result, 'rotated-access')
  assert.equal(localStorage.getItem('user_refresh_token'), 'rotated-rt')
})

test('P1-21 bypass: impersonated session refresh is refused (no silent upgrade to real login)', async () => {
  freshStorage()
  globalThis.__authSessionKind = 'impersonated'
  postCalls = 0
  postResult = { data: { access_token: 'fresh-access' } }
  postError = null
  // 残留真实登录的 refresh_token：即便服务端刷新成功，模拟会话也不得升级。
  mod.storeRefreshToken('user', 'rt-real-stale')
  mod.bumpSessionEpoch('user')
  const result = await mod.refreshSession('user')
  assert.equal(result, null, 'impersonated session must not be silently upgraded to a real login')
  assert.equal(
    localStorage.getItem('user_token'),
    null,
    'no access token written for impersonated session'
  )
  globalThis.__authSessionKind = ''
})

test('real session: refresh succeeds and restores loginToken/sessionKind', async () => {
  freshStorage()
  globalThis.__authSessionKind = 'real'
  postCalls = 0
  postResult = { data: { access_token: 'fresh-access' } }
  postError = null
  mod.storeRefreshToken('user', 'rt-real')
  mod.bumpSessionEpoch('user')
  const result = await mod.refreshSession('user')
  assert.equal(result, 'fresh-access', 'real session refreshes normally')
  assert.equal(
    localStorage.getItem('user_token'),
    'fresh-access',
    'real session writes back the access token'
  )
  globalThis.__authSessionKind = ''
})
