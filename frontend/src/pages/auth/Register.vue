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
        <a-form :model="form" layout="vertical" @finish="onSubmit">
          <a-form-item label="用户名" name="username" :rules="[{ required: true, message: '请输入用户名' }]">
            <a-input v-model:value="form.username" />
          </a-form-item>
          <a-form-item label="邮箱" name="email" :rules="[{ required: true, message: '请输入邮箱' }]">
            <a-input v-model:value="form.email" />
          </a-form-item>
          <a-form-item label="QQ" name="qq">
            <a-input v-model:value="form.qq" />
          </a-form-item>
          <a-form-item label="密码" name="password" :rules="[{ required: true, message: '请输入密码' }]">
            <a-input-password v-model:value="form.password" />
          </a-form-item>
          <a-form-item label="图形验证码" name="captcha_code" :rules="[{ required: true, message: '请输入验证码' }]">
            <div class="captcha">
              <a-input v-model:value="form.captcha_code" placeholder="验证码" />
              <div class="captcha-img" @click="refreshCaptcha">
                <img v-if="captchaImage" :src="captchaImage" alt="captcha" />
                <span v-else>点击刷新</span>
              </div>
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
import { reactive, ref, onMounted } from "vue";
import { message } from "ant-design-vue";
import { getCaptcha, userRegister } from "@/services/user";
import { useRouter } from "vue-router";
import SiteLogoMedia from "@/components/brand/SiteLogoMedia.vue";

const router = useRouter();
const loading = ref(false);
const captchaId = ref("");
const captchaImage = ref("");

const form = reactive({
  username: "",
  email: "",
  qq: "",
  password: "",
  captcha_code: ""
});

const refreshCaptcha = async () => {
  const res = await getCaptcha();
  captchaId.value = res.data?.captcha_id || "";
  const base64 = res.data?.image_base64 || "";
  captchaImage.value = base64 ? `data:image/png;base64,${base64}` : "";
};

const onSubmit = async () => {
  loading.value = true;
  try {
    await userRegister({
      username: form.username,
      email: form.email,
      qq: form.qq,
      password: form.password,
      captcha_id: captchaId.value,
      captcha_code: form.captcha_code
    });
    message.success("注册成功，请登录");
    router.replace("/login");
  } finally {
    loading.value = false;
  }
};

onMounted(refreshCaptcha);
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
