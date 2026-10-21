<template>
  <div class="art-full-height">
    <UserSearch
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
            <ElButton type="primary" v-ripple @click="openCreate">创建用户</ElButton>
            <ElButton v-ripple @click="exportCsv">导出 CSV</ElButton>
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
        <template #avatar="{ row }">
          <ElAvatar :size="36" :src="row.avatar">
            {{ row.username?.slice(0, 1)?.toUpperCase() || 'U' }}
          </ElAvatar>
        </template>

        <template #operation="{ row }">
          <div class="table-actions">
            <ArtButtonTable type="view" @click="openDetail(row)" />
            <ArtButtonTable type="edit" @click="openEdit(row)" />
            <ArtButtonMore
              :list="getMoreActions(row)"
              @click="(item) => handleMoreAction(item, row)"
            />
          </div>
        </template>
      </ArtTable>
    </ElCard>

    <UserDialog
      v-model:visible="dialogVisible"
      :mode="dialogMode"
      :form-data="dialogForm"
      :tier-groups="tierGroups"
      :submitting="dialogSubmitting"
      :realname-record="realnameRecord"
      :realname-status="realnameStatus"
      :realname-reason="realnameReason"
      :realname-updating="realnameUpdating"
      @submit="handleDialogSubmit"
      @update:realname-status="realnameStatus = $event"
      @update:realname-reason="realnameReason = $event"
      @update-realname-status="handleUpdateRealnameStatus()"
    />

    <ElDialog
      v-model="resetDialogVisible"
      title="重置密码"
      width="420px"
      destroy-on-close
      align-center
    >
      <ElForm label-position="top">
        <ElFormItem label="新密码">
          <ElInput
            v-model="resetPassword"
            type="password"
            show-password
            placeholder="请输入新密码"
            :maxlength="INPUT_LIMITS.PASSWORD"
          />
        </ElFormItem>
      </ElForm>

      <template #footer>
        <div class="dialog-footer">
          <ElButton @click="resetDialogVisible = false">取消</ElButton>
          <ElButton type="primary" :loading="resetSubmitting" @click="submitResetPassword">
            确认重置
          </ElButton>
        </div>
      </template>
    </ElDialog>

    <UserDetailDrawer
      v-model:visible="detailVisible"
      :user="detailUser"
      :loading="detailLoading"
      :tier-groups="tierGroups"
      :wallet-info="walletInfo"
      :order-records="orderRecords"
      :wallet-transactions="walletTransactions"
      :realname-record="realnameRecord"
      :realname-status="realnameStatus"
      :realname-reason="realnameReason"
      :realname-updating="realnameUpdating"
      @impersonate="handleImpersonate(detailUser)"
      @update:realname-status="realnameStatus = $event"
      @update:realname-reason="realnameReason = $event"
      @update-realname-status="handleUpdateRealnameStatus(detailUser?.id)"
    />
  </div>
</template>

<script setup lang="ts">
  import type {
    OrderRecord,
    RealNameVerification,
    UserRecord,
    UserTierGroup,
    WalletInfo,
    WalletTransaction
  } from '@/api/admin'
  import type { ButtonMoreItem } from '@/components/core/forms/art-button-more/index.vue'
  import { ElTag } from 'element-plus'
  import {
    createAdminUser,
    fetchAdminOrders,
    fetchAdminUserDetail,
    fetchAdminUsers,
    fetchAdminWalletInfo,
    fetchAdminWalletTransactions,
    fetchRealNameRecords,
    fetchUserTierGroups,
    impersonateAdminUser,
    resetAdminUserPassword,
    updateAdminUser,
    updateAdminUserRealNameStatus,
    updateAdminUserStatus,
    updateAdminUserTier
  } from '@/api/admin'
  import ArtButtonMore from '@/components/core/forms/art-button-more/index.vue'
  import ArtButtonTable from '@/components/core/forms/art-button-table/index.vue'
  import { useTableColumns } from '@/hooks/core/useTableColumns'
  import { INPUT_LIMITS } from '@/utils/constants'
  import UserDetailDrawer from './modules/user-detail-drawer.vue'
  import UserDialog from './modules/user-dialog.vue'
  import UserSearch from './modules/user-search.vue'

  defineOptions({ name: 'User' })

  interface UserSearchForm {
    keyword: string
    status?: string
  }

  interface UserDialogFormValue {
    id: number | null
    username: string
    email: string
    password: string
    qq: string
    avatar: string
    status: string
    userTierGroupId: number | null
    userTierExpireAt: Date | null
  }

  interface UserTableRow {
    id: number | null
    username: string
    email: string
    qq: string
    avatar: string
    status: string
    role: string
    createdAt: string
    updatedAt: string
    userTierGroupId: number | null
    userTierExpireAt: string | null
    userTierGroupName: string
  }

  interface UserBundle {
    profile: UserRecord
    walletInfo: WalletInfo | null
    orderRecords: OrderRecord[]
    walletTransactions: WalletTransaction[]
    realnameRecord: RealNameVerification | null
  }

  interface UserRecordLike extends Partial<UserRecord> {
    ID?: unknown
    Username?: unknown
    Email?: unknown
    QQ?: unknown
    AvatarURL?: unknown
    Status?: unknown
    Role?: unknown
    CreatedAt?: unknown
    UpdatedAt?: unknown
    UserTierGroupID?: unknown
    UserTierExpireAt?: unknown
  }

  const showSearchBar = ref(true)
  const loading = ref(false)
  const detailLoading = ref(false)
  const dialogVisible = ref(false)
  const dialogMode = ref<'create' | 'edit'>('create')
  const dialogSubmitting = ref(false)
  const resetDialogVisible = ref(false)
  const resetSubmitting = ref(false)
  const detailVisible = ref(false)

  const searchForm = ref<UserSearchForm>(createDefaultSearchForm())
  const dialogForm = ref<UserDialogFormValue>(createDefaultDialogForm())
  const resetPassword = ref('')
  const resetTarget = ref<UserTableRow | null>(null)

  const tableData = ref<UserTableRow[]>([])
  const tierGroups = ref<UserTierGroup[]>([])
  const detailUser = ref<UserTableRow | null>(null)
  const walletInfo = ref<WalletInfo | null>(null)
  const orderRecords = ref<OrderRecord[]>([])
  const walletTransactions = ref<WalletTransaction[]>([])
  const realnameRecord = ref<RealNameVerification | null>(null)
  const realnameStatus = ref('')
  const realnameReason = ref('')
  const realnameUpdating = ref(false)

  const pagination = reactive({
    current: 1,
    size: 20,
    total: 0
  })

  const { columnChecks, columns } = useTableColumns<UserTableRow>(() => [
    { type: 'index', label: '序号', width: 60 },
    { prop: 'id', label: '用户 ID', width: 90 },
    {
      prop: 'avatar',
      label: '头像',
      width: 80,
      useSlot: true
    },
    {
      prop: 'username',
      label: '用户名',
      minWidth: 150,
      showOverflowTooltip: true
    },
    {
      prop: 'email',
      label: '邮箱',
      minWidth: 220,
      showOverflowTooltip: true
    },
    {
      prop: 'userTierGroupName',
      label: '用户组',
      minWidth: 140,
      formatter: (row: UserTableRow) => row.userTierGroupName || '-'
    },
    {
      prop: 'status',
      label: '状态',
      width: 110,
      formatter: (row: UserTableRow) =>
        h(ElTag, { type: getUserStatusTagType(row.status) }, () => row.status || '-')
    },
    {
      prop: 'createdAt',
      label: '注册时间',
      minWidth: 180,
      formatter: (row: UserTableRow) => formatDateTime(row.createdAt)
    },
    {
      prop: 'operation',
      label: '操作',
      width: 150,
      fixed: 'right',
      useSlot: true
    }
  ])

  onMounted(async () => {
    await loadTierGroups()
    await fetchData()
  })

  function createDefaultSearchForm(): UserSearchForm {
    return {
      keyword: '',
      status: undefined
    }
  }

  function createDefaultDialogForm(): UserDialogFormValue {
    return {
      id: null,
      username: '',
      email: '',
      password: '',
      qq: '',
      avatar: '',
      status: 'active',
      userTierGroupId: null,
      userTierExpireAt: null
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
    if (value === null || value === undefined || value === '') {
      return null
    }

    return String(value)
  }

  function normalizeUser(row?: UserRecordLike): UserTableRow {
    const source = row || {}
    const idValue = source.id ?? source.ID

    return {
      id: normalizeNullableNumber(idValue),
      username: String(source.username ?? source.Username ?? ''),
      email: String(source.email ?? source.Email ?? ''),
      qq: String(source.qq ?? source.QQ ?? ''),
      avatar: String(source.avatar ?? source.avatar_url ?? source.AvatarURL ?? ''),
      status: String(source.status ?? source.Status ?? 'active'),
      role: String(source.role ?? source.Role ?? ''),
      createdAt: String(source.created_at ?? source.CreatedAt ?? ''),
      updatedAt: String(source.updated_at ?? source.UpdatedAt ?? ''),
      userTierGroupId: normalizeNullableNumber(source.user_tier_group_id ?? source.UserTierGroupID),
      userTierExpireAt: normalizeNullableString(
        source.user_tier_expire_at ?? source.UserTierExpireAt
      ),
      userTierGroupName: ''
    }
  }

  function attachTierNames(users: UserTableRow[]): UserTableRow[] {
    return users.map((user) => ({
      ...user,
      userTierGroupName: getTierGroupName(user.userTierGroupId)
    }))
  }

  function getTierGroupName(groupId: number | null | undefined): string {
    if (groupId === null || groupId === undefined) {
      return '-'
    }

    const matched = tierGroups.value.find((group) => Number(group.id) === Number(groupId))
    return String(matched?.name || '-')
  }

  function parseExpireAt(value?: string | null): Date | null {
    if (!value) {
      return null
    }

    const date = new Date(value)
    return Number.isNaN(date.getTime()) ? null : date
  }

  function buildDialogForm(user?: UserTableRow | null): UserDialogFormValue {
    if (!user) {
      return createDefaultDialogForm()
    }

    return {
      id: user.id,
      username: user.username,
      email: user.email,
      password: '',
      qq: user.qq,
      avatar: user.avatar,
      status: user.status || 'active',
      userTierGroupId: user.userTierGroupId ?? null,
      userTierExpireAt: parseExpireAt(user.userTierExpireAt)
    }
  }

  function clearUserExtras() {
    walletInfo.value = null
    orderRecords.value = []
    walletTransactions.value = []
    realnameRecord.value = null
    realnameStatus.value = ''
    realnameReason.value = ''
  }

  function applyUserBundle(bundle: UserBundle) {
    walletInfo.value = bundle.walletInfo
    orderRecords.value = bundle.orderRecords
    walletTransactions.value = bundle.walletTransactions
    realnameRecord.value = bundle.realnameRecord
    realnameStatus.value = bundle.realnameRecord?.status || ''
    realnameReason.value = bundle.realnameRecord?.reason || ''
  }

  function formatDateTime(value?: string | null) {
    if (!value) {
      return '-'
    }

    const date = new Date(value)
    return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN')
  }

  function getUserStatusTagType(status?: string) {
    if (status === 'active') {
      return 'success' as const
    }

    if (status === 'blocked') {
      return 'danger' as const
    }

    return 'info' as const
  }

  function getMoreActions(row: UserTableRow): ButtonMoreItem[] {
    return [
      {
        key: 'impersonate',
        label: '以此用户登录',
        icon: 'ri:login-box-line',
        disabled: row.role === 'admin'
      },
      {
        key: 'toggle',
        label: row.status === 'active' ? '禁用用户' : '启用用户',
        icon: row.status === 'active' ? 'ri:forbid-2-line' : 'ri:check-line',
        disabled: row.role === 'admin'
      },
      {
        key: 'reset-password',
        label: '重置密码',
        icon: 'ri:lock-password-line',
        color: '#e6a23c',
        disabled: row.role === 'admin'
      }
    ]
  }

  async function loadTierGroups() {
    const payload = await fetchUserTierGroups()
    tierGroups.value = payload.items || []
  }

  async function fetchData() {
    loading.value = true

    try {
      const payload = await fetchAdminUsers({
        limit: pagination.size,
        offset: (pagination.current - 1) * pagination.size
      })

      let rows = attachTierNames(
        (payload.items || [])
          .map((item) => normalizeUser(item))
          .filter((item) => item.role !== 'admin')
      )

      if (searchForm.value.status) {
        rows = rows.filter((item) => item.status === searchForm.value.status)
      }

      tableData.value = rows
      pagination.total = typeof payload.total === 'number' ? payload.total : rows.length
    } finally {
      loading.value = false
    }
  }

  function handleSearch(params: UserSearchForm) {
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

  function openCreate() {
    dialogMode.value = 'create'
    dialogForm.value = createDefaultDialogForm()
    dialogVisible.value = true
  }

  async function loadUserBundle(userId: number): Promise<UserBundle | null> {
    detailLoading.value = true
    clearUserExtras()

    try {
      const [profile, walletPayload, ordersPayload, walletTxPayload, realnamePayload] =
        await Promise.all([
          fetchAdminUserDetail(userId),
          fetchAdminWalletInfo(userId),
          fetchAdminOrders({ user_id: userId, limit: 20, offset: 0 }),
          fetchAdminWalletTransactions(userId, { limit: 20, offset: 0 }),
          fetchRealNameRecords({ user_id: userId, limit: 1, offset: 0 })
        ])

      return {
        profile,
        walletInfo: walletPayload.wallet || null,
        orderRecords: ordersPayload.items || [],
        walletTransactions: walletTxPayload.items || [],
        realnameRecord: realnamePayload.items?.[0] || null
      }
    } catch {
      return null
    } finally {
      detailLoading.value = false
    }
  }

  async function openDetail(row: UserTableRow) {
    if (row.role === 'admin') {
      ElMessage.warning('管理员账号不支持在此处查看')
      return
    }

    if (!row.id) {
      return
    }

    detailVisible.value = true
    detailUser.value = attachTierNames([row])[0]

    const bundle = await loadUserBundle(row.id)
    if (!bundle || !detailVisible.value || detailUser.value?.id !== row.id) {
      return
    }

    const normalized = attachTierNames([normalizeUser(bundle.profile)])[0]
    detailUser.value = normalized
    applyUserBundle(bundle)
  }

  async function openEdit(row: UserTableRow) {
    if (row.role === 'admin') {
      ElMessage.warning('管理员账号不支持在此处修改')
      return
    }

    if (!row.id) {
      return
    }

    dialogMode.value = 'edit'
    dialogForm.value = buildDialogForm(row)
    dialogVisible.value = true

    const bundle = await loadUserBundle(row.id)
    if (
      !bundle ||
      !dialogVisible.value ||
      dialogMode.value !== 'edit' ||
      dialogForm.value.id !== row.id
    ) {
      return
    }

    dialogForm.value = buildDialogForm(attachTierNames([normalizeUser(bundle.profile)])[0])
    applyUserBundle(bundle)
  }

  function openResetDialog(row: UserTableRow) {
    resetTarget.value = row
    resetPassword.value = ''
    resetDialogVisible.value = true
  }

  async function submitResetPassword() {
    if (!resetTarget.value?.id) {
      return
    }

    if (String(resetPassword.value || '').length > INPUT_LIMITS.PASSWORD) {
      ElMessage.error(`密码长度不能超过 ${INPUT_LIMITS.PASSWORD} 个字符`)
      return
    }

    resetSubmitting.value = true

    try {
      await resetAdminUserPassword(resetTarget.value.id, { password: resetPassword.value })
      ElMessage.success('已重置密码')
      resetDialogVisible.value = false
    } finally {
      resetSubmitting.value = false
    }
  }

  function validateCreatePayload(form: UserDialogFormValue) {
    if (String(form.username || '').length > INPUT_LIMITS.USERNAME) {
      ElMessage.error(`用户名长度不能超过 ${INPUT_LIMITS.USERNAME} 个字符`)
      return false
    }

    if (String(form.email || '').length > INPUT_LIMITS.EMAIL) {
      ElMessage.error(`邮箱长度不能超过 ${INPUT_LIMITS.EMAIL} 个字符`)
      return false
    }

    if (String(form.password || '').length > INPUT_LIMITS.PASSWORD) {
      ElMessage.error(`密码长度不能超过 ${INPUT_LIMITS.PASSWORD} 个字符`)
      return false
    }

    return true
  }

  function validateEditPayload(form: UserDialogFormValue) {
    if (String(form.username || '').length > INPUT_LIMITS.USERNAME) {
      ElMessage.error(`用户名长度不能超过 ${INPUT_LIMITS.USERNAME} 个字符`)
      return false
    }

    if (String(form.email || '').length > INPUT_LIMITS.EMAIL) {
      ElMessage.error(`邮箱长度不能超过 ${INPUT_LIMITS.EMAIL} 个字符`)
      return false
    }

    if (String(form.qq || '').length > INPUT_LIMITS.QQ) {
      ElMessage.error(`QQ 长度不能超过 ${INPUT_LIMITS.QQ} 个字符`)
      return false
    }

    if (String(form.avatar || '').length > INPUT_LIMITS.URL) {
      ElMessage.error(`头像 URL 长度不能超过 ${INPUT_LIMITS.URL} 个字符`)
      return false
    }

    return true
  }

  async function handleDialogSubmit(form: UserDialogFormValue) {
    dialogSubmitting.value = true

    try {
      if (dialogMode.value === 'create') {
        if (!validateCreatePayload(form)) {
          return
        }

        await createAdminUser({
          username: form.username.trim(),
          email: form.email.trim(),
          password: form.password
        })

        ElMessage.success('用户已创建')
      } else {
        if (!form.id) {
          return
        }

        if (!validateEditPayload(form)) {
          return
        }

        await updateAdminUser(form.id, {
          username: form.username.trim(),
          email: form.email.trim(),
          qq: form.qq.trim(),
          avatar: form.avatar.trim(),
          status: form.status
        })

        await updateAdminUserTier(form.id, {
          group_id: Number(form.userTierGroupId || 0),
          expire_at: form.userTierExpireAt ? form.userTierExpireAt.toISOString() : ''
        })

        ElMessage.success('已更新用户资料')
      }

      dialogVisible.value = false
      await fetchData()
    } finally {
      dialogSubmitting.value = false
    }
  }

  async function handleToggleStatus(row: UserTableRow) {
    if (row.role === 'admin') {
      ElMessage.warning('管理员账号不支持在此处修改')
      return
    }

    if (!row.id) {
      return
    }

    const nextStatus = row.status === 'active' ? 'blocked' : 'active'
    await updateAdminUserStatus(row.id, { status: nextStatus })
    ElMessage.success('已更新状态')

    if (detailUser.value?.id === row.id) {
      detailUser.value = {
        ...detailUser.value,
        status: nextStatus
      }
    }

    if (dialogForm.value.id === row.id) {
      dialogForm.value = {
        ...dialogForm.value,
        status: nextStatus
      }
    }

    await fetchData()
  }

  async function handleImpersonate(user?: UserTableRow | null) {
    if (!user?.id) {
      return
    }

    if (user.role === 'admin') {
      ElMessage.warning('管理员账号不支持模拟登录')
      return
    }

    const payload = await impersonateAdminUser(user.id)
    const token = payload.access_token

    if (!token) {
      ElMessage.error('未获取到用户令牌')
      return
    }

    localStorage.setItem('user_token', token)

    const consoleUrl = `/console#impersonate_token=${encodeURIComponent(token)}`
    const popup = window.open(consoleUrl, '_blank', 'noopener')

    if (!popup) {
      window.location.href = consoleUrl
      return
    }

    ElMessage.success('已切换到该用户，可在新标签页继续操作')
  }

  async function handleUpdateRealnameStatus(targetId?: number | null) {
    const userId = targetId || detailUser.value?.id || dialogForm.value.id
    if (!userId) {
      return
    }

    if (String(realnameReason.value || '').length > INPUT_LIMITS.REVIEW_REASON) {
      ElMessage.error(`审核备注长度不能超过 ${INPUT_LIMITS.REVIEW_REASON} 个字符`)
      return
    }

    realnameUpdating.value = true

    try {
      await updateAdminUserRealNameStatus(userId, {
        status: realnameStatus.value,
        reason: realnameReason.value
      })

      ElMessage.success('实名状态已更新')

      const bundle = await loadUserBundle(userId)
      if (!bundle) {
        return
      }

      applyUserBundle(bundle)

      const normalized = attachTierNames([normalizeUser(bundle.profile)])[0]
      if (detailUser.value?.id === userId) {
        detailUser.value = normalized
      }
      if (dialogForm.value.id === userId) {
        dialogForm.value = buildDialogForm(normalized)
      }
    } finally {
      realnameUpdating.value = false
    }
  }

  function handleMoreAction(item: ButtonMoreItem, row: UserTableRow) {
    switch (item.key) {
      case 'impersonate':
        handleImpersonate(row)
        break
      case 'toggle':
        handleToggleStatus(row)
        break
      case 'reset-password':
        openResetDialog(row)
        break
    }
  }

  function escapeCsvCell(value: string | number | null | undefined) {
    const text = String(value ?? '')
    return `"${text.replace(/"/g, '""')}"`
  }

  function exportCsv() {
    const rows = tableData.value.map((item) =>
      [escapeCsvCell(item.id), escapeCsvCell(item.email), escapeCsvCell(item.status)].join(',')
    )
    const content = ['id,email,status', ...rows].join('\n')
    const blob = new Blob([content], { type: 'text/csv;charset=utf-8;' })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')

    link.href = url
    link.download = 'users.csv'
    link.click()

    URL.revokeObjectURL(url)
  }
</script>

<style scoped lang="scss">
  .table-actions {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: 2px;
  }

  .dialog-footer {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
  }
</style>
