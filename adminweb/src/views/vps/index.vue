<template>
  <div class="art-full-height">
    <VpsSearch
      v-show="showSearchBar"
      v-model="searchForm"
      @search="handleSearch"
      @reset="handleReset"
    />

    <ElCard class="art-table-card" :style="{ marginTop: showSearchBar ? '12px' : '0' }">
      <ArtTableHeader
        v-model:columns="columnChecks"
        v-model:showSearchBar="showSearchBar"
        :loading="loading"
        @refresh="fetchData"
      >
        <template #left>
          <ElSpace wrap>
            <ElButton v-if="canCreate" type="primary" v-ripple @click="openCreateRecord">
              一键添加记录
            </ElButton>

            <ElRadioGroup v-model="quickStatus" class="quick-status-group">
              <ElRadioButton label="all">全部</ElRadioButton>
              <ElRadioButton v-for="item in quickStatusTabs" :key="item.value" :label="item.value">
                {{ item.label }}
              </ElRadioButton>
            </ElRadioGroup>

            <ElButton v-ripple @click="exportCsv">导出 CSV</ElButton>
          </ElSpace>
        </template>
      </ArtTableHeader>

      <ArtTable
        row-key="id"
        :loading="loading"
        :data="tableData"
        :columns="columns"
        :pagination="pagination"
        @pagination:size-change="handlePageSizeChange"
        @pagination:current-change="handlePageCurrentChange"
      >
        <template #status="{ row }">
          <ElTag :type="getVpsStatusTagType(row.status)">{{ getVpsStatusText(row.status) }}</ElTag>
        </template>

        <template #admin_status="{ row }">
          <ElTag :type="getAdminStatusTagType(row.admin_status)">
            {{ getAdminStatusText(row.admin_status) }}
          </ElTag>
        </template>

        <template #expire_at="{ row }">
          <span :class="['expire-text', row.status === 'expired_locked' ? 'is-danger' : '']">
            {{ formatDateTime(row.expire_at) }}
          </span>
        </template>

        <template #operation="{ row }">
          <div class="table-actions">
            <ArtButtonTable v-if="canView" type="view" @click="openDetail(row)" />
            <ArtButtonTable v-if="canUpdate" type="edit" @click="openEdit(row)" />

            <ArtButtonMore
              :list="getMoreActions(row)"
              @click="(item) => handleMoreAction(item, row)"
            />
          </div>
        </template>
      </ArtTable>
    </ElCard>

    <ElDialog
      v-model="createVisible"
      title="一键添加记录"
      width="760px"
      destroy-on-close
      align-center
    >
      <ElForm label-width="110px">
        <ElRow :gutter="16">
          <ElCol :span="12">
            <ElFormItem label="用户" required>
              <ElSelect v-model="createForm.user_id" filterable clearable placeholder="选择用户">
                <ElOption
                  v-for="item in createUsers"
                  :key="item.id"
                  :label="getCreateUserLabel(item)"
                  :value="item.id"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>
          <ElCol :span="12">
            <ElFormItem label="机器名" required>
              <ElInput
                v-model="createForm.name"
                placeholder="必须与自动化系统机器名一致"
                :maxlength="INPUT_LIMITS.VPS_NAME"
              />
            </ElFormItem>
          </ElCol>
        </ElRow>

        <ElRow :gutter="16">
          <ElCol :span="12">
            <ElFormItem label="商品类型" required>
              <ElSelect
                v-model="createForm.goods_type_id"
                clearable
                placeholder="选择商品类型"
                @change="onCreateGoodsTypeChange"
              >
                <ElOption
                  v-for="item in createGoodsTypes"
                  :key="item.id"
                  :label="item.name || item.code || String(item.id)"
                  :value="Number(item.id || 0)"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>
          <ElCol :span="12">
            <ElFormItem label="地区" required>
              <ElSelect
                v-model="createForm.region_id"
                clearable
                placeholder="选择地区"
                @change="onCreateRegionChange"
              >
                <ElOption
                  v-for="item in createRegions"
                  :key="item.id"
                  :label="item.name || item.code || String(item.id)"
                  :value="Number(item.id || 0)"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>
        </ElRow>

        <ElRow :gutter="16">
          <ElCol :span="12">
            <ElFormItem label="线路" required>
              <ElSelect
                v-model="createForm.line_id"
                clearable
                placeholder="选择线路"
                @change="onCreatePlanGroupChange"
              >
                <ElOption
                  v-for="item in createPlanGroups"
                  :key="item.id"
                  :label="item.name || String(item.id)"
                  :value="Number(item.id || 0)"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>
          <ElCol :span="12">
            <ElFormItem label="套餐" required>
              <ElSelect
                v-model="createForm.package_id"
                clearable
                placeholder="选择套餐"
                @change="onCreatePackageChange"
              >
                <ElOption
                  v-for="item in createPackages"
                  :key="item.id"
                  :label="item.name || String(item.id)"
                  :value="Number(item.id || 0)"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>
        </ElRow>

        <ElRow :gutter="16">
          <ElCol :span="12">
            <ElFormItem label="价格(月费)" required>
              <ElInputNumber
                v-model="createForm.monthly_price"
                :min="0"
                :precision="2"
                style="width: 100%"
              />
            </ElFormItem>
          </ElCol>
          <ElCol :span="12">
            <ElFormItem label="到期时间">
              <ElDatePicker
                v-model="createForm.expire_at"
                type="datetime"
                clearable
                placeholder="可选"
                style="width: 100%"
              />
            </ElFormItem>
          </ElCol>
        </ElRow>
      </ElForm>

      <template #footer>
        <div class="dialog-footer">
          <ElButton @click="createVisible = false">取消</ElButton>
          <ElButton type="primary" :loading="createSubmitting" @click="submitCreateRecord">
            确认创建
          </ElButton>
        </div>
      </template>
    </ElDialog>

    <ElDialog
      v-model="statusVisible"
      title="设置实例状态"
      width="480px"
      destroy-on-close
      align-center
    >
      <ElForm label-width="100px">
        <ElFormItem label="状态">
          <ElSelect v-model="statusForm.status">
            <ElOption label="normal" value="normal" />
            <ElOption label="abuse" value="abuse" />
            <ElOption label="fraud" value="fraud" />
            <ElOption label="locked" value="locked" />
          </ElSelect>
        </ElFormItem>
        <ElFormItem label="原因">
          <ElInput
            v-model="statusForm.reason"
            type="textarea"
            :rows="4"
            :maxlength="INPUT_LIMITS.REVIEW_REASON"
            show-word-limit
          />
        </ElFormItem>
      </ElForm>

      <template #footer>
        <div class="dialog-footer">
          <ElButton @click="statusVisible = false">取消</ElButton>
          <ElButton type="primary" :loading="statusSubmitting" @click="submitStatus">
            保存状态
          </ElButton>
        </div>
      </template>
    </ElDialog>

    <ElDialog v-model="renewVisible" title="紧急续费" width="420px" destroy-on-close align-center>
      <ElAlert type="info" :closable="false" show-icon>
        <template #title>将按生命周期设置中的天数、窗口和冷却策略触发一次紧急续费。</template>
      </ElAlert>

      <template #footer>
        <div class="dialog-footer">
          <ElButton @click="renewVisible = false">取消</ElButton>
          <ElButton type="primary" :loading="renewSubmitting" @click="submitRenew">
            立即触发
          </ElButton>
        </div>
      </template>
    </ElDialog>

    <ElDialog v-model="resizeVisible" title="改配" width="560px" destroy-on-close align-center>
      <ElForm label-width="110px">
        <ElRow :gutter="16">
          <ElCol :span="12">
            <ElFormItem label="CPU">
              <ElInputNumber v-model="resizeForm.cpu" :min="0" style="width: 100%" />
            </ElFormItem>
          </ElCol>
          <ElCol :span="12">
            <ElFormItem label="内存(GB)">
              <ElInputNumber v-model="resizeForm.memory_gb" :min="0" style="width: 100%" />
            </ElFormItem>
          </ElCol>
        </ElRow>
        <ElRow :gutter="16">
          <ElCol :span="12">
            <ElFormItem label="磁盘(GB)">
              <ElInputNumber v-model="resizeForm.disk_gb" :min="0" style="width: 100%" />
            </ElFormItem>
          </ElCol>
          <ElCol :span="12">
            <ElFormItem label="带宽(Mbps)">
              <ElInputNumber v-model="resizeForm.bandwidth_mbps" :min="0" style="width: 100%" />
            </ElFormItem>
          </ElCol>
        </ElRow>
      </ElForm>

      <template #footer>
        <div class="dialog-footer">
          <ElButton @click="resizeVisible = false">取消</ElButton>
          <ElButton type="primary" :loading="resizeSubmitting" @click="submitResize">
            提交改配
          </ElButton>
        </div>
      </template>
    </ElDialog>

    <ElDialog
      v-model="expireVisible"
      title="修改到期时间"
      width="440px"
      destroy-on-close
      align-center
    >
      <ElForm label-width="100px">
        <ElFormItem label="到期时间">
          <ElDatePicker
            v-model="expireForm.expire_at"
            type="datetime"
            clearable
            placeholder="请选择到期时间"
            style="width: 100%"
          />
        </ElFormItem>
      </ElForm>

      <template #footer>
        <div class="dialog-footer">
          <ElButton @click="expireVisible = false">取消</ElButton>
          <ElButton type="primary" :loading="expireSubmitting" @click="submitExpire">
            保存时间
          </ElButton>
        </div>
      </template>
    </ElDialog>

    <ElDialog v-model="deleteVisible" title="删除实例" width="500px" destroy-on-close align-center>
      <ElForm label-width="100px">
        <ElFormItem label="删除原因">
          <ElInput
            v-model="deleteReason"
            type="textarea"
            :rows="4"
            placeholder="可选，便于审计与自动退款"
            :maxlength="INPUT_LIMITS.REVIEW_REASON"
            show-word-limit
          />
        </ElFormItem>
      </ElForm>

      <template #footer>
        <div class="dialog-footer">
          <ElButton @click="deleteVisible = false">取消</ElButton>
          <ElButton type="danger" :loading="deleteSubmitting" @click="submitDelete">
            确认删除
          </ElButton>
        </div>
      </template>
    </ElDialog>

    <ElDialog v-model="editVisible" title="编辑 VPS" width="720px" destroy-on-close align-center>
      <ElForm label-width="110px">
        <ElRow :gutter="16">
          <ElCol :span="12">
            <ElFormItem label="同步模式">
              <ElSelect v-model="editForm.sync_mode">
                <ElOption label="只修改本地" value="local" />
                <ElOption label="同步到自动化" value="automation" />
              </ElSelect>
            </ElFormItem>
          </ElCol>
          <ElCol :span="12">
            <ElFormItem label="套餐 ID">
              <ElInputNumber v-model="editForm.package_id" :min="0" style="width: 100%" />
            </ElFormItem>
          </ElCol>
        </ElRow>
        <ElRow :gutter="16">
          <ElCol :span="12">
            <ElFormItem label="月费">
              <ElInputNumber
                v-model="editForm.monthly_price"
                :min="0"
                :precision="2"
                style="width: 100%"
              />
            </ElFormItem>
          </ElCol>
          <ElCol :span="12">
            <ElFormItem label="套餐名称">
              <ElInput v-model="editForm.package_name" />
            </ElFormItem>
          </ElCol>
        </ElRow>
        <ElRow :gutter="16">
          <ElCol :span="8">
            <ElFormItem label="CPU (核)">
              <ElInputNumber v-model="editForm.cpu" :min="0" style="width: 100%" />
            </ElFormItem>
          </ElCol>
          <ElCol :span="8">
            <ElFormItem label="内存 (GB)">
              <ElInputNumber v-model="editForm.memory_gb" :min="0" style="width: 100%" />
            </ElFormItem>
          </ElCol>
          <ElCol :span="8">
            <ElFormItem label="磁盘 (GB)">
              <ElInputNumber v-model="editForm.disk_gb" :min="0" style="width: 100%" />
            </ElFormItem>
          </ElCol>
        </ElRow>
        <ElRow :gutter="16">
          <ElCol :span="12">
            <ElFormItem label="带宽 (Mbps)">
              <ElInputNumber v-model="editForm.bandwidth_mbps" :min="0" style="width: 100%" />
            </ElFormItem>
          </ElCol>
          <ElCol :span="12">
            <ElFormItem label="端口数">
              <ElInputNumber v-model="editForm.port_num" :min="0" style="width: 100%" />
            </ElFormItem>
          </ElCol>
        </ElRow>
        <ElRow :gutter="16">
          <ElCol :span="12">
            <ElFormItem label="地区">
              <ElInput v-model="editForm.region" disabled />
            </ElFormItem>
          </ElCol>
          <ElCol :span="12">
            <ElFormItem label="线路 ID">
              <ElInputNumber v-model="editForm.line_id" disabled style="width: 100%" />
            </ElFormItem>
          </ElCol>
        </ElRow>
        <ElRow :gutter="16">
          <ElCol :span="12">
            <ElFormItem label="实例名称">
              <ElInput v-model="editForm.name" disabled />
            </ElFormItem>
          </ElCol>
          <ElCol :span="12">
            <ElFormItem label="自动化实例 ID">
              <ElInput v-model="editForm.automation_instance_id" disabled />
            </ElFormItem>
          </ElCol>
        </ElRow>
        <ElRow :gutter="16">
          <ElCol :span="12">
            <ElFormItem label="状态">
              <ElSelect v-model="editForm.status">
                <ElOption label="运行中" value="running" />
                <ElOption label="已关机" value="stopped" />
                <ElOption label="开通中" value="provisioning" />
              </ElSelect>
            </ElFormItem>
          </ElCol>
          <ElCol :span="12">
            <ElFormItem label="管理状态">
              <ElSelect v-model="editForm.admin_status">
                <ElOption label="normal" value="normal" />
                <ElOption label="abuse" value="abuse" />
                <ElOption label="fraud" value="fraud" />
                <ElOption label="locked" value="locked" />
              </ElSelect>
            </ElFormItem>
          </ElCol>
        </ElRow>
        <ElFormItem label="系统镜像 ID">
          <ElInputNumber v-model="editForm.system_id" :min="0" style="width: 100%" />
        </ElFormItem>
      </ElForm>

      <template #footer>
        <div class="dialog-footer">
          <ElButton @click="editVisible = false">取消</ElButton>
          <ElButton type="primary" :loading="editSubmitting" @click="submitEdit">
            保存修改
          </ElButton>
        </div>
      </template>
    </ElDialog>

    <VpsDetailDrawer
      v-model:visible="detailVisible"
      :detail="detailData"
      :loading="detailLoading"
      :can-refresh="canRefresh"
      :can-update="canUpdate"
      :can-delete="canDelete"
      :can-lock="canLock"
      :can-unlock="canUnlock"
      :can-resize="canResize"
      :can-emergency-renew="canEmergencyRenew"
      :can-update-status="canUpdateStatus"
      :can-update-expire="canUpdateExpire"
      @edit="openEdit(detailData)"
      @status="openStatus(detailData)"
      @resize="openResize(detailData)"
      @expire="openExpire(detailData)"
      @refresh="handleRefresh(detailData)"
      @renew="openRenew(detailData)"
      @lock="confirmLock(detailData)"
      @unlock="confirmUnlock(detailData)"
      @delete="openDelete(detailData)"
    />
  </div>
</template>

<script setup lang="ts">
  import type {
    CatalogGoodsType,
    CatalogPackage,
    CatalogPlanGroup,
    CatalogRegion,
    UserRecord,
    VpsRecord
  } from '@/api/admin'
  import type { ButtonMoreItem } from '@/components/core/forms/art-button-more/index.vue'
  import {
    createAdminVps,
    deleteAdminVps,
    emergencyRenewAdminVps,
    fetchAdminGoodsTypes,
    fetchAdminPackages,
    fetchAdminPlanGroups,
    fetchAdminRegions,
    fetchAdminUsers,
    fetchAdminVps,
    fetchAdminVpsDetail,
    hasAdminPermission,
    lockAdminVps,
    refreshAdminVps,
    resizeAdminVps,
    unlockAdminVps,
    updateAdminVps,
    updateAdminVpsExpire,
    updateAdminVpsStatus
  } from '@/api/admin'
  import ArtButtonMore from '@/components/core/forms/art-button-more/index.vue'
  import ArtButtonTable from '@/components/core/forms/art-button-table/index.vue'
  import { useTableColumns } from '@/hooks/core/useTableColumns'
  import { useUserStore } from '@/store/modules/user'
  import { INPUT_LIMITS } from '@/utils/constants'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import VpsDetailDrawer from './modules/vps-detail-drawer.vue'
  import VpsSearch from './modules/vps-search.vue'

  defineOptions({ name: 'VpsPage' })

  interface VpsSearchForm {
    keyword: string
    status?: string
  }

  interface VpsTableRow {
    id: number | null
    user_id: number | null
    order_item_id: number | null
    goods_type_id: number | null
    automation_instance_id: string
    name: string
    region: string
    region_id: number | null
    line_id: number | null
    package_id: number | null
    package_name: string
    cpu: number
    memory_gb: number
    disk_gb: number
    bandwidth_mbps: number
    port_num: number
    monthly_price: number
    spec: Record<string, unknown>
    system_id: number | null
    status: string
    automation_state: number | null
    admin_status: string
    expire_at: string | null
    destroy_at: string | null
    destroy_in_days: number | null
    panel_url_cache: string
    access_info: Record<string, unknown>
    last_emergency_renew_at: string | null
    created_at: string
    updated_at: string
  }

  interface CreateUserOption {
    id: number
    username: string
    email: string
  }

  interface CreateFormValue {
    user_id: number | null
    name: string
    goods_type_id: number | null
    region_id: number | null
    line_id: number | null
    package_id: number | null
    monthly_price: number
    expire_at: Date | null
  }

  interface StatusFormValue {
    status: string
    reason: string
  }

  interface ResizeFormValue {
    cpu: number
    memory_gb: number
    disk_gb: number
    bandwidth_mbps: number
  }

  interface ExpireFormValue {
    expire_at: Date | null
  }

  interface EditFormValue {
    sync_mode: string
    package_id: number
    monthly_price: number
    package_name: string
    cpu: number
    memory_gb: number
    disk_gb: number
    bandwidth_mbps: number
    port_num: number
    status: string
    admin_status: string
    system_id: number
    region: string
    line_id: number
    name: string
    automation_instance_id: string
  }

  const quickStatusTabs = [
    { label: 'normal', value: 'normal' },
    { label: 'abuse', value: 'abuse' },
    { label: 'fraud', value: 'fraud' }
  ] as const

  const showSearchBar = ref(true)
  const loading = ref(false)
  const detailLoading = ref(false)

  const createVisible = ref(false)
  const statusVisible = ref(false)
  const renewVisible = ref(false)
  const resizeVisible = ref(false)
  const expireVisible = ref(false)
  const deleteVisible = ref(false)
  const editVisible = ref(false)
  const detailVisible = ref(false)

  const createSubmitting = ref(false)
  const statusSubmitting = ref(false)
  const renewSubmitting = ref(false)
  const resizeSubmitting = ref(false)
  const expireSubmitting = ref(false)
  const deleteSubmitting = ref(false)
  const editSubmitting = ref(false)

  const searchForm = ref<VpsSearchForm>(createDefaultSearchForm())
  const tableData = ref<VpsTableRow[]>([])
  const detailData = ref<VpsTableRow | null>(null)
  const activeRecord = ref<VpsTableRow | null>(null)

  const createUsers = ref<CreateUserOption[]>([])
  const createGoodsTypes = ref<CatalogGoodsType[]>([])
  const createRegions = ref<CatalogRegion[]>([])
  const createPlanGroups = ref<CatalogPlanGroup[]>([])
  const createPackages = ref<CatalogPackage[]>([])

  const rowProcessingIds = ref<number[]>([])

  const createForm = reactive<CreateFormValue>(createDefaultCreateForm())
  const statusForm = reactive<StatusFormValue>({
    status: 'normal',
    reason: ''
  })
  const resizeForm = reactive<ResizeFormValue>({
    cpu: 0,
    memory_gb: 0,
    disk_gb: 0,
    bandwidth_mbps: 0
  })
  const expireForm = reactive<ExpireFormValue>({
    expire_at: null
  })
  const deleteReason = ref('')
  const editForm = reactive<EditFormValue>(createDefaultEditForm())

  const pagination = reactive({
    current: 1,
    size: 20,
    total: 0
  })

  const { columnChecks, columns } = useTableColumns<VpsTableRow>(() => [
    { prop: 'id', label: '实例 ID', width: 90 },
    { prop: 'user_id', label: '用户', width: 90 },
    { prop: 'name', label: '实例名', minWidth: 160, showOverflowTooltip: true },
    { prop: 'region', label: '地区', minWidth: 120, showOverflowTooltip: true },
    { prop: 'package_name', label: '套餐', minWidth: 150, showOverflowTooltip: true },
    {
      prop: 'monthly_price',
      label: '月费',
      width: 110,
      formatter: (row: VpsTableRow) => formatPrice(row.monthly_price)
    },
    { prop: 'status', label: '状态', width: 120, useSlot: true },
    { prop: 'admin_status', label: '管理状态', width: 120, useSlot: true },
    { prop: 'expire_at', label: '到期时间', minWidth: 180, useSlot: true },
    { prop: 'operation', label: '操作', width: 180, fixed: 'right', useSlot: true }
  ])

  const userStore = useUserStore()
  const { info } = storeToRefs(userStore)
  const route = useRoute()
  const router = useRouter()

  const canView = computed(() => hasAdminPermission(info.value?.buttons, ['vps.view']))
  const canCreate = computed(() => hasAdminPermission(info.value?.buttons, ['vps.create']))
  const canUpdate = computed(() => hasAdminPermission(info.value?.buttons, ['vps.update']))
  const canDelete = computed(() => hasAdminPermission(info.value?.buttons, ['vps.delete']))
  const canLock = computed(() => hasAdminPermission(info.value?.buttons, ['vps.lock']))
  const canUnlock = computed(() => hasAdminPermission(info.value?.buttons, ['vps.unlock']))
  const canResize = computed(() => hasAdminPermission(info.value?.buttons, ['vps.resize']))
  const canRefresh = computed(() => hasAdminPermission(info.value?.buttons, ['vps.refresh']))
  const canEmergencyRenew = computed(() =>
    hasAdminPermission(info.value?.buttons, ['vps.emergency_renew'])
  )
  const canUpdateStatus = computed(() =>
    hasAdminPermission(info.value?.buttons, ['vps.admin_status'])
  )
  const canUpdateExpire = computed(() =>
    hasAdminPermission(info.value?.buttons, ['vps.update_expire'])
  )

  const quickStatus = computed({
    get: () => {
      const status = searchForm.value.status || ''
      return quickStatusTabs.some((item) => item.value === status) ? status : 'all'
    },
    set: (value: string) => {
      searchForm.value = {
        ...searchForm.value,
        status: value === 'all' ? undefined : value
      }
      pagination.current = 1
      fetchData()
    }
  })

  onMounted(() => {
    fetchData()
  })

  watch(
    () => route.query.id,
    (value) => {
      const idText = Array.isArray(value) ? value[0] : value
      const id = Number(idText || 0)

      if (!id || Number.isNaN(id)) {
        detailVisible.value = false
        detailData.value = null
        return
      }

      loadDetail(id)
    },
    { immediate: true }
  )

  watch(detailVisible, (visible) => {
    if (!visible && route.query.id) {
      clearDetailQuery()
    }
  })

  function createDefaultSearchForm(): VpsSearchForm {
    return {
      keyword: '',
      status: undefined
    }
  }

  function createDefaultCreateForm(): CreateFormValue {
    return {
      user_id: null,
      name: '',
      goods_type_id: null,
      region_id: null,
      line_id: null,
      package_id: null,
      monthly_price: 0,
      expire_at: null
    }
  }

  function createDefaultEditForm(): EditFormValue {
    return {
      sync_mode: 'local',
      package_id: 0,
      monthly_price: 0,
      package_name: '',
      cpu: 0,
      memory_gb: 0,
      disk_gb: 0,
      bandwidth_mbps: 0,
      port_num: 0,
      status: 'running',
      admin_status: 'normal',
      system_id: 0,
      region: '',
      line_id: 0,
      name: '',
      automation_instance_id: ''
    }
  }

  function normalizeNullableNumber(value: unknown): number | null {
    if (value === '' || value === null || value === undefined) {
      return null
    }

    const parsed = Number(value)
    return Number.isFinite(parsed) ? parsed : null
  }

  function normalizeNullableString(value: unknown): string | null {
    if (value === '' || value === null || value === undefined) {
      return null
    }

    return String(value)
  }

  function normalizeRecord(value: unknown): Record<string, unknown> {
    if (!value || typeof value !== 'object' || Array.isArray(value)) {
      return {}
    }

    return value as Record<string, unknown>
  }

  function statusFromAutomation(state: unknown) {
    switch (Number(state)) {
      case 1:
      case 13:
        return 'provisioning'
      case 2:
        return 'running'
      case 3:
        return 'stopped'
      case 4:
        return 'reinstalling'
      case 5:
        return 'reinstall_failed'
      case 10:
        return 'locked'
      case 11:
        return 'failed'
      case 12:
        return 'deleting'
      default:
        return ''
    }
  }

  function isExpired(row?: Partial<VpsTableRow> | null) {
    const expireAt = row?.expire_at
    if (!expireAt) {
      return false
    }

    const expire = new Date(expireAt).getTime()
    if (Number.isNaN(expire)) {
      return false
    }

    return expire <= Date.now()
  }

  function shouldShowExpiredLocked(row: Partial<VpsTableRow>, status: string) {
    if (!isExpired(row)) {
      return false
    }

    return status === 'locked' || status === 'expired_locked'
  }

  function normalizeVps(row?: VpsRecord): VpsTableRow {
    const automationState = normalizeNullableNumber(row?.automation_state)
    const rawStatus = String(row?.status || '')
    const baseStatus =
      automationState !== null && automationState !== undefined
        ? statusFromAutomation(automationState)
        : rawStatus
    const resolvedStatus = shouldShowExpiredLocked(
      {
        expire_at: normalizeNullableString(row?.expire_at)
      },
      baseStatus
    )
      ? 'expired_locked'
      : baseStatus

    return {
      id: normalizeNullableNumber(row?.id),
      user_id: normalizeNullableNumber(row?.user_id),
      order_item_id: normalizeNullableNumber(row?.order_item_id),
      goods_type_id: normalizeNullableNumber(row?.goods_type_id),
      automation_instance_id: String(row?.automation_instance_id || ''),
      name: String(row?.name || ''),
      region: String(row?.region || ''),
      region_id: normalizeNullableNumber(row?.region_id),
      line_id: normalizeNullableNumber(row?.line_id),
      package_id: normalizeNullableNumber(row?.package_id),
      package_name: String(row?.package_name || ''),
      cpu: Number(row?.cpu || 0),
      memory_gb: Number(row?.memory_gb || 0),
      disk_gb: Number(row?.disk_gb || 0),
      bandwidth_mbps: Number(row?.bandwidth_mbps || 0),
      port_num: Number(row?.port_num || 0),
      monthly_price: Number(row?.monthly_price || 0),
      spec: normalizeRecord(row?.spec),
      system_id: normalizeNullableNumber(row?.system_id),
      status: resolvedStatus,
      automation_state: automationState,
      admin_status: String(row?.admin_status || ''),
      expire_at: normalizeNullableString(row?.expire_at),
      destroy_at: normalizeNullableString(row?.destroy_at),
      destroy_in_days: normalizeNullableNumber(row?.destroy_in_days),
      panel_url_cache: String(row?.panel_url_cache || ''),
      access_info: normalizeRecord(row?.access_info),
      last_emergency_renew_at: normalizeNullableString(row?.last_emergency_renew_at),
      created_at: String(row?.created_at || ''),
      updated_at: String(row?.updated_at || '')
    }
  }

  function normalizeGoodsType(item?: CatalogGoodsType) {
    return {
      id: normalizeNullableNumber(item?.id) || 0,
      code: String(item?.code || ''),
      name: String(item?.name || ''),
      active: Boolean(item?.active),
      sort_order: Number(item?.sort_order || 0)
    }
  }

  function normalizeRegion(item?: CatalogRegion) {
    return {
      id: normalizeNullableNumber(item?.id) || 0,
      goods_type_id: normalizeNullableNumber(item?.goods_type_id) || 0,
      name: String(item?.name || ''),
      code: String(item?.code || ''),
      active: Boolean(item?.active),
      visible: Boolean(item?.visible),
      sort_order: Number(item?.sort_order || 0)
    }
  }

  function normalizePlanGroup(item?: CatalogPlanGroup) {
    return {
      id: normalizeNullableNumber(item?.id) || 0,
      goods_type_id: normalizeNullableNumber(item?.goods_type_id) || 0,
      region_id: normalizeNullableNumber(item?.region_id) || 0,
      line_id: normalizeNullableNumber(item?.line_id) || 0,
      name: String(item?.name || ''),
      active: Boolean(item?.active),
      visible: Boolean(item?.visible),
      capacity_remaining: Number(item?.capacity_remaining || 0),
      sort_order: Number(item?.sort_order || 0)
    }
  }

  function normalizePackage(item?: CatalogPackage) {
    return {
      id: normalizeNullableNumber(item?.id) || 0,
      goods_type_id: normalizeNullableNumber(item?.goods_type_id) || 0,
      plan_group_id: normalizeNullableNumber(item?.plan_group_id) || 0,
      name: String(item?.name || ''),
      monthly_price: Number(item?.monthly_price || 0)
    }
  }

  function normalizeCreateUser(item?: UserRecord): CreateUserOption {
    return {
      id: normalizeNullableNumber(item?.id) || 0,
      username: String(item?.username || ''),
      email: String(item?.email || '')
    }
  }

  function getCreateUserLabel(user: CreateUserOption) {
    return `${user.username || '用户'} (#${user.id}) ${user.email || ''}`.trim()
  }

  function hasActiveFilters() {
    return Boolean(searchForm.value.keyword.trim() || searchForm.value.status)
  }

  function applyClientFilters(items: VpsTableRow[]) {
    let rows = [...items]
    const keyword = searchForm.value.keyword.trim().toLowerCase()

    if (searchForm.value.status) {
      rows = rows.filter((item) => item.admin_status === searchForm.value.status)
    }

    if (keyword) {
      rows = rows.filter((item) =>
        [
          item.id,
          item.user_id,
          item.name,
          item.region,
          item.package_name,
          item.automation_instance_id
        ]
          .map((value) => String(value ?? '').toLowerCase())
          .some((value) => value.includes(keyword))
      )
    }

    return rows
  }

  function formatPrice(value?: number | null) {
    return Number(value || 0).toFixed(2)
  }

  function formatDateTime(value?: string | null) {
    if (!value) {
      return '-'
    }

    const date = new Date(value)
    return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN')
  }

  function formatBackendDateTime(value?: Date | null) {
    if (!value) {
      return ''
    }

    const year = value.getFullYear()
    const month = String(value.getMonth() + 1).padStart(2, '0')
    const day = String(value.getDate()).padStart(2, '0')
    const hours = String(value.getHours()).padStart(2, '0')
    const minutes = String(value.getMinutes()).padStart(2, '0')
    const seconds = String(value.getSeconds()).padStart(2, '0')
    return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`
  }

  function getVpsStatusText(status?: string) {
    switch (status) {
      case 'running':
        return '运行中'
      case 'stopped':
        return '已关机'
      case 'provisioning':
        return '开通中'
      case 'reinstalling':
        return '重装中'
      case 'reinstall_failed':
        return '重装失败'
      case 'locked':
        return '已锁定'
      case 'failed':
        return '异常'
      case 'deleting':
        return '删除中'
      case 'expired_locked':
        return '已到期'
      default:
        return status || '-'
    }
  }

  function getVpsStatusTagType(status?: string) {
    switch (status) {
      case 'running':
        return 'success' as const
      case 'provisioning':
      case 'reinstalling':
      case 'deleting':
        return 'warning' as const
      case 'reinstall_failed':
      case 'locked':
      case 'failed':
      case 'expired_locked':
        return 'danger' as const
      default:
        return 'info' as const
    }
  }

  function getAdminStatusText(status?: string) {
    switch (status) {
      case 'normal':
        return 'normal'
      case 'abuse':
        return 'abuse'
      case 'fraud':
        return 'fraud'
      case 'locked':
        return 'locked'
      default:
        return status || '-'
    }
  }

  function getAdminStatusTagType(status?: string) {
    switch (status) {
      case 'normal':
        return 'success' as const
      case 'abuse':
        return 'warning' as const
      case 'fraud':
      case 'locked':
        return 'danger' as const
      default:
        return 'info' as const
    }
  }

  function isRowProcessing(row?: VpsTableRow | null) {
    return Boolean(row?.id && rowProcessingIds.value.includes(Number(row.id)))
  }

  function setRowProcessing(id: number, processing: boolean) {
    if (processing) {
      if (!rowProcessingIds.value.includes(id)) {
        rowProcessingIds.value = [...rowProcessingIds.value, id]
      }
      return
    }

    rowProcessingIds.value = rowProcessingIds.value.filter((item) => item !== id)
  }

  function getMoreActions(row: VpsTableRow): ButtonMoreItem[] {
    const processing = isRowProcessing(row)

    return [
      {
        key: 'status',
        label: '设置状态',
        icon: 'ri:shield-keyhole-line',
        disabled: !canUpdateStatus.value || processing
      },
      {
        key: 'expire',
        label: '修改到期',
        icon: 'ri:calendar-event-line',
        disabled: !canUpdateExpire.value || processing
      },
      {
        key: 'refresh',
        label: '刷新',
        icon: 'ri:refresh-line',
        disabled: !canRefresh.value || processing
      },
      {
        key: 'renew',
        label: '紧急续费',
        icon: 'ri:loop-right-line',
        disabled: !canEmergencyRenew.value || processing
      },
      {
        key: 'lock',
        label: '锁定',
        icon: 'ri:lock-2-line',
        disabled: !canLock.value || processing
      },
      {
        key: 'unlock',
        label: '解锁',
        icon: 'ri:lock-unlock-line',
        disabled: !canUnlock.value || processing
      },
      {
        key: 'resize',
        label: '改配',
        icon: 'ri:expand-width-line',
        disabled: !canResize.value || processing
      },
      {
        key: 'delete',
        label: '删除',
        icon: 'ri:delete-bin-line',
        color: '#f56c6c',
        disabled: !canDelete.value || processing
      }
    ]
  }

  async function fetchData() {
    loading.value = true

    try {
      const payload = await fetchAdminVps({
        limit: pagination.size,
        offset: (pagination.current - 1) * pagination.size
      })

      const rows = applyClientFilters((payload.items || []).map((item) => normalizeVps(item)))
      tableData.value = rows
      pagination.total = hasActiveFilters()
        ? rows.length
        : typeof payload.total === 'number'
          ? payload.total
          : rows.length
    } finally {
      loading.value = false
    }
  }

  async function loadDetail(id: number) {
    detailLoading.value = true

    try {
      const payload = await fetchAdminVpsDetail(id)
      detailData.value = normalizeVps(payload)
      detailVisible.value = true
    } catch {
      detailVisible.value = false
      detailData.value = null
      clearDetailQuery()
    } finally {
      detailLoading.value = false
    }
  }

  function clearDetailQuery() {
    const query = { ...route.query }
    delete query.id
    router.replace({ query })
  }

  async function reloadCurrentDetail() {
    if (!detailData.value?.id) {
      return
    }

    await loadDetail(detailData.value.id)
  }

  function openDetail(row?: VpsTableRow | null) {
    if (!row?.id) {
      return
    }

    if (String(route.query.id || '') === String(row.id)) {
      loadDetail(row.id)
      return
    }

    router.replace({
      query: {
        ...route.query,
        id: String(row.id)
      }
    })
  }

  function handleSearch(params: VpsSearchForm) {
    searchForm.value = { ...searchForm.value, ...params }
    pagination.current = 1
    fetchData()
  }

  function handleReset() {
    searchForm.value = createDefaultSearchForm()
    pagination.current = 1
    fetchData()
  }

  function handlePageSizeChange(size: number) {
    pagination.size = size
    pagination.current = 1
    fetchData()
  }

  function handlePageCurrentChange(page: number) {
    pagination.current = page
    fetchData()
  }

  async function loadCreateUsers() {
    const payload = await fetchAdminUsers({ limit: 200, offset: 0 })
    createUsers.value = (payload.items || [])
      .map((item) => normalizeCreateUser(item))
      .filter((item) => item.id > 0)
  }

  async function loadCreateGoodsTypes() {
    const payload = await fetchAdminGoodsTypes()
    createGoodsTypes.value = (payload.items || [])
      .map((item) => normalizeGoodsType(item))
      .filter((item) => item.id > 0)
  }

  async function loadCreateRegions(goodsTypeId?: number | null) {
    if (!goodsTypeId) {
      createRegions.value = []
      return
    }

    const payload = await fetchAdminRegions({ goods_type_id: goodsTypeId })
    createRegions.value = (payload.items || [])
      .map((item) => normalizeRegion(item))
      .filter((item) => item.id > 0)
  }

  async function loadCreatePlanGroups(goodsTypeId?: number | null, regionId?: number | null) {
    if (!goodsTypeId) {
      createPlanGroups.value = []
      return
    }

    const payload = await fetchAdminPlanGroups({ goods_type_id: goodsTypeId })
    let items = (payload.items || [])
      .map((item) => normalizePlanGroup(item))
      .filter((item) => item.id > 0)

    if (regionId) {
      items = items.filter((item) => Number(item.region_id) === Number(regionId))
    }

    createPlanGroups.value = items
  }

  async function loadCreatePackages(goodsTypeId?: number | null, lineId?: number | null) {
    if (!goodsTypeId || !lineId) {
      createPackages.value = []
      return
    }

    const payload = await fetchAdminPackages({
      goods_type_id: goodsTypeId,
      plan_group_id: lineId
    })

    createPackages.value = (payload.items || [])
      .map((item) => normalizePackage(item))
      .filter((item) => item.id > 0)
  }

  async function openCreateRecord() {
    Object.assign(createForm, createDefaultCreateForm())
    createRegions.value = []
    createPlanGroups.value = []
    createPackages.value = []
    await Promise.all([loadCreateUsers(), loadCreateGoodsTypes()])
    createVisible.value = true
  }

  async function onCreateGoodsTypeChange() {
    createForm.region_id = null
    createForm.line_id = null
    createForm.package_id = null
    createForm.monthly_price = 0
    createPlanGroups.value = []
    createPackages.value = []
    await loadCreateRegions(createForm.goods_type_id)
    await loadCreatePlanGroups(createForm.goods_type_id, null)
  }

  async function onCreateRegionChange() {
    createForm.line_id = null
    createForm.package_id = null
    createForm.monthly_price = 0
    createPackages.value = []
    await loadCreatePlanGroups(createForm.goods_type_id, createForm.region_id)
  }

  async function onCreatePlanGroupChange() {
    createForm.package_id = null
    createForm.monthly_price = 0
    await loadCreatePackages(createForm.goods_type_id, createForm.line_id)
  }

  function onCreatePackageChange() {
    const matched = createPackages.value.find(
      (item) => Number(item.id) === Number(createForm.package_id)
    )

    if (matched) {
      createForm.monthly_price = Number(matched.monthly_price || 0)
    }
  }

  async function submitCreateRecord() {
    if (
      !createForm.user_id ||
      !createForm.name.trim() ||
      !createForm.goods_type_id ||
      !createForm.region_id ||
      !createForm.line_id ||
      !createForm.package_id
    ) {
      ElMessage.error('请完整填写必填项')
      return
    }

    createSubmitting.value = true

    try {
      await createAdminVps({
        user_id: createForm.user_id,
        name: createForm.name.trim(),
        goods_type_id: createForm.goods_type_id,
        region_id: createForm.region_id,
        line_id: createForm.line_id,
        package_id: createForm.package_id,
        monthly_price: createForm.monthly_price,
        ...(createForm.expire_at ? { expire_at: createForm.expire_at.toISOString() } : {}),
        provision: false
      })

      createVisible.value = false
      ElMessage.success('记录添加成功')
      await fetchData()
    } finally {
      createSubmitting.value = false
    }
  }

  function openStatus(row?: VpsTableRow | null) {
    if (!row) {
      return
    }

    activeRecord.value = row
    statusForm.status = row.admin_status || 'normal'
    statusForm.reason = ''
    statusVisible.value = true
  }

  async function submitStatus() {
    if (!activeRecord.value?.id) {
      return
    }

    statusSubmitting.value = true

    try {
      await updateAdminVpsStatus(activeRecord.value.id, {
        status: statusForm.status,
        reason: statusForm.reason
      })

      try {
        await refreshAdminVps(activeRecord.value.id)
      } catch (error: any) {
        ElMessage.warning(error?.message || '同步状态失败')
      }

      statusVisible.value = false
      ElMessage.success('已更新状态')
      await fetchData()
      if (detailData.value?.id === activeRecord.value.id) {
        await reloadCurrentDetail()
      }
    } finally {
      statusSubmitting.value = false
    }
  }

  function openRenew(row?: VpsTableRow | null) {
    if (!row) {
      return
    }

    activeRecord.value = row
    renewVisible.value = true
  }

  async function submitRenew() {
    if (!activeRecord.value?.id) {
      return
    }

    renewSubmitting.value = true

    try {
      await emergencyRenewAdminVps(activeRecord.value.id, {})
      renewVisible.value = false
      ElMessage.success('已触发紧急续费')
      await fetchData()
      if (detailData.value?.id === activeRecord.value.id) {
        await reloadCurrentDetail()
      }
    } finally {
      renewSubmitting.value = false
    }
  }

  function openResize(row?: VpsTableRow | null) {
    if (!row) {
      return
    }

    activeRecord.value = row
    resizeForm.cpu = 0
    resizeForm.memory_gb = 0
    resizeForm.disk_gb = 0
    resizeForm.bandwidth_mbps = 0
    resizeVisible.value = true
  }

  async function submitResize() {
    if (!activeRecord.value?.id) {
      return
    }

    resizeSubmitting.value = true

    try {
      await resizeAdminVps(activeRecord.value.id, {
        cpu: resizeForm.cpu,
        memory_gb: resizeForm.memory_gb,
        disk_gb: resizeForm.disk_gb,
        bandwidth_mbps: resizeForm.bandwidth_mbps
      })

      resizeVisible.value = false
      ElMessage.success('已提交改配')
    } finally {
      resizeSubmitting.value = false
    }
  }

  function openExpire(row?: VpsTableRow | null) {
    if (!row) {
      return
    }

    activeRecord.value = row
    expireForm.expire_at = row.expire_at ? new Date(row.expire_at) : null
    expireVisible.value = true
  }

  async function submitExpire() {
    if (!activeRecord.value?.id || !expireForm.expire_at) {
      return
    }

    expireSubmitting.value = true

    try {
      await updateAdminVpsExpire(activeRecord.value.id, {
        expire_at: formatBackendDateTime(expireForm.expire_at)
      })

      expireVisible.value = false
      ElMessage.success('已修改到期时间')
      await fetchData()
      if (detailData.value?.id === activeRecord.value.id) {
        await reloadCurrentDetail()
      }
    } finally {
      expireSubmitting.value = false
    }
  }

  function openDelete(row?: VpsTableRow | null) {
    if (!row) {
      return
    }

    activeRecord.value = row
    deleteReason.value = ''
    deleteVisible.value = true
  }

  async function submitDelete() {
    if (!activeRecord.value?.id) {
      return
    }

    deleteSubmitting.value = true

    try {
      await deleteAdminVps(activeRecord.value.id, { reason: deleteReason.value })
      deleteVisible.value = false
      ElMessage.success('已删除')
      await fetchData()

      if (detailData.value?.id === activeRecord.value.id) {
        detailVisible.value = false
        detailData.value = null
      }
    } finally {
      deleteSubmitting.value = false
    }
  }

  function openEdit(row?: VpsTableRow | null) {
    if (!row) {
      return
    }

    activeRecord.value = row
    Object.assign(editForm, createDefaultEditForm(), {
      package_id: Number(row.package_id || 0),
      monthly_price: Number(row.monthly_price || 0),
      package_name: row.package_name || '',
      cpu: Number(row.cpu || 0),
      memory_gb: Number(row.memory_gb || 0),
      disk_gb: Number(row.disk_gb || 0),
      bandwidth_mbps: Number(row.bandwidth_mbps || 0),
      port_num: Number(row.port_num || 0),
      status: row.status || 'running',
      admin_status: row.admin_status || 'normal',
      system_id: Number(row.system_id || 0),
      region: row.region || '',
      line_id: Number(row.line_id || 0),
      name: row.name || '',
      automation_instance_id: row.automation_instance_id || ''
    })
    editVisible.value = true
  }

  async function submitEdit() {
    if (!activeRecord.value?.id) {
      return
    }

    editSubmitting.value = true

    try {
      await updateAdminVps(activeRecord.value.id, {
        sync_mode: editForm.sync_mode,
        package_id: editForm.package_id || undefined,
        monthly_price: editForm.monthly_price,
        package_name: editForm.package_name || undefined,
        cpu: editForm.cpu,
        memory_gb: editForm.memory_gb,
        disk_gb: editForm.disk_gb,
        bandwidth_mbps: editForm.bandwidth_mbps,
        port_num: editForm.port_num,
        status: editForm.status,
        admin_status: editForm.admin_status,
        system_id: editForm.system_id || undefined
      })

      editVisible.value = false
      ElMessage.success('已更新 VPS')
      await fetchData()
      if (detailData.value?.id === activeRecord.value.id) {
        await reloadCurrentDetail()
      }
    } finally {
      editSubmitting.value = false
    }
  }

  async function runRowAction(
    row: VpsTableRow,
    handler: () => Promise<unknown>,
    successMessage: string,
    options: { reloadTable?: boolean; reloadDetail?: boolean } = {}
  ) {
    if (!row.id) {
      return
    }

    setRowProcessing(row.id, true)

    try {
      await handler()
      ElMessage.success(successMessage)

      if (options.reloadTable !== false) {
        await fetchData()
      }

      if (options.reloadDetail !== false && detailData.value?.id === row.id) {
        await reloadCurrentDetail()
      }
    } finally {
      setRowProcessing(row.id, false)
    }
  }

  async function handleRefresh(row?: VpsTableRow | null) {
    if (!row?.id) {
      return
    }

    await runRowAction(row, () => refreshAdminVps(row.id as number), '已刷新')
  }

  async function confirmLock(row?: VpsTableRow | null) {
    if (!row?.id) {
      return
    }

    await ElMessageBox.confirm('锁定该实例？', '确认操作', { type: 'warning' })
    await runRowAction(row, () => lockAdminVps(row.id as number), '已锁定')
  }

  async function confirmUnlock(row?: VpsTableRow | null) {
    if (!row?.id) {
      return
    }

    await ElMessageBox.confirm('解锁该实例？', '确认操作', { type: 'warning' })
    await runRowAction(row, () => unlockAdminVps(row.id as number), '已解锁')
  }

  function handleMoreAction(item: ButtonMoreItem, row: VpsTableRow) {
    switch (item.key) {
      case 'status':
        openStatus(row)
        break
      case 'expire':
        openExpire(row)
        break
      case 'refresh':
        handleRefresh(row)
        break
      case 'renew':
        openRenew(row)
        break
      case 'lock':
        confirmLock(row)
        break
      case 'unlock':
        confirmUnlock(row)
        break
      case 'resize':
        openResize(row)
        break
      case 'delete':
        openDelete(row)
        break
    }
  }

  function escapeCsvCell(value: string | number | null | undefined) {
    const text = String(value ?? '')
    return `"${text.replace(/"/g, '""')}"`
  }

  function exportCsv() {
    const rows = tableData.value.map((item) =>
      [escapeCsvCell(item.id), escapeCsvCell(item.status)].join(',')
    )
    const content = ['id,status', ...rows].join('\n')
    const blob = new Blob([content], { type: 'text/csv;charset=utf-8;' })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')

    link.href = url
    link.download = 'admin-vps.csv'
    link.click()

    URL.revokeObjectURL(url)
  }
</script>

<style scoped lang="scss">
  .table-actions {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: 2px;
  }

  .dialog-footer {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
  }

  .expire-text.is-danger {
    color: var(--el-color-danger);
    font-weight: 600;
  }
</style>
