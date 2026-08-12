<template>
  <div class="art-full-height">
    <TicketSearch
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
        <template #subject="{ row }">
          <RouterLink :to="{ name: 'TicketDetail', params: { id: row.id } }" class="subject-link">
            <ArtSvgIcon icon="ri:message-2-line" class="subject-icon" />
            <span>{{ row.subject || '-' }}</span>
          </RouterLink>
        </template>

        <template #status="{ row }">
          <ElTag :type="getStatusTagType(row.status)" class="status-tag">
            <ArtSvgIcon :icon="getStatusIcon(row.status)" />
            <span>{{ getStatusText(row.status) }}</span>
          </ElTag>
        </template>

        <template #created_at="{ row }">
          <div class="time-cell">
            <ArtSvgIcon icon="ri:calendar-line" />
            <span>{{ formatDateTime(row.created_at) }}</span>
          </div>
        </template>

        <template #updated_at="{ row }">
          <div class="time-cell">
            <ArtSvgIcon icon="ri:time-line" />
            <span>{{ formatDateTime(row.updated_at) }}</span>
          </div>
        </template>

        <template #operation="{ row }">
          <div class="table-actions">
            <ArtButtonTable type="view" @click="goToDetail(row.id)" />
          </div>
        </template>
      </ArtTable>
    </ElCard>
  </div>
</template>

<script setup lang="ts">
  import type { TicketRecord } from '@/services/admin'
  import { fetchAdminTickets } from '@/services/admin'
  import { useTable } from '@/hooks/core/useTable'
  import TicketSearch from './modules/ticket-search.vue'

  defineOptions({ name: 'TicketList' })

  interface TicketSearchForm {
    status?: string
    user_id: string
    q: string
  }

  interface TicketTableParams extends Api.Common.CommonSearchParams, TicketSearchForm {}

  interface TicketTableRow {
    id: number | null
    user_id: number | null
    subject: string
    status: string
    resource_count: number
    last_reply_at: string | null
    last_reply_by: number | null
    last_reply_role: string
    closed_at: string | null
    created_at: string
    updated_at: string
  }

  const router = useRouter()
  const showSearchBar = ref(true)

  const searchForm = ref<TicketSearchForm>(createDefaultSearchForm())

  const {
    columnChecks,
    columns,
    data: tableData,
    loading,
    pagination,
    searchParams,
    getData,
    refreshData: fetchData,
    resetSearchParams,
    handleSizeChange: handlePageSizeChange,
    handleCurrentChange: handlePageCurrentChange
  } = useTable({
    core: {
      apiFn: fetchTicketTable,
      apiParams: { current: 1, size: 20, ...searchForm.value },
      columnsFactory: () => [
        { prop: 'id', label: 'ID', width: 90 },
        { prop: 'user_id', label: '用户 ID', width: 100 },
        { prop: 'subject', label: '工单标题', minWidth: 260, useSlot: true },
        { prop: 'status', label: '状态', width: 130, useSlot: true },
        { prop: 'created_at', label: '创建时间', minWidth: 180, useSlot: true },
        { prop: 'updated_at', label: '最后回复', minWidth: 180, useSlot: true },
        { prop: 'operation', label: '操作', width: 100, fixed: 'right', useSlot: true }
      ]
    }
  })

  function createDefaultSearchForm(): TicketSearchForm {
    return {
      status: undefined,
      user_id: '',
      q: ''
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

  function normalizeTicket(item?: TicketRecord): TicketTableRow {
    return {
      id: normalizeNullableNumber(item?.id),
      user_id: normalizeNullableNumber(item?.user_id),
      subject: String(item?.subject || ''),
      status: String(item?.status || ''),
      resource_count: Number(item?.resource_count || 0),
      last_reply_at: normalizeNullableString(item?.last_reply_at),
      last_reply_by: normalizeNullableNumber(item?.last_reply_by),
      last_reply_role: String(item?.last_reply_role || ''),
      closed_at: normalizeNullableString(item?.closed_at),
      created_at: String(item?.created_at || ''),
      updated_at: String(item?.updated_at || '')
    }
  }

  function normalizeUserId(value: string) {
    const trimmed = String(value || '').trim()
    if (!trimmed) {
      return undefined
    }

    const parsed = Number(trimmed)
    return Number.isFinite(parsed) && parsed > 0 ? parsed : undefined
  }

  function formatDateTime(value?: string | null) {
    if (!value) {
      return '-'
    }

    const date = new Date(value)
    return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN')
  }

  function getStatusText(status?: string) {
    switch (status) {
      case 'open':
        return '待处理'
      case 'waiting_user':
        return '等待回复'
      case 'waiting_admin':
        return '处理中'
      case 'closed':
        return '已关闭'
      default:
        return status || '-'
    }
  }

  function getStatusIcon(status?: string) {
    switch (status) {
      case 'open':
        return 'ri:error-warning-line'
      case 'waiting_user':
        return 'ri:time-line'
      case 'waiting_admin':
        return 'ri:loader-4-line'
      case 'closed':
        return 'ri:checkbox-circle-line'
      default:
        return 'ri:question-line'
    }
  }

  function getStatusTagType(status?: string) {
    switch (status) {
      case 'open':
        return 'danger' as const
      case 'waiting_user':
        return 'warning' as const
      case 'waiting_admin':
        return 'primary' as const
      case 'closed':
        return 'info' as const
      default:
        return 'info' as const
    }
  }

  async function fetchTicketTable(
    params: TicketTableParams,
    signal?: AbortSignal
  ): Promise<Api.Common.PaginatedResponse<TicketTableRow>> {
    const payload = await fetchAdminTickets(
      {
        limit: params.size,
        offset: (params.current - 1) * params.size,
        status: params.status || undefined,
        user_id: normalizeUserId(params.user_id),
        q: params.q.trim() || undefined
      },
      signal
    )
    const records = (payload.items || []).map(normalizeTicket)
    return {
      records,
      current: params.current,
      size: params.size,
      total: Number(payload.total ?? records.length)
    }
  }

  function handleSearch(params: TicketSearchForm) {
    searchForm.value = { ...searchForm.value, ...params }
    Object.assign(searchParams, params)
    getData()
  }

  async function handleReset() {
    searchForm.value = createDefaultSearchForm()
    await resetSearchParams()
  }

  function goToDetail(id: number | null) {
    if (!id) {
      return
    }

    router.push({ name: 'TicketDetail', params: { id: String(id) } })
  }
</script>

<style scoped lang="scss">
  .subject-link {
    display: inline-flex;
    gap: 8px;
    align-items: center;
    font-weight: 500;
    color: var(--el-color-primary);
    text-decoration: none;
  }

  .subject-link:hover {
    color: var(--el-color-primary-light-3);
  }

  .subject-icon {
    font-size: 15px;
  }

  .status-tag {
    gap: 6px;
  }

  .time-cell {
    display: flex;
    gap: 6px;
    align-items: center;
    color: var(--el-text-color-regular);
  }

  .table-actions {
    display: flex;
    justify-content: flex-end;
  }
</style>
