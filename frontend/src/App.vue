<template>
  <Maintenance v-if="site.maintenanceMode" :message="site.maintenanceMessage" />
  <router-view v-else v-slot="{ Component, route }">
    <transition name="fade-slide" mode="out-in">
      <ErrorBoundary>
        <component :is="Component" :key="route.matched[0]?.path || route.path" />
      </ErrorBoundary>
    </transition>
  </router-view>
</template>

<script setup lang="ts">
import { onMounted, watch } from "vue";
import Maintenance from "@/pages/public/Maintenance.vue";
import ErrorBoundary from "@/components/ErrorBoundary.vue";
import { useSiteStore } from "@/stores/site";
import defaultFaviconUrl from "@/assets/brand/default-favicon.svg";

const site = useSiteStore();

onMounted(async () => {
  site.setLang();
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
