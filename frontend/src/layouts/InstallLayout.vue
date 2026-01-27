<template>
  <a-config-provider
    :theme="{
      algorithm: theme.defaultAlgorithm,
      token: {
        colorText: 'rgba(0,0,0,0.88)',
        colorTextSecondary: 'rgba(0,0,0,0.58)',
        colorTextTertiary: 'rgba(0,0,0,0.45)',
        colorBgContainer: '#ffffff',
        colorBorder: 'rgba(0,0,0,0.08)',
        colorPrimary: '#1677ff'
      }
    }"
  >
    <router-view />
  </a-config-provider>
</template>

<script setup lang="ts">
import { ConfigProvider, theme } from "ant-design-vue";
</script>

