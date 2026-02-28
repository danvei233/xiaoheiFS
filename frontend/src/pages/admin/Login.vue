<template>
  <a-config-provider :theme="{ algorithm: theme.darkAlgorithm }">
    <div class="auth-page">
      <div class="auth-shell">
        <div class="auth-banner">
          <div class="auth-logo" aria-hidden="true">
            <SiteLogoMedia :size="32" />
          </div>
          <h1>运营管理后台</h1>
          <p>面向运营与审核的管理中心，实时掌控订单与资源。</p>
          <ul>
            <li>订单审核与开通进度追踪</li>
            <li>售卖配置与系统镜像管理</li>
            <li>自动化平台同步与审计日志</li>
          </ul>
        </div>
        <a-card class="card auth-card">
          <div class="section-title">管理员登录</div>
          <a-form :model="form" layout="vertical" @finish="onSubmit">
            <a-form-item label="用户名" name="username" :rules="[{ required: true, message: '请输入用户名' }]">
              <a-input v-model:value="form.username" />
            </a-form-item>
            <a-form-item label="密码" name="password" :rules="[{ required: true, message: '请输入密码' }]">
              <a-input-password v-model:value="form.password" />
            </a-form-item>
            <div style="text-align: right; margin-bottom: 16px;">
              <a href="#" @click.prevent="goToForgotPassword">忘记密码？</a>
            </div>
            <a-button type="primary" html-type="submit" block :loading="admin.loading">登录</a-button>
          </a-form>
        </a-card>
      </div>
    </div>
  </a-config-provider>
</template>

<script setup>
import { reactive, onMounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useAdminAuthStore } from "@/stores/adminAuth";
import { message, theme } from "ant-design-vue";
import SiteLogoMedia from "@/components/brand/SiteLogoMedia.vue";
import { buildAdminUrl, checkAdminPath, fetchAdminPath } from "@/services/adminPath";

const form = reactive({
  username: "",
  password: ""
});

const admin = useAdminAuthStore();
const router = useRouter();
const route = useRoute();

// 获取当前管理端路径
const getCurrentAdminPath = () => {
  const pathSegments = route.path.split("/").filter(Boolean);
  return pathSegments[0] || "admin";
};

// 验证当前路径是否正确
onMounted(async () => {
  const currentPath = getCurrentAdminPath();
  try {
    const result = await checkAdminPath(currentPath);
    if (!result.isAdmin) {
      // 当前路径不是管理路径，跳转到正确的路径
      message.warning("管理路径已更改，正在跳转...");
      const correctPath = await fetchAdminPath();
      router.replace(`/${correctPath}/login`);
    }
  } catch (error) {
    console.error("Failed to validate admin path:", error);
  }
});

const onSubmit = async () => {
  try {
    // 获取当前管理端路径并传递给登录接口
    const adminPath = getCurrentAdminPath();
    const token = await admin.login({
      ...form,
      admin_path: adminPath
    });
    if (!token) {
      message.error("登录失败");
      return;
    }
    
    // 使用当前的管理端路径构建跳转URL
    const redirectPath = String(route.query.redirect || `/${adminPath}/console`);
    
    // 使用 window.location 强制刷新页面，确保新的管理端路径配置生效
    window.location.href = redirectPath;
  } catch (error) {
    const errorMsg = error?.response?.data?.error || error?.response?.data?.message || error?.message;
    
    // 处理路径验证相关错误
    if (errorMsg?.includes("admin path")) {
      message.error("管理路径验证失败，请检查访问地址");
    } else if (error?.response?.status === 403) {
      message.error("访问被拒绝，请确认管理路径正确");
    } else {
      message.error(errorMsg || "登录失败");
    }
  }
};

const goToForgotPassword = () => {
  router.push(buildAdminUrl("forgot-password"));
};
</script>

<style scoped>
.auth-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background: radial-gradient(1200px 500px at 10% -10%, rgba(56, 189, 248, 0.18), transparent 65%),
    radial-gradient(900px 400px at 100% 0, rgba(99, 102, 241, 0.2), transparent 70%),
    #0b1220;
}

.auth-shell {
  display: grid;
  grid-template-columns: minmax(260px, 1fr) minmax(320px, 380px);
  gap: 24px;
  width: min(960px, 100%);
}

.auth-banner {
  background: linear-gradient(135deg, rgba(30, 41, 59, 0.92) 0%, rgba(15, 23, 42, 0.88) 100%);
  border: 1px solid #334155;
  border-radius: var(--radius-lg);
  padding: 32px;
  box-shadow: 0 14px 40px rgba(0, 0, 0, 0.3);
  color: #e2e8f0;
}

.auth-logo {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  background: linear-gradient(135deg, #722ed1, #c08cff);
  color: #fff;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 16px;
}

.auth-banner h1 {
  margin: 0 0 8px;
  font-size: 22px;
}

.auth-banner p {
  color: #cbd5e1;
}

.auth-banner ul {
  padding-left: 18px;
  color: var(--text-secondary);
}

.auth-card {
  width: 100%;
  border-radius: var(--radius-lg);
  background: rgba(15, 23, 42, 0.82);
  border-color: #334155;
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
