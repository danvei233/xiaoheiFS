<template>
  <div class="step">
    <div class="step-header">
      <div class="header-content">
        <h2 class="step-title">创建管理员</h2>
        <p class="step-subtitle">设置超级管理员账号，用于登录管理后台</p>
      </div>
      <div class="step-badge">3/4</div>
    </div>

    <a-form layout="vertical" :model="form" @finish="onSubmit">
      <div class="config-card">
        <div class="card-header">
          <div class="card-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
              <circle cx="12" cy="7" r="4"/>
            </svg>
          </div>
          <div class="card-title">管理员信息</div>
        </div>
        <div class="card-body">
          <a-form-item
            label="用户名"
            name="adminUser"
            :rules="[{ required: true, message: '请输入管理员用户名' }]"
          >
            <a-input
              v-model:value="form.adminUser"
              size="large"
              placeholder="例如：admin"
              class="styled-input"
            />
          </a-form-item>
          <a-form-item
            label="密码"
            name="adminPass"
            :rules="[
              { required: true, message: '请输入管理员密码' },
              { min: 6, message: '密码至少 6 位' }
            ]"
          >
            <a-input-password
              v-model:value="form.adminPass"
              size="large"
              placeholder="至少 6 位"
              class="styled-input"
            />
          </a-form-item>
          <a-form-item
            label="确认密码"
            name="adminPass2"
            :rules="[
              { required: true, message: '请再次输入密码' },
              { validator: validateConfirm }
            ]"
          >
            <a-input-password
              v-model:value="form.adminPass2"
              size="large"
              placeholder="再次输入密码"
              class="styled-input"
            />
          </a-form-item>
        </div>
      </div>

      <div class="config-card">
        <div class="card-header">
          <div class="card-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>
            </svg>
          </div>
          <div class="card-title">管理端路径</div>
        </div>
        <div class="card-body">
          <a-form-item
            label="自定义管理端访问路径"
            name="adminPath"
            :rules="[{ validator: validateAdminPath }]"
          >
            <a-input
              v-model:value="form.adminPath"
              size="large"
              placeholder="留空使用默认路径 /admin"
              class="styled-input"
            >
              <template #addonAfter>
                <a-button type="link" size="small" @click="generateAdminPath" :loading="generating">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width: 14px; height: 14px;">
                    <polyline points="23 4 23 10 17 10"/>
                    <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/>
                  </svg>
                  随机生成
                </a-button>
              </template>
            </a-input>
            <div class="hint-text">
              仅允许字母和数字，不可与 login/admin/api 等保留路径重复
            </div>
          </a-form-item>
        </div>
      </div>

      <!-- Warning for incomplete DB check -->
      <a-alert
        v-if="!wiz.dbChecked"
        class="warning-alert"
        type="warning"
        show-icon
        message="请先完成数据库连接测试"
      >
        <template #icon>
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width: 16px; height: 16px;">
            <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/>
            <line x1="12" y1="9" x2="12" y2="13"/>
            <line x1="12" y1="17" x2="12.01" y2="17"/>
          </svg>
        </template>
      </a-alert>

      <!-- Action Bar -->
      <div class="action-bar">
        <div class="info-box">
          <div class="info-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/>
              <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
            </svg>
          </div>
          <div class="info-content">
            <div class="info-title">安全提示</div>
            <div class="info-text">管理员账号拥有系统最高权限，请妥善保管密码和管理端路径</div>
          </div>
        </div>
        <div class="button-group">
          <a-button size="large" @click="back" class="action-btn secondary">
            <template #icon>
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width: 16px; height: 16px;">
                <polyline points="15 18 9 12 15 6"/>
              </svg>
            </template>
            上一步
          </a-button>
          <a-button
            type="primary"
            html-type="submit"
            size="large"
            :loading="submitting"
            :disabled="!wiz.dbChecked"
            class="action-btn primary"
          >
            <template #icon>
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width: 16px; height: 16px;">
                <polygon points="5 3 19 12 5 21 5 3"/>
              </svg>
            </template>
            开始安装
          </a-button>
        </div>
      </div>
    </a-form>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, watch, onMounted } from "vue";
import { message } from "ant-design-vue";
import type { RuleObject } from "ant-design-vue/es/form";
import type { StoreValue } from "ant-design-vue/es/form/interface";
import { runInstall } from "@/services/user";
import { useInstallStore } from "@/stores/install";
import { useInstallWizardStore } from "@/stores/installWizard";
import { http } from "@/services/http";

const emit = defineEmits<{
  next: [adminPath: string, restart: boolean, configFile: string]
  back: []
}>();

const install = useInstallStore();
const wiz = useInstallWizardStore();

const submitting = ref(false);
const generating = ref(false);
const form = reactive({
  adminUser: wiz.adminUser,
  adminPass: "",
  adminPass2: "",
  adminPath: wiz.adminPath || ""
});

watch(
  () => form.adminUser,
  (v) => {
    wiz.adminUser = v;
    wiz.persist();
  }
);

watch(
  () => form.adminPath,
  (v) => {
    wiz.adminPath = v;
    wiz.persist();
  }
);

const validateConfirm = async (_rule: RuleObject, value: StoreValue) => {
  const v = String(value || "");
  if (!v) return Promise.resolve();
  if (v !== form.adminPass) {
    return Promise.reject(new Error("两次输入的密码不一致"));
  }
  return Promise.resolve();
};

const validateAdminPath = async (_rule: RuleObject, value: StoreValue) => {
  const v = String(value || "").trim();
  if (!v) return Promise.resolve(); // 允许为空
  
  // 前端校验：只允许字母和数字
  if (!/^[a-zA-Z0-9]+$/.test(v)) {
    return Promise.reject(new Error("仅允许字母和数字"));
  }
  
  // 黑名单检查
  const reserved = ["login", "admin", "api", "install", "console", "register", "assets", "uploads"];
  if (reserved.includes(v.toLowerCase())) {
    return Promise.reject(new Error("该路径为保留路径，请更换"));
  }
  
  return Promise.resolve();
};

const generateAdminPath = async () => {
  generating.value = true;
  try {
    const res = await http.get<{ admin_path: string }>("/api/v1/install/generate-admin-path");
    if (res.data?.admin_path) {
      form.adminPath = res.data.admin_path;
      message.success("已生成随机路径");
    }
  } catch (error: any) {
    message.error(error.response?.data?.error || "生成失败");
  } finally {
    generating.value = false;
  }
};

const back = () => {
  emit('back');
};

const onSubmit = async () => {
  submitting.value = true;
  try {
    wiz.adminPass = form.adminPass;
    wiz.adminPath = form.adminPath;
    wiz.persist();
    const payload =
      wiz.dbType === "sqlite"
        ? { db: { type: "sqlite", path: wiz.sqlitePath } }
        : { db: { type: "mysql", dsn: wiz.mysqlDSN } };

    const res = await runInstall({
      ...payload,
      site: { name: wiz.siteName, url: wiz.siteUrl, admin_path: form.adminPath },
      admin: { username: form.adminUser, password: form.adminPass }
    });

    await install.fetchStatus();
    
    // 缓存管理端路径到 localStorage
    const finalAdminPath = form.adminPath || "admin";
    localStorage.setItem("admin_path_cache", finalAdminPath);
    
    message.success("安装完成");

    emit('next', 
      finalAdminPath, 
      res.data?.restart_required || false, 
      res.data?.config_file || ""
    );
  } catch (error: any) {
    message.error(error.response?.data?.error || "安装失败");
  } finally {
    submitting.value = false;
  }
};

onMounted(async () => {
  // 每次都自动生成一个新的随机路径
  await generateAdminPath();
});
</script>

<style scoped>
.step {
  padding: 8px;
}

/* Step Header */
.step-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 24px;
}

.header-content {
  flex: 1;
}

.step-title {
  font-size: 22px;
  font-weight: 800;
  color: #ffffff;
  margin: 0 0 6px 0;
  letter-spacing: -0.01em;
}

.step-subtitle {
  font-size: 14px;
  color: rgba(255, 255, 255, 0.5);
  margin: 0;
}

.step-badge {
  flex-shrink: 0;
  padding: 6px 12px;
  border-radius: 8px;
  background: rgba(34, 211, 238, 0.15);
  border: 1px solid rgba(34, 211, 238, 0.3);
  font-size: 12px;
  font-weight: 700;
  color: #22d3ee;
}

/* Config Card */
.config-card {
  border-radius: 0;
  background: rgba(10, 16, 28, 0.3);
  border-top: 1px solid rgba(71, 85, 105, 0.5);
  border-bottom: 1px solid rgba(71, 85, 105, 0.5);
  overflow: hidden;
  margin-bottom: 16px;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 16px;
  background: rgba(15, 23, 42, 0.2);
  border-bottom: 1px solid rgba(71, 85, 105, 0.3);
}

.card-icon {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(34, 211, 238, 0.15);
  color: #22d3ee;
}

.card-icon svg {
  width: 16px;
  height: 16px;
}

.card-title {
  font-size: 14px;
  font-weight: 700;
  color: #ffffff;
}

.card-body {
  padding: 18px 16px;
}

/* Styled Input Override */
:deep(.styled-input .ant-input),
:deep(.styled-input .ant-input-password) {
  background: rgba(15, 23, 42, 0.6);
  border-color: rgba(71, 85, 105, 0.5);
  color: #ffffff;
}

:deep(.styled-input .ant-input::placeholder) {
  color: rgba(255, 255, 255, 0.3);
}

:deep(.styled-input .ant-input:hover),
:deep(.styled-input .ant-input-password:hover) {
  border-color: rgba(34, 211, 238, 0.5);
}

:deep(.styled-input .ant-input:focus),
:deep(.styled-input .ant-input-password:focus),
:deep(.styled-input .ant-input-password-focused) {
  border-color: #22d3ee;
  box-shadow: 0 0 0 2px rgba(34, 211, 238, 0.1);
}

:deep(.styled-input .ant-input-password-icon) {
  color: rgba(255, 255, 255, 0.5);
}

:deep(.styled-input .ant-input-password-icon:hover) {
  color: #22d3ee;
}

/* Input Group Addon (随机生成按钮) */
:deep(.styled-input .ant-input-group-addon) {
  background: rgba(15, 23, 42, 0.6);
  border-color: rgba(71, 85, 105, 0.5);
  padding: 0;
}

:deep(.styled-input .ant-input-group-addon .ant-btn-link) {
  height: 38px;
  padding: 0 12px;
  color: #22d3ee;
  display: flex;
  align-items: center;
  gap: 6px;
  border: none;
  background: transparent;
}

:deep(.styled-input .ant-input-group-addon .ant-btn-link:hover) {
  color: #06b6d4;
  background: rgba(34, 211, 238, 0.1);
}

:deep(.styled-input .ant-input-group-addon .ant-btn-link svg) {
  flex-shrink: 0;
}

/* Input Labels */
:deep(.ant-form-item-label > label) {
  color: rgba(255, 255, 255, 0.7);
  font-size: 13px;
  font-weight: 500;
}

/* Hint Text */
.hint-text {
  margin-top: 6px;
  font-size: 12px;
  color: rgba(255, 255, 255, 0.65);
  line-height: 1.5;
}

/* Warning Alert */
.warning-alert {
  margin-bottom: 16px;
  border-radius: 4px;
  background: rgba(251, 191, 36, 0.1);
  border: 1px solid rgba(251, 191, 36, 0.3);
}

:deep(.warning-alert .ant-alert-icon) {
  color: #fbbf24;
}

:deep(.warning-alert .ant-alert-message) {
  color: #fbbf24;
}

/* Action Bar */
.action-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.info-box {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 12px 14px;
  border-radius: 4px;
  background: rgba(34, 211, 238, 0.05);
  border: 1px solid rgba(34, 211, 238, 0.15);
  max-width: 340px;
}

.info-icon {
  flex-shrink: 0;
  width: 28px;
  height: 28px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(34, 211, 238, 0.15);
  color: #22d3ee;
}

.info-icon svg {
  width: 14px;
  height: 14px;
}

.info-content {
  flex: 1;
  min-width: 0;
}

.info-title {
  font-size: 13px;
  font-weight: 700;
  color: #ffffff;
  margin-bottom: 2px;
}

.info-text {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.6);
  line-height: 1.4;
}

.button-group {
  display: flex;
  gap: 10px;
  flex-shrink: 0;
}

.action-btn {
  height: 44px;
  padding: 0 20px;
  border-radius: 4px;
  font-size: 14px;
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 8px;
}

.action-btn.secondary {
  background: rgba(30, 41, 59, 0.8);
  border-color: rgba(71, 85, 105, 0.5);
  color: #ffffff;
}

.action-btn.secondary:hover {
  background: rgba(30, 41, 59, 1);
  border-color: rgba(34, 211, 238, 0.5);
}

.action-btn.primary {
  background: linear-gradient(135deg, #22d3ee 0%, #06b6d4 100%);
  border: none;
  color: #0f172a;
}

.action-btn.primary:hover:not(:disabled) {
  box-shadow: 0 4px 20px rgba(34, 211, 238, 0.4);
  transform: translateY(-1px);
}

.action-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Responsive */
@media (max-width: 720px) {
  .action-bar {
    flex-direction: column;
    align-items: stretch;
  }

  .info-box {
    max-width: none;
  }

  .button-group {
    width: 100%;
  }

  .button-group :deep(.ant-btn) {
    flex: 1;
  }
}
</style>
