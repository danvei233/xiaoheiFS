<template>
  <div class="api-keys-page art-full-height">
    <ArtSearchBar
      v-show="showSearchBar"
      v-model="searchForm"
      :items="searchItems"
      :show-expand="false"
      @search="handleSearch"
      @reset="resetSearchParams"
    />

    <ElCard class="art-table-card" :style="{ marginTop: showSearchBar ? '12px' : '0' }">
      <ArtTableHeader
        v-model:columns="columnChecks"
        v-model:show-search-bar="showSearchBar"
        :loading="loading"
        @refresh="refreshData"
      >
        <template #left>
          <ElSpace wrap>
            <ElButton v-if="canCreate" type="primary" @click="openCreate">
              <ArtSvgIcon icon="ri:add-line" />
              创建 API Key
            </ElButton>
          </ElSpace>
        </template>
      </ArtTableHeader>

      <ArtTable
        row-key="id"
        :loading="loading"
        :data="data"
        :columns="columns"
        :pagination="pagination"
        @pagination:size-change="handleSizeChange"
        @pagination:current-change="handleCurrentChange"
      >
        <template #status="{ row }">
          <ElTag :type="row.status === 'active' ? 'success' : 'info'">
            {{ row.status || '-' }}
          </ElTag>
        </template>
        <template #operation="{ row }">
          <ElSpace>
            <ElTooltip content="复制 Key Hash">
              <ElButton circle plain aria-label="复制 Key Hash" @click="copyText(row.key_hash)">
                <ArtSvgIcon icon="ri:file-copy-line" />
              </ElButton>
            </ElTooltip>
            <ElTooltip v-if="canUpdate" content="切换状态">
              <ElButton circle plain aria-label="切换状态" @click="toggleStatus(row)">
                <ArtSvgIcon icon="ri:toggle-line" />
              </ElButton>
            </ElTooltip>
          </ElSpace>
        </template>
      </ArtTable>
    </ElCard>

    <ElDialog v-model="createVisible" title="创建 API Key" width="520px" destroy-on-close>
      <ArtForm
        ref="createFormRef"
        v-model="createForm"
        :items="createItems"
        :rules="createRules"
        :show-reset="false"
        :show-submit="false"
        :span="24"
        label-position="top"
      />
      <template #footer>
        <ElButton @click="createVisible = false">取消</ElButton>
        <ElButton type="primary" :loading="creating" @click="submitCreate">创建</ElButton>
      </template>
    </ElDialog>

    <ElDialog v-model="createdKeyVisible" title="API Key" width="560px" destroy-on-close>
      <ElAlert
        class="created-key-tip"
        type="warning"
        :closable="false"
        show-icon
        title="此密钥只显示一次，请立即妥善保存。"
      />
      <ElInput :model-value="createdApiKey" readonly type="textarea" :rows="4" />
      <template #footer>
        <ElButton @click="createdKeyVisible = false">关闭</ElButton>
        <ElButton type="primary" @click="copyText(createdApiKey)">复制密钥</ElButton>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import type { FormRules } from 'element-plus'
  import {
    createApiKey,
    listApiKeys,
    listPermissionGroups,
    updateApiKeyStatus
  } from '@/services/admin'
  import type { PermissionGroup } from '@/services/types'
  import { useTable } from '@/hooks/core/useTable'
  import { useAdminPermissions } from '../_shared/settings'

  defineOptions({ name: 'AdminSettingsApiKeys' })

  interface SearchForm {
    keyword?: string
    status?: string
  }

  interface TableParams extends SearchForm {
    current: number
    size: number
  }

  interface ApiKeyRow {
    id: number | null
    name: string
    permission_group_id: number | null
    permission_group_name: string
    key_hash: string
    status: string
  }

  type PermissionGroupCompat = PermissionGroup & { ID?: number; Name?: string }

  interface CreateForm extends Record<string, unknown> {
    name: string
    permission_group_id: number | null
  }

  interface ArtFormExpose {
    validate: () => Promise<boolean> | undefined
  }

  const showSearchBar = ref(true)
  const searchForm = ref<SearchForm>({ keyword: '', status: undefined })
  const permissionGroups = ref<PermissionGroup[]>([])
  const createVisible = ref(false)
  const createdKeyVisible = ref(false)
  const creating = ref(false)
  const createdApiKey = ref('')
  const createFormRef = ref<ArtFormExpose>()
  const createForm = reactive<CreateForm>({ name: '', permission_group_id: null })
  const { hasPermission } = useAdminPermissions()
  const canCreate = hasPermission('api_key.create')
  const canUpdate = hasPermission('api_key.update')

  const searchItems = computed(() => [
    {
      key: 'keyword',
      label: '关键词',
      type: 'input',
      props: { clearable: true, placeholder: 'Key Hash' }
    },
    {
      key: 'status',
      label: '状态',
      type: 'select',
      props: {
        clearable: true,
        options: [
          { label: 'active', value: 'active' },
          { label: 'disabled', value: 'disabled' }
        ]
      }
    }
  ])

  const createItems = computed(() => [
    { key: 'name', label: '名称', type: 'input' },
    {
      key: 'permission_group_id',
      label: '权限组',
      type: 'select',
      props: {
        placeholder: '请选择权限组',
        options: permissionGroups.value
          .map((group) => ({
            label: String(group.name ?? (group as PermissionGroupCompat).Name ?? ''),
            value: Number(group.id ?? (group as PermissionGroupCompat).ID)
          }))
          .filter((item) => Number.isFinite(item.value))
      }
    }
  ])
  const createRules: FormRules<CreateForm> = {
    permission_group_id: [{ required: true, message: '请选择权限组', trigger: 'change' }]
  }

  const permissionGroupMap = computed(() => {
    const map = new Map<number, string>()
    permissionGroups.value.forEach((group) => {
      const compat = group as PermissionGroupCompat
      const id = Number(compat.id ?? compat.ID)
      if (Number.isFinite(id)) map.set(id, String(compat.name ?? compat.Name ?? ''))
    })
    return map
  })

  const fetchApiKeys = async (params: TableParams, signal?: AbortSignal) => {
    const keyword = String(params.keyword ?? '')
      .trim()
      .toLowerCase()
    if (!keyword && !params.status) {
      const response = await listApiKeys(
        {
          limit: params.size,
          offset: (params.current - 1) * params.size
        },
        signal
      )
      const items = (response.data?.items ?? []).map(normalizeRow)
      return { records: items, total: response.data?.total ?? items.length }
    }

    let items = await fetchAllApiKeys()
    if (keyword) items = items.filter((item) => item.key_hash.toLowerCase().includes(keyword))
    if (params.status) items = items.filter((item) => item.status === params.status)
    const start = (params.current - 1) * params.size
    return {
      records: items.slice(start, start + params.size),
      total: items.length
    }
  }

  async function fetchAllApiKeys(): Promise<ApiKeyRow[]> {
    const rows: ApiKeyRow[] = []
    const limit = 100
    let offset = 0
    let total = 0

    do {
      const response = await listApiKeys({ limit, offset })
      const pageRows = (response.data?.items ?? []).map(normalizeRow)
      rows.push(...pageRows)
      total = Number(response.data?.total ?? rows.length)
      offset += pageRows.length
      if (pageRows.length === 0) break
    } while (offset < total)

    return rows
  }

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
    refreshData,
    refreshCreate,
    refreshUpdate
  } = useTable({
    core: {
      apiFn: fetchApiKeys,
      immediate: false,
      apiParams: { current: 1, size: 20, ...searchForm.value },
      columnsFactory: () => [
        { prop: 'id', label: 'ID', width: 90 },
        { prop: 'name', label: '名称', minWidth: 160, showOverflowTooltip: true },
        { prop: 'permission_group_name', label: '权限组', minWidth: 180 },
        { prop: 'key_hash', label: 'Key Hash', minWidth: 260, showOverflowTooltip: true },
        { prop: 'status', label: '状态', width: 120, useSlot: true },
        { prop: 'operation', label: '操作', width: 130, fixed: 'right', useSlot: true }
      ]
    }
  })

  onMounted(async () => {
    const response = await listPermissionGroups()
    permissionGroups.value = response.data?.items ?? []
    getData()
  })

  function normalizeRow(record: Record<string, unknown>): ApiKeyRow {
    const idValue = record.id ?? record.ID
    const groupValue = record.permission_group_id ?? record.PermissionGroupID
    const groupId = groupValue == null ? null : Number(groupValue)
    return {
      id: idValue == null ? null : Number(idValue),
      name: String(record.name ?? record.Name ?? ''),
      permission_group_id: Number.isFinite(groupId) ? groupId : null,
      permission_group_name:
        groupId !== null ? String(permissionGroupMap.value.get(groupId) ?? '') : '',
      key_hash: String(record.key_hash ?? record.KeyHash ?? ''),
      status: String(record.status ?? record.Status ?? '')
    }
  }

  function handleSearch(params: SearchForm): void {
    Object.assign(searchParams, params)
    getData()
  }

  function openCreate(): void {
    Object.assign(createForm, { name: '', permission_group_id: null })
    createVisible.value = true
  }

  async function submitCreate(): Promise<void> {
    const valid = await createFormRef.value?.validate()?.catch(() => false)
    if (!valid) return
    creating.value = true
    try {
      const response = await createApiKey({
        name: createForm.name.trim(),
        permission_group_id: createForm.permission_group_id
      })
      createdApiKey.value = String(response.data?.api_key ?? '')
      createVisible.value = false
      createdKeyVisible.value = true
      await refreshCreate()
    } finally {
      creating.value = false
    }
  }

  async function toggleStatus(row: ApiKeyRow): Promise<void> {
    if (row.id === null) return
    await ElMessageBox.confirm('确认切换此 API Key 的状态吗？', '切换状态', { type: 'warning' })
    await updateApiKeyStatus(row.id, {
      status: row.status === 'active' ? 'disabled' : 'active'
    })
    ElMessage.success('状态已更新')
    await refreshUpdate()
  }

  async function copyText(value: string): Promise<void> {
    if (!value) return
    await navigator.clipboard.writeText(value)
    ElMessage.success('已复制')
  }
</script>

<style lang="scss" scoped>
  .created-key-tip {
    margin-bottom: 16px;
  }
</style>
