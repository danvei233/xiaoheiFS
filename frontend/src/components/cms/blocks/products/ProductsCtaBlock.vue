<template>
  <section class="cta-section scroll-animate">
    <div class="cta-container">
      <div class="cta-content">
        <!-- Title - 可编辑 -->
        <InlineEdit
          v-if="isVisualMode"
          field-path="productsCta.title"
          v-model="localTitle"
          edit-type="text"
          label="标题"
        />
        <h2 v-else class="cta-title">{{ content.title || $t('products.cta.title') || '需要定制方案？' }}</h2>

        <!-- Description - 可编辑 -->
        <InlineEdit
          v-if="isVisualMode"
          field-path="productsCta.desc"
          v-model="localDesc"
          edit-type="textarea"
          label="描述"
          :rows="2"
        />
        <p v-else class="cta-desc">{{ content.desc || $t('products.cta.desc') || '联系我们的销售团队，为您量身定制企业级云解决方案' }}</p>

        <!-- Actions - 可编辑 -->
        <template v-if="isVisualMode">
          <div class="cta-actions">
            <InlineEdit
              field-path="productsCta.contact_text"
              v-model="localContactText"
              edit-type="text"
              label="联系按钮文案"
            />
            <InlineEdit
              field-path="productsCta.contact_link"
              v-model="localContactLink"
              edit-type="url"
              label="联系链接"
            />
            <InlineEdit
              field-path="productsCta.email"
              v-model="localEmail"
              edit-type="text"
              label="邮箱"
            />
          </div>
        </template>
        <div v-else class="cta-actions">
          <router-link :to="content.contact_link || '/console/tickets'" class="btn btn-primary">
            <span>{{ content.contact_text || $t('products.cta.contact') || '联系销售' }}</span>
          </router-link>
          <a :href="`mailto:${content.email || 'sales@example.com'}`" class="btn btn-secondary">
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
              <path d="M3 5L8 10L13 5" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
            <span>{{ content.email || 'sales@example.com' }}</span>
          </a>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { RouterLink } from 'vue-router'
import { inject, computed } from 'vue'
import InlineEdit from '@/components/InlineEdit.vue'

const props = defineProps<{
  content: { title?: string; desc?: string; contact_text?: string; contact_link?: string; email?: string }
}>()

// 获取 CMS 编辑上下文
const cmsEditContext: any = inject('cmsEditContext', null)
const isVisualMode = cmsEditContext?.editMode === 'visual'

// 本地绑定 computed 属性
const localTitle = computed({
  get: () => props.content.title || '',
  set: (val: string) => cmsEditContext?.updateField('productsCta.title', val)
})

const localDesc = computed({
  get: () => props.content.desc || '',
  set: (val: string) => cmsEditContext?.updateField('productsCta.desc', val)
})

const localContactText = computed({
  get: () => props.content.contact_text || '',
  set: (val: string) => cmsEditContext?.updateField('productsCta.contact_text', val)
})

const localContactLink = computed({
  get: () => props.content.contact_link || '',
  set: (val: string) => cmsEditContext?.updateField('productsCta.contact_link', val)
})

const localEmail = computed({
  get: () => props.content.email || '',
  set: (val: string) => cmsEditContext?.updateField('productsCta.email', val)
})
</script>

<style scoped>
/* 覆盖 InlineEdit 样式 */
.cta-content :deep(.editable-region) {
  display: block;
  margin-bottom: 12px;
}

.cta-actions :deep(.editable-region) {
  display: inline-block;
}
</style>
