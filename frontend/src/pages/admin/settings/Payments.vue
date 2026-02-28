<template>
  <div class="payments-settings-page">
    <div class="page-header">
      <h1 class="page-title">支付设置</h1>
    </div>

    <a-card :bordered="false">
      <a-table
        :columns="columns"
        :data-source="rows"
        :loading="loading"
        :row-key="rowKey"
        :pagination="false"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'type'">
            <span>{{ record.type === "plugin" ? "插件" : "内置" }}</span>
          </template>
          <template v-if="column.key === 'order_enabled'">
            <a-switch
              :checked="record.order_enabled"
              :loading="record.busy_order"
              @change="(checked: boolean) => handleToggle(record, 'order', checked)"
            />
          </template>
          <template v-if="column.key === 'wallet_enabled'">
            <a-switch
              :checked="record.wallet_enabled"
              :loading="record.busy_wallet"
              @change="(checked: boolean) => handleToggle(record, 'wallet', checked)"
            />
          </template>
        </template>
      </a-table>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue";
import { message } from "ant-design-vue";
import {
  listAdminPaymentProviders,
  listAdminPlugins,
  listAdminPluginPaymentMethods,
  updateAdminPaymentProvider
} from "@/services/admin";
import type { PaymentProvider, PluginListItem, PluginPaymentMethodItem } from "@/services/types";

const loading = ref(false);
const rows = ref<RowItem[]>([]);

type BuiltinRow = {
  type: "builtin";
  key: string;
  provider_key: string;
  name: string;
  enabled: boolean;
  order_enabled: boolean;
  wallet_enabled: boolean;
  busy_order?: boolean;
  busy_wallet?: boolean;
};

type PluginRow = {
  type: "plugin";
  key: string;
  provider_key: string;
  name: string;
  plugin_id: string;
  instance_id: string;
  method: string;
  enabled: boolean;
  order_enabled: boolean;
  wallet_enabled: boolean;
  busy_order?: boolean;
  busy_wallet?: boolean;
};

type RowItem = BuiltinRow | PluginRow;

const columns = [
  { title: "类型", key: "type", width: 90 },
  { title: "名称", dataIndex: "name", key: "name" },
  { title: "Key", dataIndex: "key", key: "key" },
  { title: "订单支付", dataIndex: "order_enabled", key: "order_enabled", width: 110 },
  { title: "钱包充值", dataIndex: "wallet_enabled", key: "wallet_enabled", width: 110 }
];

const rowKey = (r: RowItem) => r.key;

const pluginMethodsMap = async (plugin: PluginListItem): Promise<Map<string, boolean>> => {
  const category = String(plugin.category || "payment").trim() || "payment";
  const pluginID = String(plugin.plugin_id || "").trim();
  const instanceID = String(plugin.instance_id || "default").trim() || "default";
  if (!pluginID) return new Map<string, boolean>();
  const res = await listAdminPluginPaymentMethods({
    category,
    plugin_id: pluginID,
    instance_id: instanceID
  });
  const items = (res.data?.items || []) as PluginPaymentMethodItem[];
  const out = new Map<string, boolean>();
  items.forEach((x) => {
    const m = String(x.method || "").trim();
    if (!m) return;
    out.set(m, !!x.enabled);
  });
  return out;
};

const pluginMethodsFromManifest = (plugin: PluginListItem): string[] => {
  const methods = plugin.manifest?.capabilities?.payment?.methods || [];
  const uniq = new Set<string>();
  methods.forEach((m) => {
    const method = String(m || "").trim();
    if (!method) return;
    uniq.add(method);
  });
  return Array.from(uniq);
};

const buildRows = async () => {
  const [providersRes, pluginsRes] = await Promise.all([
    listAdminPaymentProviders({ include_disabled: true, include_legacy: false }),
    listAdminPlugins()
  ]);
  const providers = (providersRes.data?.items || []) as PaymentProvider[];
  const plugins = (pluginsRes.data?.items || []) as PluginListItem[];
  const providerStateMap = new Map<string, { enabled: boolean; order_enabled: boolean; wallet_enabled: boolean }>();
  providers.forEach((p) => {
    const key = String(p.key || "").trim();
    if (!key) return;
    providerStateMap.set(key, {
      enabled: !!p.enabled,
      order_enabled: p.order_enabled !== false,
      wallet_enabled: p.wallet_enabled !== false
    });
  });

  const builtinRows: RowItem[] = providers
    .filter((p) => {
      const key = String(p.key || "").trim().toLowerCase();
      if (!key) return false;
      if (key === "yipay") return false;
      if (key === "custom") return false;
      return !key.includes(".");
    })
    .map((p) => ({
      type: "builtin",
      key: String(p.key || ""),
      provider_key: String(p.key || ""),
      name: String(p.name || p.key || ""),
      enabled: !!p.enabled,
      order_enabled: p.order_enabled !== false,
      wallet_enabled: p.wallet_enabled !== false,
      busy_order: false,
      busy_wallet: false
    }));

  const enabledPaymentPlugins = plugins.filter((p) => {
    if (!p.enabled) return false;
    if (!p.loaded) return false;
    const category = String(p.category || "").trim();
    if (category !== "payment") return false;
    const methods = pluginMethodsFromManifest(p);
    return methods.length > 0;
  });

  const methodStateList = await Promise.all(
    enabledPaymentPlugins.map((p) => pluginMethodsMap(p))
  );

  const pluginRows: RowItem[] = [];
  enabledPaymentPlugins.forEach((plugin, idx) => {
    const pluginID = String(plugin.plugin_id || "").trim();
    const instanceID = String(plugin.instance_id || "default").trim() || "default";
    const methods = pluginMethodsFromManifest(plugin);
    const enabledMap = methodStateList[idx];
    methods.forEach((method) => {
      const enabled = enabledMap.has(method) ? !!enabledMap.get(method) : true;
      pluginRows.push({
        type: "plugin",
        key: `${pluginID}.${instanceID}.${method}`,
        provider_key: `${pluginID}.${method}`,
        name: `${String(plugin.name || pluginID)} / ${method}`,
        plugin_id: pluginID,
        instance_id: instanceID,
        method,
        enabled,
        order_enabled: providerStateMap.get(`${pluginID}.${method}`)?.order_enabled ?? true,
        wallet_enabled: providerStateMap.get(`${pluginID}.${method}`)?.wallet_enabled ?? true,
        busy_order: false,
        busy_wallet: false
      });
    });
  });

  rows.value = [...builtinRows, ...pluginRows].sort((a, b) => a.key.localeCompare(b.key));
};

const fetchData = async () => {
  loading.value = true;
  try {
    await buildRows();
  } finally {
    loading.value = false;
  }
};

const handleToggle = async (record: RowItem, scene: "order" | "wallet", checked: boolean) => {
  if (scene === "order") {
    record.busy_order = true;
  } else {
    record.busy_wallet = true;
  }
  try {
    await updateAdminPaymentProvider(record.provider_key, {
      scene,
      enabled: checked
    });
    if (scene === "order") {
      record.order_enabled = checked;
    } else {
      record.wallet_enabled = checked;
    }
    message.success("操作成功");
  } catch (error: any) {
    message.error(error.response?.data?.error || "操作失败");
  } finally {
    if (scene === "order") {
      record.busy_order = false;
    } else {
      record.busy_wallet = false;
    }
  }
};

onMounted(() => {
  void fetchData();
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
