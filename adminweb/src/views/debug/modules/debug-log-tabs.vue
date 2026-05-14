<template>
  <ElCard shadow="never">
    <template #header>
      <div class="card-header">
        <div>
          <div class="card-title">调试日志</div>
          <div class="card-subtitle">按模块查看审计、自动化与同步日志</div>
        </div>
        <ElButton :loading="loading" @click="emit('refresh')">刷新日志</ElButton>
      </div>
    </template>

    <ElTabs v-model="tabModel">
      <ElTabPane label="审计日志" name="audit">
        <div class="tab-toolbar">
          <ElInput
            v-model.trim="auditKeyword"
            clearable
            placeholder="搜索操作或目标"
            class="filter-input"
          />
          <ElTag type="info">总数 {{ auditPagination.total || auditLogs.length }}</ElTag>
        </div>
        <ElTable v-loading="loading" :data="filteredAuditLogs" border size="small" row-key="id">
          <ElTableColumn prop="id" label="ID" width="80" />
          <ElTableColumn prop="admin_id" label="管理员 ID" width="110" />
          <ElTableColumn prop="action" label="操作" min-width="180" show-overflow-tooltip />
          <ElTableColumn prop="target_type" label="目标类型" width="120" />
          <ElTableColumn prop="target_id" label="目标 ID" width="120" />
          <ElTableColumn label="详情" min-width="260">
            <template #default="{ row }">
              <span class="text-ellipsis" :title="formatDetail(row.detail)">{{
                formatDetail(row.detail)
              }}</span>
            </template>
          </ElTableColumn>
          <ElTableColumn label="时间" min-width="180">
            <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
          </ElTableColumn>
        </ElTable>
        <div class="pagination-wrap">
          <ElPagination
            v-model:current-page="auditPager.current"
            v-model:page-size="auditPager.size"
            layout="total, sizes, prev, pager, next"
            :page-sizes="[10, 20, 50, 100]"
            :total="auditPagination.total"
            @size-change="emit('page-size-change', 'audit', $event)"
            @current-change="emit('page-current-change', 'audit', $event)"
          />
        </div>
      </ElTabPane>

      <ElTabPane label="自动化日志" name="automation">
        <div class="tab-toolbar">
          <ElInput
            v-model.trim="automationKeyword"
            clearable
            placeholder="按 API 或动作筛选"
            class="filter-input"
          />
          <ElTag type="info">总数 {{ automationPagination.total || automationLogs.length }}</ElTag>
        </div>
        <ElTable
          v-loading="loading"
          :data="filteredAutomationLogs"
          border
          size="small"
          row-key="id"
        >
          <ElTableColumn prop="id" label="ID" width="80" />
          <ElTableColumn prop="action" label="API" min-width="190" show-overflow-tooltip />
          <ElTableColumn label="协议" width="90">
            <template #default="{ row }"
              ><ElTag>{{ getProtocol(row) }}</ElTag></template
            >
          </ElTableColumn>
          <ElTableColumn label="连接" min-width="220">
            <template #default="{ row }">
              <span class="text-ellipsis" :title="getConnection(row)">{{
                getConnection(row)
              }}</span>
            </template>
          </ElTableColumn>
          <ElTableColumn prop="order_id" label="订单 ID" width="110" />
          <ElTableColumn label="结果" width="90">
            <template #default="{ row }"
              ><ElTag :type="row.success ? 'success' : 'danger'">{{
                row.success ? '成功' : '失败'
              }}</ElTag></template
            >
          </ElTableColumn>
          <ElTableColumn prop="message" label="消息" min-width="240" show-overflow-tooltip />
          <ElTableColumn label="时间" min-width="180">
            <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
          </ElTableColumn>
          <ElTableColumn label="详情" width="100" fixed="right">
            <template #default="{ row }">
              <ElButton link type="primary" @click="emit('view-automation-detail', row)"
                >查看</ElButton
              >
            </template>
          </ElTableColumn>
        </ElTable>
        <div class="pagination-wrap">
          <ElPagination
            v-model:current-page="automationPager.current"
            v-model:page-size="automationPager.size"
            layout="total, sizes, prev, pager, next"
            :page-sizes="[10, 20, 50, 100]"
            :total="automationPagination.total"
            @size-change="emit('page-size-change', 'automation', $event)"
            @current-change="emit('page-current-change', 'automation', $event)"
          />
        </div>
      </ElTabPane>

      <ElTabPane label="同步日志" name="sync">
        <div class="tab-toolbar">
          <ElInput
            v-model.trim="syncKeyword"
            clearable
            placeholder="搜索目标或消息"
            class="filter-input"
          />
          <ElTag type="info">总数 {{ syncPagination.total || syncLogs.length }}</ElTag>
        </div>
        <ElTable v-loading="loading" :data="filteredSyncLogs" border size="small" row-key="id">
          <ElTableColumn prop="id" label="ID" width="80" />
          <ElTableColumn prop="target" label="目标" min-width="180" show-overflow-tooltip />
          <ElTableColumn prop="mode" label="模式" width="110" />
          <ElTableColumn label="状态" width="110">
            <template #default="{ row }"
              ><ElTag :type="getSyncStatusTagType(row.status)">{{
                row.status || '-'
              }}</ElTag></template
            >
          </ElTableColumn>
          <ElTableColumn prop="message" label="消息" min-width="280" show-overflow-tooltip />
          <ElTableColumn label="时间" min-width="180">
            <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
          </ElTableColumn>
        </ElTable>
        <div class="pagination-wrap">
          <ElPagination
            v-model:current-page="syncPager.current"
            v-model:page-size="syncPager.size"
            layout="total, sizes, prev, pager, next"
            :page-sizes="[10, 20, 50, 100]"
            :total="syncPagination.total"
            @size-change="emit('page-size-change', 'sync', $event)"
            @current-change="emit('page-current-change', 'sync', $event)"
          />
        </div>
      </ElTabPane>
    </ElTabs>
  </ElCard>
</template>

<script setup lang="ts">
  import type {
    AdminAuditLogRecord,
    AutomationLogRecord,
    IntegrationSyncLogRecord
  } from '@/api/admin'

  defineOptions({ name: 'DebugLogTabs' })

  type LogTabKey = 'audit' | 'automation' | 'sync'

  interface PaginationState {
    current: number
    size: number
    total: number
  }

  interface Props {
    loading?: boolean
    activeTab: LogTabKey
    auditLogs: AdminAuditLogRecord[]
    automationLogs: AutomationLogRecord[]
    syncLogs: IntegrationSyncLogRecord[]
    auditPagination: PaginationState
    automationPagination: PaginationState
    syncPagination: PaginationState
  }

  interface Emits {
    (e: 'update:activeTab', value: LogTabKey): void
    (e: 'refresh'): void
    (e: 'page-size-change', type: LogTabKey, size: number): void
    (e: 'page-current-change', type: LogTabKey, page: number): void
    (e: 'view-automation-detail', record: AutomationLogRecord): void
  }

  const props = withDefaults(defineProps<Props>(), { loading: false })
  const emit = defineEmits<Emits>()

  const auditKeyword = ref('')
  const automationKeyword = ref('')
  const syncKeyword = ref('')

  const tabModel = computed({
    get: () => props.activeTab,
    set: (value) => emit('update:activeTab', value)
  })

  const auditPager = computed(() => props.auditPagination)
  const automationPager = computed(() => props.automationPagination)
  const syncPager = computed(() => props.syncPagination)

  const filteredAuditLogs = computed(() => {
    const keyword = auditKeyword.value.trim().toLowerCase()
    if (!keyword) return props.auditLogs
    return props.auditLogs.filter((item) => {
      const action = String(item.action || '').toLowerCase()
      const target =
        `${String(item.target_type || '')}:${String(item.target_id || '')}`.toLowerCase()
      return action.includes(keyword) || target.includes(keyword)
    })
  })

  const filteredAutomationLogs = computed(() => {
    const keyword = automationKeyword.value.trim().toLowerCase()
    if (!keyword) return props.automationLogs
    return props.automationLogs.filter((item) =>
      String(item.action || '')
        .toLowerCase()
        .includes(keyword)
    )
  })

  const filteredSyncLogs = computed(() => {
    const keyword = syncKeyword.value.trim().toLowerCase()
    if (!keyword) return props.syncLogs
    return props.syncLogs.filter((item) => {
      const target = String(item.target || '').toLowerCase()
      const message = String(item.message || '').toLowerCase()
      return target.includes(keyword) || message.includes(keyword)
    })
  })

  function formatDateTime(value?: string) {
    if (!value) return '-'
    const date = new Date(value)
    return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN')
  }

  function formatDetail(detail: Record<string, unknown> | string | undefined) {
    if (!detail) return '-'
    if (typeof detail === 'string') return detail
    try {
      return JSON.stringify(detail)
    } catch {
      return String(detail)
    }
  }

  function parsePayload(payload: unknown): Record<string, unknown> {
    if (!payload) return {}
    if (typeof payload === 'string') {
      try {
        return JSON.parse(payload) as Record<string, unknown>
      } catch {
        return { body: payload }
      }
    }
    return typeof payload === 'object' ? (payload as Record<string, unknown>) : { body: payload }
  }

  function findHeaderValue(headers: unknown, targetKey: string) {
    if (!headers || typeof headers !== 'object') return ''
    const matched = Object.entries(headers as Record<string, unknown>).find(
      ([key]) => key.toLowerCase() === targetKey.toLowerCase()
    )
    return matched?.[1] == null ? '' : String(matched[1])
  }

  function getProtocol(record: AutomationLogRecord) {
    const request = parsePayload(record.request_json)
    const method = String(request.method || '')
      .trim()
      .toUpperCase()
    if (method) return method
    const transport = findHeaderValue(request.headers, 'x-transport').trim()
    return transport ? transport.toUpperCase() : 'UNKNOWN'
  }

  function getConnection(record: AutomationLogRecord) {
    const request = parsePayload(record.request_json)
    const pluginId = findHeaderValue(request.headers, 'x-plugin-id').trim()
    const instanceId = findHeaderValue(request.headers, 'x-plugin-instance-id').trim()
    if (pluginId || instanceId) return `${pluginId || '-'} / ${instanceId || '-'}`
    const urlText = String(request.url || '').trim()
    if (!urlText) return '-'
    try {
      return new URL(urlText).host || urlText
    } catch {
      return urlText
    }
  }

  function getSyncStatusTagType(status?: string) {
    if (status === 'success') return 'success' as const
    if (status === 'failed' || status === 'error') return 'danger' as const
    return 'info' as const
  }
</script>

<style scoped lang="scss">
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

  .card-subtitle {
    margin-top: 4px;
    color: var(--el-text-color-secondary);
    font-size: 13px;
  }

  .tab-toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    margin-bottom: 14px;
  }

  .filter-input {
    width: 280px;
    max-width: 100%;
  }

  .text-ellipsis {
    display: inline-block;
    max-width: 100%;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .pagination-wrap {
    display: flex;
    justify-content: flex-end;
    margin-top: 14px;
  }

  @media (max-width: 768px) {
    .card-header,
    .tab-toolbar {
      flex-direction: column;
      align-items: flex-start;
    }

    .pagination-wrap {
      justify-content: flex-start;
      overflow-x: auto;
    }
  }
</style>
