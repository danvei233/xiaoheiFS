<template>
  <div class="step">
    <div class="completion-container">
      <!-- Success Animation -->
      <div class="success-visual">
        <div class="success-ring">
          <svg viewBox="0 0 64 64" fill="none" xmlns="http://www.w3.org/2000/svg" class="check-icon">
            <circle cx="32" cy="32" r="28" stroke="url(#success-gradient)" stroke-width="3" fill="none"/>
            <path d="M20 32L28 40L44 24" stroke="url(#success-gradient)" stroke-width="3.5" stroke-linecap="round" stroke-linejoin="round" fill="none"/>
            <defs>
              <linearGradient id="success-gradient" x1="0" y1="0" x2="64" y2="64" gradientUnits="userSpaceOnUse">
                <stop offset="0%" stop-color="#22d3ee"/>
                <stop offset="100%" stop-color="#22c55e"/>
              </linearGradient>
            </defs>
          </svg>
        </div>
        <div class="success-particles">
          <span class="particle" />
          <span class="particle" />
          <span class="particle" />
          <span class="particle" />
          <span class="particle" />
          <span class="particle" />
        </div>
      </div>

      <!-- Success Message -->
      <div class="success-content">
        <h2 class="success-title">安装完成！</h2>
        <p class="success-desc">
          <span v-if="restartRequired">
            系统已成功配置 MySQL 数据库。需要重启后端服务以加载配置文件 <code>{{ configFile }}</code>
          </span>
          <span v-else>小黑云财务已成功安装并可以使用</span>
        </p>

        <!-- Info Cards -->
        <div class="info-cards">
          <div class="info-card primary">
            <div class="info-card-icon">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/>
                <polyline points="9 22 9 12 15 12 15 22"/>
              </svg>
            </div>
            <div class="info-card-content">
              <div class="info-card-title">管理后台</div>
              <div class="info-card-text">访问 <code>/admin</code> 进入管理控制台</div>
            </div>
          </div>

          <div v-if="restartRequired" class="info-card warning">
            <div class="info-card-icon">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/>
                <line x1="12" y1="9" x2="12" y2="13"/>
                <line x1="12" y1="17" x2="12.01" y2="17"/>
              </svg>
            </div>
            <div class="info-card-content">
              <div class="info-card-title">需要重启</div>
              <div class="info-card-text">重启后端服务或设置环境变量 <code>APP_DB_TYPE=mysql</code></div>
            </div>
          </div>

          <div class="info-card neutral">
            <div class="info-card-icon">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="3"/>
                <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/>
              </svg>
            </div>
            <div class="info-card-content">
              <div class="info-card-title">重新安装</div>
              <div class="info-card-text">删除 <code>install.lock</code> 文件可重新安装</div>
            </div>
          </div>
        </div>

        <!-- Action Buttons -->
        <div class="action-buttons">
          <button type="button" class="action-btn secondary" @click="goHome">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/>
              <polyline points="9 22 9 12 15 12 15 22"/>
            </svg>
            返回首页
          </button>
          <button type="button" class="action-btn primary" @click="goAdmin">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="3" y="3" width="7" height="7"/>
              <rect x="14" y="3" width="7" height="7"/>
              <rect x="14" y="14" width="7" height="7"/>
              <rect x="3" y="14" width="7" height="7"/>
            </svg>
            进入后台
          </button>
        </div>
      </div>

      <!-- Footer Branding -->
      <div class="footer-brand">
        <div class="brand-logo-mini">
          <svg viewBox="0 0 48 48" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path d="M24 4L6 14V34L24 44L42 34V14L24 4Z" stroke="currentColor" stroke-width="2" fill="none"/>
            <circle cx="24" cy="24" r="4" fill="currentColor"/>
          </svg>
        </div>
        <span class="brand-text">小黑云财务</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useRouter, useRoute } from "vue-router";

const router = useRouter();
const route = useRoute();

const restartRequired = computed(() => String(route.query.restart || "0") === "1");
const configFile = computed(() => String(route.query.config || "app.config.json"));

const goHome = async () => {
  const redirect = typeof route.query.redirect === "string" ? route.query.redirect : "/";
  await router.replace(redirect && redirect.startsWith("/") ? redirect : "/");
};

const goAdmin = async () => {
  await router.replace("/admin");
};
</script>

<style scoped>
.step {
  padding: 8px;
  animation: fadeInUp 0.6s ease-out;
}

@keyframes fadeInUp {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.completion-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  padding: 20px 8px;
}

/* Success Visual */
.success-visual {
  position: relative;
  margin-bottom: 28px;
}

.success-ring {
  width: 120px;
  height: 120px;
  margin: 0 auto;
  position: relative;
}

.check-icon {
  width: 100%;
  height: 100%;
  animation: checkPop 0.6s cubic-bezier(0.68, -0.55, 0.265, 1.55) 0.2s both;
}

@keyframes checkPop {
  0% {
    transform: scale(0) rotate(-45deg);
    opacity: 0;
  }
  50% {
    transform: scale(1.2) rotate(10deg);
  }
  100% {
    transform: scale(1) rotate(0deg);
    opacity: 1;
  }
}

.success-particles {
  position: absolute;
  inset: -10px;
  animation: particlesExplode 0.6s ease-out 0.3s both;
}

@keyframes particlesExplode {
  from {
    transform: scale(0.5);
    opacity: 0;
  }
  to {
    transform: scale(1);
    opacity: 1;
  }
}

.particle {
  position: absolute;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: linear-gradient(135deg, #22d3ee, #22c55e);
}

.particle:nth-child(1) {
  top: 0;
  left: 50%;
  transform: translateX(-50%) translateY(-20px);
  animation: particleFloat 2s ease-in-out infinite 0.1s;
}

.particle:nth-child(2) {
  top: 20%;
  right: 5%;
  transform: translateY(-50%) translateX(15px);
  animation: particleFloat 2s ease-in-out infinite 0.2s;
}

.particle:nth-child(3) {
  bottom: 20%;
  right: 5%;
  transform: translateY(-50%) translateX(15px);
  animation: particleFloat 2s ease-in-out infinite 0.3s;
}

.particle:nth-child(4) {
  bottom: 0;
  left: 50%;
  transform: translateX(-50%) translateY(20px);
  animation: particleFloat 2s ease-in-out infinite 0.4s;
}

.particle:nth-child(5) {
  bottom: 20%;
  left: 5%;
  transform: translateY(-50%) translateX(-15px);
  animation: particleFloat 2s ease-in-out infinite 0.5s;
}

.particle:nth-child(6) {
  top: 20%;
  left: 5%;
  transform: translateY(-50%) translateX(-15px);
  animation: particleFloat 2s ease-in-out infinite 0.6s;
}

@keyframes particleFloat {
  0%, 100% {
    transform: translate(var(--tx, -50%), var(--ty, -50%)) scale(1);
    opacity: 0.8;
  }
  50% {
    transform: translate(var(--tx, -50%), calc(var(--ty, -50%) - 8px)) scale(1.2);
    opacity: 1;
  }
}

/* Success Content */
.success-content {
  width: 100%;
  max-width: 560px;
}

.success-title {
  font-size: 28px;
  font-weight: 800;
  color: #ffffff;
  margin: 0 0 12px 0;
  letter-spacing: -0.02em;
  background: linear-gradient(135deg, #ffffff 0%, #22d3ee 50%, #22c55e 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.success-desc {
  font-size: 14px;
  line-height: 1.6;
  color: rgba(255, 255, 255, 0.7);
  margin: 0 0 24px 0;
}

.success-desc code {
  padding: 2px 6px;
  border-radius: 4px;
  background: rgba(34, 211, 238, 0.15);
  font-family: "SF Mono", "Consolas", monospace;
  font-size: 12px;
  color: #22d3ee;
}

.info-cards {
  display: flex;
  flex-direction: column;
  gap: 0;
  margin-bottom: 28px;
  border-top: 1px solid rgba(71, 85, 105, 0.4);
  border-bottom: 1px solid rgba(71, 85, 105, 0.4);
}

.info-card {
  display: flex;
  align-items: flex-start;
  gap: 14px;
  padding: 14px 10px;
  border-radius: 0;
  text-align: left;
  border-bottom: 1px solid rgba(71, 85, 105, 0.25);
}

.info-card.primary {
  background: rgba(34, 211, 238, 0.05);
  border-left: 3px solid rgba(34, 211, 238, 0.6);
}

.info-card.warning {
  background: rgba(251, 191, 36, 0.05);
  border-left: 3px solid rgba(251, 191, 36, 0.6);
}

.info-card.neutral {
  background: rgba(71, 85, 105, 0.1);
  border-left: 3px solid rgba(100, 116, 139, 0.7);
  border-bottom: none;
}

.info-card-icon {
  flex-shrink: 0;
  width: 32px;
  height: 32px;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.info-card.primary .info-card-icon {
  background: rgba(34, 211, 238, 0.2);
  color: #22d3ee;
}

.info-card.warning .info-card-icon {
  background: rgba(251, 191, 36, 0.2);
  color: #fbbf24;
}

.info-card.neutral .info-card-icon {
  background: rgba(100, 116, 139, 0.2);
  color: rgba(255, 255, 255, 0.7);
}

.info-card-icon svg {
  width: 18px;
  height: 18px;
}

.info-card-content {
  flex: 1;
  min-width: 0;
}

.info-card-title {
  font-size: 14px;
  font-weight: 700;
  color: #ffffff;
  margin-bottom: 4px;
}

.info-card-text {
  font-size: 13px;
  color: rgba(255, 255, 255, 0.6);
  line-height: 1.5;
}

.info-card-text code {
  padding: 1px 5px;
  border-radius: 4px;
  background: rgba(0, 0, 0, 0.3);
  font-family: "SF Mono", "Consolas", monospace;
  font-size: 11px;
  color: rgba(255, 255, 255, 0.8);
}

/* Action Buttons */
.action-buttons {
  display: flex;
  gap: 12px;
  justify-content: center;
}

.action-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  height: 48px;
  padding: 0 24px;
  border-radius: 4px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s ease;
  border: none;
}

.action-btn svg {
  width: 18px;
  height: 18px;
}

.action-btn.secondary {
  background: rgba(30, 41, 59, 0.8);
  color: #ffffff;
  border: 1px solid rgba(71, 85, 105, 0.5);
}

.action-btn.secondary:hover {
  background: rgba(30, 41, 59, 1);
  border-color: rgba(34, 211, 238, 0.5);
  transform: translateY(-1px);
}

.action-btn.primary {
  background: linear-gradient(135deg, #22d3ee 0%, #06b6d4 100%);
  color: #0f172a;
  box-shadow: 0 4px 20px rgba(34, 211, 238, 0.3);
}

.action-btn.primary:hover {
  box-shadow: 0 6px 30px rgba(34, 211, 238, 0.5);
  transform: translateY(-1px);
}

/* Footer Branding */
.footer-brand {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: 32px;
  padding-top: 24px;
  border-top: 1px solid rgba(71, 85, 105, 0.3);
}

.brand-logo-mini {
  width: 32px;
  height: 32px;
  color: rgba(34, 211, 238, 0.6);
}

.brand-logo-mini svg {
  width: 100%;
  height: 100%;
}

.brand-text {
  font-size: 13px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.4);
  letter-spacing: 0.05em;
}

/* Responsive */
@media (max-width: 600px) {
  .completion-container {
    padding: 12px;
  }

  .success-ring {
    width: 100px;
    height: 100px;
  }

  .success-title {
    font-size: 24px;
  }

  .action-buttons {
    flex-direction: column;
  }

  .action-btn {
    width: 100%;
  }
}
</style>
