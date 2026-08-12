<template>
  <div class="plugins-page art-full-height">
    <ArtSearchBar
      v-show="showSearchBar"
      v-model="searchForm"
      :items="searchItems"
      :show-expand="false"
      @search="handleSearch"
      @reset="resetSearchParams"
    />

    <ElCard class="art-table-card" :style="{ marginTop: showSearchBar ? '12px' : '0' }">
      <ArtTableHeader
        v-model:columns="columnChecks"
        v-model:show-search-bar="showSearchBar"
        :loading="loading"
        @refresh="refreshData"
      >
        <template #left>
          <ElSpace wrap>
            <ElButton v-if="canView" :loading="discoverLoading" @click="openDiscover">
              <ArtSvgIcon icon="ri:folder-search-line" />
              发现磁盘插件
            </ElButton>
            <ElUpload
              v-if="canUpload"
              :show-file-list="false"
              :http-request="installUpload"
              accept=".zip,.tar.gz,.tgz"
            >
              <ElButton type="primary" :loading="installing">
                <ArtSvgIcon icon="ri:upload-cloud-line" />
                安装插件
              </ElButton>
            </ElUpload>
          </ElSpace>
        </template>
      </ArtTableHeader>

      <ArtTable
        row-key="row_key"
        :loading="loading"
        :data="data"
        :columns="columns"
        :pagination="pagination"
        @pagination:size-change="handleSizeChange"
        @pagination:current-change="handleCurrentChange"
      >
        <template #plugin="{ row }">
          <div class="plugin-cell">
            <strong>{{ row.name || row.plugin_id }}</strong>
            <span class="secondary-text mono">
              {{ row.category }}/{{ row.plugin_id }}/{{ row.instance_id || 'default' }} · v{{
                row.version || '-'
              }}
            </span>
          </div>
        </template>
        <template #signature_status="{ row }">
          <ElTag :type="signatureType(row.signature_status)">
            {{ row.signature_status || '-' }}
          </ElTag>
        </template>
        <template #enabled="{ row }">
          <ElSwitch
            :model-value="row.enabled"
            :loading="busyKey === row.row_key"
            :disabled="!canUpdate"
            @change="(enabled: boolean) => toggleEnabled(row, enabled)"
          />
        </template>
        <template #health_status="{ row }">
          <div class="plugin-cell">
            <ElTag :type="healthType(row.health_status)">{{ row.health_status || '-' }}</ElTag>
            <span class="secondary-text">
              {{ row.last_health_at ? formatTime(row.last_health_at) : '暂无心跳' }}
              <template v-if="row.health_message"> · {{ row.health_message }}</template>
            </span>
          </div>
        </template>
        <template #capabilities="{ row }">
          <ElSpace wrap>
            <ElTag v-if="row.manifest?.capabilities?.payment" type="primary">
              payment {{ row.manifest.capabilities.payment.methods?.length || 0 }}
            </ElTag>
            <ElTag v-if="row.manifest?.capabilities?.sms" type="success">sms</ElTag>
            <ElTag v-if="row.manifest?.capabilities?.kyc" type="warning">kyc</ElTag>
            <ElTag v-if="row.manifest?.capabilities?.automation" type="info">
              automation:
              {{ row.manifest.capabilities.automation.features?.length || 0 }} features
            </ElTag>
            <ElButton link type="primary" @click="openManifest(row)">详情</ElButton>
          </ElSpace>
        </template>
        <template #operation="{ row }">
          <ElDropdown trigger="click" @command="(command: string) => handleCommand(command, row)">
            <ElButton circle plain aria-label="插件操作">
              <ArtSvgIcon icon="ri:more-2-fill" />
            </ElButton>
            <template #dropdown>
              <ElDropdownMenu>
                <ElDropdownItem v-if="canCreate" command="instance">新增实例</ElDropdownItem>
                <ElDropdownItem v-if="row.category === 'payment'" command="methods">
                  支付方法
                </ElDropdownItem>
                <ElDropdownItem v-if="canViewConfig" command="config">配置</ElDropdownItem>
                <ElDropdownItem v-if="canDelete" command="uninstall" divided>
                  卸载实例
                </ElDropdownItem>
              </ElDropdownMenu>
            </template>
          </ElDropdown>
        </template>
      </ArtTable>
    </ElCard>

    <ElDialog
      v-model="installPasswordVisible"
      title="安装确认（非官方签名）"
      width="520px"
      destroy-on-close
    >
      <ElAlert
        type="warning"
        :closable="false"
        show-icon
        title="该插件未通过官方签名校验，继续安装存在风险。"
      />
      <ArtForm
        v-model="installPasswordForm"
        :items="passwordItems"
        :show-reset="false"
        :show-submit="false"
        :span="24"
        label-position="top"
      />
      <template #footer>
        <ElButton @click="installPasswordVisible = false">取消</ElButton>
        <ElButton type="primary" :loading="installing" @click="confirmInstall">继续安装</ElButton>
      </template>
    </ElDialog>

    <ElDialog v-model="discoverVisible" title="发现磁盘插件" width="min(900px, 94vw)">
      <ElAlert
        class="dialog-alert"
        type="info"
        :closable="false"
        show-icon
        title="这些插件目录已存在于服务端 ./plugins 下，但尚未导入数据库。"
      />
      <ArtTable
        row-key="row_key"
        :show-table-header="false"
        :loading="discoverLoading"
        :data="discovered"
        :columns="discoverColumns"
      >
        <template #signature_status="{ row }">
          <ElTag :type="signatureType(row.signature_status)">{{
            row.signature_status || '-'
          }}</ElTag>
        </template>
        <template #platform="{ row }">
          <ElTag :type="row.entry?.entry_supported ? 'success' : 'danger'">
            {{ row.entry?.platform || '-' }}
          </ElTag>
          <div v-if="!row.entry?.entry_supported" class="secondary-text">
            支持：{{ row.entry?.supported_platforms?.join(', ') || '-' }}
          </div>
        </template>
        <template #operation="{ row }">
          <ElButton
            v-if="canUpload"
            link
            type="primary"
            :loading="importBusyKey === row.row_key"
            @click="startImport(row)"
          >
            导入
          </ElButton>
        </template>
      </ArtTable>
    </ElDialog>

    <ElDialog
      v-model="importPasswordVisible"
      title="导入确认（非官方签名）"
      width="520px"
      destroy-on-close
    >
      <ElAlert
        type="warning"
        :closable="false"
        show-icon
        title="该插件未通过官方签名校验，继续导入存在风险。"
      />
      <ArtForm
        v-model="importPasswordForm"
        :items="passwordItems"
        :show-reset="false"
        :show-submit="false"
        :span="24"
        label-position="top"
      />
      <template #footer>
        <ElButton @click="importPasswordVisible = false">取消</ElButton>
        <ElButton type="primary" :loading="importing" @click="confirmImport">继续导入</ElButton>
      </template>
    </ElDialog>

    <ElDialog v-model="instanceVisible" title="新增插件实例" width="520px" destroy-on-close>
      <ArtForm
        v-model="instanceForm"
        :items="instanceItems"
        :show-reset="false"
        :show-submit="false"
        :span="24"
        label-position="top"
      />
      <template #footer>
        <ElButton @click="instanceVisible = false">取消</ElButton>
        <ElButton type="primary" :loading="instanceCreating" @click="createInstance">
          创建
        </ElButton>
      </template>
    </ElDialog>

    <ElDialog
      v-model="configVisible"
      :title="`配置：${current?.name || current?.plugin_id || ''}`"
      width="min(760px, 94vw)"
    >
      <ElAlert
        v-if="schemaError"
        class="dialog-alert"
        type="error"
        :closable="false"
        show-icon
        :title="schemaError"
      />
      <div v-loading="schemaLoading" :class="{ 'schema-readonly': !canUpdate }">
        <JsonSchemaForm
          v-if="schema"
          v-model="configModel"
          :schema="schema"
          :ui-schema="uiSchema"
        />
        <ElCollapse>
          <ElCollapseItem title="原始 JSON" name="raw">
            <ElInput :model-value="prettyConfig" type="textarea" :rows="10" readonly />
          </ElCollapseItem>
        </ElCollapse>
      </div>
      <template #footer>
        <ElButton @click="configVisible = false">取消</ElButton>
        <ElButton v-if="canUpdate" type="primary" :loading="configSaving" @click="saveConfig">
          保存配置
        </ElButton>
      </template>
    </ElDialog>

    <ElDrawer v-model="manifestVisible" title="插件能力" size="520px">
      <ElDescriptions :column="1" border>
        <ElDescriptionsItem label="plugin_id">{{ current?.plugin_id || '-' }}</ElDescriptionsItem>
        <ElDescriptionsItem label="category">{{ current?.category || '-' }}</ElDescriptionsItem>
        <ElDescriptionsItem label="name">{{ current?.name || '-' }}</ElDescriptionsItem>
        <ElDescriptionsItem label="version">{{ current?.version || '-' }}</ElDescriptionsItem>
        <ElDescriptionsItem label="signature">
          {{ current?.signature_status || '-' }}
        </ElDescriptionsItem>
      </ElDescriptions>
      <ElInput
        class="manifest-json"
        :model-value="prettyManifest"
        type="textarea"
        :rows="16"
        readonly
      />
    </ElDrawer>

    <ElDialog v-model="methodsVisible" title="支付方法" width="600px">
      <ElAlert
        class="dialog-alert"
        type="info"
        :closable="false"
        show-icon
        title="方法由插件声明，启停状态由宿主管理。未显式配置时默认启用。"
      />
      <ArtTable
        row-key="method"
        :show-table-header="false"
        :loading="methodsLoading"
        :data="methodItems"
        :columns="methodColumns"
      >
        <template #enabled="{ row }">
          <ElSwitch
            :model-value="row.enabled"
            :loading="methodBusyKey === row.method"
            :disabled="!canUpdate"
            @change="(enabled: boolean) => toggleMethod(row.method, enabled)"
          />
        </template>
      </ArtTable>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import type { UploadRequestOptions } from 'element-plus'
  import JsonSchemaForm from '@/components/business/json-schema-form/index.vue'
  import {
    createAdminPluginInstance,
    deleteAdminPluginInstance,
    disableAdminPluginInstance,
    discoverAdminPlugins,
    enableAdminPluginInstance,
    getAdminPluginInstanceConfig,
    getAdminPluginInstanceConfigSchema,
    importAdminPluginFromDisk,
    installAdminPlugin,
    listAdminPluginPaymentMethods,
    listAdminPlugins,
    updateAdminPluginInstanceConfig,
    updateAdminPluginPaymentMethod
  } from '@/services/admin'
  import type {
    PluginDiscoverItem,
    PluginListItem,
    PluginPaymentMethodItem
  } from '@/services/types'
  import { useTable } from '@/hooks/core/useTable'
  import { useAdminPermissions } from '../_shared/settings'

  defineOptions({ name: 'AdminSettingsPlugins' })

  const router = useRouter()

  type PluginRow = PluginListItem & { row_key: string }
  type DiscoverRow = PluginDiscoverItem & { row_key: string }

  interface SearchForm {
    keyword?: string
    category?: string
  }

  interface TableParams extends SearchForm {
    current: number
    size: number
  }

  const showSearchBar = ref(true)
  const searchForm = ref<SearchForm>({ keyword: '', category: undefined })
  const busyKey = ref('')
  const installing = ref(false)
  const pendingInstallFile = ref<File>()
  const installPasswordVisible = ref(false)
  const installPasswordForm = reactive({ password: '' })
  const discoverVisible = ref(false)
  const discoverLoading = ref(false)
  const discovered = ref<DiscoverRow[]>([])
  const importBusyKey = ref('')
  const importing = ref(false)
  const importTarget = ref<DiscoverRow>()
  const importPasswordVisible = ref(false)
  const importPasswordForm = reactive({ password: '' })
  const instanceVisible = ref(false)
  const instanceCreating = ref(false)
  const instanceTarget = ref<PluginRow>()
  const instanceForm = reactive({ instance_id: '' })
  const configVisible = ref(false)
  const configSaving = ref(false)
  const schemaLoading = ref(false)
  const schemaError = ref('')
  const schema = ref<Record<string, unknown>>()
  const uiSchema = ref<Record<string, unknown>>({})
  const configModel = ref<Record<string, unknown>>({})
  const current = ref<PluginRow>()
  const manifestVisible = ref(false)
  const methodsVisible = ref(false)
  const methodsLoading = ref(false)
  const methodBusyKey = ref('')
  const methodItems = ref<PluginPaymentMethodItem[]>([])
  const { hasPermission } = useAdminPermissions()
  const canView = hasPermission('plugin.list')
  const canViewConfig = hasPermission('plugin.view')
  const canCreate = hasPermission('plugin.create')
  const canUpdate = hasPermission('plugin.update')
  const canDelete = hasPermission('plugin.delete')
  const canUpload = hasPermission('plugin.upload')

  const searchItems = computed(() => [
    {
      key: 'keyword',
      label: '关键词',
      type: 'input',
      props: { clearable: true, placeholder: '插件名或 plugin_id' }
    },
    {
      key: 'category',
      label: '类型',
      type: 'select',
      props: {
        clearable: true,
        options: ['payment', 'sms', 'kyc', 'automation'].map((value) => ({ label: value, value }))
      }
    }
  ])
  const passwordItems = computed(() => [
    {
      key: 'password',
      label: '管理员密码',
      type: 'input',
      props: { type: 'password', showPassword: true, autocomplete: 'current-password' }
    }
  ])
  const instanceItems = computed(() => [
    {
      key: 'instance_id',
      label: 'instance_id（可选）',
      type: 'input',
      props: { placeholder: '留空自动生成' }
    }
  ])
  const discoverColumns = [
    { prop: 'name', label: '插件', minWidth: 180 },
    { prop: 'category', label: '类型', width: 110 },
    { prop: 'signature_status', label: '签名', width: 120, useSlot: true },
    { prop: 'platform', label: '平台', width: 140, useSlot: true },
    { prop: 'operation', label: '操作', width: 100, useSlot: true }
  ]
  const methodColumns = [
    { prop: 'method', label: 'method', minWidth: 220 },
    { prop: 'enabled', label: 'enabled', width: 120, useSlot: true }
  ]
  const prettyConfig = computed(() => JSON.stringify(configModel.value, null, 2))
  const prettyManifest = computed(() => JSON.stringify(current.value?.manifest ?? {}, null, 2))

  const fetchPlugins = async (params: TableParams) => {
    const response = await listAdminPlugins()
    const keyword = String(params.keyword ?? '')
      .trim()
      .toLowerCase()
    let rows = (response.data?.items ?? []).map(normalizePlugin)
    if (keyword) {
      rows = rows.filter((row) =>
        `${row.name ?? ''} ${row.plugin_id ?? ''} ${row.instance_id ?? ''}`
          .toLowerCase()
          .includes(keyword)
      )
    }
    if (params.category) rows = rows.filter((row) => row.category === params.category)
    const start = (params.current - 1) * params.size
    return { records: rows.slice(start, start + params.size), total: rows.length }
  }

  const {
    columns,
    columnChecks,
    data,
    loading,
    pagination,
    getData,
    searchParams,
    resetSearchParams,
    handleSizeChange,
    handleCurrentChange,
    refreshData,
    refreshCreate,
    refreshUpdate,
    refreshRemove
  } = useTable({
    core: {
      apiFn: fetchPlugins,
      apiParams: { current: 1, size: 20, ...searchForm.value },
      columnsFactory: () => [
        { prop: 'plugin', label: '插件', minWidth: 280, useSlot: true },
        { prop: 'category', label: '类型', width: 110 },
        { prop: 'signature_status', label: '签名', width: 120, useSlot: true },
        { prop: 'enabled', label: '启用', width: 90, useSlot: true },
        { prop: 'health_status', label: '健康', minWidth: 220, useSlot: true },
        { prop: 'capabilities', label: '能力', minWidth: 260, useSlot: true },
        { prop: 'operation', label: '操作', width: 90, fixed: 'right', useSlot: true }
      ]
    }
  })

  function normalizePlugin(plugin: PluginListItem): PluginRow {
    return {
      ...plugin,
      row_key: `${plugin.category}/${plugin.plugin_id}/${plugin.instance_id || 'default'}`
    }
  }

  function handleSearch(params: SearchForm): void {
    Object.assign(searchParams, params)
    getData()
  }

  function signatureType(status?: string): 'success' | 'warning' | 'danger' | 'info' {
    if (status === 'official') return 'success'
    if (status === 'unsigned') return 'warning'
    if (status === 'untrusted') return 'danger'
    return 'info'
  }

  function healthType(status?: string): 'success' | 'warning' | 'danger' | 'info' {
    const value = String(status ?? '').toLowerCase()
    if (value === 'ok') return 'success'
    if (value === 'degraded') return 'warning'
    if (value === 'error') return 'danger'
    return 'info'
  }

  function formatTime(value: string): string {
    const date = new Date(value)
    return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN')
  }

  async function handleCommand(command: string, row: PluginRow): Promise<void> {
    if (command === 'instance') openInstance(row)
    if (command === 'methods') await openMethods(row)
    if (command === 'config') await openConfig(row)
    if (command === 'uninstall') await uninstall(row)
  }

  async function toggleEnabled(row: PluginRow, enabled: boolean): Promise<void> {
    busyKey.value = row.row_key
    try {
      const category = String(row.category ?? '')
      const pluginId = String(row.plugin_id ?? '')
      const instanceId = String(row.instance_id ?? 'default')
      if (enabled) await enableAdminPluginInstance(category, pluginId, instanceId)
      else await disableAdminPluginInstance(category, pluginId, instanceId)
      ElMessage.success('插件状态已更新')
      await refreshUpdate()
    } catch (error) {
      const response = (error as { response?: { data?: Record<string, unknown> } }).response?.data
      if (enabled && response?.code === 'missing_required_config') {
        const missing = Array.isArray(response.missing_fields)
          ? response.missing_fields.map(String).join('、')
          : ''
        ElMessage.error(
          `${String(response.error ?? '插件配置不完整')}${missing ? `：${missing}` : ''}`
        )
        if (row.category === 'automation') navigateCatalog()
      }
    } finally {
      busyKey.value = ''
    }
  }

  async function installUpload(options: UploadRequestOptions): Promise<void> {
    await tryInstall(options.file)
  }

  async function tryInstall(file: File, password?: string): Promise<void> {
    installing.value = true
    try {
      await installAdminPlugin(file, password)
      pendingInstallFile.value = undefined
      installPasswordVisible.value = false
      installPasswordForm.password = ''
      ElMessage.success('插件安装成功')
      await refreshCreate()
    } catch (error) {
      const response = (error as { response?: { status?: number; data?: { error?: string } } })
        .response
      if (response?.status === 403 && String(response.data?.error).includes('admin_password')) {
        pendingInstallFile.value = file
        installPasswordVisible.value = true
      }
    } finally {
      installing.value = false
    }
  }

  async function confirmInstall(): Promise<void> {
    if (!pendingInstallFile.value || !installPasswordForm.password.trim()) {
      ElMessage.error('请输入管理员密码')
      return
    }
    await tryInstall(pendingInstallFile.value, installPasswordForm.password.trim())
  }

  async function openDiscover(): Promise<void> {
    discoverVisible.value = true
    discoverLoading.value = true
    try {
      const response = await discoverAdminPlugins()
      discovered.value = (response.data?.items ?? []).map((item) => ({
        ...item,
        row_key: `${item.category}/${item.plugin_id}`
      }))
    } finally {
      discoverLoading.value = false
    }
  }

  async function startImport(row: DiscoverRow): Promise<void> {
    if (!canUpload.value) return
    if (row.signature_status !== 'official') {
      importTarget.value = row
      importPasswordForm.password = ''
      importPasswordVisible.value = true
      return
    }
    await importPlugin(row)
  }

  async function confirmImport(): Promise<void> {
    if (!canUpload.value) return
    if (!importTarget.value || !importPasswordForm.password.trim()) {
      ElMessage.error('请输入管理员密码')
      return
    }
    await importPlugin(importTarget.value, importPasswordForm.password.trim())
  }

  async function importPlugin(row: DiscoverRow, password?: string): Promise<void> {
    if (!canUpload.value) return
    importBusyKey.value = row.row_key
    importing.value = true
    try {
      await importAdminPluginFromDisk(
        String(row.category ?? ''),
        String(row.plugin_id ?? ''),
        password
      )
      importPasswordVisible.value = false
      ElMessage.success('插件导入成功')
      await Promise.all([openDiscover(), refreshCreate()])
    } finally {
      importBusyKey.value = ''
      importing.value = false
    }
  }

  function openInstance(row: PluginRow): void {
    instanceTarget.value = row
    instanceForm.instance_id = ''
    instanceVisible.value = true
  }

  async function createInstance(): Promise<void> {
    if (!instanceTarget.value) return
    instanceCreating.value = true
    try {
      await createAdminPluginInstance(
        String(instanceTarget.value.category ?? ''),
        String(instanceTarget.value.plugin_id ?? ''),
        { instance_id: instanceForm.instance_id.trim() }
      )
      instanceVisible.value = false
      ElMessage.success('插件实例已创建')
      await refreshCreate()
    } finally {
      instanceCreating.value = false
    }
  }

  function parseJson(value?: string): Record<string, unknown> | undefined {
    try {
      const parsed = JSON.parse(String(value || '{}'))
      return parsed && typeof parsed === 'object' && !Array.isArray(parsed) ? parsed : undefined
    } catch {
      return undefined
    }
  }

  async function openConfig(row: PluginRow): Promise<void> {
    if (row.category === 'automation') {
      ElMessage.info('automation 插件配置在商品类型页面维护')
      navigateCatalog()
      return
    }
    current.value = row
    configVisible.value = true
    schemaLoading.value = true
    schemaError.value = ''
    schema.value = undefined
    configModel.value = {}
    try {
      const [schemaResponse, configResponse] = await Promise.all([
        getAdminPluginInstanceConfigSchema(
          String(row.category ?? ''),
          String(row.plugin_id ?? ''),
          String(row.instance_id ?? 'default')
        ),
        getAdminPluginInstanceConfig(
          String(row.category ?? ''),
          String(row.plugin_id ?? ''),
          String(row.instance_id ?? 'default')
        )
      ])
      const parsedSchema = parseJson(schemaResponse.data?.json_schema)
      if (parsedSchema?.type !== 'object')
        schemaError.value = '插件 Schema 无法解析或不是 object 类型'
      else schema.value = parsedSchema
      uiSchema.value = parseJson(schemaResponse.data?.ui_schema) ?? {}
      configModel.value = parseJson(configResponse.data?.config_json) ?? {}
    } finally {
      schemaLoading.value = false
    }
  }

  async function saveConfig(): Promise<void> {
    if (!canUpdate.value || !current.value) return
    configSaving.value = true
    try {
      await updateAdminPluginInstanceConfig(
        String(current.value.category ?? ''),
        String(current.value.plugin_id ?? ''),
        String(current.value.instance_id ?? 'default'),
        JSON.stringify(configModel.value)
      )
      configVisible.value = false
      ElMessage.success('插件配置已保存')
      await refreshUpdate()
    } finally {
      configSaving.value = false
    }
  }

  function openManifest(row: PluginRow): void {
    current.value = row
    manifestVisible.value = true
  }

  async function openMethods(row: PluginRow): Promise<void> {
    current.value = row
    methodsVisible.value = true
    methodsLoading.value = true
    try {
      const response = await listAdminPluginPaymentMethods({
        category: String(row.category ?? 'payment'),
        plugin_id: String(row.plugin_id ?? ''),
        instance_id: String(row.instance_id ?? 'default')
      })
      methodItems.value = response.data?.items ?? []
    } finally {
      methodsLoading.value = false
    }
  }

  async function toggleMethod(method: string, enabled: boolean): Promise<void> {
    if (!current.value) return
    methodBusyKey.value = method
    try {
      await updateAdminPluginPaymentMethod({
        category: String(current.value.category ?? 'payment'),
        plugin_id: String(current.value.plugin_id ?? ''),
        instance_id: String(current.value.instance_id ?? 'default'),
        method,
        enabled
      })
      const item = methodItems.value.find((row) => row.method === method)
      if (item) item.enabled = enabled
      ElMessage.success('支付方法状态已更新')
    } finally {
      methodBusyKey.value = ''
    }
  }

  async function uninstall(row: PluginRow): Promise<void> {
    await ElMessageBox.confirm('确认卸载此插件实例吗？', '卸载插件', { type: 'warning' })
    await deleteAdminPluginInstance(
      String(row.category ?? ''),
      String(row.plugin_id ?? ''),
      String(row.instance_id ?? 'default')
    )
    ElMessage.success('插件实例已卸载')
    await refreshRemove()
  }

  function navigateCatalog(): void {
    void router.push({ name: 'CatalogPage' })
  }
</script>

<style lang="scss" scoped>
  .plugins-page {
    min-width: 0;
  }

  .plugin-cell {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .secondary-text {
    font-size: 12px;
    line-height: 1.5;
    color: var(--art-gray-600);
  }

  .mono {
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  }

  .dialog-alert {
    margin-bottom: 12px;
  }

  .manifest-json {
    margin-top: 16px;
  }

  .schema-readonly {
    pointer-events: none;
  }
</style>
