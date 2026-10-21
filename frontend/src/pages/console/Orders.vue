<template>
  <div class="orders-page">
    <!-- Header -->
    <div class="page-header">
      <div class="header-content">
        <h1 class="page-title">我的订单</h1>
        <p class="page-subtitle">{{ pagination.total }} 个订单</p>
      </div>
      <div class="header-actions">
        <a-button @click="fetchData" :loading="loading">
          <ReloadOutlined />
          刷新
        </a-button>
      </div>
    </div>

    <!-- Status Tabs -->
    <div class="status-tabs">
      <a-segmented
        v-model:value="activeTab"
        :options="tabOptions"
        @change="onTabChange"
        size="large"
      />
    </div>

    <!-- Table -->
    <div class="table-section">
      <a-table
        :columns="columns"
        :data-source="dataSource"
        :loading="loading"
        :pagination="{
          current: pagination.current,
          pageSize: pagination.pageSize,
          total: pagination.total,
          showSizeChanger: true,
          showQuickJumper: true,
          showTotal: (total) => `共 ${total} 条`
        }"
        @change="onTableChange"
        class="orders-table"
      >
        <template #bodyCell="{ column, record }">
          <!-- Order ID -->
          <template v-if="column.key === 'id'">
            <div class="order-id-cell">
              <FileTextOutlined class="cell-icon" />
              <div>
                <div class="order-id">#{{ record.id }}</div>
                <div class="order-no">{{ record.order_no || '-' }}</div>
              </div>
            </div>
          </template>

          <!-- Status -->
          <template v-else-if="column.key === 'status'">
            <OrderStatusBadge :status="record.status" />
          </template>

          <!-- Amount -->
          <template v-else-if="column.key === 'amount'">
            <div class="amount-cell">
              <span class="amount-symbol">¥</span>
              <span class="amount-value">{{ record.total_amount }}</span>
            </div>
          </template>

          <!-- Created At -->
          <template v-else-if="column.key === 'created_at'">
            <div class="date-cell">
              <CalendarOutlined class="date-icon" />
              <span>{{ record.created_at }}</span>
            </div>
          </template>

          <!-- Actions -->
          <template v-else-if="column.key === 'actions'">
            <a-space>
              <a-button type="link" @click="goDetail(record)">
                <EyeOutlined />
                详情
              </a-button>
              <a-button
                v-if="canCancel(record)"
                type="link"
                danger
                @click="cancelItem(record)"
              >
                <CloseCircleOutlined />
                撤销
              </a-button>
            </a-space>
          </template>
        </template>
      </a-table>
    </div>

    <!-- Mobile Cards -->
    <div class="mobile-cards">
      <a-card
        v-for="item in dataSource"
        :key="item.id"
        class="order-card"
      >
        <template #title>
          <div class="card-title-row">
            <FileTextOutlined class="card-title-icon" />
            <span class="card-order-id">#{{ item.id }}</span>
            <OrderStatusBadge :status="item.status" />
          </div>
        </template>

        <div class="card-body">
          <div class="card-row">
            <span class="card-label">订单号</span>
            <span class="card-value">{{ item.order_no || '-' }}</span>
          </div>
          <div class="card-row">
            <span class="card-label">金额</span>
            <span class="card-value card-amount">¥{{ item.total_amount }}</span>
          </div>
          <div class="card-row">
            <span class="card-label">创建时间</span>
            <span class="card-value">{{ item.created_at }}</span>
          </div>
          <div class="card-actions">
            <a-button size="small" @click="goDetail(item)">
              <EyeOutlined />
              详情
            </a-button>
            <a-button
              v-if="canCancel(item)"
              size="small"
              danger
              @click="cancelItem(item)"
            >
              <CloseCircleOutlined />
              撤销
            </a-button>
          </div>
        </div>
      </a-card>
    </div>
  </div>
</template>

<script setup>
import { reactive, computed, ref } from "vue";
import OrderStatusBadge from "@/components/OrderStatusBadge.vue";
import { useRouter } from "vue-router";
import { useOrdersStore } from "@/stores/orders";
import { cancelOrder } from "@/services/user";
import { message, Modal } from "ant-design-vue";
import {
  ReloadOutlined,
  FileTextOutlined,
  EyeOutlined,
  CloseCircleOutlined,
  CalendarOutlined
} from "@ant-design/icons-vue";

const router = useRouter();
const store = useOrdersStore();

const activeTab = ref('all');
const pagination = reactive({ current: 1, pageSize: 10, total: 0 });

const tabOptions = [
  { label: '全部', value: 'all' },
  { label: '待支付', value: 'pending_payment' },
  { label: '待审核', value: 'pending_review' },
  { label: '开通中', value: 'provisioning' },
  { label: '已完成', value: 'active' }
];

const columns = [
  { title: '订单', dataIndex: 'id', key: 'id', width: 200 },
  { title: '状态', dataIndex: 'status', key: 'status', width: 140 },
  { title: '金额', dataIndex: 'total_amount', key: 'amount', width: 140 },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at' },
  { title: '操作', key: 'actions', width: 180, align: 'right' }
];

const dataSource = computed(() =>
  store.items.map((row) => ({
    id: row.id ?? row.ID,
    order_no: row.order_no ?? row.OrderNo,
    status: row.status ?? row.Status,
    total_amount: row.total_amount ?? row.TotalAmount,
    created_at: row.created_at ?? row.CreatedAt
  }))
);

const loading = computed(() => store.loading);

const fetchData = () => {
  const status = activeTab.value === 'all' ? undefined : activeTab.value;
  store
    .fetchOrders({
      limit: pagination.pageSize,
      offset: (pagination.current - 1) * pagination.pageSize,
      status
    })
    .then(() => {
      pagination.total = store.total;
    });
};

const onTableChange = (pager) => {
  pagination.current = pager.current;
  pagination.pageSize = pager.pageSize;
  fetchData();
};

const onTabChange = (value) => {
  activeTab.value = value;
  pagination.current = 1;
  fetchData();
};

const goDetail = (record) => router.push(`/console/orders/${record.id}`);

const canCancel = (record) => ["pending_payment", "pending_review"].includes(record?.status);

const cancelItem = (record) => {
  if (!record?.id) return;
  Modal.confirm({
    title: "撤销订单",
    content: "撤销后订单将变为已取消，无法继续支付。确认撤销吗？",
    okText: "确认撤销",
    okButtonProps: { danger: true },
    cancelText: "暂不撤销",
    onOk: async () => {
      await cancelOrder(record.id);
      message.success("订单已撤销");
      fetchData();
    }
  });
};

fetchData();
</script>

<style scoped>
.orders-page {
  padding: 24px;
  max-width: 1400px;
  margin: 0 auto;
}

/* Header */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.header-content {
  display: flex;
  align-items: baseline;
  gap: 16px;
}

.page-title {
  font-size: 28px;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0;
}

.page-subtitle {
  font-size: 14px;
  color: var(--text-secondary);
  margin: 0;
}

.header-actions :deep(.ant-btn) {
  height: 40px;
  padding: 0 20px;
  font-weight: 500;
}

/* Status Tabs */
.status-tabs {
  margin-bottom: 24px;
  display: flex;
}

.status-tabs :deep(.ant-segmented) {
  padding: 4px;
  background: var(--bg-secondary);
  border-radius: var(--radius-md);
}

.status-tabs :deep(.ant-segmented-item) {
  border-radius: var(--radius-sm);
  padding: 8px 20px;
  font-weight: 500;
  transition: all var(--transition-base);
}

/* Table Section */
.table-section {
  background: var(--card);
  border-radius: var(--radius-lg);
  border: 1px solid var(--border);
  overflow: hidden;
}

.orders-table :deep(.ant-table) {
  background: transparent;
}

.orders-table :deep(.ant-table-thead > tr > th) {
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border);
  padding: 14px 16px;
  font-weight: 600;
  font-size: 13px;
  color: var(--text-secondary);
}

.orders-table :deep(.ant-table-tbody > tr > td) {
  padding: 16px;
  border-bottom: 1px solid var(--border-light);
}

.orders-table :deep(.ant-table-tbody > tr:hover > td) {
  background: var(--bg-secondary);
}

.orders-table :deep(.ant-table-tbody > tr:last-child > td) {
  border-bottom: none;
}

/* Order ID Cell */
.order-id-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}

.cell-icon {
  font-size: 18px;
  color: var(--primary);
}

.order-id {
  font-weight: 600;
  font-size: 14px;
  color: var(--text-primary);
}

.order-no {
  font-size: 12px;
  color: var(--text-tertiary);
  font-family: 'JetBrains Mono', monospace;
}

/* Amount Cell */
.amount-cell {
  display: flex;
  align-items: baseline;
  justify-content: flex-start;
  gap: 2px;
}

.amount-symbol {
  font-size: 14px;
  color: var(--text-tertiary);
}

.amount-value {
  font-size: 16px;
  font-weight: 700;
  color: var(--primary);
}

/* Date Cell */
.date-cell {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--text-secondary);
}

.date-icon {
  font-size: 14px;
  color: var(--text-tertiary);
}

/* Mobile Cards */
.mobile-cards {
  display: none;
}

.order-card {
  margin-bottom: 16px;
  border-radius: var(--radius-lg);
  border: 1px solid var(--border);
  overflow: hidden;
}

.order-card :deep(.ant-card-head) {
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border);
  padding: 14px 16px;
}

.order-card :deep(.ant-card-body) {
  padding: 16px;
}

.card-title-row {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 600;
}

.card-title-icon {
  color: var(--primary);
  font-size: 16px;
}

.card-order-id {
  flex: 1;
  color: var(--text-primary);
}

.card-body {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.card-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 13px;
}

.card-label {
  color: var(--text-secondary);
}

.card-value {
  font-weight: 500;
  color: var(--text-primary);
}

.card-value.card-amount {
  font-weight: 700;
  color: var(--primary);
}

.card-actions {
  display: flex;
  gap: 8px;
  padding-top: 12px;
  border-top: 1px solid var(--border-light);
}

.card-actions :deep(.ant-btn) {
  flex: 1;
}

/* Responsive */
@media (max-width: 768px) {
  .orders-page {
    padding: 16px;
  }

  .page-header {
    flex-direction: column;
    align-items: stretch;
    gap: 16px;
  }

  .header-content {
    flex-direction: column;
    gap: 4px;
  }

  .page-title {
    font-size: 22px;
  }

  .table-section {
    display: none;
  }

  .mobile-cards {
    display: block;
  }

  .status-tabs :deep(.ant-segmented-item) {
    padding: 6px 12px;
    font-size: 13px;
  }
}
</style>
