<template>
  <a-config-provider
    :theme="{
      algorithm: isDarkMode ? theme.darkAlgorithm : theme.defaultAlgorithm,
      token: isDarkMode ? {
        colorPrimary: '#3b82f6',
        colorBgContainer: '#1e2433',
        colorBgLayout: '#0f1419',
        colorBorder: '#2d3748',
        colorText: '#f1f5f9',
        colorTextSecondary: '#94a3b8'
      } : {
        colorPrimary: '#0066FF'
      }
    }"
  >
    <div class="console-layout">
      <!-- Sidebar -->
      <aside class="sidebar" :class="{ 'sidebar-collapsed': collapsed }">
        <div class="sidebar-header">
          <div class="brand-wrapper">
            <div class="brand-logo">
              <SiteLogoMedia :size="collapsed ? 24 : 28" />
            </div>
            <transition name="fade-slide">
              <div v-if="!collapsed" class="brand-info">
                <h1 class="brand-title">{{ brandName }}</h1>
                <p class="brand-subtitle">Cloud Console</p>
              </div>
            </transition>
          </div>
        </div>

        <nav class="sidebar-nav">
          <div v-for="group in menuGroups" :key="group.title" class="menu-group">
            <div v-if="!collapsed && group.title" class="menu-group-title">
              <span>{{ group.title }}</span>
            </div>
            <div class="menu-items">
              <router-link
                v-for="item in group.items"
                :key="item.key"
                :to="item.key"
                class="menu-item"
                :class="{ 'menu-item-active': isActive(item.key) }"
              >
                <component :is="item.icon" class="menu-icon" />
                <transition name="fade-slide">
                  <span v-if="!collapsed" class="menu-label">{{ item.label }}</span>
                </transition>
                <transition name="fade-slide">
                  <span v-if="!collapsed && item.badge" class="menu-badge">{{ item.badge }}</span>
                </transition>
              </router-link>
            </div>
          </div>
        </nav>

        <div class="sidebar-footer">
          <div class="sidebar-user" @click="goProfile">
            <a-avatar :size="collapsed ? 32 : 40" :src="userAvatar">
              <UserOutlined />
            </a-avatar>
            <transition name="fade-slide">
              <div v-if="!collapsed" class="user-details">
                <div class="user-name">{{ displayName }}</div>
                <div class="user-role">用户</div>
              </div>
            </transition>
          </div>
        </div>
      </aside>

      <!-- Main Content -->
      <div class="main-wrapper">
        <!-- Header -->
        <header class="top-header">
          <div class="header-left">
            <button class="collapse-btn" @click="collapsed = !collapsed">
              <MenuUnfoldOutlined v-if="collapsed" />
              <MenuFoldOutlined v-else />
            </button>
            <div class="header-search">
              <SearchOutlined class="search-icon" />
              <input
                v-model="searchValue"
                type="text"
                placeholder="搜索功能、页面..."
                @focus="showSearchResults = true"
                @blur="handleSearchBlur"
              >
              <kbd v-if="!searchValue" class="search-hint">⌘K</kbd>
              <transition name="dropdown">
                <div v-if="showSearchResults && (searchValue || searchOptions.length)" class="search-results">
                  <div v-if="!searchOptions.length" class="search-empty">未找到相关内容</div>
                  <a v-for="opt in searchOptions" :key="opt.value" href="#" class="search-item" @click.prevent="navigateTo(opt.value)">
                    <component :is="getIconForRoute(opt.value)" class="search-item-icon" />
                    <div class="search-item-content">
                      <div class="search-item-label">{{ opt.label }}</div>
                      <div class="search-item-path">{{ opt.value }}</div>
                    </div>
                  </a>
                </div>
              </transition>
            </div>
          </div>

          <div class="header-right">
            <div class="header-actions">
              <!-- Environment Badge -->
              <div class="env-badge">
                <span class="env-dot"></span>
                <span>Production</span>
              </div>

              <!-- Theme Toggle -->
              <button
                class="icon-btn theme-toggle-btn"
                @click="toggleTheme"
                :title="isDarkMode ? '切换到亮色模式' : '切换到暗色模式'"
              >
                <BulbOutlined />
              </button>

              <!-- Notifications -->
              <a-dropdown :trigger="['click']" placement="bottomRight">
                <button class="icon-btn" @click="fetchNotifications">
                  <a-badge :count="unreadCount" :offset="[-4, 4]">
                    <BellOutlined />
                  </a-badge>
                </button>
                <template #overlay>
                  <div class="notif-dropdown">
                    <div class="notif-header">
                      <h3>消息中心</h3>
                      <a-button type="link" size="small" @click.stop="markAllRead">全部已读</a-button>
                    </div>
                    <a-tabs v-model:activeKey="notifTab" size="small" @change="fetchNotifications">
                      <a-tab-pane key="unread" tab="未读" />
                      <a-tab-pane key="read" tab="已读" />
                      <a-tab-pane key="all" tab="全部" />
                    </a-tabs>
                    <div class="notif-list">
                      <a-spin :spinning="notifLoading">
                        <div v-if="!notifications.length" class="notif-empty">暂无消息</div>
                        <div v-for="item in notifications" :key="item.id" class="notif-item" @click="markRead(item)">
                          <div class="notif-icon">
                            <NotificationOutlined />
                          </div>
                          <div class="notif-content">
                            <div class="notif-title-row">
                              <span class="notif-item-title">{{ item.title || item.type }}</span>
                              <span class="notif-time">{{ formatTime(item.created_at) }}</span>
                            </div>
                            <div class="notif-text">{{ item.content }}</div>
                          </div>
                        </div>
                      </a-spin>
                    </div>
                  </div>
                </template>
              </a-dropdown>

              <!-- Help -->
              <a-tooltip title="帮助中心">
                <button class="icon-btn" @click="goHelp">
                  <QuestionCircleOutlined />
                </button>
              </a-tooltip>

              <!-- Divider -->
              <div class="header-divider"></div>

              <!-- User Menu -->
              <a-dropdown>
                <button class="user-btn">
                  <a-avatar :size="32" :src="userAvatar">
                    <UserOutlined />
                  </a-avatar>
                  <span class="user-btn-name">{{ displayName }}</span>
                  <DownOutlined class="dropdown-arrow" />
                </button>
                <template #overlay>
                  <a-menu>
                    <a-menu-item @click="goProfile">
                      <UserOutlined />
                      <span>个人信息</span>
                    </a-menu-item>
                    <a-menu-divider />
                    <a-menu-item @click="logout">
                      <LogoutOutlined />
                      <span>退出登录</span>
                    </a-menu-item>
                  </a-menu>
                </template>
              </a-dropdown>
            </div>
          </div>
        </header>

        <!-- Page Content -->
        <main class="main-content">
          <div class="content-container">
            <div class="page-breadcrumb">
              <a-breadcrumb>
                <a-breadcrumb-item>
                  <HomeOutlined />
                  <span>控制台</span>
                </a-breadcrumb-item>
                <a-breadcrumb-item>{{ breadcrumb }}</a-breadcrumb-item>
              </a-breadcrumb>
            </div>
            <router-view v-slot="{ Component }">
              <transition name="page" mode="out-in">
                <component :is="Component" />
              </transition>
            </router-view>
          </div>
        </main>
      </div>

      <!-- Mobile Drawer -->
      <a-drawer
        v-model:open="mobileDrawerOpen"
        placement="left"
        :width="280"
        :body-style="{ padding: 0 }"
        :mask-style="{ backgroundColor: 'rgba(0, 0, 0, 0.5)' }"
        class="mobile-drawer"
      >
        <div class="mobile-drawer-content">
          <div class="mobile-drawer-header">
            <div class="brand-logo">
              <SiteLogoMedia :size="32" />
            </div>
            <h2>{{ brandName }}</h2>
          </div>
          <nav class="mobile-nav">
            <div
              v-for="group in menuGroups"
              :key="group.title"
              class="mobile-menu-group"
            >
              <div v-if="group.title" class="mobile-group-title">{{ group.title }}</div>
              <router-link
                v-for="item in group.items"
                :key="item.key"
                :to="item.key"
                class="mobile-menu-item"
                :class="{ 'mobile-menu-item-active': isActive(item.key) }"
                @click="mobileDrawerOpen = false"
              >
                <component :is="item.icon" class="mobile-menu-icon" />
                <span>{{ item.label }}</span>
                <span v-if="item.badge" class="mobile-menu-badge">{{ item.badge }}</span>
              </router-link>
            </div>
          </nav>
        </div>
      </a-drawer>

      <!-- Mobile Header -->
      <div class="mobile-header">
        <button class="mobile-menu-btn" @click="mobileDrawerOpen = true">
          <MenuOutlined />
        </button>
        <div class="mobile-brand">{{ brandName }}</div>
        <a-dropdown>
          <a-avatar :size="32" :src="userAvatar">
            <UserOutlined />
          </a-avatar>
          <template #overlay>
            <a-menu>
              <a-menu-item @click="goProfile">
                <UserOutlined />
                <span>个人信息</span>
              </a-menu-item>
              <a-menu-item @click="logout">
                <LogoutOutlined />
                <span>退出登录</span>
              </a-menu-item>
            </a-menu>
          </template>
        </a-dropdown>
      </div>
    </div>
  </a-config-provider>
</template>

<script setup>
import { computed, ref, onMounted, onUnmounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useAuthStore } from "@/stores/auth";
import { useSiteStore } from "@/stores/site";
import { useCartStore } from "@/stores/cart";
import { useAppStore } from "@/stores/app";
import SiteLogoMedia from "@/components/brand/SiteLogoMedia.vue";
import { listNotifications, getUnreadCount, markNotificationRead, markAllNotificationsRead } from "@/services/user";
import { Grid, theme } from "ant-design-vue";
import {
  BellOutlined,
  QuestionCircleOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  MenuOutlined,
  UserOutlined,
  SearchOutlined,
  BulbOutlined,
  DashboardOutlined,
  CloudServerOutlined,
  ShoppingCartOutlined,
  ApiOutlined,
  FileTextOutlined,
  CustomerServiceOutlined,
  SafetyCertificateOutlined,
  CreditCardOutlined,
  WalletOutlined,
  DollarOutlined,
  HomeOutlined,
  LogoutOutlined,
  DownOutlined,
  NotificationOutlined,
  SettingOutlined,
  ThunderboltOutlined,
  AppstoreOutlined,
  TeamOutlined
} from "@ant-design/icons-vue";

const route = useRoute();
const router = useRouter();
const auth = useAuthStore();
const site = useSiteStore();
const cart = useCartStore();
const app = useAppStore();
const screens = Grid.useBreakpoint();

const collapsed = ref(false);
const mobileDrawerOpen = ref(false);
const searchValue = ref('');
const showSearchResults = ref(false);
const notifTab = ref("unread");
const notifications = ref([]);
const notifLoading = ref(false);
const unreadCount = ref(0);

const isMobile = computed(() => !screens.value?.lg);

const menuGroups = computed(() => [
  {
    title: '',
    items: [
      { key: '/console', label: '总览', icon: DashboardOutlined },
    ]
  },
  {
    title: '云服务',
    items: [
      { key: '/console/vps', label: '云服务器', icon: CloudServerOutlined },
    ]
  },
  {
    title: '订单与财务',
    items: [
      { key: '/console/cart', label: '购物车', icon: ShoppingCartOutlined, badge: cartCount.value > 0 ? cartCount.value : null },
      { key: '/console/orders', label: '订单管理', icon: FileTextOutlined },
      { key: '/console/billing', label: '钱包充值', icon: WalletOutlined },
      { key: '/console/api-keys', label: 'API 管理', icon: ApiOutlined },
    ]
  },
  {
    title: '支持与设置',
    items: [
      { key: '/console/tickets', label: '工单中心', icon: CustomerServiceOutlined },
      { key: '/console/realname', label: '实名认证', icon: SafetyCertificateOutlined },
      { key: '/console/profile', label: '个人设置', icon: SettingOutlined },
    ]
  }
]);

const cartCount = computed(() => cart.items?.length || 0);

const isActive = (path) => {
  if (route.path.startsWith('/console/buy')) return path === '/console/vps';
  if (route.path.startsWith('/console/vps')) return path === '/console/vps';
  if (route.path.startsWith('/console/orders')) return path === '/console/orders';
  if (route.path.startsWith('/console/billing')) return path === '/console/billing';
  if (route.path.startsWith('/console/api-keys')) return path === '/console/api-keys';
  if (route.path.startsWith('/console/tickets')) return path === '/console/tickets';
  if (route.path.startsWith('/console/realname')) return path === '/console/realname';
  if (route.path.startsWith('/console/cart')) return path === '/console/cart';
  if (route.path.startsWith('/console/profile')) return path === '/console/profile';
  return path === '/console';
};

const labelMap = {
  "/console": "总览",
  "/console/vps": "云服务器",
  "/console/orders": "订单管理",
  "/console/billing": "钱包充值",
  "/console/api-keys": "API 管理",
  "/console/tickets": "工单中心",
  "/console/realname": "实名认证",
  "/console/cart": "购物车",
  "/console/profile": "个人设置"
};

const breadcrumb = computed(() => labelMap[selectedKey.value] || "总览");

const selectedKey = computed(() => {
  if (route.path.startsWith("/console/buy")) return "/console/vps";
  if (route.path.startsWith("/console/vps")) return "/console/vps";
  if (route.path.startsWith("/console/orders")) return "/console/orders";
  if (route.path.startsWith("/console/billing")) return "/console/billing";
  if (route.path.startsWith("/console/api-keys")) return "/console/api-keys";
  if (route.path.startsWith("/console/tickets")) return "/console/tickets";
  if (route.path.startsWith("/console/realname")) return "/console/realname";
  if (route.path.startsWith("/console/cart")) return "/console/cart";
  if (route.path.startsWith("/console/profile")) return "/console/profile";
  return "/console";
});

const displayName = computed(() => auth.profile?.username || "用户");
const brandName = computed(() => site.siteName || "控制台");
const userAvatar = computed(() => {
  const qq = auth.profile?.qq;
  if (!qq) return "";
  return `https://q1.qlogo.cn/g?b=qq&nk=${qq}&s=100`;
});

// Theme
const isDarkMode = computed(() => app.isDarkMode);

const toggleTheme = () => {
  app.toggleConsoleTheme();
};

const searchOptions = computed(() => {
  const keyword = searchValue.value.trim().toLowerCase();
  const allItems = menuGroups.value.flatMap(g => g.items);
  const filtered = keyword
    ? allItems.filter(item => item.label.toLowerCase().includes(keyword))
    : allItems;
  return filtered.map(item => ({ label: item.label, value: item.key }));
});

const getIconForRoute = (route) => {
  const item = menuGroups.value.flatMap(g => g.items).find(i => i.key === route);
  return item?.icon || DashboardOutlined;
};

const navigateTo = (path) => {
  router.push(path);
  searchValue.value = '';
  showSearchResults.value = false;
};

const handleSearchBlur = () => {
  setTimeout(() => {
    showSearchResults.value = false;
  }, 200);
};

const logout = () => {
  auth.logout();
  router.replace("/login");
};

const goProfile = () => router.push("/console/profile");
const goHelp = () => router.push("/help");

const fetchUnreadCount = async () => {
  const res = await getUnreadCount();
  unreadCount.value = res.data?.unread || 0;
};

const fetchNotifications = async () => {
  notifLoading.value = true;
  try {
    const status = notifTab.value === "all" ? undefined : notifTab.value;
    const res = await listNotifications({ status, limit: 20, offset: 0 });
    notifications.value = res.data?.items || [];
  } finally {
    notifLoading.value = false;
  }
};

const markRead = async (item) => {
  if (!item?.id) return;
  await markNotificationRead(item.id);
  await fetchUnreadCount();
  await fetchNotifications();
};

const markAllRead = async () => {
  await markAllNotificationsRead();
  await fetchUnreadCount();
  await fetchNotifications();
};

const formatTime = (value) => {
  if (!value) return "";
  const dt = new Date(value);
  if (Number.isNaN(dt.getTime())) return value;
  return dt.toLocaleString("zh-CN", { hour12: false });
};

// Keyboard shortcut for search
const handleKeydown = (e) => {
  if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
    e.preventDefault();
    document.querySelector('.header-search input')?.focus();
  }
  if (e.key === 'Escape') {
    showSearchResults.value = false;
  }
};

onMounted(() => {
  app.initConsoleTheme();
  if (!auth.profile && auth.token) {
    auth.fetchMe();
  }
  if (auth.token) {
    fetchUnreadCount();
  }
  site.fetchSettings();
  document.addEventListener('keydown', handleKeydown);
});

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown);
});
</script>

<style>
@import '@/styles/console-dark.css';
</style>

<style scoped>
/* ==================== Layout Base ==================== */
.console-layout {
  display: flex;
  min-height: 100vh;
  background: var(--bg-primary);
  position: relative;
}

/* ==================== Sidebar ==================== */
.sidebar {
  position: fixed;
  left: 0;
  top: 0;
  bottom: 0;
  width: 240px;
  background: linear-gradient(180deg, #f8fafc 0%, #f1f5f9 100%);
  display: flex;
  flex-direction: column;
  z-index: 100;
  transition: width 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  border-right: 1px solid rgba(0, 0, 0, 0.06);
}

.sidebar-collapsed {
  width: 72px;
}

.sidebar-header {
  padding: 18px 16px;
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
  position: relative;
  z-index: 1;
}

.brand-wrapper {
  display: flex;
  align-items: center;
  gap: 12px;
}

.brand-logo {
  flex-shrink: 0;
  width: 40px;
  height: 40px;
  background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  box-shadow: 0 4px 16px rgba(99, 102, 241, 0.3);
}

.brand-info {
  flex: 1;
  min-width: 0;
}

.brand-title {
  font-size: 16px;
  font-weight: 700;
  color: #1e293b;
  margin: 0;
  line-height: 1.2;
  letter-spacing: -0.02em;
}

.brand-subtitle {
  font-size: 10px;
  color: #64748b;
  margin: 2px 0 0 0;
  text-transform: uppercase;
  letter-spacing: 0.1em;
  font-weight: 600;
}

/* ==================== Sidebar Navigation ==================== */
.sidebar-nav {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 16px 12px;
  position: relative;
  z-index: 1;
}

.sidebar-nav::-webkit-scrollbar {
  width: 3px;
}

.sidebar-nav::-webkit-scrollbar-track {
  background: transparent;
}

.sidebar-nav::-webkit-scrollbar-thumb {
  background: rgba(99, 102, 241, 0.2);
  border-radius: 3px;
}

.sidebar-nav::-webkit-scrollbar-thumb:hover {
  background: rgba(99, 102, 241, 0.4);
}

.menu-group {
  margin-bottom: 20px;
}

.menu-group:last-child {
  margin-bottom: 0;
}

.menu-group-title {
  font-size: 10px;
  font-weight: 700;
  color: #94a3b8;
  text-transform: uppercase;
  letter-spacing: 0.1em;
  padding: 0 12px 8px;
}

.menu-items {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.menu-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border-radius: 10px;
  color: #64748b;
  text-decoration: none;
  transition: all 0.2s ease;
  position: relative;
}

.menu-item::before {
  content: '';
  position: absolute;
  left: 0;
  top: 50%;
  transform: translateY(-50%);
  width: 3px;
  height: 0;
  background: linear-gradient(180deg, #6366f1 0%, #8b5cf6 100%);
  border-radius: 0 2px 2px 0;
  transition: height 0.2s ease;
}

.menu-item:hover {
  background: rgba(99, 102, 241, 0.08);
  color: #6366f1;
}

.menu-item:hover::before {
  height: 18px;
}

.menu-item-active {
  background: linear-gradient(90deg, rgba(99, 102, 241, 0.12) 0%, rgba(99, 102, 241, 0.04) 100%);
  color: #6366f1;
  font-weight: 600;
}

.menu-item-active::before {
  height: 22px;
}

.menu-icon {
  font-size: 17px;
  flex-shrink: 0;
}

.menu-label {
  flex: 1;
  font-size: 13px;
  font-weight: 500;
  white-space: nowrap;
}

.menu-badge {
  background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
  color: #fff;
  font-size: 10px;
  font-weight: 700;
  padding: 2px 8px;
  border-radius: 10px;
  min-width: 18px;
  text-align: center;
  box-shadow: 0 2px 6px rgba(99, 102, 241, 0.25);
}

/* ==================== Sidebar Footer ==================== */
.sidebar-footer {
  padding: 12px;
  border-top: 1px solid rgba(0, 0, 0, 0.06);
  position: relative;
  z-index: 1;
}

.sidebar-user {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.sidebar-user:hover {
  background: rgba(99, 102, 241, 0.08);
}

.sidebar-user :deep(.ant-avatar) {
  border: 2px solid rgba(99, 102, 241, 0.15);
  transition: all 0.2s ease;
}

.sidebar-user:hover :deep(.ant-avatar) {
  border-color: rgba(99, 102, 241, 0.4);
}

.user-details {
  flex: 1;
  min-width: 0;
}

.user-name {
  font-size: 13px;
  font-weight: 600;
  color: #1e293b;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.user-role {
  font-size: 11px;
  color: #64748b;
  font-weight: 500;
}

/* ==================== Main Wrapper ==================== */
.main-wrapper {
  flex: 1;
  margin-left: 240px;
  display: flex;
  flex-direction: column;
  min-height: 100vh;
  transition: margin-left 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.sidebar-collapsed + .main-wrapper {
  margin-left: 72px;
}

/* ==================== Top Header ==================== */
.top-header {
  position: sticky;
  top: 0;
  z-index: 50;
  height: 64px;
  background: rgba(255, 255, 255, 0.9);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  border-bottom: 1px solid var(--border);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 16px;
}

.collapse-btn {
  width: 40px;
  height: 40px;
  border: none;
  background: var(--bg-secondary);
  border-radius: 10px;
  color: var(--text-secondary);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
  transition: all 0.2s ease;
}

.collapse-btn:hover {
  background: var(--bg-tertiary);
  color: var(--primary);
}

/* ==================== Search ==================== */
.header-search {
  position: relative;
  width: 320px;
}

.header-search input {
  width: 100%;
  height: 42px;
  padding: 0 44px 0 40px;
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: 12px;
  font-size: 14px;
  color: var(--text-primary);
  transition: all 0.2s ease;
}

.header-search input:focus {
  outline: none;
  border-color: var(--primary);
  box-shadow: 0 0 0 3px rgba(0, 102, 255, 0.1);
  background: #fff;
}

.header-search input::placeholder {
  color: var(--text-tertiary);
}

.search-icon {
  position: absolute;
  left: 12px;
  top: 50%;
  transform: translateY(-50%);
  color: var(--text-tertiary);
  font-size: 16px;
  pointer-events: none;
}

.search-hint {
  position: absolute;
  right: 12px;
  top: 50%;
  transform: translateY(-50%);
  font-size: 11px;
  font-weight: 600;
  color: var(--text-tertiary);
  background: var(--bg-tertiary);
  padding: 3px 8px;
  border-radius: 6px;
  font-family: inherit;
  pointer-events: none;
}

.search-results {
  position: absolute;
  top: calc(100% + 8px);
  left: 0;
  right: 0;
  background: #fff;
  border-radius: 12px;
  box-shadow: var(--shadow-lg);
  border: 1px solid var(--border);
  overflow: hidden;
  z-index: 100;
}

.search-empty {
  padding: 24px;
  text-align: center;
  color: var(--text-tertiary);
  font-size: 14px;
}

.search-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  text-decoration: none;
  transition: all 0.15s ease;
}

.search-item:hover {
  background: var(--bg-secondary);
}

.search-item-icon {
  font-size: 16px;
  color: var(--text-secondary);
}

.search-item-content {
  flex: 1;
}

.search-item-label {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-primary);
}

.search-item-path {
  font-size: 12px;
  color: var(--text-tertiary);
  margin-top: 2px;
}

/* ==================== Header Right ==================== */
.header-right {
  display: flex;
  align-items: center;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.env-badge {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  background: rgba(5, 150, 105, 0.1);
  border: 1px solid rgba(5, 150, 105, 0.2);
  border-radius: 20px;
  font-size: 12px;
  font-weight: 600;
  color: var(--success);
}

.env-dot {
  width: 6px;
  height: 6px;
  background: var(--success);
  border-radius: 50%;
  animation: pulse 2s infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}

.icon-btn {
  width: 40px;
  height: 40px;
  border: none;
  background: transparent;
  border-radius: 10px;
  color: var(--text-secondary);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
  transition: all 0.2s ease;
  position: relative;
}

.icon-btn:hover {
  background: var(--bg-secondary);
  color: var(--primary);
}

.header-divider {
  width: 1px;
  height: 24px;
  background: var(--border);
  margin: 0 8px;
}

.user-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 12px;
  background: transparent;
  border: 1px solid transparent;
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.user-btn:hover {
  background: var(--bg-secondary);
  border-color: var(--border);
}

.user-btn-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
}

.dropdown-arrow {
  font-size: 10px;
  color: var(--text-tertiary);
}

/* ==================== Notification Dropdown ==================== */
.notif-dropdown {
  width: 380px;
  background: #fff;
  border-radius: 12px;
  box-shadow: var(--shadow-lg);
  overflow: hidden;
}

.notif-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 16px 12px;
  border-bottom: 1px solid var(--border);
}

.notif-header h3 {
  font-size: 15px;
  font-weight: 700;
  margin: 0;
  color: var(--text-primary);
}

.notif-dropdown :deep(.ant-tabs) {
  padding: 0 16px;
  margin-bottom: 0;
}

.notif-dropdown :deep(.ant-tabs-nav) {
  margin-bottom: 12px;
}

.notif-list {
  max-height: 400px;
  overflow-y: auto;
  padding: 0 16px 16px;
}

.notif-empty {
  padding: 40px 20px;
  text-align: center;
  color: var(--text-tertiary);
  font-size: 14px;
}

.notif-item {
  display: flex;
  gap: 12px;
  padding: 12px;
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.15s ease;
}

.notif-item:hover {
  background: var(--bg-secondary);
}

.notif-icon {
  width: 36px;
  height: 36px;
  background: var(--primary-gradient-subtle);
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--primary);
  flex-shrink: 0;
}

.notif-content {
  flex: 1;
  min-width: 0;
}

.notif-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 4px;
}

.notif-item-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
}

.notif-time {
  font-size: 12px;
  color: var(--text-tertiary);
  white-space: nowrap;
}

.notif-text {
  font-size: 13px;
  color: var(--text-secondary);
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

/* ==================== Main Content ==================== */
.main-content {
  flex: 1;
  padding: 0;
  background: var(--bg-primary);
}

.content-container {
  max-width: 1400px;
  margin: 0 auto;
  padding: 24px 24px 40px;
}

.page-breadcrumb {
  margin-bottom: 16px;
}

.page-breadcrumb :deep(.ant-breadcrumb) {
  font-size: 13px;
}

.page-breadcrumb :deep(.ant-breadcrumb-link) {
  display: flex;
  align-items: center;
  gap: 4px;
  color: var(--text-tertiary);
}

.page-breadcrumb :deep(.ant-breadcrumb-separator) {
  color: var(--text-tertiary);
}

/* ==================== Mobile ==================== */
.mobile-header {
  display: none;
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  height: 56px;
  background: #fff;
  border-bottom: 1px solid var(--border);
  z-index: 100;
  padding: 0 16px;
  align-items: center;
  justify-content: space-between;
}

.mobile-menu-btn {
  width: 40px;
  height: 40px;
  border: none;
  background: transparent;
  font-size: 18px;
  color: var(--text-primary);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
}

.mobile-brand {
  font-size: 16px;
  font-weight: 700;
  color: var(--text-primary);
}

.mobile-drawer :deep(.ant-drawer-content) {
  background: linear-gradient(180deg, #f8fafc 0%, #f1f5f9 100%);
}

.mobile-drawer :deep(.ant-drawer-header) {
  display: none;
}

.mobile-drawer-content {
  height: 100%;
  display: flex;
  flex-direction: column;
  padding: 18px 16px;
  position: relative;
  z-index: 1;
}

.mobile-drawer-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 20px;
  padding-bottom: 16px;
  border-bottom: 1px solid rgba(0, 0, 0, 0.08);
}

.mobile-drawer-header .brand-logo {
  width: 40px;
  height: 40px;
  background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
  border-radius: 12px;
}

.mobile-drawer-header h2 {
  font-size: 16px;
  font-weight: 700;
  color: #1e293b;
  margin: 0;
}

.mobile-nav {
  flex: 1;
  overflow-y: auto;
}

.mobile-group-title {
  font-size: 10px;
  font-weight: 700;
  color: #94a3b8;
  text-transform: uppercase;
  letter-spacing: 0.1em;
  padding: 14px 12px 8px;
}

.mobile-menu-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border-radius: 10px;
  color: #64748b;
  text-decoration: none;
  transition: all 0.2s ease;
}

.mobile-menu-item:hover {
  background: rgba(99, 102, 241, 0.08);
  color: #6366f1;
}

.mobile-menu-item-active {
  background: linear-gradient(90deg, rgba(99, 102, 241, 0.12) 0%, rgba(99, 102, 241, 0.04) 100%);
  color: #6366f1;
  font-weight: 600;
}

.mobile-menu-icon {
  font-size: 17px;
}

.mobile-menu-item span:not(.mobile-menu-icon, .mobile-menu-badge) {
  flex: 1;
  font-size: 13px;
  font-weight: 500;
}

.mobile-menu-badge {
  background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
  color: #fff;
  font-size: 10px;
  font-weight: 700;
  padding: 2px 8px;
  border-radius: 10px;
  min-width: 18px;
  text-align: center;
  box-shadow: 0 2px 6px rgba(99, 102, 241, 0.25);
}

/* ==================== Transitions ==================== */
.fade-slide-enter-active, .fade-slide-leave-active {
  transition: all 0.2s ease;
}

.fade-slide-enter-from, .fade-slide-leave-to {
  opacity: 0;
  transform: translateX(-10px);
}

.page-enter-active, .page-leave-active {
  transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
}

.page-enter-from {
  opacity: 0;
  transform: translateY(12px);
}

.page-leave-to {
  opacity: 0;
  transform: translateY(-12px);
}

.dropdown-enter-active, .dropdown-leave-active {
  transition: all 0.2s ease;
  transform-origin: top;
}

.dropdown-enter-from, .dropdown-leave-to {
  opacity: 0;
  transform: scaleY(0.95);
}

/* ==================== Responsive ==================== */
@media (max-width: 1024px) {
  .sidebar {
    transform: translateX(-100%);
  }

  .main-wrapper {
    margin-left: 0;
  }

  .sidebar-collapsed + .main-wrapper {
    margin-left: 0;
  }

  .mobile-header {
    display: flex;
  }

  .top-header {
    display: none;
  }

  .main-content {
    padding-top: 56px;
  }

  .content-container {
    padding: 16px 16px 24px;
  }

  .page-breadcrumb {
    display: none;
  }
}

@media (max-width: 768px) {
  .header-search {
    width: 240px;
  }

  .notif-dropdown {
    width: calc(100vw - 32px);
    max-width: 380px;
  }

  .header-actions .env-badge {
    display: none;
  }

  .header-divider {
    display: none;
  }

  .user-btn-name,
  .dropdown-arrow {
    display: none;
  }
}

/* ==================== Dark Mode Overrides ==================== */
.console-dark .sidebar {
  background: linear-gradient(180deg, #1a1f2e 0%, #151925 100%);
  border-right-color: rgba(255, 255, 255, 0.06);
}

.console-dark .sidebar-collapsed {
  background: linear-gradient(180deg, #1a1f2e 0%, #151925 100%);
}

.console-dark .brand-title {
  color: #f1f5f9;
}

.console-dark .brand-subtitle {
  color: #64748b;
}

.console-dark .menu-item {
  color: #94a3b8;
}

.console-dark .menu-item:hover {
  background: rgba(59, 130, 246, 0.12);
  color: #60a5fa;
}

.console-dark .menu-item-active {
  background: linear-gradient(90deg, rgba(59, 130, 246, 0.2) 0%, rgba(59, 130, 246, 0.08) 100%);
  color: #60a5fa;
}

.console-dark .menu-item-active::before {
  background: linear-gradient(180deg, #3b82f6 0%, #2563eb 100%);
}

.console-dark .menu-group-title {
  color: #64748b;
}

.console-dark .user-name {
  color: #f1f5f9;
}

.console-dark .user-role {
  color: #64748b;
}

.console-dark .top-header {
  background: rgba(15, 20, 25, 0.9);
  border-bottom-color: rgba(255, 255, 255, 0.06);
}

.console-dark .header-search input {
  background: var(--bg-secondary);
  border-color: var(--border);
  color: var(--text-primary);
}

.console-dark .header-search input:focus {
  background: var(--bg-tertiary);
  border-color: var(--primary);
}

.console-dark .search-results {
  background: var(--card);
  border-color: var(--border);
  box-shadow: 0 12px 28px rgba(0, 0, 0, 0.6);
}

.console-dark .search-item:hover {
  background: var(--bg-secondary);
}

.console-dark .collapse-btn {
  background: var(--bg-secondary);
  color: var(--text-secondary);
}

.console-dark .collapse-btn:hover {
  background: var(--bg-tertiary);
  color: var(--primary);
}

.console-dark .icon-btn {
  color: var(--text-secondary);
}

.console-dark .icon-btn:hover {
  background: var(--bg-secondary);
  color: var(--primary);
}

.console-dark .user-btn {
  color: var(--text-primary);
}

.console-dark .user-btn:hover {
  background: var(--bg-secondary);
  border-color: var(--border);
}

.console-dark .user-btn-name {
  color: var(--text-primary);
}

.console-dark .notif-dropdown {
  background: var(--card);
  border-color: var(--border);
  box-shadow: 0 12px 28px rgba(0, 0, 0, 0.6);
}

.console-dark .notif-header {
  border-bottom-color: var(--border);
}

.console-dark .notif-header h3 {
  color: var(--text-primary);
}

.console-dark .notif-item:hover {
  background: var(--bg-secondary);
}

.console-dark .notif-item-title {
  color: var(--text-primary);
}

.console-dark .notif-text {
  color: var(--text-secondary);
}

.console-dark .main-content {
  background: var(--bg-primary);
}

.console-dark .page-breadcrumb :deep(.ant-breadcrumb-link) {
  color: var(--text-tertiary);
}

.console-dark .page-breadcrumb :deep(.ant-breadcrumb-separator) {
  color: var(--text-tertiary);
}

.console-dark .env-badge {
  background: rgba(16, 185, 129, 0.15);
  border-color: rgba(16, 185, 129, 0.25);
  color: var(--success);
}

.console-dark .header-divider {
  background: var(--border);
}

.console-dark .mobile-header {
  background: #1a1f2e;
  border-bottom-color: var(--border);
}

.console-dark .mobile-brand {
  color: var(--text-primary);
}

.console-dark .mobile-drawer :deep(.ant-drawer-content) {
  background: linear-gradient(180deg, #1a1f2e 0%, #151925 100%);
}

.console-dark .mobile-drawer-header {
  border-bottom-color: rgba(255, 255, 255, 0.08);
}

.console-dark .mobile-drawer-header h2 {
  color: #f1f5f9;
}

.console-dark .mobile-group-title {
  color: #64748b;
}

.console-dark .mobile-menu-item {
  color: #94a3b8;
}

.console-dark .mobile-menu-item:hover {
  background: rgba(59, 130, 246, 0.12);
  color: #60a5fa;
}

.console-dark .mobile-menu-item-active {
  background: linear-gradient(90deg, rgba(59, 130, 246, 0.2) 0%, rgba(59, 130, 246, 0.08) 100%);
  color: #60a5fa;
}

/* Theme toggle button animation */
.theme-toggle-btn svg {
  transition: transform 0.3s ease;
}

.theme-toggle-btn:hover svg {
  transform: scale(1.1) rotate(15deg);
}
</style>
