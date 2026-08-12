<template>
  <div class="buy-page">
    <div class="buy-page__inner">
      <header class="page-heading">
        <div>
          <h1>购买 VPS</h1>
          <p>按需选择资源配置并自动计算价格</p>
        </div>
        <ElButton :icon="ShoppingCart" @click="router.push({ name: 'PublicCart' })">
          购物车
          <span v-if="cart.itemCount" class="cart-count">({{ cart.itemCount }})</span>
        </ElButton>
      </header>

      <ElSteps :active="stepIndex" finish-status="success" simple class="buy-steps">
        <ElStep title="Goods Type" />
        <ElStep title="地域" />
        <ElStep title="线路" />
        <ElStep title="套餐" />
        <ElStep title="系统" />
        <ElStep title="确认" />
      </ElSteps>

      <ElAlert
        v-if="catalogError"
        :title="catalogError"
        type="warning"
        show-icon
        :closable="false"
        class="catalog-alert"
      />

      <div v-loading="catalog.loading" class="buy-layout">
        <div class="configuration">
          <ElCard class="selection-card art-card-xs" shadow="never">
            <template #header><span class="section-title">基础选择</span></template>
            <div class="select-grid">
              <label class="field">
                <span>商品类型</span>
                <ElSelect v-model="form.goodsTypeId" placeholder="选择商品类型">
                  <ElOption
                    v-for="item in goodsTypes"
                    :key="item.id"
                    :label="item.name"
                    :value="item.id"
                  />
                </ElSelect>
              </label>
              <label class="field">
                <span>地域</span>
                <ElSelect v-model="form.regionId" placeholder="选择地域">
                  <ElOption
                    v-for="item in regions"
                    :key="item.id"
                    :label="item.name"
                    :value="item.id"
                  />
                </ElSelect>
              </label>
              <label class="field">
                <span>线路</span>
                <ElSelect v-model="form.planGroupId" placeholder="选择线路">
                  <ElOption
                    v-for="item in planGroups"
                    :key="item.id"
                    :label="item.name"
                    :value="item.id"
                    :disabled="isPlanGroupDisabled(item)"
                  >
                    <span>{{ item.name }}</span>
                    <ElTag
                      v-if="getCapacityLabel(item.capacity_remaining)"
                      size="small"
                      :type="capacityTagType(item.capacity_remaining)"
                      class="capacity-tag"
                    >
                      {{ getCapacityLabel(item.capacity_remaining) }}
                    </ElTag>
                  </ElOption>
                </ElSelect>
              </label>
            </div>
          </ElCard>

          <ElCard class="selection-card art-card-xs" shadow="never">
            <template #header><span class="section-title">选择套餐</span></template>
            <ElEmpty v-if="packages.length === 0" description="暂无可用套餐" />
            <div v-else class="package-grid">
              <button
                v-for="(item, index) in packages"
                :key="item.id"
                type="button"
                class="package-option"
                :class="{ 'package-option--selected': sameId(item.id, form.packageId) }"
                :disabled="isPackageDisabled(item)"
                @click="selectPackage(item)"
              >
                <span class="package-option__heading">
                  <strong>{{ item.name }}</strong>
                  <span class="package-option__tags">
                    <ElTag v-if="index === 0" size="small" type="danger">HOT</ElTag>
                    <ElTag
                      v-if="getCapacityLabel(item.capacity_remaining)"
                      size="small"
                      :type="capacityTagType(item.capacity_remaining)"
                    >
                      {{ getCapacityLabel(item.capacity_remaining) }}
                    </ElTag>
                  </span>
                </span>
                <span class="package-option__spec">
                  {{ item.cores || 0 }} 核 / {{ item.memory_gb || 0 }} GB /
                  {{ item.disk_gb || 0 }} GB / {{ item.bandwidth_mbps || 0 }} Mbps
                </span>
                <span v-if="item.cpu_model" class="package-option__meta">
                  CPU 型号：{{ item.cpu_model }}
                </span>
                <span v-if="item.port_num != null" class="package-option__meta">
                  端口数：{{ item.port_num }}
                </span>
                <span class="package-option__price"
                  >¥{{ Number(item.monthly_price || 0).toFixed(2) }}/月</span
                >
              </button>
            </div>
          </ElCard>

          <ElCard v-loading="systemImagesLoading" class="selection-card art-card-xs" shadow="never">
            <template #header><span class="section-title">系统镜像</span></template>
            <ElEmpty
              v-if="!systemImagesLoading && systemImages.length === 0"
              description="暂无可用系统镜像"
            />
            <ElRadioGroup v-else v-model="form.systemId" class="system-options">
              <ElRadioButton v-for="item in systemImages" :key="item.id" :value="item.id">
                {{ item.name }}
              </ElRadioButton>
            </ElRadioGroup>
          </ElCard>

          <ElCard class="selection-card art-card-xs" shadow="never">
            <template #header><span class="section-title">弹性配置</span></template>
            <div class="addon-list">
              <div v-for="addon in addonFields" :key="addon.key" class="addon-row">
                <div class="addon-row__label">
                  <span>{{ addon.label }}</span>
                  <strong>{{ addonStatus(addon) }}</strong>
                </div>
                <ElSlider
                  v-model="form[addon.key]"
                  :min="addon.rule.min"
                  :max="addon.rule.max"
                  :step="addon.rule.step"
                  :disabled="addon.rule.disabled"
                  show-stops
                />
              </div>
            </div>
          </ElCard>

          <ElCard class="selection-card art-card-xs" shadow="never">
            <template #header><span class="section-title">购买周期</span></template>
            <div class="purchase-grid">
              <label class="field">
                <span>计费周期</span>
                <ElSelect v-model="form.billingCycleId" placeholder="选择周期">
                  <ElOption
                    v-for="item in billingCycles"
                    :key="item.id"
                    :label="`${item.name}（${item.months || 0} 个月，倍率 ${item.multiplier || 0}）`"
                    :value="item.id"
                  />
                </ElSelect>
              </label>
              <label class="field">
                <span>周期数量</span>
                <ElInputNumber v-model="form.cycleQty" :min="1" :max="12" />
              </label>
              <label class="field">
                <span>实例数量</span>
                <ElInputNumber v-model="form.qty" :min="1" :max="10" />
              </label>
            </div>
          </ElCard>
        </div>

        <aside class="order-summary">
          <ElCard class="summary-card art-card-xs" shadow="never">
            <template #header><span class="section-title">配置摘要</span></template>
            <ElDescriptions :column="1" border size="small">
              <ElDescriptionsItem label="地域">{{
                selectedRegion?.name || '-'
              }}</ElDescriptionsItem>
              <ElDescriptionsItem label="线路">{{
                selectedPlanGroup?.name || '-'
              }}</ElDescriptionsItem>
              <ElDescriptionsItem label="套餐">{{
                selectedPackage?.name || '-'
              }}</ElDescriptionsItem>
              <ElDescriptionsItem label="端口">{{
                selectedPackage?.port_num ?? '-'
              }}</ElDescriptionsItem>
              <ElDescriptionsItem label="系统">{{
                selectedSystem?.name || '-'
              }}</ElDescriptionsItem>
              <ElDescriptionsItem label="周期">
                {{ selectedCycle?.name || '-' }} × {{ form.cycleQty }}
              </ElDescriptionsItem>
              <ElDescriptionsItem label="附加项">{{ addonSummary }}</ElDescriptionsItem>
            </ElDescriptions>
          </ElCard>

          <PriceCalculator
            :base-price="basePrice"
            :addon-price="addonPrice"
            :cycle-multiplier="cycleMultiplier"
            :qty="form.qty"
            :addons="previewAddons"
          />

          <ElCard class="checkout-card art-card-xs" shadow="never">
            <label class="field">
              <span>优惠码</span>
              <div class="coupon-row">
                <ElInput v-model="form.couponCode" placeholder="可选" clearable />
                <ElButton :loading="couponPreviewLoading" @click="applyCouponPreview"
                  >使用</ElButton
                >
              </div>
            </label>

            <div v-if="couponPreview" class="coupon-result">
              <span>原价 ¥{{ computedOriginalTotal.toFixed(2) }}</span>
              <span>优惠 -¥{{ Number(couponPreview.discount || 0).toFixed(2) }}</span>
              <strong>
                优惠后 ¥{{ Number(couponPreview.final_total || computedOriginalTotal).toFixed(2) }}
              </strong>
            </div>

            <div class="checkout-actions">
              <ElButton
                size="large"
                :disabled="!canCheckout"
                :loading="adding"
                @click="addToCart()"
              >
                加入购物车
              </ElButton>
              <ElButton
                type="primary"
                size="large"
                :disabled="!canCheckout"
                :loading="creating"
                @click="createOrderNow"
              >
                立即购买
              </ElButton>
            </div>
          </ElCard>
        </aside>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { ShoppingCart } from '@element-plus/icons-vue'
  import PriceCalculator from '@/components/business/price-calculator/index.vue'
  import { useAuthStore } from '@/stores/auth'
  import { useCartStore } from '@/stores/cart'
  import { useCatalogStore } from '@/stores/catalog'
  import { createOrder, listSystemImages, previewCoupon } from '@/services/user'
  import { IdempotentRequest } from '@/utils/idempotentRequest'
  import { priceBreakdownToCents } from '@/utils/pricing'
  import type {
    BillingCycle,
    CartSpec,
    CouponPreviewResponse,
    GoodsType,
    Line,
    OrderCreateRequest,
    Package,
    Region,
    SystemImage
  } from '@/services/types'

  defineOptions({ name: 'PublicBuy' })

  type AddonKey = 'add_cores' | 'add_mem_gb' | 'add_disk_gb' | 'add_bw_mbps'
  type CatalogLine = Line & { capacityRemaining?: number }
  type CatalogPackage = Package & {
    planGroupId?: number
    PlanGroupID?: number
    capacityRemaining?: number
  }
  type AddonField = {
    key: AddonKey
    label: string
    unit: string
    rule: AddonRule
    unitPrice: number
  }

  interface AddonRule {
    disabled: boolean
    min: number
    max: number
    step: number
  }

  interface BuyForm {
    goodsTypeId: number | null
    regionId: number | null
    planGroupId: number | null
    packageId: number | null
    systemId: number | null
    add_cores: number
    add_mem_gb: number
    add_disk_gb: number
    add_bw_mbps: number
    billingCycleId: number | null
    cycleQty: number
    qty: number
    couponCode: string
  }

  const router = useRouter()
  const auth = useAuthStore()
  const cart = useCartStore()
  const catalog = useCatalogStore()
  const catalogError = ref('')
  const adding = ref(false)
  const creating = ref(false)
  const couponPreviewLoading = ref(false)
  const couponPreview = ref<CouponPreviewResponse | null>(null)
  const systemImages = ref<SystemImage[]>([])
  const systemImagesLoading = ref(false)
  const orderRequest = new IdempotentRequest<OrderCreateRequest>('order')
  let systemImageRequestId = 0

  const form = reactive<BuyForm>({
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
    couponCode: ''
  })

  const sameId = (left: unknown, right: unknown) => String(left ?? '') === String(right ?? '')
  const goodsTypes = computed(() =>
    (catalog.goodsTypes as GoodsType[]).filter((item) => item.active !== false)
  )
  const regions = computed(() =>
    (catalog.regions as Region[]).filter(
      (item) => item.active !== false && sameId(item.goods_type_id, form.goodsTypeId)
    )
  )
  const planGroups = computed(() =>
    (catalog.planGroups as CatalogLine[]).filter(
      (item) =>
        item.active !== false &&
        item.visible !== false &&
        sameId(item.goods_type_id, form.goodsTypeId) &&
        sameId(item.region_id, form.regionId)
    )
  )
  const packages = computed(() =>
    (catalog.packages as CatalogPackage[]).filter((item) => {
      const groupId = item.plan_group_id ?? item.planGroupId ?? item.PlanGroupID
      return item.active !== false && item.visible !== false && sameId(groupId, form.planGroupId)
    })
  )
  const billingCycles = computed(() =>
    (catalog.billingCycles as BillingCycle[]).filter((item) => item.active !== false)
  )

  const selectedPlanGroup = computed(() =>
    planGroups.value.find((item) => sameId(item.id, form.planGroupId))
  )
  const selectedPackage = computed(() =>
    packages.value.find((item) => sameId(item.id, form.packageId))
  )
  const selectedCycle = computed(() =>
    billingCycles.value.find((item) => sameId(item.id, form.billingCycleId))
  )
  const selectedRegion = computed(() =>
    regions.value.find((item) => sameId(item.id, form.regionId))
  )
  const selectedSystem = computed(() =>
    systemImages.value.find((item) => sameId(item.id, form.systemId))
  )

  const resolveAddonRule = (
    minRaw: unknown,
    maxRaw: unknown,
    stepRaw: unknown,
    fallbackMax: number
  ): AddonRule => {
    const min = Number(minRaw ?? 0)
    const max = Number(maxRaw ?? 0)
    const step = Math.max(1, Number(stepRaw ?? 1))
    if (min === -1 || max === -1) return { disabled: true, min: 0, max: 0, step: 1 }
    const effectiveMin = min > 0 ? min : 0
    const effectiveMax = max > 0 ? max : fallbackMax
    return { disabled: false, min: effectiveMin, max: Math.max(effectiveMin, effectiveMax), step }
  }

  const clampAddonValue = (value: number, rule: AddonRule) => {
    if (rule.disabled) return 0
    const next = Math.max(rule.min, Math.min(rule.max, Number(value || 0)))
    return Math.min(rule.max, rule.min + Math.round((next - rule.min) / rule.step) * rule.step)
  }

  const addonMeta = computed(() => {
    const group = selectedPlanGroup.value ?? {}
    return {
      core: resolveAddonRule(group.add_core_min, group.add_core_max, group.add_core_step, 64),
      mem: resolveAddonRule(group.add_mem_min, group.add_mem_max, group.add_mem_step, 256),
      disk: resolveAddonRule(group.add_disk_min, group.add_disk_max, group.add_disk_step, 2000),
      bandwidth: resolveAddonRule(group.add_bw_min, group.add_bw_max, group.add_bw_step, 1000)
    }
  })

  const addonFields = computed<AddonField[]>(() => [
    {
      key: 'add_cores',
      label: '附加 CPU',
      unit: ' 核',
      rule: addonMeta.value.core,
      unitPrice: Number(selectedPlanGroup.value?.unit_core || 0)
    },
    {
      key: 'add_mem_gb',
      label: '附加内存',
      unit: ' GB',
      rule: addonMeta.value.mem,
      unitPrice: Number(selectedPlanGroup.value?.unit_mem || 0)
    },
    {
      key: 'add_disk_gb',
      label: '附加磁盘',
      unit: ' GB',
      rule: addonMeta.value.disk,
      unitPrice: Number(selectedPlanGroup.value?.unit_disk || 0)
    },
    {
      key: 'add_bw_mbps',
      label: '附加带宽',
      unit: ' Mbps',
      rule: addonMeta.value.bandwidth,
      unitPrice: Number(selectedPlanGroup.value?.unit_bw || 0)
    }
  ])
  const basePrice = computed(() => Number(selectedPackage.value?.monthly_price || 0))
  const addonPrice = computed(() => {
    const group = selectedPlanGroup.value
    if (!group) return 0
    return (
      form.add_cores * Number(group.unit_core || 0) +
      form.add_mem_gb * Number(group.unit_mem || 0) +
      form.add_disk_gb * Number(group.unit_disk || 0) +
      form.add_bw_mbps * Number(group.unit_bw || 0)
    )
  })
  const cycleMultiplier = computed(
    () =>
      Number(selectedCycle.value?.multiplier || selectedCycle.value?.months || 1) * form.cycleQty
  )
  // 附加项明细：与后端逐项整数舍入对齐（cross-module "订单价格预览"）。
  const previewAddons = computed(() =>
    addonFields.value.map((addon) => ({
      value: Number(form[addon.key] || 0),
      unitPrice: addon.unitPrice
    }))
  )
  const computedOriginalTotal = computed(() => {
    const cents = priceBreakdownToCents({
      baseMonthly: basePrice.value,
      addons: previewAddons.value,
      multiplier: cycleMultiplier.value
    })
    return (cents.totalCents * form.qty) / 100
  })
  const addonStatus = (addon: AddonField) => {
    if (addon.rule.disabled) return '已禁用'
    const value = form[addon.key]
    if (!value) return '不添加'
    return `+${value}${addon.unit} · +¥${(value * addon.unitPrice).toFixed(2)}/月`
  }
  const addonSummary = computed(() => {
    const parts = addonFields.value
      .filter((addon) => form[addon.key] > 0)
      .map((addon) => `+${form[addon.key]}${addon.unit.trim()}`)
    return parts.join(' ') || '无'
  })
  const canCheckout = computed(() => Boolean(form.packageId && form.systemId))
  const stepIndex = computed(() => {
    if (!form.goodsTypeId) return 0
    if (!form.regionId) return 1
    if (!form.planGroupId) return 2
    if (!form.packageId) return 3
    if (!form.systemId) return 4
    return 5
  })

  const getCapacityLabel = (value: unknown) => {
    const remaining = Number(value)
    if (!Number.isFinite(remaining)) return ''
    if (remaining < 0) return '不限'
    if (remaining === 0) return '售罄'
    return `余量 ${remaining}`
  }
  const capacityTagType = (value: unknown): 'danger' | 'success' | 'primary' => {
    const remaining = Number(value)
    if (remaining === 0) return 'danger'
    if (remaining < 0) return 'success'
    return 'primary'
  }
  const isPlanGroupDisabled = (item: CatalogLine) => {
    const remaining = Number(item.capacity_remaining ?? item.capacityRemaining)
    return (
      item.active === false ||
      item.visible === false ||
      (Number.isFinite(remaining) && remaining === 0)
    )
  }
  const isPackageDisabled = (item: CatalogPackage) => {
    const remaining = Number(item.capacity_remaining ?? item.capacityRemaining)
    return (
      item.active === false ||
      item.visible === false ||
      (Number.isFinite(remaining) && remaining === 0)
    )
  }
  const selectPackage = (item: CatalogPackage) => {
    if (!isPackageDisabled(item)) form.packageId = Number(item.id)
  }

  const loadSystemImages = async (planGroupId: number | null) => {
    const requestId = ++systemImageRequestId
    systemImages.value = []
    form.systemId = null
    if (!planGroupId) return

    systemImagesLoading.value = true
    try {
      const group = planGroups.value.find((item) => sameId(item.id, planGroupId))
      const response = await listSystemImages({
        plan_group_id: planGroupId,
        ...(group?.line_id ? { line_id: Number(group.line_id) } : {})
      })
      if (requestId !== systemImageRequestId) return
      systemImages.value = (response.data?.items ?? []).filter((item) => item.enabled !== false)
    } catch {
      if (requestId === systemImageRequestId) {
        systemImages.value = []
        form.systemId = null
      }
    } finally {
      if (requestId === systemImageRequestId) systemImagesLoading.value = false
    }
  }

  const buildOrderSpecPayload = (): CartSpec => ({
    add_cores: form.add_cores,
    add_mem_gb: form.add_mem_gb,
    add_disk_gb: form.add_disk_gb,
    add_bw_mbps: form.add_bw_mbps,
    billing_cycle_id: form.billingCycleId ?? undefined,
    cycle_qty: form.cycleQty,
    duration_months: Number(selectedCycle.value?.months || 1) * form.cycleQty
  })

  const cartPayload = () => ({
    package_id: form.packageId ?? undefined,
    system_id: form.systemId ?? undefined,
    spec: buildOrderSpecPayload(),
    qty: form.qty,
    amount: (basePrice.value + addonPrice.value) * cycleMultiplier.value
  })

  const addToCart = async (showMessage = true) => {
    if (!canCheckout.value) {
      ElMessage.warning('请选择套餐与系统镜像')
      return false
    }
    adding.value = true
    try {
      await cart.addItem(cartPayload())
      if (showMessage) ElMessage.success('已加入购物车')
      return true
    } finally {
      adding.value = false
    }
  }

  const applyCouponPreview = async () => {
    const code = form.couponCode.trim()
    if (!code) {
      ElMessage.warning('请先输入优惠码')
      return
    }
    if (!canCheckout.value) {
      ElMessage.warning('请先完成套餐与系统选择')
      return
    }
    if (!auth.token) {
      ElMessage.info('登录后可验证优惠码')
      return
    }
    couponPreviewLoading.value = true
    try {
      const response = await previewCoupon({
        coupon_code: code,
        items: [cartPayload()]
      })
      couponPreview.value = response.data ?? null
      ElMessage.success('优惠码已应用')
    } catch {
      couponPreview.value = null
    } finally {
      couponPreviewLoading.value = false
    }
  }

  const createOrderNow = async () => {
    if (!canCheckout.value) {
      ElMessage.warning('请选择套餐与系统镜像')
      return
    }
    if (!auth.token) {
      if (await addToCart(false)) {
        await router.push({ name: 'Login', query: { redirect: '/cart' } })
      }
      return
    }

    creating.value = true
    const request = orderRequest.prepare(() => ({
      coupon_code: form.couponCode.trim() || undefined,
      items: [cartPayload()]
    }))
    try {
      const response = await createOrder(request.payload, request.key)
      orderRequest.resolve()
      const payload = response.data as Record<string, any>
      const orderId = payload.order?.id ?? payload.order?.ID ?? payload.id ?? payload.ID
      ElMessage.success('订单已创建')
      if (orderId) await router.push({ name: 'ConsoleOrderDetail', params: { id: orderId } })
    } catch (error) {
      orderRequest.reject(error)
    } finally {
      creating.value = false
    }
  }

  watch(
    addonMeta,
    (meta) => {
      form.add_cores = clampAddonValue(form.add_cores, meta.core)
      form.add_mem_gb = clampAddonValue(form.add_mem_gb, meta.mem)
      form.add_disk_gb = clampAddonValue(form.add_disk_gb, meta.disk)
      form.add_bw_mbps = clampAddonValue(form.add_bw_mbps, meta.bandwidth)
    },
    { immediate: true, deep: true }
  )
  watch(
    () => form.goodsTypeId,
    () => {
      form.regionId = null
      form.planGroupId = null
      form.packageId = null
      form.systemId = null
    }
  )
  watch(
    () => form.regionId,
    () => {
      form.planGroupId = null
      form.packageId = null
      form.systemId = null
    }
  )
  watch(
    () => form.planGroupId,
    (planGroupId) => {
      form.packageId = null
      void loadSystemImages(planGroupId)
    }
  )
  watch(
    () => form.packageId,
    () => {
      if (!systemImages.value.some((item) => sameId(item.id, form.systemId))) {
        form.systemId = Number(systemImages.value[0]?.id) || null
      }
    }
  )
  watch(
    goodsTypes,
    (items) => {
      if (!items.some((item) => sameId(item.id, form.goodsTypeId)))
        form.goodsTypeId = Number(items[0]?.id) || null
    },
    { immediate: true }
  )
  watch(
    regions,
    (items) => {
      if (!items.some((item) => sameId(item.id, form.regionId)))
        form.regionId = Number(items[0]?.id) || null
    },
    { immediate: true }
  )
  watch(
    planGroups,
    (items) => {
      if (!items.some((item) => sameId(item.id, form.planGroupId))) {
        form.planGroupId = Number(items.find((item) => !isPlanGroupDisabled(item))?.id) || null
      }
    },
    { immediate: true }
  )
  watch(
    packages,
    (items) => {
      if (!items.some((item) => sameId(item.id, form.packageId))) {
        form.packageId = Number(items.find((item) => !isPackageDisabled(item))?.id) || null
      }
    },
    { immediate: true }
  )
  watch(
    systemImages,
    (items) => {
      if (!items.some((item) => sameId(item.id, form.systemId)))
        form.systemId = Number(items[0]?.id) || null
    },
    { immediate: true }
  )
  watch(
    billingCycles,
    (items) => {
      if (!items.some((item) => sameId(item.id, form.billingCycleId))) {
        form.billingCycleId = Number(items[0]?.id) || null
      }
    },
    { immediate: true }
  )
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
      form.qty,
      form.couponCode
    ],
    () => {
      couponPreview.value = null
    }
  )

  onMounted(async () => {
    try {
      await catalog.fetchCatalog()
    } catch {
      catalogError.value = '商品目录暂时无法加载，请稍后重试'
    }
  })
</script>

<style lang="scss" scoped>
  .buy-page {
    padding: 28px 20px 48px;
    background: var(--default-bg-color);

    &__inner {
      max-width: 1240px;
      margin: 0 auto;
    }
  }

  .page-heading {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 20px;

    h1 {
      margin: 0;
      color: var(--art-gray-900);
      font-size: 28px;
      letter-spacing: 0;
    }

    p {
      margin: 8px 0 0;
      color: var(--art-gray-600);
    }
  }

  .cart-count {
    margin-left: 4px;
  }

  .buy-steps,
  .catalog-alert {
    margin-bottom: 18px;
  }

  .buy-layout {
    display: grid;
    grid-template-columns: minmax(0, 1fr) minmax(280px, 340px);
    gap: 18px;
    align-items: start;
  }

  .configuration,
  .order-summary {
    display: grid;
    gap: 14px;
  }

  .order-summary {
    position: sticky;
    top: 82px;
  }

  .section-title {
    color: var(--art-gray-900);
    font-weight: 600;
  }

  .select-grid,
  .purchase-grid {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 14px;
  }

  .field {
    display: grid;
    gap: 7px;
    color: var(--art-gray-700);
    font-size: 13px;
  }

  .capacity-tag {
    float: right;
  }

  .package-grid {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 12px;
  }

  .package-option {
    display: grid;
    gap: 10px;
    padding: 16px;
    color: var(--art-gray-700);
    text-align: left;
    background: var(--default-box-color);
    border: 1px solid var(--default-border);
    border-radius: calc(var(--custom-radius) / 2 + 2px);
    cursor: pointer;
    transition:
      border-color 0.2s,
      background 0.2s;

    &:hover {
      background: var(--art-hover-color);
      border-color: var(--theme-color);
    }

    &:disabled {
      cursor: not-allowed;
      opacity: 0.55;
    }

    &--selected {
      background: var(--el-color-primary-light-9);
      border-color: var(--theme-color);
    }

    &__heading {
      display: flex;
      justify-content: space-between;
      gap: 8px;
      color: var(--art-gray-900);
    }

    &__tags {
      display: flex;
      flex-wrap: wrap;
      gap: 5px;
      justify-content: flex-end;
    }

    &__spec {
      font-size: 13px;
      line-height: 1.6;
    }

    &__meta {
      color: var(--art-gray-600);
      font-size: 12px;
    }

    &__price {
      color: var(--theme-color);
      font-size: 17px;
      font-weight: 700;
    }
  }

  .system-options {
    display: flex;
    flex-wrap: wrap;
  }

  .addon-list {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 18px 28px;
  }

  .addon-row {
    min-width: 0;

    &__label {
      display: flex;
      justify-content: space-between;
      margin-bottom: 4px;
      color: var(--art-gray-700);

      strong {
        color: var(--art-gray-900);
      }
    }
  }

  .coupon-row {
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto;
    gap: 8px;
  }

  .coupon-result {
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto;
    gap: 6px 12px;
    margin-top: 14px;
    padding: 10px 12px;
    color: var(--art-gray-700);
    background: var(--el-color-success-light-9);
    border-radius: calc(var(--custom-radius) / 3 + 2px);

    strong {
      grid-column: 1 / -1;
      color: var(--art-gray-900);
      text-align: right;
    }
  }

  .checkout-actions {
    display: grid;
    gap: 10px;
    margin-top: 18px;
  }

  @media (width <= 960px) {
    .buy-layout {
      grid-template-columns: 1fr;
    }

    .order-summary {
      position: static;
    }
  }

  @media (width <= 720px) {
    .buy-page {
      padding: 20px 12px 36px;
    }

    .page-heading {
      align-items: stretch;
      flex-direction: column;
      gap: 14px;

      h1 {
        font-size: 24px;
      }
    }

    .select-grid,
    .purchase-grid,
    .package-grid,
    .addon-list {
      grid-template-columns: 1fr;
    }

    .buy-steps {
      overflow-x: auto;
    }
  }
</style>
