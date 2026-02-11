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
            <a-space>
              <a-button type="link" @click="openConfigModal(record)">配置</a-button>
              <a-button v-if="pluginKeyFromProvider(record?.key)" type="link" @click="openPluginMethods(record)">
                Methods
              </a-button>
            </a-space>
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

    <!-- Plugin payment methods (host-managed) -->
    <a-modal v-model:open="methodsOpen" title="Plugin payment methods" width="560px" :footer="null">
      <a-alert
        type="info"
        show-icon
        message="ListMethods 由插件声明；启用/停用开关由宿主管理。未设置开关的 method 默认启用。"
        style="margin-bottom: 12px"
      />
      <a-spin :spinning="methodsLoading">
        <a-table :columns="methodsColumns" :data-source="methodItems" :pagination="false" row-key="method">
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'enabled'">
              <a-switch
                :checked="!!record.enabled"
                :loading="methodBusyKey === record.method"
                @change="(checked:boolean)=>toggleMethod(record.method, checked)"
              />
            </template>
          </template>
        </a-table>
      </a-spin>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed } from "vue";
import { message } from "ant-design-vue";
import {
  listAdminPaymentProviders,
  listAdminPluginPaymentMethods,
  updateAdminPaymentProvider,
  updateAdminPluginPaymentMethod
} from "@/services/admin";
import type { PluginPaymentMethodItem } from "@/services/types";

const loading = ref(false);
const saving = ref(false);
const configSaving = ref(false);
const configModalVisible = ref(false);
const providers = ref<any[]>([]);
const currentProvider = ref<any>(null);
const methodsOpen = ref(false);
const methodsLoading = ref(false);
const methodBusyKey = ref("");
const methodItems = ref<PluginPaymentMethodItem[]>([]);
const currentPlugin = ref<{ plugin_id: string; instance_id: string } | null>(null);

const configForm = reactive<Record<string, any>>({});

const columns = [
  { title: "名称", dataIndex: "name", key: "name" },
  { title: "Key", dataIndex: "key", key: "key" },
  { title: "状态", dataIndex: "enabled", key: "enabled", width: 100 },
  { title: "操作", key: "actions", width: 100 }
];

const methodsColumns = [
  { title: "method", dataIndex: "method", key: "method" },
  { title: "enabled", key: "enabled", width: 120 }
];

const pluginKeyFromProvider = (providerKey?: string) => {
  const k = String(providerKey || "").trim();
  if (!k) return "";
  const idx = k.indexOf(".");
  if (idx <= 0) return "";
  return k.slice(0, idx);
};

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
    const items = res.data?.items || [];
    providers.value = items.filter((item: any) => String(item?.key || "").toLowerCase() !== "yipay");
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

const openPluginMethods = async (record: any) => {
  const pluginID = pluginKeyFromProvider(record?.key);
  if (!pluginID) return;
  currentPlugin.value = { plugin_id: pluginID, instance_id: "default" };
  methodsOpen.value = true;
  methodsLoading.value = true;
  try {
    const res = await listAdminPluginPaymentMethods({
      category: "payment",
      plugin_id: pluginID,
      instance_id: "default"
    });
    methodItems.value = (res.data?.items || []) as PluginPaymentMethodItem[];
  } catch (e: any) {
    message.error(e?.response?.data?.error || "加载失败");
  } finally {
    methodsLoading.value = false;
  }
};

const toggleMethod = async (method: string, enabled: boolean) => {
  const cur = currentPlugin.value;
  if (!cur) return;
  methodBusyKey.value = method;
  try {
    await updateAdminPluginPaymentMethod({
      category: "payment",
      plugin_id: cur.plugin_id,
      instance_id: cur.instance_id,
      method,
      enabled
    });
    const it = methodItems.value.find((x) => x.method === method);
    if (it) it.enabled = enabled;
    message.success("OK");
  } catch (e: any) {
    message.error(e?.response?.data?.error || "failed");
  } finally {
    methodBusyKey.value = "";
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
