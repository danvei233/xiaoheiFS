<template>
  <div class="admin-tickets-page">
    <div class="page-header">
      <div>
        <div class="page-title">工单管理</div>
        <div class="page-subtitle">处理用户提交的技术支持与咨询工单</div>
      </div>
      <a-space>
        <a-button @click="resetFilters">重置筛选</a-button>
        <a-button type="primary" @click="fetchData" :loading="loading">
          <template #icon><ReloadOutlined /></template>
          刷新
        </a-button>
      </a-space>
    </div>

    <a-card :bordered="false" class="filter-card">
      <a-space wrap size="large">
        <div class="filter-item">
          <span class="filter-label">状态</span>
          <a-select
            v-model:value="filters.status"
            placeholder="全部"
            style="width: 140px"
            allow-clear
            @change="handleFilterChange"
          >
            <a-select-option value="">全部</a-select-option>
            <a-select-option value="open">待处理</a-select-option>
            <a-select-option value="waiting_user">等待回复</a-select-option>
            <a-select-option value="waiting_admin">处理中</a-select-option>
            <a-select-option value="closed">已关闭</a-select-option>
          </a-select>
        </div>
        <div class="filter-item">
          <span class="filter-label">用户 ID</span>
          <a-input
            v-model:value="filters.user_id"
            placeholder="输入用户 ID"
            style="width: 160px"
            @pressEnter="handleFilterChange"
            @blur="handleFilterChange"
          />
        </div>
        <div class="filter-item">
          <span class="filter-label">关键词</span>
          <a-input
            v-model:value="filters.q"
            placeholder="搜索标题或内容"
            style="width: 200px"
            @pressEnter="handleFilterChange"
            @blur="handleFilterChange"
          >
            <template #prefix><SearchOutlined /></template>
          </a-input>
        </div>
      </a-space>
    </a-card>

    <a-card :bordered="false" class="table-card">
      <ProTable
        :columns="columns"
        :data-source="tickets"
        :loading="loading"
        :pagination="pagination"
        @change="handleTableChange"
        row-key="id"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'subject'">
            <router-link :to="`/admin/tickets/${record.id}`" class="subject-link">
              <CommentOutlined class="subject-icon" />
              <span>{{ record.subject }}</span>
            </router-link>
          </template>
          <template v-else-if="column.key === 'status'">
            <a-tag :color="getStatusColor(record.status)" class="status-tag">
              <component :is="getStatusIcon(record.status)" class="status-icon" />
              {{ getStatusText(record.status) }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'created_at'">
            <div class="time-cell">
              <CalendarOutlined class="time-icon" />
              <span>{{ formatDate(record.created_at) }}</span>
            </div>
          </template>
          <template v-else-if="column.key === 'updated_at'">
            <div class="time-cell">
              <ClockCircleOutlined class="time-icon" />
              <span>{{ formatDate(record.updated_at) }}</span>
            </div>
          </template>
          <template v-else-if="column.key === 'actions'">
            <a-button type="link" size="small" @click="goToDetail(record.id)">
              查看详情 <ArrowRightOutlined />
            </a-button>
          </template>
        </template>
      </ProTable>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from "vue";
import { useRouter } from "vue-router";
import {
  ReloadOutlined,
  SearchOutlined,
  CommentOutlined,
  ClockCircleOutlined,
  CalendarOutlined,
  ArrowRightOutlined,
  CheckCircleOutlined,
  SyncOutlined,
  ExclamationCircleOutlined,
  StopOutlined
} from "@ant-design/icons-vue";
import ProTable from "@/components/ProTable.vue";
import { listAdminTickets } from "@/services/admin";

const router = useRouter();

const loading = ref(false);
const tickets = ref<any[]>([]);

const filters = reactive({
  status: "",
  user_id: "",
  q: ""
});

const pagination = reactive({
  current: 1,
  pageSize: 20,
  total: 0
});

const columns = [
  { title: "ID", dataIndex: "id", key: "id", width: 80 },
  { title: "用户ID", dataIndex: "user_id", key: "user_id", width: 100 },
  { title: "工单标题", dataIndex: "subject", key: "subject", ellipsis: true },
  { title: "状态", dataIndex: "status", key: "status", width: 100 },
  { title: "创建时间", dataIndex: "created_at", key: "created_at", width: 180 },
  { title: "最后回复", dataIndex: "updated_at", key: "updated_at", width: 180 },
  { title: "操作", key: "actions", width: 100 }
];

const getStatusColor = (status: string) => {
  switch (status) {
    case "open": return "blue";
    case "waiting_user": return "orange";
    case "waiting_admin": return "purple";
    case "closed": return "default";
    default: return "default";
  }
};

const getStatusIcon = (status: string) => {
  switch (status) {
    case "open": return ExclamationCircleOutlined;
    case "waiting_user": return ClockCircleOutlined;
    case "waiting_admin": return SyncOutlined;
    case "closed": return CheckCircleOutlined;
    default: return ExclamationCircleOutlined;
  }
};

const getStatusText = (status: string) => {
  switch (status) {
    case "open": return "待处理";
    case "waiting_user": return "等待回复";
    case "waiting_admin": return "处理中";
    case "closed": return "已关闭";
    default: return status;
  }
};

const formatDate = (date: string) => {
  if (!date) return "-";
  return new Date(date).toLocaleString("zh-CN");
};

const resetFilters = () => {
  filters.status = "";
  filters.user_id = "";
  filters.q = "";
  pagination.current = 1;
  fetchData();
};

const goToDetail = (id: number) => {
  router.push(`/admin/tickets/${id}`);
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
    if (filters.q) params.q = filters.q;
    const res = await listAdminTickets(params);
    tickets.value = res.data?.items || [];
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
.admin-tickets-page {
  padding: 24px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.page-title {
  font-size: 20px;
  font-weight: 600;
  margin: 0 0 4px;
}

.page-subtitle {
  color: #8c8c8c;
  font-size: 13px;
}

.filter-card {
  margin-bottom: 16px;
}

.filter-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.filter-label {
  color: #8c8c8c;
  font-size: 13px;
}

.subject-link {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--primary);
  text-decoration: none;
  font-weight: 500;
  transition: all 0.2s ease;
}

.subject-link:hover {
  text-decoration: none;
  color: #1677ff;
  transform: translateX(2px);
}

.subject-icon {
  font-size: 14px;
  color: #1677ff;
}

.status-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 12px;
}

.status-icon {
  font-size: 11px;
}

.time-cell {
  display: flex;
  align-items: center;
  gap: 6px;
  color: #595959;
  font-size: 13px;
}

.time-icon {
  font-size: 12px;
  color: #8c8c8c;
}

.table-card :deep(.ant-table) {
  font-size: 13px;
}

.table-card :deep(.ant-table-thead > tr > th) {
  font-weight: 600;
  background: #fafafa;
}

.table-card :deep(.ant-table-tbody > tr:hover) {
  background: #f5f7fa;
}
</style>
