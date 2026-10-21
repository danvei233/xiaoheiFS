<template>
  <div v-loading="loading" class="plugins-settings-page art-full-height">
    <div class="page-header">
      <div>
        <div class="page-kicker">Extensions</div>
        <div class="page-title">Plugin Management</div>
        <div class="page-subtitle">
          Manage plugin discovery, installation, instances, configuration, and payment methods.
        </div>
      </div>

      <div class="page-actions">
        <ElButton v-if="canView" :disabled="loading" @click="fetchData">Refresh</ElButton>
        <ElButton v-if="canCreate" :loading="discoverLoading" @click="openDiscoverDialog">
          Discover Plugins
        </ElButton>
        <ElUpload
          v-if="canInstall"
          :auto-upload="false"
          :show-file-list="false"
          accept=".zip,.tar.gz,.tgz"
          @change="handleInstallFileChange"
        >
          <ElButton type="primary" :loading="installing">Install Plugin</ElButton>
        </ElUpload>
      </div>
    </div>

    <ElEmpty v-if="!canView" description="You do not have permission to view plugins." />

    <template v-else>
      <ElCard shadow="never" class="filter-card">
        <div class="filter-row">
          <ElRadioGroup v-model="category">
            <ElRadioButton v-for="item in categoryOptions" :key="item.value" :label="item.value">
              {{ item.label }}
            </ElRadioButton>
          </ElRadioGroup>

          <ElInput
            v-model.trim="keyword"
            clearable
            placeholder="Search by plugin name, id, or instance"
            class="keyword-input"
          />
        </div>
      </ElCard>

      <ElCard shadow="never" class="table-card">
        <ElTable :data="filteredRows" border row-key="row_key">
          <ElTableColumn label="Plugin" min-width="260">
            <template #default="{ row }">
              <div class="plugin-cell">
                <div class="plugin-name">{{ row.name || row.plugin_id || '-' }}</div>
                <div class="plugin-meta">
                  <span class="mono">
                    {{ row.category || '-' }}/{{ row.plugin_id || '-' }}/{{
                      row.instance_id || 'default'
                    }}
                  </span>
                  <ElTag size="small" effect="plain">v{{ row.version || '-' }}</ElTag>
                </div>
              </div>
            </template>
          </ElTableColumn>

          <ElTableColumn prop="category" label="Category" width="120" />

          <ElTableColumn label="Signature" width="120">
            <template #default="{ row }">
              <ElTag :type="getSignatureTagType(row.signature_status)">
                {{ row.signature_status || '-' }}
              </ElTag>
            </template>
          </ElTableColumn>

          <ElTableColumn label="Enabled" width="110">
            <template #default="{ row }">
              <ElSwitch
                :model-value="row.enabled"
                :disabled="!canUpdate"
                :loading="busyRowKey === row.row_key"
                @change="handleToggleEnabled(row, $event)"
              />
            </template>
          </ElTableColumn>

          <ElTableColumn label="Health" min-width="220">
            <template #default="{ row }">
              <div class="health-cell">
                <ElTag :type="getHealthTagType(row.health_status)">
                  {{ row.health_status || '-' }}
                </ElTag>
                <div class="health-subtle">
                  <span v-if="row.last_health_at">{{ formatDateTime(row.last_health_at) }}</span>
                  <span v-else>No heartbeat</span>
                  <span v-if="row.health_message"> · {{ row.health_message }}</span>
                </div>
              </div>
            </template>
          </ElTableColumn>

          <ElTableColumn label="Capabilities" min-width="260">
            <template #default="{ row }">
              <div class="capability-list">
                <ElTag v-if="row.manifest?.capabilities?.payment" type="primary" effect="plain">
                  payment
                </ElTag>
                <ElTag v-if="row.manifest?.capabilities?.sms" type="success" effect="plain">
                  sms
                </ElTag>
                <ElTag v-if="row.manifest?.capabilities?.kyc" type="warning" effect="plain">
                  kyc
                </ElTag>
                <ElTag v-if="row.manifest?.capabilities?.automation" effect="plain">
                  automation
                </ElTag>
                <ElButton v-if="canPluginView" link type="primary" @click="openManifest(row)">
                  Manifest
                </ElButton>
              </div>
            </template>
          </ElTableColumn>

          <ElTableColumn label="Actions" width="280" fixed="right">
            <template #default="{ row }">
              <div class="table-actions">
                <ElButton v-if="canCreate" link type="primary" @click="openInstanceDialog(row)">
                  Add Instance
                </ElButton>
                <ElButton
                  v-if="canPluginView && row.category === 'payment'"
                  link
                  type="primary"
                  @click="openMethodsDialog(row)"
                >
                  Methods
                </ElButton>
                <ElButton v-if="canPluginView" link type="primary" @click="openConfigDialog(row)">
                  Config
                </ElButton>
                <ElPopconfirm
                  v-if="canDelete"
                  title="Delete this plugin instance?"
                  @confirm="deleteInstance(row)"
                >
                  <template #reference>
                    <ElButton link type="danger">Delete</ElButton>
                  </template>
                </ElPopconfirm>
              </div>
            </template>
          </ElTableColumn>
        </ElTable>
      </ElCard>
    </template>

    <ElDialog
      v-model="installPasswordVisible"
      title="Install Confirmation"
      width="420px"
      destroy-on-close
      align-center
    >
      <ElAlert
        type="warning"
        :closable="false"
        show-icon
        title="Unsigned Or Untrusted Plugin"
        description="This plugin is not officially signed. Enter your admin password to continue."
      />

      <ElForm label-position="top" class="dialog-form">
        <ElFormItem label="Admin Password">
          <ElInput
            v-model="installAdminPassword"
            type="password"
            show-password
            placeholder="Enter admin password"
          />
        </ElFormItem>
      </ElForm>

      <template #footer>
        <div class="dialog-footer">
          <ElButton @click="installPasswordVisible = false">Cancel</ElButton>
          <ElButton type="primary" :loading="installing" @click="confirmInstall">Confirm</ElButton>
        </div>
      </template>
    </ElDialog>

    <ElDialog
      v-model="discoverVisible"
      title="Discover Plugins On Disk"
      width="980px"
      destroy-on-close
      align-center
    >
      <ElAlert
        type="info"
        :closable="false"
        show-icon
        title="Disk Discovery"
        description="These plugin directories exist on disk but have not been imported into the database yet."
      />

      <ElTable :data="discoveredRows" border row-key="row_key" class="discover-table">
        <ElTableColumn label="Plugin" min-width="240">
          <template #default="{ row }">
            <div class="plugin-cell">
              <div class="plugin-name">{{ row.name || row.plugin_id || '-' }}</div>
              <div class="plugin-meta">
                <span class="mono">{{ row.category || '-' }}/{{ row.plugin_id || '-' }}</span>
                <ElTag size="small" effect="plain">v{{ row.version || '-' }}</ElTag>
              </div>
            </div>
          </template>
        </ElTableColumn>

        <ElTableColumn label="Signature" width="120">
          <template #default="{ row }">
            <ElTag :type="getSignatureTagType(row.signature_status)">
              {{ row.signature_status || '-' }}
            </ElTag>
          </template>
        </ElTableColumn>

        <ElTableColumn label="Platform" min-width="220">
          <template #default="{ row }">
            <div class="health-cell">
              <ElTag :type="row.entry?.entry_supported ? 'success' : 'danger'">
                {{ row.entry?.platform || '-' }}
              </ElTag>
              <div v-if="!row.entry?.entry_supported" class="health-subtle">
                Supported: {{ (row.entry?.supported_platforms || []).join(', ') || '-' }}
              </div>
            </div>
          </template>
        </ElTableColumn>

        <ElTableColumn label="Actions" width="120" fixed="right">
          <template #default="{ row }">
            <ElButton
              v-if="canCreate"
              link
              type="primary"
              :loading="importBusyKey === row.row_key"
              @click="startImport(row)"
            >
              Import
            </ElButton>
          </template>
        </ElTableColumn>
      </ElTable>
    </ElDialog>

    <ElDialog
      v-model="importPasswordVisible"
      title="Import Confirmation"
      width="420px"
      destroy-on-close
      align-center
    >
      <ElAlert
        type="warning"
        :closable="false"
        show-icon
        title="Unsigned Or Untrusted Plugin"
        description="This plugin is not officially signed. Enter your admin password to import it."
      />

      <ElForm label-position="top" class="dialog-form">
        <ElFormItem label="Admin Password">
          <ElInput
            v-model="importAdminPassword"
            type="password"
            show-password
            placeholder="Enter admin password"
          />
        </ElFormItem>
      </ElForm>

      <template #footer>
        <div class="dialog-footer">
          <ElButton @click="importPasswordVisible = false">Cancel</ElButton>
          <ElButton type="primary" :loading="importing" @click="confirmImport">Confirm</ElButton>
        </div>
      </template>
    </ElDialog>

    <ElDialog
      v-model="instanceVisible"
      title="Add Plugin Instance"
      width="420px"
      destroy-on-close
      align-center
    >
      <ElForm label-position="top" class="dialog-form">
        <ElFormItem label="Instance ID">
          <ElInput v-model="instanceId" placeholder="Leave empty to let the backend generate it" />
        </ElFormItem>
      </ElForm>

      <template #footer>
        <div class="dialog-footer">
          <ElButton @click="instanceVisible = false">Cancel</ElButton>
          <ElButton type="primary" :loading="instanceCreating" @click="confirmCreateInstance">
            Create
          </ElButton>
        </div>
      </template>
    </ElDialog>

    <ElDialog
      v-model="configVisible"
      :title="`Plugin Config: ${currentPlugin?.name || currentPlugin?.plugin_id || ''}`"
      width="920px"
      destroy-on-close
      align-center
    >
      <div class="config-layout">
        <ElCard shadow="never" class="config-card">
          <template #header>
            <div class="section-title">Schema Preview</div>
          </template>

          <ElAlert
            v-if="schemaError"
            type="error"
            :closable="false"
            show-icon
            :title="schemaError"
          />

          <ElInput v-else :model-value="prettySchema" readonly type="textarea" :rows="16" />
        </ElCard>

        <ElCard shadow="never" class="config-card">
          <template #header>
            <div class="section-title">Raw Config JSON</div>
          </template>

          <ElInput v-model="configJson" type="textarea" :rows="16" placeholder='{"key":"value"}' />
          <div class="config-tip">
            The backend validates this JSON. Leave it empty if the plugin does not need extra
            config.
          </div>
        </ElCard>
      </div>

      <template #footer>
        <div class="dialog-footer">
          <ElButton @click="configVisible = false">Cancel</ElButton>
          <ElButton type="primary" :loading="configSaving" @click="saveConfig">Save</ElButton>
        </div>
      </template>
    </ElDialog>

    <ElDrawer v-model="manifestVisible" title="Plugin Manifest" size="620px" destroy-on-close>
      <ElDescriptions :column="1" border>
        <ElDescriptionsItem label="Plugin ID">
          {{ currentPlugin?.plugin_id || '-' }}
        </ElDescriptionsItem>
        <ElDescriptionsItem label="Category">
          {{ currentPlugin?.category || '-' }}
        </ElDescriptionsItem>
        <ElDescriptionsItem label="Name">
          {{ currentPlugin?.name || '-' }}
        </ElDescriptionsItem>
        <ElDescriptionsItem label="Version">
          {{ currentPlugin?.version || '-' }}
        </ElDescriptionsItem>
      </ElDescriptions>

      <ElInput
        class="manifest-json"
        :model-value="prettyManifest"
        readonly
        type="textarea"
        :rows="22"
      />
    </ElDrawer>

    <ElDialog
      v-model="methodsVisible"
      title="Payment Methods"
      width="640px"
      destroy-on-close
      align-center
    >
      <ElAlert
        type="info"
        :closable="false"
        show-icon
        title="Host Managed Methods"
        description="Supported payment methods come from the plugin manifest. Their enable state is managed by the host."
      />

      <ElTable :data="paymentMethods" border row-key="method" class="methods-table">
        <ElTableColumn prop="method" label="Method" min-width="220" />
        <ElTableColumn label="Enabled" width="140">
          <template #default="{ row }">
            <ElSwitch
              :model-value="row.enabled"
              :disabled="!canUpdate"
              :loading="methodBusyKey === row.method"
              @change="handleToggleMethod(row.method, $event)"
            />
          </template>
        </ElTableColumn>
      </ElTable>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import type {
    AdminPluginRecord,
    PluginDiscoverRecord,
    PluginPaymentMethodRecord
  } from '@/api/admin'
  import {
    createAdminPluginInstance,
    deleteAdminPluginInstance,
    disableAdminPluginInstance,
    enableAdminPluginInstance,
    fetchAdminPluginDiscover,
    fetchAdminPluginInstanceConfig,
    fetchAdminPluginInstanceConfigSchema,
    fetchAdminPluginPaymentMethods,
    fetchAdminPlugins,
    hasAdminPermission,
    importAdminPluginFromDisk,
    installAdminPlugin,
    updateAdminPluginInstanceConfig,
    updateAdminPluginPaymentMethod
  } from '@/api/admin'
  import { useUserStore } from '@/store/modules/user'
  import { ElMessage, type UploadFile } from 'element-plus'

  defineOptions({ name: 'SettingsPluginsPage' })

  interface PluginTableRow extends AdminPluginRecord {
    row_key: string
  }

  interface PluginDiscoverRow extends PluginDiscoverRecord {
    row_key: string
  }

  interface PaymentMethodRow extends PluginPaymentMethodRecord {
    method: string
    enabled: boolean
  }

  const DEFAULT_INSTANCE_ID = 'default'

  const loading = ref(false)
  const discoverLoading = ref(false)
  const installing = ref(false)
  const importing = ref(false)
  const instanceCreating = ref(false)
  const configSaving = ref(false)
  const initialized = ref(false)

  const busyRowKey = ref('')
  const importBusyKey = ref('')
  const methodBusyKey = ref('')

  const category = ref('all')
  const keyword = ref('')

  const tableRows = ref<PluginTableRow[]>([])
  const discoveredRows = ref<PluginDiscoverRow[]>([])
  const paymentMethods = ref<PaymentMethodRow[]>([])

  const installPasswordVisible = ref(false)
  const importPasswordVisible = ref(false)
  const discoverVisible = ref(false)
  const instanceVisible = ref(false)
  const configVisible = ref(false)
  const manifestVisible = ref(false)
  const methodsVisible = ref(false)

  const installAdminPassword = ref('')
  const importAdminPassword = ref('')
  const instanceId = ref('')
  const configJson = ref('{}')
  const schemaJson = ref('{}')
  const schemaError = ref('')

  const pendingInstallFile = ref<File | null>(null)
  const pendingImportItem = ref<PluginDiscoverRow | null>(null)
  const currentPlugin = ref<PluginTableRow | null>(null)
  const instanceTarget = ref<PluginTableRow | null>(null)

  const categoryOptions = [
    { label: 'All', value: 'all' },
    { label: 'payment', value: 'payment' },
    { label: 'sms', value: 'sms' },
    { label: 'kyc', value: 'kyc' },
    { label: 'automation', value: 'automation' }
  ]

  const userStore = useUserStore()
  const { info } = storeToRefs(userStore)

  const canView = computed(() => hasAdminPermission(info.value?.buttons, ['plugin.list']))
  const canPluginView = computed(() => hasAdminPermission(info.value?.buttons, ['plugin.view']))
  const canCreate = computed(() => hasAdminPermission(info.value?.buttons, ['plugin.create']))
  const canUpdate = computed(() => hasAdminPermission(info.value?.buttons, ['plugin.update']))
  const canDelete = computed(() => hasAdminPermission(info.value?.buttons, ['plugin.delete']))
  const canInstall = computed(() =>
    hasAdminPermission(info.value?.buttons, ['plugin.create', 'plugin.upload'])
  )

  const filteredRows = computed(() => {
    const currentCategory = String(category.value || 'all').trim()
    const currentKeyword = String(keyword.value || '')
      .trim()
      .toLowerCase()

    return tableRows.value.filter((item) => {
      if (currentCategory !== 'all' && String(item.category || '') !== currentCategory) {
        return false
      }

      if (!currentKeyword) {
        return true
      }

      const haystack = [item.name, item.plugin_id, item.category, item.instance_id, item.version]
        .map((value) => String(value || '').toLowerCase())
        .join(' ')

      return haystack.includes(currentKeyword)
    })
  })

  const prettySchema = computed(() => normalizePrettyJson(schemaJson.value))
  const prettyManifest = computed(() =>
    normalizePrettyJson(JSON.stringify(currentPlugin.value?.manifest || {}))
  )

  watch(
    canView,
    (value) => {
      if (value && !initialized.value) {
        fetchData()
      }
    },
    { immediate: true }
  )

  function getRowKey(categoryValue?: string, pluginId?: string, instanceId?: string) {
    return `${String(categoryValue || '-')}/${String(pluginId || '-')}/${String(instanceId || DEFAULT_INSTANCE_ID)}`
  }

  function normalizePrettyJson(value: string) {
    const text = String(value || '').trim()
    if (!text) {
      return '{}'
    }

    try {
      return JSON.stringify(JSON.parse(text), null, 2)
    } catch {
      return text
    }
  }

  function normalizePluginRow(item?: AdminPluginRecord): PluginTableRow {
    const instanceIdValue =
      String(item?.instance_id || DEFAULT_INSTANCE_ID).trim() || DEFAULT_INSTANCE_ID

    return {
      ...item,
      instance_id: instanceIdValue,
      row_key: getRowKey(item?.category, item?.plugin_id, instanceIdValue)
    }
  }

  function normalizeDiscoverRow(item?: PluginDiscoverRecord): PluginDiscoverRow {
    return {
      ...item,
      row_key: `${String(item?.category || '-')}/${String(item?.plugin_id || '-')}`
    }
  }

  function getSignatureTagType(signature?: string) {
    if (signature === 'official') return 'success' as const
    if (signature === 'unsigned') return 'warning' as const
    if (signature === 'untrusted') return 'danger' as const
    return 'info' as const
  }

  function getHealthTagType(status?: string) {
    const normalized = String(status || '')
      .trim()
      .toLowerCase()
    if (normalized === 'ok') return 'success' as const
    if (normalized === 'degraded') return 'warning' as const
    if (normalized === 'error') return 'danger' as const
    return 'info' as const
  }

  function formatDateTime(value?: string | null) {
    if (!value) return '-'

    const date = new Date(value)
    return Number.isNaN(date.getTime()) ? String(value) : date.toLocaleString('zh-CN')
  }

  async function fetchData() {
    if (!canView.value) return

    loading.value = true

    try {
      const payload = await fetchAdminPlugins()
      tableRows.value = (payload.items || []).map((item) => normalizePluginRow(item))
      initialized.value = true
    } finally {
      loading.value = false
    }
  }

  async function handleToggleEnabled(row: PluginTableRow, checked: boolean | string | number) {
    const enabled = Boolean(checked)
    busyRowKey.value = row.row_key

    try {
      if (enabled) {
        await enableAdminPluginInstance(
          String(row.category || ''),
          String(row.plugin_id || ''),
          String(row.instance_id || DEFAULT_INSTANCE_ID)
        )
      } else {
        await disableAdminPluginInstance(
          String(row.category || ''),
          String(row.plugin_id || ''),
          String(row.instance_id || DEFAULT_INSTANCE_ID)
        )
      }

      row.enabled = enabled
      ElMessage.success('Plugin state updated successfully')
    } catch (error: any) {
      const data = error?.response?.data || {}
      const errorCode = String(data?.code || '')
      const missingFields = Array.isArray(data?.missing_fields) ? data.missing_fields : []

      if (enabled && errorCode === 'missing_required_config') {
        const hint = missingFields.length > 0 ? ` Missing fields: ${missingFields.join(', ')}` : ''
        ElMessage.error(`${String(data?.error || 'Plugin config is incomplete')}.${hint}`.trim())
      } else {
        ElMessage.error(data?.error || 'Failed to update plugin state')
      }
    } finally {
      busyRowKey.value = ''
    }
  }

  async function handleInstallFileChange(uploadFile: UploadFile) {
    if (!uploadFile.raw) return

    pendingInstallFile.value = uploadFile.raw
    await tryInstall(uploadFile.raw)
  }

  async function tryInstall(file: File, adminPassword?: string) {
    installing.value = true

    try {
      await installAdminPlugin(file, adminPassword)
      pendingInstallFile.value = null
      installAdminPassword.value = ''
      installPasswordVisible.value = false
      ElMessage.success('Plugin installed successfully')
      await fetchData()
    } catch (error: any) {
      const status = Number(error?.response?.status || 0)
      const message = String(error?.response?.data?.error || 'Install failed')

      if (status === 403 && message.includes('admin_password')) {
        installPasswordVisible.value = true
      } else {
        ElMessage.error(message)
      }
    } finally {
      installing.value = false
    }
  }

  async function confirmInstall() {
    if (!pendingInstallFile.value) return

    if (!String(installAdminPassword.value || '').trim()) {
      ElMessage.error('Please enter the admin password')
      return
    }

    await tryInstall(pendingInstallFile.value, String(installAdminPassword.value || '').trim())
  }

  async function openDiscoverDialog() {
    discoverVisible.value = true
    discoverLoading.value = true

    try {
      const payload = await fetchAdminPluginDiscover()
      discoveredRows.value = (payload.items || []).map((item) => normalizeDiscoverRow(item))
    } finally {
      discoverLoading.value = false
    }
  }

  async function startImport(row: PluginDiscoverRow) {
    if (row.signature_status !== 'official') {
      pendingImportItem.value = row
      importPasswordVisible.value = true
      return
    }

    await importPlugin(row)
  }

  async function confirmImport() {
    if (!pendingImportItem.value) return

    if (!String(importAdminPassword.value || '').trim()) {
      ElMessage.error('Please enter the admin password')
      return
    }

    await importPlugin(pendingImportItem.value, String(importAdminPassword.value || '').trim())
  }

  async function importPlugin(row: PluginDiscoverRow, adminPassword?: string) {
    importBusyKey.value = row.row_key
    importing.value = true

    try {
      await importAdminPluginFromDisk(
        String(row.category || ''),
        String(row.plugin_id || ''),
        adminPassword
      )
      pendingImportItem.value = null
      importAdminPassword.value = ''
      importPasswordVisible.value = false
      ElMessage.success('Plugin imported successfully')
      await openDiscoverDialog()
      await fetchData()
    } catch (error: any) {
      ElMessage.error(error?.response?.data?.error || 'Import failed')
    } finally {
      importBusyKey.value = ''
      importing.value = false
    }
  }

  function openInstanceDialog(row: PluginTableRow) {
    instanceTarget.value = row
    instanceId.value = ''
    instanceVisible.value = true
  }

  async function confirmCreateInstance() {
    if (!instanceTarget.value) return

    instanceCreating.value = true

    try {
      await createAdminPluginInstance(
        String(instanceTarget.value.category || ''),
        String(instanceTarget.value.plugin_id || ''),
        {
          instance_id: String(instanceId.value || '').trim()
        }
      )
      instanceVisible.value = false
      ElMessage.success('Plugin instance created successfully')
      await fetchData()
    } catch (error: any) {
      ElMessage.error(error?.response?.data?.error || 'Failed to create plugin instance')
    } finally {
      instanceCreating.value = false
    }
  }

  async function openConfigDialog(row: PluginTableRow) {
    if (String(row.category || '') === 'automation') {
      ElMessage.info('Automation plugin configuration has moved to the catalog page')
      window.location.href = '/catalog'
      return
    }

    currentPlugin.value = row
    configVisible.value = true
    schemaError.value = ''
    schemaJson.value = '{}'
    configJson.value = '{}'

    try {
      const [schemaPayload, configPayload] = await Promise.all([
        fetchAdminPluginInstanceConfigSchema(
          String(row.category || ''),
          String(row.plugin_id || ''),
          String(row.instance_id || DEFAULT_INSTANCE_ID)
        ),
        fetchAdminPluginInstanceConfig(
          String(row.category || ''),
          String(row.plugin_id || ''),
          String(row.instance_id || DEFAULT_INSTANCE_ID)
        )
      ])

      schemaJson.value = String(schemaPayload.json_schema || '{}')
      configJson.value = normalizePrettyJson(String(configPayload.config_json || '{}'))

      try {
        JSON.parse(schemaJson.value)
      } catch {
        schemaError.value = 'Schema JSON could not be parsed'
      }
    } catch (error: any) {
      schemaError.value = error?.response?.data?.error || 'Failed to load plugin config'
    }
  }

  async function saveConfig() {
    if (!currentPlugin.value) return

    configSaving.value = true

    try {
      const normalizedConfig = normalizePrettyJson(configJson.value)
      JSON.parse(normalizedConfig)

      await updateAdminPluginInstanceConfig(
        String(currentPlugin.value.category || ''),
        String(currentPlugin.value.plugin_id || ''),
        String(currentPlugin.value.instance_id || DEFAULT_INSTANCE_ID),
        normalizedConfig
      )

      configJson.value = normalizedConfig
      configVisible.value = false
      ElMessage.success('Plugin config saved successfully')
      await fetchData()
    } catch (error: any) {
      if (error instanceof SyntaxError) {
        ElMessage.error('Config JSON is invalid')
      } else {
        ElMessage.error(error?.response?.data?.error || 'Failed to save plugin config')
      }
    } finally {
      configSaving.value = false
    }
  }

  function openManifest(row: PluginTableRow) {
    currentPlugin.value = row
    manifestVisible.value = true
  }

  async function openMethodsDialog(row: PluginTableRow) {
    currentPlugin.value = row
    methodsVisible.value = true

    try {
      const payload = await fetchAdminPluginPaymentMethods({
        category: String(row.category || 'payment'),
        plugin_id: String(row.plugin_id || ''),
        instance_id: String(row.instance_id || DEFAULT_INSTANCE_ID)
      })

      paymentMethods.value = (payload.items || []).map((item) => ({
        method: String(item.method || ''),
        enabled: Boolean(item.enabled)
      }))
    } catch (error: any) {
      ElMessage.error(error?.response?.data?.error || 'Failed to load payment methods')
    }
  }

  async function handleToggleMethod(method: string, checked: boolean | string | number) {
    if (!currentPlugin.value) return

    methodBusyKey.value = method

    try {
      await updateAdminPluginPaymentMethod({
        category: String(currentPlugin.value.category || 'payment'),
        plugin_id: String(currentPlugin.value.plugin_id || ''),
        instance_id: String(currentPlugin.value.instance_id || DEFAULT_INSTANCE_ID),
        method,
        enabled: Boolean(checked)
      })

      const target = paymentMethods.value.find((item) => item.method === method)
      if (target) {
        target.enabled = Boolean(checked)
      }

      ElMessage.success('Payment method updated successfully')
    } catch (error: any) {
      ElMessage.error(error?.response?.data?.error || 'Failed to update payment method')
    } finally {
      methodBusyKey.value = ''
    }
  }

  async function deleteInstance(row: PluginTableRow) {
    try {
      await deleteAdminPluginInstance(
        String(row.category || ''),
        String(row.plugin_id || ''),
        String(row.instance_id || DEFAULT_INSTANCE_ID)
      )
      ElMessage.success('Plugin instance deleted successfully')
      await fetchData()
    } catch (error: any) {
      ElMessage.error(error?.response?.data?.error || 'Failed to delete plugin instance')
    }
  }
</script>

<style scoped lang="scss">
  .plugins-settings-page {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .page-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 16px;
  }

  .page-kicker {
    color: var(--el-text-color-placeholder);
    font-size: 12px;
    font-weight: 700;
    letter-spacing: 0.18em;
    text-transform: uppercase;
  }

  .page-title {
    margin-top: 4px;
    color: var(--el-text-color-primary);
    font-size: 22px;
    font-weight: 700;
  }

  .page-subtitle,
  .health-subtle,
  .config-tip {
    margin-top: 6px;
    color: var(--el-text-color-secondary);
    font-size: 13px;
    line-height: 1.7;
  }

  .page-actions,
  .dialog-footer,
  .table-actions,
  .capability-list {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .filter-card,
  .table-card,
  .config-card {
    border: none;
    border-radius: 18px;
  }

  .filter-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
  }

  .keyword-input {
    width: 320px;
    max-width: 100%;
  }

  .plugin-cell,
  .health-cell {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .plugin-name,
  .section-title {
    color: var(--el-text-color-primary);
    font-size: 16px;
    font-weight: 700;
  }

  .plugin-meta {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;
  }

  .mono {
    font-family:
      ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New',
      monospace;
  }

  .table-actions {
    justify-content: flex-end;
    gap: 4px;
  }

  .dialog-form,
  .discover-table,
  .methods-table,
  .manifest-json {
    margin-top: 16px;
  }

  .config-layout {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 16px;
  }

  @media (max-width: 900px) {
    .config-layout {
      grid-template-columns: 1fr;
    }
  }

  @media (max-width: 768px) {
    .page-header,
    .filter-row {
      flex-direction: column;
      align-items: stretch;
    }

    .page-actions {
      flex-wrap: wrap;
    }

    .keyword-input {
      width: 100%;
    }
  }
</style>
