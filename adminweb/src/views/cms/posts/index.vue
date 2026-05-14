<template>
  <div class="art-full-height">
    <ElCard v-loading="loading" class="art-table-card">
      <template #header>
        <div class="page-header">
          <div>
            <div class="page-title">文章管理</div>
            <div class="page-subtitle"> 管理文章内容、发布状态、定时发布时间和分类归属。 </div>
          </div>

          <div class="page-actions">
            <ElButton v-if="canView" :disabled="loading" @click="fetchData">刷新</ElButton>
            <ElButton v-if="canCreate" type="primary" @click="openCreate">新建文章</ElButton>
          </div>
        </div>
      </template>

      <ElEmpty v-if="!canView" description="你没有查看文章的权限。" />

      <template v-else>
        <div class="toolbar">
          <ElSelect
            v-model="filters.category_id"
            clearable
            placeholder="按分类筛选"
            class="toolbar-select"
            @change="handleFilterChange"
          >
            <ElOption
              v-for="item in categoryOptions"
              :key="item.id"
              :label="item.name"
              :value="item.id"
            />
          </ElSelect>

          <ElSelect
            v-model="filters.status"
            clearable
            placeholder="按状态筛选"
            class="toolbar-select"
            @change="handleFilterChange"
          >
            <ElOption label="草稿" value="draft" />
            <ElOption label="已发布" value="published" />
          </ElSelect>

          <ElSelect
            v-model="filters.lang"
            clearable
            placeholder="按语言筛选"
            class="toolbar-select"
            @change="handleFilterChange"
          >
            <ElOption
              v-for="item in languageOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </ElSelect>
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
              <ElInput
                v-model.trim="dialogForm.title"
                :maxlength="INPUT_LIMITS.CMS_TITLE"
                placeholder="请输入文章标题"
              />
            </ElFormItem>
          </ElCol>

          <ElCol :span="12">
            <ElFormItem label="分类" prop="category_id">
              <ElSelect v-model="dialogForm.category_id" placeholder="请选择分类">
                <ElOption
                  v-for="item in filteredDialogCategories"
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
              <ElInput
                v-model.trim="dialogForm.slug"
                :maxlength="INPUT_LIMITS.CMS_SLUG"
                placeholder="请输入 URL 别名"
              />
            </ElFormItem>
          </ElCol>

          <ElCol :span="12">
            <ElFormItem label="封面地址">
              <ElInput
                v-model.trim="dialogForm.cover_url"
                :maxlength="INPUT_LIMITS.URL"
                placeholder="/uploads/example.jpg"
              />
            </ElFormItem>
          </ElCol>
        </ElRow>

        <ElFormItem label="摘要">
          <ElInput
            v-model="dialogForm.summary"
            type="textarea"
            :rows="3"
            :maxlength="INPUT_LIMITS.CMS_SUMMARY"
            show-word-limit
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

        <ElFormItem label="发布时间">
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
          <ArtWangEditor
            v-model="dialogForm.content_html"
            height="360px"
            placeholder="请输入正文 HTML 内容"
            :excludeKeys="['fontFamily', 'uploadImage']"
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
  import type { CMSCategoryRecord, CMSPostRecord } from '@/api/admin'
  import {
    createCMSPost,
    deleteCMSPost,
    fetchCMSCategories,
    fetchCMSPosts,
    hasAdminPermission,
    updateCMSPost
  } from '@/api/admin'
  import ArtWangEditor from '@/components/core/forms/art-wang-editor/index.vue'
  import { useTableColumns } from '@/hooks/core/useTableColumns'
  import { useUserStore } from '@/store/modules/user'
  import { INPUT_LIMITS } from '@/utils/constants'
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
    lang?: string
  }

  const languageOptions = [
    { label: '简体中文', value: 'zh-CN' },
    { label: '英文', value: 'en-US' }
  ]

  const userStore = useUserStore()
  const { info } = storeToRefs(userStore)

  const loading = ref(false)
  const initialized = ref(false)
  const dialogVisible = ref(false)
  const dialogSubmitting = ref(false)
  const dialogMode = ref<'create' | 'edit'>('create')

  const tableData = ref<PostRow[]>([])
  const categoryOptions = ref<CategoryOption[]>([])
  const filters = reactive<FilterState>({
    category_id: undefined,
    status: undefined,
    lang: undefined
  })
  const dialogForm = reactive<PostDialogForm>(createDefaultDialogForm())
  const formRef = ref<FormInstance>()

  const pagination = reactive({
    current: 1,
    size: 20,
    total: 0
  })

  const { columnChecks, columns } = useTableColumns<PostRow>(() => [
    { prop: 'id', label: 'ID', width: 80 },
    { prop: 'title', label: '标题', minWidth: 260, useSlot: true },
    {
      prop: 'category_id',
      label: '分类',
      minWidth: 160,
      formatter: (row: PostRow) => getCategoryName(row.category_id)
    },
    { prop: 'lang', label: '语言', width: 140 },
    { prop: 'status', label: '状态', width: 120, useSlot: true },
    { prop: 'published_at', label: '发布时间', minWidth: 180, useSlot: true },
    {
      prop: 'updated_at',
      label: '更新时间',
      minWidth: 180,
      formatter: (row: PostRow) => formatDateTime(row.updated_at)
    },
    { prop: 'operation', label: '操作', width: 150, fixed: 'right', useSlot: true }
  ])

  const canView = computed(() => hasAdminPermission(info.value?.buttons, ['cms_post.list']))
  const canCreate = computed(() => hasAdminPermission(info.value?.buttons, ['cms_post.create']))
  const canUpdate = computed(() => hasAdminPermission(info.value?.buttons, ['cms_post.update']))
  const canDelete = computed(() => hasAdminPermission(info.value?.buttons, ['cms_post.delete']))

  const filteredDialogCategories = computed(() => {
    const lang = String(dialogForm.lang || '').trim()
    if (!lang) {
      return categoryOptions.value
    }

    return categoryOptions.value.filter((item) => item.lang === lang)
  })

  const rules = computed<FormRules>(() => ({
    title: [
      { required: true, message: '请输入标题', trigger: 'blur' },
      {
        max: INPUT_LIMITS.CMS_TITLE,
        message: `标题长度不能超过 ${INPUT_LIMITS.CMS_TITLE} 个字符`,
        trigger: 'blur'
      }
    ],
    slug: [
      { required: true, message: '请输入别名', trigger: 'blur' },
      {
        max: INPUT_LIMITS.CMS_SLUG,
        message: `别名长度不能超过 ${INPUT_LIMITS.CMS_SLUG} 个字符`,
        trigger: 'blur'
      }
    ],
    category_id: [{ required: true, message: '请选择分类', trigger: 'change' }],
    lang: [{ required: true, message: '请选择语言', trigger: 'change' }],
    status: [{ required: true, message: '请选择状态', trigger: 'change' }]
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

  watch(
    () => dialogForm.lang,
    (lang) => {
      if (!lang) {
        return
      }

      if (
        dialogForm.category_id &&
        !filteredDialogCategories.value.some((item) => item.id === dialogForm.category_id)
      ) {
        dialogForm.category_id = null
      }
    }
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
    await Promise.all([fetchCategories(), fetchData()])
  }

  async function fetchCategories() {
    const payload = await fetchCMSCategories()
    categoryOptions.value = (payload.items || [])
      .map((item) => normalizeCategory(item))
      .filter(Boolean) as CategoryOption[]
  }

  async function fetchData() {
    if (!canView.value) {
      return
    }

    loading.value = true

    try {
      const payload = await fetchCMSPosts({
        limit: pagination.size,
        offset: (pagination.current - 1) * pagination.size,
        category_id: filters.category_id || undefined,
        status: filters.status || undefined,
        lang: filters.lang || undefined
      })

      tableData.value = (payload.items || []).map((item) => normalizePost(item))
      pagination.total = Number(payload.total || 0)
    } finally {
      loading.value = false
    }
  }

  function handleFilterChange() {
    pagination.current = 1
    fetchData()
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
      const payload = {
        category_id: Number(dialogForm.category_id || 0),
        title: String(dialogForm.title || '').trim(),
        slug: String(dialogForm.slug || '').trim(),
        summary: String(dialogForm.summary || '').trim(),
        content_html: String(dialogForm.content_html || ''),
        cover_url: String(dialogForm.cover_url || '').trim(),
        lang: String(dialogForm.lang || 'zh-CN').trim() || 'zh-CN',
        status: String(dialogForm.status || 'draft').trim() || 'draft',
        pinned: Boolean(dialogForm.pinned),
        sort_order: Number(dialogForm.sort_order || 0),
        published_at:
          dialogForm.status === 'published' ? toISOStringOrEmpty(dialogForm.published_at) : ''
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
  .table-actions,
  .dialog-footer {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .toolbar {
    margin-bottom: 12px;
    flex-wrap: wrap;
  }

  .toolbar-select {
    width: 180px;
  }

  .post-title-cell {
    display: flex;
    align-items: center;
    gap: 8px;
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

  @media (max-width: 768px) {
    .page-header {
      align-items: flex-start;
      flex-direction: column;
    }

    .page-actions,
    .toolbar {
      width: 100%;
      flex-wrap: wrap;
    }

    .toolbar-select {
      width: 100%;
    }
  }
</style>
