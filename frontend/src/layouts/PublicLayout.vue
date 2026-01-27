<template>
  <a-config-provider
    :theme="{
      algorithm: theme.darkAlgorithm,
      token: {
        colorText: '#f1f5f9',
        colorTextSecondary: '#94a3b8',
        colorTextTertiary: '#64748b',
        colorBgContainer: '#111827',
        colorBorder: '#1e293b',
        colorPrimary: '#0ea5e9'
      }
    }"
  >
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
    />

    </div>
  </a-config-provider>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, reactive, computed } from 'vue'
import { ConfigProvider, theme } from 'ant-design-vue'
import { useAuthStore } from "@/stores/auth"
import { useSiteStore } from "@/stores/site"
import { getCmsBlocks } from "@/services/user"
import FooterBlock from "@/components/cms/blocks/FooterBlock.vue"
import SiteLogoMedia from "@/components/brand/SiteLogoMedia.vue"

const auth = useAuthStore()
const site = useSiteStore()

const isScrolled = ref(false)
const headerNavItems = computed(() => site.headerNavItems.filter((x) => x.label && x.url))

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

const defaultFooter = {
  description: "专业的云服务提供商，为企业提供可靠、安全、高性能的云计算解决方案",
  sections: [
    {
      title: "产品服务",
      links: [
        { label: "云服务器", url: "/products" },
        { label: "对象存储", url: "/products" },
        { label: "云数据库", url: "/products" },
        { label: "CDN加速", url: "/products" }
      ]
    },
    {
      title: "资源中心",
      links: [
        { label: "开发文档", url: "/docs" },
        { label: "帮助中心", url: "/help" },
        { label: "产品公告", url: "/announcements" },
        { label: "教程指南", url: "/tutorials" }
      ]
    },
    {
      title: "客户支持",
      links: [
        { label: "帮助中心", url: "/help" },
        { label: "提交工单", url: "/console/tickets" },
        { label: "联系我们", url: "#" },
        { label: "服务状态", url: "#" }
      ]
    },
    {
      title: "关于我们",
      links: [
        { label: "关于我们", url: "#" },
        { label: "加入我们", url: "#" },
        { label: "隐私政策", url: "#" },
        { label: "服务条款", url: "#" }
      ]
    }
  ],
  badges: ["99.99% Uptime", "SOC2 Certified"]
}

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
  if (content.description) footerContent.description = content.description
  if (Array.isArray(content.social_links)) footerContent.social_links = content.social_links
  if (Array.isArray(content.sections)) footerContent.sections = content.sections
  if (Array.isArray(content.badges)) footerContent.badges = content.badges
}

const footerSections = computed(() =>
  footerContent.sections && footerContent.sections.length > 0 ? footerContent.sections : defaultFooter.sections
)

const footerBadges = computed(() =>
  footerContent.badges && footerContent.badges.length > 0 ? footerContent.badges : defaultFooter.badges
)

const isExternalLink = (url) => /^https?:\/\//i.test(url)

const handleScroll = () => {
  isScrolled.value = window.scrollY > 20
}

onMounted(async () => {
  window.addEventListener('scroll', handleScroll)
  footerContent.description = defaultFooter.description
  footerContent.sections = defaultFooter.sections
  footerContent.badges = defaultFooter.badges
  try {
    const res = await getCmsBlocks({ page: "footer", lang: site.currentLang || "zh-CN" })
    const items = res.data?.items || []
    const footerBlock = items.find((item) => item.type === "footer")
    applyFooterBlock(footerBlock)
  } catch (error) {
    // fallback to defaults
  }
})

onUnmounted(() => {
  window.removeEventListener('scroll', handleScroll)
})
</script>

<style>
@import url('https://fonts.googleapis.com/css2?family=Outfit:wght@400;500;600;700&family=Work+Sans:wght@300;400;500;600&display=swap');

/* CSS Variables */
:root {
  --color-bg: #0a0e17;
  --color-bg-alt: #111827;
  --color-header-bg: rgba(10, 14, 23, 0.8);
  --color-primary: #0ea5e9;
  --color-primary-light: #38bdf8;
  --color-text: #f1f5f9;
  --color-text-muted: #94a3b8;
  --color-border: #1e293b;
  --font-heading: 'Outfit', sans-serif;
  --font-body: 'Work Sans', sans-serif;
}

.public-layout {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  background: var(--color-bg);
}
</style>

<style>
/* Global styles for public layout - not scoped */
.public-layout {
  color: #f1f5f9;
}

.public-layout .ant-typography,
.public-layout .ant-text,
.public-layout .ant-form-item-label > label,
.public-layout .ant-form-item-explain-error {
  color: inherit;
}

.public-layout :deep(.ant-typography) {
  color: #f1f5f9;
}

.public-layout :deep(.ant-text-secondary) {
  color: #94a3b8;
}

.public-layout :deep(.ant-form-item-label > label) {
  color: #f1f5f9;
}

.public-layout :deep(.ant-form-item-explain) {
  color: #94a3b8;
}

.public-layout :deep(.ant-empty-description) {
  color: #94a3b8;
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
  background: linear-gradient(135deg, var(--color-primary) 0%, #0284c7 100%);
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
  background: linear-gradient(135deg, var(--color-primary) 0%, #0284c7 100%);
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
  background: linear-gradient(135deg, var(--color-primary) 0%, #0284c7 100%);
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

.footer-bottom-left p {
  font-size: 14px;
  color: var(--color-text-muted);
  margin: 0;
}

.footer-bottom-right {
  display: flex;
  gap: 16px;
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
</style>



