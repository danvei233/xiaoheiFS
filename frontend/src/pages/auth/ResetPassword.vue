<template>
  <div class="reset-page">
    <a-card class="reset-card" title="重置密码">
      <p>请通过“找回密码”流程完成身份验证后再重置密码。</p>
      <a-button type="primary" @click="goForgot">前往找回密码</a-button>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from "vue-router";
const router = useRouter();
const goForgot = () => router.replace("/forgot-password");
</script>

<style scoped>
.reset-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
}
.reset-card {
  width: 100%;
  max-width: 420px;
}
</style>

