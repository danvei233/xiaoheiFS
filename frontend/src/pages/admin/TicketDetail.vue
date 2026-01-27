<template>
  <div class="admin-ticket-detail-page">
    <!-- Page Header -->
    <div class="page-header">
      <div class="header-main">
        <div class="title-section">
          <h1 class="ticket-title">{{ ticket?.subject || "工单详情" }}</h1>
          <div class="title-meta-row">
            <div :class="['status-badge', `status-${ticket?.status}`]">
              <span class="status-dot"></span>
              <span class="status-text">{{ getStatusText(ticket?.status) }}</span>
            </div>
            <div class="ticket-meta">
              <span class="meta-item">
                <span class="meta-label">工单编号</span>
                <span class="meta-value mono">#{{ ticket?.id }}</span>
              </span>
              <span class="meta-separator">•</span>
              <span class="meta-item">
                <span class="meta-label">用户ID</span>
                <span class="meta-value mono">{{ ticket?.user_id }}</span>
              </span>
              <span class="meta-separator">•</span>
              <span class="meta-item">
                <span class="meta-label">创建于</span>
                <span class="meta-value">{{ formatRelativeTime(ticket?.created_at) }}</span>
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Main Content Grid -->
    <div class="content-grid">
      <!-- Messages Panel -->
      <div class="messages-panel">
        <a-spin :spinning="loading" size="large">
          <div class="messages-container">
            <!-- Messages -->
            <div v-if="messages.length > 0" class="messages-list">
              <div
                v-for="(msg, index) in messages"
                :key="msg.id"
                :class="['message-item', msg.sender_role === 'admin' ? 'message-admin' : 'message-user']"
              >
                <div class="message-main">
                  <div class="message-avatar">
                    <a-avatar
                      :size="42"
                      :src="getUserAvatar(msg)"
                      :class="msg.sender_role === 'admin' ? 'avatar-admin' : 'avatar-user'"
                    >
                      <template #icon>
                        <UserOutlined v-if="msg.sender_role !== 'admin'" />
                        <span v-else>A</span>
                      </template>
                    </a-avatar>
                  </div>
                  <div class="message-body">
                    <div class="message-meta">
                      <div class="sender-info">
                        <span class="sender-name">
                          {{ msg.sender_role === 'admin' ? '我' : (msg.sender_name || '用户') }}
                        </span>
                        <span v-if="msg.sender_qq" class="sender-qq-badge">
                          <span class="qq-icon">QQ</span>
                          {{ msg.sender_qq }}
                        </span>
                      </div>
                      <span class="message-time">{{ formatMessageTime(msg.created_at) }}</span>
                    </div>
                    <div class="message-content">
                      {{ msg.content }}
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <div v-else class="empty-state">
              <div class="empty-icon">
                <MessageOutlined />
              </div>
              <h3 class="empty-title">暂无消息</h3>
              <p class="empty-description">工单还没有消息记录</p>
            </div>
          </div>

          <!-- Reply Section -->
          <div v-if="ticket?.status !== 'closed'" class="reply-section">
            <div class="reply-header">
              <span class="reply-title">回复工单</span>
              <span class="reply-hint">Enter 发送，Shift + Enter 换行</span>
            </div>
            <div class="reply-input-wrapper">
              <a-textarea
                v-model:value="replyContent"
                placeholder="输入您的回复内容..."
                :rows="4"
                :max-length="2000"
                :bordered="false"
                show-count
                class="reply-textarea"
                @keydown.ctrl="handleReply"
                @keydown.meta="handleReply"
              />
            </div>
            <div class="reply-footer">
              <div class="status-selector">
                <span class="selector-label">更新状态:</span>
                <a-select
                  v-model:value="newStatus"
                  size="large"
                  class="status-select"
                >
                  <a-select-option value="open">
                    <div class="select-option">
                      <span class="option-dot dot-open"></span>
                      <span>待处理</span>
                    </div>
                  </a-select-option>
                  <a-select-option value="waiting_user">
                    <div class="select-option">
                      <span class="option-dot dot-waiting"></span>
                      <span>等待回复</span>
                    </div>
                  </a-select-option>
                  <a-select-option value="closed">
                    <div class="select-option">
                      <span class="option-dot dot-closed"></span>
                      <span>关闭工单</span>
                    </div>
                  </a-select-option>
                </a-select>
              </div>
              <a-button
                type="primary"
                @click="handleReply"
                :loading="replying"
                :disabled="!replyContent.trim()"
                class="send-btn"
              >
                <SendOutlined />
                <span>发送回复</span>
              </a-button>
            </div>
          </div>

          <!-- Closed Notice -->
          <div v-else class="closed-notice">
            <CheckCircleOutlined class="closed-icon" />
            <div class="closed-content">
              <span class="closed-title">工单已关闭</span>
              <span class="closed-description">如需重新处理，请更改工单状态</span>
            </div>
            <a-button
              type="primary"
              @click="reopenTicket"
              :loading="replying"
              class="reopen-btn"
            >
              重新打开工单
            </a-button>
          </div>
        </a-spin>
      </div>

      <!-- Sidebar -->
      <div class="sidebar">
        <!-- Ticket Info -->
        <div class="sidebar-section">
          <div class="section-header">
            <h3 class="section-title">工单信息</h3>
          </div>
          <div class="section-body">
            <div class="info-row">
              <span class="info-label">工单编号</span>
              <span class="info-value mono">{{ ticket?.id }}</span>
            </div>
            <div class="info-row">
              <span class="info-label">用户ID</span>
              <span class="info-value mono">{{ ticket?.user_id }}</span>
            </div>
            <div class="info-row">
              <span class="info-label">当前状态</span>
              <span :class="['info-status', `status-${ticket?.status}`]">
                {{ getStatusText(ticket?.status) }}
              </span>
            </div>
            <div class="info-row">
              <span class="info-label">创建时间</span>
              <span class="info-value">{{ formatFullDate(ticket?.created_at) }}</span>
            </div>
            <div class="info-row">
              <span class="info-label">最后更新</span>
              <span class="info-value">{{ formatFullDate(ticket?.updated_at) }}</span>
            </div>
          </div>
        </div>

        <!-- Resources -->
        <div class="sidebar-section" v-if="resources.length > 0">
          <div class="section-header">
            <h3 class="section-title">相关资源</h3>
          </div>
          <div class="section-body">
            <router-link
              v-for="item in resources"
              :key="item.resource_id"
              :to="`/admin/vps?id=${item.resource_id}`"
              class="resource-link"
            >
              <div class="resource-icon">
                <CloudServerOutlined />
              </div>
              <div class="resource-info">
                <span class="resource-name">{{ item.resource_name }}</span>
                <span class="resource-type">{{ item.resource_type }}</span>
              </div>
              <ArrowRightOutlined class="resource-arrow" />
            </router-link>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import { message } from "ant-design-vue";
import {
  ArrowRightOutlined,
  MessageOutlined,
  CheckCircleOutlined,
  CloudServerOutlined,
  SendOutlined,
  UserOutlined
} from "@ant-design/icons-vue";
import { getAdminTicketDetail, addAdminTicketMessage, updateAdminTicket } from "@/services/admin";

const route = useRoute();
const router = useRouter();

const loading = ref(false);
const replying = ref(false);
const ticket = ref<any>(null);
const messages = ref<any[]>([]);
const resources = ref<any[]>([]);
const replyContent = ref("");
const newStatus = ref("waiting_user");

const getStatusColor = (status: string) => {
  switch (status) {
    case "open": return "blue";
    case "waiting_user": return "orange";
    case "waiting_admin": return "purple";
    case "closed": return "default";
    default: return "default";
  }
};

const getStatusText = (status: string) => {
  switch (status) {
    case "open": return "待处理";
    case "waiting_user": return "等待回复";
    case "waiting_admin": return "处理中";
    case "closed": return "已关闭";
    default: return status;
  }
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

const formatRelativeTime = (date: string) => {
  if (!date) return "-";
  const now = new Date();
  const then = new Date(date);
  const diff = now.getTime() - then.getTime();
  const seconds = Math.floor(diff / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);

  if (days > 7) {
    return formatFullDate(date);
  } else if (days > 0) {
    return `${days}天前`;
  } else if (hours > 0) {
    return `${hours}小时前`;
  } else if (minutes > 0) {
    return `${minutes}分钟前`;
  } else {
    return '刚刚';
  }
};

const formatMessageTime = (date: string) => {
  if (!date) return "";
  const d = new Date(date);
  const hours = String(d.getHours()).padStart(2, '0');
  const minutes = String(d.getMinutes()).padStart(2, '0');
  return `${hours}:${minutes}`;
};

const getUserAvatar = (msg: any) => {
  if (msg.sender_role === 'admin' || !msg.sender_qq) {
    return undefined;
  }
  return `https://q1.qlogo.cn/g?b=qq&nk=${msg.sender_qq}&s=100`;
};

const goBack = () => {
  router.push("/admin/tickets");
};

const fetchData = async () => {
  loading.value = true;
  try {
    const id = route.params.id as string;
    const res = await getAdminTicketDetail(id);
    ticket.value = res.data.ticket;
    messages.value = res.data.messages || [];
    resources.value = res.data.resources || [];
    newStatus.value = ticket.value.status === "open" ? "waiting_user" : ticket.value.status;
  } catch (error: any) {
    if (error?.response?.status === 404) {
      message.error("工单不存在或已被删除");
      router.push("/admin/tickets");
    } else {
      message.error(error?.response?.data?.error || "加载工单失败");
    }
  } finally {
    loading.value = false;
  }
};

const reopenTicket = async () => {
  if (!ticket.value) {
    message.error("工单信息未加载");
    return;
  }

  replying.value = true;
  try {
    await updateAdminTicket(route.params.id as string, { status: "open" });
    message.success("工单已重新打开");
    await fetchData();
  } catch (error: any) {
    message.error(error.response?.data?.error || "重新打开工单失败");
  } finally {
    replying.value = false;
  }
};

const handleReply = async () => {
  if (!ticket.value) {
    message.error("工单信息未加载");
    return;
  }
  if (!replyContent.value.trim()) {
    message.error("请输入回复内容");
    return;
  }

  replying.value = true;
  try {
    await addAdminTicketMessage(route.params.id as string, { content: replyContent.value });

    // Update status if changed
    if (newStatus.value !== ticket.value.status) {
      await updateAdminTicket(route.params.id as string, { status: newStatus.value });
    }

    message.success("回复成功");
    replyContent.value = "";
    await fetchData();
  } catch (error: any) {
    message.error(error.response?.data?.error || "回复失败");
  } finally {
    replying.value = false;
  }
};

onMounted(() => {
  fetchData();
});
</script>

<style scoped>
/* ============================================
   PROFESSIONAL ADMIN TICKET DETAIL PAGE
   Inspired by Zendesk / Linear / Slack
   ============================================ */

/* Import Google Font */
@import url('https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600&family=JetBrains+Mono:wght@400;500&display=swap');

/* CSS Variables */
.admin-ticket-detail-page {
  --color-bg: #fafafa;
  --color-surface: #ffffff;
  --color-border: #e5e7eb;
  --color-border-light: #f3f4f6;
  --color-text-primary: #111827;
  --color-text-secondary: #6b7280;
  --color-text-tertiary: #9ca3af;
  --color-primary: #8b5cf6;
  --color-primary-light: #ede9fe;
  --color-success: #10b981;
  --color-warning: #f59e0b;
  --color-danger: #ef4444;
  --color-info: #3b82f6;
  --shadow-sm: 0 1px 2px 0 rgba(0, 0, 0, 0.05);
  --shadow-md: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
  --font-sans: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
  --font-mono: 'JetBrains Mono', 'SF Mono', monospace;
  --radius-sm: 6px;
  --radius-md: 8px;
  --radius-lg: 12px;
  --radius-xl: 16px;

  min-height: 100vh;
  background: var(--color-bg);
  font-family: var(--font-sans);
}

/* ============================================
   PAGE HEADER
   ============================================ */
.page-header {
  background: var(--color-surface);
  border-bottom: 1px solid var(--color-border);
  padding: 20px 24px;
}

.header-main {
  width: 100%;
}

.title-section {
  width: 100%;
}

.ticket-title {
  margin: 0 0 16px 0;
  font-size: 26px;
  font-weight: 700;
  color: var(--color-text-primary);
  letter-spacing: -0.02em;
  line-height: 1.2;
}

.title-meta-row {
  display: flex;
  align-items: center;
  gap: 20px;
  flex-wrap: wrap;
}

.ticket-meta {
  display: flex;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
}

.meta-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
}

.meta-label {
  color: var(--color-text-tertiary);
  font-size: 12px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.meta-value {
  color: var(--color-text-secondary);
  font-weight: 500;
}

.meta-value.mono {
  font-family: var(--font-mono);
  font-size: 13px;
}

.meta-separator {
  color: var(--color-border);
  font-size: 14px;
}

/* Status Badge */
.status-badge {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  border-radius: 24px;
  font-size: 14px;
  font-weight: 600;
  background: var(--color-border-light);
  border: 1px solid var(--color-border);
  flex-shrink: 0;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: currentColor;
  animation: pulse 2s infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

.status-badge.status-open {
  background: linear-gradient(135deg, #eff6ff, #dbeafe);
  border-color: #3b82f6;
  color: #1d4ed8;
  box-shadow: 0 2px 8px rgba(59, 130, 246, 0.2);
}

.status-badge.status-waiting_user {
  background: linear-gradient(135deg, #fff7ed, #fed7aa);
  border-color: #f97316;
  color: #c2410c;
  box-shadow: 0 2px 8px rgba(249, 115, 22, 0.2);
}

.status-badge.status-waiting_admin {
  background: linear-gradient(135deg, #f5f3ff, #ddd6fe);
  border-color: #8b5cf6;
  color: #6d28d9;
  box-shadow: 0 2px 8px rgba(139, 92, 246, 0.2);
}

.status-badge.status-closed {
  background: linear-gradient(135deg, #f9fafb, #e5e7eb);
  border-color: #9ca3af;
  color: #4b5563;
  box-shadow: none;
}

.status-badge.status-closed .status-dot {
  animation: none;
}

/* ============================================
   CONTENT GRID
   ============================================ */
.content-grid {
  display: grid;
  grid-template-columns: 1fr 320px;
  gap: 0;
  max-width: 1600px;
  margin: 0 auto;
}

/* ============================================
   MESSAGES PANEL
   ============================================ */
.messages-panel {
  display: flex;
  flex-direction: column;
  border-right: 1px solid var(--color-border);
  background: var(--color-surface);
  min-height: calc(100vh - 140px);
}

.messages-container {
  flex: 1;
  padding: 24px;
  overflow-y: auto;
}

/* Messages List */
.messages-list {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.message-item {
  display: flex;
  width: 100%;
}

.message-item.message-admin {
  justify-content: flex-end;
}

.message-item.message-user {
  justify-content: flex-start;
}

.message-main {
  display: flex;
  gap: 12px;
  max-width: 75%;
}

.message-item.message-admin .message-main {
  flex-direction: row-reverse;
}

.message-item.message-user .message-main {
  flex-direction: row;
}

.message-avatar {
  flex-shrink: 0;
}

.message-avatar .avatar-admin {
  background: linear-gradient(135deg, #8b5cf6 0%, #7c3aed 100%);
  box-shadow: 0 3px 10px rgba(139, 92, 246, 0.3);
  border: 2px solid #ddd6fe;
}

.message-avatar .avatar-user {
  background: linear-gradient(135deg, #10b981 0%, #059669 100%);
  box-shadow: 0 3px 10px rgba(16, 185, 129, 0.3);
  border: 2px solid #6ee7b7;
}

.message-avatar :deep(.ant-avatar) {
  transition: all 0.2s ease;
}

.message-avatar :deep(.ant-avatar:hover) {
  transform: scale(1.08);
}

.message-body {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.message-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}

.message-item.message-admin .message-meta {
  justify-content: flex-end;
}

.message-item.message-user .message-meta {
  justify-content: flex-start;
}

.sender-info {
  display: flex;
  align-items: center;
  gap: 8px;
}

.sender-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-primary);
}

.message-item.message-admin .sender-name {
  color: #8b5cf6;
}

.message-item.message-user .sender-name {
  color: var(--color-text-primary);
}

.sender-qq-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  background: linear-gradient(135deg, #12b7f5, #00a1d6);
  color: #fff;
  font-size: 11px;
  font-weight: 500;
  padding: 3px 10px;
  border-radius: 12px;
  box-shadow: 0 2px 4px rgba(18, 183, 245, 0.2);
}

.qq-icon {
  font-size: 10px;
  font-weight: 700;
  opacity: 0.9;
}

.message-time {
  font-size: 11px;
  color: var(--color-text-tertiary);
  font-weight: 500;
}

.message-content {
  padding: 14px 18px;
  font-size: 14px;
  line-height: 1.6;
  word-break: break-word;
  white-space: pre-wrap;
  border-radius: 16px;
  position: relative;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.04);
}

.message-item.message-admin .message-content {
  background: linear-gradient(135deg, #f5f3ff 0%, #ede9fe 100%);
  border: 1px solid #ddd6fe;
  color: var(--color-text-primary);
}

.message-item.message-user .message-content {
  background: linear-gradient(135deg, #dcfce7 0%, #d1fae5 100%);
  border: 1px solid #6ee7b7;
  color: var(--color-text-primary);
}

/* Bubble arrow for admin */
.message-item.message-admin .message-content::before {
  content: '';
  position: absolute;
  right: -6px;
  top: 16px;
  width: 0;
  height: 0;
  border-top: 6px solid transparent;
  border-bottom: 6px solid transparent;
  border-left: 6px solid #ddd6fe;
}

/* Bubble arrow for user */
.message-item.message-user .message-content::before {
  content: '';
  position: absolute;
  left: -6px;
  top: 16px;
  width: 0;
  height: 0;
  border-top: 6px solid transparent;
  border-bottom: 6px solid transparent;
  border-right: 6px solid #6ee7b7;
}

/* Empty State */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 80px 20px;
  text-align: center;
}

.empty-icon {
  font-size: 48px;
  color: var(--color-border);
  margin-bottom: 16px;
}

.empty-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin: 0 0 8px 0;
}

.empty-description {
  font-size: 14px;
  color: var(--color-text-tertiary);
  margin: 0;
}

/* ============================================
   REPLY SECTION
   ============================================ */
.reply-section {
  border-top: 1px solid var(--color-border);
  padding: 16px 24px;
  background: var(--color-surface);
}

.reply-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.reply-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-primary);
}

.reply-hint {
  font-size: 12px;
  color: var(--color-text-tertiary);
}

.reply-input-wrapper {
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  overflow: hidden;
  transition: all 0.15s ease;
}

.reply-input-wrapper:focus-within {
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px rgba(139, 92, 246, 0.1);
}

.reply-textarea {
  font-family: var(--font-sans);
}

.reply-textarea :deep(.ant-input) {
  border: none;
  box-shadow: none;
  background: transparent;
  padding: 12px 16px;
  font-size: 14px;
  line-height: 1.5;
}

.reply-textarea :deep(.ant-input:focus) {
  box-shadow: none;
}

.reply-textarea :deep(.ant-input-data-count) {
  color: var(--color-text-tertiary);
  font-size: 12px;
  padding: 8px 16px;
  background: transparent;
  border-top: 1px solid var(--color-border-light);
}

.reply-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 12px;
  gap: 16px;
}

.status-selector {
  display: flex;
  align-items: center;
  gap: 12px;
}

.selector-label {
  font-size: 13px;
  font-weight: 500;
  color: var(--color-text-secondary);
  white-space: nowrap;
}

.status-select {
  min-width: 150px;
}

.status-select :deep(.ant-select-selector) {
  border-radius: var(--radius-md);
  border-color: var(--color-border);
}

.status-select :deep(.ant-select-focused .ant-select-selector) {
  border-color: var(--color-primary);
  box-shadow: 0 0 0 2px rgba(139, 92, 246, 0.1);
}

.select-option {
  display: flex;
  align-items: center;
  gap: 8px;
}

.option-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.option-dot.dot-open {
  background: #3b82f6;
}

.option-dot.dot-waiting {
  background: #f97316;
}

.option-dot.dot-closed {
  background: #6b7280;
}

.send-btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  height: 38px;
  padding: 0 20px;
  border-radius: var(--radius-md);
  font-size: 13px;
  font-weight: 500;
  background: linear-gradient(135deg, #8b5cf6, #7c3aed);
  border: none;
  box-shadow: var(--shadow-sm);
  transition: all 0.15s ease;
}

.send-btn:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: var(--shadow-md);
  background: linear-gradient(135deg, #7c3aed, #6d28d9);
}

.send-btn:disabled {
  opacity: 0.5;
  transform: none;
}

/* ============================================
   CLOSED NOTICE
   ============================================ */
.closed-notice {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 24px;
  background: linear-gradient(135deg, #f0fdf4 0%, #dcfce7 100%);
  border-top: 1px solid #bbf7d0;
}

.closed-icon {
  font-size: 24px;
  color: #10b981;
  flex-shrink: 0;
}

.closed-content {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.closed-title {
  font-size: 14px;
  font-weight: 600;
  color: #065f46;
}

.closed-description {
  font-size: 13px;
  color: #047857;
}

.reopen-btn {
  flex-shrink: 0;
  height: 36px;
  padding: 0 16px;
  font-size: 13px;
  font-weight: 500;
  background: linear-gradient(135deg, #8b5cf6, #7c3aed);
  border: none;
  box-shadow: var(--shadow-sm);
  transition: all 0.15s ease;
}

.reopen-btn:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: var(--shadow-md);
  background: linear-gradient(135deg, #7c3aed, #6d28d9);
}

/* ============================================
   SIDEBAR
   ============================================ */
.sidebar {
  background: var(--color-surface);
  padding: 24px;
  display: flex;
  flex-direction: column;
  gap: 32px;
}

.sidebar-section {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.section-header {
  padding-bottom: 12px;
  border-bottom: 1px solid var(--color-border-light);
}

.section-title {
  margin: 0;
  font-size: 12px;
  font-weight: 600;
  color: var(--color-text-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.section-body {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.info-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}

.info-label {
  font-size: 13px;
  color: var(--color-text-tertiary);
}

.info-value {
  font-size: 13px;
  color: var(--color-text-secondary);
  font-weight: 500;
  text-align: right;
}

.info-status {
  font-size: 12px;
  font-weight: 500;
  padding: 4px 10px;
  border-radius: 12px;
}

.info-status.status-open {
  background: #eff6ff;
  color: #3b82f6;
}

.info-status.status-waiting_user {
  background: #fff7ed;
  color: #f97316;
}

.info-status.status-waiting_admin {
  background: #f5f3ff;
  color: #8b5cf6;
}

.info-status.status-closed {
  background: #f3f4f6;
  color: #6b7280;
}

/* Resource Link */
.resource-link {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-md);
  text-decoration: none;
  transition: all 0.15s ease;
}

.resource-link:hover {
  background: var(--color-bg);
  border-color: var(--color-border);
  transform: translateX(2px);
}

.resource-icon {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #8b5cf6 0%, #7c3aed 100%);
  border-radius: var(--radius-sm);
  color: #fff;
  font-size: 16px;
  flex-shrink: 0;
}

.resource-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.resource-name {
  font-size: 13px;
  font-weight: 500;
  color: var(--color-text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.resource-type {
  font-size: 11px;
  color: var(--color-text-tertiary);
}

.resource-arrow {
  font-size: 12px;
  color: var(--color-text-tertiary);
  flex-shrink: 0;
}

.resource-link:hover .resource-arrow {
  color: var(--color-primary);
}

/* ============================================
   RESPONSIVE DESIGN
   ============================================ */
@media (max-width: 1024px) {
  .content-grid {
    grid-template-columns: 1fr;
  }

  .sidebar {
    border-top: 1px solid var(--color-border);
    padding: 20px;
  }

  .messages-panel {
    border-right: none;
  }
}

@media (max-width: 768px) {
  .page-header {
    padding: 16px 20px;
  }

  .title-row {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }

  .ticket-title {
    font-size: 22px;
  }

  .ticket-meta {
    gap: 12px;
  }

  .meta-separator {
    display: none;
  }

  .messages-container {
    padding: 16px;
  }

  .reply-section {
    padding: 16px;
  }

  .message-main {
    max-width: 85%;
  }

  .message-content {
    padding: 12px 14px;
    font-size: 13px;
  }

  .reply-footer {
    flex-direction: column-reverse;
    gap: 12px;
  }

  .status-selector {
    width: 100%;
    justify-content: space-between;
  }

  .status-select {
    flex: 1;
  }

  .send-btn {
    width: 100%;
    justify-content: center;
  }
}

@media (max-width: 480px) {
  .page-header {
    padding: 12px 16px;
  }

  .ticket-title {
    font-size: 20px;
  }

  .ticket-meta {
    gap: 8px;
  }

  .status-badge {
    padding: 6px 12px;
    font-size: 13px;
  }
}

/* ============================================
   SPINNER OVERRIDE
   ============================================ */
:deep(.ant-spin) {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 400px;
}

:deep(.ant-spin-dot-item) {
  background-color: var(--color-primary);
}
</style>
