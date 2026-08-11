/**
 * 回归测试：prefixRoutePaths 必须把管理后台菜单里的相对子路径拼成绝对路径。
 *
 * 背景（第二遍审查 m052 / F1）：
 *   registerAdminRoutes 把 prefixRoutePaths(asyncRoutes, adminPath) 的产物写入
 *   menuStore.setMenuList，menuStore 再靠 getFirstMenuPath 推导管理后台首页路径。
 *   旧实现只给顶层绝对路径加 admin 前缀，目录下的相对子路径原样保留
 *   （如 'console'、'review'），导致：
 *     - getFirstMenuPath 得到 '/console'、'/review' 这类错误首页路径；
 *     - 管理员登录后访问 /<admin_path>/ 或点 logo 被重定向到用户控制台域。
 *
 * 本文件没有使用 vitest/jest（前端暂无测试运行器，见 AGENTS.md），直接以
 * node 运行：`node src/router/__tests__/prefixRoutePaths.test.mjs`。
 * 通过 esbuild 把真实的 realm.ts 与 asyncRoutes 打包进内存执行，避免浏览器依赖。
 *
 * 期望：修复后子路径形如 '/<admin_path>/dashboard/console'。
 */

import { build } from 'esbuild'
import assert from 'node:assert/strict'
import path from 'node:path'
import { writeFileSync, mkdtempSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { pathToFileURL } from 'node:url'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const srcDir = path.resolve(__dirname, '../../')
const routerDir = path.join(srcDir, 'router')

/**
 * 打包真实的 realm.ts + asyncRoutes，替换浏览器端 adminPath 服务为内存 stub。
 * 返回 { prefixRoutePaths, asyncRoutes }。
 */
async function loadRouterModule() {
  const result = await build({
    stdin: {
      contents: `
        import { prefixRoutePaths } from './realm'
        import { asyncRoutes } from './routes/asyncRoutes'
        export { prefixRoutePaths, asyncRoutes }
      `,
      resolveDir: routerDir,
      sourcefile: 'bundle.ts'
    },
    bundle: true,
    platform: 'node',
    format: 'esm',
    write: false,
    plugins: [
      {
        name: 'stub-admin-path',
        setup(build) {
          build.onResolve({ filter: /^@\/services\/adminPath$/ }, () => ({
            path: 'adminpath-stub.ts',
            namespace: 'stub'
          }))
          build.onLoad({ filter: /.*/, namespace: 'stub' }, () => ({
            contents: `export const getCachedAdminPath = () => 'admin'`,
            loader: 'ts'
          }))
        }
      }
    ]
  })

  const dir = mkdtempSync(path.join(tmpdir(), 'm052-'))
  const file = path.join(dir, 'bundle.mjs')
  writeFileSync(file, result.outputFiles[0].text)
  return import(pathToFileURL(file).href)
}

/**
 * 复刻 utils/navigation/route.ts 的 getFirstMenuPath —— 该文件属于其他模块，
 * 这里只复刻逻辑来断言"由菜单树推导出的首页路径"。修复的契约是：推导结果
 * 必须是 '/<admin_path>/dashboard/console' 这类完整路径。
 */
function getFirstMenuPath(menuList) {
  if (!Array.isArray(menuList) || menuList.length === 0) return ''
  const isNavigable = (item) => {
    if (!item.path || !item.path.trim()) return false
    if (!item.meta?.isHide) return true
    return item.meta?.isFullPage === true
  }
  const normalize = (p) => (p.startsWith('/') ? p : `/${p}`)
  for (const item of menuList) {
    if (!isNavigable(item)) continue
    if (item.children?.length) {
      const child = getFirstMenuPath(item.children)
      if (child) return child
    }
    return normalize(item.path)
  }
  return ''
}

/**
 * 复刻 guards/beforeEach.ts 的 filterByPermission —— 同样的原因，只复刻逻辑
 * 用于构造"仅含部分权限"时的菜单裁剪结果。
 */
function filterByPermission(routes, permissions) {
  if (permissions.includes('*')) return routes
  const allowed = (route) => {
    const marks = (route.meta?.authList ?? []).map((a) => a.authMark)
    if (marks.length === 0) return true
    return marks.some((mark) => permissions.includes(mark))
  }
  const walk = (list) =>
    list.reduce((acc, route) => {
      const children = route.children?.length ? walk(route.children) : undefined
      if (route.children?.length && (!children || children.length === 0)) return acc
      if (!allowed(route)) return acc
      acc.push(children ? { ...route, children } : route)
      return acc
    }, [])
  return walk(routes)
}

const { prefixRoutePaths, asyncRoutes } = await loadRouterModule()

// ── 基础断言：相对子路径必须被拼成完整路径 ──────────────────────────────
const menus = prefixRoutePaths(asyncRoutes, 'mypanel')

const dashboard = menus.find((r) => r.path === '/mypanel/dashboard')
assert.ok(dashboard, '顶层 /dashboard 应被加前缀为 /mypanel/dashboard')
assert.equal(
  dashboard.children[0].path,
  '/mypanel/dashboard/console',
  'dashboard 的第一个子路由应是绝对路径 /mypanel/dashboard/console（旧实现返回裸 "console"）'
)
assert.equal(
  dashboard.children[1].path,
  '/mypanel/dashboard/revenue-analytics',
  '第二个子路由同理应为绝对路径'
)

const order = menus.find((r) => r.path === '/mypanel/order')
assert.equal(order.children[0].path, '/mypanel/order/review')

const probes = menus.find((r) => r.path === '/mypanel/probes')
assert.equal(probes.children[0].path, '/mypanel/probes/list')
assert.equal(probes.children[1].path, '/mypanel/probes/:id', '动态参数子路由也应拼成绝对路径')
assert.equal(
  probes.children[1].meta.activePath,
  '/mypanel/probes/list',
  'activePath 应随 admin 前缀改写'
)

// 叶子路由（第一级就带页面）维持原有绝对路径语义
assert.ok(menus.some((r) => r.path === '/mypanel/vps'))

// ── 复现第二遍报告中的 homePath 推导结果 ────────────────────────────────
const homeFor = (permissions) => getFirstMenuPath(filterByPermission(menus, permissions))

assert.equal(
  homeFor(['*']),
  '/mypanel/dashboard/console',
  '超管（*）的首页路径应为 /<admin_path>/dashboard/console，旧实现错误返回 /console'
)
assert.equal(
  homeFor(['order.list']),
  '/mypanel/order/review',
  '仅 order.list 权限的首页应为 /<admin_path>/order/review，旧实现错误返回 /review'
)
assert.equal(
  homeFor(['vps.list']),
  '/mypanel/vps',
  'vps.list 是叶子路由，首页应保持 /<admin_path>/vps'
)
// 无任何权限时，/system 目录仍会保留 —— 它的隐藏子页 profile（个人资料）
// authList 为空，按 filterByPermission 规则始终可见（self-service 页，所有管理员
// 都应能访问）。首页路径因此落在 /system 目录。该行为与 F1 无关，属既有契约。
assert.equal(
  homeFor([]),
  '/mypanel/system',
  '无任何权限时仅剩 /system 目录（其 profile 子页 authList 为空，始终可见）'
)

// ── admin 根路径重定向的完整断言（beforeEach.ts:302 的依赖） ────────────
// 管理员访问 /mypanel 时应跳到 homePath，而不是离开 admin 域。
assert.notEqual(homeFor(['*']), '/console', '首页路径绝不能落到用户控制台 /console')
assert.equal(homeFor(['*']).startsWith('/mypanel/'), true, '首页路径必须留在 admin 域内')

console.log('✅ prefixRoutePaths.test.mjs —— F1 全部断言通过')
