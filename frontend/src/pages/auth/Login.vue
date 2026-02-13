<template>
  <div class="login-page">
    <div class="login-card">
      <h1 class="title">用户登录</h1>
      <p class="subtitle">登录以访问控制台</p>

      <a-form :model="form" layout="vertical" @finish="onSubmit">
        <a-form-item label="账号" name="username" :rules="[{ required: true, message: '请输入账号' }]">
          <a-input v-model:value="form.username" placeholder="请输入用户名" size="large" :maxlength="INPUT_LIMITS.EMAIL" />
        </a-form-item>

        <a-form-item label="密码" name="password" :rules="[{ required: true, message: '请输入密码' }]">
          <a-input-password v-model:value="form.password" placeholder="请输入密码" size="large" :maxlength="INPUT_LIMITS.PASSWORD" />
        </a-form-item>

        <a-form-item
          v-if="settings.login_captcha_enabled"
          label="图形验证码"
          name="captcha_code"
          :rules="[{ required: true, message: '请输入验证码' }]"
        >
          <div class="captcha-row">
            <a-input v-model:value="form.captcha_code" placeholder="验证码" />
            <div class="captcha-img" @click="refreshCaptcha">
              <img v-if="captchaImage" :src="captchaImage" alt="captcha" />
              <span v-else>点击刷新</span>
            </div>
          </div>
        </a-form-item>

        <div class="actions-row">
          <router-link to="/forgot-password">忘记密码？</router-link>
          <router-link to="/register">立即注册</router-link>
        </div>

        <a-button type="primary" html-type="submit" block size="large" :loading="auth.loading">
          登录
        </a-button>
      </a-form>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref, onMounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import { message } from "ant-design-vue";
import { useAuthStore } from "@/stores/auth";
import { getAuthSettings, getCaptcha } from "@/services/user";
import { INPUT_LIMITS } from "@/constants/inputLimits";

const form = reactive({
  username: "",
  password: "",
  captcha_code: ""
});

const auth = useAuthStore();
const router = useRouter();
const route = useRoute();

const settings = reactive({
  login_captcha_enabled: false
});

const captchaId = ref("");
const captchaImage = ref("");

const refreshCaptcha = async () => {
  if (!settings.login_captcha_enabled) return;
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

const onSubmit = async () => {
  try {
    if (String(form.username || "").length > INPUT_LIMITS.EMAIL) {
      message.error(`账号长度不能超过 ${INPUT_LIMITS.EMAIL} 个字符`);
      return;
    }
    if (String(form.password || "").length > INPUT_LIMITS.PASSWORD) {
      message.error(`密码长度不能超过 ${INPUT_LIMITS.PASSWORD} 个字符`);
      return;
    }
    const token = await auth.login({
      username: form.username,
      password: form.password,
      captcha_id: captchaId.value,
      captcha_code: form.captcha_code
    });
    if (!token) {
      message.error("登录失败");
      return;
    }
    await auth.fetchMe();
    message.success("登录成功");
    router.replace(String(route.query.redirect || "/console"));
  } catch (error) {
    const msg =
      error?.response?.data?.error ||
      error?.response?.data?.message ||
      error?.message ||
      "登录失败";
    message.error(msg);
  }
};

onMounted(() => {
  loadSettings();
});
</script>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background: #0f172a;
}

.login-card {
  width: 100%;
  max-width: 420px;
  background: #111827;
  border: 1px solid #1f2937;
  border-radius: 14px;
  padding: 24px;
}

.title {
  color: #f8fafc;
  margin: 0;
}

.subtitle {
  color: #94a3b8;
  margin: 8px 0 18px;
}

.captcha-row {
  display: flex;
  gap: 10px;
}

.captcha-img {
  width: 120px;
  height: 40px;
  border: 1px dashed #334155;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  overflow: hidden;
  color: #94a3b8;
}

.captcha-img img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.actions-row {
  display: flex;
  justify-content: space-between;
  margin-bottom: 14px;
}
</style>
