<template>
  <div class="page">
    <div class="page-header">
      <div>
        <div class="page-title">权限组管理</div>
        <div class="subtle">管理系统权限组与权限分配</div>
      </div>
      <div class="page-header-actions">
        <a-button type="primary" @click="openCreate">创建权限组</a-button>
      </div>
    </div>

    <ProTable
      :columns="columns"
      :data-source="dataSource"
      :loading="loading"
      :pagination="false"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'permissions'">
          <a-tooltip v-if="record.permissions?.length" :title="getPermissionsTooltip(record.permissions)">
            <div class="permissions-cell">
              <a-tag v-for="perm in displayPermissions(record.permissions)" :key="perm" style="margin-bottom: 4px">
                {{ permissionLabel(perm) }}
              </a-tag>
              <a-tag v-if="record.permissions.length > maxDisplayPermissions" style="margin-bottom: 4px">
                +{{ record.permissions.length - maxDisplayPermissions }} 更多
              </a-tag>
            </div>
          </a-tooltip>
          <span v-else class="subtle">-</span>
        </template>
        <template v-else-if="column.key === 'action'">
          <a-space>
            <a-button type="link" @click="openEdit(record)">编辑</a-button>
            <a-button type="link" danger @click="handleDelete(record)">删除</a-button>
          </a-space>
        </template>
      </template>
    </ProTable>

    <a-modal v-model:open="formOpen" :title="isEdit ? '编辑权限组' : '创建权限组'" width="700px" @ok="handleSubmit">
      <a-form layout="vertical">
        <a-form-item label="名称">
          <a-input v-model:value="form.name" placeholder="请输入权限组名称" />
        </a-form-item>
        <a-form-item label="描述">
          <a-textarea v-model:value="form.description" placeholder="请输入权限组描述" :rows="3" />
        </a-form-item>
        <a-form-item label="权限">
          <div class="permission-search">
            <a-input
              v-model:value="permissionSearch"
              placeholder="搜索权限..."
              allow-clear
              @input="filterPermissions"
            />
          </div>
          <div class="permission-list">
            <a-collapse v-model:activeKey="activeGroups" ghost>
              <a-collapse-panel v-for="(permissions, category) in groupedPermissions" :key="category" :header="category">
                <a-checkbox-group v-model:value="form.permissions">
                  <div class="permission-item" v-for="perm in permissions" :key="perm.code">
                    <a-checkbox :value="perm.code">{{ permissionLabel(perm.code) }}</a-checkbox>
                  </div>
                </a-checkbox-group>
              </a-collapse-panel>
            </a-collapse>
          </div>
          <div class="permission-actions">
            <a-button size="small" @click="selectAll">全选</a-button>
            <a-button size="small" @click="clearAll" style="margin-left: 8px">清空</a-button>
          </div>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup>
import { reactive, ref, onMounted, computed, watch } from "vue";
import ProTable from "@/components/ProTable.vue";
import {
  listPermissionGroups,
  createPermissionGroup,
  updatePermissionGroup,
  deletePermissionGroup,
  listPermissions
} from "@/services/admin";
import { message, Modal } from "ant-design-vue";

const loading = ref(false);
const dataSource = ref([]);
const allPermissions = ref([]);
const permissionSearch = ref("");
const formOpen = ref(false);
const isEdit = ref(false);
const activeGroups = ref([]);
const maxDisplayPermissions = 5;

const form = reactive({
  id: null,
  name: "",
  description: "",
  permissions: []
});

const columns = [
  { title: "ID", dataIndex: "id", key: "id", width: 80 },
  { title: "名称", dataIndex: "name", key: "name" },
  { title: "描述", dataIndex: "description", key: "description" },
  { title: "权限", key: "permissions" },
  { title: "操作", key: "action", width: 150 }
];

const permissionLabelMap = computed(() => {
  const map = new Map();
  allPermissions.value.forEach((perm) => {
    if (!perm.code) return;
    map.set(perm.code, perm.friendly_name || perm.name || perm.code);
  });
  return map;
});

const permissionLabel = (code) => permissionLabelMap.value.get(code) || code || "-";

const displayPermissions = (permissions) => {
  if (!permissions || permissions.length <= maxDisplayPermissions) {
    return permissions;
  }
  return permissions.slice(0, maxDisplayPermissions);
};

const getPermissionsTooltip = (permissions) => {
  return permissions.map(perm => permissionLabel(perm)).join('、');
};

const filteredPermissions = computed(() => {
  if (!permissionSearch.value) {
    return allPermissions.value;
  }
  const search = permissionSearch.value.trim().toLowerCase();
  if (!search) {
    return allPermissions.value;
  }
  return allPermissions.value.filter((perm) => {
    return [perm.code, perm.name, perm.friendly_name, perm.category]
      .filter(Boolean)
      .some((val) => val.toLowerCase().includes(search));
  });
});

const groupedPermissions = computed(() => {
  const groups = {};
  filteredPermissions.value.forEach((perm) => {
    const category = perm.category || "其他";
    if (!groups[category]) {
      groups[category] = [];
    }
    groups[category].push(perm);
  });
  return groups;
});

const syncActiveGroups = () => {
  activeGroups.value = Object.keys(groupedPermissions.value);
};

const filterPermissions = () => {
  if (permissionSearch.value) {
    syncActiveGroups();
  }
};

const parsePermissions = (value) => {
  if (Array.isArray(value)) {
    return value;
  }
  if (typeof value === "string") {
    try {
      const parsed = JSON.parse(value);
      return Array.isArray(parsed) ? parsed : [];
    } catch (e) {
      return [];
    }
  }
  return [];
};

const normalize = (row) => {
  const rawPermissions =
    row.permissions ?? row.Permissions ?? row.permissions_json ?? row.PermissionsJSON;
  return {
    id: row.id ?? row.ID,
    name: row.name ?? row.Name,
    description: row.description ?? row.Description,
    permissions: parsePermissions(rawPermissions)
  };
};

const fetchData = async () => {
  loading.value = true;
  try {
    const res = await listPermissionGroups();
    dataSource.value = (res.data?.items || []).map(normalize);
  } finally {
    loading.value = false;
  }
};

const fetchAllPermissions = async () => {
  try {
    const res = await listPermissions();
    const items = res.data?.items || [];
    allPermissions.value = items
      .map((perm) => ({
        code: perm.code ?? perm.Code,
        name: perm.name ?? perm.Name,
        friendly_name: perm.friendly_name ?? perm.FriendlyName,
        category: perm.category ?? perm.Category,
        parent_code: perm.parent_code ?? perm.ParentCode,
        sort_order: perm.sort_order ?? perm.SortOrder
      }))
      .sort((a, b) => {
        const catA = a.category || "";
        const catB = b.category || "";
        if (catA !== catB) {
          return catA.localeCompare(catB);
        }
        const orderA = Number(a.sort_order ?? 0);
        const orderB = Number(b.sort_order ?? 0);
        if (orderA !== orderB) {
          return orderA - orderB;
        }
        return (a.code || "").localeCompare(b.code || "");
      });
  } catch (e) {
    console.error("Failed to fetch permissions:", e);
  }
};

const openCreate = () => {
  Object.assign(form, { id: null, name: "", description: "", permissions: [] });
  permissionSearch.value = "";
  isEdit.value = false;
  formOpen.value = true;
  syncActiveGroups();
};

const openEdit = (record) => {
  Object.assign(form, {
    id: record.id,
    name: record.name,
    description: record.description,
    permissions: Array.isArray(record.permissions) ? [...record.permissions] : []
  });
  permissionSearch.value = "";
  isEdit.value = true;
  formOpen.value = true;
  syncActiveGroups();
};

const selectAll = () => {
  const codes = allPermissions.value.map((perm) => perm.code).filter(Boolean);
  form.permissions = Array.from(new Set(codes));
};

const clearAll = () => {
  form.permissions = [];
};

const handleSubmit = async () => {
  if (!form.name.trim()) {
    message.warning("请输入权限组名称");
    return;
  }
  if (!form.permissions.length) {
    message.warning("请至少选择一个权限");
    return;
  }
  try {
    const payload = {
      name: form.name,
      description: form.description,
      permissions: form.permissions
    };

    if (!isEdit.value) {
      await createPermissionGroup(payload);
      message.success("权限组已创建");
    } else {
      await updatePermissionGroup(form.id, payload);
      message.success("权限组已更新");
    }

    formOpen.value = false;
    fetchData();
  } catch (e) {
    message.error(e.response?.data?.error || "操作失败");
  }
};

const handleDelete = (record) => {
  Modal.confirm({
    title: "确认删除",
    content: `确定要删除权限组 "${record.name}" 吗？`,
    onOk: async () => {
      try {
        await deletePermissionGroup(record.id);
        message.success("已删除");
        fetchData();
      } catch (e) {
        message.error(e.response?.data?.error || "删除失败");
      }
    }
  });
};

watch(
  () => groupedPermissions.value,
  () => {
    if (formOpen.value) {
      syncActiveGroups();
    }
  }
);

onMounted(() => {
  fetchData();
  fetchAllPermissions();
});
</script>

<style scoped>
.permission-search {
  margin-bottom: 12px;
}

.permission-list {
  max-height: 300px;
  overflow-y: auto;
  border: 1px solid #e6e8ec;
  border-radius: 4px;
  padding: 8px;
  background: #f9f9f9;
}

.permission-item {
  padding: 4px 0;
  border-bottom: 1px solid #eee;
}

.permission-item:last-child {
  border-bottom: none;
}

.permission-actions {
  margin-top: 8px;
}

.permissions-cell {
  max-height: 120px;
  overflow: hidden;
  line-height: 1.5;
}
</style>
