<template>
  <section class="install-step">
    <header class="step-header">
      <div>
        <h2>创建管理员</h2>
        <p>设置超级管理员账号，用于登录管理后台</p>
      </div>
      <ElTag effect="plain">3 / 4</ElTag>
    </header>

    <div class="form-section">
      <ArtForm
        ref="formRef"
        v-model="form"
        :items="formItems"
        :rules="rules"
        :span="24"
        label-position="top"
        :show-reset="false"
        :show-submit="false"
      >
        <template #adminPath>
          <ElInput
            v-model="form.adminPath"
            placeholder="Please input admin path"
            :maxlength="INPUT_LIMITS.URL"
            clearable
          >
            <template #append>
              <ElButton :loading="generating" @click="generateAdminPath()">
                <ElIcon><Refresh /></ElIcon>
                随机生成
              </ElButton>
            </template>
          </ElInput>
        </template>
      </ArtForm>
      <p class="path-hint">仅允许字母和数字，不可与 login/admin/api 等保留路径重复</p>
    </div>

    <ElAlert
      v-if="!wiz.dbChecked"
      type="warning"
      title="请先完成数据库连接测试"
      :closable="false"
      show-icon
      class="step-alert"
    />
    <ElAlert
      type="info"
      title="安全提示"
      description="管理员账号拥有系统最高权限，请妥善保管密码和管理端路径"
      :closable="false"
      show-icon
      class="step-alert"
    />

    <footer class="step-actions">
      <ElButton @click="emit('back')">
        <ElIcon><ArrowLeft /></ElIcon>
        上一步
      </ElButton>
      <ElButton
        type="primary"
        :loading="submitting"
        :disabled="!wiz.dbChecked || !String(form.adminPath || '').trim()"
        @click="handleSubmit"
        v-ripple
      >
        开始安装
      </ElButton>
    </footer>
  </section>
</template>

<script setup lang="ts">
  import { ArrowLeft, Refresh } from '@element-plus/icons-vue'
  import type { FormItem } from '@/components/core/forms/art-form/index.vue'
  import type { FormRules } from 'element-plus'
  import { runInstall } from '@/services/user'
  import { setAdminPathHint } from '@/services/adminPath'
  import { useInstallStore } from '@/stores/install'
  import { useInstallWizardStore } from '@/stores/installWizard'
  import { INPUT_LIMITS } from '@/constants/inputLimits'

  defineOptions({ name: 'InstallAdminStep' })

  const emit = defineEmits<{
    next: [adminPath: string, restart: boolean, configFile: string]
    back: []
  }>()

  const install = useInstallStore()
  const wiz = useInstallWizardStore()
  const formRef = ref<{ validate: () => Promise<boolean> }>()
  const submitting = ref(false)
  const generating = ref(false)
  const form = reactive<Record<string, any>>({
    adminUser: wiz.adminUser,
    adminPass: '',
    adminPass2: '',
    adminPath: wiz.adminPath || ''
  })

  const randomCharset = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789'
  const reservedAdminPaths = new Set([
    'login',
    'admin',
    'api',
    'install',
    'console',
    'register',
    'assets',
    'uploads',
    'static',
    'public',
    'user',
    'users',
    'auth',
    'logout',
    'profile',
    'settings',
    'dashboard',
    'home',
    'index',
    'help',
    'docs',
    'products',
    'about',
    'contact',
    'support',
    'forgot',
    'reset',
    'verify',
    'callback',
    'oauth',
    'download',
    'downloads',
    'file',
    'files',
    'image',
    'images',
    'video',
    'videos',
    'media',
    'css',
    'js',
    'javascript',
    'favicon',
    'robots',
    'sitemap',
    'manifest',
    'service',
    'worker',
    'sw',
    'health',
    'ping',
    'status',
    'metrics',
    'debug',
    'test',
    'demo',
    'example',
    'sample',
    'tmp',
    'temp',
    'cache',
    'backup',
    'config',
    'system',
    'root',
    'administrator',
    'webmaster',
    'moderator',
    'superuser',
    'sysadmin'
  ])

  const validateConfirm = () => {
    if (!form.adminPass2) return Promise.resolve()
    return form.adminPass2 === form.adminPass
      ? Promise.resolve()
      : Promise.reject('两次输入的密码不一致')
  }

  const validateAdminPath = () => {
    const value = String(form.adminPath || '').trim()
    if (!value) return Promise.reject('Please input admin path')
    if (!/^[a-zA-Z0-9]+$/.test(value)) return Promise.reject('仅允许字母和数字')
    if (reservedAdminPaths.has(value.toLowerCase()))
      return Promise.reject('该路径为保留路径，请更换')
    return Promise.resolve()
  }

  const formItems: FormItem[] = [
    {
      key: 'adminUser',
      label: '用户名',
      type: 'input',
      span: 24,
      props: { placeholder: '例如：admin', maxlength: INPUT_LIMITS.USERNAME, clearable: true }
    },
    {
      key: 'adminPass',
      label: '密码',
      type: 'input',
      span: 24,
      props: {
        type: 'password',
        placeholder: '至少 6 位',
        maxlength: INPUT_LIMITS.PASSWORD,
        showPassword: true
      }
    },
    {
      key: 'adminPass2',
      label: '确认密码',
      type: 'input',
      span: 24,
      props: {
        type: 'password',
        placeholder: '再次输入密码',
        maxlength: INPUT_LIMITS.PASSWORD,
        showPassword: true
      }
    },
    { key: 'adminPath', label: '自定义管理端访问路径', type: 'input', span: 24 }
  ]

  const rules: FormRules = {
    adminUser: [{ required: true, message: '请输入管理员用户名', trigger: 'blur' }],
    adminPass: [
      { required: true, message: '请输入管理员密码', trigger: 'blur' },
      { min: 6, message: '密码至少 6 位', trigger: 'blur' }
    ],
    adminPass2: [
      { required: true, message: '请再次输入密码', trigger: 'blur' },
      { validator: validateConfirm, trigger: 'blur' }
    ],
    adminPath: [{ validator: validateAdminPath, trigger: 'blur' }]
  }

  watch(
    () => [form.adminUser, form.adminPath],
    () => {
      wiz.adminUser = String(form.adminUser || '')
      wiz.adminPath = String(form.adminPath || '')
      wiz.persist()
    }
  )

  const generateAdminPath = async (options: { silent?: boolean } = {}) => {
    generating.value = true
    try {
      for (let index = 0; index < 20; index += 1) {
        const bytes = new Uint8Array(12)
        if (typeof crypto !== 'undefined' && typeof crypto.getRandomValues === 'function') {
          crypto.getRandomValues(bytes)
        } else {
          for (let byteIndex = 0; byteIndex < bytes.length; byteIndex += 1) {
            bytes[byteIndex] = Math.floor(Math.random() * 256)
          }
        }
        const candidate = Array.from(
          bytes,
          (byte) => randomCharset[byte % randomCharset.length]
        ).join('')
        if (!reservedAdminPaths.has(candidate.toLowerCase())) {
          form.adminPath = candidate
          if (!options.silent) ElMessage.success('已生成随机路径')
          return
        }
      }
      ElMessage.error('生成失败')
    } finally {
      generating.value = false
    }
  }

  const handleSubmit = async () => {
    const adminPath = String(form.adminPath || '').trim()
    if (!adminPath) {
      ElMessage.error('Please input admin path')
      return
    }
    const valid = await formRef.value?.validate().catch(() => false)
    if (!valid) return

    submitting.value = true
    try {
      wiz.adminPass = String(form.adminPass || '')
      wiz.adminPath = adminPath
      wiz.persist()
      const database =
        wiz.dbType === 'sqlite'
          ? { db: { type: 'sqlite', path: wiz.sqlitePath } }
          : { db: { type: 'mysql', dsn: wiz.mysqlDSN } }
      const response = await runInstall({
        ...database,
        site: { name: wiz.siteName, url: wiz.siteUrl, admin_path: adminPath },
        admin: { username: form.adminUser, password: form.adminPass }
      })
      await install.fetchStatus()
      setAdminPathHint(adminPath)
      // H5：安装完成立即清除内存中的 DBA/管理员密码，不再继续驻留。
      wiz.clearSecrets()
      ElMessage.success('安装完成')
      emit(
        'next',
        adminPath,
        Boolean(response.data?.restart_required),
        String(response.data?.config_file || '')
      )
    } catch (error: any) {
      ElMessage.error(error?.response?.data?.error || '安装失败')
    } finally {
      submitting.value = false
    }
  }

  onMounted(async () => {
    if (!String(form.adminPath || '').trim()) await generateAdminPath({ silent: true })
  })
</script>

<style lang="scss" scoped>
  .install-step {
    padding: 4px;
  }

  .step-header {
    display: flex;
    gap: 16px;
    align-items: flex-start;
    justify-content: space-between;
    margin-bottom: 24px;
  }

  .step-header h2 {
    margin: 0 0 6px;
    font-size: 22px;
    font-weight: 700;
    color: var(--art-gray-900);
  }

  .step-header p {
    margin: 0;
    font-size: 14px;
    color: var(--art-gray-600);
  }

  .form-section {
    padding-bottom: 12px;
    overflow: hidden;
    background: var(--default-bg-color);
    border: 1px solid var(--default-border);
    border-radius: calc(var(--custom-radius) / 2 + 2px);
  }

  .path-hint {
    padding: 0 16px;
    margin: 0;
    font-size: 12px;
    color: var(--art-gray-600);
  }

  .step-alert {
    margin-top: 14px;
  }

  .step-actions {
    display: flex;
    gap: 10px;
    justify-content: flex-end;
    padding-top: 24px;
  }

  @media (width <= 640px) {
    .step-actions {
      display: grid;
      grid-template-columns: 1fr 1fr;
    }
  }
</style>
