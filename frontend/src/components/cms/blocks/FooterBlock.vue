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
              <template v-for="(link, linkIndex) in section.links" :key="linkIndex">
                <router-link v-if="isInternalLink(link.url)" :to="link.url">{{ link.label }}</router-link>
                <a v-else :href="link.url">{{ link.label }}</a>
              </template>
            </div>
          </template>
        </div>
      </div>

      <!-- Footer Bottom -->
      <div class="footer-bottom">
        <div class="footer-bottom-left">
          <!-- 版权信息 -->
          <p class="copyright-text">© {{ copyrightText || `${new Date().getFullYear()} ${siteName}. ${$t('footer.rights') || 'All rights reserved.'}` }}</p>
        </div>

        <!-- 备案信息 - 可编辑数组 - 移到中间 -->
        <div class="footer-bottom-center">
          <div class="beian-info" :class="{ 'beian-multiline': beianInfoList.length > 4 }">
            <template v-if="isVisualMode">
              <div
                v-for="(beian, index) in editableBeianInfo"
                :key="`beian-${index}`"
                class="beian-item-edit"
              >
                <InlineEdit
                  :field-path="`footer.beian_info.${index}.number`"
                  v-model="beian.number"
                  edit-type="text"
                  label="备案号"
                  :is-array-item="true"
                  :can-add="index === editableBeianInfo.length - 1"
                  :can-remove="editableBeianInfo.length > 1"
                  @add-item="addBeianItem"
                  @remove-item="() => removeBeianItem(index)"
                />
                <InlineEdit
                  :field-path="`footer.beian_info.${index}.link_url`"
                  v-model="beian.link_url"
                  edit-type="url"
                  label="备案链接"
                />
                <InlineEdit
                  :field-path="`footer.beian_info.${index}.icon_url`"
                  v-model="beian.icon_url"
                  edit-type="url"
                  label="图标URL（可选）"
                />
              </div>
              <div v-if="editableBeianInfo.length === 0" class="empty-beian">
                <a-button type="dashed" @click="addBeianItem">+ 添加备案信息</a-button>
              </div>
            </template>
            <template v-else>
              <!-- 当备案信息大于4条时，分两行显示 -->
              <template v-if="beianInfoList.length > 4">
                <div class="beian-row beian-row-first">
                  <a
                    v-for="(beian, index) in beianInfoList.slice(0, 3)"
                    :key="index"
                    :href="beian.link_url || '#'"
                    :target="beian.link_url ? '_blank' : '_self'"
                    rel="noopener noreferrer"
                    class="beian-link"
                  >
                    <img v-if="beian.icon_url" :src="beian.icon_url" class="beian-icon" :alt="beian.number" />
                    <span>{{ beian.number }}</span>
                  </a>
                </div>
                <div class="beian-row beian-row-second">
                  <a
                    v-for="(beian, index) in beianInfoList.slice(3)"
                    :key="index + 3"
                    :href="beian.link_url || '#'"
                    :target="beian.link_url ? '_blank' : '_self'"
                    rel="noopener noreferrer"
                    class="beian-link"
                  >
                    <img v-if="beian.icon_url" :src="beian.icon_url" class="beian-icon" :alt="beian.number" />
                    <span>{{ beian.number }}</span>
                  </a>
                </div>
              </template>
              <!-- 4条或更少时，单行显示 -->
              <template v-else>
                <a
                  v-for="(beian, index) in beianInfoList"
                  :key="index"
                  :href="beian.link_url || '#'"
                  :target="beian.link_url ? '_blank' : '_self'"
                  rel="noopener noreferrer"
                  class="beian-link"
                >
                  <img v-if="beian.icon_url" :src="beian.icon_url" class="beian-icon" :alt="beian.number" />
                  <span>{{ beian.number }}</span>
                </a>
              </template>
            </template>
          </div>
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
  copyrightText?: string
  beianInfoList?: Array<{ number: string; icon_url?: string; link_url?: string }>
}>()

const isInternalLink = (url?: string) => {
  const value = String(url || "").trim()
  return value.startsWith("/")
}

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

// 编辑模式下的 beian_info 数组
const editableBeianInfo = computed({
  get: () => props.beianInfoList || [],
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

// Beian Info 数组操作
const addBeianItem = () => {
  cmsEditContext?.addArrayItem('footer.beian_info', {
    number: '新备案号',
    link_url: '',
    icon_url: ''
  })
}

const removeBeianItem = (index: number) => {
  cmsEditContext?.removeArrayItem('footer.beian_info', index)
}
</script>

<style scoped>
.empty-social,
.empty-sections,
.empty-badges,
.empty-beian {
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

/* 覆盖父级 footer-bottom 布局 - 改为三列网格布局 */
.footer-bottom {
  display: grid !important;
  grid-template-columns: 1fr auto 1fr;
  align-items: center;
  gap: 24px;
}

/* 备案信息样式 */
.footer-bottom-left {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.copyright-text {
  margin: 0;
  font-size: 14px;
  color: var(--color-text-muted);
}

.footer-bottom-center {
  display: flex;
  justify-content: center;
  align-items: center;
}

.footer-bottom-right {
  display: flex;
  gap: 16px;
  justify-content: flex-end;
}

.beian-info {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 16px;
  align-items: center;
  justify-content: center;
}

.beian-info.beian-multiline {
  flex-direction: column;
  gap: 8px;
}

.beian-row {
  display: flex;
  gap: 16px;
  align-items: center;
  justify-content: center;
}

.beian-item-edit {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 8px;
  border: 1px dashed var(--color-border);
  border-radius: 4px;
  flex: 0 0 auto;
}

.beian-link {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--color-text-muted);
  text-decoration: none;
  transition: color 0.2s;
  white-space: nowrap;
  flex: 0 0 auto;
}

.beian-link:hover {
  color: var(--color-primary-light);
}

.beian-icon {
  width: 16px;
  height: 16px;
  object-fit: contain;
  vertical-align: middle;
}

/* 响应式 - 移动端居中显示 */
@media (max-width: 768px) {
  .footer-bottom {
    grid-template-columns: 1fr !important;
    gap: 16px;
    text-align: center;
  }

  .footer-bottom-left,
  .footer-bottom-center,
  .footer-bottom-right {
    justify-content: center;
  }
}
</style>
