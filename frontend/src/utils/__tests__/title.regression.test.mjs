/**
 * 回归测试：文档标题拼接（P3-51 / m065 R-5）。
 *
 * 背景：站点品牌是后台可配置的自由文本，可能含字面 ` - `（如"小黑云 - 高性能云"）。
 * 旧实现（App.vue MutationObserver）用 lastIndexOf(' - ') 定位分隔符后逐次替换品牌段，
 * 品牌含 ` - ` 时每次替换都产生新字符串 → 无限循环卡死页签。
 *
 * 修复：composeDocumentTitle 每次整段重写，幂等；品牌原样作为后缀，不迭代增长。
 *
 * 运行：`node --test src/utils/__tests__/title.regression.test.mjs`
 */
import assert from 'node:assert/strict'

// utils/router.ts 顶层会 import @/stores/site（引 element-plus 链），
// 用 esbuild 打包并把依赖链阻断到只取 composeDocumentTitle。
import { bundleEntry } from '../../../tests/_bundle.mjs'

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
// stores/site → services/user → services/http → navigate → router → views(.vue)，
// 打包测试时把整条链在源头上断掉。
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

const { composeDocumentTitle } = await bundleEntry(
  `export { composeDocumentTitle } from '@/utils/router'`,
  import.meta.url,
  { plugins: [localesStub, siteStub] }
)

// ── 常规 ──────────────────────────────────────────────────────────────
assert.equal(composeDocumentTitle('我的订单', '小黑云'), '我的订单 - 小黑云')
assert.equal(composeDocumentTitle('', '小黑云'), '小黑云', '无页名时只显示品牌')

// ── 品牌含字面 ` - `：原样保留为后缀，不得迭代增长（R-5 核心）───────────
const brand = '小黑云 - 高性能云'
const first = composeDocumentTitle('我的订单', brand)
assert.equal(first, '我的订单 - 小黑云 - 高性能云')
// 幂等：以"已拼好"的标题再拼一次品牌不会继续增长——因为调用方总是传原始页名。
const again = composeDocumentTitle('我的订单', brand)
assert.equal(again, first, '同一输入重复调用结果不变（幂等）')

// 页名本身含 ` - ` 也正常。
assert.equal(composeDocumentTitle('开通 - 成功', '小黑云'), '开通 - 成功 - 小黑云')

console.log('✅ title.regression.test.mjs —— 标题拼接幂等断言通过')
