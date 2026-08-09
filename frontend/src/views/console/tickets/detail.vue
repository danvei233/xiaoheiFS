<template>
  <div class="ticket-detail-page pb-5" v-loading="loading">
    <div class="page-heading">
      <div class="heading-main">
        <ElButton aria-label="返回工单列表" @click="goBack">
          <ArtSvgIcon icon="ri:arrow-left-line" class="button-icon" />
          返回
        </ElButton>
        <div>
          <div class="heading-title-row">
            <h2>{{ ticketSubject }}</h2>
            <ElTag :type="statusConfig.type" effect="light">{{ statusConfig.label }}</ElTag>
          </div>
          <p>工单 #{{ ticketId }}</p>
        </div>
      </div>
    </div>

    <div v-if="ticket" class="content-grid">
      <ElCard class="art-card-xs conversation-card">
        <template #header>
          <div class="card-heading">
            <div>
              <ArtSvgIcon icon="ri:message-3-line" />
              <span>工单记录</span>
            </div>
            <ElTag type="info" effect="plain">{{ messages.length }} 条消息</ElTag>
          </div>
        </template>

        <div v-if="messages.length" class="message-list">
          <article
            v-for="(message, index) in messages"
            :key="getMessageId(message) || index"
            class="message-row"
            :class="{ 'message-row-user': !isAdminMessage(message) }"
          >
            <ElAvatar
              :size="38"
              :src="getMessageAvatar(message) || undefined"
              class="message-avatar"
            >
              <ArtSvgIcon
                :icon="isAdminMessage(message) ? 'ri:customer-service-2-line' : 'ri:user-3-line'"
              />
            </ElAvatar>
            <div class="message-column">
              <div class="message-meta">
                <span class="message-author">{{ getMessageAuthor(message) }}</span>
                <ElTag v-if="isAdminMessage(message)" size="small" type="primary" effect="plain">
                  官方
                </ElTag>
                <time>{{ formatDate(getMessageCreatedAt(message)) }}</time>
              </div>
              <div class="message-bubble">{{ getMessageContent(message) }}</div>
            </div>
          </article>
        </div>
        <ElEmpty v-else description="暂无消息" :image-size="96" />

        <div v-if="!isClosed" class="reply-section">
          <div class="reply-heading">
            <ArtSvgIcon icon="ri:reply-line" />
            <span>回复工单</span>
          </div>
          <ElInput
            v-model="replyContent"
            type="textarea"
            placeholder="请输入您的回复内容..."
            :rows="5"
            :maxlength="INPUT_LIMITS.TICKET_CONTENT"
            show-word-limit
            resize="vertical"
          />
          <div class="reply-actions">
            <ElButton v-if="ticketStatus === 'open'" :loading="closing" @click="closeCurrentTicket">
              <ArtSvgIcon icon="ri:close-circle-line" class="button-icon" />
              关闭工单
            </ElButton>
            <ElButton
              type="primary"
              :loading="replying"
              :disabled="!replyContent.trim()"
              @click="sendReply"
            >
              <ArtSvgIcon icon="ri:send-plane-2-line" class="button-icon" />
              发送回复
            </ElButton>
          </div>
        </div>
        <ElAlert
          v-else
          title="工单已关闭"
          description="此工单已被关闭，如需继续咨询请创建新工单"
          type="info"
          show-icon
          :closable="false"
          class="closed-alert"
        />
      </ElCard>

      <aside class="sidebar">
        <ElCard class="art-card-xs">
          <template #header>
            <div class="card-heading">
              <div>
                <ArtSvgIcon icon="ri:file-list-3-line" />
                <span>工单信息</span>
              </div>
            </div>
          </template>
          <ElDescriptions :column="1" border>
            <ElDescriptionsItem label="工单ID">#{{ ticketId }}</ElDescriptionsItem>
            <ElDescriptionsItem label="状态">
              <ElTag :type="statusConfig.type" size="small">{{ statusConfig.label }}</ElTag>
            </ElDescriptionsItem>
            <ElDescriptionsItem label="创建时间">{{
              formatDate(ticketCreatedAt)
            }}</ElDescriptionsItem>
            <ElDescriptionsItem label="最后更新">{{
              formatDate(ticketUpdatedAt)
            }}</ElDescriptionsItem>
          </ElDescriptions>
        </ElCard>

        <ElCard class="art-card-xs">
          <template #header>
            <div class="card-heading">
              <div>
                <ArtSvgIcon icon="ri:server-line" />
                <span>相关资源</span>
              </div>
              <ElTag v-if="resources.length" type="info" effect="plain">{{
                resources.length
              }}</ElTag>
            </div>
          </template>
          <div v-if="resources.length" class="resource-list">
            <RouterLink
              v-for="(resource, index) in resources"
              :key="getResourceId(resource) || index"
              :to="getResourceLink(resource)"
              class="resource-item"
            >
              <div class="resource-icon">
                <ArtSvgIcon icon="ri:server-line" />
              </div>
              <div class="resource-copy">
                <span>{{ getResourceName(resource) }}</span>
                <small>{{ getResourceType(resource) || 'resource' }}</small>
              </div>
              <ArtSvgIcon icon="ri:arrow-right-s-line" class="resource-arrow" />
            </RouterLink>
          </div>
          <ElEmpty v-else description="无关联资源" :image-size="80" />
        </ElCard>
      </aside>
    </div>
  </div>
</template>

<script setup lang="ts">
  import type { TagProps } from 'element-plus'
  import { addTicketMessage, closeTicket, getTicketDetail } from '@/services/user'
  import type { Ticket, TicketMessage, TicketResource } from '@/services/types'
  import { useAuthStore } from '@/stores/auth'
  import { INPUT_LIMITS } from '@/constants/inputLimits'

  defineOptions({ name: 'ConsoleTicketDetail' })

  type TagType = TagProps['type']

  interface TicketRecord extends Ticket {
    ID?: number
    Subject?: string
    Status?: string
    CreatedAt?: string
    UpdatedAt?: string
  }

  interface MessageRecord extends TicketMessage {
    ID?: number
    SenderRole?: string
    SenderName?: string
    SenderAvatar?: string
    SenderQQ?: string
    Role?: string
    Content?: string
    CreatedAt?: string
  }

  interface ResourceRecord extends TicketResource {
    ID?: number
    ResourceType?: string
    ResourceID?: number
    ResourceName?: string
  }

  interface TicketDetailPayload {
    ticket?: TicketRecord
    Ticket?: TicketRecord
    messages?: MessageRecord[]
    Messages?: MessageRecord[]
    resources?: ResourceRecord[]
    Resources?: ResourceRecord[]
  }

  interface RequestError {
    response?: {
      data?: {
        error?: unknown
        message?: unknown
      }
    }
    message?: unknown
  }

  const route = useRoute()
  const router = useRouter()
  const auth = useAuthStore()
  const loading = ref(false)
  const replying = ref(false)
  const closing = ref(false)
  const replyContent = ref('')
  const ticket = ref<TicketRecord>()
  const messages = ref<MessageRecord[]>([])
  const resources = ref<ResourceRecord[]>([])

  const getTicketId = (record?: TicketRecord): number | string => record?.id ?? record?.ID ?? '-'
  const getTicketSubject = (record?: TicketRecord): string =>
    record?.subject ?? record?.Subject ?? '工单详情'
  const getTicketStatus = (record?: TicketRecord): string => record?.status ?? record?.Status ?? ''
  const getTicketCreatedAt = (record?: TicketRecord): string =>
    record?.created_at ?? record?.CreatedAt ?? ''
  const getTicketUpdatedAt = (record?: TicketRecord): string =>
    record?.updated_at ?? record?.UpdatedAt ?? ''

  const ticketId = computed(() => getTicketId(ticket.value))
  const ticketSubject = computed(() => getTicketSubject(ticket.value))
  const ticketStatus = computed(() => getTicketStatus(ticket.value))
  const ticketCreatedAt = computed(() => getTicketCreatedAt(ticket.value))
  const ticketUpdatedAt = computed(() => getTicketUpdatedAt(ticket.value))
  const isClosed = computed(() => ticketStatus.value === 'closed')

  const STATUS_CONFIG: Record<string, { label: string; type: TagType }> = {
    open: { label: '待处理', type: 'primary' },
    waiting_user: { label: '等待回复', type: 'warning' },
    waiting_admin: { label: '处理中', type: 'warning' },
    closed: { label: '已关闭', type: 'info' }
  }
  const statusConfig = computed<{ label: string; type: TagType }>(
    () => STATUS_CONFIG[ticketStatus.value] ?? { label: ticketStatus.value || '未知', type: 'info' }
  )

  const getMessageId = (message: MessageRecord): number | string => message.id ?? message.ID ?? ''
  const getMessageRole = (message: MessageRecord): string =>
    message.sender_role ?? message.SenderRole ?? message.role ?? message.Role ?? ''
  const getMessageAuthorName = (message: MessageRecord): string =>
    message.sender_name ?? message.SenderName ?? ''
  const getMessageContent = (message: MessageRecord): string =>
    message.content ?? message.Content ?? ''
  const getMessageCreatedAt = (message: MessageRecord): string =>
    message.created_at ?? message.CreatedAt ?? ''
  const getMessageSenderQQ = (message: MessageRecord): string =>
    message.sender_qq ?? message.SenderQQ ?? ''
  const getMessageSenderAvatar = (message: MessageRecord): string =>
    message.sender_avatar ?? message.SenderAvatar ?? ''
  const isAdminMessage = (message: MessageRecord): boolean => getMessageRole(message) === 'admin'

  const getQqAvatar = (qq: string): string =>
    qq ? `https://q1.qlogo.cn/g?b=qq&nk=${encodeURIComponent(qq)}&s=100` : ''

  const getMessageAvatar = (message: MessageRecord): string => {
    const avatar = getMessageSenderAvatar(message)
    if (avatar) return avatar
    if (isAdminMessage(message)) return ''
    const qq = getMessageSenderQQ(message) || String(auth.profile?.qq ?? '')
    return getQqAvatar(qq)
  }

  const getMessageAuthor = (message: MessageRecord): string => {
    if (isAdminMessage(message)) return getMessageAuthorName(message) || 'Support Team'
    return getMessageAuthorName(message) || auth.profile?.username || '您'
  }

  const getResourceId = (resource: ResourceRecord): number | string =>
    resource.id ?? resource.ID ?? ''
  const getResourceType = (resource: ResourceRecord): string =>
    resource.resource_type ?? resource.ResourceType ?? ''
  const getResourceTargetId = (resource: ResourceRecord): number | string =>
    resource.resource_id ?? resource.ResourceID ?? ''
  const getResourceName = (resource: ResourceRecord): string =>
    resource.resource_name ?? resource.ResourceName ?? '-'
  const getResourceLink = (resource: ResourceRecord): string =>
    `/console/vps/${getResourceTargetId(resource)}`

  const formatDate = (value: string): string => {
    if (!value) return '-'
    const date = new Date(value)
    if (Number.isNaN(date.getTime())) return value
    return date.toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit'
    })
  }

  const getRequestErrorMessage = (error: unknown, fallback: string): string => {
    if (!error || typeof error !== 'object') return fallback
    const requestError = error as RequestError
    const message = requestError.response?.data?.error ?? requestError.response?.data?.message
    if (typeof message === 'string' && message.trim()) return message
    return typeof requestError.message === 'string' && requestError.message.trim()
      ? requestError.message
      : fallback
  }

  const unwrapPayload = (value: unknown): TicketDetailPayload => {
    if (!value || typeof value !== 'object') return {}
    const payload = value as Record<string, unknown>
    if (payload.data && typeof payload.data === 'object') {
      return payload.data as TicketDetailPayload
    }
    return payload as TicketDetailPayload
  }

  const fetchTicket = async (): Promise<void> => {
    const id = String(route.params.id ?? '')
    if (!id) return
    const showPageLoading = !ticket.value
    if (showPageLoading) loading.value = true
    try {
      const response = await getTicketDetail(id)
      const payload = unwrapPayload(response.data)
      ticket.value = payload.ticket ?? payload.Ticket
      messages.value = payload.messages ?? payload.Messages ?? []
      resources.value = payload.resources ?? payload.Resources ?? []
    } finally {
      if (showPageLoading) loading.value = false
    }
  }

  const goBack = (): void => {
    void router.push('/console/tickets')
  }

  const sendReply = async (): Promise<void> => {
    if (!replyContent.value.trim()) {
      ElMessage.warning('请输入回复内容')
      return
    }
    if (replyContent.value.length > INPUT_LIMITS.TICKET_CONTENT) {
      ElMessage.error(`回复长度不能超过 ${INPUT_LIMITS.TICKET_CONTENT} 个字符`)
      return
    }

    replying.value = true
    try {
      await addTicketMessage(String(route.params.id ?? ''), { content: replyContent.value })
      replyContent.value = ''
      ElMessage.success('回复成功')
      await fetchTicket()
    } catch (error) {
      ElMessage.error(getRequestErrorMessage(error, '回复失败'))
    } finally {
      replying.value = false
    }
  }

  const closeCurrentTicket = async (): Promise<void> => {
    closing.value = true
    try {
      await closeTicket(String(route.params.id ?? ''))
      ElMessage.success('工单已关闭')
      await fetchTicket()
    } catch (error) {
      ElMessage.error(getRequestErrorMessage(error, '关闭失败'))
    } finally {
      closing.value = false
    }
  }

  watch(
    () => route.params.id,
    () => {
      ticket.value = undefined
      messages.value = []
      resources.value = []
      replyContent.value = ''
      void fetchTicket()
    },
    { immediate: true }
  )
</script>

<style lang="scss" scoped>
  .page-heading {
    display: flex;
    gap: 16px;
    align-items: flex-start;
    justify-content: space-between;
    margin-bottom: 16px;
  }

  .heading-main {
    display: flex;
    gap: 12px;
    align-items: center;
    min-width: 0;

    h2 {
      max-width: min(720px, 70vw);
      margin: 0;
      overflow: hidden;
      font-size: 21px;
      color: var(--art-gray-900);
      text-overflow: ellipsis;
      letter-spacing: 0;
      white-space: nowrap;
    }

    p {
      margin: 5px 0 0;
      font-size: 12px;
      color: var(--art-gray-600);
    }
  }

  .heading-title-row {
    display: flex;
    flex-wrap: wrap;
    gap: 10px;
    align-items: center;
  }

  .button-icon {
    margin-right: 6px;
  }

  .content-grid {
    display: grid;
    grid-template-columns: minmax(0, 1fr) 320px;
    gap: 16px;
    align-items: start;
  }

  .conversation-card {
    min-width: 0;
  }

  .card-heading {
    display: flex;
    gap: 12px;
    align-items: center;
    justify-content: space-between;

    > div {
      display: flex;
      gap: 8px;
      align-items: center;
      font-weight: 600;
      color: var(--art-gray-900);

      > :first-child {
        color: var(--theme-color);
      }
    }
  }

  .message-list {
    display: flex;
    flex-direction: column;
    gap: 18px;
    padding: 4px 4px 18px;
  }

  .message-row {
    display: flex;
    gap: 10px;
    align-items: flex-start;
    width: min(82%, 720px);
  }

  .message-row-user {
    flex-direction: row-reverse;
    align-self: flex-end;

    .message-column {
      align-items: flex-end;
    }

    .message-meta {
      flex-direction: row-reverse;
    }

    .message-bubble {
      background: var(--art-active-color);
      border-color: var(--default-border);
      border-top-left-radius: calc(var(--custom-radius) / 2 + 4px);
      border-top-right-radius: 3px;
    }
  }

  .message-avatar {
    flex: none;
    color: var(--theme-color);
    background: var(--art-active-color);
  }

  .message-column {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    min-width: 0;
  }

  .message-meta {
    display: flex;
    flex-wrap: wrap;
    gap: 7px;
    align-items: center;
    margin-bottom: 6px;

    time {
      font-size: 11px;
      color: var(--art-gray-600);
    }
  }

  .message-author {
    font-size: 13px;
    font-weight: 600;
    color: var(--art-gray-900);
  }

  .message-bubble {
    max-width: 100%;
    padding: 11px 13px;
    line-height: 1.65;
    color: var(--art-gray-900);
    overflow-wrap: anywhere;
    white-space: pre-wrap;
    background: var(--el-color-primary-light-9);
    border: 1px solid var(--el-color-primary-light-7);
    border-radius: calc(var(--custom-radius) / 2 + 4px);
    border-top-left-radius: 3px;
  }

  .reply-section {
    padding-top: 18px;
    margin-top: 6px;
    border-top: 1px solid var(--default-border);
  }

  .reply-heading {
    display: flex;
    gap: 7px;
    align-items: center;
    margin-bottom: 10px;
    font-size: 13px;
    font-weight: 600;
    color: var(--art-gray-800);
  }

  .reply-actions {
    display: flex;
    gap: 10px;
    justify-content: flex-end;
    margin-top: 12px;
  }

  .closed-alert {
    margin-top: 18px;
  }

  .sidebar {
    position: sticky;
    top: 16px;
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .resource-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .resource-item {
    display: flex;
    gap: 10px;
    align-items: center;
    padding: 10px;
    color: var(--art-gray-800);
    text-decoration: none;
    background: var(--default-bg-color);
    border: 1px solid var(--default-border);
    border-radius: calc(var(--custom-radius) / 2 + 2px);
    transition:
      background-color 0.2s ease,
      border-color 0.2s ease;

    &:hover {
      background: var(--art-hover-color);
      border-color: var(--theme-color);
    }
  }

  .resource-icon {
    display: grid;
    flex: none;
    place-items: center;
    width: 34px;
    height: 34px;
    color: var(--theme-color);
    background: var(--el-color-primary-light-9);
    border-radius: calc(var(--custom-radius) / 3 + 2px);
  }

  .resource-copy {
    display: flex;
    flex: 1;
    flex-direction: column;
    min-width: 0;

    span {
      overflow: hidden;
      font-size: 13px;
      font-weight: 600;
      color: var(--art-gray-900);
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    small {
      margin-top: 2px;
      color: var(--art-gray-600);
    }
  }

  .resource-arrow {
    flex: none;
    color: var(--art-gray-500);
  }

  @media (width <= 960px) {
    .content-grid {
      grid-template-columns: 1fr;
    }

    .sidebar {
      position: static;
    }
  }

  @media (width <= 640px) {
    .page-heading {
      align-items: center;
    }

    .heading-main h2 {
      max-width: 58vw;
      font-size: 18px;
    }

    .message-row {
      width: 94%;
    }

    .message-avatar {
      width: 32px;
      height: 32px;
    }

    .reply-actions {
      flex-direction: column;
    }
  }
</style>
