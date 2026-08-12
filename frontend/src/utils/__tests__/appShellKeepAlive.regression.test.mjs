/**
 * 回归测试：App.vue 壳层不再用 route.fullPath 作为 ErrorBoundary 的 key
 * （web_bundle_mobile F1）。
 *
 * 背景：旧实现 `<ErrorBoundary :key="route.fullPath">` —— 任一导航（console 下
 *   /console/orders→/console/vps、admin 下任意 tab、同页 query 变化）都会改变
 *   fullPath → ErrorBoundary 连同整棵子树（布局、KeepAlive、当前页面）卸载
 *   重建，meta.keepAlive 缓存全部失效、每个列表页重新拉数据、整壳过渡动画。
 *
 * 修复：外层 key 只跨布局域变化（route.matched[0].path）；错误兜底状态由
 *   ErrorBoundary 内部 watch route.fullPath 复位，保留 P2-79 语义。
 *
 * 运行：`node --test src/utils/__tests__/appShellKeepAlive.regression.test.mjs`
 */
import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const here = path.dirname(fileURLToPath(import.meta.url))
const srcRoot = path.resolve(here, '../../../src')

const appSource = fs.readFileSync(path.resolve(srcRoot, 'App.vue'), 'utf8')
const errorBoundary = fs.readFileSync(
  path.resolve(srcRoot, 'components/business/error-boundary/index.vue'),
  'utf8'
)

// ErrorBoundary 的 key 不得再绑定 route.fullPath（每次路由切换整壳重挂载）。
assert.ok(
  !appSource.includes('ErrorBoundary :key="route.fullPath"'),
  'ErrorBoundary key 不得绑定 route.fullPath'
)
assert.ok(
  appSource.includes('ErrorBoundary :key="route.matched[0]?.path || route.path"'),
  'ErrorBoundary key 只跨布局域变化'
)
// 错误兜底状态在路由变化时复位（P2-79 语义不丢）。
assert.ok(
  errorBoundary.includes('() => route.fullPath'),
  'ErrorBoundary 内部必须 watch route.fullPath 复位错误状态'
)
assert.ok(errorBoundary.includes('hasError.value = false'), '路由变化必须清错误状态')

console.log('✅ appShellKeepAlive.regression.test.mjs —— 壳层 key / KeepAlive 语义断言通过')
