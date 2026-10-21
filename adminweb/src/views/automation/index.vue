<template>
  <div class="automation-page art-full-height">
    <ElCard shadow="never" class="hero-card">
      <div class="hero-header">
        <div>
          <div class="hero-title">自动化对接</div>
          <div class="hero-subtitle">
            这个旧入口现在只保留查看能力。可写的自动化插件实例请到“商品目录 / 商品类型”中配置。
          </div>
        </div>

        <ElButton type="primary" @click="goCatalog">前往商品目录</ElButton>
      </div>

      <ElAlert
        type="warning"
        show-icon
        :closable="false"
        title="旧自动化配置为只读"
        description="请使用绑定到商品类型的自动化插件实例来管理 base_url、api_key、timeout 和 dry_run 等参数。"
      />
    </ElCard>

    <ElRow :gutter="16">
      <ElCol :xs="24" :lg="14">
        <ElCard shadow="never" class="section-card">
          <template #header>
            <div class="section-header">
              <div>
                <div class="section-title">旧配置快照</div>
                <div class="section-subtitle">
                  该区域用于兼容旧逻辑和排障查看，实际编辑入口已经迁移到商品目录。
                </div>
              </div>
              <ElTag :type="config.configured ? 'success' : 'warning'">
                {{ config.configured ? '已配置' : '未配置' }}
              </ElTag>
            </div>
          </template>

          <ElAlert
            v-if="configError"
            type="error"
            show-icon
            :closable="false"
            :title="configError"
            class="section-alert"
          />

          <ElDescriptions :column="2" border>
            <ElDescriptionsItem label="基础地址">{{ config.base_url || '-' }}</ElDescriptionsItem>
            <ElDescriptionsItem label="API 密钥">{{ maskedApiKey }}</ElDescriptionsItem>
            <ElDescriptionsItem label="启用状态">
              <ElTag :type="config.enabled ? 'success' : 'info'">
                {{ config.enabled ? '已启用' : '已停用' }}
              </ElTag>
            </ElDescriptionsItem>
            <ElDescriptionsItem label="演练模式">
              <ElTag :type="config.dry_run ? 'warning' : 'success'">
                {{ config.dry_run ? '开启' : '关闭' }}
              </ElTag>
            </ElDescriptionsItem>
            <ElDescriptionsItem label="超时时间（秒）">
              {{ config.timeout_sec ?? 12 }}
            </ElDescriptionsItem>
            <ElDescriptionsItem label="重试次数">{{ config.retry ?? 0 }}</ElDescriptionsItem>
            <ElDescriptionsItem label="插件 ID">
              {{ config.plugin_id || '-' }}
            </ElDescriptionsItem>
            <ElDescriptionsItem label="实例 ID">
              {{ config.instance_id || '-' }}
            </ElDescriptionsItem>
            <ElDescriptionsItem label="配置来源">
              {{ formatConfigSource(config.config_source) }}
            </ElDescriptionsItem>
            <ElDescriptionsItem label="兼容模式">
              {{ config.compat_mode ? '是' : '否' }}
            </ElDescriptionsItem>
          </ElDescriptions>
        </ElCard>
      </ElCol>

      <ElCol :xs="24" :lg="10">
        <ElRow :gutter="16">
          <ElCol :xs="24" :sm="8" :lg="24">
            <ElCard shadow="never" class="metric-card">
              <ElStatistic title="已同步线路" :value="stats.lines ?? 0" />
              <div class="metric-help">{{ lineMetricHelp }}</div>
            </ElCard>
          </ElCol>
          <ElCol :xs="24" :sm="8" :lg="24">
            <ElCard shadow="never" class="metric-card">
              <ElStatistic title="已同步套餐" :value="stats.packages ?? 0" />
              <div class="metric-help">{{ packageMetricHelp }}</div>
            </ElCard>
          </ElCol>
          <ElCol :xs="24" :sm="8" :lg="24">
            <ElCard shadow="never" class="metric-card">
              <ElStatistic title="已同步镜像" :value="stats.images ?? 0" />
              <div class="metric-help">{{ imageMetricHelp }}</div>
            </ElCard>
          </ElCol>
        </ElRow>
      </ElCol>
    </ElRow>

    <ElCard shadow="never" class="section-card">
      <template #header>
        <div class="section-header">
          <div>
            <div class="section-title">同步日志</div>
            <div class="section-subtitle">
              展示旧自动化对接最近一次同步商品目录时留下的结果记录。
            </div>
          </div>

          <ElButton :loading="logsLoading" @click="fetchLogs">刷新</ElButton>
        </div>
      </template>

      <ElTable :data="logs" border v-loading="logsLoading" row-key="id" empty-text="暂无同步日志">
        <ElTableColumn prop="id" label="ID" width="80" />
        <ElTableColumn prop="status" label="状态" width="120">
          <template #default="{ row }">
            <ElTag :type="getLogTagType(row.status)">{{ formatLogStatus(row.status) }}</ElTag>
          </template>
        </ElTableColumn>
        <ElTableColumn prop="mode" label="模式" width="120" />
        <ElTableColumn prop="message" label="消息" min-width="320" show-overflow-tooltip />
        <ElTableColumn prop="created_at" label="创建时间" min-width="180">
          <template #default="{ row }">
            {{ formatDateTime(row.created_at) }}
          </template>
        </ElTableColumn>
      </ElTable>
    </ElCard>
  </div>
</template>

<script setup lang="ts">
  import type { AutomationConfigRecord, IntegrationSyncLogRecord } from '@/api/admin'
  import {
    fetchAdminAutomationConfig,
    fetchAdminAutomationSyncLogs,
    fetchAdminLines,
    fetchAdminPackages,
    fetchAdminSystemImages,
    hasAdminPermission
  } from '@/api/admin'
  import { useUserStore } from '@/store/modules/user'
  import { ElMessage } from 'element-plus'

  defineOptions({ name: 'AutomationPage' })

  const router = useRouter()
  const userStore = useUserStore()
  const { info } = storeToRefs(userStore)

  const configLoading = ref(false)
  const logsLoading = ref(false)
  const statsLoading = ref(false)
  const configError = ref('')
  const config = reactive<AutomationConfigRecord>({
    base_url: '',
    api_key: '',
    enabled: false,
    timeout_sec: 12,
    retry: 0,
    dry_run: false,
    configured: false,
    compat_mode: false,
    plugins_ready: false,
    config_source: 'goods_type_plugin_instance',
    plugin_id: '',
    instance_id: ''
  })
  const logs = ref<IntegrationSyncLogRecord[]>([])
  const stats = reactive({
    lines: 0,
    packages: 0,
    images: 0
  })

  const canViewLines = computed(() => hasAdminPermission(info.value?.buttons, ['line.list']))
  const canViewPackages = computed(() => hasAdminPermission(info.value?.buttons, ['package.list']))
  const canViewImages = computed(() =>
    hasAdminPermission(info.value?.buttons, ['system_image.list'])
  )

  const maskedApiKey = computed(() => {
    const raw = String(config.api_key || '')
    if (!raw) {
      return '-'
    }
    if (raw.length <= 4) {
      return '****'
    }
    return `${raw.slice(0, 2)}****${raw.slice(-2)}`
  })

  const lineMetricHelp = computed(() =>
    canViewLines.value ? '统计当前线路记录数量。' : '缺少 `line.list` 权限。'
  )
  const packageMetricHelp = computed(() =>
    canViewPackages.value ? '统计当前套餐记录数量。' : '缺少 `package.list` 权限。'
  )
  const imageMetricHelp = computed(() =>
    canViewImages.value ? '统计当前系统镜像记录数量。' : '缺少 `system_image.list` 权限。'
  )

  onMounted(() => {
    fetchPageData()
  })

  async function fetchPageData() {
    await Promise.all([fetchConfig(), fetchLogs(), fetchStats()])
  }

  async function fetchConfig() {
    configLoading.value = true
    configError.value = ''

    try {
      const payload = await fetchAdminAutomationConfig()
      Object.assign(config, payload || {})
    } catch (error: any) {
      configError.value = String(error?.message || '加载自动化配置失败')
    } finally {
      configLoading.value = false
    }
  }

  async function fetchLogs() {
    logsLoading.value = true

    try {
      const payload = await fetchAdminAutomationSyncLogs({
        limit: 20,
        offset: 0
      })
      logs.value = payload.items || []
    } catch {
      logs.value = []
    } finally {
      logsLoading.value = false
    }
  }

  async function fetchStats() {
    statsLoading.value = true

    try {
      const tasks: Promise<void>[] = []

      if (canViewLines.value) {
        tasks.push(
          fetchAdminLines().then((payload) => {
            stats.lines = Array.isArray(payload.items) ? payload.items.length : 0
          })
        )
      } else {
        stats.lines = 0
      }

      if (canViewPackages.value) {
        tasks.push(
          fetchAdminPackages().then((payload) => {
            stats.packages = Array.isArray(payload.items) ? payload.items.length : 0
          })
        )
      } else {
        stats.packages = 0
      }

      if (canViewImages.value) {
        tasks.push(
          fetchAdminSystemImages().then((payload) => {
            stats.images = Array.isArray(payload.items) ? payload.items.length : 0
          })
        )
      } else {
        stats.images = 0
      }

      await Promise.all(tasks)
    } catch {
      stats.lines = canViewLines.value ? stats.lines : 0
      stats.packages = canViewPackages.value ? stats.packages : 0
      stats.images = canViewImages.value ? stats.images : 0
    } finally {
      statsLoading.value = false
    }
  }

  function goCatalog() {
    ElMessage.info('请到商品目录的商品类型中配置自动化插件实例。')
    router.push('/catalog')
  }

  function formatConfigSource(value?: string) {
    const normalized = String(value || '').trim()
    if (!normalized || normalized === 'goods_type_plugin_instance') {
      return '商品类型插件实例'
    }
    return normalized
  }

  function formatLogStatus(status?: string) {
    const normalized = String(status || '').toLowerCase()
    if (normalized === 'success' || normalized === 'ok') {
      return '成功'
    }
    if (normalized === 'failed' || normalized === 'error') {
      return '失败'
    }
    if (normalized === 'running') {
      return '运行中'
    }
    return status || '-'
  }

  function formatDateTime(value?: string) {
    if (!value) {
      return '-'
    }

    const date = new Date(value)
    return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN')
  }

  function getLogTagType(status?: string) {
    const normalized = String(status || '').toLowerCase()
    if (normalized === 'success' || normalized === 'ok') {
      return 'success' as const
    }
    if (normalized === 'failed' || normalized === 'error') {
      return 'danger' as const
    }
    if (normalized === 'running') {
      return 'warning' as const
    }
    return 'info' as const
  }
</script>

<style scoped lang="scss">
  .automation-page {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .hero-card,
  .section-card,
  .metric-card {
    border-radius: 18px;
  }

  .hero-header,
  .section-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 16px;
  }

  .hero-title,
  .section-title {
    color: var(--el-text-color-primary);
    font-size: 20px;
    font-weight: 700;
  }

  .section-title {
    font-size: 16px;
  }

  .hero-subtitle,
  .section-subtitle,
  .metric-help {
    margin-top: 6px;
    color: var(--el-text-color-secondary);
    font-size: 13px;
    line-height: 1.7;
  }

  .section-alert {
    margin-bottom: 16px;
  }

  .metric-help {
    min-height: 42px;
  }

  @media (max-width: 900px) {
    .hero-header,
    .section-header {
      flex-direction: column;
    }
  }
</style>
