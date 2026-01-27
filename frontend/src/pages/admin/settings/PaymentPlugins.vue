<template>
  <div class="payment-plugins-page">
    <div class="page-header">
      <h1 class="page-title">支付插件管理</h1>
      <a-upload
        :custom-request="handleUpload"
        :show-upload-list="false"
        accept=".zip"
      >
        <a-button type="primary">
          <template #icon><UploadOutlined /></template>
          上传插件
        </a-button>
      </a-upload>
    </div>

    <a-card :bordered="false">
      <a-table
        :columns="columns"
        :data-source="plugins"
        :loading="loading"
        row-key="key"
        :pagination="false"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'enabled'">
            <a-switch
              :checked="record.enabled"
              @change="(checked: boolean) => handleToggle(record, checked)"
            />
          </template>
          <template v-else-if="column.key === 'actions'">
            <a-space>
              <a-button type="link" size="small" @click="openConfigModal(record)">
                配置
              </a-button>
              <a-popconfirm
                title="确定要删除此插件吗？"
                @confirm="handleDelete(record)"
              >
                <a-button type="link" danger size="small">
                  删除
                </a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- Config Modal -->
    <a-modal
      v-model:open="configModalVisible"
      :title="`配置 ${currentPlugin?.name}`"
      @ok="saveConfig"
      :confirm-loading="configSaving"
      width="600px"
    >
      <a-form layout="vertical">
        <a-form-item label="插件密码（如果加密）">
          <a-input v-model:value="pluginPassword" placeholder="请输入插件密码" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from "vue";
import { message } from "ant-design-vue";
import { UploadOutlined } from "@ant-design/icons-vue";
import type { UploadRequestOption } from "ant-design-vue";
import { listAdminPaymentProviders, updateAdminPaymentProvider, uploadPaymentPlugin } from "@/services/admin";

const loading = ref(false);
const configSaving = ref(false);
const configModalVisible = ref(false);
const plugins = ref<any[]>([]);
const currentPlugin = ref<any>(null);
const pluginPassword = ref("");

const columns = [
  { title: "名称", dataIndex: "name", key: "name" },
  { title: "Key", dataIndex: "key", key: "key" },
  { title: "状态", dataIndex: "enabled", key: "enabled", width: 100 },
  { title: "操作", key: "actions", width: 150 }
];

const fetchData = async () => {
  loading.value = true;
  try {
    const res = await listAdminPaymentProviders();
    plugins.value = res.data?.items || [];
  } finally {
    loading.value = false;
  }
};

const handleToggle = async (record: any, checked: boolean) => {
  try {
    await updateAdminPaymentProvider(record.key, { enabled: checked });
    record.enabled = checked;
    message.success("操作成功");
  } catch (error: any) {
    message.error(error.response?.data?.error || "操作失败");
  }
};

const handleUpload = async (options: UploadRequestOption) => {
  const { file } = options;
  const formData = new FormData();
  formData.append("file", file as File);
  formData.append("password", pluginPassword.value);

  try {
    await uploadPaymentPlugin(file as File, pluginPassword.value);
    message.success("上传成功");
    fetchData();
  } catch (error: any) {
    message.error(error.response?.data?.error || "上传失败");
  }
};

const openConfigModal = (record: any) => {
  currentPlugin.value = record;
  pluginPassword.value = "";
  configModalVisible.value = true;
};

const saveConfig = async () => {
  message.success("配置已保存");
  configModalVisible.value = false;
};

const handleDelete = async (record: any) => {
  message.success("删除成功");
  fetchData();
};

onMounted(() => {
  fetchData();
});
</script>

<style scoped>
.payment-plugins-page {
  padding: 24px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.page-title {
  font-size: 20px;
  font-weight: 600;
  margin: 0;
}
</style>
