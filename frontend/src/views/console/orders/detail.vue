<template>
  <div v-loading="loading" class="order-detail-page">
    <header class="page-heading">
      <div class="page-heading__main">
        <ElButton
          circle
          :icon="ArrowLeft"
          aria-label="返回订单列表"
          @click="router.push({ name: 'ConsoleOrders' })"
        />
        <div>
          <h1>订单详情</h1>
          <p>{{ order?.orderNo || `订单 ${orderId}` }}</p>
        </div>
      </div>
      <OrderStatusBadge :status="order?.status || ''" />
    </header>

    <ElCard class="overview-card art-card-xs" shadow="never">
      <div class="overview-grid">
        <div class="overview-item">
          <span>订单号</span>
          <strong>{{ order?.orderNo || '-' }}</strong>
        </div>
        <div class="overview-item">
          <span>订单金额</span>
          <strong class="overview-item__amount">{{
            formatMoney(order?.totalAmount || 0, order?.currency)
          }}</strong>
        </div>
        <div class="overview-item">
          <span>创建时间</span>
          <strong>{{ formatTime(order?.createdAt) }}</strong>
        </div>
        <div class="overview-item">
          <span>商品数量</span>
          <strong>{{ instanceCount }} 件</strong>
        </div>
      </div>
    </ElCard>

    <ElCard class="progress-card art-card-xs" shadow="never">
      <ElSteps :active="stepIndex" finish-status="success" align-center>
        <ElStep title="草稿" />
        <ElStep title="待支付" />
        <ElStep title="待审核" />
        <ElStep title="已通过" />
        <ElStep title="开通中" />
        <ElStep title="已完成" />
      </ElSteps>
    </ElCard>

    <div class="content-grid">
      <div class="main-column">
        <ElCard class="detail-card art-card-xs" shadow="never">
          <template #header>
            <div class="card-heading">
              <span>订单明细</span>
              <ElTag size="small">{{ orderItems.length }} 项</ElTag>
            </div>
          </template>
          <ElTable :data="orderItems" row-key="id" empty-text="暂无订单明细">
            <ElTableColumn prop="id" label="ID" width="90" />
            <ElTableColumn prop="packageId" label="套餐 ID" width="100">
              <template #default="{ row }">{{ row.packageId || '-' }}</template>
            </ElTableColumn>
            <ElTableColumn prop="systemId" label="系统 ID" width="100">
              <template #default="{ row }">{{ row.systemId || '-' }}</template>
            </ElTableColumn>
            <ElTableColumn label="规格" min-width="300">
              <template #default="{ row }">
                <div class="spec-cell">
                  <strong>{{ row.specText }}</strong>
                  <span v-for="(line, index) in row.specDetail" :key="index">{{ line }}</span>
                </div>
              </template>
            </ElTableColumn>
            <ElTableColumn prop="qty" label="数量" width="80" align="center" />
            <ElTableColumn label="金额" width="130" align="right">
              <template #default="{ row }">{{ formatMoney(row.amount, order?.currency) }}</template>
            </ElTableColumn>
            <ElTableColumn label="状态" width="120">
              <template #default="{ row }">
                <ElTag :type="statusTagType(row.status)">{{ row.status || '-' }}</ElTag>
              </template>
            </ElTableColumn>
          </ElTable>
        </ElCard>

        <ElCard class="detail-card art-card-xs" shadow="never">
          <template #header><span class="card-heading">付款记录</span></template>
          <ElTable :data="orderPayments" row-key="id" empty-text="暂无付款记录">
            <ElTableColumn prop="method" label="方式" width="120" />
            <ElTableColumn label="金额" width="130" align="right">
              <template #default="{ row }">{{ formatMoney(row.amount, row.currency) }}</template>
            </ElTableColumn>
            <ElTableColumn prop="tradeNo" label="交易号" min-width="180" />
            <ElTableColumn label="状态" width="120">
              <template #default="{ row }">
                <ElTag :type="statusTagType(row.status)">{{ row.status || '-' }}</ElTag>
              </template>
            </ElTableColumn>
            <ElTableColumn label="时间" min-width="170">
              <template #default="{ row }">{{ formatTime(row.createdAt) }}</template>
            </ElTableColumn>
          </ElTable>
        </ElCard>
      </div>

      <aside class="side-column">
        <ElCard class="side-card art-card-xs" shadow="never">
          <template #header><span class="card-heading">订单操作</span></template>
          <div class="action-list">
            <ElButton v-if="canPay" type="primary" size="large" @click="showPaymentDialog"
              >立即支付</ElButton
            >
            <ElButton :icon="Refresh" size="large" @click="refresh">刷新订单</ElButton>
            <ElButton v-if="canCancel" type="danger" plain size="large" @click="cancelCurrent"
              >撤销订单</ElButton
            >
          </div>
        </ElCard>

        <ElCard class="side-card art-card-xs" shadow="never">
          <template #header>
            <div class="card-heading">
              <span>开通进度</span>
              <ElTag :type="events.length ? 'success' : 'info'" size="small">
                {{ events.length ? '实时更新' : '等待中' }}
              </ElTag>
            </div>
          </template>
          <ElTimeline v-if="events.length" class="event-timeline">
            <ElTimelineItem
              v-for="(event, index) in events"
              :key="`${index}-${event}`"
              :type="index === 0 ? 'success' : 'primary'"
            >
              {{ event }}
            </ElTimelineItem>
          </ElTimeline>
          <ElEmpty v-else description="等待事件推送" :image-size="80" />
        </ElCard>

        <ElCard v-if="vpsInfo" class="side-card art-card-xs" shadow="never">
          <template #header>
            <div class="card-heading">
              <span>VPS 信息</span>
              <ElTooltip content="复制全部信息" placement="top">
                <ElButton
                  circle
                  text
                  :icon="CopyDocument"
                  aria-label="复制 VPS 信息"
                  @click="copyVpsInfo"
                />
              </ElTooltip>
            </div>
          </template>
          <ElDescriptions :column="1" border size="small">
            <ElDescriptionsItem v-for="(value, key) in vpsInfo" :key="key" :label="key">
              <span class="copyable-value">
                <span>{{ value }}</span>
                <ElTooltip :content="`复制${key}`" placement="top">
                  <ElButton
                    circle
                    text
                    :icon="CopyDocument"
                    :aria-label="`复制${key}`"
                    @click="copyText(value, `已复制${key}`)"
                  />
                </ElTooltip>
              </span>
            </ElDescriptionsItem>
          </ElDescriptions>
        </ElCard>
      </aside>
    </div>

    <ElDialog
      v-model="paymentDialogVisible"
      title="发起支付"
      width="min(560px, calc(100vw - 24px))"
      @closed="paymentHint = ''"
    >
      <ElForm ref="paymentFormRef" :model="paymentForm" label-position="top">
        <ElFormItem
          label="付款方式"
          prop="method"
          :rules="[{ required: true, message: '请选择付款方式', trigger: 'change' }]"
        >
          <ElSelect v-model="paymentForm.method" placeholder="请选择付款方式" class="full-width">
            <ElOption
              v-for="provider in providers"
              :key="provider.key"
              :label="provider.name || provider.key"
              :value="provider.key"
            />
          </ElSelect>
        </ElFormItem>

        <template v-if="paymentForm.method === 'approval'">
          <ElFormItem
            label="付款金额"
            prop="amount"
            :rules="[{ required: true, message: '请输入金额', trigger: 'blur' }]"
          >
            <ElInputNumber
              v-model="paymentForm.amount"
              :min="0"
              :precision="2"
              class="full-width"
            />
          </ElFormItem>
          <ElFormItem label="交易号">
            <ElInput v-model="paymentForm.trade_no" :maxlength="INPUT_LIMITS.PAYMENT_TRADE_NO" />
          </ElFormItem>
          <ElFormItem label="付款截图链接">
            <ElInput v-model="paymentForm.screenshot_url" :maxlength="INPUT_LIMITS.URL" />
          </ElFormItem>
          <ElFormItem label="备注">
            <ElInput
              v-model="paymentForm.note"
              type="textarea"
              :rows="3"
              :maxlength="INPUT_LIMITS.PAYMENT_NOTE"
              show-word-limit
            />
          </ElFormItem>
        </template>

        <template v-else>
          <ElFormItem
            v-for="field in schemaFields"
            :key="field.key"
            :label="field.label"
            :prop="field.key"
            :rules="
              field.required
                ? [{ required: true, message: `请输入${field.label}`, trigger: 'blur' }]
                : []
            "
          >
            <ElInputNumber
              v-if="field.type === 'number'"
              v-model="paymentForm[field.key]"
              class="full-width"
            />
            <ElSwitch v-else-if="field.type === 'boolean'" v-model="paymentForm[field.key]" />
            <ElSelect
              v-else-if="field.type === 'select'"
              v-model="paymentForm[field.key]"
              class="full-width"
              :placeholder="field.placeholder"
            >
              <ElOption
                v-for="option in field.options"
                :key="String(option.value)"
                :label="option.label"
                :value="option.value"
              />
            </ElSelect>
            <ElInput
              v-else
              v-model="paymentForm[field.key]"
              :type="
                field.type === 'textarea'
                  ? 'textarea'
                  : field.type === 'password'
                    ? 'password'
                    : 'text'
              "
              :rows="field.type === 'textarea' ? 3 : undefined"
              :placeholder="field.placeholder"
              show-password
            />
          </ElFormItem>
        </template>

        <ElAlert
          v-if="paymentForm.method === 'balance' && selectedProvider?.balance != null"
          :title="`钱包余额：${formatMoney(Number(selectedProvider.balance), order?.currency)}`"
          type="info"
          :closable="false"
        />

        <ElAlert
          v-if="selectedInstructions"
          :title="selectedInstructions"
          type="info"
          :closable="false"
        />
        <ElAlert
          v-if="paymentHint"
          :title="paymentHint"
          type="success"
          :closable="false"
          class="payment-hint"
        />
      </ElForm>
      <template #footer>
        <ElButton @click="paymentDialogVisible = false">取消</ElButton>
        <ElButton type="primary" :loading="paying" @click="submitPayment">提交支付</ElButton>
      </template>
    </ElDialog>

    <ElDialog v-model="wechatQrOpen" title="微信扫码支付" width="min(420px, calc(100vw - 24px))">
      <div v-loading="wechatQrLoading" class="qr-payment">
        <img v-if="wechatQrDataUrl" :src="wechatQrDataUrl" alt="微信支付二维码" />
        <ElEmpty v-else description="二维码生成失败" />
        <ElLink :href="wechatQrUrl" target="_blank" type="primary">打开支付链接</ElLink>
      </div>
    </ElDialog>

    <ElDialog v-model="wechatJsapiOpen" title="微信支付" width="min(560px, calc(100vw - 24px))">
      <ElButton type="primary" :disabled="!wechatJsapiParams" @click="invokeWeChatJsapi"
        >在微信中发起支付</ElButton
      >
      <pre v-if="wechatJsapiParamsJson" class="jsapi-params">{{ wechatJsapiParamsJson }}</pre>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import { ArrowLeft, CopyDocument, Refresh } from '@element-plus/icons-vue'
  import type { FormInstance } from 'element-plus'
  import QRCode from 'qrcode'
  import OrderStatusBadge from '@/components/business/order-status-badge/index.vue'
  import { INPUT_LIMITS } from '@/constants/inputLimits'
  import {
    cancelOrder,
    createOrderPayment,
    getOrderDetail,
    listPaymentProviders,
    listVps,
    refreshOrder,
    submitOrderPayment
  } from '@/services/user'
  import { createSseConnection } from '@/services/sse'
  import type {
    Order,
    OrderItem,
    OrderPayment,
    Package,
    PaymentProvider,
    PaymentRequest
  } from '@/services/types'
  import { useAuthStore } from '@/stores/auth'
  import { useCatalogStore } from '@/stores/catalog'
  import { IdempotentRequest } from '@/utils/idempotentRequest'

  defineOptions({ name: 'ConsoleOrderDetail' })

  declare global {
    interface Window {
      WeixinJSBridge?: {
        invoke: (
          method: string,
          params: Record<string, unknown>,
          callback: (result: Record<string, unknown>) => void
        ) => void
      }
    }
  }

  type CompatibleRecord = Record<string, any>

  interface OrderView {
    id: number | string
    orderNo: string
    status: string
    totalAmount: number
    currency: string
    createdAt: string
  }

  interface OrderItemView {
    id: number | string
    packageId?: number
    systemId?: number
    qty: number
    amount: number
    status: string
    action: string
    spec: CompatibleRecord
    specText: string
    specDetail: string[]
  }

  interface PaymentView {
    id: number | string
    method: string
    amount: number
    currency: string
    tradeNo: string
    status: string
    createdAt: string
  }

  interface SchemaField {
    key: string
    label: string
    type: 'text' | 'password' | 'textarea' | 'number' | 'boolean' | 'select'
    required: boolean
    placeholder: string
    default?: unknown
    options: Array<{ label: string; value: unknown }>
  }

  const route = useRoute()
  const router = useRouter()
  const auth = useAuthStore()
  const catalog = useCatalogStore()
  const orderId = computed(() => String(route.params.id || ''))
  const loading = ref(false)
  const order = ref<OrderView | null>(null)
  const orderItems = ref<OrderItemView[]>([])
  const orderPayments = ref<PaymentView[]>([])
  const events = ref<string[]>([])
  const providers = ref<PaymentProvider[]>([])
  const paying = ref(false)
  const paymentHint = ref('')
  const paymentFormRef = ref<FormInstance>()
  const paymentDialogVisible = ref(false)
  const paymentForm = reactive<CompatibleRecord>({
    method: '',
    amount: 0,
    trade_no: '',
    note: '',
    screenshot_url: ''
  })
  const approvalPaymentRequest = new IdempotentRequest<PaymentRequest>('pay')
  const wechatQrOpen = ref(false)
  const wechatQrLoading = ref(false)
  const wechatQrUrl = ref('')
  const wechatQrDataUrl = ref('')
  const wechatJsapiOpen = ref(false)
  const wechatJsapiParams = ref<Record<string, unknown> | null>(null)
  const hasAutoNavigated = ref(false)
  let sse: ReturnType<typeof createSseConnection> | undefined
  let pollingTimer: ReturnType<typeof setInterval> | undefined
  let vpsRetryTimer: ReturnType<typeof setInterval> | undefined

  const parseJson = (value: unknown): CompatibleRecord => {
    if (!value) return {}
    if (typeof value !== 'string') return value as CompatibleRecord
    try {
      return JSON.parse(value) as CompatibleRecord
    } catch {
      return {}
    }
  }

  const normalizeOrder = (row: Order & CompatibleRecord): OrderView => ({
    id: row.id ?? row.ID ?? orderId.value,
    orderNo: String(row.order_no ?? row.OrderNo ?? ''),
    status: String(row.status ?? row.Status ?? ''),
    totalAmount: Number(row.total_amount ?? row.TotalAmount ?? 0),
    currency: String(row.currency ?? row.Currency ?? 'CNY'),
    createdAt: String(row.created_at ?? row.CreatedAt ?? '')
  })

  const findPackage = (packageId?: number) =>
    (catalog.packages as Package[]).find((item) => String(item.id) === String(packageId))

  const formatSpec = (spec: CompatibleRecord, row: CompatibleRecord) => {
    if (String(row.action ?? row.Action ?? '') === 'resize') {
      return `套餐 ID ${spec.current_package_id || '-'} → ${spec.target_package_id || '-'}`
    }
    const pkg = findPackage(Number(row.package_id ?? row.PackageID ?? 0))
    const cpu = Number(pkg?.cores || 0) + Number(spec.add_cores || 0)
    const memory = Number(pkg?.memory_gb || 0) + Number(spec.add_mem_gb || 0)
    const disk = Number(pkg?.disk_gb || 0) + Number(spec.add_disk_gb || 0)
    const bandwidth = Number(pkg?.bandwidth_mbps || 0) + Number(spec.add_bw_mbps || 0)
    return `CPU ${cpu} 核 / 内存 ${memory} GB / 磁盘 ${disk} GB / 带宽 ${bandwidth} Mbps`
  }

  const formatSpecDetail = (spec: CompatibleRecord, row: CompatibleRecord) => {
    if (String(row.action ?? row.Action ?? '') !== 'resize') {
      return spec.duration_months ? [`购买时长：${spec.duration_months} 个月`] : []
    }
    const currentCpu = Number(spec.current_cpu || 0)
    const currentMem = Number(spec.current_mem_gb || 0)
    const currentDisk = Number(spec.current_disk_gb || 0)
    const currentBandwidth = Number(spec.current_bw_mbps || 0)
    const targetCpu = Number(spec.target_cpu || 0)
    const targetMem = Number(spec.target_mem_gb || 0)
    const targetDisk = Number(spec.target_disk_gb || 0)
    const targetBandwidth = Number(spec.target_bw_mbps || 0)
    const delta = (target: number, current: number) => {
      const value = target - current
      return value > 0 ? `+${value}` : String(value)
    }
    const packageChange = `套餐 ID：${spec.current_package_id || '-'} → ${spec.target_package_id || '-'}`
    const current = `原配置：CPU ${currentCpu} 核 / 内存 ${currentMem} GB / 磁盘 ${currentDisk} GB / 带宽 ${currentBandwidth} Mbps`
    const target = `新配置：CPU ${targetCpu} 核 / 内存 ${targetMem} GB / 磁盘 ${targetDisk} GB / 带宽 ${targetBandwidth} Mbps`
    const resourceDelta = `资源差量：CPU ${delta(targetCpu, currentCpu)} 核 / 内存 ${delta(targetMem, currentMem)} GB / 磁盘 ${delta(targetDisk, currentDisk)} GB / 带宽 ${delta(targetBandwidth, currentBandwidth)} Mbps`
    const monthly = `月费：${formatMoney(Number(spec.current_monthly || 0), order.value?.currency)} → ${formatMoney(Number(spec.target_monthly || 0), order.value?.currency)}`
    const settlement =
      Number(spec.charge_amount || 0) > 0
        ? `补差价：${formatMoney(Number(spec.charge_amount), order.value?.currency)}`
        : Number(spec.refund_amount || 0) > 0
          ? `退款：${formatMoney(Number(spec.refund_amount), order.value?.currency)}${spec.refund_to_wallet ? '（退回钱包）' : ''}`
          : '无额外结算'
    return [packageChange, current, target, resourceDelta, monthly, settlement]
  }

  const normalizeItem = (item: OrderItem & CompatibleRecord): OrderItemView => {
    const spec = parseJson(item.spec ?? item.Spec ?? item.spec_json ?? item.SpecJSON)
    return {
      id: item.id ?? item.ID ?? '',
      packageId: Number(item.package_id ?? item.PackageID ?? 0) || undefined,
      systemId: Number(item.system_id ?? item.SystemID ?? 0) || undefined,
      qty: Number(item.qty ?? item.Qty ?? 0),
      amount: Number(item.amount ?? item.Amount ?? 0),
      status: String(item.status ?? item.Status ?? ''),
      action: String(item.action ?? item.Action ?? ''),
      spec,
      specText: formatSpec(spec, item),
      specDetail: formatSpecDetail(spec, item)
    }
  }

  const normalizePayment = (item: OrderPayment & CompatibleRecord): PaymentView => ({
    id: item.id ?? item.ID ?? '',
    method: String(item.method ?? item.Method ?? '-'),
    amount: Number(item.amount ?? item.Amount ?? 0),
    currency: String(item.currency ?? item.Currency ?? order.value?.currency ?? 'CNY'),
    tradeNo: String(item.trade_no ?? item.TradeNo ?? '-'),
    status: String(item.status ?? item.Status ?? ''),
    createdAt: String(item.created_at ?? item.CreatedAt ?? '')
  })

  const fetchDetail = async () => {
    const response = await getOrderDetail(orderId.value)
    const payload = response.data ?? {}
    order.value = payload.order ? normalizeOrder(payload.order as Order & CompatibleRecord) : null
    orderItems.value = (payload.items ?? []).map((item) =>
      normalizeItem(item as OrderItem & CompatibleRecord)
    )
    orderPayments.value = (payload.payments ?? []).map((item) =>
      normalizePayment(item as OrderPayment & CompatibleRecord)
    )
    syncPayAmountFromOrder()
  }

  const canPay = computed(() => order.value?.status === 'pending_payment')
  const canCancel = computed(() =>
    ['pending_payment', 'pending_review'].includes(order.value?.status || '')
  )
  const instanceCount = computed(() =>
    orderItems.value.reduce((total, item) => total + Math.max(0, item.qty), 0)
  )
  const stepIndex = computed(() => {
    const steps = [
      'draft',
      'pending_payment',
      'pending_review',
      'approved',
      'provisioning',
      'active'
    ]
    return Math.max(0, steps.indexOf(order.value?.status || ''))
  })
  const isProvisioning = computed(
    () =>
      order.value?.status === 'provisioning' ||
      orderItems.value.some((item) => item.status === 'provisioning')
  )

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
  const formatTime = (value?: string) => {
    if (!value) return '-'
    const date = new Date(value)
    return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN')
  }
  const statusTagType = (status: string): 'success' | 'warning' | 'danger' | 'info' | 'primary' => {
    if (['active', 'approved', 'paid', 'success'].includes(status)) return 'success'
    if (['pending_payment', 'pending_review', 'provisioning', 'pending'].includes(status))
      return 'warning'
    if (['failed', 'rejected', 'canceled', 'cancelled'].includes(status)) return 'danger'
    return status ? 'primary' : 'info'
  }

  const syncPayAmountFromOrder = () => {
    paymentForm.amount = Number(order.value?.totalAmount || 0)
  }
  const showPaymentDialog = () => {
    syncPayAmountFromOrder()
    paymentDialogVisible.value = true
  }

  const normalizeSchemaFields = (schemaJson?: string): SchemaField[] => {
    if (!schemaJson) return []
    try {
      const parsed = JSON.parse(schemaJson) as CompatibleRecord
      if (Array.isArray(parsed)) return parsed as SchemaField[]
      if (Array.isArray(parsed.fields)) return parsed.fields as SchemaField[]
      const properties = (parsed.properties ?? {}) as CompatibleRecord
      const required = new Set<string>(Array.isArray(parsed.required) ? parsed.required : [])
      return Object.keys(properties).map((key) => {
        const property = properties[key] as CompatibleRecord
        const enumValues = Array.isArray(property.enum) ? property.enum : null
        const type: SchemaField['type'] = enumValues
          ? 'select'
          : property.format === 'password'
            ? 'password'
            : property.format === 'textarea'
              ? 'textarea'
              : ['number', 'integer'].includes(property.type)
                ? 'number'
                : property.type === 'boolean'
                  ? 'boolean'
                  : 'text'
        return {
          key,
          label: String(property.title ?? property.label ?? key),
          type,
          required: required.has(key),
          placeholder: String(property.description ?? property.placeholder ?? ''),
          default: property.default,
          options: enumValues?.map((value) => ({ label: String(value), value })) ?? []
        }
      })
    } catch {
      return []
    }
  }

  const selectedProvider = computed(() =>
    providers.value.find((item) => item.key === paymentForm.method)
  )
  const schemaFields = computed(() => {
    const provider = selectedProvider.value
    if (
      !provider?.schema_json ||
      ['approval', 'balance', 'custom', 'yipay'].includes(provider.key || '')
    )
      return []
    return normalizeSchemaFields(provider.schema_json)
  })
  const selectedInstructions = computed(() => {
    const config = parseJson(selectedProvider.value?.config_json)
    return String(config.instructions ?? config.notice ?? '')
  })
  const wechatJsapiParamsJson = computed(() =>
    wechatJsapiParams.value ? JSON.stringify(wechatJsapiParams.value, null, 2) : ''
  )

  watch(
    () => paymentForm.method,
    (_value, previous) => {
      if (previous) {
        const oldProvider = providers.value.find((item) => item.key === previous)
        normalizeSchemaFields(oldProvider?.schema_json).forEach(
          (field) => delete paymentForm[field.key]
        )
      }
      schemaFields.value.forEach((field) => {
        if (paymentForm[field.key] === undefined) {
          paymentForm[field.key] = field.default ?? (field.type === 'boolean' ? false : '')
        }
      })
      paymentHint.value = ''
    }
  )

  const openWeChatQr = async (url: string) => {
    wechatQrOpen.value = true
    wechatQrUrl.value = url
    wechatQrLoading.value = true
    try {
      wechatQrDataUrl.value = await QRCode.toDataURL(url, { width: 260, margin: 1 })
    } finally {
      wechatQrLoading.value = false
    }
  }
  const openWeChatJsapi = (paramsJson: string) => {
    try {
      wechatJsapiParams.value = JSON.parse(paramsJson) as Record<string, unknown>
    } catch {
      wechatJsapiParams.value = null
    }
    wechatJsapiOpen.value = true
  }
  const invokeWeChatJsapi = async () => {
    const bridge = window.WeixinJSBridge
    if (!bridge || !wechatJsapiParams.value) {
      ElMessage.warning('当前环境不支持 JSAPI，请在微信内打开此页面')
      return
    }
    try {
      await new Promise<void>((resolve, reject) => {
        bridge.invoke('getBrandWCPayRequest', wechatJsapiParams.value!, (result) => {
          const message = String(result.err_msg ?? result.errMsg ?? '')
          if (message === 'get_brand_wcpay_request:ok') resolve()
          else reject(new Error(message || 'pay failed'))
        })
      })
      ElMessage.success('已发起支付，请在微信中完成付款')
      wechatJsapiOpen.value = false
    } catch (error) {
      ElMessage.error((error as Error).message || '微信支付发起失败')
    }
  }

  const submitPayment = async () => {
    try {
      await paymentFormRef.value?.validate()
    } catch {
      return
    }
    const method = String(paymentForm.method || '')
    if (!method) return
    paying.value = true
    try {
      if (method === 'approval') {
        const request = approvalPaymentRequest.prepare(() => ({
          method,
          amount: Number(paymentForm.amount || 0),
          trade_no: String(paymentForm.trade_no || ''),
          note: String(paymentForm.note || ''),
          screenshot_url: String(paymentForm.screenshot_url || '')
        }))
        await submitOrderPayment(orderId.value, request.payload, request.key)
        approvalPaymentRequest.resolve()
        ElMessage.success('已提交付款信息')
        paymentDialogVisible.value = false
        await fetchDetail()
        return
      }

      const extra = schemaFields.value.reduce<Record<string, unknown>>((result, field) => {
        result[field.key] = paymentForm[field.key]
        return result
      }, {})
      const response = await createOrderPayment(orderId.value, { method, extra })
      const result = response.data ?? {}
      const resultExtra = (result.extra ?? {}) as Record<string, unknown>
      paymentHint.value = String(resultExtra.instructions ?? '')
      const payKind = String(resultExtra.pay_kind ?? '')

      if (payKind === 'qr') {
        const url = String(resultExtra.code_url ?? result.pay_url ?? '')
        if (url) await openWeChatQr(url)
      } else if (payKind === 'jsapi') {
        openWeChatJsapi(String(resultExtra.jsapi_params_json ?? ''))
      } else if (payKind === 'form') {
        const opened = window.open('', '_blank')
        if (!opened) {
          ElMessage.warning('浏览器拦截了支付窗口')
          return
        }
        opened.document.open()
        opened.document.write(String(resultExtra.form_html ?? ''))
        opened.document.close()
      } else if (payKind === 'urlscheme') {
        window.location.href = String(resultExtra.urlscheme ?? '')
      } else if (payKind === 'redirect' || result.pay_url) {
        window.open(String(resultExtra.pay_url ?? result.pay_url ?? ''), '_blank')
      } else if (result.paid) {
        ElMessage.success('支付完成')
        await fetchDetail()
      } else if (result.status === 'manual') {
        ElMessage.info('该方式需要人工处理，请按提示完成支付')
        return
      } else {
        ElMessage.success('支付请求已提交')
      }
      paymentDialogVisible.value = false
    } catch (error) {
      if (method === 'approval') approvalPaymentRequest.reject(error)
    } finally {
      paying.value = false
    }
  }

  const parseVpsInfo = (text: string) => {
    const result: Record<string, string> = {}
    text
      .split(/\n|\r/)
      .map((line) => line.trim())
      .filter(Boolean)
      .forEach((line) => {
        const separator = line.includes(':') ? ':' : line.includes('=') ? '=' : ''
        if (!separator) return
        const [key, ...value] = line.split(separator)
        result[key.trim()] = value.join(separator).trim()
      })
    return Object.keys(result).length ? result : null
  }
  const vpsInfo = computed(() => {
    for (const event of events.value) {
      const parsed = parseVpsInfo(event)
      if (parsed) return parsed
    }
    return null
  })
  const copyText = async (text: string, successMessage: string) => {
    try {
      if (navigator.clipboard?.writeText) {
        await navigator.clipboard.writeText(text)
      } else {
        const textarea = document.createElement('textarea')
        textarea.value = text
        textarea.style.position = 'fixed'
        textarea.style.opacity = '0'
        document.body.appendChild(textarea)
        textarea.select()
        const copied = document.execCommand('copy')
        document.body.removeChild(textarea)
        if (!copied) throw new Error('copy failed')
      }
      ElMessage.success(successMessage)
    } catch {
      ElMessage.error('复制失败，请手动复制')
    }
  }
  const copyVpsInfo = async () => {
    if (!vpsInfo.value) return
    const text = Object.entries(vpsInfo.value)
      .map(([key, value]) => `${key}: ${value}`)
      .join('\n')
    await copyText(text, '已复制 VPS 信息')
  }

  const startSse = () => {
    if (!auth.token) return
    sse?.close()
    const base = import.meta.env.VITE_API_BASE || ''
    sse = createSseConnection(`${base}/api/v1/orders/${orderId.value}/events`, {
      headers: { Authorization: `Bearer ${auth.token}` },
      onMessage: (message) => {
        if (message.data) events.value.unshift(message.data)
      }
    })
  }
  const stopPolling = () => {
    if (pollingTimer) clearInterval(pollingTimer)
    pollingTimer = undefined
  }
  const startPolling = () => {
    if (pollingTimer) return
    pollingTimer = setInterval(() => void fetchDetail(), 3000)
  }
  const stopVpsRetry = () => {
    if (vpsRetryTimer) clearInterval(vpsRetryTimer)
    vpsRetryTimer = undefined
  }
  const tryAutoNavigateToVps = async () => {
    if (hasAutoNavigated.value || instanceCount.value !== 1 || !orderItems.value[0]?.id) return
    const response = await listVps()
    const vps = (response.data?.items ?? []).find(
      (item: CompatibleRecord) =>
        String(item.order_item_id ?? item.OrderItemID ?? '') === String(orderItems.value[0].id)
    ) as CompatibleRecord | undefined
    const vpsId = vps?.id ?? vps?.ID
    if (!vpsId) return
    hasAutoNavigated.value = true
    stopPolling()
    stopVpsRetry()
    await router.push(`/console/vps/${vpsId}`)
  }

  watch(isProvisioning, (value) => (value ? startPolling() : stopPolling()), { immediate: true })
  watch(
    () => order.value?.status,
    async (status, previous) => {
      if (previous !== 'provisioning' || status !== 'active' || hasAutoNavigated.value) return
      await tryAutoNavigateToVps()
      if (hasAutoNavigated.value || instanceCount.value !== 1) return
      let attempts = 0
      vpsRetryTimer = setInterval(async () => {
        attempts += 1
        await tryAutoNavigateToVps()
        if (hasAutoNavigated.value || attempts >= 20) stopVpsRetry()
      }, 3000)
    }
  )

  const refresh = async () => {
    await refreshOrder(orderId.value)
    await fetchDetail()
  }
  const cancelCurrent = async () => {
    try {
      await ElMessageBox.confirm('撤销后订单将变为已取消，无法继续支付。确认撤销吗？', '撤销订单', {
        confirmButtonText: '确认撤销',
        cancelButtonText: '暂不撤销',
        type: 'warning'
      })
      await cancelOrder(orderId.value)
      ElMessage.success('订单已撤销')
      await fetchDetail()
    } catch (error) {
      if (error !== 'cancel' && error !== 'close') ElMessage.error('撤销订单失败')
    }
  }

  onMounted(async () => {
    loading.value = true
    try {
      const providerResponse = await listPaymentProviders({ scene: 'order' })
      providers.value = (providerResponse.data?.items ?? []).filter(
        (item) => item.enabled !== false && item.order_enabled !== false
      )
      paymentForm.method = providers.value[0]?.key || ''
      if (!catalog.packages.length) await catalog.fetchCatalog()
      await fetchDetail()
      startSse()
    } finally {
      loading.value = false
    }
  })

  onBeforeUnmount(() => {
    sse?.close()
    stopPolling()
    stopVpsRetry()
  })
</script>

<style lang="scss" scoped>
  .order-detail-page {
    padding-bottom: 20px;
  }

  .page-heading,
  .page-heading__main,
  .card-heading {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .page-heading {
    margin-bottom: 14px;

    &__main {
      gap: 12px;
      justify-content: flex-start;
    }

    h1 {
      margin: 0;
      font-size: 24px;
      color: var(--art-gray-900);
      letter-spacing: 0;
    }

    p {
      margin: 4px 0 0;
      color: var(--art-gray-600);
    }
  }

  .overview-card,
  .progress-card,
  .detail-card,
  .side-card {
    margin-bottom: 14px;
  }

  .overview-grid {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 18px;
  }

  .overview-item {
    display: grid;
    gap: 6px;
    min-width: 0;

    span {
      font-size: 13px;
      color: var(--art-gray-600);
    }

    strong {
      overflow: hidden;
      color: var(--art-gray-900);
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    &__amount {
      font-size: 18px;
      color: var(--theme-color);
    }
  }

  .content-grid {
    display: grid;
    grid-template-columns: minmax(0, 1fr) minmax(260px, 340px);
    gap: 14px;
    align-items: start;
  }

  .card-heading {
    width: 100%;
    font-weight: 600;
    color: var(--art-gray-900);
  }

  .spec-cell {
    display: grid;
    gap: 3px;

    strong {
      color: var(--art-gray-900);
    }

    span {
      font-size: 12px;
      line-height: 1.6;
      color: var(--art-gray-600);
    }
  }

  .side-column {
    position: sticky;
    top: 14px;
  }

  .action-list {
    display: grid;
    gap: 10px;
  }

  .copyable-value {
    display: flex;
    gap: 8px;
    align-items: center;
    justify-content: space-between;
  }

  .event-timeline {
    padding-top: 6px;
  }

  .full-width {
    width: 100%;
  }

  .payment-hint {
    margin-top: 12px;
  }

  .qr-payment {
    display: grid;
    gap: 10px;
    place-items: center;
    min-height: 290px;

    img {
      width: 260px;
      max-width: 100%;
      aspect-ratio: 1;
    }
  }

  .jsapi-params {
    max-height: 280px;
    padding: 12px;
    margin-top: 14px;
    overflow: auto;
    color: var(--art-gray-700);
    white-space: pre-wrap;
    background: var(--default-bg-color);
    border: 1px solid var(--default-border);
    border-radius: calc(var(--custom-radius) / 3 + 2px);
  }

  @media (width <= 980px) {
    .overview-grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }

    .content-grid {
      grid-template-columns: 1fr;
    }

    .side-column {
      position: static;
    }
  }

  @media (width <= 640px) {
    .page-heading {
      align-items: flex-start;
    }

    .overview-grid {
      grid-template-columns: 1fr;
    }

    .progress-card {
      overflow-x: auto;
    }
  }
</style>
