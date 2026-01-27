<template>
  <div class="http-headers-table">
    <a-table
      :columns="columns"
      :data-source="dataSource"
      :pagination="false"
      size="small"
      :show-header="true"
      row-key="key"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'key'">
          <code class="header-key">{{ record.key }}</code>
        </template>
        <template v-else-if="column.key === 'value'">
          <code class="header-value">{{ record.value }}</code>
        </template>
        <template v-else-if="column.key === 'actions' && copyable">
          <a-button type="text" size="small" @click="copyValue(record.value)">
            <CopyOutlined />
          </a-button>
        </template>
      </template>
    </a-table>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { message } from "ant-design-vue";
import { CopyOutlined } from "@ant-design/icons-vue";

interface Props {
  headers: Record<string, string> | null;
  copyable?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  copyable: true
});

const columns = [
  { title: "Key", dataIndex: "key", key: "key", width: "35%" },
  { title: "Value", dataIndex: "value", key: "value" },
  { title: "", dataIndex: "actions", key: "actions", width: 60, align: "center" }
];

const dataSource = computed(() => {
  if (!props.headers) return [];
  return Object.entries(props.headers).map(([key, value]) => ({
    key,
    value: String(value)
  }));
});

const copyValue = async (value: string) => {
  try {
    await navigator.clipboard.writeText(value);
    message.success("已复制到剪贴板");
  } catch {
    message.error("复制失败");
  }
};
</script>

<style scoped>
.http-headers-table {
  background: #fff;
  border: 1px solid rgba(0, 0, 0, 0.06);
  border-radius: 8px;
  overflow: hidden;
}

.http-headers-table :deep(.ant-table) {
  font-size: 12px;
}

.http-headers-table :deep(.ant-table-thead > tr > th) {
  background: rgba(0, 0, 0, 0.02);
  font-weight: 600;
  padding: 10px 14px;
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
  font-size: 12px;
  color: rgba(0, 0, 0, 0.88);
}

.http-headers-table :deep(.ant-table-tbody > tr > td) {
  padding: 10px 14px;
  border-bottom: 1px solid rgba(0, 0, 0, 0.04);
}

.http-headers-table :deep(.ant-table-tbody > tr:last-child > td) {
  border-bottom: none;
}

.http-headers-table :deep(.ant-table-tbody > tr:hover > td) {
  background: rgba(0, 0, 0, 0.02);
}

.header-key {
  font-family: 'SFMono-Regular', 'Consolas', 'Liberation Mono', Menlo, monospace;
  color: #1677ff;
  font-size: 12px;
  font-weight: 500;
}

.header-value {
  font-family: 'SFMono-Regular', 'Consolas', 'Liberation Mono', Menlo, monospace;
  color: rgba(0, 0, 0, 0.75);
  font-size: 12px;
  word-break: break-all;
  line-height: 1.6;
}

.http-headers-table :deep(.ant-btn-text) {
  color: rgba(0, 0, 0, 0.45);
  transition: color 0.15s;
}

.http-headers-table :deep(.ant-btn-text:hover) {
  color: #1677ff;
}
</style>
