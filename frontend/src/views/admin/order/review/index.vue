<template>
  <div class="order-review-page art-full-height">
    <OrderSearch
      v-show="showSearchBar"
      v-model="searchForm"
      @search="handleSearch"
      @reset="resetSearchParams"
    />

    <ElCard class="art-table-card" :style="{ marginTop: showSearchBar ? '12px' : '0' }">
      <ArtTableHeader
        v-model:columns="columnChecks"
        v-model:show-search-bar="showSearchBar"
        :loading="loading"
        @refresh="refreshData"
      >
        <template #left>
          <ElSpace wrap>
            <ElRadioGroup v-model="quickStatus" class="quick-status-group">
              <ElRadioButton label="all">全部</ElRadioButton>
              <ElRadioButton v-for="item in quickStatusTabs" :key="item.value" :label="item.value">
                {{ item.label }}
              </ElRadioButton>
            </ElRadioGroup>
            <ElButton v-ripple @click="exportCsv">导出 CSV</ElButton>
          </ElSpace>
        </template>
      </ArtTableHeader>

      <ArtTable
        class="desktop-order-table"
        row-key="id"
        :loading="loading"
        :data="data"
        :columns="columns"
        :pagination="pagination"
        @pagination:size-change="handleSizeChange"
        @pagination:current-change="handleCurrentChange"
      >
        <template #status="{ row }">
          <ElTag :type="getOrderStatusTagType(row.status)">{{
            getOrderStatusText(row.status)
          }}</ElTag>
        </template>
        <template #operation="{ row }">
          <div class="table-actions">
            <ArtButtonTable v-auth="'order.view'" type="view" @click="openDetail(row)" />
            <ElButton
              v-auth="'order.approve'"
              v-ripple
              link
              type="primary"
              :disabled="isReviewLocked(row)"
              @click="approve(row)"
            >
              通过
            </ElButton>
            <ElButton
              v-auth="'order.reject'"
              v-ripple
              link
              type="danger"
              :disabled="isReviewLocked(row)"
              @click="openReject(row)"
            >
              驳回
            </ElButton>
            <ArtButtonTable v-auth="'order.delete'" type="delete" @click="removeOrder(row)" />
          </div>
        </template>
      </ArtTable>

      <div v-loading="loading" class="mobile-order-list" aria-live="polite">
        <ElEmpty v-if="!loading && data.length === 0" description="暂无订单" :image-size="88" />
        <article v-for="item in data" :key="item.id || item.order_no" class="mobile-order-item">
          <div class="mobile-order-header">
            <strong class="mobile-order-no">#{{ item.order_no || item.id || '-' }}</strong>
            <ElTag :type="getOrderStatusTagType(item.status)">
              {{ getOrderStatusText(item.status) }}
            </ElTag>
          </div>
          <dl class="mobile-order-meta">
            <div>
              <dt>用户</dt>
              <dd>{{ item.user_id || '-' }}</dd>
            </div>
            <div>
              <dt>金额</dt>
              <dd>￥{{ formatAmount(item.total_amount) }}</dd>
            </div>
          </dl>
          <div class="mobile-order-actions">
            <ElButton v-auth="'order.view'" v-ripple size="small" @click="openDetail(item)">
              详情
            </ElButton>
            <ElButton
              v-auth="'order.approve'"
              v-ripple
              size="small"
              type="primary"
              :disabled="isReviewLocked(item)"
              @click="approve(item)"
            >
              通过
            </ElButton>
            <ElButton
              v-auth="'order.delete'"
              v-ripple
              size="small"
              type="danger"
              plain
              @click="removeOrder(item)"
            >
              删除
            </ElButton>
          </div>
        </article>
      </div>
    </ElCard>

    <ElDialog v-model="rejectVisible" title="驳回订单" width="460px" destroy-on-close align-center>
      <ElForm label-position="top">
        <ElFormItem label="驳回原因">
          <ElInput
            v-model="rejectReason"
            type="textarea"
            :rows="4"
            placeholder="请输入驳回原因"
            :maxlength="INPUT_LIMITS.REVIEW_REASON"
            show-word-limit
          />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <div class="dialog-footer">
          <ElButton v-ripple @click="rejectVisible = false">取消</ElButton>
          <ElButton v-ripple type="primary" :loading="rejectSubmitting" @click="submitReject"
            >确认驳回</ElButton
          >
        </div>
      </template>
    </ElDialog>

    <OrderDetailDrawer
      v-model:visible="detailVisible"
      :loading="detailLoading"
      :detail="detail"
      :related-user-name="relatedUserName"
      @approve="approve(detail?.order)"
      @reject="openReject(detail?.order)"
      @retry="retry(detail?.order)"
      @delete="removeOrder(detail?.order)"
    />
  </div>
</template>

<script setup lang="ts">
  import ArtButtonTable from '@/components/core/forms/art-button-table/index.vue'
  import { useTable } from '@/hooks/core/useTable'
  import { useLifecycleSafeTimer } from '@/hooks/core/useLifecycleSafeTimer'
  import {
    approveAdminOrder,
    deleteAdminOrder,
    getAdminOrderDetail,
    getAdminUserDetail,
    listAdminOrders,
    rejectAdminOrder,
    retryAdminOrder
  } from '@/services/admin'
  import type { Order, OrderDetailWithEventsResponse } from '@/services/types'
  import { INPUT_LIMITS } from '@/constants/inputLimits'
  import { toCsvLine } from '@/utils/csv'
  import OrderDetailDrawer from './modules/order-detail-drawer.vue'
  import OrderSearch from './modules/order-search.vue'

  defineOptions({ name: 'OrderReview' })

  interface OrderRecordLike extends Order {
    UserID?: unknown
    OrderNo?: unknown
    Source?: unknown
    CanReview?: unknown
    Currency?: unknown
    PendingReason?: unknown
    ApprovedBy?: unknown
    ApprovedAt?: unknown
    RejectedReason?: unknown
    UpdatedAt?: unknown
    can_review?: unknown
  }

  interface OrderTableRow {
    id: number | null
    user_id: number | null
    order_no: string
    source: string
    status: string
    can_review: boolean
    total_amount: number
    currency: string
    pending_reason: string
    approved_by: number | null
    approved_at: string | null
    rejected_reason: string
    created_at: string
    updated_at: string
  }

  interface OrderTableParams extends Api.Common.CommonSearchParams {
    keyword?: string
    status?: string
    user_id?: string
    order_no?: string
  }

  const quickStatusTabs = [
    { label: '待审核', value: 'pending_review' },
    { label: '开通中', value: 'provisioning' },
    { label: '失败', value: 'failed' }
  ] as const

  const showSearchBar = ref(true)
  const detailLoading = ref(false)
  const detailVisible = ref(false)
  const rejectVisible = ref(false)
  const rejectSubmitting = ref(false)
  const searchForm = ref<OrderTableParams>(createDefaultSearchForm())
  const detail = ref<OrderDetailWithEventsResponse | null>(null)
  const relatedUserName = ref('')
  const rejectReason = ref('')
  const rejectTarget = ref<OrderTableRow | Order | null>(null)
  const pollOrderId = ref<number | null>(null)
  /** 详情请求代次：慢响应/切换订单时旧响应不得覆盖新详情（F1）。 */
  let detailReqId = 0
  let detailController: AbortController | null = null
  let loadingDetail = false
  /** 上一次成功加载过用户名对应的 user_id：未变化时跳过重复拉取（F5）。 */
  let lastLoadedUserId = 0

  const quickStatus = computed({
    get: () =>
      quickStatusTabs.some((item) => item.value === searchForm.value.status)
        ? searchForm.value.status!
        : 'all',
    set: (value: string) => {
      const status = value === 'all' ? undefined : value
      searchForm.value = { ...searchForm.value, status }
      Object.assign(searchParams, { status })
      getData()
    }
  })

  async function fetchOrderTable(
    params: OrderTableParams,
    signal?: AbortSignal
  ): Promise<Api.Common.PaginatedResponse<OrderTableRow>> {
    const response = await listAdminOrders(
      {
        limit: params.size,
        offset: (params.current - 1) * params.size,
        status: params.status || undefined,
        user_id: normalizeUserId(params.user_id),
        order_no: params.order_no?.trim() || undefined
      },
      signal
    )
    const payload = response.data || { items: [], total: 0 }
    let records = (payload.items || []).map(normalizeOrder)
    const keyword = params.keyword?.trim()
    if (keyword) records = records.filter((item) => String(item.id || '').includes(keyword))
    return {
      records,
      current: params.current,
      size: params.size,
      total: payload.total ?? records.length
    }
  }

  const {
    columns,
    columnChecks,
    data,
    loading,
    pagination,
    getData,
    searchParams,
    resetSearchParams,
    handleSizeChange,
    handleCurrentChange,
    refreshData
  } = useTable({
    core: {
      apiFn: fetchOrderTable,
      apiParams: { current: 1, size: 20, ...createDefaultSearchForm() },
      columnsFactory: () => [
        { type: 'selection', width: 48 },
        { prop: 'id', label: '订单 ID', width: 100 },
        { prop: 'user_id', label: '用户 ID', width: 100 },
        { prop: 'order_no', label: '订单号', minWidth: 180, showOverflowTooltip: true },
        { prop: 'status', label: '状态', width: 120, useSlot: true },
        {
          prop: 'total_amount',
          label: '总金额',
          minWidth: 120,
          formatter: (row: OrderTableRow) => `￥${formatAmount(row.total_amount)}`
        },
        {
          prop: 'created_at',
          label: '创建时间',
          minWidth: 180,
          formatter: (row: OrderTableRow) => formatDateTime(row.created_at)
        },
        { prop: 'operation', label: '操作', width: 220, fixed: 'right', useSlot: true }
      ]
    },
    hooks: {
      resetFormCallback: () => {
        searchForm.value = createDefaultSearchForm()
      }
    }
  })

  function createDefaultSearchForm(): OrderTableParams {
    return { current: 1, size: 20, keyword: '', status: undefined, user_id: '', order_no: '' }
  }

  function normalizeNullableNumber(value: unknown) {
    if (value === '' || value === null || value === undefined) return null
    const parsed = Number(value)
    return Number.isFinite(parsed) ? parsed : null
  }

  function normalizeNullableString(value: unknown) {
    if (value === '' || value === null || value === undefined) return null
    return String(value)
  }

  function normalizeOrder(row: OrderRecordLike = {}): OrderTableRow {
    return {
      id: normalizeNullableNumber(row.id ?? row.ID),
      user_id: normalizeNullableNumber(row.user_id ?? row.UserID),
      order_no: String(row.order_no ?? row.OrderNo ?? ''),
      source: String((row as any).source ?? row.Source ?? ''),
      status: String(row.status ?? row.Status ?? ''),
      can_review: Boolean(row.can_review ?? row.CanReview ?? false),
      total_amount: Number(row.total_amount ?? row.TotalAmount ?? 0),
      currency: String(row.currency ?? row.Currency ?? 'CNY'),
      pending_reason: String(row.pending_reason ?? row.PendingReason ?? ''),
      approved_by: normalizeNullableNumber(row.approved_by ?? row.ApprovedBy),
      approved_at: normalizeNullableString(row.approved_at ?? row.ApprovedAt),
      rejected_reason: String(row.rejected_reason ?? row.RejectedReason ?? ''),
      created_at: String(row.created_at ?? row.CreatedAt ?? ''),
      updated_at: String(row.updated_at ?? row.UpdatedAt ?? '')
    }
  }

  function normalizeUserId(value?: string) {
    const trimmed = String(value || '').trim()
    if (!trimmed) return undefined
    const parsed = Number(trimmed)
    return Number.isFinite(parsed) && parsed > 0 ? parsed : undefined
  }

  function formatAmount(value?: number | null) {
    return Number(value || 0).toFixed(2)
  }
  function formatDateTime(value?: string | null) {
    if (!value) return '-'
    const date = new Date(value)
    return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN')
  }
  function getOrderStatusText(status?: string) {
    const map: Record<string, string> = {
      pending_payment: '待支付',
      pending_review: '待审核',
      approved: '已通过',
      provisioning: '开通中',
      active: '已完成',
      failed: '失败',
      rejected: '已驳回'
    }
    return map[String(status || '')] || String(status || '-')
  }
  function getOrderStatusTagType(status?: string) {
    const map: Record<string, 'info' | 'warning' | 'success' | 'danger'> = {
      pending_payment: 'info',
      pending_review: 'warning',
      approved: 'success',
      provisioning: 'warning',
      active: 'success',
      failed: 'danger',
      rejected: 'danger'
    }
    return map[String(status || '')] || 'info'
  }
  function isReviewLocked(record?: OrderTableRow | Order | null) {
    if (!record) return true
    const raw = record as OrderRecordLike
    const canReview = raw.can_review ?? raw.CanReview
    if (canReview !== undefined) return !canReview
    return !['pending_review', 'pending-review', 'pending review'].includes(
      String(raw.status ?? raw.Status ?? '')
        .trim()
        .toLowerCase()
    )
  }
  function handleSearch(params: OrderTableParams) {
    searchForm.value = { ...searchForm.value, ...params }
    Object.assign(searchParams, {
      keyword: params.keyword || undefined,
      status: params.status || undefined,
      user_id: params.user_id || undefined,
      order_no: params.order_no || undefined
    })
    getData()
  }

  async function loadDetail(orderId: number | string, options: { silent?: boolean } = {}) {
    // 在途保护 + 请求代次：慢响应/切换订单时旧响应不得覆盖新详情（F1）。
    if (loadingDetail) return
    loadingDetail = true
    detailController?.abort()
    detailController = new AbortController()
    const requestId = ++detailReqId
    // F2：后台轮询 tick 用 silent 模式，不把整个详情抽屉盖 loading 遮罩
    // （每 5s 全面板闪烁）；只有首次打开/手动刷新才显示 loading。
    if (!options.silent) detailLoading.value = true
    try {
      const response = await getAdminOrderDetail(orderId, detailController.signal)
      if (requestId !== detailReqId) return
      const payload = response.data as OrderDetailWithEventsResponse
      detail.value = { ...payload, events: payload.events || (payload as any).logs || [] }
      const userId = detail.value.order?.user_id
      if (!userId) {
        relatedUserName.value = ''
        lastLoadedUserId = 0
        return
      }
      if (requestId !== detailReqId) return
      // 用户名字段基本不变，仅当 user_id 变化才重新拉取，避免每次轮询 N+1（F5）。
      const numericUserId = Number(userId)
      if (numericUserId === lastLoadedUserId && relatedUserName.value) {
        return
      }
      try {
        const userResponse = await getAdminUserDetail(userId, detailController.signal)
        if (requestId !== detailReqId) return
        relatedUserName.value = String(
          (userResponse.data as any)?.username ?? (userResponse.data as any)?.Username ?? ''
        )
        lastLoadedUserId = numericUserId
      } catch (error) {
        if (requestId === detailReqId && !isCanceledError(error)) relatedUserName.value = ''
      }
    } catch (error) {
      if (!isCanceledError(error)) throw error
    } finally {
      if (requestId === detailReqId) detailLoading.value = false
      loadingDetail = false
    }
  }

  function isCanceledError(error: unknown): boolean {
    return Boolean(
      (error as { code?: string })?.code === 'ERR_CANCELED' ||
      (error as { message?: string })?.message === 'canceled' ||
      (error as { config?: { signal?: { aborted?: boolean } } })?.config?.signal?.aborted
    )
  }

  function getCurrentDetailId() {
    return Number(detail.value?.order?.id || 0) || null
  }
  // web_refresh_perf F1/F2：轮询用生命周期安全定时器——keep-alive 切后台时
  // onDeactivated 停表（不再后台每 5s 打两发请求），切回前台恢复；轮询 tick
  // 走 silent 模式不闪整面板 loading。
  const { schedule: scheduleDetailPoll, cancel: cancelDetailPoll } = useLifecycleSafeTimer(() => {
    const id = pollOrderId.value
    if (!id || !detailVisible.value) return
    // 错误已由 http 拦截器弹 toast，轮询 tick 静默吞掉避免 unhandled rejection。
    void loadDetail(id, { silent: true }).catch(() => {})
  }, 5000)
  function startPolling(orderId: number) {
    pollOrderId.value = orderId
    scheduleDetailPoll()
  }
  function stopPolling() {
    cancelDetailPoll()
  }
  async function openDetail(record?: OrderTableRow | Order | null) {
    const orderId = Number(record?.id || 0)
    if (!orderId) return
    detailVisible.value = true
    await loadDetail(orderId)
    startPolling(orderId)
  }

  watch(detailVisible, (visible) => {
    if (!visible) {
      stopPolling()
      detailController?.abort()
    }
  })
  onUnmounted(() => {
    stopPolling()
    detailController?.abort()
  })
  // keep-alive 切后台停表、切回前台恢复（抽屉开着时继续轮询），F1。
  onDeactivated(() => {
    cancelDetailPoll()
  })
  onActivated(() => {
    if (detailVisible.value && pollOrderId.value) scheduleDetailPoll()
  })

  async function approve(record?: OrderTableRow | Order | null) {
    const orderId = Number(record?.id || 0)
    if (!orderId) return
    try {
      await ElMessageBox.confirm('确认通过该订单？', '提示', {
        confirmButtonText: '确认',
        cancelButtonText: '取消',
        type: 'warning'
      })
    } catch {
      return
    }
    await approveAdminOrder(orderId)
    ElMessage.success('已通过')
    await refreshData()
    if (detailVisible.value) await loadDetail(orderId)
  }

  function openReject(record?: OrderTableRow | Order | null) {
    const orderId = Number(record?.id || 0)
    if (!orderId) return
    rejectTarget.value = record || null
    rejectReason.value = ''
    rejectVisible.value = true
  }
  async function submitReject() {
    const orderId = Number(rejectTarget.value?.id || 0)
    if (!orderId) return
    if (rejectReason.value.length > INPUT_LIMITS.REVIEW_REASON) {
      ElMessage.error(`驳回原因长度不能超过 ${INPUT_LIMITS.REVIEW_REASON} 个字符`)
      return
    }
    rejectSubmitting.value = true
    try {
      await rejectAdminOrder(orderId, { reason: rejectReason.value || 'manual' })
      ElMessage.success('已驳回')
      rejectVisible.value = false
      await refreshData()
      if (detailVisible.value) await loadDetail(orderId)
    } finally {
      rejectSubmitting.value = false
    }
  }
  async function retry(record?: OrderTableRow | Order | null) {
    const orderId = Number(record?.id || 0)
    if (!orderId) return
    await retryAdminOrder(orderId)
    ElMessage.success('已触发重试')
    await refreshData()
    if (detailVisible.value) await loadDetail(orderId)
  }
  async function removeOrder(record?: OrderTableRow | Order | null) {
    const orderId = Number(record?.id || 0)
    if (!orderId) return
    try {
      await ElMessageBox.confirm('该操作会删除订单及其关联记录，无法恢复。', '确认删除该订单？', {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        type: 'warning'
      })
    } catch {
      return
    }
    await deleteAdminOrder(orderId)
    ElMessage.success('订单已删除')
    if (getCurrentDetailId() === orderId) {
      detailVisible.value = false
      detail.value = null
      relatedUserName.value = ''
    }
    await refreshData()
  }

  function exportCsv() {
    const rows = data.value.map((item) =>
      toCsvLine([
        item.id,
        item.order_no,
        item.user_id,
        item.status,
        item.total_amount,
        item.currency,
        item.created_at
      ])
    )
    const content = ['id,order_no,user_id,status,total_amount,currency,created_at', ...rows].join(
      '\n'
    )
    const blob = new Blob([content], { type: 'text/csv;charset=utf-8;' })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = 'admin-orders.csv'
    link.click()
    URL.revokeObjectURL(url)
  }
</script>

<style scoped lang="scss">
  .quick-status-group {
    flex-wrap: wrap;
  }

  .table-actions {
    display: flex;
    gap: 4px;
    align-items: center;
    justify-content: flex-end;
  }

  .mobile-order-list {
    display: none;
    min-height: 180px;
  }

  .mobile-order-item {
    padding: 16px 0;
    border-bottom: 1px solid var(--art-card-border);
  }

  .mobile-order-item:last-child {
    border-bottom: 0;
  }

  .mobile-order-header {
    display: flex;
    gap: 12px;
    align-items: center;
    justify-content: space-between;
  }

  .mobile-order-no {
    min-width: 0;
    color: var(--art-gray-900);
    overflow-wrap: anywhere;
  }

  .mobile-order-meta {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 12px;
    margin: 14px 0 0;
  }

  .mobile-order-meta div {
    display: flex;
    flex-direction: column;
    gap: 4px;
    min-width: 0;
  }

  .mobile-order-meta dt {
    font-size: 12px;
    color: var(--art-gray-600);
  }

  .mobile-order-meta dd {
    margin: 0;
    font-weight: 600;
    color: var(--art-gray-900);
    overflow-wrap: anywhere;
  }

  .mobile-order-actions {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    margin-top: 14px;
  }

  .dialog-footer {
    display: flex;
    gap: 12px;
    justify-content: flex-end;
  }

  @media (width <= 768px) {
    .desktop-order-table {
      display: none;
    }

    .mobile-order-list {
      display: block;
    }

    .table-actions {
      flex-wrap: wrap;
      justify-content: flex-start;
    }
  }
</style>
