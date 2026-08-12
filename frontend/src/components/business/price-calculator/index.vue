<template>
  <ElCard class="price-card art-card-xs" shadow="never">
    <template #header>
      <div class="card-title">价格计算</div>
    </template>

    <ElDescriptions :column="1" border size="small">
      <ElDescriptionsItem label="基础价">
        <strong class="price-value">{{ format(basePrice) }}</strong>
      </ElDescriptionsItem>
      <ElDescriptionsItem label="附加项">
        <strong class="price-value">{{ format(addonPrice) }}</strong>
      </ElDescriptionsItem>
      <ElDescriptionsItem label="周期倍率">
        <strong class="price-value">{{ cycleMultiplier }}x</strong>
      </ElDescriptionsItem>
      <ElDescriptionsItem label="购买数量">
        <strong class="price-value">{{ qty }}</strong>
      </ElDescriptionsItem>
      <ElDescriptionsItem>
        <template #label>
          <span class="total-label">合计</span>
        </template>
        <strong class="price-value total-value">{{ format(total) }}</strong>
      </ElDescriptionsItem>
    </ElDescriptions>
  </ElCard>
</template>

<script setup lang="ts">
  import { priceBreakdownToCents } from '@/utils/pricing'

  defineOptions({ name: 'PriceCalculator' })

  const props = withDefaults(
    defineProps<{
      basePrice?: number
      /** 附加项总价（元/月，展示用） */
      addonPrice?: number
      cycleMultiplier?: number
      qty?: number
      /** 附加项明细（逐项单价），用于与后端一致的逐项整数舍入 */
      addons?: Array<{ value: number; unitPrice: number }>
      currency?: string
    }>(),
    {
      basePrice: 0,
      addonPrice: 0,
      cycleMultiplier: 1,
      qty: 1,
      addons: () => [],
      currency: '¥'
    }
  )

  // 合计必须与后端逐项整数舍入一致（cross-module "订单价格预览"）：
  // `round(baseMonthly*mult)` + 每个 addon `round(monthly*mult)`，再乘 qty。
  // 不能再用 `(base+addon)*mult*qty` 的浮点聚合，否则多 addon 时差 1 分。
  const total = computed(() => {
    const cents = priceBreakdownToCents({
      baseMonthly: props.basePrice,
      addons: props.addons,
      multiplier: props.cycleMultiplier
    })
    return (cents.totalCents * Math.max(1, Math.round(props.qty || 1))) / 100
  })

  const format = (value: number) => `${props.currency} ${Number(value || 0).toFixed(2)}`
</script>

<style lang="scss" scoped>
  .price-card {
    border-style: dashed;
  }

  .card-title,
  .total-label {
    font-weight: 600;
    color: var(--art-gray-900);
  }

  .price-value {
    display: block;
    font-weight: 600;
    color: var(--art-gray-900);
    text-align: right;
  }

  .total-label,
  .total-value {
    font-size: 16px;
  }

  .total-value {
    color: var(--theme-color);
  }
</style>
