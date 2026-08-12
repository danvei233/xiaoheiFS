/**
 * 会话结束时的统一清理
 *
 * 登出这件事横跨三处状态，任何一处漏掉都会留下真实缺陷：
 *
 *   1. 业务鉴权 store（`stores/auth` / `stores/adminAuth`）—— 真正的 token 在这里
 *   2. ADP 的动态路由注册与菜单（`RouteRegistry` / `menuStore`）
 *   3. ADP 的 user store —— `v-auth` 的按钮权限来源
 *
 * 模板自带的 `useUserStore().logOut()` 只清 2 和 3，且把跳转写死到用户登录页；
 * 本项目的 `adminAuth.logout()` 只清 1。两边各清一半的后果是：
 *
 *   - 管理后台点「退出登录」并没有真的退出，admin_token 还在 localStorage 里
 *   - 换一个权限更小的管理员登录，会继承上一个人的菜单和按钮，
 *     因为 `routeRegistry.isRegistered()` 仍为 true，守卫会跳过重新注册
 *
 * 所以登出必须走这里。
 *
 * @module router/session
 */
import type { RouteLocationRaw } from 'vue-router'
import { useUserStore } from '@/store/modules/user'
import { useWorktabStore } from '@/store/modules/worktab'
import { useAuthStore } from '@/stores/auth'
import { clearRefreshToken, bumpSessionEpoch } from '@/utils/auth/refresh'
import { useAdminAuthStore } from '@/stores/adminAuth'
import { getCachedAdminPath } from '@/services/adminPath'
import { RoutesAlias } from './routesAlias'
import { resetRouterState } from './guards/beforeEach'

export type SessionRealm = 'console' | 'admin'

/** 判断当前地址属于哪个需要鉴权的域 */
export function currentSessionRealm(path: string): SessionRealm {
  const adminPath = getCachedAdminPath()
  if (adminPath && (path === `/${adminPath}` || path.startsWith(`/${adminPath}/`))) {
    return 'admin'
  }
  return 'console'
}

/** 对应域的登录页 */
export function loginRouteFor(realm: SessionRealm, redirect?: string): RouteLocationRaw {
  const query = redirect ? { redirect } : undefined
  if (realm === 'admin') {
    return { path: `/${getCachedAdminPath()}/login`, query }
  }
  return { path: RoutesAlias.Login, query }
}

/**
 * 清空一个域的全部会话状态。
 *
 * @param realm    要清理的域
 * @param options  `keepStores` 用于 401 场景：鉴权 store 已由拦截器清过，
 *                 这里只需要拆掉路由与权限状态，避免重复触发。
 */
export function clearSession(realm: SessionRealm, options: { keepStores?: boolean } = {}): void {
  const userStore = useUserStore()

  // P1-21：会话 epoch +1，使任何在途刷新结果失效；同时清除 refresh_token，
  // 之后 401 不再尝试刷新而是直接登出。
  bumpSessionEpoch(realm === 'admin' ? 'admin' : 'user')
  clearRefreshToken(realm === 'admin' ? 'admin' : 'user')

  if (!options.keepStores) {
    if (realm === 'admin') {
      useAdminAuthStore().logout()
    } else {
      useAuthStore().logout()
    }
  }

  userStore.setLockStatus(false)
  userStore.setLockPassword('')

  // 跨账号隔离：console 域登出时清掉工作台标签与 keep-alive 缓存（m071 F2
  // 残余）。admin 域由 setUserInfo 的 prevUserId 比对按账号清空，这里兜底
  // 清 console。注意工作台 store 是 persist 的，必须清它的持久化状态。
  const worktabStore = useWorktabStore()
  worktabStore.opened = []
  worktabStore.keepAliveExclude = []

  if (realm === 'admin') {
    // 按钮权限来源，必须清掉，否则下一个管理员会继承上一个人的按钮
    userStore.setUserInfo({
      buttons: [],
      roles: [],
      userId: 0,
      userName: '',
      email: '',
      avatar: ''
    })
    // 动态路由与菜单，不清的话守卫会认为已注册而跳过重新注册
    resetRouterState(0)
  }
}
