import { createRouter, createWebHistory } from "vue-router";
import { Modal } from "ant-design-vue";
import PublicLayout from "@/layouts/PublicLayout.vue";
import UserLayout from "@/layouts/UserLayout.vue";
import AdminLayout from "@/layouts/AdminLayout.vue";
import { useAuthStore } from "@/stores/auth";
import { useAdminAuthStore } from "@/stores/adminAuth";
import { useInstallStore } from "@/stores/install";

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: "/",
      component: PublicLayout,
      children: [
        { path: "", name: "public-home", component: () => import("@/pages/public/Home.vue") },
        { path: "products", name: "public-products", component: () => import("@/pages/public/Products.vue") },
        { path: "help", name: "public-help", component: () => import("@/pages/public/Help.vue") },
        {
          path: "docs",
          name: "public-docs",
          component: () => import("@/pages/public/PostsList.vue"),
          meta: { categoryKey: "docs", title: "文档中心", subtitle: "官方文档与最佳实践" }
        },
        {
          path: "announcements",
          name: "public-announcements",
          component: () => import("@/pages/public/PostsList.vue"),
          meta: { categoryKey: "announcements", title: "最新公告", subtitle: "产品动态与重要通知" }
        },
        {
          path: "activities",
          name: "public-activities",
          component: () => import("@/pages/public/PostsList.vue"),
          meta: { categoryKey: "activities", title: "活动中心", subtitle: "限时活动与优惠计划" }
        },
        {
          path: "tutorials",
          name: "public-tutorials",
          component: () => import("@/pages/public/PostsList.vue"),
          meta: { categoryKey: "tutorials", title: "教程学院", subtitle: "从入门到进阶的学习路径" }
        },
        {
          path: ":category(docs|announcements|activities|tutorials)/:slug",
          name: "public-post-detail",
          component: () => import("@/pages/public/PostDetail.vue")
        }
      ]
    },
    {
      path: "/install",
      name: "install",
      component: () => import("@/pages/public/install/InstallWizard.vue")
    },
    { path: "/login", name: "login", component: () => import("@/pages/auth/Login.vue") },
    { path: "/register", name: "register", component: () => import("@/pages/auth/Register.vue") },
    { path: "/forgot-password", name: "forgot-password", component: () => import("@/pages/auth/ForgotPassword.vue") },
    { path: "/reset-password", name: "reset-password", component: () => import("@/pages/auth/ResetPassword.vue") },
    {
      path: "/console",
      component: UserLayout,
      meta: { requiresUser: true },
      children: [
        { path: "", name: "console-dashboard", component: () => import("@/pages/console/Dashboard.vue") },
        { path: "buy", name: "console-buy", component: () => import("@/pages/console/BuyVps.vue") },
        { path: "profile", name: "console-profile", component: () => import("@/pages/console/Profile.vue") },
        { path: "vps", name: "console-vps", component: () => import("@/pages/console/VpsList.vue") },
        { path: "vps/:id", name: "console-vps-detail", component: () => import("@/pages/console/VpsDetail.vue") },
        { path: "cart", name: "console-cart", component: () => import("@/pages/console/Cart.vue") },
        { path: "orders", name: "console-orders", component: () => import("@/pages/console/Orders.vue") },
        { path: "orders/:id", name: "console-order-detail", component: () => import("@/pages/console/OrderDetail.vue") },
        { path: "billing", name: "console-billing", component: () => import("@/pages/console/Billing.vue") },
        { path: "api-keys", name: "console-api-keys", component: () => import("@/pages/console/ApiKeys.vue") },
        { path: "realname", name: "console-realname", component: () => import("@/pages/console/Realname.vue") },
        { path: "tickets", name: "console-tickets", component: () => import("@/pages/console/Tickets.vue") },
        { path: "tickets/:id", name: "console-ticket-detail", component: () => import("@/pages/console/TicketDetail.vue") }
      ]
    },
    {
      path: "/:pathMatch(.*)*",
      name: "dynamic-route",
      component: () => import("@/pages/public/DynamicRoute.vue")
    }
  ]
});

router.beforeEach(async (to) => {
  if (to.path.startsWith("/console") && typeof to.hash === "string" && to.hash.includes("impersonate_token=")) {
    const auth = useAuthStore();
    const hashParams = new URLSearchParams(to.hash.replace(/^#/, ""));
    const impersonateToken = hashParams.get("impersonate_token") || "";
    if (impersonateToken) {
      auth.token = impersonateToken;
      auth.profile = null;
      localStorage.setItem("user_token", impersonateToken);
    }
    hashParams.delete("impersonate_token");
    const restHash = hashParams.toString();
    return {
      path: to.path,
      query: to.query,
      hash: restHash ? `#${restHash}` : "",
      replace: true
    };
  }

  // Check installation status
  const install = useInstallStore();
  
  if (!install.loaded) {
    await install.fetchStatus();
  }
  
  // If not installed and trying to access homepage, redirect to install
  if ((to.path === "/" || to.name === "public-home") && !install.installed) {
    return { path: "/install", replace: true };
  }

  if (to.meta.requiresUser) {
    const auth = useAuthStore();
    if (!auth.token) {
      return { name: "login", query: { redirect: to.fullPath } };
    }
  }
  if (to.meta.requiresAdmin) {
    const admin = useAdminAuthStore();
    if (!admin.token) {
      return { name: "admin-login", query: { redirect: to.fullPath } };
    }
  }
  return true;
});

let chunkErrorPrompted = false;
router.onError((err, to) => {
  const msg = String((err as any)?.message || err || "");
  // Common in production when a chunk is missing/stale (e.g. after deployment) or a route-level import fails.
  if (
    msg.includes("Failed to fetch dynamically imported module") ||
    msg.includes("Importing a module script failed") ||
    msg.includes("Loading chunk") ||
    msg.includes("ChunkLoadError")
  ) {
    if (chunkErrorPrompted) {
      return;
    }
    chunkErrorPrompted = true;
    Modal.confirm({
      title: "页面资源加载失败",
      content: "可能是资源更新或缓存导致。是否刷新页面后重试？",
      okText: "刷新",
      cancelText: "取消",
      onOk: () => window.location.reload(),
      onCancel: () => {
        chunkErrorPrompted = false;
      }
    });
    return;
  }
  // eslint-disable-next-line no-console
  console.error("Router error:", err);
});

export default router;
