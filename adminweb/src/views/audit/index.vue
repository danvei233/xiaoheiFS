<template>
  <div class="art-full-height">
    <ElCard v-loading="loading" class="art-table-card">
      <template #header>
        <div class="page-header">
          <div>
            <div class="page-title">Audit Logs</div>
            <div class="page-subtitle">
              Review sensitive system operations and security-related administrator actions.
            </div>
          </div>

          <div class="page-actions">
            <ElButton v-if="canView" :disabled="loading" @click="fetchData">Refresh</ElButton>
          </div>
        </div>
      </template>

      <ElEmpty v-if="!canView" description="You do not have permission to view audit logs." />

      <template v-else>
        <div class="toolbar">
          <ElInput
            v-model="filters.keyword"
            clearable
            placeholder="Search target or detail"
            class="toolbar-search"
          />

          <ElInput
            v-model="filters.action"
            clearable
            placeholder="Filter action"
            class="toolbar-input"
          />

          <ElInput
            v-model="filters.user"
            clearable
            placeholder="Filter operator"
            class="toolbar-input"
          />
        </div>

        <ArtTableHeader
          v-model:columns="columnChecks"
          :showSearchBar="false"
          :loading="loading"
          @refresh="fetchData"
        />

        <ArtTable
          row-key="id"
          :loading="loading"
          :data="filteredRows"
          :columns="columns"
          :pagination="pagination"
          @pagination:size-change="handlePageSizeChange"
          @pagination:current-change="handlePageCurrentChange"
        >
          <template #action="{ row }">
            <ElTag type="warning" effect="plain">{{ row.action || '-' }}</ElTag>
          </template>

          <template #target="{ row }">
            <span>{{ row.target || '-' }}</span>
          </template>

          <template #created_at="{ row }">
            {{ formatDateTime(row.created_at) }}
          </template>

          <template #meta="{ row }">
            <ElPopover
              v-if="row.meta && row.meta !== '-'"
              placement="top-start"
              trigger="hover"
              :width="560"
            >
              <template #reference>
                <span class="meta-preview">{{ row.meta }}</span>
              </template>
              <pre class="meta-popover">{{ row.meta }}</pre>
            </ElPopover>
            <span v-else>-</span>
          </template>
        </ArtTable>
      </template>
    </ElCard>
  </div>
</template>

<script setup lang="ts">
  import type { AdminAuditLogRecord } from '@/api/admin'
  import { fetchAdminAuditLogs, hasAdminPermission } from '@/api/admin'
  import { useTableColumns } from '@/hooks/core/useTableColumns'
  import { useUserStore } from '@/store/modules/user'

  defineOptions({ name: 'AuditLogsPage' })

  interface AuditRow {
    id: number | null
    actor: string
    action: string
    target: string
    meta: string
    created_at: string
  }

  interface FilterState {
    keyword: string
    action: string
    user: string
  }

  const userStore = useUserStore()
  const { info } = storeToRefs(userStore)

  const loading = ref(false)
  const initialized = ref(false)
  const tableData = ref<AuditRow[]>([])

  const filters = reactive<FilterState>({
    keyword: '',
    action: '',
    user: ''
  })

  const pagination = reactive({
    current: 1,
    size: 20,
    total: 0
  })

  const { columnChecks, columns } = useTableColumns<AuditRow>(() => [
    { prop: 'id', label: 'ID', width: 80 },
    { prop: 'actor', label: 'Operator', minWidth: 140, showOverflowTooltip: true },
    { prop: 'action', label: 'Action', minWidth: 160, useSlot: true },
    { prop: 'target', label: 'Target', minWidth: 180, useSlot: true },
    { prop: 'created_at', label: 'Created At', minWidth: 180, useSlot: true },
    { prop: 'meta', label: 'Detail', minWidth: 320, useSlot: true }
  ])

  const canView = computed(() => hasAdminPermission(info.value?.buttons, ['audit_log.view']))

  const filteredRows = computed(() => {
    const keyword = filters.keyword.trim().toLowerCase()
    const action = filters.action.trim().toLowerCase()
    const user = filters.user.trim().toLowerCase()

    return tableData.value.filter((row) => {
      if (keyword) {
        const matchKeyword = [row.target, row.meta].some((value) =>
          String(value || '')
            .toLowerCase()
            .includes(keyword)
        )
        if (!matchKeyword) {
          return false
        }
      }

      if (
        action &&
        !String(row.action || '')
          .toLowerCase()
          .includes(action)
      ) {
        return false
      }

      if (
        user &&
        !String(row.actor || '')
          .toLowerCase()
          .includes(user)
      ) {
        return false
      }

      return true
    })
  })

  watch(
    canView,
    (value) => {
      if (value && !initialized.value) {
        initialized.value = true
        fetchData()
      }
    },
    { immediate: true }
  )

  function normalizeNullableNumber(value: unknown): number | null {
    if (value === '' || value === null || value === undefined) {
      return null
    }

    const parsed = Number(value)
    return Number.isFinite(parsed) ? parsed : null
  }

  function stringifyDetail(detail: unknown) {
    if (detail === null || detail === undefined) {
      return '-'
    }

    if (typeof detail === 'string') {
      return detail || '-'
    }

    try {
      return JSON.stringify(detail)
    } catch {
      return String(detail)
    }
  }

  function normalizeRow(item?: AdminAuditLogRecord): AuditRow {
    const adminId = normalizeNullableNumber(item?.admin_id)
    const targetType = String(item?.target_type || '').trim()
    const targetId = String(item?.target_id || '').trim()

    return {
      id: normalizeNullableNumber(item?.id),
      actor: adminId ? `Admin #${adminId}` : '-',
      action: String(item?.action || ''),
      target: targetType || targetId ? `${targetType || '-'}:${targetId || '-'}` : '-',
      meta: stringifyDetail(item?.detail),
      created_at: String(item?.created_at || '')
    }
  }

  function formatDateTime(value?: string | null) {
    if (!value) {
      return '-'
    }

    const date = new Date(value)
    return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN')
  }

  async function fetchData() {
    if (!canView.value) {
      return
    }

    loading.value = true

    try {
      const payload = await fetchAdminAuditLogs({
        limit: pagination.size,
        offset: (pagination.current - 1) * pagination.size
      })
      tableData.value = (payload.items || []).map((item) => normalizeRow(item))
      pagination.total = Number(payload.total || tableData.value.length)
    } finally {
      loading.value = false
    }
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
</script>

<style scoped lang="scss">
  .page-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
  }

  .page-title {
    color: var(--el-text-color-primary);
    font-size: 20px;
    font-weight: 700;
  }

  .page-subtitle {
    margin-top: 6px;
    color: var(--el-text-color-secondary);
    font-size: 13px;
    line-height: 1.6;
  }

  .page-actions,
  .toolbar {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .toolbar {
    margin-bottom: 12px;
    flex-wrap: wrap;
  }

  .toolbar-search {
    width: min(360px, 100%);
  }

  .toolbar-input {
    width: 220px;
  }

  .meta-preview {
    display: inline-block;
    max-width: 100%;
    overflow: hidden;
    color: var(--el-text-color-regular);
    text-overflow: ellipsis;
    white-space: nowrap;
    cursor: help;
  }

  .meta-popover {
    margin: 0;
    max-height: 280px;
    overflow: auto;
    white-space: pre-wrap;
    word-break: break-word;
    font-size: 12px;
    line-height: 1.6;
  }

  @media (max-width: 768px) {
    .page-header {
      align-items: flex-start;
      flex-direction: column;
    }

    .page-actions,
    .toolbar,
    .toolbar-search,
    .toolbar-input {
      width: 100%;
    }

    .page-actions {
      flex-wrap: wrap;
    }
  }
</style>
