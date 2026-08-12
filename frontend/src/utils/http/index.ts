/**
 * HTTP 请求封装模块
 * 基于 Axios 封装的 HTTP 请求工具，提供统一的请求/响应处理
 *
 * ## 主要功能
 *
 * - 请求/响应拦截器（自动添加 Token、统一错误处理）
 * - 401 未授权自动登出（带防抖机制）
 * - 请求失败自动重试（可配置）
 * - 统一的成功/错误消息提示
 * - 支持 GET/POST/PUT/DELETE 等常用方法
 *
 * @module utils/http
 * @author Art Design Pro Team
 */

import axios, { AxiosRequestConfig, AxiosResponse, InternalAxiosRequestConfig } from 'axios'
import { useAuthStore } from '@/stores/auth'
import { refreshSession } from '@/utils/auth/refresh'
import { useAdminAuthStore } from '@/stores/adminAuth'
import { ApiStatus } from './status'
import { HttpError, handleError, showError, showSuccess } from './error'
import { $t } from '@/locales'
import { BaseResponse } from '@/types'

/** 请求配置常量 */
const REQUEST_TIMEOUT = 15000
const LOGOUT_DELAY = 500
const MAX_RETRIES = 0
const RETRY_DELAY = 1000
const UNAUTHORIZED_DEBOUNCE_TIME = 3000

/** 鉴权域：按 URL 前缀区分，401 处理必须按域分离（H6）。 */
type AuthRealm = 'user' | 'admin'

/** 401 防抖状态：按 realm 分离在途标记，用户/管理员同时 401 互不影响。 */
const unauthorizedInFlight: Record<AuthRealm, boolean> = { user: false, admin: false }

/** P1-21 单飞刷新：同一 realm 的并发 401 共享一个 refresh，成功后重放原请求。 */
const refreshInFlight: Partial<Record<AuthRealm, Promise<string | null>>> = {}
const unauthorizedTimers: Record<AuthRealm, NodeJS.Timeout | null> = { user: null, admin: null }

/**
 * 纯函数：从请求 URL 推导鉴权域。管理端 API 前缀是 `/admin/api`，其余
 * （含业务 code=401 的业务响应）按用户域处理。禁止从旧全局 URL 推断。
 */
export function realmFromUrl(url: string | undefined): AuthRealm {
  return url && url.startsWith('/admin/api') ? 'admin' : 'user'
}

/** 扩展 AxiosRequestConfig */
interface ExtendedAxiosRequestConfig extends AxiosRequestConfig {
  showErrorMessage?: boolean
  showSuccessMessage?: boolean
}

const { VITE_API_URL, VITE_WITH_CREDENTIALS } = import.meta.env

/** Axios实例 */
const axiosInstance = axios.create({
  timeout: REQUEST_TIMEOUT,
  baseURL: VITE_API_URL,
  withCredentials: VITE_WITH_CREDENTIALS === 'true',
  validateStatus: (status) => status >= 200 && status < 300,
  transformResponse: [
    (data, headers) => {
      const contentType = headers['content-type']
      if (contentType?.includes('application/json')) {
        try {
          return JSON.parse(data)
        } catch {
          return data
        }
      }
      return data
    }
  ]
})

/**
 * 决定相对 URL 应该携带哪个 Bearer token（纯函数，便于回归测试）。
 *
 * 只给站内相对路径注入 token：绝对地址（第三方上传/外链）一律返回 null，
 * 避免 JWT 泄漏（m054 F3 / P2-73）。`/admin/api` 前缀用管理端 token，其余
 * `/api` 前缀用用户 token。
 */
export function resolveAuthorizationHeader(
  url: string,
  userToken: string,
  adminToken: string
): string | null {
  if (!url.startsWith('/')) return null
  if (url.startsWith('/admin/api')) return adminToken ? `Bearer ${adminToken}` : null
  if (url.startsWith('/api')) return userToken ? `Bearer ${userToken}` : null
  return null
}

/** 请求拦截器 */
axiosInstance.interceptors.request.use(
  (request: InternalAxiosRequestConfig) => {
    const authorization = resolveAuthorizationHeader(
      String(request.url || ''),
      useAuthStore().token,
      useAdminAuthStore().token
    )
    if (authorization) {
      request.headers.set('Authorization', authorization)
    }

    if (request.data && !(request.data instanceof FormData) && !request.headers['Content-Type']) {
      request.headers.set('Content-Type', 'application/json')
      request.data = JSON.stringify(request.data)
    }

    return request
  },
  (error) => {
    showError(createHttpError($t('httpMsg.requestConfigError'), ApiStatus.error))
    return Promise.reject(error)
  }
)

/**
 * 响应拦截器。
 *
 * 401 处理按**当前 response 的 config.url** 推导 realm（H6）：业务 code=401
 * （HTTP 200 但 data.code===401）与 HTTP 401 都从当前请求取 realm，绝不从
 * 全局变量推断——旧实现里 `unauthorizedUrl` 只在 HTTP 401 分支设置，业务
 * code=401 会读到上一次请求残留的 URL，用户/管理员同时 401 时可能只登出
 * 一个或登错域。
 */
async function tryRefreshAndRetry<T = AxiosResponse<BaseResponse>>(
  config: InternalAxiosRequestConfig | undefined,
  realm: AuthRealm
): Promise<T> {
  if (!config) throw createHttpError($t('httpMsg.unauthorized'), ApiStatus.unauthorized)
  // 刷新请求/登录请求自身 401 不重试。
  const url = String(config.url || '')
  if (url.includes('/auth/refresh') || url.includes('/auth/login')) {
    handleUnauthorizedError(realm, undefined)
    throw createHttpError($t('httpMsg.unauthorized'), ApiStatus.unauthorized)
  }
  if (!refreshInFlight[realm]) {
    refreshInFlight[realm] = refreshSession(realm).finally(() => {
      refreshInFlight[realm] = undefined
    })
  }
  const refreshed = await refreshInFlight[realm]
  if (refreshed) {
    const headers = (config.headers ?? {}) as Record<string, string>
    headers.Authorization = `Bearer ${refreshed}`
    config.headers = headers as typeof config.headers
    return (await axiosInstance.request(config)) as T
  }
  // 刷新失败 → 一次性登出（unauthorizedInFlight 去重）。
  handleUnauthorizedError(realm, undefined)
  throw createHttpError($t('httpMsg.unauthorized'), ApiStatus.unauthorized)
}

axiosInstance.interceptors.response.use(
  (response: AxiosResponse<BaseResponse>) => {
    const data = (response.data ?? {}) as Record<string, unknown>
    const code = Number(data.code)
    const msg = resolveBusinessErrorMessage(data)
    if (code === ApiStatus.success) return response
    if (code === ApiStatus.unauthorized) {
      return tryRefreshAndRetry(response.config, realmFromUrl(String(response.config?.url || '')))
    }
    throw createHttpError(msg || $t('httpMsg.requestFailed'), code)
  },
  (error) => {
    if (error.response?.status === ApiStatus.unauthorized) {
      return tryRefreshAndRetry(error.config, realmFromUrl(String(error.config?.url || ''))).catch(
        (retryErr) => {
          throw handleError(retryErr)
        }
      )
    }
    return Promise.reject(handleError(error))
  }
)

/** 业务错误消息兼容后端 `{msg}` / `{error}` / `{message}`（H6）。 */
export function resolveBusinessErrorMessage(data: Record<string, unknown> | undefined): string {
  if (!data) return ''
  const msg = data.msg ?? data.error ?? data.message
  return typeof msg === 'string' ? msg : ''
}

/** 统一创建HttpError */
function createHttpError(message: string, code: number) {
  return new HttpError(message, code)
}

/** 处理401错误（带防抖，按 realm 分离） */
function handleUnauthorizedError(realm: AuthRealm, message?: string): never {
  const error = createHttpError(message || $t('httpMsg.unauthorized'), ApiStatus.unauthorized)

  if (!unauthorizedInFlight[realm]) {
    unauthorizedInFlight[realm] = true
    logOut(realm)

    if (unauthorizedTimers[realm]) clearTimeout(unauthorizedTimers[realm])
    unauthorizedTimers[realm] = setTimeout(() => {
      unauthorizedInFlight[realm] = false
      unauthorizedTimers[realm] = null
    }, UNAUTHORIZED_DEBOUNCE_TIME)

    showError(error, true)
    throw error
  }

  throw error
}

/**
 * 管理后台路径是后台可配置的，不能写死成 "/admin"。
 *
 * 与 services/http.ts 的 currentAdminPath 相同的实现：直接读
 * services/adminPath.ts 落在 localStorage 的缓存，而不是 import 它的
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

/** 当前地址是否位于管理后台路径内（决定 admin 401 后跳回哪个登录页）。 */
function isOnAdminPath(): boolean {
  const p = window.location.pathname
  const prefix = `/${currentAdminPath()}`
  return p === prefix || p.startsWith(`${prefix}/`)
}

/** 退出登录函数（清真实业务会话，P2-73 / H6） */
function logOut(realm: AuthRealm) {
  setTimeout(() => {
    // 模板 axios 的 AuthProvider（ADP user store）从未被填充；这里清真正的
    // 业务 token。realm 由当前 response 的 config.url 决定，不跨域。
    if (realm === 'admin') {
      useAdminAuthStore().logout(false)
      // 拆掉动态路由、菜单与按钮权限，避免下一个管理员继承（P2-73）。
      void import('@/router/session').then(({ clearSession }) =>
        clearSession('admin', { keepStores: true })
      )
      // 与业务栈 services/http 的 admin 401 分支一致：清完会话立即跳回管理
      // 登录页，避免"会话已清但页面僵死"（web_auth_surface F5 —— 富文本上传
      // 走本模板栈，管理员 token 过期时不能再停在原页）。
      void import('@/services/navigate').then(({ navigateReplace }) => {
        if (!isOnAdminPath()) return
        const current = `${window.location.pathname}${window.location.search}${window.location.hash}`
        navigateReplace(`/${currentAdminPath()}/login?redirect=${encodeURIComponent(current)}`)
      })
    } else {
      // 与业务栈 services/http 的 401 行为一致：用户域也清理 worktab /
      // keep-alive 缓存并跳回登录页，避免"会话已清但页面僵死/标签残留"
      // （web_auth_surface F5）。
      useAuthStore().logout(false)
      void import('@/router/session').then(({ clearSession }) =>
        clearSession('console', { keepStores: true })
      )
      void import('@/services/navigate').then(({ navigateReplace }) => {
        const current = `${window.location.pathname}${window.location.search}${window.location.hash}`
        navigateReplace(`/login?redirect=${encodeURIComponent(current)}`)
      })
    }
  }, LOGOUT_DELAY)
}

/** 是否需要重试 */
function shouldRetry(statusCode: number) {
  return [
    ApiStatus.requestTimeout,
    ApiStatus.internalServerError,
    ApiStatus.badGateway,
    ApiStatus.serviceUnavailable,
    ApiStatus.gatewayTimeout
  ].includes(statusCode)
}

/** 请求重试逻辑 */
async function retryRequest<T>(
  config: ExtendedAxiosRequestConfig,
  retries: number = MAX_RETRIES
): Promise<T> {
  try {
    return await request<T>(config)
  } catch (error) {
    if (retries > 0 && error instanceof HttpError && shouldRetry(error.code)) {
      await delay(RETRY_DELAY)
      return retryRequest<T>(config, retries - 1)
    }
    throw error
  }
}

/** 延迟函数 */
function delay(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

/** 请求函数 */
async function request<T = any>(config: ExtendedAxiosRequestConfig): Promise<T> {
  // POST | PUT 参数自动填充
  if (
    ['POST', 'PUT'].includes(config.method?.toUpperCase() || '') &&
    config.params &&
    !config.data
  ) {
    config.data = config.params
    config.params = undefined
  }

  try {
    const res = await axiosInstance.request<BaseResponse<T>>(config)

    // 显示成功消息
    if (config.showSuccessMessage && res.data.msg) {
      showSuccess(res.data.msg)
    }

    return res.data.data as T
  } catch (error) {
    if (error instanceof HttpError && error.code !== ApiStatus.unauthorized && !error.canceled) {
      const showMsg = config.showErrorMessage !== false
      showError(error, showMsg)
    }
    return Promise.reject(error)
  }
}

/** API方法集合 */
const api = {
  get<T>(config: ExtendedAxiosRequestConfig) {
    return retryRequest<T>({ ...config, method: 'GET' })
  },
  post<T>(config: ExtendedAxiosRequestConfig) {
    return retryRequest<T>({ ...config, method: 'POST' })
  },
  put<T>(config: ExtendedAxiosRequestConfig) {
    return retryRequest<T>({ ...config, method: 'PUT' })
  },
  del<T>(config: ExtendedAxiosRequestConfig) {
    return retryRequest<T>({ ...config, method: 'DELETE' })
  },
  request<T>(config: ExtendedAxiosRequestConfig) {
    return retryRequest<T>(config)
  }
}

export default api
