<template>
  <div class="http-request-bar">
    <span class="method-badge" :class="`method-${normalizedMethod.toLowerCase()}`">
      {{ normalizedMethod }}
    </span>

    <div class="request-url">{{ url || '-' }}</div>

    <div v-if="status !== undefined && status !== null" class="status-wrap">
      <ElTag size="small" :type="getStatusTagType(status)">
        {{ status }}
      </ElTag>
      <span v-if="duration !== undefined && duration !== null" class="duration-text">
        {{ duration }}ms
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
  defineOptions({ name: 'HttpRequestBar' })

  interface Props {
    method: string
    url?: string
    status?: number | string | null
    duration?: number | null
  }

  const props = withDefaults(defineProps<Props>(), {
    method: 'GET',
    url: '',
    status: undefined,
    duration: undefined
  })

  const normalizedMethod = computed(() => {
    const value = String(props.method || 'GET')
      .trim()
      .toUpperCase()
    return value || 'GET'
  })

  function getStatusTagType(status: number | string) {
    const code = Number(status)
    if (!Number.isFinite(code)) {
      return 'info' as const
    }

    if (code >= 200 && code < 300) {
      return 'success' as const
    }

    if (code >= 300 && code < 400) {
      return 'warning' as const
    }

    if (code >= 400) {
      return 'danger' as const
    }

    return 'info' as const
  }
</script>

<style scoped lang="scss">
  .http-request-bar {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px 14px;
    border: 1px solid var(--el-border-color);
    border-radius: 10px;
    background: var(--el-bg-color-page);
  }

  .method-badge {
    flex-shrink: 0;
    min-width: 58px;
    padding: 5px 10px;
    border-radius: 999px;
    font-size: 11px;
    font-weight: 700;
    text-align: center;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .method-get {
    color: #2563eb;
    background: rgb(37 99 235 / 12%);
  }

  .method-post {
    color: #16a34a;
    background: rgb(22 163 74 / 12%);
  }

  .method-put,
  .method-patch {
    color: #ea580c;
    background: rgb(234 88 12 / 12%);
  }

  .method-delete {
    color: #dc2626;
    background: rgb(220 38 38 / 12%);
  }

  .method-resp,
  .method-grpc {
    color: #7c3aed;
    background: rgb(124 58 237 / 12%);
  }

  .request-url {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    color: var(--el-text-color-primary);
    font-family: 'Consolas', 'Liberation Mono', Menlo, monospace;
    font-size: 12px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .status-wrap {
    display: flex;
    flex-shrink: 0;
    align-items: center;
    gap: 8px;
  }

  .duration-text {
    color: var(--el-text-color-secondary);
    font-family: 'Consolas', 'Liberation Mono', Menlo, monospace;
    font-size: 12px;
  }

  @media (max-width: 768px) {
    .http-request-bar {
      flex-wrap: wrap;
    }

    .request-url {
      width: 100%;
      white-space: normal;
      word-break: break-all;
    }
  }
</style>
