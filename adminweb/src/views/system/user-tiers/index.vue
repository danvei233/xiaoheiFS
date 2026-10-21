<template>
  <div class="art-full-height user-tier-page">
    <ElCard v-loading="loading.groups" class="art-table-card">
      <template #header>
        <div class="page-header">
          <div>
            <div class="page-title">用户等级</div>
            <div class="page-subtitle">管理用户组、优惠策略与自动审批条件。</div>
          </div>

          <div class="page-actions">
            <ElButton v-if="canUpdate" @click="handleRebuildAll">重建全部缓存</ElButton>
            <ElButton v-if="canUpdate" type="primary" @click="openCreateGroup">创建用户组</ElButton>
          </div>
        </div>
      </template>

      <ElEmpty v-if="!canView" description="你没有查看用户等级的权限。" />

      <template v-else>
        <ElTable
          :data="groups"
          row-key="id"
          border
          @row-click="handleSelectGroup"
          :row-class-name="getGroupRowClassName"
        >
          <ElTableColumn label="名称" min-width="180">
            <template #default="{ row }">
              <div class="group-name">
                <span class="group-dot" :style="{ backgroundColor: row.color || '#1677ff' }"></span>
                <span>{{ row.name || '-' }}</span>
              </div>
            </template>
          </ElTableColumn>

          <ElTableColumn label="图标" min-width="130">
            <template #default="{ row }">
              {{ getIconLabel(row.icon) }}
            </template>
          </ElTableColumn>

          <ElTableColumn prop="priority" label="优先级" width="100" />

          <ElTableColumn label="自动审批" width="110">
            <template #default="{ row }">
              <ElTag :type="row.auto_approve_enabled ? 'success' : 'info'">
                {{ row.auto_approve_enabled ? '开启' : '关闭' }}
              </ElTag>
            </template>
          </ElTableColumn>

          <ElTableColumn label="默认组" width="110">
            <template #default="{ row }">
              <ElTag :type="row.is_default ? 'primary' : 'info'">
                {{ row.is_default ? '默认组' : '-' }}
              </ElTag>
            </template>
          </ElTableColumn>

          <ElTableColumn label="操作" min-width="280" fixed="right">
            <template #default="{ row }">
              <div class="table-actions">
                <ElButton text type="primary" @click.stop="handleSelectGroup(row)"
                  >管理规则</ElButton
                >
                <ElButton v-if="canUpdate" text type="primary" @click.stop="openEditGroup(row)">
                  编辑
                </ElButton>
                <ElButton
                  v-if="canUpdate"
                  text
                  type="primary"
                  @click.stop="handleRebuildGroup(row)"
                >
                  重建缓存
                </ElButton>
                <ElButton
                  v-if="canUpdate"
                  text
                  type="danger"
                  :disabled="row.is_default"
                  @click.stop="handleDeleteGroup(row)"
                >
                  删除
                </ElButton>
              </div>
            </template>
          </ElTableColumn>
        </ElTable>
      </template>
    </ElCard>

    <ElCard v-loading="loading.rules" class="rule-card">
      <template #header>
        <div class="rule-header">
          <div>
            <div class="page-title">规则配置</div>
            <div class="page-subtitle">
              {{
                selectedGroup
                  ? `当前用户组：${selectedGroup.name}`
                  : '请选择一个用户组后再管理规则。'
              }}
            </div>
          </div>
        </div>
      </template>

      <ElEmpty v-if="!selectedGroup" description="请选择一个用户组管理规则。" />

      <template v-else>
        <ElTabs v-model="activeTab">
          <ElTabPane label="优惠策略" name="discount">
            <div class="tab-toolbar">
              <ElButton
                type="primary"
                :disabled="isSelectedGroupReadonly"
                @click="openCreateDiscountRule"
              >
                新增优惠规则
              </ElButton>
            </div>

            <ElTable :data="discountRules" row-key="id" border>
              <ElTableColumn label="范围" min-width="140">
                <template #default="{ row }">
                  {{ getScopeLabel(row.scope) }}
                </template>
              </ElTableColumn>

              <ElTableColumn label="折扣" width="110">
                <template #default="{ row }">
                  {{ renderPermille(row.discount_permille) }}
                </template>
              </ElTableColumn>

              <ElTableColumn label="固定价(分)" width="120">
                <template #default="{ row }">
                  {{ row.fixed_price ?? '-' }}
                </template>
              </ElTableColumn>

              <ElTableColumn label="类型/地区/线路/套餐" min-width="280">
                <template #default="{ row }">
                  {{ renderRuleTarget(row) }}
                </template>
              </ElTableColumn>

              <ElTableColumn label="操作" width="140" fixed="right">
                <template #default="{ row }">
                  <div class="table-actions">
                    <ElButton
                      text
                      type="primary"
                      :disabled="isSelectedGroupReadonly"
                      @click="openEditDiscountRule(row)"
                    >
                      编辑
                    </ElButton>
                    <ElButton
                      text
                      type="danger"
                      :disabled="isSelectedGroupReadonly"
                      @click="handleDeleteDiscountRule(row)"
                    >
                      删除
                    </ElButton>
                  </div>
                </template>
              </ElTableColumn>
            </ElTable>
          </ElTabPane>

          <ElTabPane label="自动审批条件" name="auto">
            <div class="tab-toolbar">
              <ElButton
                type="primary"
                :disabled="isSelectedGroupReadonly"
                @click="openCreateAutoRule"
              >
                新增审批条件
              </ElButton>
            </div>

            <ElTable :data="autoRules" row-key="id" border>
              <ElTableColumn prop="sort_order" label="排序" width="90" />
              <ElTableColumn prop="duration_days" label="时长(天)" width="110" />
              <ElTableColumn label="条件(JSON)" min-width="320">
                <template #default="{ row }">
                  {{ renderAutoConditions(row.conditions_json) }}
                </template>
              </ElTableColumn>
              <ElTableColumn label="操作" width="140" fixed="right">
                <template #default="{ row }">
                  <div class="table-actions">
                    <ElButton
                      text
                      type="primary"
                      :disabled="isSelectedGroupReadonly"
                      @click="openEditAutoRule(row)"
                    >
                      编辑
                    </ElButton>
                    <ElButton
                      text
                      type="danger"
                      :disabled="isSelectedGroupReadonly"
                      @click="handleDeleteAutoRule(row)"
                    >
                      删除
                    </ElButton>
                  </div>
                </template>
              </ElTableColumn>
            </ElTable>
          </ElTabPane>
        </ElTabs>
      </template>
    </ElCard>

    <GroupDialog
      v-model:visible="groupDialogVisible"
      :mode="groupDialogMode"
      :form-data="groupDialogForm"
      :submitting="submitting.group"
      @submit="handleGroupSubmit"
    />

    <DiscountRuleDialog
      v-model:visible="discountDialogVisible"
      :mode="discountDialogMode"
      :form-data="discountDialogForm"
      :goods-types="goodsTypes"
      :regions="regions"
      :plan-groups="planGroups"
      :packages="packages"
      :submitting="submitting.discount"
      @submit="handleDiscountSubmit"
    />

    <AutoRuleDialog
      v-model:visible="autoDialogVisible"
      :mode="autoDialogMode"
      :form-data="autoDialogForm"
      :submitting="submitting.auto"
      @submit="handleAutoSubmit"
    />
  </div>
</template>

<script setup lang="ts">
  import type {
    CatalogGoodsType,
    CatalogPackage,
    CatalogPlanGroup,
    CatalogRegion,
    UserTierAutoRule,
    UserTierDiscountRule,
    UserTierGroup
  } from '@/api/admin'
  import {
    createUserTierAutoRule,
    createUserTierDiscountRule,
    createUserTierGroup,
    deleteUserTierAutoRule,
    deleteUserTierDiscountRule,
    deleteUserTierGroup,
    fetchAdminGoodsTypes,
    fetchAdminPackages,
    fetchAdminPlanGroups,
    fetchAdminRegions,
    fetchUserTierAutoRules,
    fetchUserTierDiscountRules,
    fetchUserTierGroups,
    hasAdminPermission,
    rebuildUserTierCaches,
    updateUserTierAutoRule,
    updateUserTierDiscountRule,
    updateUserTierGroup
  } from '@/api/admin'
  import { useUserStore } from '@/store/modules/user'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import AutoRuleDialog, { type UserTierAutoRuleFormValue } from './modules/auto-rule-dialog.vue'
  import DiscountRuleDialog, {
    type UserTierDiscountRuleFormValue
  } from './modules/discount-rule-dialog.vue'
  import GroupDialog, { type UserTierGroupFormValue } from './modules/group-dialog.vue'

  defineOptions({ name: 'SystemUserTiersPage' })

  interface UserTierGroupRow {
    id: number | null
    name: string
    color: string
    icon: string
    priority: number
    auto_approve_enabled: boolean
    is_default: boolean
    created_at: string
    updated_at: string
  }

  interface UserTierDiscountRuleRow {
    id: number | null
    group_id: number | null
    scope: string
    goods_type_id: number
    region_id: number
    plan_group_id: number
    package_id: number
    discount_permille: number
    fixed_price: number | null
    add_core_permille: number
    add_mem_permille: number
    add_disk_permille: number
    add_bw_permille: number
  }

  interface UserTierAutoRuleRow {
    id: number | null
    group_id: number | null
    duration_days: number
    conditions_json: string
    sort_order: number
  }

  const userStore = useUserStore()
  const { info } = storeToRefs(userStore)

  const loading = reactive({
    groups: false,
    rules: false,
    lookups: false
  })

  const submitting = reactive({
    group: false,
    discount: false,
    auto: false
  })

  const groups = ref<UserTierGroupRow[]>([])
  const discountRules = ref<UserTierDiscountRuleRow[]>([])
  const autoRules = ref<UserTierAutoRuleRow[]>([])
  const goodsTypes = ref<CatalogGoodsType[]>([])
  const regions = ref<CatalogRegion[]>([])
  const planGroups = ref<CatalogPlanGroup[]>([])
  const packages = ref<CatalogPackage[]>([])

  const activeTab = ref('discount')
  const selectedGroupId = ref<number | null>(null)

  const groupDialogVisible = ref(false)
  const groupDialogMode = ref<'create' | 'edit'>('create')
  const groupDialogForm = ref<UserTierGroupFormValue>(createDefaultGroupForm())

  const discountDialogVisible = ref(false)
  const discountDialogMode = ref<'create' | 'edit'>('create')
  const discountDialogForm = ref<UserTierDiscountRuleFormValue>(createDefaultDiscountForm())

  const autoDialogVisible = ref(false)
  const autoDialogMode = ref<'create' | 'edit'>('create')
  const autoDialogForm = ref<UserTierAutoRuleFormValue>(createDefaultAutoForm())

  const iconOptions = [
    { value: 'badge', label: '认证徽章' },
    { value: 'star', label: '星标' },
    { value: 'crown', label: '皇冠' },
    { value: 'rocket', label: '火箭' },
    { value: 'trophy', label: '奖杯' },
    { value: 'fire', label: '火焰' },
    { value: 'thunder', label: '闪电' },
    { value: 'gift', label: '礼物' },
    { value: 'heart', label: '爱心' }
  ]

  const metricOptions = [
    { label: '注册时长(月)', value: 'register_months' },
    { label: '用户钱包余额(元)', value: 'wallet_balance' }
  ]

  const operatorOptions = [
    { label: '大于', value: 'gt' },
    { label: '小于', value: 'lt' },
    { label: '等于', value: 'eq' }
  ]

  const scopeLabelMap: Record<string, string> = {
    all: '全部(不含附加项)',
    all_addons: '全部附加项',
    goods_type: '类型',
    goods_type_region: '类型-地区',
    plan_group: '类型-地区-线路',
    package: '套餐',
    addon_config: '附加项配置'
  }

  const canView = computed(() => hasAdminPermission(info.value?.buttons, ['user.list']))
  const canUpdate = computed(() =>
    hasAdminPermission(info.value?.buttons, ['user.update', 'user.create'])
  )

  const selectedGroup = computed(
    () =>
      groups.value.find((item) => Number(item.id || 0) === Number(selectedGroupId.value || 0)) ||
      null
  )

  const isSelectedGroupReadonly = computed(
    () => !canUpdate.value || Boolean(selectedGroup.value?.is_default)
  )

  const goodsTypeNameMap = computed(() => {
    const map = new Map<number, string>()
    goodsTypes.value.forEach((item) => {
      map.set(Number(item.id || 0), String(item.name || `#${item.id}`))
    })
    return map
  })

  const regionNameMap = computed(() => {
    const map = new Map<number, string>()
    regions.value.forEach((item) => {
      map.set(Number(item.id || 0), String(item.name || `#${item.id}`))
    })
    return map
  })

  const planGroupNameMap = computed(() => {
    const map = new Map<number, string>()
    planGroups.value.forEach((item) => {
      map.set(Number(item.id || 0), String(item.name || `#${item.id}`))
    })
    return map
  })

  const packageNameMap = computed(() => {
    const map = new Map<number, string>()
    packages.value.forEach((item) => {
      map.set(Number(item.id || 0), String(item.name || `#${item.id}`))
    })
    return map
  })

  onMounted(async () => {
    await Promise.all([loadLookups(), fetchGroups()])
  })

  function createDefaultGroupForm(): UserTierGroupFormValue {
    return {
      id: null,
      name: '',
      color: '#1677ff',
      icon: 'badge',
      priority: 10,
      auto_approve_enabled: true,
      is_default: false
    }
  }

  function createDefaultDiscountForm(): UserTierDiscountRuleFormValue {
    return {
      id: null,
      scope: 'all',
      goods_type_id: 0,
      region_id: 0,
      plan_group_id: 0,
      package_id: 0,
      discount_permille: 1000,
      fixed_price: null,
      add_core_permille: 1000,
      add_mem_permille: 1000,
      add_disk_permille: 1000,
      add_bw_permille: 1000
    }
  }

  function createDefaultAutoForm(): UserTierAutoRuleFormValue {
    return {
      id: null,
      duration_days: -1,
      sort_order: 10,
      conditions_json: '[]'
    }
  }

  function normalizeNullableNumber(value: unknown): number | null {
    if (value === '' || value === null || value === undefined) {
      return null
    }

    const parsed = Number(value)
    return Number.isFinite(parsed) ? parsed : null
  }

  function normalizeScope(scope?: string) {
    if (scope === 'goods_type_region_plan_group') {
      return 'plan_group'
    }
    return String(scope || 'all')
  }

  function normalizeGroup(item?: UserTierGroup): UserTierGroupRow {
    return {
      id: normalizeNullableNumber(item?.id),
      name: String(item?.name || ''),
      color: String(item?.color || '#1677ff'),
      icon: String(item?.icon || 'badge'),
      priority: Number(item?.priority || 0),
      auto_approve_enabled: Boolean(item?.auto_approve_enabled),
      is_default: Boolean(item?.is_default),
      created_at: String(item?.created_at || ''),
      updated_at: String(item?.updated_at || '')
    }
  }

  function normalizeDiscountRule(item?: UserTierDiscountRule): UserTierDiscountRuleRow {
    return {
      id: normalizeNullableNumber(item?.id),
      group_id: normalizeNullableNumber(item?.group_id),
      scope: normalizeScope(item?.scope),
      goods_type_id: Number(item?.goods_type_id || 0),
      region_id: Number(item?.region_id || 0),
      plan_group_id: Number(item?.plan_group_id || 0),
      package_id: Number(item?.package_id || 0),
      discount_permille: Number(item?.discount_permille || 0),
      fixed_price:
        item?.fixed_price === null || item?.fixed_price === undefined
          ? null
          : Number(item.fixed_price),
      add_core_permille: Number(item?.add_core_permille || 0),
      add_mem_permille: Number(item?.add_mem_permille || 0),
      add_disk_permille: Number(item?.add_disk_permille || 0),
      add_bw_permille: Number(item?.add_bw_permille || 0)
    }
  }

  function normalizeAutoRule(item?: UserTierAutoRule): UserTierAutoRuleRow {
    return {
      id: normalizeNullableNumber(item?.id),
      group_id: normalizeNullableNumber(item?.group_id),
      duration_days: Number(item?.duration_days ?? -1),
      conditions_json: String(item?.conditions_json || '[]'),
      sort_order: Number(item?.sort_order || 0)
    }
  }

  async function loadLookups() {
    loading.lookups = true

    try {
      const [goodsTypePayload, regionPayload, planGroupPayload, packagePayload] = await Promise.all(
        [fetchAdminGoodsTypes(), fetchAdminRegions(), fetchAdminPlanGroups(), fetchAdminPackages()]
      )

      goodsTypes.value = goodsTypePayload.items || []
      regions.value = regionPayload.items || []
      planGroups.value = planGroupPayload.items || []
      packages.value = packagePayload.items || []
    } finally {
      loading.lookups = false
    }
  }

  async function fetchGroups(preferredGroupId?: number | null) {
    if (!canView.value) {
      return
    }

    loading.groups = true

    try {
      const payload = await fetchUserTierGroups()
      groups.value = (payload.items || []).map((item) => normalizeGroup(item))

      if (!groups.value.length) {
        selectedGroupId.value = null
        discountRules.value = []
        autoRules.value = []
        return
      }

      const preferred = preferredGroupId ?? selectedGroupId.value
      const selected =
        groups.value.find((item) => Number(item.id || 0) === Number(preferred || 0)) ||
        groups.value[0]

      await handleSelectGroup(selected)
    } finally {
      loading.groups = false
    }
  }

  async function fetchRules() {
    if (!selectedGroupId.value) {
      discountRules.value = []
      autoRules.value = []
      return
    }

    loading.rules = true

    try {
      const [discountPayload, autoPayload] = await Promise.all([
        fetchUserTierDiscountRules(selectedGroupId.value),
        fetchUserTierAutoRules(selectedGroupId.value)
      ])

      discountRules.value = (discountPayload.items || []).map((item) => normalizeDiscountRule(item))
      autoRules.value = (autoPayload.items || []).map((item) => normalizeAutoRule(item))
    } finally {
      loading.rules = false
    }
  }

  async function handleSelectGroup(row: UserTierGroupRow | null) {
    selectedGroupId.value = row?.id || null
    await fetchRules()
  }

  function getGroupRowClassName({ row }: { row: UserTierGroupRow }) {
    return Number(row.id || 0) === Number(selectedGroupId.value || 0) ? 'is-selected-group' : ''
  }

  function getIconLabel(icon?: string) {
    return iconOptions.find((item) => item.value === icon)?.label || icon || '-'
  }

  function getScopeLabel(scope?: string) {
    return scopeLabelMap[String(scope || '')] || scope || '-'
  }

  function renderPermille(value?: number) {
    return `${Number(value || 0) / 10}%`
  }

  function renderRuleTarget(row: UserTierDiscountRuleRow) {
    const goodsType = row.goods_type_id
      ? goodsTypeNameMap.value.get(Number(row.goods_type_id)) || `#${row.goods_type_id}`
      : '-'
    const region = row.region_id
      ? regionNameMap.value.get(Number(row.region_id)) || `#${row.region_id}`
      : '-'
    const planGroup = row.plan_group_id
      ? planGroupNameMap.value.get(Number(row.plan_group_id)) || `#${row.plan_group_id}`
      : '-'
    const packageName = row.package_id
      ? packageNameMap.value.get(Number(row.package_id)) || `#${row.package_id}`
      : '-'

    return `${goodsType}/${region}/${planGroup}/${packageName}`
  }

  function renderAutoConditions(raw?: string) {
    const text = String(raw || '').trim()
    if (!text) {
      return '任意'
    }

    try {
      const parsed = JSON.parse(text)
      if (!Array.isArray(parsed) || !parsed.length) {
        return '任意'
      }

      return parsed
        .map((item) => {
          const metric =
            metricOptions.find((option) => option.value === String(item?.metric || ''))?.label ||
            item?.metric ||
            '-'
          const operator =
            operatorOptions.find((option) => option.value === String(item?.operator || ''))
              ?.label ||
            item?.operator ||
            '-'
          return `${metric} ${operator} ${Number(item?.value || 0)}`
        })
        .join(' AND ')
    } catch {
      return '任意'
    }
  }

  function openCreateGroup() {
    groupDialogMode.value = 'create'
    groupDialogForm.value = createDefaultGroupForm()
    groupDialogVisible.value = true
  }

  function openEditGroup(row: UserTierGroupRow) {
    groupDialogMode.value = 'edit'
    groupDialogForm.value = {
      id: row.id,
      name: row.name,
      color: row.color,
      icon: row.icon,
      priority: row.priority,
      auto_approve_enabled: row.auto_approve_enabled,
      is_default: row.is_default
    }
    groupDialogVisible.value = true
  }

  async function handleGroupSubmit(form: UserTierGroupFormValue) {
    submitting.group = true

    try {
      if (groupDialogMode.value === 'create') {
        const created = await createUserTierGroup({
          name: form.name,
          color: form.color,
          icon: form.icon,
          priority: form.priority,
          auto_approve_enabled: form.auto_approve_enabled
        })
        ElMessage.success('用户组已创建')
        groupDialogVisible.value = false
        await fetchGroups(Number(created.id || 0) || null)
      } else {
        if (!form.id) {
          return
        }

        await updateUserTierGroup(form.id, {
          name: form.name,
          color: form.color,
          icon: form.icon,
          priority: form.priority,
          auto_approve_enabled: form.auto_approve_enabled
        })
        ElMessage.success('用户组已更新')
        groupDialogVisible.value = false
        await fetchGroups(form.id)
      }
    } finally {
      submitting.group = false
    }
  }

  async function handleDeleteGroup(row: UserTierGroupRow) {
    if (!row.id) {
      return
    }

    await ElMessageBox.confirm(`确定删除用户组 ${row.name} 吗？`, '删除确认', {
      type: 'warning',
      confirmButtonText: '确定',
      cancelButtonText: '取消'
    })

    await deleteUserTierGroup(row.id)
    ElMessage.success('用户组已删除')
    await fetchGroups()
  }

  async function handleRebuildAll() {
    await rebuildUserTierCaches()
    ElMessage.success('已触发全量缓存重建')
  }

  async function handleRebuildGroup(row: UserTierGroupRow) {
    if (!row.id) {
      return
    }

    await rebuildUserTierCaches(row.id)
    ElMessage.success('已触发分组缓存重建')
  }

  function openCreateDiscountRule() {
    discountDialogMode.value = 'create'
    discountDialogForm.value = createDefaultDiscountForm()
    discountDialogVisible.value = true
  }

  function openEditDiscountRule(row: UserTierDiscountRuleRow) {
    discountDialogMode.value = 'edit'
    discountDialogForm.value = {
      id: row.id,
      scope: normalizeScope(row.scope),
      goods_type_id: row.goods_type_id,
      region_id: row.region_id,
      plan_group_id: row.plan_group_id,
      package_id: row.package_id,
      discount_permille: row.discount_permille,
      fixed_price: row.fixed_price,
      add_core_permille: row.add_core_permille,
      add_mem_permille: row.add_mem_permille,
      add_disk_permille: row.add_disk_permille,
      add_bw_permille: row.add_bw_permille
    }
    discountDialogVisible.value = true
  }

  async function handleDiscountSubmit(form: UserTierDiscountRuleFormValue) {
    if (!selectedGroupId.value) {
      return
    }

    submitting.discount = true

    try {
      const payload = {
        scope: normalizeScope(form.scope),
        goods_type_id: Number(form.goods_type_id || 0),
        region_id: Number(form.region_id || 0),
        plan_group_id: Number(form.plan_group_id || 0),
        package_id: Number(form.package_id || 0),
        discount_permille: Number(form.discount_permille || 0),
        fixed_price:
          form.fixed_price === null || form.fixed_price === undefined
            ? null
            : Number(form.fixed_price),
        add_core_permille: Number(form.add_core_permille || 0),
        add_mem_permille: Number(form.add_mem_permille || 0),
        add_disk_permille: Number(form.add_disk_permille || 0),
        add_bw_permille: Number(form.add_bw_permille || 0)
      }

      if (discountDialogMode.value === 'create') {
        await createUserTierDiscountRule(selectedGroupId.value, payload)
        ElMessage.success('优惠规则已创建')
      } else {
        if (!form.id) {
          return
        }
        await updateUserTierDiscountRule(selectedGroupId.value, form.id, payload)
        ElMessage.success('优惠规则已更新')
      }

      discountDialogVisible.value = false
      await fetchRules()
    } finally {
      submitting.discount = false
    }
  }

  async function handleDeleteDiscountRule(row: UserTierDiscountRuleRow) {
    if (!selectedGroupId.value || !row.id) {
      return
    }

    await ElMessageBox.confirm('确定删除这条优惠规则吗？', '删除确认', {
      type: 'warning',
      confirmButtonText: '确定',
      cancelButtonText: '取消'
    })

    await deleteUserTierDiscountRule(selectedGroupId.value, row.id)
    ElMessage.success('优惠规则已删除')
    await fetchRules()
  }

  function openCreateAutoRule() {
    autoDialogMode.value = 'create'
    autoDialogForm.value = createDefaultAutoForm()
    autoDialogVisible.value = true
  }

  function openEditAutoRule(row: UserTierAutoRuleRow) {
    autoDialogMode.value = 'edit'
    autoDialogForm.value = {
      id: row.id,
      duration_days: row.duration_days,
      sort_order: row.sort_order,
      conditions_json: row.conditions_json
    }
    autoDialogVisible.value = true
  }

  async function handleAutoSubmit(form: UserTierAutoRuleFormValue) {
    if (!selectedGroupId.value) {
      return
    }

    submitting.auto = true

    try {
      const payload = {
        duration_days: Number(form.duration_days ?? -1),
        sort_order: Number(form.sort_order || 0),
        conditions_json: String(form.conditions_json || '[]')
      }

      if (autoDialogMode.value === 'create') {
        await createUserTierAutoRule(selectedGroupId.value, payload)
        ElMessage.success('审批规则已创建')
      } else {
        if (!form.id) {
          return
        }

        await updateUserTierAutoRule(selectedGroupId.value, form.id, payload)
        ElMessage.success('审批规则已更新')
      }

      autoDialogVisible.value = false
      await fetchRules()
    } finally {
      submitting.auto = false
    }
  }

  async function handleDeleteAutoRule(row: UserTierAutoRuleRow) {
    if (!selectedGroupId.value || !row.id) {
      return
    }

    await ElMessageBox.confirm('确定删除这条审批规则吗？', '删除确认', {
      type: 'warning',
      confirmButtonText: '确定',
      cancelButtonText: '取消'
    })

    await deleteUserTierAutoRule(selectedGroupId.value, row.id)
    ElMessage.success('审批规则已删除')
    await fetchRules()
  }
</script>

<style scoped lang="scss">
  .user-tier-page {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .page-header,
  .rule-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 16px;
  }

  .page-title {
    color: var(--el-text-color-primary);
    font-size: 18px;
    font-weight: 700;
  }

  .page-subtitle {
    margin-top: 6px;
    color: var(--el-text-color-secondary);
    font-size: 13px;
    line-height: 1.7;
  }

  .page-actions,
  .table-actions {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .table-actions {
    justify-content: flex-end;
    gap: 4px;
    flex-wrap: wrap;
  }

  .group-name {
    display: inline-flex;
    align-items: center;
    gap: 8px;
  }

  .group-dot {
    width: 10px;
    height: 10px;
    border-radius: 999px;
    flex: 0 0 10px;
  }

  .rule-card {
    min-height: 320px;
  }

  .tab-toolbar {
    margin-bottom: 12px;
  }

  :deep(.is-selected-group td) {
    background: rgb(37 99 235 / 6%) !important;
  }

  @media (max-width: 768px) {
    .page-header,
    .rule-header {
      flex-direction: column;
      align-items: stretch;
    }
  }
</style>
