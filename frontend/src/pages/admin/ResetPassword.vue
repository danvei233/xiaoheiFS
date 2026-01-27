<template>
  <div class="auth-page">
    <div class="auth-container">
      <div class="auth-header">
        <h1 class="auth-title">重置密码</h1>
        <p class="auth-subtitle">请输入您的新密码</p>
      </div>

      <a-form layout="vertical" @finish="handleSubmit">
        <a-form-item label="新密码" name="new_password" :rules="[{ required: true, min: 6, message: '密码至少6位' }]">
          <a-input-password v-model:value="form.new_password" placeholder="请输入新密码" size="large" />
        </a-form-item>
        <a-form-item
          label="确认密码"
          name="confirm_password"
          :rules="[
            { required: true, message: '请确认新密码' },
            { validator: validateConfirmPassword }
          ]"
        >
          <a-input-password v-model:value="form.confirm_password" placeholder="请再次输入新密码" size="large" />
        </a-form-item>
        <a-form-item>
          <a-button type="primary" html-type="submit" size="large" :loading="loading" block>
            重置密码
          </a-button>
        </a-form-item>
      </a-form>

      <div class="auth-footer">
        <router-link to="/admin/login">返回登录</router-link>
      </div>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref, onMounted } from "vue";
import { resetPassword } from "@/services/user";
import { message } from "ant-design-vue";
import { useRouter, useRoute } from "vue-router";

const router = useRouter();
const route = useRoute();
const loading = ref(false);
const token = ref("");

const form = reactive({
  new_password: "",
  confirm_password: ""
});

const validateConfirmPassword = async (_rule, value) => {
  if (value !== form.new_password) {
    return Promise.reject("两次输入的密码不一致");
  }
  return Promise.resolve();
};

const handleSubmit = async () => {
  if (!token.value) {
    message.error("无效的重置令牌");
    return;
  }

  loading.value = true;
  try {
    await resetPassword(token.value, form.new_password);
    message.success("密码已重置，请使用新密码登录");
    router.push("/admin/login");
  } catch (e) {
    message.error(e.response?.data?.error || "重置失败，令牌可能已过期");
  } finally {
    loading.value = false;
  }
};

onMounted(() => {
  token.value = route.query.token || "";
  if (!token.value) {
    message.error("缺少重置令牌");
  }
});
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