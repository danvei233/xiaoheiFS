/**
 * 回归测试：订单价格预览与后端逐项整数舍入一致（cross-module residual）。
 *
 * 背景：后端下单按分计算，`round(baseMonthly*mult)` 与每个 addon
 *   `round(monthly*mult)` **各自取整后求和**；前端购买页此前用浮点
 *   `(base+addon)*mult*qty`，单价/倍率含小数时多 addon 差 1 分。
 *
 * golden vectors 与后端 `priceBreakdownForPackage` 的语义一一对应：
 *   1. 0.05 元/月的 base + 两个 0.05 元/月 addon，倍率 0.5：
 *      base=round(5*0.5)=3，core=round(5*0.5)=3，mem=3 → total=9 分
 *      （浮点聚合 (0.05*3)*0.5=0.075 → 7.5 分，会错 1~2 分）。
 *   2. 9.99 元 base + 2×0.50 元 core + 1×0.25 元 disk，倍率 3：
 *      base=round(999*3)=2997，core=round(100*3)=300，disk=round(25*3)=75
 *      → total=3372 分 = 33.72 元。
 *
 * 运行：`node --test src/utils/__tests__/pricing.regression.test.mjs`
 */
import assert from 'node:assert/strict'

const { priceBreakdownToCents } = await import('../../utils/pricing.ts')

// ── golden vector 1：小数单价 + 小数倍率，多 addon 不得差 1 分 ───────────
const v1 = priceBreakdownToCents({
  baseMonthly: 0.05,
  addons: [
    { value: 1, unitPrice: 0.05 },
    { value: 1, unitPrice: 0.05 }
  ],
  multiplier: 0.5
})
assert.equal(v1.baseAmount, 3, 'base=round(5*0.5)=3 分')
assert.deepEqual(v1.addonAmounts, [3, 3], '每个 addon 各自 round：round(5*0.5)=3')
assert.equal(v1.totalCents, 9, 'total=3+3+3=9 分（浮点聚合会算成 7.5 分，错 1~2 分）')

// ── golden vector 2：整数场景，金额一致 ──────────────────────────────────
const v2 = priceBreakdownToCents({
  baseMonthly: 9.99,
  addons: [
    { value: 2, unitPrice: 0.5 },
    { value: 1, unitPrice: 0.25 }
  ],
  multiplier: 3
})
assert.equal(v2.baseAmount, 2997, 'base=round(999*3)=2997 分')
assert.deepEqual(v2.addonAmounts, [300, 75], 'core=round(2*50*3)=300，disk=round(25*3)=75')
assert.equal(v2.totalCents, 3372, 'total=2997+300+75=3372 分 = 33.72 元')

// ── 边界：无 addon / 零值 ────────────────────────────────────────────────
const empty = priceBreakdownToCents({ baseMonthly: 10, addons: [], multiplier: 1 })
assert.equal(empty.totalCents, 1000, '无 addon 时 total=base')
const zero = priceBreakdownToCents({ baseMonthly: 0, addons: [], multiplier: 1 })
assert.equal(zero.totalCents, 0, '零值 total=0')

// ── 源码漂移防护：购买页与 PriceCalculator 必须引用共享逐项取整 ──────────
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import path from 'node:path'
const here = path.dirname(fileURLToPath(import.meta.url))
const srcRoot = path.resolve(here, '../../../src')
const buyPage = readFileSync(path.resolve(srcRoot, 'views/public/buy/index.vue'), 'utf8')
assert.ok(buyPage.includes('priceBreakdownToCents'), '购买页合计必须用共享逐项取整')
const priceCalc = readFileSync(
  path.resolve(srcRoot, 'components/business/price-calculator/index.vue'),
  'utf8'
)
assert.ok(priceCalc.includes('priceBreakdownToCents'), 'PriceCalculator 合计必须用共享逐项取整')

console.log('✅ pricing.regression.test.mjs —— 订单价格预览 golden vectors 断言通过')
