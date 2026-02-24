<template>
  <div class="page admin-page dashboard-page">
    <a-card :bordered="false" class="hero-card" :loading="loading">
      <div class="hero-header">
        <div>
          <div class="hero-title">
            <DashboardOutlined />
            运营总览
          </div>
          <div class="hero-subtitle">企业看板: 收入、订单、资源、风险联动展示</div>
        </div>
        <a-space>
          <a-segmented
            v-model:value="period"
            :options="[
              { label: '近30天', value: 'day' },
              { label: '近6月', value: 'month' }
            ]"
            @change="fetchRevenueSeries"
          />
          <a-button :loading="loading" @click="reloadAll">刷新</a-button>
        </a-space>
      </div>
      <div class="hero-meta">
        <a-tag color="blue">收入口径: 已审批支付</a-tag>
        <a-tag color="green">订单样本: {{ orders.length }}</a-tag>
        <a-tag color="purple">实例样本: {{ vpsList.length }}</a-tag>
      </div>
    </a-card>

    <a-row :gutter="[14, 14]" class="section-gap">
      <a-col :xs="24" :sm="12" :lg="8" :xl="4">
        <a-card :bordered="false" class="kpi-card kpi-revenue" :loading="loading">
          <a-statistic title="累计收入" :value="toYuan(overview.revenue)" prefix="¥" :precision="2" />
          <div class="kpi-foot">
            <span class="trend" :class="revenueTrend.className">{{ revenueTrend.text }}</span>
          </div>
        </a-card>
      </a-col>
      <a-col :xs="24" :sm="12" :lg="8" :xl="4">
        <a-card :bordered="false" class="kpi-card" :loading="loading">
          <a-statistic title="总订单" :value="overview.total_orders" />
          <div class="kpi-foot">已处理率 {{ orderHandleRate.toFixed(2) }}%</div>
        </a-card>
      </a-col>
      <a-col :xs="24" :sm="12" :lg="8" :xl="4">
        <a-card :bordered="false" class="kpi-card" :loading="loading">
          <a-statistic title="待审核订单" :value="overview.pending_review" />
          <div class="kpi-foot danger">需运营优先处理</div>
        </a-card>
      </a-col>
      <a-col :xs="24" :sm="12" :lg="8" :xl="4">
        <a-card :bordered="false" class="kpi-card" :loading="loading">
          <a-statistic title="VPS 总数" :value="overview.vps_count" />
          <div class="kpi-foot">{{ statusSummary }}</div>
        </a-card>
      </a-col>
      <a-col :xs="24" :sm="12" :lg="8" :xl="4">
        <a-card :bordered="false" class="kpi-card" :loading="loading">
          <a-statistic title="7天内到期" :value="overview.expiring_soon" />
          <div class="kpi-foot warning">建议发送续费提醒</div>
        </a-card>
      </a-col>
      <a-col :xs="24" :sm="12" :lg="8" :xl="4">
        <a-card :bordered="false" class="kpi-card" :loading="loading">
          <a-statistic title="健康度" :value="healthScore" suffix="/100" />
          <div class="kpi-foot" :class="healthScore < 70 ? 'danger' : 'trend up'">{{ healthComment }}</div>
        </a-card>
      </a-col>
    </a-row>

    <a-row :gutter="[14, 14]" class="section-gap">
      <a-col :xs="24" :xl="16">
        <a-card :bordered="false" class="panel-card" :loading="loading">
          <template #title>
            <div class="panel-title"><AlertOutlined /> 运营告警与关注点</div>
          </template>
          <a-alert
            v-if="!alerts.length"
            type="success"
            show-icon
            message="当前无高风险告警"
            description="指标稳定，可继续观察趋势变化。"
          />
          <a-list v-else :data-source="alerts" size="small">
            <template #renderItem="{ item }">
              <a-list-item>
                <a-space>
                  <a-tag :color="item.level === 'high' ? 'red' : 'orange'">
                    {{ item.level === "high" ? "高风险" : "中风险" }}
                  </a-tag>
                  <span>{{ item.text }}</span>
                </a-space>
              </a-list-item>
            </template>
          </a-list>
        </a-card>
      </a-col>
      <a-col :xs="24" :xl="8">
        <a-card :bordered="false" class="panel-card" :loading="loading">
          <template #title>
            <div class="panel-title"><DashboardOutlined /> 运行健康仪表</div>
          </template>
          <div class="health-wrap">
            <a-progress type="dashboard" :percent="healthScore" :stroke-color="healthColor" />
            <a-space direction="vertical" style="width: 100%">
              <div class="health-item">
                <span>CPU</span>
                <a-progress :percent="toPercent(serverStatus.cpu_usage_percent)" size="small" />
              </div>
              <div class="health-item">
                <span>内存</span>
                <a-progress :percent="toPercent(serverStatus.mem_usage_percent)" size="small" />
              </div>
              <div class="health-item">
                <span>磁盘</span>
                <a-progress :percent="toPercent(serverStatus.disk_usage_percent)" size="small" :status="diskStatus" />
              </div>
            </a-space>
          </div>
        </a-card>
      </a-col>
    </a-row>

    <a-row :gutter="[14, 14]" class="section-gap">
      <a-col :xs="24" :xl="14">
        <a-card :bordered="false" class="panel-card" :loading="loading">
          <template #title>
            <div class="panel-title"><AreaChartOutlined /> 趋势分析</div>
          </template>
          <a-tabs v-model:activeKey="trendTab" size="small">
            <a-tab-pane key="revenue" tab="收入趋势">
              <LineChart
                :data="revenueChart"
                :y-axis-value-formatter="formatYuanAxis"
                :tooltip-value-formatter="formatYuanAxis"
              />
            </a-tab-pane>
            <a-tab-pane key="order" tab="订单状态分布">
              <BarChart :data="orderStatusChart" />
            </a-tab-pane>
            <a-tab-pane key="expire" tab="到期趋势">
              <LineChart :data="expiringChart" />
            </a-tab-pane>
          </a-tabs>
        </a-card>
      </a-col>
      <a-col :xs="24" :xl="10">
        <a-card :bordered="false" class="panel-card" :loading="loading">
          <template #title>
            <div class="panel-title"><PieChartOutlined /> 资源结构</div>
          </template>
          <PieChart :data="vpsStatusChart" />
          <div class="status-list">
            <div v-for="row in vpsStatusRows" :key="row.name" class="status-row">
              <div class="status-name">{{ row.name }}</div>
              <div class="status-values">
                <span>{{ row.value }}</span>
                <span>{{ row.ratio.toFixed(2) }}%</span>
              </div>
              <a-progress :percent="row.ratio" :show-info="false" size="small" />
            </div>
          </div>
        </a-card>
      </a-col>
    </a-row>

    <a-row :gutter="[14, 14]" class="section-gap">
      <a-col :xs="24" :xl="12">
        <a-card :bordered="false" class="panel-card" :loading="loading">
          <template #title>
            <div class="panel-title"><FileSearchOutlined /> 待审核订单</div>
          </template>
          <a-table
            :data-source="pendingOrders"
            :columns="pendingColumns"
            :pagination="false"
            row-key="id"
            size="small"
            :scroll="{ x: 780 }"
          >
            <template #bodyCell="{ column, text }">
              <template v-if="column.key === 'total_amount'">
                <span class="amount-up">¥{{ toYuan(Number(text || 0)) }}</span>
              </template>
              <template v-else-if="column.key === 'created_at'">
                <span class="subtle">{{ formatDateTime(String(text || "")) }}</span>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-col>
      <a-col :xs="24" :xl="12">
        <a-card :bordered="false" class="panel-card" :loading="loading">
          <template #title>
            <div class="panel-title"><ClockCircleOutlined /> 即将到期实例</div>
          </template>
          <a-table
            :data-source="expiringVps"
            :columns="expiringColumns"
            :pagination="false"
            row-key="id"
            size="small"
            :scroll="{ x: 780 }"
          >
            <template #bodyCell="{ column, text }">
              <template v-if="column.key === 'days_left'">
                <a-tag :color="Number(text || 0) <= 3 ? 'red' : Number(text || 0) <= 7 ? 'orange' : 'blue'">
                  {{ Number(text || 0) <= 0 ? "已到期" : `${text} 天` }}
                </a-tag>
              </template>
              <template v-else-if="column.key === 'status'">
                <a-tag :color="vpsStatusColor(String(text || ''))">{{ vpsStatusLabel(String(text || '')) }}</a-tag>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-col>
    </a-row>

    <a-row :gutter="[14, 14]" class="section-gap">
      <a-col :xs="24">
        <a-card :bordered="false" class="panel-card" :loading="loading">
          <template #title>
            <div class="panel-title"><CloudServerOutlined /> 服务器状态</div>
          </template>
          <a-descriptions :column="3" bordered size="small">
            <a-descriptions-item label="主机名">{{ serverStatus.hostname || "-" }}</a-descriptions-item>
            <a-descriptions-item label="系统">{{ serverStatus.os || "-" }}</a-descriptions-item>
            <a-descriptions-item label="平台">{{ serverStatus.platform || "-" }}</a-descriptions-item>
            <a-descriptions-item label="内核">{{ serverStatus.kernel_version || "-" }}</a-descriptions-item>
            <a-descriptions-item label="CPU">{{ cpuText }}</a-descriptions-item>
            <a-descriptions-item label="运行时间">{{ uptimeText }}</a-descriptions-item>
          </a-descriptions>
        </a-card>
      </a-col>
    </a-row>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { message, type TableColumnType } from "ant-design-vue";
import {
  AlertOutlined,
  AreaChartOutlined,
  ClockCircleOutlined,
  CloudServerOutlined,
  DashboardOutlined,
  FileSearchOutlined,
  PieChartOutlined
} from "@ant-design/icons-vue";
import {
  getAdminDashboardOverview,
  getAdminDashboardRevenue,
  getAdminDashboardVpsStatus,
  getServerStatus,
  listAdminOrders,
  listAdminVps
} from "@/services/admin";
import type { Order, VPSInstance } from "@/services/types";
import LineChart from "@/components/Charts/LineChart.vue";
import BarChart from "@/components/Charts/BarChart.vue";
import PieChart from "@/components/Charts/PieChart.vue";

type DashboardOverviewModel = {
  total_orders: number;
  pending_review: number;
  revenue: number;
  vps_count: number;
  expiring_soon: number;
};
type RevenuePoint = { date?: string; Date?: string; amount?: number; Amount?: number };
type StatusPoint = { status?: string; Status?: string; count?: number; Count?: number };

const loading = ref(false);
const period = ref<"day" | "month">("day");
const trendTab = ref<"revenue" | "order" | "expire">("revenue");
const orders = ref<Order[]>([]);
const vpsList = ref<VPSInstance[]>([]);
const revenueSeries = ref<RevenuePoint[]>([]);
const vpsStatusSeries = ref<StatusPoint[]>([]);

const overview = reactive<DashboardOverviewModel>({
  total_orders: 0,
  pending_review: 0,
  revenue: 0,
  vps_count: 0,
  expiring_soon: 0
});

const serverStatus = reactive({
  hostname: "",
  os: "",
  platform: "",
  kernel_version: "",
  uptime_seconds: 0,
  cpu_model: "",
  cpu_cores: 0,
  cpu_usage_percent: 0,
  mem_usage_percent: 0,
  disk_usage_percent: 0
});

const readNum = (obj: any, ...keys: string[]) => {
  for (const key of keys) {
    const val = obj?.[key];
    if (val !== undefined && val !== null) return Number(val) || 0;
  }
  return 0;
};
const readStr = (obj: any, ...keys: string[]) => {
  for (const key of keys) {
    const val = obj?.[key];
    if (val !== undefined && val !== null && String(val).trim() !== "") return String(val);
  }
  return "";
};

const toYuan = (cents: number) => (Number(cents || 0) / 100).toFixed(2);
const formatYuanAxis = (value?: number) => `¥${Number(value || 0).toFixed(2)}`;
const toPercent = (value?: number) => {
  const n = Number(value || 0);
  if (!Number.isFinite(n)) return 0;
  const normalized = n > 0 && n <= 1 ? n * 100 : n;
  return Math.max(0, Math.min(100, Math.round(normalized * 100) / 100));
};
const formatDateTime = (value?: string) => {
  if (!value) return "-";
  const d = new Date(value);
  return Number.isNaN(d.getTime()) ? value : d.toLocaleString("zh-CN", { hour12: false });
};
const formatDate = (value?: string) => {
  if (!value) return "-";
  const d = new Date(value);
  return Number.isNaN(d.getTime()) ? value : d.toLocaleDateString("zh-CN");
};

const cpuText = computed(() => {
  const cores = Number(serverStatus.cpu_cores || 0);
  const model = String(serverStatus.cpu_model || "");
  if (!model && !cores) return "-";
  return cores > 0 ? `${model || "CPU"} (${cores} Core)` : model;
});
const uptimeText = computed(() => {
  const sec = Number(serverStatus.uptime_seconds || 0);
  if (!sec) return "-";
  const days = Math.floor(sec / 86400);
  const hours = Math.floor((sec % 86400) / 3600);
  const minutes = Math.floor((sec % 3600) / 60);
  if (days > 0) return `${days}天 ${hours}小时`;
  if (hours > 0) return `${hours}小时 ${minutes}分钟`;
  return `${minutes}分钟`;
});

const diskStatus = computed(() => (toPercent(serverStatus.disk_usage_percent) >= 85 ? "exception" : "normal"));

const revenueChart = computed(() => ({
  labels: revenueSeries.value.map((p) => readStr(p, "date", "Date")),
  values: revenueSeries.value.map((p) => readNum(p, "amount", "Amount") / 100)
}));

const revenueTrend = computed(() => {
  const values = revenueChart.value.values;
  if (values.length < 2) return { className: "", text: "暂无对比数据" };
  const last = Number(values[values.length - 1] || 0);
  const prev = Number(values[values.length - 2] || 0);
  if (prev <= 0) return { className: "", text: "暂无对比数据" };
  const ratio = ((last - prev) / prev) * 100;
  return ratio >= 0
    ? { className: "up", text: `较上周期 +${ratio.toFixed(2)}%` }
    : { className: "down", text: `较上周期 ${ratio.toFixed(2)}%` };
});

const orderStatusChart = computed(() => {
  const counts = new Map<string, number>();
  for (const order of orders.value) {
    const status = orderStatusLabel(readStr(order, "status", "Status"));
    counts.set(status, (counts.get(status) || 0) + 1);
  }
  const labels = Array.from(counts.keys());
  return { labels, values: labels.map((k) => counts.get(k) || 0) };
});

const vpsStatusChart = computed(() => {
  if (vpsStatusSeries.value.length) {
    return vpsStatusSeries.value.map((it) => ({
      name: readStr(it, "status", "Status") || "unknown",
      value: readNum(it, "count", "Count")
    }));
  }
  const counts = new Map<string, number>();
  for (const vps of vpsList.value) {
    const status = readStr(vps, "status", "Status") || "unknown";
    counts.set(status, (counts.get(status) || 0) + 1);
  }
  return Array.from(counts.entries()).map(([name, value]) => ({ name, value }));
});

const vpsStatusRows = computed(() => {
  const rows = vpsStatusChart.value;
  const total = rows.reduce((sum, r) => sum + Number(r.value || 0), 0) || 1;
  return rows
    .map((r) => ({
      name: vpsStatusLabel(r.name),
      value: Number(r.value || 0),
      ratio: Number((((Number(r.value || 0) / total) * 100).toFixed(2)))
    }))
    .sort((a, b) => b.value - a.value);
});

const expiringChart = computed(() => {
  const today = new Date();
  const bucket = new Map<string, number>();
  for (let i = 0; i < 30; i++) {
    const dt = new Date(today);
    dt.setDate(today.getDate() + i);
    bucket.set(dt.toISOString().slice(0, 10), 0);
  }
  for (const vps of vpsList.value) {
    const expire = readStr(vps, "expire_at", "ExpireAt");
    if (!expire) continue;
    const key = expire.slice(0, 10);
    if (bucket.has(key)) {
      bucket.set(key, (bucket.get(key) || 0) + 1);
    }
  }
  const labels = Array.from(bucket.keys());
  return { labels, values: labels.map((k) => bucket.get(k) || 0) };
});

const orderHandleRate = computed(() => {
  const total = Number(overview.total_orders || 0);
  if (!total) return 0;
  return ((total - Number(overview.pending_review || 0)) / total) * 100;
});

const healthScore = computed(() => {
  let score = 100;
  const pendingRatio = Number(overview.total_orders || 0) > 0 ? Number(overview.pending_review || 0) / Number(overview.total_orders || 1) : 0;
  const expiringRatio = Number(overview.vps_count || 0) > 0 ? Number(overview.expiring_soon || 0) / Number(overview.vps_count || 1) : 0;
  score -= Math.min(30, pendingRatio * 120);
  score -= Math.min(20, expiringRatio * 90);
  score -= Math.max(0, (toPercent(serverStatus.cpu_usage_percent) - 75) * 0.8);
  score -= Math.max(0, (toPercent(serverStatus.mem_usage_percent) - 80) * 0.8);
  score -= Math.max(0, (toPercent(serverStatus.disk_usage_percent) - 85) * 1.2);
  return Math.max(0, Math.round(score));
});
const healthComment = computed(() => {
  if (healthScore.value >= 85) return "运行稳定";
  if (healthScore.value >= 70) return "轻度压力";
  return "需要重点关注";
});
const healthColor = computed(() => {
  if (healthScore.value >= 85) return "#52c41a";
  if (healthScore.value >= 70) return "#faad14";
  return "#ff4d4f";
});

const alerts = computed(() => {
  const items: Array<{ level: "high" | "medium"; text: string }> = [];
  if (overview.pending_review >= 10) {
    items.push({ level: "high", text: `待审核订单 ${overview.pending_review} 单，处理存在积压风险。` });
  } else if (overview.pending_review > 0) {
    items.push({ level: "medium", text: `待审核订单 ${overview.pending_review} 单，建议尽快清理。` });
  }
  if (overview.expiring_soon >= 10) {
    items.push({ level: "high", text: `7天内到期实例 ${overview.expiring_soon} 台，续费提醒压力较高。` });
  } else if (overview.expiring_soon > 0) {
    items.push({ level: "medium", text: `7天内到期实例 ${overview.expiring_soon} 台，建议安排提醒。` });
  }
  if (toPercent(serverStatus.disk_usage_percent) >= 85) {
    items.push({ level: "high", text: `磁盘使用率 ${toPercent(serverStatus.disk_usage_percent)}%，容量接近阈值。` });
  }
  return items;
});

const statusSummary = computed(() => {
  const running = vpsStatusRows.value.find((r) => r.name.includes("运行"))?.value || 0;
  return `运行中 ${running} 台`;
});

const pendingOrders = computed(() =>
  orders.value
    .filter((o) => readStr(o, "status", "Status") === "pending_review")
    .slice(0, 8)
    .map((o) => ({
      id: readNum(o, "id", "ID"),
      order_no: readStr(o, "order_no", "OrderNo"),
      user_id: readNum(o, "user_id", "UserID"),
      total_amount: readNum(o, "total_amount", "TotalAmount"),
      created_at: readStr(o, "created_at", "CreatedAt")
    }))
);

const expiringVps = computed(() =>
  vpsList.value
    .map((v) => {
      const expireAt = readStr(v, "expire_at", "ExpireAt");
      const ts = expireAt ? new Date(expireAt).getTime() : 0;
      const daysLeft = ts ? Math.ceil((ts - Date.now()) / 86400000) : 9999;
      return {
        id: readNum(v, "id", "ID"),
        name: readStr(v, "name", "Name"),
        user_id: readNum(v, "user_id", "UserID"),
        status: readStr(v, "status", "Status"),
        expire_at: expireAt,
        days_left: daysLeft
      };
    })
    .filter((v) => v.expire_at && v.days_left <= 30)
    .sort((a, b) => a.days_left - b.days_left)
    .slice(0, 8)
);

const pendingColumns: TableColumnType[] = [
  { title: "订单号", dataIndex: "order_no", key: "order_no", width: 220, ellipsis: true },
  { title: "用户ID", dataIndex: "user_id", key: "user_id", width: 100 },
  { title: "金额(分)", dataIndex: "total_amount", key: "total_amount", width: 140 },
  { title: "创建时间", dataIndex: "created_at", key: "created_at", width: 190 }
];
const expiringColumns: TableColumnType[] = [
  { title: "实例", dataIndex: "name", key: "name", width: 180, ellipsis: true },
  { title: "用户ID", dataIndex: "user_id", key: "user_id", width: 100 },
  { title: "状态", dataIndex: "status", key: "status", width: 120 },
  { title: "到期日", dataIndex: "expire_at", key: "expire_at", width: 140 },
  { title: "剩余", dataIndex: "days_left", key: "days_left", width: 100 }
];

const vpsStatusLabel = (status: string) => {
  const s = String(status || "").toLowerCase();
  if (s === "running") return "运行中";
  if (s === "stopped") return "已停机";
  if (s === "locked") return "已锁定";
  if (s === "provisioning") return "开通中";
  if (s === "failed") return "失败";
  if (!s || s === "unknown") return "未知状态";
  return s;
};
const orderStatusLabel = (status: string) => {
  const s = String(status || "").toLowerCase();
  if (s === "pending_payment") return "待支付";
  if (s === "pending_review") return "待审核";
  if (s === "approved" || s === "confirmed" || s === "completed") return "已完成";
  if (s === "rejected") return "已拒绝";
  if (s === "failed") return "失败";
  if (s === "cancelled") return "已取消";
  if (!s || s === "unknown") return "未知状态";
  return s;
};
const vpsStatusColor = (status: string) => {
  const s = String(status || "").toLowerCase();
  if (s === "running") return "green";
  if (s === "provisioning") return "blue";
  if (s === "locked") return "orange";
  if (s === "failed") return "red";
  return "default";
};

const fetchOverview = async () => {
  const res = await getAdminDashboardOverview();
  const raw = res.data || {};
  overview.total_orders = readNum(raw, "total_orders", "TotalOrders");
  overview.pending_review = readNum(raw, "pending_review", "PendingReview");
  overview.revenue = readNum(raw, "revenue", "Revenue");
  overview.vps_count = readNum(raw, "vps_count", "VPSCount");
  overview.expiring_soon = readNum(raw, "expiring_soon", "ExpiringSoon");
};
const fetchRevenueSeries = async () => {
  const res = await getAdminDashboardRevenue({ period: period.value });
  revenueSeries.value = (res.data as any)?.items || (res.data as any)?.points || [];
};
const fetchVpsStatus = async () => {
  const res = await getAdminDashboardVpsStatus();
  vpsStatusSeries.value = (res.data as any)?.items || (res.data as any)?.points || [];
};
const fetchLists = async () => {
  const pageSize = 500;
  const maxRows = 10000;

  const fetchAllOrders = async () => {
    const rows: Order[] = [];
    let offset = 0;
    while (true) {
      const res = await listAdminOrders({ limit: pageSize, offset });
      const items = (res.data?.items || []) as Order[];
      rows.push(...items);
      const total = Number((res.data as any)?.total || 0);
      if (items.length < pageSize) break;
      offset += pageSize;
      if ((total > 0 && rows.length >= total) || rows.length >= maxRows) break;
    }
    return rows;
  };

  const fetchAllVps = async () => {
    const rows: VPSInstance[] = [];
    let offset = 0;
    while (true) {
      const res = await listAdminVps({ limit: pageSize, offset });
      const items = (res.data?.items || []) as VPSInstance[];
      rows.push(...items);
      const total = Number((res.data as any)?.total || 0);
      if (items.length < pageSize) break;
      offset += pageSize;
      if ((total > 0 && rows.length >= total) || rows.length >= maxRows) break;
    }
    return rows;
  };

  const [allOrders, allVps] = await Promise.all([fetchAllOrders(), fetchAllVps()]);
  orders.value = allOrders;
  vpsList.value = allVps;
};
const fetchServer = async () => {
  const res = await getServerStatus();
  Object.assign(serverStatus, res.data || {});
};

const reloadAll = async () => {
  loading.value = true;
  try {
    await Promise.all([fetchOverview(), fetchRevenueSeries(), fetchVpsStatus(), fetchLists(), fetchServer()]);
  } catch (err: any) {
    message.error(err?.response?.data?.error || err?.message || "加载失败");
  } finally {
    loading.value = false;
  }
};

onMounted(async () => {
  await reloadAll();
});
</script>

<style scoped>
.dashboard-page {
  background:
    radial-gradient(1200px 420px at 5% -20%, rgba(27, 102, 210, 0.09), transparent 55%),
    radial-gradient(900px 360px at 98% -10%, rgba(39, 174, 96, 0.08), transparent 55%),
    #f5f7fb;
}

.hero-card,
.kpi-card,
.panel-card {
  border-radius: 14px;
  border: 1px solid #eaf0f8;
}

.hero-card {
  background: linear-gradient(120deg, #ffffff 0%, #f8fbff 65%);
}

.hero-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}

.hero-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 22px;
  font-weight: 700;
  color: #0f172a;
}

.hero-subtitle {
  margin-top: 4px;
  color: #64748b;
}

.hero-meta {
  margin-top: 12px;
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.section-gap {
  margin-top: 14px;
}

.kpi-card {
  min-height: 126px;
}

.kpi-revenue {
  background: linear-gradient(135deg, #edf5ff 0%, #ffffff 60%);
}

.kpi-foot {
  margin-top: 6px;
  font-size: 12px;
  color: #64748b;
}

.trend.up {
  color: #389e0d;
}

.trend.down,
.danger {
  color: #cf1322;
}

.warning {
  color: #d48806;
}

.panel-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
}

.health-wrap {
  display: flex;
  flex-direction: column;
  gap: 12px;
  align-items: center;
}

.health-item {
  width: 100%;
}

.health-item > span {
  font-size: 12px;
  color: #64748b;
}

.status-list {
  margin-top: 10px;
  padding-top: 8px;
  border-top: 1px solid #eef2f7;
}

.status-row + .status-row {
  margin-top: 8px;
}

.status-name {
  font-size: 12px;
  color: #334155;
}

.status-values {
  margin: 2px 0 4px;
  display: flex;
  justify-content: space-between;
  color: #64748b;
  font-size: 12px;
}

.amount-up {
  color: #1677ff;
  font-weight: 600;
}

.subtle {
  color: #64748b;
}

@media (max-width: 992px) {
  .hero-header {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>
