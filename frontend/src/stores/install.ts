import { defineStore } from 'pinia'
import { getInstallStatus } from '@/services/user'

const INSTALL_STATUS_KEY = 'install_status_cached'

interface InstallState {
  loaded: boolean
  installed: boolean
}

/**
 * 安装门禁的持久化缓存（web_bundle_mobile F6）。
 *
 * 守卫每次整页加载都 `await install.fetchStatus()` 才解析路由，首屏多 1 个
 * RTT 串行瀑布。对已安装站点该检查结果恒定不变，因此把"已安装"持久化到
 * localStorage：缓存命中时 loaded=true 直接放行，后台再静默刷新一次以捕捉
 * 重新安装/降级。未安装缓存不持久化（安装向导要每次都问后端）。
 */
function readCachedInstalled(): boolean {
  try {
    return localStorage.getItem(INSTALL_STATUS_KEY) === 'true'
  } catch {
    return false
  }
}

export const useInstallStore = defineStore('install', {
  state: (): InstallState => {
    const cached = readCachedInstalled()
    return {
      loaded: cached,
      installed: cached
    }
  },

  actions: {
    async fetchStatus() {
      try {
        const res = await getInstallStatus()
        this.installed = !!res.data?.installed
      } catch {
        // If status API is unavailable, do not hard-block navigation.
        this.installed = true
      } finally {
        this.loaded = true
        try {
          if (this.installed) {
            localStorage.setItem(INSTALL_STATUS_KEY, 'true')
          } else {
            localStorage.removeItem(INSTALL_STATUS_KEY)
          }
        } catch {
          // storage 不可用时忽略
        }
      }
    }
  }
})
