<template>
  <div class="http-headers-table">
    <ElTable :data="rows" size="small" border empty-text="暂无 Headers" row-key="key">
      <ElTableColumn prop="key" label="Key" min-width="220">
        <template #default="{ row }">
          <code class="header-key">{{ row.key }}</code>
        </template>
      </ElTableColumn>

      <ElTableColumn prop="value" label="Value" min-width="320">
        <template #default="{ row }">
          <code class="header-value">{{ row.value || '-' }}</code>
        </template>
      </ElTableColumn>

      <ElTableColumn v-if="copyable" label="" width="72" align="center">
        <template #default="{ row }">
          <ElButton link type="primary" @click="copyValue(row.value)">复制</ElButton>
        </template>
      </ElTableColumn>
    </ElTable>
  </div>
</template>

<script setup lang="ts">
  defineOptions({ name: 'HttpHeadersTable' })

  interface Props {
    headers: Record<string, string> | null
    copyable?: boolean
  }

  const props = withDefaults(defineProps<Props>(), {
    copyable: true
  })

  const rows = computed(() => {
    if (!props.headers) {
      return []
    }

    return Object.entries(props.headers).map(([key, value]) => ({
      key,
      value: String(value ?? '')
    }))
  })

  async function copyValue(value: string) {
    try {
      await navigator.clipboard.writeText(String(value || ''))
      ElMessage.success('已复制到剪贴板')
    } catch {
      ElMessage.error('复制失败')
    }
  }
</script>

<style scoped lang="scss">
  .header-key,
  .header-value {
    display: inline-block;
    font-family: 'Consolas', 'Liberation Mono', Menlo, monospace;
    font-size: 12px;
    line-height: 1.6;
    word-break: break-all;
  }

  .header-key {
    color: var(--el-color-primary);
    font-weight: 600;
  }

  .header-value {
    color: var(--el-text-color-regular);
  }
</style>
