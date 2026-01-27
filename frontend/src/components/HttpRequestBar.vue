<template>
  <div class="http-request-bar">
    <div class="method-badge" :class="`method-${method.toLowerCase()}`">
      {{ method }}
    </div>
    <div class="url">{{ url || '-' }}</div>
    <div v-if="status !== undefined" class="status-info">
      <a-tag :color="getStatusColor(status)">{{ status }}</a-tag>
      <span v-if="duration !== undefined && duration !== null" class="duration">{{ duration }}ms</span>
    </div>
  </div>
</template>

<script setup lang="ts">
interface Props {
  method: string;
  url?: string;
  status?: number;
  duration?: number;
}

const props = withDefaults(defineProps<Props>(), {
  method: 'GET',
  url: ''
});

const getStatusColor = (status: number) => {
  if (status >= 200 && status < 300) return 'success';
  if (status >= 300 && status < 400) return 'processing';
  if (status >= 400 && status < 500) return 'warning';
  if (status >= 500) return 'error';
  return 'default';
};
</script>

<style scoped>
.http-request-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 14px;
  background: #ffffff;
  border: 1px solid rgba(0, 0, 0, 0.08);
  border-radius: 8px;
}

.method-badge {
  padding: 5px 10px;
  border-radius: 6px;
  font-weight: 600;
  font-size: 11px;
  min-width: 52px;
  text-align: center;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  flex-shrink: 0;
}

.method-get {
  background: rgba(22, 119, 255, 0.1);
  color: #1677ff;
}

.method-post {
  background: rgba(82, 196, 26, 0.1);
  color: #52c41a;
}

.method-put {
  background: rgba(250, 140, 22, 0.1);
  color: #fa8c16;
}

.method-delete {
  background: rgba(255, 77, 79, 0.1);
  color: #ff4d4f;
}

.method-patch {
  background: rgba(114, 46, 209, 0.1);
  color: #722ed1;
}

.method-head,
.method-options,
.method-default {
  background: rgba(0, 0, 0, 0.06);
  color: rgba(0, 0, 0, 0.65);
}

.url {
  flex: 1;
  font-family: 'SFMono-Regular', 'Consolas', 'Liberation Mono', Menlo, monospace;
  font-size: 12px;
  color: rgba(0, 0, 0, 0.85);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.status-info {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.status-info :deep(.ant-tag) {
  font-size: 12px;
  margin: 0;
  border-radius: 4px;
}

.duration {
  font-size: 12px;
  color: rgba(0, 0, 0, 0.45);
  font-family: 'SFMono-Regular', 'Consolas', 'Liberation Mono', Menlo, monospace;
}
</style>
