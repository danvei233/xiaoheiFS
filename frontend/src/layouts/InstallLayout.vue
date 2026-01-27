<template>
  <div class="install-shell">
    <div class="install-wrap">
      <!-- Clean header -->
      <header class="install-header">
        <div class="brand-section">
          <div class="brand-logo">
            <svg viewBox="0 0 48 48" fill="none" xmlns="http://www.w3.org/2000/svg">
              <rect x="8" y="8" width="32" height="32" rx="8" fill="#1677ff" opacity="0.1"/>
              <path d="M24 16L16 21V27L24 32L32 27V21L24 16Z" stroke="#1677ff" stroke-width="2" stroke-linejoin="round"/>
              <circle cx="24" cy="24" r="3" fill="#1677ff"/>
            </svg>
          </div>
          <div class="brand-info">
            <span class="brand-badge">安装向导</span>
            <h1 class="brand-title">小黑云财务</h1>
          </div>
        </div>
        <div class="header-badge">
          <span class="badge-dot"></span>
          <span>VPS 管理平台</span>
        </div>
      </header>

      <!-- Main content card -->
      <main class="install-main">
        <!-- Already installed state -->
        <div v-if="install.loaded && install.installed" class="installed-state">
          <div class="installed-card">
            <div class="installed-icon">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10"/>
                <path d="M12 8v4m0 4h.01"/>
              </svg>
            </div>
            <div class="installed-content">
              <h2 class="installed-title">系统已安装</h2>
              <p class="installed-desc">
                如需重新安装，请备份数据库后删除 <code>install.lock</code> 文件
              </p>
            </div>
          </div>
        </div>

        <!-- Installation wizard -->
        <template v-else>
          <div class="wizard-layout">
            <!-- Left sidebar - steps -->
            <aside class="wizard-sidebar">
              <nav class="steps-nav">
                <div
                  v-for="(step, idx) in steps"
                  :key="idx"
                  class="step-nav-item"
                  :class="{
                    'step-active': idx === currentStep,
                    'step-completed': idx < currentStep
                  }"
                >
                  <div class="step-number">
                    <span v-if="idx < currentStep">
                      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                        <path d="M20 6L9 17l-5-5"/>
                      </svg>
                    </span>
                    <span v-else>{{ idx + 1 }}</span>
                  </div>
                  <div class="step-text">
                    <div class="step-name">{{ step.label }}</div>
                    <div class="step-hint">{{ step.desc }}</div>
                  </div>
                </div>
              </nav>

              <div class="sidebar-tip">
                <div class="tip-icon">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <circle cx="12" cy="12" r="10"/>
                    <path d="M12 16v-4m0-4h.01"/>
                  </svg>
                </div>
                <div class="tip-text">
                  <strong>提示</strong>
                  <span>完成数据库连接测试后才可进入下一步</span>
                </div>
              </div>
            </aside>

            <!-- Right content area -->
            <div class="wizard-content">
              <router-view />
            </div>
          </div>
        </template>
      </main>

      <!-- Footer -->
      <footer class="install-footer">
        <span>小黑云财务 v1.0.0</span>
        <span class="footer-sep">·</span>
        <span>VPS 管理平台</span>
      </footer>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useRoute } from "vue-router";
import { useInstallStore } from "@/stores/install";

const route = useRoute();
const install = useInstallStore();

const steps = [
  { label: "数据库", desc: "配置数据库连接" },
  { label: "站点信息", desc: "设置基本信息" },
  { label: "管理员", desc: "创建超级管理员" },
  { label: "完成", desc: "安装完成" }
];

const currentStep = computed(() => {
  const p = route.path;
  if (p.endsWith("/db")) return 0;
  if (p.endsWith("/site")) return 1;
  if (p.endsWith("/admin")) return 2;
  if (p.endsWith("/done")) return 3;
  return 0;
});
</script>

<style scoped>
/* ============================================
   VARIABLES & RESET
   ============================================ */
.install-shell {
  --bg-primary: #f5f7fa;
  --bg-surface: #ffffff;
  --bg-sidebar: #fafbfc;
  --border-color: #e5e7eb;
  --border-light: #f0f2f5;
  --text-primary: #1f2937;
  --text-secondary: #6b7280;
  --text-tertiary: #9ca3af;
  --color-primary: #1677ff;
  --color-primary-light: #e6f4ff;
  --color-success: #10b981;
  --color-warning: #f59e0b;
  --shadow-sm: 0 1px 2px rgba(0, 0, 0, 0.04);
  --shadow-md: 0 4px 12px rgba(0, 0, 0, 0.06);
  --shadow-lg: 0 8px 24px rgba(0, 0, 0, 0.08);
  --radius-sm: 6px;
  --radius-md: 10px;
  --radius-lg: 16px;
}

.install-shell {
  min-height: 100vh;
  background: var(--bg-primary);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 32px 20px;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
}

.install-wrap {
  width: 100%;
  max-width: 1100px;
}

/* ============================================
   HEADER
   ============================================ */
.install-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 24px;
}

.brand-section {
  display: flex;
  align-items: center;
  gap: 14px;
}

.brand-logo {
  width: 48px;
  height: 48px;
  border-radius: var(--radius-md);
  background: var(--bg-surface);
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: var(--shadow-sm);
  border: 1px solid var(--border-color);
}

.brand-logo svg {
  width: 28px;
  height: 28px;
}

.brand-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.brand-badge {
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--color-primary);
}

.brand-title {
  font-size: 22px;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0;
  letter-spacing: -0.02em;
}

.header-badge {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 14px;
  border-radius: 100px;
  background: var(--bg-surface);
  border: 1px solid var(--border-color);
  font-size: 13px;
  font-weight: 500;
  color: var(--text-secondary);
  box-shadow: var(--shadow-sm);
}

.badge-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--color-success);
}

/* ============================================
   MAIN CARD
   ============================================ */
.install-main {
  background: var(--bg-surface);
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-color);
  box-shadow: var(--shadow-md);
  overflow: hidden;
}

/* ============================================
   INSTALLED STATE
   ============================================ */
.installed-state {
  padding: 48px;
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 400px;
}

.installed-card {
  display: flex;
  gap: 20px;
  padding: 28px;
  border-radius: var(--radius-md);
  background: linear-gradient(135deg, #fffbeb 0%, #fef3c7 100%);
  border: 1px solid #fde68a;
  max-width: 480px;
}

.installed-icon {
  flex-shrink: 0;
  width: 44px;
  height: 44px;
  border-radius: var(--radius-sm);
  display: flex;
  align-items: center;
  justify-content: center;
  background: #fbbf24;
  color: #ffffff;
}

.installed-icon svg {
  width: 22px;
  height: 22px;
}

.installed-content {
  flex: 1;
}

.installed-title {
  font-size: 17px;
  font-weight: 600;
  color: #92400e;
  margin: 0 0 6px 0;
}

.installed-desc {
  font-size: 14px;
  line-height: 1.6;
  color: #b45309;
  margin: 0;
}

.installed-desc code {
  padding: 2px 6px;
  border-radius: 4px;
  background: rgba(255, 255, 255, 0.6);
  font-family: "SF Mono", Consolas, monospace;
  font-size: 13px;
  color: #92400e;
}

/* ============================================
   WIZARD LAYOUT
   ============================================ */
.wizard-layout {
  display: grid;
  grid-template-columns: 260px 1fr;
  min-height: 520px;
}

/* ============================================
   SIDEBAR - STEPS
   ============================================ */
.wizard-sidebar {
  background: var(--bg-sidebar);
  border-right: 1px solid var(--border-color);
  padding: 28px 20px;
  display: flex;
  flex-direction: column;
}

.steps-nav {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.step-nav-item {
  display: flex;
  gap: 12px;
  padding: 12px 14px;
  border-radius: var(--radius-sm);
  transition: all 0.2s ease;
}

.step-nav-item:hover {
  background: rgba(0, 0, 0, 0.02);
}

.step-nav-item.step-active {
  background: var(--color-primary-light);
}

.step-number {
  flex-shrink: 0;
  width: 28px;
  height: 28px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  font-weight: 600;
  background: var(--bg-surface);
  border: 2px solid var(--border-color);
  color: var(--text-tertiary);
  transition: all 0.2s ease;
}

.step-nav-item.step-active .step-number {
  background: var(--color-primary);
  border-color: var(--color-primary);
  color: #ffffff;
}

.step-nav-item.step-completed .step-number {
  background: var(--color-success);
  border-color: var(--color-success);
  color: #ffffff;
}

.step-nav-item.step-completed .step-number svg {
  width: 14px;
  height: 14px;
}

.step-text {
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 1px;
}

.step-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-secondary);
  transition: color 0.2s ease;
}

.step-nav-item.step-active .step-name {
  color: var(--color-primary);
}

.step-nav-item.step-completed .step-name {
  color: var(--color-success);
}

.step-hint {
  font-size: 12px;
  color: var(--text-tertiary);
}

/* Sidebar tip */
.sidebar-tip {
  margin-top: auto;
  display: flex;
  gap: 10px;
  padding: 14px;
  border-radius: var(--radius-sm);
  background: var(--bg-surface);
  border: 1px solid var(--border-color);
}

.tip-icon {
  flex-shrink: 0;
  width: 28px;
  height: 28px;
  border-radius: var(--radius-sm);
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-primary-light);
  color: var(--color-primary);
}

.tip-icon svg {
  width: 14px;
  height: 14px;
}

.tip-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.tip-text strong {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-primary);
}

.tip-text span {
  font-size: 12px;
  line-height: 1.4;
  color: var(--text-secondary);
}

/* ============================================
   CONTENT AREA
   ============================================ */
.wizard-content {
  padding: 32px 36px;
  background: var(--bg-surface);
}

/* ============================================
   FOOTER
   ============================================ */
.install-footer {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  margin-top: 20px;
  font-size: 13px;
  color: var(--text-tertiary);
}

.footer-sep {
  color: var(--border-color);
}

/* ============================================
   RESPONSIVE
   ============================================ */
@media (max-width: 900px) {
  .wizard-layout {
    grid-template-columns: 1fr;
  }

  .wizard-sidebar {
    border-right: none;
    border-bottom: 1px solid var(--border-color);
    padding: 20px;
  }

  .steps-nav {
    flex-direction: row;
    overflow-x: auto;
    padding-bottom: 8px;
  }

  .step-nav-item {
    flex-shrink: 0;
    min-width: 140px;
  }

  .step-hint {
    display: none;
  }

  .sidebar-tip {
    display: none;
  }

  .wizard-content {
    padding: 24px;
  }
}

@media (max-width: 640px) {
  .install-shell {
    padding: 16px;
  }

  .install-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }

  .header-badge {
    align-self: flex-start;
  }

  .installed-state {
    padding: 24px;
  }

  .installed-card {
    flex-direction: column;
    text-align: center;
    padding: 20px;
  }

  .wizard-content {
    padding: 20px 16px;
  }

  .install-footer {
    flex-direction: column;
    gap: 4px;
  }

  .footer-sep {
    display: none;
  }
}
</style>
