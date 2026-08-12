/**
 * 回归测试：安装向导 MySQL 密码不得明文落 sessionStorage（P1-22 / m053 H3）。
 *
 * 背景：installWizard store 的 persist() 原来把 `mysql: this.mysql` 整体写入
 *   sessionStorage，而 mysql.pass 是明文密码。浏览器同源 sessionStorage 可被
 *   XSS 读取；安装完成前浏览器崩溃/被探查都会把 DBA 密码留在磁盘上。
 *
 * 验证目标：
 *  1. serializeInstallWizard 产物不含 mysql.pass 字段，也不含密码值；
 *  2. 其余字段（dbType/sqlitePath/host/port/user/dbName/params/site/admin*）原样保留；
 *  3. 旧版本残留（已带 pass）经 serialize 后会被剔除（覆盖式修复）。
 *
 * 运行：`node --test src/stores/__tests__/installWizard.regression.test.mjs`
 */
import { bundleEntry } from '../../../tests/_bundle.mjs'
import assert from 'node:assert/strict'

const mod = await bundleEntry(
  `export { serializeInstallWizard, useInstallWizardStore } from '@/stores/installWizard'
   export { createPinia, setActivePinia } from 'pinia'`,
  import.meta.url
)
const { serializeInstallWizard } = mod

const state = {
  dbType: 'mysql',
  sqlitePath: './data/app.db',
  mysql: {
    host: '127.0.0.1',
    port: 3306,
    user: 'root',
    pass: 's3cret-db-pass',
    dbName: 'xiaohei',
    params: 'charset=utf8mb4&parseTime=True&loc=Local'
  },
  siteName: '小黑云',
  siteUrl: 'https://cloud.example.com',
  adminUser: 'admin',
  adminPath: 'myadmin'
}

const raw = serializeInstallWizard(state)
const parsed = JSON.parse(raw)

// ── 1. 密码不落盘 ──────────────────────────────────────────────────────
assert.equal('pass' in parsed.mysql, false, '序列化产物 mysql 不得含 pass 字段')
assert.ok(!raw.includes('s3cret-db-pass'), '序列化字符串不得出现密码明文')
assert.ok(!raw.includes('pass'), '序列化字符串不得出现任何 pass 键')

// ── 2. 其余字段原样保留 ────────────────────────────────────────────────
assert.equal(parsed.dbType, 'mysql')
assert.equal(parsed.sqlitePath, './data/app.db')
assert.equal(parsed.mysql.host, '127.0.0.1')
assert.equal(parsed.mysql.port, 3306)
assert.equal(parsed.mysql.user, 'root')
assert.equal(parsed.mysql.dbName, 'xiaohei')
assert.equal(parsed.mysql.params, 'charset=utf8mb4&parseTime=True&loc=Local')
assert.equal(parsed.siteName, '小黑云')
assert.equal(parsed.siteUrl, 'https://cloud.example.com')
assert.equal(parsed.adminUser, 'admin')
assert.equal(parsed.adminPath, 'myadmin')

// ── 3. 旧版本残留（带 pass）经新 serialize 后被剔除 ────────────────────
const legacy = {
  ...state,
  mysql: { ...state.mysql, pass: 'legacy-leak' }
}
const legacyRaw = serializeInstallWizard(legacy)
assert.ok(!legacyRaw.includes('legacy-leak'), '旧残留密码应被覆盖剔除')
assert.equal(JSON.parse(legacyRaw).mysql.pass, undefined, '重新序列化后仍无 pass 字段')

// ── 4. sqlite 模式不受影响 ─────────────────────────────────────────────
const sqliteRaw = serializeInstallWizard({ ...state, dbType: 'sqlite' })
assert.equal(JSON.parse(sqliteRaw).dbType, 'sqlite')

// ── 5. safeLoad 立即升级旧格式（H5）──────────────────────────────────────
// 旧格式 sessionStorage 里带着明文 pass；实例化 store 后 storage 必须已被
// 重写为无 secret 格式，而不是等到下一次 persist。
const storageStore = new Map()
globalThis.sessionStorage = {
  getItem: (k) => storageStore.get(k) ?? null,
  setItem: (k, v) => storageStore.set(k, String(v)),
  removeItem: (k) => storageStore.delete(k)
}
storageStore.set(
  'install_wizard_v1',
  JSON.stringify({
    dbType: 'mysql',
    sqlitePath: './data/app.db',
    mysql: { host: '127.0.0.1', port: 3306, user: 'root', pass: 'legacy-leak', dbName: 'xiaohei' },
    siteName: '旧站点',
    siteUrl: '',
    adminUser: 'admin',
    adminPath: ''
  })
)

mod.setActivePinia(mod.createPinia())
const wiz = mod.useInstallWizardStore()
const storedAfterLoad = storageStore.get('install_wizard_v1')
assert.ok(!storedAfterLoad.includes('legacy-leak'), 'safeLoad 读成功后立即重写无 secret 格式')
assert.equal(JSON.parse(storedAfterLoad).mysql.pass, undefined, 'storage 里不得再有 pass 字段')
assert.equal(wiz.mysql.pass, 'legacy-leak', '内存态仍保留密码供 mysqlDSN 使用')
assert.equal(wiz.mysql.host, '127.0.0.1', '其余字段原样保留')

// ── 6. 损坏 JSON → 直接清除，不留坏数据（H5）────────────────────────────
storageStore.clear()
storageStore.set('install_wizard_v1', '{broken json!!!')
mod.setActivePinia(mod.createPinia())
const wiz2 = mod.useInstallWizardStore()
assert.equal(storageStore.has('install_wizard_v1'), false, '损坏 JSON 被删除')
assert.equal(wiz2.mysql.host, '127.0.0.1', '损坏时回落默认值')

// ── 7. clearSecrets 清除内存密码并同步 storage（H5）──────────────────────
storageStore.clear()
storageStore.set(
  'install_wizard_v1',
  JSON.stringify({
    dbType: 'mysql',
    sqlitePath: './data/app.db',
    mysql: { host: '127.0.0.1', port: 3306, user: 'root', dbName: 'xiaohei' },
    siteName: '',
    siteUrl: '',
    adminUser: 'admin',
    adminPath: ''
  })
)
mod.setActivePinia(mod.createPinia())
const wiz3 = mod.useInstallWizardStore()
wiz3.mysql.pass = 'in-memory-dba-pass'
wiz3.adminPass = 'in-memory-admin-pass'
wiz3.clearSecrets()
assert.equal(wiz3.mysql.pass, '', 'clearSecrets 清空内存 DBA 密码')
assert.equal(wiz3.adminPass, '', 'clearSecrets 清空内存管理员密码')
assert.ok(
  !storageStore.get('install_wizard_v1').includes('in-memory'),
  'storage 同步无 secret 格式'
)

console.log('✅ installWizard.regression.test.mjs —— 密码不落盘/旧格式升级/损坏清除断言通过')
