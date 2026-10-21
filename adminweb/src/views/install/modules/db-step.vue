<template>
  <div class="step-shell">
    <div class="step-head">
      <div>
        <h2>数据库配置</h2>
        <p>先确认安装要使用的数据库，并完成连接测试。</p>
      </div>
      <ElTag type="info" round>第 1 步 / 共 4 步</ElTag>
    </div>

    <div class="db-type-grid">
      <button
        type="button"
        class="db-type-card"
        :class="{ active: wizard.dbType === 'sqlite' }"
        @click="selectDbType('sqlite')"
      >
        <strong>SQLite</strong>
        <span>单文件部署，适合快速安装和轻量场景。</span>
      </button>
      <button
        type="button"
        class="db-type-card"
        :class="{ active: wizard.dbType === 'mysql' }"
        @click="selectDbType('mysql')"
      >
        <strong>MySQL</strong>
        <span>外部数据库服务，适合生产部署。</span>
      </button>
    </div>

    <ElForm label-position="top">
      <ElCard shadow="never" class="step-card">
        <template #header>
          <div class="card-title">{{
            wizard.dbType === 'sqlite' ? 'SQLite 配置' : 'MySQL 配置'
          }}</div>
        </template>

        <template v-if="wizard.dbType === 'sqlite'">
          <ElFormItem label="数据库文件路径">
            <ElInput
              v-model="wizard.sqlitePath"
              placeholder="./data/app.db"
              @input="wizard.touchDB()"
            />
            <div class="field-hint">建议放在 `./data/` 下，便于备份和迁移。</div>
          </ElFormItem>
        </template>

        <template v-else>
          <div class="form-grid">
            <ElFormItem label="主机">
              <ElInput
                v-model="wizard.mysql.host"
                placeholder="127.0.0.1"
                @input="wizard.touchDB()"
              />
            </ElFormItem>
            <ElFormItem label="端口">
              <ElInputNumber
                v-model="wizard.mysql.port"
                :min="1"
                :max="65535"
                controls-position="right"
                @change="wizard.touchDB()"
              />
            </ElFormItem>
            <ElFormItem label="用户名">
              <ElInput v-model="wizard.mysql.user" placeholder="root" @input="wizard.touchDB()" />
            </ElFormItem>
            <ElFormItem label="密码">
              <ElInput
                v-model="wizard.mysql.pass"
                type="password"
                show-password
                @input="wizard.touchDB()"
              />
            </ElFormItem>
            <ElFormItem label="数据库名" class="span-2">
              <ElInput
                v-model="wizard.mysql.dbName"
                placeholder="xiaohei"
                @input="wizard.touchDB()"
              />
            </ElFormItem>
            <ElFormItem label="连接参数" class="span-2">
              <ElInput
                v-model="wizard.mysql.params"
                placeholder="charset=utf8mb4&parseTime=True&loc=Local"
                @input="wizard.touchDB()"
              />
            </ElFormItem>
          </div>

          <div class="dsn-box">
            <div class="dsn-head">
              <span>DSN 预览</span>
              <ElButton text type="primary" @click="copyDSN">复制</ElButton>
            </div>
            <code>{{ wizard.mysqlDSN }}</code>
          </div>
        </template>
      </ElCard>
    </ElForm>

    <div class="bottom-bar">
      <div class="status-box">
        <ElAlert
          v-if="wizard.dbCheckError"
          type="error"
          :closable="false"
          :title="wizard.dbCheckError"
          show-icon
        />
        <ElAlert
          v-else-if="wizard.dbChecked"
          type="success"
          :closable="false"
          title="数据库连接正常，可以继续下一步。"
          show-icon
        />
      </div>

      <div class="actions">
        <ElButton :loading="checking" @click="checkConnection">测试连接</ElButton>
        <ElButton type="primary" :disabled="!wizard.dbChecked" @click="emit('next')">
          下一步
        </ElButton>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import type { InstallDBType } from '@/api/install'
  import { ElMessage } from 'element-plus'
  import { checkInstallDB } from '@/api/install'
  import { useInstallWizardStore } from '@/store/modules/install-wizard'

  defineOptions({ name: 'InstallDbStep' })

  const emit = defineEmits<{
    next: []
  }>()

  const wizard = useInstallWizardStore()
  const checking = ref(false)

  function selectDbType(type: InstallDBType) {
    wizard.dbType = type
    wizard.touchDB()
  }

  async function copyDSN() {
    try {
      await navigator.clipboard.writeText(wizard.mysqlDSN)
      ElMessage.success('DSN 已复制到剪贴板')
    } catch {
      ElMessage.error('复制失败，请手动复制')
    }
  }

  async function checkConnection() {
    checking.value = true
    wizard.persist()

    try {
      const payload =
        wizard.dbType === 'sqlite'
          ? { db: { type: 'sqlite' as const, path: wizard.sqlitePath } }
          : { db: { type: 'mysql' as const, dsn: wizard.mysqlDSN } }

      const result = await checkInstallDB(payload)
      if (result.ok) {
        wizard.markDBChecked(true)
        ElMessage.success('数据库连接测试通过')
        return
      }

      wizard.markDBChecked(false, result.error || '数据库连接失败')
    } catch (error) {
      wizard.markDBChecked(false, resolveErrorMessage(error, '数据库连接失败'))
    } finally {
      checking.value = false
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
    gap: 22px;
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

  .step-head p {
    margin: 0;
    color: var(--el-text-color-secondary);
    line-height: 1.6;
  }

  .db-type-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 14px;
  }

  .db-type-card {
    display: flex;
    flex-direction: column;
    gap: 6px;
    padding: 18px;
    border: 1px solid var(--el-border-color);
    border-radius: 18px;
    background: #fff;
    text-align: left;
    transition:
      border-color 0.2s ease,
      transform 0.2s ease,
      box-shadow 0.2s ease;
  }

  .db-type-card:hover {
    border-color: var(--el-color-primary-light-5);
    transform: translateY(-1px);
  }

  .db-type-card.active {
    border-color: var(--el-color-primary);
    box-shadow: 0 16px 35px rgb(64 158 255 / 12%);
  }

  .db-type-card strong {
    color: var(--el-text-color-primary);
    font-size: 16px;
  }

  .db-type-card span,
  .field-hint {
    color: var(--el-text-color-secondary);
    font-size: 13px;
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

  .span-2 {
    grid-column: 1 / -1;
  }

  .dsn-box {
    display: flex;
    flex-direction: column;
    gap: 10px;
    padding: 14px;
    border-radius: 16px;
    background: var(--el-fill-color-light);
  }

  .dsn-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    color: var(--el-text-color-secondary);
    font-size: 13px;
    font-weight: 600;
  }

  .dsn-box code {
    color: var(--el-text-color-primary);
    word-break: break-all;
    line-height: 1.6;
  }

  .bottom-bar {
    display: flex;
    align-items: flex-end;
    justify-content: space-between;
    gap: 20px;
  }

  .status-box {
    flex: 1;
    min-width: 0;
  }

  .actions {
    display: flex;
    gap: 12px;
    flex-shrink: 0;
  }

  :deep(.el-input-number) {
    width: 100%;
  }

  @media (max-width: 768px) {
    .step-head,
    .bottom-bar {
      flex-direction: column;
      align-items: stretch;
    }

    .db-type-grid,
    .form-grid {
      grid-template-columns: 1fr;
    }

    .span-2 {
      grid-column: auto;
    }

    .actions {
      width: 100%;
    }

    .actions :deep(.el-button) {
      flex: 1;
    }
  }
</style>
