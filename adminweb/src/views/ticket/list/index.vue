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
        <template #left>
          <ElSpace wrap>
            <ElRadioGroup v-model="quickStatus" class="quick-status-group">
              <ElRadioButton label="all">全部</ElRadioButton>
              <ElRadioButton v-for="item in quickStatusTabs" :key="item.value" :label="item.value">
                {{ item.label }}
              </ElRadioButton>
            </ElRadioGroup>
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
        <template #subject="{ row }">
          <RouterLink :to="`/tickets/${row.id}`" class="subject-link">
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
  import type { TicketRecord } from '@/api/admin'
  import { fetchAdminTickets } from '@/api/admin'
  import { useTableColumns } from '@/hooks/core/useTableColumns'
  import TicketSearch from './modules/ticket-search.vue'

  defineOptions({ name: 'TicketList' })

  interface TicketSearchForm {
    status?: string
    user_id: string
    q: string
  }

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

  const quickStatusTabs = [
    { label: '待处理', value: 'open' },
    { label: '等待回复', value: 'waiting_user' },
    { label: '处理中', value: 'waiting_admin' },
    { label: '已关闭', value: 'closed' }
  ] as const

  const router = useRouter()
  const showSearchBar = ref(true)
  const loading = ref(false)

  const searchForm = ref<TicketSearchForm>(createDefaultSearchForm())
  const tableData = ref<TicketTableRow[]>([])

  const pagination = reactive({
    current: 1,
    size: 20,
    total: 0
  })

  const { columnChecks, columns } = useTableColumns<TicketTableRow>(() => [
    { prop: 'id', label: 'ID', width: 90 },
    { prop: 'user_id', label: '用户 ID', width: 100 },
    { prop: 'subject', label: '工单标题', minWidth: 260, useSlot: true },
    { prop: 'status', label: '状态', width: 130, useSlot: true },
    { prop: 'created_at', label: '创建时间', minWidth: 180, useSlot: true },
    { prop: 'updated_at', label: '最后回复', minWidth: 180, useSlot: true },
    { prop: 'operation', label: '操作', width: 100, fixed: 'right', useSlot: true }
  ])

  const quickStatus = computed({
    get: () => {
      const status = searchForm.value.status || ''
      return quickStatusTabs.some((item) => item.value === status) ? status : 'all'
    },
    set: (value: string) => {
      searchForm.value = {
        ...searchForm.value,
        status: value === 'all' ? undefined : value
      }
      pagination.current = 1
      fetchData()
    }
  })

  onMounted(() => {
    fetchData()
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

  async function fetchData() {
    loading.value = true

    try {
      const payload = await fetchAdminTickets({
        limit: pagination.size,
        offset: (pagination.current - 1) * pagination.size,
        status: searchForm.value.status || undefined,
        user_id: normalizeUserId(searchForm.value.user_id),
        q: searchForm.value.q.trim() || undefined
      })

      tableData.value = (payload.items || []).map((item) => normalizeTicket(item))
      pagination.total = typeof payload.total === 'number' ? payload.total : tableData.value.length
    } finally {
      loading.value = false
    }
  }

  function handleSearch(params: TicketSearchForm) {
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

  function goToDetail(id: number | null) {
    if (!id) {
      return
    }

    router.push(`/tickets/${id}`)
  }
</script>

<style scoped lang="scss">
  .quick-status-group {
    flex-wrap: wrap;
  }

  .subject-link {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    color: var(--el-color-primary);
    text-decoration: none;
    font-weight: 500;
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
    align-items: center;
    gap: 6px;
    color: var(--el-text-color-regular);
  }

  .table-actions {
    display: flex;
    justify-content: flex-end;
  }
</style>
