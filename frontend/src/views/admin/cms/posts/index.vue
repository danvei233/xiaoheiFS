<template>
  <div class="art-full-height">
    <ArtSearchBar
      v-if="canView"
      v-model="filters"
      :items="searchItems"
      :span="8"
      :show-expand="false"
      @search="handleSearch"
      @reset="handleReset"
    />

    <ElCard v-loading="loading" class="art-table-card">
      <template #header>
        <div class="page-header">
          <div>
            <div class="page-title">文章管理</div>
            <div class="page-subtitle"> 管理文章内容、发布状态、定时发布时间和分类归属。 </div>
          </div>

          <div class="page-actions">
            <ElButton v-if="canView" :disabled="loading" @click="fetchData()">刷新</ElButton>
            <ElButton v-if="canCreate" type="primary" @click="openCreate">新建文章</ElButton>
          </div>
        </div>
      </template>

      <ElEmpty v-if="!canView" description="你没有查看文章的权限。" />

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
          <template #title="{ row }">
            <div class="post-title-cell">
              <ElTag v-if="row.pinned" size="small" type="danger">置顶</ElTag>
              <span class="post-title-text">{{ row.title || '-' }}</span>
            </div>
          </template>

          <template #status="{ row }">
            <ElTag :type="row.status === 'published' ? 'success' : 'info'">
              {{ row.status === 'published' ? '已发布' : '草稿' }}
            </ElTag>
          </template>

          <template #published_at="{ row }">
            {{ formatDateTime(row.published_at) }}
          </template>

          <template #operation="{ row }">
            <div class="table-actions">
              <ElButton v-if="canUpdate" link type="primary" @click="openEdit(row)">编辑</ElButton>
              <ElButton v-if="canDelete" link type="danger" @click="handleDelete(row)">
                删除
              </ElButton>
            </div>
          </template>
        </ArtTable>
      </template>
    </ElCard>

    <ElDialog
      v-model="dialogVisible"
      :title="dialogMode === 'create' ? '新建文章' : '编辑文章'"
      width="1120px"
      destroy-on-close
      align-center
    >
      <ElForm ref="formRef" :model="dialogForm" :rules="rules" label-position="top">
        <ElRow :gutter="16">
          <ElCol :span="12">
            <ElFormItem label="标题" prop="title">
              <ElInput v-model.trim="dialogForm.title" placeholder="请输入文章标题" />
            </ElFormItem>
          </ElCol>

          <ElCol :span="12">
            <ElFormItem label="分类" prop="category_id">
              <ElSelect v-model="dialogForm.category_id" placeholder="请选择分类">
                <ElOption
                  v-for="item in categoryOptions"
                  :key="item.id"
                  :label="item.name"
                  :value="item.id"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>
        </ElRow>

        <ElRow :gutter="16">
          <ElCol :span="12">
            <ElFormItem label="别名" prop="slug">
              <ElInput v-model.trim="dialogForm.slug" placeholder="请输入 URL 别名" />
            </ElFormItem>
          </ElCol>

          <ElCol :span="12">
            <ElFormItem label="封面地址">
              <ElInput v-model.trim="dialogForm.cover_url" placeholder="/uploads/example.jpg" />
            </ElFormItem>
          </ElCol>
        </ElRow>

        <ElFormItem label="摘要">
          <ElInput
            v-model="dialogForm.summary"
            type="textarea"
            :rows="3"
            placeholder="列表页展示的文章摘要"
          />
        </ElFormItem>

        <ElRow :gutter="16">
          <ElCol :span="6">
            <ElFormItem label="语言" prop="lang">
              <ElSelect v-model="dialogForm.lang" placeholder="请选择语言">
                <ElOption
                  v-for="item in languageOptions"
                  :key="item.value"
                  :label="item.label"
                  :value="item.value"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>

          <ElCol :span="6">
            <ElFormItem label="状态" prop="status">
              <ElSelect v-model="dialogForm.status" placeholder="请选择状态">
                <ElOption label="草稿" value="draft" />
                <ElOption label="已发布" value="published" />
              </ElSelect>
            </ElFormItem>
          </ElCol>

          <ElCol :span="6">
            <ElFormItem label="排序值">
              <ElInputNumber v-model="dialogForm.sort_order" :min="0" style="width: 100%" />
            </ElFormItem>
          </ElCol>

          <ElCol :span="6">
            <ElFormItem label="置顶">
              <ElSwitch v-model="dialogForm.pinned" />
            </ElFormItem>
          </ElCol>
        </ElRow>

        <ElFormItem v-if="dialogForm.status === 'published'" label="发布时间">
          <ElDatePicker
            v-model="dialogForm.published_at"
            type="datetime"
            clearable
            value-format="YYYY-MM-DDTHH:mm:ss"
            placeholder="留空则立即发布"
            style="width: 100%"
          />
        </ElFormItem>

        <ElFormItem label="正文内容" prop="content_html">
          <ElInput
            v-model="dialogForm.content_html"
            type="textarea"
            :rows="18"
            placeholder="请输入正文 HTML 内容"
          />
        </ElFormItem>
      </ElForm>

      <template #footer>
        <div class="dialog-footer">
          <ElButton @click="dialogVisible = false">取消</ElButton>
          <ElButton type="primary" :loading="dialogSubmitting" @click="handleSubmit">
            保存
          </ElButton>
        </div>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import type { CMSCategoryRecord, CMSPostRecord } from '@/services/admin'
  import {
    createCMSPost,
    deleteCMSPost,
    fetchCMSCategories,
    fetchCMSPosts,
    updateCMSPost
  } from '@/services/admin'
  import { useAuth } from '@/hooks/core/useAuth'
  import { useTable } from '@/hooks/core/useTable'
  import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'

  defineOptions({ name: 'CmsPostsPage' })

  interface CategoryOption {
    id: number
    name: string
    lang: string
  }

  interface PostRow {
    id: number | null
    category_id: number | null
    title: string
    slug: string
    summary: string
    content_html: string
    cover_url: string
    lang: string
    status: string
    pinned: boolean
    sort_order: number
    published_at: string | null
    created_at: string
    updated_at: string
  }

  interface PostDialogForm {
    id: number | null
    category_id: number | null
    title: string
    slug: string
    summary: string
    content_html: string
    cover_url: string
    lang: string
    status: string
    pinned: boolean
    sort_order: number
    published_at: string
  }

  interface FilterState {
    category_id?: number
    status?: string
  }

  interface PostTableParams extends Api.Common.CommonSearchParams, FilterState {}

  const languageOptions = [
    { label: '简体中文', value: 'zh-CN' },
    { label: '英文', value: 'en-US' }
  ]

  const { hasAuth } = useAuth()

  const initialized = ref(false)
  const dialogVisible = ref(false)
  const dialogSubmitting = ref(false)
  const dialogMode = ref<'create' | 'edit'>('create')

  const categoryOptions = ref<CategoryOption[]>([])
  const filters = ref<FilterState>({
    category_id: undefined,
    status: undefined
  })
  const dialogForm = reactive<PostDialogForm>(createDefaultDialogForm())
  const formRef = ref<FormInstance>()

  const searchItems = computed(() => [
    {
      key: 'category_id',
      label: '分类',
      type: 'select',
      props: {
        clearable: true,
        placeholder: '全部分类',
        options: categoryOptions.value.map((item) => ({ label: item.name, value: item.id }))
      }
    },
    {
      key: 'status',
      label: '状态',
      type: 'select',
      props: {
        clearable: true,
        placeholder: '全部状态',
        options: [
          { label: '草稿', value: 'draft' },
          { label: '已发布', value: 'published' }
        ]
      }
    }
  ])

  const {
    columnChecks,
    columns,
    data: tableData,
    loading,
    pagination,
    searchParams,
    getData,
    fetchData,
    resetSearchParams,
    handleSizeChange: handlePageSizeChange,
    handleCurrentChange: handlePageCurrentChange
  } = useTable({
    core: {
      apiFn: fetchPostTable,
      apiParams: { current: 1, size: 20, ...filters.value },
      immediate: false,
      columnsFactory: () => [
        { prop: 'id', label: 'ID', width: 80 },
        { prop: 'title', label: '标题', minWidth: 260, useSlot: true },
        {
          prop: 'category_id',
          label: '分类',
          minWidth: 160,
          formatter: (row: PostRow) => getCategoryName(row.category_id)
        },
        { prop: 'status', label: '状态', width: 120, useSlot: true },
        { prop: 'published_at', label: '发布时间', minWidth: 180, useSlot: true },
        {
          prop: 'updated_at',
          label: '更新时间',
          minWidth: 180,
          formatter: (row: PostRow) => formatDateTime(row.updated_at)
        },
        { prop: 'operation', label: '操作', width: 150, fixed: 'right', useSlot: true }
      ]
    }
  })

  const canView = computed(() => hasAuth('cms_post.list'))
  const canCreate = computed(() => hasAuth('cms_post.create'))
  const canUpdate = computed(() => hasAuth('cms_post.update'))
  const canDelete = computed(() => hasAuth('cms_post.delete'))

  const rules = computed<FormRules>(() => ({
    title: [{ required: true, message: '请输入标题', trigger: 'blur' }],
    category_id: [{ required: true, message: '请选择分类', trigger: 'change' }]
  }))

  watch(
    canView,
    (value) => {
      if (value && !initialized.value) {
        initialized.value = true
        initializePage()
      }
    },
    { immediate: true }
  )

  function createDefaultDialogForm(): PostDialogForm {
    return {
      id: null,
      category_id: null,
      title: '',
      slug: '',
      summary: '',
      content_html: '',
      cover_url: '',
      lang: 'zh-CN',
      status: 'draft',
      pinned: false,
      sort_order: 0,
      published_at: ''
    }
  }

  function resetDialogForm() {
    Object.assign(dialogForm, createDefaultDialogForm())
  }

  function normalizeNullableNumber(value: unknown): number | null {
    if (value === '' || value === null || value === undefined) {
      return null
    }

    const parsed = Number(value)
    return Number.isFinite(parsed) ? parsed : null
  }

  function normalizeNullableString(value: unknown): string | null {
    if (value === '' || value === null || value === undefined) {
      return null
    }

    return String(value)
  }

  function normalizeCategory(item?: CMSCategoryRecord): CategoryOption | null {
    const id = normalizeNullableNumber(item?.id)
    if (!id) {
      return null
    }

    return {
      id,
      name: String(item?.name || item?.key || `分类 ${id}`),
      lang: String(item?.lang || 'zh-CN')
    }
  }

  function normalizePost(item?: CMSPostRecord): PostRow {
    return {
      id: normalizeNullableNumber(item?.id),
      category_id: normalizeNullableNumber(item?.category_id),
      title: String(item?.title || ''),
      slug: String(item?.slug || ''),
      summary: String(item?.summary || ''),
      content_html: String(item?.content_html || ''),
      cover_url: String(item?.cover_url || ''),
      lang: String(item?.lang || 'zh-CN'),
      status: String(item?.status || 'draft'),
      pinned: Boolean(item?.pinned),
      sort_order: Number(item?.sort_order || 0),
      published_at: normalizeNullableString(item?.published_at),
      created_at: String(item?.created_at || ''),
      updated_at: String(item?.updated_at || '')
    }
  }

  function getCategoryName(categoryId: number | null) {
    if (!categoryId) {
      return '-'
    }

    return categoryOptions.value.find((item) => item.id === categoryId)?.name || `#${categoryId}`
  }

  function formatDateTime(value?: string | null) {
    if (!value) {
      return '-'
    }

    const date = new Date(value)
    return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN')
  }

  function toPickerValue(value?: string | null) {
    if (!value) {
      return ''
    }

    const date = new Date(value)
    if (Number.isNaN(date.getTime())) {
      return ''
    }

    const pad = (num: number) => String(num).padStart(2, '0')
    return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
  }

  function toISOStringOrEmpty(value: string) {
    const trimmed = String(value || '').trim()
    if (!trimmed) {
      return ''
    }

    const date = new Date(trimmed)
    return Number.isNaN(date.getTime()) ? '' : date.toISOString()
  }

  async function initializePage() {
    await fetchCategories()
    await getData()
  }

  async function fetchCategories() {
    const payload = await fetchCMSCategories()
    categoryOptions.value = (payload.items || [])
      .map((item) => normalizeCategory(item))
      .filter(Boolean) as CategoryOption[]
  }

  async function fetchPostTable(
    params: PostTableParams,
    signal?: AbortSignal
  ): Promise<Api.Common.PaginatedResponse<PostRow>> {
    if (!canView.value) {
      return { records: [], current: params.current, size: params.size, total: 0 }
    }

    const payload = await fetchCMSPosts(
      {
        limit: params.size,
        offset: (params.current - 1) * params.size,
        category_id: params.category_id || undefined,
        status: params.status || undefined
      },
      signal
    )

    const records = (payload.items || []).map((item) => normalizePost(item))
    return {
      records,
      current: params.current,
      size: params.size,
      total: Number(payload.total || records.length)
    }
  }

  async function handleSearch(params: FilterState) {
    Object.assign(searchParams, params)
    await getData()
  }

  async function handleReset() {
    filters.value = { category_id: undefined, status: undefined }
    await resetSearchParams()
  }

  function openCreate() {
    dialogMode.value = 'create'
    resetDialogForm()
    dialogVisible.value = true
    nextTick(() => formRef.value?.clearValidate())
  }

  function openEdit(row: PostRow) {
    dialogMode.value = 'edit'
    Object.assign(dialogForm, {
      id: row.id,
      category_id: row.category_id,
      title: row.title,
      slug: row.slug,
      summary: row.summary,
      content_html: row.content_html,
      cover_url: row.cover_url,
      lang: row.lang || 'zh-CN',
      status: row.status || 'draft',
      pinned: row.pinned,
      sort_order: row.sort_order,
      published_at: toPickerValue(row.published_at)
    })
    dialogVisible.value = true
    nextTick(() => formRef.value?.clearValidate())
  }

  async function handleSubmit() {
    if (!formRef.value) {
      return
    }

    const valid = await formRef.value.validate().catch(() => false)
    if (!valid) {
      return
    }

    dialogSubmitting.value = true

    try {
      const payload: Record<string, unknown> = {
        category_id: Number(dialogForm.category_id || 0),
        title: String(dialogForm.title || '').trim(),
        slug: String(dialogForm.slug || '').trim(),
        summary: String(dialogForm.summary || '').trim(),
        content_html: String(dialogForm.content_html || ''),
        cover_url: String(dialogForm.cover_url || '').trim(),
        lang: String(dialogForm.lang || 'zh-CN').trim() || 'zh-CN',
        status: String(dialogForm.status || 'draft').trim() || 'draft',
        pinned: Boolean(dialogForm.pinned),
        sort_order: Number(dialogForm.sort_order || 0)
      }

      if (dialogForm.status === 'published' && dialogForm.published_at) {
        payload.published_at = toISOStringOrEmpty(dialogForm.published_at)
      }

      if (dialogMode.value === 'create') {
        await createCMSPost(payload)
        ElMessage.success('文章创建成功')
      } else if (dialogForm.id) {
        await updateCMSPost(dialogForm.id, payload)
        ElMessage.success('文章更新成功')
      }

      dialogVisible.value = false
      await fetchData()
    } catch (error: any) {
      ElMessage.error(error?.response?.data?.error || '操作失败')
    } finally {
      dialogSubmitting.value = false
    }
  }

  async function handleDelete(row: PostRow) {
    if (!row.id) {
      return
    }

    try {
      await ElMessageBox.confirm(`确定要删除文章“${row.title}”吗？该操作不可恢复。`, '删除文章', {
        type: 'warning'
      })

      await deleteCMSPost(row.id)
      ElMessage.success('文章删除成功')
      await fetchData()
    } catch (error: any) {
      if (error === 'cancel' || error === 'close') {
        return
      }

      ElMessage.error(error?.response?.data?.error || '删除文章失败')
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
  .table-actions,
  .dialog-footer {
    display: flex;
    gap: 12px;
    align-items: center;
  }

  .toolbar {
    flex-wrap: wrap;
    margin-bottom: 12px;
  }

  .toolbar-select {
    width: 180px;
  }

  .post-title-cell {
    display: flex;
    gap: 8px;
    align-items: center;
    min-width: 0;
  }

  .post-title-text {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .table-actions {
    justify-content: flex-end;
  }

  .dialog-footer {
    justify-content: flex-end;
  }

  @media (width <= 768px) {
    .page-header {
      flex-direction: column;
      align-items: flex-start;
    }

    .page-actions,
    .toolbar {
      flex-wrap: wrap;
      width: 100%;
    }

    .toolbar-select {
      width: 100%;
    }
  }
</style>
