<template>
  <div class="payments-settings-page">
    <div class="page-header">
      <h1 class="page-title">支付设置</h1>
      <a-button type="primary" @click="handleSave" :loading="saving">
        保存更改
      </a-button>
    </div>

    <a-card :bordered="false">
      <a-table
        :columns="columns"
        :data-source="providers"
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
            <a-button type="link" @click="openConfigModal(record)">
              配置
            </a-button>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- Config Modal -->
    <a-modal
      v-model:open="configModalVisible"
      :title="`配置 ${currentProvider?.name}`"
      @ok="saveConfig"
      :confirm-loading="configSaving"
      width="600px"
    >
      <a-form layout="vertical">
        <template v-if="configFields.length > 0">
          <a-form-item
            v-for="field in configFields"
            :key="field.key"
            :label="field.title"
          >
            <a-input
              v-if="field.type === 'string' || field.type === 'password'"
              :type="field.type === 'password' ? 'password' : 'text'"
              v-model:value="configForm[field.key]"
              :placeholder="field.description"
            />
            <a-input-number
              v-else-if="field.type === 'number'"
              v-model:value="configForm[field.key]"
              style="width: 100%"
            />
            <a-switch
              v-else-if="field.type === 'boolean'"
              v-model:checked="configForm[field.key]"
            />
          </a-form-item>
        </template>
        <a-alert v-else type="info" message="该支付方式无需配置" show-icon />
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed } from "vue";
import { message } from "ant-design-vue";
import { listAdminPaymentProviders, updateAdminPaymentProvider } from "@/services/admin";

const loading = ref(false);
const saving = ref(false);
const configSaving = ref(false);
const configModalVisible = ref(false);
const providers = ref<any[]>([]);
const currentProvider = ref<any>(null);

const configForm = reactive<Record<string, any>>({});

const columns = [
  { title: "名称", dataIndex: "name", key: "name" },
  { title: "Key", dataIndex: "key", key: "key" },
  { title: "状态", dataIndex: "enabled", key: "enabled", width: 100 },
  { title: "操作", key: "actions", width: 100 }
];

const configFields = computed(() => {
  if (!currentProvider.value?.schema_json) return [];
  try {
    return JSON.parse(currentProvider.value.schema_json);
  } catch {
    return [];
  }
});

const fetchData = async () => {
  loading.value = true;
  try {
    const res = await listAdminPaymentProviders();
    providers.value = res.data?.items || [];
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

const openConfigModal = (record: any) => {
  currentProvider.value = record;
  try {
    const config = record.config_json ? JSON.parse(record.config_json) : {};
    Object.keys(configForm).forEach(key => delete configForm[key]);
    Object.assign(configForm, config);
  } catch {
    Object.keys(configForm).forEach(key => delete configForm[key]);
  }
  configModalVisible.value = true;
};

const saveConfig = async () => {
  configSaving.value = true;
  try {
    await updateAdminPaymentProvider(currentProvider.value.key, {
      config_json: JSON.stringify(configForm)
    });
    message.success("保存成功");
    configModalVisible.value = false;
    fetchData();
  } catch (error: any) {
    message.error(error.response?.data?.error || "保存失败");
  } finally {
    configSaving.value = false;
  }
};

const handleSave = async () => {
  saving.value = true;
  try {
    message.success("保存成功");
  } finally {
    saving.value = false;
  }
};

onMounted(() => {
  fetchData();
});
</script>

<style scoped>
.payments-settings-page {
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
