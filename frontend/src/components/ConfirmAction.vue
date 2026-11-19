<template>
  <a-popconfirm :title="title" @confirm="$emit('confirm')">
    <slot />
  </a-popconfirm>
</template>

<script setup>
import { computed } from "vue";
import { t } from "@/locales";

const props = defineProps({
  title: { type: String, default: "" }
});

defineEmits(["confirm"]);

const title = computed(() => props.title || t('common.confirmAction'));
</script>
