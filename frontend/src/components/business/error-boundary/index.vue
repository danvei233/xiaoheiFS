<template>
  <slot v-if="!hasError" />

  <div v-else class="error-boundary">
    <ElCard class="error-card" shadow="never">
      <div class="title">页面渲染失败</div>
      <div class="description">通常是接口返回异常或前端运行时错误导致。可先刷新重试。</div>
      <!-- 不把原始错误信息渲染给访客：错误里可能带接口地址/token/内部字段名（P2-79 / m064 F10）。
           原始错误已 console.error，便于开发者定位。 -->
      <div class="actions">
        <ElButton type="primary" @click="reload">刷新页面</ElButton>
      </div>
    </ElCard>
  </div>
</template>

<script setup lang="ts">
  defineOptions({ name: 'ErrorBoundary' })

  const route = useRoute()
  const hasError = ref(false)
  const reload = () => window.location.reload()

  // web_bundle_mobile F1：ErrorBoundary 的 key 只跨布局域变化，不再随每次
  // 路由切换整壳重挂载（那会让 KeepAlive 缓存失效、每个列表页重新拉数据）。
  // 错误兜底状态改为在路由变化（含同域内切换）时复位，保留 P2-79 语义。
  watch(
    () => route.fullPath,
    () => {
      hasError.value = false
    }
  )

  onErrorCaptured((error) => {
    const errorWithMessage = error as { message?: unknown } | null | undefined
    const errorMessage = String(errorWithMessage?.message || error || '')

    // 页面逻辑会自行处理预期的请求错误，不能让它们触发整页兜底。
    if (errorMessage.includes('Request failed with status code')) {
      return false
    }

    hasError.value = true
    // 原始错误只进控制台，不渲染给用户（P2-79 / m064 F10）。
    console.error('[ErrorBoundary]', error)
    return false
  })
</script>

<style lang="scss" scoped>
  .error-boundary {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 100vh;
    padding: 24px;
    background: var(--default-bg-color);
  }

  .error-card {
    width: 100%;
    max-width: 720px;
    border-color: var(--art-card-border);
    border-radius: calc(var(--custom-radius) / 2 + 2px);
    box-shadow: var(--el-box-shadow-light);
  }

  .title {
    margin-bottom: 6px;
    font-size: 18px;
    font-weight: 700;
    color: var(--art-gray-900);
  }

  .description {
    margin-bottom: 12px;
    color: var(--art-gray-600);
  }

  .actions {
    display: flex;
    justify-content: flex-end;
  }
</style>
