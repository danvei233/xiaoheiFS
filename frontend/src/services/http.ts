import axios, { type AxiosRequestHeaders } from 'axios'
import { ElMessage, ElMessageBox, ElNotification } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import { refreshSession } from '@/utils/auth/refresh'
import { useAdminAuthStore } from '@/stores/adminAuth'
import { navigateReplace } from '@/services/navigate'

const apiBase = import.meta.env.VITE_API_BASE || ''

/**
 * 管理后台路径是后台可配置的，不能写死成 "/admin"。
 *
 * 这里直接读 services/adminPath.ts 落在 localStorage 的缓存，而不是 import 它的
 * getCachedAdminPath —— adminPath.ts 依赖本模块的 http 实例，静态引入会成环。
 */
function currentAdminPath(): string {
  try {
    return (
      (localStorage.getItem('admin_path_cache') || 'admin').trim().replace(/^\/+|\/+$/g, '') ||
      'admin'
    )
  } catch {
    return 'admin'
  }
}

function isOnAdminPath(): boolean {
  const p = window.location.pathname
  const prefix = `/${currentAdminPath()}`
  return p === prefix || p.startsWith(`${prefix}/`)
}

export const http = axios.create({
  baseURL: apiBase,
  timeout: 20000
})

let realnameModalOpen = false

/**
 * 401 并发去重：多个并行请求同时 401 时，登出 + 重定向只执行一次。
 * （P1-21：401 硬登出且并发重复处理。）
 *
 * H6：按 realm 分离在途标记。旧的单一 `unauthorizedHandling` 会被后到的
 * realm 覆盖——用户 401 处理在途时管理员 401 进入会把它顶掉，之后同 realm
 * 的第二个 401 就失去防抖，出现重复登出/弹窗；两个域同时 401 还可能只处理
 * 一个。这里用 `{user, admin}` 两个独立标记，互不干扰。
 */
const unauthorizedHandling: Record<'user' | 'admin', boolean> = { user: false, admin: false }

/**
 * P1-21 单飞刷新：同一 realm 的并发 401 共享一个 refresh 请求。刷新成功后
 * 每个请求都用新 token 重放；刷新失败才走登出（unauthorizedHandling 去重）。
 */
const refreshInProgress: Record<'user' | 'admin', Promise<string | null> | null> = {
  user: null,
  admin: null
}

const isAuthLoginRequest = (url: string): boolean => {
  if (!url) return false
  return url.includes('/api/v1/auth/login') || url.includes('/admin/api/v1/auth/login')
}

/**
 * 401 登出路径（P1-21）：刷新失败 / 无刷新机会 / 重放仍 401（_retried）时进入。
 * 同 realm 1s 内只执行一次登出 + 重定向 + 弹窗（unauthorizedHandling 去重）。
 */
function handleUnauthorized(realm: 'user' | 'admin', error: unknown): Promise<never> {
  if (unauthorizedHandling[realm]) {
    // 同 realm 已有 401 处理在途，本次只静默放行，避免重复登出/重定向/弹窗。
    return Promise.reject(error)
  }
  unauthorizedHandling[realm] = true
  setTimeout(() => {
    unauthorizedHandling[realm] = false
  }, 1000)
  const current = `${window.location.pathname}${window.location.search}${window.location.hash}`
  if (realm === 'admin') {
    const admin = useAdminAuthStore()
    admin.logout(false)
    // 401 之后必须把动态路由、菜单和按钮权限一起拆掉，否则下一个登录的
    // 管理员会继承上一个人的菜单（守卫看到已注册就会跳过重新注册）。
    void import('@/router/session').then(({ clearSession }) =>
      clearSession('admin', { keepStores: true })
    )
    if (isOnAdminPath()) {
      navigateReplace(`/${currentAdminPath()}/login?redirect=${encodeURIComponent(current)}`)
    }
  } else {
    const user = useAuthStore()
    user.logout(false)
    // 用户域 401 与显式登出/管理域 401 一致地清理工作台标签与 keep-alive
    // 缓存：worktab 是持久化状态，只调 logout() 会留下上一账号的标签页
    // （路径/标题含订单、VPS ID），下一账号登录直接继承（web_auth_surface
    // F2 / web_state_isolation F3）。
    void import('@/router/session').then(({ clearSession }) =>
      clearSession('console', { keepStores: true })
    )
    if (!isOnAdminPath()) {
      navigateReplace(`/login?redirect=${encodeURIComponent(current)}`)
    }
  }
  ElMessage.error('鉴权失败，请重新登录')
  return Promise.reject(error)
}

http.interceptors.request.use((config) => {
  const user = useAuthStore()
  const admin = useAdminAuthStore()
  if (config.url?.startsWith('/api') && user.token) {
    config.headers = config.headers || ({} as AxiosRequestHeaders)
    config.headers.Authorization = `Bearer ${user.token}`
  }
  if (config.url?.startsWith('/admin/api') && admin.token) {
    config.headers = config.headers || ({} as AxiosRequestHeaders)
    config.headers.Authorization = `Bearer ${admin.token}`
  }
  return config
})

http.interceptors.response.use(
  (res) => res,
  async (error) => {
    // 调用方主动取消（AbortController / ERR_CANCELED）不弹 toast、不记错误：
    // 列表切页/离开页面时在途请求被中断是正常流程（H6）。
    if (
      error?.code === 'ERR_CANCELED' ||
      error?.message === 'canceled' ||
      error?.config?.signal?.aborted
    ) {
      return Promise.reject(error)
    }
    const status = error?.response?.status
    const errCode = error?.response?.data?.code
    // H6：错误消息兼容后端 `{msg}` / `{error}` / `{message}`（后端业务错误
    // 主字段是 `{code,msg}`，之前漏了 msg 会显示英文 axios 原文）。
    const body = (error?.response?.data ?? {}) as Record<string, unknown>
    const bodyMsg =
      typeof body.msg === 'string'
        ? body.msg
        : typeof body.error === 'string'
          ? body.error
          : typeof body.message === 'string'
            ? body.message
            : ''
    const msg = bodyMsg || error?.message || 'Request failed'
    const url = error?.config?.url || ''
    if (
      status === 403 &&
      url.startsWith('/admin/') &&
      (errCode === 'admin_2fa_required' || errCode === 'admin_2fa_bind_required')
    ) {
      const admin = useAdminAuthStore()
      admin.setMfaGateState({
        mfaRequired: errCode === 'admin_2fa_required',
        mfaBindRequired: errCode === 'admin_2fa_bind_required',
        mfaUnlocked: false
      })
      return Promise.reject(error)
    }
    if (status === 401) {
      if (isAuthLoginRequest(url) || url.includes('/auth/refresh')) {
        return Promise.reject(error)
      }
      const realm: 'user' | 'admin' = url.startsWith('/admin/') ? 'admin' : 'user'

      // P1-21 residual（重试上限）：刷新成功后重放的原请求若再次 401
      // （_retried 已标记），说明 token 确实无效，不能再刷新重试，否则
      // 「刷新成功→重放→再 401→再刷新」会无限循环。直接走登出路径。
      if ((error.config as { _retried?: boolean } | undefined)?._retried) {
        return handleUnauthorized(realm, error)
      }

      // P1-21 单飞刷新：并发 401 共享同一个 refresh；成功后用新 token 重放原请求。
      if (!refreshInProgress[realm]) {
        refreshInProgress[realm] = refreshSession(realm).finally(() => {
          refreshInProgress[realm] = null
        })
      }
      const refreshed = await refreshInProgress[realm]
      if (refreshed && error.config) {
        const retryConfig = error.config
        const headers = (retryConfig.headers ?? {}) as Record<string, string>
        headers.Authorization = `Bearer ${refreshed}`
        retryConfig.headers = headers as typeof retryConfig.headers
        // 标记重试：重放仍 401 时不再刷新，直接登出（P1-21 residual）。
        ;(retryConfig as { _retried?: boolean })._retried = true
        return http.request(retryConfig)
      }

      return handleUnauthorized(realm, error)
    } else if (
      status === 403 &&
      url.startsWith('/api') &&
      msg.toLowerCase().includes('real name required')
    ) {
      if (!realnameModalOpen) {
        realnameModalOpen = true
        ElMessageBox.confirm('该操作需要完成实名认证，是否前往认证页面？', '需要实名认证', {
          confirmButtonText: '去认证',
          cancelButtonText: '稍后再说',
          type: 'warning'
        })
          .then(() => {
            navigateReplace('/console/realname')
          })
          .catch(() => {})
          .finally(() => {
            realnameModalOpen = false
          })
      }
    } else if (status >= 500) {
      ElNotification.error({ title: '服务端错误', message: msg })
    } else {
      ElMessage.error(msg)
    }
    return Promise.reject(error)
  }
)
