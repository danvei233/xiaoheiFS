<template>
  <div class="dashboard-page art-full-height">
    <ElCard class="hero-card" shadow="never">
      <div class="hero-main">
        <div>
          <h2>运营总览</h2>
          <p>围绕收入、订单、资源和风险信号的后台首页，保留旧后台 Dashboard 的核心经营视角。</p>
          <ElSpace wrap>
            <ElTag type="success" effect="light">收入口径：已确认订单</ElTag>
            <ElTag effect="light">订单样本：{{ orders.length }}</ElTag>
            <ElTag effect="light">实例样本：{{ vpsList.length }}</ElTag>
          </ElSpace>
        </div>

        <ElSpace wrap>
          <ElRadioGroup v-model="period" @change="handlePeriodChange">
            <ElRadioButton label="day">近 30 天</ElRadioButton>
            <ElRadioButton label="month">近 12 月</ElRadioButton>
          </ElRadioGroup>
          <ElButton type="primary" :loading="loading" @click="reloadAll">刷新</ElButton>
        </ElSpace>
      </div>
    </ElCard>

    <ElAlert
      v-if="!canView"
      type="warning"
      :closable="false"
      title="当前账号缺少概览权限"
      description="请为当前管理员分配 dashboard.overview 后再查看运营总览。"
    />

    <template v-else>
      <ElAlert
        v-if="loadWarnings.length"
        type="warning"
        :closable="false"
        title="部分面板已降级展示"
      >
        <template #default>
          <div class="warning-list">
            <div v-for="item in loadWarnings" :key="item">{{ item }}</div>
          </div>
        </template>
      </ElAlert>

      <ElRow :gutter="16" class="section-row">
        <ElCol :xs="24" :sm="12" :lg="8" :xl="4">
          <ElCard class="metric-card" v-loading="loading">
            <div class="metric-title">累计收入</div>
            <ElStatistic :value="toYuanNumber(overview.revenue)" prefix="¥" :precision="2" />
            <div class="metric-foot" :class="revenueTrend.className">{{ revenueTrend.text }}</div>
          </ElCard>
        </ElCol>
        <ElCol :xs="24" :sm="12" :lg="8" :xl="4">
          <ElCard class="metric-card" v-loading="loading">
            <div class="metric-title">总订单</div>
            <ElStatistic :value="overview.total_orders" />
            <div class="metric-foot">已处理率 {{ orderHandleRate.toFixed(2) }}%</div>
          </ElCard>
        </ElCol>
        <ElCol :xs="24" :sm="12" :lg="8" :xl="4">
          <ElCard class="metric-card" v-loading="loading">
            <div class="metric-title">待审核订单</div>
            <ElStatistic :value="overview.pending_review" />
            <div class="metric-foot danger">需要运营优先处理</div>
          </ElCard>
        </ElCol>
        <ElCol :xs="24" :sm="12" :lg="8" :xl="4">
          <ElCard class="metric-card" v-loading="loading">
            <div class="metric-title">VPS 总数</div>
            <ElStatistic :value="overview.vps_count" />
            <div class="metric-foot">{{ statusSummary }}</div>
          </ElCard>
        </ElCol>
        <ElCol :xs="24" :sm="12" :lg="8" :xl="4">
          <ElCard class="metric-card" v-loading="loading">
            <div class="metric-title">7 天内到期</div>
            <ElStatistic :value="overview.expiring_soon" />
            <div class="metric-foot warning">建议安排续费提醒</div>
          </ElCard>
        </ElCol>
        <ElCol :xs="24" :sm="12" :lg="8" :xl="4">
          <ElCard class="metric-card" v-loading="loading">
            <div class="metric-title">健康评分</div>
            <ElStatistic :value="healthScore" suffix="/100" />
            <div class="metric-foot" :class="healthScore < 70 ? 'danger' : 'success'">
              {{ healthComment }}
            </div>
          </ElCard>
        </ElCol>
      </ElRow>

      <ElRow :gutter="16" class="section-row">
        <ElCol :xs="24" :xl="16">
          <ElCard v-loading="loading">
            <template #header>运营告警与关注点</template>
            <ElEmpty
              v-if="!alerts.length"
              description="当前暂无高风险告警，可继续观察趋势变化。"
              :image-size="72"
            />
            <div v-else class="alert-list">
              <div v-for="item in alerts" :key="item.text" class="alert-item">
                <ElTag :type="item.level === 'high' ? 'danger' : 'warning'" effect="light">
                  {{ item.level === 'high' ? '高风险' : '中风险' }}
                </ElTag>
                <span>{{ item.text }}</span>
              </div>
            </div>
          </ElCard>
        </ElCol>

        <ElCol :xs="24" :xl="8">
          <ElCard v-loading="loading">
            <template #header>运行健康仪表</template>
            <div class="health-card">
              <ElProgress type="dashboard" :percentage="healthScore" :color="healthColor" />
              <div class="health-lines">
                <div>
                  <div class="health-line-label">
                    <span>CPU</span>
                    <strong>{{ toPercent(serverStatus.cpu_usage_percent) }}%</strong>
                  </div>
                  <ElProgress
                    :percentage="toPercent(serverStatus.cpu_usage_percent)"
                    :show-text="false"
                  />
                </div>
                <div>
                  <div class="health-line-label">
                    <span>内存</span>
                    <strong>{{ toPercent(serverStatus.mem_usage_percent) }}%</strong>
                  </div>
                  <ElProgress
                    :percentage="toPercent(serverStatus.mem_usage_percent)"
                    :show-text="false"
                  />
                </div>
                <div>
                  <div class="health-line-label">
                    <span>磁盘</span>
                    <strong>{{ toPercent(serverStatus.disk_usage_percent) }}%</strong>
                  </div>
                  <ElProgress
                    :percentage="toPercent(serverStatus.disk_usage_percent)"
                    :show-text="false"
                    :status="toPercent(serverStatus.disk_usage_percent) >= 85 ? 'exception' : ''"
                  />
                </div>
              </div>
            </div>
          </ElCard>
        </ElCol>
      </ElRow>

      <ElRow :gutter="16" class="section-row">
        <ElCol :xs="24" :xl="14">
          <ElCard v-loading="loading">
            <template #header>趋势分析</template>
            <ElTabs v-model="trendTab">
              <ElTabPane label="收入趋势" name="revenue">
                <ArtLineChart
                  v-if="revenueChart.labels.length && revenueChart.values.length"
                  height="320px"
                  :data="revenueChart.values"
                  :x-axis-data="revenueChart.labels"
                  :show-area-color="true"
                  :show-axis-line="false"
                />
                <ElEmpty v-else description="暂无收入趋势数据" :image-size="72" />
              </ElTabPane>
              <ElTabPane label="订单状态分布" name="order">
                <ArtBarChart
                  v-if="orderStatusChart.labels.length && orderStatusChart.values.length"
                  height="320px"
                  :data="orderStatusChart.values"
                  :x-axis-data="orderStatusChart.labels"
                  :show-axis-line="false"
                />
                <ElEmpty v-else description="暂无订单状态数据" :image-size="72" />
              </ElTabPane>
              <ElTabPane label="到期趋势" name="expire">
                <ArtLineChart
                  v-if="hasExpiringTrendData"
                  height="320px"
                  :data="expiringChart.values"
                  :x-axis-data="expiringChart.labels"
                  :show-area-color="true"
                  :show-axis-line="false"
                />
                <ElEmpty v-else description="暂无到期趋势数据" :image-size="72" />
              </ElTabPane>
            </ElTabs>
          </ElCard>
        </ElCol>

        <ElCol :xs="24" :xl="10">
          <ElCard v-loading="loading">
            <template #header>资源结构</template>
            <template v-if="vpsStatusChart.length">
              <ArtRingChart
                height="320px"
                :data="vpsStatusChart"
                :center-text="`运行中 ${runningVpsCount}`"
                :show-label="false"
                :show-legend="false"
              />
              <div class="status-list">
                <div v-for="row in vpsStatusRows" :key="row.name" class="status-item">
                  <div class="status-head">
                    <span>{{ row.name }}</span>
                    <span>{{ row.value }} / {{ row.ratio.toFixed(2) }}%</span>
                  </div>
                  <ElProgress :percentage="row.ratio" :show-text="false" :stroke-width="8" />
                </div>
              </div>
            </template>
            <ElEmpty v-else description="暂无资源状态数据" :image-size="72" />
          </ElCard>
        </ElCol>
      </ElRow>

      <ElRow :gutter="16" class="section-row">
        <ElCol :xs="24" :xl="12">
          <ElCard v-loading="loading">
            <template #header>待审核订单</template>
            <ElTable v-if="pendingOrders.length" :data="pendingOrders" border stripe row-key="id">
              <ElTableColumn prop="order_no" label="订单号" min-width="200" show-overflow-tooltip />
              <ElTableColumn prop="user_id" label="用户 ID" width="100" />
              <ElTableColumn label="金额" width="120">
                <template #default="{ row }">
                  <span class="amount-up">¥{{ formatMoney(row.total_amount) }}</span>
                </template>
              </ElTableColumn>
              <ElTableColumn label="创建时间" min-width="180">
                <template #default="{ row }">
                  <span class="subtle">{{ formatDateTime(row.created_at) }}</span>
                </template>
              </ElTableColumn>
            </ElTable>
            <ElEmpty v-else description="暂无待审核订单" :image-size="72" />
          </ElCard>
        </ElCol>

        <ElCol :xs="24" :xl="12">
          <ElCard v-loading="loading">
            <template #header>即将到期实例</template>
            <ElTable v-if="expiringVps.length" :data="expiringVps" border stripe row-key="id">
              <ElTableColumn prop="name" label="实例" min-width="160" show-overflow-tooltip />
              <ElTableColumn prop="user_id" label="用户 ID" width="100" />
              <ElTableColumn label="状态" width="120">
                <template #default="{ row }">
                  <ElTag :type="vpsStatusTagType(row.status)" effect="light">
                    {{ vpsStatusLabel(row.status) }}
                  </ElTag>
                </template>
              </ElTableColumn>
              <ElTableColumn label="到期时间" min-width="160">
                <template #default="{ row }">
                  {{ formatDateTime(row.expire_at) }}
                </template>
              </ElTableColumn>
              <ElTableColumn label="剩余" width="110">
                <template #default="{ row }">
                  <ElTag
                    :type="
                      row.days_left <= 3 ? 'danger' : row.days_left <= 7 ? 'warning' : 'primary'
                    "
                    effect="light"
                  >
                    {{ row.days_left <= 0 ? '已到期' : `${row.days_left} 天` }}
                  </ElTag>
                </template>
              </ElTableColumn>
            </ElTable>
            <ElEmpty v-else description="暂无到期压力实例" :image-size="72" />
          </ElCard>
        </ElCol>
      </ElRow>

      <ElCard v-loading="loading">
        <template #header>服务器状态</template>
        <template v-if="hasServerStatus">
          <ElDescriptions :column="3" border>
            <ElDescriptionsItem label="主机名">{{
              serverStatus.hostname || '-'
            }}</ElDescriptionsItem>
            <ElDescriptionsItem label="系统">{{ serverStatus.os || '-' }}</ElDescriptionsItem>
            <ElDescriptionsItem label="平台">{{ serverStatus.platform || '-' }}</ElDescriptionsItem>
            <ElDescriptionsItem label="内核">
              {{ serverStatus.kernel_version || '-' }}
            </ElDescriptionsItem>
            <ElDescriptionsItem label="CPU">{{ cpuText }}</ElDescriptionsItem>
            <ElDescriptionsItem label="运行时长">{{ uptimeText }}</ElDescriptionsItem>
          </ElDescriptions>
        </template>
        <ElEmpty v-else description="暂无服务器状态数据" :image-size="72" />
      </ElCard>
    </template>
  </div>
</template>

<script setup lang="ts">
  import { computed, reactive, ref, watch } from 'vue'
  import { storeToRefs } from 'pinia'
  import type {
    DashboardOverview,
    OrderRecord,
    RevenuePoint,
    ServerStatus,
    StatusPoint,
    VpsRecord
  } from '@/api/admin'
  import {
    fetchAdminDashboardOverview,
    fetchAdminDashboardRevenue,
    fetchAdminDashboardVpsStatus,
    fetchAdminOrders,
    fetchAdminServerStatus,
    fetchAdminVps,
    hasAdminPermission
  } from '@/api/admin'
  import ArtBarChart from '@/components/core/charts/art-bar-chart/index.vue'
  import ArtLineChart from '@/components/core/charts/art-line-chart/index.vue'
  import ArtRingChart from '@/components/core/charts/art-ring-chart/index.vue'
  import { useUserStore } from '@/store/modules/user'
  import { ElMessage } from 'element-plus'

  defineOptions({ name: 'DashboardOverviewPage' })

  interface DashboardOverviewModel {
    total_orders: number
    pending_review: number
    revenue: number
    vps_count: number
    expiring_soon: number
  }

  interface AlertItem {
    level: 'high' | 'medium'
    text: string
  }

  interface StatusRow {
    name: string
    value: number
    ratio: number
  }

  interface PendingOrderRow {
    id: number
    order_no: string
    user_id: number
    total_amount: number
    created_at: string
  }

  interface ExpiringVpsRow {
    id: number
    name: string
    user_id: number
    status: string
    expire_at: string
    days_left: number
  }

  interface ApiListLike<T> {
    items?: T[]
    total?: number
  }

  const REVENUE_WARNING = '收入趋势数据暂不可用，趋势面板已降级为基础概览。'
  const ORDER_WARNING = '订单明细权限不足或请求失败，订单状态分布与待审核列表暂不可用。'
  const VPS_WARNING = 'VPS 明细权限不足或请求失败，到期趋势与到期实例列表暂不可用。'
  const SERVER_WARNING = '服务器状态权限不足或请求失败，健康评分仅基于业务指标计算。'

  const userStore = useUserStore()
  const { info } = storeToRefs(userStore)

  const loading = ref(false)
  const initialized = ref(false)
  const period = ref<'day' | 'month'>('day')
  const trendTab = ref<'revenue' | 'order' | 'expire'>('revenue')
  const orders = ref<OrderRecord[]>([])
  const vpsList = ref<VpsRecord[]>([])
  const revenueSeries = ref<RevenuePoint[]>([])
  const vpsStatusSeries = ref<StatusPoint[]>([])
  const loadWarnings = ref<string[]>([])

  const overview = reactive<DashboardOverviewModel>(createEmptyOverview())
  const serverStatus = reactive<ServerStatus>(createEmptyServerStatus())

  const canView = computed(() => hasAdminPermission(info.value?.buttons, ['dashboard.overview']))
  const canLoadRevenue = computed(() =>
    hasAdminPermission(info.value?.buttons, ['dashboard.overview', 'dashboard.revenue'])
  )
  const canLoadOrders = computed(() =>
    hasAdminPermission(info.value?.buttons, ['order.list', 'order.view'])
  )
  const canLoadVps = computed(() =>
    hasAdminPermission(info.value?.buttons, ['vps.list', 'vps.view'])
  )
  const canLoadVpsStatus = computed(() =>
    hasAdminPermission(info.value?.buttons, ['dashboard.overview', 'dashboard.vps_status'])
  )
  const canLoadServer = computed(() => hasAdminPermission(info.value?.buttons, ['server.status']))

  const hasServerStatus = computed(() => {
    return Boolean(serverStatus.hostname || serverStatus.os || serverStatus.platform)
  })

  const revenueChart = computed(() => ({
    labels: revenueSeries.value.map((item) => readStr(item, 'date', 'Date')),
    values: revenueSeries.value.map((item) => readNum(item, 'amount', 'Amount') / 100)
  }))

  const revenueTrend = computed(() => {
    const values = revenueChart.value.values
    if (values.length < 2) {
      return { className: '', text: '暂无对比数据' }
    }

    const last = Number(values[values.length - 1] || 0)
    const prev = Number(values[values.length - 2] || 0)
    if (prev <= 0) {
      return { className: '', text: '暂无对比数据' }
    }

    const ratio = ((last - prev) / prev) * 100
    return ratio >= 0
      ? { className: 'success', text: `较上一周期 +${ratio.toFixed(2)}%` }
      : { className: 'danger', text: `较上一周期 ${ratio.toFixed(2)}%` }
  })

  const orderStatusChart = computed(() => {
    const counts = new Map<string, number>()
    orders.value.forEach((item) => {
      const key = orderStatusLabel(readStr(item, 'status', 'Status'))
      counts.set(key, (counts.get(key) || 0) + 1)
    })
    const labels = Array.from(counts.keys())
    return { labels, values: labels.map((label) => counts.get(label) || 0) }
  })

  const rawVpsStatusChart = computed(() => {
    if (vpsStatusSeries.value.length) {
      return vpsStatusSeries.value.map((item) => ({
        name: readStr(item, 'status', 'Status') || 'unknown',
        value: readNum(item, 'count', 'Count')
      }))
    }

    const counts = new Map<string, number>()
    vpsList.value.forEach((item) => {
      const key = readStr(item, 'status', 'Status') || 'unknown'
      counts.set(key, (counts.get(key) || 0) + 1)
    })

    return Array.from(counts.entries()).map(([name, value]) => ({ name, value }))
  })

  const vpsStatusChart = computed(() =>
    rawVpsStatusChart.value.map((item) => ({
      name: vpsStatusLabel(item.name),
      value: Number(item.value || 0)
    }))
  )

  const runningVpsCount = computed(() => {
    const row = rawVpsStatusChart.value.find(
      (item) => String(item.name || '').toLowerCase() === 'running'
    )
    return Number(row?.value || 0)
  })

  const vpsStatusRows = computed<StatusRow[]>(() => {
    const total =
      rawVpsStatusChart.value.reduce((sum, item) => sum + Number(item.value || 0), 0) || 1
    return rawVpsStatusChart.value
      .map((item) => ({
        name: vpsStatusLabel(item.name),
        value: Number(item.value || 0),
        ratio: Number(((Number(item.value || 0) / total) * 100).toFixed(2))
      }))
      .sort((a, b) => b.value - a.value)
  })

  const expiringChart = computed(() => {
    const today = new Date()
    const bucket = new Map<string, number>()
    for (let index = 0; index < 30; index += 1) {
      const date = new Date(today)
      date.setDate(today.getDate() + index)
      bucket.set(date.toISOString().slice(0, 10), 0)
    }

    vpsList.value.forEach((item) => {
      const expireAt = readStr(item, 'expire_at', 'ExpireAt')
      if (!expireAt) return
      const key = expireAt.slice(0, 10)
      if (bucket.has(key)) {
        bucket.set(key, (bucket.get(key) || 0) + 1)
      }
    })

    const labels = Array.from(bucket.keys())
    return { labels, values: labels.map((label) => bucket.get(label) || 0) }
  })

  const hasExpiringTrendData = computed(() => vpsList.value.length > 0)

  const orderHandleRate = computed(() => {
    if (!overview.total_orders) return 0
    return ((overview.total_orders - overview.pending_review) / overview.total_orders) * 100
  })

  const healthScore = computed(() => {
    let score = 100
    const pendingRatio =
      overview.total_orders > 0 ? overview.pending_review / overview.total_orders : 0
    const expiringRatio = overview.vps_count > 0 ? overview.expiring_soon / overview.vps_count : 0
    score -= Math.min(30, pendingRatio * 120)
    score -= Math.min(20, expiringRatio * 90)
    score -= Math.max(0, (toPercent(serverStatus.cpu_usage_percent) - 75) * 0.8)
    score -= Math.max(0, (toPercent(serverStatus.mem_usage_percent) - 80) * 0.8)
    score -= Math.max(0, (toPercent(serverStatus.disk_usage_percent) - 85) * 1.2)
    return Math.max(0, Math.round(score))
  })

  const healthComment = computed(() => {
    if (healthScore.value >= 85) return '运行稳定'
    if (healthScore.value >= 70) return '轻度压力'
    return '需要重点关注'
  })

  const healthColor = computed(() => {
    if (healthScore.value >= 85) return '#22c55e'
    if (healthScore.value >= 70) return '#f59e0b'
    return '#ef4444'
  })

  const alerts = computed<AlertItem[]>(() => {
    const items: AlertItem[] = []
    if (overview.pending_review >= 10) {
      items.push({
        level: 'high',
        text: `待审核订单 ${overview.pending_review} 单，处理存在积压风险。`
      })
    } else if (overview.pending_review > 0) {
      items.push({
        level: 'medium',
        text: `待审核订单 ${overview.pending_review} 单，建议尽快清理。`
      })
    }
    if (overview.expiring_soon >= 10) {
      items.push({
        level: 'high',
        text: `7 天内到期实例 ${overview.expiring_soon} 台，续费提醒压力较高。`
      })
    } else if (overview.expiring_soon > 0) {
      items.push({
        level: 'medium',
        text: `7 天内到期实例 ${overview.expiring_soon} 台，建议安排提醒。`
      })
    }
    if (toPercent(serverStatus.disk_usage_percent) >= 85) {
      items.push({
        level: 'high',
        text: `磁盘使用率 ${toPercent(serverStatus.disk_usage_percent)}%，容量接近阈值。`
      })
    }
    return items
  })

  const statusSummary = computed(() => {
    if (!rawVpsStatusChart.value.length) return '暂无实例状态数据'
    return `运行中 ${runningVpsCount.value} 台`
  })

  const pendingOrders = computed<PendingOrderRow[]>(() => {
    return orders.value
      .filter((item) => readStr(item, 'status', 'Status') === 'pending_review')
      .slice(0, 8)
      .map((item) => ({
        id: readNum(item, 'id', 'ID'),
        order_no: readStr(item, 'order_no', 'OrderNo'),
        user_id: readNum(item, 'user_id', 'UserID'),
        total_amount: readNum(item, 'total_amount', 'TotalAmount'),
        created_at: readStr(item, 'created_at', 'CreatedAt')
      }))
  })

  const expiringVps = computed<ExpiringVpsRow[]>(() => {
    return vpsList.value
      .map((item) => {
        const expireAt = readStr(item, 'expire_at', 'ExpireAt')
        const expireTime = expireAt ? new Date(expireAt).getTime() : 0
        const daysLeft = expireTime ? Math.ceil((expireTime - Date.now()) / 86400000) : 9999
        return {
          id: readNum(item, 'id', 'ID'),
          name: readStr(item, 'name', 'Name'),
          user_id: readNum(item, 'user_id', 'UserID'),
          status: readStr(item, 'status', 'Status'),
          expire_at: expireAt,
          days_left: daysLeft
        }
      })
      .filter((item) => item.expire_at && item.days_left <= 30)
      .sort((left, right) => left.days_left - right.days_left)
      .slice(0, 8)
  })

  const cpuText = computed(() => {
    const model = String(serverStatus.cpu_model || '').trim()
    const cores = Number(serverStatus.cpu_cores || 0)
    if (!model && !cores) return '-'
    return cores > 0 ? `${model || 'CPU'} (${cores} Core)` : model
  })

  const uptimeText = computed(() => {
    const seconds = Number(serverStatus.uptime_seconds || 0)
    if (!seconds) return '-'
    const days = Math.floor(seconds / 86400)
    const hours = Math.floor((seconds % 86400) / 3600)
    const minutes = Math.floor((seconds % 3600) / 60)
    if (days > 0) return `${days} 天 ${hours} 小时`
    if (hours > 0) return `${hours} 小时 ${minutes} 分钟`
    return `${minutes} 分钟`
  })

  watch(
    canView,
    (value) => {
      if (value && !initialized.value) {
        initialized.value = true
        reloadAll()
      }
    },
    { immediate: true }
  )

  function createEmptyOverview(): DashboardOverviewModel {
    return {
      total_orders: 0,
      pending_review: 0,
      revenue: 0,
      vps_count: 0,
      expiring_soon: 0
    }
  }

  function createEmptyServerStatus(): ServerStatus {
    return {
      hostname: '',
      os: '',
      platform: '',
      kernel_version: '',
      uptime_seconds: 0,
      cpu_model: '',
      cpu_cores: 0,
      cpu_usage_percent: 0,
      mem_total: 0,
      mem_used: 0,
      mem_usage_percent: 0,
      disk_total: 0,
      disk_used: 0,
      disk_usage_percent: 0,
      status: ''
    }
  }

  function resetServerStatus() {
    Object.assign(serverStatus, createEmptyServerStatus())
  }

  function readNum(source: unknown, ...keys: string[]) {
    const record = (source || {}) as Record<string, unknown>
    for (const key of keys) {
      const value = record[key]
      if (value !== undefined && value !== null && value !== '') {
        return Number(value) || 0
      }
    }
    return 0
  }

  function readStr(source: unknown, ...keys: string[]) {
    const record = (source || {}) as Record<string, unknown>
    for (const key of keys) {
      const value = record[key]
      if (value !== undefined && value !== null && String(value).trim() !== '') {
        return String(value)
      }
    }
    return ''
  }

  function toYuanNumber(cents?: number | null) {
    return Number((Number(cents || 0) / 100).toFixed(2))
  }

  function formatMoney(cents?: number | null) {
    return toYuanNumber(cents).toFixed(2)
  }

  function toPercent(value?: number | null) {
    const normalized = Number(value || 0)
    if (!Number.isFinite(normalized)) {
      return 0
    }
    const percent = normalized > 0 && normalized <= 1 ? normalized * 100 : normalized
    return Number(Math.max(0, Math.min(100, percent)).toFixed(2))
  }

  function formatDateTime(value?: string | null) {
    if (!value) {
      return '-'
    }
    const date = new Date(value)
    return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN', { hour12: false })
  }

  function orderStatusLabel(status?: string) {
    const normalized = String(status || '').toLowerCase()
    if (normalized === 'pending_payment') return '待支付'
    if (normalized === 'pending_review') return '待审核'
    if (normalized === 'approved' || normalized === 'confirmed' || normalized === 'completed') {
      return '已完成'
    }
    if (normalized === 'rejected') return '已拒绝'
    if (normalized === 'failed') return '失败'
    if (normalized === 'cancelled') return '已取消'
    if (!normalized || normalized === 'unknown') return '未知状态'
    return normalized
  }

  function vpsStatusLabel(status?: string) {
    const normalized = String(status || '').toLowerCase()
    if (normalized === 'running') return '运行中'
    if (normalized === 'stopped') return '已停机'
    if (normalized === 'locked') return '已锁定'
    if (normalized === 'provisioning') return '开通中'
    if (normalized === 'failed') return '失败'
    if (!normalized || normalized === 'unknown') return '未知状态'
    return normalized
  }

  function vpsStatusTagType(status?: string) {
    const normalized = String(status || '').toLowerCase()
    if (normalized === 'running') return 'success' as const
    if (normalized === 'provisioning') return 'primary' as const
    if (normalized === 'locked') return 'warning' as const
    if (normalized === 'failed') return 'danger' as const
    return 'info' as const
  }

  function formatError(error: unknown, fallback: string) {
    const message =
      (error as any)?.response?.data?.error ||
      (error as any)?.response?.data?.message ||
      (error as any)?.message
    return String(message || fallback)
  }

  async function fetchOverview() {
    const payload = (await fetchAdminDashboardOverview()) as DashboardOverview
    overview.total_orders = readNum(payload, 'total_orders', 'TotalOrders')
    overview.pending_review = readNum(payload, 'pending_review', 'PendingReview')
    overview.revenue = readNum(payload, 'revenue', 'Revenue')
    overview.vps_count = readNum(payload, 'vps_count', 'VPSCount')
    overview.expiring_soon = readNum(payload, 'expiring_soon', 'ExpiringSoon')
  }

  async function fetchRevenueSeries() {
    const payload = await fetchAdminDashboardRevenue({ period: period.value })
    revenueSeries.value = payload.items || payload.points || []
  }

  async function fetchVpsStatus() {
    const payload = await fetchAdminDashboardVpsStatus()
    vpsStatusSeries.value = payload.items || payload.points || []
  }

  async function fetchServerStatusData() {
    const payload = await fetchAdminServerStatus()
    Object.assign(serverStatus, createEmptyServerStatus(), payload || {})
  }

  async function fetchAllPages<T>(
    requester: (params: { limit: number; offset: number }) => Promise<ApiListLike<T>>
  ) {
    const pageSize = 500
    const maxRows = 10000
    const rows: T[] = []
    let offset = 0

    for (;;) {
      const payload = await requester({ limit: pageSize, offset })
      const items = payload.items || []
      const total = Number(payload.total || 0)
      rows.push(...items)
      if (items.length < pageSize) {
        break
      }
      offset += pageSize
      if ((total > 0 && rows.length >= total) || rows.length >= maxRows) {
        break
      }
    }

    return rows
  }

  async function loadRevenueBundle(warnings: string[]) {
    if (!canLoadRevenue.value) {
      revenueSeries.value = []
      warnings.push(REVENUE_WARNING)
      return
    }
    try {
      await fetchRevenueSeries()
    } catch (error) {
      revenueSeries.value = []
      warnings.push(REVENUE_WARNING)
      console.warn('[Dashboard] Failed to load revenue series', error)
    }
  }

  async function loadOrderBundle(warnings: string[]) {
    if (!canLoadOrders.value) {
      orders.value = []
      warnings.push(ORDER_WARNING)
      return
    }
    try {
      orders.value = await fetchAllPages<OrderRecord>((params) => fetchAdminOrders(params))
    } catch (error) {
      orders.value = []
      warnings.push(ORDER_WARNING)
      console.warn('[Dashboard] Failed to load order list', error)
    }
  }

  async function loadVpsBundle(warnings: string[]) {
    if (!canLoadVps.value) {
      vpsList.value = []
      warnings.push(VPS_WARNING)
      return
    }
    try {
      vpsList.value = await fetchAllPages<VpsRecord>((params) => fetchAdminVps(params))
    } catch (error) {
      vpsList.value = []
      warnings.push(VPS_WARNING)
      console.warn('[Dashboard] Failed to load VPS list', error)
    }
  }

  async function loadVpsStatusBundle() {
    if (!canLoadVpsStatus.value) {
      vpsStatusSeries.value = []
      return
    }
    try {
      await fetchVpsStatus()
    } catch (error) {
      vpsStatusSeries.value = []
      console.warn('[Dashboard] Failed to load VPS status overview', error)
    }
  }

  async function loadServerBundle(warnings: string[]) {
    if (!canLoadServer.value) {
      resetServerStatus()
      warnings.push(SERVER_WARNING)
      return
    }
    try {
      await fetchServerStatusData()
    } catch (error) {
      resetServerStatus()
      warnings.push(SERVER_WARNING)
      console.warn('[Dashboard] Failed to load server status', error)
    }
  }

  async function handlePeriodChange() {
    if (!canView.value) {
      return
    }
    await reloadAll()
  }

  async function reloadAll() {
    if (!canView.value) {
      return
    }

    loading.value = true
    const warnings: string[] = []
    try {
      await Promise.all([
        fetchOverview(),
        loadRevenueBundle(warnings),
        loadOrderBundle(warnings),
        loadVpsBundle(warnings),
        loadVpsStatusBundle(),
        loadServerBundle(warnings)
      ])
      loadWarnings.value = Array.from(new Set(warnings))
    } catch (error) {
      loadWarnings.value = Array.from(new Set(warnings))
      ElMessage.error(formatError(error, '加载运营总览失败'))
    } finally {
      loading.value = false
    }
  }
</script>

<style scoped lang="scss">
  .dashboard-page {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .hero-card {
    border-radius: 18px;
    background:
      radial-gradient(circle at top left, rgb(59 130 246 / 16%), transparent 32%),
      radial-gradient(circle at top right, rgb(34 197 94 / 12%), transparent 24%),
      linear-gradient(135deg, rgb(255 255 255 / 98%), rgb(248 250 252 / 96%));
  }

  .hero-main {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 16px;
  }

  .hero-main h2 {
    margin: 0;
    color: var(--el-text-color-primary);
    font-size: 30px;
    font-weight: 700;
  }

  .hero-main p {
    max-width: 760px;
    margin: 8px 0 16px;
    color: var(--el-text-color-secondary);
    font-size: 14px;
    line-height: 1.7;
  }

  .warning-list {
    display: flex;
    flex-direction: column;
    gap: 6px;
    font-size: 13px;
    line-height: 1.6;
  }

  .section-row {
    margin: 0 !important;
  }

  .metric-card {
    border-radius: 16px;
  }

  .metric-card :deep(.el-card__body) {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .metric-title {
    color: var(--el-text-color-secondary);
    font-size: 13px;
    font-weight: 600;
  }

  .metric-foot {
    color: var(--el-text-color-secondary);
    font-size: 13px;
  }

  .success {
    color: var(--el-color-success);
  }

  .danger {
    color: var(--el-color-danger);
  }

  .warning {
    color: var(--el-color-warning);
  }

  .alert-list,
  .status-list,
  .health-lines {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .alert-item,
  .status-item {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 12px 14px;
    border: 1px solid var(--el-border-color-light);
    border-radius: 14px;
    background: var(--el-fill-color-light);
  }

  .alert-item {
    color: var(--el-text-color-primary);
    font-size: 14px;
  }

  .health-card {
    display: flex;
    flex-direction: column;
    gap: 20px;
    align-items: center;
  }

  .health-lines {
    width: 100%;
  }

  .health-line-label,
  .status-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    margin-bottom: 8px;
    color: var(--el-text-color-secondary);
    font-size: 13px;
  }

  .health-line-label strong {
    color: var(--el-text-color-primary);
  }

  .amount-up {
    color: var(--el-color-primary);
    font-weight: 700;
  }

  .subtle {
    color: var(--el-text-color-secondary);
  }

  @media (max-width: 1200px) {
    .hero-main {
      flex-direction: column;
    }
  }

  @media (max-width: 768px) {
    .hero-main h2 {
      font-size: 24px;
    }

    .health-line-label,
    .status-head,
    .alert-item {
      align-items: flex-start;
      flex-direction: column;
    }
  }
</style>
