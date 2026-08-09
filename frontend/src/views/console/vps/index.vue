<template>
  <div class="vps-list-page art-full-height">
    <div class="page-heading">
      <div>
        <div class="eyebrow">资源管理</div>
        <h1>云服务器</h1>
        <p>查看实例状态、到期时间，并执行常用生命周期操作。</p>
      </div>
      <div class="heading-actions">
        <ElButton :icon="Refresh" :loading="loading" @click="refreshData">刷新</ElButton>
        <ElButton type="primary" :icon="Plus" @click="goBuy">购买 VPS</ElButton>
      </div>
    </div>

    <div class="stats-grid">
      <ElCard v-for="item in stats" :key="item.label" class="stat-card" shadow="never">
        <div class="stat-icon" :class="`is-${item.tone}`">
          <ElIcon><component :is="item.icon" /></ElIcon>
        </div>
        <div>
          <div class="stat-value">{{ item.value }}</div>
          <div class="stat-label">{{ item.label }}</div>
        </div>
      </ElCard>
    </div>

    <div class="status-tabs">
      <ElSegmented v-model="searchForm.status" :options="statusTabs" @change="handleStatusTab" />
    </div>

    <ArtSearchBar
      v-model="searchForm"
      :items="searchItems"
      :default-expanded="true"
      @search="handleSearch"
      @reset="handleReset"
    />

    <ElCard class="art-table-card table-card" shadow="never">
      <ArtTableHeader v-model:columns="columnChecks" :loading="loading" @refresh="refreshData">
        <template #left>
          <ElSpace wrap>
            <ElButton :icon="Download" @click="exportCsv">导出 CSV</ElButton>
          </ElSpace>
        </template>
      </ArtTableHeader>

      <ArtTable
        :loading="loading"
        :data="data"
        :columns="columns"
        :pagination="pagination"
        row-key="id"
        @pagination:size-change="handleSizeChange"
        @pagination:current-change="handleCurrentChange"
      >
        <template #name="{ row }">
          <div class="name-cell">
            <div class="name-title">{{ row.name || `VPS-${row.id}` }}</div>
            <div class="name-id">ID: {{ row.id || '-' }}</div>
          </div>
        </template>
        <template #regionLine="{ row }">
          <span>{{ row.regionLine || '-' }}</span>
        </template>
        <template #spec="{ row }">
          <ElTag effect="plain">{{ row.specText }}</ElTag>
          <div class="muted-text">{{ row.package_name || '-' }}</div>
        </template>
        <template #ip="{ row }">
          <ElText truncated class="mono">{{ row.ip || '-' }}</ElText>
        </template>
        <template #status="{ row }">
          <VpsStatusTag :status="row.status" />
        </template>
        <template #expire_at="{ row }">
          <span :class="{ 'is-expiring': isExpiring(row.expire_at) }">{{
            formatDateTime(row.expire_at)
          }}</span>
          <div v-if="row.destroy_in_days != null" class="muted-text">
            {{ row.destroy_in_days }} 天后自动删除
          </div>
        </template>
        <template #operation="{ row }">
          <ElSpace :size="4">
            <ElTooltip content="详情" placement="top">
              <ElButton link type="primary" :icon="View" @click="goDetail(row)" />
            </ElTooltip>
            <ElTooltip content="控制面板" placement="top">
              <ElButton link type="primary" :icon="Monitor" @click="openPanel(row)" />
            </ElTooltip>
            <ElButton
              v-if="emergencyRenewEligible(row)"
              link
              type="danger"
              @click="submitEmergencyRenew(row)"
            >
              紧急续费
            </ElButton>
            <ElDropdown @command="(command: string) => handleCommand(command, row)">
              <ElButton link type="primary" :icon="MoreFilled" aria-label="更多操作" />
              <template #dropdown>
                <ElDropdownMenu>
                  <ElDropdownItem command="vnc">VNC</ElDropdownItem>
                  <ElDropdownItem command="start">开机</ElDropdownItem>
                  <ElDropdownItem command="shutdown">关机</ElDropdownItem>
                  <ElDropdownItem command="reboot">重启</ElDropdownItem>
                  <ElDropdownItem command="refresh">刷新状态</ElDropdownItem>
                  <ElDropdownItem divided command="renew">续费</ElDropdownItem>
                  <ElDropdownItem v-if="emergencyRenewEligible(row)" command="urgent-renew"
                    >紧急续费</ElDropdownItem
                  >
                  <ElDropdownItem command="resize" :disabled="isExpired(row)"
                    >升降配</ElDropdownItem
                  >
                  <ElDropdownItem divided command="refund">申请退款</ElDropdownItem>
                </ElDropdownMenu>
              </template>
            </ElDropdown>
          </ElSpace>
        </template>
      </ArtTable>
    </ElCard>

    <div class="mobile-list">
      <div class="mobile-list-heading">
        <span>实例列表</span>
        <span class="muted-text">{{ pagination.total }} 台</span>
      </div>
      <ElSkeleton v-if="loading" :rows="4" animated />
      <ElEmpty v-else-if="!data.length" description="暂无云服务器" />
      <ElCard
        v-for="item in data"
        v-else
        :key="item.id"
        class="mobile-vps-card"
        shadow="never"
        @click="goDetail(item)"
      >
        <div class="mobile-card-heading">
          <div>
            <div class="name-title">{{ item.name || `VPS-${item.id}` }}</div>
            <div class="name-id">ID: {{ item.id }}</div>
          </div>
          <VpsStatusTag :status="item.status" />
        </div>
        <div class="mobile-details">
          <span
            ><ElIcon><Location /></ElIcon>{{ item.regionLine || '-' }}</span
          >
          <span
            ><ElIcon><Cpu /></ElIcon>{{ item.specText }}</span
          >
          <span
            ><ElIcon><Connection /></ElIcon>{{ item.ip || '-' }}</span
          >
          <span
            ><ElIcon><Calendar /></ElIcon>{{ formatDateTime(item.expire_at) }}</span
          >
        </div>
        <div v-if="item.destroy_in_days != null" class="mobile-destroy-warning">
          <ElIcon><Warning /></ElIcon>
          <span>将在 {{ item.destroy_in_days }} 天后自动删除</span>
        </div>
        <div class="mobile-card-actions" @click.stop>
          <ElButton link type="primary" :icon="View" @click="goDetail(item)">详情</ElButton>
          <ElButton link type="primary" :icon="Monitor" @click="openPanel(item)">面板</ElButton>
          <ElButton link type="primary" :icon="VideoCamera" @click="openVnc(item)">VNC</ElButton>
          <ElButton
            v-if="emergencyRenewEligible(item)"
            link
            type="danger"
            :icon="Calendar"
            @click="submitEmergencyRenew(item)"
            >紧急续费</ElButton
          >
          <ElButton link type="primary" :icon="MoreFilled" @click="showMobileActions(item)"
            >更多</ElButton
          >
        </div>
      </ElCard>
      <ElPagination
        v-if="pagination.total"
        class="mobile-pagination"
        :current-page="pagination.current"
        :page-size="pagination.size"
        :total="pagination.total"
        layout="prev, pager, next"
        @current-change="handleCurrentChange"
      />
    </div>

    <ElDialog v-model="renewOpen" title="续费 VPS" width="min(460px, calc(100vw - 32px))">
      <ElAlert
        title="续费会生成订单，请在订单详情中完成支付。"
        type="info"
        show-icon
        :closable="false"
      />
      <ElForm label-position="top" class="dialog-form">
        <ElFormItem label="续费时长">
          <ElInputNumber v-model="renewForm.months" :min="1" :max="120" class="full-width" />
        </ElFormItem>
        <ElAlert
          v-if="activeRecord"
          :title="`月费 ￥${Number(activeRecord.monthly_price || 0).toFixed(2)} × ${renewForm.months} 个月`"
          type="info"
          show-icon
          :closable="false"
        />
      </ElForm>
      <template #footer>
        <ElButton @click="renewOpen = false">取消</ElButton>
        <ElButton type="primary" :loading="renewing" @click="submitRenew">生成续费订单</ElButton>
      </template>
    </ElDialog>

    <ElDialog v-model="resizeOpen" title="升降配 VPS" width="min(560px, calc(100vw - 32px))">
      <ElForm label-position="top" class="dialog-form">
        <ElFormItem label="目标套餐">
          <ElSelect
            v-model="resizeForm.target_package_id"
            class="full-width"
            :disabled="!resizeEnabled"
            filterable
          >
            <ElOption
              v-for="pkg in packageOptions"
              :key="getCatalogId(pkg)"
              :label="`${pkg.name ?? pkg.Name ?? `套餐 ${getCatalogId(pkg)}`} · ￥${getPackageMonthlyPrice(pkg).toFixed(2)}/月`"
              :value="getCatalogId(pkg)"
            />
          </ElSelect>
        </ElFormItem>
        <ElAlert
          v-if="!resizeEnabled"
          title="升降配功能已关闭"
          type="warning"
          show-icon
          :closable="false"
        />
        <ElAlert
          v-if="isExpired(activeRecord)"
          title="已到期实例不支持升降配"
          type="warning"
          show-icon
          :closable="false"
        />
        <ElAlert
          v-if="isSameTargetSelection"
          title="不能选择当前套餐"
          type="warning"
          show-icon
          :closable="false"
        />
        <ElFormItem label="执行时间">
          <ElRadioGroup v-model="resizeForm.schedule_mode">
            <ElRadio value="now">立即执行</ElRadio>
            <ElRadio value="later">指定时间</ElRadio>
          </ElRadioGroup>
        </ElFormItem>
        <ElFormItem v-if="resizeForm.schedule_mode === 'later'" label="指定时间">
          <ElDatePicker
            v-model="resizeForm.scheduled_at"
            type="datetime"
            class="full-width"
            value-format="YYYY-MM-DD HH:mm:ss"
          />
        </ElFormItem>
        <ElFormItem label="附加项">
          <ElSwitch v-model="resizeForm.reset_addons" active-text="清零附加项" />
        </ElFormItem>
        <ElRow :gutter="12">
          <ElCol v-for="field in addonFields" :key="field.key" :span="12">
            <ElFormItem :label="field.label">
              <ElInputNumber
                v-model="resizeForm[field.key]"
                class="full-width"
                :min="addonMin[field.key]"
                :max="addonMax[field.key]"
                :step="addonStep[field.key]"
                :disabled="resizeForm.reset_addons"
              />
            </ElFormItem>
          </ElCol>
        </ElRow>
        <ElAlert
          v-if="resizeQuoteLoading"
          title="正在计算价格..."
          type="info"
          show-icon
          :closable="false"
        />
        <ElAlert
          v-else-if="resizeQuoteError"
          :title="resizeQuoteError"
          type="error"
          show-icon
          :closable="false"
        />
        <ElAlert
          v-else-if="resizeQuote"
          :title="resizeQuoteTitle"
          :description="resizeQuoteDescription"
          type="success"
          show-icon
          :closable="false"
        />
      </ElForm>
      <template #footer>
        <ElButton @click="resizeOpen = false">取消</ElButton>
        <ElButton
          type="primary"
          :loading="resizing"
          :disabled="isExpired(activeRecord)"
          @click="submitResize"
          >生成改配订单</ElButton
        >
      </template>
    </ElDialog>

    <ElDialog v-model="refundOpen" title="申请退款" width="min(500px, calc(100vw - 32px))">
      <ElAlert title="退款申请提交后会进入审核流程。" type="warning" show-icon :closable="false" />
      <ElForm label-position="top" class="dialog-form">
        <ElFormItem label="退款原因" required>
          <ElInput
            v-model="refundReason"
            type="textarea"
            :rows="4"
            :maxlength="INPUT_LIMITS.REFUND_REASON"
            show-word-limit
            placeholder="请说明退款原因"
          />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="refundOpen = false">取消</ElButton>
        <ElButton type="danger" :loading="refunding" @click="submitRefund">提交申请</ElButton>
      </template>
    </ElDialog>

    <ElDrawer v-model="mobileActionsOpen" title="操作" direction="btt" size="320px">
      <div class="action-grid">
        <ElButton
          v-for="action in mobileActions"
          :key="action.key"
          text
          @click="handleMobileAction(action.key)"
        >
          <ElIcon><component :is="action.icon" /></ElIcon>
          {{ action.label }}
        </ElButton>
      </div>
    </ElDrawer>
  </div>
</template>

<script setup lang="ts">
  import dayjs from 'dayjs'
  import {
    Calendar,
    Connection,
    Cpu,
    Download,
    Location,
    Loading,
    Monitor,
    MoreFilled,
    Plus,
    Refresh,
    RefreshRight,
    SwitchButton,
    VideoCamera,
    View,
    Warning
  } from '@element-plus/icons-vue'
  import { useTable } from '@/hooks/core/useTable'
  import { useCatalogStore } from '@/stores/catalog'
  import { useSiteStore } from '@/stores/site'
  import {
    createVpsRenewOrder,
    createVpsResizeOrder,
    emergencyRenewVps,
    getVpsPanelUrl,
    getVpsVncUrl,
    listVps,
    quoteVpsResizeOrder,
    rebootVps,
    refreshVps,
    requestVpsRefund,
    shutdownVps,
    startVps
  } from '@/services/user'
  import type { Line, Package, VPSInstance } from '@/services/types'
  import { INPUT_LIMITS } from '@/constants/inputLimits'

  defineOptions({ name: 'ConsoleVps' })

  type Identifier = number | string
  type JsonObject = Record<string, unknown>
  type VpsApiRecord = {
    [key: string]: unknown
    id?: Identifier
    ID?: Identifier
    name?: string
    Name?: string
    region?: string
    Region?: string
    line?: string
    Line?: string
    line_name?: string
    LineName?: string
    status?: string
    Status?: string
    automation_state?: number
    AutomationState?: number
    expire_at?: string
    ExpireAt?: string
    destroy_in_days?: number
    DestroyInDays?: number
    last_emergency_renew_at?: string
    LastEmergencyRenewAt?: string
    package_id?: Identifier
    PackageID?: Identifier
    package_name?: string
    spec?: unknown
    Spec?: unknown
    spec_json?: unknown
    SpecJSON?: unknown
    access_info?: unknown
    AccessInfo?: unknown
    access_info_json?: unknown
    AccessInfoJSON?: unknown
    monthly_price?: number
    MonthlyPrice?: number
  }
  type VpsRecord = {
    id: Identifier
    name?: string
    regionLine: string
    status: string
    expire_at?: string
    destroy_in_days?: number
    last_emergency_renew_at?: string
    package_id?: Identifier
    package_name?: string
    spec_raw?: unknown
    specText: string
    ip: string
    monthly_price?: number
    cpu?: unknown
    CPU?: unknown
    cores?: unknown
    Cores?: unknown
    memory_gb?: unknown
    mem_gb?: unknown
    MemoryGB?: unknown
    disk_gb?: unknown
    DiskGB?: unknown
    bandwidth_mbps?: unknown
    bandwidth?: unknown
    BandwidthMB?: unknown
    ExpireAt?: string
  }
  type VpsTableParams = {
    current: number
    size: number
    keyword?: string
    status?: string
    region?: string
    expire_days?: number
  }
  type CatalogPackage = Package & {
    ID?: Identifier
    Name?: string
    planGroupId?: Identifier
    PlanGroupID?: Identifier
    ProductID?: Identifier
    CPU?: number
    Cores?: number
    mem_gb?: number
    MemoryGB?: number
    DiskGB?: number
    BandwidthMB?: number
    bandwidth?: number
    Monthly?: number
    MonthlyPrice?: number
    Active?: boolean
    Visible?: boolean
  }
  type CatalogPlanGroup = Line & { ID?: Identifier }
  type PackageSpecSource = {
    id?: Identifier
    ID?: Identifier
    product_id?: Identifier
    ProductID?: Identifier
    cpu?: unknown
    CPU?: unknown
    cores?: unknown
    Cores?: unknown
    memory_gb?: unknown
    mem_gb?: unknown
    MemoryGB?: unknown
    disk_gb?: unknown
    DiskGB?: unknown
    bandwidth_mbps?: unknown
    bandwidth?: unknown
    BandwidthMB?: unknown
  }
  type ResizeForm = {
    add_cores: number
    add_mem_gb: number
    add_disk_gb: number
    add_bw_mbps: number
    target_package_id: Identifier | null
    reset_addons: boolean
    schedule_mode: 'now' | 'later'
    scheduled_at: string | null
  }

  const router = useRouter()
  const catalog = useCatalogStore()
  const site = useSiteStore()
  const allVps = ref<VpsRecord[]>([])
  const searchForm = ref({
    keyword: '',
    status: '',
    region: '',
    expire_days: undefined as number | undefined
  })
  const searchItems = computed(() => [
    {
      key: 'keyword',
      label: '关键词',
      type: 'input',
      placeholder: '名称、ID 或 IP',
      clearable: true
    },
    {
      key: 'status',
      label: '状态',
      type: 'select',
      props: {
        clearable: true,
        placeholder: '全部状态',
        options: statusOptions
      }
    },
    {
      key: 'region',
      label: '地区/线路',
      type: 'input',
      placeholder: '输入地区或线路',
      clearable: true
    },
    {
      key: 'expire_days',
      label: '到期天数',
      type: 'number',
      props: { min: 1, max: 365, controlsPosition: 'right' }
    }
  ])

  const statusOptions = [
    { label: '运行中', value: 'running' },
    { label: '已关机', value: 'stopped' },
    { label: '锁定', value: 'locked' },
    { label: '已到期', value: 'expired_locked' },
    { label: '开通中', value: 'provisioning' },
    { label: '重装系统中', value: 'reinstalling' },
    { label: '重装系统失败', value: 'reinstall_failed' },
    { label: '创建失败', value: 'failed' },
    { label: '删除中', value: 'deleting' }
  ]
  const statusTabs = [
    { label: '全部', value: '' },
    { label: '运行中', value: 'running' },
    { label: '已关机', value: 'stopped' },
    { label: '锁定', value: 'locked' },
    { label: '已到期', value: 'expired_locked' },
    { label: '重装中', value: 'reinstalling' }
  ]

  const isJsonObject = (value: unknown): value is JsonObject =>
    typeof value === 'object' && value !== null && !Array.isArray(value)

  const parseJson = (input: unknown): JsonObject => {
    if (!input) return {}
    if (typeof input === 'string') {
      try {
        const parsed: unknown = JSON.parse(input)
        return isJsonObject(parsed) ? parsed : {}
      } catch {
        return {}
      }
    }
    return isJsonObject(input) ? input : {}
  }

  const normalizeSpec = (spec: unknown, fallback: JsonObject = {}) => {
    const value = parseJson(spec)
    if (!Object.keys(value).length && !Object.keys(fallback).length) return '-'
    const cpu =
      value.cpu ?? value.cores ?? value.CPU ?? value.Cores ?? fallback.cpu ?? fallback.CPU ?? 0
    const memory =
      value.memory_gb ??
      value.mem_gb ??
      value.MemoryGB ??
      fallback.memory_gb ??
      fallback.MemoryGB ??
      0
    const disk = value.disk_gb ?? value.DiskGB ?? fallback.disk_gb ?? fallback.DiskGB ?? 0
    const bandwidth =
      value.bandwidth_mbps ??
      value.BandwidthMB ??
      value.bandwidth ??
      fallback.bandwidth_mbps ??
      fallback.BandwidthMB
    const result = [`CPU ${cpu}核`, `内存 ${memory}G`, `磁盘 ${disk}G`]
    if (bandwidth != null) result.push(`带宽 ${bandwidth}M`)
    return result.join(' / ')
  }

  const statusFromAutomation = (state: unknown) => {
    const map: Record<number, string> = {
      1: 'provisioning',
      13: 'provisioning',
      2: 'running',
      3: 'stopped',
      4: 'reinstalling',
      5: 'reinstall_failed',
      10: 'locked',
      11: 'failed',
      12: 'deleting'
    }
    return map[Number(state)] || ''
  }

  const normalizeRecord = (record: VPSInstance): VpsRecord => {
    const row = record as unknown as VpsApiRecord
    const access = parseJson(
      row.access_info ?? row.AccessInfo ?? row.access_info_json ?? row.AccessInfoJSON
    )
    const expireValue = row.expire_at ?? row.ExpireAt
    const expireAt = expireValue == null ? undefined : String(expireValue)
    const automationState = row.automation_state ?? row.AutomationState
    const rawStatus =
      automationState == null
        ? String(row.status ?? row.Status ?? '')
        : statusFromAutomation(automationState)
    const status =
      isExpired({ expire_at: expireAt }) && ['locked', 'expired_locked'].includes(rawStatus)
        ? 'expired_locked'
        : rawStatus
    const region = String(row.region ?? row.Region ?? '-')
    const line = String(row.line ?? row.Line ?? row.line_name ?? row.LineName ?? access.line ?? '')
    return {
      id: row.id ?? row.ID ?? '',
      name: row.name ?? row.Name,
      regionLine: line ? `${region}/${line}` : region,
      status,
      expire_at: expireAt,
      destroy_in_days:
        row.destroy_in_days == null && row.DestroyInDays == null
          ? undefined
          : Number(row.destroy_in_days ?? row.DestroyInDays),
      last_emergency_renew_at: row.last_emergency_renew_at ?? row.LastEmergencyRenewAt ?? undefined,
      package_id: row.package_id ?? row.PackageID,
      package_name: row.package_name,
      spec_raw: row.spec ?? row.Spec ?? row.spec_json ?? row.SpecJSON,
      specText: normalizeSpec(row.spec ?? row.Spec ?? row.spec_json ?? row.SpecJSON, row),
      ip: String(
        access.remote_ip ?? access.ip ?? access.public_ip ?? access.ipv4 ?? access.Ip ?? '-'
      ),
      monthly_price: Number(row.monthly_price ?? row.MonthlyPrice ?? 0),
      cpu: row.cpu,
      CPU: row.CPU,
      cores: row.cores,
      Cores: row.Cores,
      memory_gb: row.memory_gb,
      mem_gb: row.mem_gb,
      MemoryGB: row.MemoryGB,
      disk_gb: row.disk_gb,
      DiskGB: row.DiskGB,
      bandwidth_mbps: row.bandwidth_mbps,
      bandwidth: row.bandwidth,
      BandwidthMB: row.BandwidthMB,
      ExpireAt: row.ExpireAt
    }
  }

  const filterRows = (rows: VpsRecord[], params: VpsTableParams) => {
    let filtered = rows
    const keyword = String(params.keyword || '')
      .trim()
      .toLowerCase()
    if (keyword) {
      filtered = filtered.filter((item) =>
        [item.id, item.name, item.ip].some((value) =>
          String(value || '')
            .toLowerCase()
            .includes(keyword)
        )
      )
    }
    if (params.status) filtered = filtered.filter((item) => item.status === params.status)
    if (params.region)
      filtered = filtered.filter((item) =>
        String(item.regionLine || '').includes(String(params.region))
      )
    if (params.expire_days) {
      const days = Number(params.expire_days)
      filtered = filtered.filter((item) => {
        const expire = new Date(item.expire_at).getTime()
        const diff = Math.ceil((expire - Date.now()) / (24 * 3600 * 1000))
        return !Number.isNaN(expire) && diff <= days
      })
    }
    return filtered
  }

  // The current endpoint returns the complete user list. useTable still owns all request,
  // search, refresh, and pagination state; this adapter only normalizes and slices that response.
  const fetchVpsTable = async (
    params: VpsTableParams
  ): Promise<Api.Common.PaginatedResponse<VpsRecord>> => {
    const response = await listVps()
    const rows = (response.data?.items || []).map(normalizeRecord)
    allVps.value = rows
    const filtered = filterRows(rows, params)
    const start = (params.current - 1) * params.size
    return {
      records: filtered.slice(start, start + params.size),
      current: params.current,
      size: params.size,
      total: filtered.length
    }
  }

  const {
    columns,
    columnChecks,
    data,
    loading,
    pagination,
    getData,
    searchParams,
    resetSearchParams,
    handleSizeChange,
    handleCurrentChange,
    refreshData
  } = useTable({
    core: {
      apiFn: fetchVpsTable,
      apiParams: { current: 1, size: 10 },
      columnsFactory: () => [
        { type: 'selection', width: 48 },
        { type: 'globalIndex', width: 62, label: '序号' },
        { prop: 'name', label: '实例', minWidth: 190, useSlot: true },
        { prop: 'regionLine', label: '地区/线路', minWidth: 140, useSlot: true },
        { prop: 'spec', label: '配置', minWidth: 230, useSlot: true },
        { prop: 'ip', label: 'IP 地址', minWidth: 140, useSlot: true },
        { prop: 'status', label: '状态', width: 112, useSlot: true },
        { prop: 'expire_at', label: '到期时间', minWidth: 180, useSlot: true },
        { prop: 'operation', label: '操作', width: 180, fixed: 'right', useSlot: true }
      ]
    }
  })

  const stats = computed(() => {
    const byStatus = (status: string) =>
      allVps.value.filter((item) => item.status === status).length
    return [
      { label: '运行中', value: byStatus('running'), icon: SwitchButton, tone: 'success' },
      { label: '已关机', value: byStatus('stopped'), icon: Monitor, tone: 'muted' },
      { label: '开通中', value: byStatus('provisioning'), icon: Loading, tone: 'primary' },
      {
        label: '即将到期',
        value: allVps.value.filter((item) => isExpiring(item.expire_at)).length,
        icon: Warning,
        tone: 'warning'
      }
    ]
  })

  const handleSearch = (params: Partial<VpsTableParams>) => {
    Object.assign(searchParams, params)
    getData()
  }

  const handleStatusTab = (value: string | number | boolean) => {
    const status = String(value)
    searchForm.value.status = status
    Object.assign(searchParams, { status })
    getData()
  }

  const handleReset = () => {
    Object.assign(searchForm.value, { keyword: '', status: '', region: '', expire_days: undefined })
    resetSearchParams()
  }

  const formatDateTime = (value: unknown) => {
    if (!value) return '-'
    const date = new Date(String(value))
    return Number.isNaN(date.getTime())
      ? String(value)
      : date.toLocaleString('zh-CN', { hour12: false })
  }

  const isExpiring = (value: unknown) => {
    if (!value) return false
    const time = new Date(String(value)).getTime()
    const days = Math.ceil((time - Date.now()) / (24 * 3600 * 1000))
    return !Number.isNaN(time) && days <= 7 && days > 0
  }

  const isExpired = (row: { expire_at?: unknown; ExpireAt?: unknown } | null) => {
    const time = new Date(String(row?.expire_at ?? row?.ExpireAt ?? '')).getTime()
    return !Number.isNaN(time) && time <= Date.now()
  }

  const getErrorText = (error: unknown, fallback: string) => {
    if (!isJsonObject(error)) return fallback
    const response = isJsonObject(error.response) ? error.response : {}
    const responseData = isJsonObject(response.data) ? response.data : {}
    return String(responseData.error ?? responseData.message ?? error.message ?? fallback)
  }
  const getErrorStatus = (error: unknown) => {
    if (!isJsonObject(error) || !isJsonObject(error.response)) return undefined
    const status = Number(error.response.status)
    return Number.isFinite(status) ? status : undefined
  }
  const getIdentifier = (value: unknown): Identifier | undefined =>
    typeof value === 'string' || typeof value === 'number' ? value : undefined
  const getOrderId = (value: unknown) => {
    if (!isJsonObject(value)) return undefined
    const order = isJsonObject(value.order) ? value.order : {}
    return getIdentifier(order.id ?? order.ID ?? value.order_id ?? value.orderId ?? value.id)
  }
  const showOrderConflict = async (title: string, error: unknown, orderId?: Identifier) => {
    try {
      await ElMessageBox.confirm(getErrorText(error, title), title, {
        type: 'warning',
        confirmButtonText: orderId ? '去订单详情' : '去订单列表',
        cancelButtonText: '我知道了'
      })
      await router.push(orderId ? `/console/orders/${orderId}` : '/console/orders')
    } catch (action) {
      if (action !== 'cancel' && action !== 'close') throw action
    }
  }
  const goDetail = (record: VpsRecord) => router.push(`/console/vps/${record.id}`)
  const goBuy = () => router.push({ name: 'PublicBuy' })
  const openExternal = async (record: VpsRecord, type: 'panel' | 'vnc') => {
    const targetWindow = window.open('about:blank', '_blank')
    if (!targetWindow) {
      ElMessage.error('浏览器阻止了新窗口，请允许本站打开弹窗后重试')
      return
    }
    targetWindow.opener = null

    try {
      const response =
        type === 'panel' ? await getVpsPanelUrl(record.id) : await getVpsVncUrl(record.id)
      const targetUrl = String(response.data?.url || '').trim()
      if (!targetUrl) {
        targetWindow.close()
        ElMessage.error(type === 'panel' ? '面板地址为空' : 'VNC 地址为空')
        return
      }
      targetWindow.location.replace(targetUrl)
    } catch {
      targetWindow.close()
    }
  }
  const openPanel = (record: VpsRecord) => openExternal(record, 'panel')
  const openVnc = (record: VpsRecord) => openExternal(record, 'vnc')

  const exportCsv = () => {
    const rows = filterRows(allVps.value, searchParams)
    const lines = [
      'id,name,status,expire_at',
      ...rows.map((item) =>
        [item.id, item.name, item.status, item.expire_at]
          .map((value) => JSON.stringify(value ?? ''))
          .join(',')
      )
    ]
    const blob = new Blob([lines.join('\n')], { type: 'text/csv;charset=utf-8;' })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = 'vps.csv'
    link.click()
    URL.revokeObjectURL(url)
  }

  const runAction = async (
    record: VpsRecord,
    action: (id: number | string) => Promise<unknown>,
    label: string,
    successMessage: string
  ) => {
    try {
      await action(record.id)
      ElMessage.success(successMessage)
      await refreshData()
    } catch (error) {
      ElMessage.error(getErrorText(error, `${label}失败`))
    }
  }

  const refresh = async (record: VpsRecord) => {
    await runAction(
      record,
      async (id) => {
        const result = await refreshVps(id)
        return result
      },
      '刷新状态',
      '已刷新'
    )
  }
  const start = (record: VpsRecord) => runAction(record, startVps, '开机', '已触发开机')
  const shutdown = (record: VpsRecord) => runAction(record, shutdownVps, '关机', '已触发关机')
  const reboot = (record: VpsRecord) => runAction(record, rebootVps, '重启', '已触发重启')

  const renewOpen = ref(false)
  const resizeOpen = ref(false)
  const refundOpen = ref(false)
  const renewing = ref(false)
  const resizing = ref(false)
  const refunding = ref(false)
  const resizeQuote = ref<JsonObject | null>(null)
  const resizeQuoteLoading = ref(false)
  const resizeQuoteError = ref('')
  const activeRecord = ref<VpsRecord | null>(null)
  const renewForm = reactive({ months: 1 })
  const resizeForm = reactive<ResizeForm>({
    add_cores: 0,
    add_mem_gb: 0,
    add_disk_gb: 0,
    add_bw_mbps: 0,
    target_package_id: null,
    reset_addons: false,
    schedule_mode: 'now',
    scheduled_at: null
  })
  const refundReason = ref('')
  const addonFields = [
    { key: 'add_cores', label: 'CPU 附加' },
    { key: 'add_mem_gb', label: '内存附加 GB' },
    { key: 'add_disk_gb', label: '磁盘附加 GB' },
    { key: 'add_bw_mbps', label: '带宽附加 Mbps' }
  ] as const

  const getSettingBool = (key: string) => {
    const raw = site.settings?.[key]
    if (raw === undefined || raw === null || raw === '') return undefined
    if (typeof raw === 'boolean') return raw
    if (['true', '1', 'yes', 'on'].includes(String(raw).toLowerCase())) return true
    if (['false', '0', 'no', 'off'].includes(String(raw).toLowerCase())) return false
    return undefined
  }
  const emergencyRenewPolicy = computed(() => ({
    enabled: getSettingBool('emergency_renew_enabled') !== false,
    windowDays: Math.max(
      0,
      Number.parseInt(String(site.settings?.emergency_renew_window_days ?? 7), 10) || 7
    ),
    intervalHours: Math.max(
      24,
      Number.parseInt(String(site.settings?.emergency_renew_interval_hours ?? 720), 10) || 720
    )
  }))
  const emergencyRenewEligible = (record: VpsRecord | null) => {
    if (!record?.expire_at || !emergencyRenewPolicy.value.enabled || isExpired(record)) return false
    const expire = dayjs(record.expire_at)
    if (dayjs().isBefore(expire.subtract(emergencyRenewPolicy.value.windowDays, 'day')))
      return false
    if (
      record.last_emergency_renew_at &&
      dayjs().diff(dayjs(record.last_emergency_renew_at), 'hour', true) <
        emergencyRenewPolicy.value.intervalHours
    )
      return false
    return true
  }
  const submitEmergencyRenew = (record: VpsRecord) => {
    ElMessageBox.confirm('紧急续费将按系统策略续费固定天数，确认继续？', '紧急续费确认', {
      type: 'warning'
    })
      .then(async () => {
        await emergencyRenewVps(record.id)
        ElMessage.success('紧急续费已提交')
        await refreshData()
      })
      .catch((error) => {
        if (error !== 'cancel' && error !== 'close')
          ElMessage.error(getErrorText(error, '紧急续费失败'))
      })
  }

  const currentAddons = computed(() => {
    const spec = parseJson(activeRecord.value?.spec_raw)
    return {
      add_cores: Number(spec.add_cores ?? spec.AddCores ?? 0),
      add_mem_gb: Number(spec.add_mem_gb ?? spec.AddMemGB ?? 0),
      add_disk_gb: Number(spec.add_disk_gb ?? spec.AddDiskGB ?? 0),
      add_bw_mbps: Number(spec.add_bw_mbps ?? spec.AddBWMbps ?? 0)
    }
  })
  const catalogPackages = computed(() => catalog.packages as CatalogPackage[])
  const catalogPlanGroups = computed(() => catalog.planGroups as CatalogPlanGroup[])
  const getCatalogId = (item: { id?: Identifier; ID?: Identifier }) => item.id ?? item.ID ?? ''
  const getPackageMonthlyPrice = (item: CatalogPackage) =>
    Number(item.monthly_price ?? item.MonthlyPrice ?? item.Monthly ?? 0)
  const currentPackage = computed(
    () =>
      catalogPackages.value.find(
        (item) => String(getCatalogId(item)) === String(activeRecord.value?.package_id)
      ) || null
  )
  const currentPlanGroup = computed(() => {
    if (!currentPackage.value) return null
    const groupId =
      currentPackage.value.plan_group_id ??
      currentPackage.value.planGroupId ??
      currentPackage.value.PlanGroupID
    return (
      catalogPlanGroups.value.find((item) => String(getCatalogId(item)) === String(groupId)) || null
    )
  })
  const packageOptions = computed(() => {
    if (!currentPlanGroup.value) return []
    const groupId = currentPlanGroup.value.id ?? currentPlanGroup.value.ID
    return catalogPackages.value
      .filter((item) => {
        const packageGroup = item.plan_group_id ?? item.planGroupId ?? item.PlanGroupID
        return (
          String(packageGroup) === String(groupId) &&
          (item.active ?? item.Active) !== false &&
          (item.visible ?? item.Visible) !== false
        )
      })
      .sort((a, b) => getPackageMonthlyPrice(a) - getPackageMonthlyPrice(b))
  })
  const resizeEnabled = computed(() => getSettingBool('resize_enabled') !== false)
  const normalizePackageSpec = (pkg: PackageSpecSource | null) => ({
    cpu: Number(pkg?.cores ?? pkg?.cpu ?? pkg?.CPU ?? pkg?.Cores ?? 0),
    memory_gb: Number(pkg?.memory_gb ?? pkg?.mem_gb ?? pkg?.MemoryGB ?? 0),
    disk_gb: Number(pkg?.disk_gb ?? pkg?.DiskGB ?? 0),
    bandwidth_mbps: Number(pkg?.bandwidth_mbps ?? pkg?.BandwidthMB ?? pkg?.bandwidth ?? 0)
  })
  const currentSpecForCompare = computed(() => {
    const fallback = normalizePackageSpec(activeRecord.value)
    if (!currentPackage.value) return fallback
    const fromPackage = normalizePackageSpec(currentPackage.value)
    return Object.values(fromPackage).some((value) => value > 0) ? fromPackage : fallback
  })
  const isSamePackageOption = (pkg: CatalogPackage) => {
    const target = normalizePackageSpec(pkg)
    const current = currentSpecForCompare.value
    const packageProductId = pkg.product_id ?? pkg.ProductID
    const currentProductId = currentPackage.value?.product_id ?? currentPackage.value?.ProductID
    return (
      String(getCatalogId(pkg)) === String(getCatalogId(currentPackage.value || {})) ||
      Boolean(
        packageProductId &&
        currentProductId &&
        String(packageProductId) === String(currentProductId)
      ) ||
      (target.cpu === current.cpu &&
        target.memory_gb === current.memory_gb &&
        target.disk_gb === current.disk_gb &&
        target.bandwidth_mbps === current.bandwidth_mbps)
    )
  }
  const isSameAddonsSelection = computed(() =>
    addonFields.every(
      ({ key }) => Number(resizeForm[key] || 0) === Number(currentAddons.value[key] || 0)
    )
  )
  const isSameTargetSelection = computed(() => {
    const target = packageOptions.value.find(
      (item) => String(getCatalogId(item)) === String(resizeForm.target_package_id)
    )
    return Boolean(target && isSamePackageOption(target) && isSameAddonsSelection.value)
  })
  const addonMin = computed(() => ({ add_cores: 0, add_mem_gb: 0, add_disk_gb: 0, add_bw_mbps: 0 }))
  const addonMax = computed(() => ({
    add_cores: currentPlanGroup.value?.add_core_max ?? 64,
    add_mem_gb: currentPlanGroup.value?.add_mem_max ?? 256,
    add_disk_gb: currentPlanGroup.value?.add_disk_max ?? 2000,
    add_bw_mbps: currentPlanGroup.value?.add_bw_max ?? 1000
  }))
  const addonStep = computed(() => ({
    add_cores: currentPlanGroup.value?.add_core_step ?? 1,
    add_mem_gb: currentPlanGroup.value?.add_mem_step ?? 1,
    add_disk_gb: currentPlanGroup.value?.add_disk_step ?? 10,
    add_bw_mbps: currentPlanGroup.value?.add_bw_step ?? 10
  }))

  const openRenew = (record: VpsRecord) => {
    activeRecord.value = record
    renewForm.months = 1
    renewOpen.value = true
  }
  const submitRenew = async () => {
    if (!activeRecord.value) return
    renewing.value = true
    try {
      await createVpsRenewOrder(activeRecord.value.id, { duration_months: renewForm.months })
      ElMessage.success('已生成续费订单')
      renewOpen.value = false
    } catch (error) {
      if (getErrorStatus(error) === 409) {
        await showOrderConflict('已有待处理续费订单', error)
        return
      }
      ElMessage.error(getErrorText(error, '续费失败'))
    } finally {
      renewing.value = false
    }
  }
  const openResize = (record: VpsRecord) => {
    if (isExpired(record)) return ElMessage.warning('已到期实例不支持升降配')
    activeRecord.value = record
    Object.assign(resizeForm, currentAddons.value, {
      target_package_id: currentPackage.value ? getCatalogId(currentPackage.value) : null,
      reset_addons: false,
      schedule_mode: 'now',
      scheduled_at: null
    })
    resizeQuote.value = null
    resizeQuoteError.value = ''
    resizeOpen.value = true
  }
  const resizeQuoteAmount = computed(() =>
    Number(resizeQuote.value?.charge_amount ?? resizeQuote.value?.chargeAmount ?? 0)
  )
  const buildResizePayload = () => ({
    target_package_id: resizeForm.target_package_id,
    reset_addons: resizeForm.reset_addons,
    spec: resizeForm.reset_addons
      ? { add_cores: 0, add_mem_gb: 0, add_disk_gb: 0, add_bw_mbps: 0 }
      : Object.fromEntries(addonFields.map(({ key }) => [key, resizeForm[key]])),
    ...(resizeForm.schedule_mode === 'later' && resizeForm.scheduled_at
      ? { scheduled_at: dayjs(resizeForm.scheduled_at).toISOString() }
      : {})
  })
  const fetchResizeQuote = async () => {
    if (
      !activeRecord.value ||
      !resizeEnabled.value ||
      !resizeForm.target_package_id ||
      isSameTargetSelection.value
    ) {
      resizeQuote.value = null
      resizeQuoteError.value = ''
      return
    }
    resizeQuoteLoading.value = true
    try {
      const response = await quoteVpsResizeOrder(activeRecord.value.id, buildResizePayload())
      const responseData: unknown = response.data
      const responseObject = isJsonObject(responseData) ? responseData : {}
      resizeQuote.value = isJsonObject(responseObject.quote) ? responseObject.quote : responseObject
    } catch (error) {
      resizeQuote.value = null
      resizeQuoteError.value =
        getErrorStatus(error) === 409
          ? '已有进行中的升降配任务/订单'
          : getErrorText(error, '升降配报价失败')
    } finally {
      resizeQuoteLoading.value = false
    }
  }
  const resizeQuoteTitle = computed(() =>
    resizeQuoteAmount.value < 0
      ? `本周期差额：-￥${Math.abs(resizeQuoteAmount.value).toFixed(2)}`
      : `本周期需支付：￥${resizeQuoteAmount.value.toFixed(2)}`
  )
  const resizeQuoteDescription = computed(() =>
    resizeQuoteAmount.value < 0 ? '退款方式以支付渠道为准。' : '金额以系统最终计算为准。'
  )
  let resizeQuoteTimer: ReturnType<typeof setTimeout> | null = null
  const scheduleResizeQuote = () => {
    if (resizeQuoteTimer) clearTimeout(resizeQuoteTimer)
    resizeQuoteTimer = setTimeout(fetchResizeQuote, 300)
  }
  const submitResize = async () => {
    if (
      !activeRecord.value ||
      isExpired(activeRecord.value) ||
      !resizeEnabled.value ||
      !resizeForm.target_package_id ||
      isSameTargetSelection.value
    )
      return ElMessage.warning('请检查升降配选项')
    if (
      resizeForm.schedule_mode === 'later' &&
      (!resizeForm.scheduled_at || dayjs(resizeForm.scheduled_at).isBefore(dayjs()))
    )
      return ElMessage.warning('请选择晚于当前时间的执行时间')
    resizing.value = true
    try {
      const response = await createVpsResizeOrder(activeRecord.value.id, buildResizePayload())
      ElMessage.success('已生成改配订单')
      resizeOpen.value = false
      const orderId = getOrderId(response.data)
      if (orderId) await router.push(`/console/orders/${orderId}`)
    } catch (error) {
      if (getErrorStatus(error) === 409) {
        const response = isJsonObject(error) && isJsonObject(error.response) ? error.response : {}
        await showOrderConflict('已有进行中的升降配任务/订单', error, getOrderId(response.data))
        return
      }
      ElMessage.error(getErrorText(error, '升降配失败'))
    } finally {
      resizing.value = false
    }
  }
  const openRefund = (record: VpsRecord) => {
    activeRecord.value = record
    refundReason.value = ''
    refundOpen.value = true
  }
  const submitRefund = async () => {
    if (!activeRecord.value || !refundReason.value.trim())
      return ElMessage.warning('请填写退款原因')
    if (refundReason.value.length > INPUT_LIMITS.REFUND_REASON)
      return ElMessage.warning(`退款原因长度不能超过 ${INPUT_LIMITS.REFUND_REASON} 个字符`)
    refunding.value = true
    try {
      const response = await requestVpsRefund(activeRecord.value.id, {
        reason: refundReason.value.trim()
      })
      const orderId = getOrderId(response.data)
      ElMessage.success(orderId ? `已提交退款申请，订单ID: ${orderId}` : '已提交退款申请')
      refundOpen.value = false
    } catch (error) {
      ElMessage.error(getErrorText(error, '提交失败'))
    } finally {
      refunding.value = false
    }
  }

  const handleCommand = (command: string, record: VpsRecord) => {
    const actions: Record<string, () => unknown> = {
      vnc: () => openVnc(record),
      start: () => start(record),
      shutdown: () => shutdown(record),
      reboot: () => reboot(record),
      refresh: () => refresh(record),
      renew: () => openRenew(record),
      'urgent-renew': () => submitEmergencyRenew(record),
      resize: () => openResize(record),
      refund: () => openRefund(record)
    }
    actions[command]?.()
  }
  const mobileActionsOpen = ref(false)
  const mobileActionRecord = ref<VpsRecord | null>(null)
  const mobileActions = computed(() => [
    { key: 'detail', label: '详情', icon: View },
    { key: 'panel', label: '面板', icon: Monitor },
    { key: 'vnc', label: 'VNC', icon: VideoCamera },
    { key: 'start', label: '开机', icon: SwitchButton },
    { key: 'shutdown', label: '关机', icon: Loading },
    { key: 'reboot', label: '重启', icon: RefreshRight },
    { key: 'renew', label: '续费', icon: Calendar },
    ...(emergencyRenewEligible(mobileActionRecord.value)
      ? [{ key: 'urgent-renew', label: '紧急续费', icon: Calendar }]
      : []),
    { key: 'resize', label: '升降配', icon: Cpu },
    { key: 'refresh', label: '刷新', icon: Refresh },
    { key: 'refund', label: '退款', icon: Warning }
  ])
  const showMobileActions = (record: VpsRecord) => {
    mobileActionRecord.value = record
    mobileActionsOpen.value = true
  }
  const handleMobileAction = (action: string) => {
    const record = mobileActionRecord.value
    mobileActionsOpen.value = false
    if (!record) return
    if (action === 'detail') return goDetail(record)
    if (action === 'panel') return openPanel(record)
    handleCommand(action, record)
  }

  watch(
    () => [
      resizeForm.target_package_id,
      resizeForm.add_cores,
      resizeForm.add_mem_gb,
      resizeForm.add_disk_gb,
      resizeForm.add_bw_mbps,
      resizeForm.reset_addons
    ],
    () => {
      if (resizeOpen.value) scheduleResizeQuote()
    }
  )
  watch(resizeOpen, (open) => {
    if (!open && resizeQuoteTimer) clearTimeout(resizeQuoteTimer)
  })

  onMounted(() => {
    catalog.fetchCatalog().catch(() => undefined)
    site.fetchSettings().catch(() => undefined)
  })
</script>

<style scoped lang="scss">
  .vps-list-page {
    gap: 16px;
    overflow: auto;
  }

  .page-heading {
    display: flex;
    gap: 20px;
    align-items: flex-start;
    justify-content: space-between;
    padding: 4px 0 0;

    h1 {
      margin: 4px 0 6px;
      font-size: 24px;
      font-weight: 700;
      color: var(--art-gray-900);
    }

    p {
      margin: 0;
      font-size: 13px;
      color: var(--art-gray-600);
    }
  }

  .eyebrow {
    font-size: 12px;
    font-weight: 600;
    color: var(--theme-color);
    letter-spacing: 0;
  }

  .heading-actions {
    display: flex;
    gap: 8px;
  }

  .stats-grid {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 12px;
  }

  .status-tabs {
    overflow-x: auto;
  }

  .stat-card {
    display: flex;
    gap: 12px;
    align-items: center;
    min-height: 88px;
    background: var(--default-box-color);
    border: 1px solid var(--art-card-border);
  }

  .stat-icon {
    display: grid;
    flex: 0 0 36px;
    place-items: center;
    width: 36px;
    height: 36px;
    border-radius: 10px;
  }

  .stat-icon.is-success {
    color: var(--el-color-success);
    background: var(--el-color-success-light-9);
  }

  .stat-icon.is-muted {
    color: var(--art-gray-600);
    background: var(--art-gray-200);
  }

  .stat-icon.is-primary {
    color: var(--theme-color);
    background: var(--el-color-primary-light-9);
  }

  .stat-icon.is-warning {
    color: var(--el-color-warning);
    background: var(--el-color-warning-light-9);
  }

  .stat-value {
    font-size: 22px;
    font-weight: 700;
    line-height: 1.2;
    color: var(--art-gray-900);
  }

  .stat-label,
  .muted-text,
  .name-id {
    font-size: 12px;
    color: var(--art-gray-600);
  }

  .table-card {
    min-height: 0;
  }

  .name-title {
    overflow: hidden;
    font-weight: 600;
    color: var(--art-gray-900);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .name-cell {
    min-width: 0;
  }

  .mono {
    font-family: var(--el-font-family-monospace);
  }

  .is-expiring {
    font-weight: 600;
    color: var(--el-color-danger);
  }

  .mobile-list {
    display: none;
  }

  .dialog-form {
    margin-top: 16px;
  }

  .full-width {
    width: 100%;
  }

  .action-grid {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 12px;
  }

  .action-grid .el-button {
    display: flex;
    flex-direction: column;
    gap: 6px;
    height: 64px;
  }

  @media (width <= 768px) {
    .vps-list-page {
      height: auto;
      overflow: visible;
    }

    .page-heading {
      flex-direction: column;
    }

    .heading-actions {
      width: 100%;
    }

    .heading-actions .el-button {
      flex: 1;
    }

    .stats-grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }

    .table-card {
      display: none;
    }

    .mobile-list {
      display: flex;
      flex-direction: column;
      gap: 10px;
    }

    .mobile-list-heading {
      display: flex;
      align-items: center;
      justify-content: space-between;
      font-weight: 600;
      color: var(--art-gray-900);
    }

    .mobile-vps-card {
      cursor: pointer;
    }

    .mobile-card-heading {
      display: flex;
      gap: 12px;
      align-items: flex-start;
      justify-content: space-between;
    }

    .mobile-details {
      display: grid;
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: 8px;
      margin-top: 16px;
      font-size: 12px;
      color: var(--art-gray-700);
    }

    .mobile-details span {
      display: flex;
      gap: 5px;
      align-items: center;
      min-width: 0;
    }

    .mobile-details .el-icon {
      color: var(--art-gray-500);
    }

    .mobile-destroy-warning {
      display: flex;
      gap: 6px;
      align-items: center;
      padding: 8px 10px;
      margin-top: 12px;
      font-size: 12px;
      color: var(--el-color-danger);
      background: var(--el-color-danger-light-9);
      border: 1px solid var(--el-color-danger-light-7);
      border-radius: 6px;
    }

    .mobile-card-actions {
      display: flex;
      flex-wrap: wrap;
      gap: 2px;
      justify-content: flex-end;
      padding-top: 10px;
      margin-top: 14px;
      border-top: 1px solid var(--default-border);
    }

    .mobile-pagination {
      justify-content: center;
    }
  }
</style>
