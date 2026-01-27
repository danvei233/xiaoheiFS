<template>
  <footer class="public-footer">
    <div class="footer-container">
      <div class="footer-top">
        <div class="footer-brand">
          <div class="footer-logo">
            <div class="logo-icon" aria-hidden="true">
              <SiteLogoMedia :size="20" :src="logoUrl" :alt="siteName" />
            </div>
            <span class="footer-logo-text">{{ siteName }}</span>
          </div>

          <!-- Description - 可编辑 -->
          <InlineEdit
            v-if="isVisualMode"
            field-path="footer.description"
            v-model="localDescription"
            edit-type="textarea"
            label="网站描述"
            :rows="2"
          />
          <p v-else class="footer-desc">{{ content.description || $t('footer.description') || '专业的云服务提供商，为企业提供可靠、安全、高性能的云计算解决方案' }}</p>

          <!-- Social Links - 可编辑数组 -->
          <div class="footer-social">
            <template v-if="isVisualMode">
              <template v-for="(social, index) in editableSocialLinks" :key="`social-${index}`">
                <InlineEdit
                  :field-path="`footer.social_links.${index}.key`"
                  v-model="social.key"
                  edit-type="text"
                  label="社交平台名称"
                  :is-array-item="true"
                  :can-add="index === editableSocialLinks.length - 1"
                  :can-remove="editableSocialLinks.length > 1"
                  @add-item="addSocialItem"
                  @remove-item="() => removeSocialItem(index)"
                />
                <InlineEdit
                  :field-path="`footer.social_links.${index}.url`"
                  v-model="social.url"
                  edit-type="url"
                  label="社交平台链接"
                />
              </template>
              <div v-if="editableSocialLinks.length === 0" class="empty-social">
                <a-button type="dashed" @click="addSocialItem">+ 添加社交链接</a-button>
              </div>
            </template>
            <template v-else>
              <a :href="content.social_links?.[0]?.url || '#'"><span class="social-label">{{ content.social_links?.[0]?.key || 'GitHub' }}</span></a>
              <a :href="content.social_links?.[1]?.url || '#'"><span class="social-label">{{ content.social_links?.[1]?.key || 'Twitter' }}</span></a>
              <a :href="content.social_links?.[2]?.url || '#'"><span class="social-label">{{ content.social_links?.[2]?.key || 'Discord' }}</span></a>
            </template>
          </div>
        </div>

        <!-- Sections - 可编辑嵌套数组 -->
        <div class="footer-sections">
          <template v-if="isVisualMode">
            <div
              v-for="(section, index) in editableSections"
              :key="`section-${index}`"
              class="footer-section"
            >
              <!-- Section Title - 可编辑 -->
              <InlineEdit
                :field-path="`footer.sections.${index}.title`"
                v-model="section.title"
                edit-type="text"
                label="栏目标题"
                :is-array-item="true"
                :can-add="index === editableSections.length - 1"
                :can-remove="editableSections.length > 1"
                @add-item="addSectionItem"
                @remove-item="() => removeSectionItem(index)"
              />

              <!-- Section Links - 可编辑数组 -->
              <template v-for="(link, linkIndex) in section.links" :key="`link-${index}-${linkIndex}`">
                <InlineEdit
                  :field-path="`footer.sections.${index}.links.${linkIndex}.label`"
                  v-model="link.label"
                  edit-type="text"
                  label="链接文字"
                  :is-array-item="true"
                  :can-add="linkIndex === section.links.length - 1"
                  :can-remove="section.links.length > 1"
                  @add-item="() => addLinkItem(index)"
                  @remove-item="() => removeLinkItem(index, linkIndex)"
                />
                <InlineEdit
                  :field-path="`footer.sections.${index}.links.${linkIndex}.url`"
                  v-model="link.url"
                  edit-type="url"
                  label="链接地址"
                />
              </template>
            </div>
            <div v-if="editableSections.length === 0" class="empty-sections">
              <a-button type="dashed" @click="addSectionItem">+ 添加栏目</a-button>
            </div>
          </template>
          <template v-else>
            <div class="footer-section" v-for="(section, index) in sections" :key="index">
              <h4>{{ section.title }}</h4>
              <a v-for="(link, linkIndex) in section.links" :key="linkIndex" :href="link.url">{{ link.label }}</a>
            </div>
          </template>
        </div>
      </div>

      <!-- Footer Bottom -->
      <div class="footer-bottom">
        <div class="footer-bottom-left">
          <p>&copy; {{ new Date().getFullYear() }} {{ siteName }}. {{ $t('footer.rights') || 'All rights reserved.' }}</p>
        </div>

        <!-- Badges - 可编辑数组 -->
        <div class="footer-bottom-right">
          <template v-if="isVisualMode">
            <template v-for="(badge, index) in editableBadges" :key="`badge-${index}`">
              <InlineEdit
                :field-path="`footer.badges.${index}`"
                v-model="editableBadges[index]"
                edit-type="text"
                label="徽章文字"
                :is-array-item="true"
                :can-add="index === editableBadges.length - 1 && editableBadges.length < 4"
                :can-remove="editableBadges.length > 1"
                @add-item="addBadgeItem"
                @remove-item="() => removeBadgeItem(index)"
              />
            </template>
            <div v-if="editableBadges.length === 0" class="empty-badges">
              <a-button type="dashed" @click="addBadgeItem">+ 添加徽章</a-button>
            </div>
          </template>
          <template v-else>
            <span class="footer-badge">{{ badges[0] || $t('footer.uptime') || '99.99% Uptime' }}</span>
            <span class="footer-badge">{{ badges[1] || $t('footer.secure') || 'SOC2 Certified' }}</span>
          </template>
        </div>
      </div>
    </div>
  </footer>
</template>

<script setup lang="ts">
import { inject, computed } from 'vue'
import { Button as AButton } from 'ant-design-vue'
import InlineEdit from '@/components/InlineEdit.vue'
import SiteLogoMedia from '@/components/brand/SiteLogoMedia.vue'

const props = defineProps<{
  siteName: string
  logoUrl?: string
  content: { description?: string; social_links?: Array<{ key?: string; url?: string }> }
  sections: Array<{ title?: string; links: Array<{ label?: string; url?: string }> }>
  badges: string[]
}>()

// 获取 CMS 编辑上下文
const cmsEditContext: any = inject('cmsEditContext', null)
const isVisualMode = cmsEditContext?.editMode === 'visual'

// 本地绑定 computed 属性
const localDescription = computed({
  get: () => props.content.description || '',
  set: (val: string) => cmsEditContext?.updateField('footer.description', val)
})

// 编辑模式下的 social_links 数组
const editableSocialLinks = computed({
  get: () => props.content.social_links || [],
  set: (val) => {
    // 数组直接引用
  }
})

// 编辑模式下的 sections 数组
const editableSections = computed({
  get: () => props.sections || [],
  set: (val) => {
    // 数组直接引用
  }
})

// 编辑模式下的 badges 数组
const editableBadges = computed({
  get: () => props.badges || [],
  set: (val) => {
    // 数组直接引用
  }
})

// Social Links 数组操作
const addSocialItem = () => {
  cmsEditContext?.addArrayItem('footer.social_links', { key: '新平台', url: '#' })
}

const removeSocialItem = (index: number) => {
  cmsEditContext?.removeArrayItem('footer.social_links', index)
}

// Sections 数组操作
const addSectionItem = () => {
  cmsEditContext?.addArrayItem('footer.sections', {
    title: '新栏目',
    links: [{ label: '新链接', url: '#' }]
  })
}

const removeSectionItem = (index: number) => {
  cmsEditContext?.removeArrayItem('footer.sections', index)
}

// Links 数组操作（嵌套）
const addLinkItem = (sectionIndex: number) => {
  const section = editableSections.value[sectionIndex]
  if (section && Array.isArray(section.links)) {
    section.links.push({ label: '新链接', url: '#' })
  }
}

const removeLinkItem = (sectionIndex: number, linkIndex: number) => {
  const section = editableSections.value[sectionIndex]
  if (section && Array.isArray(section.links) && linkIndex >= 0 && linkIndex < section.links.length) {
    section.links.splice(linkIndex, 1)
  }
}

// Badges 数组操作
const addBadgeItem = () => {
  cmsEditContext?.addArrayItem('footer.badges', '新徽章')
}

const removeBadgeItem = (index: number) => {
  cmsEditContext?.removeArrayItem('footer.badges', index)
}
</script>

<style scoped>
.empty-social,
.empty-sections,
.empty-badges {
  display: flex;
  justify-content: center;
  padding: 12px;
}

/* 覆盖 InlineEdit 样式以适配 Footer */
.footer-brand :deep(.editable-region),
.footer-section :deep(.editable-region) {
  display: block;
  margin-bottom: 8px;
}

.footer-sections :deep(.editable-region) {
  display: block;
  margin-bottom: 4px;
}

.footer-social :deep(.editable-region) {
  display: inline-block;
  margin-right: 8px;
}

.footer-bottom-right :deep(.editable-region) {
  display: inline-block;
  margin-right: 8px;
}
</style>
