<template>
  <div class="art-full-height">
    <ElCard v-loading="loading" class="art-table-card">
      <template #header>
        <div class="page-header">
          <div>
            <div class="page-title">区块管理</div>
            <div class="page-subtitle">
              管理可复用页面区块、JSON 数据、自定义 HTML 和显示状态。
            </div>
          </div>

          <div class="page-actions">
            <ElButton v-if="canView" :disabled="loading" @click="fetchData">刷新</ElButton>
            <ElButton v-if="canCreate" type="primary" @click="openCreate">新建区块</ElButton>
          </div>
        </div>
      </template>

      <ElEmpty v-if="!canView" description="你没有查看区块的权限。" />

      <template v-else>
        <div class="toolbar">
          <ElSelect
            v-model="filters.page"
            clearable
            filterable
            placeholder="按页面筛选"
            class="toolbar-select"
            @change="fetchData"
          >
            <ElOption
              v-for="item in pageOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </ElSelect>

          <ElSelect
            v-model="filters.type"
            clearable
            filterable
            placeholder="按类型筛选"
            class="toolbar-select"
          >
            <ElOption
              v-for="item in allTypeOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </ElSelect>

          <ElSelect
            v-model="filters.lang"
            clearable
            placeholder="按语言筛选"
            class="toolbar-select"
            @change="fetchData"
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

        <ArtTable row-key="id" :loading="loading" :data="filteredRows" :columns="columns">
          <template #visible="{ row }">
            <ElSwitch
              :model-value="row.visible"
              :disabled="!canUpdate || row.switching"
              :loading="row.switching"
              @change="handleToggleVisible(row, $event)"
            />
          </template>

          <template #payload="{ row }">
            <div class="payload-summary">
              <div>{{ getPayloadSummary(row) }}</div>
              <ElTag size="small" type="info">
                {{ row.type === 'custom_html' ? 'HTML' : 'JSON' }}
              </ElTag>
            </div>
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
      :title="dialogMode === 'create' ? '新建区块' : '编辑区块'"
      width="980px"
      destroy-on-close
      align-center
    >
      <ElForm ref="formRef" :model="dialogForm" :rules="rules" label-position="top">
        <ElRow :gutter="16">
          <ElCol :span="8">
            <ElFormItem label="页面" prop="page">
              <ElSelect
                v-model="dialogForm.page"
                filterable
                allow-create
                default-first-option
                placeholder="home"
              >
                <ElOption
                  v-for="item in pageOptions"
                  :key="item.value"
                  :label="item.label"
                  :value="item.value"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>

          <ElCol :span="8">
            <ElFormItem label="类型" prop="type">
              <ElSelect
                v-model="dialogForm.type"
                filterable
                allow-create
                default-first-option
                placeholder="hero"
              >
                <ElOption
                  v-for="item in dialogTypeOptions"
                  :key="item.value"
                  :label="item.label"
                  :value="item.value"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>

          <ElCol :span="8">
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
        </ElRow>

        <ElRow :gutter="16">
          <ElCol :span="12">
            <ElFormItem label="标题">
              <ElInput
                v-model="dialogForm.title"
                :maxlength="INPUT_LIMITS.CMS_TITLE"
                placeholder="请输入区块标题"
              />
            </ElFormItem>
          </ElCol>

          <ElCol :span="12">
            <ElFormItem label="副标题">
              <ElInput
                v-model="dialogForm.subtitle"
                :maxlength="INPUT_LIMITS.CMS_SUBTITLE"
                placeholder="可选副标题"
              />
            </ElFormItem>
          </ElCol>
        </ElRow>

        <ElRow :gutter="16">
          <ElCol :span="12">
            <ElFormItem label="排序值">
              <ElInputNumber v-model="dialogForm.sort_order" :min="0" style="width: 100%" />
            </ElFormItem>
          </ElCol>

          <ElCol :span="12">
            <ElFormItem label="是否显示">
              <ElSwitch v-model="dialogForm.visible" />
            </ElFormItem>
          </ElCol>
        </ElRow>

        <ElFormItem v-if="dialogForm.type !== 'custom_html'" label="内容 JSON" prop="content_json">
          <ElInput
            v-model="dialogForm.content_json"
            type="textarea"
            :rows="12"
            :maxlength="INPUT_LIMITS.CMS_JSON"
            show-word-limit
            placeholder='{"items":[...]}'
          />
        </ElFormItem>

        <ElFormItem v-if="dialogForm.type === 'custom_html'" label="自定义 HTML" prop="custom_html">
          <ElInput
            v-model="dialogForm.custom_html"
            type="textarea"
            :rows="12"
            :maxlength="INPUT_LIMITS.CMS_HTML"
            show-word-limit
            placeholder="<section>...</section>"
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
  import type { CMSBlockRecord } from '@/api/admin'
  import {
    createCMSBlock,
    deleteCMSBlock,
    fetchCMSBlocks,
    hasAdminPermission,
    updateCMSBlock
  } from '@/api/admin'
  import { useTableColumns } from '@/hooks/core/useTableColumns'
  import { useUserStore } from '@/store/modules/user'
  import { INPUT_LIMITS } from '@/utils/constants'
  import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'

  defineOptions({ name: 'CmsBlocksPage' })

  interface BlockRow {
    id: number | null
    page: string
    type: string
    title: string
    subtitle: string
    content_json: string
    custom_html: string
    lang: string
    visible: boolean
    sort_order: number
    created_at: string
    updated_at: string
    switching?: boolean
  }

  interface BlockDialogForm {
    id: number | null
    page: string
    type: string
    title: string
    subtitle: string
    content_json: string
    custom_html: string
    lang: string
    visible: boolean
    sort_order: number
  }

  interface FilterState {
    page?: string
    type?: string
    lang?: string
  }

  interface OptionItem {
    label: string
    value: string
  }

  const languageOptions: OptionItem[] = [
    { label: '简体中文', value: 'zh-CN' },
    { label: '英文', value: 'en-US' }
  ]

  const pageOptions: OptionItem[] = [
    { label: '首页', value: 'home' },
    { label: '产品页', value: 'products' },
    { label: '文档页', value: 'docs' },
    { label: '公告页', value: 'announcements' },
    { label: '活动页', value: 'activities' },
    { label: '教程页', value: 'tutorials' },
    { label: '帮助页', value: 'help' },
    { label: '页脚', value: 'footer' }
  ]

  const sharedTypeOptions: OptionItem[] = [
    { label: '首屏横幅', value: 'hero' },
    { label: '功能特色', value: 'features' },
    { label: '行动引导', value: 'cta' },
    { label: '产品列表', value: 'products' },
    { label: '计算器', value: 'calculator' },
    { label: '价格模块', value: 'pricing' },
    { label: '对比模块', value: 'comparison' },
    { label: '页脚', value: 'footer' },
    { label: '文章列表', value: 'posts' },
    { label: '资源模块', value: 'resources' },
    { label: '自定义 HTML', value: 'custom_html' }
  ]

  const helpTypeOptions: OptionItem[] = [
    { label: '帮助页首屏', value: 'help_hero' },
    { label: '帮助页快捷入口', value: 'help_actions' },
    { label: '帮助页常见问题', value: 'help_faq' },
    { label: '帮助页联系方式', value: 'help_contact' },
    { label: '自定义 HTML', value: 'custom_html' }
  ]

  const userStore = useUserStore()
  const { info } = storeToRefs(userStore)

  const loading = ref(false)
  const initialized = ref(false)
  const dialogVisible = ref(false)
  const dialogSubmitting = ref(false)
  const dialogMode = ref<'create' | 'edit'>('create')

  const filters = reactive<FilterState>({
    page: undefined,
    type: undefined,
    lang: undefined
  })
  const tableData = ref<BlockRow[]>([])
  const dialogForm = reactive<BlockDialogForm>(createDefaultDialogForm())
  const formRef = ref<FormInstance>()

  const { columnChecks, columns } = useTableColumns<BlockRow>(() => [
    { prop: 'id', label: 'ID', width: 80 },
    { prop: 'page', label: '页面', width: 140 },
    { prop: 'type', label: '类型', width: 160 },
    { prop: 'title', label: '标题', minWidth: 220, showOverflowTooltip: true },
    { prop: 'payload', label: '内容摘要', minWidth: 220, useSlot: true },
    { prop: 'lang', label: '语言', width: 140 },
    { prop: 'visible', label: '显示', width: 110, useSlot: true },
    {
      prop: 'updated_at',
      label: '更新时间',
      minWidth: 180,
      formatter: (row: BlockRow) => formatDateTime(row.updated_at)
    },
    { prop: 'operation', label: '操作', width: 150, fixed: 'right', useSlot: true }
  ])

  const canView = computed(() => hasAdminPermission(info.value?.buttons, ['cms_block.list']))
  const canCreate = computed(() => hasAdminPermission(info.value?.buttons, ['cms_block.create']))
  const canUpdate = computed(() => hasAdminPermission(info.value?.buttons, ['cms_block.update']))
  const canDelete = computed(() => hasAdminPermission(info.value?.buttons, ['cms_block.delete']))

  const allTypeOptions = computed(() => {
    const map = new Map<string, OptionItem>()
    ;[...sharedTypeOptions, ...helpTypeOptions].forEach((item) => {
      map.set(item.value, item)
    })
    return Array.from(map.values())
  })

  const dialogTypeOptions = computed(() =>
    dialogForm.page === 'help' ? helpTypeOptions : sharedTypeOptions
  )

  const filteredRows = computed(() => {
    const type = String(filters.type || '').trim()
    if (!type) {
      return tableData.value
    }

    return tableData.value.filter((row) => row.type === type)
  })

  const rules = computed<FormRules>(() => ({
    page: [{ required: true, message: '请选择页面', trigger: 'change' }],
    type: [{ required: true, message: '请选择区块类型', trigger: 'change' }],
    lang: [{ required: true, message: '请选择语言', trigger: 'change' }]
  }))

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

  watch(
    () => dialogForm.page,
    (page) => {
      if (!page) {
        return
      }

      if (!dialogTypeOptions.value.some((item) => item.value === dialogForm.type)) {
        dialogForm.type = dialogTypeOptions.value[0]?.value || ''
      }
    }
  )

  function createDefaultDialogForm(): BlockDialogForm {
    return {
      id: null,
      page: 'home',
      type: 'hero',
      title: '',
      subtitle: '',
      content_json: '',
      custom_html: '',
      lang: 'zh-CN',
      visible: true,
      sort_order: 0
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

  function normalizeRow(item?: CMSBlockRecord): BlockRow {
    return {
      id: normalizeNullableNumber(item?.id),
      page: String(item?.page || ''),
      type: String(item?.type || ''),
      title: String(item?.title || ''),
      subtitle: String(item?.subtitle || ''),
      content_json: String(item?.content_json || ''),
      custom_html: String(item?.custom_html || ''),
      lang: String(item?.lang || 'zh-CN'),
      visible: Boolean(item?.visible),
      sort_order: Number(item?.sort_order || 0),
      created_at: String(item?.created_at || ''),
      updated_at: String(item?.updated_at || ''),
      switching: false
    }
  }

  function formatDateTime(value?: string | null) {
    if (!value) {
      return '-'
    }

    const date = new Date(value)
    return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN')
  }

  function getPayloadSummary(row: BlockRow) {
    const raw = row.type === 'custom_html' ? row.custom_html : row.content_json
    const trimmed = String(raw || '').trim()
    if (!trimmed) {
      return '暂无内容'
    }

    return trimmed.length > 80 ? `${trimmed.slice(0, 80)}...` : trimmed
  }

  async function fetchData() {
    if (!canView.value) {
      return
    }

    loading.value = true

    try {
      const payload = await fetchCMSBlocks({
        page: filters.page || undefined,
        lang: filters.lang || undefined
      })
      tableData.value = (payload.items || []).map((item) => normalizeRow(item))
    } finally {
      loading.value = false
    }
  }

  function openCreate() {
    dialogMode.value = 'create'
    resetDialogForm()
    dialogVisible.value = true
    nextTick(() => formRef.value?.clearValidate())
  }

  function openEdit(row: BlockRow) {
    dialogMode.value = 'edit'
    Object.assign(dialogForm, {
      id: row.id,
      page: row.page,
      type: row.type,
      title: row.title,
      subtitle: row.subtitle,
      content_json: row.content_json,
      custom_html: row.custom_html,
      lang: row.lang || 'zh-CN',
      visible: row.visible,
      sort_order: row.sort_order
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

    if (dialogForm.type !== 'custom_html' && dialogForm.content_json.trim()) {
      try {
        JSON.parse(dialogForm.content_json)
      } catch {
        ElMessage.error('内容 JSON 格式不正确')
        return
      }
    }

    dialogSubmitting.value = true

    try {
      const payload = {
        page: String(dialogForm.page || '').trim(),
        type: String(dialogForm.type || '').trim(),
        title: String(dialogForm.title || ''),
        subtitle: String(dialogForm.subtitle || ''),
        content_json:
          dialogForm.type === 'custom_html' ? '' : String(dialogForm.content_json || ''),
        custom_html: dialogForm.type === 'custom_html' ? String(dialogForm.custom_html || '') : '',
        lang: String(dialogForm.lang || 'zh-CN').trim() || 'zh-CN',
        visible: Boolean(dialogForm.visible),
        sort_order: Number(dialogForm.sort_order || 0)
      }

      if (dialogMode.value === 'create') {
        await createCMSBlock(payload)
        ElMessage.success('区块创建成功')
      } else if (dialogForm.id) {
        await updateCMSBlock(dialogForm.id, payload)
        ElMessage.success('区块更新成功')
      }

      dialogVisible.value = false
      await fetchData()
    } finally {
      dialogSubmitting.value = false
    }
  }

  async function handleToggleVisible(row: BlockRow, checked: string | number | boolean) {
    if (!row.id) {
      return
    }

    row.switching = true

    try {
      await updateCMSBlock(row.id, { visible: Boolean(checked) })
      row.visible = Boolean(checked)
      ElMessage.success('显示状态已更新')
    } catch (error: any) {
      ElMessage.error(error?.response?.data?.error || '更新显示状态失败')
    } finally {
      row.switching = false
    }
  }

  async function handleDelete(row: BlockRow) {
    if (!row.id) {
      return
    }

    try {
      await ElMessageBox.confirm(
        `确定要删除区块“${row.title || row.type}”吗？该操作不可恢复。`,
        '删除区块',
        {
          type: 'warning'
        }
      )

      await deleteCMSBlock(row.id)
      ElMessage.success('区块删除成功')
      await fetchData()
    } catch (error: any) {
      if (error === 'cancel' || error === 'close') {
        return
      }

      ElMessage.error(error?.response?.data?.error || '删除区块失败')
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
  .dialog-footer,
  .payload-summary {
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

  .payload-summary {
    justify-content: space-between;
    min-width: 0;
  }

  .payload-summary > div:first-child {
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
