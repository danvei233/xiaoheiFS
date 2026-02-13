<template>
  <div class="page">
    <div class="page-header">
      <div>
        <div class="page-title">订单审核</div>
        <div class="subtle">待审核订单与开通进度跟踪</div>
      </div>
    </div>

    <FilterBar
      v-model:filters="filters"
      :status-options="statusOptions"
      :status-tabs="statusTabs"
      @search="fetchData"
      @refresh="fetchData"
      @reset="fetchData"
      @export="exportCsv"
    >
      <template #advanced>
        <a-space direction="vertical" style="width: 260px">
          <a-input v-model:value="filters.user_id" placeholder="用户 ID" />
          <a-input v-model:value="filters.order_no" placeholder="订单号" />
        </a-space>
      </template>
    </FilterBar>

    <ProTable
      :columns="columns"
      :data-source="dataSource"
      :loading="loading"
      :pagination="pagination"
      selectable
      @change="onTableChange"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'status'">
          <OrderStatusBadge :status="record.status" />
        </template>
        <template v-else-if="column.key === 'action'">
          <a-space>
            <a-button type="link" @click="openDetail(record)">详情</a-button>
            <a-button type="link" :disabled="isReviewLocked(record)" @click="approve(record)">通过</a-button>
            <a-button type="link" danger :disabled="isReviewLocked(record)" @click="reject(record)">驳回</a-button>
            <a-button v-if="canDelete" type="link" danger @click="removeOrder(record)">删除</a-button>
          </a-space>
        </template>
      </template>

      <template #mobile>
        <a-space direction="vertical" style="width: 100%" :size="12">
          <a-card v-for="item in dataSource" :key="item.id" class="card">
            <div class="flex space-between" style="margin-bottom: 8px">
              <strong>#{{ item.order_no || item.id }}</strong>
              <OrderStatusBadge :status="item.status" />
            </div>
            <div class="subtle">用户: {{ item.user_id }}</div>
            <div class="subtle">金额: {{ item.total_amount }}</div>
            <a-space style="margin-top: 8px">
              <a-button size="small" @click="openDetail(item)">详情</a-button>
              <a-button size="small" type="primary" :disabled="isReviewLocked(item)" @click="approve(item)">通过</a-button>
              <a-button v-if="canDelete" size="small" danger @click="removeOrder(item)">删除</a-button>
            </a-space>
          </a-card>
        </a-space>
      </template>
    </ProTable>

    <a-drawer v-model:open="detailOpen" width="600" title="订单详情" @close="stopPolling">
      <a-descriptions :column="2" bordered size="small">
        <a-descriptions-item label="订单 ID">{{ detail?.order?.id || '-' }}</a-descriptions-item>
        <a-descriptions-item label="订单号">{{ detail?.order?.order_no || '-' }}</a-descriptions-item>
        <a-descriptions-item label="用户">
          {{ detail?.order?.user_id || '-' }}
          <span v-if="userInfo?.username" class="user-name">（{{ userInfo.username }}）</span>
        </a-descriptions-item>
        <a-descriptions-item label="状态">
          <OrderStatusBadge :status="detail?.order?.status || ''" />
        </a-descriptions-item>
        <a-descriptions-item label="金额">
          {{ detail?.order?.total_amount || '-' }} {{ detail?.order?.currency || 'CNY' }}
        </a-descriptions-item>
        <a-descriptions-item label="创建时间">
          {{ formatDateTime(detail?.order?.created_at) }}
        </a-descriptions-item>
        <a-descriptions-item v-if="detail?.order?.approved_by" label="审核人">
          {{ detail?.order?.approved_by }}
        </a-descriptions-item>
        <a-descriptions-item v-if="detail?.order?.approved_at" label="审核时间">
          {{ formatDateTime(detail?.order?.approved_at) }}
        </a-descriptions-item>
        <a-descriptions-item v-if="detail?.order?.rejected_reason" label="驳回原因" :span="2">
          {{ detail?.order?.rejected_reason }}
        </a-descriptions-item>
      </a-descriptions>

      <a-divider orientation="left">付款信息</a-divider>
      <a-table
        :columns="paymentColumns"
        :data-source="detailPayments"
        :pagination="false"
        row-key="id"
        size="small"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'status'">
            <a-tag :color="getPaymentStatusColor(record.status)">
              {{ getPaymentStatusText(record.status) }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'note'">
            <span>{{ record.note || '-' }}</span>
          </template>
        </template>
      </a-table>

      <a-divider orientation="left">订单项</a-divider>
      <a-table
        :columns="itemColumns"
        :data-source="detailItems"
        :pagination="false"
        row-key="id"
        size="small"
      />

      <a-divider orientation="left">事件流</a-divider>
      <a-timeline v-if="detailEvents.length" mode="left">
        <a-timeline-item v-for="ev in detailEvents" :key="ev.id">
          <template #dot>
            <span :class="['event-dot', `event-${ev.type}`]">
              <component :is="getEventIcon(ev.type)" />
            </span>
          </template>
          <div class="event-content">
            <div class="event-header">
              <span class="event-type">{{ getEventTypeText(ev.type) }}</span>
              <span class="event-time">{{ formatTime(ev.created_at) }}</span>
            </div>
            <div v-if="ev.data" class="event-details">
              <div v-if="ev.data.admin_id" class="event-operator">
                <a-tag color="blue">管理员 {{ ev.data.admin_id }}</a-tag>
              </div>
              <div v-if="ev.data.user_id && !ev.data.admin_id" class="event-operator">
                <a-tag>
                  用户 {{ ev.data.user_id }}
                  <span v-if="ev.data.user_id === detail?.order?.user_id && userInfo?.username" class="user-name-inline">（{{ userInfo.username }}）</span>
                </a-tag>
              </div>
              <div v-if="ev.data.reason" class="event-reason">
                <span class="label">原因：</span>{{ ev.data.reason }}
              </div>
              <div v-if="ev.data.message" class="event-message">
                <span class="label">消息：</span>{{ ev.data.message }}
              </div>
              <a-collapse v-if="hasMoreData(ev.data)" ghost size="small" style="margin-top: 4px">
                <a-collapse-panel header="详细信息">
                  <pre class="event-data-pre">{{ formatEventData(ev.data) }}</pre>
                </a-collapse-panel>
              </a-collapse>
            </div>
          </div>
        </a-timeline-item>
      </a-timeline>
      <a-empty v-else description="暂无事件" :image="Empty.PRESENTED_IMAGE_SIMPLE" />

      <template #footer>
        <a-space style="margin-top: 12px">
          <a-button type="primary" :disabled="isReviewLocked(detail?.order)" @click="approve(detail?.order)">通过</a-button>
          <a-button danger :disabled="isReviewLocked(detail?.order)" @click="reject(detail?.order)">驳回</a-button>
          <a-button :disabled="!canRetry(detail?.order)" @click="retry(detail?.order)">重试开通</a-button>
          <a-button v-if="canDelete" danger @click="removeOrder(detail?.order)">删除订单</a-button>
        </a-space>
      </template>
    </a-drawer>

    <a-modal v-model:open="rejectOpen" title="驳回订单" @ok="submitReject">
      <a-textarea v-model:value="rejectReason" rows="3" placeholder="请输入驳回原因" :maxlength="INPUT_LIMITS.REVIEW_REASON" show-count />
    </a-modal>
  </div>
</template>

<script setup>
import { reactive, ref, computed, h } from "vue";
import FilterBar from "@/components/FilterBar.vue";
import ProTable from "@/components/ProTable.vue";
import OrderStatusBadge from "@/components/OrderStatusBadge.vue";
import { listAdminOrders, approveAdminOrder, rejectAdminOrder, retryAdminOrder, deleteAdminOrder, getAdminOrderDetail, getAdminUserDetail } from "@/services/admin";
import { Modal, message, Empty } from "ant-design-vue";
import {
  ClockCircleOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  LoadingOutlined,
  ExclamationCircleOutlined,
  DollarOutlined,
  SyncOutlined,
  RocketOutlined
} from "@ant-design/icons-vue";
import dayjs from "dayjs";
import { useAdminAuthStore } from "@/stores/adminAuth";
import { INPUT_LIMITS } from "@/constants/inputLimits";

const filters = reactive({
  keyword: "",
  status: undefined,
  range: [],
  user_id: "",
  order_no: ""
});

const statusOptions = [
  { label: "待支付", value: "pending_payment" },
  { label: "待审核", value: "pending_review" },
  { label: "已通过", value: "approved" },
  { label: "开通中", value: "provisioning" },
  { label: "已完成", value: "active" },
  { label: "失败", value: "failed" },
  { label: "已驳回", value: "rejected" }
];

const statusTabs = [
  { label: "待审核", value: "pending_review" },
  { label: "开通中", value: "provisioning" },
  { label: "失败", value: "failed" }
];

const loading = ref(false);
const dataSource = ref([]);
const pagination = reactive({ current: 1, pageSize: 20, total: 0, showSizeChanger: true });
const adminAuth = useAdminAuthStore();
const canDelete = computed(() => {
  const perms = adminAuth.profile?.permissions || [];
  return perms.includes("*") || perms.includes("order.delete");
});

const columns = [
  { title: "订单 ID", dataIndex: "id", key: "id", sorter: true, width: 80 },
  { title: "用户 ID", dataIndex: "user_id", key: "user_id", width: 80 },
  { title: "订单号", dataIndex: "order_no", key: "order_no", width: 150 },
  { title: "状态", dataIndex: "status", key: "status", width: 100 },
  { title: "总金额", dataIndex: "total_amount", key: "total_amount", width: 100 },
  { title: "创建时间", dataIndex: "created_at", key: "created_at", sorter: true, width: 160 },
  { title: "操作", key: "action", width: 180, fixed: "right" }
];

const normalizeOrder = (row) => ({
  id: row.id ?? row.ID,
  user_id: row.user_id ?? row.UserID,
  order_no: row.order_no ?? row.OrderNo,
  status: row.status ?? row.Status,
  can_review: row.can_review ?? row.CanReview ?? false,
  total_amount: row.total_amount ?? row.TotalAmount,
  created_at: row.created_at ?? row.CreatedAt
});

const fetchData = async () => {
  loading.value = true;
  try {
    const res = await listAdminOrders({
      limit: pagination.pageSize,
      offset: (pagination.current - 1) * pagination.pageSize,
      status: filters.status || undefined,
      user_id: filters.user_id || undefined,
      order_no: filters.order_no || undefined
    });
    const payload = res.data || {};
    let items = (payload.items || []).map(normalizeOrder);
    if (filters.keyword) {
      items = items.filter((item) => String(item.id).includes(filters.keyword));
    }
    dataSource.value = items;
    pagination.total = payload.total || items.length;
  } finally {
    loading.value = false;
  }
};

const onTableChange = (pager) => {
  pagination.current = pager.current;
  pagination.pageSize = pager.pageSize;
  fetchData();
};

const exportCsv = () => {
  const csv = "id,status,total_amount\n" + dataSource.value.map((i) => `${i.id},${i.status},${i.total_amount}`).join("\n");
  const blob = new Blob([csv], { type: "text/csv;charset=utf-8;" });
  const link = document.createElement("a");
  link.href = URL.createObjectURL(blob);
  link.download = "admin-orders.csv";
  link.click();
};

const approve = (record) => {
  if (!record?.id) return;
  Modal.confirm({
    title: "确认通过该订单?",
    onOk: async () => {
      await approveAdminOrder(record.id);
      message.success("已通过");
      fetchData();
      loadDetail(record.id);
    }
  });
};

const rejectOpen = ref(false);
const rejectReason = ref("");
const rejectTarget = ref(null);

const reject = (record) => {
  if (!record?.id) return;
  rejectTarget.value = record;
  rejectReason.value = "";
  rejectOpen.value = true;
};

const submitReject = async () => {
  if (!rejectTarget.value?.id) return;
  if (String(rejectReason.value || "").length > INPUT_LIMITS.REVIEW_REASON) {
    message.error(`驳回原因长度不能超过 ${INPUT_LIMITS.REVIEW_REASON} 个字符`);
    return;
  }
  await rejectAdminOrder(rejectTarget.value.id, { reason: rejectReason.value || "manual" });
  message.success("已驳回");
  rejectOpen.value = false;
  fetchData();
  loadDetail(rejectTarget.value.id);
};

const retry = async (record) => {
  if (!record?.id) return;
  await retryAdminOrder(record.id);
  message.success("已触发重试");
  fetchData();
  loadDetail(record.id);
};

const canRetry = (record) => {
  const raw = record?.status ?? record?.Status ?? "";
  const status = String(raw).trim().toLowerCase();
  return ["approved", "provisioning", "failed"].includes(status);
};

const removeOrder = (record) => {
  if (!record?.id) return;
  Modal.confirm({
    title: "确认删除该订单?",
    content: "该操作会删除订单及其关联记录，无法恢复。",
    okText: "删除",
    okType: "danger",
    onOk: async () => {
      await deleteAdminOrder(record.id);
      message.success("订单已删除");
      if (detail.value?.order?.id === record.id) {
        detailOpen.value = false;
        stopPolling();
      }
      fetchData();
    }
  });
};

const detailOpen = ref(false);
const detail = ref({});
const userInfo = ref(null);
const detailItems = computed(() => detail.value?.items || []);
const detailPayments = computed(() => detail.value?.payments || []);
const detailEvents = computed(() => {
  const events = detail.value?.events || detail.value?.logs || [];
  return events.map(ev => {
    // 解析 data 字段（可能是 JSON 字符串或已经是对象）
    let parsedData = ev.data;
    if (typeof ev.data === 'string') {
      try {
        parsedData = JSON.parse(ev.data);
      } catch (e) {
        parsedData = {};
      }
    }
    return {
      ...ev,
      data: parsedData
    };
  });
});

const itemColumns = [
  { title: "ID", dataIndex: "id", key: "id", width: 60 },
  { title: "套餐 ID", dataIndex: "package_id", key: "package_id", width: 80 },
  { title: "系统 ID", dataIndex: "system_id", key: "system_id", width: 80 },
  { title: "数量", dataIndex: "qty", key: "qty", width: 60 },
  { title: "金额", dataIndex: "amount", key: "amount", width: 80 },
  { title: "状态", dataIndex: "status", key: "status", width: 100 }
];

const paymentColumns = [
  { title: "方式", dataIndex: "method", key: "method", width: 100 },
  { title: "金额", dataIndex: "amount", key: "amount", width: 80 },
  { title: "交易号", dataIndex: "trade_no", key: "trade_no", ellipsis: true },
  { title: "备注", dataIndex: "note", key: "note", ellipsis: true },
  { title: "状态", dataIndex: "status", key: "status", width: 100 }
];

let poller;
const loadDetail = async (id) => {
  if (!id) return;
  const res = await getAdminOrderDetail(id);
  detail.value = res.data || {};

  // 获取用户信息
  const userId = detail.value?.order?.user_id;
  if (userId) {
    try {
      const userRes = await getAdminUserDetail(userId);
      userInfo.value = userRes.data || {};
    } catch (e) {
      userInfo.value = null;
    }
  }
};

const startPolling = (id) => {
  stopPolling();
  poller = setInterval(() => loadDetail(id), 5000);
};

const stopPolling = () => {
  if (poller) clearInterval(poller);
};

const openDetail = async (record) => {
  await loadDetail(record.id);
  detailOpen.value = true;
  startPolling(record.id);
};

// 格式化时间
const formatDateTime = (dateStr) => {
  if (!dateStr) return '-';
  return dayjs(dateStr).format('YYYY-MM-DD HH:mm:ss');
};

const formatTime = (dateStr) => {
  if (!dateStr) return '';
  return dayjs(dateStr).format('HH:mm:ss');
};

// 支付状态
const getPaymentStatusColor = (status) => {
  const colors = {
    pending: 'default',
    pending_review: 'orange',
    approved: 'green',
    rejected: 'red',
    paid: 'success'
  };
  return colors[status] || 'default';
};

const getPaymentStatusText = (status) => {
  const texts = {
    pending: '待支付',
    pending_review: '待审核',
    approved: '已通过',
    rejected: '已驳回',
    paid: '已支付'
  };
  return texts[status] || status;
};

// 事件类型映射
const eventTypeMap = {
  order_created: { text: '订单创建', icon: ClockCircleOutlined, color: '#1890ff' },
  order_paid: { text: '订单支付', icon: DollarOutlined, color: '#52c41a' },
  order_approved: { text: '订单通过', icon: CheckCircleOutlined, color: '#52c41a' },
  order_rejected: { text: '订单驳回', icon: CloseCircleOutlined, color: '#ff4d4f' },
  provisioning_started: { text: '开始开通', icon: RocketOutlined, color: '#1890ff' },
  provisioning_progress: { text: '开通进度', icon: LoadingOutlined, color: '#faad14' },
  provisioning_completed: { text: '开通完成', icon: CheckCircleOutlined, color: '#52c41a' },
  provisioning_failed: { text: '开通失败', icon: ExclamationCircleOutlined, color: '#ff4d4f' },
  payment_created: { text: '支付创建', icon: DollarOutlined, color: '#1890ff' },
  payment_approved: { text: '支付通过', icon: CheckCircleOutlined, color: '#52c41a' },
  payment_rejected: { text: '支付驳回', icon: CloseCircleOutlined, color: '#ff4d4f' },
  status_changed: { text: '状态变更', icon: SyncOutlined, color: '#1890ff' }
};

const getEventTypeText = (type) => {
  return eventTypeMap[type]?.text || type;
};

const getEventIcon = (type) => {
  return eventTypeMap[type]?.icon || ClockCircleOutlined;
};

const formatEventData = (data) => {
  if (!data) return '';
  if (typeof data === 'string') {
    try {
      data = JSON.parse(data);
    } catch {
      return data;
    }
  }
  // 格式化 JSON 数据，使其更易读
  const formatted = {};
  for (const [key, value] of Object.entries(data)) {
    if (key === 'admin_id') {
      formatted[key] = `管理员(${value})`;
    } else if (key === 'user_id') {
      formatted[key] = `用户(${value})`;
    } else if (key === 'reason') {
      formatted['原因'] = value;
    } else if (key === 'message') {
      formatted['消息'] = value;
    } else if (key === 'from_status' || key === 'to_status') {
      formatted[key === 'from_status' ? '原状态' : '新状态'] = value;
    } else {
      formatted[key] = value;
    }
  }
  return JSON.stringify(formatted, null, 2);
};

const hasMoreData = (data) => {
  if (!data) return false;
  const keys = Object.keys(data);
  // 只有没有这些常见字段时才显示详细信息
  const commonKeys = ['admin_id', 'user_id', 'reason', 'message'];
  return keys.some(key => !commonKeys.includes(key));
};

const isReviewLocked = (record) => {
  if (record?.can_review !== undefined || record?.CanReview !== undefined) {
    return !(record?.can_review ?? record?.CanReview);
  }
  const raw = record?.status ?? record?.Status ?? "";
  const status = String(raw).trim().toLowerCase();
  return !["pending_review", "pending-review", "pending review"].includes(status);
};

fetchData();
</script>

<style scoped>
.user-name {
  color: #1890ff;
  font-weight: 500;
}

.user-name-inline {
  color: #1890ff;
  margin-left: 2px;
}

.event-dot {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: #f0f0f0;
  font-size: 16px;
}

.event-order_created { background: #e6f7ff; color: #1890ff; }
.event-order_paid { background: #f6ffed; color: #52c41a; }
.event-order_approved { background: #f6ffed; color: #52c41a; }
.event-order_rejected { background: #fff2f0; color: #ff4d4f; }
.event-provisioning_started { background: #e6f7ff; color: #1890ff; }
.event-provisioning_progress { background: #fffbe6; color: #faad14; }
.event-provisioning_completed { background: #f6ffed; color: #52c41a; }
.event-provisioning_failed { background: #fff2f0; color: #ff4d4f; }
.event-payment_created { background: #e6f7ff; color: #1890ff; }
.event-payment_approved { background: #f6ffed; color: #52c41a; }
.event-payment_rejected { background: #fff2f0; color: #ff4d4f; }
.event-status_changed { background: #e6f7ff; color: #1890ff; }

.event-content {
  padding-left: 8px;
}

.event-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.event-type {
  font-weight: 500;
  font-size: 14px;
}

.event-time {
  font-size: 12px;
  color: #999;
}

.event-details {
  margin-top: 8px;
}

.event-operator {
  margin-bottom: 6px;
}

.event-operator .ant-tag {
  margin: 0;
}

.event-reason,
.event-message {
  font-size: 13px;
  color: #666;
  margin-bottom: 4px;
}

.event-reason .label,
.event-message .label {
  font-weight: 500;
  color: #333;
}

.event-data-pre {
  margin: 0;
  padding: 8px 12px;
  background: #f5f5f5;
  border-radius: 4px;
  font-size: 12px;
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-word;
  color: #666;
}

:deep(.ant-collapse-header) {
  padding: 4px 0 !important;
  font-size: 12px;
}

:deep(.ant-collapse-content-box) {
  padding: 8px 0 !important;
}
</style>
