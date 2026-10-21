<template>
  <div class="ticket-detail-page">
    <!-- Page Header -->
    <div class="page-header">
      <div class="header-left">
        <a-button @click="goBack" class="back-btn">
          <ArrowLeftOutlined />
          返回
        </a-button>
        <div class="header-title-section">
          <div class="title-row">
            <CustomerServiceOutlined class="title-icon" />
            <h1 class="page-title">{{ ticket?.subject || "工单详情" }}</h1>
          </div>
        </div>
      </div>
      <div class="header-actions">
        <a-tag v-if="ticket?.status === 'open'" color="processing" class="status-tag-large">
          <ClockCircleOutlined />
          待处理
        </a-tag>
        <a-tag v-else-if="ticket?.status === 'waiting_user'" color="warning" class="status-tag-large">
          <ExclamationCircleOutlined />
          等待回复
        </a-tag>
        <a-tag v-else-if="ticket?.status === 'waiting_admin'" color="purple" class="status-tag-large">
          <SyncOutlined spin />
          处理中
        </a-tag>
        <a-tag v-else-if="ticket?.status === 'closed'" color="default" class="status-tag-large">
          <CheckCircleOutlined />
          已关闭
        </a-tag>
        <a-tag v-else class="status-tag-large">
          {{ ticket?.status }}
        </a-tag>
      </div>
    </div>

    <!-- Main Content -->
    <div class="content-grid">
      <!-- Messages Section -->
      <div class="messages-section">
        <div class="section-card">
          <div class="card-header">
            <CommentOutlined class="card-icon" />
            <h3 class="card-title">工单记录</h3>
            <a-tag class="message-count">{{ messages.length }} 条消息</a-tag>
          </div>

          <a-spin :spinning="loading" class="messages-spin">
            <!-- Messages List -->
            <div v-if="messages.length > 0" class="messages-list">
              <div
                v-for="(item, index) in messages"
                :key="item.id || index"
                class="message-item"
                :class="{ 'message-admin': item.sender_role === 'admin' }"
              >
                <div class="message-avatar">
                  <a-avatar :size="44" :src="getMessageAvatar(item)">
                    <template #icon>
                      <CustomerServiceOutlined v-if="item.sender_role === 'admin'" />
                      <UserOutlined v-else />
                    </template>
                  </a-avatar>
                  <div v-if="item.sender_role === 'admin'" class="admin-badge">
                    <CustomerServiceOutlined />
                  </div>
                </div>
                <div class="message-content-wrapper">
                  <div class="message-header">
                    <span class="message-author">{{ getUserName(item) }}</span>
                    <a-tag v-if="item.sender_role === 'admin'" size="small" color="blue" class="official-tag">
                      <CustomerServiceOutlined />
                      官方
                    </a-tag>
                    <span class="message-time">{{ formatDate(item.created_at) }}</span>
                  </div>
                  <div class="message-body">{{ item.content }}</div>
                </div>
              </div>
            </div>

            <a-empty v-else description="暂无消息" class="empty-messages" />

            <!-- Reply Section -->
            <div v-if="ticket?.status !== 'closed'" class="reply-section">
              <a-divider class="reply-divider">
                <EditOutlined class="divider-icon" />
                回复工单
              </a-divider>
              <div class="reply-box">
                <a-textarea
                  v-model:value="replyContent"
                  placeholder="请输入您的回复内容..."
                  :rows="5"
                  :max-length="INPUT_LIMITS.TICKET_CONTENT"
                  show-count
                  size="large"
                  class="reply-textarea"
                />
                <div class="reply-actions">
                  <a-button
                    v-if="ticket?.status === 'open'"
                    @click="handleClose"
                    size="large"
                  >
                    <CloseCircleOutlined />
                    关闭工单
                  </a-button>
                  <a-button
                    type="primary"
                    @click="handleReply"
                    :loading="replying"
                    :disabled="!replyContent.trim()"
                    size="large"
                  >
                    <SendOutlined />
                    发送回复
                  </a-button>
                </div>
              </div>
            </div>

            <!-- Closed Alert -->
            <a-alert
              v-else
              type="info"
              show-icon
              class="closed-alert"
            >
              <template #icon>
                <CheckCircleOutlined />
              </template>
              <template #message>
                <span class="alert-title">工单已关闭</span>
              </template>
              <template #description>
                此工单已被关闭，如需继续咨询请创建新工单
              </template>
            </a-alert>
          </a-spin>
        </div>
      </div>

      <!-- Sidebar -->
      <div class="sidebar-section">
        <!-- Ticket Info -->
        <div class="sidebar-card">
          <div class="card-header">
            <FileTextOutlined class="card-icon" />
            <h3 class="card-title">工单信息</h3>
          </div>
          <div class="info-list">
            <div class="info-item">
              <IdcardOutlined class="item-icon" />
              <span class="info-label">工单ID</span>
              <span class="info-value">#{{ ticket?.id }}</span>
            </div>
            <div class="info-item">
              <ClockCircleOutlined class="item-icon" />
              <span class="info-label">状态</span>
              <a-tag v-if="ticket?.status === 'open'" color="processing" size="small">
                <ClockCircleOutlined />
                待处理
              </a-tag>
              <a-tag v-else-if="ticket?.status === 'waiting_user'" color="warning" size="small">
                <ExclamationCircleOutlined />
                等待回复
              </a-tag>
              <a-tag v-else-if="ticket?.status === 'waiting_admin'" color="purple" size="small">
                <SyncOutlined spin />
                处理中
              </a-tag>
              <a-tag v-else-if="ticket?.status === 'closed'" color="default" size="small">
                <CheckCircleOutlined />
                已关闭
              </a-tag>
              <a-tag v-else size="small">
                {{ ticket?.status }}
              </a-tag>
            </div>
            <div class="info-item">
              <CalendarOutlined class="item-icon" />
              <span class="info-label">创建时间</span>
              <span class="info-value">{{ formatFullDate(ticket?.created_at) }}</span>
            </div>
            <div class="info-item">
              <SyncOutlined class="item-icon spin" />
              <span class="info-label">最后更新</span>
              <span class="info-value">{{ formatFullDate(ticket?.updated_at) }}</span>
            </div>
          </div>
        </div>

        <!-- Resources -->
        <div class="sidebar-card">
          <div class="card-header">
            <CloudServerOutlined class="card-icon" />
            <h3 class="card-title">相关资源</h3>
            <a-tag v-if="resources.length > 0" class="resource-count">{{ resources.length }}</a-tag>
          </div>
          <div v-if="resources.length > 0" class="resources-list">
            <router-link
              v-for="(item, index) in resources"
              :key="index"
              :to="`/console/vps/${item.resource_id}`"
              class="resource-item"
            >
              <CloudServerOutlined class="resource-icon" />
              <div class="resource-info">
                <div class="resource-name">{{ item.resource_name }}</div>
                <div class="resource-type">{{ item.resource_type }}</div>
              </div>
              <ArrowRightOutlined class="resource-arrow" />
            </router-link>
          </div>
          <a-empty v-else description="无关联资源" :image="false" class="empty-resources">
            <template #description>
              <span class="empty-text">无关联资源</span>
            </template>
          </a-empty>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import { message } from "ant-design-vue";
import {
  ArrowLeftOutlined,
  ArrowRightOutlined,
  CloudServerOutlined,
  SendOutlined,
  UserOutlined,
  CustomerServiceOutlined,
  CommentOutlined,
  EditOutlined,
  CloseCircleOutlined,
  FileTextOutlined,
  IdcardOutlined,
  ClockCircleOutlined,
  CalendarOutlined,
  SyncOutlined,
  CheckCircleOutlined,
  ExclamationCircleOutlined
} from "@ant-design/icons-vue";
import { getTicketDetail, addTicketMessage, closeTicket } from "@/services/user";
import { useAuthStore } from "@/stores/auth";
import { INPUT_LIMITS } from "@/constants/inputLimits";

const route = useRoute();
const router = useRouter();
const auth = useAuthStore();

const loading = ref(false);
const replying = ref(false);
const ticket = ref<any>(null);
const messages = ref<any[]>([]);
const resources = ref<any[]>([]);
const replyContent = ref("");

// Helper function to get QQ avatar URL
const getQQAvatar = (qq?: string) => {
  if (!qq) return "";
  return `https://q1.qlogo.cn/g?b=qq&nk=${qq}&s=100`;
};

// Get current user's QQ avatar
const currentUserAvatar = computed(() => {
  return getQQAvatar(auth.profile?.qq);
});

// Get display name for user
const getUserName = (msg: any) => {
  if (msg.sender_role === 'admin') {
    return 'Support Team';
  }
  return msg.sender_name || auth.profile?.username || '您';
};

// Get avatar for message sender
const getMessageAvatar = (msg: any) => {
  if (msg.sender_role === 'admin') {
    // Admin avatar - use a professional support avatar
    return 'https://api.dicebear.com/7.x/initials/svg?seed=Support&backgroundColor=0066FF&textColor=fff';
  }
  // User avatar - use QQ avatar if available
  const qqAvatar = currentUserAvatar.value;
  return qqAvatar || undefined;
};

const formatDate = (date: string) => {
  if (!date) return "-";
  return new Date(date).toLocaleString("zh-CN");
};

const formatFullDate = (date: string) => {
  if (!date) return "-";
  const d = new Date(date);
  const year = d.getFullYear();
  const month = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  const hours = String(d.getHours()).padStart(2, '0');
  const minutes = String(d.getMinutes()).padStart(2, '0');
  return `${year}-${month}-${day} ${hours}:${minutes}`;
};

const goBack = () => {
  router.push("/console/tickets");
};

const fetchData = async () => {
  loading.value = true;
  try {
    const id = route.params.id as string;
    const res = await getTicketDetail(id);
    ticket.value = res.data.ticket;
    messages.value = res.data.messages || [];
    resources.value = res.data.resources || [];
  } finally {
    loading.value = false;
  }
};

const handleReply = async () => {
  if (!replyContent.value.trim()) {
    message.error("请输入回复内容");
    return;
  }
  if (String(replyContent.value || "").length > INPUT_LIMITS.TICKET_CONTENT) {
    message.error(`回复长度不能超过 ${INPUT_LIMITS.TICKET_CONTENT} 个字符`);
    return;
  }

  replying.value = true;
  try {
    await addTicketMessage(route.params.id as string, { content: replyContent.value });
    message.success("回复成功");
    replyContent.value = "";
    await fetchData();
  } catch (error: any) {
    message.error(error.response?.data?.error || "回复失败");
  } finally {
    replying.value = false;
  }
};

const handleClose = async () => {
  try {
    await closeTicket(route.params.id as string);
    message.success("工单已关闭");
    await fetchData();
  } catch (error: any) {
    message.error(error.response?.data?.error || "关闭失败");
  }
};

onMounted(() => {
  fetchData();
});
</script>

<style scoped>
.ticket-detail-page {
  padding: 24px;
  max-width: 1400px;
  margin: 0 auto;
}

/* Page Header */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
  gap: 16px;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 16px;
  flex: 1;
}

.back-btn {
  height: 40px;
  padding: 0 16px;
  font-weight: 500;
}

.header-title-section {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.title-row {
  display: flex;
  align-items: center;
  gap: 10px;
}

.title-icon {
  font-size: 24px;
  color: var(--primary);
}

.page-title {
  font-size: 22px;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0;
}

.header-actions {
  flex-shrink: 0;
}

.status-tag-large {
  font-size: 14px;
  font-weight: 600;
  padding: 6px 14px;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.status-tag-large :deep(.anticon) {
  font-size: 14px;
}

/* Content Grid */
.content-grid {
  display: grid;
  grid-template-columns: 1fr 360px;
  gap: 24px;
  align-items: start;
}

/* Messages Section */
.messages-section {
  min-height: 400px;
}

.section-card {
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  overflow: hidden;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border);
  background: var(--bg-secondary);
}

.card-icon {
  font-size: 18px;
  color: var(--primary);
}

.card-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
  flex: 1;
}

.message-count {
  font-size: 12px;
  background: var(--primary-gradient);
  color: #fff;
  border: none;
  padding: 2px 10px;
}

.messages-spin {
  display: block;
}

/* Messages List */
.messages-list {
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.message-item {
  display: flex;
  gap: 14px;
  padding: 16px;
  border-radius: var(--radius-md);
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  transition: all var(--transition-base);
}

.message-item:hover {
  border-color: var(--primary);
  box-shadow: var(--shadow-sm);
}

.message-item.message-admin {
  background: linear-gradient(135deg,
    rgba(0, 102, 255, 0.05) 0%,
    rgba(0, 102, 255, 0.02) 100%);
  border-color: rgba(0, 102, 255, 0.2);
}

.message-avatar {
  position: relative;
  flex-shrink: 0;
}

.admin-badge {
  position: absolute;
  bottom: -4px;
  right: -4px;
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background: var(--primary);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 10px;
  border: 2px solid var(--card);
}

.message-content-wrapper {
  flex: 1;
  min-width: 0;
}

.message-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
  flex-wrap: wrap;
}

.message-author {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
}

.official-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  padding: 2px 8px;
  background: var(--primary-gradient);
  border: none;
  color: #fff;
}

.message-time {
  font-size: 12px;
  color: var(--text-tertiary);
  margin-left: auto;
}

.message-body {
  font-size: 14px;
  line-height: 1.6;
  color: var(--text-primary);
  white-space: pre-wrap;
  word-break: break-word;
}

/* Empty Messages */
.empty-messages {
  padding: 60px 20px;
}

/* Reply Section */
.reply-section {
  padding: 20px;
  border-top: 1px solid var(--border);
}

.reply-divider {
  margin: 0 0 20px 0;
  font-size: 13px;
  font-weight: 600;
  color: var(--text-secondary);
}

.divider-icon {
  margin-right: 6px;
}

.reply-box {
  background: var(--bg-secondary);
  padding: 16px;
  border-radius: var(--radius-md);
  border: 1px solid var(--border);
}

.reply-textarea {
  margin-bottom: 16px;
}

.reply-textarea :deep(.ant-input) {
  border-radius: var(--radius-md);
}

.reply-actions {
  display: flex;
  gap: 12px;
  justify-content: flex-end;
}

/* Closed Alert */
.closed-alert {
  margin: 20px;
}

.closed-alert :deep(.ant-alert-icon) {
  font-size: 20px;
}

.alert-title {
  font-weight: 600;
  font-size: 14px;
}

/* Sidebar */
.sidebar-section {
  display: flex;
  flex-direction: column;
  gap: 20px;
  position: sticky;
  top: 24px;
}

.sidebar-card {
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  overflow: hidden;
}

.sidebar-card .card-header {
  position: relative;
}

.resource-count {
  font-size: 11px;
  background: var(--info);
  color: #fff;
  border: none;
  padding: 2px 8px;
}

/* Info List */
.info-list {
  padding: 16px 20px;
  display: flex;
  flex-direction: column;
  gap: 0;
}

.info-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 0;
  border-bottom: 1px solid var(--border-light);
}

.info-item:last-child {
  border-bottom: none;
}

.item-icon {
  font-size: 16px;
  color: var(--text-tertiary);
  width: 20px;
  flex-shrink: 0;
}

.item-icon.spin {
  animation: spin 3s linear infinite;
}

@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

.info-label {
  font-size: 13px;
  color: var(--text-secondary);
  flex-shrink: 0;
}

.info-value {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-primary);
  margin-left: auto;
}

.info-item :deep(.ant-tag) {
  margin-left: auto;
}

/* Resources List */
.resources-list {
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.resource-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  text-decoration: none;
  transition: all var(--transition-base);
}

.resource-item:hover {
  border-color: var(--primary);
  background: rgba(0, 102, 255, 0.03);
  transform: translateX(4px);
}

.resource-icon {
  font-size: 20px;
  color: var(--primary);
  flex-shrink: 0;
}

.resource-info {
  flex: 1;
  min-width: 0;
}

.resource-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.resource-type {
  font-size: 12px;
  color: var(--text-tertiary);
  margin-top: 2px;
}

.resource-arrow {
  font-size: 14px;
  color: var(--text-tertiary);
  flex-shrink: 0;
}

.resource-item:hover .resource-arrow {
  color: var(--primary);
}

.empty-resources {
  padding: 40px 20px;
}

.empty-text {
  font-size: 13px;
  color: var(--text-secondary);
}

/* Responsive */
@media (max-width: 1024px) {
  .content-grid {
    grid-template-columns: 1fr;
  }

  .sidebar-section {
    position: static;
  }
}

@media (max-width: 768px) {
  .ticket-detail-page {
    padding: 16px;
  }

  .page-header {
    flex-direction: column;
    align-items: stretch;
    gap: 16px;
  }

  .header-left {
    flex-direction: column;
    align-items: stretch;
    gap: 12px;
  }

  .title-row {
    flex-wrap: wrap;
  }

  .page-title {
    font-size: 18px;
  }

  .message-item {
    flex-direction: column;
    gap: 12px;
  }

  .message-avatar {
    align-self: flex-start;
  }

  .message-time {
    margin-left: 0;
  }

  .reply-actions {
    flex-direction: column;
  }

  .reply-actions :deep(.ant-btn) {
    width: 100%;
  }
}
</style>
