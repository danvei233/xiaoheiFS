/**
 * 订单价格预览 —— 与后端逐项整数舍入一致的共享计算（跨层契约）。
 *
 * 背景（第四轮 cross-module residual "订单价格预览"）：后端下单
 *   （internal/app/order/order_service.go priceBreakdownForPackage）按 **分**
 *   计算：`baseAmount = round(baseMonthly * multiplier)`，每个 addon
 *   （core/mem/disk/bw）也**各自** round 后求和，最后 `total = baseAmount +
 *   addonAmounts`，订单总额再乘 qty。前端购买页此前用浮点
 *   `(basePrice + addonPrice) * multiplier * qty` 预览，当单价/倍率含小数时
 *   与后端差 1 分（round 加法与乘法不可交换）。
 *
 * 这里把前端预览收敛到与后端相同的"逐项取整"语义：单价元→分（round），
 * 每项乘倍率后各自 round，再求和。golden vectors 见
 *   `utils/__tests__/pricing.regression.test.mjs`。
 */

export interface AddonPriceInput {
  /** addon 数量（如附加 CPU 核数） */
  value: number
  /** 单价（元/月） */
  unitPrice: number
}

export interface PriceBreakdownInput {
  /** 套餐月付基础价（元） */
  baseMonthly: number
  /** 附加项列表 */
  addons: AddonPriceInput[]
  /** 计费周期倍率（周期 multiplier × 周期数量） */
  multiplier: number
}

export interface PriceBreakdownCents {
  /** 基础项取整后的金额（分） */
  baseAmount: number
  /** 每个 addon 取整后的金额（分） */
  addonAmounts: number[]
  /** base + 各 addon 之和（分），未乘 qty */
  totalCents: number
}

const toCents = (yuan: number): number => Math.round((yuan || 0) * 100)

/**
 * 按后端逐项整数舍入规则计算金额（分）。
 *
 * @example
 *   priceBreakdownToCents({ baseMonthly: 9.99, addons: [{value:3,unitPrice:0.5}], multiplier: 3 })
 *   // baseAmount = round(999 * 3) = 2997；addonAmount = round(150 * 3) = 450；total = 3447
 */
export function priceBreakdownToCents(input: PriceBreakdownInput): PriceBreakdownCents {
  const baseMonthlyCents = toCents(input.baseMonthly)
  const baseAmount = Math.round(baseMonthlyCents * (input.multiplier || 1))
  const addonAmounts = (input.addons ?? []).map((addon) => {
    const monthlyCents = toCents(addon.unitPrice) * Math.max(0, Math.round(addon.value || 0))
    return Math.round(monthlyCents * (input.multiplier || 1))
  })
  const totalCents = baseAmount + addonAmounts.reduce((sum, amount) => sum + amount, 0)
  return { baseAmount, addonAmounts, totalCents }
}
