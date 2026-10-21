<template>
  <div class="art-full-height">
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
      <ArtTableHeader
        v-model:columns="columnChecks"
        v-model:showSearchBar="showSearchBar"
        :loading="loading"
        @refresh="fetchData"
      >
        <template #left>
          <ElSpace wrap>
            <ElButton v-if="canCreate" type="primary" v-ripple @click="openCreate">
              创建探针
            </ElButton>
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
            <ArtButtonTable v-if="canView" type="view" @click="openDetail(row)" />
            <ArtButtonTable v-if="canUpdate" type="edit" @click="openEdit(row)" />

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
  import type { ProbeRecord, ProbeSnapshotRecord } from '@/api/admin'
  import type { ButtonMoreItem } from '@/components/core/forms/art-button-more/index.vue'
  import {
    createAdminProbe,
    deleteAdminProbe,
    fetchAdminProbeSla,
    fetchAdminProbes,
    fetchAdminSettings,
    hasAdminPermission,
    resetAdminProbeEnrollToken,
    updateAdminProbe,
    updateAdminSettings
  } from '@/api/admin'
  import ArtButtonMore from '@/components/core/forms/art-button-more/index.vue'
  import ArtButtonTable from '@/components/core/forms/art-button-table/index.vue'
  import { useTableColumns } from '@/hooks/core/useTableColumns'
  import { useUserStore } from '@/store/modules/user'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import ProbeDialog from './modules/probe-dialog.vue'
  import ProbeSearch from './modules/probe-search.vue'

  defineOptions({ name: 'ProbeList' })

  interface ProbeSearchForm {
    keyword: string
    status?: string
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
    created_at: string
    updated_at: string
  }

  const route = useRoute()
  const router = useRouter()
  const userStore = useUserStore()
  const { info } = storeToRefs(userStore)

  const showSearchBar = ref(true)
  const loading = ref(false)
  const dialogVisible = ref(false)
  const dialogMode = ref<'create' | 'edit'>('create')
  const dialogSubmitting = ref(false)
  const tokenVisible = ref(false)
  const currentToken = ref('')
  const settingsLoading = ref(false)
  const settingsSaving = ref(false)

  const searchForm = ref<ProbeSearchForm>(createDefaultSearchForm())
  const dialogForm = ref<ProbeDialogFormValue>(createDefaultDialogForm())
  const tableData = ref<ProbeTableRow[]>([])
  const slaMap = reactive<Record<string, number>>({})

  const settingsForm = reactive({
    heartbeat_interval_sec: 20,
    offline_grace_sec: 90
  })

  const pagination = reactive({
    current: 1,
    size: 20,
    total: 0
  })

  const canView = computed(() =>
    hasAdminPermission(info.value?.buttons, ['probe.list', 'probe.view'])
  )
  const canCreate = computed(() => hasAdminPermission(info.value?.buttons, ['probe.create']))
  const canUpdate = computed(() => hasAdminPermission(info.value?.buttons, ['probe.update']))
  const canDelete = computed(() => hasAdminPermission(info.value?.buttons, ['probe.delete']))
  const canViewSettings = computed(() => hasAdminPermission(info.value?.buttons, ['settings.view']))
  const canUpdateSettings = computed(() =>
    hasAdminPermission(info.value?.buttons, ['settings.update'])
  )

  const { columnChecks, columns } = useTableColumns<ProbeTableRow>(() => [
    { prop: 'id', label: 'ID', width: 80 },
    {
      prop: 'name',
      label: '探针名称',
      minWidth: 180,
      showOverflowTooltip: true
    },
    {
      prop: 'agent_id',
      label: 'Agent ID',
      minWidth: 180,
      showOverflowTooltip: true
    },
    {
      prop: 'status',
      label: '状态',
      width: 100,
      useSlot: true
    },
    {
      prop: 'os_type',
      label: '系统',
      width: 110
    },
    {
      prop: 'cpu_usage',
      label: 'CPU',
      minWidth: 170,
      useSlot: true
    },
    {
      prop: 'mem_usage',
      label: '内存',
      minWidth: 170,
      useSlot: true
    },
    {
      prop: 'tags',
      label: '标签',
      minWidth: 220,
      useSlot: true
    },
    {
      prop: 'last_heartbeat_at',
      label: '最后心跳',
      minWidth: 180,
      formatter: (row: ProbeTableRow) => formatDateTime(row.last_heartbeat_at)
    },
    {
      prop: 'sla',
      label: '7 天 SLA',
      width: 110,
      useSlot: true
    },
    {
      prop: 'operation',
      label: '操作',
      width: 160,
      fixed: 'right',
      useSlot: true
    }
  ])

  let poller: number | null = null

  onMounted(async () => {
    if (canViewSettings.value) {
      await loadSettings()
    }

    const status = String(route.query.status || '').trim()
    if (status) {
      searchForm.value.status = status
    }

    await fetchData()
    startPolling()
  })

  onBeforeUnmount(() => {
    stopPolling()
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

  function normalizeProbe(row?: ProbeRecord | null): ProbeTableRow {
    return {
      id: normalizeNullableNumber(row?.id),
      name: String(row?.name || ''),
      agent_id: String(row?.agent_id || ''),
      status: String(row?.status || 'offline'),
      os_type: String(row?.os_type || ''),
      tags: Array.isArray(row?.tags) ? row.tags.map((item) => String(item)) : [],
      last_heartbeat_at: normalizeNullableString(row?.last_heartbeat_at),
      last_snapshot_at: normalizeNullableString(row?.last_snapshot_at),
      snapshot: row?.snapshot || null,
      created_at: String(row?.created_at || ''),
      updated_at: String(row?.updated_at || '')
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
      type === 'cpu' ? row.snapshot?.cpu?.usage_percent : row.snapshot?.memory?.usage_percent
    const value = Number(source)

    if (!Number.isFinite(value)) {
      return null
    }

    return Math.max(0, Math.min(100, Number(value.toFixed(1))))
  }

  function getUsageColor(value?: number | null) {
    if (value === null || value === undefined) {
      return '#c0c4cc'
    }

    if (value < 60) {
      return '#67c23a'
    }

    if (value < 85) {
      return '#e6a23c'
    }

    return '#f56c6c'
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

  async function loadSla(rows: ProbeTableRow[]) {
    Object.keys(slaMap).forEach((key) => delete slaMap[key])

    await Promise.all(
      rows
        .filter((item) => item.id)
        .map(async (item) => {
          try {
            const payload = await fetchAdminProbeSla(Number(item.id), { days: 7 })
            slaMap[String(item.id)] = Number(payload.sla?.uptime_percent || 0)
          } catch {
            slaMap[String(item.id)] = 0
          }
        })
    )
  }

  async function fetchData() {
    loading.value = true

    try {
      const payload = await fetchAdminProbes({
        limit: pagination.size,
        offset: (pagination.current - 1) * pagination.size,
        keyword: searchForm.value.keyword.trim() || undefined,
        status: searchForm.value.status || undefined
      })

      const rows = (payload.items || []).map((item) => normalizeProbe(item))
      tableData.value = rows
      pagination.total = typeof payload.total === 'number' ? payload.total : rows.length
      await loadSla(rows)
    } finally {
      loading.value = false
    }
  }

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
    stopPolling()
    poller = window.setInterval(() => {
      fetchData()
    }, 10000)
  }

  function stopPolling() {
    if (!poller) {
      return
    }

    window.clearInterval(poller)
    poller = null
  }

  function handleSearch(params: ProbeSearchForm) {
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
        const payload = await createAdminProbe({
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

    try {
      await ElMessageBox.confirm('确认重置该探针的注册令牌吗？', '重置注册令牌', {
        confirmButtonText: '重置',
        cancelButtonText: '取消',
        type: 'warning'
      })
    } catch {
      return
    }

    const payload = await resetAdminProbeEnrollToken(row.id)
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
        auth: 'probe.update',
        disabled: !canUpdate.value
      },
      {
        key: 'delete',
        label: '删除探针',
        icon: 'ri:delete-bin-line',
        color: '#f56c6c',
        auth: 'probe.delete',
        disabled: !canDelete.value
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

  function escapeCsvCell(value: string | number | null | undefined) {
    const text = String(value ?? '')
    return `"${text.replace(/"/g, '""')}"`
  }

  function exportCsv() {
    const rows = tableData.value.map((item) =>
      [
        escapeCsvCell(item.id),
        escapeCsvCell(item.name),
        escapeCsvCell(item.agent_id),
        escapeCsvCell(item.status),
        escapeCsvCell(item.last_heartbeat_at)
      ].join(',')
    )

    const content = ['id,name,agent_id,status,last_heartbeat_at', ...rows].join('\n')
    const blob = new Blob([content], { type: 'text/csv;charset=utf-8;' })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')

    link.href = url
    link.download = 'admin-probes.csv'
    link.click()

    URL.revokeObjectURL(url)
  }
</script>

<style scoped lang="scss">
  .settings-card {
    margin-bottom: 0;
  }

  .settings-head {
    display: flex;
    justify-content: space-between;
    gap: 16px;
    margin-bottom: 18px;
  }

  .settings-title {
    color: var(--el-text-color-primary);
    font-size: 16px;
    font-weight: 600;
  }

  .settings-tip {
    margin-top: 6px;
    color: var(--el-text-color-secondary);
    font-size: 13px;
  }

  .settings-item {
    margin-bottom: 0;
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
    align-items: center;
    justify-content: flex-end;
    gap: 2px;
  }

  .token-box {
    margin-top: 16px;
    padding: 16px;
    border-radius: 12px;
    background: var(--el-fill-color-light);
    color: var(--el-text-color-primary);
    font-family: 'JetBrains Mono', 'Consolas', monospace;
    word-break: break-all;
  }

  .dialog-footer {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
  }

  @media (max-width: 768px) {
    .settings-head,
    .table-actions {
      flex-direction: column;
      align-items: flex-start;
    }
  }
</style>
