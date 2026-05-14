<template>
  <div class="install-shell">
    <div class="install-layout">
      <section class="install-hero">
        <span class="hero-kicker">XiaoheiFS Installer</span>
        <h1>初始化管理后台</h1>
        <p>
          先完成数据库、站点信息和管理员账号配置，安装完成后会自动切换到新的 `adminweb` 后台入口。
        </p>

        <div class="hero-points">
          <div class="point">
            <strong>1</strong>
            <span>检测数据库连接</span>
          </div>
          <div class="point">
            <strong>2</strong>
            <span>写入站点基础信息</span>
          </div>
          <div class="point">
            <strong>3</strong>
            <span>创建超级管理员</span>
          </div>
        </div>
      </section>

      <section class="install-panel">
        <div v-if="!installStore.loaded" class="loading-state">
          <ElSkeleton :rows="8" animated />
        </div>

        <ElResult
          v-else-if="showInstalledResult"
          status="404"
          title="安装向导不可用"
          sub-title="系统已经安装完成。如需重新安装，请删除 install.lock 后重试。"
        >
          <template #extra>
            <ElButton type="primary" @click="goLogin">前往登录</ElButton>
          </template>
        </ElResult>

        <template v-else>
          <div class="steps-wrap">
            <ElSteps :active="currentStep" finish-status="success" align-center>
              <ElStep title="数据库" description="连接与验证" />
              <ElStep title="站点" description="基础信息" />
              <ElStep title="管理员" description="账号与路径" />
              <ElStep title="完成" description="进入后台" />
            </ElSteps>
          </div>

          <div class="step-wrap">
            <DbStep v-if="currentStep === 0" @next="currentStep = 1" />
            <SiteStep
              v-else-if="currentStep === 1"
              @back="currentStep = 0"
              @next="currentStep = 2"
            />
            <AdminStep
              v-else-if="currentStep === 2"
              @back="currentStep = 1"
              @next="handleInstallDone"
            />
            <DoneStep
              v-else
              :admin-path="doneState.adminPath"
              :restart-required="doneState.restartRequired"
              :config-file="doneState.configFile"
            />
          </div>
        </template>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { RoutesAlias } from '@/router/routesAlias'
  import { useInstallStore } from '@/store/modules/install'
  import { buildAdminHashUrl, getCachedAdminPath } from '@/utils/adminPath'
  import AdminStep from './modules/admin-step.vue'
  import DbStep from './modules/db-step.vue'
  import DoneStep from './modules/done-step.vue'
  import SiteStep from './modules/site-step.vue'

  defineOptions({ name: 'InstallPage' })

  interface InstallDoneState {
    adminPath: string
    restartRequired: boolean
    configFile: string
  }

  const installStore = useInstallStore()
  const currentStep = ref(0)
  const doneState = reactive<InstallDoneState>({
    adminPath: 'admin',
    restartRequired: false,
    configFile: ''
  })

  const showInstalledResult = computed(() => installStore.installed && currentStep.value !== 3)

  onMounted(async () => {
    await installStore.fetchStatus()
  })

  function handleInstallDone(payload: InstallDoneState) {
    doneState.adminPath = payload.adminPath
    doneState.restartRequired = payload.restartRequired
    doneState.configFile = payload.configFile
    currentStep.value = 3
  }

  function goLogin() {
    const target = buildAdminHashUrl(getCachedAdminPath(), RoutesAlias.Login)
    window.location.replace(target)
  }
</script>

<style scoped lang="scss">
  .install-shell {
    min-height: 100vh;
    padding: 32px 20px;
    background:
      radial-gradient(circle at top left, rgb(26 188 156 / 22%), transparent 32%),
      radial-gradient(circle at bottom right, rgb(14 165 233 / 18%), transparent 28%),
      linear-gradient(145deg, #07111d 0%, #102033 52%, #0b1724 100%);
  }

  .install-layout {
    display: grid;
    grid-template-columns: minmax(280px, 360px) minmax(0, 920px);
    gap: 24px;
    max-width: 1320px;
    margin: 0 auto;
  }

  .install-hero,
  .install-panel {
    border: 1px solid rgb(255 255 255 / 8%);
    border-radius: 28px;
    backdrop-filter: blur(18px);
    box-shadow: 0 24px 70px rgb(3 8 20 / 32%);
  }

  .install-hero {
    display: flex;
    flex-direction: column;
    gap: 18px;
    padding: 32px 28px;
    background: linear-gradient(180deg, rgb(8 20 33 / 92%), rgb(9 15 25 / 88%));
    color: #f4fbff;
  }

  .hero-kicker {
    color: #79e6d7;
    font-size: 12px;
    font-weight: 700;
    letter-spacing: 0.18em;
    text-transform: uppercase;
  }

  .install-hero h1 {
    margin: 0;
    font-size: 36px;
    line-height: 1.08;
  }

  .install-hero p {
    margin: 0;
    color: rgb(226 242 255 / 72%);
    line-height: 1.7;
  }

  .hero-points {
    display: grid;
    gap: 14px;
    margin-top: auto;
  }

  .point {
    display: grid;
    grid-template-columns: 38px minmax(0, 1fr);
    gap: 12px;
    align-items: center;
    padding: 14px 16px;
    border-radius: 18px;
    background: rgb(255 255 255 / 4%);
  }

  .point strong {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 38px;
    height: 38px;
    border-radius: 50%;
    background: linear-gradient(135deg, #49d7bf, #2db0ff);
    color: #04121d;
    font-size: 16px;
  }

  .point span {
    color: rgb(244 251 255 / 84%);
    font-weight: 600;
  }

  .install-panel {
    min-width: 0;
    padding: 28px;
    background: linear-gradient(180deg, rgb(255 255 255 / 96%), rgb(246 250 252 / 96%));
  }

  .loading-state,
  .step-wrap {
    min-height: 640px;
  }

  .steps-wrap {
    padding: 6px 6px 24px;
  }

  :deep(.el-step__title) {
    font-weight: 700;
  }

  :deep(.el-step__description) {
    line-height: 1.45;
  }

  @media (max-width: 1080px) {
    .install-layout {
      grid-template-columns: 1fr;
    }

    .install-hero {
      padding: 24px;
    }
  }

  @media (max-width: 768px) {
    .install-shell {
      padding: 16px 12px;
    }

    .install-panel {
      padding: 18px 14px;
      border-radius: 22px;
    }

    .install-hero h1 {
      font-size: 30px;
    }

    .loading-state,
    .step-wrap {
      min-height: auto;
    }
  }
</style>
