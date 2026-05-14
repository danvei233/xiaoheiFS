<template>
  <div class="coupon-page art-full-height">
    <ElCard shadow="never">
      <div class="page-header">
        <div>
          <div class="page-title">Coupons</div>
          <div class="page-subtitle">Manage coupon groups, coupon codes, and batch generation.</div>
        </div>

        <ElButton :loading="loading" @click="fetchAll">Refresh</ElButton>
      </div>

      <ElEmpty
        v-if="!canView"
        description="You do not have permission to view coupon management."
      />

      <ElTabs v-else v-model="activeTab">
        <ElTabPane label="Product Groups" name="groups">
          <div class="toolbar">
            <div />

            <div class="toolbar-actions">
              <ElButton v-if="canManageGroups" type="primary" @click="openGroupDialog()">
                New Group
              </ElButton>
            </div>
          </div>

          <ElTable v-loading="loading" :data="groups" border row-key="id">
            <ElTableColumn prop="id" label="ID" width="90" />
            <ElTableColumn prop="name" label="Name" min-width="220" />
            <ElTableColumn label="Rules" width="100">
              <template #default="{ row }">
                {{ row.rules.length }}
              </template>
            </ElTableColumn>
            <ElTableColumn label="Rule Preview" min-width="280" show-overflow-tooltip>
              <template #default="{ row }">
                {{ renderRulePreview(row.rules) }}
              </template>
            </ElTableColumn>
            <ElTableColumn label="Actions" width="170" fixed="right">
              <template #default="{ row }">
                <div class="table-actions">
                  <ElButton
                    v-if="canManageGroups"
                    link
                    type="primary"
                    @click="openGroupDialog(row)"
                  >
                    Edit
                  </ElButton>
                  <ElButton v-if="canManageGroups" link type="danger" @click="removeGroup(row)">
                    Delete
                  </ElButton>
                </div>
              </template>
            </ElTableColumn>
          </ElTable>
        </ElTabPane>

        <ElTabPane label="Coupons" name="coupons">
          <div class="toolbar">
            <div class="toolbar-filters">
              <ElInput
                v-model.trim="couponKeyword"
                clearable
                placeholder="Search coupon code"
                class="inline-input"
              />

              <ElSelect
                v-model="couponGroupFilter"
                clearable
                filterable
                placeholder="Filter by group"
                class="inline-select"
              >
                <ElOption
                  v-for="item in groupOptions"
                  :key="item.value"
                  :label="item.label"
                  :value="item.value"
                />
              </ElSelect>

              <ElSelect
                v-model="couponActiveFilter"
                clearable
                placeholder="Status"
                class="inline-select"
              >
                <ElOption label="Active" value="active" />
                <ElOption label="Inactive" value="inactive" />
              </ElSelect>
            </div>

            <div class="toolbar-actions">
              <ElButton v-if="canManageCoupons" type="primary" @click="openCouponDialog()">
                New Coupon
              </ElButton>
              <ElButton v-if="canBatchGenerate" @click="openBatchDialog">Batch Generate</ElButton>
            </div>
          </div>

          <ElTable v-loading="loading" :data="filteredCoupons" border row-key="id">
            <ElTableColumn prop="id" label="ID" width="90" />
            <ElTableColumn prop="code" label="Coupon Code" min-width="180" />
            <ElTableColumn label="Discount" width="120">
              <template #default="{ row }">
                {{ formatDiscount(row.discount_permille) }}
              </template>
            </ElTableColumn>
            <ElTableColumn label="Group" min-width="200" show-overflow-tooltip>
              <template #default="{ row }">
                {{ resolveGroupName(row.product_group_id) }}
              </template>
            </ElTableColumn>
            <ElTableColumn label="Policy" min-width="180">
              <template #default="{ row }">
                {{ formatPolicy(row) }}
              </template>
            </ElTableColumn>
            <ElTableColumn label="New Users" width="100">
              <template #default="{ row }">
                <ElTag :type="row.new_user_only ? 'warning' : 'info'">
                  {{ row.new_user_only ? 'Yes' : 'No' }}
                </ElTag>
              </template>
            </ElTableColumn>
            <ElTableColumn label="Status" width="100">
              <template #default="{ row }">
                <ElTag :type="row.active ? 'success' : 'info'">
                  {{ row.active ? 'Active' : 'Inactive' }}
                </ElTag>
              </template>
            </ElTableColumn>
            <ElTableColumn prop="created_at" label="Created At" min-width="180">
              <template #default="{ row }">
                {{ formatDateTime(row.created_at) }}
              </template>
            </ElTableColumn>
            <ElTableColumn label="Actions" width="170" fixed="right">
              <template #default="{ row }">
                <div class="table-actions">
                  <ElButton
                    v-if="canManageCoupons"
                    link
                    type="primary"
                    @click="openCouponDialog(row)"
                  >
                    Edit
                  </ElButton>
                  <ElButton v-if="canManageCoupons" link type="danger" @click="removeCoupon(row)">
                    Delete
                  </ElButton>
                </div>
              </template>
            </ElTableColumn>
          </ElTable>
        </ElTabPane>
      </ElTabs>
    </ElCard>

    <CouponGroupDialog
      v-model:visible="groupDialogVisible"
      :form-data="groupForm"
      :goods-types="goodsTypes"
      :regions="regions"
      :plan-groups="planGroups"
      :packages="packages"
      :submitting="groupSubmitting"
      @submit="submitGroup"
    />

    <CouponDialog
      v-model:visible="couponDialogVisible"
      :form-data="couponForm"
      :group-options="groupOptions"
      :submitting="couponSubmitting"
      @submit="submitCoupon"
    />

    <CouponBatchDialog
      v-model:visible="batchDialogVisible"
      :form-data="batchForm"
      :group-options="groupOptions"
      :submitting="batchSubmitting"
      @submit="submitBatch"
    />
  </div>
</template>

<script setup lang="ts">
  import type {
    CatalogGoodsType,
    CatalogPackage,
    CatalogPlanGroup,
    CatalogRegion,
    CouponProductGroupRecord,
    CouponRecord
  } from '@/api/admin'
  import {
    batchGenerateAdminCoupons,
    createAdminCoupon,
    createAdminCouponGroup,
    deleteAdminCoupon,
    deleteAdminCouponGroup,
    fetchAdminCouponGroups,
    fetchAdminCoupons,
    fetchAdminGoodsTypes,
    fetchAdminPackages,
    fetchAdminPlanGroups,
    fetchAdminRegions,
    hasAdminPermission,
    updateAdminCoupon,
    updateAdminCouponGroup
  } from '@/api/admin'
  import { useUserStore } from '@/store/modules/user'
  import { INPUT_LIMITS } from '@/utils/constants'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import CouponBatchDialog from './modules/coupon-batch-dialog.vue'
  import CouponDialog from './modules/coupon-dialog.vue'
  import CouponGroupDialog from './modules/coupon-group-dialog.vue'

  defineOptions({ name: 'MarketingCoupons' })

  interface GoodsTypeOptionRow {
    id: number | null
    name: string
  }

  interface RegionOptionRow {
    id: number | null
    name: string
    goods_type_id: number | null
  }

  interface PlanGroupOptionRow {
    id: number | null
    name: string
    goods_type_id: number | null
    region_id: number | null
  }

  interface PackageOptionRow {
    id: number | null
    name: string
    plan_group_id: number | null
  }

  interface CouponRuleFormValue {
    scope: string
    goods_type_id: number | null
    region_id: number | null
    plan_group_id: number | null
    package_id: number | null
    addon_core_enabled: boolean
    addon_mem_enabled: boolean
    addon_disk_enabled: boolean
    addon_bw_enabled: boolean
  }

  interface CouponGroupRow {
    id: number | null
    name: string
    rules: CouponRuleFormValue[]
  }

  interface CouponRow {
    id: number | null
    code: string
    discount_permille: number
    product_group_id: number | null
    total_limit: number
    per_user_limit: number
    new_user_only: boolean
    active: boolean
    note: string
    created_at: string
    updated_at: string
  }

  interface CouponGroupFormValue {
    id: number | null
    name: string
    rules: CouponRuleFormValue[]
  }

  interface CouponFormValue {
    id: number | null
    code: string
    discount_permille: number
    product_group_id: number | null
    total_limit: number
    per_user_limit: number
    new_user_only: boolean
    active: boolean
    note: string
  }

  interface CouponBatchFormValue {
    prefix: string
    count: number
    length: number
    discount_permille: number
    product_group_id: number | null
    total_limit: number
    per_user_limit: number
    new_user_only: boolean
    active: boolean
    note: string
  }

  interface SelectOption {
    label: string
    value: number
  }

  const scopeOptions = [
    { label: 'All Products', value: 'all' },
    { label: 'All Addons', value: 'all_addons' },
    { label: 'Goods Type', value: 'goods_type' },
    { label: 'Goods Type + Region', value: 'goods_type_region' },
    { label: 'Plan Group', value: 'plan_group' },
    { label: 'Package', value: 'package' },
    { label: 'Addon Config', value: 'addon_config' }
  ]

  const userStore = useUserStore()
  const { info } = storeToRefs(userStore)

  const activeTab = ref('groups')
  const loading = ref(false)
  const groupSubmitting = ref(false)
  const couponSubmitting = ref(false)
  const batchSubmitting = ref(false)

  const groupDialogVisible = ref(false)
  const couponDialogVisible = ref(false)
  const batchDialogVisible = ref(false)

  const groups = ref<CouponGroupRow[]>([])
  const coupons = ref<CouponRow[]>([])
  const goodsTypes = ref<GoodsTypeOptionRow[]>([])
  const regions = ref<RegionOptionRow[]>([])
  const planGroups = ref<PlanGroupOptionRow[]>([])
  const packages = ref<PackageOptionRow[]>([])

  const couponKeyword = ref('')
  const couponGroupFilter = ref<number | null>(null)
  const couponActiveFilter = ref<'active' | 'inactive' | null>(null)

  const groupForm = ref<CouponGroupFormValue>(createDefaultGroupForm())
  const couponForm = ref<CouponFormValue>(createDefaultCouponForm())
  const batchForm = ref<CouponBatchFormValue>(createDefaultBatchForm())

  const canView = computed(() =>
    hasAdminPermission(info.value?.buttons, ['coupon_group.list', 'coupon.list'])
  )
  const canManageGroups = computed(() =>
    hasAdminPermission(info.value?.buttons, [
      'coupon_group.create',
      'coupon_group.update',
      'coupon_group.delete'
    ])
  )
  const canManageCoupons = computed(() =>
    hasAdminPermission(info.value?.buttons, ['coupon.create', 'coupon.update', 'coupon.delete'])
  )
  const canBatchGenerate = computed(() =>
    hasAdminPermission(info.value?.buttons, ['coupon.batch_generate'])
  )

  const groupOptions = computed<SelectOption[]>(() =>
    groups.value
      .filter((item) => item.id !== null)
      .map((item) => ({
        label: `${item.name || 'Group'} (#${item.id})`,
        value: Number(item.id)
      }))
  )

  const filteredCoupons = computed(() => {
    const keyword = String(couponKeyword.value || '')
      .trim()
      .toLowerCase()

    return coupons.value.filter((item) => {
      if (
        couponGroupFilter.value &&
        Number(item.product_group_id) !== Number(couponGroupFilter.value)
      ) {
        return false
      }

      if (couponActiveFilter.value === 'active' && !item.active) {
        return false
      }

      if (couponActiveFilter.value === 'inactive' && item.active) {
        return false
      }

      if (
        keyword &&
        !String(item.code || '')
          .toLowerCase()
          .includes(keyword)
      ) {
        return false
      }

      return true
    })
  })

  onMounted(() => {
    fetchAll()
  })

  function createEmptyRule(): CouponRuleFormValue {
    return {
      scope: 'all',
      goods_type_id: null,
      region_id: null,
      plan_group_id: null,
      package_id: null,
      addon_core_enabled: false,
      addon_mem_enabled: false,
      addon_disk_enabled: false,
      addon_bw_enabled: false
    }
  }

  function createDefaultGroupForm(): CouponGroupFormValue {
    return {
      id: null,
      name: '',
      rules: [createEmptyRule()]
    }
  }

  function createDefaultCouponForm(): CouponFormValue {
    return {
      id: null,
      code: '',
      discount_permille: 900,
      product_group_id: null,
      total_limit: -1,
      per_user_limit: -1,
      new_user_only: false,
      active: true,
      note: ''
    }
  }

  function createDefaultBatchForm(): CouponBatchFormValue {
    return {
      prefix: 'CP',
      count: 20,
      length: 8,
      discount_permille: 900,
      product_group_id: null,
      total_limit: -1,
      per_user_limit: -1,
      new_user_only: false,
      active: true,
      note: ''
    }
  }

  function toNullableNumber(value: unknown) {
    if (value === null || value === undefined || value === '' || Number(value) <= 0) {
      return null
    }

    const parsed = Number(value)
    return Number.isFinite(parsed) ? parsed : null
  }

  function toNumber(value: unknown, fallback = 0) {
    const parsed = Number(value)
    return Number.isFinite(parsed) ? parsed : fallback
  }

  function toBoolean(value: unknown, fallback = false) {
    if (typeof value === 'boolean') {
      return value
    }

    if (value === 'true') {
      return true
    }

    if (value === 'false') {
      return false
    }

    return fallback
  }

  function normalizeGoodsType(
    item: Partial<CatalogGoodsType> & Record<string, unknown>
  ): GoodsTypeOptionRow {
    return {
      id: toNullableNumber(item.id ?? item.ID),
      name: String(item.name ?? item.Name ?? '')
    }
  }

  function normalizeRegion(
    item: Partial<CatalogRegion> & Record<string, unknown>
  ): RegionOptionRow {
    return {
      id: toNullableNumber(item.id ?? item.ID),
      name: String(item.name ?? item.Name ?? ''),
      goods_type_id: toNullableNumber(item.goods_type_id ?? item.GoodsTypeID)
    }
  }

  function normalizePlanGroup(
    item: Partial<CatalogPlanGroup> & Record<string, unknown>
  ): PlanGroupOptionRow {
    return {
      id: toNullableNumber(item.id ?? item.ID),
      name: String(item.name ?? item.Name ?? ''),
      goods_type_id: toNullableNumber(item.goods_type_id ?? item.GoodsTypeID),
      region_id: toNullableNumber(item.region_id ?? item.RegionID)
    }
  }

  function normalizePackage(
    item: Partial<CatalogPackage> & Record<string, unknown>
  ): PackageOptionRow {
    return {
      id: toNullableNumber(item.id ?? item.ID),
      name: String(item.name ?? item.Name ?? ''),
      plan_group_id: toNullableNumber(item.plan_group_id ?? item.PlanGroupID)
    }
  }

  function normalizeRule(rule?: unknown): CouponRuleFormValue {
    const raw = (rule || {}) as Record<string, unknown>
    const scope = String(raw.scope ?? 'all')

    const normalized: CouponRuleFormValue = {
      scope,
      goods_type_id: toNullableNumber(raw.goods_type_id ?? raw.GoodsTypeID),
      region_id: toNullableNumber(raw.region_id ?? raw.RegionID),
      plan_group_id: toNullableNumber(raw.plan_group_id ?? raw.PlanGroupID),
      package_id: toNullableNumber(raw.package_id ?? raw.PackageID),
      addon_core_enabled: toBoolean(raw.addon_core_enabled ?? raw.AddonCoreEnabled, false),
      addon_mem_enabled: toBoolean(raw.addon_mem_enabled ?? raw.AddonMemEnabled, false),
      addon_disk_enabled: toBoolean(raw.addon_disk_enabled ?? raw.AddonDiskEnabled, false),
      addon_bw_enabled: toBoolean(raw.addon_bw_enabled ?? raw.AddonBWEnabled, false)
    }

    if (!needGoodsType(normalized.scope)) normalized.goods_type_id = null
    if (!needRegion(normalized.scope)) normalized.region_id = null
    if (!needPlanGroup(normalized.scope)) normalized.plan_group_id = null
    if (!needPackage(normalized.scope)) normalized.package_id = null

    if (normalized.scope !== 'addon_config') {
      normalized.addon_core_enabled = false
      normalized.addon_mem_enabled = false
      normalized.addon_disk_enabled = false
      normalized.addon_bw_enabled = false
    }

    return normalized
  }

  function normalizeRules(rules?: unknown) {
    const items = Array.isArray(rules) ? rules.map((item) => normalizeRule(item)) : []
    return items.length ? items : [createEmptyRule()]
  }

  function normalizeGroup(
    item: Partial<CouponProductGroupRecord> & Record<string, unknown>
  ): CouponGroupRow {
    return {
      id: toNullableNumber(item.id ?? item.ID),
      name: String(item.name ?? item.Name ?? ''),
      rules: normalizeRules(item.rules ?? item.Rules)
    }
  }

  function normalizeCoupon(item: Partial<CouponRecord> & Record<string, unknown>): CouponRow {
    return {
      id: toNullableNumber(item.id ?? item.ID),
      code: String(item.code ?? item.Code ?? ''),
      discount_permille: toNumber(item.discount_permille ?? item.DiscountPermille, 900),
      product_group_id: toNullableNumber(item.product_group_id ?? item.ProductGroupID),
      total_limit: toNumber(item.total_limit ?? item.TotalLimit, -1),
      per_user_limit: toNumber(item.per_user_limit ?? item.PerUserLimit, -1),
      new_user_only: toBoolean(item.new_user_only ?? item.NewUserOnly, false),
      active: toBoolean(item.active ?? item.Active, true),
      note: String(item.note ?? item.Note ?? ''),
      created_at: String(item.created_at ?? item.CreatedAt ?? ''),
      updated_at: String(item.updated_at ?? item.UpdatedAt ?? '')
    }
  }

  function needGoodsType(scope?: string) {
    return ['goods_type', 'goods_type_region', 'plan_group', 'package', 'addon_config'].includes(
      String(scope || '')
    )
  }

  function needRegion(scope?: string) {
    return ['goods_type_region', 'plan_group'].includes(String(scope || ''))
  }

  function needPlanGroup(scope?: string) {
    return ['plan_group', 'package', 'addon_config'].includes(String(scope || ''))
  }

  function needPackage(scope?: string) {
    return String(scope || '') === 'package'
  }

  function validateRule(rule: CouponRuleFormValue) {
    if (needGoodsType(rule.scope) && !rule.goods_type_id) {
      return 'Please select a goods type'
    }

    if (needRegion(rule.scope) && !rule.region_id) {
      return 'Please select a region'
    }

    if (needPlanGroup(rule.scope) && !rule.plan_group_id) {
      return 'Please select a plan group'
    }

    if (needPackage(rule.scope) && !rule.package_id) {
      return 'Please select a package'
    }

    return ''
  }

  function serializeRule(rule: CouponRuleFormValue) {
    const normalized = normalizeRule(rule)
    return {
      scope: normalized.scope,
      goods_type_id: normalized.goods_type_id || undefined,
      region_id: normalized.region_id || undefined,
      plan_group_id: normalized.plan_group_id || undefined,
      package_id: normalized.package_id || undefined,
      addon_core_enabled:
        normalized.scope === 'addon_config' ? normalized.addon_core_enabled : false,
      addon_mem_enabled: normalized.scope === 'addon_config' ? normalized.addon_mem_enabled : false,
      addon_disk_enabled:
        normalized.scope === 'addon_config' ? normalized.addon_disk_enabled : false,
      addon_bw_enabled: normalized.scope === 'addon_config' ? normalized.addon_bw_enabled : false
    }
  }

  async function fetchAll() {
    loading.value = true

    try {
      const [
        groupPayload,
        couponPayload,
        goodsTypePayload,
        regionPayload,
        planGroupPayload,
        packagePayload
      ] = await Promise.all([
        fetchAdminCouponGroups(),
        fetchAdminCoupons({ limit: 200, offset: 0 }),
        fetchAdminGoodsTypes(),
        fetchAdminRegions({ limit: 1000, offset: 0 }),
        fetchAdminPlanGroups({ limit: 2000, offset: 0 }),
        fetchAdminPackages({ limit: 3000, offset: 0 })
      ])

      groups.value = (groupPayload.items || []).map((item) =>
        normalizeGroup(item as Record<string, unknown>)
      )
      coupons.value = (couponPayload.items || []).map((item) =>
        normalizeCoupon(item as Record<string, unknown>)
      )
      goodsTypes.value = (goodsTypePayload.items || []).map((item) =>
        normalizeGoodsType(item as Record<string, unknown>)
      )
      regions.value = (regionPayload.items || []).map((item) =>
        normalizeRegion(item as Record<string, unknown>)
      )
      planGroups.value = (planGroupPayload.items || []).map((item) =>
        normalizePlanGroup(item as Record<string, unknown>)
      )
      packages.value = (packagePayload.items || []).map((item) =>
        normalizePackage(item as Record<string, unknown>)
      )
    } catch (error: any) {
      ElMessage.error(error?.response?.data?.error || 'Failed to load coupons')
    } finally {
      loading.value = false
    }
  }

  function scopeLabel(scope?: string) {
    return scopeOptions.find((item) => item.value === scope)?.label || scope || '-'
  }

  function renderRulePreview(rules: CouponRuleFormValue[]) {
    if (!rules.length) {
      return '-'
    }

    const first = rules[0]
    const text = scopeLabel(first.scope)
    return rules.length === 1 ? text : `${text} +${rules.length - 1}`
  }

  function resolveGroupName(groupId?: number | null) {
    const matched = groups.value.find((item) => Number(item.id) === Number(groupId))
    return matched?.name || (groupId ? `#${groupId}` : '-')
  }

  function formatDiscount(discountPermille?: number) {
    return `${(Number(discountPermille || 0) / 10).toFixed(1)}%`
  }

  function formatPolicy(row: CouponRow) {
    const totalText = row.total_limit < 0 ? 'unlimited' : String(row.total_limit)
    const perUserText = row.per_user_limit < 0 ? 'unlimited' : String(row.per_user_limit)
    return `total ${totalText} / user ${perUserText}`
  }

  function formatDateTime(value?: string) {
    if (!value) {
      return '-'
    }

    const date = new Date(value)
    return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN')
  }

  function openGroupDialog(row?: CouponGroupRow) {
    groupForm.value = row
      ? {
          id: row.id,
          name: row.name,
          rules: normalizeRules(row.rules)
        }
      : createDefaultGroupForm()

    groupDialogVisible.value = true
  }

  async function submitGroup(form: CouponGroupFormValue) {
    const name = String(form.name || '').trim()
    if (!name) {
      return ElMessage.error('Please enter a coupon group name')
    }

    if (name.length > INPUT_LIMITS.COUPON_GROUP_NAME) {
      return ElMessage.error(
        `Coupon group name cannot exceed ${INPUT_LIMITS.COUPON_GROUP_NAME} characters`
      )
    }

    const rules = normalizeRules(form.rules)
    for (const rule of rules) {
      const errorText = validateRule(rule)
      if (errorText) {
        return ElMessage.error(errorText)
      }
    }

    groupSubmitting.value = true

    try {
      const payload = {
        name,
        rules: rules.map((item) => serializeRule(item))
      }

      if (form.id) {
        await updateAdminCouponGroup(form.id, payload)
      } else {
        await createAdminCouponGroup(payload)
      }

      groupDialogVisible.value = false
      ElMessage.success('Coupon group saved')
      await fetchAll()
    } catch (error: any) {
      ElMessage.error(error?.response?.data?.error || 'Failed to save coupon group')
    } finally {
      groupSubmitting.value = false
    }
  }

  async function removeGroup(row: CouponGroupRow) {
    if (!row.id) {
      return
    }

    try {
      await ElMessageBox.confirm(`Delete coupon group "${row.name || row.id}"?`, 'Confirm', {
        type: 'warning'
      })
      await deleteAdminCouponGroup(row.id)
      ElMessage.success('Coupon group deleted')
      await fetchAll()
    } catch (error: any) {
      if (error !== 'cancel' && error !== 'close') {
        ElMessage.error(error?.response?.data?.error || 'Failed to delete coupon group')
      }
    }
  }

  function openCouponDialog(row?: CouponRow) {
    couponForm.value = row
      ? { ...row }
      : {
          ...createDefaultCouponForm(),
          product_group_id: couponGroupFilter.value
        }

    couponDialogVisible.value = true
  }

  async function submitCoupon(form: CouponFormValue) {
    const code = String(form.code || '').trim()
    if (!code) {
      return ElMessage.error('Please enter a coupon code')
    }

    if (code.length > INPUT_LIMITS.COUPON_CODE) {
      return ElMessage.error(`Coupon code cannot exceed ${INPUT_LIMITS.COUPON_CODE} characters`)
    }

    if (!form.product_group_id) {
      return ElMessage.error('Please select a coupon group')
    }

    if (String(form.note || '').length > INPUT_LIMITS.COUPON_NOTE) {
      return ElMessage.error(`Note cannot exceed ${INPUT_LIMITS.COUPON_NOTE} characters`)
    }

    couponSubmitting.value = true

    try {
      const payload = {
        code,
        discount_permille: toNumber(form.discount_permille, 900),
        product_group_id: Number(form.product_group_id),
        total_limit: toNumber(form.total_limit, -1),
        per_user_limit: toNumber(form.per_user_limit, -1),
        new_user_only: Boolean(form.new_user_only),
        active: Boolean(form.active),
        note: String(form.note || '').trim()
      }

      if (form.id) {
        await updateAdminCoupon(form.id, payload)
      } else {
        await createAdminCoupon(payload)
      }

      couponDialogVisible.value = false
      ElMessage.success('Coupon saved')
      await fetchAll()
    } catch (error: any) {
      ElMessage.error(error?.response?.data?.error || 'Failed to save coupon')
    } finally {
      couponSubmitting.value = false
    }
  }

  async function removeCoupon(row: CouponRow) {
    if (!row.id) {
      return
    }

    try {
      await ElMessageBox.confirm(`Delete coupon "${row.code || row.id}"?`, 'Confirm', {
        type: 'warning'
      })
      await deleteAdminCoupon(row.id)
      ElMessage.success('Coupon deleted')
      await fetchAll()
    } catch (error: any) {
      if (error !== 'cancel' && error !== 'close') {
        ElMessage.error(error?.response?.data?.error || 'Failed to delete coupon')
      }
    }
  }

  function openBatchDialog() {
    batchForm.value = {
      ...batchForm.value,
      product_group_id: couponGroupFilter.value || batchForm.value.product_group_id
    }
    batchDialogVisible.value = true
  }

  async function submitBatch(form: CouponBatchFormValue) {
    const prefix = String(form.prefix || '').trim()
    if (!prefix) {
      return ElMessage.error('Please enter a prefix')
    }

    if (prefix.length > INPUT_LIMITS.COUPON_BATCH_PREFIX) {
      return ElMessage.error(`Prefix cannot exceed ${INPUT_LIMITS.COUPON_BATCH_PREFIX} characters`)
    }

    if (!form.product_group_id) {
      return ElMessage.error('Please select a coupon group')
    }

    if (String(form.note || '').length > INPUT_LIMITS.COUPON_NOTE) {
      return ElMessage.error(`Note cannot exceed ${INPUT_LIMITS.COUPON_NOTE} characters`)
    }

    batchSubmitting.value = true

    try {
      await batchGenerateAdminCoupons({
        prefix,
        count: toNumber(form.count, 20),
        length: toNumber(form.length, 8),
        discount_permille: toNumber(form.discount_permille, 900),
        product_group_id: Number(form.product_group_id),
        total_limit: toNumber(form.total_limit, -1),
        per_user_limit: toNumber(form.per_user_limit, -1),
        new_user_only: Boolean(form.new_user_only),
        active: Boolean(form.active),
        note: String(form.note || '').trim()
      })

      batchForm.value = { ...form }
      batchDialogVisible.value = false
      ElMessage.success('Coupons generated')
      await fetchAll()
    } catch (error: any) {
      ElMessage.error(error?.response?.data?.error || 'Failed to generate coupons')
    } finally {
      batchSubmitting.value = false
    }
  }
</script>

<style scoped lang="scss">
  .coupon-page {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .page-header,
  .toolbar,
  .toolbar-actions,
  .table-actions {
    display: flex;
    align-items: center;
  }

  .page-header,
  .toolbar {
    justify-content: space-between;
    gap: 16px;
  }

  .page-header {
    margin-bottom: 16px;
  }

  .page-title {
    color: var(--el-text-color-primary);
    font-size: 28px;
    font-weight: 700;
    line-height: 1.1;
  }

  .page-subtitle {
    margin-top: 8px;
    color: var(--el-text-color-secondary);
    font-size: 14px;
    line-height: 1.6;
  }

  .toolbar {
    margin-bottom: 14px;
  }

  .toolbar-filters,
  .toolbar-actions,
  .table-actions {
    gap: 12px;
    flex-wrap: wrap;
  }

  .inline-input,
  .inline-select {
    width: 220px;
  }

  @media (max-width: 768px) {
    .page-header,
    .toolbar {
      flex-direction: column;
      align-items: stretch;
    }

    .inline-input,
    .inline-select {
      width: 100%;
    }
  }
</style>
