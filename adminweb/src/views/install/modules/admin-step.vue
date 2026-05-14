<template>
  <div class="step-shell">
    <div class="step-head">
      <div>
        <h2>管理员与隐藏路径</h2>
        <p>创建第一个超级管理员账号，并设置后台访问路径。</p>
      </div>
      <ElTag type="info" round>第 3 步 / 共 4 步</ElTag>
    </div>

    <ElForm ref="formRef" :model="form" :rules="rules" label-position="top">
      <ElCard shadow="never" class="step-card">
        <template #header>
          <div class="card-title">管理员账号</div>
        </template>

        <div class="form-grid">
          <ElFormItem label="用户名" prop="adminUser">
            <ElInput v-model="form.adminUser" placeholder="例如：admin" />
          </ElFormItem>
          <ElFormItem label="后台路径" prop="adminPath">
            <ElInput v-model="form.adminPath" placeholder="仅支持字母和数字">
              <template #append>
                <ElButton text type="primary" :loading="generating" @click="generateAdminPath()">
                  随机生成
                </ElButton>
              </template>
            </ElInput>
            <div class="field-hint">安装完成后后台地址会变成 `/路径/#/login`。</div>
          </ElFormItem>
          <ElFormItem label="密码" prop="adminPass">
            <ElInput
              v-model="form.adminPass"
              type="password"
              show-password
              placeholder="至少 6 位"
            />
          </ElFormItem>
          <ElFormItem label="确认密码" prop="adminPass2">
            <ElInput
              v-model="form.adminPass2"
              type="password"
              show-password
              placeholder="再次输入密码"
            />
          </ElFormItem>
        </div>
      </ElCard>
    </ElForm>

    <ElAlert
      v-if="!wizard.dbChecked"
      type="warning"
      :closable="false"
      title="请先回到上一步完成数据库连接测试。"
      show-icon
    />

    <div class="bottom-bar">
      <ElAlert
        type="info"
        :closable="false"
        title="管理路径会作为后台入口的一部分，建议使用随机字符串并妥善保存。"
        show-icon
      />

      <div class="actions">
        <ElButton @click="emit('back')">上一步</ElButton>
        <ElButton
          type="primary"
          :loading="submitting"
          :disabled="!wizard.dbChecked"
          @click="handleSubmit"
        >
          开始安装
        </ElButton>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import type { FormInstance, FormRules } from 'element-plus'
  import { ElMessage } from 'element-plus'
  import { runInstall, validateInstallAdminPath } from '@/api/install'
  import { useInstallStore } from '@/store/modules/install'
  import { useInstallWizardStore } from '@/store/modules/install-wizard'
  import { cacheAdminPath, normalizeAdminPath } from '@/utils/adminPath'

  defineOptions({ name: 'InstallAdminStep' })

  interface InstallDonePayload {
    adminPath: string
    restartRequired: boolean
    configFile: string
  }

  const emit = defineEmits<{
    next: [payload: InstallDonePayload]
    back: []
  }>()

  const installStore = useInstallStore()
  const wizard = useInstallWizardStore()
  const formRef = ref<FormInstance>()
  const submitting = ref(false)
  const generating = ref(false)

  const form = reactive({
    adminUser: wizard.adminUser || 'admin',
    adminPass: '',
    adminPass2: '',
    adminPath: wizard.adminPath || ''
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

  const rules: FormRules = {
    adminUser: [{ required: true, message: '请输入管理员用户名', trigger: 'blur' }],
    adminPass: [
      { required: true, message: '请输入管理员密码', trigger: 'blur' },
      { min: 6, message: '密码至少 6 位', trigger: 'blur' }
    ],
    adminPass2: [
      { required: true, message: '请再次输入密码', trigger: 'blur' },
      {
        validator: (_rule, value, callback) => {
          if (String(value || '') !== String(form.adminPass || '')) {
            callback(new Error('两次输入的密码不一致'))
            return
          }

          callback()
        },
        trigger: ['blur', 'change']
      }
    ],
    adminPath: [
      {
        validator: (_rule, value, callback) => {
          const path = normalizeAdminPath(value)
          if (!path) {
            callback(new Error('请输入后台路径'))
            return
          }

          if (!/^[A-Za-z0-9]+$/.test(path)) {
            callback(new Error('后台路径仅支持字母和数字'))
            return
          }

          if (reservedAdminPaths.has(path.toLowerCase())) {
            callback(new Error('该路径属于保留路径，请更换'))
            return
          }

          validateInstallAdminPath(path)
            .then(() => callback())
            .catch((error) => callback(new Error(resolveErrorMessage(error, '后台路径不可用'))))
        },
        trigger: 'blur'
      }
    ]
  }

  watch(
    () => form.adminUser,
    (value) => {
      wizard.adminUser = value
      wizard.persist()
    }
  )

  watch(
    () => form.adminPath,
    (value) => {
      wizard.adminPath = normalizeAdminPath(value)
      wizard.persist()
    }
  )

  onMounted(async () => {
    if (!normalizeAdminPath(form.adminPath)) {
      await generateAdminPath(true)
    }
  })

  async function generateAdminPath(silent: boolean = false) {
    generating.value = true

    try {
      for (let index = 0; index < 20; index += 1) {
        const bytes = new Uint8Array(12)

        if (typeof crypto !== 'undefined' && typeof crypto.getRandomValues === 'function') {
          crypto.getRandomValues(bytes)
        } else {
          for (let offset = 0; offset < bytes.length; offset += 1) {
            bytes[offset] = Math.floor(Math.random() * 256)
          }
        }

        const candidate = Array.from(
          bytes,
          (value) => randomCharset[value % randomCharset.length]
        ).join('')

        if (reservedAdminPaths.has(candidate.toLowerCase())) {
          continue
        }

        form.adminPath = candidate
        if (!silent) {
          ElMessage.success('已生成随机后台路径')
        }
        return
      }

      ElMessage.error('随机路径生成失败，请重试')
    } finally {
      generating.value = false
    }
  }

  async function handleSubmit() {
    if (!wizard.dbChecked) {
      ElMessage.warning('请先完成数据库连接测试')
      return
    }

    if (!formRef.value) {
      return
    }

    const valid = await formRef.value.validate().catch(() => false)
    if (!valid) {
      return
    }

    const adminPath = normalizeAdminPath(form.adminPath)
    wizard.adminUser = form.adminUser.trim()
    wizard.adminPass = form.adminPass
    wizard.adminPath = adminPath
    wizard.persist()

    submitting.value = true

    try {
      const dbPayload =
        wizard.dbType === 'sqlite'
          ? { type: 'sqlite' as const, path: wizard.sqlitePath }
          : { type: 'mysql' as const, dsn: wizard.mysqlDSN }

      const result = await runInstall({
        db: dbPayload,
        site: {
          name: wizard.siteName.trim(),
          url: wizard.siteUrl.trim(),
          admin_path: adminPath
        },
        admin: {
          username: wizard.adminUser.trim(),
          password: form.adminPass
        }
      })

      cacheAdminPath(adminPath)
      installStore.setInstalled(true)
      ElMessage.success('安装完成')

      emit('next', {
        adminPath,
        restartRequired: Boolean(result.restart_required),
        configFile: result.config_file || ''
      })
    } catch (error) {
      ElMessage.error(resolveErrorMessage(error, '安装失败'))
    } finally {
      submitting.value = false
    }
  }

  function resolveErrorMessage(error: unknown, fallback: string) {
    if (error instanceof Error && error.message) {
      return error.message
    }

    return fallback
  }
</script>

<style scoped lang="scss">
  .step-shell {
    display: flex;
    flex-direction: column;
    gap: 20px;
  }

  .step-head {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 16px;
  }

  .step-head h2 {
    margin: 0 0 8px;
    color: var(--el-text-color-primary);
    font-size: 28px;
    font-weight: 800;
  }

  .step-head p,
  .field-hint {
    margin: 0;
    color: var(--el-text-color-secondary);
    line-height: 1.6;
  }

  .step-card {
    border-radius: 22px;
  }

  .card-title {
    font-weight: 700;
  }

  .form-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 0 16px;
  }

  .bottom-bar {
    display: flex;
    align-items: flex-end;
    justify-content: space-between;
    gap: 20px;
  }

  .bottom-bar :deep(.el-alert) {
    flex: 1;
  }

  .actions {
    display: flex;
    gap: 12px;
    flex-shrink: 0;
  }

  :deep(.el-input-group__append .el-button) {
    margin: 0;
  }

  @media (max-width: 768px) {
    .step-head,
    .bottom-bar {
      flex-direction: column;
      align-items: stretch;
    }

    .form-grid {
      grid-template-columns: 1fr;
    }

    .actions {
      width: 100%;
    }

    .actions :deep(.el-button) {
      flex: 1;
    }
  }
</style>
