<template>
  <div class="realname-page">
    <!-- Header -->
    <div class="page-header">
      <div class="header-content">
        <h1 class="page-title">实名认证</h1>
        <p class="page-subtitle">完成认证后可使用全部功能</p>
      </div>
      <div class="header-actions">
        <a-button
          v-if="!isVerified && isEnabled"
          type="primary"
          @click="showVerifyModal"
          :loading="submitting"
        >
          <EditOutlined />
          {{ verification ? '重新认证' : '立即认证' }}
        </a-button>
      </div>
    </div>

    <!-- Status Banner -->
    <div class="status-banner" :class="`status-${status}`">
      <div class="status-icon">
        <SafetyOutlined v-if="isVerified" />
        <ClockCircleOutlined v-else-if="status === 'pending'" />
        <ExclamationCircleOutlined v-else-if="status === 'failed'" />
        <UserOutlined v-else />
      </div>
      <div class="status-info">
        <div class="status-title">{{ statusTitle }}</div>
        <div class="status-desc">{{ statusDescription }}</div>
      </div>
      <div class="status-action">
        <a-tag v-if="isVerified" color="success">
          <CheckCircleFilled />
          已认证
        </a-tag>
        <a-tag v-else-if="status === 'pending'" color="processing">
          <SyncOutlined spin />
          审核中
        </a-tag>
        <a-tag v-else-if="status === 'failed'" color="error">
          <CloseCircleFilled />
          未通过
        </a-tag>
        <a-tag v-else color="default">
          <InfoCircleOutlined />
          未认证
        </a-tag>
      </div>
    </div>

    <!-- Content Cards -->
    <div class="content-grid">
      <!-- Verification Info -->
      <div v-if="verification" class="info-card">
        <div class="card-header">
          <FileTextOutlined class="card-icon" />
          <h3 class="card-title">认证信息</h3>
        </div>
        <div class="info-list">
          <div class="info-item">
            <IdcardOutlined class="item-icon" />
            <span class="info-label">真实姓名</span>
            <span class="info-value">{{ maskName(verification.real_name) }}</span>
          </div>
          <div class="info-item">
            <CreditCardOutlined class="item-icon" />
            <span class="info-label">证件号码</span>
            <span class="info-value">{{ maskIdNumber(verification.id_number) }}</span>
          </div>
          <div class="info-item">
            <CalendarOutlined class="item-icon" />
            <span class="info-label">提交时间</span>
            <span class="info-value">{{ formatDate(verification.created_at) }}</span>
          </div>
          <div v-if="verification.verified_at" class="info-item">
            <CheckCircleOutlined class="item-icon success" />
            <span class="info-label">审核时间</span>
            <span class="info-value">{{ formatDate(verification.verified_at) }}</span>
          </div>
          <div v-if="verification.status === 'failed' && verification.reason" class="info-item error-item">
            <WarningOutlined class="item-icon error" />
            <span class="info-label">拒绝原因</span>
            <span class="info-value error">{{ verification.reason }}</span>
          </div>
        </div>
      </div>

      <!-- Process Steps -->
      <div v-if="isEnabled && !isVerified" class="info-card">
        <div class="card-header">
          <UnorderedListOutlined class="card-icon" />
          <h3 class="card-title">认证流程</h3>
        </div>
        <a-steps direction="vertical" :current="currentStep" size="small">
          <a-step title="填写信息" description="输入真实姓名和身份证号码" />
          <a-step title="提交审核" description="系统将验证您提交的信息" />
          <a-step title="等待结果" description="通常1-3个工作日内完成" />
          <a-step title="认证完成" description="即可使用全部功能" />
        </a-steps>
      </div>

      <!-- Security Notice -->
      <div class="info-card notice-card">
        <div class="card-header">
          <SafetyOutlined class="card-icon" />
          <h3 class="card-title">安全说明</h3>
        </div>
        <div class="notice-content">
          <p>您的信息将被严格保密，仅用于身份验证，符合《网络安全法》要求。</p>
          <p>如有疑问，请联系客服。</p>
        </div>
      </div>
    </div>

    <!-- Empty State -->
    <div v-if="!isEnabled" class="empty-state">
      <InfoCircleOutlined class="empty-icon" />
      <h3 class="empty-title">功能未启用</h3>
      <p class="empty-desc">当前系统未启用实名认证功能，如有疑问请联系管理员</p>
    </div>

    <!-- Verify Modal -->
    <a-modal
      v-model:open="verifyModalVisible"
      title="实名认证"
      @ok="handleSubmit"
      @cancel="resetForm"
      :confirm-loading="submitting"
      width="440px"
    >
      <a-alert
        type="warning"
        message="重要提示"
        description="请确保填写的信息真实有效，提交后将无法修改。虚假信息可能导致认证失败。"
        show-icon
        class="modal-alert"
      />
      <a-form
        ref="formRef"
        :model="form"
        :rules="rules"
        layout="vertical"
        class="verify-form"
      >
        <a-form-item label="真实姓名" name="real_name">
          <a-input
            v-model:value="form.real_name"
            placeholder="请输入真实姓名"
            size="large"
          />
        </a-form-item>
        <a-form-item label="身份证号码" name="id_number">
          <a-input
            v-model:value="form.id_number"
            placeholder="请输入18位身份证号码"
            size="large"
            :max-length="18"
          />
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal
      v-model:open="faceModalVisible"
      title="手机扫码完成人脸认证"
      :footer="null"
      width="420px"
      @cancel="closeFaceModal"
    >
      <div class="face-modal-body">
        <a-alert
          type="info"
          show-icon
          message="请使用手机扫码"
          description="电脑端不直接拉起人脸认证，请使用手机浏览器/微信扫码后按页面提示完成认证。"
        />
        <div class="face-qr-wrap">
          <img v-if="faceQRDataURL" :src="faceQRDataURL" alt="face-qrcode" />
          <a-empty v-else description="二维码生成失败" />
        </div>
        <div class="face-actions">
          <a-button @click="copyFaceURL" :disabled="!faceRedirectURL">复制链接</a-button>
          <a-button @click="openFaceURL" :disabled="!faceRedirectURL">在当前设备打开</a-button>
          <a-button type="primary" @click="refreshFaceStatus">我已完成，刷新状态</a-button>
        </div>
      </div>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onBeforeUnmount, onMounted } from "vue";
import { message } from "ant-design-vue";
import type { FormInstance } from "ant-design-vue";
import QRCode from "qrcode";
import {
  SafetyOutlined,
  ExclamationCircleOutlined,
  ClockCircleOutlined,
  UserOutlined,
  EditOutlined,
  FileTextOutlined,
  QuestionCircleOutlined,
  InfoCircleOutlined,
  CheckCircleFilled,
  CloseCircleFilled,
  SyncOutlined,
  IdcardOutlined,
  CreditCardOutlined,
  CalendarOutlined,
  CheckCircleOutlined,
  WarningOutlined,
  UnorderedListOutlined
} from "@ant-design/icons-vue";
import { getRealNameStatus, submitRealNameVerification } from "@/services/user";

const realname = ref<any>(null);
const loading = ref(false);
const submitting = ref(false);
const verifyModalVisible = ref(false);
const faceModalVisible = ref(false);
const faceQRDataURL = ref("");
const faceRedirectURL = ref("");
let facePollingTimer: ReturnType<typeof setInterval> | null = null;
const formRef = ref<FormInstance>();

const form = ref({
  real_name: "",
  id_number: ""
});

const rules = {
  real_name: [
    { required: true, message: "请输入真实姓名", trigger: "blur" },
    { min: 2, max: 20, message: "姓名长度应为2-20个字符", trigger: "blur" }
  ],
  id_number: [
    { required: true, message: "请输入身份证号码", trigger: "blur" },
    { pattern: /^[1-9]\d{5}(18|19|20)\d{2}(0[1-9]|1[0-2])(0[1-9]|[12]\d|3[01])\d{3}[\dXx]$/, message: "请输入正确的身份证号码", trigger: "blur" }
  ]
};

const isEnabled = computed(() => realname.value?.enabled);
const isVerified = computed(() => realname.value?.verified);
const verification = computed(() => realname.value?.verification);
const status = computed(() => {
  if (!isEnabled.value) return "disabled";
  if (isVerified.value) return "verified";
  if (verification.value?.status) return verification.value.status;
  return "unknown";
});

const currentStep = computed(() => {
  if (isVerified.value) return 3;
  if (verification.value?.status === "pending") return 2;
  return 0;
});

const statusTitle = computed(() => {
  if (!isEnabled.value) return "功能未启用";
  if (isVerified.value) return "已通过认证";
  if (verification.value?.status === "failed") return "认证未通过";
  if (verification.value?.status === "pending") return "审核中";
  return "未认证";
});

const statusDescription = computed(() => {
  if (!isEnabled.value) return "当前系统未启用实名认证功能";
  if (isVerified.value) return "您已完成实名认证，可以使用全部功能";
  if (verification.value?.status === "failed") return verification.value?.reason || "认证未通过，请重新提交";
  if (verification.value?.status === "pending") return "您的实名认证正在审核中，请耐心等待";
  return "完成实名认证后可使用更多功能";
});

const maskName = (name: string) => {
  if (!name) return "";
  if (name.length <= 2) return name.charAt(0) + "*";
  return name.charAt(0) + "*".repeat(name.length - 2) + name.charAt(name.length - 1);
};

const maskIdNumber = (idNumber: string) => {
  if (!idNumber || idNumber.length < 8) return idNumber;
  return idNumber.substring(0, 4) + "********" + idNumber.substring(idNumber.length - 4);
};

const formatDate = (date: string) => {
  if (!date) return "-";
  return new Date(date).toLocaleString("zh-CN");
};

const showVerifyModal = () => {
  verifyModalVisible.value = true;
};

const resetForm = () => {
  form.value = { real_name: "", id_number: "" };
  formRef.value?.clearValidate();
};

const handleSubmit = async () => {
  try {
    await formRef.value?.validate();
  } catch { return; }

  submitting.value = true;
  try {
    const res = await submitRealNameVerification(form.value);
    const redirectURL = String(res?.data?.redirect_url || "").trim();
    verifyModalVisible.value = false;
    resetForm();
    if (redirectURL) {
      if (isMobileUA()) {
        message.success("正在跳转到人脸认证页面");
        window.location.href = redirectURL;
        return;
      }
      await openFaceModal(redirectURL);
      message.info("请使用手机扫码完成人脸认证");
      await fetchData();
      return;
    }
    message.success("提交成功，请等待审核");
    await fetchData();
  } catch (error: any) {
    message.error(error.response?.data?.error || "提交失败");
  } finally {
    submitting.value = false;
  }
};

const fetchData = async () => {
  loading.value = true;
  try {
    const res = await getRealNameStatus();
    realname.value = res.data;
  } finally {
    loading.value = false;
  }
};

const isMobileUA = () => {
  if (typeof navigator === "undefined") return false;
  return /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Windows Phone|Mobile/i.test(navigator.userAgent || "");
};

const clearFacePolling = () => {
  if (facePollingTimer) {
    clearInterval(facePollingTimer);
    facePollingTimer = null;
  }
};

const startFacePolling = () => {
  clearFacePolling();
  facePollingTimer = setInterval(async () => {
    await fetchData();
    const st = String(realname.value?.verification?.status || "").toLowerCase();
    if (st === "verified") {
      clearFacePolling();
      faceModalVisible.value = false;
      message.success("实名认证已完成");
    }
  }, 3000);
};

const openFaceModal = async (url: string) => {
  faceRedirectURL.value = url;
  faceQRDataURL.value = await QRCode.toDataURL(url, {
    width: 220,
    margin: 1
  });
  faceModalVisible.value = true;
  startFacePolling();
};

const closeFaceModal = () => {
  faceModalVisible.value = false;
  clearFacePolling();
};

const copyFaceURL = async () => {
  if (!faceRedirectURL.value) return;
  try {
    await navigator.clipboard.writeText(faceRedirectURL.value);
    message.success("链接已复制");
  } catch {
    message.error("复制失败");
  }
};

const openFaceURL = () => {
  if (!faceRedirectURL.value) return;
  window.open(faceRedirectURL.value, "_blank");
};

const refreshFaceStatus = async () => {
  await fetchData();
  const st = String(realname.value?.verification?.status || "").toLowerCase();
  if (st === "verified") {
    closeFaceModal();
    message.success("实名认证已完成");
    return;
  }
  if (st === "failed") {
    closeFaceModal();
    message.error(realname.value?.verification?.reason || "实名认证失败");
    return;
  }
  message.info("状态仍在审核中，请稍后再试");
};

onMounted(() => {
  fetchData();
});

onBeforeUnmount(() => {
  clearFacePolling();
});
</script>

<style scoped>
.realname-page {
  padding: 24px;
  max-width: 1200px;
  margin: 0 auto;
}

/* Header */
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

/* Status Banner */
.status-banner {
  display: flex;
  align-items: center;
  gap: 20px;
  padding: 24px;
  border-radius: var(--radius-lg);
  margin-bottom: 24px;
  transition: all var(--transition-base);
}

.status-banner.status-verified {
  background: var(--success-bg);
  border: 1px solid rgba(5, 150, 105, 0.2);
}

.status-banner.status-failed {
  background: var(--danger-bg);
  border: 1px solid rgba(220, 38, 38, 0.2);
}

.status-banner.status-pending {
  background: var(--warning-bg);
  border: 1px solid rgba(217, 119, 6, 0.2);
}

.status-banner.status-disabled {
  background: var(--bg-secondary);
  border: 1px solid var(--border);
}

.status-icon {
  width: 56px;
  height: 56px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 28px;
  flex-shrink: 0;
}

.status-banner.status-verified .status-icon {
  background: var(--success);
  color: #fff;
}

.status-banner.status-failed .status-icon {
  background: var(--danger);
  color: #fff;
}

.status-banner.status-pending .status-icon {
  background: var(--warning);
  color: #fff;
}

.status-banner.status-disabled .status-icon {
  background: var(--border-dark);
  color: var(--text-tertiary);
}

.status-info {
  flex: 1;
}

.status-title {
  font-size: 18px;
  font-weight: 700;
  color: var(--text-primary);
  margin-bottom: 6px;
}

.status-desc {
  font-size: 14px;
  color: var(--text-secondary);
  line-height: 1.6;
}

.status-action :deep(.ant-tag) {
  font-weight: 600;
  padding: 6px 14px;
  font-size: 14px;
}

/* Content Grid */
.content-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 24px;
  margin-bottom: 24px;
}

/* Info Cards */
.info-card {
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  overflow: hidden;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border);
}

.card-icon {
  font-size: 18px;
  color: var(--primary);
}

.card-title {
  font-size: 16px;
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

.item-icon.error {
  color: var(--danger);
}

.info-label {
  font-size: 14px;
  color: var(--text-secondary);
  min-width: 80px;
}

.info-value {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-primary);
  margin-left: auto;
}

.info-value.error {
  color: var(--danger);
  font-weight: 600;
}

.error-item {
  background: var(--danger-bg);
  margin: 0 -16px;
  padding: 14px 16px !important;
  border-radius: var(--radius-md);
}

/* Notice Card */
.notice-card .card-header {
  background: var(--bg-secondary);
}

.notice-content {
  padding: 20px;
}

.notice-content p {
  margin: 0 0 12px 0;
  font-size: 14px;
  color: var(--text-secondary);
  line-height: 1.6;
}

.notice-content p:last-child {
  margin-bottom: 0;
}

/* Steps */
.info-card :deep(.ant-steps-vertical) {
  padding: 20px;
}

.info-card :deep(.ant-steps-item-process) .ant-steps-item-icon {
  background: var(--primary);
  border-color: var(--primary);
}

.info-card :deep(.ant-steps-item-wait .ant-steps-item-icon) {
  background: var(--bg-secondary);
  border-color: var(--border);
}

.info-card :deep(.ant-steps-item-finish .ant-steps-item-icon) {
  background: var(--success);
  border-color: var(--success);
}

/* Empty State */
.empty-state {
  text-align: center;
  padding: 80px 20px;
  background: var(--card);
  border-radius: var(--radius-lg);
  border: 1px solid var(--border);
}

.empty-icon {
  font-size: 64px;
  color: var(--border-dark);
  margin-bottom: 24px;
}

.empty-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 12px;
}

.empty-desc {
  font-size: 14px;
  color: var(--text-secondary);
  margin: 0;
}

/* Modal */
.modal-alert {
  margin-bottom: 20px;
}

.face-modal-body {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.face-qr-wrap {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 240px;
  background: #fafafa;
  border: 1px solid #f0f0f0;
  border-radius: 12px;
}

.face-qr-wrap img {
  width: 220px;
  height: 220px;
}

.face-actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
}

.verify-form {
  margin-top: 16px;
}

/* Responsive */
@media (max-width: 768px) {
  .realname-page {
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

  .status-banner {
    flex-direction: column;
    align-items: flex-start;
    gap: 16px;
    padding: 20px;
  }

  .status-icon {
    width: 48px;
    height: 48px;
    font-size: 24px;
  }

  .status-action {
    align-self: flex-start;
  }

  .content-grid {
    grid-template-columns: 1fr;
  }

  .info-item {
    flex-wrap: wrap;
    gap: 8px;
  }

  .info-value {
    margin-left: 0;
  }
}
</style>
