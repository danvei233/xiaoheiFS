<template>
  <div class="art-full-height">
    <ElCard v-loading="loading" class="art-table-card">
      <template #header>
        <div class="page-header">
          <div>
            <div class="page-title">权限组管理</div>
            <div class="page-subtitle">管理后台权限组以及每组对应的权限集合。</div>
          </div>

          <div class="page-actions">
            <ElButton v-if="canCreate" type="primary" @click="openCreate">创建权限组</ElButton>
          </div>
        </div>
      </template>

      <ElEmpty v-if="!canView" description="你没有查看权限组的权限。" />

      <template v-else>
        <div class="toolbar">
          <ElInput
            v-model="keyword"
            clearable
            placeholder="按名称、描述或权限搜索"
            class="search-input"
          />
        </div>

        <ArtTableHeader
          v-model:columns="columnChecks"
          :showSearchBar="false"
          :loading="loading"
          @refresh="fetchData"
        />

        <ArtTable row-key="id" :loading="loading" :data="filteredRows" :columns="columns">
          <template #permissions="{ row }">
            <div v-if="row.permissions.length" class="permission-tags">
              <ElTooltip
                :content="getPermissionTooltip(row.permissions)"
                placement="top"
                :show-after="150"
              >
                <div class="permission-tags-inner">
                  <ElTag
                    v-for="permission in row.permissions.slice(0, 4)"
                    :key="permission"
                    size="small"
                    type="info"
                  >
                    {{ getPermissionLabel(permission) }}
                  </ElTag>
                  <ElTag v-if="row.permissions.length > 4" size="small" type="primary">
                    +{{ row.permissions.length - 4 }}
                  </ElTag>
                </div>
              </ElTooltip>
            </div>
            <span v-else class="empty-text">-</span>
          </template>

          <template #operation="{ row }">
            <div class="table-actions">
              <ElButton v-if="canUpdate" link type="primary" @click="openEdit(row)">编辑</ElButton>
              <ElButton v-if="canDelete" link type="danger" @click="handleDelete(row)"
                >删除</ElButton
              >
            </div>
          </template>
        </ArtTable>
      </template>
    </ElCard>

    <PermissionGroupDialog
      v-model:visible="dialogVisible"
      :mode="dialogMode"
      :form-data="dialogForm"
      :permissions="permissionOptions"
      :submitting="dialogSubmitting"
      @submit="handleDialogSubmit"
    />
  </div>
</template>

<script setup lang="ts">
  import type { PermissionGroupRecord, PermissionRecord } from '@/api/admin'
  import {
    createPermissionGroup,
    deletePermissionGroup,
    fetchAdminPermissions,
    fetchPermissionGroups,
    hasAdminPermission,
    updatePermissionGroup
  } from '@/api/admin'
  import { useTableColumns } from '@/hooks/core/useTableColumns'
  import { useUserStore } from '@/store/modules/user'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import PermissionGroupDialog from './modules/permission-group-dialog.vue'

  defineOptions({ name: 'SystemPermissionGroupPage' })

  interface PermissionGroupDialogFormValue {
    id: number | null
    name: string
    description: string
    permissions: string[]
  }

  interface PermissionGroupTableRow {
    id: number | null
    name: string
    description: string
    permissions: string[]
    created_at: string
    updated_at: string
  }

  const userStore = useUserStore()
  const { info } = storeToRefs(userStore)

  const loading = ref(false)
  const keyword = ref('')
  const dialogVisible = ref(false)
  const dialogSubmitting = ref(false)
  const dialogMode = ref<'create' | 'edit'>('create')
  const initialized = ref(false)

  const tableData = ref<PermissionGroupTableRow[]>([])
  const permissionOptions = ref<PermissionRecord[]>([])
  const dialogForm = ref<PermissionGroupDialogFormValue>(createDefaultDialogForm())

  const { columnChecks, columns } = useTableColumns<PermissionGroupTableRow>(() => [
    { prop: 'id', label: 'ID', width: 90 },
    { prop: 'name', label: '名称', minWidth: 180, showOverflowTooltip: true },
    { prop: 'description', label: '描述', minWidth: 220, showOverflowTooltip: true },
    { prop: 'permissions', label: '权限', minWidth: 300, useSlot: true },
    {
      prop: 'updated_at',
      label: '更新时间',
      minWidth: 180,
      formatter: (row: PermissionGroupTableRow) => formatDateTime(row.updated_at)
    },
    { prop: 'operation', label: '操作', width: 150, fixed: 'right', useSlot: true }
  ])

  const canView = computed(() =>
    hasAdminPermission(info.value?.buttons, ['permission_group.list', 'permission_group.view'])
  )
  const canCreate = computed(() =>
    hasAdminPermission(info.value?.buttons, ['permission_group.create'])
  )
  const canUpdate = computed(() =>
    hasAdminPermission(info.value?.buttons, ['permission_group.update'])
  )
  const canDelete = computed(() =>
    hasAdminPermission(info.value?.buttons, ['permission_group.delete'])
  )

  const permissionLabelMap = computed(() => {
    const map = new Map<string, string>()

    permissionOptions.value.forEach((permission) => {
      const code = String(permission.code || '')
      if (!code) {
        return
      }

      map.set(code, String(permission.friendly_name || permission.name || permission.code || ''))
    })

    return map
  })

  const filteredRows = computed(() => {
    const search = keyword.value.trim().toLowerCase()
    if (!search) {
      return tableData.value
    }

    return tableData.value.filter((row) => {
      const joinedPermissions = row.permissions
        .map((permission) => `${permission} ${getPermissionLabel(permission)}`)
        .join(' ')

      return [row.name, row.description, joinedPermissions].some((value) =>
        String(value || '')
          .toLowerCase()
          .includes(search)
      )
    })
  })

  watch(
    canView,
    (value) => {
      if (value && !initialized.value) {
        fetchData()
      }
    },
    { immediate: true }
  )

  function createDefaultDialogForm(): PermissionGroupDialogFormValue {
    return {
      id: null,
      name: '',
      description: '',
      permissions: []
    }
  }

  function normalizeNullableNumber(value: unknown): number | null {
    if (value === '' || value === null || value === undefined) {
      return null
    }

    const parsed = Number(value)
    return Number.isFinite(parsed) ? parsed : null
  }

  function parsePermissions(value: unknown): string[] {
    if (Array.isArray(value)) {
      return value.filter(Boolean).map((item) => String(item))
    }

    if (typeof value === 'string') {
      try {
        const parsed = JSON.parse(value)
        return Array.isArray(parsed) ? parsed.filter(Boolean).map((item) => String(item)) : []
      } catch {
        return []
      }
    }

    return []
  }

  function normalizePermissionGroup(item?: PermissionGroupRecord): PermissionGroupTableRow {
    return {
      id: normalizeNullableNumber(item?.id ?? item?.ID),
      name: String(item?.name ?? item?.Name ?? ''),
      description: String(item?.description ?? item?.Description ?? ''),
      permissions: parsePermissions(item?.permissions_json ?? item?.PermissionsJSON),
      created_at: String(item?.created_at ?? item?.CreatedAt ?? ''),
      updated_at: String(item?.updated_at ?? item?.UpdatedAt ?? '')
    }
  }

  function buildDialogForm(row?: PermissionGroupTableRow | null): PermissionGroupDialogFormValue {
    if (!row) {
      return createDefaultDialogForm()
    }

    return {
      id: row.id,
      name: row.name,
      description: row.description,
      permissions: [...row.permissions]
    }
  }

  function getPermissionLabel(code: string) {
    return permissionLabelMap.value.get(code) || code || '-'
  }

  function getPermissionTooltip(permissions: string[]) {
    return permissions.map((permission) => getPermissionLabel(permission)).join('、')
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
      const [groupsPayload, permissionsPayload] = await Promise.all([
        fetchPermissionGroups(),
        fetchAdminPermissions()
      ])

      tableData.value = (groupsPayload.items || []).map((item) => normalizePermissionGroup(item))
      permissionOptions.value = permissionsPayload.items || []
      initialized.value = true
    } finally {
      loading.value = false
    }
  }

  function openCreate() {
    dialogMode.value = 'create'
    dialogForm.value = createDefaultDialogForm()
    dialogVisible.value = true
  }

  function openEdit(row: PermissionGroupTableRow) {
    dialogMode.value = 'edit'
    dialogForm.value = buildDialogForm(row)
    dialogVisible.value = true
  }

  async function handleDialogSubmit(form: PermissionGroupDialogFormValue) {
    dialogSubmitting.value = true

    try {
      if (dialogMode.value === 'create') {
        await createPermissionGroup({
          name: form.name,
          description: form.description,
          permissions: form.permissions
        })
        ElMessage.success('权限组已创建')
      } else {
        if (!form.id) {
          return
        }

        await updatePermissionGroup(form.id, {
          name: form.name,
          description: form.description,
          permissions: form.permissions
        })
        ElMessage.success('权限组已更新')
      }

      dialogVisible.value = false
      await fetchData()
    } finally {
      dialogSubmitting.value = false
    }
  }

  async function handleDelete(row: PermissionGroupTableRow) {
    if (!row.id) {
      return
    }

    await ElMessageBox.confirm(`确定要删除权限组 "${row.name}" 吗？`, '删除确认', {
      type: 'warning',
      confirmButtonText: '确定',
      cancelButtonText: '取消'
    })

    await deletePermissionGroup(row.id)
    ElMessage.success('权限组已删除')
    await fetchData()
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

  .page-actions,
  .table-actions {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .table-actions {
    justify-content: flex-end;
    gap: 4px;
  }

  .toolbar {
    display: flex;
    justify-content: flex-end;
    margin-bottom: 12px;
  }

  .search-input {
    width: min(360px, 100%);
  }

  .permission-tags {
    width: 100%;
  }

  .permission-tags-inner {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }

  .empty-text {
    color: var(--el-text-color-secondary);
  }

  @media (max-width: 768px) {
    .page-header {
      flex-direction: column;
      align-items: stretch;
    }

    .toolbar {
      justify-content: stretch;
    }

    .search-input {
      width: 100%;
    }
  }
</style>
