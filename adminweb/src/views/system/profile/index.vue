<template>
  <div v-loading="pageLoading" class="profile-page art-full-height">
    <ElRow :gutter="16">
      <ElCol :xs="24" :lg="8">
        <ElCard shadow="never" class="hero-card">
          <div class="hero-avatar-wrap">
            <ElAvatar :size="88" :src="avatarUrl">
              {{ profileInitial }}
            </ElAvatar>
          </div>

          <h2 class="hero-name">{{ profileName }}</h2>
          <p class="hero-role">{{ roleLabel }}</p>
          <ElTag :type="permissionTagType" size="large">{{ permissionGroupLabel }}</ElTag>

          <div class="hero-stats">
            <div class="stat-item">
              <span class="stat-label">Joined</span>
              <strong>{{ formatDate(profile.created_at) }}</strong>
            </div>
            <div class="stat-item">
              <span class="stat-label">Permissions</span>
              <strong>{{ permissionCount }}</strong>
            </div>
            <div class="stat-item">
              <span class="stat-label">Status</span>
              <strong>{{ statusLabel }}</strong>
            </div>
          </div>
        </ElCard>
      </ElCol>

      <ElCol :xs="24" :lg="16">
        <ElCard shadow="never" class="section-card">
          <template #header>
            <div class="section-header">
              <div>
                <div class="section-title">Basic Information</div>
                <div class="section-subtitle">
                  View current administrator profile information and permission group details.
                </div>
              </div>

              <ElButton type="primary" plain @click="openEditProfile">Edit Profile</ElButton>
            </div>
          </template>

          <ElDescriptions :column="2" border>
            <ElDescriptionsItem label="Username">{{ profile.username || '-' }}</ElDescriptionsItem>
            <ElDescriptionsItem label="Role">{{ roleLabel }}</ElDescriptionsItem>
            <ElDescriptionsItem label="Email">{{ profile.email || '-' }}</ElDescriptionsItem>
            <ElDescriptionsItem label="QQ">{{ profile.qq || '-' }}</ElDescriptionsItem>
            <ElDescriptionsItem label="Permission Group">
              <ElTag :type="permissionTagType">{{ permissionGroupLabel }}</ElTag>
            </ElDescriptionsItem>
            <ElDescriptionsItem label="Created At">
              {{ formatDateTime(profile.created_at) }}
            </ElDescriptionsItem>
          </ElDescriptions>
        </ElCard>

        <ElCard shadow="never" class="section-card">
          <template #header>
            <div class="section-header">
              <div>
                <div class="section-title">Security</div>
                <div class="section-subtitle">
                  Update your login password regularly to keep the administrator account secure.
                </div>
              </div>

              <ElButton type="primary" plain @click="openChangePassword">Change Password</ElButton>
            </div>
          </template>

          <div class="security-panel">
            <div class="security-icon">
              <ElIcon><Lock /></ElIcon>
            </div>
            <div class="security-copy">
              <div class="security-title">Login Password</div>
              <p>
                The current password is not shown for security reasons. After changing it, use the
                new password on your next login.
              </p>
            </div>
          </div>
        </ElCard>

        <ElCard shadow="never" class="section-card">
          <template #header>
            <div class="section-header">
              <div>
                <div class="section-title">Permissions</div>
                <div class="section-subtitle">
                  Current effective permissions for this administrator account.
                </div>
              </div>

              <ElTag type="info">{{ permissionCount }} items</ElTag>
            </div>
          </template>

          <div v-if="grantedPermissions.length" class="permission-list">
            <ElTag
              v-for="permission in grantedPermissions"
              :key="permission"
              type="primary"
              effect="plain"
            >
              {{ permissionLabel(permission) }}
            </ElTag>
          </div>
          <ElEmpty v-else description="No permission data available" />
        </ElCard>
      </ElCol>
    </ElRow>

    <ElDialog
      v-model="profileDialogVisible"
      title="Edit Profile"
      width="520px"
      destroy-on-close
      align-center
    >
      <ElForm ref="profileFormRef" :model="profileForm" :rules="profileRules" label-position="top">
        <ElFormItem label="Email" prop="email">
          <ElInput
            v-model.trim="profileForm.email"
            :maxlength="INPUT_LIMITS.EMAIL"
            placeholder="Enter email address"
          />
        </ElFormItem>

        <ElFormItem label="QQ" prop="qq">
          <ElInput
            v-model.trim="profileForm.qq"
            :maxlength="INPUT_LIMITS.QQ"
            placeholder="Enter QQ number"
          />
        </ElFormItem>
      </ElForm>

      <template #footer>
        <div class="dialog-footer">
          <ElButton @click="profileDialogVisible = false">Cancel</ElButton>
          <ElButton type="primary" :loading="profileSubmitting" @click="handleUpdateProfile">
            Save
          </ElButton>
        </div>
      </template>
    </ElDialog>

    <ElDialog
      v-model="passwordDialogVisible"
      title="Change Password"
      width="520px"
      destroy-on-close
      align-center
    >
      <ElForm
        ref="passwordFormRef"
        :model="passwordForm"
        :rules="passwordRules"
        label-position="top"
      >
        <ElFormItem label="Current Password" prop="old_password">
          <ElInput
            v-model="passwordForm.old_password"
            type="password"
            show-password
            :maxlength="INPUT_LIMITS.PASSWORD"
            placeholder="Enter current password"
          />
        </ElFormItem>

        <ElFormItem label="New Password" prop="new_password">
          <ElInput
            v-model="passwordForm.new_password"
            type="password"
            show-password
            :maxlength="INPUT_LIMITS.PASSWORD"
            placeholder="Enter new password"
          />
        </ElFormItem>

        <ElFormItem label="Confirm New Password" prop="confirm_password">
          <ElInput
            v-model="passwordForm.confirm_password"
            type="password"
            show-password
            :maxlength="INPUT_LIMITS.PASSWORD"
            placeholder="Enter new password again"
          />
        </ElFormItem>
      </ElForm>

      <template #footer>
        <div class="dialog-footer">
          <ElButton @click="passwordDialogVisible = false">Cancel</ElButton>
          <ElButton type="primary" :loading="passwordSubmitting" @click="handleChangePassword">
            Save
          </ElButton>
        </div>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import type { AdminProfile, PermissionGroupRecord, PermissionRecord } from '@/api/admin'
  import {
    changeAdminPassword,
    fetchAdminPermissions,
    fetchAdminProfile,
    fetchPermissionGroups,
    mapAdminProfileToUserInfo,
    updateAdminProfile
  } from '@/api/admin'
  import { useUserStore } from '@/store/modules/user'
  import { INPUT_LIMITS } from '@/utils/constants'
  import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
  import { Lock } from '@element-plus/icons-vue'

  defineOptions({ name: 'UserCenter' })

  interface ProfileFormState {
    email: string
    qq: string
  }

  interface PasswordFormState {
    old_password: string
    new_password: string
    confirm_password: string
  }

  const userStore = useUserStore()
  const { info } = storeToRefs(userStore)

  const pageLoading = ref(false)
  const profileSubmitting = ref(false)
  const passwordSubmitting = ref(false)

  const profile = ref<AdminProfile>({})
  const permissionGroups = ref<PermissionGroupRecord[]>([])
  const allPermissions = ref<PermissionRecord[]>([])

  const profileDialogVisible = ref(false)
  const passwordDialogVisible = ref(false)

  const profileFormRef = ref<FormInstance>()
  const passwordFormRef = ref<FormInstance>()

  const profileForm = reactive<ProfileFormState>({
    email: '',
    qq: ''
  })

  const passwordForm = reactive<PasswordFormState>({
    old_password: '',
    new_password: '',
    confirm_password: ''
  })

  const grantedPermissions = computed(() => {
    if (Array.isArray(profile.value.permissions) && profile.value.permissions.length) {
      return profile.value.permissions.filter(Boolean)
    }

    if (Array.isArray(info.value?.buttons) && info.value.buttons.length) {
      return info.value.buttons.filter(Boolean)
    }

    return []
  })

  const permissionCount = computed(() => grantedPermissions.value.length)

  const permissionGroupMap = computed(() => {
    const map = new Map<number, string>()

    permissionGroups.value.forEach((group) => {
      const id = normalizeNullableNumber(group.id ?? group.ID)
      if (id === null) {
        return
      }

      map.set(id, String(group.name ?? group.Name ?? '-'))
    })

    return map
  })

  const permissionLabelMap = computed(() => {
    const map = new Map<string, string>()

    allPermissions.value.forEach((permission) => {
      const code = String(permission.code ?? '').trim()
      if (!code) {
        return
      }

      map.set(code, String(permission.friendly_name ?? permission.name ?? code))
    })

    return map
  })

  const profileName = computed(() =>
    String(profile.value.username || info.value?.userName || 'Admin')
  )

  const profileInitial = computed(() => profileName.value.slice(0, 1).toUpperCase() || 'A')

  const avatarUrl = computed(() => {
    const avatar = String(
      profile.value.avatar || profile.value.avatar_url || info.value?.avatar || ''
    ).trim()
    if (avatar) {
      return avatar
    }

    const qq = String(profile.value.qq || '').trim()
    return qq ? `https://q1.qlogo.cn/g?b=qq&nk=${qq}&s=100` : ''
  })

  const roleLabel = computed(() => {
    const role = String(profile.value.role || '').trim()
    if (role === 'admin') {
      return 'Administrator'
    }
    return role || 'Unknown'
  })

  const statusLabel = computed(() => {
    const status = String(profile.value.status || '').trim()
    if (status === 'active') {
      return 'Active'
    }
    if (status === 'disabled') {
      return 'Disabled'
    }
    return status || '-'
  })

  const permissionGroupLabel = computed(() => {
    const directLabel = String(profile.value.permission_group_name || '').trim()
    if (directLabel) {
      return directLabel
    }

    const id = normalizeNullableNumber(profile.value.permission_group_id)
    if (id !== null && permissionGroupMap.value.has(id)) {
      return permissionGroupMap.value.get(id) || '-'
    }

    return roleLabel.value
  })

  const permissionTagType = computed(() => {
    if (permissionCount.value > 20) {
      return 'danger' as const
    }
    if (permissionCount.value > 10) {
      return 'warning' as const
    }
    return 'success' as const
  })

  const profileRules = computed<FormRules>(() => ({
    email: [
      { required: true, message: 'Please enter an email address', trigger: 'blur' },
      {
        type: 'email',
        message: 'Please enter a valid email address',
        trigger: ['blur', 'change']
      },
      {
        max: INPUT_LIMITS.EMAIL,
        message: `Email must be at most ${INPUT_LIMITS.EMAIL} characters`,
        trigger: 'blur'
      }
    ],
    qq: [
      {
        validator: (_rule, value, callback) => {
          const text = String(value || '').trim()
          if (text && !/^\d+$/.test(text)) {
            callback(new Error('QQ must contain digits only'))
            return
          }
          callback()
        },
        trigger: 'blur'
      },
      {
        max: INPUT_LIMITS.QQ,
        message: `QQ must be at most ${INPUT_LIMITS.QQ} characters`,
        trigger: 'blur'
      }
    ]
  }))

  const passwordRules = computed<FormRules>(() => ({
    old_password: [
      { required: true, message: 'Please enter the current password', trigger: 'blur' }
    ],
    new_password: [
      { required: true, message: 'Please enter a new password', trigger: 'blur' },
      { min: 6, message: 'New password must be at least 6 characters', trigger: 'blur' },
      {
        max: INPUT_LIMITS.PASSWORD,
        message: `New password must be at most ${INPUT_LIMITS.PASSWORD} characters`,
        trigger: 'blur'
      }
    ],
    confirm_password: [
      { required: true, message: 'Please confirm the new password', trigger: 'blur' },
      {
        validator: (_rule, value, callback) => {
          if (String(value || '') !== String(passwordForm.new_password || '')) {
            callback(new Error('The two passwords do not match'))
            return
          }
          callback()
        },
        trigger: 'blur'
      }
    ]
  }))

  onMounted(() => {
    initializePage()
  })

  function normalizeNullableNumber(value: unknown): number | null {
    if (value === '' || value === null || value === undefined) {
      return null
    }

    const parsed = Number(value)
    return Number.isFinite(parsed) ? parsed : null
  }

  function permissionLabel(code: string) {
    return permissionLabelMap.value.get(code) || code || '-'
  }

  function formatDate(value?: string | null) {
    if (!value) {
      return '-'
    }

    const date = new Date(value)
    if (Number.isNaN(date.getTime())) {
      return value
    }

    return date.toLocaleDateString('zh-CN')
  }

  function formatDateTime(value?: string | null) {
    if (!value) {
      return '-'
    }

    const date = new Date(value)
    return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN')
  }

  async function initializePage() {
    pageLoading.value = true

    try {
      await Promise.allSettled([
        fetchProfileData(),
        fetchPermissionGroupsData(),
        fetchAllPermissionsData()
      ])
    } finally {
      pageLoading.value = false
    }
  }

  async function fetchProfileData() {
    try {
      const payload = await fetchAdminProfile()
      profile.value = payload || {}
      profileForm.email = String(profile.value.email || '')
      profileForm.qq = String(profile.value.qq || '')
      userStore.setUserInfo(mapAdminProfileToUserInfo(profile.value))
    } catch (error: any) {
      ElMessage.error(error?.response?.data?.error || 'Failed to load profile')
    }
  }

  async function fetchPermissionGroupsData() {
    try {
      const payload = await fetchPermissionGroups()
      permissionGroups.value = payload.items || []
    } catch {
      permissionGroups.value = []
    }
  }

  async function fetchAllPermissionsData() {
    try {
      const payload = await fetchAdminPermissions()
      allPermissions.value = payload.items || []
    } catch {
      allPermissions.value = []
    }
  }

  function openEditProfile() {
    profileForm.email = String(profile.value.email || '')
    profileForm.qq = String(profile.value.qq || '')
    profileDialogVisible.value = true
    nextTick(() => profileFormRef.value?.clearValidate())
  }

  function openChangePassword() {
    passwordForm.old_password = ''
    passwordForm.new_password = ''
    passwordForm.confirm_password = ''
    passwordDialogVisible.value = true
    nextTick(() => passwordFormRef.value?.clearValidate())
  }

  async function handleUpdateProfile() {
    if (!profileFormRef.value) {
      return
    }

    const valid = await profileFormRef.value.validate().catch(() => false)
    if (!valid) {
      return
    }

    profileSubmitting.value = true

    try {
      await updateAdminProfile({
        email: String(profileForm.email || '').trim(),
        qq: String(profileForm.qq || '').trim()
      })
      ElMessage.success('Profile updated successfully')
      profileDialogVisible.value = false
      await fetchProfileData()
    } catch (error: any) {
      ElMessage.error(error?.response?.data?.error || 'Failed to update profile')
    } finally {
      profileSubmitting.value = false
    }
  }

  async function handleChangePassword() {
    if (!passwordFormRef.value) {
      return
    }

    const valid = await passwordFormRef.value.validate().catch(() => false)
    if (!valid) {
      return
    }

    passwordSubmitting.value = true

    try {
      await changeAdminPassword({
        old_password: passwordForm.old_password,
        new_password: passwordForm.new_password
      })
      ElMessage.success(
        'Password updated successfully. Please use the new password next time you log in.'
      )
      passwordDialogVisible.value = false
      passwordForm.old_password = ''
      passwordForm.new_password = ''
      passwordForm.confirm_password = ''
    } catch (error: any) {
      ElMessage.error(error?.response?.data?.error || 'Failed to change password')
    } finally {
      passwordSubmitting.value = false
    }
  }
</script>

<style scoped lang="scss">
  .profile-page {
    padding-top: 2px;
  }

  .hero-card,
  .section-card {
    margin-bottom: 16px;
  }

  .hero-card {
    text-align: center;
  }

  .hero-avatar-wrap {
    display: flex;
    justify-content: center;
    margin-bottom: 16px;
  }

  .hero-name {
    margin: 0;
    color: var(--el-text-color-primary);
    font-size: 24px;
    font-weight: 700;
  }

  .hero-role {
    margin: 8px 0 18px;
    color: var(--el-text-color-secondary);
    font-size: 14px;
  }

  .hero-stats {
    display: grid;
    gap: 12px;
    margin-top: 20px;
    text-align: left;
  }

  .stat-item {
    padding: 12px 14px;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 12px;
    background: var(--el-fill-color-lighter);
  }

  .stat-label {
    display: block;
    margin-bottom: 6px;
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }

  .section-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
  }

  .section-title {
    color: var(--el-text-color-primary);
    font-size: 18px;
    font-weight: 700;
  }

  .section-subtitle {
    margin-top: 6px;
    color: var(--el-text-color-secondary);
    font-size: 13px;
    line-height: 1.6;
  }

  .security-panel {
    display: flex;
    gap: 16px;
    align-items: flex-start;
  }

  .security-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 44px;
    height: 44px;
    border-radius: 12px;
    color: var(--el-color-primary);
    font-size: 18px;
    background: color-mix(in srgb, var(--el-color-primary) 12%, white);
  }

  .security-title {
    color: var(--el-text-color-primary);
    font-size: 15px;
    font-weight: 600;
  }

  .security-copy p {
    margin: 8px 0 0;
    color: var(--el-text-color-secondary);
    line-height: 1.7;
  }

  .permission-list {
    display: flex;
    flex-wrap: wrap;
    gap: 10px;
  }

  .dialog-footer {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
  }

  @media (max-width: 768px) {
    .section-header {
      align-items: flex-start;
      flex-direction: column;
    }
  }
</style>
