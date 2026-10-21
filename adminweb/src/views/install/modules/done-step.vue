<template>
  <div class="done-shell">
    <ElResult
      status="success"
      icon="success"
      title="安装完成"
      sub-title="基础配置已经写入，现在可以进入新的管理员后台。"
    >
      <template #extra>
        <div class="actions">
          <ElButton @click="goHome">返回首页</ElButton>
          <ElButton type="primary" @click="goAdmin">进入后台</ElButton>
        </div>
      </template>
    </ElResult>

    <div class="summary-grid">
      <ElCard shadow="never" class="summary-card">
        <template #header>
          <div class="card-title">后台入口</div>
        </template>
        <code>{{ adminLoginUrl }}</code>
      </ElCard>

      <ElCard v-if="restartRequired" shadow="never" class="summary-card warning-card">
        <template #header>
          <div class="card-title">后续操作</div>
        </template>
        <p>如果当前服务还没有读取到最新配置，请按需重启后端进程。</p>
        <p v-if="configFile"
          >配置文件：<code>{{ configFile }}</code></p
        >
      </ElCard>

      <ElCard shadow="never" class="summary-card">
        <template #header>
          <div class="card-title">重新安装</div>
        </template>
        <p>如需重新初始化，请删除 `install.lock` 后重新访问安装向导。</p>
      </ElCard>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { RoutesAlias } from '@/router/routesAlias'
  import { buildAdminHashUrl, getCachedAdminPath, normalizeAdminPath } from '@/utils/adminPath'

  defineOptions({ name: 'InstallDoneStep' })

  const props = withDefaults(
    defineProps<{
      adminPath?: string
      restartRequired?: boolean
      configFile?: string
    }>(),
    {
      adminPath: 'admin',
      restartRequired: false,
      configFile: ''
    }
  )

  const resolvedAdminPath = computed(() => {
    return normalizeAdminPath(props.adminPath) || getCachedAdminPath()
  })

  const adminLoginUrl = computed(() =>
    buildAdminHashUrl(resolvedAdminPath.value, RoutesAlias.Login)
  )

  function goHome() {
    window.location.replace('/')
  }

  function goAdmin() {
    window.location.replace(adminLoginUrl.value)
  }
</script>

<style scoped lang="scss">
  .done-shell {
    display: flex;
    flex-direction: column;
    gap: 20px;
    padding-top: 28px;
  }

  .actions {
    display: flex;
    gap: 12px;
    justify-content: center;
  }

  .summary-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
    gap: 16px;
  }

  .summary-card {
    border-radius: 20px;
  }

  .warning-card {
    border-color: rgb(230 162 60 / 35%);
  }

  .card-title {
    font-weight: 700;
  }

  .summary-card code {
    color: var(--el-color-primary);
    word-break: break-all;
    line-height: 1.6;
  }

  .summary-card p {
    margin: 0;
    color: var(--el-text-color-secondary);
    line-height: 1.7;
  }

  @media (max-width: 768px) {
    .actions {
      flex-direction: column;
    }
  }
</style>
