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
  draft: t('status.draft'),
  pending_payment: t('status.pendingPayment'),
  pending_review: t('status.pendingReview'),
  approved: t('status.approved'),
  provisioning: t('status.provisioning'),
  active: t('status.activeService'),
  rejected: t('status.rejected'),
  canceled: t('status.canceled'),
  failed: t('status.failed')
}));
</script>
