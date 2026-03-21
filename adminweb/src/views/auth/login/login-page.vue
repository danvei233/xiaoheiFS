<template>
  <div class="flex h-screen w-full">
    <LoginLeftView />

    <div class="relative flex-1">
      <AuthTopBar />

      <div class="auth-right-wrap">
        <div class="form">
          <h3 class="title">{{ $t('login.title') }}</h3>
          <p class="sub-title">{{ $t('login.subTitle') }}</p>

          <ElForm
            ref="formRef"
            :key="formKey"
            :model="formData"
            :rules="rules"
            style="margin-top: 25px"
            @keyup.enter="handleSubmit"
          >
            <ElFormItem prop="username">
              <ElInput
                v-model.trim="formData.username"
                class="custom-height"
                :placeholder="$t('login.placeholder.username')"
              />
            </ElFormItem>

            <ElFormItem prop="password">
              <ElInput
                v-model.trim="formData.password"
                class="custom-height"
                :placeholder="$t('login.placeholder.password')"
                type="password"
                autocomplete="off"
                show-password
              />
            </ElFormItem>

            <div class="relative mt-6 pb-5">
              <div
                class="relative z-[2] overflow-hidden rounded-lg border border-transparent select-none tad-300"
                :class="{ '!border-[#FF4E4F]': !isPassing && isClickPass }"
              >
                <ArtDragVerify
                  ref="dragVerify"
                  v-model:value="isPassing"
                  :text="$t('login.sliderText')"
                  textColor="var(--art-gray-700)"
                  :successText="$t('login.sliderSuccessText')"
                  progressBarBg="var(--main-color)"
                  :background="isDark ? '#26272F' : '#F1F1F4'"
                  handlerBg="var(--default-box-color)"
                />
              </div>

              <p
                class="absolute top-0 z-[1] mt-2 px-px text-xs text-[#f56c6c] tad-300"
                :class="{ 'translate-y-10': !isPassing && isClickPass }"
              >
                {{ $t('login.placeholder.slider') }}
              </p>
            </div>

            <div class="mt-2 flex-cb text-sm">
              <ElCheckbox v-model="formData.rememberPassword">
                {{ $t('login.rememberPwd') }}
              </ElCheckbox>

              <RouterLink class="text-theme" :to="{ name: 'ForgetPassword' }">
                {{ $t('login.forgetPwd') }}
              </RouterLink>
            </div>

            <div style="margin-top: 30px">
              <ElButton
                class="custom-height w-full"
                type="primary"
                :loading="loading"
                v-ripple
                @click="handleSubmit"
              >
                {{ $t('login.btnText') }}
              </ElButton>
            </div>
          </ElForm>
        </div>
      </div>
    </div>

    <ElDialog
      :model-value="mfaDialogVisible"
      width="560px"
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
        :title="mfaDialogAlert"
        :type="mfaState.bindRequired ? 'warning' : 'info'"
        :closable="false"
        show-icon
      />

      <ElForm label-position="top" class="mfa-form">
        <ElFormItem v-if="mfaState.bindRequired" label="登录密码">
          <ElInput
            v-model.trim="mfaForm.password"
            type="password"
            show-password
            placeholder="用于生成 2FA 绑定信息，如未要求可留空"
          />
        </ElFormItem>

        <ElFormItem v-if="mfaState.bindRequired" label="生成绑定信息">
          <ElButton class="w-full" :loading="mfaLoading.setup" @click="handleSetupTwoFactor">
            生成 2FA 绑定信息
          </ElButton>
        </ElFormItem>

        <div v-if="mfaOtpAuthUrl && mfaSecret" class="mfa-setup-panel">
          <div class="mfa-qr-card">
            <AsyncQrcodeVue :value="mfaOtpAuthUrl" :size="168" level="M" render-as="svg" />
          </div>

          <div class="mfa-secret-card">
            <div class="mfa-secret-label">手动密钥</div>
            <div class="mfa-secret-value">{{ mfaSecret }}</div>
            <div class="mfa-secret-tip">
              请使用 Google Authenticator 或 Microsoft Authenticator 扫码，然后输入 6 位验证码。
            </div>
          </div>
        </div>

        <ElFormItem label="2FA 验证码">
          <ElInput
            v-model.trim="mfaForm.totpCode"
            maxlength="6"
            inputmode="numeric"
            placeholder="请输入 6 位验证码"
            @input="handleTotpInput"
          />
        </ElFormItem>
      </ElForm>

      <div class="mfa-helper">验证码每 30 秒更新，请使用最新口令。</div>

      <template #footer>
        <div class="mfa-footer">
          <ElButton @click="resetPendingLogin">重新登录</ElButton>
          <ElButton type="primary" :loading="mfaLoading.confirm" @click="handleConfirmOrUnlock">
            {{ mfaState.bindRequired ? '完成绑定并登录' : '验证并登录' }}
          </ElButton>
        </div>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import { defineAsyncComponent } from 'vue'
  import { useI18n } from 'vue-i18n'
  import { ElMessage, ElNotification, type FormInstance, type FormRules } from 'element-plus'
  import { fetchLogin } from '@/api/auth'
  import {
    confirmAdminTwoFactor,
    setupAdminTwoFactor,
    type AdminLoginResponse,
    unlockAdminTwoFactor
  } from '@/api/admin'
  import { useSettingStore } from '@/store/modules/setting'
  import { useSiteStore } from '@/store/modules/site'
  import { useUserStore } from '@/store/modules/user'
  import { HttpError } from '@/utils/http/error'
  import {
    buildAdminHashUrl,
    ensureCurrentAdminPath,
    resolveLoginAdminPath
  } from '@/utils/adminPath'

  defineOptions({ name: 'Login' })

  const settingStore = useSettingStore()
  const siteStore = useSiteStore()
  const userStore = useUserStore()
  const AsyncQrcodeVue = defineAsyncComponent(() => import('qrcode.vue'))
  const router = useRouter()
  const route = useRoute()
  const { isDark } = storeToRefs(settingStore)
  const { t, locale } = useI18n()

  const formKey = ref(0)
  const formRef = ref<FormInstance>()
  const dragVerify = ref<{ reset?: () => void } | null>(null)
  const loading = ref(false)
  const isPassing = ref(false)
  const isClickPass = ref(false)
  const pendingRedirect = ref('/')
  const mfaSecret = ref('')
  const mfaOtpAuthUrl = ref('')

  const formData = reactive({
    username: '',
    password: '',
    rememberPassword: true
  })

  const mfaForm = reactive({
    password: '',
    totpCode: ''
  })

  const mfaLoading = reactive({
    setup: false,
    confirm: false
  })

  const mfaState = reactive({
    required: false,
    bindRequired: false,
    unlocked: true,
    totpEnabled: false
  })

  const systemName = computed(() => siteStore.resolvedSiteName)
  const mfaDialogVisible = computed(
    () => (mfaState.required || mfaState.bindRequired) && !mfaState.unlocked
  )
  const mfaDialogTitle = computed(() =>
    mfaState.bindRequired ? '管理员 2FA 绑定' : '管理员 2FA 验证'
  )
  const mfaDialogSubtitle = computed(() =>
    mfaState.bindRequired
      ? '首次登录需要先完成 2FA 绑定，然后才能进入后台。'
      : '请输入当前动态验证码，完成后台登录。'
  )
  const mfaDialogAlert = computed(() =>
    mfaState.bindRequired
      ? '需要先绑定 2FA 才能继续使用后台功能。'
      : '请输入 2FA 验证码以完成后台登录。'
  )

  const rules = computed<FormRules>(() => ({
    username: [{ required: true, message: t('login.placeholder.username'), trigger: 'blur' }],
    password: [{ required: true, message: t('login.placeholder.password'), trigger: 'blur' }]
  }))

  watch(locale, () => {
    formKey.value++
  })

  onMounted(async () => {
    if (userStore.accessToken && !userStore.isLogin) {
      userStore.setToken('', '')
    }

    await tryRestoreHiddenAdminEntry()
  })

  async function handleSubmit() {
    if (!formRef.value) {
      return
    }

    try {
      const valid = await formRef.value.validate()
      if (!valid) {
        return
      }

      if (!isPassing.value) {
        isClickPass.value = true
        return
      }

      loading.value = true
      clearMfaGateState()
      userStore.setLoginStatus(false)
      userStore.setToken('', '')

      const redirect = String((route.query.redirect as string) || '/')
      const adminPath = await resolveLoginAdminPath()
      const loginResponse = await fetchLogin({
        username: formData.username,
        password: formData.password,
        admin_path: adminPath
      })

      if (!loginResponse.access_token) {
        throw new Error('Login failed - no token received')
      }

      applyMfaGateState(loginResponse, redirect)

      if (mfaDialogVisible.value) {
        userStore.setToken(loginResponse.access_token, loginResponse.refresh_token || '')
        return
      }

      userStore.setToken(loginResponse.access_token, loginResponse.refresh_token || '')
      userStore.setLoginStatus(true)
      showLoginSuccessNotice()
      await router.push(redirect)
    } catch (error) {
      if (error instanceof HttpError && error.code === 403) {
        await tryRestoreHiddenAdminEntry(true)
      } else if (!(error instanceof HttpError)) {
        console.error('[Login] Unexpected error:', error)
      }
    } finally {
      loading.value = false
      resetDragVerify()
    }
  }

  function applyMfaGateState(loginResponse: AdminLoginResponse, redirect: string) {
    pendingRedirect.value = redirect || '/'
    mfaState.required = Boolean(loginResponse.mfa_required)
    mfaState.bindRequired = Boolean(loginResponse.mfa_bind_required)
    mfaState.unlocked = Boolean(loginResponse.mfa_unlocked)
    mfaState.totpEnabled = Boolean(loginResponse.totp_enabled)

    if (!mfaDialogVisible.value) {
      resetMfaForm()
      return
    }

    if (!mfaState.bindRequired) {
      mfaForm.password = ''
      mfaSecret.value = ''
      mfaOtpAuthUrl.value = ''
    }

    mfaForm.totpCode = ''
  }

  function resetMfaForm() {
    mfaForm.password = ''
    mfaForm.totpCode = ''
    mfaSecret.value = ''
    mfaOtpAuthUrl.value = ''
  }

  function clearMfaGateState() {
    mfaState.required = false
    mfaState.bindRequired = false
    mfaState.unlocked = true
    mfaState.totpEnabled = false
    pendingRedirect.value = '/'
    resetMfaForm()
  }

  async function handleSetupTwoFactor() {
    if (!mfaState.bindRequired || mfaLoading.setup) {
      return
    }

    mfaLoading.setup = true

    try {
      const password = String(mfaForm.password || '').trim()
      const response = await setupAdminTwoFactor({
        password: password || undefined
      })

      mfaSecret.value = String(response.secret || '')
      mfaOtpAuthUrl.value = String(response.otpauth_url || '')

      if (!mfaSecret.value || !mfaOtpAuthUrl.value) {
        ElMessage.error('生成 2FA 绑定信息失败')
        mfaSecret.value = ''
        mfaOtpAuthUrl.value = ''
        return
      }

      ElMessage.success('已生成 2FA 绑定信息')
    } catch (error) {
      if (!(error instanceof HttpError)) {
        console.error('[Login][2FA Setup] Unexpected error:', error)
      }
    } finally {
      mfaLoading.setup = false
    }
  }

  async function handleConfirmOrUnlock() {
    if (mfaLoading.confirm) {
      return
    }

    const code = String(mfaForm.totpCode || '')
      .replace(/\D/g, '')
      .slice(0, 6)

    if (!/^\d{6}$/.test(code)) {
      ElMessage.warning('请输入 6 位 2FA 验证码')
      return
    }

    mfaLoading.confirm = true

    try {
      if (mfaState.bindRequired) {
        if (!mfaSecret.value) {
          ElMessage.warning('请先生成绑定信息')
          return
        }

        await confirmAdminTwoFactor({ code })
        mfaState.bindRequired = false
        mfaState.required = true
        mfaState.unlocked = false
        mfaState.totpEnabled = true
        mfaForm.password = ''
        mfaForm.totpCode = ''
        mfaSecret.value = ''
        mfaOtpAuthUrl.value = ''
        ElMessage.success('2FA 已绑定，请输入最新验证码完成登录')
        return
      }

      const response = await unlockAdminTwoFactor({ totp_code: code })
      if (!response.access_token) {
        throw new Error('2FA unlock failed - no token received')
      }

      const redirect = pendingRedirect.value || '/'

      userStore.setToken(response.access_token, response.refresh_token || '')
      userStore.setLoginStatus(true)
      clearMfaGateState()
      resetDragVerify()
      showLoginSuccessNotice()
      await router.push(redirect)
    } catch (error) {
      if (error instanceof HttpError) {
        return
      }

      console.error('[Login][2FA Confirm] Unexpected error:', error)
    } finally {
      mfaLoading.confirm = false
    }
  }

  function handleTotpInput() {
    mfaForm.totpCode = String(mfaForm.totpCode || '')
      .replace(/\D/g, '')
      .slice(0, 6)
  }

  function resetPendingLogin() {
    clearMfaGateState()
    mfaLoading.setup = false
    mfaLoading.confirm = false
    userStore.setLoginStatus(false)
    userStore.setToken('', '')
    formData.password = ''
    resetDragVerify()
  }

  function resetDragVerify() {
    isPassing.value = false
    isClickPass.value = false
    dragVerify.value?.reset?.()
  }

  function showLoginSuccessNotice() {
    window.setTimeout(() => {
      ElNotification({
        title: t('login.success.title'),
        type: 'success',
        duration: 2500,
        zIndex: 10000,
        message: `${t('login.success.message')}, ${systemName.value}!`
      })
    }, 1000)
  }

  async function tryRestoreHiddenAdminEntry(showError: boolean = false) {
    const adminPathState = await ensureCurrentAdminPath()

    if (
      adminPathState.status !== 'invalid' ||
      !adminPathState.cachedPath ||
      adminPathState.cachedPath === adminPathState.currentPath
    ) {
      if (showError) {
        ElMessage.error('当前访问地址不是有效的管理入口，请检查隐藏路径')
      }
      return
    }

    const targetUrl = buildAdminHashUrl(adminPathState.cachedPath, route.fullPath || '/login')
    const message = `当前入口 /${adminPathState.currentPath} 无效，正在切换到 /${adminPathState.cachedPath}`

    if (showError) {
      ElMessage.error(message)
    } else {
      ElMessage.warning(message)
    }

    window.setTimeout(() => {
      window.location.replace(targetUrl)
    }, 500)
  }
</script>

<style scoped>
  @import './style.css';
</style>

<style lang="scss" scoped>
  .mfa-header {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .mfa-title {
    color: var(--el-text-color-primary);
    font-size: 20px;
    font-weight: 700;
    line-height: 1.2;
  }

  .mfa-subtitle {
    color: var(--el-text-color-secondary);
    font-size: 13px;
    line-height: 1.6;
  }

  .mfa-alert {
    margin-bottom: 16px;
  }

  .mfa-form {
    margin-top: 8px;
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
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 14px;
    background: var(--el-fill-color-light);
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
    color: var(--el-text-color-secondary);
    font-size: 12px;
    letter-spacing: 0.04em;
  }

  .mfa-secret-value {
    color: var(--el-text-color-primary);
    font-family: 'Courier New', Consolas, monospace;
    font-size: 16px;
    font-weight: 700;
    line-height: 1.5;
    word-break: break-all;
  }

  .mfa-secret-tip,
  .mfa-helper {
    color: var(--el-text-color-secondary);
    font-size: 13px;
    line-height: 1.6;
  }

  .mfa-helper {
    margin-top: -2px;
  }

  .mfa-footer {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
  }

  :deep(.el-select__wrapper) {
    height: 40px !important;
  }

  :deep(.admin-mfa-dialog .el-dialog__body) {
    padding-top: 8px;
  }

  @media (max-width: 640px) {
    .mfa-setup-panel {
      grid-template-columns: 1fr;
    }

    .mfa-qr-card {
      min-height: 180px;
    }
  }
</style>
