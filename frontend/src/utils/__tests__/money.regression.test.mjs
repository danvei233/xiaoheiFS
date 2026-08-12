/**
 * 回归测试：金额格式化（P2-72 / m059 12b）。
 *
 * 背景：仪表盘手写 `¥${value.toFixed(2)}`，币种非 CNY 时符号错误；模板的
 * MoneyFormatter 在缺货币占位符时丢 ¥。统一用 Intl.NumberFormat 按币种格式化。
 *
 * 运行：`node --test src/utils/__tests__/money.regression.test.mjs`
 */
import assert from 'node:assert/strict'

const { formatMoney } = await import('../../utils/money.ts')

// ── CNY 显示 ¥ ────────────────────────────────────────────────────────
assert.equal(formatMoney(100), '¥100.00', '默认币种 CNY 显示 ¥ 符号')
assert.equal(formatMoney(1234.5, 'CNY'), '¥1,234.50', 'CNY 千分位 + 两位小数')
assert.equal(formatMoney('100.50', 'CNY'), '¥100.50', '字符串金额可解析')

// ── 其他币种用对应符号/码（zh-CN 下 USD 显示 US$、EUR 显示 €）────────────
assert.equal(formatMoney(100, 'USD'), 'US$100.00', 'USD 显示 US$，绝不能用 ¥')
assert.ok(formatMoney(100, 'EUR').includes('€'), 'EUR 显示 €')
assert.ok(!formatMoney(100, 'USD').includes('¥'), 'USD 结果不得出现 ¥')

// ── 非法/未知币种回退为"币种码 + 数值"，不显示错误符号 ──────────────────
const xyz = formatMoney(100, 'XYZ')
assert.ok(xyz.startsWith('XYZ'), '未知币种码以币种码开头')
assert.ok(xyz.includes('100.00'), '未知币种码带正确数值')
assert.ok(!xyz.includes('¥'), '未知币种码不得出现 ¥')

// ── 非有限数值返回 '-'；缺失值（null/undefined）按 0 处理 ───────────────
assert.equal(formatMoney(NaN), '-', 'NaN 返回 -')
assert.equal(formatMoney(Infinity), '-', 'Infinity 返回 -')
assert.equal(formatMoney(undefined), '¥0.00', 'undefined 按 0 处理')
assert.equal(formatMoney(null), '¥0.00', 'null 按 0 处理')

console.log('✅ money.regression.test.mjs —— 币种格式化断言通过')
