<template>
  <a-tag :color="color">{{ label }}</a-tag>
</template>

<script setup>
import { computed } from "vue";
import { t } from "@/locales";

const props = defineProps({
  status: { type: String, default: "" },
  textMap: { type: Object, default: () => ({}) }
});

const defaultTextMap = computed(() => ({
  draft: t('status.draft'),
  pending_payment: t('status.pendingPayment'),
  pending_review: t('status.pendingReview'),
  approved: t('status.approved'),
  provisioning: t('status.provisioning'),
  active: t('status.activeService'),
  rejected: t('status.rejected'),
  canceled: t('status.canceled'),
  failed: t('status.failed'),
  running: t('status.running'),
  stopped: t('status.stopped'),
  locked: t('status.locked'),
  expired_locked: t('status.expired'),
  normal: t('status.normal'),
  abuse: t('status.abuse'),
  fraud: t('status.fraud'),
  blocked: t('status.blocked'),
  disabled: t('status.blocked'),
  success: t('status.success')
}));

const color = computed(() => {
  if (["active", "running", "approved", "success", "normal"].includes(props.status)) return "green";
  if (["pending_review", "pending", "pending_payment", "provisioning", "draft", "reinstalling", "deleting"].includes(props.status)) return "gold";
  if (["failed", "rejected", "disabled", "locked", "expired_locked", "canceled", "blocked", "abuse", "fraud", "reinstall_failed"].includes(props.status)) return "red";
  return "blue";
});

const label = computed(() => props.textMap[props.status] || defaultTextMap.value[props.status] || props.status || "-");
</script>
