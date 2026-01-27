<template>
  <a-config-provider
    :theme="{
      algorithm: theme.darkAlgorithm,
      token: {
        colorText: '#f1f5f9',
        colorTextSecondary: '#94a3b8',
        colorTextTertiary: '#64748b',
        colorBgContainer: '#111827',
        colorBorder: '#1e293b',
        colorPrimary: '#0ea5e9'
      }
    }"
  >
    <div class="login-page">
    <!-- Animated Background -->
    <canvas ref="bgCanvas" class="background-canvas"></canvas>

    <!-- Floating Decorations -->
    <div class="floating-shapes">
      <div class="shape shape-1"></div>
      <div class="shape shape-2"></div>
      <div class="shape shape-3"></div>
      <div class="shape shape-4"></div>
    </div>

    <div class="login-container">
      <!-- Left Panel - Brand -->
      <div class="brand-panel">
        <div class="brand-content">
          <div class="brand-logo">
            <div class="logo-inner">
              <SiteLogoMedia :size="28" />
            </div>
          </div>

          <h1 class="brand-title">
            <span class="title-line">欢迎回来</span>
            <span class="title-gradient">小黑云控制台</span>
          </h1>

          <p class="brand-description">
            专业的大厂级云服务平台，统一管理您的云端资产
          </p>

          <div class="feature-list">
            <div class="feature-item" v-for="feature in features" :key="feature.text">
              <component :is="feature.icon" class="feature-icon" />
              <span class="feature-text">{{ feature.text }}</span>
            </div>
          </div>

          <div class="brand-stats">
            <div class="stat-item" v-for="stat in stats" :key="stat.label">
              <span class="stat-value">{{ stat.value }}</span>
              <span class="stat-label">{{ stat.label }}</span>
            </div>
          </div>
        </div>

        <!-- Gradient Overlay -->
        <div class="brand-gradient"></div>
      </div>

      <!-- Right Panel - Login Form -->
      <div class="form-panel">
        <div class="form-content">
          <!-- Header -->
          <div class="form-header">
            <h2 class="form-title">用户登录</h2>
            <p class="form-subtitle">登录以访问您的控制台</p>
          </div>

          <!-- Form -->
          <a-form
            :model="form"
            layout="vertical"
            @finish="onSubmit"
            class="login-form"
          >
            <a-form-item
              label="账号"
              name="username"
              :rules="[{ required: true, message: '请输入账号' }]"
            >
              <a-input
                v-model:value="form.username"
                placeholder="请输入用户名"
                size="large"
                class="input-field"
              >
                <template #prefix>
                  <UserOutlined class="input-icon" />
                </template>
              </a-input>
            </a-form-item>

            <a-form-item
              label="密码"
              name="password"
              :rules="[{ required: true, message: '请输入密码' }]"
            >
              <a-input-password
                v-model:value="form.password"
                placeholder="请输入密码"
                size="large"
                class="input-field"
              >
                <template #prefix>
                  <LockOutlined class="input-icon" />
                </template>
              </a-input-password>
            </a-form-item>

            <div class="form-actions">
              <a-checkbox v-model:checked="rememberMe" class="remember-checkbox">
                记住我
              </a-checkbox>
              <router-link to="/forgot-password" class="forgot-link">
                忘记密码？
              </router-link>
            </div>

            <a-button
              type="primary"
              html-type="submit"
              block
              size="large"
              :loading="auth.loading"
              class="submit-button"
            >
              <span>登录</span>
              <span class="button-arrow">→</span>
            </a-button>
          </a-form>

          <!-- Divider -->
          <div class="form-divider">
            <span>或</span>
          </div>

          <!-- Register Link -->
          <div class="form-footer">
            <p class="footer-text">
              还没有账号？
              <router-link to="/register" class="register-link">
                立即注册
                <span class="link-arrow">→</span>
              </router-link>
            </p>
          </div>
        </div>

        <!-- Bottom Info -->
        <div class="form-bottom">
          <p class="copyright">
            © 2024 小黑云. All rights reserved.
          </p>
        </div>
      </div>
    </div>
    </div>
  </a-config-provider>
</template>

<script setup>
import { reactive, ref, onMounted, onUnmounted, h } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import SiteLogoMedia from '@/components/brand/SiteLogoMedia.vue'
import { message, ConfigProvider, theme } from 'ant-design-vue'
import { UserOutlined, LockOutlined, RocketOutlined, BarChartOutlined, LinkOutlined, SafetyOutlined } from '@ant-design/icons-vue'

const form = reactive({
  username: '',
  password: ''
})

const rememberMe = ref(false)
const auth = useAuthStore()
const router = useRouter()
const route = useRoute()
const bgCanvas = ref()

const features = [
  { icon: RocketOutlined, text: 'VPS 生命周期管理' },
  { icon: BarChartOutlined, text: '订单实时追踪' },
  { icon: LinkOutlined, text: '自动化平台对接' },
  { icon: SafetyOutlined, text: '企业级安全保障' }
]

const stats = [
  { value: '99.9%', label: '可用性' },
  { value: '24/7', label: '技术支持' },
  { value: '<5m', label: '部署时间' }
]

const onSubmit = async () => {
  const token = await auth.login(form)
  if (!token) {
    message.error('登录失败，请检查账号密码')
    return
  }
  await auth.fetchMe()
  message.success('登录成功')
  router.replace(String(route.query.redirect || '/console'))
}

// Canvas animation
let animationId
const particles = []

const initCanvas = () => {
  const canvas = bgCanvas.value
  if (!canvas) return

  const ctx = canvas.getContext('2d')
  if (!ctx) return

  const resize = () => {
    canvas.width = window.innerWidth
    canvas.height = window.innerHeight
  }

  resize()
  window.addEventListener('resize', resize)

  // Create particles
  for (let i = 0; i < 60; i++) {
    particles.push({
      x: Math.random() * canvas.width,
      y: Math.random() * canvas.height,
      vx: (Math.random() - 0.5) * 0.3,
      vy: (Math.random() - 0.5) * 0.3,
      radius: Math.random() * 2 + 1,
      opacity: Math.random() * 0.5 + 0.1
    })
  }

  const animate = () => {
    ctx.clearRect(0, 0, canvas.width, canvas.height)

    // Draw gradient background
    const gradient = ctx.createLinearGradient(0, 0, canvas.width, canvas.height)
    gradient.addColorStop(0, '#0a0e17')
    gradient.addColorStop(1, '#111827')
    ctx.fillStyle = gradient
    ctx.fillRect(0, 0, canvas.width, canvas.height)

    // Update and draw particles
    particles.forEach((particle, i) => {
      particle.x += particle.vx
      particle.y += particle.vy

      if (particle.x < 0 || particle.x > canvas.width) particle.vx *= -1
      if (particle.y < 0 || particle.y > canvas.height) particle.vy *= -1

      ctx.beginPath()
      ctx.arc(particle.x, particle.y, particle.radius, 0, Math.PI * 2)
      ctx.fillStyle = `rgba(14, 165, 233, ${particle.opacity})`
      ctx.fill()

      // Draw connections
      particles.slice(i + 1).forEach(other => {
        const dx = particle.x - other.x
        const dy = particle.y - other.y
        const distance = Math.sqrt(dx * dx + dy * dy)

        if (distance < 150) {
          ctx.beginPath()
          ctx.moveTo(particle.x, particle.y)
          ctx.lineTo(other.x, other.y)
          ctx.strokeStyle = `rgba(14, 165, 233, ${0.08 * (1 - distance / 150)})`
          ctx.stroke()
        }
      })
    })

    animationId = requestAnimationFrame(animate)
  }

  animate()
}

onMounted(() => {
  initCanvas()
})

onUnmounted(() => {
  if (animationId) {
    cancelAnimationFrame(animationId)
  }
  window.removeEventListener('resize', () => {})
})
</script>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  overflow: hidden;
}

.background-canvas {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  z-index: 0;
}

/* Floating Shapes */
.floating-shapes {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  pointer-events: none;
  z-index: 1;
  overflow: hidden;
}

.shape {
  position: absolute;
  border-radius: 50%;
  background: linear-gradient(135deg, rgba(14, 165, 233, 0.1), rgba(249, 115, 22, 0.05));
  animation: float 20s ease-in-out infinite;
}

.shape-1 {
  width: 300px;
  height: 300px;
  top: -100px;
  right: -100px;
  animation-delay: 0s;
}

.shape-2 {
  width: 200px;
  height: 200px;
  bottom: -50px;
  left: -50px;
  animation-delay: -5s;
}

.shape-3 {
  width: 150px;
  height: 150px;
  top: 50%;
  left: 10%;
  animation-delay: -10s;
}

.shape-4 {
  width: 100px;
  height: 100px;
  bottom: 20%;
  right: 15%;
  animation-delay: -15s;
}

@keyframes float {
  0%, 100% {
    transform: translate(0, 0) rotate(0deg);
  }
  25% {
    transform: translate(30px, -30px) rotate(90deg);
  }
  50% {
    transform: translate(-20px, 20px) rotate(180deg);
  }
  75% {
    transform: translate(20px, 30px) rotate(270deg);
  }
}

/* Container */
.login-container {
  position: relative;
  z-index: 2;
  display: grid;
  grid-template-columns: 1fr 1fr;
  width: 100%;
  max-width: 1200px;
  min-height: 700px;
  background: rgba(17, 24, 39, 0.8);
  backdrop-filter: blur(20px);
  border-radius: 32px;
  overflow: hidden;
  border: 1px solid rgba(30, 41, 59, 0.5);
  box-shadow: 0 25px 50px rgba(0, 0, 0, 0.3);
}

/* Brand Panel */
.brand-panel {
  position: relative;
  padding: 60px 50px;
  display: flex;
  flex-direction: column;
  justify-content: center;
  overflow: hidden;
}

.brand-gradient {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: linear-gradient(135deg, rgba(14, 165, 233, 0.15), transparent 60%);
  pointer-events: none;
}

.brand-content {
  position: relative;
  z-index: 1;
}

.brand-logo {
  margin-bottom: 32px;
}

.logo-inner {
  width: 64px;
  height: 64px;
  background: linear-gradient(135deg, var(--color-primary), var(--color-accent));
  border-radius: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 8px 32px rgba(14, 165, 233, 0.3);
}

.logo-text {
  font-size: 24px;
  font-weight: 800;
  color: white;
}

.brand-title {
  font-family: var(--font-heading);
  font-size: 42px;
  font-weight: 800;
  line-height: 1.2;
  margin: 0 0 20px;
}

.title-line {
  display: block;
  color: var(--color-text);
}

.title-gradient {
  display: block;
  background: linear-gradient(135deg, var(--color-primary-light), var(--color-accent));
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.brand-description {
  font-size: 16px;
  color: var(--color-text-muted);
  margin: 0 0 40px;
  line-height: 1.6;
}

.feature-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
  margin-bottom: 40px;
}

.feature-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  background: rgba(14, 165, 233, 0.05);
  border: 1px solid rgba(14, 165, 233, 0.1);
  border-radius: 12px;
  transition: all 0.3s;
}

.feature-item:hover {
  background: rgba(14, 165, 233, 0.1);
  border-color: rgba(14, 165, 233, 0.3);
  transform: translateX(8px);
}

.feature-icon {
  font-size: 20px;
  color: var(--color-primary-light);
}

.feature-text {
  font-size: 15px;
  color: var(--color-text);
}

.brand-stats {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 20px;
}

.stat-item {
  text-align: center;
  padding: 16px;
  background: rgba(255, 255, 255, 0.03);
  border-radius: 12px;
}

.stat-value {
  display: block;
  font-size: 24px;
  font-weight: 700;
  background: linear-gradient(135deg, var(--color-primary-light), var(--color-accent));
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  margin-bottom: 4px;
}

.stat-label {
  display: block;
  font-size: 12px;
  color: var(--color-text-muted);
}

/* Form Panel */
.form-panel {
  padding: 60px 50px;
  display: flex;
  flex-direction: column;
  background: rgba(10, 14, 23, 0.5);
  border-left: 1px solid rgba(30, 41, 59, 0.5);
}

.form-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  justify-content: center;
  max-width: 400px;
  margin: 0 auto;
  width: 100%;
}

.form-header {
  margin-bottom: 40px;
}

.form-title {
  font-family: var(--font-heading);
  font-size: 32px;
  font-weight: 700;
  color: var(--color-text);
  margin: 0 0 8px;
}

.form-subtitle {
  font-size: 14px;
  color: var(--color-text-muted);
  margin: 0;
}

.login-form {
  margin-bottom: 24px;
}

.input-field :deep(.ant-input),
.input-field :deep(.ant-input-password) {
  background: rgba(17, 24, 39, 0.8);
  border: 1px solid rgba(30, 41, 59, 1);
  border-radius: 12px;
  padding: 12px 16px;
  font-size: 15px;
  color: var(--color-text);
  transition: all 0.3s;
}

.input-field :deep(.ant-input:focus),
.input-field :deep(.ant-input-password:focus),
.input-field :deep(.ant-input-password-focused) {
  background: rgba(17, 24, 39, 1);
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px rgba(14, 165, 233, 0.1);
}

.input-field :deep(.ant-input::placeholder) {
  color: var(--color-text-muted);
}

.input-field :deep(.ant-input-prefix) {
  margin-right: 12px;
}

.input-icon {
  font-size: 16px;
  color: var(--color-text-muted);
}

.input-field :deep(.ant-form-item-label > label) {
  color: var(--color-text);
  font-weight: 500;
  font-size: 14px;
}

.form-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.remember-checkbox :deep(.ant-checkbox-checked .ant-checkbox-inner) {
  background-color: var(--color-primary);
  border-color: var(--color-primary);
}

.remember-checkbox :deep(span) {
  color: var(--color-text-muted);
}

.forgot-link {
  color: var(--color-primary-light);
  text-decoration: none;
  font-size: 14px;
  transition: color 0.3s;
}

.forgot-link:hover {
  color: var(--color-primary);
}

.submit-button {
  height: 50px;
  background: linear-gradient(135deg, var(--color-primary), var(--color-primary-dark));
  border: none;
  border-radius: 12px;
  font-size: 16px;
  font-weight: 600;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  transition: all 0.3s;
}

.submit-button:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 24px rgba(14, 165, 233, 0.4);
}

.button-arrow {
  transition: transform 0.3s;
}

.submit-button:hover .button-arrow {
  transform: translateX(4px);
}

.form-divider {
  display: flex;
  align-items: center;
  margin: 24px 0;
  color: var(--color-text-muted);
  font-size: 13px;
}

.form-divider::before,
.form-divider::after {
  content: '';
  flex: 1;
  height: 1px;
  background: rgba(30, 41, 59, 1);
}

.form-divider span {
  padding: 0 16px;
}

.form-footer {
  text-align: center;
}

.footer-text {
  font-size: 14px;
  color: var(--color-text-muted);
  margin: 0;
}

.register-link {
  color: var(--color-primary-light);
  text-decoration: none;
  font-weight: 600;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  transition: all 0.3s;
}

.register-link:hover {
  color: var(--color-primary);
}

.link-arrow {
  transition: transform 0.3s;
}

.register-link:hover .link-arrow {
  transform: translateX(4px);
}

.form-bottom {
  margin-top: 40px;
  text-align: center;
}

.copyright {
  font-size: 12px;
  color: var(--color-text-muted);
  margin: 0;
}

/* Responsive */
@media (max-width: 1024px) {
  .login-container {
    grid-template-columns: 1fr;
    max-width: 500px;
    min-height: auto;
  }

  .brand-panel {
    display: none;
  }

  .form-panel {
    border-left: none;
    padding: 40px 30px;
  }
}

@media (max-width: 480px) {
  .form-panel {
    padding: 30px 20px;
  }

  .form-content {
    max-width: 100%;
  }

  .brand-title,
  .form-title {
    font-size: 24px;
  }

  .submit-button {
    height: 46px;
  }
}
</style>

<style>
/* Global styles for login page - ensure Ant Design components use dark theme colors */
.login-page {
  color: #f1f5f9;
}

.login-page :deep(.ant-form-item-label > label) {
  color: #f1f5f9;
}

.login-page :deep(.ant-form-item-explain) {
  color: #94a3b8;
}

.login-page :deep(.ant-checkbox-wrapper) {
  color: #94a3b8;
}

.login-page :deep(.ant-input::placeholder),
.login-page :deep(.ant-input-password::placeholder) {
  color: #64748b;
}

.login-page :deep(.ant-empty-description) {
  color: #94a3b8;
}
</style>
