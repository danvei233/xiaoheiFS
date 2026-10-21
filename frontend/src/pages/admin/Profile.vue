<template>
  <div class="profile-page">
    <a-row :gutter="[24, 24]">
      <!-- 左侧卡片 -->
      <a-col :xs="24" :sm="24" :md="8" :lg="6" :xl="6">
        <a-card :bordered="false" class="profile-card">
          <div class="profile-avatar-section">
            <a-avatar :src="getAvatarUrl(profile.qq)" :size="88">
              {{ profile.username?.charAt(0)?.toUpperCase() }}
            </a-avatar>
            <div class="profile-username">{{ profile.username }}</div>
            <a-tag :color="getPermissionColor()" class="profile-tag">
              {{ permissionGroupLabel }}
            </a-tag>
          </div>

          <a-divider />

          <a-descriptions :column="1" size="small">
            <a-descriptions-item>
              <template #label>
                <UserOutlined />
              </template>
              {{ getRoleLabel() }}
            </a-descriptions-item>
            <a-descriptions-item>
              <template #label>
                <CalendarOutlined />
              </template>
              {{ formatDate(profile.created_at) }}
            </a-descriptions-item>
            <a-descriptions-item>
              <template #label>
                <SafetyOutlined />
              </template>
              {{ profile.permissions?.length || 0 }} 个权限
            </a-descriptions-item>
          </a-descriptions>
        </a-card>
      </a-col>

      <!-- 右侧内容 -->
      <a-col :xs="24" :sm="24" :md="16" :lg="18" :xl="18">
        <a-card title="基本信息" :bordered="false" class="info-card">
          <template #extra>
            <a-button type="link" @click="openEditProfile">
              <EditOutlined /> 编辑
            </a-button>
          </template>
          <a-descriptions :column="{ xs: 1, sm: 1, md: 2 }">
            <a-descriptions-item label="用户名">
              {{ profile.username || '-' }}
            </a-descriptions-item>
            <a-descriptions-item label="角色">
              {{ getRoleLabel() }}
            </a-descriptions-item>
            <a-descriptions-item label="邮箱地址">
              {{ profile.email || '-' }}
            </a-descriptions-item>
            <a-descriptions-item label="QQ 号码">
              {{ profile.qq || '-' }}
            </a-descriptions-item>
            <a-descriptions-item label="权限组">
              <a-tag :color="getPermissionColor()">{{ permissionGroupLabel }}</a-tag>
            </a-descriptions-item>
            <a-descriptions-item label="注册时间">
              {{ formatDate(profile.created_at) }}
            </a-descriptions-item>
          </a-descriptions>
        </a-card>

        <a-card title="安全设置" :bordered="false" class="security-card" style="margin-top: 24px">
          <a-list :data-source="securityItems" size="large">
            <template #renderItem="{ item }">
              <a-list-item>
                <a-list-item-meta>
                  <template #avatar>
                    <a-avatar :style="{ backgroundColor: item.color }">
                      <component :is="item.icon" />
                    </a-avatar>
                  </template>
                  <template #title>
                    {{ item.title }}
                  </template>
                  <template #description>
                    {{ item.description }}
                  </template>
                </a-list-item-meta>
                <template #actions>
                  <a-button v-if="item.type === 'password'" @click="openChangePassword">
                    修改
                  </a-button>
                </template>
              </a-list-item>
            </template>
          </a-list>
        </a-card>

        <a-card title="权限列表" :bordered="false" class="permissions-card" style="margin-top: 24px">
          <template #extra>
            <SafetyCertificateOutlined />
          </template>
          <div v-if="profile.permissions?.length">
            <a-space :size="[8, 8]" wrap>
              <a-tag v-for="perm in profile.permissions" :key="perm" color="blue">
                <CheckCircleOutlined style="margin-right: 4px" />
                {{ permissionLabel(perm) }}
              </a-tag>
            </a-space>
          </div>
          <a-empty
            v-else
            description="暂无权限信息"
            :image="Empty.PRESENTED_IMAGE_SIMPLE"
          />
        </a-card>
      </a-col>
    </a-row>

    <!-- 编辑资料弹窗 -->
    <a-modal
      v-model:open="editProfileOpen"
      title="编辑资料"
      :confirm-loading="profileLoading"
      @ok="handleUpdateProfile"
      width="500px"
    >
      <a-form
        layout="vertical"
        :model="profileForm"
        ref="profileFormRef"
        style="margin-top: 24px"
      >
        <a-form-item
          label="邮箱地址"
          name="email"
          :rules="[
            { required: true, message: '请输入邮箱' },
            { type: 'email', message: '请输入有效的邮箱格式' }
          ]"
        >
          <a-input
            v-model:value.trim="profileForm.email"
            placeholder="请输入邮箱地址"
            :maxlength="INPUT_LIMITS.EMAIL"
          >
            <template #prefix>
              <MailOutlined />
            </template>
          </a-input>
        </a-form-item>
        <a-form-item
          label="QQ 号码"
          name="qq"
          :rules="[{ validator: validateQQ, trigger: 'blur' }]"
        >
          <a-input
            v-model:value.trim="profileForm.qq"
            placeholder="请输入QQ号码"
            :maxlength="INPUT_LIMITS.QQ"
          >
            <template #prefix>
              <QqOutlined />
            </template>
          </a-input>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 修改密码弹窗 -->
    <a-modal
      v-model:open="changePasswordOpen"
      title="修改密码"
      :confirm-loading="passwordLoading"
      @ok="handleChangePassword"
      width="500px"
    >
      <a-form
        layout="vertical"
        :model="passwordForm"
        ref="passwordFormRef"
        style="margin-top: 24px"
      >
        <a-form-item
          label="当前密码"
          name="old_password"
          :rules="[{ required: true, message: '请输入当前密码' }]"
        >
          <a-input-password
            v-model:value="passwordForm.old_password"
            placeholder="请输入当前密码"
            :maxlength="INPUT_LIMITS.PASSWORD"
          >
            <template #prefix>
              <KeyOutlined />
            </template>
          </a-input-password>
        </a-form-item>
        <a-form-item
          label="新密码"
          name="new_password"
          :rules="[
            { required: true, message: '请输入新密码' },
            { min: 6, message: '密码至少需要6个字符' }
          ]"
        >
          <a-input-password
            v-model:value="passwordForm.new_password"
            placeholder="请输入新密码（至少6位）"
            :maxlength="INPUT_LIMITS.PASSWORD"
          >
            <template #prefix>
              <LockOutlined />
            </template>
          </a-input-password>
        </a-form-item>
        <a-form-item
          label="确认新密码"
          name="confirm_password"
          :dependencies="['new_password']"
          :rules="[
            { required: true, message: '请确认新密码' },
            { validator: validateConfirmPassword }
          ]"
        >
          <a-input-password
            v-model:value="passwordForm.confirm_password"
            placeholder="请再次输入新密码"
            :maxlength="INPUT_LIMITS.PASSWORD"
          >
            <template #prefix>
              <SafetyOutlined />
            </template>
          </a-input-password>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup>
import { reactive, ref, onMounted, computed } from "vue";
import {
  getAdminProfile,
  updateAdminProfile,
  changeAdminPassword,
  listPermissionGroups,
  listPermissions
} from "@/services/admin";
import { message, Empty } from "ant-design-vue";
import dayjs from "dayjs";
import {
  UserOutlined,
  CalendarOutlined,
  SafetyOutlined,
  EditOutlined,
  MailOutlined,
  QqOutlined,
  LockOutlined,
  KeyOutlined,
  SafetyCertificateOutlined,
  CheckCircleOutlined
} from "@ant-design/icons-vue";
import { INPUT_LIMITS } from "@/constants/inputLimits";

const profile = ref({});
const permissionGroups = ref([]);
const allPermissions = ref([]);
const profileLoading = ref(false);
const passwordLoading = ref(false);
const passwordFormRef = ref();
const profileFormRef = ref();
const editProfileOpen = ref(false);
const changePasswordOpen = ref(false);

const profileForm = reactive({
  email: "",
  qq: ""
});

const passwordForm = reactive({
  old_password: "",
  new_password: "",
  confirm_password: ""
});

const securityItems = [
  {
    type: 'password',
    title: '登录密码',
    description: '定期修改密码可以保护账户安全',
    icon: LockOutlined,
    color: '#1890ff'
  }
];

const permissionGroupMap = computed(() => {
  const map = new Map();
  permissionGroups.value.forEach((group) => {
    if (group?.id != null) {
      map.set(Number(group.id), group.name || "-");
    }
  });
  return map;
});

const permissionLabelMap = computed(() => {
  const map = new Map();
  allPermissions.value.forEach((perm) => {
    if (!perm.code) return;
    map.set(perm.code, perm.friendly_name || perm.name || perm.code);
  });
  return map;
});

const permissionLabel = (code) => permissionLabelMap.value.get(code) || code || "-";

const permissionGroupLabel = computed(() => {
  if (profile.value?.permission_group_name) return profile.value.permission_group_name;
  const groupId = profile.value?.permission_group_id;
  if (groupId != null) {
    const name = permissionGroupMap.value.get(Number(groupId));
    if (name) return name;
  }
  return profile.value?.role || "-";
});

const getAvatarUrl = (qq) => {
  if (!qq) return "";
  return `https://q1.qlogo.cn/g?b=qq&nk=${qq}&s=100`;
};

const formatDate = (date) => {
  if (!date) return "-";
  return dayjs(date).format("YYYY-MM-DD");
};

const getPermissionColor = () => {
  const perms = profile.value?.permissions?.length || 0;
  if (perms > 20) return "purple";
  if (perms > 10) return "blue";
  if (perms > 5) return "cyan";
  return "green";
};

const getRoleLabel = () => {
  const role = profile.value?.role;
  return role === "admin" ? "管理员" : role || "未知";
};

const validateQQ = async (_rule, value) => {
  if (!value) return Promise.resolve();
  const qqNum = Number(value);
  if (!Number.isInteger(qqNum) || qqNum <= 0) {
    return Promise.reject("QQ号必须是正整数");
  }
  return Promise.resolve();
};

const validateConfirmPassword = async (_rule, value) => {
  if (!value) return Promise.resolve();
  if (value !== passwordForm.new_password) {
    return Promise.reject("两次输入的密码不一致");
  }
  return Promise.resolve();
};

const openEditProfile = () => {
  profileForm.email = profile.value.email || "";
  profileForm.qq = profile.value.qq || "";
  editProfileOpen.value = true;
};

const openChangePassword = () => {
  changePasswordOpen.value = true;
};

const fetchProfile = async () => {
  try {
    const res = await getAdminProfile();
    profile.value = res.data || {};
    profileForm.email = profile.value.email || "";
    profileForm.qq = profile.value.qq || "";
  } catch (e) {
    message.error("获取个人资料失败");
  }
};

const fetchPermissionGroups = async () => {
  try {
    const res = await listPermissionGroups();
    permissionGroups.value = (res.data?.items || []).map((g) => ({
      id: g.id ?? g.ID,
      name: g.name ?? g.Name
    }));
  } catch (e) {
    permissionGroups.value = [];
  }
};

const fetchAllPermissions = async () => {
  try {
    const res = await listPermissions();
    allPermissions.value = (res.data?.items || []).map((perm) => ({
      code: perm.code ?? perm.Code,
      name: perm.name ?? perm.Name,
      friendly_name: perm.friendly_name ?? perm.FriendlyName,
      category: perm.category ?? perm.Category,
      parent_code: perm.parent_code ?? perm.ParentCode,
      sort_order: perm.sort_order ?? perm.SortOrder
    }));
  } catch (e) {
    allPermissions.value = [];
  }
};

const handleUpdateProfile = async () => {
  try {
    await profileFormRef.value?.validate();
  } catch (e) {
    return;
  }

  profileLoading.value = true;
  try {
    await updateAdminProfile({
      email: profileForm.email,
      qq: profileForm.qq
    });
    message.success("资料已更新");
    await fetchProfile();
    editProfileOpen.value = false;
  } catch (e) {
    message.error(e.response?.data?.error || "更新失败");
  } finally {
    profileLoading.value = false;
  }
};

const handleChangePassword = async () => {
  try {
    await passwordFormRef.value?.validate();
  } catch (e) {
    return;
  }

  passwordLoading.value = true;
  try {
    await changeAdminPassword({
      old_password: passwordForm.old_password,
      new_password: passwordForm.new_password
    });
    message.success("密码已修改，请重新登录");
    passwordFormRef.value?.resetFields();
    changePasswordOpen.value = false;
  } catch (e) {
    message.error(e.response?.data?.error || "密码修改失败");
  } finally {
    passwordLoading.value = false;
  }
};

onMounted(() => {
  fetchPermissionGroups();
  fetchAllPermissions();
  fetchProfile();
});
</script>

<style scoped>
.profile-page {
  padding: 0;
}

.profile-card {
  height: fit-content;
}

.profile-avatar-section {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 16px 0;
}

.profile-username {
  font-size: 18px;
  font-weight: 600;
  color: rgba(0, 0, 0, 0.85);
  margin-top: 16px;
  margin-bottom: 8px;
}

.profile-tag {
  font-size: 13px;
}

.info-card :deep(.ant-card-head-title),
.security-card :deep(.ant-card-head-title),
.permissions-card :deep(.ant-card-head-title) {
  font-weight: 600;
}

.info-card :deep(.ant-card-extra),
.security-card :deep(.ant-card-extra),
.permissions-card :deep(.ant-card-extra) {
  font-size: 16px;
  color: rgba(0, 0, 0, 0.45);
}

.security-card :deep(.ant-list-item-meta-avatar) {
  margin-right: 16px;
}

.security-card :deep(.ant-list-item-meta-title) {
  font-weight: 500;
}
</style>
