<template>
  <component :is="dynamicComponent" v-if="dynamicComponent" />
  <NotFound v-else-if="!loading && showNotFound" />
  <div v-else class="loading-container">
    <div class="loading-spinner"></div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch, defineAsyncComponent } from "vue";
import type { Component } from "vue";
import { useRoute, useRouter } from "vue-router";
import NotFound from "@/pages/public/NotFound.vue";
import { checkAdminPath, getCachedAdminPath } from "@/services/adminPath";
import { useAdminAuthStore } from "@/stores/adminAuth";

const route = useRoute();
const router = useRouter();
const adminAuth = useAdminAuthStore();
const loading = ref(true);
const showNotFound = ref(false);
const dynamicComponent = ref<Component | null>(null);
let lastAdminPath = ""; // 记录上一次的管理端路径

const loadComponent = async () => {
  const pathSegments = route.path.split("/").filter(Boolean);
  const firstSegment = pathSegments[0];
  const subPath = pathSegments.slice(1).join("/");
  
  // 检查是否是管理端内部导航（同一个管理端路径下的子页面切换）
  const cachedPath = getCachedAdminPath();
  const isAdminInternalNav = lastAdminPath && firstSegment === lastAdminPath && firstSegment === cachedPath;
  
  // 检查是否从登录页跳转到管理端（需要重新加载）
  const isFromLogin = !subPath || subPath === "" || subPath === "login" || subPath === "forgot-password" || subPath === "reset-password";
  const wasFromLogin = !dynamicComponent.value || dynamicComponent.value.name === "Login";
  
  // 如果是管理端内部导航，但不是从登录页跳转，不重置组件，避免闪烁
  if (isAdminInternalNav && !wasFromLogin && !isFromLogin) {
    return;
  }
  
  // 其他情况重置组件
  loading.value = true;
  showNotFound.value = false;
  dynamicComponent.value = null;
  
  if (pathSegments.length === 0) {
    loading.value = false;
    showNotFound.value = true;
    return;
  }
  
  // 跳过已知的公开路径（但不包括 "admin"，因为它可能是自定义管理端路径）
  const knownPaths = [
    "login", "register", "forgot-password", "reset-password", 
    "console", "install", "products", "help", "docs", 
    "announcements", "activities", "tutorials", "api", "uploads", "assets"
  ];
  
  if (knownPaths.includes(firstSegment)) {
    loading.value = false;
    showNotFound.value = true;
    return;
  }
  
  try {
    // 检查是否是管理端路径（内部有缓存，验证过的路径不会再调用 API）
    const result = await checkAdminPath(firstSegment);
    
    if (result.isAdmin) {
      // 记录当前管理端路径
      lastAdminPath = firstSegment;
      
      // 根据子路径加载对应的管理端组件
      if (!subPath || subPath === "") {
        // 访问 /admin 时，如果已登录跳转到 console，否则跳转到登录页
        if (adminAuth.token) {
          await router.replace(`/${firstSegment}/console`);
          loading.value = false;
          return;
        } else {
          await router.replace(`/${firstSegment}/login`);
          loading.value = false;
          return;
        }
      } else if (subPath === "login") {
        // 如果已登录，访问登录页时跳转到 console
        if (adminAuth.token) {
          await router.replace(`/${firstSegment}/console`);
          loading.value = false;
          return;
        }
        dynamicComponent.value = defineAsyncComponent(() => import("@/pages/admin/Login.vue"));
      } else if (subPath === "forgot-password") {
        dynamicComponent.value = defineAsyncComponent(() => import("@/pages/admin/ForgotPassword.vue"));
      } else if (subPath === "reset-password") {
        dynamicComponent.value = defineAsyncComponent(() => import("@/pages/admin/ResetPassword.vue"));
      } else {
        if (!adminAuth.token) {
          await router.replace({
            path: `/${firstSegment}/login`,
            query: { redirect: route.fullPath }
          });
          loading.value = false;
          return;
        }
        dynamicComponent.value = defineAsyncComponent(() => import("@/components/DynamicAdminWrapper.vue"));
      }
    } else {
      showNotFound.value = true;
    }
  } catch (error) {
    console.error("Failed to check admin path:", error);
    showNotFound.value = true;
  } finally {
    loading.value = false;
  }
};

onMounted(() => {
  loadComponent();
});

// 监听路由变化，重新加载组件
watch(() => route.path, () => {
  loadComponent();
});
</script>

<style scoped>
.loading-container {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  background: linear-gradient(135deg, #0f172a 0%, #1e293b 100%);
}

.loading-spinner {
  width: 48px;
  height: 48px;
  border: 4px solid rgba(34, 211, 238, 0.2);
  border-top-color: #22d3ee;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>

