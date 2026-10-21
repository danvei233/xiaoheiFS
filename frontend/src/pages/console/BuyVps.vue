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
              <a-step title="Goods Type" />
              <a-step title="选择地区" />
              <a-step title="选择线路" />
              <a-step title="选择套餐" />
              <a-step title="系统镜像" />
              <a-step title="确认配置" />
            </a-steps>
          </div>

          <a-divider style="margin: 20px 0" />

          <a-form layout="vertical">
            <a-form-item label="Goods Type">
              <a-select
                v-model:value="form.goodsTypeId"
                placeholder="Select goods type"
                size="large"
                :options="goodsTypeOptions"
              />
            </a-form-item>
            <template v-if="form.goodsTypeId">
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

            </template>
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
                          <span class="addon-value-placeholder" v-if="addonMeta.core.disabled">已禁用</span>
                          <span class="addon-value" v-else-if="form.add_cores > 0">
                            +{{ form.add_cores }}核 · +¥{{ (form.add_cores * (selectedPlanGroup?.unit_core || 0)).toFixed(2) }}/月
                          </span>
                          <span class="addon-value-placeholder" v-else>不添加</span>
                        </div>
                        <a-slider
                          v-model:value="form.add_cores"
                          :min="addonMeta.core.min"
                          :max="addonMeta.core.max"
                          :step="addonMeta.core.step"
                          :marks="buildSliderMarks(addonMeta.core.min, addonMeta.core.max, '')"
                          :disabled="addonMeta.core.disabled"
                          :tooltip-formatter="(val) => val + '核'"
                        />
                      </div>
                    </a-col>

                    <a-col :xs="24" :md="12">
                      <div class="addon-item">
                        <div class="addon-header">
                          <span class="addon-title">内存</span>
                          <span class="addon-value-placeholder" v-if="addonMeta.mem.disabled">已禁用</span>
                          <span class="addon-value" v-else-if="form.add_mem_gb > 0">
                            +{{ form.add_mem_gb }}GB · +¥{{ (form.add_mem_gb * (selectedPlanGroup?.unit_mem || 0)).toFixed(2) }}/月
                          </span>
                          <span class="addon-value-placeholder" v-else>不添加</span>
                        </div>
                        <a-slider
                          v-model:value="form.add_mem_gb"
                          :min="addonMeta.mem.min"
                          :max="addonMeta.mem.max"
                          :step="addonMeta.mem.step"
                          :marks="buildSliderMarks(addonMeta.mem.min, addonMeta.mem.max, 'G')"
                          :disabled="addonMeta.mem.disabled"
                          :tooltip-formatter="(val) => val + 'G'"
                        />
                      </div>
                    </a-col>

                    <a-col :xs="24" :md="12">
                      <div class="addon-item">
                        <div class="addon-header">
                          <span class="addon-title">磁盘空间</span>
                          <span class="addon-value-placeholder" v-if="addonMeta.disk.disabled">已禁用</span>
                          <span class="addon-value" v-else-if="form.add_disk_gb > 0">
                            +{{ form.add_disk_gb }}GB · +¥{{ (form.add_disk_gb * (selectedPlanGroup?.unit_disk || 0)).toFixed(2) }}/月
                          </span>
                          <span class="addon-value-placeholder" v-else>不添加</span>
                        </div>
                        <a-slider
                          v-model:value="form.add_disk_gb"
                          :min="addonMeta.disk.min"
                          :max="addonMeta.disk.max"
                          :step="addonMeta.disk.step"
                          :marks="buildSliderMarks(addonMeta.disk.min, addonMeta.disk.max, 'G')"
                          :disabled="addonMeta.disk.disabled"
                          :tooltip-formatter="(val) => val + 'G'"
                        />
                      </div>
                    </a-col>

                    <a-col :xs="24" :md="12">
                      <div class="addon-item">
                        <div class="addon-header">
                          <span class="addon-title">带宽</span>
                          <span class="addon-value-placeholder" v-if="addonMeta.bw.disabled">已禁用</span>
                          <span class="addon-value" v-else-if="form.add_bw_mbps > 0">
                            +{{ form.add_bw_mbps }}Mbps · +¥{{ (form.add_bw_mbps * (selectedPlanGroup?.unit_bw || 0)).toFixed(2) }}/月
                          </span>
                          <span class="addon-value-placeholder" v-else>不添加</span>
                        </div>
                        <a-slider
                          v-model:value="form.add_bw_mbps"
                          :min="addonMeta.bw.min"
                          :max="addonMeta.bw.max"
                          :step="addonMeta.bw.step"
                          :marks="buildSliderMarks(addonMeta.bw.min, addonMeta.bw.max, 'M')"
                          :disabled="addonMeta.bw.disabled"
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
              <a-form-item label="优惠码" style="margin-bottom: 12px">
                <a-input-group compact>
                  <a-input v-model:value="form.couponCode" placeholder="可选，输入优惠码" style="width: calc(100% - 90px)" />
                  <a-button style="width: 90px" :loading="couponPreviewLoading" @click="applyCouponPreview">使用</a-button>
                </a-input-group>
              </a-form-item>
              <div class="summary-grid" style="margin-bottom: 12px" v-if="couponPreview">
                <div class="summary-row">
                  <span class="summary-key">原价</span>
                  <span class="summary-val">¥{{ Number(couponPreview.original_total || computedOriginalTotal).toFixed(2) }}</span>
                </div>
                <div class="summary-row">
                  <span class="summary-key">优惠</span>
                  <span class="summary-val" style="color: var(--success)">-¥{{ Number(couponPreview.discount || 0).toFixed(2) }}</span>
                </div>
                <div class="summary-row">
                  <span class="summary-key">优惠后</span>
                  <span class="summary-val">¥{{ Number(couponPreview.final_total || computedOriginalTotal).toFixed(2) }}</span>
                </div>
              </div>
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
import { listSystemImages, createOrder, previewCoupon } from "@/services/user";
import { message, Empty } from "ant-design-vue";
import { useRouter } from "vue-router";
import { CheckCircleFilled } from "@ant-design/icons-vue";
import PriceCalculator from "@/components/PriceCalculator.vue";

const catalog = useCatalogStore();
const cart = useCartStore();
const router = useRouter();

const form = reactive({
  goodsTypeId: null,
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
  qty: 1,
  couponCode: ""
});

const systemImages = ref([]);
const loadingImages = ref(false);
const couponPreviewLoading = ref(false);
const couponPreview = ref(null);

const goodsTypes = computed(() => (catalog.goodsTypes || []).filter((gt) => gt.active !== false));
const goodsTypeOptions = computed(() => goodsTypes.value.map((gt) => ({ label: gt.name, value: gt.id })));

const regions = computed(() =>
  catalog.regions.filter((r) => {
    if (r.active === false) return false;
    if (!form.goodsTypeId) return false;
    return String(r.goods_type_id) === String(form.goodsTypeId);
  })
);
const regionOptions = computed(() =>
  regions.value.map(r => ({ label: r.name, value: r.id }))
);

const planGroups = computed(() =>
  catalog.planGroups.filter((g) => {
    if (g.active === false || g.visible === false) return false;
    if (!form.regionId) return false;
    if (form.goodsTypeId && String(g.goods_type_id) !== String(form.goodsTypeId)) return false;
    return g.region_id === form.regionId;
  })
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
  catalog.billingCycles.filter((cycle) => cycle.active !== false)
);

const selectedRegion = computed(() => regions.value.find((r) => r.id === form.regionId));
const selectedPlanGroup = computed(() => planGroups.value.find((g) => g.id === form.planGroupId));
const selectedPackage = computed(() => packages.value.find((p) => p.id === form.packageId));
const selectedSystem = computed(() => systemImages.value.find((s) => s.id === form.systemId));
const selectedCycle = computed(() => billingCycles.value.find((c) => c.id === form.billingCycleId));

const resolveAddonRule = (minRaw, maxRaw, stepRaw, fallbackMax) => {
  const min = Number(minRaw ?? 0);
  const max = Number(maxRaw ?? 0);
  const step = Math.max(1, Number(stepRaw ?? 1));
  if (min === -1 || max === -1) {
    return { disabled: true, min: 0, max: 0, step: 1 };
  }
  const effectiveMin = min > 0 ? min : 0;
  const effectiveMax = max > 0 ? max : fallbackMax;
  return {
    disabled: false,
    min: effectiveMin,
    max: Math.max(effectiveMin, effectiveMax),
    step
  };
};

const clampAddonValue = (value, rule) => {
  if (rule.disabled) return 0;
  const step = Math.max(1, Number(rule.step || 1));
  const min = Number(rule.min || 0);
  const max = Math.max(min, Number(rule.max || min));
  let next = Number(value || 0);
  if (!Number.isFinite(next)) next = min;
  next = Math.max(min, Math.min(max, next));
  next = min + Math.round((next - min) / step) * step;
  if (next > max) next = max;
  if (next < min) next = min;
  return next;
};

const buildSliderMarks = (min, max, suffix = "") => {
  if (max <= min) return { [min]: `${min}${suffix}` };
  return { [min]: `${min}${suffix}`, [max]: `${max}${suffix}` };
};

const addonMeta = computed(() => {
  const group = selectedPlanGroup.value || {};
  return {
    core: resolveAddonRule(group.add_core_min, group.add_core_max, group.add_core_step, 64),
    mem: resolveAddonRule(group.add_mem_min, group.add_mem_max, group.add_mem_step, 256),
    disk: resolveAddonRule(group.add_disk_min, group.add_disk_max, group.add_disk_step, 2000),
    bw: resolveAddonRule(group.add_bw_min, group.add_bw_max, group.add_bw_step, 1000)
  };
});

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
  if (!form.goodsTypeId) return 0;
  if (!form.regionId) return 1;
  if (!form.planGroupId) return 2;
  if (!form.packageId) return 3;
  if (!form.systemId) return 4;
  return 5;
});

const hasAddons = computed(() =>
  form.add_cores > 0 || form.add_mem_gb > 0 || form.add_disk_gb > 0 || form.add_bw_mbps > 0
);

const canCheckout = computed(() => form.packageId && form.systemId);
const computedOriginalTotal = computed(() =>
  (basePrice.value + addonPrice.value) * cycleMultiplier.value * Number(form.qty || 1)
);

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
watch(() => form.goodsTypeId, () => {
  form.regionId = null;
  form.planGroupId = null;
  form.packageId = null;
  form.systemId = null;
});

watch(() => form.regionId, () => {
  form.planGroupId = null;
  form.packageId = null;
  form.systemId = null;
});

watch(() => form.planGroupId, () => {
  form.packageId = null;
  form.systemId = null;
});

watch(
  addonMeta,
  (meta) => {
    form.add_cores = clampAddonValue(form.add_cores, meta.core);
    form.add_mem_gb = clampAddonValue(form.add_mem_gb, meta.mem);
    form.add_disk_gb = clampAddonValue(form.add_disk_gb, meta.disk);
    form.add_bw_mbps = clampAddonValue(form.add_bw_mbps, meta.bw);
  },
  { immediate: true, deep: true }
);

watch(() => form.packageId, () => {
  form.systemId = null;
});

// Auto-select first available option
watch(goodsTypes, (list) => {
  if (!form.goodsTypeId && list.length) {
    form.goodsTypeId = list[0].id;
  }
}, { immediate: true });

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
  const plan = planGroups.value.find((item) => item.id === val);
  const lineID = Number(plan?.line_id || 0);
  if (!lineID) {
    systemImages.value = [];
    return;
  }
  loadingImages.value = true;
  try {
    const res = await listSystemImages({ plan_group_id: val, line_id: lineID });
    systemImages.value = (res.data?.items || []).filter((img) => img.enabled !== false);
  } finally {
    loadingImages.value = false;
  }
});

watch(() => systemImages.value, (list) => {
  if (!list.length) return;
  const exists = list.some((img) => img.id === form.systemId);
  if (!form.systemId || !exists) {
    form.systemId = list[0].id;
  }
}, { deep: true });

watch(() => billingCycles.value.length, () => {
  if (!billingCycles.value.length) {
    form.billingCycleId = null;
    return;
  }
  if (!form.billingCycleId && billingCycles.value.length) {
    form.billingCycleId = billingCycles.value[0].id;
  }
}, { immediate: true });

const buildOrderSpecPayload = () => {
  const spec = {
    add_cores: form.add_cores,
    add_mem_gb: form.add_mem_gb,
    add_disk_gb: form.add_disk_gb,
    add_bw_mbps: form.add_bw_mbps,
    cycle_qty: form.cycleQty,
    duration_months: Number(selectedCycle.value?.months || 1) * Number(form.cycleQty || 1)
  };
  if (form.billingCycleId) {
    spec.billing_cycle_id = form.billingCycleId;
  }
  return spec;
};

const addToCart = async () => {
  if (!canCheckout.value) {
    message.error("请选择套餐与系统镜像");
    return;
  }
  await cart.addItem({
    package_id: form.packageId,
    system_id: form.systemId,
    spec: buildOrderSpecPayload(),
    qty: form.qty
  });
  message.success("已加入购物车");
};

const buildPreviewItems = () => [
  {
    package_id: form.packageId,
    system_id: form.systemId,
    spec: buildOrderSpecPayload(),
    qty: form.qty
  }
];

const applyCouponPreview = async () => {
  if (!canCheckout.value) {
    message.warning("请先完成套餐与系统选择");
    return;
  }
  const code = (form.couponCode || "").trim();
  if (!code) {
    message.warning("请先输入优惠码");
    return;
  }
  couponPreviewLoading.value = true;
  try {
    const res = await previewCoupon({
      coupon_code: code,
      items: buildPreviewItems()
    });
    couponPreview.value = res.data || null;
    message.success("优惠码已应用");
  } catch (err) {
    couponPreview.value = null;
    message.error(err?.response?.data?.error || "优惠码不可用");
  } finally {
    couponPreviewLoading.value = false;
  }
};

const createOrderNow = async () => {
  if (!canCheckout.value) {
    message.error("请选择套餐与系统镜像");
    return;
  }
  const res = await createOrder(
    {
      coupon_code: (form.couponCode || "").trim() || undefined,
      items: [
        {
          package_id: form.packageId,
          system_id: form.systemId,
          spec: buildOrderSpecPayload(),
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

watch(
  () => [
    form.packageId,
    form.systemId,
    form.add_cores,
    form.add_mem_gb,
    form.add_disk_gb,
    form.add_bw_mbps,
    form.billingCycleId,
    form.cycleQty,
    form.qty
  ].join("|"),
  () => {
    couponPreview.value = null;
  }
);

watch(() => form.couponCode, () => {
  couponPreview.value = null;
});

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
  border: 2px solid var(--border);
  border-radius: 12px;
  padding: 16px;
  cursor: pointer;
  transition: all 0.2s ease;
  background: var(--card);
}

.package-card:hover:not(.disabled) {
  border-color: var(--primary);
  box-shadow: 0 4px 12px rgba(59, 130, 246, 0.15);
}

.package-card.selected {
  border-color: var(--primary);
  background: var(--primary-bg);
  box-shadow: 0 4px 12px rgba(59, 130, 246, 0.2);
}

.package-card.disabled {
  opacity: 0.5;
  cursor: not-allowed;
  background: var(--bg-tertiary);
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
  color: var(--text-primary);
}

.pkg-specs {
  display: flex;
  align-items: center;
  justify-content: space-around;
  padding: 12px 0;
  margin-bottom: 12px;
  background: var(--bg-secondary);
  border-radius: 8px;
}

.spec {
  text-align: center;
}

.spec-value {
  font-size: 20px;
  font-weight: 600;
  color: var(--primary);
  display: block;
}

.spec-label {
  font-size: 12px;
  color: var(--text-secondary);
}

.spec-divider {
  width: 1px;
  height: 24px;
  background: var(--border);
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
  color: var(--danger);
  font-weight: 500;
}

.price-amount {
  font-size: 20px;
  color: var(--danger);
  font-weight: 700;
}

.price-unit {
  font-size: 12px;
  color: var(--text-secondary);
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
  color: var(--success);
  font-size: 20px;
}

/* Section */
.section-header {
  font-size: 15px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 16px;
}

/* Addon Slider */
.addon-item {
  padding: 16px;
  background: var(--bg-secondary);
  border-radius: 8px;
  border: 1px solid var(--border);
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
  color: var(--text-primary);
}

.addon-value {
  font-size: 13px;
  color: var(--success);
  font-weight: 500;
}

.addon-value-placeholder {
  font-size: 13px;
  color: var(--text-tertiary);
}

.addon-item :deep(.ant-slider) {
  margin: 0;
}

.addon-item :deep(.ant-slider-rail) {
  background: var(--border);
}

.addon-item :deep(.ant-slider-track) {
  background: var(--primary);
}

.addon-item :deep(.ant-slider-handle) {
  border-color: var(--primary);
}

.addon-item :deep(.ant-slider-handle:hover),
.addon-item :deep(.ant-slider-handle:focus) {
  border-color: var(--primary-light);
}

.addon-item :deep(.ant-slider-mark-text) {
  font-size: 11px;
  color: var(--text-secondary);
}

.addon-item :deep(.ant-slider-mark-text-active) {
  color: var(--primary);
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
  color: var(--text-secondary);
}

.summary-val {
  color: var(--text-primary);
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
