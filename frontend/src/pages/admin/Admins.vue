<template>
  <div class="page">
    <div class="page-header">
      <div>
        <div class="page-title">管理员管理</div>
        <div class="subtle">管理系统管理员账号与权限</div>
      </div>
      <div class="page-header-actions">
        <a-button type="primary" @click="openCreate">创建管理员</a-button>
      </div>
    </div>

    <ProTable
      :columns="columns"
      :data-source="dataSource"
      :loading="loading"
      :pagination="pagination"
      @change="onTableChange"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'avatar'">
          <a-avatar :src="getAvatarUrl(record.qq)" :size="40">
            {{ record.username?.charAt(0)?.toUpperCase() }}
          </a-avatar>
        </template>
        <template v-else-if="column.key === 'permission_group'">
          <a-tag color="blue">{{ record.permission_group_name || '-' }}</a-tag>
        </template>
        <template v-else-if="column.key === 'status'">
          <a-tag :color="record.status === 'active' ? 'success' : 'default'">
            {{ record.status === 'active' ? '启用' : '禁用' }}
          </a-tag>
        </template>
        <template v-else-if="column.key === 'action'">
          <a-space>
            <a-button type="link" @click="openEdit(record)">编辑</a-button>
            <a-button
              v-if="record.id !== currentAdminId"
              type="link"
              @click="toggleAdminStatus(record)"
            >
              {{ record.status === 'active' ? '禁用' : '启用' }}
            </a-button>
          </a-space>
        </template>
      </template>
    </ProTable>

    <a-modal v-model:open="formOpen" :title="isEdit ? '编辑管理员' : '创建管理员'" @ok="handleSubmit">
      <a-form ref="formRef" :model="form" :rules="formRules" layout="vertical">
        <a-form-item label="用户名" name="username">
          <a-input v-model:value="form.username" placeholder="请输入用户名" />
        </a-form-item>
        <a-form-item label="邮箱" name="email">
          <a-input v-model:value="form.email" placeholder="请输入邮箱" />
        </a-form-item>
        <a-form-item label="QQ" name="qq">
          <a-input v-model:value="form.qq" placeholder="请输入QQ号" />
        </a-form-item>
        <a-form-item v-if="!isEdit" label="密码" name="password">
          <a-input-password v-model:value="form.password" placeholder="请输入密码" />
        </a-form-item>
        <a-form-item label="权限组" name="permission_group_id">
          <a-select v-model:value="form.permission_group_id" placeholder="请选择权限组">
            <a-select-option v-for="group in permissionGroups" :key="group.id" :value="group.id">
              {{ group.name }}
            </a-select-option>
          </a-select>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup>
import { reactive, ref, onMounted, computed, watch } from "vue";
import ProTable from "@/components/ProTable.vue";
import { listAdmins, createAdmin, updateAdmin, updateAdminStatus, listPermissionGroups } from "@/services/admin";
import { useAdminAuthStore } from "@/stores/adminAuth";
import { message, Modal } from "ant-design-vue";

const adminAuth = useAdminAuthStore();
const currentAdminId = ref(adminAuth.user?.id);

const loading = ref(false);
const dataSource = ref([]);
const pagination = reactive({ current: 1, pageSize: 20, total: 0 });
const formOpen = ref(false);
const isEdit = ref(false);
const permissionGroups = ref([]);

const permissionGroupMap = computed(() => {
  const map = new Map();
  permissionGroups.value.forEach((group) => {
    if (group.id != null) {
      map.set(Number(group.id), group.name || "-");
    }
  });
  return map;
});

const form = reactive({
  id: null,
  username: "",
  email: "",
  qq: "",
  password: "",
  permission_group_id: null
});

const formRules = {
  username: [{ required: true, message: "请输入用户名", trigger: "blur" }],
  email: [
    { required: true, message: "请输入邮箱", trigger: "blur" },
    { type: "email", message: "请输入有效的邮箱地址", trigger: "blur" }
  ],
  qq: [
    {
      validator: (rule, value) => {
        if (value && value.trim() !== "") {
          const qqNum = Number(value);
          if (!Number.isInteger(qqNum) || qqNum <= 0) {
            return Promise.reject("QQ号必须是正整数");
          }
        }
        return Promise.resolve();
      },
      trigger: "blur"
    }
  ],
  password: [{ required: true, message: "请输入密码", trigger: "blur" }],
  permission_group_id: [{ required: true, message: "请选择权限组", trigger: "change" }]
};

const formRef = ref();

const toNumberOrNull = (value) => {
  if (value === null || value === undefined || value === "") return null;
  const num = Number(value);
  return Number.isNaN(num) ? null : num;
};

const columns = [
  { title: "头像", key: "avatar", width: 80 },
  { title: "用户名", dataIndex: "username", key: "username" },
  { title: "邮箱", dataIndex: "email", key: "email" },
  { title: "QQ", dataIndex: "qq", key: "qq" },
  { title: "权限组", key: "permission_group" },
  { title: "状态", dataIndex: "status", key: "status", width: 100 },
  { title: "创建时间", dataIndex: "created_at", key: "created_at" },
  { title: "操作", key: "action", width: 150 }
];

const normalize = (row) => {
  const permissionGroupId = row.permission_group_id ?? row.PermissionGroupID;
  const permissionGroupName =
    row.permission_group_name ??
    row.PermissionGroupName ??
    permissionGroupMap.value.get(Number(permissionGroupId));
  return {
    id: row.id ?? row.ID,
    username: row.username ?? row.Username,
    email: row.email ?? row.Email,
    qq: row.qq ?? row.QQ,
    avatar: row.avatar ?? row.Avatar,
    status: row.status ?? row.Status,
    permission_group_id: permissionGroupId,
    permission_group_name: permissionGroupName,
    created_at: row.created_at ?? row.CreatedAt
  };
};

const getAvatarUrl = (qq) => {
  if (!qq) return "";
  return `https://q1.qlogo.cn/g?b=qq&nk=${qq}&s=100`;
};

const fetchData = async () => {
  loading.value = true;
  try {
    const res = await listAdmins({
      limit: pagination.pageSize,
      offset: (pagination.current - 1) * pagination.pageSize,
      status: "all"
    });
    const payload = res.data || {};
    dataSource.value = (payload.items || []).map(normalize);
    pagination.total = payload.total || dataSource.value.length;
  } finally {
    loading.value = false;
  }
};

const fetchPermissionGroups = async () => {
  try {
    const res = await listPermissionGroups();
    permissionGroups.value = (res.data?.items || []).map(g => ({
      id: toNumberOrNull(g.id ?? g.ID),
      name: g.name ?? g.Name
    }));
  } catch (e) {
    console.error("Failed to fetch permission groups:", e);
  }
};

watch(permissionGroupMap, () => {
  if (!dataSource.value.length) return;
  dataSource.value = dataSource.value.map((row) => ({
    ...row,
    permission_group_name: row.permission_group_name || permissionGroupMap.value.get(Number(row.permission_group_id))
  }));
});

const onTableChange = (pager) => {
  pagination.current = pager.current;
  pagination.pageSize = pager.pageSize;
  fetchData();
};

const openCreate = () => {
  Object.assign(form, { id: null, username: "", email: "", qq: "", password: "", permission_group_id: null });
  isEdit.value = false;
  formOpen.value = true;
  formRef.value?.clearValidate();
};

const openEdit = (record) => {
  Object.assign(form, {
    id: record.id,
    username: record.username,
    email: record.email,
    qq: record.qq,
    password: "",
    permission_group_id: toNumberOrNull(record.permission_group_id)
  });
  isEdit.value = true;
  formOpen.value = true;
  formRef.value?.clearValidate();
};

const handleSubmit = async () => {
  try {
    await formRef.value?.validate();
  } catch (e) {
    return;
  }

  try {
    const payload = {
      username: form.username,
      email: form.email,
      qq: form.qq,
      permission_group_id: toNumberOrNull(form.permission_group_id)
    };

    if (!isEdit.value) {
      payload.password = form.password;
      await createAdmin(payload);
      message.success("管理员已创建");
    } else {
      await updateAdmin(form.id, payload);
      message.success("管理员已更新");
    }

    formOpen.value = false;
    fetchData();
  } catch (e) {
    message.error(e.response?.data?.error || "操作失败");
  }
};

const toggleAdminStatus = (record) => {
  const nextStatus = record.status === "active" ? "disabled" : "active";
  Modal.confirm({
    title: nextStatus === "disabled" ? "确认禁用" : "确认启用",
    content: `确定要${nextStatus === "disabled" ? "禁用" : "启用"}管理员 "${record.username}" 吗？`,
    onOk: async () => {
      try {
        await updateAdminStatus(record.id, { status: nextStatus });
        message.success("已更新管理员状态");
        fetchData();
      } catch (e) {
        message.error(e.response?.data?.error || "更新失败");
      }
    }
  });
};

onMounted(() => {
  fetchData();
  fetchPermissionGroups();
});
</script>


