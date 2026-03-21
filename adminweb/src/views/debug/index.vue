<template>
  <div class="debug-page art-full-height">
    <ElAlert
      type="warning"
      :closable="false"
      show-icon
      title="调试模式会记录详细请求日志，仅建议在排查问题时短期开启。"
    />

    <ElRow :gutter="16" class="summary-grid">
      <ElCol :xs="24" :lg="8">
        <ElCard v-loading="statusLoading" shadow="never">
          <template #header>
            <div class="card-header">
              <div>
                <div class="card-title">调试状态</div>
                <div class="card-subtitle">用于排查自动化请求与系统日志问题</div>
              </div>
              <ElTag :type="debugEnabled ? 'warning' : 'info'">{{
                debugEnabled ? '已开启' : '未开启'
              }}</ElTag>
            </div>
          </template>

          <template v-if="canViewStatus || canUpdateStatus">
            <div class="status-panel">
              <ElSwitch
                :model-value="debugEnabled"
                :disabled="!canUpdateStatus || debugUpdating"
                size="large"
                @change="handleToggleDebug"
              />
              <div>
                <div class="status-title">{{
                  debugEnabled ? '调试模式已启用' : '调试模式已禁用'
                }}</div>
                <p class="status-copy">开启后将记录 API 请求的详细入参与响应内容。</p>
              </div>
            </div>
          </template>

          <ElEmpty v-else description="当前账号没有查看调试状态的权限" />
        </ElCard>
      </ElCol>

      <ElCol :xs="24" :lg="16">
        <ElCard v-loading="retentionLoading" shadow="never">
          <template #header>
            <div class="card-header">
              <div>
                <div class="card-title">日志保留策略</div>
                <div class="card-subtitle">由系统任务每天自动清理过期日志</div>
              </div>
              <ElButton
                v-if="canUpdateSettings"
                type="primary"
                :loading="savingRetention"
                @click="saveRetentionSettings"
                >保存策略</ElButton
              >
            </div>
          </template>

          <template v-if="canViewSettings || canUpdateSettings">
            <div class="retention-grid">
              <div v-for="item in retentionItems" :key="item.key" class="retention-item">
                <span class="retention-label">{{ item.label }}</span>
                <ElInputNumber
                  v-model="retention[item.key]"
                  :min="1"
                  :max="3650"
                  :step="1"
                  step-strictly
                  controls-position="right"
                  :disabled="!canUpdateSettings || savingRetention"
                  class="retention-input"
                />
              </div>
            </div>
          </template>

          <ElEmpty v-else description="当前账号没有查看日志保留策略的权限" />
        </ElCard>
      </ElCol>
    </ElRow>

    <DebugLogTabs
      v-if="canViewLogs"
      v-model:active-tab="activeLogTab"
      :loading="loading"
      :audit-logs="auditLogs"
      :automation-logs="automationLogs"
      :sync-logs="syncLogs"
      :audit-pagination="auditPagination"
      :automation-pagination="automationPagination"
      :sync-pagination="syncPagination"
      @refresh="fetchLogs()"
      @page-size-change="handlePageSizeChange"
      @page-current-change="handlePageCurrentChange"
      @view-automation-detail="openDetail"
    />
    <ElCard v-else shadow="never"><ElEmpty description="当前账号没有查看调试日志的权限" /></ElCard>

    <AutomationLogDetailDialog v-model:visible="detailVisible" :record="detailRecord" />
  </div>
</template>

<script setup lang="ts">
  import type {
    AdminAuditLogRecord,
    AutomationLogRecord,
    DebugLogsResponse,
    IntegrationSyncLogRecord
  } from '@/api/admin'
  import {
    fetchAdminDebugLogs,
    fetchAdminDebugStatus,
    fetchAdminSettings,
    hasAdminPermission,
    updateAdminDebugStatus,
    updateAdminSettings
  } from '@/api/admin'
  import { useUserStore } from '@/store/modules/user'
  import AutomationLogDetailDialog from './modules/automation-log-detail-dialog.vue'
  import DebugLogTabs from './modules/debug-log-tabs.vue'

  defineOptions({ name: 'DebugPage' })

  type LogTabKey = 'audit' | 'automation' | 'sync'
  type RetentionKey =
    | 'automation'
    | 'audit'
    | 'sync'
    | 'task_runs'
    | 'probe_events'
    | 'probe_sessions'

  interface PaginationState {
    current: number
    size: number
    total: number
  }

  const loading = ref(false)
  const statusLoading = ref(false)
  const retentionLoading = ref(false)
  const debugUpdating = ref(false)
  const savingRetention = ref(false)

  const debugEnabled = ref(false)
  const activeLogTab = ref<LogTabKey>('audit')
  const detailVisible = ref(false)
  const detailRecord = ref<AutomationLogRecord | null>(null)

  const auditLogs = ref<AdminAuditLogRecord[]>([])
  const automationLogs = ref<AutomationLogRecord[]>([])
  const syncLogs = ref<IntegrationSyncLogRecord[]>([])

  const auditPagination = reactive<PaginationState>({ current: 1, size: 20, total: 0 })
  const automationPagination = reactive<PaginationState>({ current: 1, size: 20, total: 0 })
  const syncPagination = reactive<PaginationState>({ current: 1, size: 20, total: 0 })

  const retention = reactive<Record<RetentionKey, number>>({
    automation: 30,
    audit: 90,
    sync: 30,
    task_runs: 14,
    probe_events: 30,
    probe_sessions: 7
  })

  const retentionItems: Array<{ key: RetentionKey; label: string }> = [
    { key: 'automation', label: '自动化日志' },
    { key: 'audit', label: '审计日志' },
    { key: 'sync', label: '同步日志' },
    { key: 'task_runs', label: '计划任务运行日志' },
    { key: 'probe_events', label: '探针状态事件' },
    { key: 'probe_sessions', label: '探针日志会话' }
  ]

  const userStore = useUserStore()
  const { info } = storeToRefs(userStore)

  const canViewStatus = computed(() => hasAdminPermission(info.value?.buttons, ['debug.view']))
  const canUpdateStatus = computed(() => hasAdminPermission(info.value?.buttons, ['debug.update']))
  const canViewLogs = computed(() => hasAdminPermission(info.value?.buttons, ['debug.list']))
  const canViewSettings = computed(() => hasAdminPermission(info.value?.buttons, ['settings.view']))
  const canUpdateSettings = computed(() =>
    hasAdminPermission(info.value?.buttons, ['settings.update'])
  )

  onMounted(() => {
    initializePage()
  })

  watch(activeLogTab, (tab) => {
    if (canViewLogs.value) fetchLogs(tab)
  })

  async function initializePage() {
    const tasks: Promise<unknown>[] = []
    if (canViewStatus.value || canUpdateStatus.value) tasks.push(fetchStatus())
    if (canViewSettings.value || canUpdateSettings.value) tasks.push(fetchRetentionSettings())
    if (canViewLogs.value) tasks.push(fetchLogs('audit'))
    await Promise.allSettled(tasks)
  }

  function getPager(type: LogTabKey) {
    if (type === 'automation') return automationPagination
    if (type === 'sync') return syncPagination
    return auditPagination
  }

  function parseDays(value: unknown, fallback: number) {
    const parsed = Number(value)
    if (!Number.isFinite(parsed) || parsed <= 0) return fallback
    return Math.min(3650, Math.trunc(parsed))
  }

  function applyLogs(type: LogTabKey, payload: DebugLogsResponse) {
    if (type === 'automation') {
      automationLogs.value = payload.automation_logs?.items || []
      automationPagination.total = Number(
        payload.automation_logs?.total ?? automationLogs.value.length
      )
      return
    }
    if (type === 'sync') {
      syncLogs.value = payload.sync_logs?.items || []
      syncPagination.total = Number(payload.sync_logs?.total ?? syncLogs.value.length)
      return
    }
    auditLogs.value = payload.audit_logs?.items || []
    auditPagination.total = Number(payload.audit_logs?.total ?? auditLogs.value.length)
  }

  async function fetchStatus() {
    statusLoading.value = true
    try {
      debugEnabled.value = Boolean((await fetchAdminDebugStatus()).enabled)
    } finally {
      statusLoading.value = false
    }
  }

  async function handleToggleDebug(value: boolean | string | number) {
    const checked = Boolean(value)
    const previous = debugEnabled.value
    debugEnabled.value = checked
    debugUpdating.value = true
    try {
      await updateAdminDebugStatus({ enabled: checked })
      ElMessage.success(checked ? '调试模式已启用' : '调试模式已禁用')
    } catch {
      debugEnabled.value = previous
    } finally {
      debugUpdating.value = false
    }
  }

  async function fetchRetentionSettings() {
    retentionLoading.value = true
    try {
      const payload = await fetchAdminSettings()
      const values = new Map<string, string>()
      ;(payload.items || []).forEach(
        (item) => item.key && values.set(String(item.key), String(item.value || ''))
      )
      retention.automation = parseDays(values.get('automation_log_retention_days'), 30)
      retention.audit = parseDays(values.get('audit_log_retention_days'), 90)
      retention.sync = parseDays(values.get('integration_sync_log_retention_days'), 30)
      retention.task_runs = parseDays(values.get('scheduled_task_run_retention_days'), 14)
      retention.probe_events = parseDays(values.get('probe_status_event_retention_days'), 30)
      retention.probe_sessions = parseDays(values.get('probe_log_session_retention_days'), 7)
    } finally {
      retentionLoading.value = false
    }
  }

  async function saveRetentionSettings() {
    savingRetention.value = true
    try {
      await updateAdminSettings({
        items: [
          {
            key: 'automation_log_retention_days',
            value: String(parseDays(retention.automation, 30))
          },
          { key: 'audit_log_retention_days', value: String(parseDays(retention.audit, 90)) },
          {
            key: 'integration_sync_log_retention_days',
            value: String(parseDays(retention.sync, 30))
          },
          {
            key: 'scheduled_task_run_retention_days',
            value: String(parseDays(retention.task_runs, 14))
          },
          {
            key: 'probe_status_event_retention_days',
            value: String(parseDays(retention.probe_events, 30))
          },
          {
            key: 'probe_log_session_retention_days',
            value: String(parseDays(retention.probe_sessions, 7))
          }
        ]
      })
      ElMessage.success('日志保留策略已保存')
    } finally {
      savingRetention.value = false
    }
  }

  async function fetchLogs(type: LogTabKey = activeLogTab.value) {
    loading.value = true
    try {
      const pager = getPager(type)
      applyLogs(
        type,
        await fetchAdminDebugLogs({
          types: type,
          limit: pager.size,
          offset: (pager.current - 1) * pager.size
        })
      )
    } finally {
      loading.value = false
    }
  }

  function handlePageSizeChange(type: LogTabKey, size: number) {
    const pager = getPager(type)
    pager.size = size
    pager.current = 1
    fetchLogs(type)
  }

  function handlePageCurrentChange(type: LogTabKey, page: number) {
    const pager = getPager(type)
    pager.current = page
    fetchLogs(type)
  }

  function openDetail(record: AutomationLogRecord) {
    detailRecord.value = record
    detailVisible.value = true
  }
</script>

<style scoped lang="scss">
  .debug-page {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .summary-grid {
    margin: 0 !important;
  }

  .card-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
  }

  .card-title {
    font-size: 16px;
    font-weight: 700;
  }

  .card-subtitle,
  .status-copy {
    margin: 4px 0 0;
    color: var(--el-text-color-secondary);
    font-size: 13px;
    line-height: 1.7;
  }

  .status-panel {
    display: grid;
    grid-template-columns: auto minmax(0, 1fr);
    gap: 18px;
    align-items: center;
    min-height: 138px;
  }

  .status-title {
    font-size: 18px;
    font-weight: 700;
  }

  .retention-grid {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 16px;
  }

  .retention-item {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .retention-label {
    font-size: 13px;
    font-weight: 600;
  }

  .retention-input {
    width: 100%;
  }

  @media (max-width: 1200px) {
    .retention-grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
  }

  @media (max-width: 768px) {
    .card-header,
    .status-panel {
      grid-template-columns: 1fr;
      flex-direction: column;
      align-items: flex-start;
    }

    .retention-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
