<template>
  <div class="api-keys-page">
    <!-- Page Header -->
    <div class="page-header">
      <div class="page-header-left">
        <h1 class="page-title">API 密钥管理</h1>
        <p class="page-subtitle">管理用于开放接口鉴权的 AKID/Secret 凭证</p>
      </div>
      <div class="page-header-actions">
        <a-button type="primary" @click="openCreate">
          <template #icon><PlusOutlined /></template>
          创建新密钥
        </a-button>
      </div>
    </div>

    <!-- Info Banner -->
    <div class="info-banner">
      <div class="info-icon">
        <InfoCircleOutlined />
      </div>
      <div class="info-content">
        <div class="info-title">签名鉴权规范</div>
        <div class="info-desc">
          <span class="code-tag">X-AKID</span>
          <span class="code-tag">X-Timestamp</span>
          <span class="code-tag">X-Nonce</span>
          <span class="code-tag">X-Signature</span>
          <a-tag color="success" size="small">时间窗: ±300 秒</a-tag>
        </div>
      </div>
    </div>

    <!-- Table Card -->
    <div class="table-card">
      <div class="card-header">
        <div class="card-title">
          <KeyOutlined />
          <span>密钥列表</span>
        </div>
        <div class="table-count" v-if="rows.length > 0">{{ rows.length }} 个密钥</div>
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
            <a-badge :status="record.status === 'active' ? 'success' : 'default'" :text="record.status === 'active' ? '已启用' : '已停用'" />
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
              <a-tooltip title="复制">
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
              <a-tooltip :title="record.status === 'active' ? '停用密钥' : '启用密钥'">
                <a-button @click="toggleStatus(record)" :type="record.status === 'active' ? 'default' : 'primary'" size="small">
                  <template #icon>
                    <component :is="record.status === 'active' ? StopOutlined : PlayCircleOutlined" />
                  </template>
                </a-button>
              </a-tooltip>
              <a-popconfirm title="确认删除该密钥？删除后无法恢复" @confirm="remove(record)" ok-text="确认删除" cancel-text="取消">
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
                <span>暂无 API 密钥</span>
                <a-button type="link" @click="openCreate">创建第一个密钥</a-button>
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
          <span>签名算法示例</span>
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
      :ok-text="'创建密钥'"
      :cancel-text="'取消'"
      centered
    >
      <div class="modal-header">
        <div class="modal-icon">
          <PlusOutlined />
        </div>
        <h3>创建新 API 密钥</h3>
        <p>生成新的 AKID/Secret 凭证对，Secret 仅在创建时展示一次</p>
      </div>
      <a-form layout="vertical">
        <a-form-item label="密钥名称" required>
          <a-input
            v-model:value="createForm.name"
            maxlength="64"
            placeholder="例如：生产环境-订单服务"
            size="large"
          />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- Secret Modal -->
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
          message="重要提示"
          description="窗口关闭后将无法再次查看 Secret，请务必立即复制保存"
          class="secret-alert"
        />
        <div class="secret-form">
          <div class="secret-field">
            <label class="field-label">AKID (访问密钥ID)</label>
            <div class="field-input-wrapper">
              <a-input :value="newKey.akid" readonly size="large" class="readonly-input" />
              <a-button @click="copyText(newKey.akid)">
                <CopyOutlined />
              </a-button>
            </div>
          </div>
          <div class="secret-field secret-field-highlight">
            <label class="field-label">Secret (访问密钥)</label>
            <div class="field-input-wrapper">
              <a-input :value="newKey.secret" readonly size="large" class="readonly-input" />
              <a-button type="primary" @click="copySecret">
                <CopyOutlined />
              </a-button>
            </div>
          </div>
        </div>
        <div class="secret-actions">
          <a-button size="large" @click="secretOpen = false">关闭窗口</a-button>
          <a-button type="primary" size="large" @click="copySecret">
            <template #icon><CopyOutlined /></template>
            复制 Secret
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

const loading = ref(false);
const rows = ref([]);
const createOpen = ref(false);
const secretOpen = ref(false);
const createForm = reactive({ name: "" });
const newKey = reactive({ akid: "", secret: "" });

const columns = [
  { title: "ID", dataIndex: "id", key: "id", width: 70, className: "id-column" },
  { title: "名称", dataIndex: "name", key: "name" },
  { title: "AKID", dataIndex: "akid", key: "akid", width: 280 },
  { title: "状态", dataIndex: "status", key: "status", width: 110 },
  { title: "最近使用", dataIndex: "last_used_at", key: "last_used_at", width: 170 },
  { title: "操作", key: "action", width: 150, className: "action-column" }
];

const signSnippet = computed(
  () => `import crypto from "crypto";

const method = "POST";
const path = "/api/v1/open/orders/instant/create";
const query = "";
const ts = String(Math.floor(Date.now() / 1000));
const nonce = crypto.randomUUID().replace(/-/g, "");
const body = JSON.stringify({
  items: [{ package_id: 1, system_id: 1, qty: 1 }]
});

const canonical = [method.toUpperCase(), path, query, ts, nonce, body].join("\\n");
const sig = crypto.createHmac("sha256", process.env.OPEN_SECRET)
                .update(canonical)
                .digest("hex");`
);

const formatTime = (v) => {
  const d = new Date(v);
  if (Number.isNaN(d.getTime())) return String(v || "");
  return d.toLocaleString('zh-CN', {
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
    message.error("请输入密钥名称");
    return;
  }
  const res = await createUserApiKey({ name, scopes: [] });
  createOpen.value = false;
  newKey.akid = String(res.data?.item?.akid || "");
  newKey.secret = String(res.data?.secret || "");
  secretOpen.value = true;
  await fetchData();
};

const toggleStatus = async (record) => {
  const next = record.status === "active" ? "disabled" : "active";
  await updateUserApiKeyStatus(record.id, { status: next });
  message.success(record.status === "active" ? "密钥已停用" : "密钥已启用");
  await fetchData();
};

const remove = async (record) => {
  await deleteUserApiKey(record.id);
  message.success("密钥已删除");
  await fetchData();
};

const copySecret = async () => {
  await navigator.clipboard.writeText(newKey.secret || "");
  message.success("Secret 已复制到剪贴板");
};

const copyText = async (text) => {
  await navigator.clipboard.writeText(text || "");
  message.success("已复制到剪贴板");
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