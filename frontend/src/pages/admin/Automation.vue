<template>
  <div class="page">
    <div class="page-header">
      <div>
        <div class="page-title">自动化平台对接</div>
        <div class="subtle">此入口已废弃，配置迁移到 商品类型 -> 自动化实例（插件模板）</div>
      </div>
      <div class="page-header-actions">
        <a-button type="primary" @click="goCatalog">前往商品类型配置</a-button>
      </div>
    </div>

    <a-card class="card">
      <a-alert
        type="warning"
        show-icon
        message="旧自动化设置已只读"
        description="请在 商品类型 页面选择 automation 插件实例并按插件模板配置（base_url/api_key 等）。"
        style="margin-bottom: 12px"
      />
      <a-form layout="vertical">
        <a-form-item label="Base URL（legacy）"><a-input :value="form.base_url" readonly /></a-form-item>
        <a-form-item label="API Key（legacy）"><a-input :value="maskedApiKey" readonly /></a-form-item>
        <a-row :gutter="12">
          <a-col :span="12"><a-form-item label="启用（legacy）"><a-switch :checked="form.enabled" disabled /></a-form-item></a-col>
          <a-col :span="12"><a-form-item label="超时(秒)（legacy）"><a-input-number :value="form.timeout_sec" :min="1" style="width: 100%" disabled /></a-form-item></a-col>
        </a-row>
        <a-row :gutter="12">
          <a-col :span="12"><a-form-item label="重试次数（legacy）"><a-input-number :value="form.retry" :min="0" style="width: 100%" disabled /></a-form-item></a-col>
          <a-col :span="12"><a-form-item label="干跑模式（legacy）"><a-switch :checked="form.dry_run" disabled /></a-form-item></a-col>
        </a-row>
      </a-form>
    </a-card>

    <a-row :gutter="16" style="margin-top: 16px">
      <a-col :xs="24" :md="8">
        <a-card class="card"><a-statistic title="已同步线路" :value="syncStats.lines" /></a-card>
      </a-col>
      <a-col :xs="24" :md="8">
        <a-card class="card"><a-statistic title="已同步套餐" :value="syncStats.packages" /></a-card>
      </a-col>
      <a-col :xs="24" :md="8">
        <a-card class="card"><a-statistic title="已同步镜像" :value="syncStats.images" /></a-card>
      </a-col>
    </a-row>

    <a-card class="card" style="margin-top: 16px">
      <div class="section-title">同步日志</div>
      <a-table :columns="logColumns" :data-source="logs" row-key="id" :pagination="false" />
    </a-card>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from "vue";
import { useRouter } from "vue-router";
import {
  getAutomationConfig,
  listAutomationSyncLogs,
  listLines,
  listPackages,
  listSystemImages
} from "@/services/admin";
import { message } from "ant-design-vue";

const form = reactive({
  base_url: "",
  api_key: "",
  enabled: false,
  timeout_sec: 15,
  retry: 0,
  dry_run: false
});

const logs = ref([]);
const syncStats = reactive({ lines: 0, packages: 0, images: 0 });
const router = useRouter();
const maskedApiKey = computed(() => {
  const raw = String(form.api_key || "");
  if (!raw) return "";
  if (raw.length <= 4) return "****";
  return `${raw.slice(0, 2)}****${raw.slice(-2)}`;
});

const logColumns = [
  { title: "ID", dataIndex: "id", key: "id" },
  { title: "状态", dataIndex: "status", key: "status" },
  { title: "信息", dataIndex: "message", key: "message" },
  { title: "时间", dataIndex: "created_at", key: "created_at" }
];

const load = async () => {
  try {
    const res = await getAutomationConfig();
    const data = res.data || {};
    form.base_url = data.base_url || "";
    form.api_key = data.api_key || "";
    form.enabled = data.enabled ?? false;
    form.timeout_sec = data.timeout_sec ?? 15;
    form.retry = data.retry ?? 0;
    form.dry_run = data.dry_run ?? false;
  } catch (e) {
    const errMsg = e && e.response && e.response.data ? e.response.data.error : "";
    message.error(errMsg || "加载自动化配置失败");
  }
};

const loadLogs = async () => {
  try {
    const res = await listAutomationSyncLogs();
    logs.value = res.data?.items || [];
  } catch (e) {
    logs.value = [];
    const errMsg = e && e.response && e.response.data ? e.response.data.error : "";
    message.error(errMsg || "加载同步日志失败");
  }
};

const loadStats = async () => {
  try {
    const [linesRes, packagesRes, imagesRes] = await Promise.all([listLines(), listPackages(), listSystemImages()]);
    syncStats.lines = (linesRes.data?.items || []).length;
    syncStats.packages = (packagesRes.data?.items || []).length;
    syncStats.images = (imagesRes.data?.items || []).length;
  } catch (e) {
    syncStats.lines = 0;
    syncStats.packages = 0;
    syncStats.images = 0;
    const errMsg = e && e.response && e.response.data ? e.response.data.error : "";
    message.error(errMsg || "加载同步统计失败");
  }
};

const goCatalog = () => {
  message.info("请在商品类型中配置 automation 插件实例");
  router.push("/admin/catalog");
};

onMounted(async () => {
  await Promise.all([load(), loadLogs(), loadStats()]);
});
</script>
