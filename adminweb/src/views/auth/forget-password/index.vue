<template>
  <div class="flex w-full h-screen">
    <LoginLeftView />

    <div class="relative flex-1">
      <AuthTopBar />

      <div class="auth-right-wrap">
        <div class="form">
          <h3 class="title">{{ $t('forgetPassword.title') }}</h3>
          <p class="sub-title">{{ $t('forgetPassword.subTitle') }}</p>
          <div class="mt-5">
            <span class="input-label" v-if="showInputLabel">账号</span>
            <ElInput
              class="custom-height"
              :placeholder="$t('forgetPassword.placeholder')"
              v-model.trim="username"
            />
          </div>

          <div style="margin-top: 15px">
            <ElButton
              class="w-full custom-height"
              type="primary"
              @click="handleSubmit"
              :loading="loading"
              v-ripple
            >
              {{ $t('forgetPassword.submitBtnText') }}
            </ElButton>
          </div>

          <div style="margin-top: 15px">
            <ElButton class="w-full custom-height" plain @click="toLogin">
              {{ $t('forgetPassword.backBtnText') }}
            </ElButton>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { fetchAdminForgotPassword } from '@/api/admin'
  import { INPUT_LIMITS } from '@/utils/constants'
  import { HttpError } from '@/utils/http/error'
  import { ElMessage, ElNotification } from 'element-plus'
  import { useI18n } from 'vue-i18n'

  defineOptions({ name: 'ForgetPassword' })

  const router = useRouter()
  const { t } = useI18n()
  const loading = ref(false)
  const showInputLabel = false

  const formData = reactive({
    email: ''
  })

  const username = computed({
    get: () => formData.email,
    set: (value: string) => {
      formData.email = value
    }
  })

  async function handleSubmit() {
    const email = String(formData.email || '').trim()

    if (!email) {
      ElMessage.error(t('forgetPassword.placeholder'))
      return
    }

    if (email.length > INPUT_LIMITS.EMAIL) {
      ElMessage.error(`邮箱长度不能超过 ${INPUT_LIMITS.EMAIL} 个字符`)
      return
    }

    const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
    if (!emailPattern.test(email)) {
      ElMessage.error('请输入正确的邮箱地址')
      return
    }

    try {
      loading.value = true

      await fetchAdminForgotPassword({
        email
      })

      ElNotification({
        title: '发送成功',
        type: 'success',
        duration: 3000,
        message: '重置密码说明已发送到管理员邮箱。'
      })

      router.push({ name: 'Login' })
    } catch (error) {
      if (!(error instanceof HttpError)) {
        console.error('[ForgetPassword] Unexpected error:', error)
      }
    } finally {
      loading.value = false
    }
  }

  function toLogin() {
    router.push({ name: 'Login' })
  }
</script>

<style scoped>
  @import '../login/style.css';
</style>
