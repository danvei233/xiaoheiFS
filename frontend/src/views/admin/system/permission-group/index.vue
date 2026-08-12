<template>
  <div class="art-full-height">
    <ElCard class="art-table-card">
      <ElEmpty v-if="!canView" description="你没有查看权限组的权限。" />

      <template v-else>
        <ArtTableHeader v-model:columns="columnChecks" :loading="loading" @refresh="fetchData">
          <template #left>
            <ElButton v-if="canCreate" type="primary" @click="openCreate">创建权限组</ElButton>
          </template>
        </ArtTableHeader>

        <ArtTable row-key="id" :loading="loading" :data="tableData" :columns="columns">
          <template #permissions="{ row }">
            <div v-if="row.permissions.length" class="permission-tags">
              <ElTooltip
                :content="getPermissionTooltip(row.permissions)"
                placement="top"
                :show-after="150"
              >
                <div class="permission-tags-inner">
                  <ElTag
                    v-for="permission in row.permissions.slice(0, 5)"
                    :key="permission"
                    size="small"
                    type="info"
                  >
                    {{ getPermissionLabel(permission) }}
                  </ElTag>
                  <ElTag v-if="row.permissions.length > 5" size="small" type="primary">
                    +{{ row.permissions.length - 5 }} 更多
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
  import type { PermissionGroupRecord, PermissionRecord } from '@/services/admin'
  import {
    createPermissionGroup,
    deletePermissionGroup,
    fetchAdminPermissions,
    fetchPermissionGroups,
    updatePermissionGroup
  } from '@/services/admin'
  import { useTable } from '@/hooks/core/useTable'
  import { useAuth } from '@/hooks/core/useAuth'
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

  interface PermissionGroupRecordLike extends PermissionGroupRecord {
    Permissions?: unknown
  }

  const { hasAuth } = useAuth()
  const dialogVisible = ref(false)
  const dialogSubmitting = ref(false)
  const dialogMode = ref<'create' | 'edit'>('create')
  const initialized = ref(false)

  const permissionOptions = ref<PermissionRecord[]>([])
  const dialogForm = ref<PermissionGroupDialogFormValue>(createDefaultDialogForm())

  const {
    columnChecks,
    columns,
    data: tableData,
    loading,
    getData,
    fetchData
  } = useTable({
    core: {
      apiFn: fetchPermissionGroupTable,
      apiParams: { current: 1, size: 20 },
      immediate: false,
      columnsFactory: () => [
        { prop: 'id', label: 'ID', width: 90 },
        { prop: 'name', label: '名称', minWidth: 180, showOverflowTooltip: true },
        { prop: 'description', label: '描述', minWidth: 220, showOverflowTooltip: true },
        { prop: 'permissions', label: '权限', minWidth: 300, useSlot: true },
        { prop: 'operation', label: '操作', width: 150, fixed: 'right', useSlot: true }
      ]
    }
  })

  const canView = computed(() => hasAuth('permission_group.list'))
  const canCreate = computed(() => hasAuth('permission_group.create'))
  const canUpdate = computed(() => hasAuth('permission_group.update'))
  const canDelete = computed(() => hasAuth('permission_group.delete'))

  const permissionLabelMap = computed(() => {
    const map = new Map<string, string>()

    permissionOptions.value.forEach((permission) => {
      const code = String(permission.code ?? permission.Code ?? '')
      if (!code) {
        return
      }

      map.set(
        code,
        String(
          permission.friendly_name ??
            permission.FriendlyName ??
            permission.name ??
            permission.Name ??
            code
        )
      )
    })

    return map
  })

  watch(
    canView,
    (value) => {
      if (value && !initialized.value) {
        initialized.value = true
        getData()
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

  function normalizePermissionGroup(item?: PermissionGroupRecordLike): PermissionGroupTableRow {
    return {
      id: normalizeNullableNumber(item?.id ?? item?.ID),
      name: String(item?.name ?? item?.Name ?? ''),
      description: String(item?.description ?? item?.Description ?? ''),
      permissions: parsePermissions(
        item?.permissions ?? item?.Permissions ?? item?.permissions_json ?? item?.PermissionsJSON
      ),
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

  async function fetchPermissionGroupTable(
    _params?: unknown,
    signal?: AbortSignal
  ): Promise<Api.Common.PaginatedResponse<PermissionGroupTableRow>> {
    const [groupsPayload, permissionsPayload] = await Promise.all([
      fetchPermissionGroups(signal),
      fetchAdminPermissions(signal)
    ])
    permissionOptions.value = permissionsPayload.items || []
    const allRows = (groupsPayload.items || []).map(normalizePermissionGroup)
    return {
      records: allRows,
      current: 1,
      size: allRows.length,
      total: allRows.length
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
    gap: 16px;
    align-items: flex-start;
    justify-content: space-between;
  }

  .page-title {
    font-size: 18px;
    font-weight: 700;
    color: var(--el-text-color-primary);
  }

  .page-subtitle {
    margin-top: 6px;
    font-size: 13px;
    line-height: 1.7;
    color: var(--el-text-color-secondary);
  }

  .page-actions,
  .table-actions {
    display: flex;
    gap: 12px;
    align-items: center;
  }

  .table-actions {
    gap: 4px;
    justify-content: flex-end;
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

  @media (width <= 768px) {
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
