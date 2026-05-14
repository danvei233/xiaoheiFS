<template>
  <div class="probe-detail-page" v-loading="loading">
    <div class="detail-header art-card">
      <div class="header-main">
        <div class="header-left">
          <ElButton text @click="goBack">
            <ArtSvgIcon icon="ri:arrow-left-line" />
            <span>返回列表</span>
          </ElButton>

          <div class="title-section">
            <h1>{{ probe?.name || '探针详情' }}</h1>
            <div class="title-meta">
              <ElTag :type="getStatusTagType(probe?.status)">
                {{ getStatusText(probe?.status) }}
              </ElTag>
              <span class="meta-item">ID {{ probe?.id || '-' }}</span>
              <span class="meta-item">Agent {{ probe?.agent_id || '-' }}</span>
              <span class="meta-item">系统 {{ probe?.os_type || '-' }}</span>
            </div>
          </div>
        </div>

        <div class="header-actions">
          <ElTag type="info" effect="plain">自动刷新 5s</ElTag>
          <ElButton type="primary" :loading="refreshing" @click="refreshAll(true)">
            手动刷新
          </ElButton>
        </div>
      </div>
    </div>

    <div class="metrics-grid">
      <div class="metric-card art-card">
        <div class="metric-label">系统运行时长</div>
        <div class="metric-value">{{ formatUptime(snapshot?.system?.uptime) }}</div>
        <div class="metric-subtle">持续运行中</div>
      </div>

      <div class="metric-card art-card">
        <div class="metric-label">7 天 SLA</div>
        <div class="metric-value">{{ formatSla(sla?.uptime_percent) }}</div>
        <div class="metric-subtle">在线 {{ Number(sla?.online_seconds || 0) }} 秒</div>
      </div>

      <div class="metric-card art-card">
        <div class="metric-label">最后心跳</div>
        <div class="metric-value small">{{ formatDateTime(probe?.last_heartbeat_at) }}</div>
        <div class="metric-subtle">{{ fromNow(probe?.last_heartbeat_at) }}</div>
      </div>

      <div class="metric-card art-card">
        <div class="metric-label">最后快照</div>
        <div class="metric-value small">{{ formatDateTime(probe?.last_snapshot_at) }}</div>
        <div class="metric-subtle">{{ fromNow(probe?.last_snapshot_at) }}</div>
      </div>
    </div>

    <div class="detail-grid">
      <div class="main-column">
        <div class="art-card panel-card">
          <div class="panel-title">资源概览</div>

          <div class="usage-grid">
            <div class="usage-card">
              <div class="usage-head">
                <span>CPU 使用率</span>
                <span>{{ formatPercent(getNumber(snapshot?.cpu?.usage_percent)) }}</span>
              </div>
              <ElProgress
                :percentage="clampPercent(getNumber(snapshot?.cpu?.usage_percent))"
                :stroke-width="10"
                :show-text="false"
                :color="getUsageColor(getNumber(snapshot?.cpu?.usage_percent))"
              />
              <div class="usage-meta">
                <span>{{ toDisplayString(snapshot?.cpu?.model) }}</span>
                <span>{{ toDisplayString(snapshot?.cpu?.cores) }} 核</span>
              </div>
            </div>

            <div class="usage-card">
              <div class="usage-head">
                <span>内存使用率</span>
                <span>{{ formatPercent(getNumber(snapshot?.memory?.usage_percent)) }}</span>
              </div>
              <ElProgress
                :percentage="clampPercent(getNumber(snapshot?.memory?.usage_percent))"
                :stroke-width="10"
                :show-text="false"
                :color="getUsageColor(getNumber(snapshot?.memory?.usage_percent))"
              />
              <div class="usage-meta">
                <span>{{ formatBytes(snapshot?.memory?.used) }}</span>
                <span>/ {{ formatBytes(snapshot?.memory?.total) }}</span>
              </div>
            </div>
          </div>

          <ElDescriptions :column="2" border class="descriptions">
            <ElDescriptionsItem label="主机名">
              {{ toDisplayString(snapshot?.system?.hostname) }}
            </ElDescriptionsItem>
            <ElDescriptionsItem label="平台">
              {{ toDisplayString(snapshot?.system?.platform) }}
            </ElDescriptionsItem>
            <ElDescriptionsItem label="内核">
              {{ toDisplayString(snapshot?.system?.kernel) }}
            </ElDescriptionsItem>
            <ElDescriptionsItem label="系统类型">
              {{ toDisplayString(probe?.os_type) }}
            </ElDescriptionsItem>
          </ElDescriptions>
        </div>

        <div class="art-card panel-card">
          <div class="panel-title">磁盘明细</div>

          <ElTable :data="diskRows" border size="small" row-key="mount">
            <ElTableColumn prop="mount" label="挂载点" min-width="120" />
            <ElTableColumn prop="fs" label="文件系统" min-width="120" />
            <ElTableColumn label="总量" min-width="120">
              <template #default="{ row }">{{ formatBytes(row.total) }}</template>
            </ElTableColumn>
            <ElTableColumn label="已用" min-width="120">
              <template #default="{ row }">{{ formatBytes(row.used) }}</template>
            </ElTableColumn>
            <ElTableColumn label="使用率" min-width="140">
              <template #default="{ row }">
                <ElTag :type="getSlaTagType(getNumber(row.usage_percent))">
                  {{ formatPercent(getNumber(row.usage_percent)) }}
                </ElTag>
              </template>
            </ElTableColumn>
          </ElTable>
        </div>

        <div v-if="portRows.length" class="art-card panel-card">
          <div class="panel-title">端口监听</div>

          <ElTable :data="portRows" border size="small" row-key="port">
            <ElTableColumn prop="port" label="端口" min-width="90" />
            <ElTableColumn prop="proto" label="协议" min-width="90" />
            <ElTableColumn label="状态" min-width="100">
              <template #default="{ row }">
                <ElTag :type="row.listen ? 'success' : 'info'">
                  {{ row.listen ? '监听中' : '未监听' }}
                </ElTag>
              </template>
            </ElTableColumn>
            <ElTableColumn prop="process_name" label="进程" min-width="160" />
          </ElTable>
        </div>

        <div class="art-card panel-card">
          <div class="panel-head">
            <div class="panel-title">日志查看</div>
            <div class="panel-subtle">
              当前状态：{{ logRunning ? '连接中' : '未连接' }}，共 {{ logLines.length }} 行
            </div>
          </div>

          <div class="log-controls">
            <ElSelect
              v-model="logForm.source"
              clearable
              placeholder="日志来源"
              class="control-source"
            >
              <ElOption
                v-for="item in sourceOptions"
                :key="item.value"
                :label="item.label"
                :value="item.value"
              />
            </ElSelect>

            <ElInput
              v-model="logForm.keyword"
              clearable
              placeholder="日志关键词过滤"
              class="control-keyword"
            />

            <ElInputNumber v-model="logForm.lines" :min="50" :max="2000" class="control-lines" />

            <ElSwitch v-model="logForm.follow" active-text="跟随" inactive-text="一次性" />
            <ElSwitch v-model="autoScroll" active-text="自动滚动" inactive-text="手动滚动" />

            <ElButton type="primary" :loading="logLoading" @click="startLog">
              {{ logRunning ? '重新连接' : '开始日志' }}
            </ElButton>
            <ElButton :disabled="!logRunning" @click="stopLog">停止</ElButton>
            <ElButton @click="clearLog">清空</ElButton>
          </div>

          <div ref="logBoxRef" class="log-box">
            <template v-if="logLines.length">
              <div
                v-for="(line, index) in logLines"
                :key="`${index}-${line.slice(0, 24)}`"
                :class="['log-line', getLogLevelClass(line)]"
              >
                {{ line || ' ' }}
              </div>
            </template>

            <div v-else class="log-placeholder">暂无日志输出，点击“开始日志”后会在这里展示。</div>
          </div>
        </div>
      </div>

      <div class="side-column">
        <div class="art-card panel-card">
          <div class="panel-title">探针信息</div>

          <ElDescriptions :column="1" border>
            <ElDescriptionsItem label="探针 ID">{{ probe?.id || '-' }}</ElDescriptionsItem>
            <ElDescriptionsItem label="Agent ID">
              {{ probe?.agent_id || '-' }}
            </ElDescriptionsItem>
            <ElDescriptionsItem label="状态">
              <ElTag :type="getStatusTagType(probe?.status)">
                {{ getStatusText(probe?.status) }}
              </ElTag>
            </ElDescriptionsItem>
            <ElDescriptionsItem label="创建时间">
              {{ formatDateTime(probe?.created_at) }}
            </ElDescriptionsItem>
            <ElDescriptionsItem label="更新时间">
              {{ formatDateTime(probe?.updated_at) }}
            </ElDescriptionsItem>
            <ElDescriptionsItem label="标签">
              <div class="tag-list">
                <ElTag v-for="tag in probe?.tags || []" :key="tag" effect="plain">
                  {{ tag }}
                </ElTag>
                <span v-if="!(probe?.tags || []).length">-</span>
              </div>
            </ElDescriptionsItem>
          </ElDescriptions>
        </div>

        <div class="art-card panel-card">
          <div class="panel-title">状态事件</div>

          <div v-if="statusEvents.length" class="event-list">
            <div v-for="item in statusEvents" :key="item.id || item.created_at" class="event-item">
              <ElTag :type="getStatusTagType(item.status)">
                {{ getStatusText(item.status) }}
              </ElTag>

              <div class="event-meta">
                <div class="event-time">{{ formatDateTime(item.at || item.created_at) }}</div>
                <div class="event-reason">{{ item.reason || '系统状态变更' }}</div>
              </div>
            </div>
          </div>

          <ElEmpty v-else description="暂无状态事件" />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import type {
    ProbeRecord,
    ProbeSlaRecord,
    ProbeSnapshotRecord,
    ProbeStatusEventRecord
  } from '@/api/admin'
  import type { SseMessage } from '@/utils/http/sse'
  import {
    createAdminProbeLogSession,
    fetchAdminProbeDetail,
    fetchAdminProbeSla
  } from '@/api/admin'
  import { useUserStore } from '@/store/modules/user'
  import { createSseConnection } from '@/utils/http/sse'
  import { ElMessage } from 'element-plus'

  defineOptions({ name: 'ProbeDetailPage' })

  type SnapshotRow = Record<string, unknown>

  const route = useRoute()
  const router = useRouter()
  const userStore = useUserStore()

  const loading = ref(true)
  const refreshing = ref(false)
  const logLoading = ref(false)
  const logRunning = ref(false)
  const autoScroll = ref(true)

  const probe = ref<ProbeRecord | null>(null)
  const sla = ref<ProbeSlaRecord | null>(null)
  const logLines = ref<string[]>([])
  const logBoxRef = ref<HTMLElement | null>(null)

  const logForm = reactive({
    source: '',
    keyword: '',
    lines: 300,
    follow: true
  })

  const sourceOptions = [
    { label: '默认日志', value: '' },
    { label: 'Linux Journal(system)', value: 'journal:system' },
    { label: 'Linux Journal(pveproxy)', value: 'journal:pveproxy' },
    { label: 'Windows System 关键日志', value: 'eventlog:System:important' },
    { label: 'Windows System 全量日志', value: 'eventlog:System:full' },
    { label: 'Windows 开关机日志', value: 'eventlog:System:power' },
    { label: 'Windows Application 关键日志', value: 'eventlog:Application:important' },
    { label: 'Windows Hyper-V 关键日志', value: 'eventlog:Hyper-V-Worker:important' }
  ]

  const probeId = computed(() => Number(route.params.id || 0))
  const snapshot = computed<ProbeSnapshotRecord | null>(() => probe.value?.snapshot || null)
  const diskRows = computed<SnapshotRow[]>(() =>
    Array.isArray(snapshot.value?.disks) ? snapshot.value?.disks : []
  )
  const portRows = computed<SnapshotRow[]>(() =>
    Array.isArray(snapshot.value?.ports) ? snapshot.value?.ports : []
  )
  const statusEvents = computed<ProbeStatusEventRecord[]>(() => sla.value?.events || [])

  let poller: number | null = null
  let logConnection: ReturnType<typeof createSseConnection> | null = null
  let requestInFlight = false

  watch(
    probeId,
    async (value) => {
      stopPolling()
      stopLog()
      clearLog()

      if (!value) {
        loading.value = false
        return
      }

      await fetchData()
      startPolling()
    },
    { immediate: true }
  )

  watch(
    () => autoScroll.value,
    async (enabled) => {
      if (!enabled) {
        return
      }

      await nextTick()
      scrollLogToBottom()
    }
  )

  onBeforeUnmount(() => {
    stopPolling()
    stopLog()
  })

  function startPolling() {
    stopPolling()
    poller = window.setInterval(() => {
      fetchData(true)
    }, 5000)
  }

  function stopPolling() {
    if (!poller) {
      return
    }

    window.clearInterval(poller)
    poller = null
  }

  async function fetchData(silent = false, forceSnapshot = false) {
    if (!probeId.value || requestInFlight) {
      return
    }

    requestInFlight = true

    if (!silent) {
      if (forceSnapshot) {
        refreshing.value = true
      } else {
        loading.value = true
      }
    }

    try {
      const [detailPayload, slaPayload] = await Promise.all([
        fetchAdminProbeDetail(probeId.value, { refresh: forceSnapshot ? 1 : 0 }),
        fetchAdminProbeSla(probeId.value, { days: 7 })
      ])

      probe.value = detailPayload.probe || null
      sla.value = slaPayload.sla || null

      if (forceSnapshot && detailPayload.online === false) {
        ElMessage.warning('探针当前离线，无法触发新的实时快照')
      }
    } catch (error: any) {
      if (error?.code === 404 || error?.response?.status === 404) {
        ElMessage.error('探针不存在或已被删除')
        router.push('/probes/list')
      } else if (!silent) {
        ElMessage.error(error?.message || '加载探针详情失败')
      }
    } finally {
      loading.value = false
      refreshing.value = false
      requestInFlight = false
    }
  }

  async function refreshAll(forceSnapshot = false) {
    await fetchData(false, forceSnapshot)
  }

  function goBack() {
    router.push('/probes/list')
  }

  function getStatusText(status?: string) {
    return status === 'online' ? '在线' : '离线'
  }

  function getStatusTagType(status?: string) {
    return status === 'online' ? ('success' as const) : ('info' as const)
  }

  function formatDateTime(value?: string | null) {
    if (!value) {
      return '-'
    }

    const date = new Date(value)
    return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN')
  }

  function fromNow(value?: string | null) {
    if (!value) {
      return '-'
    }

    const time = new Date(value).getTime()
    if (Number.isNaN(time)) {
      return value
    }

    const diff = Date.now() - time
    const minutes = Math.floor(diff / 60000)
    const hours = Math.floor(diff / 3600000)
    const days = Math.floor(diff / 86400000)

    if (days > 0) {
      return `${days} 天前`
    }

    if (hours > 0) {
      return `${hours} 小时前`
    }

    if (minutes > 0) {
      return `${minutes} 分钟前`
    }

    return '刚刚'
  }

  function getNumber(value: unknown) {
    const parsed = Number(value)
    return Number.isFinite(parsed) ? parsed : 0
  }

  function clampPercent(value: number) {
    return Math.max(0, Math.min(100, Number(value.toFixed(1))))
  }

  function formatPercent(value: number) {
    return `${clampPercent(value).toFixed(1)}%`
  }

  function getUsageColor(value: number) {
    if (value < 60) {
      return '#67c23a'
    }

    if (value < 85) {
      return '#e6a23c'
    }

    return '#f56c6c'
  }

  function getSlaTagType(value: number) {
    if (value >= 99.9) {
      return 'success' as const
    }

    if (value >= 99) {
      return 'warning' as const
    }

    return 'danger' as const
  }

  function formatSla(value?: number | null) {
    if (value === null || value === undefined) {
      return '-'
    }

    return `${Number(value).toFixed(2)}%`
  }

  function formatUptime(value: unknown) {
    const seconds = getNumber(value)
    if (!seconds) {
      return '-'
    }

    const days = Math.floor(seconds / 86400)
    const hours = Math.floor((seconds % 86400) / 3600)
    const minutes = Math.floor((seconds % 3600) / 60)

    if (days > 0) {
      return `${days} 天 ${hours} 小时`
    }

    return `${hours} 小时 ${minutes} 分钟`
  }

  function formatBytes(value: unknown) {
    const size = getNumber(value)
    if (!size) {
      return '0 B'
    }

    const units = ['B', 'KB', 'MB', 'GB', 'TB']
    let current = size
    let index = 0

    while (current >= 1024 && index < units.length - 1) {
      current /= 1024
      index += 1
    }

    return `${current.toFixed(index === 0 ? 0 : 2)} ${units[index]}`
  }

  function toDisplayString(value: unknown) {
    if (value === null || value === undefined || String(value).trim() === '') {
      return '-'
    }

    return String(value)
  }

  function clearLog() {
    logLines.value = []
  }

  function stopLog() {
    if (logConnection) {
      logConnection.close()
      logConnection = null
    }

    logRunning.value = false
    logLoading.value = false
  }

  async function appendLog(text: string) {
    const normalized = String(text || '').replace(/\r\n/g, '\n')
    if (!normalized) {
      return
    }

    logLines.value.push(
      ...normalized
        .split('\n')
        .map((line) => normalizeDotNetDate(line))
        .filter((line, index, lines) => line !== '' || index !== lines.length - 1)
    )

    if (logLines.value.length > 4000) {
      logLines.value = logLines.value.slice(logLines.value.length - 4000)
    }

    await nextTick()
    scrollLogToBottom()
  }

  function appendSystemLog(message: string) {
    return appendLog(`[system] ${message}`)
  }

  function scrollLogToBottom() {
    if (!autoScroll.value || !logBoxRef.value) {
      return
    }

    logBoxRef.value.scrollTop = logBoxRef.value.scrollHeight
  }

  function normalizeDotNetDate(line: string) {
    return line.replace(/\/Date\((\d+)(?:[+-]\d+)?\)\//g, (raw, value) => {
      const timestamp = Number(value)
      if (!Number.isFinite(timestamp)) {
        return raw
      }

      return new Date(timestamp).toLocaleString('zh-CN')
    })
  }

  function getAuthorizationHeader() {
    const token = String(userStore.accessToken || '').trim()
    if (!token) {
      return ''
    }

    return token.startsWith('Bearer ') ? token : `Bearer ${token}`
  }

  function resolveSseUrl(path: string) {
    const base = String(import.meta.env.VITE_API_URL || '').trim()

    if (/^https?:\/\//i.test(path)) {
      return path
    }

    if (/^https?:\/\//i.test(base)) {
      return `${base.replace(/\/$/, '')}${path}`
    }

    return path
  }

  function safeParsePayload(raw: string) {
    try {
      return JSON.parse(raw) as { type?: string; data?: string }
    } catch {
      return null
    }
  }

  async function handleSseMessage(message: SseMessage) {
    if (!message.data) {
      return
    }

    const payload = safeParsePayload(message.data)
    const type = String(payload?.type || message.event || '')

    if (type === 'log_chunk') {
      await appendLog(String(payload?.data || ''))
      return
    }

    if (type === 'log_end') {
      await appendSystemLog('日志会话已结束')
      stopLog()
      return
    }

    await appendLog(message.data)
  }

  async function startLog() {
    if (!probeId.value) {
      return
    }

    stopLog()
    clearLog()
    logLoading.value = true

    try {
      const payload = await createAdminProbeLogSession(probeId.value, {
        source: logForm.source || undefined,
        keyword: logForm.keyword.trim() || undefined,
        follow: logForm.follow,
        lines: Number(logForm.lines || 300)
      })

      const streamPath = String(payload.stream_path || '')
      if (!streamPath) {
        throw new Error('日志流地址为空')
      }

      const authorization = getAuthorizationHeader()
      logRunning.value = true
      logConnection = createSseConnection(resolveSseUrl(streamPath), {
        headers: authorization ? { Authorization: authorization } : {},
        onMessage: (message) => {
          void handleSseMessage(message)
        },
        onError: () => {
          void appendSystemLog('日志连接中断，正在自动重试...')
        }
      })

      await appendSystemLog('日志连接已建立')
    } catch (error: any) {
      stopLog()
      ElMessage.error(error?.message || '启动日志失败')
    } finally {
      logLoading.value = false
    }
  }

  function getLogLevelClass(line: string) {
    const value = String(line || '').toLowerCase()

    if (value.includes('[error]') || value.includes('[critical]') || value.includes('[fail]')) {
      return 'is-error'
    }

    if (value.includes('[warning]') || value.includes('[warn]')) {
      return 'is-warning'
    }

    if (value.includes('[info]')) {
      return 'is-info'
    }

    if (value.includes('[debug]')) {
      return 'is-debug'
    }

    if (value.includes('[system]')) {
      return 'is-system'
    }

    return ''
  }
</script>

<style scoped lang="scss">
  .probe-detail-page {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .detail-header {
    padding: 20px;
  }

  .header-main,
  .header-actions,
  .header-left {
    display: flex;
  }

  .header-main {
    justify-content: space-between;
    gap: 16px;
  }

  .header-left {
    flex-direction: column;
    gap: 14px;
  }

  .header-actions {
    align-items: flex-start;
    gap: 12px;
  }

  .title-section h1 {
    margin: 0;
    color: var(--el-text-color-primary);
    font-size: 26px;
    font-weight: 700;
  }

  .title-meta {
    display: flex;
    flex-wrap: wrap;
    gap: 10px;
    margin-top: 10px;
    color: var(--el-text-color-secondary);
  }

  .meta-item {
    display: inline-flex;
    align-items: center;
  }

  .metrics-grid {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 16px;
  }

  .metric-card {
    padding: 18px;
  }

  .metric-label,
  .metric-subtle,
  .panel-subtle,
  .usage-meta,
  .event-time,
  .event-reason {
    color: var(--el-text-color-secondary);
    font-size: 13px;
  }

  .metric-value {
    margin-top: 10px;
    color: var(--el-text-color-primary);
    font-size: 28px;
    font-weight: 700;
    line-height: 1.2;
  }

  .metric-value.small {
    font-size: 18px;
  }

  .detail-grid {
    display: grid;
    grid-template-columns: minmax(0, 2fr) minmax(320px, 1fr);
    gap: 16px;
  }

  .main-column,
  .side-column {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .panel-card {
    padding: 18px;
  }

  .panel-title {
    color: var(--el-text-color-primary);
    font-size: 16px;
    font-weight: 600;
  }

  .panel-head {
    display: flex;
    justify-content: space-between;
    gap: 12px;
    margin-bottom: 14px;
  }

  .usage-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 16px;
    margin: 16px 0;
  }

  .usage-card {
    padding: 16px;
    border-radius: 12px;
    background: var(--el-fill-color-light);
  }

  .usage-head {
    display: flex;
    justify-content: space-between;
    gap: 12px;
    margin-bottom: 10px;
    color: var(--el-text-color-primary);
    font-weight: 600;
  }

  .usage-meta {
    display: flex;
    justify-content: space-between;
    gap: 12px;
    margin-top: 10px;
  }

  .descriptions {
    margin-top: 8px;
  }

  .tag-list {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }

  .event-list {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .event-item {
    display: flex;
    gap: 12px;
    padding: 12px;
    border-radius: 12px;
    background: var(--el-fill-color-light);
  }

  .event-meta {
    min-width: 0;
  }

  .event-reason {
    margin-top: 4px;
    word-break: break-word;
  }

  .log-controls {
    display: flex;
    flex-wrap: wrap;
    gap: 12px;
    margin: 14px 0;
  }

  .control-source {
    width: 220px;
  }

  .control-keyword {
    width: 220px;
  }

  .control-lines {
    width: 120px;
  }

  .log-box {
    min-height: 320px;
    max-height: 460px;
    overflow: auto;
    padding: 14px;
    border-radius: 14px;
    background: #0f172a;
    font-family: 'JetBrains Mono', 'Consolas', monospace;
    font-size: 13px;
    line-height: 1.65;
  }

  .log-line {
    padding: 2px 0;
    color: #e2e8f0;
    word-break: break-word;
    white-space: pre-wrap;
  }

  .log-line.is-error {
    color: #fda4af;
  }

  .log-line.is-warning {
    color: #fcd34d;
  }

  .log-line.is-info {
    color: #93c5fd;
  }

  .log-line.is-debug {
    color: #94a3b8;
  }

  .log-line.is-system {
    color: #86efac;
  }

  .log-placeholder {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 280px;
    color: #94a3b8;
    text-align: center;
  }

  @media (max-width: 1200px) {
    .metrics-grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }

    .detail-grid {
      grid-template-columns: 1fr;
    }
  }

  @media (max-width: 768px) {
    .header-main,
    .header-actions,
    .usage-grid,
    .panel-head {
      flex-direction: column;
    }

    .metrics-grid,
    .usage-grid {
      grid-template-columns: 1fr;
    }

    .control-source,
    .control-keyword,
    .control-lines {
      width: 100%;
    }
  }
</style>
