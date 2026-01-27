<template>
  <a-card class="card price-card">
    <div class="section-title">价格计算</div>
    <div class="price-rows">
      <div class="price-row">
        <span>基础价</span>
        <strong>{{ format(basePrice) }}</strong>
      </div>
      <div class="price-row">
        <span>附加项</span>
        <strong>{{ format(addonPrice) }}</strong>
      </div>
      <div class="price-row">
        <span>周期倍率</span>
        <strong>{{ cycleMultiplier }}x</strong>
      </div>
      <div class="price-row">
        <span>购买数量</span>
        <strong>{{ qty }}</strong>
      </div>
      <div class="price-divider"></div>
      <div class="price-row total">
        <span>合计</span>
        <strong>{{ format(total) }}</strong>
      </div>
    </div>
  </a-card>
</template>

<script setup>
import { computed } from "vue";

const props = defineProps({
  basePrice: { type: Number, default: 0 },
  addonPrice: { type: Number, default: 0 },
  cycleMultiplier: { type: Number, default: 1 },
  qty: { type: Number, default: 1 },
  currency: { type: String, default: "¥" }
});

const total = computed(() => (props.basePrice + props.addonPrice) * props.cycleMultiplier * props.qty);

const format = (val) => `${props.currency} ${Number(val || 0).toFixed(2)}`;
</script>

<style scoped>
.price-card {
  border-style: dashed;
}

.price-rows {
  display: grid;
  gap: 10px;
}

.price-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  color: var(--text2);
}

.price-row strong {
  color: var(--text);
  font-weight: 600;
}

.price-divider {
  height: 1px;
  background: var(--border);
}

.price-row.total {
  color: var(--text);
  font-size: 16px;
}
</style>
