<template>
  <div class="dashboard-page">
    <!-- Page Header -->
    <div class="page-header">
      <div class="page-header-left">
        <h1 class="page-title">控制台总览</h1>
        <p class="page-subtitle">查看您的云服务使用情况和账户信息</p>
      </div>
      <div class="page-header-actions">
        <a-button @click="refreshDashboard" :loading="loading">
          <ReloadOutlined />
          刷新
        </a-button>
      </div>
    </div>

    <!-- Stats Grid -->
    <div class="stats-section">
      <div class="section-title">数据概览</div>
      <div class="stats-grid">
        <router-link
          v-for="stat in mainStats"
          :key="stat.key"
          :to="stat.route"
          class="stat-card"
        >
          <div class="stat-icon" :style="{ background: stat.color + '15', color: stat.color }">
            <component :is="stat.icon" />
          </div>
          <div class="stat-content">
            <div class="stat-value" :style="{ color: stat.color }">{{ stat.value }}</div>
            <div class="stat-label">{{ stat.label }}</div>
          </div>
          <div class="stat-arrow">
            <RightOutlined />
          </div>
        </router-link>
      </div>
    </div>

    <!-- Status Cards -->
    <div class="status-section">
      <div class="section-title">快捷入口</div>
      <div class="status-grid">
        <router-link
          v-for="item in quickActions"
          :key="item.key"
          :to="item.route"
          class="status-card"
        >
          <div class="status-icon" :style="{ color: item.color }">
            <component :is="item.icon" />
          </div>
          <div class="status-content">
            <div class="status-label">{{ item.label }}</div>
            <div class="status-value">{{ item.value }}</div>
          </div>
        </router-link>
      </div>
    </div>

    <!-- Content Grid -->
    <div class="content-grid">
      <!-- Spend Trend -->
      <div class="content-card">
        <div class="card-header">
          <div class="card-title">
            <LineChartOutlined />
            <span>消费趋势</span>
          </div>
          <a-tag color="blue">近30天</a-tag>
        </div>
        <div class="chart-wrapper">
          <LineChart :data="charts.spendTrend" />
        </div>
      </div>

      <!-- Order Status -->
      <div class="content-card">
        <div class="card-header">
          <div class="card-title">
            <PieChartOutlined />
            <span>订单分布</span>
          </div>
        </div>
        <div class="chart-wrapper chart-wrapper-pie">
          <PieChart :data="charts.orderStatus" />
        </div>
      </div>
    </div>

    <!-- Expiring Instances -->
    <div class="expiring-section">
      <div class="expiring-card">
        <div class="expiring-header">
          <div class="expiring-title">
            <ClockCircleOutlined />
            <span>即将到期</span>
          </div>
          <a-button type="link" size="small" @click="() => router.push('/console/vps')">
            查看全部
            <RightOutlined />
          </a-button>
        </div>

        <a-empty
          v-if="expiring.length === 0"
          description=" "
          :image="Empty.PRESENTED_IMAGE_SIMPLE"
          class="expiring-empty"
        >
          <template #description>
            <div class="empty-state">
              <CheckCircleOutlined />
              <span>暂无即将到期的实例</span>
            </div>
          </template>
        </a-empty>

        <div v-else class="expiring-list">
          <div
            v-for="item in expiring"
            :key="item.id || item.ID"
            class="expiring-item"
            :class="{ 'expiring-critical': isCritical(item.expire_at || item.ExpireAt) }"
          >
            <div class="expiring-icon" :style="{ background: getExpireBgColor(item.expire_at || item.ExpireAt) }">
              <CloudServerOutlined />
            </div>
            <div class="expiring-info">
              <div class="expiring-name">{{ item.name || item.Name }}</div>
              <div class="expiring-meta">
                <span class="expiring-id">ID: {{ item.id || item.ID }}</span>
                <a-tag :color="getExpireTagColor(item.expire_at || item.ExpireAt)" size="small">
                  {{ getExpireLabel(item.expire_at || item.ExpireAt) }}
                </a-tag>
              </div>
            </div>
            <div class="expiring-time">{{ formatExpireTime(item.expire_at || item.ExpireAt) }}</div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, onMounted } from "vue";
import { useRouter } from "vue-router";
import { useAuthStore } from "@/stores/auth";
import { useDashboardStore } from "@/stores/dashboard";
import {
  ReloadOutlined,
  WalletOutlined,
  CloudServerOutlined,
  FileTextOutlined,
  CreditCardOutlined,
  ClockCircleOutlined,
  SafetyOutlined,
  ShoppingCartOutlined,
  LineChartOutlined,
  PieChartOutlined,
  RightOutlined,
  CheckCircleOutlined
} from "@ant-design/icons-vue";
import { Empty } from "ant-design-vue";
import LineChart from "@/components/Charts/LineChart.vue";
import PieChart from "@/components/Charts/PieChart.vue";

const router = useRouter();
const auth = useAuthStore();
const store = useDashboardStore();
const loading = ref(false);
Empty.PRESENTED_IMAGE_SIMPLE = Empty.PRESENTED_IMAGE_SIMPLE;

const formatMoney = (amount, currency = "CNY") => {
  const value = Number(amount ?? 0);
  if (Number.isNaN(value)) return "-";
  return `¥${value.toFixed(2)}`;
};

const mainStats = computed(() => {
  const balance = store.metrics.balance ?? 0;
  const currency = store.metrics.currency || "CNY";

  return [
    {
      key: "balance",
      label: "账户余额",
      value: formatMoney(balance, currency),
      color: "#0066FF",
      icon: WalletOutlined,
      route: "/console/billing"
    },
    {
      key: "vps",
      label: "云服务器",
      value: `${store.metrics.vps_total || 0} 台`,
      color: "#059669",
      icon: CloudServerOutlined,
      route: "/console/vps"
    },
    {
      key: "orders",
      label: "全部订单",
      value: `${store.metrics.orders_total || 0} 单`,
      color: "#8b5cf6",
      icon: FileTextOutlined,
      route: "/console/orders"
    },
    {
      key: "spend",
      label: "近30天消费",
      value: formatMoney(store.metrics.spend_30d, currency),
      color: "#f59e0b",
      icon: CreditCardOutlined,
      route: "/console/orders"
    }
  ];
});

const quickActions = computed(() => {
  const realnameStatus = store.metrics.realname_status;
  const realnameLabels = {
    verified: "已认证",
    failed: "未通过",
    pending: "审核中",
    disabled: "未启用",
    unverified: "未认证"
  };

  return [
    {
      key: "realname",
      label: "实名认证",
      value: realnameLabels[realnameStatus] || "未认证",
      icon: SafetyOutlined,
      color: realnameStatus === "verified" ? "#059669" : "#f59e0b",
      route: "/console/realname"
    },
    {
      key: "expiring",
      label: "即将到期",
      value: `${store.metrics.expiring || 0} 台`,
      icon: ClockCircleOutlined,
      color: store.metrics.expiring > 0 ? "#ef4444" : "#94a3b8",
      route: "/console/vps"
    },
    {
      key: "cart",
      label: "购物车",
      value: `${store.metrics.cart_items || 0} 件`,
      icon: ShoppingCartOutlined,
      color: "#0066FF",
      route: "/console/cart"
    },
    {
      key: "pending",
      label: "待处理",
      value: `${(store.metrics.pending_orders || 0) + (store.metrics.pending_payment || 0)} 单`,
      icon: FileTextOutlined,
      color: "#ec4899",
      route: "/console/orders"
    }
  ];
});

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
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit"
  });
};

const isCritical = (value) => {
  if (!value) return false;
  const daysLeft = Math.ceil((new Date(value).getTime() - Date.now()) / (24 * 3600 * 1000));
  return daysLeft <= 1;
};

const getExpireLabel = (value) => {
  if (!value) return "-";
  const daysLeft = Math.ceil((new Date(value).getTime() - Date.now()) / (24 * 3600 * 1000));
  if (daysLeft <= 0) return "已到期";
  if (daysLeft === 1) return "今天到期";
  if (daysLeft <= 3) return `${daysLeft}天后`;
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

const getExpireBgColor = (value) => {
  if (!value) return "#94a3b8";
  const daysLeft = Math.ceil((new Date(value).getTime() - Date.now()) / (24 * 3600 * 1000));
  if (daysLeft <= 1) return "#ef4444";
  if (daysLeft <= 3) return "#f59e0b";
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
}

/* Page Header */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 32px;
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

/* Section */
.section-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin-bottom: 16px;
}

/* Stats Section */
.stats-section {
  margin-bottom: 32px;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
}

.stat-card {
  display: flex;
  align-items: center;
  gap: 16px;
  background: #fff;
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 20px;
  text-decoration: none;
  transition: all 0.2s ease;
  position: relative;
  overflow: hidden;
}

.stat-card::before {
  content: '';
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 3px;
  background: var(--stat-color, var(--primary));
  transform: scaleY(0);
  transition: transform 0.2s ease;
}

.stat-card:hover {
  border-color: var(--primary);
  box-shadow: var(--shadow-md);
  transform: translateY(-2px);
}

.stat-card:hover::before {
  transform: scaleY(1);
}

.stat-icon {
  width: 48px;
  height: 48px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  flex-shrink: 0;
}

.stat-content {
  flex: 1;
  min-width: 0;
}

.stat-value {
  font-size: 22px;
  font-weight: 700;
  line-height: 1.2;
  margin-bottom: 2px;
}

.stat-label {
  font-size: 13px;
  color: var(--text-secondary);
}

.stat-arrow {
  font-size: 12px;
  color: var(--text-tertiary);
  transition: all 0.2s ease;
}

.stat-card:hover .stat-arrow {
  color: var(--primary);
  transform: translateX(4px);
}

/* Status Section */
.status-section {
  margin-bottom: 32px;
}

.status-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
}

.status-card {
  display: flex;
  align-items: center;
  gap: 12px;
  background: #fff;
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 16px;
  text-decoration: none;
  transition: all 0.2s ease;
}

.status-card:hover {
  border-color: var(--primary);
  box-shadow: var(--shadow-sm);
  transform: translateX(4px);
}

.status-icon {
  font-size: 18px;
  flex-shrink: 0;
}

.status-content {
  flex: 1;
  min-width: 0;
}

.status-label {
  font-size: 12px;
  color: var(--text-tertiary);
  margin-bottom: 2px;
}

.status-value {
  font-size: 15px;
  font-weight: 600;
  color: var(--text-primary);
}

/* Content Grid */
.content-grid {
  display: grid;
  grid-template-columns: 2fr 1fr;
  gap: 16px;
  margin-bottom: 32px;
}

.content-card {
  background: #fff;
  border: 1px solid var(--border);
  border-radius: 12px;
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

.chart-wrapper {
  padding: 20px;
  height: 280px;
}

.chart-wrapper-pie {
  display: flex;
  align-items: center;
  justify-content: center;
}

/* Expiring Section */
.expiring-section {
  margin-bottom: 32px;
}

.expiring-card {
  background: #fff;
  border: 1px solid var(--border);
  border-radius: 12px;
  overflow: hidden;
}

.expiring-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border);
}

.expiring-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 600;
  color: var(--text-primary);
}

.expiring-title .anticon {
  font-size: 16px;
  color: var(--primary);
}

.expiring-empty {
  padding: 60px 20px;
}

.empty-state {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--success);
  font-size: 14px;
}

.empty-state .anticon {
  font-size: 16px;
}

.expiring-list {
  padding: 12px;
}

.expiring-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px;
  background: var(--bg-secondary);
  border-radius: 10px;
  margin-bottom: 8px;
  border: 1px solid transparent;
  transition: all 0.15s ease;
  cursor: pointer;
}

.expiring-item:last-child {
  margin-bottom: 0;
}

.expiring-item:hover {
  background: #fff;
  border-color: var(--border);
  box-shadow: var(--shadow-sm);
}

.expiring-item.expiring-critical {
  background: rgba(239, 68, 68, 0.08);
  border-color: rgba(239, 68, 68, 0.2);
}

.expiring-icon {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 16px;
  flex-shrink: 0;
}

.expiring-info {
  flex: 1;
  min-width: 0;
}

.expiring-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 4px;
}

.expiring-meta {
  display: flex;
  align-items: center;
  gap: 8px;
}

.expiring-id {
  font-size: 12px;
  color: var(--text-tertiary);
  font-family: 'Consolas', 'Monaco', monospace;
}

.expiring-time {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-secondary);
  white-space: nowrap;
}

/* Responsive */
@media (max-width: 1200px) {
  .stats-grid,
  .status-grid {
    grid-template-columns: repeat(2, 1fr);
  }

  .content-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 768px) {
  .page-header {
    flex-direction: column;
    gap: 16px;
  }

  .stats-grid,
  .status-grid {
    grid-template-columns: 1fr;
  }

  .page-title {
    font-size: 20px;
  }

  .chart-wrapper {
    height: 240px;
    padding: 16px;
  }

  .expiring-item {
    flex-wrap: wrap;
  }

  .expiring-time {
    width: 100%;
    margin-top: 8px;
  }
}
</style>
