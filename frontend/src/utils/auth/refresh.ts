/**
 * P1-21 单飞（singleflight）会话刷新。
 *
 * 401 硬登出 + 并发重复处理是 Issue #51 P1-21 的核心：多个并行请求同时 401 时
 * 旧实现每个请求都各自登出/重定向。本模块引入「同一 realm 只发一个 refresh 请求」
 * 的单飞语义：并发 401 共享同一个刷新 Promise，刷新成功后重放原请求，失败才
 * 一次性登出。
 *
 * 会话用 realm + epoch 绑定：
 *   - realm：user（/api/v1）与 admin（/admin/api/v1）各自独立的 token / 刷新链。
 *   - epoch：每次登出/登录 +1；刷新发起时记录 epoch，响应回来若 epoch 已变
 *     （期间发生过登出/登录），刷新结果被丢弃，绝不把旧会话的 token 写回新会话。
 *
 * SSE 使用同一授权生命周期：SSE 从 auth store 读取 token，刷新成功后后续重连
 * 自动使用新 token；刷新失败（401 → 登出）后 SSE 停在终端状态。
 */

export type AuthRealm = 'user' | 'admin'

const REFRESH_TOKEN_KEY: Record<AuthRealm, string> = {
  user: 'user_refresh_token',
  admin: 'admin_refresh_token'
}

const EPOCH_KEY: Record<AuthRealm, string> = {
  user: 'user_session_epoch',
  admin: 'admin_session_epoch'
}

const ACCESS_TOKEN_KEY: Record<AuthRealm, string> = {
  user: 'user_token',
  admin: 'admin_token'
}

/** 每个 realm 的刷新单飞：在途刷新共享同一个 Promise。 */
const inFlight: Partial<Record<AuthRealm, Promise<string | null>>> = {}

/** 读取当前会话 epoch（localStorage 不可用时按 0 处理）。 */
export function currentSessionEpoch(realm: AuthRealm): number {
  try {
    return Number(localStorage.getItem(EPOCH_KEY[realm]) || '0') || 0
  } catch {
    return 0
  }
}

/** 登出/登录时递增会话 epoch，使在途刷新结果失效。 */
export function bumpSessionEpoch(realm: AuthRealm): void {
  try {
    localStorage.setItem(EPOCH_KEY[realm], String(currentSessionEpoch(realm) + 1))
  } catch {
    /* storage 不可用时忽略：进程内单飞标记仍能防止重复登出 */
  }
}

/** 持久化登录/刷新返回的 refresh_token。 */
export function storeRefreshToken(realm: AuthRealm, token: string): void {
  try {
    if (token) {
      localStorage.setItem(REFRESH_TOKEN_KEY[realm], token)
    } else {
      localStorage.removeItem(REFRESH_TOKEN_KEY[realm])
    }
  } catch {
    /* ignore */
  }
}

/** 清空 refresh_token（登出时调用）。 */
export function clearRefreshToken(realm: AuthRealm): void {
  try {
    localStorage.removeItem(REFRESH_TOKEN_KEY[realm])
  } catch {
    /* ignore */
  }
}

function accessTokenStorageKey(realm: AuthRealm): string {
  return ACCESS_TOKEN_KEY[realm]
}

/**
 * 单飞刷新：同一 realm 并发调用只发一个 refresh 请求，共享同一个结果。
 * 返回新 access token；刷新失败 / 无 refresh_token / epoch 已变返回 null。
 */
export function refreshSession(realm: AuthRealm): Promise<string | null> {
  if (inFlight[realm]) {
    return inFlight[realm]!
  }
  inFlight[realm] = doRefresh(realm).finally(() => {
    inFlight[realm] = undefined
  })
  return inFlight[realm]!
}

async function doRefresh(realm: AuthRealm): Promise<string | null> {
  let refreshToken = ''
  try {
    refreshToken = localStorage.getItem(REFRESH_TOKEN_KEY[realm]) || ''
  } catch {
    return null
  }
  if (!refreshToken) return null

  const epochAtStart = currentSessionEpoch(realm)
  const url = realm === 'admin' ? '/admin/api/v1/auth/refresh' : '/api/v1/auth/refresh'

  // 直接用裸 axios，避免命中本模块自己的 401 拦截器造成刷新递归。
  const { default: axios } = await import('axios')
  let res: { data?: { access_token?: string; refresh_token?: string } }
  try {
    res = await axios.post(url, { refresh_token: refreshToken }, { timeout: 15000 })
  } catch {
    return null
  }

  const access = res?.data?.access_token
  if (!access) return null

  // 会话 epoch 绑定：刷新期间发生过登出/登录，结果作废。
  if (currentSessionEpoch(realm) !== epochAtStart) return null

  // 写回 auth store 与 localStorage，后续请求自动使用新 token。懒加载避免
  // 静态引入 auth store 时把整条 services/http → router → .vue 链拉进打包。
  try {
    if (realm === 'admin') {
      const { useAdminAuthStore } = await import('@/stores/adminAuth')
      ;(useAdminAuthStore() as { token: string }).token = access
    } else {
      const { useAuthStore } = await import('@/stores/auth')
      const store = useAuthStore() as {
        token: string
        loginToken: string
        sessionKind: string
      }
      // P1-21（纵深防御）：模拟登录（impersonated）是受限会话。
      // applyImpersonatedToken 已清 refresh_token 并 bump epoch，这里再兜一层：
      // 即便 localStorage 残留真实 refresh_token，也绝不把模拟会话升级成真实
      // 登录——不写回 loginToken / sessionKind='real'，直接放弃刷新。
      if (store.sessionKind === 'impersonated') return null
      store.token = access
      store.loginToken = access
      store.sessionKind = 'real'
    }
    localStorage.setItem(accessTokenStorageKey(realm), access)
  } catch {
    return null
  }

  // 服务端可能轮换 refresh_token；顺手持久化新的。
  if (res?.data?.refresh_token) {
    storeRefreshToken(realm, res.data.refresh_token)
  }
  return access
}
