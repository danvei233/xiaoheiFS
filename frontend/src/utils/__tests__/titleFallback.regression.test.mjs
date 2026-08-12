/**
 * 回归测试：无 meta.title 时立即回退品牌 + 单一可清理 timer（H7）。
 *
 * 背景：旧实现 `setPageTitle` 在无 title 时只清 `currentPageTitle` 不写
 *   `document.title`，上一个页面的标题一直残留；有 title 时每次导航新建一个
 *   setTimeout(150)，快速导航 A→B 时旧 timer 晚到会把 A 的标题覆盖到 B。
 *
 * 目标契约：
 *  - 无 meta.title → 立即写"品牌"，不等 150ms；
 *  - 快速导航 A→B → 只保留 B 的标题（旧 timer 被清掉）；
 *  - A 之后立即导航到无 title 页 → 立即回退品牌，且不再被 A 的晚到 timer 覆盖。
 *
 * 运行：`node --test src/utils/__tests__/titleFallback.regression.test.mjs`
 */
import { bundleEntry } from '../../../tests/_bundle.mjs'
import assert from 'node:assert/strict'

let title = ''
globalThis.document = {
  get title() {
    return title
  },
  set title(v) {
    title = v
  }
}

const localesStub = {
  name: 'stub-locales',
  setup(build) {
    build.onResolve({ filter: /^@\/locales$/ }, () => ({ path: 'locales-stub', namespace: 'stub' }))
    build.onLoad({ filter: /.*/, namespace: 'stub' }, () => ({
      contents: `export const $t = (key) => key; export default { global: { te: () => true } }`,
      loader: 'js'
    }))
  }
}
const siteStub = {
  name: 'stub-site',
  setup(build) {
    build.onResolve({ filter: /^@\/stores\/site$/ }, () => ({
      path: 'site-stub',
      namespace: 'site'
    }))
    build.onLoad({ filter: /.*/, namespace: 'site' }, () => ({
      contents: `export const useSiteStore = () => ({ siteName: '小黑云' })`,
      loader: 'js'
    }))
  }
}
const configStub = {
  name: 'stub-config',
  setup(build) {
    build.onResolve({ filter: /^@\/config$/ }, () => ({ path: 'config-stub', namespace: 'config' }))
    build.onLoad({ filter: /.*/, namespace: 'config' }, () => ({
      contents: `export default { systemInfo: { name: '构建期品牌' } }`,
      loader: 'js'
    }))
  }
}
const nprogressStub = {
  name: 'stub-nprogress',
  setup(build) {
    build.onResolve({ filter: /^nprogress$/ }, () => ({
      path: 'nprogress-stub',
      namespace: 'nprogress'
    }))
    build.onLoad({ filter: /.*/, namespace: 'nprogress' }, () => ({
      contents: `export default { configure: () => {} }`,
      loader: 'js'
    }))
  }
}

const { setPageTitle } = await bundleEntry(
  `export { setPageTitle } from '@/utils/router'`,
  import.meta.url,
  { plugins: [localesStub, siteStub, configStub, nprogressStub] }
)

const route = (meta) => ({ meta: meta || {}, matched: [] })

// ── 1. 无 meta.title → 立即回退品牌 ────────────────────────────────────
title = ''
setPageTitle(route({}))
assert.equal(title, '小黑云', '无页名时立即写品牌，不等 150ms')

// ── 2. 有 title → 150ms 后写"页名 - 品牌" ─────────────────────────────
title = ''
setPageTitle(route({ title: '我的订单' }))
assert.equal(title, '', '有页名时延后一拍（等 DOM 更新）')
await new Promise((resolve) => setTimeout(resolve, 200))
assert.equal(title, '我的订单 - 小黑云', '写入页名-品牌')

// ── 3. 快速导航 A→B：旧 timer 被清理，只保留 B ─────────────────────────
title = ''
setPageTitle(route({ title: 'A页' }))
setPageTitle(route({ title: 'B页' }))
await new Promise((resolve) => setTimeout(resolve, 200))
assert.equal(title, 'B页 - 小黑云', '快速导航后只保留后一个页面的标题')

// ── 4. A 之后立即导航到无 title 页：立即回退品牌且不被晚到 timer 覆盖 ──
title = ''
setPageTitle(route({ title: 'A页' }))
setPageTitle(route({})) // 无 title → 立即回退
assert.equal(title, '小黑云', '无 title 页立即回退品牌')
await new Promise((resolve) => setTimeout(resolve, 200))
assert.equal(title, '小黑云', 'A 的晚到 timer 不得把旧标题覆盖回来')

console.log('✅ titleFallback.regression.test.mjs —— 标题回退与单一 timer 断言通过')
