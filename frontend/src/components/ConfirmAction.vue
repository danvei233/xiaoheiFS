<template>
  <a-popconfirm :title="title" @confirm="$emit('confirm')">
    <slot />
  </a-popconfirm>
</template>

<script setup>
defineProps({
  title: { type: String, default: "确认操作?" }
});

defineEmits(["confirm"]);
</script>
