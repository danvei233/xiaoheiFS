<template>
  <div class="ticket-detail-page" v-loading="loading">
    <div class="detail-header art-card">
      <div class="header-main">
        <div class="header-left">
          <ElButton text @click="goBack">
            <ArtSvgIcon icon="ri:arrow-left-line" />
            <span>返回列表</span>
          </ElButton>

          <div class="title-section">
            <h1>{{ ticket?.subject || '工单详情' }}</h1>
            <div class="title-meta">
              <ElTag :type="getStatusTagType(ticket?.status)">{{
                getStatusText(ticket?.status)
              }}</ElTag>
              <span class="meta-item">工单 #{{ ticket?.id || '-' }}</span>
              <span class="meta-item">用户 {{ ticket?.user_id || '-' }}</span>
              <span class="meta-item">{{ formatRelativeTime(ticket?.created_at) }}</span>
            </div>
          </div>
        </div>

        <div v-if="canDelete" class="header-actions">
          <ElButton type="danger" plain @click="handleDelete">删除工单</ElButton>
        </div>
      </div>
    </div>

    <div class="detail-grid">
      <div class="messages-panel art-card">
        <div class="panel-title">消息记录</div>

        <div v-if="messages.length" class="messages-list">
          <div
            v-for="message in messages"
            :key="message.id"
            :class="['message-item', message.sender_role === 'admin' ? 'is-admin' : 'is-user']"
          >
            <ElAvatar :size="42" class="message-avatar">
              {{ message.sender_role === 'admin' ? 'A' : 'U' }}
            </ElAvatar>

            <div class="message-body">
              <div class="message-meta">
                <div class="message-author">
                  <span class="author-name">
                    {{ message.sender_role === 'admin' ? '管理员' : message.sender_name || '用户' }}
                  </span>
                  <span v-if="message.sender_qq" class="author-qq">QQ {{ message.sender_qq }}</span>
                </div>
                <span class="message-time">{{ formatMessageTime(message.created_at) }}</span>
              </div>

              <div class="message-content">{{ message.content || '-' }}</div>
            </div>
          </div>
        </div>

        <ElEmpty v-else description="暂无消息" />

        <div v-if="ticket?.status !== 'closed'" class="reply-section">
          <div class="reply-head">
            <span class="reply-title">回复工单</span>
            <span class="reply-hint">支持 Ctrl/Cmd + Enter 快速发送</span>
          </div>

          <ElInput
            v-model="replyContent"
            type="textarea"
            :rows="5"
            :maxlength="INPUT_LIMITS.TICKET_CONTENT"
            show-word-limit
            placeholder="输入您的回复内容..."
            @keydown.ctrl.enter.prevent="handleReply"
            @keydown.meta.enter.prevent="handleReply"
          />

          <div class="reply-footer">
            <div class="status-box">
              <span class="status-label">更新状态</span>
              <ElSelect v-model="newStatus" style="width: 180px">
                <ElOption label="待处理" value="open" />
                <ElOption label="等待回复" value="waiting_user" />
                <ElOption label="已关闭" value="closed" />
              </ElSelect>
            </div>

            <ElButton
              type="primary"
              :loading="replying"
              :disabled="!replyContent.trim() || !canReply"
              @click="handleReply"
            >
              发送回复
            </ElButton>
          </div>
        </div>

        <div v-else class="closed-box">
          <div>
            <div class="closed-title">工单已关闭</div>
            <div class="closed-desc">如需重新处理，可以重新打开工单。</div>
          </div>

          <ElButton type="primary" :loading="replying" :disabled="!canUpdate" @click="reopenTicket">
            重新打开工单
          </ElButton>
        </div>
      </div>

      <div class="sidebar">
        <div class="art-card info-card">
          <div class="panel-title">工单信息</div>
          <ElDescriptions :column="1" border>
            <ElDescriptionsItem label="工单编号">#{{ ticket?.id || '-' }}</ElDescriptionsItem>
            <ElDescriptionsItem label="用户 ID">{{ ticket?.user_id || '-' }}</ElDescriptionsItem>
            <ElDescriptionsItem label="当前状态">
              <ElTag :type="getStatusTagType(ticket?.status)">{{
                getStatusText(ticket?.status)
              }}</ElTag>
            </ElDescriptionsItem>
            <ElDescriptionsItem label="创建时间">
              {{ formatDateTime(ticket?.created_at) }}
            </ElDescriptionsItem>
            <ElDescriptionsItem label="最后更新">
              {{ formatDateTime(ticket?.updated_at) }}
            </ElDescriptionsItem>
          </ElDescriptions>
        </div>

        <div class="art-card info-card" v-if="resources.length">
          <div class="panel-title">相关资源</div>

          <div class="resource-list">
            <div v-for="item in resources" :key="item.id" class="resource-item">
              <div class="resource-main">
                <div class="resource-name">{{ item.resource_name || '-' }}</div>
                <div class="resource-meta">
                  <span>{{ item.resource_type || '-' }}</span>
                  <span>#{{ item.resource_id || '-' }}</span>
                </div>
              </div>

              <ElButton text :disabled="!canOpenResource(item)" @click="openResource(item)">
                查看
              </ElButton>
            </div>
          </div>

          <div v-if="resources.length && !hasOpenableResource" class="resource-tip">
            关联资源详情页尚未迁移，当前先展示资源信息。
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import type {
    TicketDetailResponse,
    TicketMessageRecord,
    TicketRecord,
    TicketResourceRecord
  } from '@/api/admin'
  import {
    createAdminTicketMessage,
    deleteAdminTicket,
    fetchAdminTicketDetail,
    hasAdminPermission,
    updateAdminTicket
  } from '@/api/admin'
  import { useUserStore } from '@/store/modules/user'
  import { INPUT_LIMITS } from '@/utils/constants'
  import { ElMessage, ElMessageBox } from 'element-plus'

  defineOptions({ name: 'TicketDetail' })

  const route = useRoute()
  const router = useRouter()
  const userStore = useUserStore()
  const { info } = storeToRefs(userStore)

  const loading = ref(false)
  const replying = ref(false)
  const detail = ref<TicketDetailResponse | null>(null)
  const replyContent = ref('')
  const newStatus = ref('waiting_user')

  const ticket = computed<TicketRecord | undefined>(() => detail.value?.ticket)
  const messages = computed<TicketMessageRecord[]>(() => detail.value?.messages || [])
  const resources = computed<TicketResourceRecord[]>(() => detail.value?.resources || [])

  const canUpdate = computed(() => hasAdminPermission(info.value?.buttons, ['tickets.update']))
  const canReply = computed(() =>
    hasAdminPermission(info.value?.buttons, ['tickets.messages', 'tickets.update'])
  )
  const canDelete = computed(() => hasAdminPermission(info.value?.buttons, ['tickets.delete']))
  const hasOpenableResource = computed(() => resources.value.some((item) => canOpenResource(item)))

  onMounted(() => {
    fetchData()
  })

  async function fetchData() {
    loading.value = true

    try {
      const id = String(route.params.id || '')
      const payload = await fetchAdminTicketDetail(id)
      detail.value = payload
      newStatus.value =
        payload.ticket?.status === 'open'
          ? 'waiting_user'
          : payload.ticket?.status || 'waiting_user'
    } catch (error: any) {
      if (error?.code === 404 || error?.response?.status === 404) {
        ElMessage.error('工单不存在或已被删除')
        router.push('/tickets/list')
      } else {
        ElMessage.error(error?.message || '加载工单失败')
      }
    } finally {
      loading.value = false
    }
  }

  function getStatusText(status?: string) {
    switch (status) {
      case 'open':
        return '待处理'
      case 'waiting_user':
        return '等待回复'
      case 'waiting_admin':
        return '处理中'
      case 'closed':
        return '已关闭'
      default:
        return status || '-'
    }
  }

  function getStatusTagType(status?: string) {
    switch (status) {
      case 'open':
        return 'danger' as const
      case 'waiting_user':
        return 'warning' as const
      case 'waiting_admin':
        return 'primary' as const
      case 'closed':
        return 'info' as const
      default:
        return 'info' as const
    }
  }

  function formatDateTime(value?: string | null) {
    if (!value) {
      return '-'
    }

    const date = new Date(value)
    return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN')
  }

  function formatRelativeTime(value?: string | null) {
    if (!value) {
      return '-'
    }

    const now = Date.now()
    const target = new Date(value).getTime()
    if (Number.isNaN(target)) {
      return value
    }

    const diff = now - target
    const minutes = Math.floor(diff / 60000)
    const hours = Math.floor(diff / 3600000)
    const days = Math.floor(diff / 86400000)

    if (days > 7) {
      return formatDateTime(value)
    }
    if (days > 0) {
      return `${days}天前`
    }
    if (hours > 0) {
      return `${hours}小时前`
    }
    if (minutes > 0) {
      return `${minutes}分钟前`
    }

    return '刚刚'
  }

  function formatMessageTime(value?: string | null) {
    if (!value) {
      return '-'
    }

    const date = new Date(value)
    if (Number.isNaN(date.getTime())) {
      return value
    }

    return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', hour12: false })
  }

  function goBack() {
    router.push('/tickets/list')
  }

  async function reopenTicket() {
    const id = String(route.params.id || '')
    if (!id) {
      return
    }

    replying.value = true

    try {
      await updateAdminTicket(id, { status: 'open' })
      ElMessage.success('工单已重新打开')
      await fetchData()
    } finally {
      replying.value = false
    }
  }

  async function handleReply() {
    const id = String(route.params.id || '')
    if (!id) {
      return
    }

    if (!replyContent.value.trim()) {
      ElMessage.error('请输入回复内容')
      return
    }

    if (String(replyContent.value || '').length > INPUT_LIMITS.TICKET_CONTENT) {
      ElMessage.error(`回复长度不能超过 ${INPUT_LIMITS.TICKET_CONTENT} 个字符`)
      return
    }

    replying.value = true

    try {
      await createAdminTicketMessage(id, { content: replyContent.value })

      if (newStatus.value !== ticket.value?.status) {
        await updateAdminTicket(id, { status: newStatus.value })
      }

      ElMessage.success('回复成功')
      replyContent.value = ''
      await fetchData()
    } finally {
      replying.value = false
    }
  }

  async function handleDelete() {
    const id = String(route.params.id || '')
    if (!id) {
      return
    }

    try {
      await ElMessageBox.confirm('删除后无法恢复，确认继续吗？', '删除工单', {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        type: 'warning'
      })
    } catch {
      return
    }

    await deleteAdminTicket(id)
    ElMessage.success('工单已删除')
    router.push('/tickets/list')
  }

  function canOpenResource(item: TicketResourceRecord) {
    if (item.resource_type !== 'vps' || !item.resource_id) {
      return false
    }

    return (
      router.resolve({ path: '/vps', query: { id: String(item.resource_id) } }).matched.length > 0
    )
  }

  function openResource(item: TicketResourceRecord) {
    if (!canOpenResource(item)) {
      return
    }

    router.push({ path: '/vps', query: { id: String(item.resource_id) } })
  }
</script>

<style scoped lang="scss">
  .ticket-detail-page {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .detail-header {
    padding: 20px;
  }

  .header-main {
    display: flex;
    justify-content: space-between;
    gap: 16px;
  }

  .header-left {
    display: flex;
    flex-direction: column;
    gap: 14px;
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

  .detail-grid {
    display: grid;
    grid-template-columns: minmax(0, 2fr) minmax(280px, 1fr);
    gap: 16px;
  }

  .messages-panel,
  .info-card {
    padding: 18px;
  }

  .panel-title {
    margin-bottom: 16px;
    color: var(--el-text-color-primary);
    font-size: 16px;
    font-weight: 600;
  }

  .messages-list {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .message-item {
    display: flex;
    gap: 12px;
    padding: 14px;
    border-radius: 14px;
    border: 1px solid var(--el-border-color-light);
  }

  .message-item.is-admin {
    background: rgb(14 165 233 / 6%);
  }

  .message-item.is-user {
    background: rgb(34 197 94 / 6%);
  }

  .message-body {
    flex: 1;
    min-width: 0;
  }

  .message-meta {
    display: flex;
    justify-content: space-between;
    gap: 12px;
    margin-bottom: 8px;
  }

  .message-author {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    color: var(--el-text-color-secondary);
  }

  .author-name {
    color: var(--el-text-color-primary);
    font-weight: 600;
  }

  .author-qq,
  .message-time {
    font-size: 13px;
    color: var(--el-text-color-secondary);
  }

  .message-content {
    color: var(--el-text-color-regular);
    line-height: 1.8;
    white-space: pre-wrap;
    word-break: break-word;
  }

  .reply-section {
    margin-top: 20px;
    padding-top: 20px;
    border-top: 1px solid var(--el-border-color-light);
  }

  .reply-head {
    display: flex;
    justify-content: space-between;
    gap: 12px;
    margin-bottom: 12px;
  }

  .reply-title {
    color: var(--el-text-color-primary);
    font-weight: 600;
  }

  .reply-hint,
  .closed-desc,
  .resource-tip {
    color: var(--el-text-color-secondary);
    font-size: 13px;
  }

  .reply-footer {
    display: flex;
    justify-content: space-between;
    align-items: end;
    gap: 16px;
    margin-top: 12px;
  }

  .status-box {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .status-label {
    color: var(--el-text-color-secondary);
    font-size: 13px;
  }

  .closed-box {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 16px;
    margin-top: 20px;
    padding: 18px;
    border-radius: 12px;
    background: var(--el-fill-color-light);
  }

  .closed-title,
  .resource-name {
    color: var(--el-text-color-primary);
    font-weight: 600;
  }

  .sidebar {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .resource-list {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .resource-item {
    display: flex;
    justify-content: space-between;
    gap: 12px;
    padding: 12px;
    border-radius: 12px;
    background: var(--el-fill-color-light);
  }

  .resource-meta {
    display: flex;
    gap: 8px;
    margin-top: 4px;
    color: var(--el-text-color-secondary);
    font-size: 13px;
  }

  @media (max-width: 1024px) {
    .detail-grid {
      grid-template-columns: 1fr;
    }
  }

  @media (max-width: 768px) {
    .header-main,
    .reply-head,
    .reply-footer,
    .closed-box,
    .message-meta {
      flex-direction: column;
      align-items: flex-start;
    }
  }
</style>
