<template>
  <a-config-provider :theme="darkTheme">
    <div class="auth-page register-page console-dark">
      <div class="auth-shell">
        <div class="auth-banner register-banner">
          <div class="banner-content">
            <div class="banner-logo" aria-hidden="true">
              <SiteLogoMedia :size="32" />
            </div>
            <div class="banner-text">
              <p class="banner-eyebrow">XIAOHEI CLOUD</p>
              <h1 class="banner-title">注册账户</h1>
              <p class="banner-desc">开启您的云端之旅</p>
            </div>
            <div class="banner-divider"></div>
            <div class="banner-stats">
              <div class="stat-item">
                <span class="stat-value">99.9%</span>
                <span class="stat-label">服务可用性</span>
              </div>
              <div class="stat-item">
                <span class="stat-value">100+</span>
                <span class="stat-label">数据中心</span>
              </div>
              <div class="stat-item">
                <span class="stat-value">24/7</span>
                <span class="stat-label">技术支持</span>
              </div>
            </div>
          </div>
        </div>
        <a-card class="auth-card register-card" :bordered="false">
        <div class="card-header">
          <h2 class="section-title">创建账户</h2>
          <p class="register-subtitle">填写以下信息完成注册</p>
        </div>
        <a-alert
          v-if="!settings.register_enabled"
          type="warning"
          show-icon
          message="当前已关闭注册"
          style="margin-bottom: 16px"
        />
        <a-form :model="form" layout="vertical" @finish="onSubmit" class="register-form">
          <a-tabs
            v-if="showChannelTabs"
            v-model:activeKey="activeRegisterTab"
            class="register-tabs"
          >
            <a-tab-pane key="email" tab="邮箱注册" />
            <a-tab-pane key="sms" tab="手机号注册" />
          </a-tabs>
          <a-form-item
            label="用户名"
            name="username"
            :rules="[{ required: isRequired('username'), message: '请输入用户名' }]"
          >
            <a-input v-model:value="form.username" placeholder="请输入用户名" size="large">
              <template #prefix>
                <UserOutlined />
              </template>
            </a-input>
          </a-form-item>
          <a-form-item
            v-if="showEmailField"
            :label="verifyChannel === 'email' ? '邮箱' : '邮箱（选填）'"
            name="email"
            :rules="[{ required: verifyChannel === 'email', message: '请输入邮箱' }]"
          >
            <a-input v-model:value="form.email" placeholder="请输入邮箱地址" size="large">
              <template #prefix>
                <MailOutlined />
              </template>
            </a-input>
          </a-form-item>
          <a-form-item v-if="showField('qq')" label="QQ" name="qq" :rules="[{ required: isRequired('qq'), message: '请输入QQ' }]">
            <a-input v-model:value="form.qq" placeholder="请输入QQ号" size="large">
              <template #prefix>
                <QqOutlined />
              </template>
            </a-input>
          </a-form-item>
          <a-form-item
            v-if="showPhoneField"
            label="手机号"
            name="phone"
            :rules="[{ required: isRequired('phone') || verifyChannel === 'sms', message: '请输入手机号' }]"
          >
            <a-input v-model:value="form.phone" placeholder="请输入手机号" size="large">
              <template #prefix>
                <PhoneOutlined />
              </template>
            </a-input>
          </a-form-item>
          <a-form-item
            label="密码"
            name="password"
            :rules="[{ required: isRequired('password'), message: '请输入密码' }]"
          >
            <a-input-password v-model:value="form.password" placeholder="请设置登录密码" size="large">
              <template #prefix>
                <LockOutlined />
              </template>
            </a-input-password>
          </a-form-item>
          <a-form-item
            v-if="settings.register_captcha_enabled"
            :label="settings.captcha_provider === 'geetest' ? '行为验证码' : '图形验证码'"
            name="captcha_code"
            :rules="settings.captcha_provider === 'geetest' ? [] : [{ required: true, message: '请输入验证码' }]"
          >
            <div v-if="settings.captcha_provider !== 'geetest'" class="captcha">
              <a-input v-model:value="form.captcha_code" placeholder="请输入验证码" size="large" class="captcha-input" />
              <div class="captcha-img" @click="refreshCaptcha">
                <img v-if="captchaImage" :src="captchaImage" alt="captcha" />
                <span v-else>加载中</span>
              </div>
            </div>
            <div v-else class="captcha-geetest">
              <a-button @click="verifyGeeTest" :disabled="!geetest.ready" :loading="geetest.loading" block>
                {{ geetest.passed ? "已通过验证，点击重试" : "点击完成极验验证" }}
              </a-button>
              <span class="captcha-geetest-status">{{ geetest.passed ? "验证通过" : "未验证" }}</span>
            </div>
          </a-form-item>
          <a-form-item
            v-if="verifyChannels.length > 0"
            :label="verifyChannel === 'email' ? '邮箱验证码' : '短信验证码'"
            name="verify_code"
            :rules="[{ required: true, message: '请输入验证码' }]"
          >
            <div class="verify-code">
              <a-input v-model:value="form.verify_code" placeholder="请输入验证码" size="large" class="verify-input" />
              <a-button :disabled="sendCooling || !canSendCode" @click="sendCode" size="large" class="verify-btn">
                {{ sendCooling ? `${sendCount}s` : "获取验证码" }}
              </a-button>
            </div>
          </a-form-item>
          <a-button type="primary" html-type="submit" block :loading="loading" size="large" class="submit-btn">
            立即注册
          </a-button>
        </a-form>
        <div class="auth-footer">
          <span>已有账号？</span>
          <router-link to="/login" class="login-link">立即登录</router-link>
        </div>
      </a-card>
    </div>
  </div>
  </a-config-provider>
</template>

<script setup>
import { reactive, ref, onMounted, computed, watch } from "vue";
import { message, ConfigProvider, theme } from "ant-design-vue";
import { UserOutlined, MailOutlined, LockOutlined, PhoneOutlined, QqOutlined } from "@ant-design/icons-vue";
import { getCaptcha, userRegister, getAuthSettings, requestRegisterCode } from "@/services/user";
import { useRouter } from "vue-router";
import SiteLogoMedia from "@/components/brand/SiteLogoMedia.vue";
import { INPUT_LIMITS } from "@/constants/inputLimits";

const router = useRouter();

const darkTheme = {
  algorithm: theme.darkAlgorithm,
  token: {
    colorPrimary: '#3b82f6',
    colorBgBase: '#1e2433',
    colorBgContainer: '#1e2433',
    colorBgElevated: '#252b3d',
    colorBorder: '#2d3748',
    colorText: '#f1f5f9',
    colorTextSecondary: '#94a3b8',
    colorTextTertiary: '#64748b',
    colorTextQuaternary: '#4a5568',
    colorBorderSecondary: '#374151',
    colorFillAlter: '#252b3d',
    colorFillContent: '#252b3d',
    colorFillTextHover: 'rgba(59, 130, 246, 0.08)',
    colorFill: '#252b3d',
    colorFillTertiary: '#2d3748',
    colorBgLayout: '#0f1419',
    colorBgSpotlight: '#252b3d',
    borderRadius: 8,
    fontSize: 14,
    fontFamily: 'Inter, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto',
  },
  components: {
    Input: {
      colorBgContainer: '#252b3d',
      colorBorder: '#2d3748',
      colorTextPlaceholder: '#64748b',
      colorText: '#f1f5f9',
    },
    InputNumber: {
      colorBgContainer: '#252b3d',
      colorBorder: '#2d3748',
    },
    Select: {
      colorBgElevated: '#252b3d',
      colorBgSpotlight: '#2d3748',
    },
    Button: {
      colorPrimary: '#3b82f6',
      colorPrimaryHover: '#60a5fa',
      colorPrimaryActive: '#2563eb',
      colorBgTextHover: 'rgba(59, 130, 246, 0.08)',
    },
    Alert: {
      colorWarningBg: 'rgba(245, 158, 11, 0.15)',
      colorWarningBorder: 'rgba(245, 158, 11, 0.3)',
      colorWarningText: '#fbbf24',
    },
    Tabs: {
      inkBarColor: '#3b82f6',
      itemActiveColor: '#3b82f6',
      itemSelectedColor: '#3b82f6',
    },
  },
};

const loading = ref(false);
const captchaId = ref("");
const captchaImage = ref("");
const sendCooling = ref(false);
const sendCount = ref(60);
let sendTimer;

const form = reactive({
  username: "",
  email: "",
  qq: "",
  phone: "",
  password: "",
  captcha_code: "",
  verify_code: ""
});

const settings = reactive({
  register_enabled: true,
  register_required_fields: ["username", "password"],
  register_email_required: true,
  register_verify_type: "none",
  register_verify_channels: [],
  register_captcha_enabled: true,
  captcha_provider: "image"
});
const geetest = reactive({
  widget: null,
  ready: false,
  loading: false,
  passed: false,
  lot_number: "",
  captcha_output: "",
  pass_token: "",
  gen_time: ""
});
const verifyChannels = computed(() => {
  let list = Array.isArray(settings.register_verify_channels) ? [...settings.register_verify_channels] : [];
  if (list.length === 0 && (settings.register_verify_type === "email" || settings.register_verify_type === "sms")) {
    list = [settings.register_verify_type];
  }
  return list;
});
const verifyChannel = ref("email");
const activeRegisterTab = ref("email");
const showChannelTabs = computed(() => verifyChannels.value.includes("email") && verifyChannels.value.includes("sms"));

const requiredSet = computed(() => new Set((settings.register_required_fields || []).map((v) => String(v).toLowerCase())));
const isRequired = (field) => requiredSet.value.has(String(field).toLowerCase());
const showField = (field) => isRequired(field) || field === "qq";
const showEmailField = computed(() => verifyChannel.value === "email");
const showPhoneField = computed(() => verifyChannel.value === "sms" || isRequired("phone"));

const canSendCode = computed(() => {
  if (verifyChannel.value === "email") {
    return String(form.email || "").trim().length > 0;
  }
  if (verifyChannel.value === "sms") {
    return String(form.phone || "").trim().length > 0;
  }
  return false;
});

const resetGeeTestResult = () => {
  geetest.passed = false;
  geetest.lot_number = "";
  geetest.captcha_output = "";
  geetest.pass_token = "";
  geetest.gen_time = "";
};

const ensureGeeTestScript = async () => {
  if (window.initGeetest4) return;
  await new Promise((resolve, reject) => {
    const existed = document.querySelector("script[data-geetest='gt4']");
    if (existed) {
      existed.addEventListener("load", resolve, { once: true });
      existed.addEventListener("error", reject, { once: true });
      return;
    }
    const script = document.createElement("script");
    script.src = "https://static.geetest.com/v4/gt4.js";
    script.async = true;
    script.defer = true;
    script.dataset.geetest = "gt4";
    script.onload = resolve;
    script.onerror = reject;
    document.head.appendChild(script);
  });
};

const initGeeTest = async (captchaID) => {
  resetGeeTestResult();
  geetest.ready = false;
  geetest.widget = null;
  if (!captchaID) return;
  await ensureGeeTestScript();
  await new Promise((resolve) => {
    window.initGeetest4(
      { captchaId: captchaID, product: "bind", language: "zho" },
      (captchaObj) => {
        geetest.widget = captchaObj;
        geetest.ready = true;
        captchaObj.onSuccess(() => {
          const result = captchaObj.getValidate ? captchaObj.getValidate() : null;
          geetest.lot_number = String(result?.lot_number || "");
          geetest.captcha_output = String(result?.captcha_output || "");
          geetest.pass_token = String(result?.pass_token || "");
          geetest.gen_time = String(result?.gen_time || "");
          geetest.passed = Boolean(
            geetest.lot_number && geetest.captcha_output && geetest.pass_token && geetest.gen_time
          );
        });
        captchaObj.onError(() => {
          resetGeeTestResult();
          message.error("极验初始化失败");
        });
        resolve(true);
      }
    );
  }).catch(() => {
    message.error("极验脚本加载失败");
  });
};

const verifyGeeTest = async () => {
  if (!geetest.widget || !geetest.ready) {
    message.warning("极验尚未就绪，请稍后");
    return;
  }
  geetest.loading = true;
  try {
    resetGeeTestResult();
    geetest.widget.showCaptcha();
  } finally {
    geetest.loading = false;
  }
};

const refreshCaptcha = async () => {
  if (!settings.register_captcha_enabled) return;
  const res = await getCaptcha();
  const provider = String(res.data?.captcha_provider || settings.captcha_provider || "image").toLowerCase();
  settings.captcha_provider = provider === "geetest" ? "geetest" : "image";
  captchaId.value = String(res.data?.captcha_id || "");
  if (settings.captcha_provider === "geetest") {
    captchaImage.value = "";
    await initGeeTest(captchaId.value);
    return;
  }
  const base64 = String(res.data?.image_base64 || "");
  captchaImage.value = base64 ? `data:image/png;base64,${base64}` : "";
  resetGeeTestResult();
};

const loadSettings = async () => {
  try {
    const res = await getAuthSettings();
    Object.assign(settings, res.data || {});
    if (verifyChannels.value.length > 0 && !verifyChannels.value.includes(verifyChannel.value)) {
      verifyChannel.value = verifyChannels.value[0];
    }
    activeRegisterTab.value = verifyChannel.value;
  } catch (error) {
    console.error("Failed to fetch auth settings:", error);
  } finally {
    refreshCaptcha();
  }
};

watch(activeRegisterTab, (v) => {
  if (v === "email" || v === "sms") {
    verifyChannel.value = v;
    form.verify_code = "";
  }
});

watch(verifyChannel, (v) => {
  if (showChannelTabs.value && activeRegisterTab.value !== v) {
    activeRegisterTab.value = v;
  }
});

const sendCode = async () => {
  if (!canSendCode.value || sendCooling.value) return;
  if (settings.register_captcha_enabled && settings.captcha_provider === "geetest" && !geetest.passed) {
    message.warning("请先完成极验验证");
    return;
  }
  sendCooling.value = true;
  sendCount.value = 60;
  try {
    await requestRegisterCode({
      channel: verifyChannel.value,
      email: verifyChannel.value === "email" ? form.email : "",
      phone: verifyChannel.value === "sms" ? form.phone : "",
      captcha_id: captchaId.value,
      captcha_code: form.captcha_code,
      lot_number: geetest.lot_number,
      captcha_output: geetest.captcha_output,
      pass_token: geetest.pass_token,
      gen_time: geetest.gen_time
    });
    message.success("验证码已发送");
    form.captcha_code = "";
    await refreshCaptcha();
    sendTimer = setInterval(() => {
      sendCount.value -= 1;
      if (sendCount.value <= 0) {
        clearInterval(sendTimer);
        sendTimer = null;
        sendCooling.value = false;
      }
    }, 1000);
  } catch (error) {
    message.error("发送失败，请稍后重试");
    sendCooling.value = false;
    await refreshCaptcha();
  }
};

const onSubmit = async () => {
  loading.value = true;
  try {
    if (!settings.register_enabled) {
      message.warning("当前已关闭注册");
      return;
    }
    if (String(form.username || "").length > INPUT_LIMITS.USERNAME) {
      message.error(`用户名长度不能超过 ${INPUT_LIMITS.USERNAME} 个字符`);
      return;
    }
    if (String(form.email || "").length > INPUT_LIMITS.EMAIL) {
      message.error(`邮箱长度不能超过 ${INPUT_LIMITS.EMAIL} 个字符`);
      return;
    }
    if (String(form.qq || "").length > INPUT_LIMITS.QQ) {
      message.error(`QQ 长度不能超过 ${INPUT_LIMITS.QQ} 个字符`);
      return;
    }
    if (String(form.phone || "").length > INPUT_LIMITS.PHONE) {
      message.error(`手机号长度不能超过 ${INPUT_LIMITS.PHONE} 个字符`);
      return;
    }
    if (String(form.password || "").length > INPUT_LIMITS.PASSWORD) {
      message.error(`密码长度不能超过 ${INPUT_LIMITS.PASSWORD} 个字符`);
      return;
    }
    if (settings.register_captcha_enabled && settings.captcha_provider === "geetest" && !geetest.passed) {
      message.warning("请先完成极验验证");
      return;
    }
    await userRegister({
      username: form.username,
      email: verifyChannel.value === "email" ? form.email : "",
      qq: form.qq,
      phone: verifyChannel.value === "sms" ? form.phone : "",
      password: form.password,
      verify_channel: verifyChannel.value,
      captcha_id: captchaId.value,
      captcha_code: form.captcha_code,
      lot_number: geetest.lot_number,
      captcha_output: geetest.captcha_output,
      pass_token: geetest.pass_token,
      gen_time: geetest.gen_time,
      verify_code: form.verify_code
    });
    message.success("注册成功，请登录");
    router.replace("/login");
  } finally {
    loading.value = false;
  }
};

onMounted(loadSettings);
</script>

<style scoped>
.auth-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background: var(--bg-primary);
}

.auth-shell {
  display: grid;
  grid-template-columns: minmax(380px, 1fr) minmax(380px, 1fr);
  gap: 0;
  width: min(1000px, 100%);
  background: var(--card);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-lg);
  overflow: hidden;
}

.auth-banner {
  padding: 0;
  background: #1a1f2e;
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
}

.auth-banner::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background:
    radial-gradient(circle at 20% 30%, rgba(59, 130, 246, 0.04) 0%, transparent 40%),
    radial-gradient(circle at 80% 20%, rgba(139, 92, 246, 0.03) 0%, transparent 35%),
    radial-gradient(circle at 60% 80%, rgba(59, 130, 246, 0.02) 0%, transparent 30%);
  pointer-events: none;
}

.auth-banner::after {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-image:
    linear-gradient(45deg, transparent 48%, rgba(59, 130, 246, 0.015) 50%, transparent 52%),
    linear-gradient(-45deg, transparent 48%, rgba(59, 130, 246, 0.015) 50%, transparent 52%);
  background-size: 60px 60px;
  pointer-events: none;
  opacity: 0.5;
}

.banner-content {
  padding: 48px 40px;
  display: flex;
  flex-direction: column;
  gap: 32px;
  position: relative;
  z-index: 1;
}

.banner-logo {
  align-self: flex-start;
  width: 48px;
  height: 48px;
  border-radius: var(--radius-md);
  background: rgba(59, 130, 246, 0.15);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #3b82f6;
}

.banner-text {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.banner-eyebrow {
  margin: 0;
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 2px;
  text-transform: uppercase;
  color: rgba(255, 255, 255, 0.6);
}

.banner-title {
  margin: 0;
  font-size: 24px;
  font-weight: 700;
  color: #fff;
  letter-spacing: -0.5px;
}

.banner-desc {
  margin: 0;
  font-size: 14px;
  color: rgba(255, 255, 255, 0.7);
  line-height: 1.5;
}

.banner-divider {
  height: 1px;
  background: linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.1), transparent);
}

.banner-stats {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.stat-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.stat-value {
  font-size: 20px;
  font-weight: 700;
  color: #3b82f6;
}

.stat-label {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.6);
}

.auth-card {
  width: 100%;
  border-radius: 0;
  box-shadow: none;
}

.register-card {
  background: var(--card) !important;
  border: none;
}

.register-card :deep(.ant-card-body) {
  background: var(--card) !important;
}

.register-card :deep(.ant-alert-warning) {
  background: rgba(245, 158, 11, 0.15);
  border-color: rgba(245, 158, 11, 0.3);
}

.register-card :deep(.ant-alert-warning .ant-alert-message) {
  color: #fbbf24;
}

.card-header {
  padding: 36px 40px 20px;
  text-align: center;
}

.section-title {
  margin: 0 0 6px;
  font-size: 22px;
  font-weight: 700;
  color: var(--text-primary);
  letter-spacing: -0.5px;
}

.register-subtitle {
  margin: 0;
  color: var(--text-secondary);
  font-size: 13px;
}

.register-form {
  padding: 0 40px 32px;
}

.register-tabs {
  margin-bottom: 16px;
}

.register-tabs :deep(.ant-tabs-nav) {
  margin-bottom: 0;
}

.register-tabs :deep(.ant-tabs-tab) {
  padding: 10px 16px;
  font-size: 14px;
  font-weight: 500;
}

.register-tabs :deep(.ant-tabs-tab-active .ant-tabs-tab-btn) {
  color: var(--primary);
  font-weight: 600;
}

.register-tabs :deep(.ant-tabs-ink-bar) {
  background: var(--primary-gradient);
}

.register-form :deep(.ant-form-item) {
  margin-bottom: 16px;
}

.register-form :deep(.ant-form-item-label > label) {
  color: var(--text-primary);
  font-weight: 600;
  font-size: 13px;
  height: auto;
}

.register-form :deep(.ant-input),
.register-form :deep(.ant-input-affix-wrapper),
.register-form :deep(.ant-input-password),
.register-form :deep(.ant-input-number),
.register-form :deep(.ant-select-selector) {
  border-radius: var(--radius-md);
  border-color: var(--border);
  background: var(--bg-primary);
  font-size: 14px;
}

/* Fix browser autofill background color */
.register-form :deep(input:-webkit-autofill),
.register-form :deep(input:-webkit-autofill:hover),
.register-form :deep(input:-webkit-autofill:focus),
.register-form :deep(input:-webkit-autofill:active),
.register-form :deep(textarea:-webkit-autofill),
.register-form :deep(textarea:-webkit-autofill:hover),
.register-form :deep(textarea:-webkit-autofill:focus),
.register-form :deep(textarea:-webkit-autofill:active),
.register-form :deep(select:-webkit-autofill),
.register-form :deep(select:-webkit-autofill:hover),
.register-form :deep(select:-webkit-autofill:focus) {
  -webkit-box-shadow: 0 0 0 30px #252b3d inset !important;
  -webkit-text-fill-color: #f1f5f9 !important;
  transition: background-color 5000s ease-in-out 0s;
  caret-color: #f1f5f9;
}

.register-form :deep(input:-webkit-autofill::first-line),
.register-form :deep(textarea:-webkit-autofill::first-line) {
  color: #f1f5f9 !important;
  font-size: 14px;
}

.register-form :deep(.ant-input:hover),
.register-form :deep(.ant-input-affix-wrapper:hover),
.register-form :deep(.ant-input-password:hover) {
  border-color: var(--primary-light);
}

.register-form :deep(.ant-input:focus),
.register-form :deep(.ant-input-affix-wrapper-focused),
.register-form :deep(.ant-input-password:focus),
.register-form :deep(.ant-select-focused .ant-select-selector) {
  border-color: var(--primary);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.register-form :deep(.ant-input-affix-wrapper-focused),
.register-form :deep(.ant-input-password-focused) {
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.register-form :deep(.ant-input-affix-wrapper > .ant-input),
.register-form :deep(.ant-input-password) {
  font-size: 14px;
}

.register-form :deep(.anticon) {
  color: var(--text-tertiary);
}

.register-form :deep(.ant-btn-primary),
.register-form :deep(.ant-btn-default) {
  border-radius: var(--radius-md);
  font-weight: 600;
  font-size: 14px;
  letter-spacing: 0.3px;
}

.register-form :deep(.ant-btn-primary) {
  background: var(--primary-gradient);
  border: none;
  box-shadow: var(--shadow-md), var(--shadow-glow-sm);
}

.register-form :deep(.ant-btn-primary:hover) {
  background: linear-gradient(135deg, #2563eb 0%, #1d4ed8 50%, #1e40af 100%);
  box-shadow: var(--shadow-lg), var(--shadow-glow);
}

.register-form :deep(.ant-btn-default) {
  border-color: var(--border-dark);
  color: var(--text-primary);
}

.register-form :deep(.ant-btn-default:hover:not(:disabled)) {
  border-color: var(--primary);
  color: var(--primary);
  background: var(--primary-gradient-subtle);
}

.submit-btn {
  margin-top: 8px;
  height: 42px;
}

.auth-footer {
  margin-top: 16px;
  text-align: center;
  color: var(--text-secondary);
  font-size: 13px;
}

.login-link {
  color: var(--primary);
  font-weight: 600;
  text-decoration: none;
  transition: color var(--transition-base);
}

.login-link:hover {
  color: var(--primary-dark);
}

.captcha {
  display: flex;
  gap: 10px;
}

.captcha-input {
  flex: 1;
}

.captcha-img {
  flex: 0 0 120px;
  height: 36px;
  border: 1.5px solid var(--border);
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  overflow: hidden;
  background: var(--bg-primary);
  transition: border-color var(--transition-base);
}

.captcha-img:hover {
  border-color: var(--primary);
}

.captcha-img img {
  height: 100%;
  width: 100%;
  object-fit: contain;
}

.captcha-img span {
  font-size: 11px;
  color: var(--text-tertiary);
}

.captcha-geetest {
  display: flex;
  align-items: center;
  gap: 10px;
}

.captcha-geetest :deep(.ant-btn) {
  flex: 1;
}

.captcha-geetest-status {
  color: var(--text-secondary);
  font-size: 13px;
  white-space: nowrap;
}

.verify-code {
  display: flex;
  gap: 10px;
}

.verify-input {
  flex: 1;
}

.verify-btn {
  flex: 0 0 110px;
}

@media (max-width: 1024px) {
  .auth-shell {
    grid-template-columns: 1fr;
    max-width: 420px;
  }

  .auth-banner {
    display: none;
  }

  .card-header {
    padding: 32px 28px 18px;
  }

  .register-form {
    padding: 0 28px 28px;
  }

  .section-title {
    font-size: 20px;
  }
}

@media (max-width: 640px) {
  .auth-page {
    padding: 16px;
  }

  .auth-shell {
    border-radius: var(--radius-lg);
  }

  .card-header {
    padding: 28px 20px 16px;
  }

  .register-form {
    padding: 0 20px 24px;
  }

  .section-title {
    font-size: 18px;
  }

  .verify-btn {
    flex-basis: 95px;
  }

  .captcha-img {
    flex-basis: 95px;
  }
}
</style>