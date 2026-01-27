<template>
  <StatusTag :status="status" :textMap="textMap" />
</template>

<script setup>
import StatusTag from "@/components/StatusTag.vue";

defineProps({
  status: { type: String, default: "" }
});

const textMap = {
  running: "运行中",
  stopped: "关机",
  locked: "锁定",
  normal: "正常",
  abuse: "滥用",
  fraud: "欺诈",
  provisioning: "创建中",
  active: "已开通",
  reinstalling: "重装系统中",
  reinstall_failed: "重装系统失败",
  failed: "创建失败",
  deleting: "删除中"
};
</script>
