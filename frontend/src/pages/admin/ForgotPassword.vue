<template>
  <div class="auth-page">
    <div class="auth-container">
      <div class="auth-header">
        <h1 class="auth-title">找回密码</h1>
        <p class="auth-subtitle">输入您的邮箱地址，我们将发送重置密码的链接</p>
      </div>

      <a-form layout="vertical" @finish="handleSubmit">
        <a-form-item label="邮箱" name="email" :rules="[{ required: true, type: 'email', message: '请输入有效的邮箱' }]">
          <a-input v-model:value="form.email" placeholder="请输入邮箱" size="large" />
        </a-form-item>
        <a-form-item>
          <a-button type="primary" html-type="submit" size="large" :loading="loading" block>
            发送重置邮件
          </a-button>
        </a-form-item>
      </a-form>

      <div class="auth-footer">
        <a href="#" @click.prevent="goToLogin">返回登录</a>
      </div>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref } from "vue";
import { forgotPassword } from "@/services/user";
import { message } from "ant-design-vue";
import { useRouter, useRoute } from "vue-router";

const router = useRouter();
const route = useRoute();
const loading = ref(false);

const form = reactive({
  email: ""
});

// 获取当前管理端路径
const getCurrentAdminPath = () => {
  const pathSegments = route.path.split("/").filter(Boolean);
  return pathSegments[0] || "admin";
};

const handleSubmit = async () => {
  loading.value = true;
  try {
    await forgotPassword(form.email);
    message.success("重置邮件已发送，请查收邮箱");
    const adminPath = getCurrentAdminPath();
    router.push(`/${adminPath}/login`);
  } catch (e) {
    message.error(e.response?.data?.error || "发送失败，请检查邮箱是否正确");
  } finally {
    loading.value = false;
  }
};

const goToLogin = () => {
  const adminPath = getCurrentAdminPath();
  router.push(`/${adminPath}/login`);
};
</script>

<style scoped>
.auth-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.auth-container {
  width: 400px;
  background: white;
  border-radius: 12px;
  padding: 40px;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.1);
}

.auth-header {
  text-align: center;
  margin-bottom: 32px;
}

.auth-title {
  font-size: 28px;
  font-weight: 600;
  color: #1f2329;
  margin: 0 0 8px;
}

.auth-subtitle {
  font-size: 14px;
  color: #6b7280;
  margin: 0;
}

.auth-footer {
  text-align: center;
  margin-top: 24px;
}

.auth-footer a {
  color: #1890ff;
  text-decoration: none;
}

.auth-footer a:hover {
  text-decoration: underline;
}
</style>