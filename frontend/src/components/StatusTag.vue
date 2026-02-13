<template>
  <a-tag :color="color">{{ label }}</a-tag>
</template>

<script setup>
import { computed } from "vue";

const props = defineProps({
  status: { type: String, default: "" },
  textMap: { type: Object, default: () => ({}) }
});

const defaultTextMap = {
  draft: "草稿",
  pending_payment: "待支付",
  pending_review: "待审核",
  approved: "已通过",
  provisioning: "开通中",
  active: "已完成",
  rejected: "已驳回",
  canceled: "已取消",
  failed: "失败",
  running: "运行中",
  stopped: "已关机",
  locked: "已锁定",
  expired_locked: "已到期",
  normal: "正常",
  abuse: "Abuse",
  fraud: "Fraud",
  blocked: "禁用",
  disabled: "禁用",
  success: "成功"
};

const color = computed(() => {
  if (["active", "running", "approved", "success", "normal"].includes(props.status)) return "green";
  if (["pending_review", "pending", "pending_payment", "provisioning", "draft", "reinstalling", "deleting"].includes(props.status)) return "gold";
  if (["failed", "rejected", "disabled", "locked", "expired_locked", "canceled", "blocked", "abuse", "fraud", "reinstall_failed"].includes(props.status)) return "red";
  return "blue";
});

const label = computed(() => props.textMap[props.status] || defaultTextMap[props.status] || props.status || "-");
</script>
