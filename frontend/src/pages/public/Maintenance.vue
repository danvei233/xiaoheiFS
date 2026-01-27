<template>
  <div class="maintenance-page">
    <div class="maintenance-container">
      <div class="maintenance-icon">🔧</div>
      <h1 class="maintenance-title">{{ $t("maintenance.title") || "系统维护中" }}</h1>
      <p class="maintenance-message">{{ message || $t("maintenance.defaultMessage") || "系统正在进行维护，请稍后再试" }}</p>
      <div class="maintenance-animation">
        <div class="dot"></div>
        <div class="dot"></div>
        <div class="dot"></div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { defineProps } from "vue";

defineProps<{
  message?: string;
}>();
</script>

<style scoped>
.maintenance-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: 24px;
}

.maintenance-container {
  text-align: center;
  color: #fff;
  max-width: 500px;
}

.maintenance-icon {
  font-size: 80px;
  margin-bottom: 24px;
  animation: bounce 2s ease-in-out infinite;
}

@keyframes bounce {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-20px); }
}

.maintenance-title {
  font-size: 36px;
  font-weight: 600;
  margin: 0 0 16px;
}

.maintenance-message {
  font-size: 18px;
  opacity: 0.9;
  margin: 0 0 40px;
}

.maintenance-animation {
  display: flex;
  justify-content: center;
  gap: 12px;
}

.dot {
  width: 12px;
  height: 12px;
  background: #fff;
  border-radius: 50%;
  animation: pulse 1.4s ease-in-out infinite;
}

.dot:nth-child(1) { animation-delay: 0s; }
.dot:nth-child(2) { animation-delay: 0.2s; }
.dot:nth-child(3) { animation-delay: 0.4s; }

@keyframes pulse {
  0%, 100% {
    opacity: 0.3;
    transform: scale(1);
  }
  50% {
    opacity: 1;
    transform: scale(1.2);
  }
}
</style>
