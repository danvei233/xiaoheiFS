<template>
  <div class="tickets-page">
    <!-- Header -->
    <div class="page-header">
      <div class="header-content">
        <h1 class="page-title">我的工单</h1>
        <p class="page-subtitle">技术支持与问题反馈</p>
      </div>
      <div class="header-actions">
        <a-button type="primary" @click="showCreateModal">
          <PlusOutlined />
          新建工单
        </a-button>
      </div>
    </div>

    <!-- Status Tabs -->
    <div class="status-tabs">
      <a-segmented
        v-model:value="activeTab"
        :options="tabOptions"
        @change="handleTabChange"
        size="large"
      />
    </div>

    <!-- Table -->
    <div class="table-section">
      <a-table
        :columns="columns"
        :data-source="tickets"
        :loading="loading"
        :scroll="{ x: 900 }"
        :pagination="{
          current: pagination.current,
          pageSize: pagination.pageSize,
          total: pagination.total,
          showSizeChanger: true,
          showTotal: (total) => `共 ${total} 条`
        }"
        row-key="id"
        @change="handleTableChange"
        class="tickets-table"
      >
        <template #bodyCell="{ column, record }">
          <!-- Subject -->
          <template v-if="column.key === 'subject'">
            <div class="subject-cell">
              <CustomerServiceOutlined class="subject-icon" />
              <div class="subject-content">
                <a-tag size="small" class="ticket-id">#{{ record.id }}</a-tag>
                <span class="subject-text">{{ record.subject }}</span>
              </div>
            </div>
          </template>

          <!-- Status -->
          <template v-else-if="column.key === 'status'">
            <a-tag v-if="record.status === 'open'" color="processing">
              <ClockCircleOutlined />
              待处理
            </a-tag>
            <a-tag v-else-if="record.status === 'waiting_user'" color="warning">
              <ExclamationCircleOutlined />
              等待回复
            </a-tag>
            <a-tag v-else-if="record.status === 'waiting_admin'" color="purple">
              <SyncOutlined spin />
              处理中
            </a-tag>
            <a-tag v-else-if="record.status === 'closed'" color="default">
              <CheckCircleOutlined />
              已关闭
            </a-tag>
            <a-tag v-else>
              {{ record.status }}
            </a-tag>
          </template>

          <!-- Last Message -->
          <template v-else-if="column.key === 'last_message'">
            <div class="message-cell">
              <CommentOutlined class="message-icon" />
              <span class="message-text">{{ record.last_message || '-' }}</span>
            </div>
          </template>

          <!-- Resources -->
          <template v-else-if="column.key === 'resources'">
            <div class="resources-cell">
              <CloudServerOutlined class="resource-icon" />
              <span>{{ record.resource_count || 0 }}</span>
            </div>
          </template>

          <!-- Created At -->
          <template v-else-if="column.key === 'created_at'">
            <div class="time-cell">
              <CalendarOutlined class="time-icon" />
              <span>{{ formatDate(record.created_at) }}</span>
            </div>
          </template>

          <!-- Actions -->
          <template v-else-if="column.key === 'actions'">
            <a-button type="link" @click="router.push(`/console/tickets/${record.id}`)">
              <EyeOutlined />
              查看
            </a-button>
          </template>
        </template>
      </a-table>
    </div>

    <!-- Create Modal -->
    <a-modal
      v-model:open="createModalVisible"
      title="新建工单"
      :confirm-loading="creating"
      @ok="handleCreate"
      @cancel="resetForm"
      width="540px"
    >
      <a-form ref="formRef" :model="form" layout="vertical">
        <a-form-item label="标题" name="subject" :rules="[{ required: true, message: '请输入工单标题' }]">
          <a-input
            v-model:value="form.subject"
            placeholder="简要描述您的问题"
            size="large"
          />
        </a-form-item>

        <a-form-item label="描述" name="content" :rules="[{ required: true, message: '请输入问题描述' }]">
          <a-textarea
            v-model:value="form.content"
            placeholder="详细描述您的问题"
            :rows="5"
          />
        </a-form-item>

        <a-form-item label="关联资源" name="resources">
          <a-select
            v-model:value="form.resources"
            mode="multiple"
            placeholder="选择相关 VPS 实例"
            :filter-option="filterOption"
            show-search
            allow-clear
            :options="vpsOptions"
            size="large"
          >
            <template #option="{ label, region, cpu, memory, disk, bandwidth }">
              <div class="vps-option">
                <div class="vps-option-header">
                  <span class="vps-option-name">{{ label }}</span>
                  <a-tag size="small">{{ region }}</a-tag>
                </div>
                <div class="vps-option-specs">
                  <span>{{ cpu }}核</span>
                  <span>{{ memory }}GB</span>
                  <span>{{ disk }}GB</span>
                  <span>{{ bandwidth }}Mbps</span>
                </div>
              </div>
            </template>
          </a-select>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed } from "vue";
import { useRouter } from "vue-router";
import { message } from "ant-design-vue";
import {
  PlusOutlined,
  CloudServerOutlined,
  CustomerServiceOutlined,
  ClockCircleOutlined,
  ExclamationCircleOutlined,
  SyncOutlined,
  CheckCircleOutlined,
  CommentOutlined,
  CalendarOutlined,
  EyeOutlined
} from "@ant-design/icons-vue";
import { listTickets, createTicket, listVps } from "@/services/user";

const router = useRouter();
const formRef = ref();

const loading = ref(false);
const creating = ref(false);
const createModalVisible = ref(false);
const activeTab = ref("all");

const tickets = ref<any[]>([]);
const vpsList = ref<any[]>([]);

const pagination = reactive({
  current: 1,
  pageSize: 10,
  total: 0
});

const form = ref({
  subject: "",
  content: "",
  resources: [] as number[]
});

const columns = [
  { title: '工单', dataIndex: 'subject', key: 'subject', width: 280 },
  { title: '状态', dataIndex: 'status', key: 'status', width: 140 },
  { title: '最后回复', dataIndex: 'last_message', key: 'last_message', width: 280, ellipsis: true },
  { title: '关联资源', dataIndex: 'resources', key: 'resources', width: 100 },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 180 },
  { title: '操作', key: 'actions', width: 100 }
];

const tabOptions = computed(() => [
  { label: '全部', value: 'all' },
  { label: '待处理', value: 'open' },
  { label: '处理中', value: 'waiting_admin' },
  { label: '已关闭', value: 'closed' }
]);

const vpsOptions = computed(() =>
  vpsList.value.map(v => ({
    label: v.name,
    value: v.id,
    region: v.region || '-',
    cpu: v.cpu || 0,
    memory: v.memory_gb || 0,
    disk: v.disk_gb || 0,
    bandwidth: v.bandwidth_mbps || 0
  }))
);

const filterOption = (input: string, option: any) => {
  const searchText = input.toLowerCase();
  return (
    option.label?.toLowerCase().includes(searchText) ||
    option.region?.toLowerCase().includes(searchText)
  );
};

const formatDate = (date: string) => {
  if (!date) return '-';
  return new Date(date).toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  });
};

const fetchTickets = async () => {
  loading.value = true;
  try {
    const params: any = {
      limit: pagination.pageSize,
      offset: (pagination.current - 1) * pagination.pageSize
    };
    if (activeTab.value !== 'all') {
      params.status = activeTab.value;
    }
    const res = await listTickets(params);
    tickets.value = res.data?.items || [];
    pagination.total = res.data?.total || 0;
  } finally {
    loading.value = false;
  }
};

const fetchVps = async () => {
  try {
    const res = await listVps();
    vpsList.value = res.data?.items || [];
  } catch (error) {
    console.error('Failed to fetch VPS list:', error);
  }
};

const handleTabChange = (value: string) => {
  activeTab.value = value;
  pagination.current = 1;
  fetchTickets();
};

const handleTableChange = (pag: any) => {
  pagination.current = pag.current;
  pagination.pageSize = pag.pageSize;
  fetchTickets();
};

const showCreateModal = () => {
  createModalVisible.value = true;
};

const resetForm = () => {
  form.value = { subject: '', content: '', resources: [] };
  formRef.value?.resetFields();
};

const handleCreate = async () => {
  try {
    await formRef.value.validate();
  } catch { return; }

  creating.value = true;
  try {
    const payload: any = {
      subject: form.value.subject,
      content: form.value.content
    };
    if (form.value.resources.length > 0) {
      const nameMap = new Map(vpsList.value.map((item: any) => [item.id, item.name]));
      payload.resources = form.value.resources.map((id) => ({
        resource_type: "vps",
        resource_id: id,
        resource_name: nameMap.get(id) || `VPS-${id}`
      }));
    }
    const res = await createTicket(payload);
    message.success('工单创建成功');
    createModalVisible.value = false;
    resetForm();
    router.push(`/console/tickets/${res.data.ticket.id}`);
  } catch (error: any) {
    message.error(error.response?.data?.error || '创建失败');
  } finally {
    creating.value = false;
  }
};

onMounted(() => {
  fetchTickets();
  fetchVps();
});
</script>

<style scoped>
.tickets-page {
  padding: 24px;
  max-width: 1400px;
  margin: 0 auto;
}

/* Header */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.header-content {
  display: flex;
  align-items: baseline;
  gap: 16px;
}

.page-title {
  font-size: 28px;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0;
}

.page-subtitle {
  font-size: 14px;
  color: var(--text-secondary);
  margin: 0;
}

.header-actions :deep(.ant-btn) {
  height: 40px;
  padding: 0 20px;
  font-weight: 500;
}

/* Status Tabs */
.status-tabs {
  margin-bottom: 24px;
}

.status-tabs :deep(.ant-segmented) {
  padding: 4px;
  background: var(--bg-secondary);
  border-radius: var(--radius-md);
}

.status-tabs :deep(.ant-segmented-item) {
  border-radius: var(--radius-sm);
  padding: 8px 20px;
  font-weight: 500;
}

/* Table Section */
.table-section {
  background: var(--card);
  border-radius: var(--radius-lg);
  border: 1px solid var(--border);
  overflow: hidden;
}

.tickets-table :deep(.ant-table) {
  background: transparent;
}

.tickets-table :deep(.ant-table-thead > tr > th) {
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border);
  padding: 14px 16px;
  font-weight: 600;
  font-size: 13px;
  color: var(--text-secondary);
}

.tickets-table :deep(.ant-table-tbody > tr > td) {
  padding: 16px;
  border-bottom: 1px solid var(--border-light);
}

.tickets-table :deep(.ant-table-tbody > tr:hover > td) {
  background: var(--bg-secondary);
}

.tickets-table :deep(.ant-table-tbody > tr:last-child > td) {
  border-bottom: none;
}

/* Subject Cell */
.subject-cell {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.subject-icon {
  font-size: 18px;
  color: var(--primary);
  margin-top: 2px;
}

.subject-content {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.ticket-id {
  font-size: 12px;
  padding: 2px 8px;
  background: var(--bg-tertiary);
  color: var(--text-secondary);
  border: none;
  border-radius: var(--radius-sm);
}

.subject-text {
  font-weight: 500;
  color: var(--text-primary);
}

/* Message Cell */
.message-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.message-icon {
  font-size: 14px;
  color: var(--text-tertiary);
}

.message-text {
  color: var(--text-secondary);
  font-size: 13px;
}

/* Resources Cell */
.resources-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.resource-icon {
  font-size: 16px;
  color: var(--info);
}

/* Time Cell */
.time-cell {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--text-secondary);
  font-size: 13px;
}

.time-icon {
  font-size: 14px;
  color: var(--text-tertiary);
}

/* VPS Option */
.vps-option {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 4px 0;
}

.vps-option-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.vps-option-name {
  font-weight: 500;
  color: var(--text-primary);
  font-size: 14px;
}

.vps-option-specs {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--text-secondary);
}

/* Modal Form */
:deep(.ant-form-item) {
  margin-bottom: 20px;
}

:deep(.ant-form-item-label > label) {
  font-weight: 500;
}

/* Responsive */
@media (max-width: 768px) {
  .tickets-page {
    padding: 16px;
  }

  .page-header {
    flex-direction: column;
    align-items: stretch;
    gap: 16px;
  }

  .header-content {
    flex-direction: column;
    gap: 4px;
  }

  .page-title {
    font-size: 22px;
  }

  .status-tabs :deep(.ant-segmented-item) {
    padding: 6px 12px;
    font-size: 13px;
  }
}
</style>
