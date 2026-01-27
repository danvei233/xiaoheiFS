<template>
  <div class="step">
    <div class="step-header">
      <div class="header-content">
        <h2 class="step-title">数据库设置</h2>
        <p class="step-subtitle">选择数据库类型并配置连接信息</p>
      </div>
      <div class="step-badge">1/4</div>
    </div>

    <a-form layout="vertical">
      <!-- Database Type Selection -->
      <div class="type-selector">
        <label class="selector-label">数据库类型</label>
        <div class="type-options">
          <button
            type="button"
            class="type-option"
            :class="{ active: wiz.dbType === 'sqlite' }"
            @click="selectDbType('sqlite')"
          >
            <div class="option-icon sqlite-icon">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M21 7c0 1.5-2 2.5-4 2.5s-4-1-4-2.5 2-2.5 4-2.5 4 1 4 2.5z"/>
                <path d="M17 9.5c0 1.5-2 2.5-4 2.5s-4-1-4-2.5"/>
                <path d="M21 7V17c0 1.5-2 2.5-4 2.5s-4-1-4-2.5"/>
                <path d="M17 14.5c0 1.5-2 2.5-4 2.5s-4-1-4-2.5"/>
                <path d="M9 9.5V17"/>
              </svg>
            </div>
            <div class="option-content">
              <div class="option-title">SQLite</div>
              <div class="option-desc">内置数据库，零配置</div>
            </div>
          </button>
          <button
            type="button"
            class="type-option"
            :class="{ active: wiz.dbType === 'mysql' }"
            @click="selectDbType('mysql')"
          >
            <div class="option-icon mysql-icon">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <ellipse cx="12" cy="5" rx="8" ry="3"/>
                <path d="M4 5v14c0 1.66 3.58 3 8 3s8-1.34 8-3V5"/>
              </svg>
            </div>
            <div class="option-content">
              <div class="option-title">MySQL</div>
              <div class="option-desc">外部数据库服务器</div>
            </div>
          </button>
        </div>
      </div>

      <!-- SQLite Configuration -->
      <div v-if="wiz.dbType === 'sqlite'" class="config-card sqlite-card">
        <div class="card-header">
          <div class="card-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
              <polyline points="14 2 14 8 20 8"/>
              <path d="M12 18v-6"/>
              <path d="M9 15l3 3 3-3"/>
            </svg>
          </div>
          <div class="card-title">SQLite 配置</div>
        </div>
        <div class="card-body">
          <a-form-item label="文件路径">
            <a-input
              v-model:value="wiz.sqlitePath"
              size="large"
              placeholder="./data/app.db"
              @change="wiz.touchDB()"
              class="styled-input"
            />
            <div class="input-hint">相对路径默认以服务运行目录为基准，建议放在 ./data/ 下</div>
          </a-form-item>
        </div>
      </div>

      <!-- MySQL Configuration -->
      <div v-else class="config-card mysql-card">
        <div class="card-header">
          <div class="card-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <ellipse cx="12" cy="5" rx="8" ry="3"/>
              <path d="M4 5v14c0 1.66 3.58 3 8 3s8-1.34 8-3V5"/>
            </svg>
          </div>
          <div class="card-title">MySQL 连接配置</div>
        </div>
        <div class="card-body">
          <div class="form-grid">
            <a-form-item label="主机">
              <a-input v-model:value="wiz.mysql.host" size="large" placeholder="127.0.0.1" @change="wiz.touchDB()" class="styled-input" />
            </a-form-item>
            <a-form-item label="端口">
              <a-input-number
                v-model:value="wiz.mysql.port"
                size="large"
                :min="1"
                :max="65535"
                style="width: 100%"
                @change="wiz.touchDB()"
                class="styled-input"
              />
            </a-form-item>
            <a-form-item label="用户名">
              <a-input v-model:value="wiz.mysql.user" size="large" placeholder="root" @change="wiz.touchDB()" class="styled-input" />
            </a-form-item>
            <a-form-item label="密码">
              <a-input-password v-model:value="wiz.mysql.pass" size="large" @change="wiz.touchDB()" class="styled-input" />
            </a-form-item>
            <a-form-item label="数据库名" class="span-full">
              <a-input v-model:value="wiz.mysql.dbName" size="large" placeholder="xiaohei" @change="wiz.touchDB()" class="styled-input" />
            </a-form-item>
            <a-form-item label="连接参数（可选）" class="span-full">
              <a-input
                v-model:value="wiz.mysql.params"
                size="large"
                placeholder="charset=utf8mb4&parseTime=True&loc=Local"
                @change="wiz.touchDB()"
                class="styled-input"
              />
            </a-form-item>
          </div>

          <!-- DSN Preview -->
          <div class="dsn-preview">
            <div class="dsn-header">
              <span class="dsn-label">DSN（自动生成）</span>
              <button type="button" class="copy-btn" @click="copyDSN" title="复制 DSN">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <rect x="9" y="9" width="13" height="13" rx="2" ry="2"/>
                  <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>
                </svg>
              </button>
            </div>
            <div class="dsn-content">{{ wiz.mysqlDSN }}</div>
          </div>
        </div>
      </div>

      <!-- Action Bar -->
      <div class="action-bar">
        <div class="status-area">
          <div v-if="wiz.dbCheckError" class="status-badge error">
            <div class="badge-icon">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10"/>
                <path d="M15 9l-6 6m0-6l6 6"/>
              </svg>
            </div>
            <div class="badge-content">
              <div class="badge-title">连接失败</div>
              <div class="badge-text">{{ wiz.dbCheckError }}</div>
            </div>
          </div>
          <div v-else-if="wiz.dbChecked" class="status-badge success">
            <div class="badge-icon">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/>
                <polyline points="22 4 12 14.01 9 11.01"/>
              </svg>
            </div>
            <div class="badge-content">
              <div class="badge-title">连接测试通过</div>
              <div class="badge-text">数据库连接正常</div>
            </div>
          </div>
        </div>
        <div class="button-group">
          <a-button size="large" :loading="checking" @click="onCheck" class="action-btn secondary">
            <template #icon>
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width: 16px; height: 16px;">
                <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/>
                <polyline points="22 4 12 14.01 9 11.01"/>
              </svg>
            </template>
            测试连接
          </a-button>
          <a-button type="primary" size="large" :disabled="!wiz.dbChecked" @click="next" class="action-btn primary">
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
import { ref } from "vue";
import { message } from "ant-design-vue";
import { useRouter, useRoute } from "vue-router";
import { checkInstallDB } from "@/services/user";
import { useInstallWizardStore } from "@/stores/installWizard";

const router = useRouter();
const route = useRoute();
const wiz = useInstallWizardStore();

const checking = ref(false);

const selectDbType = (type: "sqlite" | "mysql") => {
  wiz.dbType = type;
  wiz.touchDB();
};

const copyDSN = () => {
  navigator.clipboard.writeText(wiz.mysqlDSN);
  message.success("DSN 已复制到剪贴板");
};

const onCheck = async () => {
  wiz.persist();
  checking.value = true;
  try {
    const payload =
      wiz.dbType === "sqlite"
        ? { db: { type: "sqlite", path: wiz.sqlitePath } }
        : { db: { type: "mysql", dsn: wiz.mysqlDSN } };

    const res = await checkInstallDB(payload);
    if (res.data?.ok) {
      wiz.markDBChecked(true);
      message.success("数据库连接正常");
    } else {
      wiz.markDBChecked(false, res.data?.error || "unknown error");
    }
  } catch (e: any) {
    wiz.markDBChecked(false, e?.response?.data?.error || e?.message || "unknown error");
  } finally {
    checking.value = false;
  }
};

const next = async () => {
  const q = route.query && typeof route.query.redirect === "string" ? { redirect: route.query.redirect } : {};
  await router.push({ path: "/install/site", query: q });
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

/* Type Selector */
.type-selector {
  margin-bottom: 20px;
}

.selector-label {
  display: block;
  font-size: 13px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.7);
  margin-bottom: 12px;
}

.type-options {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 12px;
}

.type-option {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px;
  border-radius: 14px;
  background: rgba(30, 41, 59, 0.5);
  border: 2px solid rgba(71, 85, 105, 0.5);
  cursor: pointer;
  transition: all 0.3s ease;
  text-align: left;
}

.type-option:hover {
  border-color: rgba(34, 211, 238, 0.5);
  background: rgba(30, 41, 59, 0.8);
}

.type-option.active {
  border-color: #22d3ee;
  background: rgba(34, 211, 238, 0.1);
  box-shadow: 0 0 20px rgba(34, 211, 238, 0.2);
}

.option-icon {
  flex-shrink: 0;
  width: 40px;
  height: 40px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(34, 211, 238, 0.15);
  color: #22d3ee;
}

.option-icon svg {
  width: 20px;
  height: 20px;
}

.option-content {
  flex: 1;
  min-width: 0;
}

.option-title {
  font-size: 14px;
  font-weight: 700;
  color: #ffffff;
  margin-bottom: 2px;
}

.option-desc {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.5);
}

/* Config Card */
.config-card {
  border-radius: 16px;
  background: rgba(30, 41, 59, 0.4);
  border: 1px solid rgba(71, 85, 105, 0.5);
  overflow: hidden;
  margin-bottom: 20px;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px 18px;
  background: rgba(15, 23, 42, 0.5);
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
  padding: 18px;
}

/* Form Grid */
.form-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 14px;
}

.span-full {
  grid-column: 1 / -1;
}

/* Styled Input Override */
:deep(.styled-input .ant-input),
:deep(.styled-input .ant-input-number),
:deep(.styled-input .ant-input-password) {
  background: rgba(15, 23, 42, 0.6);
  border-color: rgba(71, 85, 105, 0.5);
  color: #ffffff;
}

:deep(.styled-input .ant-input::placeholder),
:deep(.styled-input .ant-input input::placeholder) {
  color: rgba(255, 255, 255, 0.3);
}

:deep(.styled-input .ant-input:hover),
:deep(.styled-input .ant-input-number:hover),
:deep(.styled-input .ant-input-password:hover) {
  border-color: rgba(34, 211, 238, 0.5);
}

:deep(.styled-input .ant-input:focus),
:deep(.styled-input .ant-input-number:focus),
:deep(.styled-input .ant-input-password:focus),
:deep(.styled-input .ant-input-number-focused),
:deep(.styled-input .ant-input-password-focused) {
  border-color: #22d3ee;
  box-shadow: 0 0 0 2px rgba(34, 211, 238, 0.1);
}

:deep(.styled-input .ant-input-number-handler-wrap) {
  background: rgba(15, 23, 42, 0.6);
}

:deep(.styled-input .ant-input-number-handler) {
  color: rgba(255, 255, 255, 0.5);
}

:deep(.styled-input .ant-input-number-handler:hover) {
  color: #22d3ee;
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

/* DSN Preview */
.dsn-preview {
  margin-top: 14px;
  padding: 14px;
  border-radius: 10px;
  background: rgba(15, 23, 42, 0.6);
  border: 1px solid rgba(71, 85, 105, 0.3);
}

.dsn-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.dsn-label {
  font-size: 12px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.6);
}

.copy-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  padding: 0;
  border: none;
  border-radius: 6px;
  background: rgba(34, 211, 238, 0.15);
  color: #22d3ee;
  cursor: pointer;
  transition: all 0.2s ease;
}

.copy-btn:hover {
  background: rgba(34, 211, 238, 0.25);
}

.copy-btn svg {
  width: 14px;
  height: 14px;
}

.dsn-content {
  padding: 10px 12px;
  border-radius: 8px;
  background: rgba(0, 0, 0, 0.3);
  font-family: "SF Mono", "Consolas", monospace;
  font-size: 12px;
  color: rgba(255, 255, 255, 0.8);
  word-break: break-all;
  line-height: 1.5;
}

/* Action Bar */
.action-bar {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 16px;
}

.status-area {
  flex: 1;
  min-width: 0;
}

.status-badge {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 12px 14px;
  border-radius: 12px;
  max-width: 320px;
}

.status-badge.error {
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.3);
}

.status-badge.success {
  background: rgba(34, 197, 94, 0.1);
  border: 1px solid rgba(34, 197, 94, 0.3);
}

.badge-icon {
  flex-shrink: 0;
  width: 28px;
  height: 28px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.status-badge.error .badge-icon {
  background: rgba(239, 68, 68, 0.2);
  color: #ef4444;
}

.status-badge.success .badge-icon {
  background: rgba(34, 197, 94, 0.2);
  color: #22c55e;
}

.badge-icon svg {
  width: 14px;
  height: 14px;
}

.badge-content {
  flex: 1;
  min-width: 0;
}

.badge-title {
  font-size: 13px;
  font-weight: 700;
  color: #ffffff;
  margin-bottom: 2px;
}

.badge-text {
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
  border-radius: 10px;
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

.action-btn.secondary:hover:not(:disabled) {
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
  .type-options {
    grid-template-columns: 1fr;
  }

  .form-grid {
    grid-template-columns: 1fr;
  }

  .span-full {
    grid-column: auto;
  }

  .action-bar {
    flex-direction: column;
    align-items: stretch;
  }

  .status-area {
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
