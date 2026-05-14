<template>
  <div class="revenue-page art-full-height">
    <div class="page-hero">
      <div>
        <h2>收入统计</h2>
        <p>经营视角下钻、趋势洞察、用户贡献与明细审计</p>
      </div>

      <ElSpace wrap>
        <ElTag type="success" effect="light">口径：订单确认收入</ElTag>
        <ElButton :loading="exporting" :disabled="!canView" @click="handleExportAudit">
          导出审计 CSV
        </ElButton>
        <ElButton :loading="loading || userRankLoading" :disabled="!canView" @click="reloadAll">
          刷新数据
        </ElButton>
      </ElSpace>
    </div>

    <ElAlert
      v-if="!canView"
      type="warning"
      :closable="false"
      title="当前账号缺少收入统计权限"
      description="请为当前管理员分配 dashboard.revenue 或 dashboard.revenue_analytics_overview 权限。"
    />

    <template v-else>
      <ElCard class="filter-card" shadow="never">
        <div class="filter-grid">
          <ElSelect
            v-model="query.goods_type_id"
            clearable
            filterable
            placeholder="选择类型"
            @change="handleGoodsTypeChange"
          >
            <ElOption
              v-for="option in goodsTypeOptions"
              :key="option.value"
              :label="option.label"
              :value="option.value"
            />
          </ElSelect>

          <ElSelect
            v-model="query.region_id"
            clearable
            filterable
            placeholder="选择地区"
            :disabled="!query.goods_type_id"
            @change="handleRegionChange"
          >
            <ElOption
              v-for="option in regionOptions"
              :key="option.value"
              :label="option.label"
              :value="option.value"
            />
          </ElSelect>

          <ElSelect
            v-model="query.line_id"
            clearable
            filterable
            placeholder="选择线路"
            :disabled="!query.region_id"
            @change="handleLineChange"
          >
            <ElOption
              v-for="option in lineOptions"
              :key="option.value"
              :label="option.label"
              :value="option.value"
            />
          </ElSelect>

          <ElSelect
            v-model="query.package_id"
            clearable
            filterable
            placeholder="选择套餐"
            :disabled="!query.line_id"
            @change="handlePackageChange"
          >
            <ElOption
              v-for="option in packageOptions"
              :key="option.value"
              :label="option.label"
              :value="option.value"
            />
          </ElSelect>

          <ElInputNumber
            v-model="query.user_id"
            :min="1"
            :precision="0"
            :controls="false"
            placeholder="用户 ID"
            @change="handleUserFilterChange"
          />

          <ElDatePicker
            v-model="rangeValue"
            type="daterange"
            range-separator="至"
            start-placeholder="开始日期"
            end-placeholder="结束日期"
            unlink-panels
            @change="handleRangeChange"
          />

          <div class="filter-actions">
            <ElButton type="primary" :loading="loading" @click="reloadAll">查询</ElButton>
            <ElButton @click="handleResetFilters">重置</ElButton>
          </div>
        </div>

        <ElDivider />

        <div class="filter-footer">
          <div class="quick-range">
            <span class="subtle">快捷时间</span>
            <ElRadioGroup v-model="quickRangeKey" @change="handleQuickRangeChange">
              <ElRadioButton label="today">今天</ElRadioButton>
              <ElRadioButton label="7d">近 7 天</ElRadioButton>
              <ElRadioButton label="30d">近 30 天</ElRadioButton>
              <ElRadioButton label="month">本月</ElRadioButton>
            </ElRadioGroup>
          </div>

          <div v-if="hasActiveFilters" class="active-filters">
            <span class="subtle">当前筛选</span>
            <ElTag v-if="query.goods_type_id" type="primary">
              类型：{{ lookupLabel(goodsTypeOptions, query.goods_type_id) }}
            </ElTag>
            <ElTag v-if="query.region_id" type="success">
              地区：{{ lookupLabel(regionOptions, query.region_id) }}
            </ElTag>
            <ElTag v-if="query.line_id" type="warning">
              线路：{{ lookupLabel(lineOptions, query.line_id) }}
            </ElTag>
            <ElTag v-if="query.package_id" type="danger">
              套餐：{{ lookupLabel(packageOptions, query.package_id) }}
            </ElTag>
            <ElTag v-if="query.user_id" type="info">用户：{{ query.user_id }}</ElTag>
          </div>
        </div>
      </ElCard>

      <ElRow :gutter="16" class="card-row">
        <ElCol :xs="24" :sm="12" :lg="6">
          <ElCard class="metric-card metric-card-revenue" shadow="hover" v-loading="loading">
            <div class="metric-label">总收入</div>
            <div class="metric-value"
              >¥{{ formatCents(overview.summary?.total_revenue_cents) }}</div
            >
            <div class="metric-footer">
              <span>订单数：{{ overview.summary?.order_count || 0 }}</span>
              <ElTag :type="periodCompare.tagType" effect="light">{{ periodCompare.tag }}</ElTag>
            </div>
          </ElCard>
        </ElCol>

        <ElCol :xs="24" :sm="12" :lg="6">
          <ElCard class="metric-card metric-card-mom" shadow="hover" v-loading="loading">
            <div class="metric-label">环比变化</div>
            <div class="metric-value">{{ periodCompare.percentText || '0.00%' }}</div>
            <div class="metric-trend" :class="periodCompare.className">
              {{ periodCompare.text }}
            </div>
          </ElCard>
        </ElCol>

        <ElCol :xs="24" :sm="12" :lg="6">
          <ElCard class="metric-card metric-card-yoy" shadow="hover" v-loading="loading">
            <div class="metric-label">同比 / 环比</div>
            <div class="metric-value">
              {{ formatRatio(overview.summary?.yoy_ratio, overview.summary?.yoy_comparable) }}
            </div>
            <div class="dual-compare">
              <span
                :class="ratioClass(overview.summary?.yoy_ratio, overview.summary?.yoy_comparable)"
              >
                同比
              </span>
              <span
                :class="ratioClass(overview.summary?.mom_ratio, overview.summary?.mom_comparable)"
              >
                环比
              </span>
            </div>
          </ElCard>
        </ElCol>

        <ElCol :xs="24" :sm="12" :lg="6">
          <ElCard class="metric-card metric-card-net" shadow="hover" v-loading="loading">
            <div class="metric-label">当前页净额</div>
            <div class="metric-value">¥{{ formatCents(detailInsight.net_cents) }}</div>
            <div class="metric-trend" :class="netTrend.className">{{ netTrend.text }}</div>
            <div class="metric-subtle">
              退款订单：{{ detailInsight.refund_count }}，活跃用户：{{ detailInsight.user_count }}
            </div>
          </ElCard>
        </ElCol>
      </ElRow>

      <ElRow :gutter="16" class="card-row">
        <ElCol :xs="24" :lg="12">
          <ElCard class="chart-card" shadow="hover" v-loading="loading">
            <template #header>
              <div class="section-header">
                <div>
                  <div class="section-title">收入占比</div>
                  <div class="section-subtitle">点击下方维度继续下钻</div>
                </div>
              </div>
            </template>

            <ArtRingChart
              height="320px"
              :data="shareChartData"
              :center-text="shareCenterText"
              :show-label="false"
              :show-legend="false"
            />

            <div class="share-list">
              <div v-if="shareLeaders.length" class="share-items">
                <button
                  v-for="item in shareLeaders"
                  :key="`${item.dimension_id}-${item.dimension_name}`"
                  class="share-item"
                  type="button"
                  @click="handleDrillDown(item)"
                >
                  <div class="share-item-top">
                    <span class="share-name">{{ item.dimension_name || '-' }}</span>
                    <span class="share-value">
                      ¥{{ formatCents(item.revenue_cents) }} / {{ formatPercent(item.ratio) }}
                    </span>
                  </div>
                  <ElProgress
                    :percentage="Math.min(100, Number((Number(item.ratio || 0) * 100).toFixed(2)))"
                    :show-text="false"
                    :stroke-width="8"
                  />
                </button>
              </div>

              <ElEmpty v-else description="暂无占比数据" :image-size="72" />
            </div>
          </ElCard>
        </ElCol>

        <ElCol :xs="24" :lg="12">
          <ElCard class="chart-card" shadow="hover" v-loading="loading">
            <template #header>
              <div class="section-header">
                <div>
                  <div class="section-title">收入趋势</div>
                  <div class="section-subtitle">按时间桶观察收入变化</div>
                </div>
              </div>
            </template>

            <ArtLineChart
              height="320px"
              :data="trendChartData.values"
              :x-axis-data="trendChartData.labels"
              :show-area-color="true"
              :show-axis-line="false"
            />

            <div class="trend-insight-grid">
              <div class="insight-card">
                <span class="insight-label">峰值日</span>
                <strong>{{ trendInsight.peakLabel }}</strong>
                <span class="insight-value">¥{{ trendInsight.peakAmount }}</span>
              </div>
              <div class="insight-card">
                <span class="insight-label">低谷日</span>
                <strong>{{ trendInsight.lowLabel }}</strong>
                <span class="insight-value">¥{{ trendInsight.lowAmount }}</span>
              </div>
              <div class="insight-card">
                <span class="insight-label">最新日</span>
                <strong>{{ trendInsight.latestLabel }}</strong>
                <span class="insight-value">¥{{ trendInsight.latestAmount }}</span>
              </div>
            </div>
          </ElCard>
        </ElCol>
      </ElRow>

      <ElRow :gutter="16" class="card-row">
        <ElCol :xs="24" :lg="8">
          <ElCard class="panel-card" shadow="hover" v-loading="loading || userRankLoading">
            <ElTabs v-model="leftPanelTab">
              <ElTabPane label="维度 Top 5" name="dimension">
                <div v-if="topItems.length" class="ranking-list">
                  <button
                    v-for="item in topItems"
                    :key="`${item.rank}-${item.dimension_id}-${item.dimension_name}`"
                    class="ranking-item"
                    type="button"
                    @click="handleDrillDown(item)"
                  >
                    <div class="ranking-left">
                      <ElTag :type="item.rank && item.rank <= 3 ? 'warning' : 'info'" size="small">
                        #{{ item.rank || '-' }}
                      </ElTag>
                      <span class="ranking-name">{{ item.dimension_name || '-' }}</span>
                    </div>
                    <div class="ranking-right">
                      <strong>¥{{ formatCents(item.revenue_cents) }}</strong>
                      <span>{{ formatPercent(item.ratio) }}</span>
                    </div>
                  </button>
                </div>

                <ElEmpty v-else description="暂无维度排行" :image-size="72" />
              </ElTabPane>

              <ElTabPane label="用户消费榜" name="user">
                <div v-if="userRankList.length" class="ranking-list">
                  <button
                    v-for="item in userRankList"
                    :key="`${item.rank}-${item.user_id}`"
                    class="ranking-item"
                    type="button"
                    @click="openUserFinance(item.user_id)"
                  >
                    <div class="ranking-left">
                      <ElTag :type="item.rank <= 3 ? 'warning' : 'info'" size="small">
                        #{{ item.rank }}
                      </ElTag>
                      <ElAvatar :size="24" :src="userAvatar(item.user_id)">
                        {{ String(item.user_id).slice(-2) }}
                      </ElAvatar>
                      <span class="ranking-name">{{ formatUserLabel(item.user_id) }}</span>
                    </div>
                    <div class="ranking-right">
                      <strong>¥{{ formatCents(item.revenue_cents) }}</strong>
                      <span>{{ item.order_count }} 单</span>
                    </div>
                  </button>
                </div>

                <ElEmpty v-else description="当前筛选下无用户消费数据" :image-size="72" />
              </ElTabPane>
            </ElTabs>
          </ElCard>
        </ElCol>

        <ElCol :xs="24" :lg="16">
          <ElCard class="detail-card" shadow="hover" v-loading="loading">
            <template #header>
              <div class="section-header">
                <div>
                  <div class="section-title">明细表</div>
                  <div class="section-subtitle">支付流水与订单明细审计</div>
                </div>
                <ElButton size="small" @click="fetchDetails">刷新当前页</ElButton>
              </div>
            </template>

            <ElTable :data="details" border stripe row-key="payment_id">
              <ElTableColumn prop="payment_id" label="支付 ID" width="110" />
              <ElTableColumn prop="order_no" label="订单号" min-width="210" show-overflow-tooltip />
              <ElTableColumn label="用户" min-width="220">
                <template #default="{ row }">
                  <div class="user-cell">
                    <ElAvatar :size="24" :src="userAvatar(row.user_id)">
                      {{ String(row.user_id || '').slice(-2) }}
                    </ElAvatar>
                    <ElButton link type="primary" @click="openUserFinance(row.user_id)">
                      {{ formatUserLabel(row.user_id) }}
                    </ElButton>
                  </div>
                </template>
              </ElTableColumn>
              <ElTableColumn label="金额" width="140">
                <template #default="{ row }">
                  <span :class="amountClass(Number(row.amount_cents || 0))">
                    {{ amountPrefix(Number(row.amount_cents || 0)) }}¥{{
                      formatCents(Math.abs(Number(row.amount_cents || 0)))
                    }}
                  </span>
                </template>
              </ElTableColumn>
              <ElTableColumn label="状态" width="120">
                <template #default="{ row }">
                  <ElTag :type="statusMeta(row.status).type" effect="light">
                    {{ statusMeta(row.status).text }}
                  </ElTag>
                </template>
              </ElTableColumn>
              <ElTableColumn label="支付时间" min-width="180">
                <template #default="{ row }">
                  {{ formatDateTime(row.paid_at) }}
                </template>
              </ElTableColumn>
            </ElTable>

            <div class="table-pagination">
              <ElPagination
                layout="total, sizes, prev, pager, next, jumper"
                :current-page="Number(query.page || 1)"
                :page-size="Number(query.page_size || 20)"
                :page-sizes="[10, 20, 50, 100]"
                :total="detailTotal"
                @current-change="handlePageChange"
                @size-change="handlePageSizeChange"
              />
            </div>
          </ElCard>
        </ElCol>
      </ElRow>

      <UserFinanceDrawer
        v-model:visible="userFinanceVisible"
        :loading="userFinanceLoading"
        :user-id="selectedUserId"
        :user-label="selectedUserLabel"
        :user-avatar="selectedUserAvatar"
        :profile="userFinanceProfile"
        :summary="userFinanceSummary"
        :rows="userFinanceRows"
        @apply-user-filter="applyUserFilterFromDrawer"
      />
    </template>
  </div>
</template>

<script setup lang="ts">
  import { computed, onMounted, reactive, ref, watch } from 'vue'
  import { storeToRefs } from 'pinia'
  import type {
    CatalogGoodsType,
    CatalogPackage,
    CatalogPlanGroup,
    CatalogRegion,
    RevenueAnalyticsDetailRecord,
    RevenueAnalyticsLevel,
    RevenueAnalyticsOverviewResponse,
    RevenueAnalyticsQuery,
    RevenueAnalyticsShareItem,
    RevenueAnalyticsTopItem,
    RevenueAnalyticsTrendPoint,
    UserRecord
  } from '@/api/admin'
  import {
    exportAdminRevenueAnalyticsAudit,
    fetchAdminGoodsTypes,
    fetchAdminPackages,
    fetchAdminPlanGroups,
    fetchAdminRegions,
    fetchAdminRevenueAnalyticsDetails,
    fetchAdminRevenueAnalyticsOverview,
    fetchAdminRevenueAnalyticsTop,
    fetchAdminRevenueAnalyticsTrend,
    fetchAdminUserDetail
  } from '@/api/admin'
  import ArtLineChart from '@/components/core/charts/art-line-chart/index.vue'
  import ArtRingChart from '@/components/core/charts/art-ring-chart/index.vue'
  import { useUserStore } from '@/store/modules/user'
  import UserFinanceDrawer from './modules/user-finance-drawer.vue'

  defineOptions({ name: 'RevenueAnalyticsPage' })

  interface OptionItem {
    label: string
    value: number
  }

  interface UserMetaInfo {
    username?: string
    avatar?: string
    email?: string
  }

  interface UserRankItem {
    rank: number
    user_id: number
    revenue_cents: number
    order_count: number
  }

  interface UserFinanceSummary {
    total_revenue_cents: number
    order_count: number
    positive_order_count: number
    negative_order_count: number
    avg_order_cents: number
    last_paid_at: string
  }

  interface DrillTarget {
    dimension_id?: number
    dimension_name?: string
  }

  const loading = ref(false)
  const userRankLoading = ref(false)
  const exporting = ref(false)
  const leftPanelTab = ref<'dimension' | 'user'>('dimension')

  const overview = ref<RevenueAnalyticsOverviewResponse>({})
  const trend = ref<RevenueAnalyticsTrendPoint[]>([])
  const topItems = ref<RevenueAnalyticsTopItem[]>([])
  const details = ref<RevenueAnalyticsDetailRecord[]>([])
  const detailTotal = ref(0)

  const goodsTypes = ref<CatalogGoodsType[]>([])
  const regions = ref<CatalogRegion[]>([])
  const planGroups = ref<CatalogPlanGroup[]>([])
  const packages = ref<CatalogPackage[]>([])
  const userMetaMap = ref<Record<number, UserMetaInfo>>({})
  const userRankList = ref<UserRankItem[]>([])

  const userFinanceVisible = ref(false)
  const userFinanceLoading = ref(false)
  const selectedUserId = ref<number>()
  const userFinanceProfile = ref<UserRecord | null>(null)
  const userFinanceRows = ref<RevenueAnalyticsDetailRecord[]>([])
  const userFinanceSummary = ref<UserFinanceSummary>(createEmptyUserFinanceSummary())

  const userStore = useUserStore()
  const { info } = storeToRefs(userStore)

  const query = reactive<RevenueAnalyticsQuery>(createDefaultQuery())
  const rangeValue = ref<[Date, Date]>(defaultRange())
  const quickRangeKey = ref('30d')

  const goodsTypeOptions = computed<OptionItem[]>(() =>
    goodsTypes.value.map((item) => ({
      label: String(item.name || item.code || `类型-${item.id || ''}`),
      value: Number(item.id || 0)
    }))
  )

  const regionOptions = computed<OptionItem[]>(() =>
    regions.value.map((item) => ({
      label: String(item.name || item.code || `地区-${item.id || ''}`),
      value: Number(item.id || 0)
    }))
  )

  const lineOptions = computed<OptionItem[]>(() => {
    const unique = new Map<number, OptionItem>()

    for (const item of planGroups.value) {
      const lineId = Number(item.line_id || 0)
      if (!lineId || unique.has(lineId)) {
        continue
      }

      unique.set(lineId, {
        label: `${item.name || '线路'} (${lineId})`,
        value: lineId
      })
    }

    return Array.from(unique.values())
  })

  const packageOptions = computed<OptionItem[]>(() => {
    const lineId = Number(query.line_id || 0)
    if (!lineId) {
      return []
    }

    const allowedPlanGroupIds = new Set(
      planGroups.value
        .filter((item) => Number(item.line_id || 0) === lineId)
        .map((item) => Number(item.id || 0))
        .filter(Boolean)
    )

    return packages.value
      .filter((item) => allowedPlanGroupIds.has(Number(item.plan_group_id || 0)))
      .map((item) => ({
        label: String(item.name || `套餐-${item.id || ''}`),
        value: Number(item.id || 0)
      }))
  })

  const canView = computed(() => {
    const perms = Array.isArray(info.value?.buttons) ? info.value.buttons : []
    const roles = Array.isArray((info.value as any)?.roles) ? (info.value as any).roles : []

    if (roles.includes('R_SUPER')) {
      return true
    }

    return (
      hasPermission(perms, '*') ||
      hasPermission(perms, 'dashboard.revenue') ||
      hasPermission(perms, 'dashboard.revenue_analytics_overview')
    )
  })

  const hasActiveFilters = computed(() => {
    return Boolean(
      query.goods_type_id || query.region_id || query.line_id || query.package_id || query.user_id
    )
  })

  const detailInsight = computed(() => {
    let net = 0
    let refundCount = 0
    const users = new Set<number>()

    for (const row of details.value) {
      const amount = Number(row.amount_cents || 0)
      net += amount

      if (amount < 0) {
        refundCount += 1
      }

      const userId = Number(row.user_id || 0)
      if (userId > 0) {
        users.add(userId)
      }
    }

    return {
      net_cents: net,
      refund_count: refundCount,
      user_count: users.size
    }
  })

  const shareLeaders = computed(() => (overview.value.share_items || []).slice(0, 3))

  const shareChartData = computed(() =>
    (overview.value.share_items || []).map((item) => ({
      name: String(item.dimension_name || '-'),
      value: Number(item.revenue_cents || 0)
    }))
  )

  const shareCenterText = computed(
    () => `总收入\n¥${formatCents(overview.value.summary?.total_revenue_cents)}`
  )

  const trendChartData = computed(() => ({
    labels: trend.value.map((item) => String(item.bucket || '-')),
    values: trend.value.map((item) => Number(item.revenue_cents || 0) / 100)
  }))

  const trendInsight = computed(() => {
    const items = trend.value.map((item) => ({
      label: String(item.bucket || '-'),
      value: Number(item.revenue_cents || 0) / 100
    }))

    if (!items.length) {
      return {
        peakLabel: '-',
        peakAmount: '0.00',
        lowLabel: '-',
        lowAmount: '0.00',
        latestLabel: '-',
        latestAmount: '0.00'
      }
    }

    let peak = items[0]
    let low = items[0]

    for (const item of items) {
      if (item.value > peak.value) {
        peak = item
      }

      if (item.value < low.value) {
        low = item
      }
    }

    const latest = items[items.length - 1]

    return {
      peakLabel: peak.label,
      peakAmount: peak.value.toFixed(2),
      lowLabel: low.label,
      lowAmount: low.value.toFixed(2),
      latestLabel: latest.label,
      latestAmount: latest.value.toFixed(2)
    }
  })

  const periodCompare = computed(() => {
    const current = Number(overview.value.summary?.total_revenue_cents || 0)
    const ratio = overview.value.summary?.mom_ratio
    const comparable = Boolean(overview.value.summary?.mom_comparable) && ratio != null

    if (!comparable || ratio == null || ratio <= -1) {
      return {
        tag: '不可比',
        tagType: 'info' as const,
        className: 'compare-neutral',
        text: '暂无可比数据',
        percentText: ''
      }
    }

    const previous = current / (1 + ratio)
    const delta = current - previous
    const isUp = delta >= 0

    return {
      tag: isUp ? '增长' : '下降',
      tagType: isUp ? ('success' as const) : ('danger' as const),
      className: isUp ? 'compare-up' : 'compare-down',
      text: `${isUp ? '增长' : '下降'} ¥${(Math.abs(delta) / 100).toFixed(2)}`,
      percentText: `${isUp ? '+' : '-'}${Math.abs(ratio * 100).toFixed(2)}%`
    }
  })

  const netTrend = computed(() => {
    const net = Number(detailInsight.value.net_cents || 0)

    if (net > 0) {
      return { className: 'compare-up', text: '净流入' }
    }

    if (net < 0) {
      return { className: 'compare-down', text: '净流出' }
    }

    return { className: 'compare-neutral', text: '收支持平' }
  })

  const selectedUserLabel = computed(() => formatUserLabel(selectedUserId.value))
  const selectedUserAvatar = computed(() => userAvatar(selectedUserId.value))

  onMounted(async () => {
    await loadGoodsTypes()
    syncInferredLevel()

    if (canView.value) {
      await reloadAll()
    }
  })

  watch(
    () => details.value,
    async (rows) => {
      await ensureUserMeta(rows.map((item) => Number(item.user_id || 0)))
    },
    { immediate: true }
  )

  function cloneDate(date: Date) {
    return new Date(date.getTime())
  }

  function toDate(value?: string | Date | null) {
    if (!value) {
      return null
    }

    const date = value instanceof Date ? new Date(value.getTime()) : new Date(value)
    return Number.isNaN(date.getTime()) ? null : date
  }

  function startOfDay(date: Date) {
    const next = cloneDate(date)
    next.setHours(0, 0, 0, 0)
    return next
  }

  function endOfDay(date: Date) {
    const next = cloneDate(date)
    next.setHours(23, 59, 59, 999)
    return next
  }

  function startOfMonth(date: Date) {
    const next = startOfDay(date)
    next.setDate(1)
    return next
  }

  function subtractDays(date: Date, days: number) {
    const next = cloneDate(date)
    next.setDate(next.getDate() - days)
    return next
  }

  function subtractMonths(date: Date, months: number) {
    const next = cloneDate(date)
    next.setMonth(next.getMonth() - months)
    return next
  }

  function formatDateForFilename(date = new Date()) {
    const year = date.getFullYear()
    const month = String(date.getMonth() + 1).padStart(2, '0')
    const day = String(date.getDate()).padStart(2, '0')
    const hour = String(date.getHours()).padStart(2, '0')
    const minute = String(date.getMinutes()).padStart(2, '0')
    const second = String(date.getSeconds()).padStart(2, '0')

    return `${year}${month}${day}_${hour}${minute}${second}`
  }

  function defaultRange(): [Date, Date] {
    const now = new Date()
    return [startOfDay(subtractMonths(now, 1)), endOfDay(now)]
  }

  function createDefaultQuery(): RevenueAnalyticsQuery {
    const [from, to] = defaultRange()

    return {
      from_at: from.toISOString(),
      to_at: to.toISOString(),
      level: 'overall',
      page: 1,
      page_size: 20,
      sort_field: 'paid_at',
      sort_order: 'desc'
    }
  }

  function createEmptyUserFinanceSummary(): UserFinanceSummary {
    return {
      total_revenue_cents: 0,
      order_count: 0,
      positive_order_count: 0,
      negative_order_count: 0,
      avg_order_cents: 0,
      last_paid_at: ''
    }
  }

  function hasPermission(perms: string[], required: string) {
    const auth = String(required || '').trim()
    if (!auth) {
      return true
    }

    for (const permission of perms) {
      if (permission === '*' || permission === auth) {
        return true
      }

      if (typeof permission === 'string' && permission.endsWith('*')) {
        const prefix = permission.slice(0, -1)
        if (auth.startsWith(prefix)) {
          return true
        }
      }
    }

    return false
  }

  function inferLevelFromSelection(): RevenueAnalyticsLevel {
    if (!query.goods_type_id) {
      return 'overall'
    }

    if (query.package_id) {
      return 'package'
    }

    if (query.line_id) {
      return 'line'
    }

    if (query.region_id) {
      return 'region'
    }

    return 'goods_type'
  }

  function syncInferredLevel() {
    query.level = inferLevelFromSelection()
  }

  function createQueryPayload(extra: Partial<RevenueAnalyticsQuery> = {}): RevenueAnalyticsQuery {
    return {
      ...query,
      ...extra,
      level: inferLevelFromSelection()
    }
  }

  function normalizeNumber(value: unknown) {
    const parsed = Number(value || 0)
    return Number.isFinite(parsed) && parsed > 0 ? parsed : undefined
  }

  function formatCents(value?: number | null) {
    return (Number(value || 0) / 100).toFixed(2)
  }

  function formatPercent(value?: number | null) {
    return `${(Number(value || 0) * 100).toFixed(2)}%`
  }

  function formatDateTime(value?: string | null) {
    if (!value) {
      return '-'
    }

    const date = toDate(value)
    if (!date) {
      return String(value)
    }

    return new Intl.DateTimeFormat('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: false
    }).format(date)
  }

  function formatRatio(ratio?: number | null, comparable?: boolean) {
    if (!comparable || ratio == null) {
      return '不可比'
    }

    return `${(ratio * 100).toFixed(2)}%`
  }

  function ratioClass(ratio?: number | null, comparable?: boolean) {
    if (!comparable || ratio == null) {
      return 'compare-neutral'
    }

    if (ratio > 0) {
      return 'compare-up'
    }

    if (ratio < 0) {
      return 'compare-down'
    }

    return 'compare-neutral'
  }

  function amountClass(value: number) {
    if (value > 0) {
      return 'amount-up'
    }

    if (value < 0) {
      return 'amount-down'
    }

    return 'amount-neutral'
  }

  function amountPrefix(value: number) {
    if (value > 0) {
      return '+'
    }

    if (value < 0) {
      return '-'
    }

    return ''
  }

  function statusMeta(status?: string | null) {
    const value = String(status || '').toLowerCase()

    if (value === 'approved' || value === 'active') {
      return { type: 'success' as const, text: '已完成' }
    }

    if (value === 'pending_payment') {
      return { type: 'info' as const, text: '待支付' }
    }

    if (value === 'pending_review') {
      return { type: 'warning' as const, text: '待审核' }
    }

    if (value === 'failed' || value === 'rejected') {
      return { type: 'danger' as const, text: '失败' }
    }

    if (value === 'canceled') {
      return { type: 'info' as const, text: '已取消' }
    }

    return { type: 'info' as const, text: value || '-' }
  }

  function lookupLabel(options: OptionItem[], value?: number) {
    const id = Number(value || 0)
    if (!id) {
      return '-'
    }

    return options.find((item) => Number(item.value) === id)?.label || String(id)
  }

  function detectQuickRangeKey(from: Date, to: Date) {
    const now = new Date()
    const todayStart = startOfDay(now)
    const todayEnd = endOfDay(now)
    const sevenDayStart = startOfDay(subtractDays(now, 6))
    const thirtyDayStart = startOfDay(subtractMonths(now, 1))
    const monthStart = startOfMonth(now)

    if (from.getTime() === todayStart.getTime() && to.getTime() === todayEnd.getTime()) {
      return 'today'
    }

    if (from.getTime() === sevenDayStart.getTime() && to.getTime() === todayEnd.getTime()) {
      return '7d'
    }

    if (from.getTime() === thirtyDayStart.getTime() && to.getTime() === todayEnd.getTime()) {
      return '30d'
    }

    if (from.getTime() === monthStart.getTime() && to.getTime() === todayEnd.getTime()) {
      return 'month'
    }

    return '30d'
  }

  function ensureQueryValid(silent = false) {
    const level = inferLevelFromSelection()

    if (level === 'overall') {
      return true
    }

    if (level === 'region' && !query.region_id) {
      if (!silent) {
        ElMessage.warning('当前层级为地区，请先选择地区')
      }
      return false
    }

    if (level === 'line' && (!query.region_id || !query.line_id)) {
      if (!silent) {
        ElMessage.warning('当前层级为线路，请先选择地区和线路')
      }
      return false
    }

    if (level === 'package' && (!query.region_id || !query.line_id || !query.package_id)) {
      if (!silent) {
        ElMessage.warning('当前层级为套餐，请先选择地区、线路和套餐')
      }
      return false
    }

    return true
  }

  function setUserMeta(
    userId: number,
    payload?: Partial<UserRecord> | Record<string, unknown> | null
  ) {
    if (!userId) {
      return
    }

    const existing = userMetaMap.value[userId] || {}

    userMetaMap.value[userId] = {
      username: String(
        (payload as any)?.username || (payload as any)?.Username || existing.username || ''
      ),
      avatar: String(
        (payload as any)?.avatar ||
          (payload as any)?.avatar_url ||
          (payload as any)?.AvatarURL ||
          existing.avatar ||
          ''
      ),
      email: String((payload as any)?.email || (payload as any)?.Email || existing.email || '')
    }
  }

  async function ensureUserMeta(userIds: number[]) {
    const uniqueIds = Array.from(new Set(userIds.filter((item) => item > 0)))
    const missing = uniqueIds.filter(
      (item) => !userMetaMap.value[item]?.username && !userMetaMap.value[item]?.avatar
    )

    if (!missing.length) {
      return
    }

    await Promise.allSettled(
      missing.map(async (userId) => {
        const payload = await fetchAdminUserDetail(userId)
        setUserMeta(userId, payload)
      })
    )
  }

  function formatUserLabel(userId?: number | null) {
    const id = Number(userId || 0)
    if (!id) {
      return '未知用户'
    }

    return userMetaMap.value[id]?.username || `用户 #${id}`
  }

  function userAvatar(userId?: number | null) {
    const id = Number(userId || 0)
    if (!id) {
      return ''
    }

    return userMetaMap.value[id]?.avatar || ''
  }

  async function loadGoodsTypes() {
    const payload = await fetchAdminGoodsTypes()
    goodsTypes.value = payload.items || []
  }

  async function loadRegions() {
    if (!query.goods_type_id) {
      regions.value = []
      return
    }

    const payload = await fetchAdminRegions({
      goods_type_id: query.goods_type_id
    })
    regions.value = payload.items || []
  }

  async function loadPlanGroups() {
    if (!query.goods_type_id || !query.region_id) {
      planGroups.value = []
      return
    }

    const payload = await fetchAdminPlanGroups({
      goods_type_id: query.goods_type_id,
      region_id: query.region_id
    })
    planGroups.value = payload.items || []
  }

  async function loadPackages() {
    if (!query.goods_type_id || !query.region_id) {
      packages.value = []
      return
    }

    const payload = await fetchAdminPackages({
      goods_type_id: query.goods_type_id
    })
    packages.value = payload.items || []
  }

  async function fetchOverviewBundle() {
    const payload = createQueryPayload()
    const [overviewPayload, trendPayload, topPayload] = await Promise.all([
      fetchAdminRevenueAnalyticsOverview(payload),
      fetchAdminRevenueAnalyticsTrend(payload),
      fetchAdminRevenueAnalyticsTop(payload)
    ])

    overview.value = overviewPayload || {}
    trend.value = trendPayload.items || []
    topItems.value = topPayload.items || []
  }

  async function fetchDetails() {
    const payload = await fetchAdminRevenueAnalyticsDetails(createQueryPayload())
    details.value = payload.items || []
    detailTotal.value = Number(payload.total || 0)
  }

  async function fetchAnalyticsData() {
    loading.value = true

    try {
      await Promise.all([fetchOverviewBundle(), fetchDetails()])
    } finally {
      loading.value = false
    }
  }

  async function fetchAllDetailRows(extra: Partial<RevenueAnalyticsQuery> = {}) {
    const base = createQueryPayload({
      page: 1,
      page_size: 200,
      sort_field: 'paid_at',
      sort_order: 'desc',
      ...extra
    })

    const rows: RevenueAnalyticsDetailRecord[] = []
    let page = 1
    const pageSize = Number(base.page_size || 200)

    for (;;) {
      const payload = await fetchAdminRevenueAnalyticsDetails({
        ...base,
        page
      })
      const items = payload.items || []
      const total = Number(payload.total || 0)

      rows.push(...items)

      if (items.length < pageSize || rows.length >= total || page >= 50) {
        break
      }

      page += 1
    }

    return rows
  }

  async function refreshUserRanking() {
    userRankLoading.value = true

    try {
      const rows = await fetchAllDetailRows({ user_id: undefined })
      const aggregate = new Map<number, { revenue_cents: number; orders: Set<number> }>()

      for (const row of rows) {
        const userId = Number(row.user_id || 0)
        if (!userId) {
          continue
        }

        if (!aggregate.has(userId)) {
          aggregate.set(userId, { revenue_cents: 0, orders: new Set<number>() })
        }

        const record = aggregate.get(userId)!
        record.revenue_cents += Number(row.amount_cents || 0)

        if (row.order_id) {
          record.orders.add(Number(row.order_id))
        }
      }

      userRankList.value = Array.from(aggregate.entries())
        .map(([userId, item]) => ({
          user_id: userId,
          revenue_cents: item.revenue_cents,
          order_count: item.orders.size
        }))
        .sort((a, b) => b.revenue_cents - a.revenue_cents)
        .slice(0, 20)
        .map((item, index) => ({
          ...item,
          rank: index + 1
        }))

      await ensureUserMeta(userRankList.value.map((item) => item.user_id))
    } finally {
      userRankLoading.value = false
    }
  }

  async function reloadAll() {
    if (!canView.value || !ensureQueryValid(false)) {
      return
    }

    syncInferredLevel()
    query.page = 1

    await Promise.all([fetchAnalyticsData(), refreshUserRanking()])
  }

  async function handleGoodsTypeChange(value?: number) {
    query.goods_type_id = normalizeNumber(value)
    query.region_id = undefined
    query.line_id = undefined
    query.package_id = undefined
    regions.value = []
    planGroups.value = []
    packages.value = []
    syncInferredLevel()
    await loadRegions()
  }

  async function handleRegionChange(value?: number) {
    query.region_id = normalizeNumber(value)
    query.line_id = undefined
    query.package_id = undefined
    planGroups.value = []
    packages.value = []
    syncInferredLevel()
    await Promise.all([loadPlanGroups(), loadPackages()])
  }

  async function handleLineChange(value?: number) {
    query.line_id = normalizeNumber(value)
    query.package_id = undefined
    syncInferredLevel()
    await loadPackages()
  }

  function handlePackageChange(value?: number) {
    query.package_id = normalizeNumber(value)
    syncInferredLevel()
  }

  function handleUserFilterChange(value?: number | null) {
    query.user_id = normalizeNumber(value)
    query.page = 1
  }

  function handleRangeChange(value?: [Date, Date] | null) {
    if (!value || value.length !== 2) {
      return
    }

    rangeValue.value = value

    const from = startOfDay(value[0])
    const to = endOfDay(value[1])
    quickRangeKey.value = detectQuickRangeKey(from, to)
    query.from_at = from.toISOString()
    query.to_at = to.toISOString()
    query.page = 1
  }

  function applyQuickRange(key: string) {
    const now = new Date()
    let from = startOfDay(now)
    const to = endOfDay(now)

    if (key === '7d') {
      from = startOfDay(subtractDays(now, 6))
    } else if (key === '30d') {
      from = startOfDay(subtractMonths(now, 1))
    } else if (key === 'month') {
      from = startOfMonth(now)
    }

    rangeValue.value = [from, to]
    query.from_at = from.toISOString()
    query.to_at = to.toISOString()
    query.page = 1
  }

  async function handleQuickRangeChange(value?: string | number | boolean) {
    const key = String(value || '30d')
    applyQuickRange(key)
    await reloadAll()
  }

  async function handlePageChange(page: number) {
    query.page = page
    await fetchDetails()
  }

  async function handlePageSizeChange(pageSize: number) {
    query.page_size = pageSize
    query.page = 1
    await fetchDetails()
  }

  async function handleDrillDown(
    item: DrillTarget | RevenueAnalyticsShareItem | RevenueAnalyticsTopItem
  ) {
    const dimensionId = Number(item.dimension_id || 0)
    if (!dimensionId) {
      return
    }

    const currentLevel = inferLevelFromSelection()

    if (currentLevel === 'overall') {
      await handleGoodsTypeChange(dimensionId)
    } else if (currentLevel === 'goods_type') {
      await handleRegionChange(dimensionId)
    } else if (currentLevel === 'region') {
      await handleLineChange(dimensionId)
    } else if (currentLevel === 'line') {
      handlePackageChange(dimensionId)
    } else {
      return
    }

    if (!ensureQueryValid(true)) {
      return
    }

    query.page = 1
    await Promise.all([fetchAnalyticsData(), refreshUserRanking()])
  }

  function summarizeUserFinance(rows: RevenueAnalyticsDetailRecord[]) {
    const orderMap = new Map<number, number>()
    let lastPaid = ''

    for (const row of rows) {
      const orderId = Number(row.order_id || 0)
      const amount = Number(row.amount_cents || 0)

      if (orderId > 0) {
        orderMap.set(orderId, (orderMap.get(orderId) || 0) + amount)
      }

      const paidAt = String(row.paid_at || '')
      const paidDate = toDate(paidAt)
      const lastPaidDate = toDate(lastPaid)

      if (paidDate && (!lastPaidDate || paidDate.getTime() > lastPaidDate.getTime())) {
        lastPaid = paidDate.toISOString()
      }
    }

    const orderTotals = Array.from(orderMap.values())
    const total = orderTotals.reduce((sum, item) => sum + item, 0)

    return {
      total_revenue_cents: total,
      order_count: orderTotals.length,
      positive_order_count: orderTotals.filter((item) => item > 0).length,
      negative_order_count: orderTotals.filter((item) => item < 0).length,
      avg_order_cents: orderTotals.length ? Math.trunc(total / orderTotals.length) : 0,
      last_paid_at: lastPaid ? formatDateTime(lastPaid) : ''
    }
  }

  async function openUserFinance(userId?: number | null) {
    const id = Number(userId || 0)
    if (!id) {
      return
    }

    userFinanceVisible.value = true
    userFinanceLoading.value = true
    selectedUserId.value = id

    try {
      const [profile, rows] = await Promise.all([
        fetchAdminUserDetail(id).catch(() => null),
        fetchAllDetailRows({ user_id: id })
      ])

      userFinanceProfile.value = profile
      setUserMeta(id, profile)
      userFinanceRows.value = rows.slice(0, 100)
      userFinanceSummary.value = summarizeUserFinance(rows)
      await ensureUserMeta(rows.map((item) => Number(item.user_id || 0)))
    } finally {
      userFinanceLoading.value = false
    }
  }

  async function applyUserFilterFromDrawer() {
    if (!selectedUserId.value) {
      return
    }

    query.user_id = selectedUserId.value
    query.page = 1
    userFinanceVisible.value = false
    await reloadAll()
  }

  async function handleResetFilters() {
    const [from, to] = defaultRange()

    Object.assign(query, createDefaultQuery(), {
      from_at: from.toISOString(),
      to_at: to.toISOString()
    })

    regions.value = []
    planGroups.value = []
    packages.value = []
    rangeValue.value = [from, to]
    quickRangeKey.value = '30d'
    syncInferredLevel()

    await reloadAll()
  }

  async function handleExportAudit() {
    if (!ensureQueryValid(false)) {
      return
    }

    exporting.value = true

    try {
      const blob = await exportAdminRevenueAnalyticsAudit(createQueryPayload())
      const url = window.URL.createObjectURL(blob)
      const link = document.createElement('a')
      const fileName = `revenue_analytics_audit_${formatDateForFilename()}.csv`

      link.href = url
      link.download = fileName
      document.body.appendChild(link)
      link.click()
      link.remove()
      window.URL.revokeObjectURL(url)

      ElMessage.success('审计导出已开始')
    } catch (error: any) {
      ElMessage.error(String(error?.message || '导出失败'))
    } finally {
      exporting.value = false
    }
  }
</script>

<style scoped lang="scss">
  .revenue-page {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .page-hero {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 16px;
    padding: 24px;
    border-radius: 18px;
    background:
      radial-gradient(circle at top left, rgb(34 197 94 / 18%), transparent 34%),
      radial-gradient(circle at top right, rgb(59 130 246 / 18%), transparent 28%),
      linear-gradient(135deg, rgb(255 255 255 / 96%), rgb(248 250 252 / 96%));
    border: 1px solid var(--el-border-color-light);
  }

  .page-hero h2 {
    margin: 0;
    color: var(--el-text-color-primary);
    font-size: 28px;
    font-weight: 700;
  }

  .page-hero p {
    margin: 8px 0 0;
    color: var(--el-text-color-secondary);
    font-size: 14px;
    line-height: 1.7;
  }

  .filter-card :deep(.el-card__body) {
    padding: 20px;
  }

  .filter-grid {
    display: grid;
    grid-template-columns: repeat(6, minmax(0, 1fr));
    gap: 12px;
    align-items: center;
  }

  .filter-grid :deep(.el-input-number),
  .filter-grid :deep(.el-date-editor),
  .filter-grid :deep(.el-select) {
    width: 100%;
  }

  .filter-actions {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
  }

  .filter-footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    flex-wrap: wrap;
  }

  .quick-range {
    display: flex;
    align-items: center;
    gap: 12px;
    flex-wrap: wrap;
  }

  .active-filters {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: 8px;
    flex-wrap: wrap;
  }

  .subtle {
    color: var(--el-text-color-secondary);
    font-size: 13px;
  }

  .card-row {
    margin: 0 !important;
  }

  .metric-card {
    overflow: hidden;
    border-radius: 16px;
  }

  .metric-card :deep(.el-card__body) {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .metric-card-revenue {
    background: linear-gradient(135deg, rgb(34 197 94 / 14%), rgb(255 255 255 / 100%));
  }

  .metric-card-mom {
    background: linear-gradient(135deg, rgb(59 130 246 / 14%), rgb(255 255 255 / 100%));
  }

  .metric-card-yoy {
    background: linear-gradient(135deg, rgb(249 115 22 / 14%), rgb(255 255 255 / 100%));
  }

  .metric-card-net {
    background: linear-gradient(135deg, rgb(168 85 247 / 14%), rgb(255 255 255 / 100%));
  }

  .metric-label {
    color: var(--el-text-color-secondary);
    font-size: 13px;
    font-weight: 600;
  }

  .metric-value {
    color: var(--el-text-color-primary);
    font-size: 30px;
    font-weight: 700;
    line-height: 1.1;
  }

  .metric-footer,
  .dual-compare {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    flex-wrap: wrap;
    color: var(--el-text-color-secondary);
    font-size: 13px;
  }

  .metric-trend,
  .metric-subtle {
    font-size: 13px;
    font-weight: 600;
  }

  .metric-subtle {
    color: var(--el-text-color-secondary);
    font-weight: 400;
  }

  .section-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }

  .section-title {
    color: var(--el-text-color-primary);
    font-size: 16px;
    font-weight: 700;
  }

  .section-subtitle {
    margin-top: 4px;
    color: var(--el-text-color-secondary);
    font-size: 13px;
  }

  .chart-card,
  .panel-card,
  .detail-card {
    border-radius: 18px;
  }

  .share-list {
    margin-top: 18px;
  }

  .share-items,
  .ranking-list {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .share-item,
  .ranking-item {
    width: 100%;
    padding: 12px 14px;
    border: 1px solid var(--el-border-color-light);
    border-radius: 14px;
    background: var(--el-bg-color);
    cursor: pointer;
    transition:
      transform 0.2s ease,
      border-color 0.2s ease,
      box-shadow 0.2s ease;
  }

  .share-item:hover,
  .ranking-item:hover {
    transform: translateY(-1px);
    border-color: rgb(59 130 246 / 32%);
    box-shadow: 0 12px 24px rgb(15 23 42 / 8%);
  }

  .share-item-top,
  .ranking-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }

  .share-name,
  .ranking-name {
    color: var(--el-text-color-primary);
    font-weight: 600;
  }

  .share-value,
  .ranking-right {
    display: flex;
    align-items: center;
    gap: 10px;
    color: var(--el-text-color-secondary);
    font-size: 13px;
  }

  .ranking-left {
    display: flex;
    align-items: center;
    gap: 10px;
    min-width: 0;
  }

  .trend-insight-grid {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 12px;
    margin-top: 18px;
  }

  .insight-card {
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding: 14px;
    border-radius: 14px;
    background: var(--el-fill-color-light);
  }

  .insight-label {
    color: var(--el-text-color-secondary);
    font-size: 13px;
  }

  .insight-card strong {
    color: var(--el-text-color-primary);
    font-size: 16px;
  }

  .insight-value {
    color: var(--el-color-primary);
    font-size: 13px;
    font-weight: 600;
  }

  .user-cell {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .table-pagination {
    display: flex;
    justify-content: flex-end;
    margin-top: 16px;
  }

  .compare-up {
    color: var(--el-color-success);
  }

  .compare-down {
    color: var(--el-color-danger);
  }

  .compare-neutral {
    color: var(--el-text-color-secondary);
  }

  .amount-up {
    color: var(--el-color-success);
    font-weight: 700;
  }

  .amount-down {
    color: var(--el-color-danger);
    font-weight: 700;
  }

  .amount-neutral {
    color: var(--el-text-color-secondary);
    font-weight: 700;
  }

  @media (max-width: 1400px) {
    .filter-grid {
      grid-template-columns: repeat(3, minmax(0, 1fr));
    }
  }

  @media (max-width: 900px) {
    .page-hero {
      flex-direction: column;
    }

    .trend-insight-grid {
      grid-template-columns: 1fr;
    }
  }

  @media (max-width: 768px) {
    .filter-grid {
      grid-template-columns: 1fr;
    }

    .filter-actions {
      justify-content: flex-start;
    }

    .table-pagination {
      justify-content: flex-start;
      overflow-x: auto;
    }
  }
</style>
