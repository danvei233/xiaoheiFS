<template>
  <div class="art-full-height">
    <SystemImageSearch
      v-show="showSearchBar"
      v-model="searchForm"
      @search="handleSearch"
      @reset="handleReset"
    />

    <ElCard class="art-table-card" :style="{ marginTop: showSearchBar ? '12px' : '0' }">
      <ArtTableHeader
        v-model:columns="columnChecks"
        v-model:showSearchBar="showSearchBar"
        :loading="loading"
        @refresh="refreshPage"
      >
        <template #left>
          <ElSpace wrap>
            <ElButton
              v-if="canBulkDelete"
              type="danger"
              :disabled="!selectedImageIds.length"
              @click="removeSelectedImages"
            >
              {{ t('systemImage.page.actions.bulkDelete') }}
            </ElButton>

            <ElButton v-if="canSync" @click="openSyncDialog">
              {{ t('systemImage.page.actions.sync') }}
            </ElButton>

            <ElButton v-if="canConfigLineImages" @click="openLineConfigDialog">
              {{ t('systemImage.page.actions.lineConfig') }}
            </ElButton>

            <ElButton v-if="canCreate" type="primary" @click="openCreateDialog">
              {{ t('systemImage.page.actions.create') }}
            </ElButton>
          </ElSpace>
        </template>
      </ArtTableHeader>

      <div class="page-tip">{{ t('systemImage.page.tip') }}</div>

      <ArtTable
        row-key="id"
        :loading="loading"
        :data="tableData"
        :columns="columns"
        :pagination="pagination"
        @selection-change="onSelectionChange"
        @pagination:size-change="handlePageSizeChange"
        @pagination:current-change="handlePageCurrentChange"
      >
        <ElTableColumn v-if="canBulkDelete" type="selection" width="48" />

        <template #type="{ row }">
          <ElTag :type="getTypeTagType(row.type)">
            {{ formatImageType(row.type) }}
          </ElTag>
        </template>

        <template #enabled="{ row }">
          <ElTag :type="row.enabled ? 'success' : 'danger'">
            {{ row.enabled ? t('systemImage.status.enabled') : t('systemImage.status.disabled') }}
          </ElTag>
        </template>

        <template #operation="{ row }">
          <div class="table-actions">
            <ElButton v-if="canUpdate" link type="primary" @click="openEditDialog(row)">{{
              t('systemImage.page.operation.edit')
            }}</ElButton>
            <ElButton v-if="canDelete" link type="danger" @click="removeImage(row)">{{
              t('systemImage.page.operation.delete')
            }}</ElButton>
            <span v-if="!canUpdate && !canDelete" class="muted">-</span>
          </div>
        </template>
      </ArtTable>
    </ElCard>

    <SystemImageDialog
      v-model:visible="dialogVisible"
      :form-data="dialogForm"
      :submitting="dialogSubmitting"
      @submit="handleDialogSubmit"
    />

    <LineImageDialog
      v-model:visible="lineDialogVisible"
      :mode="lineDialogMode"
      :line-id="lineDialogLineId"
      :image-ids="lineDialogImageIds"
      :lines="lines"
      :image-options="imageOptions"
      :line-image-count-map="lineImageCountMap"
      :submitting="lineDialogSubmitting"
      @update:line-id="lineDialogLineId = $event"
      @update:image-ids="lineDialogImageIds = $event"
      @submit="handleLineDialogSubmit"
    />
  </div>
</template>

<script setup lang="ts">
  import type { CatalogPlanGroup, CatalogSystemImage } from '@/api/admin'
  import { createDefaultSystemImageDialogForm } from '@/components/business/system-image-dialog/model'
  import type { SystemImageDialogFormValue } from '@/components/business/system-image-dialog/model'
  import {
    bulkDeleteAdminSystemImages,
    createAdminSystemImage,
    deleteAdminSystemImage,
    fetchAdminLines,
    fetchAdminSystemImages,
    hasAdminPermission,
    setAdminLineSystemImages,
    syncAdminSystemImages,
    updateAdminSystemImage
  } from '@/api/admin'
  import { useTableColumns } from '@/hooks/core/useTableColumns'
  import { useUserStore } from '@/store/modules/user'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import { useI18n } from 'vue-i18n'
  import LineImageDialog from './modules/line-image-dialog.vue'
  import SystemImageDialog from './modules/system-image-dialog.vue'
  import SystemImageSearch from './modules/system-image-search.vue'

  defineOptions({ name: 'SystemsPage' })

  interface SystemImageSearchForm {
    keyword: string
    status?: string
  }

  interface SystemImageRow {
    id: number | null
    image_id: number | null
    name: string
    type: string
    enabled: boolean
  }

  interface LineRow {
    id: number | null
    name: string
    line_id: number | null
  }

  const userStore = useUserStore()
  const { info } = storeToRefs(userStore)
  const { t } = useI18n()

  const showSearchBar = ref(true)
  const loading = ref(false)
  const dialogVisible = ref(false)
  const dialogSubmitting = ref(false)
  const lineDialogVisible = ref(false)
  const lineDialogSubmitting = ref(false)
  const lineDialogMode = ref<'config' | 'sync'>('config')

  const searchForm = ref<SystemImageSearchForm>(createDefaultSearchForm())
  const allRows = ref<SystemImageRow[]>([])
  const selectedImageIds = ref<number[]>([])
  const lines = ref<LineRow[]>([])
  const lineDialogLineId = ref<number | null>(null)
  const lineDialogImageIds = ref<number[]>([])
  const lineImageCountMap = ref<Record<number, number>>({})
  const dialogForm = ref<SystemImageDialogFormValue>(createDefaultSystemImageDialogForm())

  const pagination = reactive({
    current: 1,
    size: 20,
    total: 0
  })

  const { columnChecks, columns } = useTableColumns<SystemImageRow>(() => [
    { prop: 'id', label: 'ID', width: 80 },
    { prop: 'image_id', label: t('systemImage.page.columns.imageId'), width: 110 },
    {
      prop: 'name',
      label: t('systemImage.page.columns.name'),
      minWidth: 220,
      showOverflowTooltip: true
    },
    { prop: 'type', label: t('systemImage.page.columns.type'), width: 120, useSlot: true },
    { prop: 'enabled', label: t('systemImage.page.columns.status'), width: 110, useSlot: true },
    {
      prop: 'operation',
      label: t('systemImage.page.columns.operation'),
      width: 150,
      fixed: 'right',
      useSlot: true
    }
  ])

  const canView = computed(() => hasAdminPermission(info.value?.buttons, ['system_image.list']))
  const canCreate = computed(() => hasAdminPermission(info.value?.buttons, ['system_image.create']))
  const canUpdate = computed(() => hasAdminPermission(info.value?.buttons, ['system_image.update']))
  const canDelete = computed(() => hasAdminPermission(info.value?.buttons, ['system_image.delete']))
  const canBulkDelete = computed(
    () =>
      hasAdminPermission(info.value?.buttons, ['system_image.delete']) ||
      hasAdminPermission(info.value?.buttons, ['system_image.bulk_delete'])
  )
  const canViewLines = computed(() => hasAdminPermission(info.value?.buttons, ['line.list']))
  const canConfigLineImages = computed(
    () => canViewLines.value && hasAdminPermission(info.value?.buttons, ['line.set_system_images'])
  )
  const canSync = computed(
    () => canViewLines.value && hasAdminPermission(info.value?.buttons, ['system_image.sync'])
  )

  const filteredRows = computed(() => {
    const keyword = String(searchForm.value.keyword || '')
      .trim()
      .toLowerCase()
    const status = searchForm.value.status

    return allRows.value.filter((row) => {
      const matchesKeyword =
        !keyword ||
        String(row.id || '')
          .toLowerCase()
          .includes(keyword) ||
        String(row.image_id || '')
          .toLowerCase()
          .includes(keyword) ||
        String(row.name || '')
          .toLowerCase()
          .includes(keyword)

      const matchesStatus =
        !status || (status === 'enabled' && row.enabled) || (status === 'disabled' && !row.enabled)

      return matchesKeyword && matchesStatus
    })
  })

  const tableData = computed(() => {
    const start = (pagination.current - 1) * pagination.size
    return filteredRows.value.slice(start, start + pagination.size)
  })

  const imageOptions = computed(() =>
    allRows.value.map((item) => ({
      label: `${item.name || '-'} (${item.type || '-'})`,
      value: Number(item.id || 0)
    }))
  )

  watch(
    filteredRows,
    (rows) => {
      pagination.total = rows.length
      const maxPage = Math.max(1, Math.ceil(rows.length / pagination.size))
      if (pagination.current > maxPage) {
        pagination.current = 1
      }
    },
    { immediate: true }
  )

  watch(lineDialogLineId, async (value) => {
    if (!lineDialogVisible.value || lineDialogMode.value !== 'config') {
      return
    }

    if (!value) {
      lineDialogImageIds.value = []
      return
    }

    await loadLineImages(value)
  })

  onMounted(() => {
    refreshPage()
  })

  function createDefaultSearchForm(): SystemImageSearchForm {
    return {
      keyword: '',
      status: undefined
    }
  }

  function normalizeNullableNumber(value: unknown): number | null {
    if (value === '' || value === null || value === undefined) {
      return null
    }

    const parsed = Number(value)
    return Number.isFinite(parsed) ? parsed : null
  }

  function normalizeSystemImage(
    row?: CatalogSystemImage | Record<string, unknown>
  ): SystemImageRow {
    const source = (row || {}) as Record<string, unknown>

    return {
      id: normalizeNullableNumber(source.id ?? source.ID),
      image_id: normalizeNullableNumber(source.image_id ?? source.ImageID),
      name: String(source.name ?? source.Name ?? ''),
      type: String(source.type ?? source.Type ?? ''),
      enabled: Boolean(source.enabled ?? source.Enabled)
    }
  }

  function normalizeLine(row?: CatalogPlanGroup | Record<string, unknown>): LineRow {
    const source = (row || {}) as Record<string, unknown>

    return {
      id: normalizeNullableNumber(source.id ?? source.ID),
      name: String(source.name ?? source.Name ?? source.line_name ?? source.LineName ?? ''),
      line_id: normalizeNullableNumber(source.line_id ?? source.LineID)
    }
  }

  async function refreshPage() {
    await Promise.all([fetchData(), loadLines()])
  }

  async function fetchData() {
    if (!canView.value) {
      allRows.value = []
      return
    }

    loading.value = true

    try {
      const payload = await fetchAdminSystemImages()
      allRows.value = (payload.items || []).map((item) => normalizeSystemImage(item))
      selectedImageIds.value = []
    } finally {
      loading.value = false
    }
  }

  async function loadLines() {
    if (!canViewLines.value) {
      lines.value = []
      lineImageCountMap.value = {}
      return
    }

    const payload = await fetchAdminLines()
    lines.value = (payload.items || []).map((item) => normalizeLine(item))
    await loadLineImageCounts()
  }

  async function loadLineImages(localLineId: number) {
    const cloudLineId = getCloudLineId(localLineId)
    if (!cloudLineId) {
      lineDialogImageIds.value = []
      return
    }

    const payload = await fetchAdminSystemImages({
      line_id: cloudLineId
    })

    lineDialogImageIds.value = (payload.items || [])
      .map((item) => normalizeSystemImage(item).id)
      .filter((item): item is number => Number.isFinite(Number(item)) && Number(item) > 0)
  }

  async function loadLineImageCounts() {
    const map: Record<number, number> = {}

    await Promise.all(
      lines.value.map(async (line) => {
        if (!line.id || !line.line_id) {
          map[Number(line.id || 0)] = 0
          return
        }

        try {
          const payload = await fetchAdminSystemImages({
            line_id: line.line_id
          })
          map[line.id] = Array.isArray(payload.items) ? payload.items.length : 0
        } catch {
          map[line.id] = 0
        }
      })
    )

    lineImageCountMap.value = map
  }

  function getCloudLineId(localLineId: number) {
    const matched = lines.value.find((line) => Number(line.id) === Number(localLineId))
    return matched?.line_id ?? null
  }

  function getTypeTagType(type?: string) {
    const normalized = String(type || '').toLowerCase()
    if (normalized.includes('win')) {
      return 'primary' as const
    }
    if (normalized.includes('linux')) {
      return 'success' as const
    }
    return 'info' as const
  }

  function formatImageType(type?: string) {
    return type ? String(type) : '-'
  }

  function handleSearch(params: SystemImageSearchForm) {
    searchForm.value = { ...searchForm.value, ...params }
    pagination.current = 1
  }

  function handleReset() {
    searchForm.value = createDefaultSearchForm()
    pagination.current = 1
  }

  function handlePageSizeChange(size: number) {
    pagination.size = size
    pagination.current = 1
  }

  function handlePageCurrentChange(page: number) {
    pagination.current = page
  }

  function onSelectionChange(rows: SystemImageRow[]) {
    selectedImageIds.value = rows
      .map((row) => Number(row.id || 0))
      .filter((item) => Number.isFinite(item) && item > 0)
  }

  function openCreateDialog() {
    dialogForm.value = createDefaultSystemImageDialogForm()
    dialogVisible.value = true
  }

  function openEditDialog(row: SystemImageRow) {
    dialogForm.value = {
      id: row.id,
      image_id: row.image_id,
      name: row.name,
      type: String(row.type || 'linux').toLowerCase() || 'linux',
      enabled: row.enabled
    }
    dialogVisible.value = true
  }

  async function handleDialogSubmit(form: SystemImageDialogFormValue) {
    const imageId = Number(form.image_id || 0)
    if (!Number.isInteger(imageId) || imageId <= 0) {
      ElMessage.error(t('systemImage.messages.imageIdPositive'))
      return
    }

    if (!String(form.name || '').trim()) {
      ElMessage.error(t('systemImage.messages.nameRequired'))
      return
    }

    const type = String(form.type || '')
      .trim()
      .toLowerCase()
    if (!['linux', 'windows'].includes(type)) {
      ElMessage.error(t('systemImage.messages.typeRequired'))
      return
    }

    dialogSubmitting.value = true

    try {
      const payload = {
        image_id: imageId,
        name: String(form.name || '').trim(),
        type,
        enabled: Boolean(form.enabled)
      }

      if (form.id) {
        await updateAdminSystemImage(form.id, payload)
      } else {
        await createAdminSystemImage(payload)
      }

      ElMessage.success(t('systemImage.messages.saved'))
      dialogVisible.value = false
      await fetchData()
    } finally {
      dialogSubmitting.value = false
    }
  }

  async function removeImage(row: SystemImageRow) {
    if (!row.id) {
      return
    }

    try {
      await ElMessageBox.confirm(
        t('systemImage.messages.confirmDelete'),
        t('systemImage.messages.confirmDeleteTitle'),
        {
          type: 'warning'
        }
      )
    } catch {
      return
    }

    await deleteAdminSystemImage(row.id)
    ElMessage.success(t('systemImage.messages.deleted'))
    await fetchData()
  }

  async function removeSelectedImages() {
    if (!selectedImageIds.value.length) {
      return
    }

    try {
      await ElMessageBox.confirm(
        t('systemImage.messages.confirmBulkDelete', { count: selectedImageIds.value.length }),
        t('systemImage.messages.confirmBulkDeleteTitle'),
        { type: 'warning' }
      )
    } catch {
      return
    }

    await bulkDeleteAdminSystemImages(selectedImageIds.value)
    selectedImageIds.value = []
    ElMessage.success(t('systemImage.messages.bulkDeleted'))
    await fetchData()
  }

  function openSyncDialog() {
    lineDialogMode.value = 'sync'
    lineDialogLineId.value = null
    lineDialogImageIds.value = []
    lineDialogVisible.value = true
  }

  async function openLineConfigDialog() {
    lineDialogMode.value = 'config'
    lineDialogLineId.value = lineDialogLineId.value || null
    lineDialogVisible.value = true

    if (lineDialogLineId.value) {
      await loadLineImages(lineDialogLineId.value)
    } else {
      lineDialogImageIds.value = []
    }
  }

  async function handleLineDialogSubmit() {
    if (!lineDialogLineId.value) {
      ElMessage.error(t('systemImage.messages.lineRequired'))
      return
    }

    lineDialogSubmitting.value = true

    try {
      if (lineDialogMode.value === 'sync') {
        const cloudLineId = getCloudLineId(lineDialogLineId.value)
        if (!cloudLineId) {
          ElMessage.error(t('systemImage.messages.resolveLineFailed'))
          return
        }

        await syncAdminSystemImages({
          line_id: cloudLineId
        })

        ElMessage.success(t('systemImage.messages.syncStarted'))
      } else {
        await setAdminLineSystemImages(lineDialogLineId.value, {
          image_ids: lineDialogImageIds.value
        })

        ElMessage.success(t('systemImage.messages.lineConfigSaved'))
      }

      lineDialogVisible.value = false
      await Promise.all([fetchData(), loadLineImageCounts()])
    } finally {
      lineDialogSubmitting.value = false
    }
  }
</script>

<style scoped lang="scss">
  .page-tip {
    margin-bottom: 12px;
    color: var(--el-text-color-secondary);
    font-size: 13px;
    line-height: 1.7;
  }

  .table-actions {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: 2px;
  }

  .muted {
    color: var(--el-text-color-secondary);
  }
</style>
