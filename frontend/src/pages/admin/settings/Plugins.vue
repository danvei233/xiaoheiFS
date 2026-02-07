<template>
  <div class="plugins-page">
    <!-- payment-method toggles are host-managed -->
    <div class="hero">
      <div class="hero-left">
        <div class="hero-title">插件管理</div>
        <div class="hero-subtle">统一管理 SMS / 支付 / 实名 / 未来更多类型：安装、启停、配置与健康</div>
      </div>
      <div class="hero-actions">
        <a-button @click="fetchData" :loading="loading">刷新</a-button>
        <a-button @click="openDiscover" :loading="discoverLoading">发现磁盘插件</a-button>
        <a-upload :custom-request="onInstallUpload" :show-upload-list="false" accept=".zip,.tar.gz,.tgz">
          <a-button type="primary">安装插件</a-button>
        </a-upload>
      </div>
    </div>

    <a-card :bordered="false" class="card">
      <div class="filters">
        <a-segmented v-model:value="category" :options="categoryOptions" />
        <a-input-search v-model:value="keyword" placeholder="搜索插件名 / plugin_id" style="width: 260px" allow-clear />
      </div>

      <a-table
        :columns="columns"
        :data-source="filtered"
        :loading="loading"
        :row-key="rowKey"
        :pagination="false"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'plugin'">
            <div class="plugin-cell">
              <div class="plugin-name">{{ record.name || record.plugin_id }}</div>
              <div class="plugin-meta">
                <span class="mono">{{ record.category }}/{{ record.plugin_id }}/{{ record.instance_id || "default" }}</span>
                <a-tag color="default" class="mono">v{{ record.version || "-" }}</a-tag>
              </div>
            </div>
          </template>

          <template v-else-if="column.key === 'signature'">
            <a-tag :color="signatureColor(record.signature_status)">
              {{ signatureLabel(record.signature_status) }}
            </a-tag>
          </template>

          <template v-else-if="column.key === 'instance_id'">
            <span class="mono">{{ record.instance_id || "default" }}</span>
          </template>

          <template v-else-if="column.key === 'enabled'">
            <a-switch
              :checked="!!record.enabled"
              :loading="busyKey === `${record.category}/${record.plugin_id}/${record.instance_id || 'default'}`"
              @change="(checked:boolean)=>toggleEnabled(record, checked)"
            />
          </template>

          <template v-else-if="column.key === 'health'">
            <div class="health-cell">
              <a-tag :color="healthColor(record.health_status)">
                {{ record.health_status || "-" }}
              </a-tag>
              <div class="health-subtle">
                <span v-if="record.last_health_at">最后：{{ formatTime(record.last_health_at) }}</span>
                <span v-else>暂无心跳</span>
                <span v-if="record.health_message" class="health-msg">· {{ record.health_message }}</span>
              </div>
            </div>
          </template>

          <template v-else-if="column.key === 'capabilities'">
            <div class="caps">
              <a-tag v-if="record.manifest?.capabilities?.payment" color="blue">
                payment: {{ (record.manifest.capabilities.payment.methods || []).length }} methods
              </a-tag>
              <a-tag v-if="record.manifest?.capabilities?.sms" color="cyan">sms</a-tag>
              <a-tag v-if="record.manifest?.capabilities?.kyc" color="purple">kyc</a-tag>
              <a-tag v-if="record.manifest?.capabilities?.automation" color="gold">
                automation: {{ (record.manifest.capabilities.automation.features || []).length }} features
              </a-tag>
              <a-button type="link" size="small" @click="openManifest(record)">详情</a-button>
            </div>
          </template>

          <template v-else-if="column.key === 'actions'">
            <a-space>
              <a-button type="link" size="small" @click="openCreateInstance(record)">Add instance</a-button>
              <a-button
                v-if="String(record.category || '') === 'payment'"
                type="link"
                size="small"
                @click="openPaymentMethods(record)"
              >
                Methods
              </a-button>
              <a-button type="link" size="small" @click="openConfig(record)">
                {{ String(record.category || "") === "automation" ? "商品配置" : "配置" }}
              </a-button>
              <a-popconfirm title="确定要卸载该插件吗？" @confirm="uninstall(record)">
                <a-button type="link" danger size="small">卸载</a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- Install: admin password (for untrusted/unsigned) -->
    <a-modal v-model:open="adminPwdOpen" title="安装确认（非官方签名）" :confirm-loading="installing" @ok="confirmInstall">
      <a-alert
        type="warning"
        show-icon
        message="该插件未通过官方签名校验。继续安装存在风险。"
        style="margin-bottom: 12px"
      />
      <a-form layout="vertical">
        <a-form-item label="管理员密码">
          <a-input-password v-model:value="adminPassword" placeholder="请输入管理员密码确认安装" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- Discover modal -->
    <a-modal v-model:open="discoverOpen" title="发现磁盘插件（未导入）" width="860px" @ok="discoverOpen=false">
      <a-alert
        type="info"
        show-icon
        message="这些插件目录已存在于服务器 ./plugins 下，但尚未写入数据库。官方签名插件会在后续启动自动导入；非官方需要手动导入并输入管理员密码。"
        style="margin-bottom: 12px"
      />
      <a-table :columns="discoverColumns" :data-source="discovered" :loading="discoverLoading" :pagination="false" :row-key="rowKey">
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'plugin'">
            <div class="plugin-cell">
              <div class="plugin-name">{{ record.name || record.plugin_id }}</div>
              <div class="plugin-meta">
                <span class="mono">{{ record.category }}/{{ record.plugin_id }}</span>
                <a-tag color="default" class="mono">v{{ record.version || "-" }}</a-tag>
              </div>
            </div>
          </template>
          <template v-else-if="column.key === 'signature'">
            <a-tag :color="signatureColor(record.signature_status)">{{ signatureLabel(record.signature_status) }}</a-tag>
          </template>
          <template v-else-if="column.key === 'platform'">
            <a-tag :color="record.entry?.entry_supported ? 'green' : 'red'">
              {{ record.entry?.platform || "-" }}
            </a-tag>
            <div class="health-subtle" v-if="!record.entry?.entry_supported">
              支持：{{ (record.entry?.supported_platforms || []).join(", ") || "-" }}
            </div>
          </template>
          <template v-else-if="column.key === 'actions'">
            <a-button
              type="link"
              size="small"
              :loading="importBusyKey === `${record.category}/${record.plugin_id}`"
              @click="startImport(record)"
            >
              导入
            </a-button>
          </template>
        </template>
      </a-table>
    </a-modal>

    <!-- Import: admin password (for untrusted/unsigned) -->
    <a-modal v-model:open="importPwdOpen" title="导入确认（非官方签名）" :confirm-loading="importing" @ok="confirmImport">
      <a-alert
        type="warning"
        show-icon
        message="该插件未通过官方签名校验。继续导入存在风险。"
        style="margin-bottom: 12px"
      />
      <a-form layout="vertical">
        <a-form-item label="管理员密码">
          <a-input-password v-model:value="importAdminPassword" placeholder="请输入管理员密码确认导入" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- Create instance -->
    <a-modal v-model:open="instanceOpen" title="Add instance" :confirm-loading="instanceCreating" @ok="confirmCreateInstance">
      <a-form layout="vertical">
        <a-form-item label="instance_id (optional)">
          <a-input v-model:value="instanceId" placeholder="Leave empty to auto-generate" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- Config modal -->
    <a-modal v-model:open="configOpen" :title="`配置：${current?.name || current?.plugin_id || ''}`" width="760px" @ok="saveConfig" :confirm-loading="saving">
      <a-alert v-if="schemaError" type="error" show-icon :message="schemaError" style="margin-bottom: 12px" />
      <a-spin :spinning="schemaLoading">
        <JsonSchemaForm v-if="schemaObj" v-model:modelValue="configModel" :schema="schemaObj" :uiSchema="uiObj" />
        <a-divider />
        <a-collapse>
          <a-collapse-panel key="raw" header="查看原始 JSON（调试用）">
            <a-textarea :value="prettyConfig" :rows="10" readonly />
          </a-collapse-panel>
        </a-collapse>
      </a-spin>
    </a-modal>

    <!-- Manifest drawer -->
    <a-drawer v-model:open="manifestOpen" title="插件能力" width="520">
      <a-descriptions bordered size="small" :column="1">
        <a-descriptions-item label="plugin_id">{{ current?.plugin_id }}</a-descriptions-item>
        <a-descriptions-item label="category">{{ current?.category }}</a-descriptions-item>
        <a-descriptions-item label="name">{{ current?.name }}</a-descriptions-item>
        <a-descriptions-item label="version">{{ current?.version }}</a-descriptions-item>
        <a-descriptions-item label="signature">{{ signatureLabel(current?.signature_status) }}</a-descriptions-item>
      </a-descriptions>
      <a-divider />
      <a-typography-title :level="5">Manifest JSON</a-typography-title>
      <a-textarea :value="prettyManifest" :rows="14" readonly />
    </a-drawer>

    <!-- Payment methods -->
    <a-modal v-model:open="methodsOpen" title="Payment methods" width="560px" :footer="null">
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
import { computed, ref } from "vue";
import { message } from "ant-design-vue";
import type { UploadRequestOption } from "ant-design-vue";
import JsonSchemaForm from "@/components/forms/JsonSchemaForm.vue";
import {
  createAdminPluginInstance,
  discoverAdminPlugins,
  deleteAdminPluginInstance,
  disableAdminPluginInstance,
  enableAdminPluginInstance,
  getAdminPluginInstanceConfig,
  getAdminPluginInstanceConfigSchema,
  importAdminPluginFromDisk,
  installAdminPlugin,
  listAdminPluginPaymentMethods,
  listAdminPlugins,
  updateAdminPluginPaymentMethod,
  updateAdminPluginInstanceConfig
} from "@/services/admin";
import type { PluginDiscoverItem, PluginListItem, PluginPaymentMethodItem } from "@/services/types";

const loading = ref(false);
const installing = ref(false);
const saving = ref(false);
const busyKey = ref("");

const instanceOpen = ref(false);
const instanceCreating = ref(false);
const instanceId = ref("");
const instanceTarget = ref<PluginListItem | null>(null);

const items = ref<PluginListItem[]>([]);
const category = ref<string>("all");
const keyword = ref<string>("");

const categoryOptions = [
  { label: "全部", value: "all" },
  { label: "payment", value: "payment" },
  { label: "sms", value: "sms" },
  { label: "kyc", value: "kyc" },
  { label: "automation", value: "automation" }
];

const rowKey = (r: any) => `${r.category}/${r.plugin_id}/${r.instance_id || "default"}`;

const columns = [
  { title: "插件", key: "plugin" },
  { title: "类型", dataIndex: "category", key: "category", width: 110 },
  { title: "instance", dataIndex: "instance_id", key: "instance_id", width: 140 },
  { title: "签名", key: "signature", width: 120 },
  { title: "启用", key: "enabled", width: 90 },
  { title: "健康", key: "health", width: 220 },
  { title: "能力", key: "capabilities", width: 240 },
  { title: "操作", key: "actions", width: 150 }
];

const filtered = computed(() => {
  const kw = keyword.value.trim().toLowerCase();
  return (items.value || []).filter((it) => {
    if (category.value !== "all" && String(it.category || "") !== category.value) return false;
    if (!kw) return true;
    const hay = `${it.name || ""} ${it.plugin_id || ""} ${it.category || ""} ${it.instance_id || ""}`.toLowerCase();
    return hay.includes(kw);
  });
});

const fetchData = async () => {
  loading.value = true;
  try {
    const res = await listAdminPlugins();
    items.value = (res.data?.items || []) as PluginListItem[];
  } catch (e: any) {
    message.error(e?.response?.data?.error || "加载失败");
  } finally {
    loading.value = false;
  }
};
fetchData();

const signatureLabel = (s: any) => {
  if (s === "official") return "official";
  if (s === "unsigned") return "unsigned";
  if (s === "untrusted") return "untrusted";
  return s || "-";
};
const signatureColor = (s: any) => {
  if (s === "official") return "green";
  if (s === "unsigned") return "orange";
  if (s === "untrusted") return "red";
  return "blue";
};
const healthColor = (s: any) => {
  const v = String(s || "").toLowerCase();
  if (!v) return "default";
  if (v === "ok") return "green";
  if (v === "degraded") return "orange";
  if (v === "error") return "red";
  return "blue";
};
const formatTime = (iso: string) => {
  try {
    const d = new Date(iso);
    return d.toLocaleString();
  } catch {
    return iso;
  }
};

const toggleEnabled = async (record: any, checked: boolean) => {
  const key = `${record.category}/${record.plugin_id}/${record.instance_id || "default"}`;
  busyKey.value = key;
  try {
    if (checked) await enableAdminPluginInstance(record.category, record.plugin_id, record.instance_id || "default");
    else await disableAdminPluginInstance(record.category, record.plugin_id, record.instance_id || "default");
    record.enabled = checked;
    message.success("操作成功");
  } catch (e: any) {
    message.error(e?.response?.data?.error || "操作失败");
  } finally {
    busyKey.value = "";
  }
};

const openCreateInstance = (record: PluginListItem) => {
  instanceTarget.value = record;
  instanceId.value = "";
  instanceOpen.value = true;
};

const confirmCreateInstance = async () => {
  if (!instanceTarget.value) return;
  instanceCreating.value = true;
  try {
    await createAdminPluginInstance(String(instanceTarget.value.category || ""), String(instanceTarget.value.plugin_id || ""), {
      instance_id: instanceId.value.trim()
    });
    message.success("OK");
    instanceOpen.value = false;
    await fetchData();
  } catch (e: any) {
    message.error(e?.response?.data?.error || "failed");
  } finally {
    instanceCreating.value = false;
  }
};

// Install flow
const pendingFile = ref<File | null>(null);
const adminPwdOpen = ref(false);
const adminPassword = ref("");

const tryInstall = async (file: File, pwd?: string) => {
  installing.value = true;
  try {
    await installAdminPlugin(file, pwd);
    message.success("安装成功");
    adminPwdOpen.value = false;
    adminPassword.value = "";
    pendingFile.value = null;
    fetchData();
    return true;
  } catch (e: any) {
    const status = e?.response?.status;
    const err = e?.response?.data?.error || "安装失败";
    if (status === 403 && String(err).includes("admin_password")) {
      pendingFile.value = file;
      adminPwdOpen.value = true;
      return false;
    }
    message.error(err);
    return false;
  } finally {
    installing.value = false;
  }
};

const onInstallUpload = async (opt: UploadRequestOption) => {
  const file = opt.file as File;
  const ok = await tryInstall(file);
  if (ok) opt.onSuccess?.({}, file as any);
  else opt.onError?.(new Error("install failed"));
};

const confirmInstall = async () => {
  if (!pendingFile.value) return;
  const pwd = adminPassword.value.trim();
  if (!pwd) {
    message.error("请输入管理员密码");
    return;
  }
  await tryInstall(pendingFile.value, pwd);
};

// Discover / import from disk
const discoverOpen = ref(false);
const discoverLoading = ref(false);
const discovered = ref<PluginDiscoverItem[]>([]);
const importBusyKey = ref("");
const importing = ref(false);
const importPwdOpen = ref(false);
const importAdminPassword = ref("");
const importTarget = ref<PluginDiscoverItem | null>(null);

const discoverColumns = [
  { title: "插件", key: "plugin" },
  { title: "类型", dataIndex: "category", key: "category", width: 110 },
  { title: "签名", key: "signature", width: 120 },
  { title: "平台", key: "platform", width: 220 },
  { title: "操作", key: "actions", width: 120 }
];

const openDiscover = async () => {
  discoverOpen.value = true;
  discoverLoading.value = true;
  try {
    const res = await discoverAdminPlugins();
    discovered.value = (res.data?.items || []) as PluginDiscoverItem[];
  } catch (e: any) {
    message.error(e?.response?.data?.error || "发现失败");
  } finally {
    discoverLoading.value = false;
  }
};

const doImport = async (item: PluginDiscoverItem, pwd?: string) => {
  const key = `${item.category}/${item.plugin_id}`;
  importBusyKey.value = key;
  importing.value = true;
  try {
    await importAdminPluginFromDisk(String(item.category || ""), String(item.plugin_id || ""), pwd);
    message.success("导入成功");
    await openDiscover();
    await fetchData();
  } catch (e: any) {
    const status = e?.response?.status;
    const err = e?.response?.data?.error || "导入失败";
    if (status === 403 && String(err).includes("admin_password")) {
      importTarget.value = item;
      importPwdOpen.value = true;
      return;
    }
    message.error(err);
  } finally {
    importBusyKey.value = "";
    importing.value = false;
  }
};

const startImport = async (item: PluginDiscoverItem) => {
  if (item.signature_status !== "official") {
    importTarget.value = item;
    importPwdOpen.value = true;
    return;
  }
  await doImport(item);
};

const confirmImport = async () => {
  if (!importTarget.value) return;
  const pwd = importAdminPassword.value.trim();
  if (!pwd) {
    message.error("请输入管理员密码");
    return;
  }
  importPwdOpen.value = false;
  const item = importTarget.value;
  importAdminPassword.value = "";
  importTarget.value = null;
  await doImport(item, pwd);
};

const uninstall = async (record: any) => {
  try {
    await deleteAdminPluginInstance(record.category, record.plugin_id, record.instance_id || "default");
    message.success("卸载成功");
    fetchData();
  } catch (e: any) {
    message.error(e?.response?.data?.error || "卸载失败");
  }
};

// Config flow
const configOpen = ref(false);
const schemaLoading = ref(false);
const schemaError = ref("");
const current = ref<PluginListItem | null>(null);
const schemaObj = ref<any>(null);
const uiObj = ref<any>(null);
const configModel = ref<Record<string, any>>({});

const prettyConfig = computed(() => {
  try {
    return JSON.stringify(configModel.value || {}, null, 2);
  } catch {
    return String(configModel.value || "");
  }
});

const safeJson = (s: string) => {
  try {
    return JSON.parse(String(s || "{}"));
  } catch {
    return null;
  }
};

const openConfig = async (record: PluginListItem) => {
  if (String(record.category || "") === "automation") {
    message.info("automation 插件配置已迁移到 商品类型 页面");
    window.location.href = "/admin/catalog";
    return;
  }
  current.value = record;
  configOpen.value = true;
  schemaLoading.value = true;
  schemaError.value = "";
  schemaObj.value = null;
  uiObj.value = null;
  configModel.value = {};
  try {
    const [schemaRes, cfgRes] = await Promise.all([
      getAdminPluginInstanceConfigSchema(record.category || "", record.plugin_id || "", record.instance_id || "default"),
      getAdminPluginInstanceConfig(record.category || "", record.plugin_id || "", record.instance_id || "default")
    ]);
    const sj = safeJson(schemaRes.data?.json_schema || "{}");
    const uj = safeJson(schemaRes.data?.ui_schema || "{}") || {};
    const cfg = safeJson(cfgRes.data?.config_json || "{}") || {};
    if (!sj || String(sj.type || "") !== "object") {
      schemaError.value = "插件 schema 无法解析或不是 object 类型";
    } else {
      schemaObj.value = sj;
      uiObj.value = uj;
    }
    configModel.value = cfg;
  } catch (e: any) {
    schemaError.value = e?.response?.data?.error || "加载配置失败";
  } finally {
    schemaLoading.value = false;
  }
};

const saveConfig = async () => {
  if (!current.value) return;
  saving.value = true;
  try {
    const payload = JSON.stringify(configModel.value || {});
    await updateAdminPluginInstanceConfig(
      current.value.category || "",
      current.value.plugin_id || "",
      current.value.instance_id || "default",
      payload
    );
    message.success("保存成功");
    configOpen.value = false;
    fetchData();
  } catch (e: any) {
    message.error(e?.response?.data?.error || "保存失败");
  } finally {
    saving.value = false;
  }
};

// Manifest drawer
const manifestOpen = ref(false);
const openManifest = (record: PluginListItem) => {
  current.value = record;
  manifestOpen.value = true;
};
const prettyManifest = computed(() => {
  try {
    return JSON.stringify(current.value?.manifest || {}, null, 2);
  } catch {
    return "";
  }
});

// Payment method toggles (host-managed)
const methodsOpen = ref(false);
const methodsLoading = ref(false);
const methodBusyKey = ref("");
const methodItems = ref<PluginPaymentMethodItem[]>([]);

const methodsColumns = [
  { title: "method", dataIndex: "method", key: "method" },
  { title: "enabled", key: "enabled", width: 120 }
];

const openPaymentMethods = async (record: PluginListItem) => {
  if (String(record.category || "") !== "payment") return;
  current.value = record;
  methodsOpen.value = true;
  methodsLoading.value = true;
  try {
    const res = await listAdminPluginPaymentMethods({
      category: String(record.category || "payment"),
      plugin_id: String(record.plugin_id || ""),
      instance_id: String(record.instance_id || "default")
    });
    methodItems.value = (res.data?.items || []) as PluginPaymentMethodItem[];
  } catch (e: any) {
    message.error(e?.response?.data?.error || "加载失败");
  } finally {
    methodsLoading.value = false;
  }
};

const toggleMethod = async (method: string, enabled: boolean) => {
  const rec = current.value;
  if (!rec) return;
  methodBusyKey.value = method;
  try {
    await updateAdminPluginPaymentMethod({
      category: String(rec.category || "payment"),
      plugin_id: String(rec.plugin_id || ""),
      instance_id: String(rec.instance_id || "default"),
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
</script>

<style scoped>
.plugins-page {
  padding: 24px;
}
.hero {
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  gap: 16px;
  padding: 18px 18px 16px;
  border-radius: 14px;
  background:
    radial-gradient(1100px 420px at 20% 10%, rgba(22, 119, 255, 0.14), transparent 60%),
    radial-gradient(900px 420px at 88% 30%, rgba(114, 46, 209, 0.14), transparent 55%),
    linear-gradient(180deg, rgba(255, 255, 255, 0.92), rgba(255, 255, 255, 0.88));
  border: 1px solid rgba(0, 0, 0, 0.06);
  margin-bottom: 14px;
}
.hero-title {
  font-size: 18px;
  font-weight: 800;
  letter-spacing: 0.2px;
}
.hero-subtle {
  margin-top: 4px;
  font-size: 12px;
  color: rgba(0, 0, 0, 0.5);
}
.hero-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}
.card {
  border-radius: 14px;
}
.filters {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}
.plugin-cell {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.plugin-name {
  font-weight: 700;
}
.plugin-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  color: rgba(0, 0, 0, 0.55);
}
.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
}
.health-cell {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.health-subtle {
  font-size: 12px;
  color: rgba(0, 0, 0, 0.45);
  line-height: 1.2;
}
.health-msg {
  color: rgba(0, 0, 0, 0.55);
}
.caps {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}
</style>
