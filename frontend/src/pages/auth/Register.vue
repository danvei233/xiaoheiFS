<template>
  <div class="auth-page register-page">
    <div class="register-noise" aria-hidden="true"></div>
    <div class="register-orb register-orb-a" aria-hidden="true"></div>
    <div class="register-orb register-orb-b" aria-hidden="true"></div>
    <div class="auth-shell">
      <div class="auth-banner register-banner">
        <div class="auth-logo register-logo" aria-hidden="true">
          <SiteLogoMedia :size="24" />
        </div>
        <p class="banner-eyebrow">XIAOHEI CLOUD</p>
        <h1 class="banner-title">开始使用小黑云</h1>
        <p class="banner-desc">注册后即可进入控制台，选购 VPS 并跟踪订单进度。</p>
        <div class="banner-points">
          <div class="banner-point">快速开通实例</div>
          <div class="banner-point">订单状态实时同步</div>
          <div class="banner-point">工单支持与通知提醒</div>
        </div>
        <div class="subtle banner-tip">请确保验证码正确以完成注册。</div>
      </div>
      <a-card class="card auth-card register-card" :bordered="false">
        <div class="section-title register-title">用户注册</div>
        <p class="register-subtitle">创建账户后即可使用完整功能</p>
        <a-alert
          v-if="!settings.register_enabled"
          type="warning"
          show-icon
          message="当前已关闭注册"
          style="margin-bottom: 16px"
        />
        <a-form :model="form" layout="vertical" @finish="onSubmit">
          <a-form-item
            label="用户名"
            name="username"
            :rules="[{ required: isRequired('username'), message: '请输入用户名' }]"
          >
            <a-input v-model:value="form.username" :maxlength="INPUT_LIMITS.USERNAME" />
          </a-form-item>
          <a-tabs
            v-if="showChannelTabs"
            v-model:activeKey="activeRegisterTab"
            size="small"
            style="margin-bottom: 8px"
          >
            <a-tab-pane key="email" tab="邮箱注册" />
            <a-tab-pane key="sms" tab="手机号注册" />
          </a-tabs>
          <a-form-item
            v-if="showEmailField"
            :label="verifyChannel === 'email' ? '邮箱' : '邮箱（选填）'"
            name="email"
            :rules="[{ required: verifyChannel === 'email', message: '请输入邮箱' }]"
          >
            <a-input v-model:value="form.email" :maxlength="INPUT_LIMITS.EMAIL" />
          </a-form-item>
          <a-form-item v-if="showField('qq')" label="QQ" name="qq" :rules="[{ required: isRequired('qq'), message: '请输入QQ' }]">
            <a-input v-model:value="form.qq" :maxlength="INPUT_LIMITS.QQ" />
          </a-form-item>
          <a-form-item
            v-if="showPhoneField"
            label="手机号"
            name="phone"
            :rules="[{ required: isRequired('phone') || verifyChannel === 'sms', message: '请输入手机号' }]"
          >
            <a-input v-model:value="form.phone" :maxlength="INPUT_LIMITS.PHONE" />
          </a-form-item>
          <a-form-item
            label="密码"
            name="password"
            :rules="[{ required: isRequired('password'), message: '请输入密码' }]"
          >
            <a-input-password v-model:value="form.password" :maxlength="INPUT_LIMITS.PASSWORD" />
          </a-form-item>
          <a-form-item
            v-if="settings.register_captcha_enabled"
            label="图形验证码"
            name="captcha_code"
            :rules="[{ required: true, message: '请输入验证码' }]"
          >
            <div class="captcha form-inline">
              <a-input v-model:value="form.captcha_code" placeholder="验证码" />
              <div class="captcha-img" @click="refreshCaptcha">
                <img v-if="captchaImage" :src="captchaImage" alt="captcha" />
                <span v-else>点击刷新</span>
              </div>
            </div>
          </a-form-item>
          <a-form-item
            v-if="verifyChannels.length > 0"
            :label="verifyChannel === 'email' ? '邮箱验证码' : '短信验证码'"
            name="verify_code"
            :rules="[{ required: true, message: '请输入验证码' }]"
          >
            <div class="verify-code form-inline">
              <a-input v-model:value="form.verify_code" placeholder="验证码" />
              <a-button :disabled="sendCooling || !canSendCode" @click="sendCode">
                {{ sendCooling ? `${sendCount}s` : "发送验证码" }}
              </a-button>
            </div>
          </a-form-item>
          <a-button type="primary" html-type="submit" block :loading="loading">注册</a-button>
        </a-form>
        <div class="auth-footer">
          已有账号？<router-link to="/login">去登录</router-link>
        </div>
      </a-card>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref, onMounted, computed, watch } from "vue";
import { message } from "ant-design-vue";
import { getCaptcha, userRegister, getAuthSettings, requestRegisterCode } from "@/services/user";
import { useRouter } from "vue-router";
import SiteLogoMedia from "@/components/brand/SiteLogoMedia.vue";
import { INPUT_LIMITS } from "@/constants/inputLimits";

const router = useRouter();
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
  register_captcha_enabled: true
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

const refreshCaptcha = async () => {
  if (!settings.register_captcha_enabled) return;
  const res = await getCaptcha();
  captchaId.value = res.data?.captcha_id || "";
  const base64 = res.data?.image_base64 || "";
  captchaImage.value = base64 ? `data:image/png;base64,${base64}` : "";
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
  sendCooling.value = true;
  sendCount.value = 60;
  try {
    await requestRegisterCode({
      channel: verifyChannel.value,
      email: verifyChannel.value === "email" ? form.email : "",
      phone: verifyChannel.value === "sms" ? form.phone : "",
      captcha_id: captchaId.value,
      captcha_code: form.captcha_code
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
    await userRegister({
      username: form.username,
      email: verifyChannel.value === "email" ? form.email : "",
      qq: form.qq,
      phone: verifyChannel.value === "sms" ? form.phone : "",
      password: form.password,
      verify_channel: verifyChannel.value,
      captcha_id: captchaId.value,
      captcha_code: form.captcha_code,
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
@import url("https://fonts.googleapis.com/css2?family=Outfit:wght@500;700&family=Noto+Sans+SC:wght@400;500;700&display=swap");

.auth-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 32px 24px;
}

.register-page {
  position: relative;
  isolation: isolate;
  overflow: hidden;
  background:
    radial-gradient(1200px 460px at 10% -10%, rgba(38, 132, 255, 0.24), transparent 70%),
    radial-gradient(900px 400px at 90% 110%, rgba(16, 185, 129, 0.2), transparent 65%),
    linear-gradient(145deg, #f5f9ff 0%, #edf6ff 45%, #f7fff9 100%);
}

.register-noise {
  position: absolute;
  inset: 0;
  background-image: radial-gradient(rgba(18, 32, 68, 0.08) 0.6px, transparent 0.6px);
  background-size: 4px 4px;
  opacity: 0.3;
  pointer-events: none;
  z-index: -2;
}

.register-orb {
  position: absolute;
  border-radius: 999px;
  filter: blur(5px);
  pointer-events: none;
  z-index: -1;
}

.register-orb-a {
  width: 320px;
  height: 320px;
  top: -120px;
  right: -60px;
  background: radial-gradient(circle at 35% 35%, rgba(37, 99, 235, 0.62), rgba(37, 99, 235, 0));
}

.register-orb-b {
  width: 280px;
  height: 280px;
  bottom: -120px;
  left: -90px;
  background: radial-gradient(circle at 45% 45%, rgba(5, 150, 105, 0.55), rgba(5, 150, 105, 0));
}

.auth-shell {
  display: grid;
  grid-template-columns: minmax(280px, 1.15fr) minmax(340px, 0.85fr);
  gap: 28px;
  width: min(1040px, 100%);
  position: relative;
  z-index: 1;
}

.auth-banner {
  border-radius: 28px;
  padding: 36px;
  color: #f8fbff;
}

.register-banner {
  position: relative;
  overflow: hidden;
  background:
    linear-gradient(150deg, rgba(19, 31, 70, 0.96) 0%, rgba(12, 20, 52, 0.92) 52%, rgba(7, 29, 44, 0.92) 100%),
    radial-gradient(600px 280px at 10% 8%, rgba(45, 114, 255, 0.38), transparent 72%);
  border: 1px solid rgba(120, 167, 255, 0.2);
  box-shadow: 0 24px 70px rgba(12, 18, 44, 0.35);
}

.register-banner::before {
  content: "";
  position: absolute;
  inset: auto -40% -52% auto;
  width: 76%;
  aspect-ratio: 1;
  border-radius: 50%;
  border: 1px solid rgba(167, 215, 255, 0.18);
  transform: rotate(-15deg);
}

.auth-logo {
  width: 52px;
  height: 52px;
  border-radius: 14px;
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 14px;
}

.register-logo {
  background: linear-gradient(140deg, #2f7bff 0%, #5ed3ff 100%);
  box-shadow: 0 10px 28px rgba(46, 129, 255, 0.45);
}

.banner-eyebrow {
  margin: 0 0 6px;
  letter-spacing: 0.22em;
  font-size: 11px;
  font-family: Outfit, "Segoe UI", sans-serif;
  color: rgba(169, 214, 255, 0.95);
}

.banner-title {
  margin: 0;
  font-family: Outfit, "Noto Sans SC", "PingFang SC", sans-serif;
  font-size: clamp(28px, 3.4vw, 38px);
  line-height: 1.12;
  letter-spacing: 0.01em;
}

.banner-desc {
  margin: 14px 0 18px;
  color: rgba(228, 244, 255, 0.9);
  line-height: 1.7;
  max-width: 38ch;
}

.banner-points {
  display: grid;
  gap: 10px;
}

.banner-point {
  position: relative;
  padding: 10px 12px 10px 34px;
  border-radius: 12px;
  background: rgba(132, 172, 255, 0.12);
  border: 1px solid rgba(165, 209, 255, 0.16);
  color: rgba(236, 247, 255, 0.98);
  backdrop-filter: blur(2px);
}

.banner-point::before {
  content: "";
  position: absolute;
  left: 13px;
  top: 50%;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  transform: translateY(-50%);
  background: linear-gradient(140deg, #55ccff, #77f3ba);
  box-shadow: 0 0 0 4px rgba(110, 230, 255, 0.16);
}

.banner-tip {
  margin-top: 16px;
  color: rgba(194, 228, 255, 0.72);
}

.auth-card {
  width: 100%;
  border-radius: 26px;
}

.register-card {
  background: rgba(255, 255, 255, 0.82);
  border: 1px solid rgba(178, 205, 255, 0.3);
  box-shadow: 0 20px 54px rgba(16, 56, 123, 0.14);
  backdrop-filter: blur(12px);
}

.register-title {
  margin-bottom: 4px;
  font-family: Outfit, "Noto Sans SC", "PingFang SC", sans-serif;
  font-size: 24px;
}

.register-subtitle {
  margin: 0 0 14px;
  color: #51607a;
}

.register-card :deep(.ant-form-item-label > label) {
  color: #1a2a49;
  font-weight: 600;
}

.register-card :deep(.ant-input),
.register-card :deep(.ant-input-affix-wrapper),
.register-card :deep(.ant-input-password),
.register-card :deep(.ant-input-number),
.register-card :deep(.ant-select-selector) {
  border-radius: 11px;
  border-color: #c7d8fb;
  background: rgba(255, 255, 255, 0.92);
}

.register-card :deep(.ant-input:focus),
.register-card :deep(.ant-input-affix-wrapper-focused),
.register-card :deep(.ant-input-password:focus),
.register-card :deep(.ant-select-focused .ant-select-selector) {
  border-color: #2a7dff;
  box-shadow: 0 0 0 2px rgba(42, 125, 255, 0.16);
}

.register-card :deep(.ant-btn-primary),
.register-card :deep(.ant-btn-default) {
  height: 42px;
  border-radius: 11px;
}

.register-card :deep(.ant-btn-primary) {
  border: none;
  background: linear-gradient(130deg, #1a74ff 0%, #2c8bff 48%, #26b4ff 100%);
  box-shadow: 0 10px 20px rgba(41, 124, 255, 0.25);
}

.auth-footer {
  margin-top: 14px;
  text-align: center;
  color: #5d6a82;
}

.captcha {
  display: flex;
  gap: 8px;
}

.verify-code {
  display: flex;
  gap: 8px;
}

.verify-code :deep(.ant-btn) {
  flex: 0 0 120px;
}

.form-inline :deep(.ant-input) {
  height: 40px;
}

.captcha-img {
  flex: 0 0 120px;
  height: 40px;
  border: 1px dashed #adc5f7;
  border-radius: 10px;
  background: rgba(250, 253, 255, 0.96);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  overflow: hidden;
}

.captcha-img img {
  height: 100%;
}

@media (max-width: 768px) {
  .auth-shell {
    grid-template-columns: 1fr;
    gap: 16px;
  }

  .auth-banner {
    order: 2;
    padding: 24px;
  }

  .banner-title {
    font-size: 30px;
  }

  .register-card {
    border-radius: 20px;
  }

  .register-page {
    padding: 20px 14px 26px;
  }

  .verify-code :deep(.ant-btn) {
    flex-basis: 112px;
  }
}
</style>
