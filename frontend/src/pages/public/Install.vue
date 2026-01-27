<template>
  <div class="install-shell">
    <!-- Animated gradient background with subtle pattern -->
    <div class="install-bg" aria-hidden="true">
      <div class="bg-gradient-primary" />
      <div class="bg-gradient-secondary" />
      <div class="bg-pattern" />
      <div class="floating-orb orb-1" />
      <div class="floating-orb orb-2" />
      <div class="floating-orb orb-3" />
    </div>

    <div class="install-wrap">
      <!-- Premium brand header with animated gradient -->
      <header class="install-header">
        <div class="brand-container">
          <div class="brand-logo">
            <div class="logo-inner">
              <svg viewBox="0 0 48 48" fill="none" xmlns="http://www.w3.org/2000/svg" class="logo-icon">
                <path d="M24 4L6 14V34L24 44L42 34V14L24 4Z" stroke="url(#logo-gradient)" stroke-width="2.5" fill="none"/>
                <path d="M24 24L6 14" stroke="url(#logo-gradient)" stroke-width="2" stroke-linecap="round"/>
                <path d="M24 24V44" stroke="url(#logo-gradient)" stroke-width="2" stroke-linecap="round"/>
                <path d="M24 24L42 14" stroke="url(#logo-gradient)" stroke-width="2" stroke-linecap="round"/>
                <circle cx="24" cy="24" r="4" fill="url(#logo-gradient)"/>
                <defs>
                  <linearGradient id="logo-gradient" x1="6" y1="4" x2="42" y2="44" gradientUnits="userSpaceOnUse">
                    <stop offset="0%" stop-color="#22d3ee"/>
                    <stop offset="50%" stop-color="#06b6d4"/>
                    <stop offset="100%" stop-color="#0891b2"/>
                  </linearGradient>
                </defs>
              </svg>
            </div>
          </div>
          <div class="brand-text">
            <div class="brand-kicker">安装向导</div>
            <h1 class="brand-title">小黑云财务</h1>
          </div>
        </div>
        <div class="install-meta">
          <span class="meta-dot" />
          <span>SQLite / MySQL</span>
        </div>
      </header>

      <!-- Main card with glassmorphism effect -->
      <main class="install-card">
        <div class="card-inner">
          <!-- Already installed alert -->
          <div v-if="install.loaded && install.installed" class="installed-alert">
            <div class="alert-icon">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/>
              </svg>
            </div>
            <div class="alert-content">
              <div class="alert-title">系统已安装</div>
              <div class="alert-desc">如需重新安装，请备份数据库后删除 <code>install.lock</code> 文件</div>
            </div>
          </div>

          <!-- Installation wizard -->
          <template v-else>
            <div class="install-grid">
              <!-- Left rail with steps -->
              <aside class="install-rail">
                <div class="rail-inner">
                  <div class="steps-container">
                    <div
                      v-for="(step, idx) in steps"
                      :key="idx"
                      class="step-item"
                      :class="{ active: idx === currentStep, completed: idx < currentStep }"
                    >
                      <div class="step-indicator">
                        <div class="step-dot" />
                        <div v-if="idx < steps.length - 1" class="step-line" />
                      </div>
                      <div class="step-info">
                        <div class="step-label">{{ step.label }}</div>
                        <div class="step-desc">{{ step.desc }}</div>
                      </div>
                    </div>
                  </div>

                  <div class="install-tip">
                    <div class="tip-icon">
                      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <circle cx="12" cy="12" r="10"/>
                        <path d="M12 16v-4m0-4h.01"/>
                      </svg>
                    </div>
                    <div class="tip-content">
                      <div class="tip-title">提示</div>
                      <div class="tip-text">完成数据库连接测试后才可进入下一步</div>
                    </div>
                  </div>
                </div>
              </aside>

              <!-- Main stage with step content -->
              <div class="install-stage">
                <div class="stage-inner">
                  <router-view />
                </div>
              </div>
            </div>
          </template>
        </div>
      </main>

      <!-- Footer with version info -->
      <footer class="install-footer">
        <span class="footer-text">小黑云财务 v1.0.0</span>
        <span class="footer-divider">·</span>
        <span class="footer-text">VPS 管理平台</span>
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
   BACKGROUND & SHELL
   ============================================ */
.install-shell {
  min-height: 100vh;
  padding: 24px 16px 32px;
  position: relative;
  overflow: hidden;
  background: #0f172a;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "SF Pro Text", system-ui, sans-serif;
}

.install-bg {
  position: fixed;
  inset: 0;
  pointer-events: none;
  z-index: 0;
}

.bg-gradient-primary {
  position: absolute;
  top: -20%;
  right: -10%;
  width: 70%;
  height: 70%;
  background: radial-gradient(ellipse at center, rgba(34, 211, 238, 0.15) 0%, transparent 70%);
  animation: float 20s ease-in-out infinite;
}

.bg-gradient-secondary {
  position: absolute;
  bottom: -15%;
  left: -5%;
  width: 60%;
  height: 60%;
  background: radial-gradient(ellipse at center, rgba(6, 182, 212, 0.12) 0%, transparent 70%);
  animation: float 25s ease-in-out infinite reverse;
}

.bg-pattern {
  position: absolute;
  inset: 0;
  background-image:
    radial-gradient(rgba(34, 211, 238, 0.03) 1px, transparent 1px),
    radial-gradient(rgba(6, 182, 212, 0.02) 1px, transparent 1px);
  background-size: 32px 32px, 48px 48px;
  background-position: 0 0, 16px 16px;
  opacity: 0.8;
}

@keyframes float {
  0%, 100% { transform: translate(0, 0) scale(1); }
  33% { transform: translate(30px, -30px) scale(1.05); }
  66% { transform: translate(-20px, 20px) scale(0.95); }
}

/* Floating orbs */
.floating-orb {
  position: absolute;
  border-radius: 50%;
  filter: blur(60px);
  opacity: 0.4;
  animation: orb-float 15s ease-in-out infinite;
}

.orb-1 {
  width: 300px;
  height: 300px;
  top: 10%;
  left: 15%;
  background: radial-gradient(circle, rgba(34, 211, 238, 0.3) 0%, transparent 70%);
  animation-delay: 0s;
}

.orb-2 {
  width: 200px;
  height: 200px;
  top: 60%;
  right: 20%;
  background: radial-gradient(circle, rgba(6, 182, 212, 0.25) 0%, transparent 70%);
  animation-delay: -5s;
}

.orb-3 {
  width: 250px;
  height: 250px;
  bottom: 20%;
  left: 50%;
  background: radial-gradient(circle, rgba(8, 145, 178, 0.2) 0%, transparent 70%);
  animation-delay: -10s;
}

@keyframes orb-float {
  0%, 100% { transform: translate(0, 0); }
  25% { transform: translate(40px, -30px); }
  50% { transform: translate(-30px, 40px); }
  75% { transform: translate(30px, 30px); }
}

.install-wrap {
  position: relative;
  z-index: 1;
  width: 100%;
  max-width: 1000px;
  margin: 0 auto;
}

/* ============================================
   HEADER WITH BRAND
   ============================================ */
.install-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  margin-bottom: 20px;
  animation: fadeSlideDown 0.8s ease-out;
}

@keyframes fadeSlideDown {
  from {
    opacity: 0;
    transform: translateY(-20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.brand-container {
  display: flex;
  align-items: center;
  gap: 16px;
}

.brand-logo {
  position: relative;
  width: 56px;
  height: 56px;
}

.logo-inner {
  width: 100%;
  height: 100%;
  border-radius: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, rgba(34, 211, 238, 0.2) 0%, rgba(6, 182, 212, 0.1) 100%);
  backdrop-filter: blur(20px);
  border: 1px solid rgba(34, 211, 238, 0.3);
  box-shadow:
    0 8px 32px rgba(34, 211, 238, 0.2),
    inset 0 1px 0 rgba(255, 255, 255, 0.1);
  animation: logo-pulse 3s ease-in-out infinite;
}

@keyframes logo-pulse {
  0%, 100% {
    box-shadow:
      0 8px 32px rgba(34, 211, 238, 0.2),
      inset 0 1px 0 rgba(255, 255, 255, 0.1);
  }
  50% {
    box-shadow:
      0 8px 40px rgba(34, 211, 238, 0.3),
      inset 0 1px 0 rgba(255, 255, 255, 0.15);
  }
}

.logo-icon {
  width: 32px;
  height: 32px;
}

.brand-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.brand-kicker {
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.15em;
  text-transform: uppercase;
  color: rgba(34, 211, 238, 0.9);
}

.brand-title {
  font-size: 26px;
  font-weight: 800;
  color: #ffffff;
  margin: 0;
  letter-spacing: -0.02em;
  background: linear-gradient(135deg, #ffffff 0%, #22d3ee 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.install-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 14px;
  border-radius: 100px;
  background: rgba(34, 211, 238, 0.1);
  backdrop-filter: blur(10px);
  border: 1px solid rgba(34, 211, 238, 0.2);
  font-size: 12px;
  font-weight: 500;
  color: rgba(255, 255, 255, 0.8);
}

.meta-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #22d3ee;
  box-shadow: 0 0 10px rgba(34, 211, 238, 0.6);
  animation: dot-pulse 2s ease-in-out infinite;
}

@keyframes dot-pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

/* ============================================
   MAIN CARD
   ============================================ */
.install-card {
  background: rgba(15, 23, 42, 0.6);
  backdrop-filter: blur(20px);
  border: 1px solid rgba(34, 211, 238, 0.15);
  border-radius: 24px;
  overflow: hidden;
  box-shadow:
    0 24px 48px rgba(0, 0, 0, 0.4),
    0 0 0 1px rgba(255, 255, 255, 0.05) inset;
  animation: fadeSlideUp 0.8s ease-out 0.1s both;
}

@keyframes fadeSlideUp {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.card-inner {
  padding: 24px;
}

/* Already installed alert */
.installed-alert {
  display: flex;
  align-items: flex-start;
  gap: 16px;
  padding: 20px;
  border-radius: 16px;
  background: linear-gradient(135deg, rgba(251, 191, 36, 0.1) 0%, rgba(245, 158, 11, 0.05) 100%);
  border: 1px solid rgba(251, 191, 36, 0.3);
}

.alert-icon {
  flex-shrink: 0;
  width: 40px;
  height: 40px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(251, 191, 36, 0.2);
  color: #fbbf24;
}

.alert-icon svg {
  width: 20px;
  height: 20px;
}

.alert-content {
  flex: 1;
}

.alert-title {
  font-size: 15px;
  font-weight: 700;
  color: #fbbf24;
  margin-bottom: 4px;
}

.alert-desc {
  font-size: 13px;
  line-height: 1.5;
  color: rgba(255, 255, 255, 0.7);
}

.alert-desc code {
  padding: 2px 6px;
  border-radius: 4px;
  background: rgba(0, 0, 0, 0.3);
  font-family: "SF Mono", "Consolas", monospace;
  font-size: 12px;
  color: #fbbf24;
}

/* ============================================
   INSTALL GRID
   ============================================ */
.install-grid {
  display: grid;
  grid-template-columns: 260px 1fr;
  gap: 24px;
  animation: fadeIn 0.6s ease-out 0.2s both;
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

/* ============================================
   LEFT RAIL - STEPS
   ============================================ */
.install-rail {
  position: sticky;
  top: 24px;
  height: fit-content;
}

.rail-inner {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.steps-container {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.step-item {
  display: flex;
  gap: 12px;
  position: relative;
}

.step-indicator {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.step-dot {
  position: relative;
  z-index: 1;
  width: 32px;
  height: 32px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  font-weight: 700;
  background: rgba(30, 41, 59, 0.8);
  border: 2px solid rgba(71, 85, 105, 0.5);
  color: rgba(255, 255, 255, 0.5);
  transition: all 0.3s ease;
}

.step-item.active .step-dot {
  background: linear-gradient(135deg, #22d3ee 0%, #06b6d4 100%);
  border-color: #22d3ee;
  color: #0f172a;
  box-shadow: 0 0 20px rgba(34, 211, 238, 0.4);
}

.step-item.completed .step-dot {
  background: rgba(34, 211, 238, 0.2);
  border-color: #22d3ee;
  color: #22d3ee;
}

.step-item.completed .step-dot::before {
  content: "✓";
  font-size: 12px;
}

.step-line {
  position: absolute;
  top: 32px;
  left: 50%;
  width: 2px;
  height: calc(100% + 12px);
  background: rgba(71, 85, 105, 0.3);
  transform: translateX(-50%);
}

.step-item.completed .step-line {
  background: linear-gradient(180deg, #22d3ee 0%, rgba(34, 211, 238, 0.2) 100%);
}

.step-item:last-child .step-line {
  display: none;
}

.step-info {
  padding-top: 4px;
  padding-bottom: 20px;
}

.step-label {
  font-size: 14px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.5);
  margin-bottom: 2px;
  transition: color 0.3s ease;
}

.step-item.active .step-label {
  color: #ffffff;
}

.step-item.completed .step-label {
  color: rgba(34, 211, 238, 0.9);
}

.step-desc {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.4);
}

.step-item.active .step-desc {
  color: rgba(34, 211, 238, 0.7);
}

/* Tip box */
.install-tip {
  display: flex;
  gap: 12px;
  padding: 16px;
  border-radius: 14px;
  background: rgba(34, 211, 238, 0.05);
  border: 1px solid rgba(34, 211, 238, 0.15);
}

.tip-icon {
  flex-shrink: 0;
  width: 32px;
  height: 32px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(34, 211, 238, 0.15);
  color: #22d3ee;
}

.tip-icon svg {
  width: 16px;
  height: 16px;
}

.tip-content {
  flex: 1;
}

.tip-title {
  font-size: 13px;
  font-weight: 700;
  color: #ffffff;
  margin-bottom: 4px;
}

.tip-text {
  font-size: 12px;
  line-height: 1.5;
  color: rgba(255, 255, 255, 0.6);
}

/* ============================================
   STAGE AREA
   ============================================ */
.install-stage {
  min-height: 400px;
}

.stage-inner {
  min-height: 400px;
}

/* ============================================
   FOOTER
   ============================================ */
.install-footer {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  margin-top: 16px;
  padding: 12px;
  animation: fadeIn 0.8s ease-out 0.3s both;
}

.footer-text {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.4);
}

.footer-divider {
  color: rgba(255, 255, 255, 0.2);
}

/* ============================================
   RESPONSIVE
   ============================================ */
@media (max-width: 900px) {
  .install-grid {
    grid-template-columns: 1fr;
  }

  .install-rail {
    position: static;
  }

  .rail-inner {
    flex-direction: row;
    align-items: flex-start;
    justify-content: space-between;
    flex-wrap: wrap;
  }

  .steps-container {
    flex: 1;
    min-width: 0;
  }

  .install-tip {
    width: 100%;
  }

  .install-meta {
    display: none;
  }

  .install-footer {
    flex-direction: column;
    gap: 4px;
  }

  .footer-divider {
    display: none;
  }
}

@media (max-width: 600px) {
  .install-header {
    flex-direction: column;
    align-items: flex-start;
  }

  .brand-title {
    font-size: 22px;
  }

  .card-inner {
    padding: 16px;
  }

  .steps-container {
    flex-direction: row;
    overflow-x: auto;
    padding-bottom: 8px;
    margin: 0 -16px;
    padding-left: 16px;
  }

  .step-item {
    flex-shrink: 0;
  }

  .step-indicator {
    flex-direction: row;
    align-items: flex-start;
    gap: 8px;
  }

  .step-dot {
    width: 28px;
    height: 28px;
    font-size: 12px;
  }

  .step-line {
    top: 50%;
    left: 28px;
    width: calc(100% + 8px);
    height: 2px;
    transform: translateY(-50%);
  }

  .step-info {
    padding-top: 0;
    padding-bottom: 0;
    padding-right: 16px;
  }

  .step-desc {
    display: none;
  }
}
</style>
