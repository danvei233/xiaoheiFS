<template>
  <div class="api-keys-page">
    <!-- Page Header -->
    <div class="page-header">
      <div class="page-header-left">
        <h1 class="page-title">{{ t('console.apiKeys.title') }}</h1>
        <p class="page-subtitle">{{ t('console.apiKeys.subtitle') }}</p>
      </div>
      <div class="page-header-actions">
        <a-button type="primary" @click="openCreate">
          <template #icon><PlusOutlined /></template>
          {{ t('console.apiKeys.create') }}
        </a-button>
      </div>
    </div>

    <!-- Info Banner -->
    <div class="info-banner">
      <div class="info-icon">
        <InfoCircleOutlined />
      </div>
      <div class="info-content">
        <div class="info-title">{{ t('console.apiKeys.signSpec') }}</div>
        <div class="info-desc">
          <span class="code-tag">X-AKID</span>
          <span class="code-tag">X-Timestamp</span>
          <span class="code-tag">X-Nonce</span>
          <span class="code-tag">X-Signature</span>
          <a-tag color="success" size="small">{{ t('console.apiKeys.timeWindow') }}</a-tag>
        </div>
      </div>
    </div>

    <!-- Table Card -->
    <div class="table-card">
      <div class="card-header">
        <div class="card-title">
          <KeyOutlined />
          <span>{{ t('console.apiKeys.list') }}</span>
        </div>
        <div class="table-count" v-if="rows.length > 0">{{ t('console.apiKeys.count', { n: rows.length }) }}</div>
      </div>
      <a-table
        :columns="columns"
        :data-source="rows"
        row-key="id"
        :loading="loading"
        :pagination="false"
        class="api-table"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'status'">
            <a-badge :status="record.status === 'active' ? 'success' : 'default'" :text="record.status === 'active' ? t('common.enabled') : t('common.disabled')" />
          </template>
          <template v-else-if="column.key === 'name'">
            <div class="name-cell">
              <div class="name-text">{{ record.name }}</div>
              <div class="akid-preview">{{ record.akid }}</div>
            </div>
          </template>
          <template v-else-if="column.key === 'akid'">
            <div class="akid-cell">
              <code class="akid-code">{{ record.akid }}</code>
              <a-tooltip :title="t('common.copy')">
                <a-button type="text" size="small" @click="copyText(record.akid)">
                  <CopyOutlined />
                </a-button>
              </a-tooltip>
            </div>
          </template>
          <template v-else-if="column.key === 'last_used_at'">
            <span class="time-cell">
              {{ record.last_used_at ? formatTime(record.last_used_at) : "—" }}
            </span>
          </template>
          <template v-else-if="column.key === 'action'">
            <a-space :size="8">
              <a-tooltip :title="record.status === 'active' ? t('console.apiKeys.disableKey') : t('console.apiKeys.enableKey')">
                <a-button @click="toggleStatus(record)" :type="record.status === 'active' ? 'default' : 'primary'" size="small">
                  <template #icon>
                    <component :is="record.status === 'active' ? StopOutlined : PlayCircleOutlined" />
                  </template>
                </a-button>
              </a-tooltip>
              <a-popconfirm :title="t('console.apiKeys.deleteConfirm')" @confirm="remove(record)" :ok-text="t('console.apiKeys.confirmDelete')" :cancel-text="t('common.cancel')">
                <a-button danger size="small">
                  <template #icon><DeleteOutlined /></template>
                </a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
        <template #emptyText>
          <a-empty description=" ">
            <template #description>
              <div class="empty-state">
                <KeyOutlined class="empty-icon" />
                <span>{{ t('console.apiKeys.noKeys') }}</span>
                <a-button type="link" @click="openCreate">{{ t('console.apiKeys.createFirst') }}</a-button>
              </div>
            </template>
          </a-empty>
        </template>
      </a-table>
    </div>

    <!-- Code Example Card -->
    <div class="code-card">
      <div class="card-header">
        <div class="card-title">
          <CodeOutlined />
          <span>{{ t('console.apiKeys.signExample') }}</span>
        </div>
        <a-tag color="blue">Node.js</a-tag>
      </div>
      <div class="code-wrapper">
        <pre class="code">{{ signSnippet }}</pre>
      </div>
    </div>

    <!-- Create Modal -->
    <a-modal
      v-model:open="createOpen"
      title=""
      @ok="create"
      :ok-text="t('console.apiKeys.createKey')"
      :cancel-text="t('common.cancel')"
      centered
    >
      <div class="modal-header">
        <div class="modal-icon">
          <PlusOutlined />
        </div>
        <h3>{{ t('console.apiKeys.createTitle') }}</h3>
        <p>{{ t('console.apiKeys.createDesc') }}</p>
      </div>
      <a-form layout="vertical">
        <a-form-item :label="t('console.apiKeys.keyName')" required>
          <a-input
            v-model:value="createForm.name"
            maxlength="64"
            :placeholder="t('console.apiKeys.keyNamePlaceholder')"
            size="large"
          />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- Key Modal -->
    <a-modal
      v-model:open="secretOpen"
      title=""
      :footer="null"
      centered
      :maskClosable="false"
    >
      <div class="secret-content">
        <a-alert
          type="warning"
          show-icon
          :message="t('console.apiKeys.important')"
          :description="t('console.apiKeys.keyWarning')"
          class="secret-alert"
        />
        <div class="secret-form">
          <div class="secret-field">
            <label class="field-label">{{ t('console.apiKeys.akid') }}</label>
            <div class="field-input-wrapper">
              <a-input :value="newKey.akid" readonly size="large" class="readonly-input" />
              <a-button @click="copyText(newKey.akid)">
                <CopyOutlined />
              </a-button>
            </div>
          </div>
          <div class="secret-field secret-field-highlight">
            <label class="field-label">{{ t('console.apiKeys.key') }}</label>
            <div class="field-input-wrapper">
              <a-input :value="newKey.key" readonly size="large" class="readonly-input" />
              <a-button type="primary" @click="copyKey">
                <CopyOutlined />
              </a-button>
            </div>
          </div>
        </div>
        <div class="secret-actions">
          <a-button size="large" @click="secretOpen = false">{{ t('console.apiKeys.closeWindow') }}</a-button>
          <a-button type="primary" size="large" @click="copyKey">
            <template #icon><CopyOutlined /></template>
            {{ t('console.apiKeys.copyKey') }}
          </a-button>
        </div>
      </div>
    </a-modal>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from "vue";
import { message } from "ant-design-vue";
import {
  PlusOutlined,
  InfoCircleOutlined,
  KeyOutlined,
  CopyOutlined,
  StopOutlined,
  PlayCircleOutlined,
  DeleteOutlined,
  CodeOutlined
} from "@ant-design/icons-vue";
import { createUserApiKey, deleteUserApiKey, listUserApiKeys, updateUserApiKeyStatus } from "@/services/user";
import { t, locale } from "@/locales";

const loading = ref(false);
const rows = ref([]);
const createOpen = ref(false);
const secretOpen = ref(false);
const createForm = reactive({ name: "" });
const newKey = reactive({ akid: "", key: "" });

const columns = computed(() => [
  { title: "ID", dataIndex: "id", key: "id", width: 70, className: "id-column" },
  { title: t('common.name'), dataIndex: "name", key: "name" },
  { title: "AKID", dataIndex: "akid", key: "akid", width: 280 },
  { title: t('common.status'), dataIndex: "status", key: "status", width: 110 },
  { title: t('console.apiKeys.lastUsed'), dataIndex: "last_used_at", key: "last_used_at", width: 170 },
  { title: t('common.actions'), key: "action", width: 150, className: "action-column" }
]);

const signSnippet = computed(
  () => `import crypto from "crypto";

const method = "POST";
const path = "/api/v1/open/orders/instant/create";
const query = "";
const ts = new Date().toISOString(); // RFC3339
const nonce = crypto.randomUUID().replace(/-/g, "");
const body = JSON.stringify({
  items: [{ package_id: 1, system_id: 1, qty: 1 }]
});

const bodyHash = crypto.createHash("sha256").update(body).digest("hex");
const canonical = [method.toUpperCase(), path, query, ts, nonce, bodyHash].join("\\n");
const sig = crypto.createHmac("sha256", process.env.OPEN_KEY)
                .update(canonical)
                .digest("hex");`
);

const formatTime = (v) => {
  const d = new Date(v);
  if (Number.isNaN(d.getTime())) return String(v || "");
  return d.toLocaleString(locale.value, {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  });
};

const fetchData = async () => {
  loading.value = true;
  try {
    const res = await listUserApiKeys({ limit: 100, offset: 0 });
    rows.value = res.data?.items || [];
  } finally {
    loading.value = false;
  }
};

const openCreate = () => {
  createForm.name = "";
  createOpen.value = true;
};

const create = async () => {
  const name = String(createForm.name || "").trim();
  if (!name) {
    message.error(t('console.apiKeys.nameRequired'));
    return;
  }
  const res = await createUserApiKey({ name, scopes: [] });
  createOpen.value = false;
  newKey.akid = String(res.data?.item?.akid || "");
  newKey.key = String(res.data?.key || res.data?.secret || "");
  secretOpen.value = true;
  await fetchData();
};

const toggleStatus = async (record) => {
  const next = record.status === "active" ? "disabled" : "active";
  await updateUserApiKeyStatus(record.id, { status: next });
  message.success(record.status === "active" ? t('console.apiKeys.keyDisabled') : t('console.apiKeys.keyEnabled'));
  await fetchData();
};

const remove = async (record) => {
  await deleteUserApiKey(record.id);
  message.success(t('console.apiKeys.keyDeleted'));
  await fetchData();
};

const copyKey = async () => {
  await navigator.clipboard.writeText(newKey.key || "");
  message.success(t('console.apiKeys.keyCopied'));
};

const copyText = async (text) => {
  await navigator.clipboard.writeText(text || "");
  message.success(t('common.copiedToClipboard'));
};

onMounted(fetchData);
</script>

<style scoped>
.api-keys-page {
  padding: 0;
}

/* Page Header */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 24px;
  padding-bottom: 20px;
  border-bottom: 1px solid var(--border);
}

.page-header-left {
  flex: 1;
}

.page-title {
  font-size: 24px;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0 0 6px 0;
  letter-spacing: -0.01em;
}

.page-subtitle {
  font-size: 14px;
  color: var(--text-secondary);
  margin: 0;
}

.page-header-actions {
  display: flex;
  gap: 12px;
}

/* Info Banner */
.info-banner {
  display: flex;
  gap: 16px;
  padding: 20px;
  background: var(--primary-bg);
  border: 1px solid rgba(0, 102, 255, 0.15);
  border-radius: var(--radius-md);
  margin-bottom: 24px;
}

.info-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  background: var(--primary-gradient);
  border-radius: 10px;
  color: white;
  font-size: 18px;
  flex-shrink: 0;
}

.info-content {
  flex: 1;
}

.info-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 8px;
}

.info-desc {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  font-size: 13px;
  color: var(--text-secondary);
}

.code-tag {
  display: inline-flex;
  align-items: center;
  padding: 4px 10px;
  background: var(--bg-tertiary);
  border: 1px solid var(--border);
  border-radius: 6px;
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 12px;
  color: var(--primary);
}

/* Card Styles */
.table-card,
.code-card {
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  margin-bottom: 24px;
  overflow: hidden;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border);
}

.card-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 600;
  color: var(--text-primary);
}

.card-title .anticon {
  font-size: 16px;
  color: var(--primary);
}

.table-count {
  font-size: 13px;
  color: var(--text-secondary);
  padding: 4px 12px;
  background: var(--bg-secondary);
  border-radius: 12px;
}

/* Table Styles */
.api-table {
  border: none;
}

.api-table :deep(.ant-table) {
  background: transparent;
}

.api-table :deep(.ant-table-thead > tr > th) {
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border);
  color: var(--text-secondary);
  font-weight: 600;
  font-size: 13px;
  padding: 16px;
}

.api-table :deep(.ant-table-tbody > tr > td) {
  background: transparent;
  border-bottom: 1px solid var(--border);
  padding: 16px;
  transition: background 0.2s ease;
}

.api-table :deep(.ant-table-tbody > tr:hover > td) {
  background: var(--bg-secondary);
}

.api-table :deep(.ant-table-tbody > tr:last-child > td) {
  border-bottom: none;
}

.api-table :deep(.id-column) {
  color: var(--text-tertiary);
  font-size: 13px;
}

.api-table :deep(.action-column) {
  text-align: right;
}

/* Name Cell */
.name-cell {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.name-text {
  font-weight: 500;
  color: var(--text-primary);
}

.akid-preview {
  font-size: 12px;
  color: var(--text-tertiary);
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
}

/* AKID Cell */
.akid-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.akid-code {
  padding: 4px 8px;
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: 6px;
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 12px;
  color: var(--primary);
}

/* Time Cell */
.time-cell {
  font-size: 13px;
  color: var(--text-secondary);
}

/* Empty State */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 48px 24px;
  color: var(--text-secondary);
  gap: 12px;
}

.empty-icon {
  font-size: 48px;
  color: var(--text-tertiary);
}

/* Code Card */
.code-wrapper {
  padding: 20px;
  background: var(--bg-secondary);
}

.code {
  margin: 0;
  padding: 0;
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 13px;
  line-height: 1.7;
  color: var(--text-primary);
  white-space: pre-wrap;
  word-break: break-word;
  overflow-x: auto;
}

/* Modal Styles */
.modal-header {
  text-align: center;
  margin-bottom: 24px;
}

.modal-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 56px;
  height: 56px;
  margin: 0 auto 16px;
  background: var(--primary-gradient);
  border-radius: 12px;
  color: white;
  font-size: 24px;
}

.modal-header h3 {
  margin: 0 0 8px 0;
  font-size: 18px;
  font-weight: 700;
  color: var(--text-primary);
}

.modal-header p {
  margin: 0;
  font-size: 13px;
  color: var(--text-secondary);
}

:deep(.ant-modal-header) {
  display: none;
}

:deep(.ant-modal-body) {
  padding: 28px;
}

:deep(.ant-modal-footer) {
  padding-top: 16px;
  border-top: 1px solid var(--border);
}

/* Secret Content */
.secret-content {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.secret-alert {
  flex-shrink: 0;
}

.secret-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.secret-field {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.secret-field-highlight {
  position: relative;
}

.secret-field-highlight::before {
  content: '';
  position: absolute;
  left: -12px;
  top: 0;
  bottom: 0;
  width: 3px;
  background: var(--warning);
  border-radius: 2px;
}

.field-label {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-secondary);
}

.field-input-wrapper {
  display: flex;
  gap: 8px;
}

.field-input-wrapper .ant-input-wrapper {
  flex: 1;
}

.readonly-input :deep(.ant-input) {
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 13px;
  color: var(--text-primary);
}

.secret-actions {
  display: flex;
  gap: 12px;
  justify-content: flex-end;
  padding-top: 8px;
  border-top: 1px solid var(--border);
}

.secret-actions .ant-btn {
  flex: 1;
}

/* Responsive */
@media (max-width: 1024px) {
  .page-header {
    flex-direction: column;
    gap: 16px;
  }

  .page-header-actions {
    width: 100%;
  }

  .page-header-actions .ant-btn {
    flex: 1;
  }
}

@media (max-width: 768px) {
  .page-title {
    font-size: 20px;
  }

  .info-banner {
    flex-direction: column;
    align-items: flex-start;
  }

  .info-desc {
    gap: 8px;
  }

  .card-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }

  .secret-actions {
    flex-direction: column;
  }
}
</style>
