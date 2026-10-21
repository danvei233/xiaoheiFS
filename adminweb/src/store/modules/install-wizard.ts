import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import type { InstallDBType } from '@/api/install'

interface InstallWizardMysqlConfig {
  host: string
  port: number
  user: string
  pass: string
  dbName: string
  params: string
}

interface InstallWizardSnapshot {
  dbType: InstallDBType
  sqlitePath: string
  mysql: InstallWizardMysqlConfig
  siteName: string
  siteUrl: string
  adminUser: string
  adminPath: string
}

const STORAGE_KEY = 'install_wizard_v1'

function createDefaultMysqlConfig(): InstallWizardMysqlConfig {
  return {
    host: '127.0.0.1',
    port: 3306,
    user: 'root',
    pass: '',
    dbName: '',
    params: 'charset=utf8mb4&parseTime=True&loc=Local'
  }
}

function loadSnapshot(): Partial<InstallWizardSnapshot> | null {
  try {
    const raw = sessionStorage.getItem(STORAGE_KEY)
    if (!raw) {
      return null
    }

    return JSON.parse(raw) as Partial<InstallWizardSnapshot>
  } catch {
    return null
  }
}

export const useInstallWizardStore = defineStore('installWizardStore', () => {
  const saved = loadSnapshot() || {}
  const savedMysql: Partial<InstallWizardMysqlConfig> = saved.mysql || {}

  const dbType = ref<InstallDBType>(saved.dbType === 'sqlite' ? 'sqlite' : 'mysql')
  const sqlitePath = ref(saved.sqlitePath || './data/app.db')
  const mysql = ref<InstallWizardMysqlConfig>({
    ...createDefaultMysqlConfig(),
    ...savedMysql,
    port:
      typeof savedMysql.port === 'number' && Number.isFinite(savedMysql.port)
        ? savedMysql.port
        : 3306
  })
  const dbChecked = ref(false)
  const dbCheckError = ref('')

  const siteName = ref(saved.siteName || '')
  const siteUrl = ref(saved.siteUrl || '')

  const adminUser = ref(saved.adminUser || 'admin')
  const adminPass = ref('')
  const adminPath = ref(saved.adminPath || '')

  const mysqlDSN = computed(() => {
    const user = encodeURIComponent(mysql.value.user || '')
    const pass = encodeURIComponent(mysql.value.pass || '')
    const host = mysql.value.host || '127.0.0.1'
    const port = Number(mysql.value.port || 3306)
    const dbName = mysql.value.dbName || ''
    const params = mysql.value.params ? `?${mysql.value.params}` : ''

    return `${user}:${pass}@tcp(${host}:${port})/${dbName}${params}`
  })

  function persist() {
    try {
      sessionStorage.setItem(
        STORAGE_KEY,
        JSON.stringify({
          dbType: dbType.value,
          sqlitePath: sqlitePath.value,
          mysql: mysql.value,
          siteName: siteName.value,
          siteUrl: siteUrl.value,
          adminUser: adminUser.value,
          adminPath: adminPath.value
        } satisfies InstallWizardSnapshot)
      )
    } catch {
      // ignore storage failures
    }
  }

  function touchDB() {
    dbChecked.value = false
    dbCheckError.value = ''
    persist()
  }

  function markDBChecked(ok: boolean, errorMessage: string = '') {
    dbChecked.value = ok
    dbCheckError.value = ok ? '' : errorMessage
    persist()
  }

  function reset() {
    dbType.value = 'mysql'
    sqlitePath.value = './data/app.db'
    mysql.value = createDefaultMysqlConfig()
    dbChecked.value = false
    dbCheckError.value = ''
    siteName.value = ''
    siteUrl.value = ''
    adminUser.value = 'admin'
    adminPass.value = ''
    adminPath.value = ''

    try {
      sessionStorage.removeItem(STORAGE_KEY)
    } catch {
      // ignore storage failures
    }
  }

  return {
    dbType,
    sqlitePath,
    mysql,
    dbChecked,
    dbCheckError,
    siteName,
    siteUrl,
    adminUser,
    adminPass,
    adminPath,
    mysqlDSN,
    persist,
    touchDB,
    markDBChecked,
    reset
  }
})
