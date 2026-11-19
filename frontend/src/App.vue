<template>
  <a-config-provider :locale="antdLocale" :direction="direction" :theme="{ algorithm: app.isDarkMode ? theme.darkAlgorithm : theme.defaultAlgorithm, token: app.themeTokens }">
  <Maintenance v-if="site.maintenanceMode" :message="site.maintenanceMessage" />
  <router-view v-else v-slot="{ Component, route }">
    <transition name="fade-slide" mode="out-in">
      <ErrorBoundary>
        <component :is="Component" :key="route.matched[0]?.path || route.path" />
      </ErrorBoundary>
    </transition>
  </router-view>
  </a-config-provider>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, watch } from "vue";
import { theme } from "ant-design-vue";
import { useAppStore } from "@/stores/app";
import { antdLocale, direction, initLocale, locale } from "@/locales";
import Maintenance from "@/pages/public/Maintenance.vue";
import ErrorBoundary from "@/components/ErrorBoundary.vue";
import { useSiteStore } from "@/stores/site";
import defaultFaviconUrl from "@/assets/brand/default-favicon.svg";

const site = useSiteStore();
const app = useAppStore();
const disposeTheme = app.initConsoleTheme();
const disposeLocale = initLocale();
watch(locale, () => site.syncLocale(), { immediate: true, flush: 'sync' });
onUnmounted(() => { disposeTheme(); disposeLocale(); });

onMounted(async () => {
  await site.fetchSettings();
});

const setFavicon = (href?: string) => {
  const url = (href || "").trim() || defaultFaviconUrl;
  let link = document.querySelector("link[rel~='icon']") as HTMLLinkElement | null;
  if (!link) {
    link = document.createElement("link");
    link.rel = "icon";
    document.head.appendChild(link);
  }
  link.href = url;
};

watch(
  () => site.siteName,
  (name) => {
    if (name) {
      document.title = name;
    }
  },
  { immediate: true }
);

watch(
  () => site.faviconUrl,
  (url) => setFavicon(url),
  { immediate: true }
);
</script>
