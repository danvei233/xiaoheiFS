import { createRouter, createWebHistory } from "vue-router";
import PublicLayout from "@/layouts/PublicLayout.vue";
import InstallLayout from "@/layouts/InstallLayout.vue";
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
      component: InstallLayout,
      children: [
        { path: "", redirect: { name: "install-db" } },
        { path: "db", name: "install-db", component: () => import("@/pages/public/install/DbStep.vue") },
        { path: "site", name: "install-site", component: () => import("@/pages/public/install/SiteStep.vue") },
        { path: "admin", name: "install-admin", component: () => import("@/pages/public/install/AdminStep.vue") },
        { path: "done", name: "install-done", component: () => import("@/pages/public/install/DoneStep.vue") }
      ]
    },
    { path: "/login", name: "login", component: () => import("@/pages/auth/Login.vue") },
    { path: "/register", name: "register", component: () => import("@/pages/auth/Register.vue") },
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
        { path: "realname", name: "console-realname", component: () => import("@/pages/console/Realname.vue") },
        { path: "tickets", name: "console-tickets", component: () => import("@/pages/console/Tickets.vue") },
        { path: "tickets/:id", name: "console-ticket-detail", component: () => import("@/pages/console/TicketDetail.vue") }
      ]
    },
    { path: "/admin/login", name: "admin-login", component: () => import("@/pages/admin/Login.vue") },
{ path: "/admin/forgot-password", name: "admin-forgot-password", component: () => import("@/pages/admin/ForgotPassword.vue") },
{ path: "/admin/reset-password", name: "admin-reset-password", component: () => import("@/pages/admin/ResetPassword.vue") },
    {
      path: "/admin",
      component: AdminLayout,
      meta: { requiresAdmin: true },
      children: [
        { path: "", redirect: "console" },
        { path: "console", name: "admin-dashboard", component: () => import("@/pages/admin/Dashboard.vue") },
        { path: "orders", name: "admin-orders", component: () => import("@/pages/admin/Orders.vue") },
        { path: "wallet/orders", name: "admin-wallet-orders", component: () => import("@/pages/admin/WalletOrders.vue") },
        { path: "vps", name: "admin-vps", component: () => import("@/pages/admin/Vps.vue") },
        { path: "users", name: "admin-users", component: () => import("@/pages/admin/Users.vue") },
        { path: "admins", name: "admin-admins", component: () => import("@/pages/admin/Admins.vue") },
        { path: "permission-groups", name: "admin-permission-groups", component: () => import("@/pages/admin/PermissionGroups.vue") },
        { path: "profile", name: "admin-profile", component: () => import("@/pages/admin/Profile.vue") },
        { path: "catalog", name: "admin-catalog", component: () => import("@/pages/admin/Catalog.vue") },
        { path: "systems", name: "admin-systems", component: () => import("@/pages/admin/Systems.vue") },
        { path: "settings/site", name: "admin-settings-site", component: () => import("@/pages/admin/settings/Site.vue") },
        { path: "settings/email", name: "admin-settings-email", component: () => import("@/pages/admin/settings/Email.vue") },
        { path: "settings/apikey", name: "admin-settings-apikey", component: () => import("@/pages/admin/settings/ApiKey.vue") },
        { path: "settings/webhook", name: "admin-settings-webhook", component: () => import("@/pages/admin/settings/Webhook.vue") },
        { path: "settings/payments", name: "admin-settings-payments", component: () => import("@/pages/admin/settings/Payments.vue") },
        { path: "settings/pricing", name: "admin-settings-pricing", component: () => import("@/pages/admin/settings/Pricing.vue") },
        { path: "settings/lifecycle", name: "admin-settings-lifecycle", component: () => import("@/pages/admin/settings/Lifecycle.vue") },
        {
          path: "settings/payment-plugins",
          name: "admin-settings-payment-plugins",
          component: () => import("@/pages/admin/settings/PaymentPlugins.vue")
        },
        { path: "realname/providers", name: "admin-realname-providers", component: () => import("@/pages/admin/realname/Providers.vue") },
        { path: "realname/config", name: "admin-realname-config", component: () => import("@/pages/admin/realname/Config.vue") },
        { path: "realname/records", name: "admin-realname-records", component: () => import("@/pages/admin/realname/Records.vue") },
        { path: "automation", name: "admin-automation", component: () => import("@/pages/admin/Automation.vue") },
        { path: "scheduled-tasks", name: "admin-scheduled-tasks", component: () => import("@/pages/admin/ScheduledTasks.vue") },
        { path: "debug", name: "admin-debug", component: () => import("@/pages/admin/Debug.vue") },
        { path: "audit", name: "admin-audit", component: () => import("@/pages/admin/Audit.vue") },
        { path: "tickets", name: "admin-tickets", component: () => import("@/pages/admin/Tickets.vue") },
        { path: "tickets/:id", name: "admin-ticket-detail", component: () => import("@/pages/admin/TicketDetail.vue") },
        { path: "cms/categories", name: "admin-cms-categories", component: () => import("@/pages/admin/cms/Categories.vue") },
        { path: "cms/posts", name: "admin-cms-posts", component: () => import("@/pages/admin/cms/Posts.vue") },
        { path: "cms/blocks", name: "admin-cms-blocks", component: () => import("@/pages/admin/cms/Blocks.vue") },
        { path: "cms/nav-items", name: "admin-cms-nav-items", component: () => import("@/pages/admin/cms/NavItems.vue") },
        { path: "cms/uploads", name: "admin-cms-uploads", component: () => import("@/pages/admin/cms/Uploads.vue") }
      ]
    }
  ]
});

router.beforeEach(async (to) => {
  const install = useInstallStore();
  if (!install.loaded) {
    await install.fetchStatus();
  }
  if (install.loaded && !install.installed && !to.path.startsWith("/install")) {
    return { path: "/install/db", query: { redirect: to.fullPath } };
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

export default router;
