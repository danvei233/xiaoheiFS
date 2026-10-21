<template>
  <div class="flex w-full h-screen">
    <LoginLeftView />

    <div class="relative flex-1">
      <AuthTopBar />

      <div class="auth-right-wrap">
        <div class="form">
          <h3 class="title">Reset Password</h3>
          <p class="sub-title">Enter the reset token and choose a new administrator password.</p>

          <ElForm
            ref="formRef"
            class="mt-7.5"
            :model="formData"
            :rules="rules"
            @keyup.enter="handleSubmit"
          >
            <ElFormItem prop="token">
              <ElInput
                class="custom-height"
                v-model.trim="formData.token"
                :maxlength="INPUT_LIMITS.PASSWORD"
                placeholder="Enter reset token"
              />
            </ElFormItem>

            <ElFormItem prop="new_password">
              <ElInput
                class="custom-height"
                v-model.trim="formData.new_password"
                type="password"
                autocomplete="off"
                show-password
                :maxlength="INPUT_LIMITS.PASSWORD"
                placeholder="Enter new password"
              />
            </ElFormItem>

            <ElFormItem prop="confirm_password">
              <ElInput
                class="custom-height"
                v-model.trim="formData.confirm_password"
                type="password"
                autocomplete="off"
                show-password
                :maxlength="INPUT_LIMITS.PASSWORD"
                placeholder="Confirm new password"
              />
            </ElFormItem>

            <div class="help-copy">
              The reset token usually comes from the password reset email sent to the administrator
              address.
            </div>

            <div style="margin-top: 18px">
              <ElButton
                class="w-full custom-height"
                type="primary"
                @click="handleSubmit"
                :loading="loading"
                v-ripple
              >
                Reset Password
              </ElButton>
            </div>

            <div style="margin-top: 15px">
              <ElButton class="w-full custom-height" plain @click="toLogin">Back To Login</ElButton>
            </div>
          </ElForm>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { fetchAdminResetPassword } from '@/api/admin'
  import { INPUT_LIMITS } from '@/utils/constants'
  import { HttpError } from '@/utils/http/error'
  import { ElNotification, type FormInstance, type FormRules } from 'element-plus'

  defineOptions({ name: 'ResetPassword' })

  interface ResetPasswordForm {
    token: string
    new_password: string
    confirm_password: string
  }

  const router = useRouter()
  const route = useRoute()
  const formRef = ref<FormInstance>()
  const loading = ref(false)

  const formData = reactive<ResetPasswordForm>({
    token: '',
    new_password: '',
    confirm_password: ''
  })

  onMounted(() => {
    formData.token = String(route.query.token || '')
  })

  const rules = computed<FormRules<ResetPasswordForm>>(() => ({
    token: [{ required: true, message: 'Please enter the reset token', trigger: 'blur' }],
    new_password: [
      { required: true, message: 'Please enter a new password', trigger: 'blur' },
      {
        min: 6,
        max: INPUT_LIMITS.PASSWORD,
        message: `Password must be between 6 and ${INPUT_LIMITS.PASSWORD} characters`,
        trigger: 'blur'
      }
    ],
    confirm_password: [
      { required: true, message: 'Please confirm the new password', trigger: 'blur' },
      {
        validator: (_rule, value: string, callback) => {
          if (String(value || '') !== String(formData.new_password || '')) {
            callback(new Error('The two passwords do not match'))
            return
          }
          callback()
        },
        trigger: 'blur'
      }
    ]
  }))

  async function handleSubmit() {
    if (!formRef.value) {
      return
    }

    try {
      const valid = await formRef.value.validate()
      if (!valid) {
        return
      }

      loading.value = true

      await fetchAdminResetPassword({
        token: String(formData.token || '').trim(),
        new_password: formData.new_password
      })

      ElNotification({
        title: 'Password Reset Complete',
        type: 'success',
        duration: 3000,
        message: 'The administrator password has been updated. Please sign in again.'
      })

      router.push({ name: 'Login' })
    } catch (error) {
      if (!(error instanceof HttpError)) {
        console.error('[ResetPassword] Unexpected error:', error)
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

  .help-copy {
    color: var(--el-text-color-secondary);
    font-size: 13px;
    line-height: 1.6;
  }
</style>
