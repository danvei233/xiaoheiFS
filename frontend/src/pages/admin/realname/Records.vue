<template>
  <div class="realname-records-page">
    <div class="page-header">
      <h1 class="page-title">实名认证记录</h1>
      <a-button @click="fetchData" :loading="loading">
        <template #icon><ReloadOutlined /></template>
        刷新
      </a-button>
    </div>

    <a-card :bordered="false">
      <ProTable
        :columns="columns"
        :data-source="records"
        :loading="loading"
        :pagination="pagination"
        @change="handleTableChange"
        row-key="id"
      >
        <template #toolbar>
          <a-input
            v-model:value="filters.user_id"
            placeholder="用户ID"
            style="width: 120px"
            @change="handleFilterChange"
          />
          <a-select
            v-model:value="filters.status"
            placeholder="状态筛选"
            style="width: 120px"
            allow-clear
            @change="handleFilterChange"
          >
            <a-select-option value="">全部</a-select-option>
            <a-select-option value="pending">待审核</a-select-option>
            <a-select-option value="verified">已通过</a-select-option>
            <a-select-option value="failed">未通过</a-select-option>
          </a-select>
        </template>

        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'real_name'">
            {{ maskName(record.real_name) }}
          </template>
          <template v-else-if="column.key === 'id_number'">
            {{ maskIdNumber(record.id_number) }}
          </template>
          <template v-else-if="column.key === 'status'">
            <a-tag :color="getStatusColor(record.status)">
              {{ getStatusText(record.status) }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'created_at'">
            {{ formatDate(record.created_at) }}
          </template>
          <template v-else-if="column.key === 'verified_at'">
            {{ formatDate(record.verified_at) }}
          </template>
        </template>
      </ProTable>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from "vue";
import { ReloadOutlined } from "@ant-design/icons-vue";
import ProTable from "@/components/ProTable.vue";
import { listRealNameRecords } from "@/services/admin";

const loading = ref(false);
const records = ref<any[]>([]);

const filters = reactive({
  user_id: "",
  status: ""
});

const pagination = reactive({
  current: 1,
  pageSize: 20,
  total: 0
});

const columns = [
  { title: "ID", dataIndex: "id", key: "id", width: 80 },
  { title: "用户ID", dataIndex: "user_id", key: "user_id", width: 100 },
  { title: "真实姓名", dataIndex: "real_name", key: "real_name", width: 120 },
  { title: "证件号码", dataIndex: "id_number", key: "id_number", width: 180 },
  { title: "服务商", dataIndex: "provider", key: "provider", width: 100 },
  { title: "状态", dataIndex: "status", key: "status", width: 100 },
  { title: "提交时间", dataIndex: "created_at", key: "created_at", width: 180 },
  { title: "审核时间", dataIndex: "verified_at", key: "verified_at", width: 180 }
];

const maskName = (name: string) => {
  if (!name) return "";
  if (name.length <= 2) return name.charAt(0) + "*";
  return name.charAt(0) + "*".repeat(name.length - 2) + name.charAt(name.length - 1);
};

const maskIdNumber = (idNumber: string) => {
  if (!idNumber || idNumber.length < 8) return idNumber;
  return idNumber.substring(0, 4) + "********" + idNumber.substring(idNumber.length - 4);
};

const getStatusColor = (status: string) => {
  switch (status) {
    case "pending": return "orange";
    case "verified": return "success";
    case "failed": return "error";
    default: return "default";
  }
};

const getStatusText = (status: string) => {
  switch (status) {
    case "pending": return "待审核";
    case "verified": return "已通过";
    case "failed": return "未通过";
    default: return status;
  }
};

const formatDate = (date: string) => {
  if (!date) return "-";
  return new Date(date).toLocaleString("zh-CN");
};

const fetchData = async () => {
  loading.value = true;
  try {
    const params: any = {
      limit: pagination.pageSize,
      offset: (pagination.current - 1) * pagination.pageSize
    };
    if (filters.status) params.status = filters.status;
    if (filters.user_id) params.user_id = filters.user_id;
    const res = await listRealNameRecords(params);
    records.value = res.data?.items || [];
    pagination.total = res.data?.total || 0;
  } finally {
    loading.value = false;
  }
};

const handleTableChange = (pag: any) => {
  pagination.current = pag.current;
  fetchData();
};

const handleFilterChange = () => {
  pagination.current = 1;
  fetchData();
};

onMounted(() => {
  fetchData();
});
</script>

<style scoped>
.realname-records-page {
  padding: 24px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.page-title {
  font-size: 20px;
  font-weight: 600;
  margin: 0;
}
</style>
