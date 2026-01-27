<template>
  <section class="comparison-section scroll-animate">
    <div class="comparison-container">
      <h2 class="section-title">{{ content.title || $t('products.compare.title') || '详细配置对比' }}</h2>
      <div class="comparison-table">
        <div class="comparison-row header">
          <div class="col-feature">配置项</div>
          <div class="col-plan" v-for="(product, index) in products" :key="index">
            {{ product.name }}
          </div>
        </div>
        <div class="comparison-row" v-for="(row, index) in rows" :key="index">
          <div class="col-feature">{{ row.feature }}</div>
          <div class="col-plan" v-for="(product, idx) in products" :key="idx">
            <span class="value">{{ row.values[idx] }}</span>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
interface ProductItem {
  name?: string
}

defineProps<{
  content: { title?: string }
  products: ProductItem[]
  rows: Array<{ feature?: string; values?: string[] }>
}>()
</script>

