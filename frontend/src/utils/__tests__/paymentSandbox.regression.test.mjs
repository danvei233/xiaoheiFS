/**
 * 回归测试：支付表单 iframe sandbox 合同（H3 / P2-74）。
 *
 * 背景：旧 sandbox 含 `allow-top-navigation`，被攻陷/恶意的支付网关脚本可以
 *   无用户交互地把顶层窗口导航到钓鱼页；`allow-same-origin` 让脚本直接读
 *   到本应用 token。
 *
 * 目标契约：
 *  - 必需能力：allow-forms / allow-scripts（表单自动提交 + 网关 JS）；
 *  - 禁止令牌：allow-top-navigation（无条件顶层导航）、allow-same-origin；
 *  - 顶层导航只允许 allow-top-navigation-by-user-activation（用户激活后）。
 *
 * 运行：`node --test src/utils/__tests__/paymentSandbox.regression.test.mjs`
 */
import { bundleEntry } from '../../../tests/_bundle.mjs'
import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const { PAYMENT_FORM_SANDBOX, assertPaymentSandboxContract } = await bundleEntry(
  `export { PAYMENT_FORM_SANDBOX, assertPaymentSandboxContract } from '@/utils/paymentSandbox'`,
  import.meta.url
)

// ── 常量本身满足合同 ───────────────────────────────────────────────────
assert.ok(assertPaymentSandboxContract(PAYMENT_FORM_SANDBOX), '发布用的 sandbox 常量必须满足合同')

const tokens = PAYMENT_FORM_SANDBOX.split(/\s+/).filter(Boolean)
assert.ok(tokens.includes('allow-forms'), '必须允许表单自动提交')
assert.ok(tokens.includes('allow-scripts'), '必须允许支付网关脚本')
assert.ok(!tokens.includes('allow-top-navigation'), '不得包含无条件 allow-top-navigation')
assert.ok(
  tokens.includes('allow-top-navigation-by-user-activation'),
  '顶层导航只允许用户激活后触发'
)
assert.ok(!tokens.includes('allow-same-origin'), '不得包含 allow-same-origin')

// ── 合同校验器拒绝危险配置 ─────────────────────────────────────────────
assert.equal(
  assertPaymentSandboxContract('allow-forms allow-scripts allow-top-navigation allow-popups'),
  false,
  '含无条件 allow-top-navigation 违反合同'
)
assert.equal(
  assertPaymentSandboxContract('allow-forms allow-scripts allow-same-origin'),
  false,
  '含 allow-same-origin 违反合同'
)
assert.equal(
  assertPaymentSandboxContract(
    'allow-scripts allow-popups allow-top-navigation-by-user-activation'
  ),
  false,
  '缺 allow-forms 违反合同'
)

// ── 源码漂移防护：订单详情页必须引用常量，而不是手写内联 sandbox ───────
const here = path.dirname(fileURLToPath(import.meta.url))
const detailPath = path.resolve(here, '../../../src/views/console/orders/detail.vue')
const source = fs.readFileSync(detailPath, 'utf8')
assert.ok(
  source.includes('PAYMENT_FORM_SANDBOX'),
  'orders/detail.vue 必须引用 PAYMENT_FORM_SANDBOX 常量'
)
assert.ok(
  !/sandbox="[^"]*allow-top-navigation(?!-by-user-activation)/.test(source),
  'orders/detail.vue 不得内联无条件 allow-top-navigation'
)

console.log('✅ paymentSandbox.regression.test.mjs —— sandbox 合同断言通过')
