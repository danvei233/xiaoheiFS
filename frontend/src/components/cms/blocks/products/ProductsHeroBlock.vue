<template>
  <section class="products-hero">
    <div class="hero-container">
      <!-- Badge - 可编辑 -->
      <InlineEdit
        v-if="isVisualMode"
        field-path="productsHero.badge"
        v-model="localBadge"
        edit-type="text"
        label="徽标"
      />
      <div v-else class="hero-badge scroll-animate">
        <span class="badge-dot"></span>
        <span>{{ content.badge || $t('products.badge') || '灵活配置，按需选择' }}</span>
      </div>

      <!-- Title - 可编辑 -->
      <InlineEdit
        v-if="isVisualMode"
        field-path="productsHero.title"
        v-model="localTitle"
        edit-type="text"
        label="标题"
      />
      <h1 v-else class="hero-title scroll-animate">
        {{ content.title || $t('products.title') || '选择最适合您的云服务方案' }}
      </h1>

      <!-- Subtitle - 可编辑 -->
      <InlineEdit
        v-if="isVisualMode"
        field-path="productsHero.subtitle"
        v-model="localSubtitle"
        edit-type="textarea"
        label="副标题"
        :rows="2"
      />
      <p v-else class="hero-subtitle scroll-animate">
        {{
          content.subtitle ||
          $t('products.subtitle') ||
          '从入门到企业级，我们提供全面的云服务器解决方案。所有套餐均包含99.99% SLA保障。'
        }}
      </p>

      <!-- Features - 可编辑数组 -->
      <div class="hero-features scroll-animate">
        <template v-if="isVisualMode">
          <div
            v-for="(feature, index) in editableFeatures"
            :key="`feature-${index}`"
            class="hero-feature"
          >
            <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
              <path
                d="M16.6667 5L7.50001 14.1667L3.33334 10"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
              />
            </svg>
            <InlineEdit
              :field-path="`productsHero.features.${index}`"
              v-model="editableFeatures[index]"
              edit-type="text"
              label="特性"
              :is-array-item="true"
              :can-add="index === editableFeatures.length - 1"
              :can-remove="editableFeatures.length > 1"
              @add-item="addFeatureItem"
              @remove-item="() => removeFeatureItem(index)"
            />
          </div>
          <div v-if="editableFeatures.length === 0" class="empty-features">
            <a-button type="dashed" @click="addFeatureItem">+ 添加特性</a-button>
          </div>
        </template>
        <template v-else>
          <div class="hero-feature" v-for="(feature, index) in content.features || []" :key="index">
            <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
              <path
                d="M16.6667 5L7.50001 14.1667L3.33334 10"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
              />
            </svg>
            <span>{{ feature }}</span>
          </div>
        </template>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { inject, computed } from 'vue'
import { Button as AButton } from 'ant-design-vue'
import InlineEdit from '@/components/InlineEdit.vue'

const props = defineProps<{
  content: { badge?: string; title?: string; subtitle?: string; features?: string[] }
}>()

// 获取 CMS 编辑上下文
const cmsEditContext: any = inject('cmsEditContext', null)
const isVisualMode = cmsEditContext?.editMode === 'visual'

// 本地绑定 computed 属性
const localBadge = computed({
  get: () => props.content.badge || '',
  set: (val: string) => cmsEditContext?.updateField('productsHero.badge', val)
})

const localTitle = computed({
  get: () => props.content.title || '',
  set: (val: string) => cmsEditContext?.updateField('productsHero.title', val)
})

const localSubtitle = computed({
  get: () => props.content.subtitle || '',
  set: (val: string) => cmsEditContext?.updateField('productsHero.subtitle', val)
})

// 编辑模式下的 features 数组
const editableFeatures = computed({
  get: () => props.content.features || [],
  set: (val) => {
    // 数组直接引用
  }
})

// Features 数组操作
const addFeatureItem = () => {
  cmsEditContext?.addArrayItem('productsHero.features', '新特性')
}

const removeFeatureItem = (index: number) => {
  cmsEditContext?.removeArrayItem('productsHero.features', index)
}
</script>

<style scoped>
.empty-features {
  width: 100%;
  display: flex;
  justify-content: center;
  padding: 20px;
}

/* 覆盖 InlineEdit 样式 */
.hero-container :deep(.editable-region) {
  display: block;
  margin-bottom: 12px;
}

.hero-feature :deep(.editable-region) {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}
</style>
