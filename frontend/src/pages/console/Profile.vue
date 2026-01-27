<template>
  <div class="profile-page">
    <!-- Page Header -->
    <div class="page-header">
      <div class="header-content">
        <h1 class="page-title">个人资料</h1>
        <p class="page-subtitle">管理您的账户信息和偏好设置</p>
      </div>
      <div class="header-actions">
        <a-button type="primary" @click="openEditModal">
          <EditOutlined />
          编辑资料
        </a-button>
      </div>
    </div>

    <!-- Profile Header Card -->
    <div class="profile-header-card">
      <div class="profile-banner">
        <div class="banner-bg"></div>
        <div class="banner-content">
          <div class="avatar-section">
            <div class="avatar-wrapper">
              <a-avatar :size="100" :src="avatarSrc" class="user-avatar">
                {{ profile?.username?.slice(0, 1) || "U" }}
              </a-avatar>
              <div class="avatar-badge">
                <CheckCircleFilled v-if="profile?.status === 'active'" class="badge-icon active" />
                <LockFilled v-else class="badge-icon inactive" />
              </div>
            </div>
          </div>
          <div class="user-section">
            <div class="user-name-row">
              <h2 class="user-name">{{ profile?.username || "用户" }}</h2>
              <div class="user-tags">
                <a-tag color="blue">{{ profile?.role || "user" }}</a-tag>
                <a-tag :color="realnameTagColor">
                  <component :is="realnameTagIcon" />
                  {{ realnameTag }}
                </a-tag>
              </div>
            </div>
            <div class="user-stats">
              <div class="stat-item">
                <WalletOutlined class="stat-icon" />
                <div class="stat-content">
                  <span class="stat-label">钱包余额</span>
                  <span class="stat-value stat-value-balance">{{ balanceText }}</span>
                </div>
              </div>
              <div class="stat-divider"></div>
              <div class="stat-item">
                <IdcardOutlined class="stat-icon" />
                <div class="stat-content">
                  <span class="stat-label">用户ID</span>
                  <span class="stat-value">#{{ profile?.id || "-" }}</span>
                </div>
              </div>
              <div class="stat-divider"></div>
              <div class="stat-item">
                <CalendarOutlined class="stat-icon" />
                <div class="stat-content">
                  <span class="stat-label">注册时间</span>
                  <span class="stat-value">{{ formatTime(profile?.created_at) }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Info Cards Grid -->
    <div class="info-grid">
      <!-- Contact Info -->
      <div class="info-card">
        <div class="card-header">
          <MailOutlined class="card-icon" />
          <h3 class="card-title">联系信息</h3>
        </div>
        <div class="info-list">
          <div class="info-item">
            <MailOutlined class="item-icon" />
            <div class="info-content">
              <span class="info-label">邮箱</span>
              <span class="info-value">{{ profile?.email || "-" }}</span>
            </div>
          </div>
          <div class="info-item">
            <QqOutlined class="item-icon" />
            <div class="info-content">
              <span class="info-label">QQ</span>
              <span class="info-value">{{ profile?.qq || "-" }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Account Status -->
      <div class="info-card">
        <div class="card-header">
          <SafetyOutlined class="card-icon" />
          <h3 class="card-title">账户状态</h3>
        </div>
        <div class="info-list">
          <div class="info-item">
            <CheckCircleOutlined class="item-icon" :class="profile?.status === 'active' ? 'success' : 'default'" />
            <div class="info-content">
              <span class="info-label">状态</span>
              <a-tag :color="profile?.status === 'active' ? 'success' : 'default'">
                {{ profile?.status === 'active' ? '正常' : (profile?.status || '-') }}
              </a-tag>
            </div>
          </div>
          <div class="info-item">
            <UserOutlined class="item-icon" />
            <div class="info-content">
              <span class="info-label">角色</span>
              <span class="info-value">{{ profile?.role || "-" }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Wallet Info -->
      <div class="info-card wallet-card">
        <div class="card-header">
          <WalletOutlined class="card-icon" />
          <h3 class="card-title">钱包信息</h3>
        </div>
        <div class="info-list">
          <div class="info-item info-item-balance">
            <TransactionOutlined class="item-icon" />
            <div class="info-content">
              <span class="info-label">账户余额</span>
              <span class="info-value info-value-balance">{{ balanceText }}</span>
            </div>
          </div>
          <div class="info-item">
            <DollarOutlined class="item-icon" />
            <div class="info-content">
              <span class="info-label">币种</span>
              <span class="info-value">{{ wallet?.currency || "CNY" }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Edit Profile Modal -->
    <a-modal
      v-model:open="editModalVisible"
      title="编辑个人资料"
      :width="500"
      @ok="handleSubmit"
      @cancel="resetForm"
      :confirm-loading="submitting"
    >
      <a-alert
        type="info"
        message="提示"
        description="更新个人资料后，相关信息将在所有页面同步更新"
        show-icon
        class="modal-alert"
      />
      <a-form ref="formRef" layout="vertical" :model="form" :rules="rules" class="edit-form">
        <a-form-item label="用户名" name="username">
          <a-input
            v-model:value="form.username"
            placeholder="请输入用户名"
            size="large"
            prefix="<UserOutlined />"
          >
            <template #prefix>
              <UserOutlined />
            </template>
          </a-input>
        </a-form-item>
        <a-form-item label="邮箱地址" name="email">
          <a-input
            v-model:value="form.email"
            placeholder="请输入邮箱地址"
            size="large"
          >
            <template #prefix>
              <MailOutlined />
            </template>
          </a-input>
        </a-form-item>
        <a-form-item label="QQ号码" name="qq">
          <a-input
            v-model:value="form.qq"
            placeholder="请输入QQ号码"
            size="large"
          >
            <template #prefix>
              <QqOutlined />
            </template>
          </a-input>
        </a-form-item>
        <a-divider class="form-divider">
          <LockOutlined class="divider-icon" />
          修改密码
        </a-divider>
        <a-form-item label="新密码" name="password">
          <a-input-password
            v-model:value="form.password"
            placeholder="留空表示不修改密码"
            size="large"
          >
            <template #prefix>
              <LockOutlined />
            </template>
          </a-input-password>
        </a-form-item>
        <a-form-item label="确认密码" name="confirmPassword" v-if="form.password">
          <a-input-password
            v-model:value="form.confirmPassword"
            placeholder="请再次输入新密码"
            size="large"
          >
            <template #prefix>
              <LockOutlined />
            </template>
          </a-input-password>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref, watch } from "vue";
import dayjs from "dayjs";
import {
  EditOutlined,
  MailOutlined,
  SafetyOutlined,
  WalletOutlined,
  CheckCircleFilled,
  LockFilled,
  IdcardOutlined,
  CalendarOutlined,
  UserOutlined,
  CheckCircleOutlined,
  QqOutlined,
  TransactionOutlined,
  DollarOutlined,
  SyncOutlined,
  CloseCircleOutlined,
  InfoCircleOutlined
} from "@ant-design/icons-vue";
import { useAuthStore } from "@/stores/auth";
import { getWallet, getRealNameStatus } from "@/services/user";
import { normalizeWallet } from "@/utils/wallet";
import { message } from "ant-design-vue";

const formatTime = (value) => {
  if (!value) return "-";
  return dayjs(value).format("YYYY-MM-DD HH:mm:ss");
};

const auth = useAuthStore();
const profile = computed(() => auth.profile);
const wallet = ref({ balance: 0, currency: "CNY" });
const realname = ref(null);
const editModalVisible = ref(false);
const submitting = ref(false);
const formRef = ref();
const form = reactive({
  username: "",
  email: "",
  qq: "",
  password: "",
  confirmPassword: ""
});

const rules = {
  username: [
    { required: true, message: "请输入用户名", trigger: "blur" },
    { min: 2, max: 20, message: "用户名长度应为2-20个字符", trigger: "blur" }
  ],
  email: [
    { type: "email", message: "请输入有效的邮箱地址", trigger: "blur" }
  ],
  confirmPassword: [
    {
      validator: (rule, value) => {
        if (form.password && value !== form.password) {
          return Promise.reject("两次输入的密码不一致");
        }
        return Promise.resolve();
      },
      trigger: "blur"
    }
  ]
};

onMounted(() => {
  if (!auth.profile) {
    auth.fetchMe();
  }
  fetchExtras();
});

watch(
  profile,
  (val) => {
    if (!val) return;
    form.username = val.username || "";
    form.email = val.email || "";
    form.qq = val.qq || "";
  },
  { immediate: true }
);

const openEditModal = () => {
  form.username = profile.value?.username || "";
  form.email = profile.value?.email || "";
  form.qq = profile.value?.qq || "";
  form.password = "";
  form.confirmPassword = "";
  editModalVisible.value = true;
};

const resetForm = () => {
  form.password = "";
  form.confirmPassword = "";
  formRef.value?.clearValidate();
};

const handleSubmit = async () => {
  try {
    await formRef.value?.validate();
  } catch { return; }

  submitting.value = true;
  try {
    const payload = {
      username: form.username,
      email: form.email,
      qq: form.qq
    };
    if (form.password) {
      payload.password = form.password;
    }
    await auth.updateProfile(payload);
    editModalVisible.value = false;
    resetForm();
    message.success("资料已更新");
  } catch (error) {
    // error handled by store
  } finally {
    submitting.value = false;
  }
};

const fetchExtras = async () => {
  try {
    const [walletRes, realnameRes] = await Promise.all([getWallet(), getRealNameStatus()]);
    wallet.value = normalizeWallet(walletRes.data) || wallet.value;
    realname.value = realnameRes.data || realname.value;
  } catch {
    // ignore
  }
};

const balanceText = computed(() => {
  const value = Number(wallet.value?.balance ?? 0);
  const currency = wallet.value?.currency || "CNY";
  if (Number.isNaN(value)) return "-";
  const prefix = currency === "CNY" ? "¥" : `${currency} `;
  return `${prefix}${value.toFixed(2)}`;
});

const realnameTag = computed(() => {
  if (realname.value?.verified) return "已认证";
  const status = realname.value?.verification?.status;
  if (status === "pending") return "审核中";
  if (status === "failed") return "未通过";
  return "未认证";
});

const realnameTagColor = computed(() => {
  if (realname.value?.verified) return "success";
  const status = realname.value?.verification?.status;
  if (status === "pending") return "processing";
  if (status === "failed") return "error";
  return "default";
});

const realnameTagIcon = computed(() => {
  if (realname.value?.verified) return CheckCircleOutlined;
  const status = realname.value?.verification?.status;
  if (status === "pending") return SyncOutlined;
  if (status === "failed") return CloseCircleOutlined;
  return InfoCircleOutlined;
});

const avatarSrc = computed(() => {
  if (profile.value?.avatar) return profile.value.avatar;
  const qq = profile.value?.qq;
  if (!qq) return "";
  return `https://q1.qlogo.cn/g?b=qq&nk=${qq}&s=100`;
});
</script>

<style scoped>
.profile-page {
  padding: 24px;
  max-width: 1200px;
  margin: 0 auto;
}

/* Page Header */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.header-content {
  display: flex;
  align-items: baseline;
  gap: 16px;
}

.page-title {
  font-size: 28px;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0;
}

.page-subtitle {
  font-size: 14px;
  color: var(--text-secondary);
  margin: 0;
}

.header-actions :deep(.ant-btn) {
  height: 40px;
  padding: 0 20px;
  font-weight: 500;
}

/* Profile Header Card */
.profile-header-card {
  margin-bottom: 24px;
  border-radius: var(--radius-lg);
  overflow: hidden;
  box-shadow: var(--shadow-lg);
}

.profile-banner {
  position: relative;
  background: var(--primary-gradient);
  border-radius: var(--radius-lg);
  overflow: hidden;
}

.banner-bg {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: linear-gradient(135deg,
    rgba(255, 255, 255, 0.1) 0%,
    rgba(255, 255, 255, 0.05) 50%,
    rgba(0, 0, 0, 0.05) 100%);
  backdrop-filter: blur(10px);
}

.banner-content {
  position: relative;
  display: flex;
  align-items: center;
  gap: 32px;
  padding: 40px;
  color: #fff;
}

.avatar-section {
  flex-shrink: 0;
}

.avatar-wrapper {
  position: relative;
  display: inline-block;
}

.user-avatar {
  border: 4px solid rgba(255, 255, 255, 0.9);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.2);
  background: rgba(255, 255, 255, 0.95);
  font-size: 42px;
  font-weight: 700;
  color: var(--primary);
}

.avatar-badge {
  position: absolute;
  bottom: 4px;
  right: 4px;
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
}

.badge-icon {
  font-size: 16px;
}

.badge-icon.active {
  color: var(--success);
}

.badge-icon.inactive {
  color: var(--text-tertiary);
}

.user-section {
  flex: 1;
}

.user-name-row {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 20px;
  flex-wrap: wrap;
}

.user-name {
  font-size: 28px;
  font-weight: 700;
  color: #fff;
  margin: 0;
  text-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.user-tags {
  display: flex;
  gap: 8px;
}

.user-tags :deep(.ant-tag) {
  background: rgba(255, 255, 255, 0.2);
  border-color: rgba(255, 255, 255, 0.3);
  color: #fff;
  backdrop-filter: blur(10px);
  font-weight: 500;
}

.user-stats {
  display: flex;
  align-items: center;
  gap: 24px;
}

.stat-item {
  display: flex;
  align-items: center;
  gap: 10px;
}

.stat-icon {
  font-size: 20px;
  color: rgba(255, 255, 255, 0.8);
}

.stat-content {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.stat-label {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.8);
}

.stat-value {
  font-size: 16px;
  font-weight: 600;
  color: #fff;
}

.stat-value-balance {
  font-size: 20px;
  font-weight: 700;
  color: #fff;
  text-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
}

.stat-divider {
  width: 1px;
  height: 32px;
  background: rgba(255, 255, 255, 0.3);
}

/* Info Cards Grid */
.info-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 24px;
}

.info-card {
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  overflow: hidden;
  transition: all var(--transition-base);
}

.info-card:hover {
  border-color: var(--primary);
  box-shadow: var(--shadow-md);
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
}

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
  padding: 14px 0;
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

.item-icon.success {
  color: var(--success);
}

.item-icon.default {
  color: var(--text-tertiary);
}

.info-content {
  flex: 1;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
}

.info-label {
  font-size: 13px;
  color: var(--text-secondary);
}

.info-value {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-primary);
}

.info-value-balance {
  font-size: 18px;
  font-weight: 700;
  color: var(--primary);
}

.wallet-card {
  border: 1px solid rgba(0, 102, 255, 0.15);
  background: linear-gradient(135deg,
    rgba(0, 102, 255, 0.03) 0%,
    rgba(0, 102, 255, 0.01) 100%);
}

.wallet-card .card-icon {
  color: var(--success);
}

.info-item-balance {
  background: var(--success-bg);
  margin: 0 -16px;
  padding: 16px !important;
  border-radius: var(--radius-md);
  border: none !important;
}

.info-item-balance .item-icon {
  color: var(--success);
  font-size: 20px;
}

/* Modal */
.modal-alert {
  margin-bottom: 20px;
}

.edit-form {
  margin-top: 16px;
}

.edit-form :deep(.ant-form-item) {
  margin-bottom: 20px;
}

.edit-form :deep(.ant-form-item-label > label) {
  font-weight: 500;
}

.form-divider {
  margin: 20px 0;
  font-size: 13px;
  font-weight: 600;
  color: var(--text-secondary);
}

.divider-icon {
  margin-right: 6px;
}

/* Responsive */
@media (max-width: 768px) {
  .profile-page {
    padding: 16px;
  }

  .page-header {
    flex-direction: column;
    align-items: stretch;
    gap: 16px;
  }

  .header-content {
    flex-direction: column;
    gap: 4px;
  }

  .page-title {
    font-size: 22px;
  }

  .banner-content {
    flex-direction: column;
    align-items: flex-start;
    gap: 20px;
    padding: 24px;
  }

  .user-name {
    font-size: 22px;
  }

  .user-stats {
    flex-direction: column;
    align-items: flex-start;
    gap: 16px;
  }

  .stat-divider {
    display: none;
  }

  .info-grid {
    grid-template-columns: 1fr;
  }

  .info-item {
    flex-wrap: wrap;
  }

  .info-content {
    flex-direction: column;
    align-items: flex-start;
    gap: 4px;
  }
}
</style>
