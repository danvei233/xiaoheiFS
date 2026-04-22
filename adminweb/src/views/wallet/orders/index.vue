<template>
  <div class="art-full-height">
    <WalletOrderSearch
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
        <template #type="{ row }">
          <ElTag :type="getTypeTagType(row.type)">
            {{ getTypeText(row.type) }}
          </ElTag>
        </template>

        <template #status="{ row }">
          <ElTag :type="getStatusTagType(row.status)">
            {{ getStatusText(row.status) }}
          </ElTag>
        </template>

        <template #amount="{ row }">
          <span :class="['amount-text', row.type === 'withdraw' ? 'is-negative' : 'is-positive']">
            {{ row.type === 'withdraw' ? '-' : '+' }}￥{{ formatAmount(row.amount) }}
          </span>
        </template>

        <template #created_at="{ row }">
          {{ formatDateTime(row.created_at) }}
        </template>

        <template #operation="{ row }">
          <div class="table-actions">
            <ElButton
              v-if="row.status === 'pending_review' && canApprove"
              link
              type="primary"
              @click="handleApprove(row)"
            >
              通过
            </ElButton>

            <ElButton
              v-if="row.status === 'pending_review' && canReject"
              link
              type="danger"
              @click="handleReject(row)"
            >
              驳回
            </ElButton>

            <span
              v-if="row.status !== 'pending_review' || (!canApprove && !canReject)"
              class="muted"
            >
              -
            </span>
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
          <ElButton type="primary" :loading="rejecting" @click="confirmReject">确认驳回</ElButton>
        </div>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import type { WalletOrderRecord } from '@/api/admin'
  import {
    approveAdminWalletOrder,
    fetchAdminWalletOrders,
    hasAdminPermission,
    rejectAdminWalletOrder
  } from '@/api/admin'
  import { useTableColumns } from '@/hooks/core/useTableColumns'
  import { useUserStore } from '@/store/modules/user'
  import { INPUT_LIMITS } from '@/utils/constants'
  import { ElMessage } from 'element-plus'
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

  const quickStatusTabs = [
    { label: '待审核', value: 'pending_review' },
    { label: '已通过', value: 'approved' },
    { label: '已驳回', value: 'rejected' }
  ] as const

  const showSearchBar = ref(true)
  const loading = ref(false)
  const rejecting = ref(false)
  const rejectVisible = ref(false)

  const searchForm = ref<WalletOrderSearchForm>(createDefaultSearchForm())
  const tableData = ref<WalletOrderTableRow[]>([])
  const rejectReason = ref('')
  const currentOrder = ref<WalletOrderTableRow | null>(null)

  const pagination = reactive({
    current: 1,
    size: 20,
    total: 0
  })

  const { columnChecks, columns } = useTableColumns<WalletOrderTableRow>(() => [
    { prop: 'id', label: 'ID', width: 90 },
    { prop: 'user_id', label: '用户 ID', width: 100 },
    { prop: 'type', label: '类型', width: 110, useSlot: true },
    { prop: 'amount', label: '金额', minWidth: 140, useSlot: true },
    { prop: 'note', label: '备注', minWidth: 220, showOverflowTooltip: true },
    { prop: 'status', label: '状态', width: 120, useSlot: true },
    { prop: 'created_at', label: '创建时间', minWidth: 180, useSlot: true },
    { prop: 'operation', label: '操作', width: 140, fixed: 'right', useSlot: true }
  ])

  const userStore = useUserStore()
  const { info } = storeToRefs(userStore)

  const canApprove = computed(() =>
    hasAdminPermission(info.value?.buttons, ['wallet_order.approve'])
  )

  const canReject = computed(() => hasAdminPermission(info.value?.buttons, ['wallet_order.reject']))

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

  function createDefaultSearchForm(): WalletOrderSearchForm {
    return {
      status: undefined,
      user_id: ''
    }
  }

  function normalizeNullableNumber(value: unknown): number | null {
    if (value === '' || value === null || value === undefined) {
      return null
    }

    const parsed = Number(value)
    return Number.isFinite(parsed) ? parsed : null
  }

  function normalizeRecord(value: unknown): Record<string, unknown> {
    if (!value || typeof value !== 'object' || Array.isArray(value)) {
      return {}
    }

    return value as Record<string, unknown>
  }

  function normalizeOrder(item?: WalletOrderRecord): WalletOrderTableRow {
    return {
      id: normalizeNullableNumber(item?.id),
      user_id: normalizeNullableNumber(item?.user_id),
      type: String(item?.type || ''),
      amount: Number(item?.amount || 0),
      currency: String(item?.currency || 'CNY'),
      status: String(item?.status || ''),
      note: String(item?.note || ''),
      meta: normalizeRecord(item?.meta),
      reviewed_by: normalizeNullableNumber(item?.reviewed_by),
      review_reason: String(item?.review_reason || ''),
      created_at: String(item?.created_at || ''),
      updated_at: String(item?.updated_at || '')
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

  function getTypeText(type?: string) {
    switch (type) {
      case 'recharge':
        return '充值'
      case 'withdraw':
        return '提现'
      case 'refund':
        return '退款'
      default:
        return type || '-'
    }
  }

  function getTypeTagType(type?: string) {
    switch (type) {
      case 'recharge':
        return 'success' as const
      case 'withdraw':
        return 'warning' as const
      case 'refund':
        return 'info' as const
      default:
        return 'info' as const
    }
  }

  function getStatusText(status?: string) {
    switch (status) {
      case 'pending_review':
        return '待审核'
      case 'approved':
        return '已通过'
      case 'rejected':
        return '已驳回'
      default:
        return status || '-'
    }
  }

  function getStatusTagType(status?: string) {
    switch (status) {
      case 'pending_review':
        return 'warning' as const
      case 'approved':
        return 'success' as const
      case 'rejected':
        return 'danger' as const
      default:
        return 'info' as const
    }
  }

  async function fetchData() {
    loading.value = true

    try {
      const payload = await fetchAdminWalletOrders({
        limit: pagination.size,
        offset: (pagination.current - 1) * pagination.size,
        status: searchForm.value.status || undefined,
        user_id: normalizeUserId(searchForm.value.user_id)
      })

      tableData.value = (payload.items || []).map((item) => normalizeOrder(item))
      pagination.total = typeof payload.total === 'number' ? payload.total : tableData.value.length
    } finally {
      loading.value = false
    }
  }

  function handleSearch(params: WalletOrderSearchForm) {
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

  async function handleApprove(record: WalletOrderTableRow) {
    const orderId = Number(record.id || 0)
    if (!orderId) {
      return
    }

    await approveAdminWalletOrder(orderId)
    ElMessage.success('操作成功')
    await fetchData()
  }

  function handleReject(record: WalletOrderTableRow) {
    currentOrder.value = record
    rejectReason.value = ''
    rejectVisible.value = true
  }

  async function confirmReject() {
    const orderId = Number(currentOrder.value?.id || 0)
    if (!orderId) {
      return
    }

    if (String(rejectReason.value || '').length > INPUT_LIMITS.REVIEW_REASON) {
      ElMessage.error(`驳回原因长度不能超过 ${INPUT_LIMITS.REVIEW_REASON} 个字符`)
      return
    }

    rejecting.value = true

    try {
      await rejectAdminWalletOrder(orderId, { reason: rejectReason.value })
      ElMessage.success('操作成功')
      rejectVisible.value = false
      await fetchData()
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
    color: #16a34a;
  }

  .is-negative {
    color: #dc2626;
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

  .muted {
    color: var(--el-text-color-secondary);
  }
</style>
