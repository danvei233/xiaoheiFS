<template>
  <svg
    :width="size"
    :height="size"
    viewBox="0 0 64 64"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
    role="img"
    aria-label="Logo"
  >
    <path
      d="M22.8 46.5H44.9C52.1 46.5 58 40.7 58 33.5C58 27.2 53.5 21.9 47.4 20.7C46 13.8 39.9 8.6 32.6 8.6C24.2 8.6 17.3 15.5 17.3 23.9C17.3 24.4 17.3 24.9 17.4 25.4C12.1 27.2 8.4 32.3 8.4 38.2C8.4 43 12.3 46.5 17.2 46.5H22.8Z"
      fill="currentColor"
      fill-rule="evenodd"
      clip-rule="evenodd"
    />
    <path
      d="M46.8 13.6L48 17.1L51.5 18.3L48 19.5L46.8 23L45.6 19.5L42.1 18.3L45.6 17.1L46.8 13.6Z"
      fill="currentColor"
      opacity="0.92"
    />
    <path
      d="M25 31.2C25 28.7 27 26.7 29.5 26.7H37.8C40.3 26.7 42.3 28.7 42.3 31.2C42.3 33.7 40.3 35.7 37.8 35.7H29.5C27 35.7 25 33.7 25 31.2Z"
      fill="white"
      opacity="0.18"
    />
  </svg>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    size?: number;
  }>(),
  {
    size: 22
  }
);
</script>

