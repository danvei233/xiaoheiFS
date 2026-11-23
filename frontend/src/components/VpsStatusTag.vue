<template>
  <StatusTag :status="status" :textMap="textMap" />
</template>

<script setup>
import { computed } from "vue";
import StatusTag from "@/components/StatusTag.vue";
import { t } from "@/locales";

defineProps({
  status: { type: String, default: "" }
});

const textMap = computed(() => ({
  running: t('status.running'),
  stopped: t('console.vps.stopped'),
  locked: t('console.vps.locked'),
  expired_locked: t('status.expired'),
  normal: t('status.normal'),
  abuse: t('status.abuse'),
  fraud: t('status.fraud'),
  provisioning: t('status.creating'),
  active: t('status.activated'),
  reinstalling: t('status.reinstalling'),
  reinstall_failed: t('status.reinstallFailed'),
  failed: t('status.createFailed'),
  deleting: t('status.deleting')
}));
</script>
