<template>
  <div class="art-full-height">
    <ElCard class="art-table-card">
      <ArtTableHeader v-model:columns="columnChecks" :loading="loading" @refresh="fetchData" />

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
          <ElTag>{{ row.strategy || '-' }}</ElTag>
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
  </div>
</template>

<script setup lang="ts">
  import type { ScheduledTaskRecord, ScheduledTaskStrategy } from '@/services/admin'
  import { fetchAdminScheduledTasks, updateAdminScheduledTask } from '@/services/admin'
  import { useTable } from '@/hooks/core/useTable'
  import { useAuth } from '@/hooks/core/useAuth'
  import { ElMessage } from 'element-plus'
  import TaskConfigDialog from './modules/task-config-dialog.vue'

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
    strategy: ScheduledTaskStrategy
    interval_sec: number
    daily_at: string
    last_run_at: string | null
    next_run_at: string | null
    running: boolean
    last_status: string
    last_error: string
    last_elapsed_sec?: number
  }

  const configSubmitting = ref(false)
  const configVisible = ref(false)

  const currentTask = ref<ScheduledTaskTableRow | null>(null)
  const updatingKeys = ref<string[]>([])

  const { hasAuth } = useAuth()

  const {
    columnChecks,
    columns,
    data: tableData,
    loading,
    refreshData
  } = useTable({
    core: {
      apiFn: fetchScheduledTaskTable,
      apiParams: { current: 1, size: 20 },
      immediate: false,
      columnsFactory: () => [
        { prop: 'name', label: '任务名称', minWidth: 220, showOverflowTooltip: true },
        { prop: 'description', label: '描述', minWidth: 280, showOverflowTooltip: true },
        { prop: 'enabled', label: '启用', width: 100, useSlot: true },
        { prop: 'strategy', label: '执行计划', minWidth: 180, useSlot: true },
        {
          prop: 'last_run_at',
          label: '上次运行',
          minWidth: 180,
          formatter: (row: ScheduledTaskTableRow) => formatDateTime(row.last_run_at)
        },
        {
          prop: 'next_run_at',
          label: '下次运行',
          minWidth: 180,
          formatter: (row: ScheduledTaskTableRow) => formatDateTime(row.next_run_at)
        },
        { prop: 'operation', label: '操作', width: 170, fixed: 'right', useSlot: true }
      ]
    }
  })

  const canUpdate = computed(() => hasAuth('scheduled_tasks.update'))

  onMounted(() => {
    fetchData()
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
      strategy: item?.strategy === 'daily' ? 'daily' : 'interval',
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

  function formatDateTime(value?: string | null) {
    if (!value) {
      return '-'
    }

    const date = new Date(value)
    return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN')
  }

  async function fetchScheduledTaskTable(
    _params?: unknown,
    signal?: AbortSignal
  ): Promise<Api.Common.PaginatedResponse<ScheduledTaskTableRow>> {
    const payload = await fetchAdminScheduledTasks(signal)
    const allRows = (payload.items || []).map(normalizeTask)
    return {
      records: allRows,
      current: 1,
      size: allRows.length,
      total: allRows.length
    }
  }

  async function fetchData() {
    await refreshData()
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
        daily_at: form.strategy === 'daily' && form.daily_at ? form.daily_at : undefined
      })

      ElMessage.success('任务配置已保存')
      configVisible.value = false
      await fetchData()
    } finally {
      configSubmitting.value = false
    }
  }
</script>

<style scoped lang="scss">
  .table-actions {
    display: flex;
    gap: 4px;
    align-items: center;
    justify-content: flex-end;
  }
</style>
