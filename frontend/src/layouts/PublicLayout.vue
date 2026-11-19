<template>
    <div class="public-layout">
      <!-- Header -->
      <header class="public-header" :class="{ 'scrolled': isScrolled }">
      <div class="header-container">
        <div class="header-left">
          <router-link to="/" class="logo">
            <div class="logo-icon">
              <SiteLogoMedia :size="20" />
            </div>
            <span class="logo-text">{{ site.siteName }}</span>
          </router-link>
          <nav class="main-nav">
            <router-link to="/">{{ $t("nav.home") || "首页" }}</router-link>
            <template v-for="(item, idx) in headerNavItems" :key="`${item.lang || 'all'}-${item.label}-${item.url}-${idx}`">
              <router-link v-if="isInternal(item.url) && item.target !== '_blank'" :to="item.url">
                {{ item.label }}
              </router-link>
              <a
                v-else
                :href="item.url"
                :target="item.target || '_self'"
                rel="noopener noreferrer"
              >
                {{ item.label }}
              </a>
            </template>
          </nav>
        </div>
        <div class="header-right">
          <div class="header-actions">
            <AppearanceSettings />
            <router-link v-if="!auth.token" to="/login" class="btn btn-ghost">
              {{ $t("auth.login") || "登录" }}
            </router-link>
            <router-link v-if="!auth.token" to="/register" class="btn btn-primary">
              {{ $t("auth.register") || "注册" }}
            </router-link>
            <router-link v-if="auth.token" to="/console" class="btn btn-primary">
              <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
                <path d="M2 6L8 2L14 6V13C14 13.5304 13.7893 14.0391 13.4142 14.4142C13.0391 14.7893 12.5304 15 12 15H4C3.46957 15 2.96086 14.7893 2.58579 14.4142C2.21071 14.0391 2 13.5304 2 13V6Z" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
                <path d="M6 15V9H10V15" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
              </svg>
              <span>{{ $t("nav.console") || "控制台" }}</span>
            </router-link>
          </div>
        </div>
      </div>
    </header>

    <!-- Main Content -->
    <main class="public-main">
      <router-view />
    </main>

    <!-- Footer -->
    <FooterBlock
      :site-name="site.siteName"
      :logo-url="site.logoUrl"
      :content="footerContent"
      :sections="footerSections"
      :badges="footerBadges"
      :copyright-text="copyrightText"
      :beian-info-list="beianInfoList"
    />

    </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, reactive, computed, watch } from 'vue'
import AppearanceSettings from '@/components/AppearanceSettings.vue'
import { useAuthStore } from "@/stores/auth"
import { useSiteStore } from "@/stores/site"
import { getCmsBlocks, getSiteSettings } from "@/services/user"
import FooterBlock from "@/components/cms/blocks/FooterBlock.vue"
import SiteLogoMedia from "@/components/brand/SiteLogoMedia.vue"
import { t } from "@/locales"

const auth = useAuthStore()
const site = useSiteStore()

const isScrolled = ref(false)
const headerNavItems = computed(() => site.headerNavItems.filter((x) => x.label && x.url))
const copyrightText = ref('')
const beianInfoList = ref<Array<{ number: string; icon_url?: string; link_url?: string }>>([])

const isInternal = (url: string) => {
  const u = (url || "").trim()
  return u.startsWith("/")
}

const footerContent = reactive({
  description: "",
  social_links: [
    { key: "github", url: "#" },
    { key: "twitter", url: "#" },
    { key: "discord", url: "#" }
  ],
  sections: [],
  badges: []
})

const defaultFooterSections = () => [
  {
    title: t('footer.section.products'),
    links: [
      { label: t('footer.link.cloudServer'), url: "/products" },
      { label: t('footer.link.objectStorage'), url: "/products" },
      { label: t('footer.link.cloudDatabase'), url: "/products" },
      { label: t('footer.link.cdn'), url: "/products" }
    ]
  },
  {
    title: t('footer.section.resources'),
    links: [
      { label: t('footer.link.docs'), url: "/docs" },
      { label: t('footer.link.help'), url: "/help" },
      { label: t('footer.link.announcements'), url: "/announcements" },
      { label: t('footer.link.tutorials'), url: "/tutorials" }
    ]
  },
  {
    title: t('footer.section.support'),
    links: [
      { label: t('footer.link.help'), url: "/help" },
      { label: t('footer.link.ticket'), url: "/console/tickets" },
      { label: t('footer.link.contact'), url: "#" },
      { label: t('footer.link.serviceStatus'), url: "#" }
    ]
  },
  {
    title: t('footer.section.about'),
    links: [
      { label: t('footer.link.aboutUs'), url: "#" },
      { label: t('footer.link.joinUs'), url: "#" },
      { label: t('footer.link.privacy'), url: "#" },
      { label: t('footer.link.terms'), url: "#" }
    ]
  }
]

const parseContentJson = (raw) => {
  if (!raw) return {}
  try {
    return JSON.parse(raw)
  } catch (error) {
    return {}
  }
}

const applyFooterBlock = (block) => {
  if (!block?.content_json) return
  const content = parseContentJson(block.content_json)
  console.log('Footer block content:', content)
  if (content.description) footerContent.description = content.description
  if (Array.isArray(content.social_links)) footerContent.social_links = content.social_links
  if (Array.isArray(content.sections)) footerContent.sections = content.sections
  if (Array.isArray(content.badges)) footerContent.badges = content.badges
  // 从 CMS blocks 中获取备案信息
  if (Array.isArray(content.beian_info) && content.beian_info.length > 0) {
    beianInfoList.value = content.beian_info
    console.log('Beian info loaded from CMS blocks:', beianInfoList.value)
  } else {
    console.log('No beian_info in CMS blocks, using site settings')
  }
}

const footerSections = computed(() =>
  footerContent.sections && footerContent.sections.length > 0 ? footerContent.sections : defaultFooterSections()
)

const footerBadges = computed(() =>
  footerContent.badges && footerContent.badges.length > 0 ? footerContent.badges : [t('footer.uptime'), t('footer.secure')]
)

const isExternalLink = (url) => /^https?:\/\//i.test(url)

const handleScroll = () => {
  isScrolled.value = window.scrollY > 20
}

const fetchSiteSettings = async () => {
  try {
    const res = await getSiteSettings()
    const items = res.data?.items || []
    
    console.log('Site settings items:', items)
    
    // 将 items 数组转换为对象
    const settings = {}
    items.forEach(item => {
      settings[item.key] = item.value
    })
    
    console.log('Parsed settings:', settings)
    
    // 获取版权信息
    if (settings.copyright_text) {
      copyrightText.value = settings.copyright_text
      console.log('Copyright text loaded:', copyrightText.value)
    }
    
    // 获取备案信息列表
    if (settings.beian_info_list) {
      try {
        const parsed = typeof settings.beian_info_list === 'string' 
          ? JSON.parse(settings.beian_info_list) 
          : settings.beian_info_list
        if (Array.isArray(parsed) && parsed.length > 0) {
          beianInfoList.value = parsed
          console.log('Beian info loaded from site settings:', beianInfoList.value)
        }
      } catch (e) {
        console.error('Failed to parse beian_info_list:', e)
      }
    }
  } catch (error) {
    console.error('Failed to fetch site settings:', error)
  }
}

let footerRequest = 0
const loadFooter = async () => {
  const request = ++footerRequest
  const lang = site.currentLang
  footerContent.description = ""
  footerContent.sections = []
  footerContent.badges = []
  try {
    const res = await getCmsBlocks({ page: 'footer', lang })
    if (request !== footerRequest || lang !== site.currentLang) return
    applyFooterBlock((res.data?.items || []).find(item => item.type === 'footer'))
  } catch { /* Keep the default footer if CMS content is unavailable. */ }
}
watch(() => site.currentLang, loadFooter)

onMounted(async () => {
  window.addEventListener('scroll', handleScroll)
  footerContent.description = ""
  footerContent.sections = []
  footerContent.badges = []
  
  // 先获取站点设置（版权信息作为备用）
  await fetchSiteSettings()
  
  await loadFooter()
})

onUnmounted(() => {
  window.removeEventListener('scroll', handleScroll)
})
</script>

<style>
@import url('https://fonts.googleapis.com/css2?family=Outfit:wght@400;500;600;700&family=Work+Sans:wght@300;400;500;600&display=swap');

/* CSS Variables */
.public-layout {
  --color-bg: var(--bg-primary);
  --color-bg-alt: var(--card);
  --color-header-bg: var(--theme-surface);
  --color-primary: var(--primary);
  --color-primary-light: var(--primary);
  --color-text: var(--text-primary);
  --color-text-muted: var(--text-secondary);
  --color-border: var(--border);
  --font-heading: 'Outfit', sans-serif;
  --font-body: 'Work Sans', sans-serif;
}

.public-layout {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  background: var(--theme-canvas, var(--color-bg));
  background-size: 32px 32px;
}
</style>

<style>
/* Global styles for public layout - not scoped */
.public-layout {
  color: var(--color-text);
}

.public-layout .ant-typography,
.public-layout .ant-text,
.public-layout .ant-form-item-label > label,
.public-layout .ant-form-item-explain-error {
  color: inherit;
}

.public-layout :deep(.ant-typography) {
  color: var(--color-text);
}

.public-layout :deep(.ant-text-secondary) {
  color: var(--color-text-muted);
}

.public-layout :deep(.ant-form-item-label > label) {
  color: var(--color-text);
}

.public-layout :deep(.ant-form-item-explain) {
  color: var(--color-text-muted);
}

.public-layout :deep(.ant-empty-description) {
  color: var(--color-text-muted);
}

/* Header */
.public-header {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  z-index: 1000;
  background: transparent;
  border-bottom: 1px solid transparent;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.public-header.scrolled {
  background: var(--color-header-bg);
  border-bottom-color: rgba(30, 41, 59, 0.5);
  backdrop-filter: blur(12px);
}

.header-container {
  max-width: 1400px;
  margin: 0 auto;
  padding: 0 24px;
  height: 72px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 48px;
}

.logo {
  display: flex;
  align-items: center;
  gap: 12px;
  text-decoration: none;
  transition: opacity 0.2s;
}

.logo:hover {
  opacity: 0.8;
}

.logo-icon {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-primary);
  border-radius: 10px;
  color: white;
}

.logo-text {
  font-family: var(--font-heading);
  font-size: 20px;
  font-weight: 700;
  color: var(--color-text);
}

.logo-image {
  height: 32px;
  max-width: 140px;
  object-fit: contain;
}

.main-nav {
  display: flex;
  gap: 8px;
}

.main-nav a {
  position: relative;
  padding: 8px 16px;
  color: var(--color-text-muted);
  text-decoration: none;
  font-family: var(--font-body);
  font-size: 15px;
  font-weight: 500;
  border-radius: 8px;
  transition: all 0.2s;
}

.main-nav a:hover {
  color: var(--color-text);
  background: rgba(255, 255, 255, 0.05);
}

.main-nav a.router-link-active {
  color: var(--color-primary-light);
}

.header-right {
  display: flex;
  align-items: center;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 10px 20px;
  border-radius: 10px;
  font-family: var(--font-body);
  font-size: 14px;
  font-weight: 600;
  text-decoration: none;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  cursor: pointer;
}

.btn-ghost {
  background: transparent;
  color: var(--color-text-muted);
  border: 1px solid transparent;
}

.btn-ghost:hover {
  color: var(--color-text);
  background: rgba(255, 255, 255, 0.05);
}

.btn-primary {
  background: var(--color-primary);
  color: white;
  box-shadow: 0 2px 12px rgba(14, 165, 233, 0.3);
}

.btn-primary:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 20px rgba(14, 165, 233, 0.4);
}

/* Main */
.public-main {
  flex: 1;
  padding-top: 72px;
}

/* Footer */
.public-footer {
  background: var(--color-bg-alt);
  border-top: 1px solid var(--color-border);
  padding: 72px 0 32px;
  margin-top: auto;
}

.footer-container {
  max-width: 1400px;
  margin: 0 auto;
  padding: 0 24px;
}

.footer-top {
  display: grid;
  grid-template-columns: 320px 1fr;
  gap: 64px;
  margin-bottom: 48px;
}

.footer-brand {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.footer-logo {
  display: flex;
  align-items: center;
  gap: 12px;
}

.footer-logo .logo-icon {
  width: 40px;
  height: 40px;
  background: var(--color-primary);
  border-radius: 12px;
}

.footer-logo-text {
  font-family: var(--font-heading);
  font-size: 22px;
  font-weight: 700;
  color: var(--color-text);
}

.footer-desc {
  font-size: 15px;
  line-height: 1.7;
  color: var(--color-text-muted);
  margin: 0;
}

.footer-social {
  display: flex;
  gap: 12px;
}

.social-link {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid var(--color-border);
  border-radius: 10px;
  color: var(--color-text-muted);
  transition: all 0.3s;
}

.social-link:hover {
  background: rgba(14, 165, 233, 0.1);
  border-color: var(--color-primary);
  color: var(--color-primary-light);
  transform: translateY(-2px);
}

.footer-sections {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 32px;
}

.footer-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.footer-section h4 {
  font-family: var(--font-heading);
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text);
  margin: 0;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.footer-section a {
  font-size: 14px;
  color: var(--color-text-muted);
  text-decoration: none;
  transition: color 0.2s;
  line-height: 1.6;
}

.footer-section a:hover {
  color: var(--color-primary-light);
}

.footer-bottom {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-top: 32px;
  border-top: 1px solid var(--color-border);
}

.footer-bottom-left {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.footer-bottom-left p {
  font-size: 14px;
  color: var(--color-text-muted);
  margin: 0;
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

.footer-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  background: rgba(16, 185, 129, 0.1);
  border: 1px solid rgba(16, 185, 129, 0.2);
  border-radius: 100px;
  font-size: 12px;
  font-weight: 600;
  color: #10b981;
}

/* Responsive */
@media (max-width: 1024px) {
  .footer-top {
    grid-template-columns: 1fr;
    gap: 48px;
  }

  .footer-sections {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 768px) {
  .header-container {
    padding: 0 16px;
  }

  .header-left {
    gap: 16px;
  }

  .main-nav {
    display: none;
  }

  .footer-sections {
    grid-template-columns: 1fr;
  }

  .footer-bottom {
    flex-direction: column;
    gap: 16px;
    text-align: center;
  }
}

@media (max-width: 420px) {
  .header-container { gap: 8px; padding-inline: 10px; }
  .header-left, .logo { min-width: 0; }
  .header-left { gap: 8px; flex: 1 1 auto; overflow: hidden; }
  .logo { gap: 7px; flex: 0 1 auto; }
  .logo-text { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 15px; }
  .header-right, .header-actions { min-width: 0; flex: 0 0 auto; }
  .header-actions { gap: 5px; }
  .header-actions .btn { flex: 0 0 auto; padding: 8px 9px; white-space: nowrap; font-size: 12px; }
}
</style>



