<template>
  <div class="not-found-page">
    <canvas ref="particleCanvas" class="particle-canvas"></canvas>
    
    <div class="not-found-container">
      <div class="error-visual">
        <div class="error-code">404</div>
        <div class="glitch-overlay" aria-hidden="true">404</div>
        <div class="glitch-overlay" aria-hidden="true">404</div>
      </div>

      <div class="error-content">
        <h1 class="error-title">页面未找到</h1>
        <p class="error-description">
          抱歉，您访问的页面不存在或已被移除
        </p>

        <div class="error-actions">
          <a href="/" class="btn btn-primary">
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
              <path d="M2 6L8 2L14 6V13C14 13.5304 13.7893 14.0391 13.4142 14.4142C13.0391 14.7893 12.5304 15 12 15H4C3.46957 15 2.96086 14.7893 2.58579 14.4142C2.21071 14.0391 2 13.5304 2 13V6Z" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
              <path d="M6 15V9H10V15" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
            返回首页
          </a>
          <button @click="goBack" class="btn btn-secondary">
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
              <path d="M8 14L2 8L8 2" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
              <path d="M2 8H14" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
            返回上一页
          </button>
        </div>
      </div>

      <div class="floating-elements">
        <div class="floating-icon icon-1">
          <svg width="32" height="32" viewBox="0 0 24 24" fill="none">
            <path d="M12 2L2 7L12 12L22 7L12 2Z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
            <path d="M2 17L12 22L22 17" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
            <path d="M2 12L12 17L22 12" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
          </svg>
        </div>
        <div class="floating-icon icon-2">
          <svg width="32" height="32" viewBox="0 0 24 24" fill="none">
            <rect x="2" y="3" width="20" height="14" rx="2" stroke="currentColor" stroke-width="2"/>
            <path d="M8 21H16" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
            <path d="M12 17V21" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
          </svg>
        </div>
        <div class="floating-icon icon-3">
          <svg width="32" height="32" viewBox="0 0 24 24" fill="none">
            <circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="2"/>
            <path d="M12 6V12L16 14" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
          </svg>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue';
import { useRouter } from 'vue-router';

const router = useRouter();
const particleCanvas = ref<HTMLCanvasElement | null>(null);

const goBack = () => {
  if (window.history.length > 1) {
    router.back();
  } else {
    router.push('/');
  }
};

class Particle {
  x: number;
  y: number;
  vx: number;
  vy: number;
  radius: number;
  opacity: number;

  constructor(canvas: HTMLCanvasElement) {
    this.x = Math.random() * canvas.width;
    this.y = Math.random() * canvas.height;
    this.vx = (Math.random() - 0.5) * 0.5;
    this.vy = (Math.random() - 0.5) * 0.5;
    this.radius = Math.random() * 2 + 1;
    this.opacity = Math.random() * 0.5 + 0.2;
  }

  update(canvas: HTMLCanvasElement) {
    this.x += this.vx;
    this.y += this.vy;

    if (this.x < 0 || this.x > canvas.width) this.vx *= -1;
    if (this.y < 0 || this.y > canvas.height) this.vy *= -1;
  }

  draw(ctx: CanvasRenderingContext2D) {
    ctx.beginPath();
    ctx.arc(this.x, this.y, this.radius, 0, Math.PI * 2);
    ctx.fillStyle = `rgba(14, 165, 233, ${this.opacity})`;
    ctx.fill();
  }
}

let particles: Particle[] = [];
let animationId: number | null = null;

const initParticles = () => {
  const canvas = particleCanvas.value;
  if (!canvas) return;

  canvas.width = window.innerWidth;
  canvas.height = window.innerHeight;

  particles = [];
  const particleCount = Math.floor((canvas.width * canvas.height) / 20000);

  for (let i = 0; i < particleCount; i++) {
    particles.push(new Particle(canvas));
  }
};

const animateParticles = () => {
  const canvas = particleCanvas.value;
  const ctx = canvas?.getContext('2d');
  if (!canvas || !ctx) return;

  ctx.clearRect(0, 0, canvas.width, canvas.height);

  particles.forEach((particle) => {
    particle.update(canvas);
    particle.draw(ctx);
  });

  animationId = requestAnimationFrame(animateParticles);
};

onMounted(() => {
  initParticles();
  animateParticles();
  window.addEventListener('resize', initParticles);
});

onUnmounted(() => {
  if (animationId) cancelAnimationFrame(animationId);
  window.removeEventListener('resize', initParticles);
});
</script>

<style scoped>
@import url('https://fonts.googleapis.com/css2?family=Outfit:wght@700;800;900&family=Work+Sans:wght@400;500;600&display=swap');

.not-found-page {
  min-height: 100vh;
  background: #0a0e17;
  position: relative;
  overflow: hidden;
  font-family: 'Work Sans', sans-serif;
}

.particle-canvas {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  pointer-events: none;
  z-index: 0;
}

.not-found-container {
  position: relative;
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 24px;
  z-index: 1;
}

.error-visual {
  position: relative;
  margin-bottom: 48px;
}

.error-code {
  font-family: 'Outfit', sans-serif;
  font-size: 180px;
  font-weight: 900;
  line-height: 1;
  background: linear-gradient(135deg, #0ea5e9 0%, #38bdf8 50%, #0284c7 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  position: relative;
  animation: pulse 3s ease-in-out infinite;
}

.glitch-overlay {
  position: absolute;
  top: 0;
  left: 0;
  font-family: 'Outfit', sans-serif;
  font-size: 180px;
  font-weight: 900;
  line-height: 1;
  background: linear-gradient(135deg, #0ea5e9 0%, #38bdf8 50%, #0284c7 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  opacity: 0.8;
  pointer-events: none;
}

.glitch-overlay:nth-child(2) {
  animation: glitch1 2.5s infinite;
  color: #0ea5e9;
  z-index: -1;
}

.glitch-overlay:nth-child(3) {
  animation: glitch2 2.5s infinite;
  color: #f97316;
  z-index: -2;
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.8;
  }
}

@keyframes glitch1 {
  0%, 100% {
    transform: translate(0);
    opacity: 0;
  }
  20% {
    transform: translate(-3px, 3px);
    opacity: 0.8;
  }
  40% {
    transform: translate(-3px, -3px);
    opacity: 0;
  }
  60% {
    transform: translate(3px, 3px);
    opacity: 0.8;
  }
  80% {
    transform: translate(3px, -3px);
    opacity: 0;
  }
}

@keyframes glitch2 {
  0%, 100% {
    transform: translate(0);
    opacity: 0;
  }
  25% {
    transform: translate(2px, -2px);
    opacity: 0.6;
  }
  50% {
    transform: translate(-2px, 2px);
    opacity: 0;
  }
  75% {
    transform: translate(2px, 2px);
    opacity: 0.6;
  }
}

.error-content {
  text-align: center;
  max-width: 600px;
  animation: fadeInUp 0.8s ease-out;
}

.error-title {
  font-family: 'Outfit', sans-serif;
  font-size: 42px;
  font-weight: 700;
  color: #f1f5f9;
  margin: 0 0 16px;
  letter-spacing: -0.02em;
}

.error-description {
  font-size: 18px;
  line-height: 1.7;
  color: #94a3b8;
  margin: 0 0 40px;
}

.error-actions {
  display: flex;
  gap: 16px;
  justify-content: center;
  flex-wrap: wrap;
}

.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 14px 28px;
  border-radius: 12px;
  font-family: 'Work Sans', sans-serif;
  font-size: 15px;
  font-weight: 600;
  text-decoration: none;
  border: none;
  cursor: pointer;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  position: relative;
  overflow: hidden;
}

.btn::before {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(45deg, transparent, rgba(255, 255, 255, 0.1), transparent);
  transform: translateX(-100%);
  transition: transform 0.6s;
}

.btn:hover::before {
  transform: translateX(100%);
}

.btn-primary {
  background: linear-gradient(135deg, #0ea5e9 0%, #0284c7 100%);
  color: white;
  box-shadow: 0 4px 20px rgba(14, 165, 233, 0.3);
}

.btn-primary:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 30px rgba(14, 165, 233, 0.4);
}

.btn-secondary {
  background: transparent;
  color: #f1f5f9;
  border: 1px solid #1e293b;
}

.btn-secondary:hover {
  background: rgba(255, 255, 255, 0.05);
  border-color: #94a3b8;
}

.floating-elements {
  position: absolute;
  inset: 0;
  pointer-events: none;
}

.floating-icon {
  position: absolute;
  width: 64px;
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(14, 165, 233, 0.1);
  border: 1px solid rgba(14, 165, 233, 0.2);
  border-radius: 16px;
  color: #38bdf8;
  animation: float 6s ease-in-out infinite;
}

.icon-1 {
  top: 15%;
  left: 10%;
  animation-delay: 0s;
}

.icon-2 {
  top: 60%;
  right: 15%;
  animation-delay: 2s;
}

.icon-3 {
  bottom: 20%;
  left: 15%;
  animation-delay: 4s;
}

@keyframes float {
  0%, 100% {
    transform: translateY(0) rotate(0deg);
  }
  50% {
    transform: translateY(-20px) rotate(5deg);
  }
}

@keyframes fadeInUp {
  from {
    opacity: 0;
    transform: translateY(30px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@media (max-width: 768px) {
  .error-code {
    font-size: 120px;
  }

  .glitch-overlay {
    font-size: 120px;
  }

  .error-title {
    font-size: 32px;
  }

  .error-description {
    font-size: 16px;
  }

  .error-actions {
    flex-direction: column;
    width: 100%;
  }

  .btn {
    width: 100%;
  }

  .floating-icon {
    width: 48px;
    height: 48px;
  }

  .floating-icon svg {
    width: 24px;
    height: 24px;
  }
}

@media (max-width: 480px) {
  .error-code {
    font-size: 80px;
  }

  .glitch-overlay {
    font-size: 80px;
  }

  .error-title {
    font-size: 24px;
  }

  .error-description {
    font-size: 14px;
  }

  .not-found-container {
    padding: 24px 16px;
  }
}
</style>
