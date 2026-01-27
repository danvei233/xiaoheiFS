<template>
  <div class="dashboard-page">
    <!-- Page Header -->
    <div class="page-header animate-fade-in-up">
      <div class="page-title-content">
        <h1 class="page-title">控制台</h1>
        <p class="page-description">查看您的云服务概览</p>
      </div>
      <a-space>
        <a-button @click="refreshDashboard" :loading="loading" class="refresh-btn">
          <template #icon>
            <ReloadOutlined />
          </template>
          刷新
        </a-button>
      </a-space>
    </div>

    <!-- Statistics Cards -->
    <a-row :gutter="[16, 16]" class="dashboard-stats-row stagger-children">
      <a-col :xs="24" :sm="12" :md="6" :lg="6" :xl="6" :xxl="6" v-for="stat in displayStats" :key="stat.key">
        <div class="stat-card metric-card" :style="{ '--accent-color': stat.color }">
          <div class="stat-icon-wrapper" :style="{ background: `${stat.color}15`, color: stat.color }">
            <component :is="getStatIcon(stat.key)" class="stat-icon" />
          </div>
          <div class="stat-content">
            <div class="stat-title">{{ stat.title }}</div>
            <div class="stat-value" :style="{ color: stat.color }">
              <template v-if="stat.formatter">{{ stat.formatter }}</template>
              <template v-else>{{ stat.value }}</template>
              <span class="stat-unit" v-if="stat.tag">{{ stat.tag }}</span>
            </div>
            <div class="stat-meta">
              <a-tag :color="stat.tagColor" v-if="stat.tag" class="stat-tag">{{ stat.tag }}</a-tag>
              <span class="stat-desc">{{ stat.description }}</span>
            </div>
          </div>
          <div class="stat-glow"></div>
        </div>
      </a-col>
    </a-row>

    <!-- Charts Section -->
    <a-row :gutter="[16, 16]" class="charts-section">
      <a-col :xs="24" :lg="12">
        <div class="chart-card elevated-card">
          <div class="chart-card-header">
            <h3 class="chart-card-title">近30天消费走势</h3>
            <a-tag color="blue" class="chart-card-tag">消费趋势</a-tag>
          </div>
          <div class="chart-wrapper">
            <LineChart :data="charts.spendTrend" />
          </div>
        </div>
      </a-col>
      <a-col :xs="24" :lg="12">
        <div class="chart-card elevated-card">
          <div class="chart-card-header">
            <h3 class="chart-card-title">订单状态分布</h3>
            <a-tag color="cyan" class="chart-card-tag">状态统计</a-tag>
          </div>
          <div class="chart-wrapper chart-wrapper-pie">
            <PieChart :data="charts.orderStatus" />
          </div>
        </div>
      </a-col>
    </a-row>

    <!-- Expiring Instances -->
    <div class="expiring-card elevated-card">
      <div class="expiring-header">
        <h3 class="expiring-title">即将到期的实例</h3>
        <div class="expiring-badge-wrapper">
          <a-badge :count="expiring.length" :number-style="{ backgroundColor: '#ef4444', boxShadow: '0 2px 8px rgba(239, 68, 68, 0.3)' }" />
        </div>
      </div>
      <a-empty
        v-if="expiring.length === 0"
        description="暂无即将到期的实例"
        :image="Empty.PRESENTED_IMAGE_SIMPLE"
        class="expiring-empty"
      />
      <a-list
        v-else
        :data-source="expiring"
        :split="false"
        class="expiring-list"
      >
        <template #renderItem="{ item }">
          <a-list-item class="expiring-item">
            <a-list-item-meta>
              <template #avatar>
                <a-avatar :style="{ backgroundColor: getExpireAvatarColor(item.expire_at || item.ExpireAt) }">
                  <CloudServerOutlined />
                </a-avatar>
              </template>
              <template #title>
                <a-space>
                  <span class="instance-name">{{ item.name || item.Name }}</span>
                  <a-tag :color="getExpireTagColor(item.expire_at || item.ExpireAt)" size="small" class="expire-tag">
                    {{ getExpireLabel(item.expire_at || item.ExpireAt) }}
                  </a-tag>
                </a-space>
              </template>
              <template #description>
                <span class="instance-id">ID: {{ item.id || item.ID }}</span>
              </template>
            </a-list-item-meta>
            <template #actions>
              <span class="expire-time">
                <ClockCircleOutlined />
                {{ formatExpireTime(item.expire_at || item.ExpireAt) }}
              </span>
            </template>
          </a-list-item>
        </template>
      </a-list>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, h } from "vue";
import {
  ReloadOutlined,
  WalletOutlined,
  SafetyOutlined,
  CloudServerOutlined,
  ShoppingOutlined,
  AlertOutlined,
  ClockCircleOutlined,
  FileTextOutlined,
  CreditCardOutlined
} from "@ant-design/icons-vue";
import { Empty } from "ant-design-vue";
import { useDashboardStore } from "@/stores/dashboard";
import LineChart from "@/components/Charts/LineChart.vue";
import PieChart from "@/components/Charts/PieChart.vue";

const store = useDashboardStore();
const loading = ref(false);
Empty.PRESENTED_IMAGE_SIMPLE = Empty.PRESENTED_IMAGE_SIMPLE;

const formatMoney = (amount, currency = "CNY") => {
  const value = Number(amount ?? 0);
  if (Number.isNaN(value)) return "-";
  const prefix = currency === "CNY" ? "¥" : `${currency} `;
  return `${prefix}${value.toFixed(2)}`;
};

const realnameLabel = (status) => {
  if (status === "verified") return "已认证";
  if (status === "failed") return "未通过";
  if (status === "pending") return "审核中";
  if (status === "disabled") return "未启用";
  return "未认证";
};

const realnameStatusTag = (status) => {
  if (status === "verified") return { color: "success", text: "已认证" };
  if (status === "failed") return { color: "error", text: "未通过" };
  if (status === "pending") return { color: "warning", text: "审核中" };
  if (status === "disabled") return { color: "default", text: "未启用" };
  return { color: "default", text: "未认证" };
};

const stats = computed(() => {
  const balance = store.metrics.balance ?? 0;
  const currency = store.metrics.currency || "CNY";
  const realnameStatus = store.metrics.realname_status;
  const realnameTag = realnameStatusTag(realnameStatus);

  return [
    {
      key: "balance",
      title: "钱包余额",
      value: null,
      formatter: formatMoney(balance, currency),
      description: "可用账户余额",
      tag: currency,
      tagColor: "blue",
      color: "#0066FF"
    },
    {
      key: "realname",
      title: "实名认证",
      value: null,
      formatter: realnameLabel(realnameStatus),
      description: "账户认证状态",
      tag: realnameTag.text,
      tagColor: realnameTag.color,
      color: realnameTag.color === "success" ? "#059669" : realnameTag.color === "error" ? "#dc2626" : realnameTag.color === "warning" ? "#d97706" : "#94a3b8"
    },
    {
      key: "vps",
      title: "VPS 实例",
      value: store.metrics.vps_total || 0,
      description: "运行中的云服务器",
      tag: "台",
      tagColor: "cyan",
      color: "#0284c7"
    },
    {
      key: "orders",
      title: "全部订单",
      value: store.metrics.orders_total || 0,
      description: "订单总数量",
      tag: "单",
      tagColor: "geekblue",
      color: "#1e40af"
    },
    {
      key: "expiring",
      title: "即将到期",
      value: store.metrics.expiring || 0,
      description: "7天内到期",
      tag: "台",
      tagColor: store.metrics.expiring > 0 ? "orange" : "default",
      color: store.metrics.expiring > 0 ? "#ea580c" : "#94a3b8"
    },
    {
      key: "pending",
      title: "待处理",
      value: (store.metrics.pending_orders || 0) + (store.metrics.pending_payment || 0),
      description: "待审核+待支付",
      tag: "单",
      tagColor: "magenta",
      color: "#be185d"
    },
    {
      key: "spend",
      title: "近30天消费",
      value: null,
      formatter: formatMoney(store.metrics.spend_30d, currency),
      description: "月度支出统计",
      tag: "CNY",
      tagColor: "green",
      color: "#059669"
    }
  ];
});

// 显示所有7个统计卡片
const displayStats = computed(() => stats.value);

const getStatIcon = (key) => {
  const icons = {
    balance: WalletOutlined,
    realname: SafetyOutlined,
    vps: CloudServerOutlined,
    orders: FileTextOutlined,
    expiring: AlertOutlined,
    pending: ClockCircleOutlined,
    spend: CreditCardOutlined
  };
  return icons[key] || FileTextOutlined;
};

const charts = computed(() => ({
  spendTrend: store.charts.spendTrend || { labels: [], values: [] },
  orderStatus: store.charts.orderStatus || []
}));

const expiring = computed(() => store.charts.expiringList || []);

const formatExpireTime = (value) => {
  if (!value) return "-";
  const dt = new Date(value);
  if (Number.isNaN(dt.getTime())) return value;
  return dt.toLocaleString("zh-CN", {
    hour12: false,
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit"
  });
};

const getExpireLabel = (value) => {
  if (!value) return "-";
  const daysLeft = Math.ceil((new Date(value).getTime() - Date.now()) / (24 * 3600 * 1000));
  if (daysLeft <= 0) return "已到期";
  if (daysLeft === 1) return "今天到期";
  if (daysLeft <= 3) return `${daysLeft}天后到期`;
  return `${daysLeft}天后`;
};

const getExpireTagColor = (value) => {
  if (!value) return "default";
  const daysLeft = Math.ceil((new Date(value).getTime() - Date.now()) / (24 * 3600 * 1000));
  if (daysLeft <= 0) return "error";
  if (daysLeft === 1) return "error";
  if (daysLeft <= 3) return "warning";
  return "default";
};

const getExpireAvatarColor = (value) => {
  if (!value) return "#94a3b8";
  const daysLeft = Math.ceil((new Date(value).getTime() - Date.now()) / (24 * 3600 * 1000));
  if (daysLeft <= 0) return "#dc2626";
  if (daysLeft === 1) return "#dc2626";
  if (daysLeft <= 3) return "#d97706";
  return "#0066FF";
};

const refreshDashboard = async () => {
  loading.value = true;
  try {
    await store.fetchUserDashboard();
  } finally {
    loading.value = false;
  }
};

onMounted(() => {
  store.fetchUserDashboard();
});
</script>

<style scoped>
.dashboard-page {
  padding: 0;
  min-height: 100vh;
}

/* Page Header */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 32px;
  flex-wrap: wrap;
  gap: 20px;
  padding: 24px 24px 0;
}

.page-title-content {
  flex: 1;
}

.page-title {
  font-size: 28px;
  font-weight: 800;
  color: var(--text-primary);
  margin: 0 0 6px 0;
  line-height: 1.2;
  letter-spacing: -0.02em;
}

.page-description {
  font-size: 15px;
  color: var(--text-secondary);
  margin: 0;
  font-weight: 500;
}

.refresh-btn {
  border-radius: var(--radius-md);
  height: 42px;
  padding: 0 20px;
  font-weight: 600;
  box-shadow: var(--shadow-sm);
  transition: all var(--transition-base);
}

.refresh-btn:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
}

/* Stats Row */
.dashboard-stats-row {
  padding: 0 24px;
  margin-bottom: 24px;
}

.stat-card {
  background: linear-gradient(135deg, var(--card) 0%, rgba(255, 255, 255, 0.95) 100%);
  border-radius: var(--radius-lg);
  padding: var(--spacing-xl);
  border: 1px solid var(--border);
  box-shadow: var(--shadow-sm);
  transition: all var(--transition-base);
  position: relative;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  gap: 16px;
  cursor: pointer;
}

.stat-card::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 4px;
  background: var(--accent-color, var(--primary));
  transform: scaleX(0);
  transform-origin: left;
  transition: transform var(--transition-base);
}

.stat-card::after {
  content: '';
  position: absolute;
  top: -50%;
  right: -50%;
  width: 100%;
  height: 100%;
  background: radial-gradient(circle, var(--accent-color, rgba(79, 70, 229, 0.08)) 0%, transparent 70%);
  opacity: 0;
  transition: opacity var(--transition-base);
}

.stat-card:hover {
  box-shadow: var(--shadow-lg);
  transform: translateY(-6px);
  border-color: var(--border-dark);
}

.stat-card:hover::before {
  transform: scaleX(1);
}

.stat-card:hover::after {
  opacity: 1;
}

.stat-glow {
  position: absolute;
  top: -20px;
  right: -20px;
  width: 80px;
  height: 80px;
  background: var(--accent-color, var(--primary));
  border-radius: 50%;
  filter: blur(40px);
  opacity: 0;
  transition: opacity var(--transition-base);
}

.stat-card:hover .stat-glow {
  opacity: 0.15;
}

.stat-icon-wrapper {
  width: 52px;
  height: 52px;
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all var(--transition-base);
  position: relative;
}

.stat-icon-wrapper::before {
  content: '';
  position: absolute;
  inset: -2px;
  border-radius: var(--radius-md);
  background: var(--accent-color, var(--primary));
  opacity: 0;
  transition: opacity var(--transition-base);
}

.stat-card:hover .stat-icon-wrapper::before {
  opacity: 0.2;
}

.stat-icon {
  font-size: 24px;
  transition: transform var(--transition-base);
}

.stat-card:hover .stat-icon {
  transform: scale(1.1) rotate(5deg);
}

.stat-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.stat-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.stat-value {
  font-size: 32px;
  font-weight: 800;
  color: var(--text-primary);
  line-height: 1.1;
  display: flex;
  align-items: baseline;
  gap: 4px;
}

.stat-unit {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-tertiary);
}

.stat-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.stat-tag {
  font-weight: 600;
  border-radius: var(--radius-full);
  padding: 2px 10px;
  font-size: 11px;
  border: 1px solid currentColor;
  opacity: 0.9;
}

.stat-desc {
  font-size: 13px;
  color: var(--text-tertiary);
  font-weight: 500;
}

/* Charts Section */
.charts-section {
  padding: 0 24px;
  margin-bottom: 24px;
}

.chart-card {
  background: var(--card);
  border-radius: var(--radius-lg);
  padding: 24px;
  box-shadow: var(--shadow-sm);
  border: 1px solid var(--border-light);
  transition: all var(--transition-base);
}

.chart-card:hover {
  box-shadow: var(--shadow-lg);
  transform: translateY(-4px);
}

.chart-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.chart-card-title {
  font-size: 18px;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0;
  letter-spacing: -0.01em;
}

.chart-card-tag {
  font-weight: 600;
  border-radius: var(--radius-full);
  font-size: 12px;
  padding: 4px 14px;
  border: 1px solid currentColor;
  opacity: 0.9;
}

.chart-wrapper {
  height: 300px;
  position: relative;
}

.chart-wrapper-pie {
  display: flex;
  align-items: center;
  justify-content: center;
}

/* Expiring Card */
.expiring-card {
  background: var(--card);
  border-radius: var(--radius-lg);
  padding: 24px;
  margin: 0 24px 24px;
  box-shadow: var(--shadow-sm);
  border: 1px solid var(--border-light);
  transition: all var(--transition-base);
}

.expiring-card:hover {
  box-shadow: var(--shadow-lg);
}

.expiring-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--border);
}

.expiring-title {
  font-size: 18px;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0;
  letter-spacing: -0.01em;
}

.expiring-badge-wrapper :deep(.ant-badge-count) {
  font-weight: 600;
}

.expiring-empty {
  padding: 40px 0;
}

.expiring-list {
  max-height: 500px;
  overflow-y: auto;
}

.expiring-list::-webkit-scrollbar {
  width: 6px;
}

.expiring-list::-webkit-scrollbar-track {
  background: transparent;
}

.expiring-list::-webkit-scrollbar-thumb {
  background: var(--border-dark);
  border-radius: 3px;
}

.expiring-list::-webkit-scrollbar-thumb:hover {
  background: var(--primary);
}

.expiring-item {
  padding: 16px;
  transition: all var(--transition-base);
  border-radius: var(--radius-md);
  margin: 8px 0;
  border: 1px solid var(--border-light);
  background: var(--bg-secondary);
}

.expiring-item:hover {
  background: var(--card);
  transform: translateX(4px);
  border-color: var(--primary);
  box-shadow: var(--shadow-md);
}

.expiring-item :deep(.ant-list-item-meta) {
  align-items: center;
}

.expiring-item :deep(.ant-list-item-meta-avatar) {
  margin-right: 14px;
}

.expiring-item :deep(.ant-list-item-meta-avatar .ant-avatar) {
  box-shadow: var(--shadow-sm);
  transition: all var(--transition-base);
}

.expiring-item:hover :deep(.ant-list-item-meta-avatar .ant-avatar) {
  transform: scale(1.05);
  box-shadow: var(--shadow-md);
}

.instance-name {
  font-weight: 600;
  font-size: 15px;
  color: var(--text-primary);
}

.instance-id {
  font-family: 'JetBrains Mono', Consolas, 'Liberation Mono', Menlo, monospace;
  font-size: 13px;
  color: var(--text-tertiary);
  background: var(--bg-tertiary);
  padding: 2px 8px;
  border-radius: 4px;
}

.expire-tag {
  font-weight: 600;
  border-radius: var(--radius-full);
  border: 1px solid currentColor;
}

.expire-time {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--text-primary);
  font-size: 14px;
  font-weight: 600;
}

.expire-time :deep(.anticon) {
  font-size: 16px;
  color: var(--primary);
}

/* Ant Design overrides for cards */
:deep(.ant-card) {
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-sm);
  border: 1px solid var(--border-light);
  transition: all var(--transition-base);
}

:deep(.ant-card:hover) {
  box-shadow: var(--shadow-lg);
}

:deep(.ant-card-body) {
  padding: 24px;
}

:deep(.ant-statistic-title) {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin-bottom: 12px;
}

:deep(.ant-statistic-content) {
  display: flex;
  align-items: baseline;
  gap: 8px;
}

:deep(.ant-card-head) {
  border-bottom: 1px solid var(--border);
  padding: 20px 24px;
}

:deep(.ant-card-head-title) {
  font-size: 18px;
  font-weight: 700;
  color: var(--text-primary);
}

:deep(.ant-card-head-wrapper) {
  align-items: center;
}

:deep(.ant-list-item) {
  padding: 0;
}

:deep(.ant-list-item-meta-title) {
  font-size: 15px;
  font-weight: 600;
  color: var(--text-primary);
}

:deep(.ant-list-item-meta-description) {
  font-size: 13px;
  color: var(--text-tertiary);
}

:deep(.ant-list-item-action) {
  margin-left: 16px;
}

:deep(.ant-empty-description) {
  color: var(--text-tertiary);
  font-weight: 500;
}

/* Responsive */
@media (max-width: 768px) {
  .dashboard-page {
    padding: 0;
  }

  .page-header {
    flex-direction: column;
    padding: 20px 20px 0;
    margin-bottom: 24px;
  }

  .page-title {
    font-size: 24px;
  }

  .page-description {
    font-size: 14px;
  }

  .dashboard-stats-row {
    padding: 0 20px;
    margin-bottom: 20px;
  }

  .stat-card {
    padding: var(--spacing-lg);
  }

  .stat-value {
    font-size: 28px;
  }

  .charts-section {
    padding: 0 20px;
  }

  .chart-card {
    padding: 16px;
  }

  .chart-card-header {
    margin-bottom: 16px;
  }

  .chart-card-title {
    font-size: 16px;
  }

  .chart-wrapper {
    height: 240px;
  }

  .expiring-card {
    margin: 0 20px 20px;
    padding: 16px;
  }

  .expiring-header {
    margin-bottom: 16px;
    padding-bottom: 12px;
  }

  .expiring-title {
    font-size: 16px;
  }

  .expiring-item {
    padding: 12px;
  }
}

@media (max-width: 576px) {
  .dashboard-stats-row :deep(.ant-col) {
    flex: 0 0 100%;
    max-width: 100%;
  }
}
</style>
