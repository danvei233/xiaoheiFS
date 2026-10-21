<template>
  <section class="features-section">
    <div class="features-container">
      <div class="section-header scroll-animate">
        <!-- Badge - 可编辑 -->
        <InlineEdit
          v-if="isVisualMode"
          field-path="features.badge"
          v-model="localBadge"
          edit-type="text"
          label="模块徽标"
        />
        <div v-show="!isVisualMode" class="section-badge">{{ content.badge || $t('home.features.badge') || '核心优势' }}</div>

        <!-- Title - 可编辑 -->
        <InlineEdit
          v-if="isVisualMode"
          field-path="features.title"
          v-model="localTitle"
          edit-type="text"
          label="模块标题"
        />
        <h2 v-show="!isVisualMode" class="section-title">{{ content.title || $t('home.features.title') || '为什么选择我们的云服务' }}</h2>

        <!-- Description - 可编辑 -->
        <InlineEdit
          v-if="isVisualMode"
          field-path="features.desc"
          v-model="localDesc"
          edit-type="textarea"
          label="模块描述"
          :rows="2"
        />
        <p v-show="!isVisualMode" class="section-desc">{{ content.desc || $t('home.features.desc') || '我们提供企业级基础设施，助力您的业务快速增长' }}</p>
      </div>

      <div class="features-grid">
        <!-- Features Items - 可编辑数组 -->
        <template v-if="isVisualMode">
          <div
            v-for="(feature, index) in editableFeatures"
            :key="`feature-${index}`"
            class="feature-card scroll-animate"
            :style="{ '--delay': `${index * 0.1}s`, '--card-index': index }"
          >
            <div class="feature-bg"></div>
            <div class="feature-glow" :class="`feature-glow-${index}`"></div>

            <!-- Icon - 暂时保持原样 -->
            <div class="feature-icon">
              <component :is="feature.icon" v-if="feature.icon" />
              <svg v-else width="32" height="32" viewBox="0 0 24 24" fill="none">
                <path d="M13 2L3 14H12L11 22L21 10H12L13 2Z" stroke="currentColor" stroke-width="2"/>
              </svg>
            </div>

            <!-- Title - 可编辑 -->
            <InlineEdit
              :field-path="`features.items.${index}.title`"
              v-model="feature.title"
              edit-type="text"
              label="特性标题"
              :is-array-item="true"
              :can-add="index === editableFeatures.length - 1"
              :can-remove="editableFeatures.length > 1"
              @add-item="addFeatureItem"
              @remove-item="() => removeFeatureItem(index)"
            />

            <!-- Description - 可编辑 -->
            <InlineEdit
              :field-path="`features.items.${index}.description`"
              v-model="feature.description"
              edit-type="textarea"
              label="特性描述"
              :rows="3"
            />

            <!-- Learn More 链接在编辑模式下隐藏 -->
          </div>
        </template>
        <template v-else>
          <div
            v-for="(feature, index) in editableFeatures"
            :key="`feature-${index}`"
            class="feature-card scroll-animate"
            :style="{ '--delay': `${index * 0.1}s`, '--card-index': index }"
          >
            <div class="feature-bg"></div>
            <div class="feature-glow" :class="`feature-glow-${index}`"></div>

            <!-- Icon -->
            <div class="feature-icon">
              <component :is="feature.icon" v-if="feature.icon" />
              <svg v-else width="32" height="32" viewBox="0 0 24 24" fill="none">
                <path d="M13 2L3 14H12L11 22L21 10H12L13 2Z" stroke="currentColor" stroke-width="2"/>
              </svg>
            </div>

            <!-- Title -->
            <h3>{{ feature.title }}</h3>

            <!-- Description -->
            <p>{{ feature.description }}</p>

            <!-- Learn More 链接 -->
          </div>
        </template>

        <!-- 空数组时显示添加按钮 -->
        <div v-if="editableFeatures.length === 0" class="empty-features">
          <a-button type="dashed" @click="addFeatureItem">+ 添加特性</a-button>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { inject, computed, onMounted } from 'vue'
import { Button as AButton } from 'ant-design-vue'
import InlineEdit from '@/components/InlineEdit.vue'

interface FeatureItem {
  icon?: any
  title?: string
  description?: string
}

interface FeaturesContent {
  badge?: string
  title?: string
  desc?: string
}

const props = defineProps<{
  content: FeaturesContent
  features: FeatureItem[]
  handleFeatureGlow?: (event: MouseEvent, index: number) => void
  resetFeatureGlow?: (index: number) => void
  registerFeatureBg?: (el: HTMLElement | null, index: number) => void
}>()

// 获取 CMS 编辑上下文
const cmsEditContext: any = inject('cmsEditContext', null)
const isVisualMode = cmsEditContext?.editMode === 'visual'

// Debug: Log when component mounts
onMounted(() => {
  console.log('[HomeFeaturesBlock] Component mounted!', {
    isVisualMode,
    content: props.content,
    featuresCount: props.features?.length,
    cmsEditContext: cmsEditContext ? 'exists' : 'NULL'
  })
})

// 本地绑定 computed 属性
const localBadge = computed({
  get: () => props.content.badge || '',
  set: (val: string) => cmsEditContext?.updateField('features.badge', val)
})

const localTitle = computed({
  get: () => props.content.title || '',
  set: (val: string) => cmsEditContext?.updateField('features.title', val)
})

const localDesc = computed({
  get: () => props.content.desc || '',
  set: (val: string) => cmsEditContext?.updateField('features.desc', val)
})

// 编辑模式下的 features 数组（直接绑定到 formContent）
const editableFeatures = computed({
  get: () => props.features || [],
  set: (val) => {
    // 数组直接引用，不需要 setter
  }
})

// Attach background ref (非编辑模式)
const attachBg = (el: HTMLElement | null, index: number) => {
  props.registerFeatureBg?.(el, index)
}

// Features 数组操作
const addFeatureItem = () => {
  cmsEditContext?.addArrayItem('features.items', {
    title: '新特性',
    description: '特性描述'
  })
}

const removeFeatureItem = (index: number) => {
  cmsEditContext?.removeArrayItem('features.items', index)
}
</script>

<style scoped>
.empty-features {
  grid-column: 1 / -1;
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 40px;
}

/* 覆盖 InlineEdit 样式以适配 features */
.section-header :deep(.editable-region) {
  display: block;
  margin-bottom: 8px;
}

.feature-card :deep(.editable-region) {
  display: block;
  margin-bottom: 12px;
}

.feature-card :deep(.editable-region:hover) {
  background: transparent;
}
</style>
