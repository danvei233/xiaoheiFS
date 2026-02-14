<template>
  <div class="order-detail-page">
    <!-- Page Header -->
    <div class="page-header">
      <div class="header-left">
        <a-button @click="$router.push('/console/orders')" class="back-btn">
          <ArrowLeftOutlined />
          返回列表
        </a-button>
        <div class="header-title-section">
          <div class="title-row">
            <ShoppingCartOutlined class="title-icon" />
            <h1 class="page-title">订单详情</h1>
          </div>
        </div>
      </div>
      <div class="header-actions">
        <OrderStatusBadge :status="order?.status || ''" />
      </div>
    </div>

    <!-- Order Stats Banner -->
    <div class="stats-banner">
      <div class="stat-item">
        <FileTextOutlined class="stat-icon" />
        <div class="stat-content">
          <span class="stat-label">订单号</span>
          <span class="stat-value">{{ order?.order_no || '-' }}</span>
        </div>
      </div>
      <div class="stat-divider"></div>
      <div class="stat-item">
        <PayCircleOutlined class="stat-icon amount" />
        <div class="stat-content">
          <span class="stat-label">订单金额</span>
          <span class="stat-value stat-amount">¥{{ order?.total_amount || '-' }}</span>
        </div>
      </div>
      <div class="stat-divider"></div>
      <div class="stat-item">
        <ClockCircleOutlined class="stat-icon" />
        <div class="stat-content">
          <span class="stat-label">创建时间</span>
          <span class="stat-value">{{ order?.created_at || '-' }}</span>
        </div>
      </div>
    </div>

    <!-- Progress Steps -->
    <div class="progress-section">
      <a-steps :current="stepIndex" size="small" class="order-steps">
        <a-step title="草稿" />
        <a-step title="待支付" />
        <a-step title="待审核" />
        <a-step title="已通过" />
        <a-step title="开通中" />
        <a-step title="已完成" />
      </a-steps>
    </div>

    <!-- Main Content Grid -->
    <div class="content-grid">
      <!-- Left Column -->
      <div class="left-column">
        <!-- Order Items Card -->
        <div class="section-card">
          <div class="card-header">
            <UnorderedListOutlined class="card-icon" />
            <h3 class="card-title">订单明细</h3>
            <a-tag class="items-count">{{ orderItems.length }} 件商品</a-tag>
          </div>
          <div class="card-body">
            <a-table
              :columns="itemColumns"
              :data-source="orderItems"
              :pagination="false"
              :scroll="{ x: 900 }"
              row-key="id"
              size="middle"
              class="items-table"
            >
              <template #bodyCell="{ column, record }">
                <template v-if="column.key === 'specText'">
                  <div class="spec-cell">
                    <ApiOutlined class="spec-icon" />
                    <div class="spec-content">
                      <div class="spec-main">{{ record.specText }}</div>
                      <div v-if="record.specDetail?.length" class="spec-detail">
                        <div v-for="(line, idx) in record.specDetail" :key="idx" class="spec-line">{{ line }}</div>
                      </div>
                    </div>
                  </div>
                </template>
                <template v-else-if="column.key === 'amount'">
                  <div class="amount-cell">
                    <span class="amount-symbol">¥</span>
                    <span class="amount-value">{{ record.amount }}</span>
                  </div>
                </template>
                <template v-else-if="column.key === 'status'">
                  <a-tag :color="getItemStatusColor(record.status)">{{ record.status }}</a-tag>
                </template>
              </template>
            </a-table>
          </div>
        </div>

        <!-- Payment Card -->
        <div class="section-card">
          <div class="card-header">
            <HistoryOutlined class="card-icon" />
            <h3 class="card-title">历史付款</h3>
          </div>
          <div class="card-body">
            <a-table
              :columns="paymentColumns"
              :data-source="orderPayments"
              :pagination="false"
              :scroll="{ x: 760 }"
              row-key="id"
              size="small"
              class="history-table"
            >
              <template #bodyCell="{ column, record }">
                <template v-if="column.key === 'method'">
                  <a-tag color="blue">{{ record.method }}</a-tag>
                </template>
                <template v-else-if="column.key === 'amount'">
                  <a-tag color="green">{{ record.amount }}</a-tag>
                </template>
                <template v-else-if="column.key === 'status'">
                  <a-tag :color="getPaymentStatusColor(record.status)">{{ record.status }}</a-tag>
                </template>
              </template>
            </a-table>
            <a-empty v-if="!orderPayments.length" description="暂无付款记录" />
          </div>
        </div>
      </div>

      <!-- Right Column -->
      <div class="right-column">
        <!-- Progress Card -->
        <div class="sidebar-card">
          <div class="card-header">
            <SyncOutlined class="card-icon" :class="{ spin: events.length > 0 }" />
            <h3 class="card-title">开通进度</h3>
            <a-tag :color="events.length > 0 ? 'processing' : 'default'">
              {{ events.length > 0 ? '实时更新' : '等待中' }}
            </a-tag>
          </div>
          <div class="card-body">
            <a-timeline v-if="events.length" class="progress-timeline">
              <a-timeline-item v-for="(ev, idx) in events" :key="idx" :color="idx === 0 ? 'green' : 'blue'">
                {{ ev }}
              </a-timeline-item>
            </a-timeline>
            <a-empty v-else description="等待事件推送" />
          </div>
        </div>

        <!-- VPS Info Card -->
        <div v-if="vpsInfo" class="sidebar-card">
          <div class="card-header">
            <CloudServerOutlined class="card-icon" />
            <h3 class="card-title">VPS 信息</h3>
            <a-button type="link" size="small" @click="copyVpsInfo" class="copy-btn">
              <CopyOutlined />
            </a-button>
          </div>
          <div class="card-body">
            <a-descriptions :column="1" bordered size="small" class="vps-descriptions">
              <a-descriptions-item v-for="(val, key) in vpsInfo" :key="key" :label="key">
                <a-typography-text copyable>{{ val }}</a-typography-text>
              </a-descriptions-item>
            </a-descriptions>
          </div>
        </div>

        <!-- Action Card -->
        <div class="action-card">
          <a-button
            v-if="order?.status === 'pending_payment'"
            type="primary"
            size="large"
            block
            @click="showPaymentModal"
            class="action-btn action-btn-primary"
          >
            <PayCircleOutlined />
            立即支付
          </a-button>
          <a-button @click="refresh" size="large" block class="action-btn">
            <ReloadOutlined />
            刷新订单
          </a-button>
          <a-button v-if="canCancel" danger size="large" block @click="cancelCurrent" class="action-btn">
            <CloseCircleOutlined />
            撤销订单
          </a-button>
        </div>
      </div>
    </div>

    <!-- Payment Modal -->
    <a-modal
      v-model:open="paymentModalVisible"
      title="发起支付"
      :width="540"
      @cancel="closePaymentModal"
      :footer="null"
    >
      <a-form layout="vertical" :model="formModel" ref="paymentFormRef" @finish="submitPayment">
        <a-form-item label="付款方式" name="method" :rules="[{ required: true, message: '请选择付款方式' }]">
          <a-select v-model:value="formModel.method" placeholder="请选择付款方式" size="large">
            <a-select-option v-for="item in providers" :key="item.key" :value="item.key">
              <template #default>
                <span style="display: inline-flex; align-items: center; gap: 8px;">
                  <CreditCardOutlined />
                  {{ item.name || item.key }}
                </span>
              </template>
            </a-select-option>
          </a-select>
        </a-form-item>

        <a-alert
          v-if="selectedProvider?.key === 'balance'"
          type="info"
          show-icon
          class="payment-alert"
          :message="`钱包余额：¥${selectedProvider?.balance ?? '-'}`"
        >
          <template #icon><WalletOutlined /></template>
        </a-alert>

        <a-alert
          v-if="paymentHint"
          type="success"
          show-icon
          class="payment-alert"
          :message="paymentHint"
        >
          <template #icon><InfoCircleOutlined /></template>
        </a-alert>

        <a-alert
          v-if="selectedInstructions"
          type="warning"
          show-icon
          class="payment-alert"
          :message="selectedInstructions"
        >
          <template #icon><BellOutlined /></template>
        </a-alert>

        <!-- Approval Payment Form -->
        <template v-if="selectedProvider?.key === 'approval'">
          <a-divider class="form-divider">
            <DollarOutlined class="divider-icon" />
            人工付款信息
          </a-divider>
          <a-form-item label="付款金额" name="amount" :rules="[{ required: true, message: '请输入金额' }]">
            <a-input-number v-model:value="formModel.amount" :min="0" :precision="2" style="width: 100%" size="large" :prefix="h(DollarOutlined)" />
          </a-form-item>
          <a-form-item label="交易号">
            <a-input v-model:value="formModel.trade_no" placeholder="请输入交易号" size="large" :maxlength="INPUT_LIMITS.PAYMENT_TRADE_NO" />
          </a-form-item>
          <a-form-item label="截图 URL">
            <a-input v-model:value="formModel.screenshot_url" placeholder="请输入付款截图链接" size="large" :maxlength="INPUT_LIMITS.URL" />
          </a-form-item>
          <a-form-item label="备注">
            <a-textarea v-model:value="formModel.note" :rows="3" placeholder="请输入备注信息" :maxlength="INPUT_LIMITS.PAYMENT_NOTE" show-count />
          </a-form-item>
          <a-button type="primary" html-type="submit" :loading="paying" size="large" block class="submit-btn">
            <CheckOutlined />
            提交付款信息
          </a-button>
        </template>

        <!-- Dynamic Schema Form -->
        <template v-else>
          <a-divider class="form-divider">
            <PayCircleOutlined class="divider-icon" />
            支付信息
          </a-divider>
          <template v-for="field in schemaFields" :key="field.key">
            <a-form-item
              :label="field.label"
              :name="field.key"
              :rules="field.required ? [{ required: true, message: `请输入${field.label}` }] : []"
            >
              <a-input
                v-if="field.type === 'text'"
                v-model:value="formModel[field.key]"
                :placeholder="field.placeholder"
                size="large"
              />
              <a-input-password
                v-else-if="field.type === 'password'"
                v-model:value="formModel[field.key]"
                :placeholder="field.placeholder"
                size="large"
              />
              <a-input-number
                v-else-if="field.type === 'number'"
                v-model:value="formModel[field.key]"
                :placeholder="field.placeholder"
                style="width: 100%"
                size="large"
              />
              <a-switch v-else-if="field.type === 'boolean'" v-model:checked="formModel[field.key]" />
              <a-textarea
                v-else-if="field.type === 'textarea'"
                v-model:value="formModel[field.key]"
                :placeholder="field.placeholder"
                :rows="3"
              />
              <a-select v-else-if="field.type === 'select'" v-model:value="formModel[field.key]" :placeholder="field.placeholder" size="large">
                <a-select-option v-for="opt in field.options || []" :key="opt.value" :value="opt.value">
                  {{ opt.label }}
                </a-select-option>
              </a-select>
              <a-input v-else v-model:value="formModel[field.key]" :placeholder="field.placeholder" size="large" />
            </a-form-item>
          </template>
          <a-button type="primary" html-type="submit" :loading="paying" size="large" block class="submit-btn">
            <PayCircleOutlined />
            发起支付
          </a-button>
        </template>
      </a-form>
    </a-modal>

    <!-- WeChat QR Code Modal (wechat_native) -->
    <a-modal v-model:open="wechatQrOpen" title="微信扫码支付" :footer="null" :width="420">
      <div style="display:flex; flex-direction:column; align-items:center; gap: 12px;">
        <a-spin :spinning="wechatQrLoading">
          <img v-if="wechatQrDataUrl" :src="wechatQrDataUrl" alt="wechat-qrcode" style="width: 260px; height: 260px;" />
        </a-spin>
        <div style="color: rgba(0,0,0,0.65); font-size: 12px; text-align:center;">
          请使用微信扫描二维码完成支付；支付完成后返回此页点击“刷新订单”。
        </div>
        <a-typography-text v-if="wechatQrUrl" copyable>{{ wechatQrUrl }}</a-typography-text>
      </div>
    </a-modal>

    <!-- WeChat JSAPI Modal (wechat_jsapi) -->
    <a-modal v-model:open="wechatJsapiOpen" title="微信支付（JSAPI）" :footer="null" :width="560">
      <a-alert
        type="info"
        show-icon
        message="JSAPI 仅在微信内置浏览器可直接调起；如果你在普通浏览器中，请复制参数并在微信内打开页面发起支付。"
        style="margin-bottom: 12px"
      />
      <a-button type="primary" :disabled="!wechatJsapiParams" @click="invokeWeChatJsapi" style="margin-bottom: 12px;">
        在微信内调起支付
      </a-button>
      <a-textarea :value="wechatJsapiParamsJson" :rows="10" readonly />
    </a-modal>
  </div>
</template>

<script setup>
import { computed, onMounted, onBeforeUnmount, reactive, ref, watch, h } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useOrdersStore } from "@/stores/orders";
import { useCatalogStore } from "@/stores/catalog";
import { useAuthStore } from "@/stores/auth";
import { submitOrderPayment, listPaymentProviders, createOrderPayment, cancelOrder, listVps } from "@/services/user";
import { message, Modal } from "ant-design-vue";
import { INPUT_LIMITS } from "@/constants/inputLimits";
import OrderStatusBadge from "@/components/OrderStatusBadge.vue";
import { createSseConnection } from "@/services/sse";
import QRCode from "qrcode";

// Icons
import {
  ShoppingCartOutlined,
  FileTextOutlined,
  PayCircleOutlined,
  ClockCircleOutlined,
  CheckCircleOutlined,
  SyncOutlined,
  ExclamationCircleOutlined,
  ReloadOutlined,
  CreditCardOutlined,
  WalletOutlined,
  InfoCircleOutlined,
  BellOutlined,
  DollarOutlined,
  CheckOutlined,
  HistoryOutlined,
  CopyOutlined,
  ArrowLeftOutlined,
  UnorderedListOutlined,
  ApiOutlined,
  CloudServerOutlined,
  CloseCircleOutlined
} from "@ant-design/icons-vue";

const route = useRoute();
const router = useRouter();
const store = useOrdersStore();
const auth = useAuthStore();
const catalog = useCatalogStore();
const id = route.params.id;

const order = computed(() => {
  const row = store.currentOrder;
  if (!row) return null;
  return {
    id: row.id ?? row.ID,
    order_no: row.order_no ?? row.OrderNo,
    status: row.status ?? row.Status,
    total_amount: row.total_amount ?? row.TotalAmount,
    currency: row.currency ?? row.Currency,
    created_at: row.created_at ?? row.CreatedAt
  };
});

const canCancel = computed(() => {
  const status = order.value?.status || "";
  return status === "pending_payment" || status === "pending_review";
});

const orderItems = computed(() =>
  store.orderItems.map((row) => ({
    id: row.id ?? row.ID,
    package_id: row.package_id ?? row.PackageID,
    system_id: row.system_id ?? row.SystemID,
    action: row.action ?? row.Action,
    spec: row.spec ?? row.Spec,
    specText: formatSpec(row.spec ?? row.Spec, row),
    specDetail: formatSpecDetail(row.spec ?? row.Spec, row),
    qty: row.qty ?? row.Qty,
    amount: row.amount ?? row.Amount,
    status: row.status ?? row.Status,
    duration_months: row.duration_months ?? row.DurationMonths
  }))
);

const orderPayments = computed(() =>
  store.orderPayments.map((row) => ({
    id: row.id ?? row.ID,
    method: row.method ?? row.Method,
    amount: row.amount ?? row.Amount,
    trade_no: row.trade_no ?? row.TradeNo,
    status: row.status ?? row.Status,
    created_at: row.created_at ?? row.CreatedAt
  }))
);

const itemColumns = [
  { title: "Item ID", dataIndex: "id", key: "id", width: 90 },
  { title: "套餐 ID", dataIndex: "package_id", key: "package_id", width: 110 },
  { title: "系统 ID", dataIndex: "system_id", key: "system_id", width: 110 },
  { title: "规格", dataIndex: "specText", key: "specText", width: 280 },
  { title: "数量", dataIndex: "qty", key: "qty", width: 80 },
  { title: "金额", dataIndex: "amount", key: "amount", width: 110 },
  { title: "状态", dataIndex: "status", key: "status", width: 120 }
];

const paymentColumns = [
  { title: "方式", dataIndex: "method", key: "method", width: 120 },
  { title: "金额", dataIndex: "amount", key: "amount", width: 120 },
  { title: "交易号", dataIndex: "trade_no", key: "trade_no", width: 240 },
  { title: "状态", dataIndex: "status", key: "status", width: 120 },
  { title: "时间", dataIndex: "created_at", key: "created_at", width: 180 }
];

const findPackage = (packageId) => {
  if (!packageId) return null;
  return catalog.packages.find((pkg) => String(pkg.id) === String(packageId)) || null;
};

const formatSpec = (spec, row) => {
  const payload = typeof spec === "string" ? tryParseJson(spec) : spec || {};
  const action = String(row?.action ?? row?.Action ?? "");
  if (action === "resize") {
    const currentCpu = toNum(payload?.current_cpu);
    const currentMem = toNum(payload?.current_mem_gb);
    const currentDisk = toNum(payload?.current_disk_gb);
    const currentBw = toNum(payload?.current_bw_mbps);
    const targetCpu = toNum(payload?.target_cpu);
    const targetMem = toNum(payload?.target_mem_gb);
    const targetDisk = toNum(payload?.target_disk_gb);
    const targetBw = toNum(payload?.target_bw_mbps);
    if (targetCpu || targetMem || targetDisk || targetBw || currentCpu || currentMem || currentDisk || currentBw) {
      const summary = [];
      summary.push(`CPU ${fmtPair(currentCpu, targetCpu)}`);
      summary.push(`内存 ${fmtPair(currentMem, targetMem)}G`);
      summary.push(`磁盘 ${fmtPair(currentDisk, targetDisk)}G`);
      summary.push(`带宽 ${fmtPair(currentBw, targetBw)}M`);
      return summary.join(" / ");
    }
  }
  const pkg = findPackage(row?.package_id ?? row?.PackageID);
  const baseCores = Number(pkg?.cores ?? 0);
  const baseMem = Number(pkg?.memory_gb ?? 0);
  const baseDisk = Number(pkg?.disk_gb ?? 0);
  const baseBw = Number(pkg?.bandwidth_mbps ?? 0);
  const addCores = Number(payload?.add_cores ?? 0);
  const addMem = Number(payload?.add_mem_gb ?? 0);
  const addDisk = Number(payload?.add_disk_gb ?? 0);
  const addBw = Number(payload?.add_bw_mbps ?? 0);
  const totalCores = baseCores + addCores;
  const totalMem = baseMem + addMem;
  const totalDisk = baseDisk + addDisk;
  const totalBw = baseBw + addBw;
  const duration = payload?.duration_months ?? row?.duration_months ?? row?.DurationMonths;
  const parts = [];
  if (totalCores || totalMem || totalDisk || totalBw || baseCores || baseMem || baseDisk || baseBw) {
    parts.push(`CPU ${totalCores}`);
    parts.push(`内存 ${totalMem}G`);
    parts.push(`磁盘 ${totalDisk}G`);
    parts.push(`带宽 ${totalBw}M`);
  }
  if (duration) {
    parts.push(`时长 ${duration} 个月`);
  }
  return parts.length ? parts.join(" / ") : "-";
};

const formatSpecDetail = (spec, row) => {
  const payload = typeof spec === "string" ? tryParseJson(spec) : spec || {};
  const action = String(row?.action ?? row?.Action ?? "");
  if (action !== "resize") return [];

  const currentCpu = toNum(payload?.current_cpu);
  const currentMem = toNum(payload?.current_mem_gb);
  const currentDisk = toNum(payload?.current_disk_gb);
  const currentBw = toNum(payload?.current_bw_mbps);
  const targetCpu = toNum(payload?.target_cpu);
  const targetMem = toNum(payload?.target_mem_gb);
  const targetDisk = toNum(payload?.target_disk_gb);
  const targetBw = toNum(payload?.target_bw_mbps);
  const currentPkg = toNum(payload?.current_package_id);
  const targetPkg = toNum(payload?.target_package_id);
  const currentMonthly = Number(payload?.current_monthly || 0);
  const targetMonthly = Number(payload?.target_monthly || 0);
  const chargeAmount = Number(payload?.charge_amount || 0);
  const refundAmount = Number(payload?.refund_amount || 0);
  const refundToWallet = !!payload?.refund_to_wallet;

  const lines = [];
  lines.push(
    `原配置：CPU ${showNum(currentCpu)}核 / 内存 ${showNum(currentMem)}G / 磁盘 ${showNum(currentDisk)}G / 带宽 ${showNum(currentBw)}M` +
      (currentPkg ? ` / 套餐ID ${currentPkg}` : "")
  );
  lines.push(
    `新配置：CPU ${showNum(targetCpu)}核 / 内存 ${showNum(targetMem)}G / 磁盘 ${showNum(targetDisk)}G / 带宽 ${showNum(targetBw)}M` +
      (targetPkg ? ` / 套餐ID ${targetPkg}` : "")
  );

  const changes = [];
  pushDelta(changes, "CPU", currentCpu, targetCpu, "核");
  pushDelta(changes, "内存", currentMem, targetMem, "G");
  pushDelta(changes, "磁盘", currentDisk, targetDisk, "G");
  pushDelta(changes, "带宽", currentBw, targetBw, "M");
  if (currentMonthly || targetMonthly) {
    changes.push(`月费 ${fmtMoney(currentMonthly)} -> ${fmtMoney(targetMonthly)}`);
  }
  if (chargeAmount > 0) {
    changes.push(`补差价 ${fmtMoney(chargeAmount)}`);
  } else if (refundAmount > 0) {
    changes.push(`退款 ${fmtMoney(refundAmount)}${refundToWallet ? "（退回钱包）" : ""}`);
  }
  lines.push(`变动说明：${changes.length ? changes.join("，") : "无配置变化"}`);
  return lines;
};

const toNum = (val) => {
  const n = Number(val);
  return Number.isFinite(n) ? n : 0;
};

const showNum = (val) => (Number.isFinite(Number(val)) ? Number(val) : 0);

const fmtPair = (from, to) => {
  if (!from && !to) return "-";
  return `${showNum(from)} -> ${showNum(to)}`;
};

const fmtMoney = (val) => `¥${Number(val || 0).toFixed(2)}`;

const pushDelta = (out, label, from, to, unit) => {
  if (!Number.isFinite(from) || !Number.isFinite(to)) return;
  const diff = to - from;
  if (diff === 0) {
    out.push(`${label} 无变化`);
    return;
  }
  const sign = diff > 0 ? "+" : "";
  out.push(`${label} ${sign}${diff}${unit}`);
};

const tryParseJson = (text) => {
  try {
    return JSON.parse(text);
  } catch {
    return {};
  }
};

const stepIndex = computed(() => {
  const status = order.value?.status || "";
  const steps = ["draft", "pending_payment", "pending_review", "approved", "provisioning", "active"];
  const idx = steps.indexOf(status);
  return idx === -1 ? 0 : idx;
});

const isProvisioning = computed(() => {
  if ((order.value?.status || "") === "provisioning") return true;
  return orderItems.value.some((item) => (item.status || "") === "provisioning");
});

const instanceCount = computed(() =>
  orderItems.value.reduce((sum, item) => sum + Math.max(0, Number(item.qty || 0)), 0)
);

// Unified form model for validation
const formModel = reactive({
  method: "",
  amount: 0,
  trade_no: "",
  note: "",
  screenshot_url: ""
});

const providers = ref([]);
const paying = ref(false);
const paymentHint = ref("");
const paymentFormRef = ref();
const paymentModalVisible = ref(false);

// WeChat payment UX helpers (for plugin methods wechatpay_v3.wechat_native / wechat_jsapi)
const wechatQrOpen = ref(false);
const wechatQrLoading = ref(false);
const wechatQrUrl = ref("");
const wechatQrDataUrl = ref("");

const wechatJsapiOpen = ref(false);
const wechatJsapiParams = ref(null);
const wechatJsapiParamsJson = computed(() => {
  try {
    return wechatJsapiParams.value ? JSON.stringify(wechatJsapiParams.value, null, 2) : "";
  } catch {
    return "";
  }
});

const syncPayAmountFromOrder = () => {
  const raw = order.value?.total_amount ?? order.value?.totalAmount;
  if (raw === undefined || raw === null || raw === "") return;
  const amount = Number(raw);
  if (!Number.isFinite(amount)) return;
  formModel.amount = amount;
};

watch(
  () => order.value?.total_amount ?? order.value?.totalAmount,
  () => {
    syncPayAmountFromOrder();
  },
  { immediate: true }
);

const showPaymentModal = () => {
  syncPayAmountFromOrder();
  paymentModalVisible.value = true;
};

const closePaymentModal = () => {
  paymentModalVisible.value = false;
  paymentHint.value = "";
};

const openWeChatQr = async (codeUrl) => {
  const url = String(codeUrl || "");
  if (!url) return;
  wechatQrOpen.value = true;
  wechatQrUrl.value = url;
  wechatQrLoading.value = true;
  try {
    wechatQrDataUrl.value = await QRCode.toDataURL(url, { width: 260, margin: 1 });
  } catch {
    wechatQrDataUrl.value = "";
  } finally {
    wechatQrLoading.value = false;
  }
};

const openWeChatJsapi = (paramsJson) => {
  try {
    const parsed = JSON.parse(String(paramsJson || ""));
    wechatJsapiParams.value = parsed;
  } catch {
    wechatJsapiParams.value = null;
  }
  wechatJsapiOpen.value = true;
};

const invokeWeChatJsapi = async () => {
  const params = wechatJsapiParams.value;
  if (!params) return;
  const bridge = window.WeixinJSBridge;
  if (!bridge || typeof bridge.invoke !== "function") {
    message.warning("当前环境不支持 JSAPI，请在微信内打开此页面");
    return;
  }
  try {
    await new Promise((resolve, reject) => {
      bridge.invoke("getBrandWCPayRequest", params, (res) => {
        const msg = String(res?.err_msg || res?.errMsg || "");
        if (msg === "get_brand_wcpay_request:ok") return resolve(true);
        reject(new Error(msg || "pay failed"));
      });
    });
    message.success("已发起支付，请在微信中完成付款");
    wechatJsapiOpen.value = false;
  } catch (e) {
    message.error(e?.message || "微信支付发起失败");
  }
};

const normalizeSchemaFields = (schemaJson) => {
  if (!schemaJson) return [];
  try {
    const parsed = JSON.parse(schemaJson);
    if (Array.isArray(parsed)) return parsed;
    if (Array.isArray(parsed.fields)) return parsed.fields;
    if (parsed && typeof parsed === "object") {
      const props = parsed.properties || {};
      const required = new Set(parsed.required || []);
      return Object.keys(props).map((key) => {
        const prop = props[key] || {};
        const enumValues = Array.isArray(prop.enum) ? prop.enum : null;
        const type = enumValues
          ? "select"
          : prop.format === "password"
            ? "password"
            : prop.format === "textarea"
              ? "textarea"
              : prop.type === "number" || prop.type === "integer"
                ? "number"
                : prop.type === "boolean"
                  ? "boolean"
                  : "text";
        return {
          key,
          label: prop.title || prop.label || key,
          type,
          required: required.has(key),
          placeholder: prop.description || prop.placeholder || "",
          default: prop.default,
          options: enumValues
            ? enumValues.map((value) => ({
                label: String(value),
                value
              }))
            : []
        };
      });
    }
    return [];
  } catch {
    return [];
  }
};

const selectedProvider = computed(() => providers.value.find((item) => item.key === formModel.method));
const schemaFields = computed(() => {
  const provider = selectedProvider.value;
  if (!provider?.schema_json) return [];
  if (["approval", "balance", "custom", "yipay"].includes(provider.key || "")) return [];
  return normalizeSchemaFields(provider.schema_json);
});
const selectedInstructions = computed(() => {
  const configJson = selectedProvider.value?.config_json;
  if (!configJson) return "";
  try {
    const parsed = JSON.parse(configJson);
    return parsed.instructions || parsed.notice || "";
  } catch {
    return "";
  }
});

watch(
  () => formModel.method,
  (val, oldVal) => {
    if (!val) return;
    paymentHint.value = "";
    // Clean up previous schema fields from formModel
    if (oldVal) {
      const oldProvider = providers.value.find(p => p.key === oldVal);
      if (oldProvider && oldProvider.schema_json) {
        const oldFields = normalizeSchemaFields(oldProvider.schema_json);
        oldFields.forEach(field => {
          delete formModel[field.key];
        });
      }
    }
    // Initialize new schema fields
    schemaFields.value.forEach((field) => {
      if (formModel[field.key] === undefined) {
        if (field.type === "boolean") {
          formModel[field.key] = field.default ?? false;
        } else {
          formModel[field.key] = field.default ?? "";
        }
      }
    });
  }
);

const submitPayment = async () => {
  // Validate form before submission
  try {
    await paymentFormRef.value?.validate();
  } catch (e) {
    return;
  }

  if (!formModel.method) {
    message.warning("请选择付款方式");
    return;
  }
  if (String(formModel.method || "").length > INPUT_LIMITS.PAYMENT_METHOD) {
    message.error(`付款方式长度不能超过 ${INPUT_LIMITS.PAYMENT_METHOD} 个字符`);
    return;
  }
  if (String(formModel.trade_no || "").length > INPUT_LIMITS.PAYMENT_TRADE_NO) {
    message.error(`交易号长度不能超过 ${INPUT_LIMITS.PAYMENT_TRADE_NO} 个字符`);
    return;
  }
  if (String(formModel.screenshot_url || "").length > INPUT_LIMITS.URL) {
    message.error(`截图链接长度不能超过 ${INPUT_LIMITS.URL} 个字符`);
    return;
  }
  if (String(formModel.note || "").length > INPUT_LIMITS.PAYMENT_NOTE) {
    message.error(`备注长度不能超过 ${INPUT_LIMITS.PAYMENT_NOTE} 个字符`);
    return;
  }
  paying.value = true;
  try {
    if (formModel.method === "approval") {
      await submitOrderPayment(id, { ...formModel, amount: Number(formModel.amount || 0) }, `pay-${Date.now()}`);
      message.success("已提交付款信息");
      closePaymentModal();
      await store.fetchOrderDetail(id);
      return;
    }
    const base = import.meta.env.VITE_API_BASE || window.location.origin;
    // Extract only schema fields for extra
    const schemaFieldKeys = schemaFields.value.map(f => f.key);
    /** @type {Object} */
    const extra = {};
    schemaFieldKeys.forEach(key => {
      if (formModel[key] !== undefined) {
        extra[key] = formModel[key];
      }
    });
    const payload = {
      method: formModel.method,
      return_url: `${window.location.origin}/console/orders/${id}`,
      notify_url: `${base}/api/v1/payments/notify/${formModel.method}`,
      extra
    };
    const res = await createOrderPayment(id, payload);
    const result = res.data || {};
    if (result.extra?.instructions) {
      paymentHint.value = result.extra.instructions;
    }

    const payKind = String(result.extra?.pay_kind || "");
    if (payKind === "qr") {
      const url = String(result.extra?.code_url || result.pay_url || "");
      if (url) {
        await openWeChatQr(url);
        message.info("请扫码完成支付");
        closePaymentModal();
        return;
      }
    }
    if (payKind === "jsapi") {
      const paramsJson = String(result.extra?.jsapi_params_json || "");
      if (paramsJson) {
        openWeChatJsapi(paramsJson);
        closePaymentModal();
        return;
      }
    }
    if (payKind === "form") {
      const formHtml = String(result.extra?.form_html || "");
      if (formHtml) {
        const w = window.open("", "_blank");
        if (!w) {
          message.warning("浏览器拦截了弹窗，请允许弹窗后重试");
          return;
        }
        w.document.open();
        w.document.write(formHtml);
        w.document.close();
        message.info("已打开支付页面，请完成支付");
        closePaymentModal();
        return;
      }
    }
    if (payKind === "urlscheme") {
      const scheme = String(result.extra?.urlscheme || "");
      if (scheme) {
        window.location.href = scheme;
        message.info("正在拉起支付应用，请完成支付");
        closePaymentModal();
        return;
      }
    }
    if (payKind === "redirect") {
      const url = String(result.extra?.pay_url || result.pay_url || "");
      if (url) {
        window.open(url, "_blank");
        message.info("已打开支付页面，请完成支付");
        closePaymentModal();
        return;
      }
    }
    if (result.paid) {
      message.success("支付完成");
      closePaymentModal();
      await store.fetchOrderDetail(id);
      return;
    }
    if (result.pay_url) {
      window.open(result.pay_url, "_blank");
      message.info("已打开支付页面，请完成支付");
      closePaymentModal();
      return;
    }
    if (result.status === "manual") {
      message.info("该方式需要人工处理，请按提示完成支付");
      return;
    }
    message.success("支付请求已提交");
    closePaymentModal();
  } catch (e) {
    message.error(e.response?.data?.error || "发起支付失败");
  } finally {
    paying.value = false;
  }
};

const events = ref([]);
let sse;
let pollingTimer;
let vpsRetryTimer;
const hasAutoNavigated = ref(false);

const getBase = () => import.meta.env.VITE_API_BASE || "";

const startSse = () => {
  if (!auth.token) return;
  sse?.close();
  sse = createSseConnection(`${getBase()}/api/v1/orders/${id}/events`, {
    headers: { Authorization: `Bearer ${auth.token}` },
    onMessage: (msg) => {
      if (msg.data) {
        events.value.unshift(msg.data);
      }
    }
  });
};

const parseVpsInfo = (text) => {
  const info = {};
  const lines = text.split(/\n|\r/).map((line) => line.trim()).filter(Boolean);
  lines.forEach((line) => {
    if (line.includes(":")) {
      const [key, ...rest] = line.split(":");
      info[key.trim()] = rest.join(":").trim();
    } else if (line.includes("=")) {
      const [key, ...rest] = line.split("=");
      info[key.trim()] = rest.join("=").trim();
    }
  });
  return Object.keys(info).length ? info : null;
};

const vpsInfo = computed(() => {
  for (const ev of events.value) {
    const parsed = parseVpsInfo(ev);
    if (parsed) return parsed;
  }
  return null;
});

const copyVpsInfo = async () => {
  if (!vpsInfo.value) return;
  const text = Object.entries(vpsInfo.value)
    .map(([k, v]) => `${k}: ${v}`)
    .join("\n");
  await navigator.clipboard.writeText(text);
  message.success("已复制 VPS 信息");
};

const stopPolling = () => {
  if (pollingTimer) {
    clearInterval(pollingTimer);
    pollingTimer = undefined;
  }
};

const startPolling = () => {
  if (pollingTimer) return;
  pollingTimer = setInterval(async () => {
    try {
      await store.fetchOrderDetail(id);
    } catch {
      // ignore transient errors; next tick will retry
    }
  }, 3000);
};

const stopVpsRetry = () => {
  if (vpsRetryTimer) {
    clearInterval(vpsRetryTimer);
    vpsRetryTimer = undefined;
  }
};

const tryAutoNavigateToVps = async () => {
  if (hasAutoNavigated.value) return;
  if (instanceCount.value !== 1) return;
  const orderItemId = orderItems.value[0]?.id;
  if (!orderItemId) return;

  try {
    const res = await listVps();
    const items = res.data?.items || [];
    const vps = items.find((row) => String(row.order_item_id) === String(orderItemId));
    const vpsId = vps?.id;
    if (!vpsId) return;
    hasAutoNavigated.value = true;
    stopPolling();
    stopVpsRetry();
    await router.push(`/console/vps/${vpsId}`);
  } catch {
    // ignore and retry if scheduled
  }
};

onMounted(async () => {
  const providersRes = await listPaymentProviders();
  providers.value = providersRes.data?.items || [];
  if (!formModel.method && providers.value.length) {
    formModel.method = providers.value[0].key;
  }
  if (!catalog.packages.length) {
    await catalog.fetchCatalog();
  }
  await store.fetchOrderDetail(id);
  startSse();
});

watch(
  () => isProvisioning.value,
  (val) => {
    if (val) startPolling();
    else stopPolling();
  },
  { immediate: true }
);

watch(
  () => order.value?.status,
  async (status, prev) => {
    if (hasAutoNavigated.value) return;
    if (prev === "provisioning" && status === "active") {
      await tryAutoNavigateToVps();
      if (!hasAutoNavigated.value && instanceCount.value === 1) {
        stopVpsRetry();
        let attempts = 0;
        vpsRetryTimer = setInterval(async () => {
          attempts += 1;
          await tryAutoNavigateToVps();
          if (hasAutoNavigated.value || attempts >= 20) {
            stopVpsRetry();
          }
        }, 3000);
      }
    }
  }
);

const refresh = async () => {
  await store.refreshOrder(id);
};

const cancelCurrent = async () => {
  if (!canCancel.value) return;
  Modal.confirm({
    title: "撤销订单",
    content: "撤销后订单将变为已取消，无法继续支付。确认撤销吗？",
    okText: "确认撤销",
    cancelText: "暂不撤销",
    onOk: async () => {
      await cancelOrder(id);
      message.success("订单已撤销");
      await store.fetchOrderDetail(id);
    }
  });
};

onBeforeUnmount(() => {
  sse?.close();
  stopPolling();
  stopVpsRetry();
});

// Helper functions for styling
const getStatusGradient = (status) => {
  const gradients = {
    active: 'background: linear-gradient(135deg, #52c41a 0%, #73d13d 100%);',
    approved: 'background: linear-gradient(135deg, #1890ff 0%, #40a9ff 100%);',
    provisioning: 'background: linear-gradient(135deg, #722ed1 0%, #9254de 100%);',
    pending_review: 'background: linear-gradient(135deg, #faad14 0%, #ffc53d 100%);',
    pending_payment: 'background: linear-gradient(135deg, #ff4d4f 0%, #ff7875 100%);',
    failed: 'background: linear-gradient(135deg, #ff4d4f 0%, #ff7875 100%);',
    canceled: 'background: linear-gradient(135deg, #8c8c8c 0%, #bfbfbf 100%);',
    rejected: 'background: linear-gradient(135deg, #ff4d4f 0%, #ff7875 100%);'
  };
  return gradients[status] || 'background: linear-gradient(135deg, #8c8c8c 0%, #bfbfbf 100%);';
};

const getItemsCountColor = () => {
  const count = orderItems.value.length;
  if (count === 0) return 'default';
  if (count <= 2) return 'blue';
  if (count <= 5) return 'cyan';
  return 'purple';
};

const getItemStatusColor = (status) => {
  const colors = {
    pending: 'orange',
    active: 'green',
    canceled: 'red',
    failed: 'red'
  };
  return colors[status] || 'default';
};

const getPaymentStatusColor = (status) => {
  const colors = {
    pending: 'orange',
    success: 'green',
    failed: 'red',
    canceled: 'default'
  };
  return colors[status] || 'default';
};
</script>

<style scoped>
.order-detail-page {
  padding: 24px;
  max-width: 1400px;
  margin: 0 auto;
}

/* Page Header */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
  gap: 16px;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 16px;
  flex: 1;
}

.back-btn {
  height: 40px;
  padding: 0 16px;
  font-weight: 500;
  flex-shrink: 0;
}

.header-title-section {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.title-row {
  display: flex;
  align-items: center;
  gap: 10px;
}

.title-icon {
  font-size: 24px;
  color: var(--primary);
}

.page-title {
  font-size: 22px;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0;
}

.header-actions {
  flex-shrink: 0;
}

/* Stats Banner */
.stats-banner {
  display: flex;
  align-items: center;
  gap: 24px;
  padding: 24px 32px;
  background: var(--primary-gradient);
  border-radius: var(--radius-lg);
  margin-bottom: 24px;
  box-shadow: var(--shadow-lg);
  color: #fff;
}

.stat-item {
  display: flex;
  align-items: center;
  gap: 14px;
  flex: 1;
}

.stat-icon {
  font-size: 28px;
  color: rgba(255, 255, 255, 0.9);
}

.stat-icon.amount {
  font-size: 32px;
}

.stat-content {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.stat-label {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.8);
}

.stat-value {
  font-size: 16px;
  font-weight: 600;
  color: #fff;
}

.stat-amount {
  font-size: 24px;
  font-weight: 700;
}

.stat-divider {
  width: 1px;
  height: 40px;
  background: rgba(255, 255, 255, 0.3);
}

/* Progress Section */
.progress-section {
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: 24px 32px;
  margin-bottom: 24px;
}

.order-steps :deep(.ant-steps-item-process .ant-steps-item-icon),
.order-steps :deep(.ant-steps-item-finish .ant-steps-item-icon) {
  background: var(--primary);
  border-color: var(--primary);
}

.order-steps :deep(.ant-steps-item-process .ant-steps-item-title),
.order-steps :deep(.ant-steps-item-finish .ant-steps-item-title) {
  color: var(--primary);
}
.order-steps :deep(.ant-steps-icon) {
  color: white !important;
}
/* Content Grid */
.content-grid {
  display: grid;
  grid-template-columns: 1fr 380px;
  gap: 24px;
  align-items: start;
}

/* Section Cards */
.section-card {
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  overflow: hidden;
  margin-bottom: 24px;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border);
  background: var(--bg-secondary);
}

.card-icon {
  font-size: 18px;
  color: var(--primary);
}

.card-icon.spin {
  animation: spin 3s linear infinite;
}

@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

.card-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
  flex: 1;
}

.items-count {
  font-size: 12px;
  background: var(--primary-gradient);
  color: #fff;
  border: none;
  padding: 2px 10px;
}

.card-body {
  padding: 20px;
}

/* Items Table */
.items-table :deep(.ant-table) {
  background: transparent;
}

.items-table :deep(.ant-table-thead > tr > th) {
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border);
  padding: 12px 16px;
  font-weight: 600;
  font-size: 13px;
  color: var(--text-secondary);
}

.items-table :deep(.ant-table-tbody > tr > td) {
  padding: 14px 16px;
  border-bottom: 1px solid var(--border-light);
}

.items-table :deep(.ant-table-tbody > tr:hover > td) {
  background: var(--bg-secondary);
}

.spec-cell {
  display: flex;
  align-items: flex-start;
  gap: 8px;
}

.spec-icon {
  font-size: 14px;
  color: var(--primary);
  margin-top: 2px;
}

.spec-content {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.spec-main {
  color: var(--text-primary);
}

.spec-detail {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.spec-line {
  color: var(--text-secondary);
  font-size: 12px;
  line-height: 1.45;
}

.amount-cell {
  display: flex;
  align-items: baseline;
  gap: 2px;
  font-weight: 600;
  color: var(--primary);
}

.amount-symbol {
  font-size: 14px;
}

.amount-value {
  font-size: 16px;
}

/* Payment Alert */
.payment-alert {
  margin-bottom: 16px;
  border-radius: var(--radius-md);
}

/* Form Divider */
.form-divider {
  margin: 20px 0;
  font-size: 13px;
  font-weight: 600;
  color: var(--text-secondary);
}

.divider-icon {
  margin-right: 6px;
}

/* Submit Button */
.submit-btn {
  height: 44px;
  font-weight: 600;
}

/* Payment History Table */
.history-table :deep(.ant-table) {
  background: var(--bg-secondary);
  border-radius: var(--radius-md);
}

.history-table :deep(.ant-table-thead > tr > th) {
  background: transparent;
  font-weight: 600;
  font-size: 12px;
}

.history-table :deep(.ant-table-tbody > tr > td) {
  padding: 10px 12px;
  font-size: 13px;
}

/* Sidebar Cards */
.sidebar-card {
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  overflow: hidden;
  margin-bottom: 20px;
}

.sidebar-card .card-body {
  padding: 16px 20px;
}

.copy-btn {
  padding: 0;
  height: auto;
  color: var(--primary);
}

.progress-timeline {
  margin-top: 8px;
}

/* VPS Descriptions */
.vps-descriptions :deep(.ant-descriptions-item-label) {
  font-weight: 500;
  background: var(--bg-secondary);
}

.vps-descriptions :deep(.ant-descriptions-item-content) {
  background: transparent;
}

/* Action Card */
.action-card {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.action-btn {
  height: 44px;
  font-weight: 600;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.action-btn-primary {
  height: 50px;
  font-size: 16px;
  background: var(--primary-gradient);
  border: none;
  box-shadow: 0 4px 12px rgba(0, 102, 255, 0.3);
}

.action-btn-primary:hover {
  background: var(--primary-gradient);
  box-shadow: 0 6px 16px rgba(0, 102, 255, 0.4);
  transform: translateY(-1px);
}

/* Responsive */
@media (max-width: 1024px) {
  .content-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 768px) {
  .order-detail-page {
    padding: 16px;
  }

  .page-header {
    flex-direction: column;
    align-items: stretch;
    gap: 16px;
  }

  .header-left {
    flex-direction: column;
    align-items: stretch;
    gap: 12px;
  }

  .title-row {
    flex-wrap: wrap;
  }

  .page-title {
    font-size: 18px;
  }

  .stats-banner {
    flex-direction: column;
    align-items: flex-start;
    gap: 16px;
    padding: 20px;
  }

  .stat-divider {
    display: none;
  }

  .stat-item {
    width: 100%;
  }

  .progress-section {
    padding: 20px 16px;
  }

  :deep(.ant-steps) {
    flex-direction: column;
  }

  :deep(.ant-steps-item) {
    flex-direction: row;
  }
}
</style>
