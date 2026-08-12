<template>
  <div class="wallet-orders-page art-full-height">
    <WalletOrderSearch
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
          <ElRadioGroup v-model="quickStatus" class="quick-status-group">
            <ElRadioButton label="all">全部</ElRadioButton>
            <ElRadioButton v-for="item in quickStatusTabs" :key="item.value" :label="item.value">
              {{ item.label }}
            </ElRadioButton>
          </ElRadioGroup>
        </template>
      </ArtTableHeader>

      <ArtTable
        row-key="id"
        :loading="loading"
        :data="data"
        :columns="columns"
        :pagination="pagination"
        @pagination:size-change="handleSizeChange"
        @pagination:current-change="handleCurrentChange"
      >
        <template #type="{ row }">
          <ElTag :type="getTypeTagType(row.type)">{{ getTypeText(row.type) }}</ElTag>
        </template>
        <template #status="{ row }">
          <ElTag :type="getStatusTagType(row.status)">{{ getStatusText(row.status) }}</ElTag>
        </template>
        <template #amount="{ row }">
          <span :class="['amount-text', row.type === 'withdraw' ? 'is-negative' : 'is-positive']">
            {{ row.type === 'withdraw' ? '-' : '+' }}￥{{ formatAmount(row.amount) }}
          </span>
        </template>
        <template #created_at="{ row }">{{ formatDateTime(row.created_at) }}</template>
        <template #operation="{ row }">
          <div class="table-actions">
            <ElButton
              v-if="row.status === 'pending_review'"
              v-auth="'wallet_order.approve'"
              v-ripple
              link
              type="primary"
              @click="handleApprove(row)"
            >
              通过
            </ElButton>
            <ElButton
              v-if="row.status === 'pending_review'"
              v-auth="'wallet_order.reject'"
              v-ripple
              link
              type="danger"
              @click="handleReject(row)"
            >
              拒绝
            </ElButton>
            <span v-if="row.status !== 'pending_review'" class="muted">-</span>
          </div>
        </template>
      </ArtTable>
    </ElCard>

    <ElDialog v-model="rejectVisible" title="拒绝订单" width="460px" destroy-on-close align-center>
      <ElForm label-position="top">
        <ElFormItem label="拒绝原因">
          <ElInput
            v-model="rejectReason"
            type="textarea"
            :rows="4"
            placeholder="请输入拒绝原因"
            :maxlength="INPUT_LIMITS.REVIEW_REASON"
            show-word-limit
          />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <div class="dialog-footer">
          <ElButton v-ripple @click="rejectVisible = false">取消</ElButton>
          <ElButton v-ripple type="primary" :loading="rejecting" @click="confirmReject"
            >确认拒绝</ElButton
          >
        </div>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import { useTable } from '@/hooks/core/useTable'
  import {
    approveAdminWalletOrder,
    listAdminWalletOrders,
    rejectAdminWalletOrder
  } from '@/services/admin'
  import type { WalletOrder } from '@/services/types'
  import { INPUT_LIMITS } from '@/constants/inputLimits'
  import WalletOrderSearch from './modules/wallet-order-search.vue'

  defineOptions({ name: 'WalletOrders' })

  interface WalletOrderSearchForm {
    status?: string
    user_id: string
  }

  interface WalletOrderTableRow {
    id: number | null
    user_id: number | null
    type: string
    amount: number
    currency: string
    status: string
    note: string
    meta: Record<string, unknown>
    reviewed_by: number | null
    review_reason: string
    created_at: string
    updated_at: string
  }

  interface WalletOrderRecordLike extends WalletOrder {
    ID?: unknown
    UserID?: unknown
    Type?: unknown
    Amount?: unknown
    Currency?: unknown
    Status?: unknown
    Note?: unknown
    Meta?: unknown
    ReviewedBy?: unknown
    ReviewReason?: unknown
    CreatedAt?: unknown
    UpdatedAt?: unknown
  }

  interface WalletOrderTableParams extends Api.Common.CommonSearchParams {
    status?: string
    user_id?: string
  }

  const quickStatusTabs = [
    { label: '待审核', value: 'pending_review' },
    { label: '已通过', value: 'approved' },
    { label: '已拒绝', value: 'rejected' }
  ] as const

  const showSearchBar = ref(true)
  const rejecting = ref(false)
  const rejectVisible = ref(false)
  const rejectReason = ref('')
  const currentOrder = ref<WalletOrderTableRow | null>(null)
  const searchForm = ref<WalletOrderSearchForm>(createDefaultSearchForm())

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

  async function fetchWalletOrders(
    params: WalletOrderTableParams,
    signal?: AbortSignal
  ): Promise<Api.Common.PaginatedResponse<WalletOrderTableRow>> {
    const response = await listAdminWalletOrders(
      {
        limit: params.size,
        offset: (params.current - 1) * params.size,
        status: params.status || undefined,
        user_id: normalizeUserId(params.user_id)
      },
      signal
    )
    const payload = response.data || { items: [], total: 0 }
    const records = (payload.items || []).map(normalizeOrder)
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
      apiFn: fetchWalletOrders,
      apiParams: { current: 1, size: 20, ...createDefaultSearchForm() },
      columnsFactory: () => [
        { prop: 'id', label: 'ID', width: 90 },
        { prop: 'user_id', label: '用户 ID', width: 100 },
        { prop: 'type', label: '类型', width: 110, useSlot: true },
        { prop: 'amount', label: '金额', minWidth: 140, useSlot: true },
        { prop: 'note', label: '备注', minWidth: 220, showOverflowTooltip: true },
        { prop: 'status', label: '状态', width: 120, useSlot: true },
        { prop: 'created_at', label: '创建时间', minWidth: 180, useSlot: true },
        { prop: 'operation', label: '操作', width: 140, fixed: 'right', useSlot: true }
      ]
    },
    hooks: {
      resetFormCallback: () => {
        searchForm.value = createDefaultSearchForm()
      }
    }
  })

  function createDefaultSearchForm(): WalletOrderSearchForm {
    return { status: undefined, user_id: '' }
  }

  function normalizeNullableNumber(value: unknown) {
    if (value === '' || value === null || value === undefined) return null
    const parsed = Number(value)
    return Number.isFinite(parsed) ? parsed : null
  }

  function normalizeRecord(value: unknown): Record<string, unknown> {
    return value && typeof value === 'object' && !Array.isArray(value)
      ? (value as Record<string, unknown>)
      : {}
  }

  function normalizeOrder(item: WalletOrderRecordLike = {}): WalletOrderTableRow {
    return {
      id: normalizeNullableNumber(item.id ?? item.ID),
      user_id: normalizeNullableNumber(item.user_id ?? item.UserID),
      type: String(item.type ?? item.Type ?? ''),
      amount: Number(item.amount ?? item.Amount ?? 0),
      currency: String(item.currency ?? item.Currency ?? 'CNY'),
      status: String(item.status ?? item.Status ?? ''),
      note: String(item.note ?? item.Note ?? ''),
      meta: normalizeRecord(item.meta ?? item.Meta),
      reviewed_by: normalizeNullableNumber(item.reviewed_by ?? item.ReviewedBy),
      review_reason: String(item.review_reason ?? item.ReviewReason ?? ''),
      created_at: String(item.created_at ?? item.CreatedAt ?? ''),
      updated_at: String(item.updated_at ?? item.UpdatedAt ?? '')
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
  function getTypeText(type?: string) {
    return (
      ({ recharge: '充值', withdraw: '提现', refund: '退款' } as Record<string, string>)[
        String(type || '')
      ] ||
      type ||
      '-'
    )
  }
  function getTypeTagType(type?: string) {
    return (
      (
        { recharge: 'success', withdraw: 'warning', refund: 'info' } as Record<
          string,
          'info' | 'warning' | 'success'
        >
      )[String(type || '')] || 'info'
    )
  }
  function getStatusText(status?: string) {
    return (
      (
        { pending_review: '待审核', approved: '已通过', rejected: '已拒绝' } as Record<
          string,
          string
        >
      )[String(status || '')] ||
      status ||
      '-'
    )
  }
  function getStatusTagType(status?: string) {
    return (
      (
        { pending_review: 'warning', approved: 'success', rejected: 'danger' } as Record<
          string,
          'info' | 'warning' | 'success' | 'danger'
        >
      )[String(status || '')] || 'info'
    )
  }

  function handleSearch(params: WalletOrderSearchForm) {
    searchForm.value = { ...searchForm.value, ...params }
    Object.assign(searchParams, {
      status: params.status || undefined,
      user_id: params.user_id || undefined
    })
    getData()
  }

  async function handleApprove(record: WalletOrderTableRow) {
    const orderId = Number(record.id || 0)
    if (!orderId) return
    await approveAdminWalletOrder(orderId)
    ElMessage.success('操作成功')
    await refreshData()
  }

  function handleReject(record: WalletOrderTableRow) {
    currentOrder.value = record
    rejectReason.value = ''
    rejectVisible.value = true
  }

  async function confirmReject() {
    const orderId = Number(currentOrder.value?.id || 0)
    if (!orderId) return
    if (rejectReason.value.length > INPUT_LIMITS.REVIEW_REASON) {
      ElMessage.error(`拒绝原因长度不能超过 ${INPUT_LIMITS.REVIEW_REASON} 个字符`)
      return
    }
    rejecting.value = true
    try {
      await rejectAdminWalletOrder(orderId, { reason: rejectReason.value })
      ElMessage.success('操作成功')
      rejectVisible.value = false
      await refreshData()
    } finally {
      rejecting.value = false
    }
  }
</script>

<style scoped lang="scss">
  .quick-status-group {
    flex-wrap: wrap;
  }

  .amount-text {
    font-weight: 600;
  }

  .is-positive {
    color: var(--el-color-success);
  }

  .is-negative {
    color: var(--el-color-danger);
  }

  .table-actions {
    display: flex;
    gap: 4px;
    align-items: center;
    justify-content: flex-end;
  }

  .dialog-footer {
    display: flex;
    gap: 12px;
    justify-content: flex-end;
  }

  .muted {
    color: var(--el-text-color-secondary);
  }
</style>
