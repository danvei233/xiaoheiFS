<template>
  <div class="dashboard-page art-full-height">
    <div class="page-heading">
      <div>
        <h1>控制台总览</h1>
        <p>查看您的云服务使用情况和账户信息</p>
      </div>
      <div class="heading-actions">
        <ElButton :icon="Refresh" :loading="loading" @click="refreshDashboard">刷新</ElButton>
      </div>
    </div>

    <div class="metrics-grid">
      <ElCard
        v-for="item in metrics"
        :key="item.key"
        class="metric-card"
        shadow="never"
        tabindex="0"
        @click="router.push(item.to)"
        @keyup.enter="router.push(item.to)"
      >
        <div class="metric-content">
          <div class="metric-icon" :class="`is-${item.tone}`">
            <ElIcon><component :is="item.icon" /></ElIcon>
          </div>
          <div class="metric-copy">
            <span>{{ item.label }}</span>
            <strong>{{ item.value }}</strong>
          </div>
          <ElIcon class="metric-arrow"><ArrowRight /></ElIcon>
        </div>
      </ElCard>
    </div>

    <div class="charts-grid">
      <ElCard class="chart-card" shadow="never">
        <template #header>
          <div class="card-heading">
            <div>
              <h2>消费趋势</h2>
              <p>近30天</p>
            </div>
            <ElTag effect="plain">{{ currency }}</ElTag>
          </div>
        </template>
        <ArtLineChart
          :data="spendTrend.values"
          :x-axis-data="spendTrend.labels"
          :loading="loading"
          height="280px"
          show-area-color
        />
      </ElCard>

      <ElCard class="chart-card" shadow="never">
        <template #header>
          <div class="card-heading">
            <div>
              <h2>订单分布</h2>
            </div>
          </div>
        </template>
        <ArtRingChart
          :data="orderStatus"
          :loading="loading"
          height="280px"
          show-legend
          legend-position="right"
        />
      </ElCard>
    </div>

    <ElCard class="expiring-card" shadow="never">
      <template #header>
        <div class="card-heading">
          <div>
            <h2>即将到期</h2>
          </div>
          <ElButton link type="primary" @click="router.push('/console/vps')">
            查看全部
            <ElIcon class="el-icon--right"><ArrowRight /></ElIcon>
          </ElButton>
        </div>
      </template>

      <ElEmpty v-if="!expiringList.length" description="暂无即将到期的实例" />
      <div v-else class="expiring-list">
        <div v-for="item in expiringList" :key="item.id ?? item.ID" class="expiring-row">
          <div class="instance-icon">
            <ElIcon><Monitor /></ElIcon>
          </div>
          <div class="instance-copy">
            <strong>{{ item.name ?? item.Name ?? `VPS-${item.id ?? item.ID}` }}</strong>
            <span>ID: {{ item.id ?? item.ID ?? '-' }}</span>
          </div>
          <ElTag :type="expireType(item.expire_at ?? item.ExpireAt)" effect="plain">
            {{ expireLabel(item.expire_at ?? item.ExpireAt) }}
          </ElTag>
          <time>{{ formatDateTime(item.expire_at ?? item.ExpireAt) }}</time>
        </div>
      </div>
    </ElCard>
  </div>
</template>

<script setup lang="ts">
  import {
    ArrowRight,
    Clock,
    CreditCard,
    Document,
    Finished,
    Monitor,
    Refresh,
    ShoppingCart,
    Wallet
  } from '@element-plus/icons-vue'
  import type { Component } from 'vue'
  import type { RouteLocationRaw } from 'vue-router'
  import { useDashboardStore } from '@/stores/dashboard'
  import { formatMoney as formatMoneyUtil } from '@/utils/money'

  defineOptions({ name: 'ConsoleDashboard' })

  type DashboardMetrics = {
    balance?: number
    currency?: string
    cart_items?: number
    expiring?: number
    orders_total?: number
    pending_orders?: number
    pending_payment?: number
    realname_status?: string
    spend_30d?: number
    vps_total?: number
  }

  type DashboardInstance = {
    id?: number | string
    ID?: number | string
    name?: string
    Name?: string
    expire_at?: string
    ExpireAt?: string
  }

  type OrderStatusDatum = {
    name?: string
    value?: number
  }

  type DashboardCharts = {
    expiringList?: DashboardInstance[]
    orderStatus?: OrderStatusDatum[]
    spendTrend?: { labels?: unknown[]; values?: unknown[] }
  }

  type DashboardViewStore = Omit<ReturnType<typeof useDashboardStore>, 'metrics' | 'charts'> & {
    metrics: DashboardMetrics
    charts: DashboardCharts
  }

  type DashboardMetric = {
    key: string
    label: string
    value: string
    to: RouteLocationRaw
    icon: Component
    tone: string
  }

  const router = useRouter()
  const dashboard = useDashboardStore() as unknown as DashboardViewStore
  const loading = ref(false)

  const currency = computed(() => String(dashboard.metrics.currency || 'CNY'))
  const formatMoney = (amount: unknown) => formatMoneyUtil(amount, currency.value)

  const realnameLabel = computed(() => {
    const labels: Record<string, string> = {
      verified: '已认证',
      pending: '审核中',
      failed: '未通过',
      rejected: '未通过',
      disabled: '未启用',
      unverified: '未认证'
    }
    return labels[String(dashboard.metrics.realname_status || '')] || '未认证'
  })

  const metrics = computed<DashboardMetric[]>(() => [
    {
      key: 'balance',
      label: '账户余额',
      value: formatMoney(dashboard.metrics.balance),
      to: '/console/billing',
      icon: Wallet,
      tone: 'primary'
    },
    {
      key: 'vps',
      label: '云服务器',
      value: `${dashboard.metrics.vps_total || 0} 台`,
      to: '/console/vps',
      icon: Monitor,
      tone: 'success'
    },
    {
      key: 'orders',
      label: '全部订单',
      value: `${dashboard.metrics.orders_total || 0} 单`,
      to: '/console/orders',
      icon: Document,
      tone: 'info'
    },
    {
      key: 'spend',
      label: '近 30 天消费',
      value: formatMoney(dashboard.metrics.spend_30d),
      to: '/console/orders',
      icon: CreditCard,
      tone: 'warning'
    },
    {
      key: 'realname',
      label: '实名认证',
      value: realnameLabel.value,
      to: '/console/realname',
      icon: Finished,
      tone: dashboard.metrics.realname_status === 'verified' ? 'success' : 'muted'
    },
    {
      key: 'expiring',
      label: '即将到期',
      value: `${dashboard.metrics.expiring || 0} 台`,
      to: '/console/vps',
      icon: Clock,
      tone: Number(dashboard.metrics.expiring || 0) > 0 ? 'danger' : 'muted'
    },
    {
      key: 'cart',
      label: '购物车',
      value: `${dashboard.metrics.cart_items || 0} 件`,
      to: { name: 'PublicCart' },
      icon: ShoppingCart,
      tone: 'primary'
    },
    {
      key: 'pending',
      label: '待处理',
      value: `${(dashboard.metrics.pending_orders || 0) + (dashboard.metrics.pending_payment || 0)} 单`,
      to: '/console/orders',
      icon: Document,
      tone: 'warning'
    }
  ])

  const spendTrend = computed(() => ({
    labels: (dashboard.charts.spendTrend?.labels || []).map(String),
    values: (dashboard.charts.spendTrend?.values || []).map((value: unknown) => Number(value || 0))
  }))

  const orderStatusLabels: Record<string, string> = {
    draft: '草稿',
    pending_payment: '待支付',
    pending_review: '待审核',
    approved: '已通过',
    provisioning: '开通中',
    active: '已完成',
    rejected: '已驳回',
    canceled: '已取消',
    failed: '失败'
  }
  const orderStatus = computed(() =>
    (dashboard.charts.orderStatus || []).map((item) => ({
      name: orderStatusLabels[item.name] || item.name || '未知',
      value: Number(item.value || 0)
    }))
  )
  const expiringList = computed(() => dashboard.charts.expiringList || [])

  const daysUntil = (value: unknown) => {
    if (!value) return Number.POSITIVE_INFINITY
    const time = new Date(String(value)).getTime()
    return Number.isNaN(time)
      ? Number.POSITIVE_INFINITY
      : Math.ceil((time - Date.now()) / (24 * 3600 * 1000))
  }
  const expireLabel = (value: unknown) => {
    const days = daysUntil(value)
    if (!Number.isFinite(days)) return '-'
    if (days <= 0) return '已到期'
    if (days === 1) return '今天到期'
    return `${days} 天后`
  }
  const expireType = (value: unknown): 'danger' | 'warning' | 'info' => {
    const days = daysUntil(value)
    if (days <= 1) return 'danger'
    if (days <= 3) return 'warning'
    return 'info'
  }
  const formatDateTime = (value: unknown) => {
    if (!value) return '-'
    const date = new Date(String(value))
    return Number.isNaN(date.getTime())
      ? String(value)
      : date.toLocaleString('zh-CN', {
          hour12: false,
          month: '2-digit',
          day: '2-digit',
          hour: '2-digit',
          minute: '2-digit'
        })
  }

  const refreshDashboard = async () => {
    loading.value = true
    try {
      await dashboard.fetchUserDashboard()
    } finally {
      loading.value = false
    }
  }

  onMounted(refreshDashboard)
</script>

<style scoped lang="scss">
  .dashboard-page {
    gap: 16px;
    overflow: auto;
  }

  .page-heading {
    display: flex;
    gap: 20px;
    align-items: flex-start;
    justify-content: space-between;
    padding-top: 4px;

    h1 {
      margin: 0 0 6px;
      font-size: 24px;
      font-weight: 700;
      color: var(--art-gray-900);
    }

    p {
      margin: 0;
      font-size: 13px;
      color: var(--art-gray-600);
    }
  }

  .heading-actions {
    display: flex;
    gap: 8px;
  }

  .metrics-grid {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 12px;
  }

  .metric-card {
    cursor: pointer;
    background: var(--default-box-color);
    border: 1px solid var(--art-card-border);
    transition:
      border-color 0.18s ease,
      transform 0.18s ease;

    &:hover,
    &:focus-visible {
      border-color: var(--el-color-primary-light-5);
      outline: none;
      transform: translateY(-2px);
    }
  }

  .metric-content {
    display: flex;
    gap: 12px;
    align-items: center;
  }

  .metric-icon {
    display: grid;
    flex: 0 0 40px;
    place-items: center;
    width: 40px;
    height: 40px;
    border-radius: 10px;

    &.is-primary {
      color: var(--theme-color);
      background: var(--el-color-primary-light-9);
    }

    &.is-success {
      color: var(--el-color-success);
      background: var(--el-color-success-light-9);
    }

    &.is-info {
      color: var(--el-color-info);
      background: var(--el-color-info-light-9);
    }

    &.is-warning {
      color: var(--el-color-warning);
      background: var(--el-color-warning-light-9);
    }

    &.is-danger {
      color: var(--el-color-danger);
      background: var(--el-color-danger-light-9);
    }

    &.is-muted {
      color: var(--art-gray-600);
      background: var(--art-gray-200);
    }
  }

  .metric-copy {
    display: flex;
    flex: 1;
    flex-direction: column;
    gap: 5px;
    min-width: 0;

    span {
      font-size: 12px;
      color: var(--art-gray-600);
    }

    strong {
      overflow: hidden;
      font-size: 20px;
      color: var(--art-gray-900);
      text-overflow: ellipsis;
      white-space: nowrap;
    }
  }

  .metric-arrow {
    color: var(--art-gray-400);
  }

  .charts-grid {
    display: grid;
    grid-template-columns: minmax(0, 1.45fr) minmax(320px, 0.85fr);
    gap: 12px;
  }

  .chart-card,
  .expiring-card {
    background: var(--default-box-color);
    border: 1px solid var(--art-card-border);
  }

  .card-heading {
    display: flex;
    gap: 16px;
    align-items: center;
    justify-content: space-between;

    h2 {
      margin: 0;
      font-size: 15px;
      font-weight: 600;
      color: var(--art-gray-900);
    }

    p {
      margin: 4px 0 0;
      font-size: 12px;
      color: var(--art-gray-600);
    }
  }

  .expiring-list {
    display: flex;
    flex-direction: column;
  }

  .expiring-row {
    display: grid;
    grid-template-columns: 36px minmax(160px, 1fr) auto 150px;
    gap: 12px;
    align-items: center;
    width: 100%;
    padding: 12px 4px;
    color: inherit;
    text-align: left;
    border-bottom: 1px solid var(--default-border);

    &:last-child {
      border-bottom: 0;
    }

    time {
      font-size: 12px;
      color: var(--art-gray-600);
      text-align: right;
    }
  }

  .instance-icon {
    display: grid;
    place-items: center;
    width: 34px;
    height: 34px;
    color: var(--theme-color);
    background: var(--el-color-primary-light-9);
    border-radius: 9px;
  }

  .instance-copy {
    display: flex;
    flex-direction: column;
    gap: 3px;
    min-width: 0;

    strong {
      overflow: hidden;
      color: var(--art-gray-900);
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    span {
      font-size: 12px;
      color: var(--art-gray-600);
    }
  }

  @media (width <= 980px) {
    .metrics-grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }

    .charts-grid {
      grid-template-columns: 1fr;
    }
  }

  @media (width <= 640px) {
    .dashboard-page {
      height: auto;
      overflow: visible;
    }

    .page-heading {
      flex-direction: column;
    }

    .heading-actions {
      width: 100%;

      .el-button {
        flex: 1;
      }
    }

    .metrics-grid {
      grid-template-columns: 1fr;
    }

    .expiring-row {
      grid-template-columns: 36px minmax(0, 1fr) auto;

      time {
        display: none;
      }
    }
  }
</style>
