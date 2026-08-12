/**
 * 金额格式化（P2-72 / P3-51 12b）。
 *
 * 统一用 Intl.NumberFormat 按币种格式化，避免各页手写 `¥` 前缀：
 *  - 币种为 CNY 时正确显示 ¥ 符号；
 *  - 未知/非法币种码回退为"币种码 + 数值"，绝不用错符号；
 *  - 非有限数值返回 '-'，不渲染 NaN/Infinity。
 */
export function formatMoney(amount: unknown, currency?: string, locale = 'zh-CN'): string {
  const value = Number(amount ?? 0)
  if (!Number.isFinite(value)) return '-'
  const normalizedCurrency =
    String(currency || 'CNY')
      .trim()
      .toUpperCase() || 'CNY'
  try {
    return new Intl.NumberFormat(locale, {
      style: 'currency',
      currency: normalizedCurrency,
      minimumFractionDigits: 2
    }).format(value)
  } catch {
    return `${normalizedCurrency} ${value.toFixed(2)}`
  }
}
