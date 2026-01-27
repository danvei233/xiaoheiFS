<template>
  <StatusTag :status="status" :textMap="textMap" />
</template>

<script setup>
import StatusTag from "@/components/StatusTag.vue";

defineProps({
  status: { type: String, default: "" }
});

const textMap = {
  draft: "草稿",
  pending_payment: "待支付",
  pending_review: "待审核",
  approved: "已通过",
  provisioning: "开通中",
  active: "已完成",
  rejected: "已驳回",
  canceled: "已取消",
  failed: "失败"
};
</script>
