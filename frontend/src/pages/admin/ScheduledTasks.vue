<template>
  <div class="scheduled-tasks-page">
    <div class="page-header">
      <h1 class="page-title">定时任务</h1>
      <a-button @click="fetchData" :loading="loading">
        <template #icon><ReloadOutlined /></template>
        刷新
      </a-button>
    </div>

    <a-card :bordered="false">
      <a-table
        :columns="columns"
        :data-source="tasks"
        :loading="loading"
        row-key="key"
        :pagination="false"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'enabled'">
            <a-switch
              :checked="record.enabled"
              @change="(checked: boolean) => handleToggle(record, checked)"
            />
          </template>
          <template v-else-if="column.key === 'strategy'">
            <a-tag>{{ record.strategy }}</a-tag>
          </template>
          <template v-else-if="column.key === 'last_run_at'">
            {{ formatDate(record.last_run_at) }}
          </template>
          <template v-else-if="column.key === 'next_run_at'">
            {{ formatDate(record.next_run_at) }}
          </template>
          <template v-else-if="column.key === 'actions'">
            <a-button type="link" @click="openConfigModal(record)">
              配置
            </a-button>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- Config Modal -->
    <a-modal
      v-model:open="configModalVisible"
      :title="`配置 ${currentTask?.name}`"
      @ok="saveConfig"
      :confirm-loading="configSaving"
      width="500px"
    >
      <a-form layout="vertical">
        <a-form-item label="启用状态">
          <a-switch v-model:checked="configForm.enabled" />
        </a-form-item>

        <a-form-item label="执行策略">
          <a-select v-model:value="configForm.strategy">
            <a-select-option value="interval">间隔执行</a-select-option>
            <a-select-option value="daily">每日执行</a-select-option>
          </a-select>
        </a-form-item>

        <a-form-item v-if="configForm.strategy === 'interval'" label="执行间隔（秒）">
          <a-input-number v-model:value="configForm.interval_sec" :min="1" style="width: 100%" />
        </a-form-item>

        <a-form-item v-if="configForm.strategy === 'daily'" label="执行时间">
          <a-time-picker v-model:value="dailyAtTime" format="HH:mm" style="width: 100%" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from "vue";
import { message } from "ant-design-vue";
import { ReloadOutlined } from "@ant-design/icons-vue";
import { listAdminScheduledTasks, updateAdminScheduledTask } from "@/services/admin";
import dayjs, { Dayjs } from "dayjs";

const loading = ref(false);
const configSaving = ref(false);
const configModalVisible = ref(false);
const tasks = ref<any[]>([]);
const currentTask = ref<any>(null);
const dailyAtTime = ref<Dayjs>();

const configForm = reactive({
  enabled: false,
  strategy: "interval",
  interval_sec: 3600,
  daily_at: "00:00"
});

const columns = [
  { title: "任务名称", dataIndex: "name", key: "name" },
  { title: "描述", dataIndex: "description", key: "description" },
  { title: "状态", dataIndex: "enabled", key: "enabled", width: 100 },
  { title: "策略", dataIndex: "strategy", key: "strategy", width: 120 },
  { title: "上次运行", dataIndex: "last_run_at", key: "last_run_at", width: 180 },
  { title: "下次运行", dataIndex: "next_run_at", key: "next_run_at", width: 180 },
  { title: "操作", key: "actions", width: 100 }
];

const formatDate = (date: string) => {
  if (!date) return "-";
  return new Date(date).toLocaleString("zh-CN");
};

const fetchData = async () => {
  loading.value = true;
  try {
    const res = await listAdminScheduledTasks();
    tasks.value = res.data?.items || [];
  } finally {
    loading.value = false;
  }
};

const handleToggle = async (record: any, checked: boolean) => {
  try {
    await updateAdminScheduledTask(record.key, { enabled: checked });
    record.enabled = checked;
    message.success("操作成功");
  } catch (error: any) {
    message.error(error.response?.data?.error || "操作失败");
  }
};

const openConfigModal = (record: any) => {
  currentTask.value = record;
  configForm.enabled = record.enabled;
  configForm.strategy = record.strategy;
  configForm.interval_sec = record.interval_sec || 3600;
  if (record.daily_at) {
    dailyAtTime.value = dayjs(record.daily_at, "HH:mm");
  }
  configModalVisible.value = true;
};

const saveConfig = async () => {
  configSaving.value = true;
  try {
    const payload: any = {
      enabled: configForm.enabled,
      strategy: configForm.strategy
    };
    if (configForm.strategy === "interval") {
      payload.interval_sec = configForm.interval_sec;
    } else if (configForm.strategy === "daily" && dailyAtTime.value) {
      payload.daily_at = dailyAtTime.value.format("HH:mm");
    }
    await updateAdminScheduledTask(currentTask.value.key, payload);
    message.success("保存成功");
    configModalVisible.value = false;
    fetchData();
  } catch (error: any) {
    message.error(error.response?.data?.error || "保存失败");
  } finally {
    configSaving.value = false;
  }
};

onMounted(() => {
  fetchData();
});
</script>

<style scoped>
.scheduled-tasks-page {
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
