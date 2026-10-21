<template>
  <div class="art-full-height">
    <ElCard v-loading="loading" class="art-table-card">
      <template #header>
        <div class="page-header">
          <div>
            <div class="page-title">内容分类</div>
            <div class="page-subtitle"> 管理文章分类的标识、语言版本、排序和显示状态。 </div>
          </div>

          <div class="page-actions">
            <ElButton v-if="canView" :disabled="loading" @click="fetchData">刷新</ElButton>
            <ElButton v-if="canCreate" type="primary" @click="openCreate">新建分类</ElButton>
          </div>
        </div>
      </template>

      <ElEmpty v-if="!canView" description="你没有查看内容分类的权限。" />

      <template v-else>
        <div class="toolbar">
          <ElSelect
            v-model="langFilter"
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

          <ElInput
            v-model="keyword"
            clearable
            placeholder="按标识或名称搜索"
            class="toolbar-search"
          />
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
      :title="dialogMode === 'create' ? '新建分类' : '编辑分类'"
      width="520px"
      destroy-on-close
      align-center
    >
      <ElForm ref="formRef" :model="dialogForm" :rules="rules" label-position="top">
        <ElFormItem label="分类标识" prop="key">
          <ElInput
            v-model.trim="dialogForm.key"
            :disabled="dialogMode === 'edit'"
            :maxlength="INPUT_LIMITS.CMS_KEY"
            placeholder="docs"
          />
        </ElFormItem>

        <ElFormItem label="显示名称" prop="name">
          <ElInput
            v-model.trim="dialogForm.name"
            :maxlength="INPUT_LIMITS.CMS_NAME"
            placeholder="文档中心"
          />
        </ElFormItem>

        <ElRow :gutter="12">
          <ElCol :span="12">
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

          <ElCol :span="12">
            <ElFormItem label="排序值" prop="sort_order">
              <ElInputNumber v-model="dialogForm.sort_order" :min="0" style="width: 100%" />
            </ElFormItem>
          </ElCol>
        </ElRow>

        <ElFormItem label="是否显示">
          <ElSwitch v-model="dialogForm.visible" />
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
  import type { CMSCategoryRecord } from '@/api/admin'
  import {
    createCMSCategory,
    deleteCMSCategory,
    fetchCMSCategories,
    hasAdminPermission,
    updateCMSCategory
  } from '@/api/admin'
  import { useTableColumns } from '@/hooks/core/useTableColumns'
  import { useUserStore } from '@/store/modules/user'
  import { INPUT_LIMITS } from '@/utils/constants'
  import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'

  defineOptions({ name: 'CmsCategoriesPage' })

  interface CategoryRow {
    id: number | null
    key: string
    name: string
    lang: string
    sort_order: number
    visible: boolean
    created_at: string
    updated_at: string
    switching?: boolean
  }

  interface CategoryDialogForm {
    id: number | null
    key: string
    name: string
    lang: string
    sort_order: number
    visible: boolean
  }

  const languageOptions = [
    { label: '简体中文', value: 'zh-CN' },
    { label: '英文', value: 'en-US' }
  ]

  const userStore = useUserStore()
  const { info } = storeToRefs(userStore)

  const loading = ref(false)
  const initialized = ref(false)
  const keyword = ref('')
  const langFilter = ref<string | undefined>()
  const dialogVisible = ref(false)
  const dialogSubmitting = ref(false)
  const dialogMode = ref<'create' | 'edit'>('create')

  const tableData = ref<CategoryRow[]>([])
  const dialogForm = reactive<CategoryDialogForm>(createDefaultDialogForm())
  const formRef = ref<FormInstance>()

  const { columnChecks, columns } = useTableColumns<CategoryRow>(() => [
    { prop: 'id', label: 'ID', width: 80 },
    { prop: 'key', label: '分类标识', minWidth: 160, showOverflowTooltip: true },
    { prop: 'name', label: '显示名称', minWidth: 180, showOverflowTooltip: true },
    { prop: 'lang', label: '语言', width: 140 },
    { prop: 'sort_order', label: '排序', width: 90 },
    { prop: 'visible', label: '显示', width: 110, useSlot: true },
    {
      prop: 'updated_at',
      label: '更新时间',
      minWidth: 180,
      formatter: (row: CategoryRow) => formatDateTime(row.updated_at)
    },
    { prop: 'operation', label: '操作', width: 150, fixed: 'right', useSlot: true }
  ])

  const canView = computed(() => hasAdminPermission(info.value?.buttons, ['cms_category.list']))
  const canCreate = computed(() => hasAdminPermission(info.value?.buttons, ['cms_category.create']))
  const canUpdate = computed(() => hasAdminPermission(info.value?.buttons, ['cms_category.update']))
  const canDelete = computed(() => hasAdminPermission(info.value?.buttons, ['cms_category.delete']))

  const rules = computed<FormRules>(() => ({
    key: [
      { required: true, message: '请输入分类标识', trigger: 'blur' },
      {
        max: INPUT_LIMITS.CMS_KEY,
        message: `分类标识长度不能超过 ${INPUT_LIMITS.CMS_KEY} 个字符`,
        trigger: 'blur'
      }
    ],
    name: [
      { required: true, message: '请输入显示名称', trigger: 'blur' },
      {
        max: INPUT_LIMITS.CMS_NAME,
        message: `显示名称长度不能超过 ${INPUT_LIMITS.CMS_NAME} 个字符`,
        trigger: 'blur'
      }
    ],
    lang: [{ required: true, message: '请选择语言', trigger: 'change' }]
  }))

  const filteredRows = computed(() => {
    const search = keyword.value.trim().toLowerCase()
    if (!search) {
      return tableData.value
    }

    return tableData.value.filter((row) =>
      [row.key, row.name].some((value) =>
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

  function createDefaultDialogForm(): CategoryDialogForm {
    return {
      id: null,
      key: '',
      name: '',
      lang: 'zh-CN',
      sort_order: 0,
      visible: true
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

  function normalizeRow(item?: CMSCategoryRecord): CategoryRow {
    return {
      id: normalizeNullableNumber(item?.id),
      key: String(item?.key || ''),
      name: String(item?.name || ''),
      lang: String(item?.lang || 'zh-CN'),
      sort_order: Number(item?.sort_order || 0),
      visible: Boolean(item?.visible),
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

  async function fetchData() {
    if (!canView.value) {
      return
    }

    loading.value = true

    try {
      const payload = await fetchCMSCategories({
        lang: langFilter.value || undefined
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

  function openEdit(row: CategoryRow) {
    dialogMode.value = 'edit'
    Object.assign(dialogForm, {
      id: row.id,
      key: row.key,
      name: row.name,
      lang: row.lang || 'zh-CN',
      sort_order: row.sort_order,
      visible: row.visible
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
        key: String(dialogForm.key || '').trim(),
        name: String(dialogForm.name || '').trim(),
        lang: String(dialogForm.lang || 'zh-CN').trim() || 'zh-CN',
        sort_order: Number(dialogForm.sort_order || 0),
        visible: Boolean(dialogForm.visible)
      }

      if (dialogMode.value === 'create') {
        await createCMSCategory(payload)
        ElMessage.success('分类创建成功')
      } else if (dialogForm.id) {
        await updateCMSCategory(dialogForm.id, payload)
        ElMessage.success('分类更新成功')
      }

      dialogVisible.value = false
      await fetchData()
    } finally {
      dialogSubmitting.value = false
    }
  }

  async function handleToggleVisible(row: CategoryRow, checked: string | number | boolean) {
    if (!row.id) {
      return
    }

    row.switching = true

    try {
      await updateCMSCategory(row.id, { visible: Boolean(checked) })
      row.visible = Boolean(checked)
      ElMessage.success('显示状态已更新')
    } catch (error: any) {
      ElMessage.error(error?.response?.data?.error || '更新显示状态失败')
    } finally {
      row.switching = false
    }
  }

  async function handleDelete(row: CategoryRow) {
    if (!row.id) {
      return
    }

    try {
      await ElMessageBox.confirm(
        `确定要删除分类“${row.name || row.key}”吗？该操作不可恢复。`,
        '删除分类',
        {
          type: 'warning'
        }
      )

      await deleteCMSCategory(row.id)
      ElMessage.success('分类删除成功')
      await fetchData()
    } catch (error: any) {
      if (error === 'cancel' || error === 'close') {
        return
      }

      ElMessage.error(error?.response?.data?.error || '删除分类失败')
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

  .toolbar-search {
    width: min(360px, 100%);
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

    .page-actions {
      width: 100%;
      flex-wrap: wrap;
    }

    .toolbar-select,
    .toolbar-search {
      width: 100%;
    }
  }
</style>
