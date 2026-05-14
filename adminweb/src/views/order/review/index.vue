<template>
  <div class="art-full-height">
    <OrderSearch
      v-show="showSearchBar"
      v-model="searchForm"
      @search="handleSearch"
      @reset="handleReset"
    />

    <ElCard class="art-table-card" :style="{ marginTop: showSearchBar ? '12px' : '0' }">
      <ArtTableHeader
        v-model:columns="columnChecks"
        v-model:showSearchBar="showSearchBar"
        :loading="loading"
        @refresh="fetchData"
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
        row-key="id"
        :loading="loading"
        :data="tableData"
        :columns="columns"
        :pagination="pagination"
        @pagination:size-change="handlePageSizeChange"
        @pagination:current-change="handlePageCurrentChange"
      >
        <template #status="{ row }">
          <ElTag :type="getOrderStatusTagType(row.status)">
            {{ getOrderStatusText(row.status) }}
          </ElTag>
        </template>

        <template #operation="{ row }">
          <div class="table-actions">
            <ArtButtonTable type="view" @click="openDetail(row)" />

            <ElButton link type="primary" :disabled="isReviewLocked(row)" @click="approve(row)">
              通过
            </ElButton>

            <ElButton link type="danger" :disabled="isReviewLocked(row)" @click="openReject(row)">
              驳回
            </ElButton>

            <ArtButtonMore
              :list="getMoreActions(row)"
              @click="(item) => handleMoreAction(item, row)"
            />
          </div>
        </template>
      </ArtTable>
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
          <ElButton @click="rejectVisible = false">取消</ElButton>
          <ElButton type="primary" :loading="rejectSubmitting" @click="submitReject">
            确认驳回
          </ElButton>
        </div>
      </template>
    </ElDialog>

    <OrderDetailDrawer
      v-model:visible="detailVisible"
      :loading="detailLoading"
      :detail="detail"
      :related-user-name="relatedUserName"
      :can-delete="canDelete"
      @approve="approve(detail?.order)"
      @reject="openReject(detail?.order)"
      @retry="retry(detail?.order)"
      @delete="removeOrder(detail?.order)"
    />
  </div>
</template>

<script setup lang="ts">
  import type { OrderDetailResponse, OrderRecord, UserRecord } from '@/api/admin'
  import type { ButtonMoreItem } from '@/components/core/forms/art-button-more/index.vue'
  import {
    approveAdminOrder,
    deleteAdminOrder,
    fetchAdminOrderDetail,
    fetchAdminOrders,
    fetchAdminUserDetail,
    hasAdminPermission,
    rejectAdminOrder,
    retryAdminOrder
  } from '@/api/admin'
  import ArtButtonMore from '@/components/core/forms/art-button-more/index.vue'
  import ArtButtonTable from '@/components/core/forms/art-button-table/index.vue'
  import { useTableColumns } from '@/hooks/core/useTableColumns'
  import { useUserStore } from '@/store/modules/user'
  import { INPUT_LIMITS } from '@/utils/constants'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import OrderDetailDrawer from './modules/order-detail-drawer.vue'
  import OrderSearch from './modules/order-search.vue'

  defineOptions({ name: 'OrderReview' })

  interface OrderRecordLike extends Partial<OrderRecord> {
    ID?: unknown
    UserID?: unknown
    OrderNo?: unknown
    Source?: unknown
    Status?: unknown
    CanReview?: unknown
    TotalAmount?: unknown
    Currency?: unknown
    PendingReason?: unknown
    ApprovedBy?: unknown
    ApprovedAt?: unknown
    RejectedReason?: unknown
    CreatedAt?: unknown
    UpdatedAt?: unknown
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

  interface OrderSearchForm {
    keyword: string
    status?: string
    user_id: string
    order_no: string
  }

  const quickStatusTabs = [
    { label: '待审核', value: 'pending_review' },
    { label: '开通中', value: 'provisioning' },
    { label: '失败', value: 'failed' }
  ] as const

  const showSearchBar = ref(true)
  const loading = ref(false)
  const detailLoading = ref(false)
  const detailVisible = ref(false)
  const rejectVisible = ref(false)
  const rejectSubmitting = ref(false)

  const searchForm = ref<OrderSearchForm>(createDefaultSearchForm())
  const tableData = ref<OrderTableRow[]>([])
  const detail = ref<OrderDetailResponse | null>(null)
  const relatedUserName = ref('')
  const rejectReason = ref('')
  const rejectTarget = ref<OrderRecord | OrderTableRow | null>(null)

  const pagination = reactive({
    current: 1,
    size: 20,
    total: 0
  })

  const poller = ref<number | null>(null)

  const { columnChecks, columns } = useTableColumns<OrderTableRow>(() => [
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
    {
      prop: 'operation',
      label: '操作',
      width: 250,
      fixed: 'right',
      useSlot: true
    }
  ])

  const userStore = useUserStore()
  const { info } = storeToRefs(userStore)

  const canDelete = computed(() => hasAdminPermission(info.value?.buttons, 'order.delete'))

  const quickStatus = computed({
    get: () => {
      const status = searchForm.value.status || ''
      return quickStatusTabs.some((item) => item.value === status) ? status : 'all'
    },
    set: (value: string) => {
      searchForm.value = {
        ...searchForm.value,
        status: value === 'all' ? undefined : value
      }
      pagination.current = 1
      fetchData()
    }
  })

  onMounted(() => {
    fetchData()
  })

  onUnmounted(() => {
    stopPolling()
  })

  watch(detailVisible, (visible) => {
    if (!visible) {
      stopPolling()
    }
  })

  function createDefaultSearchForm(): OrderSearchForm {
    return {
      keyword: '',
      status: undefined,
      user_id: '',
      order_no: ''
    }
  }

  function normalizeNullableNumber(value: unknown): number | null {
    if (value === '' || value === null || value === undefined) {
      return null
    }

    const parsed = Number(value)
    return Number.isFinite(parsed) ? parsed : null
  }

  function normalizeNullableString(value: unknown): string | null {
    if (value === '' || value === null || value === undefined) {
      return null
    }

    return String(value)
  }

  function normalizeOrder(row?: OrderRecordLike): OrderTableRow {
    const source = row || {}

    return {
      id: normalizeNullableNumber(source.id ?? source.ID),
      user_id: normalizeNullableNumber(source.user_id ?? source.UserID),
      order_no: String(source.order_no ?? source.OrderNo ?? ''),
      source: String(source.source ?? source.Source ?? ''),
      status: String(source.status ?? source.Status ?? ''),
      can_review: Boolean(source.can_review ?? source.CanReview ?? false),
      total_amount: Number(source.total_amount ?? source.TotalAmount ?? 0),
      currency: String(source.currency ?? source.Currency ?? 'CNY'),
      pending_reason: String(source.pending_reason ?? source.PendingReason ?? ''),
      approved_by: normalizeNullableNumber(source.approved_by ?? source.ApprovedBy),
      approved_at: normalizeNullableString(source.approved_at ?? source.ApprovedAt),
      rejected_reason: String(source.rejected_reason ?? source.RejectedReason ?? ''),
      created_at: String(source.created_at ?? source.CreatedAt ?? ''),
      updated_at: String(source.updated_at ?? source.UpdatedAt ?? '')
    }
  }

  function normalizeUserId(value: string) {
    const trimmed = String(value || '').trim()
    if (!trimmed) {
      return undefined
    }

    const parsed = Number(trimmed)
    return Number.isFinite(parsed) && parsed > 0 ? parsed : undefined
  }

  function formatAmount(value?: number | null) {
    return Number(value || 0).toFixed(2)
  }

  function formatDateTime(value?: string | null) {
    if (!value) {
      return '-'
    }

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

  function isReviewLocked(record?: OrderTableRow | Partial<OrderRecord> | null) {
    if (!record) {
      return true
    }

    if (record.can_review !== undefined) {
      return !record.can_review
    }

    return (
      String(record.status || '')
        .trim()
        .toLowerCase() !== 'pending_review'
    )
  }

  function canRetry(record?: OrderTableRow | Partial<OrderRecord> | null) {
    const status = String(record?.status || '')
      .trim()
      .toLowerCase()
    return ['approved', 'provisioning', 'failed'].includes(status)
  }

  function getCurrentDetailId() {
    return Number(detail.value?.order?.id || 0) || null
  }

  async function fetchData() {
    loading.value = true

    try {
      const payload = await fetchAdminOrders({
        limit: pagination.size,
        offset: (pagination.current - 1) * pagination.size,
        status: searchForm.value.status || undefined,
        user_id: normalizeUserId(searchForm.value.user_id),
        order_no: searchForm.value.order_no.trim() || undefined
      })

      let rows = (payload.items || []).map((item) => normalizeOrder(item))
      const keyword = searchForm.value.keyword.trim()

      if (keyword) {
        rows = rows.filter((item) => String(item.id || '').includes(keyword))
      }

      tableData.value = rows
      pagination.total = typeof payload.total === 'number' ? payload.total : rows.length
    } finally {
      loading.value = false
    }
  }

  function handleSearch(params: OrderSearchForm) {
    searchForm.value = { ...searchForm.value, ...params }
    pagination.current = 1
    fetchData()
  }

  function handleReset() {
    searchForm.value = createDefaultSearchForm()
    pagination.current = 1
    fetchData()
  }

  function handlePageSizeChange(size: number) {
    pagination.size = size
    pagination.current = 1
    fetchData()
  }

  function handlePageCurrentChange(page: number) {
    pagination.current = page
    fetchData()
  }

  async function loadDetail(orderId: number | string) {
    detailLoading.value = true

    try {
      const payload = await fetchAdminOrderDetail(orderId)
      detail.value = payload

      const userId = payload.order?.user_id
      if (!userId) {
        relatedUserName.value = ''
        return
      }

      try {
        const user = await fetchAdminUserDetail(userId)
        relatedUserName.value = resolveUserName(user)
      } catch {
        relatedUserName.value = ''
      }
    } finally {
      detailLoading.value = false
    }
  }

  function resolveUserName(user?: UserRecord | null) {
    return String(user?.username || '')
  }

  function startPolling(orderId: number) {
    stopPolling()
    poller.value = window.setInterval(() => {
      loadDetail(orderId)
    }, 5000)
  }

  function stopPolling() {
    if (poller.value) {
      window.clearInterval(poller.value)
      poller.value = null
    }
  }

  async function openDetail(record?: OrderTableRow | Partial<OrderRecord> | null) {
    const orderId = Number(record?.id || 0)
    if (!orderId) {
      return
    }

    detailVisible.value = true
    await loadDetail(orderId)
    startPolling(orderId)
  }

  async function refreshDetail(orderId?: number | null) {
    const targetId = orderId || getCurrentDetailId()
    if (!targetId || !detailVisible.value) {
      return
    }

    await loadDetail(targetId)
  }

  async function approve(record?: OrderTableRow | Partial<OrderRecord> | null) {
    const orderId = Number(record?.id || 0)
    if (!orderId) {
      return
    }

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
    await fetchData()
    await refreshDetail(orderId)
  }

  function openReject(record?: OrderTableRow | Partial<OrderRecord> | null) {
    const orderId = Number(record?.id || 0)
    if (!orderId) {
      return
    }

    rejectTarget.value = record as OrderRecord
    rejectReason.value = ''
    rejectVisible.value = true
  }

  async function submitReject() {
    const orderId = Number(rejectTarget.value?.id || 0)
    if (!orderId) {
      return
    }

    if (String(rejectReason.value || '').length > INPUT_LIMITS.REVIEW_REASON) {
      ElMessage.error(`驳回原因长度不能超过 ${INPUT_LIMITS.REVIEW_REASON} 个字符`)
      return
    }

    rejectSubmitting.value = true

    try {
      await rejectAdminOrder(orderId, {
        reason: rejectReason.value || 'manual'
      })

      ElMessage.success('已驳回')
      rejectVisible.value = false
      await fetchData()
      await refreshDetail(orderId)
    } finally {
      rejectSubmitting.value = false
    }
  }

  async function retry(record?: OrderTableRow | Partial<OrderRecord> | null) {
    const orderId = Number(record?.id || 0)
    if (!orderId) {
      return
    }

    await retryAdminOrder(orderId)
    ElMessage.success('已触发重试')
    await fetchData()
    await refreshDetail(orderId)
  }

  async function removeOrder(record?: OrderTableRow | Partial<OrderRecord> | null) {
    const orderId = Number(record?.id || 0)
    if (!orderId) {
      return
    }

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
      stopPolling()
    }

    await fetchData()
  }

  function getMoreActions(row: OrderTableRow): ButtonMoreItem[] {
    return [
      {
        key: 'retry',
        label: '重试开通',
        icon: 'ri:restart-line',
        disabled: !canRetry(row)
      },
      {
        key: 'delete',
        label: '删除订单',
        icon: 'ri:delete-bin-5-line',
        color: '#f56c6c',
        auth: 'order.delete'
      }
    ]
  }

  function handleMoreAction(item: ButtonMoreItem, row: OrderTableRow) {
    switch (item.key) {
      case 'retry':
        retry(row)
        break
      case 'delete':
        removeOrder(row)
        break
    }
  }

  function escapeCsvCell(value: string | number | null | undefined) {
    const text = String(value ?? '')
    return `"${text.replace(/"/g, '""')}"`
  }

  function exportCsv() {
    const rows = tableData.value.map((item) =>
      [escapeCsvCell(item.id), escapeCsvCell(item.status), escapeCsvCell(item.total_amount)].join(
        ','
      )
    )

    const content = ['id,status,total_amount', ...rows].join('\n')
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
    align-items: center;
    justify-content: flex-end;
    gap: 4px;
  }

  .dialog-footer {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
  }

  @media (max-width: 768px) {
    .table-actions {
      flex-wrap: wrap;
      justify-content: flex-start;
    }
  }
</style>
