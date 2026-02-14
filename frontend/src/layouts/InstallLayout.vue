<template>
  <a-config-provider :theme="{ algorithm: antTheme.darkAlgorithm }">
    <div class="install-shell">
    <!-- Animated background elements -->
    <div class="bg-orbs">
      <div class="orb orb-1"></div>
      <div class="orb orb-2"></div>
      <div class="orb orb-3"></div>
    </div>

    <div class="install-wrap">
      <!-- Clean header -->
      <header class="install-header">
        <div class="brand-section">
          <div class="brand-logo">
            <div class="logo-inner">
              <svg viewBox="0 0 48 48" fill="none" xmlns="http://www.w3.org/2000/svg">
                <path d="M24 6L8 17V31L24 42L40 31V17L24 6Z" stroke="currentColor" stroke-width="2.5" stroke-linejoin="round"/>
                <circle cx="24" cy="24" r="5" fill="currentColor"/>
              </svg>
            </div>
            <div class="logo-glow"></div>
          </div>
          <div class="brand-info">
            <span class="brand-badge">安装向导</span>
            <h1 class="brand-title">小黑云</h1>
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
        <span class="footer-logo">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M12 2L4 8.5V17.5L12 24L20 17.5V8.5L12 2Z"/>
          </svg>
        </span>
        <span>小黑云 v1.0.0</span>
        <span class="footer-dot">·</span>
        <span>VPS 管理平台</span>
      </footer>
    </div>
    </div>
  </a-config-provider>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useRoute } from "vue-router";
import { theme as antTheme } from "ant-design-vue";
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
.install-shell {
  --bg-main: #0b111c;
  --bg-panel: #0f1726;
  --bg-line: rgba(148, 163, 184, 0.22);
  --bg-line-soft: rgba(148, 163, 184, 0.14);
  --text-primary: #e2e8f0;
  --text-secondary: rgba(226, 232, 240, 0.72);
  --text-tertiary: rgba(226, 232, 240, 0.45);
  --brand: #38bdf8;
  --brand-soft: rgba(56, 189, 248, 0.16);
  --success: #22c55e;
  --warn: #f59e0b;
}

.install-shell {
  min-height: 100vh;
  background:
    radial-gradient(circle at 12% 18%, rgba(56, 189, 248, 0.14), transparent 42%),
    radial-gradient(circle at 82% 78%, rgba(20, 184, 166, 0.1), transparent 40%),
    linear-gradient(180deg, #0b111c 0%, #080d16 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 34px 24px;
  font-family: "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif;
  position: relative;
  overflow: hidden;
}

.install-shell::before {
  content: "";
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-image: linear-gradient(to right, rgba(148, 163, 184, 0.06) 1px, transparent 1px),
    linear-gradient(to bottom, rgba(148, 163, 184, 0.06) 1px, transparent 1px);
  background-size: 40px 40px;
  pointer-events: none;
  z-index: 0;
}

.install-wrap {
  width: 100%;
  max-width: 1180px;
  position: relative;
  z-index: 1;
}

.install-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 20px;
  animation: fadeInDown 0.6s ease-out;
  border-bottom: 1px solid var(--bg-line-soft);
  padding-bottom: 16px;
}

@keyframes fadeInDown {
  from {
    opacity: 0;
    transform: translateY(-10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.brand-section {
  display: flex;
  align-items: center;
  gap: 14px;
}

.brand-logo {
  width: 46px;
  height: 46px;
  border-radius: 10px;
  background: linear-gradient(145deg, #0284c7, #0ea5e9);
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: inset 0 0 0 1px rgba(226, 232, 240, 0.28);
  position: relative;
  overflow: hidden;
}

.brand-logo::before {
  content: "";
  position: absolute;
  top: 0;
  left: -100%;
  width: 50%;
  height: 100%;
  background: linear-gradient(
    90deg,
    transparent,
    rgba(255, 255, 255, 0.3),
    transparent
  );
  animation: shimmer 3s infinite;
}

@keyframes shimmer {
  0% {
    left: -100%;
  }
  100% {
    left: 200%;
  }
}

.brand-logo svg {
  width: 30px;
  height: 30px;
  position: relative;
  z-index: 1;
}

.brand-info {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.brand-badge {
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--brand);
}

.brand-title {
  font-size: 26px;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0;
  letter-spacing: 0.01em;
}

.header-badge {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 0;
  border-bottom: 2px solid var(--bg-line);
  font-size: 12px;
  font-weight: 700;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.08em;
}

.badge-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--success);
  animation: pulse-dot 2s ease-in-out infinite;
}

@keyframes pulse-dot {
  0%, 100% {
    opacity: 0.65;
  }
  50% {
    opacity: 1;
  }
}

.install-main {
  background: linear-gradient(180deg, rgba(15, 23, 38, 0.82), rgba(10, 16, 28, 0.86));
  border-top: 1px solid var(--bg-line);
  border-bottom: 1px solid var(--bg-line);
  overflow: clip;
  animation: scaleIn 0.5s ease-out 0.1s both;
}

@keyframes scaleIn {
  from {
    opacity: 0;
    transform: scale(0.95);
  }
  to {
    opacity: 1;
    transform: scale(1);
  }
}

.installed-state {
  padding: 48px;
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 420px;
}

.installed-card {
  display: flex;
  gap: 18px;
  padding: 26px 0 26px 22px;
  border-left: 4px solid rgba(245, 158, 11, 0.8);
  background: linear-gradient(90deg, rgba(245, 158, 11, 0.15), rgba(245, 158, 11, 0.02));
  max-width: 560px;
  width: 100%;
}

.installed-icon {
  flex-shrink: 0;
  width: 40px;
  height: 40px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(245, 158, 11, 0.2);
  color: #ffffff;
  border: 1px solid rgba(245, 158, 11, 0.4);
}

.installed-icon svg {
  width: 24px;
  height: 24px;
}

.installed-content {
  flex: 1;
}

.installed-title {
  font-size: 20px;
  font-weight: 700;
  color: #fde68a;
  margin: 0 0 8px 0;
}

.installed-desc {
  font-size: 14px;
  line-height: 1.6;
  color: rgba(253, 224, 71, 0.8);
  margin: 0;
}

.installed-desc code {
  padding: 2px 7px;
  border-radius: 4px;
  background: rgba(0, 0, 0, 0.3);
  font-family: "SF Mono", Consolas, monospace;
  font-size: 12px;
  color: #fcd34d;
}

.wizard-layout {
  display: grid;
  grid-template-columns: 260px 1fr;
  min-height: 600px;
}

.wizard-sidebar {
  background: rgba(11, 18, 30, 0.6);
  border-right: 1px solid var(--bg-line);
  padding: 26px 20px 20px;
  display: flex;
  flex-direction: column;
}

.steps-nav {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.step-nav-item {
  display: flex;
  gap: 12px;
  padding: 14px 8px;
  border-bottom: 1px solid var(--bg-line-soft);
  transition: all 0.25s ease;
}

.step-nav-item:hover {
  background: rgba(148, 163, 184, 0.06);
}

.step-nav-item.step-active {
  background: linear-gradient(90deg, var(--brand-soft), transparent);
  border-bottom-color: rgba(56, 189, 248, 0.5);
}

.step-number {
  flex-shrink: 0;
  width: 28px;
  height: 28px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 700;
  background: rgba(148, 163, 184, 0.09);
  border: 1px solid var(--bg-line-soft);
  color: var(--text-tertiary);
  transition: all 0.25s ease;
}

.step-nav-item.step-active .step-number {
  background: #0284c7;
  border-color: #38bdf8;
  color: #ffffff;
}

.step-nav-item.step-completed .step-number {
  background: #166534;
  border-color: #16a34a;
  color: #ffffff;
}

.step-nav-item.step-completed .step-number svg {
  width: 15px;
  height: 15px;
}

.step-text {
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 2px;
}

.step-name {
  font-size: 14px;
  font-weight: 700;
  color: var(--text-secondary);
  transition: color 0.3s ease;
}

.step-nav-item.step-active .step-name {
  color: var(--text-primary);
}

.step-nav-item.step-completed .step-name {
  color: #4ade80;
}

.step-hint {
  font-size: 12px;
  color: var(--text-tertiary);
}

.sidebar-tip {
  margin-top: auto;
  display: flex;
  gap: 10px;
  padding: 14px 0 0;
  border-top: 1px dashed var(--bg-line);
}

.tip-icon {
  flex-shrink: 0;
  width: 28px;
  height: 28px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--brand-soft);
  color: var(--brand);
}

.tip-icon svg {
  width: 15px;
  height: 15px;
}

.tip-text {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.tip-text strong {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-primary);
}

.tip-text span {
  font-size: 12px;
  line-height: 1.5;
  color: var(--text-secondary);
}

.wizard-content {
  padding: 34px 38px;
  background: rgba(14, 22, 36, 0.45);
  animation: fadeInRight 0.4s ease-out 0.2s both;
}

@keyframes fadeInRight {
  from {
    opacity: 0;
    transform: translateX(20px);
  }
  to {
    opacity: 1;
    transform: translateX(0);
  }
}

.install-footer {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  margin-top: 14px;
  font-size: 13px;
  color: var(--text-tertiary);
  animation: fadeInUp 0.5s ease-out 0.3s both;
}

@keyframes fadeInUp {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.footer-dot {
  color: rgba(148, 163, 184, 0.4);
}

@media (max-width: 900px) {
  .wizard-layout {
    grid-template-columns: 1fr;
  }

  .wizard-sidebar {
    border-right: none;
    border-bottom: 1px solid var(--bg-line);
    padding: 20px;
  }

  .steps-nav {
    flex-direction: row;
    overflow-x: auto;
    padding-bottom: 8px;
  }

  .step-nav-item {
    flex-shrink: 0;
    min-width: 152px;
    border-bottom: none;
    border-right: 1px solid var(--bg-line-soft);
  }

  .step-hint {
    display: none;
  }

  .sidebar-tip {
    display: none;
  }

  .wizard-content {
    padding: 24px 20px;
  }
}

@media (max-width: 640px) {
  .install-shell {
    padding: 20px 16px;
  }

  .install-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 16px;
  }

  .header-badge {
    align-self: flex-start;
  }

  .installed-state {
    padding: 24px 16px;
  }

  .installed-card {
    flex-direction: row;
    padding: 18px 0 18px 14px;
  }

  .wizard-content {
    padding: 24px 16px;
  }

  .install-footer {
    flex-direction: column;
    gap: 6px;
  }

  .footer-dot {
    display: inline;
  }
}
</style>
