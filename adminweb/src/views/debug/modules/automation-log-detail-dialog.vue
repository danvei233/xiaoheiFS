<template>
  <ElDialog
    v-model="dialogVisible"
    title="自动化请求详情"
    width="1200px"
    top="20px"
    destroy-on-close
    align-center
  >
    <template v-if="record">
      <div class="detail-dialog">
        <div class="detail-summary">
          <div class="summary-row"
            ><span>动作</span><strong>{{ record.action || '-' }}</strong></div
          >
          <div class="summary-row"
            ><span>订单 ID</span><strong>{{ record.order_id || '-' }}</strong></div
          >
          <div class="summary-row"
            ><span>协议</span><ElTag>{{ detailProtocol }}</ElTag></div
          >
          <div class="summary-row"
            ><span>连接</span><strong>{{ detailConnection }}</strong></div
          >
          <div class="summary-row"
            ><span>结果</span
            ><ElTag :type="record.success ? 'success' : 'danger'">{{
              record.success ? '成功' : '失败'
            }}</ElTag></div
          >
          <div class="summary-row"
            ><span>时间</span><strong>{{ formatDateTime(record.created_at) }}</strong></div
          >
        </div>

        <ElAlert
          :type="record.success ? 'success' : 'error'"
          :closable="false"
          show-icon
          :title="record.message || 'No message'"
        />

        <div class="detail-panels">
          <section class="detail-panel">
            <div class="panel-title">Request</div>
            <HttpRequestBar
              :method="detailRequest.method || 'GET'"
              :url="detailRequest.url || '-'"
            />
            <ElTabs v-model="requestActiveTab">
              <ElTabPane label="Headers" name="headers">
                <HttpHeadersTable :headers="requestHeaders" copyable />
              </ElTabPane>
              <ElTabPane label="Body" name="body">
                <pre class="code-block">{{ formatJson(detailRequest.body) }}</pre>
              </ElTabPane>
            </ElTabs>
          </section>

          <section class="detail-panel">
            <div class="panel-title">Response</div>
            <HttpRequestBar
              method="RESP"
              :status="detailResponse.status"
              :duration="detailResponse.duration_ms"
            />
            <ElTabs v-model="responseActiveTab">
              <ElTabPane label="Headers" name="headers">
                <HttpHeadersTable :headers="responseHeaders" copyable />
              </ElTabPane>
              <ElTabPane label="Body" name="body">
                <template v-if="responsePreview.type === 'html'">
                  <div class="html-preview">
                    <iframe :srcdoc="responsePreview.content" title="response-preview" />
                  </div>
                </template>
                <template v-else>
                  <pre class="code-block">{{ responsePreview.content }}</pre>
                </template>
              </ElTabPane>
            </ElTabs>
          </section>
        </div>
      </div>
    </template>

    <template #footer>
      <div class="dialog-footer">
        <ElButton @click="copyAllDetails">复制全部</ElButton>
        <ElButton type="primary" @click="dialogVisible = false">关闭</ElButton>
      </div>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  import type { AutomationLogRecord } from '@/api/admin'
  import HttpHeadersTable from './http-headers-table.vue'
  import HttpRequestBar from './http-request-bar.vue'

  defineOptions({ name: 'AutomationLogDetailDialog' })

  interface ParsedRequestPayload {
    method?: string
    url?: string
    headers?: Record<string, unknown>
    body?: unknown
  }

  interface ParsedResponsePayload {
    status?: number | string
    headers?: Record<string, unknown>
    body?: unknown
    body_json?: unknown
    duration_ms?: number
    format?: string
  }

  interface Props {
    visible: boolean
    record?: AutomationLogRecord | null
  }

  interface Emits {
    (e: 'update:visible', value: boolean): void
  }

  const props = withDefaults(defineProps<Props>(), {
    record: null
  })
  const emit = defineEmits<Emits>()

  const requestActiveTab = ref('headers')
  const responseActiveTab = ref('headers')

  const dialogVisible = computed({
    get: () => props.visible,
    set: (value) => emit('update:visible', value)
  })

  watch(dialogVisible, (visible) => {
    if (visible) {
      requestActiveTab.value = 'headers'
      responseActiveTab.value = 'headers'
    }
  })

  const detailRequest = computed<ParsedRequestPayload>(() => {
    return parsePayload(props.record?.request_json) as ParsedRequestPayload
  })

  const detailResponse = computed<ParsedResponsePayload>(() => {
    return parsePayload(props.record?.response_json) as ParsedResponsePayload
  })

  const requestHeaders = computed(() => normalizeHeaders(detailRequest.value.headers))
  const responseHeaders = computed(() => normalizeHeaders(detailResponse.value.headers))
  const detailProtocol = computed(() => requestProtocol(detailRequest.value))
  const detailConnection = computed(() => requestConnection(detailRequest.value))

  const responsePreview = computed(() => {
    if (detailResponse.value.body_json !== undefined) {
      return { type: 'json' as const, content: formatJson(detailResponse.value.body_json) }
    }

    if (detailResponse.value.body === undefined || detailResponse.value.body === null) {
      return { type: 'text' as const, content: '-' }
    }

    const bodyText =
      typeof detailResponse.value.body === 'string'
        ? detailResponse.value.body
        : formatJson(detailResponse.value.body)
    const lowered = bodyText.trim().toLowerCase()

    if (
      detailResponse.value.format === 'html' ||
      lowered.startsWith('<!doctype') ||
      lowered.startsWith('<html')
    ) {
      return { type: 'html' as const, content: bodyText }
    }

    return { type: 'text' as const, content: bodyText }
  })

  function parsePayload(payload: unknown): Record<string, unknown> {
    if (!payload) return {}
    if (typeof payload === 'string') {
      try {
        return JSON.parse(payload) as Record<string, unknown>
      } catch {
        return { body: payload }
      }
    }
    return typeof payload === 'object' ? (payload as Record<string, unknown>) : { body: payload }
  }

  function normalizeHeaders(headers: unknown): Record<string, string> {
    if (!headers || typeof headers !== 'object') return {}
    return Object.fromEntries(
      Object.entries(headers as Record<string, unknown>).map(([key, value]) => [
        key,
        value == null ? '' : String(value)
      ])
    )
  }

  function findHeaderValue(headers: unknown, targetKey: string) {
    if (!headers || typeof headers !== 'object') return ''
    const matched = Object.entries(headers as Record<string, unknown>).find(
      ([key]) => key.toLowerCase() === targetKey.toLowerCase()
    )
    return matched?.[1] == null ? '' : String(matched[1])
  }

  function requestProtocol(request: ParsedRequestPayload) {
    const method = String(request.method || '')
      .trim()
      .toUpperCase()
    if (method) return method
    const transport = findHeaderValue(request.headers, 'x-transport').trim()
    return transport ? transport.toUpperCase() : 'UNKNOWN'
  }

  function requestConnection(request: ParsedRequestPayload) {
    const pluginId = findHeaderValue(request.headers, 'x-plugin-id').trim()
    const instanceId = findHeaderValue(request.headers, 'x-plugin-instance-id').trim()
    if (pluginId || instanceId) return `${pluginId || '-'} / ${instanceId || '-'}`
    const urlText = String(request.url || '').trim()
    if (!urlText) return '-'
    try {
      return new URL(urlText).host || urlText
    } catch {
      return urlText
    }
  }

  function formatJson(payload: unknown) {
    if (payload === undefined || payload === null) return '-'
    if (typeof payload === 'string') return payload
    try {
      return JSON.stringify(payload, null, 2)
    } catch {
      return String(payload)
    }
  }

  function formatDateTime(value?: string) {
    if (!value) return '-'
    const date = new Date(value)
    return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN')
  }

  async function copyAllDetails() {
    if (!props.record) return
    try {
      await navigator.clipboard.writeText(
        JSON.stringify(
          {
            action: props.record.action,
            order_id: props.record.order_id,
            success: props.record.success,
            message: props.record.message,
            request: detailRequest.value,
            response: detailResponse.value,
            created_at: props.record.created_at
          },
          null,
          2
        )
      )
      ElMessage.success('全部详情已复制到剪贴板')
    } catch {
      ElMessage.error('复制失败')
    }
  }
</script>

<style scoped lang="scss">
  .detail-dialog,
  .detail-panel {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .detail-summary,
  .detail-panels {
    display: grid;
    gap: 16px;
  }

  .detail-summary {
    grid-template-columns: repeat(3, minmax(0, 1fr));
    padding: 16px;
    border: 1px solid var(--el-border-color-light);
    border-radius: 12px;
    background: linear-gradient(135deg, rgb(251 191 36 / 10%), rgb(59 130 246 / 8%));
  }

  .summary-row {
    display: flex;
    gap: 10px;
    align-items: center;
    min-width: 0;
  }

  .summary-row span {
    color: var(--el-text-color-secondary);
    font-size: 12px;
    white-space: nowrap;
  }

  .summary-row strong {
    min-width: 0;
    color: var(--el-text-color-primary);
    font-size: 13px;
    word-break: break-all;
  }

  .detail-panels {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .detail-panel {
    padding: 16px;
    border: 1px solid var(--el-border-color-light);
    border-radius: 12px;
  }

  .panel-title {
    font-weight: 700;
  }

  .code-block {
    min-height: 280px;
    margin: 0;
    padding: 14px;
    overflow: auto;
    border-radius: 12px;
    background: #111827;
    color: #e5eefc;
    font-family: 'Consolas', 'Liberation Mono', Menlo, monospace;
    font-size: 12px;
    line-height: 1.65;
    white-space: pre-wrap;
    word-break: break-word;
  }

  .html-preview {
    overflow: hidden;
    border: 1px solid var(--el-border-color-light);
    border-radius: 12px;
    background: #fff;
  }

  .html-preview iframe {
    width: 100%;
    min-height: 420px;
    border: 0;
  }

  .dialog-footer {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
  }

  @media (max-width: 1200px) {
    .detail-summary,
    .detail-panels {
      grid-template-columns: 1fr;
    }
  }
</style>
