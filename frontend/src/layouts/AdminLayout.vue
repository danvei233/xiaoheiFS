<template>
  <a-config-provider :theme="{ algorithm: theme.defaultAlgorithm }">
    <a-layout class="layout admin-layout">
    <a-layout-header class="header">
      <div class="header-left">
        <a-button class="menu-trigger" type="text" @click="toggleMenu">
          <component :is="menuIcon" />
        </a-button>
        <div class="brand">
          <div class="brand-logo" aria-hidden="true">
            <SiteLogoMedia :size="20" />
          </div>
          <div>
            <div class="brand-name">{{ brandTitle }}</div>
            <!-- <div class="brand-subtle">Admin Console</div> -->
          </div>
        </div>
      </div>
      <div class="header-actions">
        <div v-if="!isMobile" class="header-search-wrapper">
          <a-auto-complete
            v-model:value="searchValue"
            class="header-search"
            placeholder="搜索功能..."
            :options="searchOptions"
            @select="handleSelect"
            @search="handleSearch"
          >
              <template #default>
                <a-input @pressEnter="handleEnterSearch">
                  <template #prefix>
                    <SearchOutlined />
                  </template>
                </a-input>
              </template>
          </a-auto-complete>
        </div>
        <a-space :size="12">
          <a-tag color="blue">Admin</a-tag>
          <a-tooltip title="消息中心">
            <a-badge :count="5" size="small">
              <a-button type="text" shape="circle">
                <BellOutlined />
              </a-button>
            </a-badge>
          </a-tooltip>
            <a-tooltip title="帮助中心">
              <a-button type="text" shape="circle" @click="goHelp">
                <QuestionCircleOutlined />
              </a-button>
            </a-tooltip>
          <a-dropdown>
            <div class="user-info">
              <a-avatar size="small" :src="adminAvatar" style="background-color: #722ed1">
                <UserOutlined />
              </a-avatar>
              <span class="user-name">{{ adminName }}</span>
            </div>
            <template #overlay>
              <a-menu>
                <a-menu-item @click="goToProfile">个人资料</a-menu-item>
                <a-menu-item @click="logout">退出登录</a-menu-item>
              </a-menu>
            </template>
          </a-dropdown>
        </a-space>
      </div>
    </a-layout-header>

    <a-layout>
      <a-layout-sider
        v-if="!isMobile"
        class="sider"
        :collapsed="collapsed"
        collapsible
        :trigger="null"
        width="230"
        @collapse="collapsed = $event"
      >
        <a-menu mode="inline" :selectedKeys="[selectedKey]" @click="onMenu">
          <AdminMenuNode v-for="node in visibleMenuTree" :key="node.key" :node="node" />
        </a-menu>
      </a-layout-sider>
      <a-drawer
        v-model:open="drawerOpen"
        placement="left"
        width="230"
        :body-style="{ padding: 0 }"
        :mask-style="{ backgroundColor: 'rgba(0, 0, 0, 0.45)' }"
        :wrap-style="{ position: 'absolute' }"
        :z-index="1001"
        class="mobile-drawer"
      >
        <div class="drawer-content">
          <a-menu mode="inline" :selectedKeys="[selectedKey]" @click="onMenu" class="drawer-menu">
            <AdminMenuNode v-for="node in visibleMenuTree" :key="node.key" :node="node" variant="drawer" />
          </a-menu>
        </div>
      </a-drawer>

      <a-layout>
        <a-layout-content class="content">
          <a-breadcrumb style="margin-bottom: 12px">
            <a-breadcrumb-item>管理端</a-breadcrumb-item>
            <a-breadcrumb-item>{{ breadcrumb }}</a-breadcrumb-item>
          </a-breadcrumb>
          <router-view />
        </a-layout-content>
      </a-layout>
    </a-layout>
  </a-layout>
  </a-config-provider>
</template>

<script setup>
import { computed, ref, onMounted, onUnmounted, nextTick } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useAdminAuthStore } from "@/stores/adminAuth";
import { useSiteStore } from "@/stores/site";
import SiteLogoMedia from "@/components/brand/SiteLogoMedia.vue";
import AdminMenuNode from "@/components/admin/AdminMenuNode.vue";
import { Grid, theme } from "ant-design-vue";
import {
  BellOutlined,
  QuestionCircleOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  UserOutlined,
  SearchOutlined,
  DashboardOutlined,
  CloudServerOutlined,
  ShoppingCartOutlined,
  WalletOutlined,
  TeamOutlined,
  AppstoreOutlined,
  FileTextOutlined,
  LayoutOutlined,
  CloudUploadOutlined,
  ApiOutlined,
  MailOutlined,
  KeyOutlined,
  LinkOutlined,
  SettingOutlined,
  CustomerServiceOutlined,
  CreditCardOutlined,
  DollarOutlined,
  ToolOutlined,
  MenuOutlined,
  SafetyCertificateOutlined,
  FileSearchOutlined,
  BugOutlined,
  ClockCircleOutlined
} from "@ant-design/icons-vue";

const route = useRoute();
const router = useRouter();
const admin = useAdminAuthStore();
const site = useSiteStore();
const screens = Grid.useBreakpoint();

const collapsed = ref(false);
const drawerOpen = ref(false);
const searchValue = ref('');

const grantedPerms = computed(() => admin.profile?.permissions || []);
const hasPermission = (required) => {
  const requiredPerm = String(required || "").trim();
  if (!requiredPerm) return true;

  const perms = grantedPerms.value;
  if (!Array.isArray(perms) || perms.length === 0) return false;

  for (const p of perms) {
    if (p === "*" || p === requiredPerm) return true;
    if (typeof p === "string" && p.endsWith("*")) {
      const prefix = p.slice(0, -1);
      if (requiredPerm.startsWith(prefix)) return true;
    }
  }
  return false;
};

const hasAnyPermission = (requirements) => {
  if (!requirements || requirements.length === 0) return true;
  return requirements.some((r) => hasPermission(r));
};

const menuTree = [
  { key: "/admin/console", label: "总览", icon: DashboardOutlined, requireAny: ["dashboard.overview"] },
  {
    key: "group-business",
    label: "业务与订单",
    icon: ShoppingCartOutlined,
    children: [
      { key: "/admin/orders", label: "订单审核", icon: ShoppingCartOutlined, requireAny: ["order.list"] },
      { key: "/admin/wallet/orders", label: "钱包订单", icon: WalletOutlined, requireAny: ["wallet_order.list"] },
      { key: "/admin/vps", label: "VPS 管理", icon: CloudServerOutlined, requireAny: ["vps.list"] },
      { key: "/admin/probes", label: "探针监控", icon: ApiOutlined, requireAny: ["probe.list"] },
      {
        key: "/admin/catalog",
        label: "售卖配置",
        icon: AppstoreOutlined,
        requireAny: ["regions.list", "plan_group.list", "line.list", "packages.list", "system_image.list", "billing_cycle.list"]
      }
    ]
  },
  {
    key: "group-support",
    label: "用户与支持",
    icon: TeamOutlined,
    children: [
      { key: "/admin/users", label: "用户管理", icon: TeamOutlined, requireAny: ["user.list"] },
      { key: "/admin/tickets", label: "工单管理", icon: CustomerServiceOutlined, requireAny: ["tickets.list"] }
    ]
  },
  {
    key: "group-cms",
    label: "CMS 内容",
    icon: FileTextOutlined,
    children: [
      { key: "/admin/cms/categories", label: "分类管理", icon: AppstoreOutlined, requireAny: ["cms_category.list"] },
      { key: "/admin/cms/posts", label: "内容管理", icon: FileTextOutlined, requireAny: ["cms_post.list"] },
      { key: "/admin/cms/blocks", label: "页面模块", icon: LayoutOutlined, requireAny: ["cms_block.list"] },
      { key: "/admin/cms/nav-items", label: "主页顶栏", icon: MenuOutlined, requireAny: ["settings.view"] },
      { key: "/admin/cms/uploads", label: "资源上传", icon: CloudUploadOutlined, requireAny: ["upload.list"] }
    ]
  },
  {
    key: "group-system",
    label: "系统设置",
    icon: SettingOutlined,
    children: [
      {
        key: "group-system-base",
        label: "基础设置",
        icon: SettingOutlined,
        children: [
          { key: "/admin/settings/site", label: "站点设置", icon: SettingOutlined, requireAny: ["settings.view"] },
          { key: "/admin/audit", label: "审计日志", icon: SettingOutlined, requireAny: ["audit_log.view"] },
          { key: "/admin/debug", label: "调试中心", icon: BugOutlined, requireAny: ["debug.view", "debug.list"] },
          { key: "/admin/scheduled-tasks", label: "计划任务", icon: ClockCircleOutlined, requireAny: ["scheduled_tasks.list"] }
        ]
      },
      {
        key: "group-system-security",
        label: "账号与权限",
        icon: KeyOutlined,
        children: [
          { key: "/admin/admins", label: "管理员列表", icon: UserOutlined, requireAny: ["admin.list"] },
          { key: "/admin/permission-groups", label: "权限组", icon: KeyOutlined, requireAny: ["permission_group.list"] }
        ]
      },
      {
        key: "group-system-realname",
        label: "实名认证",
        icon: SafetyCertificateOutlined,
        children: [
          { key: "/admin/realname/providers", label: "供应商列表", icon: AppstoreOutlined, requireAny: ["realname.list"] },
          { key: "/admin/realname/config", label: "认证配置", icon: SafetyCertificateOutlined, requireAny: ["realname.view"] },
          { key: "/admin/realname/records", label: "认证记录", icon: FileSearchOutlined, requireAny: ["realname.list"] }
        ]
      },
      {
        key: "group-system-integration",
        label: "集成与通知",
        icon: ApiOutlined,
        children: [
          { key: "/admin/settings/email", label: "邮件与模板", icon: MailOutlined, requireAny: ["smtp.view", "email_template.list"] },
          { key: "/admin/settings/payments", label: "支付设置", icon: CreditCardOutlined, requireAny: ["payment.list"] },
          { key: "/admin/settings/pricing", label: "价格与退款", icon: DollarOutlined, requireAny: ["settings.view"] },
          { key: "/admin/settings/lifecycle", label: "Lifecycle", icon: ClockCircleOutlined, requireAny: ["settings.view"] },
          { key: "/admin/settings/plugins", label: "插件管理", icon: ToolOutlined, requireAny: ["plugin.list"] },
          { key: "/admin/settings/apikey", label: "API Keys", icon: KeyOutlined, requireAny: ["api_key.list"] },
          { key: "/admin/settings/webhook", label: "Webhook", icon: LinkOutlined, requireAny: ["robot.view"] },
          { key: "/admin/automation", label: "自动化对接", icon: ApiOutlined, requireAny: ["automation.view"] }
        ]
      }
    ]
  },
  { key: "/admin/profile", label: "个人资料", icon: UserOutlined },
  { key: "/admin/settings/auth", label: "注册与登录", icon: SafetyCertificateOutlined, requireAny: ["settings.view"] }
];

const filterMenuTree = (nodes) => {
  const out = [];
  for (const node of nodes || []) {
    const children = Array.isArray(node.children) ? filterMenuTree(node.children) : [];
    const isLeaf = !node.children || node.children.length === 0;
    const allowed = hasAnyPermission(node.requireAny);

    if (isLeaf) {
      if (allowed) out.push(node);
      continue;
    }
    if (children.length > 0) out.push({ ...node, children });
  }
  return out;
};

const visibleMenuTree = computed(() => filterMenuTree(menuTree));

const flattenMenuLeaves = (nodes) => {
  const out = [];
  for (const node of nodes || []) {
    if (node.children && node.children.length) {
      out.push(...flattenMenuLeaves(node.children));
      continue;
    }
    if (typeof node.key === "string" && node.key.startsWith("/")) out.push(node);
  }
  return out;
};

const searchOptions = computed(() => {
  const keyword = searchValue.value.trim().toLowerCase();
  const leaves = flattenMenuLeaves(visibleMenuTree.value);
  const filtered = keyword ? leaves.filter((item) => item.label.toLowerCase().includes(keyword)) : leaves;
  return filtered.map((item) => ({ value: item.key, label: item.label }));
});

const isMobile = computed(() => screens.value?.lg === false);
const menuIcon = computed(() => {
  if (isMobile.value) return MenuUnfoldOutlined;
  return collapsed.value ? MenuUnfoldOutlined : MenuFoldOutlined;
});

const selectedKey = computed(() => {
  if (route.path.startsWith("/admin/settings")) return route.path;
  if (route.path.startsWith("/admin/cms")) return route.path;
  if (route.path.startsWith("/admin/tickets")) return "/admin/tickets";
  if (route.path.startsWith("/admin/probes")) return "/admin/probes";
  if (route.path.startsWith("/admin/audit")) return "/admin/audit";
  if (route.path.startsWith("/admin/debug")) return "/admin/debug";
  if (route.path.startsWith("/admin/console")) return "/admin/console";
  if (route.path === "/admin") return "/admin/console";
  return route.path;
});

const labelMap = computed(() => {
  const out = {};
  for (const node of flattenMenuLeaves(menuTree)) {
    out[node.key] = node.label;
  }
  return out;
});

const breadcrumb = computed(() => labelMap.value[selectedKey.value] || "总览");
const brandTitle = computed(() => site.siteName || "运营管理后台");
const adminName = computed(() => admin.profile?.username || "管理员");
const adminAvatar = computed(() => {
  const qq = admin.profile?.qq;
  if (!qq) return "";
  return `https://q1.qlogo.cn/g?b=qq&nk=${qq}&s=100`;
});

const onMenu = async ({ key }) => {
  const target = String(key);
  if (target === route.path) {
    if (isMobile.value) {
      drawerOpen.value = false;
    }
    return;
  }
  if (isMobile.value) {
    drawerOpen.value = false;
    await nextTick();
  }
  router.push(target);
};

const toggleMenu = () => {
  if (isMobile.value) {
    drawerOpen.value = true;
  } else {
    collapsed.value = !collapsed.value;
  }
};

const logout = () => {
  admin.logout();
  router.replace("/admin/login");
};

const goToProfile = () => {
  router.push("/admin/profile");
};

const handleSearch = () => {
  // AutoComplete only updates options via v-model
};

const handleEnterSearch = () => {
  const keyword = searchValue.value.trim();
  if (!keyword) {
    return;
  }
  const options = searchOptions.value;
  const target =
    options.find((item) => item.label.toLowerCase() === keyword.toLowerCase()) || options[0];
  if (target?.value) {
    router.push(String(target.value));
    searchValue.value = "";
  }
};

const handleSelect = (value) => {
  if (value) {
    router.push(String(value));
    searchValue.value = "";
  }
};

const goHelp = () => {
  router.push("/help");
};

onMounted(() => {
  document.body.classList.add("admin-theme");
  if (!admin.profile && admin.token) {
    admin.fetchProfile();
  }
  site.fetchSettings();
});

onUnmounted(() => {
  document.body.classList.remove("admin-theme");
});
</script>

<style scoped>
@import url('https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap');

.layout {
  min-height: 100vh;
  height: 100vh;
  overflow: hidden;
}

.admin-layout > :deep(.ant-layout) {
  height: calc(100vh - 64px);
  overflow: hidden;
}

/* ========== Header 专业商业风格 ========== */
.header {
  background: linear-gradient(135deg, #1a1d23 0%, #2b2f36 50%, #1a1d23 100%);
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15), 0 1px 2px rgba(0, 0, 0, 0.1);
  color: #e6e9ef;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 0 20px;
  height: 64px;
  position: sticky;
  top: 0;
  z-index: 1000;
  backdrop-filter: blur(10px);
}

.header-left {
  display: flex;
  align-items: center;
  gap: 16px;
}

.menu-trigger {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  color: #e6e9ef;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.menu-trigger:hover {
  background: rgba(255, 255, 255, 0.1);
  transform: scale(1.05);
}

.brand {
  display: flex;
  align-items: center;
  gap: 12px;
}

.brand-logo {
  width: 38px;
  height: 38px;
  border-radius: 10px;
  background: linear-gradient(135deg, #1677ff 0%, #4096ff 50%, #6ea4ff 100%);
  color: #fff;
  font-weight: 700;
  font-size: 15px;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 4px 12px rgba(22, 119, 255, 0.4);
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.brand-logo:hover {
  transform: translateY(-2px);
  box-shadow: 0 6px 16px rgba(22, 119, 255, 0.5);
}

.brand-name {
  font-weight: 600;
  font-size: 16px;
  letter-spacing: 0.3px;
  background: linear-gradient(135deg, #ffffff 0%, #e6e9ef 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.brand-subtle {
  font-size: 12px;
  color: #9ca3af;
  margin-top: 2px;
}

/* ========== 搜索框 ========== */
.header-search-wrapper {
  position: relative;
  width: 280px;
  transition: all 0.2s ease;
}

.header-search-wrapper:hover {
  width: 300px;
}

.header-search {
  width: 100%;
  height: 36px;
  border-radius: 6px;
  border: 1px solid rgba(255, 255, 255, 0.12);
  background: rgba(255, 255, 255, 0.04);
  transition: all 0.2s ease;
}

/* 搜索框容器样式 - 针对.ant-input-affix-wrapper */
.header-search-wrapper :deep(.ant-input-affix-wrapper) {
  background: rgba(255, 255, 255, 0.04) !important;
  border: 1px solid rgba(255, 255, 255, 0.12) !important;
  border-radius: 6px !important;
  height: 36px;
  padding: 0;
  transition: all 0.2s ease;
  box-shadow: none !important;
}

.header-search-wrapper :deep(.ant-input-affix-wrapper:hover) {
  background: rgba(255, 255, 255, 0.1) !important;
  border-color: rgba(255, 255, 255, 0.25) !important;
}

.header-search-wrapper :deep(.ant-input-affix-wrapper:focus),
.header-search-wrapper :deep(.ant-input-affix-wrapper-focused) {
  background: rgba(255, 255, 255, 0.08) !important;
  border-color: rgba(22, 119, 255, 0.5) !important;
  box-shadow: 0 0 0 3px rgba(22, 119, 255, 0.1) !important;
  outline: none;
}

/* 内部输入框样式 */
.header-search-wrapper :deep(.ant-input) {
  background: transparent !important;
  border: none !important;
  color: #e6e9ef !important;
  font-size: 14px;
  padding: 0 12px;
}

.header-search-wrapper :deep(.ant-input::placeholder) {
  color: rgba(230, 233, 239, 0.4) !important;
}

/* 搜索框前缀图标 */
.header-search-wrapper :deep(.ant-input-prefix) {
  margin: 0;
  padding-left: 10px;
  color: rgba(255, 255, 255, 0.45);
  display: flex;
  align-items: center;
}

.header-search-wrapper :deep(.ant-input-prefix .anticon) {
  color: rgba(255, 255, 255, 0.45);
  font-size: 14px;
}

/* 清除按钮 */
.header-search-wrapper :deep(.ant-input-clear-icon) {
  color: rgba(255, 255, 255, 0.35);
  font-size: 12px;
}

.header-search-wrapper :deep(.ant-input-clear-icon:hover) {
  color: rgba(255, 255, 255, 0.65);
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.sider {
  background: #0f1115;
  height: 100%;
  overflow: auto;
  /* 悬浮滚动条 */
  overflow-y: overlay;
  overflow-x: overlay;
}

.sider :deep(.ant-menu) {
  background: #0f1115;
  color: rgba(255, 255, 255, 0.7);
}

.sider :deep(.ant-menu-item),
.sider :deep(.ant-menu-submenu-title) {
  color: rgba(255, 255, 255, 0.7);
}

.sider :deep(.ant-menu-item:hover),
.sider :deep(.ant-menu-submenu-title:hover) {
  color: #fff;
  background: rgba(255, 255, 255, 0.08);
}

.sider :deep(.ant-menu-item-selected) {
  background: #1f2937;
  color: #fff;
}

.sider :deep(.ant-menu-item-selected .anticon) {
  color: #4096ff;
}

.sider :deep(.ant-menu-submenu-arrow) {
  color: rgba(255, 255, 255, 0.45);
}

/* ========== Sider 悬浮滚动条样式 ========== */
.sider::-webkit-scrollbar {
  width: 6px;
  height: 6px;
}

.sider::-webkit-scrollbar-track {
  background: transparent;
}

.sider::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.2);
  border-radius: 3px;
  border: 2px solid transparent;
  background-clip: padding-box;
  transition: background 0.2s;
}

.sider::-webkit-scrollbar-thumb:hover {
  background: rgba(255, 255, 255, 0.35);
  background-clip: padding-box;
}

/* ========== Content 悬浮滚动条样式 ========== */
.content {
  overflow: auto;
  overflow-y: auto;
  overflow-x: hidden;
}

.content::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

.content::-webkit-scrollbar-track {
  background: transparent;
}

.content::-webkit-scrollbar-thumb {
  background: rgba(0, 0, 0, 0.2);
  border-radius: 4px;
  border: 2px solid transparent;
  background-clip: padding-box;
  transition: background 0.2s;
}

.content::-webkit-scrollbar-thumb:hover {
  background: rgba(0, 0, 0, 0.35);
  background-clip: padding-box;
}

/* ========== 用户信息 ========== */
.user-info {
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  padding: 6px 12px;
  border-radius: 6px;
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
}

.user-info:hover {
  background: rgba(255, 255, 255, 0.1);
}

.user-name {
  font-weight: 500;
  font-size: 14px;
  color: #e6e9ef;
}

/* ========== Header 组件样式 ========== */
.header :deep(.ant-btn) {
  border: none;
  box-shadow: none;
}

.header :deep(.ant-btn:hover) {
  color: #ffffff;
  background: rgba(255, 255, 255, 0.1);
  transform: scale(1.05);
}

.header :deep(.ant-btn-text) {
  color: #e6e9ef;
}

.header :deep(.anticon) {
  font-size: 16px;
}

.header :deep(.ant-badge-count) {
  background: #ff4d4f;
  box-shadow: 0 2px 4px rgba(255, 77, 79, 0.3);
}

.header :deep(.ant-tag) {
  background: rgba(22, 119, 255, 0.15);
  border: 1px solid rgba(22, 119, 255, 0.3);
  color: #6ea4ff;
  font-weight: 500;
  padding: 2px 10px;
  border-radius: 6px;
  font-size: 12px;
}

.header :deep(.ant-avatar) {
  border: 2px solid rgba(22, 119, 255, 0.3);
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.header :deep(.ant-avatar:hover) {
  border-color: rgba(22, 119, 255, 0.6);
}

/* ========== 响应式 ========== */
@media (max-width: 768px) {
  .header {
    padding: 0 16px;
    height: 52px;
  }

  .header-search-wrapper {
    display: none;
  }

  .brand-name {
    font-size: 15px;
  }

  .brand-logo {
    width: 28px;
    height: 28px;
    font-size: 12px;
  }

  .menu-trigger {
    width: 32px;
    height: 32px;
  }
}

/* ========== Drawer 移动端菜单 - 深色风格 ========== */
.mobile-drawer :deep(.ant-drawer-body) {
  padding: 0;
  background: #0f1115;
}

.mobile-drawer :deep(.ant-drawer-content) {
  background: #0f1115;
}

.mobile-drawer :deep(.ant-drawer-header) {
  display: none;
}

.drawer-content {
  height: 100%;
  overflow-y: auto;
  background: #0f1115;
}

.drawer-menu {
  border: none;
  background: transparent;
  padding: 12px 0;
}

.drawer-menu :deep(.ant-menu-item) {
  margin: 2px 0;
  padding: 12px 16px;
  height: auto;
  display: flex;
  align-items: center;
  gap: 12px;
  color: rgba(255, 255, 255, 0.65);
  font-size: 14px;
  border-radius: 8px;
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
}

.drawer-menu :deep(.ant-menu-item:hover) {
  background: rgba(255, 255, 255, 0.08);
  color: #fff;
}

.drawer-menu :deep(.ant-menu-item-selected) {
  background: #1f2937;
  color: #fff;
  font-weight: 500;
  border-left: none;
}

.drawer-menu :deep(.ant-menu-item .anticon) {
  font-size: 16px;
  color: inherit;
}

.drawer-menu :deep(.ant-menu-item-selected .anticon) {
  color: #4096ff;
}

.drawer-menu :deep(.ant-menu-item span) {
  font-size: 14px;
}

.drawer-menu :deep(.ant-menu-submenu-title) {
  color: rgba(226, 232, 240, 0.75);
  border-radius: 8px;
  margin: 2px 0;
  padding: 12px 16px;
  height: auto;
  line-height: 1.5;
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
}

.drawer-menu :deep(.ant-menu-submenu-title:hover) {
  color: #fff;
  background: rgba(255, 255, 255, 0.08);
}

.drawer-menu :deep(.ant-menu-submenu-arrow) {
  color: rgba(226, 232, 240, 0.55);
}

.drawer-menu :deep(.ant-menu-submenu-open > .ant-menu-submenu-title) {
  color: #fff;
}

.drawer-menu :deep(.ant-menu) {
  background: transparent;
}

/* ========== Drawer 滚动条 - 深色风格 ========== */
.drawer-content::-webkit-scrollbar {
  width: 4px;
}

.drawer-content::-webkit-scrollbar-track {
  background: rgba(255, 255, 255, 0.05);
}

.drawer-content::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.15);
  border-radius: 2px;
}

.drawer-content::-webkit-scrollbar-thumb:hover {
  background: rgba(255, 255, 255, 0.25);
}

/* ========== Breadcrumb 面包屑样式 ========== */
.content :deep(.ant-breadcrumb) {
  padding: 12px 16px 0 !important;
  background: linear-gradient(135deg, #fafbfc 0%, #f5f7fa 100%);
  border-radius: 8px;
  border: 1px solid #e8ecf1;
  margin-bottom: 0 !important;
}

.content :deep(.ant-breadcrumb-link) {
  color: var(--text2);
  font-weight: 500;
  font-size: 13px;
  transition: color 0.2s ease;
}

.content :deep(.ant-breadcrumb-link:hover) {
  color: var(--primary);
}

.content :deep(.ant-breadcrumb-separator) {
  color: var(--text3);
}
</style>
