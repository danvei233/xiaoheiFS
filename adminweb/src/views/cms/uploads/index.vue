<template>
  <div class="art-full-height">
    <ElCard v-loading="loading" class="art-table-card">
      <template #header>
        <div class="page-header">
          <div>
            <div class="page-title">素材上传</div>
            <div class="page-subtitle">
              上传内容管理所需图片，并在文章或区块中复用已有素材地址。
            </div>
          </div>

          <div class="page-actions">
            <ElButton v-if="canView" :disabled="loading" @click="fetchData">刷新</ElButton>
            <ElUpload
              v-if="canUpload"
              :auto-upload="false"
              :show-file-list="false"
              accept="image/png,image/jpeg,image/gif,image/webp"
              @change="handleFileSelect"
            >
              <ElButton type="primary" :loading="uploading">上传图片</ElButton>
            </ElUpload>
          </div>
        </div>
      </template>

      <ElEmpty v-if="!canView" description="你没有查看素材的权限。" />

      <template v-else>
        <div class="toolbar">
          <ElInput
            v-model="keyword"
            clearable
            placeholder="按文件名或 MIME 类型搜索"
            class="toolbar-search"
          />
        </div>

        <ArtTableHeader
          v-model:columns="columnChecks"
          :showSearchBar="false"
          :loading="loading"
          @refresh="fetchData"
        />

        <ArtTable
          row-key="id"
          :loading="loading"
          :data="filteredRows"
          :columns="columns"
          :pagination="pagination"
          @pagination:size-change="handlePageSizeChange"
          @pagination:current-change="handlePageCurrentChange"
        >
          <template #preview="{ row }">
            <ElImage
              v-if="isImage(row.mime)"
              :src="row.url"
              :preview-src-list="[row.url]"
              fit="cover"
              class="upload-preview"
              preview-teleported
            />
            <div v-else class="upload-placeholder">文件</div>
          </template>

          <template #name="{ row }">
            <a :href="row.url" target="_blank" rel="noopener noreferrer" class="upload-link">
              {{ row.name || row.url || '-' }}
            </a>
          </template>

          <template #mime="{ row }">
            <ElTag size="small">{{ row.mime || '-' }}</ElTag>
          </template>

          <template #size="{ row }">
            {{ formatSize(row.size) }}
          </template>

          <template #created_at="{ row }">
            {{ formatDateTime(row.created_at) }}
          </template>

          <template #operation="{ row }">
            <div class="table-actions">
              <ElButton link type="primary" @click="copyUrl(row.url)">复制地址</ElButton>
              <ElButton link @click="openUrl(row.url)">打开</ElButton>
            </div>
          </template>
        </ArtTable>
      </template>
    </ElCard>
  </div>
</template>

<script setup lang="ts">
  import type { UploadAssetRecord } from '@/api/admin'
  import { fetchAdminUploads, hasAdminPermission, uploadAdminAsset } from '@/api/admin'
  import { useTableColumns } from '@/hooks/core/useTableColumns'
  import { useUserStore } from '@/store/modules/user'
  import { ElMessage, type UploadFile } from 'element-plus'

  defineOptions({ name: 'CmsUploadsPage' })

  interface UploadRow {
    id: number | null
    name: string
    path: string
    url: string
    mime: string
    size: number
    uploader_id: number | null
    created_at: string
  }

  const userStore = useUserStore()
  const { info } = storeToRefs(userStore)

  const loading = ref(false)
  const initialized = ref(false)
  const uploading = ref(false)
  const keyword = ref('')
  const tableData = ref<UploadRow[]>([])

  const pagination = reactive({
    current: 1,
    size: 20,
    total: 0
  })

  const { columnChecks, columns } = useTableColumns<UploadRow>(() => [
    { prop: 'id', label: 'ID', width: 80 },
    { prop: 'preview', label: '预览', width: 92, useSlot: true },
    { prop: 'name', label: '文件名', minWidth: 220, useSlot: true },
    { prop: 'mime', label: '文件类型', width: 180, useSlot: true },
    { prop: 'size', label: '大小', width: 120, useSlot: true },
    { prop: 'uploader_id', label: '上传者 ID', width: 120 },
    { prop: 'created_at', label: '创建时间', minWidth: 180, useSlot: true },
    { prop: 'operation', label: '操作', width: 180, fixed: 'right', useSlot: true }
  ])

  const canView = computed(() => hasAdminPermission(info.value?.buttons, ['upload.list']))
  const canUpload = computed(() => hasAdminPermission(info.value?.buttons, ['upload.create']))

  const filteredRows = computed(() => {
    const search = keyword.value.trim().toLowerCase()
    if (!search) {
      return tableData.value
    }

    return tableData.value.filter((row) =>
      [row.name, row.mime, row.url].some((value) =>
        String(value || '')
          .toLowerCase()
          .includes(search)
      )
    )
  })

  watch(
    canView,
    (value) => {
      if (value && !initialized.value) {
        initialized.value = true
        fetchData()
      }
    },
    { immediate: true }
  )

  function normalizeNullableNumber(value: unknown): number | null {
    if (value === '' || value === null || value === undefined) {
      return null
    }

    const parsed = Number(value)
    return Number.isFinite(parsed) ? parsed : null
  }

  function normalizeRow(item?: UploadAssetRecord): UploadRow {
    return {
      id: normalizeNullableNumber(item?.id),
      name: String(item?.name || ''),
      path: String(item?.path || ''),
      url: String(item?.url || ''),
      mime: String(item?.mime || ''),
      size: Number(item?.size || 0),
      uploader_id: normalizeNullableNumber(item?.uploader_id),
      created_at: String(item?.created_at || '')
    }
  }

  function isImage(mime: string) {
    return String(mime || '').startsWith('image/')
  }

  function formatSize(size: number) {
    if (!size) {
      return '-'
    }

    if (size < 1024) {
      return `${size} B`
    }

    if (size < 1024 * 1024) {
      return `${(size / 1024).toFixed(2)} KB`
    }

    return `${(size / (1024 * 1024)).toFixed(2)} MB`
  }

  function formatDateTime(value?: string | null) {
    if (!value) {
      return '-'
    }

    const date = new Date(value)
    return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN')
  }

  async function copyUrl(url: string) {
    const text = String(url || '').trim()
    if (!text) {
      return
    }

    try {
      await navigator.clipboard.writeText(text)
      ElMessage.success('地址复制成功')
    } catch {
      ElMessage.error('复制地址失败')
    }
  }

  function openUrl(url: string) {
    const text = String(url || '').trim()
    if (!text) {
      return
    }

    window.open(text, '_blank', 'noopener')
  }

  async function fetchData() {
    if (!canView.value) {
      return
    }

    loading.value = true

    try {
      const payload = await fetchAdminUploads({
        limit: pagination.size,
        offset: (pagination.current - 1) * pagination.size
      })

      tableData.value = (payload.items || []).map((item) => normalizeRow(item))
      pagination.total = Number(payload.total || 0)
    } finally {
      loading.value = false
    }
  }

  function handlePageSizeChange(size: number) {
    pagination.size = size
    pagination.current = 1
    fetchData()
  }

  function handlePageCurrentChange(page: number) {
    pagination.current = page
    fetchData()
  }

  async function handleFileSelect(uploadFile: UploadFile) {
    if (!uploadFile.raw) {
      return
    }

    uploading.value = true

    try {
      await uploadAdminAsset(uploadFile.raw)
      ElMessage.success('图片上传成功')
      await fetchData()
    } catch (error: any) {
      ElMessage.error(error?.response?.data?.error || '图片上传失败')
    } finally {
      uploading.value = false
    }
  }
</script>

<style scoped lang="scss">
  .page-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
  }

  .page-title {
    color: var(--el-text-color-primary);
    font-size: 20px;
    font-weight: 700;
  }

  .page-subtitle {
    margin-top: 6px;
    color: var(--el-text-color-secondary);
    font-size: 13px;
    line-height: 1.6;
  }

  .page-actions,
  .toolbar,
  .table-actions {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .toolbar {
    margin-bottom: 12px;
    flex-wrap: wrap;
  }

  .toolbar-search {
    width: min(360px, 100%);
  }

  .table-actions {
    justify-content: flex-end;
  }

  .upload-preview,
  .upload-placeholder {
    width: 48px;
    height: 48px;
    border-radius: 10px;
  }

  .upload-placeholder {
    display: flex;
    align-items: center;
    justify-content: center;
    border: 1px solid var(--el-border-color);
    color: var(--el-text-color-secondary);
    font-size: 11px;
    font-weight: 700;
    letter-spacing: 0.08em;
    background: var(--el-fill-color-light);
  }

  .upload-link {
    color: var(--el-color-primary);
    text-decoration: none;
  }

  @media (max-width: 768px) {
    .page-header {
      align-items: flex-start;
      flex-direction: column;
    }

    .page-actions,
    .toolbar,
    .toolbar-search {
      width: 100%;
    }

    .page-actions {
      flex-wrap: wrap;
    }
  }
</style>
