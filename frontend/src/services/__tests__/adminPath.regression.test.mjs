/**
 * 回归测试：admin_path 不得把 localStorage 当验证权威（H1 / P2-68）。
 *
 * 背景：旧实现把"localStorage 路径 + 时间戳"当作已验证，攻击者可同时伪造
 *   两项，24h TTL 无安全意义。本次改为：
 *   - localStorage 只作路径提示（admin_path_cache），不携带"已验证"语义；
 *   - 每次页面启动/首次进入管理域必须向后端 `POST /check-admin-path` 复核一次；
 *   - fast-path 只认**本页面内成功响应建立的内存态**（路径匹配 + 代数一致）；
 *   - `admin_path_epoch` 单调递增：改路径/清缓存时自增，旧 tab 内存代数失配后
 *     回落后端复核，路径被改则 fail-closed（404）。
 *
 * 运行：`node --test src/services/__tests__/adminPath.regression.test.mjs`
 */
import { bundleEntry } from '../../../tests/_bundle.mjs'
import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const here = path.dirname(fileURLToPath(import.meta.url))
const httpStubPath = path.resolve(here, '../../../tests/stubs/http-stub.mjs')

/** 可编程 http 桩：控制 is_admin 并统计调用次数。 */
const makeProgrammableStub = ({ fail = false } = {}) => ({
  name: `programmable-http${fail ? '-fail' : ''}`,
  setup(build) {
    build.onResolve({ filter: /^\.\/http$/ }, (args) => {
      if (args.importer.includes('/services/')) {
        return { path: httpStubPath }
      }
      return null
    })
    build.onLoad({ filter: /http-stub/ }, () => ({
      // 状态放到 globalThis，多个 bundle 共享同一套桩状态，避免后加载的
      // bundle 覆盖 setter 后让先前的断言拿到错乱数据。
      contents: `
        globalThis.__postCountValue = globalThis.__postCountValue ?? 0
        const state = () => ({
          adminPath: globalThis.__fakeAdminPath ?? '',
          fail: globalThis.__fakeFail === true
        })
        export const http = {
          post: async (url, payload) => {
            globalThis.__postCountValue += 1
            const s = state()
            if (s.fail) throw new Error('network down')
            return { data: { is_admin: s.adminPath !== '' && payload?.path === s.adminPath } }
          },
          get: async () => ({ data: {} }),
          request: async () => ({ data: {} })
        }
        globalThis.__postCount = () => globalThis.__postCountValue ?? 0
        globalThis.__resetPostCount = () => { globalThis.__postCountValue = 0 }
        globalThis.__setAdminPath = (p) => { globalThis.__fakeAdminPath = p }
        globalThis.__setFail = (v) => { globalThis.__fakeFail = v }
        if (${fail ? 'true' : 'false'}) globalThis.__fakeFail = true
      `,
      loader: 'js'
    }))
  }
})

const mainMod = await bundleEntry(
  `export { checkAdminPath, clearAdminPathCache, setAdminPathHint, getCachedAdminPath } from '@/services/adminPath'`,
  import.meta.url,
  { plugins: [makeProgrammableStub()] }
)
const { checkAdminPath, clearAdminPathCache, setAdminPathHint, getCachedAdminPath } = mainMod

const store = new Map()
globalThis.localStorage = {
  getItem: (k) => store.get(k) ?? null,
  setItem: (k, v) => store.set(k, String(v)),
  removeItem: (k) => store.delete(k)
}
const isAdmin = async (p) => (await checkAdminPath(p)).isAdmin
const epoch = () => Number(store.get('admin_path_epoch') || '0')

// ── 1. localStorage 只作提示：伪造路径+时间戳不产生任何信任 ───────────────
store.clear()
globalThis.__resetPostCount()
store.set('admin_path_cache', 'myadmin')
store.set('admin_path_validated_at', String(Date.now())) // 旧键无任何安全意义
globalThis.__setAdminPath('') // 后端不认任何路径
assert.equal(await isAdmin('myadmin'), false, '仅 localStorage 伪造值不通过（需后端复核）')
assert.equal(globalThis.__postCount(), 1, '首次进入必须打一次后端')

// ── 2. 后端确认建立内存态 → fast-path（同页面内不再发请求）────────────────
store.clear()
globalThis.__resetPostCount()
globalThis.__setAdminPath('admin')
assert.equal(await isAdmin('admin'), true, '后端确认是管理路径')
assert.equal(globalThis.__postCount(), 1, '后端确认一次')
assert.equal(await isAdmin('admin'), true, '内存态 fast-path 仍为管理域')
assert.equal(globalThis.__postCount(), 1, 'fast-path 不重复打后端')
assert.equal(await isAdmin('other'), false, '不同路径不享受 fast-path')
assert.equal(globalThis.__postCount(), 2, '不同路径触发第二次后端复核')

// ── 3. 切换 admin_path 后旧 tab 内存态作废，fail-closed ──────────────────
store.clear()
globalThis.__resetPostCount()
globalThis.__setAdminPath('admin')
assert.equal(await isAdmin('admin'), true, '旧 tab 先验证过 admin')
const tabEpochBefore = epoch()

// 另一个 tab 改了 admin_path → clearAdminPathCache 使代数自增。
clearAdminPathCache()
assert.equal(epoch(), tabEpochBefore + 1, 'clear 自增路径代数（跨 tab 失效信号）')
assert.equal(store.get('admin_path_cache'), undefined, 'clear 清掉路径提示')

// 后端现在不认旧路径 → 旧 tab 本次导航必须 fail-closed。
globalThis.__setAdminPath('newadmin')
assert.equal(await isAdmin('admin'), false, '切换路径后旧 tab 对旧路径 fail-closed')

// ── 3b. 另一种跨 tab：他人改代数但本 tab 内存态仍在 → 复核并 fail ────────
store.clear()
globalThis.__resetPostCount()
globalThis.__setAdminPath('admin')
assert.equal(await isAdmin('admin'), true, '本 tab 内存态已验证 admin')
assert.equal(globalThis.__postCount(), 1)
store.set('admin_path_epoch', String(epoch() + 1)) // 别的 tab 把代数推高
globalThis.__setAdminPath('newadmin') // 后端路径已切换
assert.equal(await isAdmin('admin'), false, '代数失配时不得 fast-path（须复核）')
assert.equal(globalThis.__postCount(), 2, '代数失配触发后端复核')

// ── 4. setAdminPathHint 只写提示、不影响验证 ─────────────────────────────
store.clear()
setAdminPathHint('brand-new-admin')
assert.equal(store.get('admin_path_cache'), 'brand-new-admin')
assert.equal(store.has('admin_path_validated_at'), false, '提示写入不产生旧时间戳键')
globalThis.__setAdminPath('brand-new-admin')
assert.equal(await isAdmin('brand-new-admin'), true, '提示路径经后端确认后可用')
assert.equal(getCachedAdminPath(), 'brand-new-admin')

// ── 5. 后端网络错误 → fail-closed（不建立验证态，含伪造 hint）─────────────
const failingMod = await bundleEntry(
  `export { checkAdminPath } from '@/services/adminPath'`,
  import.meta.url,
  { plugins: [makeProgrammableStub({ fail: true })] }
)
store.clear()
globalThis.__setAdminPath('anything')
globalThis.__setFail(true)
assert.equal(
  (await failingMod.checkAdminPath('anything')).isAdmin,
  false,
  '后端不可达时 fail-closed'
)

// ── 6. 后端明确否认已验证路径 → 内存 fast-path 立即作废 ──────────────────
// 背景：后端改 admin_path 而没走 clearAdminPathCache（换实例/直接改库），
// 代数未变时旧实现会继续 fast-path 放行旧路径。本次在"后端否认"时同时作废
// 内存态，即使之后代数被重置回验证时的值也不会再 fast-path。
store.clear()
globalThis.__resetPostCount()
globalThis.__setFail(false)
globalThis.__setAdminPath('admin')
assert.equal(await isAdmin('admin'), true, '先验证 admin（建立内存态）')
assert.equal(globalThis.__postCount(), 1)
// 模拟其他 tab 改路径：代数自增 + 后端已切到 newadmin。
clearAdminPathCache()
globalThis.__setAdminPath('newadmin')
assert.equal(await isAdmin('admin'), false, '代数失配 + 后端否认 → fail-closed')
assert.equal(globalThis.__postCount(), 2, '否认时必须走后端复核')
// 把代数重置回验证时的值（storage 异常恢复）——内存态若还活着就会 fast-path。
store.set('admin_path_epoch', '0')
assert.equal(await isAdmin('admin'), false, '内存态已被作废 → 即使代数复位也不 fast-path')
assert.equal(globalThis.__postCount(), 3, '作废后每次导航都重新复核')

// ── 7. 路由守卫 catch 分支不得回退 localStorage（源码漂移防护）────────────
const guardPath = path.resolve(here, '../../../src/router/guards/beforeEach.ts')
const guardSource = fs.readFileSync(guardPath, 'utf8')
assert.ok(
  guardSource.includes("'[RouteGuard] admin path verification threw:'"),
  '守卫 catch 分支应记错误'
)
assert.ok(
  !/return segment === getCachedAdminPath\(\)/.test(guardSource),
  '守卫 catch 分支不得回退 getCachedAdminPath（localStorage 提示不可当验证权威）'
)

console.log('✅ adminPath.regression.test.mjs —— localStorage 不作验证权威断言通过')
