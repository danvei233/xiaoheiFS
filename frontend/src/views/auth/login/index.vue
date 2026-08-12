<template>
  <div class="auth-page">
    <LoginLeftView />

    <main class="auth-main">
      <AuthTopBar />

      <div class="auth-right-wrap">
        <div class="auth-form">
          <h1 class="auth-title">用户登录</h1>
          <p class="auth-subtitle">登录以访问控制台</p>

          <ElForm
            ref="formRef"
            :model="form"
            :rules="rules"
            label-position="top"
            @keyup.enter="handleSubmit"
          >
            <ElSegmented v-model="loginMode" :options="loginModeOptions" class="login-modes" />

            <ElFormItem v-if="loginMode === 'account'" label="账号" prop="username">
              <ElInput
                v-model="form.username"
                placeholder="请输入用户名/邮箱"
                size="large"
                :maxlength="INPUT_LIMITS.EMAIL"
                clearable
              >
                <template #prefix><ArtSvgIcon icon="ri:user-3-line" /></template>
              </ElInput>
            </ElFormItem>

            <ElFormItem v-else label="手机号" prop="phone">
              <ElInput
                v-model="form.phone"
                placeholder="请输入手机号"
                size="large"
                :maxlength="INPUT_LIMITS.PHONE"
                clearable
              >
                <template #prefix><ArtSvgIcon icon="ri:smartphone-line" /></template>
              </ElInput>
            </ElFormItem>

            <ElFormItem label="密码" prop="password">
              <ElInput
                v-model="form.password"
                type="password"
                placeholder="请输入密码"
                size="large"
                :maxlength="INPUT_LIMITS.PASSWORD"
                autocomplete="current-password"
                show-password
              >
                <template #prefix><ArtSvgIcon icon="ri:lock-password-line" /></template>
              </ElInput>
            </ElFormItem>

            <ElFormItem
              v-if="settings.login_captcha_enabled"
              :label="settings.captcha_provider === 'geetest' ? '行为验证码' : '图形验证码'"
              prop="captcha_code"
            >
              <div v-if="settings.captcha_provider !== 'geetest'" class="captcha-row">
                <ElInput v-model="form.captcha_code" placeholder="验证码" size="large" />
                <button class="captcha-image" type="button" @click="refreshCaptcha">
                  <img v-if="captchaImage" :src="captchaImage" alt="captcha" />
                  <span v-else>点击刷新</span>
                </button>
              </div>

              <div v-else class="geetest-row">
                <ElButton
                  :disabled="!geetest.ready"
                  :loading="geetest.loading"
                  @click="verifyGeeTest"
                >
                  {{ geetest.passed ? '已通过验证，点击重试' : '点击完成极验验证' }}
                </ElButton>
                <ElTag :type="geetest.passed ? 'success' : 'info'">
                  {{ geetest.passed ? '验证通过' : '未验证' }}
                </ElTag>
              </div>
            </ElFormItem>

            <div class="auth-actions">
              <RouterLink to="/forgot-password">忘记密码？</RouterLink>
              <RouterLink to="/register">立即注册</RouterLink>
            </div>

            <ElButton
              class="submit-button"
              type="primary"
              size="large"
              :loading="auth.loading"
              @click="handleSubmit"
              v-ripple
            >
              登录
            </ElButton>
          </ElForm>
        </div>
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
  import type { FormInstance, FormRules } from 'element-plus'
  import { getAuthSettings, getCaptcha } from '@/services/user'
  import { useAuthStore } from '@/stores/auth'
  import { INPUT_LIMITS } from '@/constants/inputLimits'
  import { resolveUserRedirect } from '@/utils/auth'

  defineOptions({ name: 'Login' })

  type LoginMode = 'account' | 'phone'

  interface GeeTestValidation {
    lot_number?: string
    captcha_output?: string
    pass_token?: string
    gen_time?: string
  }

  interface GeeTestWidget {
    showCaptcha: () => void
    getValidate?: () => GeeTestValidation | null
    onSuccess: (callback: () => void) => void
    onError: (callback: () => void) => void
  }

  type InitGeeTest = (
    options: { captchaId: string; product: string; language: string },
    callback: (widget: GeeTestWidget) => void
  ) => void

  const auth = useAuthStore()
  const router = useRouter()
  const route = useRoute()
  const formRef = ref<FormInstance>()
  const loginMode = ref<LoginMode>('account')
  const captchaId = ref('')
  const captchaImage = ref('')

  const loginModeOptions = [
    { label: '账号登录', value: 'account' },
    { label: '手机号登录', value: 'phone' }
  ]

  const form = reactive({
    username: '',
    phone: '',
    password: '',
    captcha_code: ''
  })

  const settings = reactive({
    login_captcha_enabled: false,
    captcha_provider: 'image' as 'image' | 'geetest'
  })

  const geetest = reactive({
    widget: null as GeeTestWidget | null,
    ready: false,
    loading: false,
    passed: false,
    lot_number: '',
    captcha_output: '',
    pass_token: '',
    gen_time: ''
  })

  const validatePhoneLogin = () => {
    const value = String(form.phone || '').trim()
    if (!value) return Promise.resolve()
    if (!/^[0-9+\-\s]{6,20}$/.test(value)) return Promise.reject('请输入有效手机号')
    return Promise.resolve()
  }

  const rules = computed<FormRules>(() => ({
    username:
      loginMode.value === 'account'
        ? [{ required: true, message: '请输入账号', trigger: 'blur' }]
        : [],
    phone:
      loginMode.value === 'phone'
        ? [
            { required: true, message: '请输入手机号', trigger: 'blur' },
            { validator: validatePhoneLogin, trigger: 'blur' }
          ]
        : [],
    password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
    captcha_code:
      settings.login_captcha_enabled && settings.captcha_provider !== 'geetest'
        ? [{ required: true, message: '请输入验证码', trigger: 'blur' }]
        : []
  }))

  const resetGeeTestResult = () => {
    geetest.passed = false
    geetest.lot_number = ''
    geetest.captcha_output = ''
    geetest.pass_token = ''
    geetest.gen_time = ''
  }

  const getInitGeeTest = () => (window as Window & { initGeetest4?: InitGeeTest }).initGeetest4

  const ensureGeeTestScript = async () => {
    if (getInitGeeTest()) return
    await new Promise<void>((resolve, reject) => {
      const existed = document.querySelector<HTMLScriptElement>("script[data-geetest='gt4']")
      if (existed) {
        existed.addEventListener('load', () => resolve(), { once: true })
        existed.addEventListener('error', reject, { once: true })
        return
      }
      const script = document.createElement('script')
      script.src = 'https://static.geetest.com/v4/gt4.js'
      script.async = true
      script.defer = true
      script.dataset.geetest = 'gt4'
      script.onload = () => resolve()
      script.onerror = reject
      document.head.appendChild(script)
    })
  }

  const initGeeTest = async (captchaID: string) => {
    resetGeeTestResult()
    geetest.ready = false
    geetest.widget = null
    if (!captchaID) return
    try {
      await ensureGeeTestScript()
      await new Promise<void>((resolve, reject) => {
        const init = getInitGeeTest()
        if (!init) {
          reject(new Error('GeeTest is unavailable'))
          return
        }
        init({ captchaId: captchaID, product: 'bind', language: 'zho' }, (widget) => {
          geetest.widget = widget
          geetest.ready = true
          widget.onSuccess(() => {
            const result = widget.getValidate?.()
            geetest.lot_number = String(result?.lot_number || '')
            geetest.captcha_output = String(result?.captcha_output || '')
            geetest.pass_token = String(result?.pass_token || '')
            geetest.gen_time = String(result?.gen_time || '')
            geetest.passed = Boolean(
              geetest.lot_number && geetest.captcha_output && geetest.pass_token && geetest.gen_time
            )
          })
          widget.onError(() => {
            resetGeeTestResult()
            ElMessage.error('极验初始化失败')
          })
          resolve()
        })
      })
    } catch {
      ElMessage.error('极验脚本加载失败')
    }
  }

  const verifyGeeTest = () => {
    if (!geetest.widget || !geetest.ready) {
      ElMessage.warning('极验尚未就绪，请稍后')
      return
    }
    geetest.loading = true
    try {
      resetGeeTestResult()
      geetest.widget.showCaptcha()
    } finally {
      geetest.loading = false
    }
  }

  const refreshCaptcha = async () => {
    if (!settings.login_captcha_enabled) return
    try {
      const response = await getCaptcha()
      const provider = String(
        response.data?.captcha_provider || settings.captcha_provider || 'image'
      ).toLowerCase()
      settings.captcha_provider = provider === 'geetest' ? 'geetest' : 'image'
      captchaId.value = String(response.data?.captcha_id || '')
      form.captcha_code = ''
      if (settings.captcha_provider === 'geetest') {
        captchaImage.value = ''
        await initGeeTest(captchaId.value)
        return
      }
      const base64 = String(response.data?.image_base64 || '')
      captchaImage.value = base64 ? `data:image/png;base64,${base64}` : ''
      resetGeeTestResult()
    } catch (error) {
      console.error('Failed to refresh captcha:', error)
    }
  }

  const loadSettings = async () => {
    try {
      const response = await getAuthSettings()
      Object.assign(settings, response.data || {})
    } catch (error) {
      console.error('Failed to fetch auth settings:', error)
    } finally {
      await refreshCaptcha()
    }
  }

  const handleSubmit = async () => {
    if (!formRef.value) return
    const valid = await formRef.value.validate().catch(() => false)
    if (!valid) return

    const loginAccount =
      loginMode.value === 'phone'
        ? String(form.phone || '').trim()
        : String(form.username || '').trim()
    if (loginMode.value === 'phone' && loginAccount.length > INPUT_LIMITS.PHONE) {
      ElMessage.error(`手机号长度不能超过 ${INPUT_LIMITS.PHONE} 个字符`)
      return
    }
    if (loginMode.value === 'account' && loginAccount.length > INPUT_LIMITS.EMAIL) {
      ElMessage.error(`账号长度不能超过 ${INPUT_LIMITS.EMAIL} 个字符`)
      return
    }
    if (String(form.password || '').length > INPUT_LIMITS.PASSWORD) {
      ElMessage.error(`密码长度不能超过 ${INPUT_LIMITS.PASSWORD} 个字符`)
      return
    }
    if (
      settings.login_captcha_enabled &&
      settings.captcha_provider === 'geetest' &&
      !geetest.passed
    ) {
      ElMessage.warning('请先完成极验验证')
      return
    }

    try {
      const token = await auth.login({
        username: loginAccount,
        password: form.password,
        captcha_id: captchaId.value,
        captcha_code: form.captcha_code,
        lot_number: geetest.lot_number,
        captcha_output: geetest.captcha_output,
        pass_token: geetest.pass_token,
        gen_time: geetest.gen_time
      })

      if (!token) {
        ElMessage.error('登录失败')
        if (settings.login_captcha_enabled) await refreshCaptcha()
        return
      }

      // A 401 clears the token in the HTTP interceptor; transient profile failures keep it for retry.
      await auth.fetchMe()
      ElMessage.success('登录成功')
      // redirect query 是 URL 可控数据，与 admin 域一致走白名单校验，防止将来
      // 改成 location.href 时变成开放重定向（web_auth_surface F4）。
      await router.replace(resolveUserRedirect(route.query.redirect))
    } catch (error) {
      const responseError = error as {
        response?: { data?: { error?: string; message?: string } }
        message?: string
      }
      const message =
        responseError.response?.data?.error ||
        responseError.response?.data?.message ||
        responseError.message ||
        '登录失败'
      ElMessage.error(message)
      if (settings.login_captcha_enabled) await refreshCaptcha()
    }
  }

  watch(loginMode, () => {
    formRef.value?.clearValidate()
  })

  onMounted(loadSettings)
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

  .auth-title {
    margin: 0;
    font-size: 36px;
    font-weight: 600;
    color: var(--art-gray-900);
    letter-spacing: 0;
  }

  .auth-subtitle {
    margin: 10px 0 24px;
    font-size: 14px;
    color: var(--art-gray-600);
  }

  .login-modes {
    width: 100%;
    margin-bottom: 18px;
  }

  .captcha-row,
  .geetest-row {
    display: flex;
    gap: 10px;
    align-items: center;
    width: 100%;
  }

  .captcha-image {
    display: flex;
    flex: 0 0 120px;
    align-items: center;
    justify-content: center;
    height: 40px;
    padding: 0;
    overflow: hidden;
    font: inherit;
    color: var(--art-gray-600);
    cursor: pointer;
    background: var(--default-bg-color);
    border: 1px dashed var(--default-border-dashed);
    border-radius: calc(var(--custom-radius) / 2 + 2px);
  }

  .captcha-image img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .auth-actions {
    display: flex;
    justify-content: space-between;
    margin: 8px 0 22px;
    font-size: 14px;
  }

  .auth-actions a {
    color: var(--theme-color);
  }

  .submit-button {
    width: 100%;
    height: 40px;
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
  }
</style>
