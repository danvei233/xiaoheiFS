<template>
  <div class="billing-page">
    <ElCard class="balance-card art-card-xs" shadow="never">
      <div class="balance-main">
        <div class="balance-icon"><ArtSvgIcon icon="ri:wallet-3-line" /></div>
        <div>
          <span>当前余额</span>
          <strong>{{ formatMoney(wallet.balance, wallet.currency) }}</strong>
        </div>
      </div>
      <div class="balance-actions">
        <ElButton type="primary" :icon="Plus" @click="openRechargeDialog">充值</ElButton>
        <ElButton :icon="Upload" @click="withdrawDialogVisible = true">提现</ElButton>
        <ElTooltip content="刷新钱包" placement="top">
          <ElButton
            circle
            :icon="Refresh"
            :loading="overviewLoading"
            aria-label="刷新钱包"
            @click="refreshAll"
          />
        </ElTooltip>
      </div>
    </ElCard>

    <div class="stats-grid">
      <ElCard class="stat-card art-card-xs" shadow="never">
        <span>本月充值</span>
        <strong>{{ formatMoney(monthStats.recharge, wallet.currency) }}</strong>
      </ElCard>
      <ElCard class="stat-card art-card-xs" shadow="never">
        <span>本月提现</span>
        <strong>{{ formatMoney(monthStats.withdraw, wallet.currency) }}</strong>
      </ElCard>
      <ElCard class="stat-card art-card-xs" shadow="never">
        <span>待审核</span>
        <strong>{{ monthStats.pending }}</strong>
      </ElCard>
      <ElCard class="stat-card art-card-xs" shadow="never">
        <span>总交易</span>
        <strong>{{ monthStats.total }}</strong>
      </ElCard>
    </div>

    <div class="billing-filters">
      <label>
        <span>状态</span>
        <ElSegmented v-model="selectedStatus" :options="statusOptions" @change="applyFilters" />
      </label>
      <label>
        <span>类型</span>
        <ElSegmented v-model="selectedType" :options="typeOptions" @change="applyFilters" />
      </label>
    </div>

    <ElCard class="billing-table-card art-table-card">
      <ArtTableHeader :loading="loading" @refresh="refreshAll">
        <template #left><span class="table-title">钱包订单</span></template>
      </ArtTableHeader>
      <ArtTable
        row-key="id"
        :loading="loading"
        :data="data"
        :columns="columns"
        :pagination="pagination"
        empty-text="暂无钱包订单"
        @pagination:size-change="handleSizeChange"
        @pagination:current-change="handleCurrentChange"
      >
        <template #type="{ row }">
          <span class="type-cell">
            <ArtSvgIcon :icon="typeIcon(row.type)" />
            {{ typeLabel(row.type) }}
          </span>
        </template>
        <template #amount="{ row }">
          <strong :class="['amount', { 'amount--income': row.type === 'recharge' }]">
            {{ row.type === 'recharge' ? '+' : '-'
            }}{{ formatMoney(row.amount, row.currency || wallet.currency) }}
          </strong>
        </template>
        <template #status="{ row }">
          <ElTag :type="statusTagType(row)">{{ statusLabel(row) }}</ElTag>
        </template>
        <template #operation="{ row }">
          <div v-if="canContinuePay(row) || canCancel(row)" class="table-actions">
            <ElButton v-if="canContinuePay(row)" link type="primary" @click="continuePay(row)"
              >继续支付</ElButton
            >
            <ElButton v-if="canCancel(row)" link type="danger" @click="cancelPendingOrder(row)"
              >取消</ElButton
            >
          </div>
          <span v-else>-</span>
        </template>
      </ArtTable>
    </ElCard>

    <ElDialog
      v-model="rechargeDialogVisible"
      title="账户充值"
      width="min(460px, calc(100vw - 24px))"
      @closed="resetRechargeForm"
    >
      <ElAlert title="充值订单审核通过后到账" type="info" show-icon :closable="false" />
      <ElForm ref="rechargeFormRef" :model="recharge" label-position="top" class="dialog-form">
        <ElFormItem
          label="支付方式"
          prop="method"
          :rules="[{ required: true, message: '请选择支付方式', trigger: 'change' }]"
        >
          <ElSelect v-model="recharge.method" class="full-width">
            <ElOption
              v-for="provider in rechargeMethods"
              :key="provider.key"
              :label="provider.name || provider.key"
              :value="provider.key"
            />
          </ElSelect>
        </ElFormItem>
        <ElFormItem
          label="充值金额"
          prop="amount"
          :rules="[{ required: true, message: '请输入充值金额', trigger: 'blur' }]"
        >
          <ElInputNumber
            v-model="recharge.amount"
            :min="0.01"
            :precision="2"
            :step="100"
            class="full-width"
          />
        </ElFormItem>
        <ElFormItem label="备注">
          <ElInput
            v-model="recharge.note"
            type="textarea"
            :rows="3"
            :maxlength="200"
            show-word-limit
          />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="rechargeDialogVisible = false">取消</ElButton>
        <ElButton type="primary" :loading="submitting" @click="submitRecharge">提交</ElButton>
      </template>
    </ElDialog>

    <ElDialog
      v-model="withdrawDialogVisible"
      title="申请提现"
      width="min(460px, calc(100vw - 24px))"
      @closed="resetWithdrawForm"
    >
      <ElAlert title="提现订单审核通过后打款" type="warning" show-icon :closable="false" />
      <div class="available-balance">
        <span>可用余额</span>
        <strong>{{ formatMoney(wallet.balance, wallet.currency) }}</strong>
      </div>
      <ElForm ref="withdrawFormRef" :model="withdraw" label-position="top" class="dialog-form">
        <ElFormItem
          label="提现金额"
          prop="amount"
          :rules="[{ required: true, message: '请输入提现金额', trigger: 'blur' }]"
        >
          <ElInputNumber
            v-model="withdraw.amount"
            :min="0.01"
            :max="wallet.balance"
            :precision="2"
            :step="100"
            class="full-width"
          />
        </ElFormItem>
        <ElFormItem
          label="收款方式"
          prop="note"
          :rules="[{ required: true, message: '请填写收款方式', trigger: 'blur' }]"
        >
          <ElInput
            v-model="withdraw.note"
            type="textarea"
            :rows="3"
            :maxlength="200"
            show-word-limit
            placeholder="微信、支付宝或银行卡信息"
          />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="withdrawDialogVisible = false">取消</ElButton>
        <ElButton type="primary" :loading="submitting" @click="submitWithdraw">提交</ElButton>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import { Plus, Refresh, Upload } from '@element-plus/icons-vue'
  import type { FormInstance } from 'element-plus'
  import { useTable } from '@/hooks/core/useTable'
  import {
    cancelWalletOrder,
    createWalletRecharge,
    createWalletWithdraw,
    getWallet,
    getWalletOrderSummary,
    listPaymentProviders,
    listWalletOrders,
    payWalletOrder
  } from '@/services/user'
  import type { PaymentProvider, WalletOrder } from '@/services/types'
  import { normalizeWallet } from '@/utils/wallet'
  import { openPaymentUrl } from '@/utils/paymentUrl'

  defineOptions({ name: 'ConsoleBilling' })

  interface WalletModel {
    balance: number
    currency: string
  }

  interface WalletOrderRow {
    id: number | string
    type: string
    amount: number
    currency: string
    status: string
    note: string
    meta: Record<string, unknown>
    createdAt: string
  }

  interface WalletSearchParams {
    current: number
    size: number
    status?: string
    type?: string
  }

  type CompatibleRecord = Record<string, any>

  const overviewLoading = ref(false)
  const submitting = ref(false)
  const rechargeDialogVisible = ref(false)
  const withdrawDialogVisible = ref(false)
  const rechargeFormRef = ref<FormInstance>()
  const withdrawFormRef = ref<FormInstance>()
  const rechargeMethods = ref<PaymentProvider[]>([])
  const wallet = ref<WalletModel>({ balance: 0, currency: 'CNY' })
  const recharge = reactive<{ method: string; amount: number | null; note: string }>({
    method: 'approval',
    amount: null,
    note: ''
  })
  const withdraw = reactive<{ amount: number | null; note: string }>({ amount: null, note: '' })
  const selectedStatus = ref('')
  const selectedType = ref('')
  const statusOptions = [
    { label: '全部', value: '' },
    { label: '待处理', value: 'pending_review' },
    { label: '已通过', value: 'approved' },
    { label: '已拒绝', value: 'rejected' }
  ]
  const typeOptions = [
    { label: '全部', value: '' },
    { label: '充值', value: 'recharge' },
    { label: '提现', value: 'withdraw' }
  ]

  const normalizeOrder = (item: WalletOrder & CompatibleRecord): WalletOrderRow => ({
    id: item.id ?? item.ID ?? '',
    type: String(item.type ?? item.Type ?? '')
      .trim()
      .toLowerCase(),
    amount: Number(item.amount ?? item.Amount ?? 0),
    currency: String(item.currency ?? item.Currency ?? 'CNY'),
    status: String(item.status ?? item.Status ?? '')
      .trim()
      .toLowerCase(),
    note: String(item.note ?? item.Note ?? ''),
    meta: (item.meta ?? item.Meta ?? {}) as Record<string, unknown>,
    createdAt: String(item.created_at ?? item.CreatedAt ?? '')
  })

  const fetchWalletOrderPage = async (
    { current, size, status, type }: WalletSearchParams,
    signal?: AbortSignal
  ) => {
    // cross-module residual：后端 /wallet/orders 现支持 status/type 服务端过滤
    // 与真实分页；前端按页拉取并展示后端 total，不再整批 500 条后本地切片
    // 伪装完整结果（P3-42 / m059 F4 契约缺口本轮闭环）。
    const response = await listWalletOrders(
      {
        limit: size,
        offset: (current - 1) * size,
        status: status || undefined,
        type: type || undefined
      },
      signal
    )
    const orders = (response.data?.items ?? []).map((item) =>
      normalizeOrder(item as WalletOrder & CompatibleRecord)
    )
    return {
      records: orders,
      current,
      size,
      total: Number(response.data?.total ?? orders.length)
    }
  }

  const loadOverview = async () => {
    try {
      const response = await getWalletOrderSummary()
      const summary = response.data
      monthStats.value = {
        recharge: Number(summary.recharge || 0),
        withdraw: Number(summary.withdraw || 0),
        pending: Number(summary.pending || 0),
        total: Number(summary.total || 0)
      }
    } catch {
      // 概览失败不阻塞明细表
    }
  }

  const {
    columns,
    data,
    loading,
    pagination,
    getData,
    searchParams,
    handleSizeChange,
    handleCurrentChange,
    refreshData
  } = useTable<typeof fetchWalletOrderPage>({
    core: {
      apiFn: fetchWalletOrderPage,
      apiParams: { current: 1, size: 10, status: undefined, type: undefined },
      columnsFactory: () => [
        { prop: 'type', label: '类型', width: 120, useSlot: true },
        { prop: 'amount', label: '金额', width: 150, align: 'right', useSlot: true },
        { prop: 'status', label: '状态', width: 130, useSlot: true },
        { prop: 'note', label: '备注', minWidth: 220, showOverflowTooltip: true },
        { prop: 'createdAt', label: '时间', minWidth: 180 },
        {
          prop: 'operation',
          label: '操作',
          width: 170,
          fixed: 'right',
          align: 'right',
          useSlot: true
        }
      ]
    }
  })

  const monthStats = ref({ recharge: 0, withdraw: 0, pending: 0, total: 0 })

  const formatMoney = (amount: number, currency = 'CNY') => {
    try {
      return new Intl.NumberFormat('zh-CN', {
        style: 'currency',
        currency: currency || 'CNY'
      }).format(Number(amount || 0))
    } catch {
      return `${currency || 'CNY'} ${Number(amount || 0).toFixed(2)}`
    }
  }
  const typeLabel = (value: string) =>
    ({ recharge: '充值', withdraw: '提现', refund: '退款' })[value] || value || '其他'
  const typeIcon = (value: string) =>
    ({
      recharge: 'ri:add-circle-line',
      withdraw: 'ri:upload-2-line',
      refund: 'ri:arrow-go-back-line'
    })[value] || 'ri:exchange-funds-line'

  const isPendingPayRecharge = (row: WalletOrderRow) => {
    const method = String(row.meta.payment_method || '')
    return (
      row.type === 'recharge' &&
      row.status === 'pending_review' &&
      Boolean(method && !['approval', 'balance'].includes(method))
    )
  }
  const statusLabel = (row: WalletOrderRow) => {
    if (isPendingPayRecharge(row)) return '待支付'
    return (
      {
        pending: '待处理',
        pending_review: '待审核',
        approved: '已通过',
        rejected: '已拒绝',
        canceled: '已取消',
        cancelled: '已取消',
        paid: '已支付'
      }[row.status] ||
      row.status ||
      '-'
    )
  }
  const statusTagType = (row: WalletOrderRow): 'success' | 'warning' | 'danger' | 'info' => {
    if (['approved', 'paid'].includes(row.status)) return 'success'
    if (['pending', 'pending_review'].includes(row.status)) return 'warning'
    if (['rejected', 'canceled', 'cancelled'].includes(row.status)) return 'danger'
    return 'info'
  }
  const canContinuePay = (row: WalletOrderRow) => isPendingPayRecharge(row)
  const canCancel = (row: WalletOrderRow) =>
    ['recharge', 'refund'].includes(row.type) && row.status === 'pending_review'

  const applyFilters = () => {
    searchParams.status = selectedStatus.value || undefined
    searchParams.type = selectedType.value || undefined
    getData()
  }
  const openPayment = (payload: CompatibleRecord) => {
    const payment = (payload.payment ?? payload) as CompatibleRecord
    const url = String(payment.pay_url ?? payment.payURL ?? '')
    if (!url) return false
    // H2：pay_url 是外部可控数据，统一过 URL 安全策略，非法值不导航。
    return openPaymentUrl(url, { onInvalid: () => ElMessage.error('支付链接无效，请重试') })
  }
  const fetchRechargeMethods = async () => {
    try {
      const response = await listPaymentProviders({ scene: 'wallet' })
      rechargeMethods.value = (response.data?.items ?? []).filter(
        (item) => item.enabled !== false && item.wallet_enabled !== false && item.key !== 'balance'
      )
      if (!rechargeMethods.value.some((item) => item.key === 'approval')) {
        rechargeMethods.value.unshift({ key: 'approval', name: '人工审核' })
      }
    } catch {
      rechargeMethods.value = [{ key: 'approval', name: '人工审核' }]
    }
    if (!rechargeMethods.value.some((item) => item.key === recharge.method)) {
      recharge.method = rechargeMethods.value[0]?.key || 'approval'
    }
  }
  const fetchOverview = async () => {
    overviewLoading.value = true
    try {
      const walletResponse = await getWallet()
      const normalized = normalizeWallet(walletResponse.data)
      wallet.value = {
        balance: Number(normalized.balance || 0),
        currency: String(normalized.currency || 'CNY')
      }
      await fetchRechargeMethods()
    } finally {
      overviewLoading.value = false
    }
  }
  const refreshAll = async () => {
    await Promise.all([refreshData(), fetchOverview(), loadOverview()])
  }

  const openRechargeDialog = () => {
    void fetchRechargeMethods()
    rechargeDialogVisible.value = true
  }
  const resetRechargeForm = () => {
    recharge.method = 'approval'
    recharge.amount = null
    recharge.note = ''
    rechargeFormRef.value?.clearValidate()
  }
  const resetWithdrawForm = () => {
    withdraw.amount = null
    withdraw.note = ''
    withdrawFormRef.value?.clearValidate()
  }
  const submitRecharge = async () => {
    try {
      await rechargeFormRef.value?.validate()
    } catch {
      return
    }
    if (!recharge.amount || recharge.amount <= 0) return
    submitting.value = true
    try {
      const response = await createWalletRecharge({
        amount: recharge.amount,
        note: recharge.note,
        method: recharge.method,
        meta: {}
      })
      openPayment(response.data as CompatibleRecord)
      ElMessage.success('充值订单已创建')
      rechargeDialogVisible.value = false
      await refreshAll()
    } finally {
      submitting.value = false
    }
  }
  const submitWithdraw = async () => {
    try {
      await withdrawFormRef.value?.validate()
    } catch {
      return
    }
    if (!withdraw.amount || withdraw.amount <= 0) return
    if (withdraw.amount > wallet.value.balance) {
      ElMessage.warning('提现金额不能超过可用余额')
      return
    }
    submitting.value = true
    try {
      await createWalletWithdraw({
        amount: withdraw.amount,
        note: withdraw.note,
        meta: { channel: 'manual' }
      })
      ElMessage.success('提现订单已提交')
      withdrawDialogVisible.value = false
      await refreshAll()
    } finally {
      submitting.value = false
    }
  }

  const continuePay = async (row: WalletOrderRow) => {
    const method = String(row.meta.payment_method || '')
    const response = await payWalletOrder(row.id, { method })
    if (openPayment(response.data as CompatibleRecord)) {
      ElMessage.success('已拉起支付')
    } else {
      ElMessage.warning('未获取到支付链接')
    }
  }
  const cancelPendingOrder = async (row: WalletOrderRow) => {
    try {
      await ElMessageBox.confirm(`确认取消该${typeLabel(row.type)}订单吗？`, '取消钱包订单', {
        confirmButtonText: '确认取消',
        cancelButtonText: '暂不取消',
        type: 'warning'
      })
      await cancelWalletOrder(row.id, { reason: 'user_cancel' })
      ElMessage.success('已取消')
      await refreshAll()
    } catch (error) {
      if (error !== 'cancel' && error !== 'close') ElMessage.error('取消失败')
    }
  }

  onMounted(() => {
    void fetchOverview()
    void loadOverview()
  })
</script>

<style lang="scss" scoped>
  .billing-page {
    padding-bottom: 20px;
  }

  .balance-card {
    margin-bottom: 14px;
  }

  .balance-card :global(.el-card__body) {
    display: flex;
    gap: 18px;
    align-items: center;
    justify-content: space-between;
  }

  .balance-main {
    display: flex;
    gap: 14px;
    align-items: center;

    > div:last-child {
      display: grid;
      gap: 4px;
    }

    span,
    small {
      color: var(--art-gray-600);
    }

    strong {
      font-size: 30px;
      color: var(--theme-color);
      letter-spacing: 0;
    }
  }

  .balance-icon {
    display: grid;
    place-items: center;
    width: 52px;
    height: 52px;
    font-size: 27px;
    color: var(--theme-color);
    background: var(--el-color-primary-light-9);
    border-radius: calc(var(--custom-radius) / 2 + 4px);
  }

  .balance-actions {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }

  .stats-grid {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 12px;
    margin-bottom: 14px;
  }

  .stat-card :global(.el-card__body) {
    display: grid;
    gap: 7px;
  }

  .stat-card {
    span {
      font-size: 13px;
      color: var(--art-gray-600);
    }

    strong {
      font-size: 21px;
      color: var(--art-gray-900);
    }
  }

  .billing-filters {
    display: flex;
    flex-wrap: wrap;
    gap: 12px 24px;

    label {
      display: flex;
      gap: 10px;
      align-items: center;
      font-size: 13px;
      color: var(--art-gray-700);
    }
  }

  .billing-table-card {
    min-height: 500px;
    margin-top: 12px;
  }

  .table-title {
    font-weight: 600;
    color: var(--art-gray-900);
  }

  .type-cell {
    display: flex;
    gap: 7px;
    align-items: center;
    color: var(--art-gray-800);

    .art-svg-icon {
      color: var(--theme-color);
    }
  }

  .amount {
    color: var(--el-color-danger);

    &--income {
      color: var(--el-color-success);
    }
  }

  .table-actions {
    display: flex;
    justify-content: flex-end;
  }

  .dialog-form {
    margin-top: 16px;
  }

  .full-width {
    width: 100%;
  }

  .available-balance {
    display: flex;
    justify-content: space-between;
    margin-top: 16px;
    color: var(--art-gray-700);

    strong {
      color: var(--art-gray-900);
    }
  }

  @media (width <= 900px) {
    .stats-grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }

    .billing-filters,
    .billing-filters label {
      flex-direction: column;
      align-items: stretch;
    }
  }

  @media (width <= 640px) {
    .balance-card :global(.el-card__body) {
      flex-direction: column;
      align-items: stretch;
    }

    .balance-actions > .el-button {
      flex: 1;
    }

    .stats-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
