<template>
  <svg
    :width="size"
    :height="size"
    viewBox="0 0 64 64"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
    role="img"
    aria-label="OpenIDC"
  >
    <defs>
      <mask :id="maskId" maskUnits="userSpaceOnUse" x="0" y="0" width="64" height="64">
        <rect width="64" height="64" fill="#fff" />
        <circle cx="32" cy="35.5" r="7.6" fill="#000" />
      </mask>
    </defs>
    <g :mask="`url(#${maskId})`" fill="currentColor">
      <rect x="13" y="27" width="38" height="20" rx="10" />
      <circle cx="21" cy="29" r="8.6" />
      <circle cx="32" cy="24.5" r="11" />
      <circle cx="43" cy="29" r="8.8" />
    </g>
    <path
      d="M48.5 11.8l1.6 4.2 4.2 1.6-4.2 1.6-1.6 4.2-1.6-4.2-4.2-1.6 4.2-1.6 1.6-4.2z"
      fill="currentColor"
      opacity="0.9"
    />
  </svg>
</template>

<script setup lang="ts">
import { getCurrentInstance } from "vue";

withDefaults(
  defineProps<{
    size?: number;
  }>(),
  {
    size: 22
  }
);

const maskId = `openidc-mark-${getCurrentInstance()?.uid ?? "base"}`;
</script>
