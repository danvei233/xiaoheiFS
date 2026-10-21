<template>
  <div>
  <slot v-if="!hasError" />
  <div v-else class="error-boundary">
    <div class="error-card">
      <div class="title">页面渲染失败</div>
      <div class="desc">通常是接口返回异常或前端运行时错误导致。可先刷新重试。</div>
      <pre v-if="message" class="message">{{ message }}</pre>
      <div class="actions">
        <a-button type="primary" @click="reload">刷新页面</a-button>
      </div>
    </div>
  </div>
  </div>
</template>

<script setup lang="ts">
import { onErrorCaptured, ref } from "vue";

const hasError = ref(false);
const message = ref("");

const reload = () => window.location.reload();

onErrorCaptured((err) => {
  const msg = String((err as any)?.message || err || "");
  // Keep the app running for expected request errors handled by page logic.
  if (msg.includes("Request failed with status code")) {
    return false;
  }
  hasError.value = true;
  message.value = msg;
  // prevent bubbling to global handler so we don't end up with a blank page
  return false;
});
</script>

<style scoped>
.error-boundary {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background: #f5f7fa;
}

.error-card {
  width: 100%;
  max-width: 720px;
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  padding: 18px 18px 16px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.06);
}

.title {
  font-size: 18px;
  font-weight: 700;
  color: #111827;
  margin-bottom: 6px;
}

.desc {
  color: #6b7280;
  margin-bottom: 12px;
}

.message {
  white-space: pre-wrap;
  background: #0b1220;
  color: #e5e7eb;
  border-radius: 10px;
  padding: 12px;
  margin: 0 0 12px;
  max-height: 240px;
  overflow: auto;
  font-size: 12px;
}

.actions {
  display: flex;
  justify-content: flex-end;
}
</style>
