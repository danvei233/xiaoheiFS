<template>
  <div class="step">
    <div class="step-header">
      <div class="header-content">
        <h2 class="step-title">站点信息</h2>
        <p class="step-subtitle">配置网站基本名称和访问地址</p>
      </div>
      <div class="step-badge">2/4</div>
    </div>

    <a-form layout="vertical" :model="form" @finish="next">
      <div class="config-card">
        <div class="card-header">
          <div class="card-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"/>
              <line x1="2" y1="12" x2="22" y2="12"/>
              <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/>
            </svg>
          </div>
          <div class="card-title">基本设置</div>
        </div>
        <div class="card-body">
          <a-form-item
            label="站点名称"
            name="siteName"
            :rules="[{ required: true, message: '请输入站点名称' }]"
          >
            <a-input
              v-model:value="form.siteName"
              size="large"
              placeholder="例如：小黑云"
              @change="wiz.persist()"
              class="styled-input"
            />
          </a-form-item>
          <a-form-item label="站点 URL（可选）" name="siteUrl">
            <a-input
              v-model:value="form.siteUrl"
              size="large"
              placeholder="例如：https://example.com"
              @change="wiz.persist()"
              class="styled-input"
            />
            <div class="input-hint">用于生成邮件链接、API 回调等，可后续在后台修改</div>
          </a-form-item>
        </div>
      </div>

      <!-- Action Bar -->
      <div class="action-bar">
        <div class="info-box">
          <div class="info-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"/>
              <line x1="12" y1="16" x2="12" y2="12"/>
              <line x1="12" y1="8" x2="12.01" y2="8"/>
            </svg>
          </div>
          <div class="info-content">
            <div class="info-title">提示</div>
            <div class="info-text">站点名称将显示在浏览器标签页和邮件通知中</div>
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
          <a-button type="primary" html-type="submit" size="large" class="action-btn primary">
            下一步
            <template #suffix>
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width: 16px; height: 16px;">
                <polyline points="9 18 15 12 9 6"/>
              </svg>
            </template>
          </a-button>
        </div>
      </div>
    </a-form>
  </div>
</template>

<script setup lang="ts">
import { reactive, watch } from "vue";
import { useRouter, useRoute } from "vue-router";
import { useInstallWizardStore } from "@/stores/installWizard";

const router = useRouter();
const route = useRoute();
const wiz = useInstallWizardStore();

const form = reactive({
  siteName: wiz.siteName,
  siteUrl: wiz.siteUrl
});

watch(
  () => ({ ...form }),
  (v) => {
    wiz.siteName = v.siteName;
    wiz.siteUrl = v.siteUrl;
    wiz.persist();
  },
  { deep: true }
);

const back = async () => {
  const q = route.query && typeof route.query.redirect === "string" ? { redirect: route.query.redirect } : {};
  await router.push({ path: "/install/db", query: q });
};

const next = async () => {
  const q = route.query && typeof route.query.redirect === "string" ? { redirect: route.query.redirect } : {};
  await router.push({ path: "/install/admin", query: q });
};
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
  margin-bottom: 20px;
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

/* Input Labels */
:deep(.ant-form-item-label > label) {
  color: rgba(255, 255, 255, 0.7);
  font-size: 13px;
  font-weight: 500;
}

.input-hint {
  margin-top: 6px;
  font-size: 12px;
  color: rgba(255, 255, 255, 0.4);
  line-height: 1.5;
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
  max-width: 320px;
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

.action-btn.primary:hover {
  box-shadow: 0 4px 20px rgba(34, 211, 238, 0.4);
  transform: translateY(-1px);
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
