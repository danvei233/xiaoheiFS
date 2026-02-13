<template>
  <div class="probes-page">
    <div class="page-head">
      <div>
        <div class="title">探针监控</div>
        <div class="subtle">查看探针在线状态、SLA 与基础信息</div>
        <div class="subtle">上次刷新：{{ formatDate(lastRefreshAt) }}<span v-if="refreshError">，刷新失败：{{ refreshError }}</span></div>
      </div>
      <a-space>
        <a-button @click="fetchList" :loading="loading">刷新</a-button>
        <a-button type="primary" @click="openCreate">新增探针</a-button>
      </a-space>
    </div>

    <a-card :bordered="false">
      <a-space style="margin-bottom: 12px">
        <a-input v-model:value="filters.keyword" allow-clear placeholder="按名称/AgentID 搜索" style="width: 240px" />
        <a-select v-model:value="filters.status" style="width: 140px" :options="statusOptions" allow-clear placeholder="状态" />
        <a-button type="primary" @click="onSearch">查询</a-button>
      </a-space>
      <a-table
        row-key="id"
        :columns="columns"
        :data-source="rows"
        :loading="loading"
        :pagination="pagination"
        @change="onTableChange"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'status'">
            <a-tag :color="record.status === 'online' ? 'success' : 'default'">
              {{ record.status === "online" ? "在线" : "离线" }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'tags'">
            <a-tag v-for="tag in record.tags || []" :key="tag">{{ tag }}</a-tag>
            <span v-if="!(record.tags || []).length" class="subtle">-</span>
          </template>
          <template v-else-if="column.key === 'last_heartbeat_at'">
            {{ formatDate(record.last_heartbeat_at) }}
          </template>
          <template v-else-if="column.key === 'sla'">
            {{ formatSla(record.id) }}
          </template>
          <template v-else-if="column.key === 'cpu_usage'">
            <div class="usage-cell">
              <a-progress
                v-if="readUsage(record, 'cpu') != null"
                :percent="readUsage(record, 'cpu') || 0"
                size="small"
                :stroke-color="usageColor(readUsage(record, 'cpu'))"
              />
              <span v-else class="subtle">-</span>
            </div>
          </template>
          <template v-else-if="column.key === 'mem_usage'">
            <div class="usage-cell">
              <a-progress
                v-if="readUsage(record, 'mem') != null"
                :percent="readUsage(record, 'mem') || 0"
                size="small"
                :stroke-color="usageColor(readUsage(record, 'mem'))"
              />
              <span v-else class="subtle">-</span>
            </div>
          </template>
          <template v-else-if="column.key === 'action'">
            <a-space>
              <a-button type="link" @click="goDetail(record)">详情</a-button>
              <a-button type="link" @click="resetEnroll(record)">重置注册码</a-button>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <a-modal v-model:open="createOpen" title="新增探针" @ok="submitCreate" :confirm-loading="creating">
      <a-form layout="vertical">
        <a-form-item label="探针名称">
          <a-input v-model:value="createForm.name" placeholder="例如：香港节点-A" />
        </a-form-item>
        <a-form-item label="Agent ID">
          <a-input v-model:value="createForm.agent_id" placeholder="唯一标识，例如 hk-node-a" />
        </a-form-item>
        <a-form-item label="OS 类型">
          <a-select v-model:value="createForm.os_type" :options="osOptions" />
        </a-form-item>
        <a-form-item label="标签">
          <a-select v-model:value="createForm.tags" mode="tags" :token-separators="[',']" placeholder="region:hkg,role:edge" />
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal v-model:open="tokenOpen" title="一次性注册码" :footer="null">
      <a-alert type="warning" show-icon message="仅展示一次，请尽快配置到探针端。" />
      <a-typography-paragraph copyable style="margin-top: 12px; word-break: break-all;">
        {{ latestToken }}
      </a-typography-paragraph>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, reactive, ref } from "vue";
import { message } from "ant-design-vue";
import { useRouter } from "vue-router";
import { createAdminProbe, getAdminProbeSla, listAdminProbes, resetAdminProbeEnrollToken } from "@/services/admin";

const router = useRouter();
const loading = ref(false);
const creating = ref(false);
const createOpen = ref(false);
const tokenOpen = ref(false);
const latestToken = ref("");
const rows = ref<any[]>([]);
const refreshError = ref("");
const lastRefreshAt = ref("");
const slaMap = reactive<Record<string, number>>({});
const filters = reactive({ keyword: "", status: undefined as string | undefined });
const pagination = reactive({ current: 1, pageSize: 20, total: 0, showSizeChanger: true });
let refreshTimer: number | null = null;
let listInFlight = false;

const createForm = reactive({
  name: "",
  agent_id: "",
  os_type: "linux",
  tags: [] as string[]
});

const statusOptions = [
  { label: "在线", value: "online" },
  { label: "离线", value: "offline" }
];

const osOptions = [
  { label: "Linux", value: "linux" },
  { label: "Windows", value: "windows" }
];

const columns = [
  { title: "ID", dataIndex: "id", key: "id", width: 80 },
  { title: "名称", dataIndex: "name", key: "name", width: 180 },
  { title: "Agent ID", dataIndex: "agent_id", key: "agent_id", width: 180 },
  { title: "状态", dataIndex: "status", key: "status", width: 90 },
  { title: "OS", dataIndex: "os_type", key: "os_type", width: 100 },
  { title: "CPU", key: "cpu_usage", width: 180 },
  { title: "内存", key: "mem_usage", width: 180 },
  { title: "标签", dataIndex: "tags", key: "tags" },
  { title: "最后心跳", dataIndex: "last_heartbeat_at", key: "last_heartbeat_at", width: 180 },
  { title: "7天 SLA", key: "sla", width: 110 },
  { title: "操作", key: "action", width: 220 }
];

const formatDate = (v?: string) => (v ? new Date(v).toLocaleString("zh-CN") : "-");
const formatSla = (id?: number) => {
  if (!id) return "-";
  const v = slaMap[String(id)];
  if (v == null) return "-";
  return `${v.toFixed(2)}%`;
};

const toPercent = (v: unknown) => {
  const n = Number(v);
  if (!Number.isFinite(n)) return null;
  return Math.max(0, Math.min(100, Number(n.toFixed(1))));
};

const readUsage = (record: any, kind: "cpu" | "mem") => {
  if (kind === "cpu") {
    return toPercent(record?.snapshot?.cpu?.usage_percent ?? record?.cpu_usage_percent);
  }
  return toPercent(record?.snapshot?.memory?.usage_percent ?? record?.mem_usage_percent);
};

const usageColor = (value: number | null) => {
  if (value == null) return "#d9d9d9";
  if (value < 60) return "#52c41a";
  if (value < 85) return "#faad14";
  return "#ff4d4f";
};

const loadSla = async () => {
  const ts = Date.now();
  const jobs = rows.value.map(async (item) => {
    const id = item.id;
    if (!id) return;
    try {
      const res = await getAdminProbeSla(id, { days: 7, _t: ts });
      slaMap[String(id)] = Number(res.data?.sla?.uptime_percent || 0);
    } catch {
      slaMap[String(id)] = 0;
    }
  });
  await Promise.all(jobs);
};

const fetchList = async () => {
  if (listInFlight) return;
  listInFlight = true;
  loading.value = true;
  try {
    const ts = Date.now();
    const res = await listAdminProbes({
      keyword: filters.keyword || undefined,
      status: filters.status || undefined,
      limit: pagination.pageSize,
      offset: (pagination.current - 1) * pagination.pageSize,
      _t: ts
    });
    rows.value = res.data?.items || [];
    pagination.total = res.data?.total || rows.value.length;
    await loadSla();
    lastRefreshAt.value = new Date().toISOString();
    refreshError.value = "";
  } catch (e: any) {
    refreshError.value = e?.message || "请求失败";
    message.error(`刷新失败: ${refreshError.value}`);
  } finally {
    loading.value = false;
    listInFlight = false;
  }
};

const onSearch = () => {
  pagination.current = 1;
  fetchList();
};

const onTableChange = (pager: any) => {
  pagination.current = pager.current || 1;
  pagination.pageSize = pager.pageSize || 20;
  fetchList();
};

const openCreate = () => {
  createForm.name = "";
  createForm.agent_id = "";
  createForm.os_type = "linux";
  createForm.tags = [];
  createOpen.value = true;
};

const submitCreate = async () => {
  if (!createForm.agent_id.trim()) {
    message.warning("请输入 Agent ID");
    return;
  }
  creating.value = true;
  try {
    const res = await createAdminProbe({
      name: createForm.name,
      agent_id: createForm.agent_id.trim(),
      os_type: createForm.os_type,
      tags: createForm.tags
    });
    latestToken.value = String(res.data?.enroll_token || "");
    tokenOpen.value = !!latestToken.value;
    createOpen.value = false;
    message.success("创建成功");
    fetchList();
  } catch (e: any) {
    message.error(e?.message || "创建失败");
  } finally {
    creating.value = false;
  }
};

const resetEnroll = async (record: any) => {
  try {
    const res = await resetAdminProbeEnrollToken(record.id);
    latestToken.value = String(res.data?.enroll_token || "");
    tokenOpen.value = !!latestToken.value;
    message.success("已重置注册码");
  } catch (e: any) {
    message.error(e?.message || "重置失败");
  }
};

const goDetail = (record: any) => {
  router.push(`/admin/probes/${record.id}`);
};

onMounted(() => {
  fetchList();
  refreshTimer = window.setInterval(() => {
    fetchList();
  }, 10000);
});

onBeforeUnmount(() => {
  if (refreshTimer != null) {
    window.clearInterval(refreshTimer);
    refreshTimer = null;
  }
});
</script>

<style scoped>
.probes-page {
  padding: 16px;
}

.probes-page .page-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}
.probes-page .title {
  font-size: 20px;
  font-weight: 600;
}
.probes-page .subtle {
  color: #8c8c8c;
}

.usage-cell {
  min-width: 150px;
}
</style>
