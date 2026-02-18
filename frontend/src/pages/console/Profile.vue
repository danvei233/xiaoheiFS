<template>
  <div class="profile-page">
    <!-- Page Header -->
    <div class="page-header">
      <div class="header-content">
        <h1 class="page-title">个人资料</h1>
        <p class="page-subtitle">管理您的账户信息和偏好设置</p>
      </div>
      <div class="header-actions">
        <a-button type="primary" @click="openProtectedFlow('edit')">
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
              <a-badge :status="profile?.status === 'active' ? 'success' : 'default'" :text="profile?.status === 'active' ? '正常' : (profile?.status || '-')" />
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

      <!-- Security Settings -->
      <div class="info-card">
        <div class="card-header">
          <LockOutlined class="card-icon" />
          <h3 class="card-title">安全设置</h3>
        </div>
        <div class="info-list">
          <div class="info-item">
            <KeyOutlined class="item-icon" />
            <div class="info-content">
              <span class="info-label">密码</span>
              <span class="info-value">已设置</span>
            </div>
            <a-button type="text" size="small" @click="openProtectedFlow('password')">修改</a-button>
          </div>
          <div class="info-item">
            <SafetyOutlined class="item-icon" />
            <div class="info-content">
              <span class="info-label">双因素认证</span>
              <a-badge :status="twoFAEnabled ? 'success' : 'default'" :text="twoFAEnabled ? '已启用' : '未启用'" />
            </div>
            <a-button type="text" size="small" @click="openProtectedFlow('twofa')">设置</a-button>
          </div>
          <div class="info-item">
            <MailOutlined class="item-icon" />
            <div class="info-content">
              <span class="info-label">邮箱绑定</span>
              <a-badge :status="securityContacts.email_bound ? 'success' : 'default'" :text="securityContacts.email_bound ? '已绑定' : '未绑定'" />
            </div>
            <a-button type="text" size="small" @click="openProtectedFlow('email')">{{ securityContacts.email_bound ? '更新' : '绑定' }}</a-button>
          </div>
          <div class="info-item">
            <PhoneOutlined class="item-icon" />
            <div class="info-content">
              <span class="info-label">手机绑定</span>
              <a-badge :status="securityContacts.phone_bound ? 'success' : 'default'" :text="securityContacts.phone_bound ? '已绑定' : '未绑定'" />
            </div>
            <a-button type="text" size="small" @click="openProtectedFlow('phone')">{{ securityContacts.phone_bound ? '更新' : '绑定' }}</a-button>
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
            :maxlength="INPUT_LIMITS.USERNAME"
          >
            <template #prefix>
              <UserOutlined />
            </template>
          </a-input>
        </a-form-item>
        <a-form-item label="QQ号码" name="qq">
          <a-input
            v-model:value="form.qq"
            placeholder="请输入QQ号码"
            size="large"
            :maxlength="INPUT_LIMITS.QQ"
          >
            <template #prefix>
              <QqOutlined />
            </template>
          </a-input>
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal
      v-model:open="passwordModalVisible"
      title="修改登录密码"
      :width="460"
      @ok="submitPasswordChange"
      @cancel="resetPasswordForm"
      :confirm-loading="securityLoading.password"
    >
      <a-form ref="passwordFormRef" layout="vertical" :model="securityForm" :rules="passwordRules">
        <a-form-item label="当前密码" name="current_password">
          <a-input-password
            v-model:value="securityForm.current_password"
            placeholder="请输入当前密码"
            :maxlength="INPUT_LIMITS.PASSWORD"
          />
        </a-form-item>
        <a-form-item label="新密码" name="new_password">
          <a-input-password
            v-model:value="securityForm.new_password"
            placeholder="请输入新密码"
            :maxlength="INPUT_LIMITS.PASSWORD"
          />
        </a-form-item>
        <a-form-item label="确认新密码" name="confirm_new_password">
          <a-input-password
            v-model:value="securityForm.confirm_new_password"
            placeholder="请再次输入新密码"
            :maxlength="INPUT_LIMITS.PASSWORD"
          />
        </a-form-item>
      </a-form>
      <div class="subtle">
        忘记当前密码？
        <router-link to="/forgot-password">通过找回密码重置</router-link>
      </div>
    </a-modal>

    <a-modal v-model:open="securityModalVisible" :title="securityModalTitle" :width="720" :footer="null">
      <div v-if="securityModalType === 'twofa'" class="security-dialog">
        <a-alert
          :type="twoFAEnabled ? 'success' : 'info'"
          show-icon
          :message="twoFAEnabled ? '当前已启用 2FA' : '当前未启用 2FA'"
          :description="twoFAEnabled ? '换绑流程：验证当前 2FA 后生成新绑定信息，再用新设备验证码确认。' : '首次绑定流程：验证登录密码后生成绑定信息，再输入验证码确认。'"
        />
        <div class="security-block">
          <div class="twofa-flow">
            <div>1. 验证身份并生成二维码</div>
            <div>2. 打开验证器 App 扫描二维码</div>
            <div>3. 输入 6 位动态验证码完成绑定</div>
          </div>
          <div class="field-grid single">
            <a-input-password
              v-if="!twoFAEnabled"
              v-model:value="securityForm.twofa_password"
              placeholder="当前登录密码"
              :maxlength="INPUT_LIMITS.PASSWORD"
            />
            <a-input
              v-else
              v-model:value="securityForm.twofa_current_code"
              placeholder="当前 2FA 验证码（6 位）"
              :maxlength="6"
            />
            <a-button type="primary" :disabled="!canSubmitTwoFASetup" :loading="securityLoading.setup" @click="submitTwoFASetup">
              {{ twoFAEnabled ? "重新生成绑定信息" : "生成绑定信息" }}
            </a-button>
          </div>
        </div>
        <div v-if="twoFASecret" class="twofa-setup">
          <div class="twofa-qr">
            <img v-if="twoFAQRCode" :src="twoFAQRCode" alt="2FA QRCode" />
            <div v-else class="twofa-qr-placeholder">二维码生成失败</div>
          </div>
          <div class="twofa-meta">
            <div class="twofa-meta-title">请用 Google / Microsoft Authenticator 扫码</div>
            <div class="twofa-meta-subtitle">仅支持扫码绑定，不提供手动密钥兜底。</div>
          </div>
        </div>
        <div v-if="twoFASecret" class="security-block confirm-block">
          <div class="confirm-row">
            <a-input
              v-model:value="securityForm.twofa_code"
              placeholder="输入验证器中的 6 位验证码"
              :maxlength="6"
            />
            <a-button type="primary" :disabled="!canSubmitTwoFAConfirm" :loading="securityLoading.confirm" @click="submitTwoFAConfirm">
              完成绑定
            </a-button>
          </div>
        </div>
      </div>

      <div v-else-if="securityModalType === 'email'" class="security-dialog">
        <a-alert
          :type="securityContacts.email_bound ? 'success' : 'info'"
          show-icon
          :message="securityContacts.email_bound ? `当前绑定：${securityContacts.email_masked || '-'}` : '当前未绑定邮箱'"
          description="先完成一次身份校验，再发送邮箱验证码并确认绑定。"
        />
        <div class="security-block">
          <div class="field-grid single">
            <a-input
              v-model:value="securityForm.bind_email_value"
              placeholder="请输入要绑定的新邮箱"
              :maxlength="INPUT_LIMITS.EMAIL"
            />
            <a-alert v-if="twoFAEnabled" type="success" show-icon message="已完成 2FA 身份校验，可继续绑定流程" />
            <a-input-password
              v-else
              v-model:value="securityForm.bind_email_password"
              placeholder="当前登录密码"
              :maxlength="INPUT_LIMITS.PASSWORD"
            />
            <a-input
              v-model:value="securityForm.bind_email_code"
              placeholder="输入邮箱验证码（4-8 位）"
              :maxlength="8"
            />
            <div class="confirm-row">
              <a-button type="default" :disabled="!canSendEmailCode" :loading="securityLoading.emailSend" @click="sendEmailBindCode">
                {{ emailCodeCooldown > 0 ? `${emailCodeCooldown}s 后重发` : "发送验证码" }}
              </a-button>
              <a-button type="primary" :disabled="!canConfirmEmailBind" :loading="securityLoading.emailConfirm" @click="submitEmailBind">
                确认绑定
              </a-button>
            </div>
          </div>
        </div>
      </div>

      <div v-else class="security-dialog">
        <a-alert
          :type="securityContacts.phone_bound ? 'success' : 'info'"
          show-icon
          :message="securityContacts.phone_bound ? `当前绑定：${securityContacts.phone_masked || '-'}` : '当前未绑定手机号'"
          description="先完成一次身份校验，再发送短信验证码并确认绑定。"
        />
        <div class="security-block">
          <div class="field-grid single">
            <a-input
              v-model:value="securityForm.bind_phone_value"
              placeholder="请输入要绑定的新手机号"
              :maxlength="INPUT_LIMITS.PHONE"
            />
            <a-alert v-if="twoFAEnabled" type="success" show-icon message="已完成 2FA 身份校验，可继续绑定流程" />
            <a-input-password
              v-else
              v-model:value="securityForm.bind_phone_password"
              placeholder="当前登录密码"
              :maxlength="INPUT_LIMITS.PASSWORD"
            />
            <a-input
              v-model:value="securityForm.bind_phone_code"
              placeholder="输入短信验证码（4-8 位）"
              :maxlength="8"
            />
            <div class="confirm-row">
              <a-button type="default" :disabled="!canSendPhoneCode" :loading="securityLoading.phoneSend" @click="sendPhoneBindCode">
                {{ phoneCodeCooldown > 0 ? `${phoneCodeCooldown}s 后重发` : "发送验证码" }}
              </a-button>
              <a-button type="primary" :disabled="!canConfirmPhoneBind" :loading="securityLoading.phoneConfirm" @click="submitPhoneBind">
                确认绑定
              </a-button>
            </div>
          </div>
        </div>
      </div>
    </a-modal>

    <a-modal
      v-model:open="precheckVisible"
      :title="precheckTitle"
      :confirm-loading="precheckLoading"
      ok-text="验证并继续"
      cancel-text="取消"
      @ok="submitPrecheck"
      @cancel="resetPrecheck"
    >
      <a-alert
        type="info"
        show-icon
        message="先验证 2FA，再进入下一步"
        description="请输入当前验证器中的 6 位动态验证码。"
        class="modal-alert"
      />
      <a-input
        v-model:value="precheckCode"
        :maxlength="6"
        placeholder="请输入 6 位 2FA 验证码"
      />
    </a-modal>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from "vue";
import dayjs from "dayjs";
import QRCode from "qrcode";
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
  PhoneOutlined,
  TransactionOutlined,
  DollarOutlined,
  SyncOutlined,
  CloseCircleOutlined,
  InfoCircleOutlined,
  KeyOutlined,
  LockOutlined
} from "@ant-design/icons-vue";
import { useAuthStore } from "@/stores/auth";
import {
  changeMyPassword,
  confirmMyEmailBind,
  confirmMyPhoneBind,
  confirmTwoFA,
  getMySecurityContacts,
  getRealNameStatus,
  getTwoFAStatus,
  getWallet,
  sendMyEmailBindCode,
  sendMyPhoneBindCode,
  verifyMyEmailBind2FA,
  verifyMyPhoneBind2FA,
  setupTwoFA
} from "@/services/user";
import { normalizeWallet } from "@/utils/wallet";
import { message } from "ant-design-vue";
import { INPUT_LIMITS } from "@/constants/inputLimits";

const formatTime = (value) => {
  if (!value) return "-";
  return dayjs(value).format("YYYY-MM-DD HH:mm:ss");
};

const extractPayload = (res) => {
  const body = res?.data;
  if (body && typeof body === "object" && body.data && typeof body.data === "object") {
    return body.data;
  }
  return body || {};
};

const auth = useAuthStore();
const profile = computed(() => auth.profile);
const wallet = ref({ balance: 0, currency: "CNY" });
const realname = ref(null);
const editModalVisible = ref(false);
const passwordModalVisible = ref(false);
const securityModalVisible = ref(false);
const securityModalType = ref("twofa");
const precheckVisible = ref(false);
const precheckTarget = ref("");
const precheckCode = ref("");
const precheckLoading = ref(false);
const submitting = ref(false);
const formRef = ref();
const passwordFormRef = ref();
const form = reactive({
  username: "",
  qq: ""
});
const securityForm = reactive({
  current_password: "",
  new_password: "",
  confirm_new_password: "",
  password_totp: "",
  profile_totp: "",
  twofa_password: "",
  twofa_current_code: "",
  twofa_code: "",
  bind_email_value: "",
  bind_email_code: "",
  bind_email_password: "",
  bind_email_totp: "",
  bind_email_ticket: "",
  bind_phone_value: "",
  bind_phone_code: "",
  bind_phone_password: "",
  bind_phone_totp: "",
  bind_phone_ticket: ""
});
const securityLoading = reactive({
  password: false,
  setup: false,
  confirm: false,
  emailSend: false,
  emailConfirm: false,
  phoneSend: false,
  phoneConfirm: false
});
const securityContacts = reactive({
  email_bound: false,
  phone_bound: false,
  email_masked: "",
  phone_masked: ""
});
const twoFAEnabled = ref(false);
const twoFASecret = ref("");
const twoFAUrl = ref("");
const twoFAQRCode = ref("");
const emailCodeCooldown = ref(0);
const phoneCodeCooldown = ref(0);
const emailCodeSent = ref(false);
const phoneCodeSent = ref(false);
let emailCodeTimer = null;
let phoneCodeTimer = null;

const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
const phonePattern = /^[0-9+\-\s]{6,20}$/;
const otpPattern = /^\d{6}$/;

const rules = {
  username: [
    { required: true, message: "请输入用户名", trigger: "blur" },
    { min: 2, max: INPUT_LIMITS.USERNAME, message: `用户名长度应为2-${INPUT_LIMITS.USERNAME}个字符`, trigger: "blur" }
  ],
  qq: []
};

const passwordRules = {
  current_password: [
    { required: true, message: "请输入当前密码", trigger: "blur" }
  ],
  new_password: [
    { required: true, message: "请输入新密码", trigger: "blur" },
    { min: 6, max: INPUT_LIMITS.PASSWORD, message: `密码长度应为6-${INPUT_LIMITS.PASSWORD}位`, trigger: "blur" }
  ],
  confirm_new_password: [
    {
      validator: (_rule, value) => {
        if (String(value || "") !== String(securityForm.new_password || "")) {
          return Promise.reject("两次输入密码不一致");
        }
        return Promise.resolve();
      },
      trigger: "blur"
    }
  ]
};

const normalizedTwoFACurrentCode = computed(() => String(securityForm.twofa_current_code || "").trim());
const normalizedTwoFAConfirmCode = computed(() => String(securityForm.twofa_code || "").trim());
const normalizedProfileTotp = computed(() => String(securityForm.profile_totp || "").trim());
const normalizedPasswordTotp = computed(() => String(securityForm.password_totp || "").trim());
const normalizedEmail = computed(() => String(securityForm.bind_email_value || "").trim());
const normalizedPhone = computed(() => String(securityForm.bind_phone_value || "").trim());
const normalizedEmailCode = computed(() => String(securityForm.bind_email_code || "").trim());
const normalizedPhoneCode = computed(() => String(securityForm.bind_phone_code || "").trim());

const isEmailValid = computed(() => emailPattern.test(normalizedEmail.value));
const isPhoneValid = computed(() => phonePattern.test(normalizedPhone.value));
const emailTicketReady = computed(() => String(securityForm.bind_email_ticket || "").trim().length > 0);
const phoneTicketReady = computed(() => String(securityForm.bind_phone_ticket || "").trim().length > 0);

const canSubmitTwoFASetup = computed(() => {
  if (securityLoading.setup) return false;
  if (twoFAEnabled.value) return otpPattern.test(normalizedTwoFACurrentCode.value);
  return String(securityForm.twofa_password || "").trim().length > 0;
});

const canSubmitTwoFAConfirm = computed(() => {
  if (!twoFASecret.value || !twoFAQRCode.value || securityLoading.confirm) return false;
  return otpPattern.test(normalizedTwoFAConfirmCode.value);
});

const canSendEmailCode = computed(() => {
  if (securityLoading.emailSend || emailCodeCooldown.value > 0) return false;
  if (!isEmailValid.value) return false;
  if (twoFAEnabled.value) return emailTicketReady.value;
  return String(securityForm.bind_email_password || "").trim().length > 0;
});

const canSendPhoneCode = computed(() => {
  if (securityLoading.phoneSend || phoneCodeCooldown.value > 0) return false;
  if (!isPhoneValid.value) return false;
  if (twoFAEnabled.value) return phoneTicketReady.value;
  return String(securityForm.bind_phone_password || "").trim().length > 0;
});

const canConfirmEmailBind = computed(() => {
  if (securityLoading.emailConfirm) return false;
  if (!emailCodeSent.value) return false;
  if (!isEmailValid.value) return false;
  return /^[0-9A-Za-z]{4,8}$/.test(normalizedEmailCode.value);
});

const canConfirmPhoneBind = computed(() => {
  if (securityLoading.phoneConfirm) return false;
  if (!phoneCodeSent.value) return false;
  if (!isPhoneValid.value) return false;
  return /^[0-9A-Za-z]{4,8}$/.test(normalizedPhoneCode.value);
});

const stopEmailCooldown = () => {
  if (!emailCodeTimer) return;
  clearInterval(emailCodeTimer);
  emailCodeTimer = null;
};

const stopPhoneCooldown = () => {
  if (!phoneCodeTimer) return;
  clearInterval(phoneCodeTimer);
  phoneCodeTimer = null;
};

const startEmailCooldown = (seconds = 60) => {
  stopEmailCooldown();
  emailCodeCooldown.value = seconds;
  emailCodeTimer = setInterval(() => {
    if (emailCodeCooldown.value <= 1) {
      emailCodeCooldown.value = 0;
      stopEmailCooldown();
      return;
    }
    emailCodeCooldown.value -= 1;
  }, 1000);
};

const startPhoneCooldown = (seconds = 60) => {
  stopPhoneCooldown();
  phoneCodeCooldown.value = seconds;
  phoneCodeTimer = setInterval(() => {
    if (phoneCodeCooldown.value <= 1) {
      phoneCodeCooldown.value = 0;
      stopPhoneCooldown();
      return;
    }
    phoneCodeCooldown.value -= 1;
  }, 1000);
};

const resetTwoFAState = () => {
  securityForm.twofa_password = "";
  securityForm.twofa_current_code = "";
  securityForm.twofa_code = "";
  twoFASecret.value = "";
  twoFAUrl.value = "";
  twoFAQRCode.value = "";
};

const resetEmailBindState = () => {
  securityForm.bind_email_value = "";
  securityForm.bind_email_code = "";
  securityForm.bind_email_password = "";
  securityForm.bind_email_totp = "";
  securityForm.bind_email_ticket = "";
  emailCodeSent.value = false;
  emailCodeCooldown.value = 0;
  stopEmailCooldown();
};

const resetPhoneBindState = () => {
  securityForm.bind_phone_value = "";
  securityForm.bind_phone_code = "";
  securityForm.bind_phone_password = "";
  securityForm.bind_phone_totp = "";
  securityForm.bind_phone_ticket = "";
  phoneCodeSent.value = false;
  phoneCodeCooldown.value = 0;
  stopPhoneCooldown();
};

onMounted(() => {
  if (auth.token) {
    auth.fetchMe().catch(() => {});
  } else if (!auth.profile) {
    auth.fetchMe().catch(() => {});
  }
  fetchExtras();
  fetchTwoFAStatus();
  fetchSecurityContacts();
});

onBeforeUnmount(() => {
  stopEmailCooldown();
  stopPhoneCooldown();
});

watch(
  profile,
  (val) => {
    if (!val) return;
    form.username = val.username || "";
    form.qq = val.qq || "";
  },
  { immediate: true }
);

const openEditModal = () => {
  form.username = profile.value?.username || "";
  form.qq = profile.value?.qq || "";
  editModalVisible.value = true;
};

const resetForm = () => {
  securityForm.profile_totp = "";
  formRef.value?.clearValidate();
};

const openPasswordModal = () => {
  securityForm.current_password = "";
  securityForm.new_password = "";
  securityForm.confirm_new_password = "";
  passwordModalVisible.value = true;
};

const resetPasswordForm = () => {
  securityForm.current_password = "";
  securityForm.new_password = "";
  securityForm.confirm_new_password = "";
  securityForm.password_totp = "";
  passwordFormRef.value?.clearValidate();
};

const openSecurityModal = async (type, options = {}) => {
  const keepTicket = !!options.keepTicket;
  securityModalType.value = type;
  securityModalVisible.value = true;
  if (type === "twofa") {
    resetTwoFAState();
    await fetchTwoFAStatus();
    return;
  }
  if (type === "email") {
    const ticket = keepTicket ? String(securityForm.bind_email_ticket || "").trim() : "";
    resetEmailBindState();
    if (ticket) securityForm.bind_email_ticket = ticket;
    await Promise.all([fetchTwoFAStatus(), fetchSecurityContacts()]);
    return;
  }
  const ticket = keepTicket ? String(securityForm.bind_phone_ticket || "").trim() : "";
  resetPhoneBindState();
  if (ticket) securityForm.bind_phone_ticket = ticket;
  await Promise.all([fetchTwoFAStatus(), fetchSecurityContacts()]);
};

const precheckTitle = computed(() => {
  if (precheckTarget.value === "edit") return "验证 2FA 后编辑资料";
  if (precheckTarget.value === "password") return "验证 2FA 后修改密码";
  if (precheckTarget.value === "email") return "验证 2FA 后绑定邮箱";
  if (precheckTarget.value === "phone") return "验证 2FA 后绑定手机";
  return "验证 2FA 后进入设置";
});

const resetPrecheck = () => {
  precheckCode.value = "";
  precheckLoading.value = false;
  precheckVisible.value = false;
  precheckTarget.value = "";
};

const openProtectedFlow = async (target) => {
  await fetchTwoFAStatus();
  if (!twoFAEnabled.value) {
    if (target === "edit") return openEditModal();
    if (target === "password") return openPasswordModal();
    if (target === "twofa" || target === "email" || target === "phone") return openSecurityModal(target);
    return;
  }
  precheckTarget.value = target;
  precheckCode.value = "";
  precheckVisible.value = true;
};

const submitPrecheck = async () => {
  const code = String(precheckCode.value || "").trim();
  if (!otpPattern.test(code)) {
    message.warning("请输入 6 位 2FA 验证码");
    return;
  }
  precheckLoading.value = true;
  try {
    if (precheckTarget.value === "email") {
      const res = await verifyMyEmailBind2FA({ totp_code: code });
      const d = extractPayload(res);
      securityForm.bind_email_ticket = String(d.security_ticket || "").trim();
      if (!securityForm.bind_email_ticket) {
        message.error("2FA 校验失败");
        return;
      }
    } else if (precheckTarget.value === "phone") {
      const res = await verifyMyPhoneBind2FA({ totp_code: code });
      const d = extractPayload(res);
      securityForm.bind_phone_ticket = String(d.security_ticket || "").trim();
      if (!securityForm.bind_phone_ticket) {
        message.error("2FA 校验失败");
        return;
      }
    } else if (precheckTarget.value === "edit") {
      securityForm.profile_totp = code;
    } else if (precheckTarget.value === "password") {
      securityForm.password_totp = code;
    } else if (precheckTarget.value === "twofa") {
      securityForm.twofa_current_code = code;
    }

    const target = precheckTarget.value;
    precheckVisible.value = false;
    precheckCode.value = "";
    precheckTarget.value = "";
    message.success("2FA 验证通过");

    if (target === "edit") return openEditModal();
    if (target === "password") return openPasswordModal();
    if (target === "email" || target === "phone") return openSecurityModal(target, { keepTicket: true });
    if (target === "twofa") return openSecurityModal(target);
  } catch (e) {
    message.error(e?.response?.data?.error || "2FA 校验失败");
  } finally {
    precheckLoading.value = false;
  }
};

watch(securityModalVisible, (open) => {
  if (open) return;
  resetTwoFAState();
  resetEmailBindState();
  resetPhoneBindState();
});

const handleSubmit = async () => {
  try {
    await formRef.value?.validate();
  } catch { return; }

  const currentUsername = String(profile.value?.username || "").trim();
  const nextUsername = String(form.username || "").trim();
  const sensitiveChanged = nextUsername && nextUsername !== currentUsername;
  if (twoFAEnabled.value && sensitiveChanged && !otpPattern.test(normalizedProfileTotp.value)) {
    message.warning("已启用2FA，修改账号需输入6位验证码");
    return;
  }

  submitting.value = true;
  try {
    const payload = {
      username: form.username,
      qq: form.qq,
      totp_code: twoFAEnabled.value && sensitiveChanged ? normalizedProfileTotp.value : undefined
    };
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

const fetchTwoFAStatus = async () => {
  try {
    const res = await getTwoFAStatus();
    const d = extractPayload(res);
    if (typeof d.enabled !== "undefined") {
      twoFAEnabled.value = !!d.enabled;
      return;
    }
    if (typeof d.totp_enabled !== "undefined") {
      twoFAEnabled.value = !!d.totp_enabled;
    }
  } catch {
    const profileTotp = (auth.profile || {})?.totp_enabled;
    if (typeof profileTotp !== "undefined") {
      twoFAEnabled.value = !!profileTotp;
      return;
    }
    const fromProfile = Boolean((auth.profile || {})?.totp_enabled);
    if (fromProfile) {
      twoFAEnabled.value = true;
    }
  }
};

const fetchSecurityContacts = async () => {
  try {
    const res = await getMySecurityContacts();
    const d = extractPayload(res);
    securityContacts.email_bound = !!d.email_bound;
    securityContacts.phone_bound = !!d.phone_bound;
    securityContacts.email_masked = String(d.email_masked || "");
    securityContacts.phone_masked = String(d.phone_masked || "");
    if (typeof d.totp_enabled !== "undefined") {
      twoFAEnabled.value = !!d.totp_enabled;
    }
  } catch {
    const profileEmailBound = (auth.profile || {})?.email_bound;
    const profilePhoneBound = (auth.profile || {})?.phone_bound;
    const profileEmailMasked = String((auth.profile || {})?.email_masked || "").trim();
    const profilePhoneMasked = String((auth.profile || {})?.phone_masked || "").trim();
    if (typeof profileEmailBound !== "undefined") {
      securityContacts.email_bound = !!profileEmailBound;
    }
    if (typeof profilePhoneBound !== "undefined") {
      securityContacts.phone_bound = !!profilePhoneBound;
    }
    if (!securityContacts.email_masked && profileEmailMasked) {
      securityContacts.email_masked = profileEmailMasked;
    }
    if (!securityContacts.phone_masked && profilePhoneMasked) {
      securityContacts.phone_masked = profilePhoneMasked;
    }

    const profileEmail = String((auth.profile || {})?.email || "").trim();
    const profilePhone = String((auth.profile || {})?.phone || "").trim();
    if (!securityContacts.email_masked && profileEmail) {
      securityContacts.email_masked = profileEmail;
    }
    if (!securityContacts.phone_masked && profilePhone) {
      securityContacts.phone_masked = profilePhone;
    }
    if (!securityContacts.email_bound && profileEmail) {
      securityContacts.email_bound = true;
    }
    if (!securityContacts.phone_bound && profilePhone) {
      securityContacts.phone_bound = true;
    }
  }
};

const submitPasswordChange = async () => {
  try {
    await passwordFormRef.value?.validate();
  } catch {
    return;
  }
  if (twoFAEnabled.value && !otpPattern.test(normalizedPasswordTotp.value)) {
    message.warning("已启用2FA，修改密码需输入6位验证码");
    return;
  }
  securityLoading.password = true;
  try {
    await changeMyPassword({
      current_password: securityForm.current_password,
      new_password: securityForm.new_password,
      totp_code: twoFAEnabled.value ? normalizedPasswordTotp.value : undefined
    });
    resetPasswordForm();
    passwordModalVisible.value = false;
    message.success("密码已更新");
  } catch (e) {
    message.error(e?.response?.data?.error || "更新失败");
  } finally {
    securityLoading.password = false;
  }
};

const submitTwoFASetup = async () => {
  if (!canSubmitTwoFASetup.value) {
    message.warning(twoFAEnabled.value ? "请输入 6 位当前 2FA 验证码" : "请输入当前登录密码");
    return;
  }
  securityLoading.setup = true;
  try {
    const res = await setupTwoFA({
      password: twoFAEnabled.value ? undefined : String(securityForm.twofa_password || "").trim(),
      current_code: twoFAEnabled.value ? normalizedTwoFACurrentCode.value : undefined
    });
    const d = extractPayload(res);
    twoFASecret.value = String(d.secret || "");
    twoFAUrl.value = String(d.otpauth_url || "");
    if (!twoFASecret.value) {
      message.error("未获取到2FA信息");
      return;
    }
    if (twoFAUrl.value) {
      try {
        twoFAQRCode.value = await QRCode.toDataURL(twoFAUrl.value, {
          width: 180,
          margin: 1
        });
      } catch {
        twoFAQRCode.value = "";
        twoFASecret.value = "";
        twoFAUrl.value = "";
        message.error("二维码生成失败，请重试");
        return;
      }
    }
    if (!twoFAQRCode.value) {
      twoFASecret.value = "";
      twoFAUrl.value = "";
      message.error("二维码生成失败，请重试");
      return;
    }
    message.success("已生成，请使用验证器添加后输入验证码确认");
  } catch (e) {
    const raw = String(e?.response?.data?.error || "").toLowerCase();
    if (raw.includes("unauthorized")) {
      message.error(twoFAEnabled.value ? "当前 2FA 验证码错误" : "登录密码错误");
    } else {
      message.error(e?.response?.data?.error || "生成失败");
    }
  } finally {
    securityLoading.setup = false;
  }
};

const submitTwoFAConfirm = async () => {
  if (!canSubmitTwoFAConfirm.value) {
    message.warning("请输入 6 位验证码完成确认");
    return;
  }
  securityLoading.confirm = true;
  try {
    await confirmTwoFA({ code: normalizedTwoFAConfirmCode.value });
    securityForm.twofa_code = "";
    twoFASecret.value = "";
    twoFAUrl.value = "";
    twoFAQRCode.value = "";
    await fetchTwoFAStatus();
    message.success("2FA 已开启");
  } catch (e) {
    const raw = String(e?.response?.data?.error || "").toLowerCase();
    if (raw.includes("unauthorized")) {
      message.error("验证码错误，请检查验证器时间后重试");
    } else {
      message.error(e?.response?.data?.error || "确认失败");
    }
  } finally {
    securityLoading.confirm = false;
  }
};

const sendEmailBindCode = async () => {
  if (!isEmailValid.value) {
    message.warning("请输入有效邮箱地址");
    return;
  }
  if (twoFAEnabled.value && !emailTicketReady.value) {
    message.warning("请先完成 2FA 校验");
    return;
  }
  if (!twoFAEnabled.value && !String(securityForm.bind_email_password || "").trim()) {
    message.warning("请输入当前登录密码");
    return;
  }
  securityLoading.emailSend = true;
  try {
    await sendMyEmailBindCode({
      value: normalizedEmail.value,
      current_password: twoFAEnabled.value ? undefined : String(securityForm.bind_email_password || "").trim(),
      security_ticket: twoFAEnabled.value ? String(securityForm.bind_email_ticket || "").trim() : undefined
    });
    emailCodeSent.value = true;
    startEmailCooldown();
    message.success("邮箱验证码已发送");
  } catch (e) {
    message.error(e?.response?.data?.error || "发送失败");
  } finally {
    securityLoading.emailSend = false;
  }
};

const submitEmailBind = async () => {
  if (!canConfirmEmailBind.value) {
    message.warning("请输入邮箱和有效验证码后再提交");
    return;
  }
  securityLoading.emailConfirm = true;
  try {
    await confirmMyEmailBind({
      value: normalizedEmail.value,
      code: normalizedEmailCode.value,
      security_ticket: twoFAEnabled.value ? String(securityForm.bind_email_ticket || "").trim() : undefined
    });
    resetEmailBindState();
    message.success("邮箱绑定已更新");
    await Promise.all([auth.fetchMe(), fetchSecurityContacts()]);
  } catch (e) {
    message.error(e?.response?.data?.error || "提交失败");
  } finally {
    securityLoading.emailConfirm = false;
  }
};

const sendPhoneBindCode = async () => {
  if (!isPhoneValid.value) {
    message.warning("请输入有效手机号");
    return;
  }
  if (twoFAEnabled.value && !phoneTicketReady.value) {
    message.warning("请先完成 2FA 校验");
    return;
  }
  if (!twoFAEnabled.value && !String(securityForm.bind_phone_password || "").trim()) {
    message.warning("请输入当前登录密码");
    return;
  }
  securityLoading.phoneSend = true;
  try {
    await sendMyPhoneBindCode({
      value: normalizedPhone.value,
      current_password: twoFAEnabled.value ? undefined : String(securityForm.bind_phone_password || "").trim(),
      security_ticket: twoFAEnabled.value ? String(securityForm.bind_phone_ticket || "").trim() : undefined
    });
    phoneCodeSent.value = true;
    startPhoneCooldown();
    message.success("短信验证码已发送");
  } catch (e) {
    message.error(e?.response?.data?.error || "发送失败");
  } finally {
    securityLoading.phoneSend = false;
  }
};

const submitPhoneBind = async () => {
  if (!canConfirmPhoneBind.value) {
    message.warning("请输入手机号和有效验证码后再提交");
    return;
  }
  securityLoading.phoneConfirm = true;
  try {
    await confirmMyPhoneBind({
      value: normalizedPhone.value,
      code: normalizedPhoneCode.value,
      security_ticket: twoFAEnabled.value ? String(securityForm.bind_phone_ticket || "").trim() : undefined
    });
    resetPhoneBindState();
    message.success("手机号绑定已更新");
    await Promise.all([auth.fetchMe(), fetchSecurityContacts()]);
  } catch (e) {
    message.error(e?.response?.data?.error || "提交失败");
  } finally {
    securityLoading.phoneConfirm = false;
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

const securityModalTitle = computed(() => {
  if (securityModalType.value === "email") return "邮箱绑定 / 换绑";
  if (securityModalType.value === "phone") return "手机绑定 / 换绑";
  return "双因素认证（2FA）";
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

.info-item :deep(.ant-btn-text) {
  padding: 0 8px;
  height: auto;
  font-size: 12px;
}

.security-dialog {
  margin-top: 8px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.security-block {
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 14px;
}

.field-grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: 10px;
}

.field-grid.single {
  gap: 12px;
}

.twofa-flow {
  display: grid;
  gap: 6px;
  margin-bottom: 12px;
  font-size: 12px;
  color: var(--text-secondary);
}

.confirm-row {
  display: flex;
  gap: 10px;
  align-items: center;
  flex-wrap: wrap;
}

.confirm-row :deep(.ant-input),
.confirm-row :deep(.ant-input-affix-wrapper) {
  flex: 1;
}

.twofa-setup {
  border: 1px dashed var(--border);
  border-radius: 12px;
  padding: 12px;
  display: grid;
  grid-template-columns: 180px 1fr;
  gap: 14px;
  align-items: start;
}

.confirm-block {
  margin-top: 0;
}

.twofa-qr {
  width: 180px;
  height: 180px;
  border-radius: 10px;
  border: 1px solid var(--border);
  background: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}

.twofa-qr img {
  width: 100%;
  height: 100%;
}

.twofa-qr-placeholder {
  font-size: 12px;
  color: var(--text-tertiary);
}

.twofa-meta {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
}

.twofa-meta-title {
  font-size: 14px;
  font-weight: 600;
}

.twofa-meta-subtitle {
  font-size: 12px;
  color: var(--text-secondary);
}

.twofa-meta code {
  display: inline-block;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  background: rgba(15, 23, 42, 0.04);
  padding: 2px 6px;
  border-radius: 6px;
  border: 1px solid var(--border-light);
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

  .confirm-row {
    flex-direction: column;
    align-items: stretch;
  }

  .twofa-setup {
    grid-template-columns: 1fr;
  }

  .twofa-qr {
    margin: 0 auto;
  }
}
</style>
