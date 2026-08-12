<template>
  <div class="orders-page art-full-height">
    <ElSegmented
      v-model="selectedStatus"
      :options="statusOptions"
      class="status-filter"
      @change="handleStatusChange"
    />

    <ElCard class="art-table-card">
      <ArtTableHeader :loading="loading" @refresh="refreshData">
        <template #left>
          <div class="table-summary">
            <span class="table-summary__title">我的订单</span>
            <span class="table-summary__count">共 {{ pagination.total }} 单</span>
          </div>
        </template>
      </ArtTableHeader>

      <ArtTable
        class="orders-table"
        row-key="id"
        :loading="loading"
        :data="data"
        :columns="columns"
        :pagination="pagination"
        empty-text="暂无订单"
        @pagination:size-change="handleSizeChange"
        @pagination:current-change="handleCurrentChange"
      >
        <template #order="{ row }">
          <div class="order-cell">
            <ArtSvgIcon icon="ri:file-list-3-line" class="order-cell__icon" />
            <div class="order-cell__content">
              <span class="order-cell__number">{{ row.orderNo || `#${row.id}` }}</span>
              <span class="order-cell__id">ID: {{ row.id || '-' }}</span>
            </div>
          </div>
        </template>

        <template #status="{ row }">
          <OrderStatusBadge :status="row.status" />
        </template>

        <template #amount="{ row }">
          <span class="amount">{{ formatMoney(row.totalAmount, row.currency) }}</span>
        </template>

        <template #operation="{ row }">
          <div class="table-actions">
            <ElTooltip content="查看详情" placement="top">
              <ArtButtonTable type="view" @click="goDetail(row)" />
            </ElTooltip>
            <ElTooltip v-if="canCancel(row)" content="撤销订单" placement="top">
              <ArtButtonTable type="delete" @click="cancelItem(row)" />
            </ElTooltip>
          </div>
        </template>
      </ArtTable>

      <div v-loading="loading" class="mobile-orders">
        <ElEmpty v-if="!loading && data.length === 0" description="暂无订单" />

        <article v-for="row in data" :key="row.id" class="mobile-order">
          <header class="mobile-order__header">
            <div class="mobile-order__title">
              <ArtSvgIcon icon="ri:file-list-3-line" class="mobile-order__icon" />
              <span>{{ row.orderNo || `#${row.id}` }}</span>
            </div>
            <OrderStatusBadge :status="row.status" />
          </header>

          <dl class="mobile-order__details">
            <div class="mobile-order__detail">
              <dt>订单 ID</dt>
              <dd>{{ row.id || '-' }}</dd>
            </div>
            <div class="mobile-order__detail">
              <dt>金额</dt>
              <dd class="mobile-order__amount">
                {{ formatMoney(row.totalAmount, row.currency) }}
              </dd>
            </div>
            <div class="mobile-order__detail">
              <dt>创建时间</dt>
              <dd>{{ row.createdAt }}</dd>
            </div>
          </dl>

          <footer class="mobile-order__actions">
            <ElButton class="mobile-order__action" size="small" @click="goDetail(row)">
              <ArtSvgIcon icon="ri:eye-line" />
              详情
            </ElButton>
            <ElButton
              v-if="canCancel(row)"
              class="mobile-order__action"
              size="small"
              type="danger"
              plain
              @click="cancelItem(row)"
            >
              <ArtSvgIcon icon="ri:close-circle-line" />
              撤销
            </ElButton>
          </footer>
        </article>
      </div>
    </ElCard>
  </div>
</template>

<script setup lang="ts">
  import { useTable } from '@/hooks/core/useTable'
  import { cancelOrder, listOrders } from '@/services/user'
  import type { Order } from '@/services/types'
  import OrderStatusBadge from '@/components/business/order-status-badge/index.vue'

  defineOptions({ name: 'ConsoleOrders' })

  interface OrderSearchParams {
    current: number
    size: number
    status?: string
  }

  interface OrderRow {
    id: number | string
    orderNo: string
    status: string
    totalAmount: number
    currency: string
    createdAt: string
  }

  const router = useRouter()
  const selectedStatus = ref('')
  const statusOptions = [
    { label: '全部', value: '' },
    { label: '待支付', value: 'pending_payment' },
    { label: '待审核', value: 'pending_review' },
    { label: '开通中', value: 'provisioning' },
    { label: '已完成', value: 'active' }
  ]

  const fetchOrderPage = async (
    { current, size, status }: OrderSearchParams,
    signal?: AbortSignal
  ) => {
    const response = await listOrders(
      {
        limit: size,
        offset: (current - 1) * size,
        ...(status ? { status } : {})
      },
      signal
    )
    const items = response.data?.items ?? []
    return {
      records: items.map((row) => normalizeOrder(row as Order & Record<string, unknown>)),
      current,
      size,
      total: response.data?.total ?? items.length
    }
  }

  const normalizeOrder = (row: Order & Record<string, unknown>): OrderRow => ({
    id: (row.id ?? row.ID ?? '') as number | string,
    orderNo: String(row.order_no ?? row.OrderNo ?? ''),
    status: String(row.status ?? row.Status ?? ''),
    totalAmount: Number(row.total_amount ?? row.TotalAmount ?? 0),
    currency: String(row.currency ?? row.Currency ?? 'CNY'),
    createdAt: String(row.created_at ?? row.CreatedAt ?? '-')
  })

  const {
    columns,
    data,
    loading,
    pagination,
    getData,
    searchParams,
    handleSizeChange,
    handleCurrentChange,
    refreshData,
    refreshUpdate
  } = useTable<typeof fetchOrderPage>({
    core: {
      apiFn: fetchOrderPage,
      apiParams: { current: 1, size: 10, status: undefined },
      columnsFactory: () => [
        { prop: 'order', label: '订单', minWidth: 220, useSlot: true },
        { prop: 'status', label: '状态', width: 130, useSlot: true },
        { prop: 'amount', label: '金额', width: 140, useSlot: true },
        { prop: 'createdAt', label: '创建时间', minWidth: 180 },
        {
          prop: 'operation',
          label: '操作',
          width: 112,
          fixed: 'right',
          align: 'right',
          useSlot: true
        }
      ]
    }
  })

  const handleStatusChange = (status: string | number | boolean) => {
    searchParams.status = String(status || '') || undefined
    getData()
  }

  const formatMoney = (amount: number, currency: string) => {
    const normalizedCurrency = currency || 'CNY'
    try {
      return new Intl.NumberFormat('zh-CN', {
        style: 'currency',
        currency: normalizedCurrency,
        minimumFractionDigits: 2
      }).format(Number(amount || 0))
    } catch {
      return `${normalizedCurrency} ${Number(amount || 0).toFixed(2)}`
    }
  }

  const goDetail = (row: OrderRow) => {
    if (row.id) router.push({ name: 'ConsoleOrderDetail', params: { id: row.id } })
  }

  const canCancel = (row: OrderRow) => ['pending_payment', 'pending_review'].includes(row.status)

  const cancelItem = async (row: OrderRow) => {
    if (!row.id) return

    try {
      await ElMessageBox.confirm('撤销后订单将变为已取消，无法继续支付。确认撤销吗？', '撤销订单', {
        confirmButtonText: '确认撤销',
        cancelButtonText: '暂不撤销',
        type: 'warning'
      })
      await cancelOrder(row.id)
      ElMessage.success('订单已撤销')
      await refreshUpdate()
    } catch (error) {
      if (error !== 'cancel' && error !== 'close') {
        ElMessage.error('撤销订单失败，请稍后重试')
      }
    }
  }
</script>

<style lang="scss" scoped>
  .status-filter {
    margin-bottom: 12px;
  }

  .table-summary {
    display: flex;
    gap: 10px;
    align-items: baseline;

    &__title {
      font-size: 15px;
      font-weight: 600;
      color: var(--art-gray-900);
    }

    &__count {
      font-size: 13px;
      color: var(--art-gray-600);
    }
  }

  .order-cell {
    display: flex;
    gap: 10px;
    align-items: center;

    &__icon {
      flex: 0 0 auto;
      font-size: 20px;
      color: var(--theme-color);
    }

    &__content {
      display: flex;
      flex-direction: column;
      min-width: 0;
    }

    &__number {
      overflow: hidden;
      font-weight: 600;
      color: var(--art-gray-900);
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    &__id {
      font-size: 12px;
      color: var(--art-gray-600);
    }
  }

  .amount {
    font-weight: 700;
    color: var(--theme-color);
  }

  .table-actions {
    display: flex;
    justify-content: flex-end;
  }

  .mobile-orders {
    display: none;
  }

  @media (width <= 768px) {
    .table-summary {
      flex-direction: column;
      gap: 2px;
      align-items: flex-start;
    }

    .orders-table {
      display: none;
    }

    .mobile-orders {
      display: flex;
      flex: 1;
      flex-direction: column;
      gap: 12px;
      min-height: 0;
      padding: 4px;
      overflow-y: auto;
    }

    .mobile-order {
      flex: 0 0 auto;
      overflow: hidden;
      color: var(--art-gray-700);
      background: var(--default-box-color);
      border: 1px solid var(--default-border);
      border-radius: calc(var(--custom-radius) / 2 + 2px);

      &__header {
        display: flex;
        gap: 12px;
        align-items: center;
        justify-content: space-between;
        padding: 12px 14px;
        background: var(--default-bg-color);
        border-bottom: 1px solid var(--default-border);
      }

      &__title {
        display: flex;
        gap: 8px;
        align-items: center;
        min-width: 0;
        overflow: hidden;
        font-weight: 600;
        color: var(--art-gray-900);
        text-overflow: ellipsis;
        white-space: nowrap;
      }

      &__icon {
        flex: 0 0 auto;
        color: var(--theme-color);
      }

      &__details {
        display: flex;
        flex-direction: column;
        gap: 10px;
        padding: 14px;
        margin: 0;
      }

      &__detail {
        display: grid;
        grid-template-columns: minmax(64px, auto) minmax(0, 1fr);
        gap: 16px;
        align-items: start;

        dt {
          color: var(--art-gray-600);
        }

        dd {
          min-width: 0;
          margin: 0;
          color: var(--art-gray-900);
          text-align: right;
          overflow-wrap: anywhere;
        }
      }

      &__amount {
        font-weight: 700;
        color: var(--theme-color);
      }

      &__actions {
        display: flex;
        gap: 8px;
        padding: 12px 14px;
        border-top: 1px solid var(--default-border);
      }

      &__action {
        flex: 1;
      }
    }
  }
</style>
