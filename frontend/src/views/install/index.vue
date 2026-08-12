<template>
  <div v-if="showNotFound" class="not-found-page art-full-height">
    <ThemeSvg :src="notFoundImage" class="not-found-image" />
    <h1>404</h1>
    <p>页面不存在</p>
    <div class="not-found-actions">
      <ElButton type="primary" @click="router.push('/')">
        <ArtSvgIcon icon="ri:home-5-line" />
        <span>返回首页</span>
      </ElButton>
      <ElButton @click="router.back()">
        <ArtSvgIcon icon="ri:arrow-left-line" />
        <span>返回上一页</span>
      </ElButton>
    </div>
  </div>

  <div v-else class="install-page art-full-height">
    <AuthTopBar />

    <main class="install-main">
      <header class="install-brand">
        <ArtLogo size="42" />
        <div>
          <h1>安装向导</h1>
          <p>完成基础配置后即可开始使用</p>
        </div>
      </header>

      <ElCard class="wizard-card art-card-xs" shadow="never">
        <ElSteps :active="currentStep" finish-status="success" align-center class="wizard-steps">
          <ElStep
            v-for="item in steps"
            :key="item.title"
            :title="item.title"
            :description="item.description"
          />
        </ElSteps>

        <div class="step-content">
          <DbStep v-if="currentStep === 0" @next="handleDbNext" />
          <SiteStep v-else-if="currentStep === 1" @next="handleSiteNext" @back="currentStep = 0" />
          <AdminStep
            v-else-if="currentStep === 2"
            @next="handleAdminNext"
            @back="currentStep = 1"
          />
          <DoneStep
            v-else
            :admin-path="doneData.adminPath"
            :restart="doneData.restart"
            :config-file="doneData.configFile"
          />
        </div>
      </ElCard>
    </main>
  </div>
</template>

<script setup lang="ts">
  import notFoundImage from '@imgs/svg/404.svg'
  import { useInstallStore } from '@/stores/install'
  import { useInstallWizardStore } from '@/stores/installWizard'
  import DbStep from './modules/db-step.vue'
  import SiteStep from './modules/site-step.vue'
  import AdminStep from './modules/admin-step.vue'
  import DoneStep from './modules/done-step.vue'

  defineOptions({ name: 'Install' })

  const router = useRouter()
  const install = useInstallStore()
  const currentStep = ref(0)
  const showNotFound = ref(false)
  const steps = [
    { title: '数据库', description: '连接配置' },
    { title: '站点', description: '基础信息' },
    { title: '管理员', description: '账号与路径' },
    { title: '完成', description: '开始使用' }
  ]
  const doneData = reactive({
    adminPath: 'admin',
    restart: false,
    configFile: ''
  })
  const handleDbNext = () => {
    currentStep.value = 1
  }

  const handleSiteNext = () => {
    currentStep.value = 2
  }

  const handleAdminNext = (adminPath: string, restart: boolean, configFile: string) => {
    doneData.adminPath = adminPath
    doneData.restart = restart
    doneData.configFile = configFile
    currentStep.value = 3
  }

  onMounted(async () => {
    if (!install.loaded) await install.fetchStatus()
    if (install.installed) showNotFound.value = true
  })

  // H5：离开安装流程即清除内存中的 DBA 密码，避免在页面生命周期内继续驻留。
  onBeforeUnmount(() => {
    useInstallWizardStore().clearSecrets()
  })
</script>

<style lang="scss" scoped>
  .not-found-page {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    width: 100%;
    min-height: 100vh;
    padding: 40px 24px;
    text-align: center;
    background: var(--default-bg-color);
  }

  .not-found-image {
    width: min(400px, 90vw);
  }

  .not-found-page h1 {
    margin: 24px 0 8px;
    font-size: 36px;
    color: var(--art-gray-900);
    letter-spacing: 0;
  }

  .not-found-page p {
    margin: 0;
    font-size: 16px;
    color: var(--art-gray-600);
  }

  .not-found-actions {
    display: flex;
    flex-wrap: wrap;
    gap: 10px;
    justify-content: center;
    margin-top: 28px;
  }

  .install-page {
    position: relative;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 100%;
    min-height: 100vh;
    padding: 72px 24px 36px;
    overflow: auto;
    background: var(--default-bg-color);
  }

  .install-main {
    width: min(940px, 100%);
  }

  .install-brand {
    display: flex;
    gap: 12px;
    align-items: center;
    margin-bottom: 18px;
  }

  .install-brand h1 {
    margin: 0;
    font-size: 22px;
    font-weight: 700;
    color: var(--art-gray-900);
  }

  .install-brand p {
    margin: 3px 0 0;
    font-size: 13px;
    color: var(--art-gray-600);
  }

  .wizard-card {
    width: 100%;
    background: var(--default-box-color);
    border-color: var(--art-card-border);
  }

  .wizard-steps {
    padding: 12px 8px 28px;
    border-bottom: 1px solid var(--default-border);
  }

  .step-content {
    padding: 28px 8px 4px;
  }

  @media (width <= 640px) {
    .install-page {
      align-items: flex-start;
      height: auto;
      padding: 68px 12px 20px;
    }

    .wizard-steps {
      padding-inline: 0;
    }

    .step-content {
      padding: 22px 0 0;
    }
  }
</style>
