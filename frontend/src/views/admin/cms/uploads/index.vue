<template>
  <div class="art-full-height">
    <ElCard v-loading="loading" class="art-table-card">
      <template #header>
        <div class="page-header">
          <div>
            <div class="page-title">文件上传</div>
            <div class="page-subtitle">
              上传内容管理所需文件，并在文章或区块中复用已有素材地址。
            </div>
          </div>

          <div class="page-actions">
            <ElButton v-if="canView" :disabled="loading" @click="fetchData()">刷新</ElButton>
            <ElUpload
              v-if="canUpload"
              :auto-upload="false"
              :show-file-list="false"
              accept="image/png,image/jpeg,image/gif,image/webp"
              @change="handleFileSelect"
            >
              <ElButton type="primary" :loading="uploading">上传文件</ElButton>
            </ElUpload>
          </div>
        </div>
      </template>

      <ElEmpty v-if="!canView" description="你没有查看素材的权限。" />

      <template v-else>
        <ArtTableHeader
          v-model:columns="columnChecks"
          :showSearchBar="false"
          :loading="loading"
          @refresh="fetchData"
        />

        <ArtTable
          row-key="id"
          :loading="loading"
          :data="tableData"
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
            <div v-else class="upload-placeholder">{{ getFileTypeLabel(row.mime, row.name) }}</div>
          </template>

          <template #name="{ row }">
            <a
              :href="safeHref(row.url)"
              target="_blank"
              rel="noopener noreferrer"
              class="upload-link"
            >
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
  import type { UploadAssetRecord } from '@/services/admin'
  import { fetchAdminUploads, uploadAdminAsset } from '@/services/admin'
  import { useAuth } from '@/hooks/core/useAuth'
  import { useTable } from '@/hooks/core/useTable'
  import { ElMessage, type UploadFile } from 'element-plus'
  import { openExternalUrl, safeHref } from '@/utils/urlPolicy'

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

  type UploadTableParams = Api.Common.CommonSearchParams

  const { hasAuth } = useAuth()

  const initialized = ref(false)
  const uploading = ref(false)
  const {
    columnChecks,
    columns,
    data: tableData,
    loading,
    pagination,
    fetchData,
    handleSizeChange: handlePageSizeChange,
    handleCurrentChange: handlePageCurrentChange
  } = useTable({
    core: {
      apiFn: fetchUploadTable,
      apiParams: { current: 1, size: 20 },
      immediate: false,
      columnsFactory: () => [
        { prop: 'id', label: 'ID', width: 80 },
        { prop: 'preview', label: '预览', width: 92, useSlot: true },
        { prop: 'name', label: '文件名', minWidth: 220, useSlot: true },
        { prop: 'mime', label: '文件类型', width: 180, useSlot: true },
        { prop: 'size', label: '大小', width: 120, useSlot: true },
        { prop: 'uploader_id', label: '上传者 ID', width: 120 },
        { prop: 'created_at', label: '创建时间', minWidth: 180, useSlot: true },
        { prop: 'operation', label: '操作', width: 180, fixed: 'right', useSlot: true }
      ]
    }
  })

  const canView = computed(() => hasAuth('upload.list'))
  const canUpload = computed(() => hasAuth('upload.create'))

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

  function getFileTypeLabel(mime: string, name: string) {
    const value = `${mime} ${name}`.toLowerCase()
    if (value.includes('pdf')) return 'PDF'
    if (value.includes('word') || /\.docx?$/i.test(name)) return 'Word'
    if (value.includes('excel') || value.includes('spreadsheet') || /\.xlsx?$/i.test(name)) {
      return 'Excel'
    }
    if (value.includes('powerpoint') || value.includes('presentation') || /\.pptx?$/i.test(name)) {
      return 'PPT'
    }
    if (value.includes('zip') || value.includes('compressed') || /\.(rar|7z|tar|gz)$/i.test(name)) {
      return '压缩包'
    }
    if (value.includes('video')) return '视频'
    if (value.includes('audio')) return '音频'
    if (value.includes('text')) return '文本'
    return '文件'
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

    // H9：文件 URL 来自后端列表数据，直接 window.open 可被 javascript:/
    // data: 注入；统一过 URL 安全策略，新窗口强制 noopener,noreferrer。
    // allowRelative：素材 URL 是后端生成的同源相对路径（/uploads/...），
    // 不做相对路径放行的话"打开"按钮对每个素材都判无效（web_url_content F3）。
    openExternalUrl(text, {
      allowRelative: true,
      onInvalid: () => ElMessage.error('链接无效，无法打开')
    })
  }

  async function fetchUploadTable(
    params: UploadTableParams,
    signal?: AbortSignal
  ): Promise<Api.Common.PaginatedResponse<UploadRow>> {
    if (!canView.value) {
      return { records: [], current: params.current, size: params.size, total: 0 }
    }

    const payload = await fetchAdminUploads(
      {
        limit: params.size,
        offset: (params.current - 1) * params.size
      },
      signal
    )
    const records = (payload.items || []).map((item) => normalizeRow(item))

    return {
      records,
      current: params.current,
      size: params.size,
      total: Number(payload.total || records.length)
    }
  }

  async function handleFileSelect(uploadFile: UploadFile) {
    if (!uploadFile.raw) {
      return
    }

    uploading.value = true

    try {
      await uploadAdminAsset(uploadFile.raw)
      ElMessage.success('文件上传成功')
      await fetchData()
    } catch (error: any) {
      ElMessage.error(error?.response?.data?.error || '文件上传失败')
    } finally {
      uploading.value = false
    }
  }
</script>

<style scoped lang="scss">
  .page-header {
    display: flex;
    gap: 16px;
    align-items: center;
    justify-content: space-between;
  }

  .page-title {
    font-size: 20px;
    font-weight: 700;
    color: var(--el-text-color-primary);
  }

  .page-subtitle {
    margin-top: 6px;
    font-size: 13px;
    line-height: 1.6;
    color: var(--el-text-color-secondary);
  }

  .page-actions,
  .toolbar,
  .table-actions {
    display: flex;
    gap: 12px;
    align-items: center;
  }

  .toolbar {
    flex-wrap: wrap;
    margin-bottom: 12px;
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
    border-radius: 8px;
  }

  .upload-placeholder {
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 11px;
    font-weight: 700;
    color: var(--el-text-color-secondary);
    letter-spacing: 0.08em;
    background: var(--el-fill-color-light);
    border: 1px solid var(--el-border-color);
  }

  .upload-link {
    color: var(--el-color-primary);
    text-decoration: none;
  }

  @media (width <= 768px) {
    .page-header {
      flex-direction: column;
      align-items: flex-start;
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
