<template>
  <div class="wallet-orders-page">
    <div class="page-header">
      <div>
        <div class="page-title">钱包订单</div>
        <div class="page-subtitle">充值与提现申请的审核记录</div>
      </div>
      <a-space>
        <a-button @click="resetFilters">重置筛选</a-button>
        <a-button type="primary" @click="fetchData" :loading="loading">
          <template #icon><ReloadOutlined /></template>
          刷新
        </a-button>
      </a-space>
    </div>

    <a-card :bordered="false" class="filter-card">
      <a-space wrap size="large">
        <div class="filter-item">
          <span class="filter-label">状态</span>
          <a-select
            v-model:value="filters.status"
            placeholder="全部"
            style="width: 140px"
            allow-clear
            @change="handleFilterChange"
          >
            <a-select-option value="">全部</a-select-option>
            <a-select-option value="pending_review">待审核</a-select-option>
            <a-select-option value="approved">已通过</a-select-option>
            <a-select-option value="rejected">已拒绝</a-select-option>
          </a-select>
        </div>
        <div class="filter-item">
          <span class="filter-label">用户 ID</span>
          <a-input
            v-model:value="filters.user_id"
            placeholder="输入用户 ID"
            style="width: 160px"
            @pressEnter="handleFilterChange"
            @blur="handleFilterChange"
          />
        </div>
      </a-space>
    </a-card>

    <a-card :bordered="false" class="table-card">
      <ProTable
        :columns="columns"
        :data-source="orders"
        :loading="loading"
        :pagination="pagination"
        @change="handleTableChange"
        row-key="id"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'type'">
            <a-tag :color="record.type === 'recharge' ? 'green' : record.type === 'withdraw' ? 'orange' : 'blue'">
              {{ getTypeText(record.type) }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'status'">
            <a-tag :color="getStatusColor(record.status)">
              {{ getStatusText(record.status) }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'amount'">
            <span :style="{ color: record.type === 'withdraw' ? '#ff4d4f' : '#52c41a' }">
              {{ record.type === 'withdraw' ? '-' : '+' }}￥{{ formatAmount(record.amount) }}
            </span>
          </template>
          <template v-else-if="column.key === 'created_at'">
            {{ formatDate(record.created_at) }}
          </template>
          <template v-else-if="column.key === 'actions'">
            <a-space>
              <a-button
                v-if="record.status === 'pending_review'"
                type="link"
                size="small"
                @click="handleApprove(record)"
              >
                通过
              </a-button>
              <a-button
                v-if="record.status === 'pending_review'"
                type="link"
                danger
                size="small"
                @click="handleReject(record)"
              >
                拒绝
              </a-button>
              <a-typography-text v-else type="secondary">-</a-typography-text>
            </a-space>
          </template>
        </template>
      </ProTable>
    </a-card>

    <a-modal
      v-model:open="rejectModalVisible"
      title="拒绝订单"
      @ok="confirmReject"
      :confirm-loading="rejecting"
    >
      <a-form layout="vertical">
        <a-form-item label="拒绝原因">
          <a-textarea v-model:value="rejectReason" placeholder="请输入拒绝原因" :rows="3" :maxlength="INPUT_LIMITS.REVIEW_REASON" show-count />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from "vue";
import { message } from "ant-design-vue";
import { ReloadOutlined } from "@ant-design/icons-vue";
import ProTable from "@/components/ProTable.vue";
import { listAdminWalletOrders, approveAdminWalletOrder, rejectAdminWalletOrder } from "@/services/admin";
import { INPUT_LIMITS } from "@/constants/inputLimits";

const loading = ref(false);
const rejecting = ref(false);
const rejectModalVisible = ref(false);
const rejectReason = ref("");
const currentOrder = ref<any>(null);

const orders = ref<any[]>([]);

const filters = reactive({
  status: "",
  user_id: ""
});

const pagination = reactive({
  current: 1,
  pageSize: 20,
  total: 0
});

const columns = [
  { title: "ID", dataIndex: "id", key: "id", width: 80 },
  { title: "用户 ID", dataIndex: "user_id", key: "user_id", width: 100 },
  { title: "类型", dataIndex: "type", key: "type", width: 110 },
  { title: "金额", dataIndex: "amount", key: "amount", width: 140 },
  { title: "备注", dataIndex: "note", key: "note", ellipsis: true },
  { title: "状态", dataIndex: "status", key: "status", width: 120 },
  { title: "创建时间", dataIndex: "created_at", key: "created_at", width: 180 },
  { title: "操作", key: "actions", width: 140, fixed: "right" }
];

const formatAmount = (amount: number) => Number(amount || 0).toFixed(2);

const getTypeText = (type: string) => {
  switch (type) {
    case "recharge":
      return "充值";
    case "withdraw":
      return "提现";
    case "refund":
      return "退款";
    default:
      return type || "-";
  }
};

const getStatusColor = (status: string) => {
  switch (status) {
    case "pending_review":
      return "orange";
    case "approved":
      return "success";
    case "rejected":
      return "error";
    default:
      return "default";
  }
};

const getStatusText = (status: string) => {
  switch (status) {
    case "pending_review":
      return "待审核";
    case "approved":
      return "已通过";
    case "rejected":
      return "已拒绝";
    default:
      return status || "-";
  }
};

const formatDate = (date: string) => {
  if (!date) return "-";
  return new Date(date).toLocaleString("zh-CN");
};

const fetchData = async () => {
  loading.value = true;
  try {
    const params: any = {
      limit: pagination.pageSize,
      offset: (pagination.current - 1) * pagination.pageSize
    };
    if (filters.status) params.status = filters.status;
    if (filters.user_id) params.user_id = filters.user_id;
    const res = await listAdminWalletOrders(params);
    orders.value = res.data?.items || [];
    pagination.total = res.data?.total || 0;
  } finally {
    loading.value = false;
  }
};

const handleTableChange = (pag: any) => {
  pagination.current = pag.current;
  fetchData();
};

const handleFilterChange = () => {
  pagination.current = 1;
  fetchData();
};

const resetFilters = () => {
  filters.status = "";
  filters.user_id = "";
  pagination.current = 1;
  fetchData();
};

const handleApprove = async (record: any) => {
  try {
    await approveAdminWalletOrder(record.id);
    message.success("操作成功");
    fetchData();
  } catch (error: any) {
    message.error(error.response?.data?.error || "操作失败");
  }
};

const handleReject = (record: any) => {
  currentOrder.value = record;
  rejectReason.value = "";
  rejectModalVisible.value = true;
};

const confirmReject = async () => {
  rejecting.value = true;
  try {
    if (String(rejectReason.value || "").length > INPUT_LIMITS.REVIEW_REASON) {
      message.error(`拒绝原因长度不能超过 ${INPUT_LIMITS.REVIEW_REASON} 个字符`);
      return;
    }
    await rejectAdminWalletOrder(currentOrder.value.id, { reason: rejectReason.value });
    message.success("操作成功");
    rejectModalVisible.value = false;
    fetchData();
  } catch (error: any) {
    message.error(error.response?.data?.error || "操作失败");
  } finally {
    rejecting.value = false;
  }
};

onMounted(() => {
  fetchData();
});
</script>

<style scoped>
.wallet-orders-page {
  padding: 24px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.page-title {
  font-size: 20px;
  font-weight: 600;
  margin: 0 0 4px;
}

.page-subtitle {
  color: #8c8c8c;
  font-size: 13px;
}

.filter-card {
  margin-bottom: 16px;
}

.filter-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.filter-label {
  color: #8c8c8c;
  font-size: 13px;
}
</style>
