import { defineStore } from 'pinia'
import { userLogin, userLogout, getMe, updateMe } from '@/services/user'
import { storeRefreshToken, clearRefreshToken, bumpSessionEpoch } from '@/utils/auth/refresh'

const STORAGE_KEY = 'user_token'
/** 会话种类键：区分"真实登录"与"模拟登录（impersonation）"。 */
const AUTH_KIND_KEY = 'user_token_kind'

export type AuthSessionKind = 'real' | 'impersonated' | ''

export interface StoredAuthSession {
  token: string
  /** 只有真实登录建立的 loginToken 才会在刷新后恢复；模拟登录保持为空。 */
  loginToken: string
  kind: AuthSessionKind
}

/**
 * 纯函数：从 localStorage 还原会话状态（H4）。
 *
 * loginToken 不能从 token 推断——模拟登录（impersonation）直接写 token，
 * 刷新后如果把它当成真实登录，`cart.mergeGuestCart` 会把设备级游客购物车
 * 并进被模拟用户的服务端账号。这里显式持久化会话种类，只有
 * `user_token_kind === 'real'` 时才恢复 loginToken。
 */
export function readStoredAuthSession(): StoredAuthSession {
  let token = ''
  let kind: AuthSessionKind = ''
  try {
    token = localStorage.getItem(STORAGE_KEY) || ''
    const rawKind = localStorage.getItem(AUTH_KIND_KEY) || ''
    kind = rawKind === 'real' || rawKind === 'impersonated' ? rawKind : ''
  } catch {
    // storage 不可用时按空会话处理
  }
  return {
    token,
    loginToken: kind === 'real' ? token : '',
    // 无 token 时 kind 无意义，统一归空。
    kind: token ? kind : ''
  }
}

export const useAuthStore = defineStore('auth', {
  state: () => {
    const session = readStoredAuthSession()
    return {
      token: session.token,
      /**
       * 当前会话是否来自"真实登录"（auth.login()）。
       *
       * 模拟登录（impersonation）直接写 `auth.token`，不走 login()，此时
       * loginToken 保持为空。购物车合并（mergeGuestCart）只有真实登录才允许，
       * 防止管理员以用户身份进入时把管理员匿名浏览攒下的游客购物车并进
       * 真实用户账号（P2-70 / P3-37 / m053 H2）。刷新后凭持久化的
       * `user_token_kind` 区分，绝不从 token 推断。
       */
      loginToken: session.loginToken,
      /** 会话种类：'real' | 'impersonated' | ''（空=未登录） */
      sessionKind: session.kind as AuthSessionKind,
      loading: false,
      profile: null
    }
  },
  actions: {
    async login(payload) {
      this.loading = true
      try {
        const res = await userLogin(payload)
        const token = res.data?.access_token || ''
        this.profile = res.data?.user || null
        this.token = token
        this.loginToken = token
        this.sessionKind = token ? 'real' : ''
        if (token) {
          localStorage.setItem(STORAGE_KEY, token)
          localStorage.setItem(AUTH_KIND_KEY, 'real')
          storeRefreshToken(
            'user',
            (res.data as { refresh_token?: string } | undefined)?.refresh_token || ''
          )
          bumpSessionEpoch('user')
          // 把登录前攒下的游客购物车并入服务端购物车。
          //
          // cart store 自己 watch 了 token，但只有在它已经被实例化时才会触发；
          // 直接打开 /login 时公开站外壳没挂载过，watcher 不存在。游客购物车是
          // 持久化的，所以这里显式实例化并合并一次，避免要等用户走到购物车页才生效。
          //
          // 故意不覆盖模拟登录（impersonation）：那是管理员以用户身份进入，
          // 把管理员匿名浏览时攒的购物车并进真实用户的账号是错的。
          void (await import('@/stores/cart')).useCartStore().mergeGuestCart()
        } else {
          localStorage.removeItem(STORAGE_KEY)
          localStorage.removeItem(AUTH_KIND_KEY)
        }
        return token
      } catch {
        return ''
      } finally {
        this.loading = false
      }
    },
    /**
     * 模拟登录（impersonation）：写入用户 token 但标记为 'impersonated'，
     * 使 loginToken 保持为空，刷新后也不会被误认成真实登录。
     *
     * 切换账号必须与 logout() 一致地清掉用户域业务数据（vps / dashboard /
     * cart 服务端购物车），否则上一账号的仪表盘指标、VPS 列表、服务端购物车
     * 会残留给被模拟用户（web_security F3 / web_perf F2）。动态导入避免
     * store 环（vps/dashboard/cart → services → http → auth）。
     */
    applyImpersonatedToken(token: string) {
      this.token = token || ''
      this.loginToken = ''
      this.sessionKind = token ? 'impersonated' : ''
      this.profile = null
      if (token) {
        localStorage.setItem(STORAGE_KEY, token)
        localStorage.setItem(AUTH_KIND_KEY, 'impersonated')
        // P1-21（绕过）：模拟登录必须是受限会话。清掉真实登录残留的
        // refresh_token 并递增会话 epoch（作废在途刷新），否则 access token
        // 过期后 doRefresh 会用残留的真实 refresh_token 把模拟会话静默升级成
        // 原用户的真实登录（loginToken/sessionKind='real'）。
        clearRefreshToken('user')
        bumpSessionEpoch('user')
      } else {
        localStorage.removeItem(STORAGE_KEY)
        localStorage.removeItem(AUTH_KIND_KEY)
      }
      void import('@/stores/vps').then(({ useVpsStore }) => useVpsStore().reset())
      void import('@/stores/dashboard').then(({ useDashboardStore }) => useDashboardStore().reset())
      void import('@/stores/cart').then(({ useCartStore }) =>
        useCartStore().resetForAccountSwitch()
      )
    },
    async fetchMe() {
      const res = await getMe()
      this.profile = res.data || null
    },
    async updateProfile(payload) {
      const res = await updateMe(payload)
      this.profile = res.data || null
    },
    logout(notifyServer = true) {
      const accessToken = this.token
      if (notifyServer && accessToken && this.sessionKind === 'real') {
        void userLogout(accessToken).catch(() => undefined)
      }
      this.token = ''
      this.loginToken = ''
      this.sessionKind = ''
      this.profile = null
      localStorage.removeItem(STORAGE_KEY)
      localStorage.removeItem(AUTH_KIND_KEY)
      clearRefreshToken('user')
      bumpSessionEpoch('user')
      // 清掉用户域业务 store，避免跨账号残留明文 VPS access_info / 仪表盘数据
      // （P2-69 / m053 H1）。动态导入避免 store 环（vps/dashboard → services → http → auth）。
      void import('@/stores/vps').then(({ useVpsStore }) => useVpsStore().reset())
      void import('@/stores/dashboard').then(({ useDashboardStore }) => useDashboardStore().reset())
    }
  }
})
