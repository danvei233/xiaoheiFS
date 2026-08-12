<template>
  <div class="audit-page art-full-height">
    <ArtSearchBar
      v-model="searchForm"
      :items="searchItems"
      :span="6"
      @reset="handleReset"
      @search="handleSearch"
    />

    <ElCard class="art-table-card">
      <ArtTableHeader v-model:columns="columnChecks" :loading="loading" @refresh="refreshData" />

      <ArtTable
        row-key="id"
        :loading="loading"
        :data="data"
        :columns="columns"
        :pagination="pagination"
        @pagination:size-change="handleSizeChange"
        @pagination:current-change="handleCurrentChange"
      >
        <template #action="{ row }">
          <ElTag type="warning" effect="plain">{{ row.action || '-' }}</ElTag>
        </template>

        <template #createdAt="{ row }">
          {{ formatDateTime(row.createdAt) }}
        </template>

        <template #detail="{ row }">
          <ElPopover v-if="row.detail !== '-'" placement="top-start" trigger="hover" :width="560">
            <template #reference>
              <span class="detail-preview">{{ row.detail }}</span>
            </template>
            <pre class="detail-content">{{ row.detail }}</pre>
          </ElPopover>
          <span v-else>-</span>
        </template>
      </ArtTable>
    </ElCard>
  </div>
</template>

<script setup lang="ts">
  import { useTable } from '@/hooks/core/useTable'
  import { listAuditLogs } from '@/services/admin'

  defineOptions({ name: 'AuditLogs' })

  interface AuditTableParams extends Api.Common.CommonSearchParams {
    keyword?: string
    status?: string
    range?: string[]
    action?: string
    user?: string
  }

  interface AuditRow {
    id: number | string | null
    actor: string
    action: string
    target: string
    createdAt: string
    detail: string
  }

  type AuditSource = Record<string, unknown>

  type AuditSearchForm = Pick<AuditTableParams, 'keyword' | 'status' | 'range' | 'action' | 'user'>

  const searchForm = ref<AuditSearchForm>({
    keyword: '',
    status: undefined,
    range: [],
    action: '',
    user: ''
  })

  const searchItems = [
    {
      key: 'keyword',
      label: '关键词',
      type: 'input',
      props: { clearable: true, placeholder: '搜索对象或详情' }
    },
    {
      key: 'status',
      label: '状态',
      type: 'select',
      props: { clearable: true, placeholder: '状态', options: [] }
    },
    {
      key: 'range',
      label: '日期范围',
      type: 'daterange',
      props: {
        type: 'daterange',
        clearable: true,
        rangeSeparator: '至',
        startPlaceholder: '开始日期',
        endPlaceholder: '结束日期'
      }
    },
    {
      key: 'action',
      label: '动作',
      type: 'input',
      props: { clearable: true, placeholder: '筛选操作类型' }
    },
    {
      key: 'user',
      label: '操作者',
      type: 'input',
      props: { clearable: true, placeholder: '筛选操作者' }
    }
  ]

  const {
    columns,
    columnChecks,
    data,
    loading,
    pagination,
    getData,
    searchParams,
    resetSearchParams,
    handleSizeChange,
    handleCurrentChange,
    refreshData
  } = useTable({
    core: {
      apiFn: fetchAuditLogs,
      apiParams: {
        current: 1,
        size: 20
      },
      columnsFactory: () => [
        { prop: 'id', label: 'ID', width: 90 },
        { prop: 'actor', label: '操作者', minWidth: 140, showOverflowTooltip: true },
        { prop: 'action', label: '动作', minWidth: 180, useSlot: true },
        { prop: 'target', label: '对象', minWidth: 180, showOverflowTooltip: true },
        { prop: 'createdAt', label: '时间', minWidth: 180, useSlot: true },
        { prop: 'detail', label: '详情', minWidth: 320, useSlot: true }
      ]
    }
  })

  async function fetchAuditLogs(
    params: AuditTableParams,
    signal?: AbortSignal
  ): Promise<Api.Common.PaginatedResponse<AuditRow>> {
    const response = await listAuditLogs(
      {
        limit: params.size,
        offset: (params.current - 1) * params.size,
        keyword: params.keyword,
        action: params.action,
        user: params.user
      },
      signal
    )
    const records = (response.data?.items || []).map(normalizeRow)

    return {
      records,
      current: params.current,
      size: params.size,
      total: Number(response.data?.total ?? records.length)
    }
  }

  function handleSearch(params: AuditSearchForm) {
    Object.assign(searchParams, params)
    getData()
  }

  const handleReset = () => resetSearchParams()

  function normalizeRow(row: AuditSource): AuditRow {
    const adminId = row.admin_id ?? row.AdminID
    const targetType = stringValue(row.target_type ?? row.TargetType)
    const targetId = stringValue(row.target_id ?? row.TargetID)

    return {
      id: normalizeId(row.id ?? row.ID),
      actor:
        stringValue(row.actor ?? row.Actor ?? row.user ?? row.User) ||
        (adminId ? `管理员#${adminId}` : '-'),
      action: stringValue(row.action ?? row.Action),
      target:
        stringValue(row.target ?? row.Target) ||
        (targetType || targetId ? `${targetType || '-'}:${targetId || '-'}` : '-'),
      createdAt: stringValue(row.created_at ?? row.CreatedAt),
      detail: stringifyDetail(row.meta ?? row.Meta ?? row.detail ?? row.Detail)
    }
  }

  function normalizeId(value: unknown): number | string | null {
    if (typeof value === 'number' || typeof value === 'string') {
      return value
    }
    return null
  }

  function stringValue(value: unknown): string {
    return typeof value === 'string' || typeof value === 'number' ? String(value) : ''
  }

  function stringifyDetail(value: unknown): string {
    if (value === null || value === undefined || value === '') {
      return '-'
    }
    if (typeof value === 'string') {
      return value
    }
    try {
      return JSON.stringify(value)
    } catch {
      return String(value)
    }
  }

  function formatDateTime(value: string): string {
    if (!value) {
      return '-'
    }
    const date = new Date(value)
    return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN')
  }
</script>

<style lang="scss" scoped>
  .detail-preview {
    display: inline-block;
    max-width: 100%;
    overflow: hidden;
    color: var(--art-gray-700);
    text-overflow: ellipsis;
    white-space: nowrap;
    cursor: help;
  }

  .detail-content {
    max-height: 280px;
    margin: 0;
    overflow: auto;
    font-size: 12px;
    line-height: 1.6;
    color: var(--art-gray-800);
    word-break: break-word;
    white-space: pre-wrap;
  }
</style>
