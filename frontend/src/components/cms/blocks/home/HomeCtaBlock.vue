<template>
  <section class="cta-section">
    <div class="cta-container">
      <div class="cta-content scroll-animate">
        <!-- Title - 可编辑 -->
        <InlineEdit
          v-if="isVisualMode"
          field-path="cta.title"
          v-model="localTitle"
          edit-type="text"
          label="标题"
        />
        <h2 v-else class="cta-title">
          {{ content.title || $t('home.cta.title') || '准备好开启您的云端之旅了吗？' }}
        </h2>

        <!-- Description - 可编辑 -->
        <InlineEdit
          v-if="isVisualMode"
          field-path="cta.desc"
          v-model="localDesc"
          edit-type="textarea"
          label="描述"
          :rows="2"
        />
        <p v-else class="cta-desc">
          {{ content.desc || $t('home.cta.desc') || '立即注册，新用户享受免费试用额度，体验企业级云服务' }}
        </p>

        <!-- Button - 可编辑 -->
        <template v-if="isVisualMode">
          <div class="cta-actions">
            <InlineEdit
              field-path="cta.button_text"
              v-model="localButtonText"
              edit-type="text"
              label="按钮文案"
            />
            <InlineEdit
              field-path="cta.button_link"
              v-model="localButtonLink"
              edit-type="url"
              label="按钮链接"
            />
          </div>
        </template>
        <div v-else class="cta-actions">
          <router-link :to="content.button_link || '/register'" class="btn btn-primary btn-large">
            <span>{{ content.button_text || $t('home.cta.button') || '免费开始使用' }}</span>
            <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
              <path d="M4 10H16M16 10L11 5M16 10L11 15" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </router-link>
        </div>

        <!-- Features - 可编辑数组 -->
        <div class="cta-features">
          <template v-if="isVisualMode">
            <div
              v-for="(feature, index) in editableFeatures"
              :key="`feature-${index}`"
              class="cta-feature"
              :style="{ '--delay': `${index * 0.1}s` }"
            >
              <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
                <path d="M16.6667 5L7.50001 14.1667L3.33334 10" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
              </svg>
              <InlineEdit
                :field-path="`cta.features.${index}`"
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
            <!-- 空数组时显示添加按钮 -->
            <div v-if="editableFeatures.length === 0" class="empty-cta-features">
              <a-button type="dashed" @click="addFeatureItem">+ 添加特性</a-button>
            </div>
          </template>
          <template v-else>
            <div class="cta-feature" v-for="(feature, index) in features" :key="index" :style="{ '--delay': `${index * 0.1}s` }">
              <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
                <path d="M16.6667 5L7.50001 14.1667L3.33334 10" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
              </svg>
              <span>{{ feature }}</span>
            </div>
          </template>
        </div>
      </div>

      <!-- Visual decoration - 静态 -->
      <div class="cta-visual">
        <div class="cta-orb orb-1"></div>
        <div class="cta-orb orb-2"></div>
        <div class="cta-orb orb-3"></div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { RouterLink } from 'vue-router'
import { inject, computed } from 'vue'
import { Button as AButton } from 'ant-design-vue'
import InlineEdit from '@/components/InlineEdit.vue'

interface CtaContent {
  title?: string
  desc?: string
  button_text?: string
  button_link?: string
}

const props = defineProps<{
  content: CtaContent
  features: string[]
}>()

// 获取 CMS 编辑上下文
const cmsEditContext: any = inject('cmsEditContext', null)
const isVisualMode = cmsEditContext?.editMode === 'visual'

// 本地绑定 computed 属性
const localTitle = computed({
  get: () => props.content.title || '',
  set: (val: string) => cmsEditContext?.updateField('cta.title', val)
})

const localDesc = computed({
  get: () => props.content.desc || '',
  set: (val: string) => cmsEditContext?.updateField('cta.desc', val)
})

const localButtonText = computed({
  get: () => props.content.button_text || '',
  set: (val: string) => cmsEditContext?.updateField('cta.button_text', val)
})

const localButtonLink = computed({
  get: () => props.content.button_link || '',
  set: (val: string) => cmsEditContext?.updateField('cta.button_link', val)
})

// 编辑模式下的 features 数组
const editableFeatures = computed({
  get: () => props.features || [],
  set: (val) => {
    // 数组直接引用，不需要 setter
  }
})

// Features 数组操作
const addFeatureItem = () => {
  cmsEditContext?.addArrayItem('cta.features', '新特性')
}

const removeFeatureItem = (index: number) => {
  cmsEditContext?.removeArrayItem('cta.features', index)
}
</script>

<style scoped>
.empty-cta-features {
  width: 100%;
  display: flex;
  justify-content: center;
  padding: 20px;
}

/* 覆盖 InlineEdit 样式以适配 CTA */
.cta-content :deep(.editable-region) {
  display: block;
  margin-bottom: 12px;
}

.cta-actions :deep(.editable-region) {
  display: inline-block;
}

.cta-feature :deep(.editable-region) {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.cta-feature :deep(.editable-region:hover) {
  background: transparent;
}
</style>
