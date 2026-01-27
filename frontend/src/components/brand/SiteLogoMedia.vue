<template>
  <img
    v-if="logoSrc"
    :src="logoSrc"
    :alt="altText"
    class="site-logo-media site-logo-img"
    :style="mediaStyle"
    decoding="async"
    loading="eager"
  />
  <DefaultLogoMark v-else class="site-logo-media site-logo-mark" :size="size" :style="mediaStyle" />
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useSiteStore } from "@/stores/site";
import DefaultLogoMark from "@/components/brand/DefaultLogoMark.vue";

const props = withDefaults(
  defineProps<{
    size?: number;
    alt?: string;
    src?: string;
  }>(),
  {
    size: 22,
    alt: "",
    src: ""
  }
);

const site = useSiteStore();
const altText = computed(() => props.alt || site.siteName || "logo");
const logoSrc = computed(() => (props.src || "").trim() || site.logoUrl);
const mediaStyle = computed(() => ({
  width: `${props.size}px`,
  height: `${props.size}px`
}));
</script>

<style scoped>
.site-logo-media {
  display: block;
}

.site-logo-img {
  object-fit: contain;
}
</style>
