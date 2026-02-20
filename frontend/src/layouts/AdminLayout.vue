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
  <a-modal
    :open="mfaModalOpen"
    :title="null"
    :closable="false"
    :maskClosable="false"
    :footer="null"
    width="460px"
    wrap-class-name="admin-mfa-modal-wrap"
    class="admin-mfa-modal"
  >
    <div class="mfa-modal-head">
      <div class="mfa-modal-badge">
        <SafetyCertificateOutlined />
      </div>
      <div class="mfa-modal-copy">
        <div class="mfa-modal-title">{{ mfaTitle }}</div>
        <div class="mfa-modal-subtitle">
          {{ admin.mfaBindRequired ? "首次使用需完成绑定后继续管理操作" : "输入动态验证码以恢复后台敏感操作权限" }}
        </div>
      </div>
    </div>
    <a-alert
      v-if="admin.mfaBindRequired"
      type="warning"
      show-icon
      message="需要绑定 2FA 才能继续使用后台功能"
      class="mfa-alert"
    />
    <a-alert
      v-else
      type="info"
      show-icon
      message="请输入 2FA 验证码以解锁后台操作"
      class="mfa-alert"
    />
    <a-form layout="vertical" class="mfa-form">
      <a-form-item v-if="admin.mfaBindRequired" label="登录密码">
        <a-input-password v-model:value="mfaForm.password" placeholder="用于生成绑定信息" class="mfa-input" />
      </a-form-item>
      <a-form-item v-if="admin.mfaBindRequired" label="生成绑定信息">
        <a-button type="default" :loading="mfaLoading.setup" block class="mfa-setup-btn" @click="handleSetup2FA">
          生成 2FA 绑定信息
        </a-button>
      </a-form-item>
      <div v-if="mfaSecret" class="twofa-setup">
        <div class="twofa-qr">
          <img v-if="mfaQRCode" :src="mfaQRCode" alt="2FA QRCode" />
        </div>
        <div class="twofa-meta">
          <div class="twofa-label">手动密钥</div>
          <div class="twofa-secret">{{ mfaSecret }}</div>
        </div>
      </div>
      <a-form-item label="2FA 验证码">
        <a-input
          v-model:value="mfaForm.totpCode"
          placeholder="请输入 6 位验证码"
          maxlength="6"
          inputmode="numeric"
          class="mfa-code-input"
        />
      </a-form-item>
      <div class="mfa-helper">验证码每 30 秒更新，请使用最新口令。</div>
      <a-button type="primary" block class="mfa-confirm-btn" :loading="mfaLoading.confirm" @click="handleConfirmOrUnlock">
          {{ admin.mfaBindRequired ? "完成绑定并解锁" : "解锁" }}
      </a-button>
    </a-form>
  </a-modal>
  </a-config-provider>
</template>

<script setup>
import { computed, ref, onMounted, onUnmounted, nextTick, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useAdminAuthStore } from "@/stores/adminAuth";
import { useSiteStore } from "@/stores/site";
import { admin2FAUnlock } from "@/services/admin";
import { adminSetupTwoFA, adminConfirmTwoFA } from "@/services/admin";
import SiteLogoMedia from "@/components/brand/SiteLogoMedia.vue";
import AdminMenuNode from "@/components/admin/AdminMenuNode.vue";
import { Grid, theme, message } from "ant-design-vue";
import QRCode from "qrcode";
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
  MessageOutlined,
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
    key: "group-trade",
    label: "交易与资源",
    icon: ShoppingCartOutlined,
    children: [
      { key: "/admin/orders", label: "订单审核", icon: ShoppingCartOutlined, requireAny: ["order.list"] },
      { key: "/admin/wallet/orders", label: "钱包订单", icon: WalletOutlined, requireAny: ["wallet_order.list"] },
      { key: "/admin/vps", label: "VPS 管理", icon: CloudServerOutlined, requireAny: ["vps.list"] },
      { key: "/admin/probes", label: "探针监控", icon: ApiOutlined, requireAny: ["probe.list"] },
      {
        key: "/admin/catalog",
        label: "售卖目录",
        icon: AppstoreOutlined,
        requireAny: ["regions.list", "plan_group.list", "line.list", "packages.list", "system_image.list", "billing_cycle.list"]
      }
    ]
  },
  {
    key: "group-support",
    label: "用户与客服",
    icon: TeamOutlined,
    children: [
      { key: "/admin/users", label: "用户管理", icon: TeamOutlined, requireAny: ["user.list"] },
      { key: "/admin/user-tiers", label: "用户等级", icon: SafetyCertificateOutlined, requireAny: ["user.list"] },
      { key: "/admin/coupons", label: "优惠码", icon: DollarOutlined, requireAny: ["settings.view"] },
      { key: "/admin/tickets", label: "工单管理", icon: CustomerServiceOutlined, requireAny: ["tickets.list"] },
      { key: "/admin/realname/providers", label: "实名供应商", icon: AppstoreOutlined, requireAny: ["realname.list"] },
      { key: "/admin/realname/config", label: "实名认证配置", icon: SafetyCertificateOutlined, requireAny: ["realname.view"] },
      { key: "/admin/realname/records", label: "实名认证记录", icon: FileSearchOutlined, requireAny: ["realname.list"] }
    ]
  },
  {
    key: "group-cms",
    label: "内容运营",
    icon: FileTextOutlined,
    children: [
      { key: "/admin/cms/categories", label: "文章分类", icon: AppstoreOutlined, requireAny: ["cms_category.list"] },
      { key: "/admin/cms/posts", label: "文章内容", icon: FileTextOutlined, requireAny: ["cms_post.list"] },
      { key: "/admin/cms/blocks", label: "页面模块", icon: LayoutOutlined, requireAny: ["cms_block.list"] },
      { key: "/admin/cms/nav-items", label: "导航菜单", icon: MenuOutlined, requireAny: ["settings.view"] },
      { key: "/admin/cms/uploads", label: "媒体资源", icon: CloudUploadOutlined, requireAny: ["upload.list"] }
    ]
  },
  {
    key: "group-platform",
    label: "平台配置",
    icon: SettingOutlined,
    children: [
      { key: "/admin/settings/site", label: "站点设置", icon: SettingOutlined, requireAny: ["settings.view"] },
      { key: "/admin/settings/auth", label: "登录与注册", icon: SafetyCertificateOutlined, requireAny: ["settings.view"] },
      { key: "/admin/settings/pricing", label: "价格与退款", icon: DollarOutlined, requireAny: ["settings.view"] },
      { key: "/admin/settings/payments", label: "支付设置", icon: CreditCardOutlined, requireAny: ["payment.list"] },
      { key: "/admin/settings/lifecycle", label: "生命周期", icon: ClockCircleOutlined, requireAny: ["settings.view"] },
      { key: "/admin/settings/plugins", label: "插件管理", icon: ToolOutlined, requireAny: ["plugin.list"] },
      { key: "/admin/automation", label: "自动化对接", icon: ApiOutlined, requireAny: ["automation.view"] },
      { key: "/admin/settings/webhook", label: "Webhook", icon: LinkOutlined, requireAny: ["robot.view"] },
      { key: "/admin/scheduled-tasks", label: "计划任务", icon: ClockCircleOutlined, requireAny: ["scheduled_tasks.list"] }
    ]
  },
  {
    key: "group-notify",
    label: "通知与集成",
    icon: MessageOutlined,
    children: [
      { key: "/admin/settings/email", label: "邮件与模板", icon: MailOutlined, requireAny: ["smtp.view", "email_template.list"] },
      { key: "/admin/settings/sms", label: "短信设置", icon: MessageOutlined, requireAny: ["sms.view", "sms_template.list"] },
      { key: "/admin/settings/fcm", label: "FCM 推送", icon: BellOutlined, requireAny: ["settings.view"] }
    ]
  },
  {
    key: "group-security",
    label: "安全与审计",
    icon: KeyOutlined,
    children: [
      { key: "/admin/admins", label: "管理员列表", icon: UserOutlined, requireAny: ["admin.list"] },
      { key: "/admin/permission-groups", label: "权限组", icon: KeyOutlined, requireAny: ["permission_group.list"] },
      { key: "/admin/settings/apikey", label: "API Keys", icon: KeyOutlined, requireAny: ["api_key.list"] },
      { key: "/admin/settings/captcha", label: "验证码设置", icon: SafetyCertificateOutlined, requireAny: ["settings.view"] },
      { key: "/admin/audit", label: "审计日志", icon: SettingOutlined, requireAny: ["audit_log.view"] },
      { key: "/admin/debug", label: "调试中心", icon: BugOutlined, requireAny: ["debug.view", "debug.list"] }
    ]
  },
  { key: "/admin/profile", label: "个人资料", icon: UserOutlined }
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

const mfaForm = ref({
  password: "",
  totpCode: ""
});
const mfaSecret = ref("");
const mfaQRCode = ref("");
const mfaLoading = ref({
  setup: false,
  confirm: false
});

const mfaModalOpen = computed(() => (admin.mfaBindRequired || admin.mfaRequired) && !admin.mfaUnlocked);
const mfaTitle = computed(() => (admin.mfaBindRequired ? "管理员 2FA 绑定" : "管理员 2FA 解锁"));

const resetMfaForm = () => {
  mfaForm.value.password = "";
  mfaForm.value.totpCode = "";
  mfaSecret.value = "";
  mfaQRCode.value = "";
};

const handleSetup2FA = async () => {
  if (!admin.mfaBindRequired || mfaLoading.value.setup) return;
  mfaLoading.value.setup = true;
  try {
    const password = String(mfaForm.value.password || "").trim();
    const res = await adminSetupTwoFA({ password: password || undefined });
    mfaSecret.value = String(res.data?.secret || "");
    const url = String(res.data?.otpauth_url || "");
    if (!mfaSecret.value || !url) {
      message.error("生成绑定信息失败");
      resetMfaForm();
      return;
    }
    mfaQRCode.value = await QRCode.toDataURL(url, { width: 200, margin: 1 });
  } catch (error) {
    message.error(error?.response?.data?.error || "生成绑定信息失败");
  } finally {
    mfaLoading.value.setup = false;
  }
};

const handleConfirmOrUnlock = async () => {
  if (mfaLoading.value.confirm) return;
  const code = String(mfaForm.value.totpCode || "").trim();
  if (!/^\d{6}$/.test(code)) {
    message.warning("请输入 6 位 2FA 验证码");
    return;
  }
  mfaLoading.value.confirm = true;
  try {
    if (admin.mfaBindRequired) {
      if (!mfaSecret.value) {
        message.warning("请先生成绑定信息");
        return;
      }
      await adminConfirmTwoFA({ code });
    }
    const res = await admin2FAUnlock({ totp_code: code });
    const token = res.data?.access_token || "";
    if (token) {
      admin.setToken(token);
    }
    admin.setMfaGateState({
      mfaRequired: false,
      mfaBindRequired: false,
      mfaUnlocked: true,
      totpEnabled: true
    });
    await admin.fetchProfile();
    resetMfaForm();
    message.success("2FA 已解锁");
  } catch (error) {
    message.error(error?.response?.data?.error || "2FA 校验失败");
  } finally {
    mfaLoading.value.confirm = false;
  }
};

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

watch(
  () => mfaModalOpen.value,
  (open) => {
    if (open) {
      resetMfaForm();
    }
  }
);

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

.twofa-setup {
  display: flex;
  gap: 16px;
  align-items: center;
  margin-bottom: 16px;
}

.twofa-qr img {
  width: 140px;
  height: 140px;
  border-radius: 8px;
  border: 1px solid rgba(0, 0, 0, 0.08);
}

.twofa-meta {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.twofa-label {
  font-size: 12px;
  color: rgba(15, 23, 42, 0.6);
}

.twofa-secret {
  font-family: "SFMono-Regular", Consolas, "Liberation Mono", Menlo, monospace;
  font-size: 13px;
  padding: 6px 10px;
  background: #f5f7fb;
  border-radius: 6px;
  word-break: break-all;
}

:deep(.admin-mfa-modal-wrap .ant-modal-content) {
  border-radius: 18px;
  padding: 0;
  overflow: hidden;
  border: 1px solid #d7e3f8;
  background:
    radial-gradient(120% 100% at 100% -10%, rgba(22, 119, 255, 0.15) 0%, rgba(22, 119, 255, 0) 60%),
    linear-gradient(180deg, #f9fcff 0%, #ffffff 56%, #f7faff 100%);
  box-shadow: 0 20px 42px rgba(12, 39, 80, 0.24);
}

:deep(.admin-mfa-modal-wrap .ant-modal-body) {
  padding: 24px;
}

.mfa-modal-head {
  display: flex;
  align-items: flex-start;
  gap: 14px;
  margin-bottom: 14px;
}

.mfa-modal-badge {
  width: 38px;
  height: 38px;
  border-radius: 12px;
  background: linear-gradient(135deg, #1677ff 0%, #3a95ff 100%);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
  box-shadow: 0 8px 20px rgba(22, 119, 255, 0.32);
}

.mfa-modal-copy {
  min-width: 0;
}

.mfa-modal-title {
  color: #1f2937;
  font-size: 30px;
  line-height: 1.1;
  font-weight: 700;
  margin-bottom: 6px;
}

.mfa-modal-subtitle {
  color: #5b6b85;
  font-size: 13px;
  line-height: 1.6;
}

.mfa-alert {
  margin-bottom: 16px;
  border-radius: 12px;
}

.mfa-form :deep(.ant-form-item-label > label) {
  color: #334155;
  font-weight: 600;
}

.mfa-form :deep(.ant-input),
.mfa-form :deep(.ant-input-password) {
  border-radius: 10px;
  min-height: 42px;
  border-color: #cdd8ea;
  background: #ffffff;
}

.mfa-form :deep(.ant-input:hover),
.mfa-form :deep(.ant-input-password:hover) {
  border-color: #9cb4db;
}

.mfa-form :deep(.ant-input:focus),
.mfa-form :deep(.ant-input-focused),
.mfa-form :deep(.ant-input-password-focused) {
  border-color: #1677ff;
  box-shadow: 0 0 0 3px rgba(22, 119, 255, 0.12);
}

.mfa-code-input {
  font-family: "SFMono-Regular", Consolas, "Liberation Mono", Menlo, monospace;
  letter-spacing: 6px;
  text-align: center;
  font-size: 20px;
  font-weight: 700;
}

.mfa-helper {
  margin-top: -4px;
  margin-bottom: 14px;
  color: #64748b;
  font-size: 12px;
}

.mfa-setup-btn {
  border-radius: 10px;
  height: 40px;
}

.mfa-confirm-btn {
  border-radius: 10px;
  height: 44px;
  font-size: 17px;
  font-weight: 700;
  letter-spacing: 3px;
}

@media (max-width: 640px) {
  :deep(.admin-mfa-modal-wrap .ant-modal) {
    max-width: calc(100vw - 20px);
  }

  :deep(.admin-mfa-modal-wrap .ant-modal-body) {
    padding: 18px;
  }

  .mfa-modal-title {
    font-size: 24px;
  }

  .mfa-code-input {
    letter-spacing: 4px;
    font-size: 18px;
  }
}
</style>
