<template>
  <div class="art-full-height">
    <ElCard class="art-table-card">
      <ArtTableHeader v-model:columns="columnChecks" :loading="loading" @refresh="fetchData">
        <template #left>
          <ElSpace wrap>
            <ElTag type="info">任务数 {{ tableData.length }}</ElTag>
            <ElTag type="success">已启用 {{ enabledCount }}</ElTag>
            <ElTag type="warning">运行中 {{ runningCount }}</ElTag>
          </ElSpace>
        </template>
      </ArtTableHeader>

      <ArtTable
        row-key="key"
        :loading="loading"
        :data="tableData"
        :columns="columns"
        empty-text="暂无计划任务"
      >
        <template #enabled="{ row }">
          <ElSwitch
            :model-value="row.enabled"
            :disabled="!canUpdate || isTaskUpdating(row.key)"
            @change="handleToggle(row, Boolean($event))"
          />
        </template>

        <template #strategy="{ row }">
          <div class="strategy-cell">
            <ElTag :type="row.strategy === 'daily' ? 'success' : 'info'">
              {{ getStrategyLabel(row.strategy) }}
            </ElTag>
            <span class="strategy-text">{{ getScheduleText(row) }}</span>
          </div>
        </template>

        <template #runtime="{ row }">
          <div class="runtime-cell">
            <ElTag :type="getRuntimeTagType(row)">
              {{ getRuntimeText(row) }}
            </ElTag>
            <span class="runtime-text">耗时 {{ formatElapsed(row.last_elapsed_sec) }}</span>
            <span v-if="row.last_error" class="runtime-error" :title="row.last_error">
              {{ row.last_error }}
            </span>
          </div>
        </template>

        <template #last_run_at="{ row }">
          {{ formatDateTime(row.last_run_at) }}
        </template>

        <template #next_run_at="{ row }">
          {{ formatDateTime(row.next_run_at) }}
        </template>

        <template #operation="{ row }">
          <div class="table-actions">
            <ElButton
              link
              type="primary"
              :disabled="!canUpdate || isTaskUpdating(row.key)"
              @click="openConfig(row)"
            >
              配置
            </ElButton>

            <ElButton
              link
              :disabled="!canViewRuns || isTaskUpdating(row.key)"
              @click="openRuns(row)"
            >
              运行记录
            </ElButton>
          </div>
        </template>
      </ArtTable>
    </ElCard>

    <TaskConfigDialog
      v-model:visible="configVisible"
      :task="currentTask"
      :submitting="configSubmitting"
      @submit="handleSubmitConfig"
    />

    <TaskRunsDrawer
      v-model:visible="runsVisible"
      :task="currentTask"
      :runs="runRows"
      :loading="runsLoading"
      @refresh="refreshRuns"
    />
  </div>
</template>

<script setup lang="ts">
  import type {
    ScheduledTaskRecord,
    ScheduledTaskRunRecord,
    ScheduledTaskStrategy
  } from '@/api/admin'
  import {
    fetchAdminScheduledTaskRuns,
    fetchAdminScheduledTasks,
    hasAdminPermission,
    updateAdminScheduledTask
  } from '@/api/admin'
  import { useTableColumns } from '@/hooks/core/useTableColumns'
  import { useUserStore } from '@/store/modules/user'
  import { ElMessage } from 'element-plus'
  import TaskConfigDialog from './modules/task-config-dialog.vue'
  import TaskRunsDrawer from './modules/task-runs-drawer.vue'

  defineOptions({ name: 'ScheduledTasksPage' })

  interface TaskConfigFormValue {
    enabled: boolean
    strategy: ScheduledTaskStrategy
    interval_sec: number
    daily_at: string
  }

  interface ScheduledTaskTableRow {
    key: string
    name: string
    description: string
    enabled: boolean
    strategy: string
    interval_sec: number
    daily_at: string
    last_run_at: string | null
    next_run_at: string | null
    running: boolean
    last_status: string
    last_error: string
    last_elapsed_sec?: number
  }

  interface TaskRunTableRow {
    id: number | null
    task_key: string
    status: string
    started_at: string
    finished_at: string | null
    duration_sec?: number
    message: string
    created_at: string
  }

  const loading = ref(false)
  const runsLoading = ref(false)
  const configSubmitting = ref(false)
  const configVisible = ref(false)
  const runsVisible = ref(false)

  const tableData = ref<ScheduledTaskTableRow[]>([])
  const currentTask = ref<ScheduledTaskTableRow | null>(null)
  const runRows = ref<TaskRunTableRow[]>([])
  const updatingKeys = ref<string[]>([])
  const poller = ref<number | null>(null)

  const userStore = useUserStore()
  const { info } = storeToRefs(userStore)

  const { columnChecks, columns } = useTableColumns<ScheduledTaskTableRow>(() => [
    { prop: 'name', label: '任务名称', minWidth: 220, showOverflowTooltip: true },
    { prop: 'description', label: '描述', minWidth: 280, showOverflowTooltip: true },
    { prop: 'enabled', label: '启用', width: 100, useSlot: true },
    { prop: 'strategy', label: '执行计划', minWidth: 180, useSlot: true },
    { prop: 'runtime', label: '运行状态', minWidth: 200, useSlot: true },
    { prop: 'last_run_at', label: '上次运行', minWidth: 180, useSlot: true },
    { prop: 'next_run_at', label: '下次运行', minWidth: 180, useSlot: true },
    { prop: 'operation', label: '操作', width: 170, fixed: 'right', useSlot: true }
  ])

  const enabledCount = computed(() => tableData.value.filter((item) => item.enabled).length)
  const runningCount = computed(() => tableData.value.filter((item) => item.running).length)

  const canUpdate = computed(() =>
    hasAdminPermission(info.value?.buttons, ['scheduled_tasks.update'])
  )

  const canViewRuns = computed(() =>
    hasAdminPermission(info.value?.buttons, ['scheduled_tasks.runs'])
  )

  onMounted(() => {
    fetchData()
  })

  onUnmounted(() => {
    stopPolling()
  })

  watch(runsVisible, (visible) => {
    if (!visible) {
      stopPolling()
      return
    }

    startPolling()
  })

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

  function normalizeTask(item?: ScheduledTaskRecord): ScheduledTaskTableRow {
    return {
      key: String(item?.key || ''),
      name: String(item?.name || ''),
      description: String(item?.description || ''),
      enabled: Boolean(item?.enabled),
      strategy: String(item?.strategy || 'interval'),
      interval_sec: Number(item?.interval_sec || 0),
      daily_at: String(item?.daily_at || ''),
      last_run_at: normalizeNullableString(item?.last_run_at),
      next_run_at: normalizeNullableString(item?.next_run_at),
      running: Boolean(item?.running),
      last_status: String(item?.last_status || ''),
      last_error: String(item?.last_error || ''),
      last_elapsed_sec: normalizeNullableNumber(item?.last_elapsed_sec) ?? undefined
    }
  }

  function normalizeRun(item?: ScheduledTaskRunRecord): TaskRunTableRow {
    return {
      id: normalizeNullableNumber(item?.id ?? item?.ID),
      task_key: String(item?.task_key ?? item?.TaskKey ?? ''),
      status: String(item?.status ?? item?.Status ?? ''),
      started_at: String(item?.started_at ?? item?.StartedAt ?? ''),
      finished_at: normalizeNullableString(item?.finished_at ?? item?.FinishedAt),
      duration_sec: normalizeNullableNumber(item?.duration_sec ?? item?.DurationSec) ?? undefined,
      message: String(item?.message ?? item?.Message ?? ''),
      created_at: String(item?.created_at ?? item?.CreatedAt ?? '')
    }
  }

  function sortTasks(items: ScheduledTaskTableRow[]) {
    return [...items].sort(
      (a, b) => a.name.localeCompare(b.name, 'zh-CN') || a.key.localeCompare(b.key, 'zh-CN')
    )
  }

  function syncCurrentTask() {
    if (!currentTask.value?.key) {
      return
    }

    const matched = tableData.value.find((item) => item.key === currentTask.value?.key)
    if (matched) {
      currentTask.value = { ...matched }
    }
  }

  function isTaskUpdating(key: string) {
    return updatingKeys.value.includes(key)
  }

  function setTaskUpdating(key: string, value: boolean) {
    if (value) {
      if (!updatingKeys.value.includes(key)) {
        updatingKeys.value = [...updatingKeys.value, key]
      }
      return
    }

    updatingKeys.value = updatingKeys.value.filter((item) => item !== key)
  }

  function getStrategyLabel(strategy?: string) {
    return strategy === 'daily' ? '每日执行' : '间隔执行'
  }

  function getScheduleText(task: ScheduledTaskTableRow) {
    if (task.strategy === 'daily') {
      return task.daily_at ? `每天 ${task.daily_at}` : '每天执行'
    }

    return `每 ${Number(task.interval_sec || 0)} 秒`
  }

  function getRuntimeText(task: ScheduledTaskTableRow) {
    if (task.running) {
      return '运行中'
    }

    switch (task.last_status) {
      case 'success':
        return '上次成功'
      case 'failed':
        return '上次失败'
      case 'running':
        return '运行中'
      default:
        return '未运行'
    }
  }

  function getRuntimeTagType(task: ScheduledTaskTableRow) {
    if (task.running) {
      return 'warning' as const
    }

    switch (task.last_status) {
      case 'success':
        return 'success' as const
      case 'failed':
        return 'danger' as const
      case 'running':
        return 'warning' as const
      default:
        return 'info' as const
    }
  }

  function formatElapsed(value?: number | null) {
    if (value === null || value === undefined) {
      return '-'
    }

    return `${Number(value)}s`
  }

  function formatDateTime(value?: string | null) {
    if (!value) {
      return '-'
    }

    const date = new Date(value)
    return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN')
  }

  async function fetchData(options: { silent?: boolean } = {}) {
    if (!options.silent) {
      loading.value = true
    }

    try {
      const payload = await fetchAdminScheduledTasks()
      tableData.value = sortTasks((payload.items || []).map((item) => normalizeTask(item)))
      syncCurrentTask()
    } finally {
      if (!options.silent) {
        loading.value = false
      }
    }
  }

  async function fetchRuns(options: { silent?: boolean } = {}) {
    if (!currentTask.value?.key) {
      return
    }

    if (!options.silent) {
      runsLoading.value = true
    }

    try {
      const payload = await fetchAdminScheduledTaskRuns(currentTask.value.key, { limit: 20 })
      runRows.value = (payload.items || [])
        .map((item) => normalizeRun(item))
        .sort((a, b) => {
          const timeA = new Date(a.started_at || a.created_at || 0).getTime()
          const timeB = new Date(b.started_at || b.created_at || 0).getTime()
          return timeB - timeA
        })
    } finally {
      if (!options.silent) {
        runsLoading.value = false
      }
    }
  }

  async function handleToggle(task: ScheduledTaskTableRow, enabled: boolean) {
    if (!canUpdate.value || !task.key) {
      return
    }

    setTaskUpdating(task.key, true)

    try {
      await updateAdminScheduledTask(task.key, { enabled })
      ElMessage.success('任务状态已更新')
      await fetchData()
      if (runsVisible.value && currentTask.value?.key === task.key) {
        await fetchRuns({ silent: true })
      }
    } finally {
      setTaskUpdating(task.key, false)
    }
  }

  function openConfig(task: ScheduledTaskTableRow) {
    currentTask.value = { ...task }
    configVisible.value = true
  }

  async function handleSubmitConfig(form: TaskConfigFormValue) {
    if (!currentTask.value?.key) {
      return
    }

    configSubmitting.value = true

    try {
      await updateAdminScheduledTask(currentTask.value.key, {
        enabled: form.enabled,
        strategy: form.strategy,
        interval_sec: form.strategy === 'interval' ? form.interval_sec : undefined,
        daily_at: form.strategy === 'daily' ? form.daily_at : undefined
      })

      ElMessage.success('任务配置已保存')
      configVisible.value = false
      await fetchData()
      if (runsVisible.value && currentTask.value?.key) {
        await fetchRuns({ silent: true })
      }
    } finally {
      configSubmitting.value = false
    }
  }

  async function openRuns(task: ScheduledTaskTableRow) {
    currentTask.value = { ...task }
    runsVisible.value = true
    await Promise.all([fetchData({ silent: true }), fetchRuns()])
  }

  async function refreshRuns() {
    await Promise.all([fetchData({ silent: true }), fetchRuns()])
  }

  function startPolling() {
    stopPolling()

    if (!runsVisible.value) {
      return
    }

    poller.value = window.setInterval(() => {
      if (!runsVisible.value || !currentTask.value?.key) {
        return
      }

      fetchData({ silent: true })
      fetchRuns({ silent: true })
    }, 5000)
  }

  function stopPolling() {
    if (poller.value !== null) {
      window.clearInterval(poller.value)
      poller.value = null
    }
  }
</script>

<style scoped lang="scss">
  .strategy-cell,
  .runtime-cell {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .strategy-text,
  .runtime-text {
    color: var(--el-text-color-secondary);
    font-size: 12px;
    line-height: 1.5;
  }

  .runtime-error {
    overflow: hidden;
    color: var(--el-color-danger);
    font-size: 12px;
    line-height: 1.5;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .table-actions {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: 4px;
  }
</style>
