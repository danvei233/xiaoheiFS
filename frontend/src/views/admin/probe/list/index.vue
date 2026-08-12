<template>
  <div class="probe-list-page art-full-height">
    <ProbeSearch
      v-show="showSearchBar"
      v-model="searchForm"
      @search="handleSearch"
      @reset="handleReset"
    />

    <ElCard
      v-if="canViewSettings"
      class="settings-card"
      :style="{ marginTop: showSearchBar ? '12px' : '0' }"
    >
      <div class="settings-head">
        <div>
          <div class="settings-title">探针心跳设置</div>
          <div class="settings-tip">用于控制探针在线判定与心跳容忍时间。</div>
        </div>

        <ElSpace wrap>
          <ElButton :loading="settingsLoading" @click="loadSettings">刷新设置</ElButton>
          <ElButton
            type="primary"
            v-auth="'settings.update'"
            :loading="settingsSaving"
            :disabled="!canUpdateSettings"
            @click="saveSettings"
          >
            保存设置
          </ElButton>
        </ElSpace>
      </div>

      <ElForm label-position="top">
        <ElRow :gutter="16">
          <ElCol :xs="24" :md="12" :lg="8">
            <ElFormItem label="心跳间隔（秒）" class="settings-item">
              <ElInputNumber
                v-model="settingsForm.heartbeat_interval_sec"
                :min="5"
                :max="300"
                :disabled="!canUpdateSettings"
                style="width: 100%"
              />
            </ElFormItem>
          </ElCol>

          <ElCol :xs="24" :md="12" :lg="8">
            <ElFormItem label="离线宽限（秒）" class="settings-item">
              <ElInputNumber
                v-model="settingsForm.offline_grace_sec"
                :min="15"
                :max="1800"
                :disabled="!canUpdateSettings"
                style="width: 100%"
              />
            </ElFormItem>
          </ElCol>
        </ElRow>
      </ElForm>
    </ElCard>

    <ElCard
      class="art-table-card"
      :style="{ marginTop: canViewSettings || showSearchBar ? '12px' : '0' }"
    >
      <div class="refresh-status">
        <span>上次刷新：{{ formatDateTime(lastRefreshAt) }}</span>
        <span v-if="refreshError">，刷新失败：{{ refreshError }}</span>
      </div>
      <ArtTableHeader
        v-model:columns="columnChecks"
        v-model:showSearchBar="showSearchBar"
        :loading="loading"
        @refresh="refreshTable"
      >
        <template #left>
          <ElSpace wrap>
            <ElButton v-auth="'probe.create'" type="primary" v-ripple @click="openCreate">
              创建探针
            </ElButton>
          </ElSpace>
        </template>
      </ArtTableHeader>

      <ArtTable
        row-key="id"
        :loading="loading"
        :data="tableData"
        :columns="columns"
        :pagination="pagination"
        @pagination:size-change="handleSizeChange"
        @pagination:current-change="handleCurrentChange"
      >
        <template #status="{ row }">
          <ElTag :type="getStatusTagType(row.status)">
            {{ getStatusText(row.status) }}
          </ElTag>
        </template>

        <template #cpu_usage="{ row }">
          <div class="usage-cell">
            <template v-if="getUsagePercent(row, 'cpu') !== null">
              <ElProgress
                :percentage="getUsagePercent(row, 'cpu') || 0"
                :stroke-width="8"
                :show-text="false"
                :color="getUsageColor(getUsagePercent(row, 'cpu'))"
              />
              <span>{{ formatUsage(getUsagePercent(row, 'cpu')) }}</span>
            </template>
            <span v-else>-</span>
          </div>
        </template>

        <template #mem_usage="{ row }">
          <div class="usage-cell">
            <template v-if="getUsagePercent(row, 'memory') !== null">
              <ElProgress
                :percentage="getUsagePercent(row, 'memory') || 0"
                :stroke-width="8"
                :show-text="false"
                :color="getUsageColor(getUsagePercent(row, 'memory'))"
              />
              <span>{{ formatUsage(getUsagePercent(row, 'memory')) }}</span>
            </template>
            <span v-else>-</span>
          </div>
        </template>

        <template #tags="{ row }">
          <div class="tag-list">
            <ElTag v-for="tag in row.tags" :key="tag" effect="plain">{{ tag }}</ElTag>
            <span v-if="!row.tags.length">-</span>
          </div>
        </template>

        <template #sla="{ row }">
          <template v-if="getSlaPercent(row.id) !== null">
            <ElTag :type="getSlaTagType(getSlaPercent(row.id))">
              {{ formatSla(getSlaPercent(row.id)) }}
            </ElTag>
          </template>
          <span v-else>-</span>
        </template>

        <template #operation="{ row }">
          <div class="table-actions">
            <ArtButtonTable v-auth="'probe.view'" type="view" @click="openDetail(row)" />
            <ArtButtonTable v-auth="'probe.update'" type="edit" @click="openEdit(row)" />

            <ArtButtonMore
              :list="getMoreActions()"
              @click="(item) => handleMoreAction(item, row)"
            />
          </div>
        </template>
      </ArtTable>
    </ElCard>

    <ProbeDialog
      v-model:visible="dialogVisible"
      :mode="dialogMode"
      :form-data="dialogForm"
      :submitting="dialogSubmitting"
      @submit="handleDialogSubmit"
    />

    <ElDialog
      v-model="tokenVisible"
      title="探针注册令牌"
      width="620px"
      destroy-on-close
      align-center
    >
      <ElAlert
        type="warning"
        :closable="false"
        show-icon
        title="令牌只会显示一次，请尽快复制并交给探针端使用。"
      />

      <div class="token-box">{{ currentToken || '-' }}</div>

      <template #footer>
        <div class="dialog-footer">
          <ElButton @click="tokenVisible = false">关闭</ElButton>
          <ElButton type="primary" :disabled="!currentToken" @click="copyCurrentToken">
            复制令牌
          </ElButton>
        </div>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import type {
    ProbeNode as ProbeRecord,
    ProbeSnapshot as ProbeSnapshotRecord
  } from '@/services/types'
  import type { ButtonMoreItem } from '@/components/core/forms/art-button-more/index.vue'
  import {
    createAdminProbe,
    deleteAdminProbe,
    getAdminProbeSla,
    listAdminProbes,
    listSettings,
    resetAdminProbeEnrollToken,
    updateAdminProbe,
    updateSetting
  } from '@/services/admin'
  import ArtButtonMore from '@/components/core/forms/art-button-more/index.vue'
  import ArtButtonTable from '@/components/core/forms/art-button-table/index.vue'
  import { useAuth } from '@/hooks/core/useAuth'
  import { useTable } from '@/hooks/core/useTable'
  import { useLifecycleSafeTimer } from '@/hooks/core/useLifecycleSafeTimer'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import ProbeDialog from './modules/probe-dialog.vue'
  import ProbeSearch from './modules/probe-search.vue'

  defineOptions({ name: 'ProbeList' })

  const fetchAdminProbes = async (params: Record<string, unknown> = {}, signal?: AbortSignal) =>
    (await listAdminProbes(params, signal)).data
  const fetchAdminProbeSla = async (
    id: number | string,
    params: Record<string, unknown> = {},
    signal?: AbortSignal
  ) => (await getAdminProbeSla(id, params, signal)).data
  const fetchAdminSettings = async () => (await listSettings()).data
  const updateAdminSettings = updateSetting
  const createProbe = async (payload: Record<string, unknown>) =>
    (await createAdminProbe(payload)).data
  const resetProbeEnrollToken = async (id: number | string) =>
    (await resetAdminProbeEnrollToken(id)).data

  interface ProbeSearchForm {
    keyword: string
    status?: string
  }

  interface ProbeTableParams extends Api.Common.CommonSearchParams, ProbeSearchForm {}

  interface ProbeRecordLike extends ProbeRecord {
    cpu_usage_percent?: unknown
    mem_usage_percent?: unknown
    CPUUsagePercent?: unknown
    MemUsagePercent?: unknown
    cpuUsagePercent?: unknown
    memUsagePercent?: unknown
    CpuUsagePercent?: unknown
    MEMUsagePercent?: unknown
    ID?: unknown
    Name?: unknown
    AgentID?: unknown
    Status?: unknown
    OSType?: unknown
    Tags?: unknown
    LastHeartbeatAt?: unknown
    LastSnapshotAt?: unknown
    Snapshot?: ProbeSnapshotRecord
    CreatedAt?: unknown
    UpdatedAt?: unknown
  }

  interface ProbeDialogFormValue {
    id: number | null
    name: string
    agent_id: string
    os_type: string
    tags: string[]
  }

  interface ProbeTableRow {
    id: number | null
    name: string
    agent_id: string
    status: string
    os_type: string
    tags: string[]
    last_heartbeat_at: string | null
    last_snapshot_at: string | null
    snapshot: ProbeSnapshotRecord | null
    cpu_usage_percent: number | null
    mem_usage_percent: number | null
    created_at: string
    updated_at: string
  }

  const route = useRoute()
  const router = useRouter()
  const { hasAuth } = useAuth()

  const showSearchBar = ref(true)
  const dialogVisible = ref(false)
  const dialogMode = ref<'create' | 'edit'>('create')
  const dialogSubmitting = ref(false)
  const tokenVisible = ref(false)
  const currentToken = ref('')
  const settingsLoading = ref(false)
  const settingsSaving = ref(false)
  const lastRefreshAt = ref('')
  const refreshError = ref('')

  const searchForm = ref<ProbeSearchForm>(createDefaultSearchForm())
  const dialogForm = ref<ProbeDialogFormValue>(createDefaultDialogForm())
  const slaMap = reactive<Record<string, number>>({})
  const slaUpdatedAt = new Map<string, number>()

  const SLA_CACHE_MS = 5 * 60 * 1000
  const SLA_CONCURRENCY = 4

  const settingsForm = reactive({
    heartbeat_interval_sec: 20,
    offline_grace_sec: 90
  })

  const canViewSettings = computed(() => hasAuth('settings.view'))
  const canUpdateSettings = computed(() => hasAuth('settings.update'))

  const {
    columns,
    columnChecks,
    data: tableData,
    loading,
    pagination,
    getData,
    searchParams,
    resetSearchParams,
    handleSizeChange,
    handleCurrentChange,
    refreshData,
    refreshSoft
  } = useTable({
    core: {
      apiFn: fetchProbeTable,
      apiParams: { current: 1, size: 20, ...createDefaultSearchForm() },
      immediate: false,
      columnsFactory: () => [
        { prop: 'id', label: 'ID', width: 80 },
        { prop: 'name', label: '探针名称', minWidth: 180, showOverflowTooltip: true },
        { prop: 'agent_id', label: 'Agent ID', minWidth: 180, showOverflowTooltip: true },
        { prop: 'status', label: '状态', width: 100, useSlot: true },
        { prop: 'os_type', label: '系统', width: 110 },
        { prop: 'cpu_usage', label: 'CPU', minWidth: 170, useSlot: true },
        { prop: 'mem_usage', label: '内存', minWidth: 170, useSlot: true },
        { prop: 'tags', label: '标签', minWidth: 220, useSlot: true },
        {
          prop: 'last_heartbeat_at',
          label: '最后心跳',
          minWidth: 180,
          formatter: (row: ProbeTableRow) => formatDateTime(row.last_heartbeat_at)
        },
        { prop: 'sla', label: '7 天 SLA', width: 110, useSlot: true },
        { prop: 'operation', label: '操作', width: 160, fixed: 'right', useSlot: true }
      ]
    },
    hooks: {
      resetFormCallback: () => {
        searchForm.value = createDefaultSearchForm()
      }
    }
  })

  let polling = false
  let pollingStopped = true
  let slaController: AbortController | null = null
  let forceSlaRefresh = false

  // web_refresh_perf F1：keep-alive 页面切后台时组件不卸载、onBeforeUnmount
  // 不触发，裸 setTimeout 轮询会在缓存里无限续跑（后台请求风暴）。用
  // useLifecycleSafeTimer 统一在 onDeactivated 停表、onActivated 恢复。
  const { schedule: schedulePolling, cancel: cancelPolling } = useLifecycleSafeTimer(() => {
    if (polling || loading.value) return
    polling = true
    refreshSoft()
      .catch(() => undefined)
      .finally(() => {
        polling = false
      })
  }, 10000)

  onMounted(async () => {
    if (canViewSettings.value) {
      void loadSettings().catch(() => undefined)
    }

    const status = String(route.query.status || '').trim()
    if (status) {
      searchForm.value.status = status
    }

    Object.assign(searchParams, { status: status || undefined })
    startPolling()
    await getData().catch(() => undefined)
  })

  onBeforeUnmount(() => {
    stopPolling()
  })

  onActivated(() => {
    // keep-alive 切回前台：恢复轮询（切走时 onDeactivated 已停表）。
    if (!pollingStopped) schedulePolling()
  })

  function createDefaultSearchForm(): ProbeSearchForm {
    return {
      keyword: '',
      status: undefined
    }
  }

  function createDefaultDialogForm(): ProbeDialogFormValue {
    return {
      id: null,
      name: '',
      agent_id: '',
      os_type: 'linux',
      tags: []
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

  function normalizeProbe(row?: ProbeRecordLike | null): ProbeTableRow {
    const cpuUsage = normalizeNullableNumber(
      row?.cpu_usage_percent ?? row?.CPUUsagePercent ?? row?.cpuUsagePercent ?? row?.CpuUsagePercent
    )
    const memoryUsage = normalizeNullableNumber(
      row?.mem_usage_percent ?? row?.MemUsagePercent ?? row?.memUsagePercent ?? row?.MEMUsagePercent
    )
    return {
      id: normalizeNullableNumber(row?.id ?? row?.ID),
      name: String(row?.name ?? row?.Name ?? ''),
      agent_id: String(row?.agent_id ?? row?.AgentID ?? ''),
      status: String(row?.status ?? row?.Status ?? 'offline'),
      os_type: String(row?.os_type ?? row?.OSType ?? ''),
      tags: Array.isArray(row?.tags ?? row?.Tags)
        ? (row?.tags ?? (row?.Tags as unknown[])).map((item) => String(item))
        : [],
      last_heartbeat_at: normalizeNullableString(row?.last_heartbeat_at ?? row?.LastHeartbeatAt),
      last_snapshot_at: normalizeNullableString(row?.last_snapshot_at ?? row?.LastSnapshotAt),
      snapshot: row?.snapshot ?? row?.Snapshot ?? null,
      cpu_usage_percent: cpuUsage,
      mem_usage_percent: memoryUsage,
      created_at: String(row?.created_at ?? row?.CreatedAt ?? ''),
      updated_at: String(row?.updated_at ?? row?.UpdatedAt ?? '')
    }
  }

  function toPositiveInt(value: unknown, fallback: number, min: number) {
    const parsed = Number.parseInt(String(value ?? '').trim(), 10)
    if (!Number.isFinite(parsed)) {
      return fallback
    }

    return Math.max(min, parsed)
  }

  function getStatusText(status?: string) {
    return status === 'online' ? '在线' : '离线'
  }

  function getStatusTagType(status?: string) {
    return status === 'online' ? ('success' as const) : ('info' as const)
  }

  function getUsagePercent(row: ProbeTableRow, type: 'cpu' | 'memory') {
    const source =
      type === 'cpu'
        ? (row.snapshot?.cpu?.usage_percent ?? row.cpu_usage_percent)
        : (row.snapshot?.memory?.usage_percent ?? row.mem_usage_percent)
    const value = Number(source)

    if (!Number.isFinite(value)) {
      return null
    }

    return Math.max(0, Math.min(100, Number(value.toFixed(1))))
  }

  function getUsageColor(value?: number | null) {
    if (value === null || value === undefined) {
      return 'var(--el-text-color-placeholder)'
    }

    if (value < 60) {
      return 'var(--el-color-success)'
    }

    if (value < 85) {
      return 'var(--el-color-warning)'
    }

    return 'var(--el-color-danger)'
  }

  function formatUsage(value?: number | null) {
    if (value === null || value === undefined) {
      return '-'
    }

    return `${Number(value).toFixed(1)}%`
  }

  function formatDateTime(value?: string | null) {
    if (!value) {
      return '-'
    }

    const date = new Date(value)
    return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN')
  }

  function getSlaPercent(id?: number | null) {
    if (!id) {
      return null
    }

    const value = slaMap[String(id)]
    return Number.isFinite(value) ? value : null
  }

  function getSlaTagType(value?: number | null) {
    if (value === null || value === undefined) {
      return 'info' as const
    }

    if (value >= 99.9) {
      return 'success' as const
    }

    if (value >= 99) {
      return 'warning' as const
    }

    return 'danger' as const
  }

  function formatSla(value?: number | null) {
    if (value === null || value === undefined) {
      return '-'
    }

    return `${Number(value).toFixed(2)}%`
  }

  function isAbortError(error: unknown) {
    const requestError = error as { name?: string; code?: string }
    return requestError?.name === 'AbortError' || requestError?.code === 'ERR_CANCELED'
  }

  async function loadSla(rows: ProbeTableRow[], signal: AbortSignal, force: boolean) {
    const now = Date.now()
    const pending = rows.filter((item) => {
      if (!item.id) return false
      const updatedAt = slaUpdatedAt.get(String(item.id)) || 0
      return force || now - updatedAt >= SLA_CACHE_MS
    })
    let nextIndex = 0

    const worker = async () => {
      while (!signal.aborted) {
        const item = pending[nextIndex++]
        if (!item?.id) return

        try {
          const payload = await fetchAdminProbeSla(Number(item.id), { days: 7 }, signal)
          if (signal.aborted) return
          const key = String(item.id)
          slaMap[key] = Number(payload.sla?.uptime_percent || 0)
          slaUpdatedAt.set(key, Date.now())
        } catch (error) {
          if (signal.aborted || isAbortError(error)) return
          const key = String(item.id)
          slaMap[key] = 0
          slaUpdatedAt.set(key, Date.now())
        }
      }
    }

    await Promise.all(
      Array.from({ length: Math.min(SLA_CONCURRENCY, pending.length) }, () => worker())
    )
  }

  async function fetchProbeTable(
    params: ProbeTableParams,
    signal?: AbortSignal
  ): Promise<Api.Common.PaginatedResponse<ProbeTableRow>> {
    try {
      const payload = await fetchAdminProbes(
        {
          limit: params.size,
          offset: (params.current - 1) * params.size,
          keyword: params.keyword.trim() || undefined,
          status: params.status || undefined
        },
        signal
      )
      const records = (payload.items || []).map((item) => normalizeProbe(item))
      slaController?.abort()
      slaController = new AbortController()
      const force = forceSlaRefresh
      forceSlaRefresh = false
      void loadSla(records, slaController.signal, force)
      lastRefreshAt.value = new Date().toISOString()
      refreshError.value = ''
      return {
        records,
        current: params.current,
        size: params.size,
        total: payload.total ?? records.length
      }
    } catch (error: any) {
      if (signal?.aborted || isAbortError(error)) throw error
      refreshError.value = String(error?.message || '请求失败')
      throw error
    }
  }

  async function refreshTable() {
    forceSlaRefresh = true
    await refreshData()
  }

  const fetchData = refreshTable

  async function loadSettings() {
    settingsLoading.value = true

    try {
      const payload = await fetchAdminSettings()
      const map = new Map<string, string>()

      ;(payload.items || []).forEach((item) => {
        map.set(String(item.key || ''), String(item.value || ''))
      })

      settingsForm.heartbeat_interval_sec = toPositiveInt(
        map.get('probe_heartbeat_interval_sec'),
        20,
        5
      )
      settingsForm.offline_grace_sec = toPositiveInt(map.get('probe_offline_grace_sec'), 90, 15)
    } finally {
      settingsLoading.value = false
    }
  }

  async function saveSettings() {
    const heartbeat = Math.max(5, Number(settingsForm.heartbeat_interval_sec || 20))
    const grace = Math.max(heartbeat * 3, Number(settingsForm.offline_grace_sec || 90))

    settingsSaving.value = true

    try {
      await updateAdminSettings({
        items: [
          { key: 'probe_heartbeat_interval_sec', value: String(heartbeat) },
          { key: 'probe_offline_grace_sec', value: String(grace) }
        ]
      })

      settingsForm.heartbeat_interval_sec = heartbeat
      settingsForm.offline_grace_sec = grace
      ElMessage.success('探针设置已保存')
    } finally {
      settingsSaving.value = false
    }
  }

  function startPolling() {
    pollingStopped = false
    schedulePolling()
  }

  function stopPolling() {
    pollingStopped = true
    cancelPolling()
    slaController?.abort()
    slaController = null
  }

  function handleSearch(params: ProbeSearchForm) {
    searchForm.value = { ...searchForm.value, ...params }
    Object.assign(searchParams, {
      keyword: params.keyword || undefined,
      status: params.status || undefined
    })
    getData()
  }

  function handleReset() {
    searchForm.value = createDefaultSearchForm()
    resetSearchParams()
  }

  function openCreate() {
    dialogMode.value = 'create'
    dialogForm.value = createDefaultDialogForm()
    dialogVisible.value = true
  }

  function openEdit(row?: ProbeTableRow | null) {
    if (!row) {
      return
    }

    dialogMode.value = 'edit'
    dialogForm.value = {
      id: row.id,
      name: row.name,
      agent_id: row.agent_id,
      os_type: row.os_type || 'linux',
      tags: [...row.tags]
    }
    dialogVisible.value = true
  }

  function openDetail(row?: ProbeTableRow | null) {
    if (!row?.id) {
      return
    }

    router.push({
      name: 'ProbeDetail',
      params: { id: String(row.id) }
    })
  }

  async function handleDialogSubmit(form: ProbeDialogFormValue) {
    dialogSubmitting.value = true

    try {
      if (dialogMode.value === 'create') {
        const payload = await createProbe({
          name: form.name.trim(),
          agent_id: form.agent_id.trim(),
          os_type: form.os_type.trim(),
          tags: form.tags
        })

        currentToken.value = String(payload.enroll_token || '')
        tokenVisible.value = Boolean(currentToken.value)
        ElMessage.success('探针创建成功')
      } else if (form.id) {
        await updateAdminProbe(form.id, {
          name: form.name.trim(),
          os_type: form.os_type.trim(),
          tags: form.tags
        })

        ElMessage.success('探针已更新')
      }

      dialogVisible.value = false
      await fetchData()
    } finally {
      dialogSubmitting.value = false
    }
  }

  async function handleResetEnrollToken(row?: ProbeTableRow | null) {
    if (!row?.id) {
      return
    }

    const payload = await resetProbeEnrollToken(row.id)
    currentToken.value = String(payload.enroll_token || '')
    tokenVisible.value = Boolean(currentToken.value)
    ElMessage.success('注册令牌已重置')
  }

  async function handleDelete(row?: ProbeTableRow | null) {
    if (!row?.id) {
      return
    }

    try {
      await ElMessageBox.confirm('删除后无法恢复，确认继续吗？', '删除探针', {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        type: 'warning'
      })
    } catch {
      return
    }

    await deleteAdminProbe(row.id)
    ElMessage.success('探针已删除')
    await fetchData()
  }

  function getMoreActions(): ButtonMoreItem[] {
    return [
      {
        key: 'reset-token',
        label: '重置注册令牌',
        icon: 'ri:key-2-line',
        auth: 'probe.update'
      },
      {
        key: 'delete',
        label: '删除探针',
        icon: 'ri:delete-bin-line',
        color: 'var(--el-color-danger)',
        auth: 'probe.delete'
      }
    ]
  }

  function handleMoreAction(item: ButtonMoreItem, row: ProbeTableRow) {
    switch (item.key) {
      case 'reset-token':
        handleResetEnrollToken(row)
        break
      case 'delete':
        handleDelete(row)
        break
    }
  }

  async function copyCurrentToken() {
    if (!currentToken.value) {
      return
    }

    await navigator.clipboard.writeText(currentToken.value)
    ElMessage.success('令牌已复制')
  }
</script>

<style scoped lang="scss">
  .settings-card {
    margin-bottom: 0;
  }

  .settings-head {
    display: flex;
    gap: 16px;
    justify-content: space-between;
    margin-bottom: 18px;
  }

  .settings-title {
    font-size: 16px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  .settings-tip {
    margin-top: 6px;
    font-size: 13px;
    color: var(--el-text-color-secondary);
  }

  .settings-item {
    margin-bottom: 0;
  }

  .refresh-status {
    padding: 12px 16px 0;
    font-size: 13px;
    color: var(--el-text-color-secondary);
  }

  .usage-cell {
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto;
    gap: 10px;
    align-items: center;
    min-width: 140px;
  }

  .tag-list {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }

  .table-actions {
    display: flex;
    gap: 2px;
    align-items: center;
    justify-content: flex-end;
  }

  .token-box {
    padding: 16px;
    margin-top: 16px;
    font-family: 'JetBrains Mono', Consolas, monospace;
    color: var(--el-text-color-primary);
    word-break: break-all;
    background: var(--el-fill-color-light);
    border-radius: 8px;
  }

  .dialog-footer {
    display: flex;
    gap: 12px;
    justify-content: flex-end;
  }

  @media (width <= 768px) {
    .settings-head,
    .table-actions {
      flex-direction: column;
      align-items: flex-start;
    }
  }
</style>
