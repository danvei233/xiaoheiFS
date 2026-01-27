<template>
  <a-layout class="layout">
    <a-layout-header class="header">
      <div class="header-left">
        <a-button class="menu-trigger" type="text" @click="toggleMenu">
          <component :is="menuIcon" />
        </a-button>
        <div v-if="!isMobile" class="brand">
          <div class="brand-logo" aria-hidden="true">
            <SiteLogoMedia :size="22" />
          </div>
          <div>
            <div class="brand-name">{{ brandName }}</div>
            <!-- <div class="brand-subtle">Cloud VPS Console</div> -->
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
          <a-tag v-if="!isMobile" color="blue">Production</a-tag>
          <a-popover
            v-model:open="notifOpen"
            trigger="click"
            placement="bottomRight"
            overlay-class-name="notif-popover"
            @openChange="handleNotifOpen"
          >
            <template #content>
              <div class="notif-panel">
                <div class="notif-header">
                  <span class="notif-title">消息中心</span>
                  <a-button type="link" size="small" @click.stop="markAllRead">全部已读</a-button>
                </div>
                <a-tabs v-model:activeKey="notifTab" size="small" @change="fetchNotifications">
                  <a-tab-pane key="unread" tab="未读" />
                  <a-tab-pane key="read" tab="已读" />
                  <a-tab-pane key="all" tab="全部" />
                </a-tabs>
                <a-list
                  :data-source="notifications"
                  :loading="notifLoading"
                  :locale="{ emptyText: '暂无消息' }"
                  item-layout="horizontal"
                  size="small"
                >
                  <template #renderItem="{ item }">
                    <a-list-item @click="markRead(item)">
                      <a-list-item-meta :title="item.title || item.type" :description="item.content" />
                      <div class="notif-time">{{ formatTime(item.created_at) }}</div>
                    </a-list-item>
                  </template>
                </a-list>
              </div>
            </template>
            <a-badge :count="unreadCount" size="small">
              <a-button type="text" shape="circle">
                <BellOutlined />
              </a-button>
            </a-badge>
          </a-popover>
          <a-tooltip v-if="!isMobile" title="帮助中心">
            <a-button type="text" shape="circle" @click="goHelp">
              <QuestionCircleOutlined />
            </a-button>
          </a-tooltip>
          <a-dropdown>
            <div class="user-info">
              <a-avatar size="small" :src="userAvatar" style="background-color: #1677ff">
                <UserOutlined />
              </a-avatar>
              <span class="user-name">{{ displayName }}</span>
            </div>
            <template #overlay>
              <a-menu>
                <a-menu-item @click="goProfile">个人信息</a-menu-item>
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
        width="220"
        @collapse="collapsed = $event"
      >
        <a-menu mode="inline" :selectedKeys="[selectedKey]" @click="onMenu">
          <a-menu-item key="/console">总览</a-menu-item>
          <a-menu-item key="/console/vps">VPS</a-menu-item>
          <a-menu-item key="/console/cart">购物车</a-menu-item>
          <a-menu-item key="/console/orders">订单/财务</a-menu-item>
          <a-menu-item key="/console/billing">钱包/充值</a-menu-item>
          <a-menu-item key="/console/tickets">工单中心</a-menu-item>
          <a-menu-item key="/console/realname">实名认证</a-menu-item>
          <a-menu-item key="/console/profile">个人信息</a-menu-item>
        </a-menu>
      </a-layout-sider>
      <a-drawer
        v-model:open="drawerOpen"
        placement="left"
        width="220"
        :body-style="{ padding: 0 }"
        :mask-style="{ backgroundColor: 'rgba(0, 0, 0, 0.45)' }"
        :wrap-style="{ position: 'absolute' }"
        :z-index="1001"
        class="mobile-drawer"
      >
        <div class="drawer-content">
          <a-menu mode="inline" :selectedKeys="[selectedKey]" @click="onMenu" class="drawer-menu">
            <a-menu-item key="/console" class="drawer-menu-item">
              <DashboardOutlined />
              <span>总览</span>
            </a-menu-item>
            <a-menu-item key="/console/vps" class="drawer-menu-item">
              <CloudServerOutlined />
              <span>VPS</span>
            </a-menu-item>
            <a-menu-item key="/console/cart" class="drawer-menu-item">
              <ShoppingCartOutlined />
              <span>购物车</span>
            </a-menu-item>
            <a-menu-item key="/console/orders" class="drawer-menu-item">
              <FileTextOutlined />
              <span>订单/财务</span>
            </a-menu-item>
            <a-menu-item key="/console/billing" class="drawer-menu-item">
              <CreditCardOutlined />
              <span>钱包/充值</span>
            </a-menu-item>
            <a-menu-item key="/console/tickets" class="drawer-menu-item">
              <CustomerServiceOutlined />
              <span>工单中心</span>
            </a-menu-item>
            <a-menu-item key="/console/realname" class="drawer-menu-item">
              <SafetyCertificateOutlined />
              <span>实名认证</span>
            </a-menu-item>
            <a-menu-item key="/console/profile" class="drawer-menu-item">
              <UserOutlined />
              <span>个人信息</span>
            </a-menu-item>
          </a-menu>
        </div>
      </a-drawer>

      <a-layout>
        <a-layout-content class="content">
          <div class="breadcrumb-wrap">
            <a-breadcrumb>
              <a-breadcrumb-item>用户中心</a-breadcrumb-item>
              <a-breadcrumb-item>{{ breadcrumb }}</a-breadcrumb-item>
            </a-breadcrumb>
          </div>
          <router-view />
        </a-layout-content>
      </a-layout>
    </a-layout>
  </a-layout>
</template>

<script setup>
import { computed, ref, onMounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useAuthStore } from "@/stores/auth";
import { useSiteStore } from "@/stores/site";
import SiteLogoMedia from "@/components/brand/SiteLogoMedia.vue";
import { listNotifications, getUnreadCount, markNotificationRead, markAllNotificationsRead } from "@/services/user";
import { Grid } from "ant-design-vue";
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
  FileTextOutlined,
  CustomerServiceOutlined,
  SafetyCertificateOutlined,
  CreditCardOutlined
} from "@ant-design/icons-vue";

const route = useRoute();
const router = useRouter();
const auth = useAuthStore();
const site = useSiteStore();
const screens = Grid.useBreakpoint();

const collapsed = ref(false);
const drawerOpen = ref(false);
const searchValue = ref('');
const notifOpen = ref(false);
const notifTab = ref("unread");
const notifications = ref([]);
const notifLoading = ref(false);
const unreadCount = ref(0);

const isMobile = computed(() => !screens.value?.lg);
const menuIcon = computed(() => {
  if (isMobile.value) return MenuUnfoldOutlined;
  return collapsed.value ? MenuUnfoldOutlined : MenuFoldOutlined;
});

const selectedKey = computed(() => {
  if (route.path.startsWith("/console/buy")) return "/console/vps";
  if (route.path.startsWith("/console/vps")) return "/console/vps";
  if (route.path.startsWith("/console/orders")) return "/console/orders";
  if (route.path.startsWith("/console/billing")) return "/console/billing";
  if (route.path.startsWith("/console/tickets")) return "/console/tickets";
  if (route.path.startsWith("/console/realname")) return "/console/realname";
  if (route.path.startsWith("/console/cart")) return "/console/cart";
  if (route.path.startsWith("/console/profile")) return "/console/profile";
  return "/console";
});

const labelMap = {
  "/console": "总览",
  "/console/vps": "VPS",
  "/console/orders": "订单/财务",
  "/console/billing": "钱包/充值",
  "/console/tickets": "工单中心",
  "/console/realname": "实名认证",
  "/console/cart": "购物车",
  "/console/profile": "个人信息"
};

const breadcrumb = computed(() => labelMap[selectedKey.value] || "总览");
const displayName = computed(() => auth.profile?.username || "用户");
const brandName = computed(() => site.siteName || "用户控制台");
const userAvatar = computed(() => {
  const qq = auth.profile?.qq;
  if (!qq) return "";
  return `https://q1.qlogo.cn/g?b=qq&nk=${qq}&s=100`;
});

const onMenu = ({ key }) => {
  router.push(String(key));
  drawerOpen.value = false;
};

const toggleMenu = () => {
  if (isMobile.value) {
    drawerOpen.value = true;
  } else {
    collapsed.value = !collapsed.value;
  }
};

const logout = () => {
  auth.logout();
  router.replace("/login");
};

const goProfile = () => router.push("/console/profile");

const handleSearch = () => {
  // AutoComplete only updates options via v-model
};

const handleEnterSearch = () => {
  const keyword = searchValue.value.trim();
  if (!keyword) {
    return;
  }
  const options = searchOptions.value;
  const target = options.find((item) => item.label.toLowerCase() === keyword.toLowerCase()) || options[0];
  if (target?.value) {
    router.push(String(target.value));
    searchValue.value = "";
  }
};

const searchOptions = computed(() => {
  const keyword = searchValue.value.trim().toLowerCase();
  const items = [
    { label: "总览", value: "/console" },
    { label: "VPS", value: "/console/vps" },
    { label: "购物车", value: "/console/cart" },
    { label: "订单/财务", value: "/console/orders" },
    { label: "钱包/充值", value: "/console/billing" },
    { label: "工单中心", value: "/console/tickets" },
    { label: "实名认证", value: "/console/realname" },
    { label: "个人信息", value: "/console/profile" }
  ];
  const filtered = keyword ? items.filter((item) => item.label.toLowerCase().includes(keyword)) : items;
  return filtered.map((item) => ({
    value: item.value,
    label: item.label
  }));
});

const handleSelect = (value) => {
  if (value) {
    router.push(String(value));
    searchValue.value = "";
  }
};

const goHelp = () => {
  router.push("/help");
};

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

const handleNotifOpen = (open) => {
  notifOpen.value = open;
  if (open) {
    fetchUnreadCount();
    fetchNotifications();
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

onMounted(() => {
  if (!auth.profile && auth.token) {
    auth.fetchMe();
  }
  if (auth.token) {
    fetchUnreadCount();
  }
  site.fetchSettings();
});
</script>

<style scoped>
.layout {
  min-height: 100vh;
  background: linear-gradient(180deg, var(--bg-primary) 0%, var(--bg-secondary) 50%, var(--bg-tertiary) 100%);
  position: relative;
}

.layout :deep(.ant-layout),
.layout :deep(.ant-layout-content) {
  min-width: 0;
}

.layout::before {
  content: '';
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background:
    radial-gradient(ellipse at 0% 0%, rgba(0, 102, 255, 0.03) 0%, transparent 50%),
    radial-gradient(ellipse at 100% 100%, rgba(0, 51, 153, 0.02) 0%, transparent 50%);
  pointer-events: none;
  z-index: 0;
}

.header {
  background: rgba(255, 255, 255, 0.85);
  backdrop-filter: blur(30px) saturate(180%);
  -webkit-backdrop-filter: blur(30px) saturate(180%);
  border-bottom: 1px solid var(--border);
  box-shadow: var(--shadow-sm), var(--shadow-inner);
  color: var(--text-primary);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 0 28px;
  height: 72px;
  position: sticky;
  top: 0;
  z-index: 1000;
  transition: all var(--transition-smooth);
}

.header::after {
  content: '';
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 2px;
  background: var(--primary-gradient);
  transform: scaleX(0);
  transform-origin: center;
  transition: transform var(--transition-smooth);
  opacity: 0.5;
}

.header:hover::after {
  transform: scaleX(1);
}

.header-left {
  display: flex;
  align-items: center;
  gap: 18px;
  position: relative;
  z-index: 1;
}

.menu-trigger {
  width: 44px;
  height: 44px;
  border-radius: var(--radius-md);
  transition: all var(--transition-smooth);
  color: var(--text-secondary);
  position: relative;
  overflow: hidden;
}

.menu-trigger::before {
  content: '';
  position: absolute;
  inset: 0;
  background: var(--primary-gradient);
  opacity: 0;
  transition: opacity var(--transition-smooth);
}

.menu-trigger:hover::before {
  opacity: 0.1;
}

.menu-trigger:hover {
  color: var(--primary);
  transform: scale(1.08);
}

.brand {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 4px 0;
}

.brand-logo {
  width: 44px;
  height: 44px;
  border-radius: var(--radius-md);
  background: var(--primary-gradient);
  color: #fff;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 17px;
  box-shadow: var(--shadow-md), var(--shadow-glow-sm);
  transition: all var(--transition-smooth);
  position: relative;
  overflow: hidden;
}

.brand-logo::before {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(135deg, rgba(255, 255, 255, 0.25) 0%, transparent 50%);
  opacity: 0;
  transition: opacity var(--transition-smooth);
}

.brand-logo::after {
  content: '';
  position: absolute;
  top: -50%;
  left: -50%;
  width: 200%;
  height: 200%;
  background: radial-gradient(circle, rgba(255, 255, 255, 0.3) 0%, transparent 70%);
  opacity: 0;
  transition: all var(--transition-spring);
}

.brand-logo:hover {
  transform: translateY(-4px) scale(1.05);
  box-shadow: var(--shadow-xl), var(--shadow-glow);
}

.brand-logo:hover::before {
  opacity: 1;
}

.brand-logo:hover::after {
  opacity: 1;
  animation: shimmer 1.5s infinite;
}

.brand-logo-img {
  width: 44px;
  height: 44px;
  border-radius: var(--radius-md);
  object-fit: cover;
  box-shadow: var(--shadow-md), var(--shadow-glow-sm);
  transition: all var(--transition-smooth);
}

.brand-logo-img:hover {
  transform: translateY(-4px) scale(1.05);
  box-shadow: var(--shadow-xl), var(--shadow-glow);
}

.brand-name {
  font-weight: 800;
  font-size: 19px;
  letter-spacing: -0.02em;
  background: var(--primary-gradient);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  transition: all var(--transition-smooth);
}

.brand:hover .brand-name {
  filter: brightness(1.1);
}

.brand-subtle {
  font-size: 12px;
  color: var(--text-tertiary);
}

.header-search-wrapper {
  position: relative;
  width: 300px;
  transition: all var(--transition-smooth);
}

.header-search-wrapper:hover {
  width: 340px;
}

.header-search {
  width: 100%;
  border-radius: var(--radius-md);
}

.header-search :deep(.ant-input-affix-wrapper) {
  height: 44px;
  padding: 0 14px;
  background: var(--glass-bg);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  transition: all var(--transition-smooth);
  box-shadow: var(--shadow-sm), var(--shadow-inner);
}

.header-search :deep(.ant-input-affix-wrapper:hover) {
  background: var(--glass-bg-dark);
  border-color: var(--primary-light);
  box-shadow: var(--shadow-md);
}

.header-search :deep(.ant-input-affix-wrapper-focused) {
  background: var(--glass-bg-dark);
  border-color: var(--primary);
  box-shadow: 0 0 0 4px rgba(0, 102, 255, 0.08), var(--shadow-md);
}

.header-search :deep(.ant-input) {
  height: 40px;
  line-height: 40px;
  background: transparent;
  color: var(--text-primary);
  padding: 0 10px;
  font-weight: 500;
}

.header-search :deep(.ant-input::placeholder) {
  color: var(--text-tertiary);
  font-weight: 400;
}

.header-search :deep(.ant-input-prefix) {
  color: var(--text-tertiary);
  margin-right: 10px;
}

.header-search :deep(.ant-input-prefix .anticon) {
  transition: all var(--transition-smooth);
}

.header-search:hover :deep(.ant-input-prefix .anticon) {
  color: var(--primary);
  transform: scale(1.1);
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.sider {
  background: rgba(255, 255, 255, 0.7);
  backdrop-filter: blur(20px) saturate(180%);
  -webkit-backdrop-filter: blur(20px) saturate(180%);
  border-right: 1px solid var(--border);
  transition: all var(--transition-smooth);
  position: relative;
}

.sider::before {
  content: '';
  position: absolute;
  top: 0;
  right: 0;
  width: 1px;
  height: 100%;
  background: linear-gradient(180deg, transparent 0%, var(--primary) 50%, transparent 100%);
  opacity: 0.1;
}

.sider :deep(.ant-menu) {
  background: transparent;
  border: none;
  padding: 16px 12px;
}

.sider :deep(.ant-menu-item) {
  margin: 6px 0;
  padding: 12px 16px;
  height: auto;
  border-radius: var(--radius-md);
  transition: all var(--transition-smooth);
  color: var(--text-secondary);
  font-weight: 500;
  position: relative;
  overflow: hidden;
}

.sider :deep(.ant-menu-item)::before {
  content: '';
  position: absolute;
  left: 0;
  top: 50%;
  transform: translateY(-50%);
  width: 3px;
  height: 0;
  background: var(--primary-gradient);
  border-radius: 0 4px 4px 0;
  transition: height var(--transition-smooth);
}

.sider :deep(.ant-menu-item:hover)::before {
  height: 24px;
}

.sider :deep(.ant-menu-item:hover) {
  background: var(--primary-gradient-subtle);
  color: var(--primary);
  transform: translateX(4px);
}

.sider :deep(.ant-menu-item-selected) {
  background: var(--primary-gradient);
  color: #fff !important;
  font-weight: 600;
  box-shadow: var(--shadow-md), var(--shadow-glow-sm);
  transform: translateX(4px);
}

.sider :deep(.ant-menu-item-selected)::before {
  height: 28px;
}

.content {
  width: 100%;
  min-width: 0;
  padding: 0;
  background: transparent;
  position: relative;
  z-index: 1;
}

.breadcrumb-wrap {
  max-width: 1400px;
  margin: 0 auto;
  padding: 18px 28px 0;
}

.breadcrumb-wrap :deep(.ant-breadcrumb) {
  margin-bottom: 12px;
}

.content :deep(.ant-table-container),
.content :deep(.ant-table-content) {
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 12px;
  cursor: pointer;
  padding: 10px 16px;
  border-radius: var(--radius-md);
  transition: all var(--transition-smooth);
  border: 1px solid transparent;
  position: relative;
  overflow: hidden;
}

.user-info::before {
  content: '';
  position: absolute;
  inset: 0;
  background: var(--primary-gradient);
  opacity: 0;
  transition: opacity var(--transition-smooth);
}

.user-info:hover {
  border-color: rgba(0, 102, 255, 0.15);
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
}

.user-info:hover::before {
  opacity: 0.05;
}

.user-name {
  font-weight: 700;
  font-size: 14px;
  color: var(--text-primary);
  position: relative;
}

.header :deep(.ant-btn),
.header :deep(.ant-btn-text) {
  color: var(--text-secondary);
  transition: all var(--transition-smooth);
  position: relative;
  overflow: hidden;
}

.header :deep(.ant-btn)::before {
  content: '';
  position: absolute;
  inset: 0;
  background: var(--primary-gradient);
  opacity: 0;
  transition: opacity var(--transition-smooth);
}

.header :deep(.ant-btn:hover)::before {
  opacity: 0.08;
}

.header :deep(.ant-btn:hover),
.header :deep(.ant-btn:focus) {
  color: var(--primary);
  transform: translateY(-2px) scale(1.05);
}

.header :deep(.anticon) {
  transition: all var(--transition-smooth);
}

.header :deep(.ant-btn:hover .anticon) {
  transform: scale(1.1);
}

.header :deep(.ant-input-search) {
  border-radius: var(--radius-md);
  overflow: hidden;
}

.header :deep(.ant-input) {
  background: transparent;
  color: var(--text-primary);
  font-size: 14px;
  padding: 6px 40px 6px 36px;
}

.header :deep(.ant-input::placeholder) {
  color: var(--text-tertiary);
  font-size: 14px;
  font-weight: 400;
}

.header :deep(.ant-input-prefix) {
  color: var(--text-tertiary);
  margin-right: 8px;
}

.header :deep(.ant-input-prefix .anticon) {
  font-size: 16px;
  transition: all var(--transition-smooth);
}

.header :deep(.ant-input:focus) .ant-input-prefix .anticon {
  color: var(--primary);
  transform: scale(1.1);
}

.header :deep(.ant-badge-count) {
  background: var(--primary-gradient);
  box-shadow: var(--shadow-md), var(--shadow-glow-sm);
  font-weight: 700;
  font-size: 11px;
}

.header :deep(.ant-tag) {
  background: var(--primary-gradient-subtle);
  border: 1px solid rgba(0, 102, 255, 0.15);
  color: var(--primary);
  font-weight: 700;
  padding: 5px 14px;
  border-radius: var(--radius-full);
  font-size: 12px;
  box-shadow: var(--shadow-xs);
}

.header :deep(.ant-tag:hover) {
  box-shadow: var(--shadow-sm);
  transform: translateY(-1px);
}

.header :deep(.ant-avatar) {
  border: 2px solid var(--border-light);
  transition: all var(--transition-smooth);
  box-shadow: var(--shadow-sm);
  position: relative;
}

.header :deep(.ant-avatar)::after {
  content: '';
  position: absolute;
  inset: -2px;
  border-radius: inherit;
  background: var(--primary-gradient);
  opacity: 0;
  transition: opacity var(--transition-smooth);
  z-index: -1;
}

.header :deep(.ant-avatar:hover)::after {
  opacity: 0.2;
}

.header :deep(.ant-avatar:hover) {
  border-color: var(--primary);
  transform: scale(1.08);
  box-shadow: var(--shadow-glow-sm);
}

/* 响应式优化 */
@media (max-width: 768px) {
  .header {
    padding: 0 16px;
    height: 60px;
    gap: 10px;
  }

  .header-search-wrapper {
    display: none;
  }

  .brand-name {
    font-size: 15px;
  }

  .brand-logo {
    width: 36px;
    height: 36px;
    font-size: 14px;
  }

  .header-left {
    gap: 10px;
  }

  .header-actions {
    gap: 8px;
  }

  .menu-trigger {
    width: 38px;
    height: 38px;
  }

  .user-info {
    padding: 8px 10px;
    gap: 10px;
  }

  .user-name {
    max-width: 110px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .breadcrumb-wrap {
    padding: 14px 16px 0;
  }
}

/* Drawer 样式优化 */
.mobile-drawer :deep(.ant-drawer-body) {
  padding: 0;
  background: var(--card);
}

.mobile-drawer :deep(.ant-drawer-content) {
  background: var(--card);
  border-right: 1px solid var(--border);
}

.mobile-drawer :deep(.ant-drawer-header) {
  display: none;
}

.drawer-content {
  height: 100%;
  overflow-y: auto;
  background: var(--card);
}

.drawer-menu {
  border: none;
  background: transparent;
  padding: 12px 8px;
}

.drawer-menu :deep(.ant-menu-item) {
  margin: 4px 0;
  padding: 12px 16px;
  height: auto;
  line-height: 1.5;
  border-radius: var(--radius-md);
  transition: all var(--transition-base);
  display: flex;
  align-items: center;
  gap: 12px;
  color: var(--text-secondary);
  font-weight: 500;
}

.drawer-menu :deep(.ant-menu-item:hover) {
  background: var(--primary-gradient-subtle);
  color: var(--primary);
}

.drawer-menu :deep(.ant-menu-item-selected) {
  background: var(--primary-gradient);
  color: #fff;
  font-weight: 600;
  box-shadow: var(--shadow-glow-sm);
}

.drawer-menu :deep(.ant-menu-item .anticon) {
  font-size: 18px;
  min-width: 18px;
  transition: all var(--transition-base);
}

.drawer-menu :deep(.ant-menu-item:hover .anticon) {
  transform: scale(1.1);
}

.drawer-menu :deep(.ant-menu-item-selected .anticon) {
  color: #fff;
}

.drawer-menu :deep(.ant-menu-item span) {
  font-size: 14px;
  font-weight: 500;
}

/* Drawer 滚动条样式优化 */
.drawer-content::-webkit-scrollbar {
  width: 4px;
}

.drawer-content::-webkit-scrollbar-track {
  background: transparent;
}

.drawer-content::-webkit-scrollbar-thumb {
  background: var(--border-dark);
  border-radius: 2px;
}

.drawer-content::-webkit-scrollbar-thumb:hover {
  background: var(--primary);
}

:deep(.notif-popover .ant-popover-inner) {
  padding: 0;
  border-radius: var(--radius-lg);
  overflow: hidden;
  box-shadow: var(--shadow-lg);
}

.notif-panel {
  width: 360px;
  padding: 16px 16px 12px;
  max-height: min(70vh, 520px) !important;
  display: flex;
  flex-direction: column;
  overflow: hidden !important;
}

.notif-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--border);
}

.notif-title {
  font-weight: 700;
  font-size: 15px;
  color: var(--text-primary);
}

.notif-time {
  font-size: 12px;
  color: var(--text-tertiary);
  white-space: nowrap;
}

.notif-panel :deep(.ant-tabs) {
  margin-bottom: 12px;
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.notif-panel :deep(.ant-tabs-content-holder) {
  min-height: 0;
  flex: 1;
  overflow: hidden !important;
}

.notif-panel :deep(.ant-tabs-content) {
  height: 100%;
}

.notif-panel :deep(.ant-tabs-tabpane) {
  height: 100%;
  overflow: hidden !important;
}

.notif-panel :deep(.ant-list) {
  flex: 1;
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.notif-panel :deep(.ant-spin-nested-loading) {
  flex: 1;
  min-height: 0;
  height: 100%;
  overflow-y: auto !important;
  -webkit-overflow-scrolling: touch;
}

.notif-panel :deep(.ant-spin-container) {
  flex: 1;
  min-height: 0;
  height: 100%;
  overflow: visible !important;
}

.notif-panel :deep(.ant-list-items) {
  overflow-y: auto !important;
  max-height: 100% !important;
  padding-right: 2px;
}

.notif-panel :deep(.ant-list-items::-webkit-scrollbar) {
  width: 4px;
}

.notif-panel :deep(.ant-list-items::-webkit-scrollbar-thumb) {
  background: var(--border-dark);
  border-radius: 2px;
}

.notif-panel :deep(.ant-list-items::-webkit-scrollbar-thumb:hover) {
  background: var(--primary);
}

.notif-panel :deep(.ant-tabs-tab) {
  font-weight: 500;
  color: var(--text-secondary);
}

.notif-panel :deep(.ant-tabs-tab-active) {
  color: var(--primary);
  font-weight: 600;
}

.notif-panel :deep(.ant-list-item) {
  cursor: pointer;
  transition: all var(--transition-base);
  border-radius: var(--radius-md);
  padding: 10px 12px;
  margin: 4px 0;
}

.notif-panel :deep(.ant-list-item:hover) {
  background: var(--bg-secondary);
  transform: translateX(4px);
}
</style>
