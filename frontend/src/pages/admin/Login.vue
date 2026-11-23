<template>
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
</template>

<script setup>
import { reactive } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useAdminAuthStore } from "@/stores/adminAuth";
import { message } from "ant-design-vue";
import SiteLogoMedia from "@/components/brand/SiteLogoMedia.vue";
import { buildAdminUrl } from "@/services/adminPath";

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
  padding: 32px 24px;
  background: var(--theme-canvas, var(--bg-primary));
  color: var(--text-primary);
}

.auth-shell {
  display: grid;
  grid-template-columns: minmax(260px, 1fr) minmax(320px, 380px);
  gap: 20px;
  width: min(900px, 100%);
}

.auth-banner {
  padding: 32px;
  border: 1px solid color-mix(in srgb, var(--primary) 22%, var(--border));
  border-radius: var(--theme-card-radius, 16px);
  background: color-mix(in srgb, var(--primary) 9%, var(--card));
  color: var(--text-primary);
}

.auth-logo {
  display: flex;
  width: 48px;
  height: 48px;
  align-items: center;
  justify-content: center;
  margin-bottom: 16px;
  border-radius: var(--theme-control-radius, 12px);
  background: var(--primary);
  color: var(--theme-primary-foreground, #fff);
  font-weight: 700;
}

.auth-banner h1 {
  margin: 0 0 8px;
  color: var(--text-primary);
  font-size: 22px;
  font-weight: 650;
  letter-spacing: -.01em;
}

.auth-banner p {
  margin: 0 0 18px;
  color: var(--text-secondary);
  font-size: 13px;
  line-height: 1.7;
}

.auth-banner ul {
  margin: 0;
  padding-left: 18px;
  color: var(--text-secondary);
  font-size: 13px;
  line-height: 1.9;
}

.auth-banner li::marker {
  color: var(--primary);
}

.auth-card {
  width: 100%;
  border: 1px solid var(--border);
  border-radius: var(--theme-card-radius, 16px);
  background: var(--theme-surface, var(--card));
  box-shadow: var(--theme-shadow, none);
}

.auth-card .section-title {
  margin-bottom: 18px;
  color: var(--text-primary);
  font-size: 16px;
  font-weight: 650;
}

@media (max-width: 768px) {
  .auth-page {
    padding: 20px 16px;
  }

  .auth-shell {
    grid-template-columns: 1fr;
  }

  .auth-banner {
    order: 2;
    padding: 24px;
  }
}
</style>
