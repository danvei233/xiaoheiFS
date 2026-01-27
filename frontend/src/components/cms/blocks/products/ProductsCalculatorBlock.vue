<template>
  <section class="calculator-section scroll-animate">
    <div class="calculator-container">
      <h2 class="section-title">{{ content.title || $t('products.calculator.title') || '智能推荐' }}</h2>
      <p class="section-desc">{{ content.desc || $t('products.calculator.desc') || '选择您的使用场景，我们将为您推荐最佳配置' }}</p>
      <div class="scenario-buttons">
        <button
          v-for="(scenario, index) in scenarios"
          :key="index"
          class="scenario-btn"
          :class="{ active: selectedScenario === index }"
          @click="() => onSelect?.(index)"
        >
          <span class="scenario-icon">{{ scenario.icon }}</span>
          <span class="scenario-name">{{ scenario.name }}</span>
        </button>
      </div>
      <div class="recommendation" v-if="selectedScenario !== null">
        <div class="rec-label">{{ $t('products.calculator.recommended') || '推荐配置' }}</div>
        <div class="rec-config">{{ scenarios[selectedScenario].recommended }}</div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
interface Scenario {
  icon?: string
  name?: string
  recommended?: string
  plan?: number
}

defineProps<{
  content: { title?: string; desc?: string }
  scenarios: Scenario[]
  selectedScenario: number | null
  onSelect?: (index: number) => void
}>()
</script>

