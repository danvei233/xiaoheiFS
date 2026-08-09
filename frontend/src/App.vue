<template>
  <!-- 维护模式下整站替换为维护页（功能不变量 #7） -->
  <MaintenancePage v-if="site.maintenanceMode" :message="site.maintenanceMessage" />
  <RouterView v-else v-slot="{ Component, route }">
    <Transition name="fade-slide" mode="out-in">
      <ErrorBoundary>
        <component :is="Component" :key="route.matched[0]?.path || route.path" />
      </ErrorBoundary>
    </Transition>
  </RouterView>
</template>

<script setup lang="ts">
  import ErrorBoundary from '@/components/business/error-boundary/index.vue'
  import { useSiteStore } from '@/stores/site'
  import defaultFaviconUrl from '@/assets/brand/default-favicon.svg'

  defineOptions({ name: 'App' })

  const MaintenancePage = defineAsyncComponent(() => import('@/views/public/maintenance/index.vue'))
  const site = useSiteStore()

  onMounted(async () => {
    site.setLang()
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

  watch(
    () => site.siteName,
    (name) => {
      if (name) document.title = name
    },
    { immediate: true }
  )

  watch(
    () => site.faviconUrl,
    (url) => setFavicon(url),
    { immediate: true }
  )
</script>
