<template>
  <div class="art-full-height">
    <ApiKeySearch
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
            <div class="page-title">API Keys</div>
            <div class="page-subtitle">Create and manage admin API keys.</div>
          </div>

          <div class="page-actions">
            <ElButton v-if="canCreate" type="primary" @click="openCreate">Create API Key</ElButton>
          </div>
        </div>
      </template>

      <ElEmpty v-if="!canView" description="You do not have permission to view API keys." />

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
          <template #status="{ row }">
            <ElTag :type="getStatusTagType(row.status)">
              {{ row.status || '-' }}
            </ElTag>
          </template>

          <template #operation="{ row }">
            <div class="table-actions">
              <ElButton link type="primary" @click="copyKeyHash(row)">Copy</ElButton>
              <ElButton
                v-if="canUpdate"
                link
                :type="row.status === 'active' ? 'danger' : 'primary'"
                @click="toggleStatus(row)"
              >
                {{ row.status === 'active' ? 'Disable' : 'Enable' }}
              </ElButton>
            </div>
          </template>
        </ArtTable>
      </template>
    </ElCard>

    <ElDialog
      v-model="createVisible"
      title="Create API Key"
      width="460px"
      destroy-on-close
      align-center
    >
      <ElForm label-position="top">
        <ElFormItem label="Name">
          <ElInput v-model="createForm.name" maxlength="120" placeholder="Optional display name" />
        </ElFormItem>

        <ElFormItem label="Permission Group">
          <ElSelect
            v-model="createForm.permission_group_id"
            clearable
            filterable
            placeholder="Select a permission group"
            class="full-width"
          >
            <ElOption
              v-for="item in permissionGroupOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </ElSelect>
        </ElFormItem>
      </ElForm>

      <template #footer>
        <div class="dialog-footer">
          <ElButton @click="createVisible = false">Cancel</ElButton>
          <ElButton type="primary" :loading="creating" @click="submitCreate">Create</ElButton>
        </div>
      </template>
    </ElDialog>

    <ElDialog
      v-model="createdKeyVisible"
      title="API Key"
      width="520px"
      destroy-on-close
      align-center
    >
      <div class="created-key-tip">
        This key is only shown once. Copy it now before closing this dialog.
      </div>

      <ElInput :model-value="createdApiKey" readonly type="textarea" :rows="4" />

      <template #footer>
        <div class="dialog-footer">
          <ElButton @click="createdKeyVisible = false">Close</ElButton>
          <ElButton type="primary" @click="copyCreatedKey">Copy Key</ElButton>
        </div>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import type { APIKeyRecord, PermissionGroupRecord } from '@/api/admin'
  import {
    createAdminApiKey,
    fetchAdminApiKeys,
    fetchPermissionGroups,
    hasAdminPermission,
    updateAdminApiKeyStatus
  } from '@/api/admin'
  import { useTableColumns } from '@/hooks/core/useTableColumns'
  import { useUserStore } from '@/store/modules/user'
  import { ElMessage } from 'element-plus'
  import ApiKeySearch from './modules/api-key-search.vue'

  defineOptions({ name: 'SettingsApiKeysPage' })

  interface ApiKeySearchForm {
    keyword: string
    status?: string
  }

  interface ApiKeyTableRow {
    id: number | null
    name: string
    key_hash: string
    status: string
    permission_group_id: number | null
    permission_group_name: string
    created_at: string
    updated_at: string
    last_used_at: string | null
  }

  interface CreateFormValue {
    name: string
    permission_group_id: number | null
  }

  const showSearchBar = ref(true)
  const loading = ref(false)
  const creating = ref(false)
  const initialized = ref(false)

  const createVisible = ref(false)
  const createdKeyVisible = ref(false)
  const createdApiKey = ref('')

  const searchForm = ref<ApiKeySearchForm>(createDefaultSearchForm())
  const createForm = reactive<CreateFormValue>(createDefaultCreateForm())
  const tableData = ref<ApiKeyTableRow[]>([])
  const permissionGroups = ref<PermissionGroupRecord[]>([])

  const pagination = reactive({
    current: 1,
    size: 20,
    total: 0
  })

  const { columnChecks, columns } = useTableColumns<ApiKeyTableRow>(() => [
    { prop: 'id', label: 'ID', width: 90 },
    { prop: 'name', label: 'Name', minWidth: 160, showOverflowTooltip: true },
    {
      prop: 'permission_group_name',
      label: 'Permission Group',
      minWidth: 180,
      showOverflowTooltip: true
    },
    { prop: 'key_hash', label: 'Key Hash', minWidth: 260, showOverflowTooltip: true },
    { prop: 'status', label: 'Status', width: 120, useSlot: true },
    { prop: 'operation', label: 'Operation', width: 150, fixed: 'right', useSlot: true }
  ])

  const userStore = useUserStore()
  const { info } = storeToRefs(userStore)

  const canView = computed(() => hasAdminPermission(info.value?.buttons, ['api_key.list']))
  const canCreate = computed(() => hasAdminPermission(info.value?.buttons, ['api_key.create']))
  const canUpdate = computed(() => hasAdminPermission(info.value?.buttons, ['api_key.update']))

  const permissionGroupMap = computed(() => {
    const map = new Map<number, string>()

    permissionGroups.value.forEach((group) => {
      const idValue = normalizeNullableNumber(group.id ?? group.ID)
      const nameValue = String(group.name ?? group.Name ?? '')

      if (idValue !== null) {
        map.set(idValue, nameValue)
      }
    })

    return map
  })

  const permissionGroupOptions = computed(
    () =>
      permissionGroups.value
        .map((group) => {
          const value = normalizeNullableNumber(group.id ?? group.ID)
          return {
            label: String(group.name ?? group.Name ?? ''),
            value
          }
        })
        .filter((item) => item.value !== null) as Array<{ label: string; value: number }>
  )

  watch(
    canView,
    (value) => {
      if (value && !initialized.value) {
        fetchData()
      }
    },
    { immediate: true }
  )

  function createDefaultSearchForm(): ApiKeySearchForm {
    return {
      keyword: '',
      status: undefined
    }
  }

  function createDefaultCreateForm(): CreateFormValue {
    return {
      name: '',
      permission_group_id: null
    }
  }

  function normalizeNullableNumber(value: unknown): number | null {
    if (value === '' || value === null || value === undefined) {
      return null
    }

    const parsed = Number(value)
    return Number.isFinite(parsed) ? parsed : null
  }

  function normalizeApiKey(item?: APIKeyRecord): ApiKeyTableRow {
    const permissionGroupId = normalizeNullableNumber(item?.permission_group_id)

    return {
      id: normalizeNullableNumber(item?.id),
      name: String(item?.name || ''),
      key_hash: String(item?.key_hash || ''),
      status: String(item?.status || ''),
      permission_group_id: permissionGroupId,
      permission_group_name: permissionGroupId
        ? String(permissionGroupMap.value.get(permissionGroupId) || '')
        : '',
      created_at: String(item?.created_at || ''),
      updated_at: String(item?.updated_at || ''),
      last_used_at: item?.last_used_at ? String(item.last_used_at) : null
    }
  }

  async function fetchData() {
    if (!canView.value) {
      return
    }

    loading.value = true

    try {
      const [groupsPayload, apiKeysPayload] = await Promise.all([
        fetchPermissionGroups().catch(() => ({ items: [] as PermissionGroupRecord[] })),
        fetchAdminApiKeys({
          limit: pagination.size,
          offset: (pagination.current - 1) * pagination.size
        })
      ])

      permissionGroups.value = (groupsPayload.items || []) as PermissionGroupRecord[]

      let rows = (apiKeysPayload.items || []).map((item) => normalizeApiKey(item))

      const keyword = String(searchForm.value.keyword || '')
        .trim()
        .toLowerCase()
      if (keyword) {
        rows = rows.filter((item) => item.key_hash.toLowerCase().includes(keyword))
      }

      if (searchForm.value.status) {
        rows = rows.filter((item) => item.status === searchForm.value.status)
      }

      tableData.value = rows
      pagination.total =
        keyword || searchForm.value.status
          ? rows.length
          : Number(apiKeysPayload.total || rows.length)
      initialized.value = true
    } finally {
      loading.value = false
    }
  }

  function handleSearch(params: ApiKeySearchForm) {
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

  function getStatusTagType(status?: string) {
    return status === 'active' ? ('success' as const) : ('info' as const)
  }

  function openCreate() {
    Object.assign(createForm, createDefaultCreateForm())
    createVisible.value = true
  }

  async function submitCreate() {
    if (!createForm.permission_group_id) {
      ElMessage.error('Please select a permission group')
      return
    }

    creating.value = true

    try {
      const payload = await createAdminApiKey({
        name: String(createForm.name || '').trim(),
        permission_group_id: createForm.permission_group_id
      })

      createdApiKey.value = String(payload.api_key || '')
      createVisible.value = false
      createdKeyVisible.value = true
      ElMessage.success('API key created successfully')
      await fetchData()
    } finally {
      creating.value = false
    }
  }

  async function toggleStatus(row: ApiKeyTableRow) {
    if (!row.id) {
      return
    }

    const nextStatus = row.status === 'active' ? 'disabled' : 'active'
    await updateAdminApiKeyStatus(row.id, { status: nextStatus })
    ElMessage.success('Status updated successfully')
    await fetchData()
  }

  async function copyText(value: string, successMessage: string) {
    const text = String(value || '')
    if (!text) {
      ElMessage.warning('Nothing to copy')
      return
    }

    try {
      await navigator.clipboard.writeText(text)
    } catch {
      const textarea = document.createElement('textarea')
      textarea.value = text
      textarea.style.position = 'fixed'
      textarea.style.left = '-9999px'
      document.body.appendChild(textarea)
      textarea.focus()
      textarea.select()
      document.execCommand('copy')
      document.body.removeChild(textarea)
    }

    ElMessage.success(successMessage)
  }

  async function copyKeyHash(row: ApiKeyTableRow) {
    await copyText(row.key_hash, 'Key hash copied')
  }

  async function copyCreatedKey() {
    await copyText(createdApiKey.value, 'API key copied')
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

  .page-subtitle,
  .created-key-tip {
    margin-top: 6px;
    color: var(--el-text-color-secondary);
    font-size: 13px;
    line-height: 1.7;
  }

  .page-actions,
  .dialog-footer,
  .table-actions {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .dialog-footer {
    justify-content: flex-end;
  }

  .table-actions {
    justify-content: flex-end;
    gap: 4px;
  }

  .full-width {
    width: 100%;
  }

  @media (max-width: 768px) {
    .page-header {
      flex-direction: column;
      align-items: stretch;
    }
  }
</style>
