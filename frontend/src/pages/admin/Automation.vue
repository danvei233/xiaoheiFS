<template>
  <div class="page">
    <div class="page-header">
      <div>
        <div class="page-title">自动化平台对接</div>
        <div class="subtle">配置云平台 API 并同步资源</div>
      </div>
      <div class="page-header-actions">
        <a-button @click="sync">立即同步</a-button>
      </div>
    </div>

    <a-card class="card">
      <a-form layout="vertical">
        <a-form-item label="Base URL"><a-input v-model:value="form.base_url" /></a-form-item>
        <a-form-item label="API Key"><a-input v-model:value="form.api_key" /></a-form-item>
        <a-row :gutter="12">
          <a-col :span="12"><a-form-item label="启用"><a-switch v-model:checked="form.enabled" /></a-form-item></a-col>
          <a-col :span="12"><a-form-item label="超时(秒)"><a-input-number v-model:value="form.timeout_sec" :min="1" style="width: 100%" /></a-form-item></a-col>
        </a-row>
        <a-row :gutter="12">
          <a-col :span="12"><a-form-item label="重试次数"><a-input-number v-model:value="form.retry" :min="0" style="width: 100%" /></a-form-item></a-col>
          <a-col :span="12"><a-form-item label="干跑模式"><a-switch v-model:checked="form.dry_run" /></a-form-item></a-col>
        </a-row>
        <a-space>
          <a-button type="primary" @click="save">保存</a-button>
        </a-space>
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
import { onMounted, reactive, ref } from "vue";
import {
  getAutomationConfig,
  updateAutomationConfig,
  syncAutomationCatalog,
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

const logColumns = [
  { title: "ID", dataIndex: "id", key: "id" },
  { title: "状态", dataIndex: "status", key: "status" },
  { title: "信息", dataIndex: "message", key: "message" },
  { title: "时间", dataIndex: "created_at", key: "created_at" }
];

const load = async () => {
  const res = await getAutomationConfig();
  const data = res.data || {};
  form.base_url = data.base_url || "";
  form.api_key = data.api_key || "";
  form.enabled = data.enabled ?? false;
  form.timeout_sec = data.timeout_sec ?? 15;
  form.retry = data.retry ?? 0;
  form.dry_run = data.dry_run ?? false;
};

const loadLogs = async () => {
  const res = await listAutomationSyncLogs();
  logs.value = res.data?.items || [];
};

const loadStats = async () => {
  const [linesRes, packagesRes, imagesRes] = await Promise.all([listLines(), listPackages(), listSystemImages()]);
  syncStats.lines = (linesRes.data?.items || []).length;
  syncStats.packages = (packagesRes.data?.items || []).length;
  syncStats.images = (imagesRes.data?.items || []).length;
};

const save = async () => {
  await updateAutomationConfig({ ...form });
  message.success("已保存");
};

const sync = async () => {
  await syncAutomationCatalog();
  message.success("已触发同步");
  loadLogs();
};

onMounted(() => {
  load();
  loadLogs();
  loadStats();
});
</script>
