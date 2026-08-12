<template>
  <ElDrawer v-model="drawerVisible" title="订单详情" size="760px" destroy-on-close>
    <div v-loading="loading" class="detail-panel">
      <div class="detail-hero">
        <div class="hero-main">
          <h3>#{{ order?.order_no || order?.id || '-' }}</h3>
          <div class="hero-meta">
            <span>用户：{{ order?.user_id || '-' }}</span>
            <span v-if="relatedUserName" class="user-name">（{{ relatedUserName }}）</span>
            <ElTag :type="getOrderStatusTagType(order?.status)">
              {{ getOrderStatusText(order?.status) }}
            </ElTag>
          </div>
        </div>

        <div class="hero-amount">
          <div class="amount">￥{{ formatAmount(order?.total_amount) }}</div>
          <div class="currency">{{ order?.currency || 'CNY' }}</div>
        </div>
      </div>

      <ElDescriptions :column="2" border>
        <ElDescriptionsItem label="订单 ID">{{ order?.id || '-' }}</ElDescriptionsItem>
        <ElDescriptionsItem label="订单号">{{ order?.order_no || '-' }}</ElDescriptionsItem>
        <ElDescriptionsItem label="用户 ID">{{ order?.user_id || '-' }}</ElDescriptionsItem>
        <ElDescriptionsItem label="订单状态">
          <ElTag :type="getOrderStatusTagType(order?.status)">
            {{ getOrderStatusText(order?.status) }}
          </ElTag>
        </ElDescriptionsItem>
        <ElDescriptionsItem label="订单来源">{{ order?.source || '-' }}</ElDescriptionsItem>
        <ElDescriptionsItem label="创建时间">{{
          formatDateTime(order?.created_at)
        }}</ElDescriptionsItem>
        <ElDescriptionsItem label="审核人">{{ order?.approved_by || '-' }}</ElDescriptionsItem>
        <ElDescriptionsItem label="审核时间">{{
          formatDateTime(order?.approved_at)
        }}</ElDescriptionsItem>
        <ElDescriptionsItem label="待处理原因" :span="2">{{
          order?.pending_reason || '-'
        }}</ElDescriptionsItem>
        <ElDescriptionsItem label="驳回原因" :span="2">{{
          order?.rejected_reason || '-'
        }}</ElDescriptionsItem>
      </ElDescriptions>

      <section class="detail-section">
        <div class="section-title">付款信息</div>
        <ElTable :data="payments" border size="small" row-key="id">
          <ElTableColumn prop="method" label="方式" min-width="120" />
          <ElTableColumn label="金额" min-width="120">
            <template #default="{ row }">￥{{ formatAmount(row.amount) }}</template>
          </ElTableColumn>
          <ElTableColumn prop="trade_no" label="交易号" min-width="180" show-overflow-tooltip />
          <ElTableColumn prop="note" label="备注" min-width="180" show-overflow-tooltip />
          <ElTableColumn label="状态" min-width="120">
            <template #default="{ row }">
              <ElTag :type="getPaymentStatusTagType(row.status)">
                {{ getPaymentStatusText(row.status) }}
              </ElTag>
            </template>
          </ElTableColumn>
        </ElTable>
      </section>

      <section class="detail-section">
        <div class="section-title">订单项</div>
        <ElTable :data="items" border size="small" row-key="id">
          <ElTableColumn prop="id" label="ID" width="80" />
          <ElTableColumn prop="package_id" label="套餐 ID" min-width="100" />
          <ElTableColumn prop="system_id" label="系统 ID" min-width="100" />
          <ElTableColumn prop="qty" label="数量" width="80" />
          <ElTableColumn label="金额" min-width="120">
            <template #default="{ row }">￥{{ formatAmount(row.amount) }}</template>
          </ElTableColumn>
          <ElTableColumn prop="status" label="状态" min-width="120" />
        </ElTable>
      </section>

      <section class="detail-section">
        <div class="section-title">事件流</div>
        <!-- web_api_contract F4：详情只返回最近 200 条事件，总数更大时提示
             避免管理员误以为事件流被截断是数据丢失。 -->
        <ElAlert
          v-if="eventsTruncated"
          type="info"
          :closable="false"
          show-icon
          class="events-truncated-note"
          :title="`仅显示最近 ${normalizedEvents.length} 条事件（共 ${props.detail?.events_total ?? 0} 条）`"
        />
        <ElEmpty v-if="!normalizedEvents.length" description="暂无事件" />
        <ElTimeline v-else>
          <ElTimelineItem
            v-for="event in normalizedEvents"
            :key="event.id || `${event.type}-${event.seq}`"
            :timestamp="formatTime(event.created_at)"
            placement="top"
            :type="getEventTimelineType(event.type)"
          >
            <div class="event-card">
              <div class="event-header">
                <div class="event-title-wrap">
                  <span class="event-dot" :class="`event-${sanitizeEventType(event.type)}`">
                    <component :is="getEventIcon(event.type)" />
                  </span>
                  <span class="event-title">{{ getEventTypeText(event.type) }}</span>
                </div>
                <span class="event-time">{{ formatDateTime(event.created_at) }}</span>
              </div>
              <div v-if="event.parsedData.admin_id" class="event-line">
                <ElTag type="primary">管理员 {{ event.parsedData.admin_id }}</ElTag>
              </div>
              <div v-if="event.parsedData.user_id && !event.parsedData.admin_id" class="event-line">
                <ElTag>
                  用户 {{ event.parsedData.user_id }}
                  <span v-if="isCurrentOrderUser(event.parsedData.user_id) && relatedUserName">
                    （{{ relatedUserName }}）
                  </span>
                </ElTag>
              </div>
              <div v-if="event.parsedData.reason" class="event-line">
                <span class="label">原因：</span>{{ String(event.parsedData.reason) }}
              </div>
              <div v-if="event.parsedData.message" class="event-line">
                <span class="label">消息：</span>{{ String(event.parsedData.message) }}
              </div>
              <ElCollapse v-if="hasMoreData(event.parsedData)" class="event-collapse">
                <ElCollapseItem
                  title="详细信息"
                  :name="event.id || event.seq || event.type || 'detail'"
                >
                  <pre class="event-data-pre">{{ formatEventData(event.parsedData) }}</pre>
                </ElCollapseItem>
              </ElCollapse>
            </div>
          </ElTimelineItem>
        </ElTimeline>
      </section>

      <div class="drawer-actions">
        <ElButton
          v-auth="'order.approve'"
          type="primary"
          :disabled="isReviewLocked(order)"
          @click="emit('approve')"
        >
          通过
        </ElButton>
        <ElButton
          v-auth="'order.reject'"
          type="danger"
          plain
          :disabled="isReviewLocked(order)"
          @click="emit('reject')"
        >
          驳回
        </ElButton>
        <ElButton v-auth="'order.approve'" :disabled="!canRetry(order)" @click="emit('retry')">
          重试开通
        </ElButton>
        <ElButton v-auth="'order.delete'" type="danger" plain @click="emit('delete')">
          删除订单
        </ElButton>
      </div>
    </div>
  </ElDrawer>
</template>

<script setup lang="ts">
  import type {
    Order,
    OrderDetailWithEventsResponse,
    OrderEvent,
    OrderItem,
    OrderPayment
  } from '@/services/types'
  import {
    Check,
    CircleClose,
    Clock,
    Loading,
    Money,
    Promotion,
    Warning
  } from '@element-plus/icons-vue'

  defineOptions({ name: 'OrderDetailDrawer' })

  interface Props {
    visible: boolean
    loading?: boolean
    detail?: OrderDetailWithEventsResponse | null
    relatedUserName?: string
  }

  const props = withDefaults(defineProps<Props>(), {
    loading: false,
    detail: null,
    relatedUserName: ''
  })

  const emit = defineEmits<{
    (event: 'update:visible', value: boolean): void
    (event: 'approve'): void
    (event: 'reject'): void
    (event: 'retry'): void
    (event: 'delete'): void
  }>()

  interface OrderLike extends Order {
    source?: unknown
    UserID?: unknown
    OrderNo?: unknown
    Source?: unknown
    Currency?: unknown
    PendingReason?: unknown
    ApprovedBy?: unknown
    ApprovedAt?: unknown
    RejectedReason?: unknown
    CanReview?: unknown
    can_review?: unknown
  }

  interface ParsedOrderEvent extends OrderEvent {
    parsedData: Record<string, unknown>
  }

  const drawerVisible = computed({
    get: () => props.visible,
    set: (value) => emit('update:visible', value)
  })

  const order = computed(() => (props.detail?.order || {}) as OrderLike)
  const items = computed(() => (props.detail?.items || []).map(normalizeItem))
  const payments = computed(() => (props.detail?.payments || []).map(normalizePayment))
  const normalizedEvents = computed<ParsedOrderEvent[]>(() =>
    (props.detail?.events || []).map((event) => ({
      ...event,
      parsedData: parseEventData(event.data)
    }))
  )
  const eventsTruncated = computed(
    () => Number(props.detail?.events_total ?? 0) > (props.detail?.events?.length ?? 0)
  )

  function normalizeItem(item: OrderItem & Record<string, unknown>) {
    return {
      ...item,
      id: item.id ?? (item.ID as number | undefined),
      package_id: item.package_id ?? (item.PackageID as number | undefined),
      system_id: item.system_id ?? (item.SystemID as number | undefined),
      qty: item.qty ?? (item.Qty as number | undefined),
      amount: item.amount ?? (item.Amount as number | undefined),
      status: item.status ?? (item.Status as string | undefined)
    }
  }

  function normalizePayment(payment: OrderPayment & Record<string, unknown>) {
    return {
      ...payment,
      id: payment.id ?? (payment.ID as number | undefined),
      method: payment.method ?? (payment.Method as string | undefined),
      amount: payment.amount ?? (payment.Amount as number | undefined),
      trade_no: payment.trade_no ?? (payment.TradeNo as string | undefined),
      note: payment.note ?? (payment.Note as string | undefined),
      status: payment.status ?? (payment.Status as string | undefined)
    }
  }

  function parseEventData(value: unknown) {
    if (!value) return {}
    if (typeof value === 'string') {
      try {
        const parsed = JSON.parse(value)
        return isRecord(parsed) ? parsed : {}
      } catch {
        return {}
      }
    }
    return isRecord(value) ? value : {}
  }

  function isRecord(value: unknown): value is Record<string, unknown> {
    return Boolean(value && typeof value === 'object' && !Array.isArray(value))
  }

  function formatDateTime(value?: string | null) {
    if (!value) return '-'
    const date = new Date(value)
    return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN')
  }

  function formatTime(value?: string | null) {
    if (!value) return '-'
    const date = new Date(value)
    return Number.isNaN(date.getTime())
      ? value
      : date.toLocaleTimeString('zh-CN', { hour12: false })
  }

  function formatAmount(value?: number | null) {
    return Number(value || 0).toFixed(2)
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

  function getPaymentStatusTagType(status?: string) {
    const map: Record<string, 'info' | 'warning' | 'success' | 'danger'> = {
      pending: 'info',
      pending_review: 'warning',
      approved: 'success',
      rejected: 'danger',
      paid: 'success'
    }
    return map[String(status || '')] || 'info'
  }

  function getPaymentStatusText(status?: string) {
    const map: Record<string, string> = {
      pending: '待支付',
      pending_review: '待审核',
      approved: '已通过',
      rejected: '已驳回',
      paid: '已支付'
    }
    return map[String(status || '')] || String(status || '-')
  }

  function getEventTypeText(type?: string) {
    const map: Record<string, string> = {
      order_created: '订单创建',
      order_paid: '订单支付',
      order_approved: '订单通过',
      order_rejected: '订单驳回',
      provisioning_started: '开始开通',
      provisioning_progress: '开通进度',
      provisioning_completed: '开通完成',
      provisioning_failed: '开通失败',
      payment_created: '支付创建',
      payment_approved: '支付通过',
      payment_rejected: '支付驳回',
      status_changed: '状态变更'
    }
    return map[String(type || '')] || String(type || '-')
  }

  function getEventIcon(type?: string) {
    const map: Record<string, any> = {
      order_created: Clock,
      order_paid: Money,
      order_approved: Check,
      order_rejected: CircleClose,
      provisioning_started: Promotion,
      provisioning_progress: Loading,
      provisioning_completed: Check,
      provisioning_failed: Warning,
      payment_created: Money,
      payment_approved: Check,
      payment_rejected: CircleClose,
      status_changed: Loading
    }
    return map[String(type || '')] || Clock
  }

  function getEventTimelineType(type?: string) {
    if (
      ['order_paid', 'order_approved', 'provisioning_completed', 'payment_approved'].includes(
        String(type || '')
      )
    )
      return 'success'
    if (['provisioning_progress', 'status_changed'].includes(String(type || ''))) return 'warning'
    if (['order_rejected', 'provisioning_failed', 'payment_rejected'].includes(String(type || '')))
      return 'danger'
    return 'primary'
  }

  function formatEventData(data: Record<string, unknown>) {
    const formatted: Record<string, unknown> = {}
    for (const [key, value] of Object.entries(data || {})) {
      if (key === 'admin_id') formatted[key] = `管理员(${value})`
      else if (key === 'user_id') formatted[key] = `用户(${value})`
      else if (key === 'reason') formatted['原因'] = value
      else if (key === 'message') formatted['消息'] = value
      else if (key === 'from_status' || key === 'to_status')
        formatted[key === 'from_status' ? '原状态' : '新状态'] = value
      else formatted[key] = value
    }
    return JSON.stringify(formatted, null, 2)
  }

  function hasMoreData(data: Record<string, unknown>) {
    return Object.keys(data || {}).some(
      (key) => !['admin_id', 'user_id', 'reason', 'message'].includes(key)
    )
  }

  function isReviewLocked(record?: OrderLike) {
    if (!record) return true
    const canReview = record.can_review ?? record.CanReview
    if (canReview !== undefined) return !canReview
    return !['pending_review', 'pending-review', 'pending review'].includes(
      String(record.status ?? record.Status ?? '')
        .trim()
        .toLowerCase()
    )
  }

  function isCurrentOrderUser(value: unknown) {
    return String(value ?? '') === String(order.value.user_id ?? order.value.UserID ?? '')
  }

  function canRetry(record?: OrderLike) {
    return ['approved', 'provisioning', 'failed'].includes(
      String(record?.status ?? record?.Status ?? '')
        .trim()
        .toLowerCase()
    )
  }

  function sanitizeEventType(type?: string) {
    return String(type || 'unknown').replace(/[^a-z0-9_-]/gi, '-')
  }
</script>

<style scoped lang="scss">
  .detail-panel {
    display: flex;
    flex-direction: column;
    gap: 20px;
  }

  .detail-hero {
    display: flex;
    gap: 16px;
    align-items: flex-start;
    justify-content: space-between;
    padding: 18px;
    background: var(--el-fill-color-light);
    border-radius: calc(var(--custom-radius) / 2 + 2px);
  }

  .hero-main {
    min-width: 0;
  }

  .hero-main h3 {
    margin: 0;
    font-size: 22px;
    font-weight: 700;
    color: var(--el-text-color-primary);
  }

  .hero-meta {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    align-items: center;
    margin-top: 8px;
    color: var(--el-text-color-secondary);
  }

  .user-name {
    color: var(--el-color-primary);
  }

  .hero-amount {
    min-width: 120px;
    text-align: right;
  }

  .amount {
    font-size: 28px;
    font-weight: 700;
    line-height: 1;
    color: var(--el-text-color-primary);
  }

  .currency {
    margin-top: 6px;
    color: var(--el-text-color-secondary);
  }

  .detail-section {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .section-title {
    font-size: 15px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  .event-card {
    padding: 12px 14px;
    background: var(--el-bg-color-page);
    border: 1px solid var(--el-border-color-light);
    border-radius: calc(var(--custom-radius) / 2 + 2px);
  }

  .event-header {
    display: flex;
    gap: 12px;
    justify-content: space-between;
  }

  .event-title-wrap {
    display: flex;
    gap: 10px;
    align-items: center;
  }

  .event-title {
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  .event-time {
    font-size: 12px;
    color: var(--el-text-color-secondary);
    white-space: nowrap;
  }

  .event-dot {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 30px;
    height: 30px;
    font-size: 14px;
    border-radius: 999px;
  }

  .event-order_created,
  .event-provisioning_started,
  .event-payment_created,
  .event-status_changed {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
  }

  .event-order_paid,
  .event-order_approved,
  .event-provisioning_completed,
  .event-payment_approved {
    color: var(--el-color-success);
    background: var(--el-color-success-light-9);
  }

  .event-order_rejected,
  .event-provisioning_failed,
  .event-payment_rejected {
    color: var(--el-color-danger);
    background: var(--el-color-danger-light-9);
  }

  .event-provisioning_progress {
    color: var(--el-color-warning);
    background: var(--el-color-warning-light-9);
  }

  .event-line {
    margin-top: 10px;
    line-height: 1.6;
    color: var(--el-text-color-regular);
  }

  .label {
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  .event-collapse {
    margin-top: 8px;
  }

  .event-data-pre {
    padding: 10px 12px;
    margin: 0;
    overflow: auto;
    font-size: 12px;
    line-height: 1.6;
    color: var(--el-text-color-secondary);
    word-break: break-word;
    white-space: pre-wrap;
    background: var(--el-fill-color-light);
    border-radius: calc(var(--custom-radius) / 3 + 2px);
  }

  .drawer-actions {
    display: flex;
    flex-wrap: wrap;
    gap: 12px;
    padding-top: 8px;
  }

  @media (width <= 768px) {
    .detail-hero,
    .event-header {
      flex-direction: column;
    }

    .hero-amount {
      text-align: left;
    }
  }
</style>
