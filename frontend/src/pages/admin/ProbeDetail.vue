<template>
  <div class="probe-detail-page">
    <div class="page-header">
      <a-breadcrumb>
        <a-breadcrumb-item @click="$router.push('/admin/probes')" class="clickable">探针列表</a-breadcrumb-item>
        <a-breadcrumb-item>{{ probe?.name || '详情' }}</a-breadcrumb-item>
      </a-breadcrumb>
    </div>

    <a-card :bordered="false" class="header-card">
      <template #extra>
        <a-space>
          <a-tag color="blue">
            <template #icon><ClockCircleOutlined /></template>
            自动刷新 5s
          </a-tag>
          <a-button @click="refreshAll(true)" :loading="loading">
            <template #icon><ReloadOutlined /></template>
            手动刷新
          </a-button>
        </a-space>
      </template>

      <a-row :gutter="24" align="middle">
        <a-col :span="16">
          <div class="probe-title">
            <div class="name">{{ probe?.name || '-' }}</div>
            <div class="meta">
              <a-tag :color="probe?.status === 'online' ? 'success' : 'default'" size="small">
                {{ probe?.status === 'online' ? '在线' : '离线' }}
              </a-tag>
              <span class="subtle">ID: {{ probe?.id || '-' }}</span>
              <span class="subtle">Agent: {{ probe?.agent_id || '-' }}</span>
            </div>
          </div>
        </a-col>
        <a-col :span="8">
          <div class="last-refresh">
            <ClockCircleOutlined class="icon" />
            <span>上次刷新：{{ formatDate(lastRefreshAt) }}</span>
          </div>
        </a-col>
      </a-row>
    </a-card>

    <a-row :gutter="16" class="metrics-row">
      <a-col :span="6">
        <a-card :bordered="false" class="metric-card">
          <template #title>
            <span class="metric-title"><DesktopOutlined /> 系统运行时长</span>
          </template>
          <div class="metric-value">{{ formatUptime(snapshot?.system?.uptime) }}</div>
          <div class="metric-label">持续运行中</div>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card :bordered="false" class="metric-card">
          <template #title>
            <span class="metric-title"><HeartOutlined /> 7天 SLA</span>
          </template>
          <div class="metric-value">{{ Number(sla?.uptime_percent || 0).toFixed(2) }}%</div>
          <div class="metric-label">在线 {{ sla?.online_seconds || 0 }} 秒</div>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card :bordered="false" class="metric-card">
          <template #title>
            <span class="metric-title"><ClockCircleOutlined /> 最后心跳</span>
          </template>
          <div class="metric-value small">{{ formatDateShort(probe?.last_heartbeat_at) }}</div>
          <div class="metric-label">{{ fromNow(probe?.last_heartbeat_at) }}</div>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card :bordered="false" class="metric-card">
          <template #title>
            <span class="metric-title"><CloudOutlined /> 最后快照</span>
          </template>
          <div class="metric-value small">{{ formatDateShort(probe?.last_snapshot_at) }}</div>
          <div class="metric-label">{{ fromNow(probe?.last_snapshot_at) }}</div>
        </a-card>
      </a-col>
    </a-row>

    <a-row :gutter="16" class="charts-row">
      <a-col :span="6">
        <a-card :bordered="false" title="CPU 使用率" size="small">
          <GaugeChart :value="snapshot?.cpu?.usage_percent || 0" max="100" />
          <div class="info-text">型号：{{ snapshot?.cpu?.model || '-' }}</div>
          <div class="info-text">核心：{{ snapshot?.cpu?.cores || '-' }}</div>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card :bordered="false" title="内存使用率" size="small">
          <GaugeChart :value="snapshot?.memory?.usage_percent || 0" max="100" />
          <div class="info-text">总量：{{ formatBytes(snapshot?.memory?.total) }}</div>
          <div class="info-text">已用：{{ formatBytes(snapshot?.memory?.used) }}</div>
        </a-card>
      </a-col>
      <a-col :span="6" v-for="(disk, idx) in topDisks" :key="disk.mount">
        <a-card :bordered="false" :title="`磁盘: ${disk.mount}`" size="small">
          <RingProgress :value="disk.usage_percent" :subtitle="`${formatBytes(disk.used)} / ${formatBytes(disk.total)}`" />
          <div class="info-text small">{{ disk.fs || '-' }}</div>
        </a-card>
      </a-col>
    </a-row>

    <a-row :gutter="16" style="margin-top: 16px">
      <a-col :span="12">
        <a-card :bordered="false" title="系统信息" size="small">
          <a-descriptions :column="2" size="small">
            <a-descriptions-item label="主机名">{{ stringVal(snapshot?.system?.hostname) }}</a-descriptions-item>
            <a-descriptions-item label="平台">{{ stringVal(snapshot?.system?.platform) }}</a-descriptions-item>
            <a-descriptions-item label="内核">{{ stringVal(snapshot?.system?.kernel) }}</a-descriptions-item>
            <a-descriptions-item label="OS 类型">{{ stringVal(probe?.os_type) }}</a-descriptions-item>
          </a-descriptions>
        </a-card>
      </a-col>
      <a-col :span="12">
        <a-card :bordered="false" title="磁盘详情" size="small">
          <a-table
            size="small"
            :pagination="false"
            :data-source="diskRows"
            :columns="diskColumns"
            :row-key="(_, idx) => idx"
            :scroll="{ y: 200 }"
          />
        </a-card>
      </a-col>
    </a-row>

    <a-row :gutter="16" style="margin-top: 16px" v-if="portRows.length > 0">
      <a-col :span="24">
        <a-card :bordered="false" title="端口监听" size="small">
          <a-table
            size="small"
            :pagination="{ pageSize: 10, size: 'small' }"
            :data-source="portRows"
            :columns="portColumns"
            :scroll="{ x: 600 }"
          />
        </a-card>
      </a-col>
    </a-row>

    <a-row :gutter="16" style="margin-top: 16px">
      <a-col :span="24">
        <a-card :bordered="false" title="日志查看" size="small">
          <template #extra>
            <Space>
              <span class="subtle">会话状态：</span>
              <StatusTag :status="logRunning ? 'running' : 'stopped'" />
              <Badge :count="logLines.length" :numberStyle="{ backgroundColor: '#1890ff' }" />
            </Space>
          </template>

          <div class="log-controls">
            <Space wrap>
              <a-select v-model:value="logForm.source" style="width: 200px" size="small" :options="sourceOptions" />
              <a-input v-model:value="logForm.keyword" placeholder="关键字过滤（可空）" style="width: 180px" size="small" allow-clear />
              <a-input-number v-model:value="logForm.lines" :min="50" :max="2000" style="width: 110px" size="small" />
              <a-switch v-model:checked="logForm.follow" checked-children="跟随" un-checked-children="一次性" size="small" />
              <a-switch v-model:checked="autoScroll" checked-children="自动滚动" un-checked-children="手动滚动" size="small" />
              <a-button type="primary" size="small" @click="startLog" :loading="logLoading">
                <template #icon><PlayCircleOutlined v-if="!logRunning" /><PauseCircleOutlined v-else /></template>
                {{ logRunning ? '重新开始' : '开始' }}
              </a-button>
              <a-button size="small" @click="stopLog" :disabled="!logRunning">
                <template #icon><StopOutlined /></template>
                停止
              </a-button>
              <a-button size="small" @click="clearLog">
                <template #icon><ClearOutlined /></template>
                清空
              </a-button>
            </Space>
          </div>

          <div class="log-box" ref="logBoxRef">
            <template v-if="logLines.length">
              <div v-for="(line, idx) in logLines" :key="idx" class="log-line" :class="logLevelClass(line)">
                {{ line || ' ' }}
              </div>
            </template>
            <div v-else class="log-placeholder">
              <FileTextOutlined class="placeholder-icon" />
              <span>暂无日志输出，点击"开始"按钮获取日志</span>
            </div>
          </div>
        </a-card>
      </a-col>
    </a-row>
  </div>
</template>

<script setup lang="ts">
import { computed, h, nextTick, onBeforeUnmount, onMounted, reactive, ref } from "vue";
import { Badge, Space, Tag, message } from "ant-design-vue";
import { useRoute } from "vue-router";
import { useAdminAuthStore } from "@/stores/adminAuth";
import { createAdminProbeLogSession, getAdminProbeDetail, getAdminProbeSla } from "@/services/admin";
import { createSseConnection } from "@/services/sse";
import {
  ClockCircleOutlined,
  ReloadOutlined,
  DesktopOutlined,
  HeartOutlined,
  CloudOutlined,
  PlayCircleOutlined,
  PauseCircleOutlined,
  StopOutlined,
  ClearOutlined,
  FileTextOutlined
} from "@ant-design/icons-vue";
import GaugeChart from "@/components/Charts/GaugeChart.vue";
import RingProgress from "@/components/Charts/RingProgress.vue";
import StatusTag from "@/components/StatusTag.vue";

const apiBase = import.meta.env.VITE_API_BASE || "";
const route = useRoute();
const adminAuth = useAdminAuthStore();
const id = computed(() => String(route.params.id || ""));

const loading = ref(false);
const logLoading = ref(false);
const logRunning = ref(false);
const probe = ref<any>(null);
const sla = ref<any>(null);
const refreshError = ref("");
const lastRefreshAt = ref("");
const logLines = ref<string[]>([]);
const autoScroll = ref(true);
const logBoxRef = ref<HTMLElement | null>(null);
const MAX_LOG_LINES = 4000;
let refreshTimer: number | null = null;
let refreshInFlight = false;

const logForm = reactive({
  source: "file:logs",
  keyword: "",
  lines: 300,
  follow: true
});

const sourceOptions = [
  { label: "文件日志（logs）", value: "file:logs" },
  { label: "Linux Journal(system)", value: "journal:system" },
  { label: "Linux Journal(pveproxy)", value: "journal:pveproxy" },
  { label: "Windows 系统关键日志", value: "eventlog:System:important" },
  { label: "Windows 系统全部日志", value: "eventlog:System:full" },
  { label: "Windows 开关机/崩溃", value: "eventlog:System:power" },
  { label: "Windows 应用关键日志", value: "eventlog:Application:important" },
  { label: "Windows Hyper-V 关键日志", value: "eventlog:Hyper-V-Worker:important" }
];

const snapshot = computed(() => probe.value?.snapshot || {});
const formatDate = (v?: string) => (v ? new Date(v).toLocaleString("zh-CN") : "-");
const formatDateShort = (v?: string) => (v ? new Date(v).toLocaleString("zh-CN", { month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit" }) : "-");
const fromNow = (v?: string) => {
  if (!v) return "-";
  const diff = Date.now() - new Date(v).getTime();
  const sec = Math.floor(diff / 1000);
  const min = Math.floor(sec / 60);
  const hour = Math.floor(min / 60);
  const day = Math.floor(hour / 24);
  if (day > 0) return `${day}天前`;
  if (hour > 0) return `${hour}小时前`;
  if (min > 0) return `${min}分钟前`;
  return `${sec}秒前`;
};
const numberVal = (v: unknown) => {
  const n = Number(v);
  return Number.isFinite(n) ? n : 0;
};
const stringVal = (v: unknown) => (v == null || String(v).trim() === "" ? "-" : String(v));
const formatUptime = (v: unknown) => {
  const s = numberVal(v);
  if (!s) return "-";
  const d = Math.floor(s / 86400);
  const h = Math.floor((s % 86400) / 3600);
  const m = Math.floor((s % 3600) / 60);
  if (d > 0) return `${d}天 ${h}小时`;
  return `${h}小时 ${m}分`;
};
const formatBytes = (v: unknown) => {
  const n = numberVal(v);
  if (!n) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  let size = n;
  let i = 0;
  while (size >= 1024 && i < units.length - 1) {
    size /= 1024;
    i += 1;
  }
  return `${size.toFixed(i === 0 ? 0 : 2)} ${units[i]}`;
};

const diskRows = computed(() => (Array.isArray(snapshot.value?.disks) ? snapshot.value.disks : []));
const topDisks = computed(() => diskRows.value.slice(0, 2));
const portRows = computed(() => (Array.isArray(snapshot.value?.ports) ? snapshot.value.ports : []));

const diskColumns = [
  { title: "挂载点", dataIndex: "mount", key: "mount", width: 100 },
  { title: "文件系统", dataIndex: "fs", key: "fs", width: 90 },
  { title: "总量", key: "total", customRender: ({ record }: any) => formatBytes(record.total) },
  { title: "使用率", key: "usage_percent", customRender: ({ record }: any) => {
    const v = numberVal(record.usage_percent);
    const color = v < 60 ? "success" : v < 85 ? "warning" : "error";
    return h(Tag, { color, style: "margin:0" }, { default: () => `${v.toFixed(1)}%` });
  } }
];
const portColumns = [
  { title: "端口", dataIndex: "port", key: "port", width: 90 },
  { title: "协议", dataIndex: "proto", key: "proto", width: 80 },
  { title: "状态", key: "listen", width: 80, customRender: ({ record }: any) => {
    const color = record.listen ? "success" : "default";
    const text = record.listen ? "监听" : "未监听";
    return h(Tag, { color, style: "margin:0" }, { default: () => text });
  }},
  { title: "进程", dataIndex: "process_name", key: "process_name" }
];

let streamConn: { close: () => void } | null = null;

const refreshAll = async (forceSnapshot = false) => {
  if (refreshInFlight || !id.value) return;
  refreshInFlight = true;
  loading.value = true;
  try {
    const ts = Date.now();
    const [detailRes, slaRes] = await Promise.all([
      getAdminProbeDetail(id.value, { _t: ts, refresh: forceSnapshot ? 1 : 0 }),
      getAdminProbeSla(id.value, { days: 7, _t: ts })
    ]);
    probe.value = detailRes.data?.probe || null;
    sla.value = slaRes.data?.sla || null;
    lastRefreshAt.value = new Date().toISOString();
    refreshError.value = "";
  } catch (e: any) {
    refreshError.value = e?.message || "请求失败";
    message.error(`刷新失败: ${refreshError.value}`);
  } finally {
    loading.value = false;
    refreshInFlight = false;
  }
};

const stopLog = () => {
  if (streamConn) {
    streamConn.close();
    streamConn = null;
  }
  logRunning.value = false;
  logLoading.value = false;
};

const clearLog = () => {
  logLines.value = [];
};

const scrollLogToBottom = () => {
  if (!autoScroll.value || !logBoxRef.value) return;
  logBoxRef.value.scrollTop = logBoxRef.value.scrollHeight;
};

const appendLog = async (raw: string) => {
  const text = String(raw || "");
  if (!text) return;
  const incoming = text
    .replace(/\r\n/g, "\n")
    .split("\n")
    .map((line) => normalizeDotNetDate(line));
  logLines.value.push(...incoming);
  if (logLines.value.length > MAX_LOG_LINES) {
    logLines.value = logLines.value.slice(logLines.value.length - MAX_LOG_LINES);
  }
  await nextTick();
  scrollLogToBottom();
};

const normalizeDotNetDate = (line: string) =>
  line.replace(/\/Date\((\d+)(?:[+-]\d+)?\)\//g, (_, msRaw: string) => {
    const ms = Number(msRaw);
    if (!Number.isFinite(ms)) return _;
    return new Date(ms).toLocaleString("zh-CN");
  });

const logLevelClass = (line: string) => {
  const s = String(line || "").toLowerCase();
  if (s.includes("[error]") || s.includes("[critical]") || s.includes("[fail]")) return "is-error";
  if (s.includes("[warning]") || s.includes("[warn]")) return "is-warning";
  if (s.includes("[info]")) return "is-info";
  if (s.includes("[debug]")) return "is-debug";
  return "";
};

const startLog = async () => {
  stopLog();
  logLoading.value = true;
  clearLog();
  try {
    const res = await createAdminProbeLogSession(id.value, {
      source: logForm.source,
      keyword: logForm.keyword,
      lines: logForm.lines,
      follow: logForm.follow
    });
    const streamPath = String(res.data?.stream_path || "");
    if (!streamPath) throw new Error("no stream path");
    const url = `${apiBase}${streamPath}`;
    logRunning.value = true;
    streamConn = createSseConnection(url, {
      headers: { Authorization: `Bearer ${adminAuth.token}` },
      onMessage: async (msg) => {
        if (!msg.data) return;
        try {
          const parsed = JSON.parse(msg.data);
          if (parsed.type === "log_chunk" && parsed.data != null) {
            await appendLog(String(parsed.data));
          }
          if (parsed.type === "log_end") {
            logRunning.value = false;
          }
        } catch {
          await appendLog(msg.data);
        }
      },
      onError: () => {
        if (logRunning.value) {
          message.warning("日志连接中断，请重新连接");
        }
      }
    });
    logLoading.value = false;
  } catch (e: any) {
    logLoading.value = false;
    message.error(e?.message || "启动日志失败");
    return;
  }
};

onMounted(() => {
  refreshAll(false);
  refreshTimer = window.setInterval(() => {
    refreshAll(false);
  }, 5000);
});

onBeforeUnmount(() => {
  stopLog();
  if (refreshTimer != null) {
    window.clearInterval(refreshTimer);
    refreshTimer = null;
  }
});
</script>

<style scoped>
.probe-detail-page {
  padding: 16px;
  min-height: calc(100vh - 64px);
  background: #f0f2f5;
}

.page-header {
  margin-bottom: 16px;
}

.clickable {
  cursor: pointer;
}

.clickable:hover {
  color: #1677ff;
}

.header-card {
  margin-bottom: 16px;
}

.probe-title .name {
  font-size: 22px;
  font-weight: 600;
  color: #1f2937;
  margin-bottom: 8px;
}

.probe-title .meta .subtle {
  color: #6b7280;
  margin: 0 8px;
}

.last-refresh {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 6px;
  color: #6b7280;
  font-size: 13px;
}

.last-refresh .icon {
  font-size: 14px;
}

.metrics-row {
  margin-bottom: 16px;
}

.metric-card :deep(.ant-card-head-title) {
  .metric-title {
    font-size: 13px;
    color: #6b7280;
    display: flex;
    align-items: center;
    gap: 6px;
  }
}

.metric-card .metric-value {
  font-size: 28px;
  font-weight: 600;
  color: #1f2937;
  margin: 8px 0 4px;
}

.metric-card .metric-value.small {
  font-size: 20px;
}

.metric-card .metric-label {
  font-size: 13px;
  color: #6b7280;
}

.charts-row {
  margin-bottom: 16px;
}

.charts-row :deep(.ant-card) {
  height: 100%;
}

.info-text {
  margin-top: 8px;
  font-size: 12px;
  color: #6b7280;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.info-text.small {
  margin-top: 4px;
}

.log-controls {
  margin-bottom: 12px;
  padding: 12px;
  background: #f8fafc;
  border-radius: 6px;
}

.log-box {
  min-height: 300px;
  max-height: 450px;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-word;
  background: #0f172a;
  border-radius: 8px;
  padding: 12px;
  font-family: "JetBrains Mono", "Fira Code", "Consolas", "Courier New", monospace;
  font-size: 13px;
  line-height: 1.6;
}

.log-placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 300px;
  color: #475569;
}

.log-placeholder .placeholder-icon {
  font-size: 48px;
  margin-bottom: 12px;
  color: #334155;
}

.log-line {
  padding: 2px 0;
  border-bottom: 1px solid rgba(148, 163, 184, 0.1);
}

.log-line:last-child {
  border-bottom: none;
}

.log-line.is-error {
  color: #f87171;
  background: rgba(239, 68, 68, 0.1);
  padding: 4px 8px;
  border-radius: 4px;
  margin: 2px -4px;
}

.log-line.is-warning {
  color: #fbbf24;
}

.log-line.is-info {
  color: #60a5fa;
}

.log-line.is-debug {
  color: #9ca3af;
}

.subtle {
  color: #6b7280;
}
</style>
