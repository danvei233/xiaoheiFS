<template>
  <div class="page admin-page">
    <div class="page-header">
      <div>
        <div class="page-title">
          <DashboardOutlined class="title-icon" />
          运营总览
        </div>
        <div class="subtle">核心指标与资源运行概览</div>
      </div>
      <div class="page-header-actions">
        <a-radio-group v-model:value="granularity" @change="fetchRevenue">
          <a-radio-button value="day">按天</a-radio-button>
          <a-radio-button value="month">按月</a-radio-button>
        </a-radio-group>
      </div>
    </div>

    <!-- 统计卡片 -->
    <a-row :gutter="16">
      <a-col :xs="24" :sm="12" :lg="8">
        <a-card class="stat-card" :bordered="false">
          <div class="stat-wrapper">
            <div class="stat-content">
              <div class="stat-title">已审批收入</div>
              <div class="stat-value">¥{{ formatNumber(stats.total_revenue) }}</div>
              <div class="stat-trend positive">
                <RiseOutlined />
                <span>较上月 +12.5%</span>
              </div>
            </div>
            <div class="stat-icon primary">
              <DollarOutlined />
            </div>
          </div>
        </a-card>
      </a-col>
      <a-col :xs="24" :sm="12" :lg="8">
        <a-card class="stat-card" :bordered="false">
          <div class="stat-wrapper">
            <div class="stat-content">
              <div class="stat-title">今日已审批收入</div>
              <div class="stat-value">¥{{ formatNumber(stats.today_revenue) }}</div>
              <div class="stat-trend positive">
                <RiseOutlined />
                <span>较昨日 +8.3%</span>
              </div>
            </div>
            <div class="stat-icon success">
              <ArrowUpOutlined />
            </div>
          </div>
        </a-card>
      </a-col>
      <a-col :xs="24" :sm="12" :lg="8">
        <a-card class="stat-card" :bordered="false">
          <div class="stat-wrapper">
            <div class="stat-content">
              <div class="stat-title">待审核订单</div>
              <div class="stat-value">{{ stats.pending_orders }}</div>
              <div class="stat-trend warning">
                <ClockCircleOutlined />
                <span>需要处理</span>
              </div>
            </div>
            <div class="stat-icon warning">
              <FileTextOutlined />
            </div>
          </div>
        </a-card>
      </a-col>
    </a-row>

    <a-row :gutter="16" style="margin-top: 16px">
      <a-col :xs="24" :sm="12" :lg="8">
        <a-card class="stat-card" :bordered="false">
          <div class="stat-wrapper">
            <div class="stat-content">
              <div class="stat-title">开通中</div>
              <div class="stat-value">{{ stats.provisioning }}</div>
              <div class="stat-trend info">
                <LoadingOutlined />
                <span>正在部署</span>
              </div>
            </div>
            <div class="stat-icon info">
              <CloudServerOutlined />
            </div>
          </div>
        </a-card>
      </a-col>
      <a-col :xs="24" :sm="12" :lg="8">
        <a-card class="stat-card" :bordered="false">
          <div class="stat-wrapper">
            <div class="stat-content">
              <div class="stat-title">失败数</div>
              <div class="stat-value">{{ stats.failed }}</div>
              <div class="stat-trend negative">
                <WarningOutlined />
                <span>需要关注</span>
              </div>
            </div>
            <div class="stat-icon error">
              <CloseCircleOutlined />
            </div>
          </div>
        </a-card>
      </a-col>
      <a-col :xs="24" :sm="12" :lg="8">
        <a-card class="stat-card" :bordered="false">
          <div class="stat-wrapper">
            <div class="stat-content">
              <div class="stat-title">VPS 总数</div>
              <div class="stat-value">{{ stats.vps_total }}</div>
              <div class="stat-trend positive">
                <CheckCircleOutlined />
                <span>运行正常</span>
              </div>
            </div>
            <div class="stat-icon secondary">
              <AppstoreOutlined />
            </div>
          </div>
        </a-card>
      </a-col>
    </a-row>

    <!-- 图表区域 -->
    <a-row :gutter="16" style="margin-top: 16px">
      <a-col :xs="24" :lg="12">
        <a-card :bordered="false">
          <template #title>
            <div class="card-title">
              <LineChartOutlined class="title-icon" />
              收入趋势
            </div>
          </template>
          <LineChart :data="charts.revenue" />
        </a-card>
      </a-col>
      <a-col :xs="24" :lg="12">
        <a-card :bordered="false">
          <template #title>
            <div class="card-title">
              <BarChartOutlined class="title-icon" />
              订单状态分布
            </div>
          </template>
          <BarChart :data="charts.orderStatus" />
        </a-card>
      </a-col>
    </a-row>

    <a-row :gutter="16" style="margin-top: 16px">
      <a-col :xs="24" :lg="12">
        <a-card :bordered="false">
          <template #title>
            <div class="card-title">
              <PieChartOutlined class="title-icon" />
              VPS 状态
            </div>
          </template>
          <PieChart :data="charts.vpsStatus" />
        </a-card>
      </a-col>
      <a-col :xs="24" :lg="12">
        <a-card :bordered="false">
          <template #title>
            <div class="card-title">
              <LineChartOutlined class="title-icon" />
              到期趋势
            </div>
          </template>
          <LineChart :data="charts.expiring" />
        </a-card>
      </a-col>
    </a-row>

    <a-row :gutter="16" style="margin-top: 16px">
      <a-col :xs="24">
        <a-card :bordered="false">
          <template #title>
            <div class="card-title">
              <CloudServerOutlined class="title-icon" />
              服务器状态
            </div>
          </template>
          <a-descriptions :column="3" bordered size="small">
            <a-descriptions-item label="主机名">{{ serverStatus.hostname || "-" }}</a-descriptions-item>
            <a-descriptions-item label="系统">{{ serverStatus.os || "-" }}</a-descriptions-item>
            <a-descriptions-item label="平台">{{ serverStatus.platform || "-" }}</a-descriptions-item>
            <a-descriptions-item label="内核">{{ serverStatus.kernel_version || "-" }}</a-descriptions-item>
            <a-descriptions-item label="CPU">{{ cpuText }}</a-descriptions-item>
            <a-descriptions-item label="运行时间">{{ uptimeText }}</a-descriptions-item>
          </a-descriptions>
          <a-row :gutter="16" style="margin-top: 12px">
            <a-col :xs="24" :md="8">
            <a-progress :percent="cpuPercent" size="small" />
            <div class="metric-label">CPU 使用率</div>
          </a-col>
          <a-col :xs="24" :md="8">
            <a-progress :percent="memPercent" size="small" />
            <div class="metric-label">内存使用率</div>
          </a-col>
          <a-col :xs="24" :md="8">
            <a-progress :percent="diskPercent" size="small" />
            <div class="metric-label">磁盘使用率</div>
          </a-col>
          </a-row>
        </a-card>
      </a-col>
    </a-row>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from "vue";
import {
  getAdminDashboardOverview,
  getAdminDashboardRevenue,
  getAdminDashboardVpsStatus,
  getServerStatus,
  listAdminOrders,
  listAdminVps
} from "@/services/admin";
import LineChart from "@/components/Charts/LineChart.vue";
import BarChart from "@/components/Charts/BarChart.vue";
import PieChart from "@/components/Charts/PieChart.vue";

import {
  DashboardOutlined,
  RiseOutlined,
  DollarOutlined,
  ArrowUpOutlined,
  ClockCircleOutlined,
  FileTextOutlined,
  LoadingOutlined,
  CloudServerOutlined,
  WarningOutlined,
  CloseCircleOutlined,
  CheckCircleOutlined,
  AppstoreOutlined,
  LineChartOutlined,
  BarChartOutlined,
  PieChartOutlined
} from '@ant-design/icons-vue';

const stats = reactive({
  total_revenue: 0,
  today_revenue: 0,
  pending_orders: 0,
  provisioning: 0,
  failed: 0,
  vps_total: 0
});

const charts = reactive({
  revenue: { labels: [], values: [] },
  orderStatus: { labels: [], values: [] },
  vpsStatus: [],
  expiring: { labels: [], values: [] }
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
  mem_total: 0,
  mem_usage_percent: 0,
  disk_total: 0,
  disk_usage_percent: 0
});

const granularity = ref("day");

const cpuText = computed(() => {
  if (!serverStatus.cpu_model) return serverStatus.cpu_cores ? `${serverStatus.cpu_cores} Core` : "-";
  const suffix = serverStatus.cpu_cores ? ` (${serverStatus.cpu_cores} Core)` : "";
  return `${serverStatus.cpu_model}${suffix}`;
});

const formatPercent = (value) => {
  const num = Number(value || 0);
  if (Number.isNaN(num)) return 0;
  return Math.round(num * 100) / 100;
};

const cpuPercent = computed(() => formatPercent(serverStatus.cpu_usage_percent));
const memPercent = computed(() => formatPercent(serverStatus.mem_usage_percent));
const diskPercent = computed(() => formatPercent(serverStatus.disk_usage_percent));

const uptimeText = computed(() => {
  const seconds = Number(serverStatus.uptime_seconds || 0);
  if (!seconds) return "-";
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  if (days > 0) return `${days}天 ${hours}小时`;
  if (hours > 0) return `${hours}小时 ${minutes}分钟`;
  return `${minutes}分钟`;
});

const formatNumber = (num) => {
  if (!num) return "0";
  return Number(num).toLocaleString('zh-CN', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2
  });
};

const toDateKey = (d) => {
  const dt = new Date(d);
  if (Number.isNaN(dt.getTime())) return "";
  return dt.toISOString().slice(0, 10);
};

const fetchOverview = async () => {
  const res = await getAdminDashboardOverview();
  const data = res.data || {};
  stats.total_revenue = data.total_revenue || 0;
  stats.today_revenue = data.today_revenue || 0;
  stats.pending_orders = data.pending_orders || 0;
  stats.provisioning = data.provisioning || 0;
  stats.failed = data.failed || 0;
  stats.vps_total = data.vps_total || 0;
};

const fetchRevenue = async () => {
  const res = await getAdminDashboardRevenue({ granularity: granularity.value });
  const points = res.data?.points || [];
  if (points.length) {
    charts.revenue = {
      labels: points.map((p) => p.date || p.Date),
      values: points.map((p) => p.amount || p.Amount || 0)
    };
  }
};

const fetchStatusCharts = async () => {
  const [ordersRes, vpsRes, vpsStatusRes] = await Promise.all([
    listAdminOrders({ limit: 200, offset: 0 }),
    listAdminVps({ limit: 200, offset: 0 }),
    getAdminDashboardVpsStatus()
  ]);

  const orders = ordersRes.data?.items || [];
  const vpsList = vpsRes.data?.items || [];
  const statusMap = new Map();
  orders.forEach((o) => {
    const key = o.status ?? o.Status ?? "unknown";
    statusMap.set(key, (statusMap.get(key) || 0) + 1);
  });
  charts.orderStatus = {
    labels: Array.from(statusMap.keys()),
    values: Array.from(statusMap.values())
  };

  const vpsStatusPoints = vpsStatusRes.data?.points || [];
  if (vpsStatusPoints.length) {
    charts.vpsStatus = vpsStatusPoints.map((p) => ({ name: p.status || p.Status, value: p.count || p.Count }));
  } else {
    const vpsMap = new Map();
    vpsList.forEach((v) => {
      const key = v.status ?? v.Status ?? "unknown";
      vpsMap.set(key, (vpsMap.get(key) || 0) + 1);
    });
    charts.vpsStatus = Array.from(vpsMap.entries()).map(([name, value]) => ({ name, value }));
  }

  const expiringMap = new Map();
  vpsList.forEach((v) => {
    const key = toDateKey(v.expire_at ?? v.ExpireAt);
    if (!key) return;
    expiringMap.set(key, (expiringMap.get(key) || 0) + 1);
  });
  const expLabels = Array.from(expiringMap.keys()).sort();
  charts.expiring = { labels: expLabels, values: expLabels.map((k) => expiringMap.get(k)) };
};

const fetchServerStatus = async () => {
  const res = await getServerStatus();
  Object.assign(serverStatus, res.data || {});
};

onMounted(async () => {
  await fetchOverview();
  await fetchRevenue();
  await fetchStatusCharts();
  await fetchServerStatus();
});
</script>

<style scoped>
.admin-page {
  position: relative;
}

.page-title {
  display: flex;
  align-items: center;
  gap: 8px;
}

.title-icon {
  font-size: 20px;
  color: #1677ff;
}

/* ========== 统计卡片 ========== */
.stat-card {
  height: 100%;
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
}

.stat-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.08) !important;
}

.stat-wrapper {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
}

.stat-content {
  flex: 1;
}

.stat-title {
  font-size: 13px;
  font-weight: 500;
  color: #6b7280;
  margin-bottom: 8px;
}

.stat-value {
  font-size: 28px;
  font-weight: 600;
  color: #1f2937;
  line-height: 1.2;
  margin-bottom: 8px;
  font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
}

.stat-trend {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  font-weight: 500;
  padding: 2px 8px;
  border-radius: 4px;
}

.stat-trend .anticon {
  font-size: 14px;
}

.stat-trend.positive {
  color: #52c41a;
  background: #f6ffed;
}

.stat-trend.negative {
  color: #ff4d4f;
  background: #fff1f0;
}

.stat-trend.warning {
  color: #faad14;
  background: #fffbe6;
}

.stat-trend.info {
  color: #1677ff;
  background: #e6f4ff;
}

.stat-icon {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  flex-shrink: 0;
}

.stat-icon .anticon {
  font-size: 24px;
}

.stat-icon.primary {
  background: linear-gradient(135deg, #1677ff 0%, #0958d9 100%);
  color: #fff;
}

.stat-icon.success {
  background: linear-gradient(135deg, #52c41a 0%, #389e0d 100%);
  color: #fff;
}

.stat-icon.warning {
  background: linear-gradient(135deg, #faad14 0%, #d48806 100%);
  color: #fff;
}

.stat-icon.error {
  background: linear-gradient(135deg, #ff4d4f 0%, #cf1322 100%);
  color: #fff;
}

.stat-icon.info {
  background: linear-gradient(135deg, #1677ff 0%, #0958d9 100%);
  color: #fff;
}

.stat-icon.secondary {
  background: linear-gradient(135deg, #8b5cf6 0%, #7c3aed 100%);
  color: #fff;
}

/* ========== 图表卡片 ========== */
.card-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  color: #1f2937;
}

.card-title .title-icon {
  font-size: 16px;
  color: #1677ff;
}

.metric-label {
  font-size: 12px;
  color: #6b7280;
  margin-top: 4px;
}

:deep(.ant-card-head-title) {
  font-weight: 600;
  color: #1f2937;
}

/* ========== 响应式 ========== */
@media (max-width: 768px) {
  .stat-value {
    font-size: 24px;
  }

  .stat-icon {
    width: 40px;
    height: 40px;
  }

  .stat-icon .anticon {
    font-size: 20px;
  }
}
</style>
