<template>
  <div class="auth-page">
    <div class="auth-shell">
      <div class="auth-banner">
        <div class="auth-logo" aria-hidden="true">
          <SiteLogoMedia :size="22" />
        </div>
        <h1>开始使用小黑云</h1>
        <p>注册后即可进入控制台，选购 VPS 并跟踪订单进度。</p>
        <div class="subtle">请确保验证码正确以完成注册。</div>
      </div>
      <a-card class="card auth-card">
        <div class="section-title">用户注册</div>
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
          <a-form-item
            label="邮箱"
            name="email"
            :rules="[{ required: isRequired('email'), message: '请输入邮箱' }]"
          >
            <a-input v-model:value="form.email" :maxlength="INPUT_LIMITS.EMAIL" />
          </a-form-item>
          <a-form-item v-if="showField('qq')" label="QQ" name="qq" :rules="[{ required: isRequired('qq'), message: '请输入QQ' }]">
            <a-input v-model:value="form.qq" :maxlength="INPUT_LIMITS.QQ" />
          </a-form-item>
          <a-form-item
            v-if="showField('phone') || settings.register_verify_type === 'sms'"
            label="手机号"
            name="phone"
            :rules="[{ required: isRequired('phone') || settings.register_verify_type === 'sms', message: '请输入手机号' }]"
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
            <div class="captcha">
              <a-input v-model:value="form.captcha_code" placeholder="验证码" />
              <div class="captcha-img" @click="refreshCaptcha">
                <img v-if="captchaImage" :src="captchaImage" alt="captcha" />
                <span v-else>点击刷新</span>
              </div>
            </div>
          </a-form-item>
          <a-form-item
            v-if="settings.register_verify_type !== 'none'"
            :label="settings.register_verify_type === 'email' ? '邮箱验证码' : '短信验证码'"
            name="verify_code"
            :rules="[{ required: true, message: '请输入验证码' }]"
          >
            <div class="verify-code">
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
import { reactive, ref, onMounted, computed } from "vue";
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
  register_required_fields: ["username", "email", "password"],
  register_verify_type: "none",
  register_captcha_enabled: true
});

const requiredSet = computed(() => new Set((settings.register_required_fields || []).map((v) => String(v).toLowerCase())));
const isRequired = (field) => requiredSet.value.has(String(field).toLowerCase());
const showField = (field) => isRequired(field) || field === "qq" || field === "phone";

const canSendCode = computed(() => {
  if (settings.register_verify_type === "email") {
    return String(form.email || "").trim().length > 0;
  }
  if (settings.register_verify_type === "sms") {
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
  } catch (error) {
    console.error("Failed to fetch auth settings:", error);
  } finally {
    refreshCaptcha();
  }
};

const sendCode = async () => {
  if (!canSendCode.value || sendCooling.value) return;
  sendCooling.value = true;
  sendCount.value = 60;
  try {
    await requestRegisterCode({
      email: form.email,
      phone: form.phone,
      captcha_id: captchaId.value,
      captcha_code: form.captcha_code
    });
    message.success("验证码已发送");
  } catch (error) {
    message.error("发送失败，请稍后重试");
  } finally {
    sendTimer = setInterval(() => {
      sendCount.value -= 1;
      if (sendCount.value <= 0) {
        clearInterval(sendTimer);
        sendTimer = null;
        sendCooling.value = false;
      }
    }, 1000);
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
      email: form.email,
      qq: form.qq,
      phone: form.phone,
      password: form.password,
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
.auth-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
}

.auth-shell {
  display: grid;
  grid-template-columns: minmax(260px, 1fr) minmax(320px, 400px);
  gap: 24px;
  width: min(960px, 100%);
}

.auth-banner {
  background: linear-gradient(135deg, #f0f6ff 0%, #ffffff 70%);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: 32px;
  box-shadow: var(--shadow-sm);
}

.auth-logo {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  background: linear-gradient(135deg, #1677ff, #6ea4ff);
  color: #fff;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 16px;
}

.auth-card {
  width: 100%;
  border-radius: var(--radius-lg);
}

.auth-footer {
  margin-top: 12px;
  text-align: center;
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

.captcha-img {
  flex: 0 0 120px;
  height: 32px;
  border: 1px dashed var(--border);
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
  }

  .auth-banner {
    order: 2;
  }
}
</style>
