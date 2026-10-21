<template>
  <div class="realname-providers-page">
    <div class="page-header">
      <h1 class="page-title">实名认证服务商</h1>
    </div>

    <a-card :bordered="false">
      <a-table
        :columns="columns"
        :data-source="providers"
        :loading="loading"
        row-key="key"
        :pagination="false"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'actions'">
            <a-button type="link" @click="showInfo(record)">
              查看详情
            </a-button>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- Info Modal -->
    <a-modal
      v-model:open="infoModalVisible"
      :title="currentProvider?.name"
      :footer="null"
    >
      <a-descriptions :column="1" bordered>
        <a-descriptions-item label="服务商Key">
          {{ currentProvider?.key }}
        </a-descriptions-item>
        <a-descriptions-item label="名称">
          {{ currentProvider?.name }}
        </a-descriptions-item>
      </a-descriptions>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue";
import { listRealNameProviders } from "@/services/admin";

const loading = ref(false);
const infoModalVisible = ref(false);
const providers = ref<any[]>([]);
const currentProvider = ref<any>(null);

const columns = [
  { title: "服务商Key", dataIndex: "key", key: "key" },
  { title: "名称", dataIndex: "name", key: "name" },
  { title: "操作", key: "actions", width: 120 }
];

const fetchData = async () => {
  loading.value = true;
  try {
    const res = await listRealNameProviders();
    providers.value = res.data?.items || [];
  } finally {
    loading.value = false;
  }
};

const showInfo = (record: any) => {
  currentProvider.value = record;
  infoModalVisible.value = true;
};

onMounted(() => {
  fetchData();
});
</script>

<style scoped>
.realname-providers-page {
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
