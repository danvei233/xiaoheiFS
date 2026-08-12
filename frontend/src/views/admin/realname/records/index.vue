<template>
  <div class="realname-records-page art-full-height">
    <ArtSearchBar
      v-show="showSearchBar"
      v-model="searchForm"
      :items="searchItems"
      :show-expand="false"
      :show-search="false"
      @reset="resetSearchParams"
    />

    <ElCard class="art-table-card" :style="{ marginTop: showSearchBar ? '12px' : '0' }">
      <ArtTableHeader
        v-model:columns="columnChecks"
        v-model:show-search-bar="showSearchBar"
        :loading="loading"
        @refresh="refreshData"
      />

      <ArtTable
        row-key="id"
        :loading="loading"
        :data="data"
        :columns="columns"
        :pagination="pagination"
        @pagination:size-change="handleSizeChange"
        @pagination:current-change="handleCurrentChange"
      >
        <template #real_name="{ row }">{{ maskName(row.real_name) }}</template>
        <template #id_number="{ row }">{{ maskIdNumber(row.id_number) }}</template>
        <template #status="{ row }">
          <ElTag :type="statusType(row.status)">{{ statusText(row.status) }}</ElTag>
        </template>
        <template #created_at="{ row }">{{ formatDate(row.created_at) }}</template>
        <template #verified_at="{ row }">{{ formatDate(row.verified_at) }}</template>
      </ArtTable>
    </ElCard>
  </div>
</template>

<script setup lang="ts">
  import { listRealNameRecords } from '@/services/admin'
  import type { RealNameVerification } from '@/services/types'
  import { useTable } from '@/hooks/core/useTable'

  defineOptions({ name: 'AdminRealnameRecords' })

  interface SearchForm {
    user_id?: string
    status?: string
  }

  interface TableParams extends SearchForm {
    current: number
    size: number
  }

  interface RecordRow {
    id: number | null
    user_id: number | null
    real_name: string
    id_number: string
    provider: string
    status: string
    created_at: string
    verified_at: string
  }

  const showSearchBar = ref(true)
  const searchForm = ref<SearchForm>({ user_id: '', status: undefined })
  const searchItems = computed(() => [
    {
      key: 'user_id',
      label: '用户 ID',
      type: 'input',
      props: {
        clearable: true,
        placeholder: '按用户 ID 搜索',
        // F8：逐键 onInput 直接 getData 会每键发一次请求（前一个被 Abort 但后端
        // 已执行 SQL）。输入走 300ms 防抖，状态下拉是单选事件保持即时。
        onInput: (value: string) => triggerFilterDebounced('user_id', value)
      }
    },
    {
      key: 'status',
      label: '状态',
      type: 'select',
      props: {
        clearable: true,
        onChange: (value?: string) => triggerFilter('status', value),
        options: [
          { label: '待审核', value: 'pending' },
          { label: '已通过', value: 'verified' },
          { label: '未通过', value: 'failed' }
        ]
      }
    }
  ])

  const fetchRecords = async (params: TableParams, signal?: AbortSignal) => {
    const userId = String(params.user_id ?? '').trim()
    const response = await listRealNameRecords(
      {
        limit: params.size,
        offset: (params.current - 1) * params.size,
        user_id: userId || undefined,
        status: params.status || undefined
      },
      signal
    )
    const rows = (response.data?.items ?? []).map(normalizeRecord)
    return { records: rows, total: Number(response.data?.total ?? rows.length) }
  }

  const {
    columns,
    columnChecks,
    data,
    loading,
    pagination,
    getData,
    getDataDebounced,
    searchParams,
    resetSearchParams,
    handleSizeChange,
    handleCurrentChange,
    refreshData
  } = useTable({
    core: {
      apiFn: fetchRecords,
      apiParams: { current: 1, size: 20, ...searchForm.value },
      columnsFactory: () => [
        { prop: 'id', label: 'ID', width: 80 },
        { prop: 'user_id', label: '用户 ID', width: 100 },
        { prop: 'real_name', label: '真实姓名', width: 120, useSlot: true },
        { prop: 'id_number', label: '证件号码', minWidth: 180, useSlot: true },
        { prop: 'provider', label: '服务商', minWidth: 200, showOverflowTooltip: true },
        { prop: 'status', label: '状态', width: 100, useSlot: true },
        { prop: 'created_at', label: '提交时间', minWidth: 180, useSlot: true },
        { prop: 'verified_at', label: '审核时间', minWidth: 180, useSlot: true }
      ]
    }
  })

  function normalizeRecord(item: RealNameVerification): RecordRow {
    const record = item as RealNameVerification & Record<string, unknown>
    const id = record.id ?? record.ID
    const userId = record.user_id ?? record.UserID
    return {
      id: id == null ? null : Number(id),
      user_id: userId == null ? null : Number(userId),
      real_name: String(record.real_name ?? record.RealName ?? ''),
      id_number: String(record.id_number ?? record.IDNumber ?? ''),
      provider: String(record.provider ?? record.Provider ?? ''),
      status: String(record.status ?? record.Status ?? ''),
      created_at: String(record.created_at ?? record.CreatedAt ?? ''),
      verified_at: String(record.verified_at ?? record.VerifiedAt ?? '')
    }
  }

  function triggerFilter(key: keyof SearchForm, value?: string): void {
    searchForm.value[key] = value
    Object.assign(searchParams, searchForm.value, { current: 1 })
    getData()
  }

  function triggerFilterDebounced(key: keyof SearchForm, value?: string): void {
    searchForm.value[key] = value
    Object.assign(searchParams, searchForm.value, { current: 1 })
    getDataDebounced()
  }

  function maskName(name: string): string {
    if (!name) return ''
    if (name.length <= 2) return `${name.slice(0, 1)}*`
    return `${name.slice(0, 1)}${'*'.repeat(name.length - 2)}${name.slice(-1)}`
  }

  function maskIdNumber(value: string): string {
    if (!value || value.length < 8 || value.includes('*')) return value || '-'
    return `${value.slice(0, 4)}********${value.slice(-4)}`
  }

  function statusText(status: string): string {
    if (status === 'pending') return '待审核'
    if (status === 'verified') return '已通过'
    if (status === 'failed') return '未通过'
    return status || '-'
  }

  function statusType(status: string): 'warning' | 'success' | 'danger' | 'info' {
    if (status === 'pending') return 'warning'
    if (status === 'verified') return 'success'
    if (status === 'failed') return 'danger'
    return 'info'
  }

  function formatDate(value: string): string {
    if (!value) return '-'
    const date = new Date(value)
    return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN')
  }
</script>

<style lang="scss" scoped>
  .realname-records-page {
    min-width: 0;
  }
</style>
