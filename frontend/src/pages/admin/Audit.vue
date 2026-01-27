<template>
  <div class="page">
    <div class="page-header">
      <div>
        <div class="page-title">审计日志</div>
        <div class="subtle">系统操作与安全审计</div>
      </div>
    </div>

    <FilterBar
      v-model:filters="filters"
      :status-options="[]"
      @search="fetchData"
      @refresh="fetchData"
      @reset="fetchData"
      :show-export="false"
    >
      <template #advanced>
        <a-space direction="vertical" style="width: 260px">
          <a-input v-model:value="filters.action" placeholder="操作类型" />
          <a-input v-model:value="filters.user" placeholder="操作者" />
        </a-space>
      </template>
    </FilterBar>

    <ProTable :columns="columns" :data-source="dataSource" :loading="loading" :pagination="pagination" @change="onTableChange">
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'meta'">
          <span>{{ record.meta }}</span>
        </template>
      </template>
    </ProTable>
  </div>
</template>

<script setup>
import { reactive, ref } from "vue";
import FilterBar from "@/components/FilterBar.vue";
import ProTable from "@/components/ProTable.vue";
import { listAuditLogs } from "@/services/admin";

const filters = reactive({ keyword: "", status: undefined, range: [], action: "", user: "" });

const loading = ref(false);
const dataSource = ref([]);
const pagination = reactive({ current: 1, pageSize: 20, total: 0, showSizeChanger: true });

const columns = [
  { title: "ID", dataIndex: "id", key: "id" },
  { title: "操作者", dataIndex: "actor", key: "actor" },
  { title: "动作", dataIndex: "action", key: "action" },
  { title: "对象", dataIndex: "target", key: "target" },
  { title: "时间", dataIndex: "created_at", key: "created_at" },
  { title: "详情", dataIndex: "meta", key: "meta" }
];

const normalize = (row) => {
  const adminId = row.admin_id ?? row.AdminID;
  const targetType = row.target_type ?? row.TargetType;
  const targetId = row.target_id ?? row.TargetID;
  const detail = row.detail ?? row.Detail;
  let metaText = "";
  if (typeof detail === "string") {
    metaText = detail;
  } else if (detail && typeof detail === "object") {
    try {
      metaText = JSON.stringify(detail);
    } catch (e) {
      metaText = String(detail);
    }
  }
  return {
    id: row.id ?? row.ID,
    actor: row.actor ?? row.Actor ?? row.user ?? row.User ?? (adminId ? `管理员#${adminId}` : "-"),
    action: row.action ?? row.Action,
    target: row.target ?? row.Target ?? (targetType || targetId ? `${targetType || "-"}:${targetId || "-"}` : "-"),
    created_at: row.created_at ?? row.CreatedAt,
    meta: row.meta ?? row.Meta ?? metaText
  };
};

const fetchData = async () => {
  loading.value = true;
  try {
    const res = await listAuditLogs({
      limit: pagination.pageSize,
      offset: (pagination.current - 1) * pagination.pageSize
    });
    const payload = res.data || {};
    dataSource.value = (payload.items || []).map(normalize);
    pagination.total = payload.total || dataSource.value.length;
  } finally {
    loading.value = false;
  }
};

const onTableChange = (pager) => {
  pagination.current = pager.current;
  pagination.pageSize = pager.pageSize;
  fetchData();
};

fetchData();
</script>
