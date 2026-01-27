<template>
  <section class="pricing-section">
    <div class="pricing-container">
      <div class="pricing-grid">
        <div
          v-for="(product, index) in products"
          :key="index"
          class="pricing-card scroll-animate"
          :class="{ recommended: product.recommended, selected: selectedPlan === index }"
          :style="{ '--delay': `${index * 0.1}s` }"
          @mouseenter="() => onHover?.(index)"
          @click="() => onSelect?.(index)"
        >
          <div class="recommended-badge" v-if="product.recommended">
            <span>{{ $t('products.recommended') || '推荐' }}</span>
          </div>
          <div class="card-header">
            <div class="product-icon" :class="`icon-${index + 1}`">
              <component :is="product.icon" v-if="product.icon" />
            </div>
            <h3 class="product-name">{{ product.name }}</h3>
            <p class="product-desc">{{ product.description }}</p>
            <div class="product-price">
              <span class="price-currency">￥</span>
              <span class="price-value">{{ product.price }}</span>
              <span class="price-unit">/月</span>
            </div>
          </div>
          <div class="resource-bars">
            <div class="resource-item" v-for="resource in product.resources" :key="resource.label">
              <div class="resource-header">
                <span class="resource-label">{{ resource.label }}</span>
                <span class="resource-value">{{ resource.value }}</span>
              </div>
              <div class="resource-bar">
                <div class="resource-fill" :style="{ width: `${resource.percent}%` }"></div>
              </div>
            </div>
          </div>
          <div class="features-list">
            <div class="feature-item" v-for="(feature, idx) in product.features" :key="idx">
              <div class="feature-check">
                <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
                  <path d="M13.5 4.5L6 12L2.5 8.5" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                </svg>
              </div>
              <span>{{ feature }}</span>
            </div>
          </div>
          <router-link to="/console" class="card-btn" :class="product.recommended ? 'btn-primary' : 'btn-secondary'">
            <span>{{ product.cta }}</span>
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
              <path d="M3 8H13M13 8L9 4M13 8L9 12" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </router-link>
          <div class="card-glow"></div>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { RouterLink } from 'vue-router'

interface ProductItem {
  icon?: any
  name?: string
  description?: string
  price?: string
  recommended?: boolean
  cta?: string
  resources?: Array<{ label?: string; value?: string; percent?: number }>
  features?: string[]
}

defineProps<{
  products: ProductItem[]
  selectedPlan: number
  onSelect?: (index: number) => void
  onHover?: (index: number) => void
}>()
</script>

