<template>
  <div class="debug-page">
    <div class="page-header">
      <div>
        <div class="page-title">调试模式</div>
        <div class="subtle">用于排查自动化请求与系统日志问题</div>
      </div>
      <div class="page-actions">
        <a-button @click="fetchLogs" :loading="loading">
          <template #icon><ReloadOutlined /></template>
          刷新日志
        </a-button>
      </div>
    </div>

    <a-alert
      type="warning"
      message="警告"
      description="调试模式会记录详细的请求日志，仅用于问题排查，生产环境请谨慎使用。"
      show-icon
      style="margin-bottom: 16px"
    />

    <a-card :bordered="false" title="调试状态">
      <a-form layout="vertical">
        <a-form-item label="启用调试模式">
          <a-switch v-model:checked="debugEnabled" @change="handleToggleDebug" />
          <span class="hint">开启后将记录所有 API 请求的详细信息</span>
        </a-form-item>
      </a-form>
    </a-card>

    <a-card :bordered="false" title="日志保留策略（天）" style="margin-top: 16px">
      <a-form layout="vertical">
        <a-row :gutter="12">
          <a-col :xs="24" :md="8">
            <a-form-item label="自动化日志">
              <a-input-number v-model:value="retention.automation" :min="1" :max="3650" style="width: 100%" />
            </a-form-item>
          </a-col>
          <a-col :xs="24" :md="8">
            <a-form-item label="审计日志">
              <a-input-number v-model:value="retention.audit" :min="1" :max="3650" style="width: 100%" />
            </a-form-item>
          </a-col>
          <a-col :xs="24" :md="8">
            <a-form-item label="同步日志">
              <a-input-number v-model:value="retention.sync" :min="1" :max="3650" style="width: 100%" />
            </a-form-item>
          </a-col>
          <a-col :xs="24" :md="8">
            <a-form-item label="计划任务运行日志">
              <a-input-number v-model:value="retention.task_runs" :min="1" :max="3650" style="width: 100%" />
            </a-form-item>
          </a-col>
          <a-col :xs="24" :md="8">
            <a-form-item label="探针状态事件">
              <a-input-number v-model:value="retention.probe_events" :min="1" :max="3650" style="width: 100%" />
            </a-form-item>
          </a-col>
          <a-col :xs="24" :md="8">
            <a-form-item label="探针日志会话">
              <a-input-number v-model:value="retention.probe_sessions" :min="1" :max="3650" style="width: 100%" />
            </a-form-item>
          </a-col>
        </a-row>
      </a-form>
      <a-space>
        <a-button type="primary" :loading="savingRetention" @click="saveRetentionSettings">保存保留策略</a-button>
        <span class="hint">设置后由系统任务每天自动清理过期日志</span>
      </a-space>
    </a-card>

    <a-card :bordered="false" title="日志查询" style="margin-top: 16px">
      <a-tabs v-model:activeKey="activeLogTab">
        <a-tab-pane key="audit" tab="审计日志">
          <div class="tab-toolbar">
            <a-input
              v-model:value="auditFilter.keyword"
              allow-clear
              placeholder="搜索操作/目标"
              style="width: 240px"
            />
          </div>
          <a-table
            :columns="logColumns"
            :data-source="filteredAuditLogs"
            :loading="loading"
            :pagination="auditPagination"
            row-key="id"
            size="small"
            @change="(pager) => onTableChange('audit', pager)"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'created_at'">
                {{ formatDate(record.created_at) }}
              </template>
              <template v-else-if="column.key === 'detail'">
                <a-typography-text ellipsis :content="formatDetail(record.detail)" style="max-width: 320px" />
              </template>
            </template>
          </a-table>
        </a-tab-pane>

        <a-tab-pane key="automation" tab="自动化日志">
          <div class="tab-toolbar">
            <a-input
              v-model:value="automationFilter.api"
              allow-clear
              placeholder="按 API/动作筛选（模糊匹配）"
              style="width: 260px"
            />
          </div>
          <a-table
            :columns="automationColumns"
            :data-source="filteredAutomationLogs"
            :loading="loading"
            :pagination="automationPagination"
            row-key="id"
            size="small"
            @change="(pager) => onTableChange('automation', pager)"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'created_at'">
                {{ formatDate(record.created_at) }}
              </template>
              <template v-else-if="column.key === 'success'">
                <a-tag :color="record.success ? 'success' : 'error'">
                  {{ record.success ? '成功' : '失败' }}
                </a-tag>
              </template>
              <template v-else-if="column.key === 'message'">
                <a-typography-text ellipsis :content="record.message" style="max-width: 260px" />
              </template>
              <template v-else-if="column.key === 'action'">
                <a-typography-text ellipsis :content="record.action" style="max-width: 200px" />
              </template>
              <template v-else-if="column.key === 'protocol'">
                <a-tag :color="automationProtocol(record) === 'GRPC' ? 'purple' : 'blue'">
                  {{ automationProtocol(record) }}
                </a-tag>
              </template>
              <template v-else-if="column.key === 'connection'">
                <a-typography-text ellipsis :content="automationConnection(record)" style="max-width: 220px" />
              </template>
              <template v-else-if="column.key === 'detail'">
                <a-button type="link" @click="openDetail(record)">查看详情</a-button>
              </template>
            </template>
          </a-table>
        </a-tab-pane>

        <a-tab-pane key="sync" tab="同步日志">
          <div class="tab-toolbar">
            <a-input
              v-model:value="syncFilter.keyword"
              allow-clear
              placeholder="搜索目标/消息"
              style="width: 240px"
            />
          </div>
          <a-table
            :columns="syncColumns"
            :data-source="filteredSyncLogs"
            :loading="loading"
            :pagination="syncPagination"
            row-key="id"
            size="small"
            @change="(pager) => onTableChange('sync', pager)"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'created_at'">
                {{ formatDate(record.created_at) }}
              </template>
              <template v-else-if="column.key === 'status'">
                <a-tag :color="record.status === 'success' ? 'success' : 'error'">
                  {{ record.status }}
                </a-tag>
              </template>
              <template v-else-if="column.key === 'message'">
                <a-typography-text ellipsis :content="record.message" style="max-width: 320px" />
              </template>
            </template>
          </a-table>
        </a-tab-pane>
      </a-tabs>
    </a-card>

    <a-modal
      v-model:open="detailOpen"
      title="自动化请求详情"
      :width="1200"
      :style="{ maxWidth: '95vw', top: '20px' }"
      :footer="null"
      :destroy-on-close="true"
    >
      <div v-if="detailRecord" class="debug-modal">
        <!-- 顶部信息栏 -->
        <div class="modal-header-info">
          <a-space size="large">
            <span class="info-item">
              <span class="info-label">动作:</span>
              <span class="info-value">{{ detailRecord.action }}</span>
            </span>
            <span class="info-item">
              <span class="info-label">订单 ID:</span>
              <span class="info-value">{{ detailRecord.order_id || '-' }}</span>
            </span>
            <span class="info-item">
              <span class="info-label">协议:</span>
              <a-tag :color="detailProtocol === 'GRPC' ? 'purple' : 'blue'">{{ detailProtocol }}</a-tag>
            </span>
            <span class="info-item">
              <span class="info-label">连接:</span>
              <span class="info-value">{{ detailConnection }}</span>
            </span>
            <a-tag :color="detailRecord.success ? 'success' : 'error'">
              {{ detailRecord.success ? '成功' : '失败' }}
            </a-tag>
            <span class="info-item">
              <span class="info-value subtle">{{ formatDate(detailRecord.created_at) }}</span>
            </span>
          </a-space>
        </div>

        <!-- 消息 -->
        <a-alert
          :type="detailRecord.success ? 'success' : 'error'"
          :message="detailRecord.message || 'No message'"
          show-icon
        />

        <!-- 左右布局：Request | Response -->
        <div class="request-response-layout">
          <!-- 左侧：Request -->
          <div class="panel-left">
            <div class="panel-header">
              <span class="panel-title">Request</span>
            </div>
            <div class="panel-body">
              <HttpRequestBar
                :method="detailRequest.method || 'GET'"
                :url="detailRequest.url || '-'"
              />
              <a-tabs type="card" size="small" v-model:activeKey="requestActiveTab">
                <a-tab-pane key="headers" tab="Headers">
                  <HttpHeadersTable :headers="detailRequest.headers" copyable />
                </a-tab-pane>
                <a-tab-pane key="body" tab="Body">
                  <pre class="code-block">{{ formatJson(detailRequest.body) }}</pre>
                </a-tab-pane>
              </a-tabs>
            </div>
          </div>

          <!-- 右侧：Response -->
          <div class="panel-right">
            <div class="panel-header">
              <span class="panel-title">Response</span>
            </div>
            <div class="panel-body">
              <HttpRequestBar
                method="RESP"
                :status="detailResponse.status"
                :duration="detailResponse.duration_ms"
              />
              <a-tabs type="card" size="small" v-model:activeKey="responseActiveTab">
                <a-tab-pane key="headers" tab="Headers">
                  <HttpHeadersTable :headers="detailResponse.headers" copyable />
                </a-tab-pane>
                <a-tab-pane key="body" tab="Body">
                  <template v-if="responsePreview.type === 'json'">
                    <pre class="code-block">{{ responsePreview.content }}</pre>
                  </template>
                  <template v-else-if="responsePreview.type === 'html'">
                    <div class="html-preview">
                      <iframe :srcdoc="responsePreview.content" />
                    </div>
                  </template>
                  <template v-else>
                    <pre class="code-block">{{ responsePreview.content }}</pre>
                  </template>
                </a-tab-pane>
              </a-tabs>
            </div>
          </div>
        </div>

        <!-- 底部操作 -->
        <div class="modal-footer">
          <a-space>
            <a-button @click="copyAllDetails">
              <template #icon><CopyOutlined /></template>
              复制全部
            </a-button>
            <a-button type="primary" @click="detailOpen = false">关闭</a-button>
          </a-space>
        </div>
      </div>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed, watch } from "vue";
import { message } from "ant-design-vue";
import { ReloadOutlined, CopyOutlined } from "@ant-design/icons-vue";
import { getDebugStatus, getDebugLogs, listSettings, updateDebugStatus, updateSetting } from "@/services/admin";
import HttpHeadersTable from "@/components/HttpHeadersTable.vue";
import HttpRequestBar from "@/components/HttpRequestBar.vue";

const loading = ref(false);
const debugEnabled = ref(false);
const savingRetention = ref(false);
const activeLogTab = ref("audit");
const auditLogs = ref<any[]>([]);
const automationLogs = ref<any[]>([]);
const syncLogs = ref<any[]>([]);

const detailOpen = ref(false);
const detailRecord = ref<any | null>(null);
const requestActiveTab = ref('headers');
const responseActiveTab = ref('headers');

const auditFilter = reactive({ keyword: "" });
const automationFilter = reactive({ api: "" });
const syncFilter = reactive({ keyword: "" });

const auditPagination = reactive({ current: 1, pageSize: 20, total: 0, showSizeChanger: true });
const automationPagination = reactive({ current: 1, pageSize: 20, total: 0, showSizeChanger: true });
const syncPagination = reactive({ current: 1, pageSize: 20, total: 0, showSizeChanger: true });
const retention = reactive({
  automation: 30,
  audit: 90,
  sync: 30,
  task_runs: 14,
  probe_events: 30,
  probe_sessions: 7
});

const logColumns = [
  { title: "ID", dataIndex: "id", key: "id", width: 80 },
  { title: "管理员ID", dataIndex: "admin_id", key: "admin_id", width: 100 },
  { title: "操作", dataIndex: "action", key: "action", width: 150 },
  { title: "目标类型", dataIndex: "target_type", key: "target_type", width: 100 },
  { title: "目标ID", dataIndex: "target_id", key: "target_id", width: 100 },
  { title: "详情", dataIndex: "detail", key: "detail" },
  { title: "时间", dataIndex: "created_at", key: "created_at", width: 180 }
];

const automationColumns = [
  { title: "ID", dataIndex: "id", key: "id", width: 80 },
  { title: "API", dataIndex: "action", key: "action", width: 200 },
  { title: "协议", dataIndex: "protocol", key: "protocol", width: 90 },
  { title: "连接", dataIndex: "connection", key: "connection", width: 220 },
  { title: "订单ID", dataIndex: "order_id", key: "order_id", width: 100 },
  { title: "结果", dataIndex: "success", key: "success", width: 80 },
  { title: "消息", dataIndex: "message", key: "message" },
  { title: "时间", dataIndex: "created_at", key: "created_at", width: 180 },
  { title: "详情", dataIndex: "detail", key: "detail", width: 90 }
];

const syncColumns = [
  { title: "ID", dataIndex: "id", key: "id", width: 80 },
  { title: "目标", dataIndex: "target", key: "target", width: 150 },
  { title: "模式", dataIndex: "mode", key: "mode", width: 100 },
  { title: "状态", dataIndex: "status", key: "status", width: 100 },
  { title: "消息", dataIndex: "message", key: "message" },
  { title: "时间", dataIndex: "created_at", key: "created_at", width: 180 }
];

const formatDate = (date: string) => {
  if (!date) return "-";
  return new Date(date).toLocaleString("zh-CN");
};

const formatDetail = (detail: any) => {
  if (!detail) return "-";
  if (typeof detail === "string") return detail;
  try {
    return JSON.stringify(detail);
  } catch (e) {
    return String(detail);
  }
};

const parsePayload = (payload: any) => {
  if (!payload) return {};
  if (typeof payload === "string") {
    try {
      return JSON.parse(payload);
    } catch (e) {
      return { body: payload };
    }
  }
  return payload;
};

const normalizeHeaders = (headers: any): Record<string, string> => {
  if (!headers || typeof headers !== "object") return {};
  const out: Record<string, string> = {};
  Object.entries(headers).forEach(([key, value]) => {
    out[String(key)] = value == null ? "" : String(value);
  });
  return out;
};

const requestProtocol = (request: any): string => {
  const method = String(request?.method || "").trim().toUpperCase();
  const headers = normalizeHeaders(request?.headers);
  if (method) return method;
  if (String(headers["x-transport"] || "").trim()) {
    return String(headers["x-transport"]).trim().toUpperCase();
  }
  return "UNKNOWN";
};

const requestConnection = (request: any): string => {
  const headers = normalizeHeaders(request?.headers);
  const pluginID = String(headers["x-plugin-id"] || "").trim();
  const instanceID = String(headers["x-plugin-instance-id"] || "").trim();
  if (pluginID || instanceID) {
    return `${pluginID || "-"} / ${instanceID || "-"}`;
  }
  const urlText = String(request?.url || "").trim();
  if (!urlText) return "-";
  try {
    const parsed = new URL(urlText);
    return parsed.host || urlText;
  } catch {
    return urlText;
  }
};

const formatJson = (payload: any) => {
  if (payload === undefined || payload === null) return "-";
  if (typeof payload === "string") return payload;
  try {
    return JSON.stringify(payload, null, 2);
  } catch (e) {
    return String(payload);
  }
};

const responsePreview = computed(() => {
  const response = detailResponse.value;
  if (!response || !response.body) {
    return { type: "text", content: "-" };
  }
  if (response.body_json) {
    return { type: "json", content: formatJson(response.body_json) };
  }
  const bodyText = typeof response.body === "string" ? response.body : formatJson(response.body);
  const lowered = bodyText.trim().toLowerCase();
  if (response.format === "html" || lowered.startsWith("<!doctype") || lowered.startsWith("<html")) {
    return { type: "html", content: bodyText };
  }
  return { type: "text", content: bodyText };
});

const detailRequest = computed(() => parsePayload(detailRecord.value?.request_json));
const detailResponse = computed(() => parsePayload(detailRecord.value?.response_json));
const detailProtocol = computed(() => requestProtocol(detailRequest.value));
const detailConnection = computed(() => requestConnection(detailRequest.value));

const automationProtocol = (record: any) => {
  const req = parsePayload(record?.request_json);
  return requestProtocol(req);
};

const automationConnection = (record: any) => {
  const req = parsePayload(record?.request_json);
  return requestConnection(req);
};

const filteredAutomationLogs = computed(() => {
  const api = automationFilter.api.trim().toLowerCase();
  if (!api) return automationLogs.value;
  return automationLogs.value.filter((item) => String(item.action || "").toLowerCase().includes(api));
});

const filteredAuditLogs = computed(() => {
  const keyword = auditFilter.keyword.trim().toLowerCase();
  if (!keyword) return auditLogs.value;
  return auditLogs.value.filter((item) => {
    const action = String(item.action || "").toLowerCase();
    const target = String(item.target_type || "").toLowerCase() + ":" + String(item.target_id || "").toLowerCase();
    return action.includes(keyword) || target.includes(keyword);
  });
});

const filteredSyncLogs = computed(() => {
  const keyword = syncFilter.keyword.trim().toLowerCase();
  if (!keyword) return syncLogs.value;
  return syncLogs.value.filter((item) => {
    const target = String(item.target || "").toLowerCase();
    const message = String(item.message || "").toLowerCase();
    return target.includes(keyword) || message.includes(keyword);
  });
});

// 复制到剪贴板工具函数
const copyToClipboard = async (text: string, label: string = '内容') => {
  try {
    await navigator.clipboard.writeText(text);
    message.success(`${label}已复制到剪贴板`);
  } catch {
    message.error('复制失败');
  }
};

// 复制全部详情
const copyAllDetails = () => {
  if (!detailRecord.value) return;
  const details = {
    action: detailRecord.value.action,
    order_id: detailRecord.value.order_id,
    success: detailRecord.value.success,
    message: detailRecord.value.message,
    request: detailRequest.value,
    response: detailResponse.value,
    created_at: detailRecord.value.created_at
  };
  copyToClipboard(JSON.stringify(details, null, 2), '全部详情');
};

// 获取状态码颜色
const getStatusColor = (status: number) => {
  if (status >= 200 && status < 300) return 'success';
  if (status >= 300 && status < 400) return 'processing';
  if (status >= 400 && status < 500) return 'warning';
  if (status >= 500) return 'error';
  return 'default';
};

const openDetail = (record: any) => {
  detailRecord.value = record;
  detailOpen.value = true;
};

const fetchStatus = async () => {
  try {
    const res = await getDebugStatus();
    debugEnabled.value = res.data?.enabled || false;
  } catch (error) {
    console.error("Failed to fetch debug status:", error);
  }
};

const handleToggleDebug = async (checked: boolean) => {
  try {
    await updateDebugStatus({ enabled: checked });
    message.success(checked ? "调试模式已启用" : "调试模式已禁用");
  } catch (error: any) {
    message.error(error.response?.data?.error || "操作失败");
    debugEnabled.value = !checked;
  }
};

const parseDays = (value: any, fallback: number) => {
  const n = Number(value);
  if (!Number.isFinite(n) || n <= 0) return fallback;
  if (n > 3650) return 3650;
  return Math.trunc(n);
};

const fetchRetentionSettings = async () => {
  try {
    const res = await listSettings();
    const items = (res.data?.items || []) as Array<{ key?: string; value?: string }>;
    const map = new Map<string, string>();
    for (const item of items) {
      const key = String(item?.key || "");
      if (!key) continue;
      map.set(key, String(item?.value || ""));
    }
    retention.automation = parseDays(map.get("automation_log_retention_days"), 30);
    retention.audit = parseDays(map.get("audit_log_retention_days"), 90);
    retention.sync = parseDays(map.get("integration_sync_log_retention_days"), 30);
    retention.task_runs = parseDays(map.get("scheduled_task_run_retention_days"), 14);
    retention.probe_events = parseDays(map.get("probe_status_event_retention_days"), 30);
    retention.probe_sessions = parseDays(map.get("probe_log_session_retention_days"), 7);
  } catch (error) {
    console.error("Failed to fetch retention settings:", error);
  }
};

const saveRetentionSettings = async () => {
  savingRetention.value = true;
  try {
    await updateSetting({
      items: [
        { key: "automation_log_retention_days", value: String(parseDays(retention.automation, 30)) },
        { key: "audit_log_retention_days", value: String(parseDays(retention.audit, 90)) },
        { key: "integration_sync_log_retention_days", value: String(parseDays(retention.sync, 30)) },
        { key: "scheduled_task_run_retention_days", value: String(parseDays(retention.task_runs, 14)) },
        { key: "probe_status_event_retention_days", value: String(parseDays(retention.probe_events, 30)) },
        { key: "probe_log_session_retention_days", value: String(parseDays(retention.probe_sessions, 7)) }
      ]
    });
    message.success("日志保留策略已保存");
  } catch (error: any) {
    message.error(error.response?.data?.error || "保存失败");
  } finally {
    savingRetention.value = false;
  }
};

const fetchLogs = async (type = activeLogTab.value) => {
  loading.value = true;
  try {
    let pager = auditPagination;
    if (type === "automation") pager = automationPagination;
    if (type === "sync") pager = syncPagination;
    const res = await getDebugLogs({
      types: type,
      limit: pager.pageSize,
      offset: (pager.current - 1) * pager.pageSize,
      page: pager.current,
      pages: pager.pageSize
    });
    if (type === "audit") {
      auditLogs.value = res.data?.audit_logs?.items || [];
      auditPagination.total = res.data?.audit_logs?.total || auditLogs.value.length;
    }
    if (type === "automation") {
      automationLogs.value = res.data?.automation_logs?.items || [];
      automationPagination.total = res.data?.automation_logs?.total || automationLogs.value.length;
    }
    if (type === "sync") {
      syncLogs.value = res.data?.sync_logs?.items || [];
      syncPagination.total = res.data?.sync_logs?.total || syncLogs.value.length;
    }
  } finally {
    loading.value = false;
  }
};

const onTableChange = (type: string, pager: any) => {
  if (type === "audit") {
    auditPagination.current = pager.current;
    auditPagination.pageSize = pager.pageSize;
  }
  if (type === "automation") {
    automationPagination.current = pager.current;
    automationPagination.pageSize = pager.pageSize;
  }
  if (type === "sync") {
    syncPagination.current = pager.current;
    syncPagination.pageSize = pager.pageSize;
  }
  fetchLogs(type);
};

watch(activeLogTab, (tab) => {
  fetchLogs(tab);
});

onMounted(() => {
  fetchStatus();
  fetchRetentionSettings();
  fetchLogs("audit");
});
</script>

<style scoped>
.debug-page {
  padding: 24px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.page-title {
  font-size: 20px;
  font-weight: 600;
  margin: 0;
}

.page-actions {
  display: flex;
  gap: 12px;
}

.hint {
  margin-left: 8px;
  color: rgba(0, 0, 0, 0.45);
}

.tab-toolbar {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 12px;
}

/* Modal 样式 */
.debug-modal {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

/* 顶部信息栏 */
.modal-header-info {
  padding: 14px 16px;
  background: #ffffff;
  border-radius: 12px;
  border: 1px solid rgba(0, 0, 0, 0.06);
}

.info-item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.info-label {
  font-size: 12px;
  color: rgba(0, 0, 0, 0.45);
  font-weight: 400;
}

.info-value {
  font-size: 13px;
  color: rgba(0, 0, 0, 0.88);
  font-weight: 500;
}

.subtle {
  color: rgba(0, 0, 0, 0.45);
}

/* 左右布局 */
.request-response-layout {
  display: flex;
  gap: 20px;
  min-height: 440px;
}

.panel-left,
.panel-right {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  background: #ffffff;
  border: 1px solid rgba(0, 0, 0, 0.08);
  border-radius: 12px;
  overflow: hidden;
}

.panel-header {
  padding: 14px 18px;
  background: rgba(0, 0, 0, 0.02);
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
}

.panel-title {
  font-size: 13px;
  font-weight: 600;
  color: rgba(0, 0, 0, 0.88);
  letter-spacing: 0.3px;
}

.panel-body {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: #fafbfc;
}

.panel-body :deep(.http-request-bar) {
  margin: 12px;
  border-radius: 6px;
}

.panel-body :deep(.ant-tabs) {
  flex: 1;
  display: flex;
  flex-direction: column;
  margin: 0 12px 12px 12px;
}

.panel-body :deep(.ant-tabs-content-holder) {
  flex: 1;
  overflow: hidden;
}

.panel-body :deep(.ant-tabs-content) {
  height: 100%;
}

.panel-body :deep(.ant-tabs-tabpane) {
  height: 100%;
}

.modal-footer {
  margin-top: 12px;
  padding-top: 16px;
  border-top: 1px solid rgba(0, 0, 0, 0.06);
  display: flex;
  justify-content: flex-end;
}

/* Code block 优化 */
.code-block {
  background: #1e1e1e;
  color: #d4d4d4;
  padding: 16px;
  border-radius: 8px;
  font-size: 12px;
  line-height: 1.6;
  font-family: 'SFMono-Regular', 'Consolas', 'Liberation Mono', Menlo, monospace;
  white-space: pre-wrap;
  word-break: break-all;
  max-height: 300px;
  overflow-y: auto;
}

.code-block::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

.code-block::-webkit-scrollbar-track {
  background: transparent;
}

.code-block::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.2);
  border-radius: 4px;
}

.code-block::-webkit-scrollbar-thumb:hover {
  background: rgba(255, 255, 255, 0.3);
}

.html-preview {
  border: 1px solid rgba(0, 0, 0, 0.08);
  border-radius: 8px;
  overflow: hidden;
  height: 300px;
  background: #fff;
}

.html-preview iframe {
  width: 100%;
  height: 100%;
  border: none;
}

/* Tabs 样式 - 现代简洁 */
.panel-body :deep(.ant-tabs) {
  background: transparent;
}

.panel-body :deep(.ant-tabs-nav) {
  margin-bottom: 0;
}

.panel-body :deep(.ant-tabs-tab) {
  padding: 10px 18px;
  margin: 0 4px 0 0;
  background: transparent;
  border: none;
  border-radius: 8px 8px 0 0;
  color: rgba(0, 0, 0, 0.65);
  font-size: 13px;
  font-weight: 500;
  transition: all 0.15s ease;
}

.panel-body :deep(.ant-tabs-tab:hover) {
  color: rgba(0, 0, 0, 0.88);
  background: rgba(0, 0, 0, 0.03);
}

.panel-body :deep(.ant-tabs-tab-active) {
  color: #1677ff;
  background: #fff;
}

.panel-body :deep(.ant-tabs-ink-bar) {
  display: none;
}

.panel-body :deep(.ant-tabs-content) {
  background: #fff;
  border-radius: 0 0 8px 8px;
  padding: 16px;
}

.panel-body :deep(.ant-tabs-tabpane) {
  height: 100%;
}

/* 响应式调整 */
@media (max-width: 1200px) {
  .request-response-layout {
    flex-direction: column;
  }

  .panel-left,
  .panel-right {
    min-height: 380px;
  }
}
</style>
