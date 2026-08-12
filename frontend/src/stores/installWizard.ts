import { defineStore } from 'pinia'

type DBType = 'sqlite' | 'mysql'

interface InstallWizardState {
  dbType: DBType
  sqlitePath: string
  mysql: {
    host: string
    port: number
    user: string
    pass: string
    dbName: string
    params: string
  }
  dbChecked: boolean
  dbCheckError: string

  siteName: string
  siteUrl: string

  adminUser: string
  adminPass: string
  adminPath: string
}

const STORAGE_KEY = 'install_wizard_v1'

/**
 * 把安装向导状态序列化为可安全落 sessionStorage 的字符串。
 *
 * 纯函数，便于回归测试（P1-22：MySQL 密码不得明文进 sessionStorage）。
 * 密码只在内存态（供 mysqlDSN 使用），持久化时显式剔除 `mysql.pass`；
 * 旧版本里已经带 `pass` 的残留会在下一次 persist 时被新格式覆盖剔除。
 *
 * H5：`safeLoad` 读成功后**立即**重写为无 secret 格式（升级旧残留），
 * 损坏 JSON 直接删除，不让旧明文继续躺在 sessionStorage 里。
 */
export function serializeInstallWizard(state: {
  dbType: DBType
  sqlitePath: string
  mysql: { host: string; port: number; user: string; pass: string; dbName: string; params: string }
  siteName: string
  siteUrl: string
  adminUser: string
  adminPath: string
}): string {
  return JSON.stringify({
    dbType: state.dbType,
    sqlitePath: state.sqlitePath,
    mysql: {
      host: state.mysql.host,
      port: state.mysql.port,
      user: state.mysql.user,
      dbName: state.mysql.dbName,
      params: state.mysql.params
    },
    siteName: state.siteName,
    siteUrl: state.siteUrl,
    adminUser: state.adminUser,
    adminPath: state.adminPath
  })
}

function safeLoad(): Partial<InstallWizardState> | null {
  try {
    const raw = sessionStorage.getItem(STORAGE_KEY)
    if (!raw) return null

    let parsed: any
    try {
      parsed = JSON.parse(raw)
    } catch {
      // 损坏 JSON：直接清除，不留坏数据（H5）。
      sessionStorage.removeItem(STORAGE_KEY)
      return null
    }

    if (typeof parsed !== 'object' || parsed === null) {
      sessionStorage.removeItem(STORAGE_KEY)
      return null
    }

    // 旧格式残留（已带 mysql.pass）：读成功后立即重写为无 secret 格式，
    // 而不是等下一次 persist 才覆盖——旧明文不能继续留在 sessionStorage。
    const mysql = parsed.mysql
    if (mysql && typeof mysql === 'object' && 'pass' in mysql) {
      const cleaned = { ...parsed, mysql: { ...mysql } }
      delete cleaned.mysql.pass
      try {
        sessionStorage.setItem(STORAGE_KEY, JSON.stringify(cleaned))
      } catch {
        // 重写失败不阻塞加载；persist() 仍会写无 secret 格式
      }
    }

    return parsed as Partial<InstallWizardState>
  } catch {
    return null
  }
}

export const useInstallWizardStore = defineStore('installWizard', {
  state: (): InstallWizardState => {
    const saved = safeLoad() || {}
    return {
      dbType: saved.dbType === 'sqlite' ? 'sqlite' : 'mysql',
      sqlitePath: saved.sqlitePath || './data/app.db',
      mysql: {
        host: saved.mysql?.host || '127.0.0.1',
        port: typeof saved.mysql?.port === 'number' ? saved.mysql.port : 3306,
        user: saved.mysql?.user || 'root',
        pass: saved.mysql?.pass || '',
        dbName: saved.mysql?.dbName || '',
        params: saved.mysql?.params || 'charset=utf8mb4&parseTime=True&loc=Local'
      },
      dbChecked: false,
      dbCheckError: '',

      siteName: saved.siteName || '',
      siteUrl: saved.siteUrl || '',

      adminUser: saved.adminUser || 'admin',
      adminPass: '',
      adminPath: saved.adminPath || ''
    }
  },

  getters: {
    mysqlDSN(state): string {
      const u = encodeURIComponent(state.mysql.user || '')
      const p = encodeURIComponent(state.mysql.pass || '')
      const host = state.mysql.host || '127.0.0.1'
      const port = Number(state.mysql.port || 3306)
      const dbName = state.mysql.dbName || ''
      const params = state.mysql.params ? `?${state.mysql.params}` : ''
      return `${u}:${p}@tcp(${host}:${port})/${dbName}${params}`
    }
  },

  actions: {
    touchDB() {
      this.dbChecked = false
      this.dbCheckError = ''
      this.persist()
    },
    markDBChecked(ok: boolean, errMsg = '') {
      this.dbChecked = ok
      this.dbCheckError = ok ? '' : errMsg
      this.persist()
    },
    persist() {
      try {
        sessionStorage.setItem(STORAGE_KEY, serializeInstallWizard(this))
      } catch {
        // ignore
      }
    },
    /**
     * 清除内存密码：安装完成 / 离开安装流程时调用，避免 DBA 密码在
     * 当前页面生命周期内继续驻留内存。同步重写 storage（无 secret 格式）。
     */
    clearSecrets() {
      this.mysql.pass = ''
      this.adminPass = ''
      this.persist()
    },
    reset() {
      sessionStorage.removeItem(STORAGE_KEY)
      this.$reset()
    }
  }
})
