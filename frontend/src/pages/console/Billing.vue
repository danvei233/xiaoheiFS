<template>
  <div class="billing-page">
    <!-- Balance Card -->
    <div class="balance-card">
      <div class="balance-info">
        <div class="balance-icon">
          <WalletOutlined />
        </div>
        <div class="balance-details">
          <div class="balance-label">当前余额</div>
          <div class="balance-amount">{{ formatMoney(wallet.balance) }}</div>
        </div>
      </div>
      <div class="balance-actions">
        <a-button type="primary" size="large" @click="openRechargeModal">
          <TransactionOutlined />
          充值
        </a-button>
        <a-button size="large" @click="openWithdrawModal">
          <ExportOutlined />
          提现
        </a-button>
        <a-button size="large" @click="fetchAll" :loading="loading">
          <ReloadOutlined />
        </a-button>
      </div>
    </div>

    <!-- Stats Cards -->
    <div class="stats-grid">
      <div class="stat-card recharge-stat">
        <div class="stat-icon">
          <TransactionOutlined />
        </div>
        <div class="stat-info">
          <div class="stat-label">本月充值</div>
          <div class="stat-value">{{ formatMoney(monthStats.recharge) }}</div>
        </div>
      </div>
      <div class="stat-card withdraw-stat">
        <div class="stat-icon">
          <ExportOutlined />
        </div>
        <div class="stat-info">
          <div class="stat-label">本月提现</div>
          <div class="stat-value">{{ formatMoney(monthStats.withdraw) }}</div>
        </div>
      </div>
      <div class="stat-card pending-stat">
        <div class="stat-icon">
          <ClockCircleOutlined />
        </div>
        <div class="stat-info">
          <div class="stat-label">待审核</div>
          <div class="stat-value">{{ monthStats.pending }}</div>
        </div>
      </div>
      <div class="stat-card total-stat">
        <div class="stat-icon">
          <OrderedListOutlined />
        </div>
        <div class="stat-info">
          <div class="stat-label">总交易</div>
          <div class="stat-value">{{ monthStats.total }}</div>
        </div>
      </div>
    </div>

    <!-- Transaction Table -->
    <div class="table-section">
      <div class="table-header">
        <h3 class="table-title">交易记录</h3>
        <a-space>
          <a-select v-model:value="filterStatus" placeholder="状态" style="width: 110px" allow-clear @change="handleFilterChange">
            <a-select-option value="">全部</a-select-option>
            <a-select-option value="pending_review">待处理</a-select-option>
            <a-select-option value="approved">已通过</a-select-option>
            <a-select-option value="rejected">已拒绝</a-select-option>
          </a-select>
          <a-select v-model:value="filterType" placeholder="类型" style="width: 100px" allow-clear @change="handleFilterChange">
            <a-select-option value="">全部</a-select-option>
            <a-select-option value="recharge">充值</a-select-option>
            <a-select-option value="withdraw">提现</a-select-option>
          </a-select>
        </a-space>
      </div>

      <a-table
        :columns="columns"
        :data-source="filteredOrders"
        :loading="loading"
        :scroll="{ x: 780 }"
        :pagination="{
          current: pagination.current,
          pageSize: pagination.pageSize,
          total: pagination.total,
          showSizeChanger: true,
          showTotal: (total) => `共 ${total} 条`
        }"
        @change="handleTableChange"
        class="billing-table"
      >
        <template #bodyCell="{ column, record }">
          <!-- Type -->
          <template v-if="column.key === 'type'">
            <div class="type-cell">
              <TransactionOutlined v-if="record.type === 'recharge'" class="type-icon recharge-icon" />
              <ExportOutlined v-else-if="record.type === 'withdraw'" class="type-icon withdraw-icon" />
              <SwapOutlined v-else-if="record.type === 'refund'" class="type-icon refund-icon" />
              <span>{{ typeLabel(record.type) }}</span>
            </div>
          </template>

          <!-- Amount -->
          <template v-else-if="column.key === 'amount'">
            <span :class="['amount-cell', record.type === 'recharge' ? 'amount-plus' : 'amount-minus']">
              {{ record.type === 'recharge' ? '+' : '-' }}{{ formatMoney(record.amount) }}
            </span>
          </template>

          <!-- Status -->
          <template v-else-if="column.key === 'status'">
            <a-tag v-if="record.status === 'pending' || record.status === 'pending_review'" color="processing">
              <ClockCircleOutlined />
              {{ isPendingPayRecharge(record) ? "待支付" : "待审核" }}
            </a-tag>
            <a-tag v-else-if="record.status === 'approved'" color="success">
              <CheckCircleFilled />
              已通过
            </a-tag>
            <a-tag v-else-if="record.status === 'rejected'" color="error">
              <CloseCircleFilled />
              已拒绝
            </a-tag>
            <a-tag v-else>
              <InfoCircleOutlined />
              {{ record.status }}
            </a-tag>
          </template>

          <!-- Note -->
          <template v-else-if="column.key === 'note'">
            <span class="note-cell">{{ record.note || '-' }}</span>
          </template>

          <!-- Time -->
          <template v-else-if="column.key === 'created_at'">
            <div class="time-cell">
              <CalendarOutlined class="time-icon" />
              <span>{{ formatTime(record.created_at) }}</span>
            </div>
          </template>
          <template v-else-if="column.key === 'actions'">
            <a-space v-if="isCancelableWalletOrder(record)">
              <a-button v-if="isPendingPayRecharge(record)" type="link" size="small" @click="continuePay(record)">继续支付</a-button>
              <a-popconfirm :title="cancelConfirmText(record)" @confirm="cancelPendingOrder(record)">
                <a-button type="link" size="small" danger>取消</a-button>
              </a-popconfirm>
            </a-space>
            <span v-else>-</span>
          </template>
        </template>
      </a-table>
    </div>

    <!-- Recharge Modal -->
    <a-modal
      v-model:open="rechargeModalVisible"
      title="账户充值"
      :width="440"
      :confirm-loading="rechargeLoading"
      @ok="submitRecharge"
      @cancel="resetRechargeForm"
    >
      <a-alert
        message="充值将在审核通过后到账"
        type="info"
        show-icon
        class="modal-alert"
      />
      <a-form ref="rechargeFormRef" :model="recharge" layout="vertical" class="modal-form">
        <a-form-item label="支付方式" name="method" :rules="[{ required: true, message: '请选择支付方式' }]">
          <a-select
            v-model:value="recharge.method"
            placeholder="请选择支付方式"
            size="large"
            style="width: 100%"
            :options="rechargeMethodOptions"
          />
        </a-form-item>
        <a-form-item label="充值金额" name="amount" :rules="[{ required: true, message: '请输入充值金额' }]">
          <a-input-number
            v-model:value="recharge.amount"
            :min="0.01"
            :precision="2"
            :step="100"
            placeholder="请输入充值金额"
            size="large"
            style="width: 100%"
          >
            <template #prefix>¥</template>
          </a-input-number>
        </a-form-item>
        <a-form-item label="备注" name="note">
          <a-textarea
            v-model:value="recharge.note"
            placeholder="选填，可填写付款信息"
            :rows="3"
            :maxlength="200"
            show-count
          />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- Withdraw Modal -->
    <a-modal
      v-model:open="withdrawModalVisible"
      title="申请提现"
      :width="440"
      :confirm-loading="withdrawLoading"
      @ok="submitWithdraw"
      @cancel="resetWithdrawForm"
    >
      <a-alert
        message="提现将在审核通过后打款"
        type="warning"
        show-icon
        class="modal-alert"
      />
      <div class="balance-info">
        <span>可用余额：</span>
        <span class="balance-amount">{{ formatMoney(wallet.balance) }}</span>
      </div>
      <a-form ref="withdrawFormRef" :model="withdraw" layout="vertical" class="modal-form">
        <a-form-item label="提现金额" name="amount" :rules="[{ required: true, message: '请输入提现金额' }]">
          <a-input-number
            v-model:value="withdraw.amount"
            :min="0.01"
            :max="wallet.balance"
            :precision="2"
            :step="100"
            placeholder="请输入提现金额"
            size="large"
            style="width: 100%"
          >
            <template #prefix>¥</template>
          </a-input-number>
        </a-form-item>
        <a-form-item label="收款方式" name="note" :rules="[{ required: true, message: '请填写收款方式' }]">
          <a-textarea
            v-model:value="withdraw.note"
            placeholder="请填写收款方式（微信/支付宝/银行卡等）"
            :rows="3"
            :maxlength="200"
            show-count
          />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from "vue";
import { message } from "ant-design-vue";
import {
  ReloadOutlined,
  WalletOutlined,
  TransactionOutlined,
  ExportOutlined,
  OrderedListOutlined,
  InfoCircleOutlined,
  CheckCircleFilled,
  CloseCircleFilled,
  ClockCircleOutlined,
  CalendarOutlined,
  SwapOutlined
} from "@ant-design/icons-vue";
import { cancelWalletOrder, createWalletRecharge, createWalletWithdraw, getWallet, listPaymentProviders, listWalletOrders, payWalletOrder } from "@/services/user";
import { normalizeWallet } from "@/utils/wallet";

const loading = ref(false);
const rechargeLoading = ref(false);
const withdrawLoading = ref(false);
const wallet = ref({ balance: 0, currency: "CNY" });
const orders = ref([]);

const rechargeModalVisible = ref(false);
const withdrawModalVisible = ref(false);
const rechargeFormRef = ref();
const withdrawFormRef = ref();
const rechargeMethods = ref([]);

const filterStatus = ref("");
const filterType = ref("");

const pagination = reactive({
  current: 1,
  pageSize: 10,
  total: 0
});

const recharge = reactive({ method: "approval", amount: null, note: "" });
const withdraw = reactive({ amount: null, note: "" });

const rechargeMethodOptions = computed(() =>
  rechargeMethods.value.map((item) => ({
    label: item.name || item.key,
    value: item.key
  }))
);

const columns = [
  { title: '类型', dataIndex: 'type', key: 'type', width: 100 },
  { title: '金额', dataIndex: 'amount', key: 'amount', width: 140 },
  { title: '状态', dataIndex: 'status', key: 'status', width: 120 },
  { title: '备注', dataIndex: 'note', key: 'note', width: 240, ellipsis: true },
  { title: '时间', dataIndex: 'created_at', key: 'created_at', width: 180 },
  { title: '操作', dataIndex: 'actions', key: 'actions', width: 180 }
];

const monthStats = computed(() => {
  const now = new Date();
  const currentMonth = now.getMonth();
  const currentYear = now.getFullYear();

  const monthOrders = orders.value.filter(order => {
    const orderDate = new Date(order.created_at);
    return orderDate.getMonth() === currentMonth && orderDate.getFullYear() === currentYear;
  });

  return {
    recharge: monthOrders.filter(o => o.type === 'recharge' && o.status === 'approved').reduce((sum, o) => sum + Number(o.amount || 0), 0),
    withdraw: monthOrders.filter(o => o.type === 'withdraw' && o.status === 'approved').reduce((sum, o) => sum + Number(o.amount || 0), 0),
    pending: monthOrders.filter(o => o.status === 'pending' || o.status === 'pending_review').length,
    total: monthOrders.length
  };
});

const filteredOrders = computed(() => {
  let result = [...orders.value];
  if (filterStatus.value) {
    result = result.filter(order => order.status === filterStatus.value);
  }
  if (filterType.value) {
    result = result.filter(order => order.type === filterType.value);
  }
  return result;
});

const formatMoney = (amount) => {
  const value = Number(amount ?? 0);
  if (Number.isNaN(value)) return "-";
  return `¥${value.toFixed(2)}`;
};

const typeLabel = (value) => {
  const t = String(value || '').trim().toLowerCase();
  if (t === "recharge") return "充值";
  if (t === "withdraw") return "提现";
  if (t === "refund") return "退款";
  return t || "其他";
};

const formatTime = (value) => {
  if (!value) return "-";
  const dt = new Date(value);
  if (Number.isNaN(dt.getTime())) return value;
  return dt.toLocaleString("zh-CN", { hour12: false });
};

const isPendingPayRecharge = (record) => {
  const type = String(record?.type || "").trim().toLowerCase();
  const status = String(record?.status || "").trim().toLowerCase();
  if (type !== "recharge" || status !== "pending_review") {
    return false;
  }
  const method = String(record?.meta?.payment_method || "").trim();
  return method !== "" && method !== "approval" && method !== "balance";
};

const isCancelableWalletOrder = (record) => {
  const type = String(record?.type || "").trim().toLowerCase();
  if (type !== "recharge" && type !== "refund") {
    return false;
  }
  return String(record?.status || "").trim().toLowerCase() === "pending_review";
};

const cancelConfirmText = (record) =>
  String(record?.type || "").trim().toLowerCase() === "refund"
    ? "确认取消该退款订单？"
    : "确认取消该充值订单？";

const resolvePayMethod = (record) => {
  const method = String(record?.meta?.payment_method || "").trim();
  if (method !== "" && method !== "approval" && method !== "balance") {
    return method;
  }
  return "";
};

const isPendingRecharge = (record) =>
  String(record?.type || "").trim().toLowerCase() === "recharge"
  && String(record?.status || "").trim().toLowerCase() === "pending_review";

const continuePay = async (record) => {
  try {
    if (!isPendingRecharge(record)) {
      message.warning("该订单状态不支持继续支付");
      return;
    }
    if (rechargeMethods.value.length === 0) {
      await fetchRechargeMethods();
    }
    const method = resolvePayMethod(record);
    if (!method) {
      message.warning("该订单未绑定可继续支付的支付方式");
      return;
    }
    const res = await payWalletOrder(record.id, {
      method,
    });
    const payURL = res.data?.payment?.pay_url || res.data?.payment?.payURL || "";
    if (payURL) {
      window.open(payURL, "_blank");
      message.success("已拉起支付");
    } else {
      message.warning("未获取到支付链接");
    }
  } catch (error) {
    message.error(error.response?.data?.error || "拉起支付失败");
  }
};

const cancelPendingOrder = async (record) => {
  try {
    await cancelWalletOrder(record.id, { reason: "user_cancel" });
    message.success("已取消");
    fetchAll();
  } catch (error) {
    message.error(error.response?.data?.error || "取消失败");
  }
};

const fetchAll = async () => {
  loading.value = true;
  try {
    const [walletRes, ordersRes] = await Promise.all([
      getWallet(),
      listWalletOrders({ limit: 100, offset: 0 })
    ]);
    wallet.value = normalizeWallet(walletRes.data) || wallet.value;
    orders.value = (ordersRes.data?.items || []).map((item) => ({
      ...item,
      type: String(item.type || '').trim().toLowerCase()
    }));
    pagination.total = orders.value.length;
    await fetchRechargeMethods();
  } finally {
    loading.value = false;
  }
};

const openRechargeModal = () => {
  void fetchRechargeMethods();
  rechargeModalVisible.value = true;
};

const openWithdrawModal = () => {
  withdrawModalVisible.value = true;
};

const resetRechargeForm = () => {
  recharge.method = "approval";
  recharge.amount = null;
  recharge.note = "";
  rechargeFormRef.value?.clearValidate();
};

const fetchRechargeMethods = async () => {
  try {
    const res = await listPaymentProviders({ scene: "wallet" });
    const methods = (res.data?.items || [])
      .filter((x) => {
        const key = String(x?.key || "").trim();
        if (!key) return false;
        if (key === "balance") return false;
        return x?.enabled !== false;
      })
      .map((x) => ({ key: x.key, name: x.name }));
    if (!methods.find((m) => m.key === "approval")) {
      methods.unshift({ key: "approval", name: "人工审核" });
    }
    rechargeMethods.value = methods;
  } catch {
    rechargeMethods.value = [{ key: "approval", name: "人工审核" }];
  }
  if (!rechargeMethods.value.find((m) => m.key === recharge.method)) {
    recharge.method = rechargeMethods.value[0]?.key || "";
  }
};

const resetWithdrawForm = () => {
  withdraw.amount = null;
  withdraw.note = "";
  withdrawFormRef.value?.clearValidate();
};

const submitRecharge = async () => {
  try {
    await rechargeFormRef.value.validate();
  } catch { return; }

  if (!recharge.amount || recharge.amount <= 0) {
    message.warning("请填写有效的充值金额");
    return;
  }
  if (!recharge.method) {
    message.warning("请选择支付方式");
    return;
  }

  rechargeLoading.value = true;
  try {
    const res = await createWalletRecharge({
      amount: recharge.amount,
      note: recharge.note,
      meta: {},
      method: recharge.method,
    });
    message.success("充值订单已创建");
    const payURL = res.data?.payment?.pay_url || res.data?.payment?.payURL || "";
    if (payURL) {
      window.open(payURL, "_blank");
    }
    rechargeModalVisible.value = false;
    resetRechargeForm();
    fetchAll();
  } catch (error) {
    message.error(error.response?.data?.error || "充值失败");
  } finally {
    rechargeLoading.value = false;
  }
};

const submitWithdraw = async () => {
  try {
    await withdrawFormRef.value.validate();
  } catch { return; }

  if (!withdraw.amount || withdraw.amount <= 0) {
    message.warning("请填写有效的提现金额");
    return;
  }

  if (withdraw.amount > wallet.value.balance) {
    message.warning("提现金额不能超过可用余额");
    return;
  }

  withdrawLoading.value = true;
  try {
    await createWalletWithdraw({ amount: withdraw.amount, note: withdraw.note, meta: { channel: "manual" } });
    message.success("提现订单已提交");
    withdrawModalVisible.value = false;
    resetWithdrawForm();
    fetchAll();
  } catch (error) {
    message.error(error.response?.data?.error || "提现失败");
  } finally {
    withdrawLoading.value = false;
  }
};

const handleFilterChange = () => {
  pagination.current = 1;
};

const handleTableChange = (pag) => {
  pagination.current = pag.current;
  pagination.pageSize = pag.pageSize;
};

onMounted(fetchAll);
</script>

<style scoped>
.billing-page {
  padding: 24px;
  max-width: 1400px;
  margin: 0 auto;
}

/* Balance Card */
.balance-card {
  background: var(--primary-gradient);
  border-radius: var(--radius-xl);
  padding: 32px;
  margin-bottom: 24px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  box-shadow: var(--shadow-lg), var(--shadow-glow-sm);
  color: #fff;
  flex-wrap: wrap;
  gap: 24px;
}

.balance-info {
  display: flex;
  align-items: center;
  gap: 20px;
  padding: 20px 28px;
  background: rgba(0, 0, 0, 0.15) !important;
  backdrop-filter: blur(10px);
  border-radius: var(--radius-xl);
  border: 1px solid rgba(0, 0, 0, 0.1);
}

.balance-icon {
  width: 72px;
  height: 72px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.2);
  backdrop-filter: blur(10px);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 32px;
  color: #fff;
}

.balance-details {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.balance-label {
  font-size: 15px;
  color: rgba(255, 255, 255, 0.9);
  font-weight: 500;
}

.balance-amount {
  font-size: 48px;
  font-weight: 800;
  line-height: 1;
  letter-spacing: -0.02em;
  color: #fff;
  text-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.balance-actions {
  display: flex;
  gap: 12px;
}

.balance-actions :deep(.ant-btn) {
  height: 48px;
  padding: 0 24px;
  font-weight: 600;
  border-radius: var(--radius-md);
}

.balance-actions :deep(.ant-btn-primary) {
  background: #fff;
  color: var(--primary);
  border: none;
}

.balance-actions :deep(.ant-btn-primary:hover) {
  background: rgba(255, 255, 255, 0.9);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.balance-actions :deep(.ant-btn:not(.ant-btn-primary)) {
  background: rgba(255, 255, 255, 0.15);
  border: 1px solid rgba(255, 255, 255, 0.3);
  color: #fff;
}

.balance-actions :deep(.ant-btn:not(.ant-btn-primary):hover) {
  background: rgba(255, 255, 255, 0.25);
  border-color: rgba(255, 255, 255, 0.5);
}

/* Stats Grid */
.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-bottom: 24px;
}

.stat-card {
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: 20px;
  display: flex;
  align-items: center;
  gap: 16px;
  transition: all var(--transition-base);
}

.stat-card:hover {
  box-shadow: var(--shadow-md);
  transform: translateY(-2px);
}

.stat-icon {
  width: 48px;
  height: 48px;
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  color: #fff;
  flex-shrink: 0;
}

.recharge-stat .stat-icon {
  background: var(--success);
}

.withdraw-stat .stat-icon {
  background: var(--warning);
}

.pending-stat .stat-icon {
  background: var(--info);
}

.total-stat .stat-icon {
  background: var(--primary);
}

.stat-info {
  flex: 1;
}

.stat-label {
  font-size: 13px;
  color: var(--text-secondary);
  margin-bottom: 4px;
}

.stat-value {
  font-size: 20px;
  font-weight: 700;
  color: var(--text-primary);
}

/* Table Section */
.table-section {
  background: var(--card);
  border-radius: var(--radius-lg);
  border: 1px solid var(--border);
  overflow: hidden;
}

.table-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border);
}

.table-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
}

.billing-table :deep(.ant-table) {
  background: transparent;
}

.billing-table :deep(.ant-table-thead > tr > th) {
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border);
  padding: 14px 16px;
  font-weight: 600;
  font-size: 13px;
  color: var(--text-secondary);
}

.billing-table :deep(.ant-table-tbody > tr > td) {
  padding: 16px;
  border-bottom: 1px solid var(--border-light);
}

.billing-table :deep(.ant-table-tbody > tr:hover > td) {
  background: var(--bg-secondary);
}

.billing-table :deep(.ant-table-tbody > tr:last-child > td) {
  border-bottom: none;
}

/* Table Cells */
.type-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.type-icon {
  font-size: 16px;
}

.recharge-icon {
  color: var(--success);
}

.withdraw-icon {
  color: var(--warning);
}

.refund-icon {
  color: var(--info);
}

.amount-cell {
  font-family: 'JetBrains Mono', monospace;
  font-weight: 600;
  font-size: 14px;
}

.amount-plus {
  color: var(--success);
}

.amount-minus {
  color: var(--text-primary);
}

.note-cell {
  color: var(--text-secondary);
  font-size: 13px;
}

.time-cell {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--text-secondary);
  font-size: 13px;
}

.time-icon {
  font-size: 14px;
  color: var(--text-tertiary);
}

/* Modal */
.modal-alert {
  margin-bottom: 20px;
}

.balance-info {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: var(--bg-secondary);
  border-radius: var(--radius-md);
  margin-bottom: 20px;
  font-size: 14px;
}

.balance-amount {
  font-size: 18px;
  font-weight: 700;
  color: var(--primary);
}

.modal-form {
  margin-top: 16px;
}

/* Responsive */
@media (max-width: 1024px) {
  .stats-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 768px) {
  .billing-page {
    padding: 16px;
  }

  .balance-card {
    padding: 24px;
    flex-direction: column;
    align-items: stretch;
  }

  .balance-icon {
    width: 56px;
    height: 56px;
    font-size: 24px;
  }

  .balance-amount {
    font-size: 36px;
  }

  .balance-actions {
    flex-wrap: wrap;
  }

  .balance-actions :deep(.ant-btn) {
    flex: 1;
    min-width: calc(50% - 6px);
    justify-content: center;
  }

  .stats-grid {
    grid-template-columns: 1fr;
  }

  .table-header {
    flex-direction: column;
    align-items: stretch;
    gap: 12px;
  }
}
</style>
