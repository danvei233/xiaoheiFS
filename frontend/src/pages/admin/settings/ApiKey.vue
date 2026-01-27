<template>
  <div class="page">
    <div class="page-header">
      <div>
        <div class="page-title">API Keys</div>
        <div class="subtle">创建与管理 API Key</div>
      </div>
      <div class="page-header-actions">
        <a-button type="primary" @click="openCreate">创建 API Key</a-button>
      </div>
    </div>

    <FilterBar
      v-model:filters="filters"
      :status-options="statusOptions"
      @search="fetchData"
      @refresh="fetchData"
      @reset="fetchData"
      :show-export="false"
    />
    <ProTable
      :columns="columns"
      :data-source="dataSource"
      :loading="loading"
      :pagination="pagination"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'action'">
          <a-space>
            <a-button size="small" @click="copy(record)">复制</a-button>
            <ConfirmAction title="确认切换状态?" @confirm="toggle(record)">
              <a-button size="small">切换状态</a-button>
            </ConfirmAction>
          </a-space>
        </template>
      </template>
    </ProTable>

    <a-modal v-model:open="createOpen" title="创建 API Key" @ok="create">
      <a-form layout="vertical">
        <a-form-item label="名称"><a-input v-model:value="createForm.name" /></a-form-item>
        <a-form-item label="权限组">
          <a-select
            v-model:value="createForm.permission_group_id"
            placeholder="请选择权限组"
            :options="permissionGroupOptions"
            style="width: 100%"
          />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup>
import { computed, reactive, ref } from "vue";
import FilterBar from "@/components/FilterBar.vue";
import ProTable from "@/components/ProTable.vue";
import ConfirmAction from "@/components/ConfirmAction.vue";
import { listApiKeys, updateApiKeyStatus, createApiKey, listPermissionGroups } from "@/services/admin";
import { message, Modal } from "ant-design-vue";

const filters = reactive({ keyword: "", status: undefined, range: [] });
const statusOptions = [
  { label: "active", value: "active" },
  { label: "disabled", value: "disabled" }
];

const loading = ref(false);
const dataSource = ref([]);
const pagination = reactive({ current: 1, pageSize: 20, total: 0, showSizeChanger: true });

const createOpen = ref(false);
const createForm = reactive({ name: "", permission_group_id: null });

const permissionGroups = ref([]);
const permissionGroupMap = computed(() => {
  const map = new Map();
  permissionGroups.value.forEach((g) => {
    const id = g.id ?? g.ID;
    const name = g.name ?? g.Name;
    if (id != null) map.set(Number(id), name || "");
  });
  return map;
});

const permissionGroupOptions = computed(() =>
  permissionGroups.value
    .map((g) => ({
      label: g.name ?? g.Name,
      value: g.id ?? g.ID
    }))
    .filter((x) => x.value != null)
);

const columns = [
  { title: "ID", dataIndex: "id", key: "id" },
  { title: "名称", dataIndex: "name", key: "name" },
  { title: "权限组", dataIndex: "permission_group_name", key: "permission_group_name" },
  { title: "Key Hash", dataIndex: "key_hash", key: "key_hash" },
  { title: "状态", dataIndex: "status", key: "status" },
  { title: "操作", key: "action" }
];

const normalizeKey = (row) => ({
  id: row.id ?? row.ID,
  name: row.name ?? row.Name,
  key_hash: row.key_hash ?? row.KeyHash,
  status: row.status ?? row.Status,
  permission_group_id: row.permission_group_id ?? row.PermissionGroupID
});

const fetchData = async () => {
  loading.value = true;
  try {
    const res = await listApiKeys({
      limit: pagination.pageSize,
      offset: (pagination.current - 1) * pagination.pageSize
    });
    const payload = res.data || {};
    let items = (payload.items || []).map((row) => {
      const item = normalizeKey(row);
      const gid = item.permission_group_id != null ? Number(item.permission_group_id) : null;
      item.permission_group_name = gid ? permissionGroupMap.value.get(gid) || "" : "";
      return item;
    });
    if (filters.keyword) {
      items = items.filter((item) => item.key_hash?.includes(filters.keyword));
    }
    if (filters.status) {
      items = items.filter((item) => item.status === filters.status);
    }
    dataSource.value = items;
    pagination.total = payload.total || items.length;
  } finally {
    loading.value = false;
  }
};

const toggle = async (record) => {
  const status = record.status === "active" ? "disabled" : "active";
  await updateApiKeyStatus(record.id, { status });
  message.success("已更新");
  fetchData();
};

const openCreate = () => {
  createForm.name = "";
  createForm.permission_group_id = null;
  createOpen.value = true;
};

const create = async () => {
  if (!createForm.permission_group_id) {
    message.error("请选择权限组");
    return;
  }
  const res = await createApiKey({ name: createForm.name, permission_group_id: createForm.permission_group_id });
  const apiKey = res.data?.api_key;
  Modal.info({ title: "API Key", content: apiKey || "创建成功" });
  createOpen.value = false;
  fetchData();
};

const copy = async (record) => {
  await navigator.clipboard.writeText(record.key_hash || "");
  message.success("已复制");
};

const fetchPermissionGroups = async () => {
  try {
    const res = await listPermissionGroups();
    permissionGroups.value = res.data?.items || [];
  } catch {
    // ignore
  }
};

fetchPermissionGroups();
fetchData();
</script>
