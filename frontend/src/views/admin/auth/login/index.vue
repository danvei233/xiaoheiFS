<template>
  <div class="auth-page">
    <LoginLeftView />

    <main class="auth-main">
      <AuthTopBar />

      <div class="auth-right-wrap">
        <div class="auth-form">
          <div class="admin-kicker">运营管理后台</div>
          <h1 class="auth-title">管理员登录</h1>
          <p class="auth-subtitle">面向运营与审核的管理中心，实时掌控订单与资源。</p>

          <ElForm
            ref="formRef"
            :model="form"
            :rules="rules"
            label-position="top"
            @keyup.enter="handleSubmit"
          >
            <ElFormItem label="用户名" prop="username">
              <ElInput v-model="form.username" size="large" autocomplete="username" clearable>
                <template #prefix><ArtSvgIcon icon="ri:admin-line" /></template>
              </ElInput>
            </ElFormItem>
            <ElFormItem label="密码" prop="password">
              <ElInput
                v-model="form.password"
                type="password"
                size="large"
                :maxlength="INPUT_LIMITS.PASSWORD"
                autocomplete="current-password"
                show-password
              >
                <template #prefix><ArtSvgIcon icon="ri:lock-password-line" /></template>
              </ElInput>
            </ElFormItem>

            <div class="auth-actions">
              <RouterLink :to="buildAdminUrl('forgot-password')">忘记密码？</RouterLink>
            </div>
            <ElButton
              class="submit-button"
              type="primary"
              size="large"
              :loading="admin.loading"
              @click="handleSubmit"
              v-ripple
            >
              登录
            </ElButton>
          </ElForm>
        </div>
      </div>
    </main>

    <ElDialog
      :model-value="mfaDialogVisible"
      width="min(560px, calc(100vw - 32px))"
      align-center
      :show-close="false"
      :close-on-click-modal="false"
      :close-on-press-escape="false"
      class="admin-mfa-dialog"
    >
      <template #header>
        <div class="mfa-header">
          <div class="mfa-title">{{ mfaDialogTitle }}</div>
          <div class="mfa-subtitle">{{ mfaDialogSubtitle }}</div>
        </div>
      </template>

      <ElAlert
        class="mfa-alert"
        :title="mfaError || mfaDialogAlert"
        :type="mfaError ? 'error' : admin.mfaBindRequired ? 'warning' : 'info'"
        :closable="false"
        show-icon
      />

      <ElForm label-position="top" class="mfa-form" @submit.prevent>
        <ElFormItem v-if="admin.mfaBindRequired" label="登录密码">
          <ElInput
            v-model="mfaForm.password"
            type="password"
            :maxlength="INPUT_LIMITS.PASSWORD"
            autocomplete="current-password"
            show-password
            placeholder="用于生成 2FA 绑定信息，如未要求可留空"
          />
        </ElFormItem>

        <ElFormItem v-if="admin.mfaBindRequired" label="生成绑定信息">
          <ElButton class="mfa-setup-button" :loading="mfaLoading.setup" @click="handleSetup">
            生成 2FA 绑定信息
          </ElButton>
        </ElFormItem>

        <div v-if="mfaOtpAuthUrl && mfaSecret" class="mfa-setup-panel">
          <div class="mfa-qr-card">
            <QrcodeVue :value="mfaOtpAuthUrl" :size="168" level="M" render-as="svg" />
          </div>
          <div class="mfa-secret-card">
            <div class="mfa-secret-label">手动密钥</div>
            <div class="mfa-secret-value">{{ mfaSecret }}</div>
            <div class="mfa-secret-tip">
              请使用身份验证器扫码，然后输入当前显示的 6 位验证码。
            </div>
          </div>
        </div>

        <ElFormItem label="2FA 验证码">
          <ElInput
            v-model="mfaForm.totpCode"
            maxlength="6"
            inputmode="numeric"
            autocomplete="one-time-code"
            placeholder="请输入 6 位验证码"
            @input="handleTotpInput"
            @keyup.enter="handleConfirmOrUnlock"
          />
        </ElFormItem>
      </ElForm>

      <div class="mfa-helper">验证码每 30 秒更新，请使用最新口令。</div>

      <template #footer>
        <div class="mfa-footer">
          <ElButton @click="resetPendingLogin">重新登录</ElButton>
          <ElButton
            type="primary"
            :loading="mfaLoading.confirm"
            :disabled="admin.mfaBindRequired && !mfaSecret"
            @click="handleConfirmOrUnlock"
          >
            {{ admin.mfaBindRequired ? '完成绑定并登录' : '验证并登录' }}
          </ElButton>
        </div>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import type { FormInstance, FormRules } from 'element-plus'
  import QrcodeVue from 'qrcode.vue'
  import { admin2FAUnlock, adminConfirmTwoFA, adminSetupTwoFA } from '@/services/admin'
  import { useAdminAuthStore } from '@/stores/adminAuth'
  import { buildAdminUrl, getCachedAdminPath } from '@/services/adminPath'
  import { INPUT_LIMITS } from '@/constants/inputLimits'
  import { clearSession } from '@/router/session'
  import { resolveAdminRedirect } from '@/utils/auth'

  defineOptions({ name: 'AdminLogin' })

  const admin = useAdminAuthStore()
  const router = useRouter()
  const route = useRoute()
  const formRef = ref<FormInstance>()
  const form = reactive({ username: '', password: '' })
  const mfaForm = reactive({ password: '', totpCode: '' })
  const mfaLoading = reactive({ setup: false, confirm: false })
  const mfaSecret = ref('')
  const mfaOtpAuthUrl = ref('')
  const mfaError = ref('')
  const rules: FormRules = {
    username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
    password: [{ required: true, message: '请输入密码', trigger: 'blur' }]
  }

  const getCurrentAdminPath = () => {
    const pathSegments = route.path.split('/').filter(Boolean)
    return pathSegments[0] || getCachedAdminPath()
  }

  const mfaDialogVisible = computed(() => admin.requiresMfa)
  const mfaDialogTitle = computed(() =>
    admin.mfaBindRequired ? '管理员 2FA 绑定' : '管理员 2FA 验证'
  )
  const mfaDialogSubtitle = computed(() =>
    admin.mfaBindRequired
      ? '首次登录需要先完成 2FA 绑定，然后才能进入后台。'
      : '请输入当前动态验证码，完成后台登录。'
  )
  const mfaDialogAlert = computed(() =>
    admin.mfaBindRequired
      ? '需要先绑定 2FA 才能继续使用后台功能。'
      : '请输入 2FA 验证码以完成后台登录。'
  )

  onMounted(() => {
    // 允许从已登录后台主动打开登录页切换账号，同时保留刷新后的待 MFA 会话。
    if (admin.token && !admin.requiresMfa) {
      clearSession('admin')
    }
  })

  const handleSubmit = async () => {
    const valid = await formRef.value?.validate().catch(() => false)
    if (!valid) return
    if (form.password.length > INPUT_LIMITS.PASSWORD) {
      ElMessage.error(`密码长度不能超过 ${INPUT_LIMITS.PASSWORD} 个字符`)
      return
    }

    const adminPath = getCurrentAdminPath()
    try {
      resetMfaForm()
      const response = await admin.login({
        ...form,
        admin_path: adminPath
      })
      if (!response.access_token) {
        ElMessage.error('登录失败：服务端未返回访问令牌')
        admin.logout()
        return
      }

      if (admin.requiresMfa) {
        if (admin.mfaBindRequired) mfaForm.password = form.password
        return
      }

      await continueToAdmin()
    } catch {
      // 请求层已经展示后端返回的登录错误。
    }
  }

  const handleSetup = async () => {
    if (!admin.mfaBindRequired || mfaLoading.setup) return

    mfaLoading.setup = true
    mfaError.value = ''
    try {
      const password = mfaForm.password.trim()
      const response = await adminSetupTwoFA({ password: password || undefined })
      mfaSecret.value = String(response.data?.secret || '')
      mfaOtpAuthUrl.value = String(response.data?.otpauth_url || '')
      if (!mfaSecret.value || !mfaOtpAuthUrl.value) {
        mfaSecret.value = ''
        mfaOtpAuthUrl.value = ''
        mfaError.value = '生成 2FA 绑定信息失败，请重试'
        return
      }
      ElMessage.success('已生成 2FA 绑定信息')
    } catch (error) {
      mfaError.value = getErrorMessage(error, '生成 2FA 绑定信息失败')
    } finally {
      mfaLoading.setup = false
    }
  }

  const handleConfirmOrUnlock = async () => {
    if (mfaLoading.confirm) return

    const code = normalizeTotpCode(mfaForm.totpCode)
    if (!/^\d{6}$/.test(code)) {
      mfaError.value = '请输入 6 位 2FA 验证码'
      return
    }
    if (admin.mfaBindRequired && !mfaSecret.value) {
      mfaError.value = '请先生成 2FA 绑定信息'
      return
    }

    mfaLoading.confirm = true
    mfaError.value = ''
    let bindCompleted = false
    try {
      if (admin.mfaBindRequired) {
        await adminConfirmTwoFA({ code })
        bindCompleted = true
      }

      const response = await admin2FAUnlock({ totp_code: code })
      const token = response.data?.access_token || ''
      if (!token) {
        throw new Error('2FA 验证成功，但服务端未返回访问令牌')
      }

      admin.completeMfa(token)
      resetMfaForm()
      ElMessage.success(bindCompleted ? '2FA 已绑定，登录成功' : '2FA 验证成功')
      await continueToAdmin()
    } catch (error) {
      if (bindCompleted) {
        admin.setMfaGateState({
          totpEnabled: true,
          mfaRequired: true,
          mfaBindRequired: false,
          mfaUnlocked: false
        })
        mfaForm.password = ''
        mfaForm.totpCode = ''
        mfaSecret.value = ''
        mfaOtpAuthUrl.value = ''
      }
      mfaError.value = getErrorMessage(error, '2FA 验证失败，请使用最新验证码重试')
    } finally {
      mfaLoading.confirm = false
    }
  }

  const handleTotpInput = () => {
    mfaForm.totpCode = normalizeTotpCode(mfaForm.totpCode)
    mfaError.value = ''
  }

  const continueToAdmin = async () => {
    const target = resolveAdminRedirect(route.query.redirect, getCurrentAdminPath())
    await router.replace(target)
  }

  const resetPendingLogin = () => {
    clearSession('admin')
    resetMfaForm()
    mfaLoading.setup = false
    mfaLoading.confirm = false
    form.password = ''
  }

  const resetMfaForm = () => {
    mfaForm.password = ''
    mfaForm.totpCode = ''
    mfaSecret.value = ''
    mfaOtpAuthUrl.value = ''
    mfaError.value = ''
  }

  const normalizeTotpCode = (value: string): string =>
    String(value || '')
      .replace(/\D/g, '')
      .slice(0, 6)

  const getErrorMessage = (error: unknown, fallback: string): string => {
    if (!error || typeof error !== 'object') return fallback
    const response = (error as { response?: { data?: { error?: unknown; message?: unknown } } })
      .response
    const message = response?.data?.error || response?.data?.message || (error as Error).message
    return typeof message === 'string' && message.trim() ? message : fallback
  }
</script>

<style lang="scss" scoped>
  .auth-page {
    display: flex;
    width: 100%;
    height: 100vh;
    min-height: 100vh;
    background: var(--default-box-color);
  }

  .auth-main {
    position: relative;
    flex: 1;
    min-width: 0;
  }

  .auth-right-wrap {
    position: absolute;
    inset: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    width: min(440px, 100%);
    min-height: 620px;
    padding: 72px 28px 32px;
    margin: auto;
  }

  .auth-form {
    width: 100%;
  }

  .admin-kicker {
    margin-bottom: 8px;
    font-size: 13px;
    font-weight: 600;
    color: var(--theme-color);
    letter-spacing: 0.08em;
  }

  .auth-title {
    margin: 0;
    font-size: 36px;
    font-weight: 600;
    color: var(--art-gray-900);
  }

  .auth-subtitle {
    margin: 10px 0 24px;
    font-size: 14px;
    line-height: 1.7;
    color: var(--art-gray-600);
  }

  .auth-actions {
    margin: 6px 0 20px;
    font-size: 14px;
    text-align: right;
  }

  .auth-actions a {
    color: var(--theme-color);
  }

  .submit-button {
    width: 100%;
    height: 40px;
  }

  .mfa-header {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .mfa-title {
    font-size: 20px;
    font-weight: 700;
    line-height: 1.2;
    color: var(--art-gray-900);
  }

  .mfa-subtitle,
  .mfa-secret-tip,
  .mfa-helper {
    font-size: 13px;
    line-height: 1.6;
    color: var(--art-gray-600);
  }

  .mfa-alert {
    margin-bottom: 16px;
  }

  .mfa-form {
    margin-top: 8px;
  }

  .mfa-setup-button {
    width: 100%;
  }

  .mfa-setup-panel {
    display: grid;
    grid-template-columns: 168px minmax(0, 1fr);
    gap: 16px;
    align-items: center;
    margin-bottom: 18px;
  }

  .mfa-qr-card,
  .mfa-secret-card {
    padding: 14px;
    background: var(--art-gray-100);
    border: 1px solid var(--default-border);
    border-radius: calc(var(--custom-radius) / 2 + 2px);
  }

  .mfa-qr-card {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 196px;
  }

  .mfa-secret-card {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .mfa-secret-label {
    font-size: 12px;
    color: var(--art-gray-600);
  }

  .mfa-secret-value {
    font-family: 'Courier New', Consolas, monospace;
    font-size: 16px;
    font-weight: 700;
    line-height: 1.5;
    color: var(--art-gray-900);
    word-break: break-all;
  }

  .mfa-helper {
    margin-top: -2px;
  }

  .mfa-footer {
    display: flex;
    gap: 12px;
    justify-content: flex-end;
  }

  @media (width <= 640px) {
    .auth-right-wrap {
      position: relative;
      min-height: 100vh;
      padding-inline: 24px;
    }

    .auth-title {
      font-size: 30px;
    }

    .mfa-setup-panel {
      grid-template-columns: 1fr;
    }
  }
</style>
