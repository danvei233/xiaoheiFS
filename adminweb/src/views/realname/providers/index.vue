<template>
  <div class="art-full-height">
    <ElCard v-loading="loading" class="art-table-card">
      <template #header>
        <div class="page-header">
          <div>
            <div class="page-title">实名供应商</div>
            <div class="page-subtitle">查看当前可用的实名认证服务商与实例标识。</div>
          </div>

          <div class="page-actions">
            <ElButton v-if="canView" :disabled="loading" @click="fetchData">刷新</ElButton>
          </div>
        </div>
      </template>

      <ElEmpty v-if="!canView" description="当前账号没有查看实名供应商的权限。" />

      <template v-else>
        <ArtTableHeader v-model:columns="columnChecks" :loading="loading" @refresh="fetchData" />

        <ArtTable row-key="key" :loading="loading" :data="tableData" :columns="columns">
          <template #operation="{ row }">
            <div class="table-actions">
              <ElButton link type="primary" @click="openDetail(row)">查看详情</ElButton>
            </div>
          </template>
        </ArtTable>
      </template>
    </ElCard>

    <ElDialog
      v-model="detailVisible"
      title="供应商详情"
      width="520px"
      destroy-on-close
      align-center
    >
      <ElDescriptions :column="1" border>
        <ElDescriptionsItem label="服务商 Key">
          {{ activeProvider?.key || '-' }}
        </ElDescriptionsItem>
        <ElDescriptionsItem label="名称">
          {{ activeProvider?.name || '-' }}
        </ElDescriptionsItem>
      </ElDescriptions>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import type { RealNameProviderRecord } from '@/api/admin'
  import { fetchRealNameProviders, hasAdminPermission } from '@/api/admin'
  import { useTableColumns } from '@/hooks/core/useTableColumns'
  import { useUserStore } from '@/store/modules/user'

  defineOptions({ name: 'RealnameProvidersPage' })

  interface ProviderTableRow {
    key: string
    name: string
  }

  const userStore = useUserStore()
  const { info } = storeToRefs(userStore)

  const loading = ref(false)
  const detailVisible = ref(false)
  const activeProvider = ref<ProviderTableRow | null>(null)
  const tableData = ref<ProviderTableRow[]>([])

  const canView = computed(() => hasAdminPermission(info.value?.buttons, ['realname.list']))

  const { columnChecks, columns } = useTableColumns<ProviderTableRow>(() => [
    { prop: 'key', label: '服务商 Key', minWidth: 260, showOverflowTooltip: true },
    { prop: 'name', label: '名称', minWidth: 180, showOverflowTooltip: true },
    { prop: 'operation', label: '操作', width: 120, fixed: 'right', useSlot: true }
  ])

  onMounted(() => {
    fetchData()
  })

  function normalizeProvider(item?: RealNameProviderRecord): ProviderTableRow {
    return {
      key: String(item?.key || ''),
      name: String(item?.name || '')
    }
  }

  async function fetchData() {
    if (!canView.value) {
      return
    }

    loading.value = true

    try {
      const payload = await fetchRealNameProviders()
      tableData.value = (payload.items || []).map((item) => normalizeProvider(item))
    } finally {
      loading.value = false
    }
  }

  function openDetail(row: ProviderTableRow) {
    activeProvider.value = row
    detailVisible.value = true
  }
</script>

<style scoped lang="scss">
  .page-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 16px;
  }

  .page-title {
    color: var(--el-text-color-primary);
    font-size: 18px;
    font-weight: 700;
  }

  .page-subtitle {
    margin-top: 6px;
    color: var(--el-text-color-secondary);
    font-size: 13px;
    line-height: 1.7;
  }

  .page-actions,
  .table-actions {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .table-actions {
    justify-content: flex-end;
    gap: 4px;
  }

  @media (max-width: 768px) {
    .page-header {
      flex-direction: column;
      align-items: stretch;
    }
  }
</style>
