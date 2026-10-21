<template>
  <div class="art-full-height">
    <RecordSearch
      v-if="canView"
      v-show="showSearchBar"
      v-model="searchForm"
      @search="handleSearch"
      @reset="handleReset"
    />

    <ElCard
      v-loading="loading"
      class="art-table-card"
      :style="{ marginTop: canView && showSearchBar ? '12px' : '0' }"
    >
      <template #header>
        <div class="page-header">
          <div>
            <div class="page-title">实名认证记录</div>
            <div class="page-subtitle">查看实名审核结果、认证服务商和敏感信息脱敏后的记录。</div>
          </div>

          <div class="page-actions">
            <ElButton v-if="canView" :disabled="loading" @click="fetchData">刷新</ElButton>
          </div>
        </div>
      </template>

      <ElEmpty v-if="!canView" description="当前账号没有查看实名认证记录的权限。" />

      <template v-else>
        <ArtTableHeader
          v-model:columns="columnChecks"
          v-model:showSearchBar="showSearchBar"
          :loading="loading"
          @refresh="fetchData"
        />

        <ArtTable
          row-key="id"
          :loading="loading"
          :data="tableData"
          :columns="columns"
          :pagination="pagination"
          @pagination:size-change="handlePageSizeChange"
          @pagination:current-change="handlePageCurrentChange"
        >
          <template #real_name="{ row }">
            {{ maskName(row.real_name) }}
          </template>

          <template #id_number="{ row }">
            {{ maskIdNumber(row.id_number) }}
          </template>

          <template #status="{ row }">
            <ElTag :type="getStatusTagType(row.status)">
              {{ getStatusText(row.status) }}
            </ElTag>
          </template>

          <template #created_at="{ row }">
            {{ formatDateTime(row.created_at) }}
          </template>

          <template #verified_at="{ row }">
            {{ formatDateTime(row.verified_at) }}
          </template>
        </ArtTable>
      </template>
    </ElCard>
  </div>
</template>

<script setup lang="ts">
  import type { RealNameVerification } from '@/api/admin'
  import { fetchRealNameRecords, hasAdminPermission } from '@/api/admin'
  import { useTableColumns } from '@/hooks/core/useTableColumns'
  import { useUserStore } from '@/store/modules/user'
  import RecordSearch from './modules/record-search.vue'

  defineOptions({ name: 'RealnameRecordsPage' })

  interface RecordSearchForm {
    user_id: string
    status?: string
  }

  interface RecordTableRow {
    id: number | null
    user_id: number | null
    real_name: string
    id_number: string
    status: string
    provider: string
    reason: string
    created_at: string
    verified_at: string | null
  }

  const userStore = useUserStore()
  const { info } = storeToRefs(userStore)

  const showSearchBar = ref(true)
  const loading = ref(false)

  const searchForm = ref<RecordSearchForm>(createDefaultSearchForm())
  const tableData = ref<RecordTableRow[]>([])

  const pagination = reactive({
    current: 1,
    size: 20,
    total: 0
  })

  const canView = computed(() => hasAdminPermission(info.value?.buttons, ['realname.list']))

  const { columnChecks, columns } = useTableColumns<RecordTableRow>(() => [
    { prop: 'id', label: 'ID', width: 80 },
    { prop: 'user_id', label: '用户 ID', width: 100 },
    { prop: 'real_name', label: '真实姓名', width: 120, useSlot: true },
    { prop: 'id_number', label: '证件号码', minWidth: 180, useSlot: true },
    { prop: 'provider', label: '服务商', minWidth: 200, showOverflowTooltip: true },
    { prop: 'status', label: '状态', width: 100, useSlot: true },
    { prop: 'created_at', label: '提交时间', minWidth: 180, useSlot: true },
    { prop: 'verified_at', label: '审核时间', minWidth: 180, useSlot: true }
  ])

  onMounted(() => {
    fetchData()
  })

  function createDefaultSearchForm(): RecordSearchForm {
    return {
      user_id: '',
      status: undefined
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

  function normalizeUserId(value: string) {
    const trimmed = String(value || '').trim()
    if (!trimmed) {
      return undefined
    }

    const parsed = Number(trimmed)
    return Number.isFinite(parsed) && parsed > 0 ? parsed : undefined
  }

  function normalizeRecord(item?: RealNameVerification): RecordTableRow {
    return {
      id: normalizeNullableNumber(item?.id),
      user_id: normalizeNullableNumber(item?.user_id),
      real_name: String(item?.real_name || ''),
      id_number: String(item?.id_number || ''),
      status: String(item?.status || ''),
      provider: String(item?.provider || ''),
      reason: String(item?.reason || ''),
      created_at: String(item?.created_at || ''),
      verified_at: normalizeNullableString(item?.verified_at)
    }
  }

  function maskName(name: string) {
    if (!name) {
      return ''
    }

    if (name.length <= 2) {
      return `${name.slice(0, 1)}*`
    }

    return `${name.slice(0, 1)}${'*'.repeat(name.length - 2)}${name.slice(-1)}`
  }

  function maskIdNumber(idNumber: string) {
    if (!idNumber || idNumber.length < 8) {
      return idNumber || '-'
    }

    if (idNumber.includes('*')) {
      return idNumber
    }

    return `${idNumber.slice(0, 4)}********${idNumber.slice(-4)}`
  }

  function getStatusText(status?: string) {
    switch (status) {
      case 'pending':
        return '待审核'
      case 'verified':
        return '已通过'
      case 'failed':
        return '未通过'
      default:
        return status || '-'
    }
  }

  function getStatusTagType(status?: string) {
    switch (status) {
      case 'pending':
        return 'warning' as const
      case 'verified':
        return 'success' as const
      case 'failed':
        return 'danger' as const
      default:
        return 'info' as const
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
      const payload = await fetchRealNameRecords({
        limit: pagination.size,
        offset: (pagination.current - 1) * pagination.size,
        user_id: normalizeUserId(searchForm.value.user_id)
      })

      let rows = (payload.items || []).map((item) => normalizeRecord(item))

      if (searchForm.value.status) {
        rows = rows.filter((item) => item.status === searchForm.value.status)
      }

      tableData.value = rows
      pagination.total = searchForm.value.status
        ? rows.length
        : Number(payload.total || rows.length)
    } finally {
      loading.value = false
    }
  }

  function handleSearch(params: RecordSearchForm) {
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
</script>

<style scoped lang="scss">
  .page-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 16px;
  }

  .page-title {
    color: var(--el-text-color-primary);
    font-size: 18px;
    font-weight: 700;
  }

  .page-subtitle {
    margin-top: 6px;
    color: var(--el-text-color-secondary);
    font-size: 13px;
    line-height: 1.7;
  }

  .page-actions {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  @media (max-width: 768px) {
    .page-header {
      flex-direction: column;
      align-items: stretch;
    }
  }
</style>
