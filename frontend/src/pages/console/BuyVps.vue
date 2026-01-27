<template>
  <div class="buy-page">
    <div class="page-header">
      <div>
        <div class="page-title">购买 VPS</div>
        <div class="subtle">按需选择资源配置并自动计算价格</div>
      </div>
    </div>

    <a-row :gutter="24">
      <a-col :xs="24" :lg="16">
        <a-card class="config-card" :bordered="false">
          <!-- Step Indicator -->
          <div class="steps-wrapper">
            <a-steps :current="stepIndex" size="small">
              <a-step title="选择地区" />
              <a-step title="选择线路" />
              <a-step title="选择套餐" />
              <a-step title="系统镜像" />
              <a-step title="确认配置" />
            </a-steps>
          </div>

          <a-divider style="margin: 20px 0" />

          <a-form layout="vertical">
            <!-- Region Selection -->
            <a-form-item label="选择地区">
              <a-select
                v-model:value="form.regionId"
                placeholder="请选择地区"
                size="large"
                :options="regionOptions"
              >
              </a-select>
            </a-form-item>

            <!-- Line Selection -->
            <template v-if="form.regionId">
              <a-form-item label="选择线路">
                <a-select
                  v-model:value="form.planGroupId"
                  placeholder="请选择线路"
                  size="large"
                  :loading="catalog.loading"
                >
                  <a-select-option v-for="group in planGroups" :key="group.id" :value="group.id" :disabled="isPlanGroupDisabled(group)">
                    <div class="select-option-content">
                      <span>{{ group.name }}</span>
                      <a-tag
                        v-if="getCapacityLabel(group.capacity_remaining ?? group.capacityRemaining)"
                        :color="getCapacityColor(group.capacity_remaining ?? group.capacityRemaining)"
                        size="small"
                      >
                        {{ getCapacityLabel(group.capacity_remaining ?? group.capacityRemaining) }}
                      </a-tag>
                    </div>
                  </a-select-option>
                </a-select>
              </a-form-item>

              <!-- Package Selection -->
              <template v-if="form.planGroupId">
                <a-form-item label="选择套餐">
                  <template v-if="packages.length === 0">
                    <a-empty :image="Empty.PRESENTED_IMAGE_SIMPLE" description="暂无可用套餐" />
                  </template>
                  <template v-else>
                    <a-row :gutter="[16, 16]">
                      <a-col v-for="pkg in packages" :key="pkg.id" :xs="24" :md="12" :lg="8">
                        <div
                          class="package-card"
                          :class="{ selected: form.packageId === pkg.id, disabled: isPackageDisabled(pkg) }"
                          @click="selectPackage(pkg)"
                        >
                          <div class="pkg-header">
                            <div class="pkg-name">{{ pkg.name }}</div>
                            <a-tag v-if="pkg.name?.includes('推荐') || pkg.name?.includes('热门')" color="red" size="small">HOT</a-tag>
                          </div>
                          <div class="pkg-specs">
                            <div class="spec">
                              <span class="spec-value">{{ pkg.cores }}</span>
                              <span class="spec-label">核</span>
                            </div>
                            <div class="spec-divider"></div>
                            <div class="spec">
                              <span class="spec-value">{{ pkg.memory_gb }}</span>
                              <span class="spec-label">GB</span>
                            </div>
                            <div class="spec-divider"></div>
                            <div class="spec">
                              <span class="spec-value">{{ pkg.disk_gb }}</span>
                              <span class="spec-label">GB</span>
                            </div>
                          </div>
                          <div class="pkg-footer">
                            <div class="pkg-price">
                              <span class="price-symbol">¥</span>
                              <span class="price-amount">{{ Number(pkg.monthly_price || 0).toFixed(2) }}</span>
                              <span class="price-unit">/月</span>
                            </div>
                            <div class="pkg-tags">
                              <a-tag v-if="pkg.port_num" size="small">{{ pkg.port_num }}端口</a-tag>
                              <a-tag v-if="pkg.cpu_model" size="small">{{ pkg.cpu_model }}</a-tag>
                            </div>
                          </div>
                          <div v-if="form.packageId === pkg.id" class="pkg-check">
                            <CheckCircleFilled />
                          </div>
                        </div>
                      </a-col>
                    </a-row>
                  </template>
                </a-form-item>

                <!-- System Image Selection -->
                <template v-if="form.packageId">
                  <a-form-item label="系统镜像">
                    <a-select
                      v-model:value="form.systemId"
                      placeholder="请选择系统镜像"
                      size="large"
                      :loading="loadingImages"
                    >
                      <a-select-option v-for="img in systemImages" :key="img.id" :value="img.id">
                        {{ img.name }} ({{ img.type }})
                      </a-select-option>
                    </a-select>
                  </a-form-item>

                  <!-- Addons -->
                  <a-divider style="margin: 24px 0 16px" />
                  <div class="section-header">附加配置</div>

                  <a-row :gutter="[24, 16]">
                    <a-col :xs="24" :md="12">
                      <div class="addon-item">
                        <div class="addon-header">
                          <span class="addon-title">CPU 核心</span>
                          <span class="addon-value" v-if="form.add_cores > 0">
                            +{{ form.add_cores }}核 · +¥{{ (form.add_cores * (selectedPlanGroup?.unit_core || 0)).toFixed(2) }}/月
                          </span>
                          <span class="addon-value-placeholder" v-else>不添加</span>
                        </div>
                        <a-slider
                          v-model:value="form.add_cores"
                          :min="addonRule.add_core_min"
                          :max="addonRule.add_core_max"
                          :step="addonRule.add_core_step"
                          :marks="{ 0: '0', [addonRule.add_core_max]: addonRule.add_core_max + '' }"
                          :tooltip-formatter="(val) => val + '核'"
                        />
                      </div>
                    </a-col>

                    <a-col :xs="24" :md="12">
                      <div class="addon-item">
                        <div class="addon-header">
                          <span class="addon-title">内存</span>
                          <span class="addon-value" v-if="form.add_mem_gb > 0">
                            +{{ form.add_mem_gb }}GB · +¥{{ (form.add_mem_gb * (selectedPlanGroup?.unit_mem || 0)).toFixed(2) }}/月
                          </span>
                          <span class="addon-value-placeholder" v-else>不添加</span>
                        </div>
                        <a-slider
                          v-model:value="form.add_mem_gb"
                          :min="addonRule.add_mem_min"
                          :max="addonRule.add_mem_max"
                          :step="addonRule.add_mem_step"
                          :marks="{ 0: '0', [addonRule.add_mem_max]: addonRule.add_mem_max + 'G' }"
                          :tooltip-formatter="(val) => val + 'G'"
                        />
                      </div>
                    </a-col>

                    <a-col :xs="24" :md="12">
                      <div class="addon-item">
                        <div class="addon-header">
                          <span class="addon-title">磁盘空间</span>
                          <span class="addon-value" v-if="form.add_disk_gb > 0">
                            +{{ form.add_disk_gb }}GB · +¥{{ (form.add_disk_gb * (selectedPlanGroup?.unit_disk || 0)).toFixed(2) }}/月
                          </span>
                          <span class="addon-value-placeholder" v-else>不添加</span>
                        </div>
                        <a-slider
                          v-model:value="form.add_disk_gb"
                          :min="addonRule.add_disk_min"
                          :max="addonRule.add_disk_max"
                          :step="addonRule.add_disk_step"
                          :marks="{ 0: '0', [addonRule.add_disk_max]: addonRule.add_disk_max + 'G' }"
                          :tooltip-formatter="(val) => val + 'G'"
                        />
                      </div>
                    </a-col>

                    <a-col :xs="24" :md="12">
                      <div class="addon-item">
                        <div class="addon-header">
                          <span class="addon-title">带宽</span>
                          <span class="addon-value" v-if="form.add_bw_mbps > 0">
                            +{{ form.add_bw_mbps }}Mbps · +¥{{ (form.add_bw_mbps * (selectedPlanGroup?.unit_bw || 0)).toFixed(2) }}/月
                          </span>
                          <span class="addon-value-placeholder" v-else>不添加</span>
                        </div>
                        <a-slider
                          v-model:value="form.add_bw_mbps"
                          :min="addonRule.add_bw_min"
                          :max="addonRule.add_bw_max"
                          :step="addonRule.add_bw_step"
                          :marks="{ 0: '0', [addonRule.add_bw_max]: addonRule.add_bw_max + 'M' }"
                          :tooltip-formatter="(val) => val + 'Mbps'"
                        />
                      </div>
                    </a-col>
                  </a-row>

                  <!-- Billing Cycle -->
                  <a-divider style="margin: 24px 0 16px" />
                  <div class="section-header">计费周期</div>

                  <a-row :gutter="16">
                    <a-col :xs="24" :md="12">
                      <a-form-item label="购买周期">
                        <a-select v-model:value="form.billingCycleId" placeholder="选择周期" size="large">
                          <a-select-option v-for="cycle in billingCycles" :key="cycle.id" :value="cycle.id">
                            {{ cycle.name }} ({{ cycle.months }}个月)
                            <template v-if="cycle.multiplier > 1">
                              <a-tag color="orange" size="small">{{ cycle.multiplier }}倍</a-tag>
                            </template>
                          </a-select-option>
                        </a-select>
                      </a-form-item>
                    </a-col>
                    <a-col :xs="24" :md="12">
                      <a-form-item label="购买数量">
                        <a-input-number v-model:value="form.qty" :min="1" :max="10" size="large" style="width: 100%">
                          <template #addonAfter>台</template>
                        </a-input-number>
                      </a-form-item>
                    </a-col>
                  </a-row>

                  <a-form-item label="周期数量" v-if="form.billingCycleId">
                    <a-input-number v-model:value="form.cycleQty" :min="1" :max="12" size="large" style="width: 200px">
                      <template #addonAfter>个周期</template>
                    </a-input-number>
                  </a-form-item>
                </template>
              </template>
            </template>
          </a-form>
        </a-card>
      </a-col>

      <!-- Sidebar -->
      <a-col :xs="24" :lg="8">
        <a-affix :offset-top="24">
          <div>
            <PriceCalculator
              :base-price="basePrice"
              :addon-price="addonPrice"
              :cycle-multiplier="cycleMultiplier"
              :qty="form.qty"
            />

            <a-card class="summary-card" :bordered="false" title="配置摘要">
              <div class="summary-grid">
                <div class="summary-row">
                  <span class="summary-key">地区</span>
                  <span class="summary-val">{{ selectedRegion?.name || '-' }}</span>
                </div>
                <div class="summary-row">
                  <span class="summary-key">线路</span>
                  <span class="summary-val">{{ selectedPlanGroup?.name || '-' }}</span>
                </div>
                <div class="summary-row">
                  <span class="summary-key">套餐</span>
                  <span class="summary-val">{{ selectedPackage?.name || '-' }}</span>
                </div>
                <div class="summary-row" v-if="selectedPackage?.port_num">
                  <span class="summary-key">端口</span>
                  <span class="summary-val">{{ selectedPackage.port_num }}</span>
                </div>
                <div class="summary-row">
                  <span class="summary-key">系统</span>
                  <span class="summary-val">{{ selectedSystem?.name || '-' }}</span>
                </div>
                <div class="summary-row">
                  <span class="summary-key">周期</span>
                  <span class="summary-val">{{ selectedCycle?.name || '-' }} × {{ form.cycleQty }}</span>
                </div>
                <div class="summary-row" v-if="hasAddons">
                  <span class="summary-key">附加</span>
                  <span class="summary-val">
                    <a-tag size="small" color="blue" v-if="form.add_cores">+{{ form.add_cores }}核</a-tag>
                    <a-tag size="small" color="blue" v-if="form.add_mem_gb">+{{ form.add_mem_gb }}G</a-tag>
                    <a-tag size="small" color="blue" v-if="form.add_disk_gb">+{{ form.add_disk_gb }}G</a-tag>
                    <a-tag size="small" color="blue" v-if="form.add_bw_mbps">+{{ form.add_bw_mbps }}M</a-tag>
                  </span>
                </div>
              </div>
              <a-divider style="margin: 12px 0" />
              <a-space style="width: 100%" direction="vertical" :size="12">
                <a-button size="large" block @click="addToCart" :disabled="!canCheckout">
                  加入购物车
                </a-button>
                <a-button type="primary" size="large" block @click="createOrderNow" :disabled="!canCheckout">
                  立即下单
                </a-button>
              </a-space>
            </a-card>
          </div>
        </a-affix>
      </a-col>
    </a-row>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref, watch } from "vue";
import { useCatalogStore } from "@/stores/catalog";
import { useCartStore } from "@/stores/cart";
import { listSystemImages, createOrder } from "@/services/user";
import { message, Empty } from "ant-design-vue";
import { useRouter } from "vue-router";
import { CheckCircleFilled } from "@ant-design/icons-vue";
import PriceCalculator from "@/components/PriceCalculator.vue";

const catalog = useCatalogStore();
const cart = useCartStore();
const router = useRouter();

const form = reactive({
  regionId: null,
  planGroupId: null,
  packageId: null,
  systemId: null,
  add_cores: 0,
  add_mem_gb: 0,
  add_disk_gb: 0,
  add_bw_mbps: 0,
  billingCycleId: null,
  cycleQty: 1,
  qty: 1
});

const systemImages = ref([]);
const loadingImages = ref(false);

const regions = computed(() => catalog.regions.filter((r) => r.active !== false));
const regionOptions = computed(() =>
  regions.value.map(r => ({ label: r.name, value: r.id }))
);

const planGroups = computed(() =>
  catalog.planGroups.filter((g) => g.active !== false && g.visible !== false && g.region_id === form.regionId)
);

const packages = computed(() => {
  if (!form.planGroupId) return [];
  return catalog.packages.filter((pkg) => {
    if (pkg.active === false || pkg.visible === false) return false;
    const groupId = pkg.plan_group_id ?? pkg.planGroupId ?? pkg.PlanGroupID;
    return groupId === form.planGroupId;
  });
});

const billingCycles = computed(() =>
  catalog.billingCycles.length
    ? catalog.billingCycles.filter((cycle) => cycle.active !== false)
    : [{ id: 1, name: "按月", months: 1, multiplier: 1 }]
);

const selectedRegion = computed(() => regions.value.find((r) => r.id === form.regionId));
const selectedPlanGroup = computed(() => planGroups.value.find((g) => g.id === form.planGroupId));
const selectedPackage = computed(() => packages.value.find((p) => p.id === form.packageId));
const selectedSystem = computed(() => systemImages.value.find((s) => s.id === form.systemId));
const selectedCycle = computed(() => billingCycles.value.find((c) => c.id === form.billingCycleId));

const addonRule = computed(() => ({
  add_core_min: selectedPlanGroup.value?.add_core_min ?? 0,
  add_core_max: selectedPlanGroup.value?.add_core_max ?? 64,
  add_core_step: selectedPlanGroup.value?.add_core_step ?? 1,
  add_mem_min: selectedPlanGroup.value?.add_mem_min ?? 0,
  add_mem_max: selectedPlanGroup.value?.add_mem_max ?? 256,
  add_mem_step: selectedPlanGroup.value?.add_mem_step ?? 1,
  add_disk_min: selectedPlanGroup.value?.add_disk_min ?? 0,
  add_disk_max: selectedPlanGroup.value?.add_disk_max ?? 2000,
  add_disk_step: selectedPlanGroup.value?.add_disk_step ?? 10,
  add_bw_min: selectedPlanGroup.value?.add_bw_min ?? 0,
  add_bw_max: selectedPlanGroup.value?.add_bw_max ?? 1000,
  add_bw_step: selectedPlanGroup.value?.add_bw_step ?? 10
}));

const basePrice = computed(() => Number(selectedPackage.value?.monthly_price || 0));
const addonPrice = computed(() => {
  if (!selectedPlanGroup.value) return 0;
  return (
    Number(form.add_cores || 0) * Number(selectedPlanGroup.value.unit_core || 0) +
    Number(form.add_mem_gb || 0) * Number(selectedPlanGroup.value.unit_mem || 0) +
    Number(form.add_disk_gb || 0) * Number(selectedPlanGroup.value.unit_disk || 0) +
    Number(form.add_bw_mbps || 0) * Number(selectedPlanGroup.value.unit_bw || 0)
  );
});

const cycleMultiplier = computed(() => {
  const cycle = selectedCycle.value;
  const multiplier = Number(cycle?.multiplier || cycle?.months || 1);
  return multiplier * Number(form.cycleQty || 1);
});

const stepIndex = computed(() => {
  if (!form.regionId) return 0;
  if (!form.planGroupId) return 1;
  if (!form.packageId) return 2;
  if (!form.systemId) return 3;
  return 4;
});

const hasAddons = computed(() =>
  form.add_cores > 0 || form.add_mem_gb > 0 || form.add_disk_gb > 0 || form.add_bw_mbps > 0
);

const canCheckout = computed(() => form.packageId && form.systemId);

const getCapacityLabel = (value) => {
  const remaining = Number(value);
  if (!Number.isFinite(remaining)) return "";
  if (remaining < 0) return "不限";
  if (remaining === 0) return "售罄";
  return `余量 ${remaining}`;
};

const getCapacityColor = (value) => {
  const label = getCapacityLabel(value);
  if (label === "售罄") return "red";
  if (label === "不限") return "green";
  return "blue";
};

const isPlanGroupDisabled = (group) => {
  if (!group) return true;
  if (group.active === false || group.visible === false) return true;
  const remaining = Number(group.capacity_remaining ?? group.capacityRemaining);
  return Number.isFinite(remaining) ? remaining === 0 : false;
};

const isPackageDisabled = (pkg) => {
  if (!pkg) return true;
  if (pkg.active === false || pkg.visible === false) return true;
  const remaining = Number(pkg.capacity_remaining ?? pkg.capacityRemaining);
  return Number.isFinite(remaining) ? remaining === 0 : false;
};

const selectPackage = (pkg) => {
  if (isPackageDisabled(pkg)) return;
  form.packageId = pkg.id;
};

// Reset downstream selections when upstream changes
watch(() => form.regionId, () => {
  form.planGroupId = null;
  form.packageId = null;
  form.systemId = null;
});

watch(() => form.planGroupId, () => {
  form.packageId = null;
  form.systemId = null;
});

watch(() => form.packageId, () => {
  form.systemId = null;
});

// Auto-select first available option
watch(regions, (list) => {
  if (!form.regionId && list.length) {
    form.regionId = list[0].id;
  }
}, { immediate: true });

watch(planGroups, (list) => {
  if (!list.length) {
    form.planGroupId = null;
    return;
  }
  const available = list.filter((item) => !isPlanGroupDisabled(item));
  const next = available[0] || null;
  const exists = list.some((item) => item.id === form.planGroupId);
  if (!form.planGroupId || !exists) {
    form.planGroupId = next?.id ?? null;
  }
}, { immediate: true });

watch(packages, (list) => {
  if (!list.length) {
    form.packageId = null;
    return;
  }
  const available = list.filter((item) => !isPackageDisabled(item));
  const next = available[0] || null;
  const exists = list.some((item) => item.id === form.packageId);
  if (!form.packageId || !exists) {
    form.packageId = next?.id ?? null;
  }
}, { immediate: true });

watch(() => form.planGroupId, async (val) => {
  form.systemId = null;
  if (!val) {
    systemImages.value = [];
    return;
  }
  loadingImages.value = true;
  try {
    const res = await listSystemImages({ plan_group_id: val });
    systemImages.value = (res.data?.items || []).filter((img) => img.enabled !== false);
  } finally {
    loadingImages.value = false;
  }
});

watch(systemImages.value, (list) => {
  if (!list.length) return;
  const exists = list.some((img) => img.id === form.systemId);
  if (!form.systemId || !exists) {
    form.systemId = list[0].id;
  }
}, { deep: true });

watch(() => billingCycles.value.length, () => {
  if (!form.billingCycleId && billingCycles.value.length) {
    form.billingCycleId = billingCycles.value[0].id;
  }
}, { immediate: true });

const addToCart = async () => {
  if (!canCheckout.value) {
    message.error("请选择套餐与系统镜像");
    return;
  }
  await cart.addItem({
    package_id: form.packageId,
    system_id: form.systemId,
    spec: {
      add_cores: form.add_cores,
      add_mem_gb: form.add_mem_gb,
      add_disk_gb: form.add_disk_gb,
      add_bw_mbps: form.add_bw_mbps,
      billing_cycle_id: form.billingCycleId,
      cycle_qty: form.cycleQty,
      duration_months: Number(selectedCycle.value?.months || 1) * Number(form.cycleQty || 1)
    },
    qty: form.qty
  });
  message.success("已加入购物车");
};

const createOrderNow = async () => {
  if (!canCheckout.value) {
    message.error("请选择套餐与系统镜像");
    return;
  }
  const res = await createOrder(
    {
      items: [
        {
          package_id: form.packageId,
          system_id: form.systemId,
          spec: {
            add_cores: form.add_cores,
            add_mem_gb: form.add_mem_gb,
            add_disk_gb: form.add_disk_gb,
            add_bw_mbps: form.add_bw_mbps,
            billing_cycle_id: form.billingCycleId,
            cycle_qty: form.cycleQty,
            duration_months: Number(selectedCycle.value?.months || 1) * Number(form.cycleQty || 1)
          },
          qty: form.qty
        }
      ]
    },
    `order-${Date.now()}`
  );
  const orderId = res.data?.order?.id || res.data?.id;
  message.success("订单已创建");
  if (orderId) {
    router.push(`/console/orders/${orderId}`);
  }
};

onMounted(() => {
  catalog.fetchCatalog();
});
</script>

<style scoped>
.buy-page {
  padding: 0;
}

.page-header {
  margin-bottom: 20px;
}

.page-title {
  font-size: 20px;
  font-weight: 600;
  color: #1a1a1a;
  margin-bottom: 4px;
}

.subtle {
  color: #8c8c8c;
  font-size: 14px;
}

.steps-wrapper {
  padding: 0 8px;
}

.steps-wrapper :deep(.ant-steps-item-process .ant-steps-item-icon) {
  background: #1677ff;
  border-color: #1677ff;
}

.config-card {
  border-radius: 8px;
}

.config-card :deep(.ant-card-body) {
  padding: 24px;
}

.select-option-content {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  width: 100%;
}

/* Package Card */
.package-card {
  position: relative;
  border: 2px solid #e5e7eb;
  border-radius: 12px;
  padding: 16px;
  cursor: pointer;
  transition: all 0.2s ease;
  background: #fff;
}

.package-card:hover:not(.disabled) {
  border-color: #1677ff;
  box-shadow: 0 4px 12px rgba(22, 119, 255, 0.1);
}

.package-card.selected {
  border-color: #1677ff;
  background: #f0f7ff;
  box-shadow: 0 4px 12px rgba(22, 119, 255, 0.15);
}

.package-card.disabled {
  opacity: 0.5;
  cursor: not-allowed;
  background: #f5f5f5;
}

.pkg-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.pkg-name {
  font-size: 15px;
  font-weight: 600;
  color: #1a1a1a;
}

.pkg-specs {
  display: flex;
  align-items: center;
  justify-content: space-around;
  padding: 12px 0;
  margin-bottom: 12px;
  background: #fafafa;
  border-radius: 8px;
}

.spec {
  text-align: center;
}

.spec-value {
  font-size: 20px;
  font-weight: 600;
  color: #1677ff;
  display: block;
}

.spec-label {
  font-size: 12px;
  color: #8c8c8c;
}

.spec-divider {
  width: 1px;
  height: 24px;
  background: #e5e7eb;
}

.pkg-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.pkg-price {
  display: flex;
  align-items: baseline;
  gap: 2px;
}

.price-symbol {
  font-size: 14px;
  color: #ff4d4f;
  font-weight: 500;
}

.price-amount {
  font-size: 20px;
  color: #ff4d4f;
  font-weight: 700;
}

.price-unit {
  font-size: 12px;
  color: #8c8c8c;
}

.pkg-tags {
  display: flex;
  gap: 4px;
  flex-wrap: wrap;
}

.pkg-check {
  position: absolute;
  top: 8px;
  right: 8px;
  color: #52c41a;
  font-size: 20px;
}

/* Section */
.section-header {
  font-size: 15px;
  font-weight: 600;
  color: #1a1a1a;
  margin-bottom: 16px;
}

/* Addon Slider */
.addon-item {
  padding: 16px;
  background: #fafafa;
  border-radius: 8px;
  border: 1px solid #e5e7eb;
}

.addon-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.addon-title {
  font-size: 14px;
  font-weight: 500;
  color: #1a1a1a;
}

.addon-value {
  font-size: 13px;
  color: #52c41a;
  font-weight: 500;
}

.addon-value-placeholder {
  font-size: 13px;
  color: #bfbfbf;
}

.addon-item :deep(.ant-slider) {
  margin: 0;
}

.addon-item :deep(.ant-slider-rail) {
  background: #e5e7eb;
}

.addon-item :deep(.ant-slider-track) {
  background: #1677ff;
}

.addon-item :deep(.ant-slider-handle) {
  border-color: #1677ff;
}

.addon-item :deep(.ant-slider-handle:hover),
.addon-item :deep(.ant-slider-handle:focus) {
  border-color: #4096ff;
}

.addon-item :deep(.ant-slider-mark-text) {
  font-size: 11px;
  color: #8c8c8c;
}

.addon-item :deep(.ant-slider-mark-text-active) {
  color: #1677ff;
}

/* Summary */
.summary-card {
  margin-top: 16px;
  border-radius: 8px;
}

.summary-card :deep(.ant-card-head) {
  border-bottom: 1px solid #e5e7eb;
  min-height: 48px;
  padding: 0 16px;
}

.summary-card :deep(.ant-card-head-title) {
  font-weight: 600;
  font-size: 15px;
  padding: 12px 0;
}

.summary-card :deep(.ant-card-body) {
  padding: 16px;
}

.summary-grid {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.summary-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 14px;
}

.summary-key {
  color: #595959;
}

.summary-val {
  color: #1a1a1a;
  font-weight: 500;
  text-align: right;
  flex: 1;
  margin-left: 16px;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 4px;
  flex-wrap: wrap;
}
</style>
