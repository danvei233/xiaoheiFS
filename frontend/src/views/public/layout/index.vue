<template>
  <div class="public-layout">
    <header :class="['public-header', { 'is-scrolled': isScrolled }]">
      <div class="public-header__inner">
        <RouterLink to="/" class="public-header__brand" aria-label="返回首页">
          <SiteLogoMedia :size="30" />
          <span>{{ siteName }}</span>
        </RouterLink>

        <nav class="public-header__nav" aria-label="主导航">
          <RouterLink to="/" class="public-header__link">首页</RouterLink>
          <template v-for="(item, index) in navItems" :key="navKey(item, index)">
            <RouterLink
              v-if="isInternal(item.url) && item.target !== '_blank'"
              :to="item.url"
              class="public-header__link"
            >
              {{ item.label }}
            </RouterLink>
            <a
              v-else
              :href="safeHref(item.url)"
              :target="item.target || '_self'"
              rel="noopener noreferrer"
              class="public-header__link"
            >
              {{ item.label }}
            </a>
          </template>
        </nav>

        <div class="public-header__actions">
          <ElTooltip content="切换主题" placement="bottom">
            <ElButton text circle aria-label="切换主题" @click="toggleTheme">
              <ArtSvgIcon :icon="isDark ? 'ri:sun-line' : 'ri:moon-line'" />
            </ElButton>
          </ElTooltip>

          <ElBadge :value="cartCount" :hidden="cartCount === 0" :max="99">
            <ElTooltip content="购物车" placement="bottom">
              <ElButton text circle aria-label="购物车" @click="router.push('/cart')">
                <ArtSvgIcon icon="ri:shopping-cart-2-line" />
              </ElButton>
            </ElTooltip>
          </ElBadge>

          <template v-if="isLoggedIn">
            <ElButton type="primary" @click="router.push('/console')">控制台</ElButton>
          </template>
          <template v-else>
            <ElButton text class="public-header__desktop-action" @click="router.push('/login')"
              >登录</ElButton
            >
            <ElButton
              type="primary"
              class="public-header__desktop-action"
              @click="router.push('/register')"
              >注册</ElButton
            >
          </template>

          <ElButton
            text
            circle
            class="public-header__menu"
            aria-label="打开导航"
            @click="drawerVisible = true"
          >
            <ArtSvgIcon icon="ri:menu-3-line" />
          </ElButton>
        </div>
      </div>
    </header>

    <main class="public-main">
      <RouterView v-slot="{ Component }">
        <Transition name="public-fade" mode="out-in">
          <component :is="Component" />
        </Transition>
      </RouterView>
    </main>

    <!-- P3-45：渲染 CMS 管理员配置的 custom_html 公开区块（受信管理内容）。 -->
    <div
      v-for="block in customHtmlBlocks"
      :key="String(block.id ?? block.title ?? block.custom_html ?? 'custom-html')"
      class="public-custom-html"
      v-html="block.custom_html"
    ></div>

    <FooterBlock
      :site-name="siteName"
      :logo-url="siteStore.logoUrl"
      :content="footerContent"
      :sections="footerSections"
      :badges="footerBadges"
      :copyright-text="copyrightText"
      :beian-info-list="beianInfoList"
    />

    <ElDrawer v-model="drawerVisible" direction="rtl" size="min(84vw, 340px)" :with-header="false">
      <div class="public-drawer">
        <div class="public-drawer__header">
          <RouterLink to="/" class="public-header__brand" @click="drawerVisible = false">
            <SiteLogoMedia :size="30" />
            <span>{{ siteName }}</span>
          </RouterLink>
          <ElButton text circle aria-label="关闭导航" @click="drawerVisible = false">
            <ArtSvgIcon icon="ri:close-line" />
          </ElButton>
        </div>

        <nav class="public-drawer__nav" aria-label="移动端导航">
          <RouterLink to="/" @click="drawerVisible = false">首页</RouterLink>
          <template v-for="(item, index) in navItems" :key="navKey(item, index)">
            <RouterLink
              v-if="isInternal(item.url) && item.target !== '_blank'"
              :to="item.url"
              @click="drawerVisible = false"
            >
              {{ item.label }}
            </RouterLink>
            <a
              v-else
              :href="safeHref(item.url)"
              :target="item.target || '_self'"
              rel="noopener noreferrer"
            >
              {{ item.label }}
            </a>
          </template>
        </nav>

        <div class="public-drawer__actions">
          <ElButton v-if="isLoggedIn" type="primary" @click="navigateFromDrawer('/console')"
            >控制台</ElButton
          >
          <template v-else>
            <ElButton @click="navigateFromDrawer('/login')">登录</ElButton>
            <ElButton type="primary" @click="navigateFromDrawer('/register')">注册</ElButton>
          </template>
        </div>
      </div>
    </ElDrawer>
  </div>
</template>

<script setup lang="ts">
  import { SystemThemeEnum } from '@/enums/appEnum'
  import { useTheme } from '@/hooks/core/useTheme'
  import { useSettingStore } from '@/store/modules/setting'
  import { useAuthStore } from '@/stores/auth'
  import { useCartStore } from '@/stores/cart'
  import { useSiteStore, type SiteNavItem } from '@/stores/site'
  import FooterBlock from '@/components/business/cms-blocks/FooterBlock.vue'
  import SiteLogoMedia from '@/components/brand/SiteLogoMedia.vue'
  import { safeHref } from '@/utils/urlPolicy'

  defineOptions({ name: 'PublicLayout' })

  type FooterContent = {
    description?: string
    social_links?: Array<{ key?: string; url?: string }>
    sections?: FooterSection[]
    badges?: string[]
    beian_info?: BeianInfo[]
  }
  type FooterSection = { title?: string; links: Array<{ label?: string; url?: string }> }
  type BeianInfo = { number: string; icon_url?: string; link_url?: string }

  const router = useRouter()
  const route = useRoute()
  const settingStore = useSettingStore()
  const { setSystemTheme } = useTheme()
  const siteStore = useSiteStore()
  const cartStore = useCartStore()
  const authStore = useAuthStore()

  const drawerVisible = ref(false)
  const isScrolled = ref(false)
  const isDark = computed(() => settingStore.isDark)
  const siteName = computed(() => siteStore.siteName || '小黑云')
  const navItems = computed(() => siteStore.headerNavItems.filter((item) => item.label && item.url))
  const isLoggedIn = computed(() => Boolean(authStore.token))
  const cartCount = computed(() =>
    (cartStore.items || []).reduce(
      (total: number, item: any) => total + Number(item.qty ?? item.Qty ?? 1),
      0
    )
  )

  const defaultFooter: Required<Pick<FooterContent, 'description' | 'sections' | 'badges'>> = {
    description: '专业的云服务提供商，为企业提供可靠、安全、高性能的云计算解决方案',
    sections: [
      {
        title: '产品服务',
        links: [
          { label: '云服务器', url: '/products' },
          { label: '弹性计算', url: '/products' }
        ]
      },
      {
        title: '资源中心',
        links: [
          { label: '开发文档', url: '/docs' },
          { label: '教程指南', url: '/tutorials' }
        ]
      },
      {
        title: '客户支持',
        links: [
          { label: '帮助中心', url: '/help' },
          { label: '提交工单', url: '/console/tickets' }
        ]
      },
      {
        title: '站点信息',
        links: [
          { label: '产品公告', url: '/announcements' },
          { label: '活动中心', url: '/activities' }
        ]
      }
    ],
    badges: ['99.99% Uptime', 'Secure Service']
  }

  const parseSettingJson = <T,>(value: unknown, fallback: T): T => {
    if (!value) return fallback
    if (typeof value !== 'string') return value as T
    try {
      return JSON.parse(value) as T
    } catch {
      return fallback
    }
  }

  const footerBlockContent = computed<FooterContent>(() => {
    const block = (siteStore.blocks.footer || []).find(
      (item: any) => item?.type === 'footer'
    ) as any
    if (!block) return {}
    if (block.content && typeof block.content === 'object') return block.content
    return parseSettingJson<FooterContent>(block.content_json, {})
  })

  const footerContent = computed(() => ({
    description: footerBlockContent.value.description || defaultFooter.description,
    social_links: footerBlockContent.value.social_links || []
  }))
  const footerSections = computed(() =>
    footerBlockContent.value.sections?.length
      ? footerBlockContent.value.sections
      : defaultFooter.sections
  )
  const footerBadges = computed(() =>
    footerBlockContent.value.badges?.length ? footerBlockContent.value.badges : defaultFooter.badges
  )
  const copyrightText = computed(() => String(siteStore.settings.copyright_text || ''))
  const beianInfoList = computed<BeianInfo[]>(() =>
    footerBlockContent.value.beian_info?.length
      ? footerBlockContent.value.beian_info
      : parseSettingJson<BeianInfo[]>(siteStore.settings.beian_info_list, [])
  )

  // P3-45：custom_html 类型的 CMS 公开区块此前从未被渲染（死特性）。这里在
  // 公共布局里按「当前页 + footer 页」的区块渲染 v-html。custom_html 是 CMS
  // 管理员显式编写的 HTML（与 posts/detail.vue 渲染 content_html 同源），
  // 属于受信管理内容，不作为用户输入渲染。
  const currentPageKey = computed(() => {
    const path = route.path
    if (path === '/' || path === '/home') return 'home'
    return String(path.split('/').filter(Boolean)[0] || 'home')
  })
  const customHtmlBlocks = computed(() => {
    const current = (siteStore.blocks[currentPageKey.value] || []) as Array<Record<string, any>>
    const footer = (siteStore.blocks.footer || []) as Array<Record<string, any>>
    return [...footer, ...current]
      .filter(
        (block) => block?.type === 'custom_html' && block?.visible !== false && block?.custom_html
      )
      .sort((a, b) => Number(a.sort_order || 0) - Number(b.sort_order || 0))
  })

  const isInternal = (url: string) =>
    String(url || '')
      .trim()
      .startsWith('/')
  const navKey = (item: SiteNavItem, index: number) =>
    `${item.lang || 'all'}-${item.label}-${item.url}-${index}`
  const toggleTheme = () =>
    setSystemTheme(isDark.value ? SystemThemeEnum.LIGHT : SystemThemeEnum.DARK)
  const handleScroll = () => {
    isScrolled.value = window.scrollY > 12
  }
  const navigateFromDrawer = (path: string) => {
    drawerVisible.value = false
    router.push(path)
  }

  onMounted(async () => {
    window.addEventListener('scroll', handleScroll, { passive: true })
    await siteStore.fetchSettings()
    await siteStore.fetchBlocks('footer')
  })

  onUnmounted(() => window.removeEventListener('scroll', handleScroll))
</script>

<style lang="scss" scoped>
  .public-layout {
    display: flex;
    flex-direction: column;
    min-height: 100vh;
    color: var(--art-gray-800);
    background: var(--default-bg-color);
  }

  .public-header {
    position: sticky;
    top: 0;
    z-index: 100;
    background: color-mix(in srgb, var(--default-box-color) 94%, transparent);
    backdrop-filter: blur(18px);
    border-bottom: 1px solid var(--art-card-border);
    transition: box-shadow 0.2s ease;
  }

  .public-header.is-scrolled {
    box-shadow: 0 10px 30px color-mix(in srgb, var(--art-gray-900) 7%, transparent);
  }

  .public-header__inner {
    display: flex;
    gap: 28px;
    align-items: center;
    max-width: 1240px;
    height: 66px;
    padding: 0 24px;
    margin: 0 auto;
  }

  .public-header__brand {
    display: inline-flex;
    flex: 0 0 auto;
    gap: 10px;
    align-items: center;
    min-width: 0;
    color: var(--art-gray-900);
    text-decoration: none;
  }

  .public-header__brand span {
    max-width: 190px;
    overflow: hidden;
    font-size: 17px;
    font-weight: 680;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .public-header__nav {
    display: flex;
    flex: 1;
    gap: 4px;
    align-items: center;
    min-width: 0;
  }

  .public-header__link {
    padding: 8px 10px;
    color: var(--art-gray-600);
    text-decoration: none;
    border-radius: calc(var(--custom-radius) / 2 + 1px);
    transition:
      color 0.2s ease,
      background 0.2s ease;
  }

  .public-header__link:hover,
  .public-header__link.router-link-exact-active {
    color: var(--theme-color);
    background: color-mix(in srgb, var(--theme-color) 8%, var(--default-box-color));
  }

  .public-header__actions {
    display: flex;
    flex: 0 0 auto;
    gap: 6px;
    align-items: center;
  }

  .public-header__menu {
    display: none;
  }

  .public-main {
    flex: 1;
    min-width: 0;
  }

  .public-drawer {
    display: flex;
    flex-direction: column;
    height: 100%;
  }

  .public-drawer__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding-bottom: 20px;
    border-bottom: 1px solid var(--default-border);
  }

  .public-drawer__nav {
    display: flex;
    flex: 1;
    flex-direction: column;
    gap: 5px;
    padding: 20px 0;
  }

  .public-drawer__nav a {
    padding: 12px 14px;
    font-size: 16px;
    color: var(--art-gray-700);
    text-decoration: none;
    border-radius: calc(var(--custom-radius) / 2 + 2px);
  }

  .public-drawer__nav a:hover,
  .public-drawer__nav .router-link-exact-active {
    color: var(--theme-color);
    background: var(--art-hover-color);
  }

  .public-drawer__actions {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 10px;
    padding-top: 18px;
    border-top: 1px solid var(--default-border);
  }

  .public-drawer__actions > :only-child {
    grid-column: 1 / -1;
  }

  .public-fade-enter-active,
  .public-fade-leave-active {
    transition:
      opacity 0.18s ease,
      transform 0.18s ease;
  }

  .public-fade-enter-from,
  .public-fade-leave-to {
    opacity: 0;
    transform: translateY(6px);
  }

  @media (width <= 900px) {
    .public-header__nav,
    .public-header__desktop-action {
      display: none;
    }

    .public-header__inner {
      padding-inline: 18px;
    }

    .public-header__actions {
      margin-left: auto;
    }

    .public-header__menu {
      display: inline-flex;
    }
  }

  @media (width <= 440px) {
    .public-header__brand span {
      max-width: 110px;
    }
  }
</style>
