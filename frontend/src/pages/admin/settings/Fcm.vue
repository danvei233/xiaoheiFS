<template>
  <div class="fcm-settings-page">
    <div class="page-header">
      <h1 class="page-title">FCM 推送设置</h1>
      <a-button type="primary" :loading="saving" @click="handleSave">保存更改</a-button>
    </div>

    <a-card :bordered="false">
      <a-form layout="vertical">
        <a-form-item label="启用 FCM 推送">
          <a-switch v-model:checked="form.fcm_enabled" />
          <span class="switch-tip">关闭后不会向管理员设备发送推送通知</span>
        </a-form-item>

        <a-form-item label="FCM Server Key">
          <a-textarea
            v-model:value="form.fcm_server_key"
            :rows="3"
            placeholder="（可选）旧版 Legacy Server Key，建议留空"
          />
        </a-form-item>

        <a-form-item label="FCM Project ID">
          <a-input
            v-model:value="form.fcm_project_id"
            placeholder="your-firebase-project-id"
          />
        </a-form-item>

        <a-form-item label="Service Account JSON">
          <a-textarea
            v-model:value="form.fcm_service_account_json"
            :rows="8"
            placeholder='{"type":"service_account","project_id":"...","private_key":"..."}'
          />
        </a-form-item>

        <a-alert
          type="info"
          show-icon
          message="说明"
          description="优先使用 HTTP v1（Project ID + Service Account JSON）。管理员设备需先调用 /admin/api/v1/push-tokens 注册 token。"
        />
      </a-form>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import { message } from "ant-design-vue";
import { listSettings, updateSetting } from "@/services/admin";

const saving = ref(false);
const form = reactive({
  fcm_enabled: false,
  fcm_server_key: "",
  fcm_project_id: "",
  fcm_service_account_json: ""
});

const fetchData = async () => {
  try {
    const res = await listSettings();
    const items = res.data?.items || [];
    for (const item of items) {
      const key = String(item.key || "");
      if (key === "fcm_enabled") {
        form.fcm_enabled = String(item.value || "").toLowerCase() === "true";
      } else if (key === "fcm_server_key") {
        form.fcm_server_key = String(item.value || "");
      } else if (key === "fcm_project_id") {
        form.fcm_project_id = String(item.value || "");
      } else if (key === "fcm_service_account_json") {
        form.fcm_service_account_json = String(item.value || "");
      }
    }
  } catch (error) {
    console.error("Failed to load FCM settings:", error);
  }
};

const handleSave = async () => {
  saving.value = true;
  try {
    await updateSetting({
      items: [
        { key: "fcm_enabled", value: form.fcm_enabled ? "true" : "false" },
        { key: "fcm_server_key", value: form.fcm_server_key || "" },
        { key: "fcm_project_id", value: form.fcm_project_id || "" },
        { key: "fcm_service_account_json", value: form.fcm_service_account_json || "" }
      ]
    });
    message.success("保存成功");
  } catch (error: any) {
    message.error(error?.response?.data?.error || "保存失败");
  } finally {
    saving.value = false;
  }
};

onMounted(() => {
  fetchData();
});
</script>

<style scoped>
.fcm-settings-page {
  padding: 24px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.page-title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
}

.switch-tip {
  margin-left: 8px;
  color: rgba(0, 0, 0, 0.45);
}
</style>
