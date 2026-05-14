<template>
  <ElDrawer
    v-model="drawerVisible"
    :title="drawerTitle"
    size="860px"
    destroy-on-close
    class="task-runs-drawer"
  >
    <div class="drawer-panel">
      <div class="drawer-toolbar">
        <div class="drawer-heading">
          <h3>{{ task?.name || '-' }}</h3>
          <p>{{ task?.description || '暂无任务描述' }}</p>
        </div>

        <ElButton :loading="loading" @click="emit('refresh')">刷新记录</ElButton>
      </div>

      <ElAlert
        v-if="task?.last_error"
        type="error"
        :closable="false"
        show-icon
        class="drawer-alert"
      >
        <template #title>{{ task.last_error }}</template>
      </ElAlert>

      <ElDescriptions :column="2" border>
        <ElDescriptionsItem label="任务键名">{{ task?.key || '-' }}</ElDescriptionsItem>
        <ElDescriptionsItem label="启用状态">
          <ElTag :type="task?.enabled ? 'success' : 'info'">
            {{ task?.enabled ? '已启用' : '已停用' }}
          </ElTag>
        </ElDescriptionsItem>
        <ElDescriptionsItem label="执行策略">
          <ElTag :type="task?.strategy === 'daily' ? 'success' : 'info'">
            {{ getStrategyText(task?.strategy) }}
          </ElTag>
        </ElDescriptionsItem>
        <ElDescriptionsItem label="执行计划">
          {{ getScheduleText(task) }}
        </ElDescriptionsItem>
        <ElDescriptionsItem label="当前状态">
          <ElTag :type="getRuntimeTagType(task)">
            {{ getRuntimeText(task) }}
          </ElTag>
        </ElDescriptionsItem>
        <ElDescriptionsItem label="上次耗时">
          {{ formatElapsed(task?.last_elapsed_sec) }}
        </ElDescriptionsItem>
        <ElDescriptionsItem label="上次运行">
          {{ formatDateTime(task?.last_run_at) }}
        </ElDescriptionsItem>
        <ElDescriptionsItem label="下次运行">
          {{ formatDateTime(task?.next_run_at) }}
        </ElDescriptionsItem>
      </ElDescriptions>

      <section class="drawer-section">
        <div class="section-title">最近运行记录</div>

        <ElTable v-loading="loading" :data="runs" border size="small">
          <ElTableColumn label="状态" min-width="110">
            <template #default="{ row }">
              <ElTag :type="getRunStatusTagType(row.status)">
                {{ getRunStatusText(row.status) }}
              </ElTag>
            </template>
          </ElTableColumn>

          <ElTableColumn label="开始时间" min-width="180">
            <template #default="{ row }">
              {{ formatDateTime(row.started_at) }}
            </template>
          </ElTableColumn>

          <ElTableColumn label="结束时间" min-width="180">
            <template #default="{ row }">
              {{ formatDateTime(row.finished_at) }}
            </template>
          </ElTableColumn>

          <ElTableColumn label="耗时" min-width="100">
            <template #default="{ row }">
              {{ formatElapsed(row.duration_sec) }}
            </template>
          </ElTableColumn>

          <ElTableColumn prop="message" label="消息" min-width="280" show-overflow-tooltip />
        </ElTable>

        <div v-if="!runs.length && !loading" class="empty-tip">暂无运行记录</div>
      </section>
    </div>
  </ElDrawer>
</template>

<script setup lang="ts">
  import type { ScheduledTaskRecord } from '@/api/admin'

  defineOptions({ name: 'TaskRunsDrawer' })

  interface TaskRunRow {
    id: number | null
    task_key: string
    status: string
    started_at: string
    finished_at: string | null
    duration_sec?: number
    message: string
    created_at: string
  }

  interface Props {
    visible: boolean
    task?: ScheduledTaskRecord | null
    runs?: TaskRunRow[]
    loading?: boolean
  }

  interface Emits {
    (e: 'update:visible', value: boolean): void
    (e: 'refresh'): void
  }

  const props = withDefaults(defineProps<Props>(), {
    task: null,
    runs: () => [],
    loading: false
  })

  const emit = defineEmits<Emits>()

  const drawerVisible = computed({
    get: () => props.visible,
    set: (value) => emit('update:visible', value)
  })

  const drawerTitle = computed(() => {
    const name = String(props.task?.name || '')
    return name ? `运行记录 · ${name}` : '运行记录'
  })

  function getStrategyText(strategy?: string) {
    return strategy === 'daily' ? '每日执行' : '间隔执行'
  }

  function getScheduleText(task?: ScheduledTaskRecord | null) {
    if (!task) {
      return '-'
    }

    if (task.strategy === 'daily') {
      return task.daily_at ? `每天 ${task.daily_at}` : '每天执行'
    }

    return `每 ${Number(task.interval_sec || 0)} 秒`
  }

  function getRuntimeText(task?: ScheduledTaskRecord | null) {
    if (!task) {
      return '-'
    }

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

  function getRuntimeTagType(task?: ScheduledTaskRecord | null) {
    if (task?.running) {
      return 'warning' as const
    }

    switch (task?.last_status) {
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

  function getRunStatusText(status?: string) {
    switch (status) {
      case 'success':
        return '成功'
      case 'failed':
        return '失败'
      case 'running':
        return '运行中'
      default:
        return status || '-'
    }
  }

  function getRunStatusTagType(status?: string) {
    switch (status) {
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
</script>

<style scoped lang="scss">
  .drawer-panel {
    display: flex;
    flex-direction: column;
    gap: 18px;
  }

  .drawer-toolbar {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 16px;
    padding: 18px;
    border-radius: 12px;
    background: linear-gradient(135deg, rgb(16 185 129 / 10%), rgb(59 130 246 / 9%));
  }

  .drawer-heading h3 {
    margin: 0;
    color: var(--el-text-color-primary);
    font-size: 20px;
    font-weight: 700;
  }

  .drawer-heading p {
    margin: 6px 0 0;
    color: var(--el-text-color-secondary);
    line-height: 1.6;
  }

  .drawer-alert {
    margin-top: -4px;
  }

  .drawer-section {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .section-title {
    color: var(--el-text-color-primary);
    font-size: 15px;
    font-weight: 600;
  }

  .empty-tip {
    color: var(--el-text-color-secondary);
    font-size: 13px;
  }

  @media (max-width: 768px) {
    .drawer-toolbar {
      flex-direction: column;
      align-items: stretch;
    }
  }
</style>
