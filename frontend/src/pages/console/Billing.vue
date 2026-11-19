<template>
  <div class="finance-page">
    <!-- 余额概览 -->
    <section class="wallet-hero">
      <div class="wallet-hero-glow" aria-hidden="true"></div>
      <div class="wallet-hero-main">
        <span class="wallet-hero-label">{{ t('console.billing.currentBalance') }}</span>
        <div class="wallet-hero-amount">{{ formatMoney(wallet.balance) }}</div>
        <div class="wallet-hero-meta">
          <span class="wallet-hero-chip">
            <em>{{ monthStats.total }}</em>{{ t('console.billing.totalTransactions') }}
          </span>
          <span class="wallet-hero-chip" :class="{ 'is-alert': monthStats.pending > 0 }">
            <em>{{ monthStats.pending }}</em>{{ t('console.billing.pendingReview') }}
          </span>
        </div>
      </div>
      <div class="wallet-hero-actions">
        <a-button type="primary" size="large" class="wallet-action-primary" @click="openRechargeModal">
          <TransactionOutlined />
          <span>{{ t('console.billing.recharge') }}</span>
        </a-button>
        <a-button size="large" class="wallet-action-ghost" @click="openWithdrawModal">
          <ExportOutlined />
          <span>{{ t('console.billing.withdraw') }}</span>
        </a-button>
        <a-tooltip :title="t('common.refresh')">
          <a-button size="large" class="wallet-action-icon" :loading="loading" @click="fetchAll">
            <ReloadOutlined />
          </a-button>
        </a-tooltip>
      </div>
    </section>

    <!-- 本月指标 -->
    <section class="wallet-stats">
      <article v-for="stat in statCards" :key="stat.key" class="wallet-stat">
        <span class="wallet-stat-icon" :class="`is-${stat.tone}`">
          <component :is="stat.icon" />
        </span>
        <div class="wallet-stat-body">
          <span class="wallet-stat-label">{{ stat.label }}</span>
          <strong class="wallet-stat-value">{{ stat.value }}</strong>
        </div>
      </article>
    </section>

    <!-- 交易记录 -->
    <section class="records-card">
      <header class="records-head">
        <div class="records-title">
          <h3>{{ t('console.billing.records') }}</h3>
          <span class="records-count">{{ pagination.total }}</span>
        </div>
        <div class="records-filters">
          <a-segmented v-model:value="typeTab" :options="typeFilterOptions" class="records-segmented" />
          <a-select
            v-model:value="filterStatus"
            class="records-select"
            :placeholder="t('common.status')"
            allow-clear
            @change="handleFilterChange"
          >
            <a-select-option value="">{{ t('common.all') }}</a-select-option>
            <a-select-option value="pending_review">{{ t('common.pending') }}</a-select-option>
            <a-select-option value="approved">{{ t('status.approved') }}</a-select-option>
            <a-select-option value="rejected">{{ t('status.rejected') }}</a-select-option>
          </a-select>
        </div>
      </header>

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
          showTotal: (total) => t('common.total', { n: total })
        }"
        @change="handleTableChange"
        class="records-table"
      >
        <template #empty>
          <div class="records-empty">
            <span class="records-empty-icon"><TransactionOutlined /></span>
            <p class="records-empty-title">{{ t('common.noData') || '暂无交易记录' }}</p>
            <p class="records-empty-hint">{{ t('console.billing.emptyHint') || '充值或提现后，记录会出现在这里' }}</p>
          </div>
        </template>
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
              {{ isPendingPayRecharge(record) ? t('status.pendingPayment') : t('status.pendingReview') }}
            </a-tag>
            <a-tag v-else-if="record.status === 'approved'" color="success">
              <CheckCircleFilled />
              {{ t('status.approved') }}
            </a-tag>
            <a-tag v-else-if="record.status === 'rejected'" color="error">
              <CloseCircleFilled />
              {{ t('status.rejected') }}
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
              <a-button v-if="isPendingPayRecharge(record)" type="link" size="small" @click="continuePay(record)">{{ t('console.billing.continuePay') }}</a-button>
              <a-popconfirm :title="cancelConfirmText(record)" @confirm="cancelPendingOrder(record)">
                <a-button type="link" size="small" danger>{{ t('common.cancel') }}</a-button>
              </a-popconfirm>
            </a-space>
            <span v-else>-</span>
          </template>
        </template>
      </a-table>
    </section>

    <!-- Recharge Modal -->
    <a-modal
      v-model:open="rechargeModalVisible"
      :title="t('console.billing.rechargeTitle')"
      :width="440"
      :confirm-loading="rechargeLoading"
      @ok="submitRecharge"
      @cancel="resetRechargeForm"
    >
      <a-alert
        :message="t('console.billing.rechargeNotice')"
        type="info"
        show-icon
        class="modal-alert"
      />
      <a-form ref="rechargeFormRef" :model="recharge" layout="vertical" class="modal-form">
        <a-form-item :label="t('console.billing.paymentMethod')" name="method" :rules="[{ required: true, message: t('console.billing.selectPayment') }]">
          <a-select
            v-model:value="recharge.method"
            :placeholder="t('console.billing.selectPayment')"
            size="large"
            style="width: 100%"
            :options="rechargeMethodOptions"
          />
        </a-form-item>
        <a-form-item :label="t('console.billing.rechargeAmount')" name="amount" :rules="[{ required: true, message: t('console.billing.enterRechargeAmount') }]">
          <a-input-number
            v-model:value="recharge.amount"
            :min="0.01"
            :precision="2"
            :step="100"
            :placeholder="t('console.billing.enterRechargeAmount')"
            size="large"
            style="width: 100%"
          >
            <template #prefix>¥</template>
          </a-input-number>
        </a-form-item>
        <a-form-item :label="t('common.remark')" name="note">
          <a-textarea
            v-model:value="recharge.note"
            :placeholder="t('console.billing.notePlaceholder')"
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
      :title="t('console.billing.withdrawTitle')"
      :width="440"
      :confirm-loading="withdrawLoading"
      @ok="submitWithdraw"
      @cancel="resetWithdrawForm"
    >
      <a-alert
        :message="t('console.billing.withdrawNotice')"
        type="warning"
        show-icon
        class="modal-alert"
      />
      <div class="withdraw-summary">
        <span>{{ t('console.billing.availableBalance') }}</span>
        <strong>{{ formatMoney(wallet.balance) }}</strong>
      </div>
      <a-form ref="withdrawFormRef" :model="withdraw" layout="vertical" class="modal-form">
        <a-form-item :label="t('console.billing.withdrawAmount')" name="amount" :rules="[{ required: true, message: t('console.billing.enterWithdrawAmount') }]">
          <a-input-number
            v-model:value="withdraw.amount"
            :min="0.01"
            :max="wallet.balance"
            :precision="2"
            :step="100"
            :placeholder="t('console.billing.enterWithdrawAmount')"
            size="large"
            style="width: 100%"
          >
            <template #prefix>¥</template>
          </a-input-number>
        </a-form-item>
        <a-form-item :label="t('console.billing.payoutMethod')" name="note" :rules="[{ required: true, message: t('console.billing.payoutRequired') }]">
          <a-textarea
            v-model:value="withdraw.note"
            :placeholder="t('console.billing.payoutPlaceholder')"
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
import { t, locale } from "@/locales";

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

const columns = computed(() => [
  { title: t('common.type'), dataIndex: 'type', key: 'type', width: 100 },
  { title: t('common.amount'), dataIndex: 'amount', key: 'amount', width: 140 },
  { title: t('common.status'), dataIndex: 'status', key: 'status', width: 120 },
  { title: t('common.remark'), dataIndex: 'note', key: 'note', width: 240, ellipsis: true },
  { title: t('common.time'), dataIndex: 'created_at', key: 'created_at', width: 180 },
  { title: t('common.actions'), dataIndex: 'actions', key: 'actions', width: 180 }
]);

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

/** 交易类型分段筛选：'all' 对应空筛选值 */
const typeTab = computed({
  get: () => filterType.value || 'all',
  set: (value) => {
    filterType.value = value === 'all' ? '' : value;
    pagination.current = 1;
  }
});

const typeFilterOptions = computed(() => [
  { label: t('common.all'), value: 'all' },
  { label: t('console.billing.rechargeType'), value: 'recharge' },
  { label: t('console.billing.withdrawType'), value: 'withdraw' }
]);

const statCards = computed(() => [
  { key: 'recharge', tone: 'in', icon: TransactionOutlined, label: t('console.billing.monthRecharge'), value: formatMoney(monthStats.value.recharge) },
  { key: 'withdraw', tone: 'out', icon: ExportOutlined, label: t('console.billing.monthWithdraw'), value: formatMoney(monthStats.value.withdraw) },
  { key: 'pending', tone: 'pending', icon: ClockCircleOutlined, label: t('console.billing.pendingReview'), value: monthStats.value.pending },
  { key: 'total', tone: 'total', icon: OrderedListOutlined, label: t('console.billing.totalTransactions'), value: monthStats.value.total }
]);

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
  const key = String(value || '').trim().toLowerCase();
  if (key === "recharge") return t('console.billing.rechargeType');
  if (key === "withdraw") return t('console.billing.withdrawType');
  if (key === "refund") return t('console.billing.refundType');
  return key || t('console.billing.otherType');
};

const formatTime = (value) => {
  if (!value) return "-";
  const dt = new Date(value);
  if (Number.isNaN(dt.getTime())) return value;
  return dt.toLocaleString(locale.value, { hour12: false });
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
    ? t('console.billing.cancelRefundConfirm')
    : t('console.billing.cancelRechargeConfirm');

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
      message.warning(t('console.billing.cannotContinuePay'));
      return;
    }
    if (rechargeMethods.value.length === 0) {
      await fetchRechargeMethods();
    }
    const method = resolvePayMethod(record);
    if (!method) {
      message.warning(t('console.billing.noPaymentMethod'));
      return;
    }
    const res = await payWalletOrder(record.id, {
      method,
    });
    const payURL = res.data?.payment?.pay_url || res.data?.payment?.payURL || "";
    if (payURL) {
      window.open(payURL, "_blank");
      message.success(t('console.billing.paymentStarted'));
    } else {
      message.warning(t('console.billing.noPaymentUrl'));
    }
  } catch (error) {
    message.error(error.response?.data?.error || t('console.billing.paymentFailed'));
  }
};

const cancelPendingOrder = async (record) => {
  try {
    await cancelWalletOrder(record.id, { reason: "user_cancel" });
    message.success(t('status.canceled'));
    fetchAll();
  } catch (error) {
    message.error(error.response?.data?.error || t('console.billing.cancelFailed'));
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
      methods.unshift({ key: "approval", name: t('console.billing.manualReview') });
    }
    rechargeMethods.value = methods;
  } catch {
    rechargeMethods.value = [{ key: "approval", name: t('console.billing.manualReview') }];
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
    message.warning(t('console.billing.invalidAmount'));
    return;
  }
  if (!recharge.method) {
    message.warning(t('console.billing.selectPayment'));
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
    message.success(t('console.billing.rechargeCreated'));
    const payURL = res.data?.payment?.pay_url || res.data?.payment?.payURL || "";
    if (payURL) {
      window.open(payURL, "_blank");
    }
    rechargeModalVisible.value = false;
    resetRechargeForm();
    fetchAll();
  } catch (error) {
    message.error(error.response?.data?.error || t('console.billing.rechargeFailed'));
  } finally {
    rechargeLoading.value = false;
  }
};

const submitWithdraw = async () => {
  try {
    await withdrawFormRef.value.validate();
  } catch { return; }

  if (!withdraw.amount || withdraw.amount <= 0) {
    message.warning(t('console.billing.invalidWithdrawAmount'));
    return;
  }

  if (withdraw.amount > wallet.value.balance) {
    message.warning(t('console.billing.exceedsBalance'));
    return;
  }

  withdrawLoading.value = true;
  try {
    await createWalletWithdraw({ amount: withdraw.amount, note: withdraw.note, meta: { channel: "manual" } });
    message.success(t('console.billing.withdrawSubmitted'));
    withdrawModalVisible.value = false;
    resetWithdrawForm();
    fetchAll();
  } catch (error) {
    message.error(error.response?.data?.error || t('console.billing.withdrawFailed'));
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
.finance-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}

/* ==================== 余额概览 ==================== */
.wallet-hero {
  position: relative;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 20px 24px;
  padding: 24px 26px;
  overflow: hidden;
  border-radius: var(--theme-card-radius, 10px);
  background:
    radial-gradient(120% 180% at 0% 0%, color-mix(in srgb, var(--primary) 42%, transparent), transparent 58%),
    linear-gradient(135deg, color-mix(in srgb, var(--primary) 90%, #061320), var(--primary));
  color: var(--theme-primary-foreground, #fff);
  box-shadow: 0 14px 32px color-mix(in srgb, var(--primary) 20%, transparent);
}

.wallet-hero-glow {
  position: absolute;
  inset-block-start: -60px;
  inset-inline-end: -40px;
  width: 260px;
  height: 260px;
  border-radius: 50%;
  background: radial-gradient(circle, color-mix(in srgb, var(--theme-primary-foreground, #fff) 18%, transparent), transparent 68%);
  pointer-events: none;
}

.wallet-hero-main {
  position: relative;
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 6px;
}

.wallet-hero-label {
  font-size: 12.5px;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  opacity: 0.78;
}

.wallet-hero-amount {
  font-family: var(--font-mono, ui-monospace, monospace);
  font-size: clamp(30px, 3vw, 40px);
  font-weight: 700;
  line-height: 1.1;
  letter-spacing: -0.02em;
  font-variant-numeric: tabular-nums;
}

.wallet-hero-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 6px;
}

.wallet-hero-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 3px 10px;
  border: 1px solid color-mix(in srgb, var(--theme-primary-foreground, #fff) 22%, transparent);
  border-radius: 999px;
  background: color-mix(in srgb, var(--theme-primary-foreground, #fff) 10%, transparent);
  font-size: 11.5px;
  opacity: 0.92;
}

.wallet-hero-chip em {
  font-style: normal;
  font-family: var(--font-mono, ui-monospace, monospace);
  font-weight: 700;
}

.wallet-hero-chip.is-alert {
  border-color: color-mix(in srgb, var(--warning, #e6a23c) 55%, transparent);
  background: color-mix(in srgb, var(--warning, #e6a23c) 26%, transparent);
  opacity: 1;
}

.wallet-hero-actions {
  position: relative;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}

.wallet-hero-actions :deep(.ant-btn) {
  height: 42px;
  padding: 0 18px;
  border-radius: var(--theme-control-radius, 8px);
  font-weight: 600;
}

.wallet-hero-actions :deep(.wallet-action-primary) {
  border: none !important;
  background: var(--theme-primary-foreground, #fff) !important;
  color: var(--primary) !important;
}

.wallet-hero-actions :deep(.wallet-action-primary:hover) {
  background: color-mix(in srgb, var(--theme-primary-foreground, #fff) 88%, transparent) !important;
  color: var(--primary) !important;
}

.wallet-hero-actions :deep(.wallet-action-ghost) {
  border: 1px solid color-mix(in srgb, var(--theme-primary-foreground, #fff) 34%, transparent) !important;
  background: color-mix(in srgb, var(--theme-primary-foreground, #fff) 12%, transparent) !important;
  color: inherit !important;
}

.wallet-hero-actions :deep(.wallet-action-ghost:hover) {
  border-color: color-mix(in srgb, var(--theme-primary-foreground, #fff) 55%, transparent) !important;
  background: color-mix(in srgb, var(--theme-primary-foreground, #fff) 20%, transparent) !important;
  color: inherit !important;
}

.wallet-hero-actions :deep(.wallet-action-icon) {
  width: 42px;
  padding: 0 !important;
  border: 1px solid color-mix(in srgb, var(--theme-primary-foreground, #fff) 28%, transparent) !important;
  background: transparent !important;
  color: inherit !important;
}

.wallet-hero-actions :deep(.wallet-action-icon:hover) {
  border-color: color-mix(in srgb, var(--theme-primary-foreground, #fff) 48%, transparent) !important;
  background: color-mix(in srgb, var(--theme-primary-foreground, #fff) 12%, transparent) !important;
  color: inherit !important;
}

/* ==================== 本月指标 ==================== */
.wallet-stats {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: var(--theme-card-radius, 10px);
  background: var(--card);
}

.wallet-stat {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
  padding: 16px 18px;
  border-inline-start: 1px solid var(--border);
}

.wallet-stat:first-child {
  border-inline-start: 0;
}

.wallet-stat-icon {
  display: grid;
  width: 34px;
  height: 34px;
  flex: 0 0 34px;
  place-items: center;
  border-radius: 9px;
  font-size: 16px;
}

.wallet-stat-icon.is-in {
  color: var(--success);
  background: color-mix(in srgb, var(--success) 12%, transparent);
}

.wallet-stat-icon.is-out {
  color: var(--warning);
  background: color-mix(in srgb, var(--warning) 14%, transparent);
}

.wallet-stat-icon.is-pending {
  color: var(--info);
  background: color-mix(in srgb, var(--info) 12%, transparent);
}

.wallet-stat-icon.is-total {
  color: var(--primary);
  background: color-mix(in srgb, var(--primary) 12%, transparent);
}

.wallet-stat-body {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 2px;
}

.wallet-stat-label {
  overflow: hidden;
  color: var(--text-secondary);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.wallet-stat-value {
  color: var(--text-primary);
  font-family: var(--font-mono, ui-monospace, monospace);
  font-size: 18px;
  font-weight: 650;
  line-height: 1.2;
  font-variant-numeric: tabular-nums;
}

/* ==================== 交易记录 ==================== */
.records-card {
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: var(--theme-card-radius, 10px);
  background: var(--card);
}

.records-head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 10px 16px;
  padding: 14px 18px;
  border-bottom: 1px solid var(--border);
}

.records-title {
  display: flex;
  align-items: center;
  gap: 8px;
}

.records-title h3 {
  margin: 0;
  color: var(--text-primary);
  font-size: 15px;
  font-weight: 650;
}

.records-count {
  min-width: 20px;
  height: 20px;
  padding: 0 7px;
  border-radius: 999px;
  background: var(--bg-secondary);
  color: var(--text-secondary);
  font-family: var(--font-mono, ui-monospace, monospace);
  font-size: 11px;
  line-height: 20px;
  text-align: center;
}

.records-filters {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.records-segmented :deep(.ant-segmented-item-label) {
  min-height: 26px;
  padding: 0 12px;
  font-size: 12px;
  line-height: 26px;
}

.records-select {
  width: 122px;
}

.records-table :deep(.ant-table) {
  background: transparent;
}

.records-table :deep(.ant-table-thead > tr > th) {
  padding: 11px 16px;
  border-bottom: 1px solid var(--border);
  background: var(--bg-secondary);
  color: var(--text-secondary);
  font-size: 12px;
  font-weight: 600;
}

.records-table :deep(.ant-table-tbody > tr > td) {
  padding: 14px 16px;
  border-bottom: 1px solid var(--border-light, var(--border));
}

.records-table :deep(.ant-table-tbody > tr:last-child > td) {
  border-bottom: none;
}

.records-table :deep(.ant-table-tbody > tr:hover > td) {
  background: var(--bg-secondary);
}

.records-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 28px 0 24px;
}

.records-empty-icon {
  display: grid;
  width: 44px;
  height: 44px;
  margin-bottom: 6px;
  place-items: center;
  border-radius: 12px;
  background: var(--bg-secondary);
  color: var(--text-tertiary);
  font-size: 20px;
}

.records-empty-title {
  margin: 0;
  color: var(--text-secondary);
  font-size: 13px;
}

.records-empty-hint {
  margin: 0;
  color: var(--text-tertiary);
  font-size: 11.5px;
}

/* ==================== 表格单元格 ==================== */
.type-cell {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 500;
}

.type-icon {
  font-size: 15px;
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
  font-family: var(--font-mono, ui-monospace, monospace);
  font-size: 13.5px;
  font-weight: 650;
  font-variant-numeric: tabular-nums;
}

.amount-plus {
  color: var(--success);
}

.amount-minus {
  color: var(--text-primary);
}

.note-cell {
  color: var(--text-secondary);
  font-size: 12.5px;
}

.time-cell {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--text-secondary);
  font-size: 12.5px;
  font-variant-numeric: tabular-nums;
}

.time-icon {
  color: var(--text-tertiary);
  font-size: 13px;
}

/* ==================== 弹窗 ==================== */
.modal-alert {
  margin-bottom: 20px;
}

.withdraw-summary {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 20px;
  padding: 12px 16px;
  border: 1px solid var(--border);
  border-radius: var(--theme-control-radius, 8px);
  background: var(--bg-secondary);
  font-size: 13px;
}

.withdraw-summary strong {
  color: var(--primary);
  font-family: var(--font-mono, ui-monospace, monospace);
  font-size: 17px;
  font-variant-numeric: tabular-nums;
}

.modal-form {
  margin-top: 16px;
}

/* ==================== 响应式 ==================== */
@media (max-width: 1080px) {
  .wallet-stats {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .wallet-stat:nth-child(3) {
    border-inline-start: 0;
  }

  .wallet-stat:nth-child(n + 3) {
    border-top: 1px solid var(--border);
  }
}

@media (max-width: 768px) {
  .wallet-hero {
    padding: 18px;
  }

  .wallet-hero-actions {
    width: 100%;
  }

  .wallet-hero-actions :deep(.ant-btn) {
    flex: 1 1 auto;
    justify-content: center;
  }

  .wallet-hero-actions :deep(.wallet-action-icon) {
    flex: 0 0 42px;
  }

  .records-head {
    align-items: stretch;
  }

  .records-filters {
    width: 100%;
  }

  .records-segmented {
    flex: 1 1 auto;
  }

  .records-select {
    width: 118px;
  }
}

@media (max-width: 480px) {
  .wallet-stats {
    grid-template-columns: minmax(0, 1fr);
  }

  .wallet-stat {
    border-inline-start: 0;
  }

  .wallet-stat + .wallet-stat {
    border-top: 1px solid var(--border);
  }
}

</style>
