<template>
  <!-- 维护模式下整站替换为维护页（功能不变量 #7） -->
  <MaintenancePage v-if="site.maintenanceMode" :message="site.maintenanceMessage" />
  <RouterView v-else v-slot="{ Component, route }">
    <Transition name="fade-slide" mode="out-in">
      <!-- web_bundle_mobile F1：key 只用布局域路径，不再用 route.fullPath。
           旧实现每次路由切换都换 key → ErrorBoundary 连同整棵子树卸载重建，
           meta.keepAlive 缓存全部失效、每个列表页重新拉数据、整壳过渡动画。
           错误兜底状态改由 ErrorBoundary 内部 watch route.fullPath 复位
           （P2-79 语义保留）。 -->
      <ErrorBoundary :key="route.matched[0]?.path || route.path">
        <component :is="Component" :key="route.matched[0]?.path || route.path" />
      </ErrorBoundary>
    </Transition>
  </RouterView>
</template>

<script setup lang="ts">
  import ErrorBoundary from '@/components/business/error-boundary/index.vue'
  import { useSiteStore } from '@/stores/site'
  import defaultFaviconUrl from '@/assets/brand/default-favicon.svg'
  import { useI18n } from 'vue-i18n'
  import { LanguageEnum } from '@/enums/appEnum'
  import { refreshDocumentTitle } from '@/utils/router'

  defineOptions({ name: 'App' })

  const MaintenancePage = defineAsyncComponent(() => import('@/views/public/maintenance/index.vue'))
  const site = useSiteStore()
  const { locale } = useI18n()

  // P2-80 / m065 #2：i18n locale（zh/en）与站点 CMS 语言（zh-CN/en-US）双轨对齐。
  // 只认这两档，非标准浏览器语言（en-GB/zh-TW）不再直接把 navigator.language
  // 塞给后端——否则 CMS blocks/posts 按 zh-CN/en-US 精确匹配会拿空。
  watch(
    locale,
    (val) => {
      site.setLang(val === LanguageEnum.EN ? 'en-US' : 'zh-CN')
    },
    { immediate: true }
  )

  onMounted(async () => {
    await site.fetchSettings()
  })

  // 站点白标：标题与 favicon 跟随后台设置（功能不变量 #10）
  const setFavicon = (href?: string) => {
    const url = (href || '').trim() || defaultFaviconUrl
    let link = document.querySelector("link[rel~='icon']") as HTMLLinkElement | null
    if (!link) {
      link = document.createElement('link')
      link.rel = 'icon'
      document.head.appendChild(link)
    }
    link.href = url
  }

  // 站点品牌加载完成后重拼"页名 - 品牌"标题（P3-51 / m065 R-5）。
  // 不能直接 document.title = siteName（会把页面名丢掉）；也不能用
  // MutationObserver 逐次替换（品牌含字面 " - " 时会无限循环）。
  watch(
    () => site.siteName,
    () => {
      refreshDocumentTitle()
    },
    { immediate: true }
  )

  watch(
    () => site.faviconUrl,
    (url) => setFavicon(url),
    { immediate: true }
  )
</script>
