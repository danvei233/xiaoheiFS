<!-- 授权页右上角组件 -->
<template>
  <div
    class="absolute w-full flex-cb top-4.5 z-10 flex-c !justify-end max-[1180px]:!justify-between"
  >
    <div class="flex-cc !hidden max-[1180px]:!flex ml-2 max-sm:ml-6">
      <ArtLogo class="icon" size="46" />
      <h1 class="text-xl ont-mediumf ml-2">{{ systemName }}</h1>
    </div>

    <div class="mr-2 max-sm:mr-5">
      <ThemePreferenceControls />
    </div>
  </div>
</template>

<script setup lang="ts">
  import { useSiteStore } from '@/store/modules/site'
  import ThemePreferenceControls from '@/components/core/layouts/art-header-bar/widget/ThemePreferenceControls.vue'

  defineOptions({ name: 'AuthTopBar' })

  const siteStore = useSiteStore()
  const systemName = computed(() => siteStore.resolvedSiteName)
</script>
