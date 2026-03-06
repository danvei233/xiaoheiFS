<template>
  <AdminLayout>
    <template #default>
      <Transition name="fade" mode="out-in">
        <component :is="pageComponent" v-if="pageComponent" :key="route.path" />
        <div v-else class="loading">加载中...</div>
      </Transition>
    </template>
  </AdminLayout>
</template>

<script setup lang="ts">
import { ref, watch, defineAsyncComponent } from "vue";
import type { Component } from "vue";
import { useRoute } from "vue-router";
import AdminLayout from "@/layouts/AdminLayout.vue";

const route = useRoute();
const pageComponent = ref<Component | null>(null);

const loadPage = () => {
  // 重置组件，确保路由变化时重新加载
  pageComponent.value = null;
  
  // 解析路径，加载对应的管理端页面
  const pathSegments = route.path.split("/").filter(Boolean);
  const subPath = pathSegments.slice(1).join("/");
  
  // 路由映射
  const routeMap: Record<string, () => Promise<any>> = {
    "": () => import("@/pages/admin/Dashboard.vue"),
    "console": () => import("@/pages/admin/Dashboard.vue"),
    "revenue-analytics": () => import("@/pages/admin/RevenueAnalytics.vue"),
    "orders": () => import("@/pages/admin/Orders.vue"),
    "wallet/orders": () => import("@/pages/admin/WalletOrders.vue"),
    "vps": () => import("@/pages/admin/Vps.vue"),
    "probes": () => import("@/pages/admin/Probes.vue"),
    "users": () => import("@/pages/admin/Users.vue"),
    "user-tiers": () => import("@/pages/admin/UserTiers.vue"),
    "coupons": () => import("@/pages/admin/Coupons.vue"),
    "admins": () => import("@/pages/admin/Admins.vue"),
    "permission-groups": () => import("@/pages/admin/PermissionGroups.vue"),
    "profile": () => import("@/pages/admin/Profile.vue"),
    "catalog": () => import("@/pages/admin/Catalog.vue"),
    "systems": () => import("@/pages/admin/Systems.vue"),
    "settings/site": () => import("@/pages/admin/settings/Site.vue"),
    "settings/auth": () => import("@/pages/admin/settings/Auth.vue"),
    "settings/captcha": () => import("@/pages/admin/settings/Captcha.vue"),
    "settings/email": () => import("@/pages/admin/settings/Email.vue"),
    "settings/sms": () => import("@/pages/admin/settings/SMS.vue"),
    "settings/apikey": () => import("@/pages/admin/settings/ApiKey.vue"),
    "settings/webhook": () => import("@/pages/admin/settings/Webhook.vue"),
    "settings/payments": () => import("@/pages/admin/settings/Payments.vue"),
    "settings/fcm": () => import("@/pages/admin/settings/Fcm.vue"),
    "settings/pricing": () => import("@/pages/admin/settings/Pricing.vue"),
    "settings/lifecycle": () => import("@/pages/admin/settings/Lifecycle.vue"),
    "settings/plugins": () => import("@/pages/admin/settings/Plugins.vue"),
    "realname/providers": () => import("@/pages/admin/realname/Providers.vue"),
    "realname/config": () => import("@/pages/admin/realname/Config.vue"),
    "realname/records": () => import("@/pages/admin/realname/Records.vue"),
    "automation": () => import("@/pages/admin/Automation.vue"),
    "scheduled-tasks": () => import("@/pages/admin/ScheduledTasks.vue"),
    "debug": () => import("@/pages/admin/Debug.vue"),
    "audit": () => import("@/pages/admin/Audit.vue"),
    "tickets": () => import("@/pages/admin/Tickets.vue"),
    "cms/categories": () => import("@/pages/admin/cms/Categories.vue"),
    "cms/posts": () => import("@/pages/admin/cms/Posts.vue"),
    "cms/blocks": () => import("@/pages/admin/cms/Blocks.vue"),
    "cms/nav-items": () => import("@/pages/admin/cms/NavItems.vue"),
    "cms/uploads": () => import("@/pages/admin/cms/Uploads.vue")
  };
  
  // 检查动态路由
  if (subPath.match(/^probes\/\d+$/)) {
    pageComponent.value = defineAsyncComponent(() => import("@/pages/admin/ProbeDetail.vue"));
  } else if (subPath.match(/^tickets\/\d+$/)) {
    pageComponent.value = defineAsyncComponent(() => import("@/pages/admin/TicketDetail.vue"));
  } else if (routeMap[subPath]) {
    pageComponent.value = defineAsyncComponent(routeMap[subPath]);
  } else {
    // 未找到对应页面，直接显示 404（不延迟）
    pageComponent.value = defineAsyncComponent(() => import("@/pages/public/NotFound.vue"));
  }
};

// 初始加载和监听路由变化
loadPage();
watch(() => route.path, loadPage);
</script>

<style scoped>
.loading {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 400px;
  font-size: 16px;
  color: rgba(255, 255, 255, 0.7);
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.15s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>

