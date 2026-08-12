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
              {{ config.timeout_sec ?? 15 }}
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

    <ElCard shadow="never" class="section-card art-table-card">
      <template #header>
        <div class="section-header">
          <div>
            <div class="section-title">同步日志</div>
            <div class="section-subtitle">
              展示旧自动化对接最近一次同步商品目录时留下的结果记录。
            </div>
          </div>
        </div>
      </template>

      <ArtTableHeader
        v-model:columns="columnChecks"
        :show-search-bar="false"
        :loading="logsLoading"
        @refresh="refreshLogs"
      />

      <ArtTable :data="logs" :columns="columns" :loading="logsLoading" row-key="id">
        <template #status="{ row }">
          <ElTag :type="getLogTagType(row.status)">{{ formatLogStatus(row.status) }}</ElTag>
        </template>
        <template #created_at="{ row }">
          {{ formatDateTime(row.created_at) }}
        </template>
      </ArtTable>
    </ElCard>
  </div>
</template>

<script setup lang="ts">
  import type { AutomationConfigRecord, IntegrationSyncLogRecord } from '@/services/admin'
  import {
    fetchAdminAutomationConfig,
    fetchAdminAutomationSyncLogs,
    fetchAdminLines,
    fetchAdminPackages,
    fetchAdminSystemImages
  } from '@/services/admin'
  import { useTable } from '@/hooks/core/useTable'
  import { useAuth } from '@/hooks/core/useAuth'
  import { ElMessage } from 'element-plus'

  defineOptions({ name: 'AutomationPage' })

  const router = useRouter()
  const { hasAuth } = useAuth()

  const configLoading = ref(false)
  const statsLoading = ref(false)
  const configError = ref('')
  const config = reactive<AutomationConfigRecord>({
    base_url: '',
    api_key: '',
    enabled: false,
    timeout_sec: 15,
    retry: 0,
    dry_run: false,
    configured: false,
    compat_mode: false,
    plugins_ready: false,
    config_source: 'goods_type_plugin_instance',
    plugin_id: '',
    instance_id: ''
  })
  const {
    columnChecks,
    columns,
    data: logs,
    loading: logsLoading,
    getData: loadLogs
  } = useTable({
    core: {
      apiFn: fetchLogTable,
      apiParams: { current: 1, size: 200 },
      immediate: false,
      columnsFactory: () => [
        { prop: 'id', label: 'ID', width: 80 },
        { prop: 'status', label: '状态', width: 120, useSlot: true },
        { prop: 'message', label: '消息', minWidth: 320, showOverflowTooltip: true },
        { prop: 'created_at', label: '创建时间', minWidth: 180, useSlot: true }
      ]
    }
  })
  const stats = reactive({
    lines: 0,
    packages: 0,
    images: 0
  })

  const canViewLines = computed(() => hasAuth('line.list'))
  const canViewPackages = computed(() => hasAuth('package.list'))
  const canViewImages = computed(() => hasAuth('system_image.list'))

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
    await Promise.all([fetchConfig(), refreshLogs(), fetchStats()])
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

  async function fetchLogTable(
    _params?: unknown,
    signal?: AbortSignal
  ): Promise<Api.Common.PaginatedResponse<IntegrationSyncLogRecord>> {
    const payload = await fetchAdminAutomationSyncLogs(
      {
        limit: 200,
        offset: 0
      },
      signal
    )
    const records = payload.items || []
    return {
      records,
      current: 1,
      size: 200,
      total: Number(payload.total || records.length)
    }
  }

  async function refreshLogs() {
    try {
      await loadLogs()
    } catch {
      ElMessage.error('加载同步日志失败')
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
      ElMessage.error('加载同步统计失败')
      stats.lines = canViewLines.value ? stats.lines : 0
      stats.packages = canViewPackages.value ? stats.packages : 0
      stats.images = canViewImages.value ? stats.images : 0
    } finally {
      statsLoading.value = false
    }
  }

  function goCatalog() {
    ElMessage.info('请到商品目录的商品类型中配置自动化插件实例。')
    router.push({ name: 'CatalogPage' })
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
    border-radius: 8px;
  }

  .hero-header,
  .section-header {
    display: flex;
    gap: 16px;
    align-items: flex-start;
    justify-content: space-between;
  }

  .hero-title,
  .section-title {
    font-size: 20px;
    font-weight: 700;
    color: var(--el-text-color-primary);
  }

  .section-title {
    font-size: 16px;
  }

  .hero-subtitle,
  .section-subtitle,
  .metric-help {
    margin-top: 6px;
    font-size: 13px;
    line-height: 1.7;
    color: var(--el-text-color-secondary);
  }

  .section-alert {
    margin-bottom: 16px;
  }

  .metric-help {
    min-height: 42px;
  }

  @media (width <= 900px) {
    .hero-header,
    .section-header {
      flex-direction: column;
    }
  }
</style>
