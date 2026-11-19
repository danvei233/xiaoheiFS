<template>
  <div class="cart-page">
    <!-- Header -->
    <div class="cart-header">
      <div class="header-left">
        <div class="title-section">
          <h1 class="page-title">{{ t('console.cart.title') }}</h1>
          <p class="page-subtitle">{{ t('console.cart.items', { n: dataSource.length }) }}</p>
        </div>
      </div>
      <div class="header-actions">
        <a-button @click="cart.fetchCart" :loading="cart.loading">
          <ReloadOutlined />
          {{ t('common.refresh') }}
        </a-button>
      </div>
    </div>

    <!-- Empty State -->
    <div v-if="!dataSource.length && !cart.loading" class="empty-state">
      <ShoppingCartOutlined class="empty-icon" />
      <h3 class="empty-title">{{ t('console.cart.empty') }}</h3>
      <p class="empty-desc">{{ t('console.cart.emptyDesc') }}</p>
      <a-button type="primary" @click="$router.push('/console/buy')" size="large">
        <ShoppingOutlined />
        {{ t('console.cart.goShop') }}
      </a-button>
    </div>

    <!-- Cart Content -->
    <div v-else class="cart-grid">
      <!-- Items List -->
      <div class="items-section">
        <a-table
          :columns="columns"
          :data-source="dataSource"
          :loading="cart.loading"
          :scroll="{ x: 1140 }"
          :pagination="false"
          row-key="id"
          class="cart-table"
        >
          <template #bodyCell="{ column, record }">
            <!-- Package -->
            <template v-if="column.key === 'package'">
              <div class="item-package">
                <div class="package-icon">
                  <DesktopOutlined />
                </div>
                <div class="package-info">
                  <div class="package-name">{{ getPackageName(record.package_id) }}</div>
                  <div class="package-id">ID: {{ record.package_id }}</div>
                </div>
              </div>
            </template>

            <!-- System -->
            <template v-else-if="column.key === 'system'">
              <div class="item-system">
                <CodeOutlined class="system-icon" />
                <span>{{ getSystemName(record.system_id) }}</span>
              </div>
            </template>

            <!-- Specs -->
            <template v-else-if="column.key === 'specs'">
              <div class="item-specs">
                <div class="spec-item">
                  <ApiOutlined class="spec-icon cpu" />
                  <span>{{ t('console.vps.cpuSpec', { n: getCpu(record) }) }}</span>
                </div>
                <div class="spec-item">
                  <DatabaseOutlined class="spec-icon memory" />
                  <span>{{ getMemory(record) }}G</span>
                </div>
                <div class="spec-item">
                  <HddOutlined class="spec-icon disk" />
                  <span>{{ getDisk(record) }}G</span>
                </div>
                <div class="spec-item">
                  <CloudServerOutlined class="spec-icon bandwidth" />
                  <span>{{ getBandwidth(record) }}M</span>
                </div>
                <div class="spec-item">
                  <ClockCircleOutlined class="spec-icon duration" />
                  <span>{{ getDuration(record) }}</span>
                </div>
              </div>
            </template>

            <!-- Quantity -->
            <template v-else-if="column.key === 'quantity'">
              <a-input-number
                :value="record.qty"
                :min="1"
                :max="99"
                @update:value="(val) => updateQty(record, val)"
              />
            </template>

            <!-- Price -->
            <template v-else-if="column.key === 'price'">
              <div class="item-price">
                <span class="price-symbol">¥</span>
                <span class="price-value">{{ (Number(record.amount) * Number(record.qty)).toFixed(2) }}</span>
              </div>
            </template>

            <!-- Actions -->
            <template v-else-if="column.key === 'actions'">
              <a-popconfirm
                :title="t('console.cart.removeConfirm')"
                @confirm="cart.removeItem(record.id)"
                :ok-text="t('common.confirm')"
                :cancel-text="t('common.cancel')"
              >
                <a-button type="text" danger>
                  <DeleteOutlined />
                </a-button>
              </a-popconfirm>
            </template>
          </template>
        </a-table>
      </div>

      <!-- Summary Card -->
      <div class="summary-section">
        <div class="summary-card">
          <div class="summary-header">
            <FileTextOutlined class="summary-header-icon" />
            <span>{{ t('console.cart.summary') }}</span>
          </div>

          <div class="summary-body">
            <div class="summary-row">
              <span class="summary-label">{{ t('console.cart.itemCount') }}</span>
              <span class="summary-value">{{ totalItems }}</span>
            </div>

            <div class="summary-row">
              <span class="summary-label">{{ t('console.cart.subtotal') }}</span>
              <span class="summary-value">¥{{ totalAmount.toFixed(2) }}</span>
            </div>
            <div class="summary-row">
              <span class="summary-label">{{ t('console.cart.coupon') }}</span>
              <a-space>
                <a-input
                  v-model:value="couponCode"
                  size="small"
                  :placeholder="t('console.cart.couponPlaceholder')"
                  style="width: 140px"
                />
                <a-button size="small" :loading="couponLoading" @click="applyCoupon">{{ t('console.cart.use') }}</a-button>
              </a-space>
            </div>
            <div class="summary-row" v-if="couponPreview">
              <span class="summary-label">{{ t('console.cart.discount') }}</span>
              <span class="summary-value" style="color: var(--success)">-¥{{ Number(couponPreview.discount || 0).toFixed(2) }}</span>
            </div>

            <a-divider class="summary-divider" />

            <div class="summary-row summary-total">
              <span class="summary-label">{{ t('console.cart.total') }}</span>
              <span class="summary-value summary-price">¥{{ payableTotal.toFixed(2) }}</span>
            </div>
          </div>

          <a-button
            type="primary"
            size="large"
            block
            :loading="submitting"
            @click="submitOrder"
            class="checkout-btn"
          >
            <CheckCircleOutlined />
            {{ t('console.cart.checkout') }}
          </a-button>

          <div class="summary-footer">
            <SafetyCertificateOutlined />
            <span>{{ t('console.cart.securePay') }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from "vue";
import { useCartStore } from "@/stores/cart";
import { useCatalogStore } from "@/stores/catalog";
import { createOrderFromCartWithCoupon, previewCoupon } from "@/services/user";
import { message } from "ant-design-vue";
import { useRouter } from "vue-router";
import { t } from "@/locales";
import {
  ShoppingCartOutlined,
  ReloadOutlined,
  DesktopOutlined,
  DeleteOutlined,
  CheckCircleOutlined,
  ShoppingOutlined,
  CodeOutlined,
  FileTextOutlined,
  SafetyCertificateOutlined,
  ApiOutlined,
  DatabaseOutlined,
  HddOutlined,
  CloudServerOutlined,
  ClockCircleOutlined
} from "@ant-design/icons-vue";

const cart = useCartStore();
const router = useRouter();
const catalog = useCatalogStore();
const submitting = ref(false);
const couponCode = ref("");
const couponLoading = ref(false);
const couponPreview = ref(null);

const columns = computed(() => [
  { title: t('console.cart.product'), dataIndex: "package", key: "package", width: 280 },
  { title: t('console.cart.systemImage'), dataIndex: "system", key: "system", width: 180 },
  { title: t('console.vps.spec'), key: "specs", width: 360 },
  { title: t('common.quantity'), key: "quantity", width: 120, align: "center" },
  { title: t('common.amount'), key: "price", width: 140, align: "right" },
  { title: "", key: "actions", width: 60, align: "center" }
]);

const dataSource = computed(() =>
  cart.items.map((item) => ({
    ...item,
    specText: formatSpec(item.spec, item)
  }))
);

const findPackage = (packageId) => {
  if (!packageId) return null;
  return catalog.packages.find((pkg) => String(pkg.id) === String(packageId)) || null;
};

const findSystemImage = (systemId) => {
  if (!systemId) return null;
  return catalog.systemImages.find((img) => String(img.id) === String(systemId)) || null;
};

const getPackageName = (packageId) => {
  const pkg = findPackage(packageId);
  return pkg?.name || t('console.cart.packageFallback', { id: packageId });
};

const getSystemName = (systemId) => {
  const img = findSystemImage(systemId);
  return img?.name || t('console.cart.systemFallback', { id: systemId });
};

const getCpu = (item) => {
  const pkg = findPackage(item.package_id);
  const baseCores = Number(pkg?.cores || 0);
  const addCores = Number(item.spec?.add_cores || 0);
  return baseCores + addCores;
};

const getMemory = (item) => {
  const pkg = findPackage(item.package_id);
  const baseMem = Number(pkg?.memory_gb || 0);
  const addMem = Number(item.spec?.add_mem_gb || 0);
  return baseMem + addMem;
};

const getDisk = (item) => {
  const pkg = findPackage(item.package_id);
  const baseDisk = Number(pkg?.disk_gb || 0);
  const addDisk = Number(item.spec?.add_disk_gb || 0);
  return baseDisk + addDisk;
};

const getBandwidth = (item) => {
  const pkg = findPackage(item.package_id);
  const baseBw = Number(pkg?.bandwidth_mbps || 0);
  const addBw = Number(item.spec?.add_bw_mbps || 0);
  return baseBw + addBw;
};

const getDuration = (item) => {
  const spec = item.spec;
  if (spec?.duration_months) {
    return t('console.cart.durationMonths', { n: spec.duration_months });
  }
  if (spec?.cycle_qty && spec?.billing_cycle_id) {
    return t('console.cart.cycle', { n: spec.cycle_qty });
  }
  return "-";
};

const formatSpec = (spec, item) => {
  if (!spec) return "-";
  const parts = [];
  const cpu = getCpu(item);
  const mem = getMemory(item);
  const disk = getDisk(item);
  const bw = getBandwidth(item);
  if (cpu) parts.push(t('console.vps.cpuSpec', { n: cpu }));
  if (mem) parts.push(t('console.vps.memSpec', { n: mem }));
  if (disk) parts.push(t('console.vps.diskSpec', { n: disk }));
  if (bw) parts.push(t('console.vps.bwSpec', { n: bw }));
  if (spec.duration_months) {
    parts.push(t('console.cart.duration', { n: spec.duration_months }));
  } else if (spec.cycle_qty && spec.billing_cycle_id) {
    parts.push(t('console.cart.cycle', { n: spec.cycle_qty }) + ` x ID ${spec.billing_cycle_id}`);
  }
  return parts.length ? parts.join(" / ") : "-";
};

const totalAmount = computed(() =>
  dataSource.value.reduce((sum, item) => sum + Number(item.amount || 0) * Number(item.qty || 1), 0)
);

const totalItems = computed(() =>
  dataSource.value.reduce((sum, item) => sum + Number(item.qty || 1), 0)
);
const payableTotal = computed(() => {
  if (!couponPreview.value) return totalAmount.value;
  const finalTotal = Number(couponPreview.value.final_total || totalAmount.value);
  return finalTotal >= 0 ? finalTotal : 0;
});

const updateQty = (record, val) => {
  cart.updateItem(record.id, { spec: record.spec, qty: val });
};

const submitOrder = async () => {
  submitting.value = true;
  try {
    const coupon = (couponCode.value || "").trim();
    const res = await createOrderFromCartWithCoupon(coupon ? { coupon_code: coupon } : undefined, `order-${Date.now()}`);
    const orderId = res.data?.order?.id || res.data?.order?.ID || res.data?.id || res.data?.ID;
    message.success(t('console.cart.orderCreated'));
    if (orderId) {
      router.push(`/console/orders/${orderId}`);
    }
    cart.fetchCart();
  } catch (err) {
    message.error(err?.response?.data?.message || t('console.cart.orderFailed'));
  } finally {
    submitting.value = false;
  }
};

const applyCoupon = async () => {
  const code = (couponCode.value || "").trim();
  if (!code) {
    message.warning(t('console.cart.couponRequired'));
    return;
  }
  couponLoading.value = true;
  try {
    const res = await previewCoupon({ coupon_code: code });
    couponPreview.value = res.data || null;
    message.success(t('console.cart.couponApplied'));
  } catch (err) {
    couponPreview.value = null;
    message.error(err?.response?.data?.error || t('console.cart.couponInvalid'));
  } finally {
    couponLoading.value = false;
  }
};

watch(couponCode, () => {
  couponPreview.value = null;
});

watch(
  () => dataSource.value.map((item) => `${item.id}:${item.qty}:${item.amount}`).join("|"),
  () => {
    couponPreview.value = null;
  }
);

onMounted(() => {
  if (!catalog.packages.length) {
    catalog.fetchCatalog();
  }
  cart.fetchCart();
});
</script>

<style scoped>
.cart-page {
  max-width: 1400px;
  margin: 0 auto;
  padding: 24px;
}

/* Header */
.cart-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 32px;
}

.title-section {
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

/* Empty State */
.empty-state {
  text-align: center;
  padding: 80px 20px;
  background: var(--card);
  border-radius: var(--radius-lg);
  border: 1px solid var(--border);
}

.empty-icon {
  font-size: 64px;
  color: var(--border-dark);
  margin-bottom: 24px;
}

.empty-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 8px;
}

.empty-desc {
  font-size: 14px;
  color: var(--text-secondary);
  margin: 0 0 32px;
}

/* Grid */
.cart-grid {
  display: grid;
  grid-template-columns: 1fr 340px;
  gap: 24px;
  align-items: start;
}

/* Table Section */
.items-section {
  background: var(--card);
  border-radius: var(--radius-lg);
  border: 1px solid var(--border);
  overflow: hidden;
}

.cart-table :deep(.ant-table) {
  background: transparent;
}

.cart-table :deep(.ant-table-thead > tr > th) {
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border);
  padding: 14px 16px;
  font-weight: 600;
  font-size: 13px;
  color: var(--text-secondary);
}

.cart-table :deep(.ant-table-tbody > tr > td) {
  padding: 16px;
  border-bottom: 1px solid var(--border-light);
}

.cart-table :deep(.ant-table-tbody > tr:hover > td) {
  background: var(--bg-secondary);
}

.cart-table :deep(.ant-table-tbody > tr:last-child > td) {
  border-bottom: none;
}

/* Item Package */
.item-package {
  display: flex;
  align-items: center;
  gap: 12px;
}

.package-icon {
  width: 40px;
  height: 40px;
  border-radius: var(--radius-md);
  background: var(--primary-gradient-subtle);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--primary);
  font-size: 18px;
  flex-shrink: 0;
}

.package-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.package-name {
  font-weight: 500;
  font-size: 14px;
  color: var(--text-primary);
}

.package-id {
  font-size: 12px;
  color: var(--text-tertiary);
  font-family: 'JetBrains Mono', monospace;
}

/* Item System */
.item-system {
  display: flex;
  align-items: center;
  gap: 8px;
}

.system-icon {
  font-size: 16px;
  color: var(--text-tertiary);
}

/* Item Specs */
.item-specs {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.spec-item {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 8px;
  background: var(--bg-secondary);
  border-radius: var(--radius-sm);
  font-size: 12px;
  color: var(--text-secondary);
}

.spec-icon {
  font-size: 12px;
}

.spec-icon.cpu { color: var(--primary); }
.spec-icon.memory { color: var(--success); }
.spec-icon.disk { color: var(--info); }
.spec-icon.bandwidth { color: var(--warning); }
.spec-icon.duration { color: var(--accent); }

/* Item Price */
.item-price {
  display: flex;
  align-items: baseline;
  justify-content: flex-end;
  gap: 2px;
}

.price-symbol {
  font-size: 14px;
  color: var(--text-tertiary);
}

.price-value {
  font-size: 18px;
  font-weight: 700;
  color: var(--primary);
}

/* Summary Section */
.summary-section {
  position: sticky;
  top: 24px;
}

.summary-card {
  background: var(--card);
  border-radius: var(--radius-lg);
  border: 1px solid var(--border);
  padding: 20px;
  box-shadow: var(--shadow-sm);
}

.summary-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 20px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--border);
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
}

.summary-header-icon {
  font-size: 18px;
  color: var(--primary);
}

.summary-body {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-bottom: 20px;
}

.summary-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.summary-label {
  font-size: 14px;
  color: var(--text-secondary);
}

.summary-value {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-primary);
}

.summary-divider {
  margin: 8px 0;
}

.summary-total {
  padding: 12px 0;
}

.summary-total .summary-label {
  font-size: 15px;
  font-weight: 600;
  color: var(--text-primary);
}

.summary-price {
  font-size: 24px;
  font-weight: 700;
  color: var(--primary);
}

.checkout-btn {
  height: 48px;
  font-weight: 600;
  margin-bottom: 16px;
}

.summary-footer {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 12px;
  background: var(--bg-secondary);
  border-radius: var(--radius-md);
  font-size: 12px;
  color: var(--text-tertiary);
}

.summary-footer :deep(.anticon) {
  font-size: 14px;
  color: var(--success);
}

/* Responsive */
@media (max-width: 1024px) {
  .cart-grid {
    grid-template-columns: 1fr;
  }

  .summary-section {
    position: static;
  }
}

@media (max-width: 768px) {
  .cart-page {
    padding: 16px;
  }

  .cart-header {
    flex-direction: column;
    align-items: stretch;
    gap: 16px;
  }

  .title-section {
    flex-direction: column;
    gap: 4px;
  }

  .page-title {
    font-size: 22px;
  }

  .cart-table :deep(.ant-table-thead > tr > th),
  .cart-table :deep(.ant-table-tbody > tr > td) {
    padding: 12px;
  }

  .item-specs {
    gap: 6px;
  }

  .spec-item {
    font-size: 11px;
    padding: 3px 6px;
  }
}
</style>
