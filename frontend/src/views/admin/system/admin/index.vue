<template>
  <div class="art-full-height">
    <ElCard v-loading="loading" class="art-table-card">
      <template #header>
        <div class="page-header">
          <div>
            <div class="page-title">管理员管理</div>
            <div class="page-subtitle">管理系统管理员账号与权限组分配。</div>
          </div>

          <div class="page-actions">
            <ElButton v-if="canCreate" type="primary" @click="openCreate">创建管理员</ElButton>
          </div>
        </div>
      </template>

      <ElEmpty v-if="!canView" description="你没有查看管理员列表的权限。" />

      <template v-else>
        <ArtTableHeader v-model:columns="columnChecks" :loading="loading" @refresh="fetchData" />

        <ArtTable
          row-key="id"
          :loading="loading"
          :data="tableData"
          :columns="columns"
          :pagination="pagination"
          @pagination:size-change="handlePageSizeChange"
          @pagination:current-change="handlePageCurrentChange"
        >
          <template #avatar="{ row }">
            <ElAvatar :size="40" :src="resolveAvatar(row)">
              {{ row.username?.slice(0, 1)?.toUpperCase() || 'A' }}
            </ElAvatar>
          </template>

          <template #permission_group_name="{ row }">
            <ElTag type="primary">{{ row.permission_group_name || '-' }}</ElTag>
          </template>

          <template #status="{ row }">
            <ElTag :type="row.status === 'active' ? 'success' : 'info'">
              {{ row.status === 'active' ? '启用' : '禁用' }}
            </ElTag>
          </template>

          <template #operation="{ row }">
            <div class="table-actions">
              <ElButton v-if="canUpdate" link type="primary" @click="openEdit(row)">编辑</ElButton>
              <ElButton
                v-if="canUpdate && row.id !== currentAdminId"
                link
                :type="row.status === 'active' ? 'danger' : 'primary'"
                @click="toggleStatus(row)"
              >
                {{ row.status === 'active' ? '禁用' : '启用' }}
              </ElButton>
            </div>
          </template>
        </ArtTable>
      </template>
    </ElCard>

    <AdminDialog
      v-model:visible="dialogVisible"
      :mode="dialogMode"
      :form-data="dialogForm"
      :permission-groups="permissionGroups"
      :submitting="dialogSubmitting"
      @submit="handleDialogSubmit"
    />
  </div>
</template>

<script setup lang="ts">
  import type { PermissionGroupRecord, UserRecord } from '@/services/admin'
  import {
    createAdminAccount,
    fetchAdminAccounts,
    fetchPermissionGroups,
    updateAdminAccount,
    updateAdminAccountStatus
  } from '@/services/admin'
  import { useTable } from '@/hooks/core/useTable'
  import { useAuth } from '@/hooks/core/useAuth'
  import { useAdminAuthStore } from '@/stores/adminAuth'
  import { INPUT_LIMITS } from '@/constants/inputLimits'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import AdminDialog from './modules/admin-dialog.vue'

  defineOptions({ name: 'SystemAdminPage' })

  type AdminTableParams = Api.Common.CommonSearchParams

  interface AdminDialogFormValue {
    id: number | null
    username: string
    email: string
    qq: string
    password: string
    permission_group_id: number | null
  }

  interface AdminTableRow {
    id: number | null
    username: string
    email: string
    qq: string
    avatar: string
    status: string
    created_at: string
    updated_at: string
    permission_group_id: number | null
    permission_group_name: string
  }

  interface UserRecordLike extends Partial<UserRecord> {
    ID?: unknown
    Username?: unknown
    Email?: unknown
    QQ?: unknown
    AvatarURL?: unknown
    Status?: unknown
    CreatedAt?: unknown
    UpdatedAt?: unknown
    PermissionGroupID?: unknown
  }

  const adminAuthStore = useAdminAuthStore()
  const { profile: info } = storeToRefs(adminAuthStore)
  const { hasAuth } = useAuth()

  const dialogVisible = ref(false)
  const dialogSubmitting = ref(false)
  const dialogMode = ref<'create' | 'edit'>('create')
  const initialized = ref(false)

  const dialogForm = ref<AdminDialogFormValue>(createDefaultDialogForm())
  const permissionGroups = ref<PermissionGroupRecord[]>([])

  const {
    columnChecks,
    columns,
    data: tableData,
    loading,
    pagination,
    getData,
    fetchData,
    handleSizeChange: handlePageSizeChange,
    handleCurrentChange: handlePageCurrentChange
  } = useTable({
    core: {
      apiFn: fetchAdminTable,
      apiParams: { current: 1, size: 20 },
      immediate: false,
      columnsFactory: () => [
        { prop: 'avatar', label: '头像', width: 90, useSlot: true },
        { prop: 'username', label: '用户名', minWidth: 140, showOverflowTooltip: true },
        { prop: 'email', label: '邮箱', minWidth: 220, showOverflowTooltip: true },
        { prop: 'qq', label: 'QQ', width: 140, showOverflowTooltip: true },
        { prop: 'permission_group_name', label: '权限组', minWidth: 160, useSlot: true },
        { prop: 'status', label: '状态', width: 110, useSlot: true },
        {
          prop: 'created_at',
          label: '创建时间',
          minWidth: 180,
          formatter: (row: AdminTableRow) => formatDateTime(row.created_at)
        },
        { prop: 'operation', label: '操作', width: 150, fixed: 'right', useSlot: true }
      ]
    }
  })

  const currentAdminId = computed(() => Number(info.value?.id || 0) || null)
  const canView = computed(() => hasAuth('admin.list'))
  const canCreate = computed(() => hasAuth('admin.create'))
  const canUpdate = computed(() => hasAuth('admin.update'))

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

  function createDefaultDialogForm(): AdminDialogFormValue {
    return {
      id: null,
      username: '',
      email: '',
      qq: '',
      password: '',
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

  function normalizeAdmin(item?: UserRecordLike): AdminTableRow {
    const permissionGroupId = normalizeNullableNumber(
      item?.permission_group_id ?? item?.PermissionGroupID
    )

    return {
      id: normalizeNullableNumber(item?.id ?? item?.ID),
      username: String(item?.username ?? item?.Username ?? ''),
      email: String(item?.email ?? item?.Email ?? ''),
      qq: String(item?.qq ?? item?.QQ ?? ''),
      avatar: String(item?.avatar ?? item?.avatar_url ?? item?.AvatarURL ?? ''),
      status: String(item?.status ?? item?.Status ?? 'active'),
      created_at: String(item?.created_at ?? item?.CreatedAt ?? ''),
      updated_at: String(item?.updated_at ?? item?.UpdatedAt ?? ''),
      permission_group_id: permissionGroupId,
      permission_group_name: permissionGroupId
        ? String(permissionGroupMap.value.get(permissionGroupId) || '-')
        : '-'
    }
  }

  function buildDialogForm(row?: AdminTableRow | null): AdminDialogFormValue {
    if (!row) {
      return createDefaultDialogForm()
    }

    return {
      id: row.id,
      username: row.username,
      email: row.email,
      qq: row.qq,
      password: '',
      permission_group_id: row.permission_group_id
    }
  }

  function resolveAvatar(row: AdminTableRow) {
    if (row.avatar) {
      return row.avatar
    }

    const qq = String(row.qq || '').trim()
    if (!qq) {
      return ''
    }

    return `https://q1.qlogo.cn/g?b=qq&nk=${qq}&s=100`
  }

  function formatDateTime(value?: string | null) {
    if (!value) {
      return '-'
    }

    const date = new Date(value)
    return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN')
  }

  async function fetchAdminTable(
    params: AdminTableParams,
    signal?: AbortSignal
  ): Promise<Api.Common.PaginatedResponse<AdminTableRow>> {
    const [groupsPayload, adminsPayload] = await Promise.all([
      fetchPermissionGroups(signal),
      fetchAdminAccounts(
        {
          limit: params.size,
          offset: (params.current - 1) * params.size,
          status: 'all'
        },
        signal
      )
    ])
    permissionGroups.value = groupsPayload.items || []
    const records = (adminsPayload.items || []).map(normalizeAdmin)
    return {
      records,
      current: params.current,
      size: params.size,
      total: Number(adminsPayload.total ?? records.length)
    }
  }

  function openCreate() {
    dialogMode.value = 'create'
    dialogForm.value = createDefaultDialogForm()
    dialogVisible.value = true
  }

  function openEdit(row: AdminTableRow) {
    dialogMode.value = 'edit'
    dialogForm.value = buildDialogForm(row)
    dialogVisible.value = true
  }

  async function handleDialogSubmit(form: AdminDialogFormValue) {
    if (String(form.username || '').length > INPUT_LIMITS.USERNAME) {
      ElMessage.error(`用户名长度不能超过 ${INPUT_LIMITS.USERNAME} 个字符`)
      return
    }

    if (String(form.email || '').length > INPUT_LIMITS.EMAIL) {
      ElMessage.error(`邮箱长度不能超过 ${INPUT_LIMITS.EMAIL} 个字符`)
      return
    }

    if (String(form.qq || '').length > INPUT_LIMITS.QQ) {
      ElMessage.error(`QQ 长度不能超过 ${INPUT_LIMITS.QQ} 个字符`)
      return
    }

    if (
      dialogMode.value === 'create' &&
      String(form.password || '').length > INPUT_LIMITS.PASSWORD
    ) {
      ElMessage.error(`密码长度不能超过 ${INPUT_LIMITS.PASSWORD} 个字符`)
      return
    }

    dialogSubmitting.value = true

    try {
      if (dialogMode.value === 'create') {
        await createAdminAccount({
          username: form.username,
          email: form.email,
          qq: form.qq,
          password: form.password,
          permission_group_id: form.permission_group_id
        })
        ElMessage.success('管理员已创建')
      } else {
        if (!form.id) {
          return
        }

        await updateAdminAccount(form.id, {
          username: form.username,
          email: form.email,
          qq: form.qq,
          permission_group_id: form.permission_group_id
        })
        ElMessage.success('管理员已更新')
      }

      dialogVisible.value = false
      await fetchData()
    } finally {
      dialogSubmitting.value = false
    }
  }

  async function toggleStatus(row: AdminTableRow) {
    if (!row.id) {
      return
    }

    const nextStatus = row.status === 'active' ? 'disabled' : 'active'
    const actionText = nextStatus === 'disabled' ? '禁用' : '启用'

    await ElMessageBox.confirm(
      `确定要${actionText}管理员 "${row.username}" 吗？`,
      `${actionText}确认`,
      {
        type: 'warning',
        confirmButtonText: '确定',
        cancelButtonText: '取消'
      }
    )

    await updateAdminAccountStatus(row.id, { status: nextStatus })
    ElMessage.success('管理员状态已更新')
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

  @media (width <= 768px) {
    .page-header {
      flex-direction: column;
      align-items: stretch;
    }
  }
</style>
