<template>
  <div v-loading="loading" class="sms-settings-page art-full-height">
    <div class="page-header">
      <div>
        <div class="page-kicker">Messaging</div>
        <div class="page-title">SMS Settings</div>
        <div class="page-subtitle">
          Manage SMS plugin binding, default templates, preview rendering, and test delivery.
        </div>
      </div>

      <div class="page-actions">
        <ElButton v-if="canView" :disabled="loading || saving" @click="loadInitialData">
          Refresh
        </ElButton>
        <ElButton v-if="canUpdate" type="primary" :loading="saving" @click="handleSaveConfig">
          Save Config
        </ElButton>
      </div>
    </div>

    <ElEmpty v-if="!canView" description="You do not have permission to view SMS settings." />

    <template v-else>
      <ElCard shadow="never" class="section-card">
        <template #header>
          <div class="section-title">SMS Channel</div>
        </template>

        <ElForm label-position="top" :disabled="formDisabled">
          <ElRow :gutter="16">
            <ElCol :xs="24" :md="8">
              <ElFormItem label="Enable SMS">
                <div class="switch-row">
                  <ElSwitch v-model="config.enabled" />
                  <span class="field-tip">
                    Disabling SMS will block any workflow that depends on text delivery.
                  </span>
                </div>
              </ElFormItem>
            </ElCol>

            <ElCol :xs="24" :md="8">
              <ElFormItem label="Plugin Instance">
                <ElSelect
                  v-model="selectedBinding"
                  clearable
                  filterable
                  placeholder="Select an SMS plugin instance"
                  class="full-width"
                  @change="handleBindingChange"
                >
                  <ElOption
                    v-for="item in smsPluginOptions"
                    :key="item.value"
                    :label="item.label"
                    :value="item.value"
                  />
                </ElSelect>
              </ElFormItem>
            </ElCol>

            <ElCol :xs="24" :md="8">
              <ElFormItem label="Provider Template ID">
                <ElInput
                  v-model="config.provider_template_id"
                  maxlength="255"
                  placeholder="Optional provider-side template code"
                />
              </ElFormItem>
            </ElCol>
          </ElRow>

          <ElRow :gutter="16">
            <ElCol :xs="24" :md="8">
              <ElFormItem label="Default Template">
                <ElSelect
                  v-model="config.default_template_id"
                  clearable
                  placeholder="Choose a default template"
                  class="full-width"
                >
                  <ElOption
                    v-for="item in templates"
                    :key="String(item.id || '')"
                    :label="item.name || '-'"
                    :value="String(item.id || '')"
                  />
                </ElSelect>
              </ElFormItem>
            </ElCol>

            <ElCol :xs="24" :md="16">
              <ElFormItem label="Quick Test Phones">
                <div class="test-row">
                  <ElInput
                    v-model="quickTestPhone"
                    maxlength="500"
                    placeholder="13800138000,13800138001"
                  />
                  <ElButton
                    type="primary"
                    :loading="quickTesting"
                    :disabled="formDisabled"
                    @click="handleQuickTest"
                  >
                    Send Test
                  </ElButton>
                </div>
              </ElFormItem>
            </ElCol>
          </ElRow>
        </ElForm>
      </ElCard>

      <ElCard shadow="never" class="section-card">
        <template #header>
          <div class="table-header">
            <div class="section-title">SMS Templates</div>
            <div class="table-actions">
              <ElButton :disabled="loading || templateLoading" @click="loadTemplates">
                Refresh List
              </ElButton>
              <ElButton v-if="canUpdate" type="primary" @click="openTemplateDialog()">
                New Template
              </ElButton>
            </div>
          </div>
        </template>

        <ElTable
          :data="templates"
          border
          row-key="id"
          v-loading="templateLoading"
          class="template-table"
        >
          <ElTableColumn prop="id" label="ID" width="88" />
          <ElTableColumn prop="name" label="Name" min-width="180" />
          <ElTableColumn label="Content" min-width="420" show-overflow-tooltip>
            <template #default="{ row }">
              <span>{{ row.content || '-' }}</span>
            </template>
          </ElTableColumn>
          <ElTableColumn label="Enabled" width="100">
            <template #default="{ row }">
              <ElTag :type="row.enabled !== false ? 'success' : 'danger'">
                {{ row.enabled !== false ? 'On' : 'Off' }}
              </ElTag>
            </template>
          </ElTableColumn>
          <ElTableColumn label="Actions" width="180" fixed="right">
            <template #default="{ row }">
              <ElSpace>
                <ElButton link type="primary" @click="openTemplateDialog(row)">Edit</ElButton>
                <ElButton
                  link
                  type="danger"
                  :disabled="!canUpdate"
                  @click="handleDeleteTemplate(row)"
                >
                  Delete
                </ElButton>
              </ElSpace>
            </template>
          </ElTableColumn>
        </ElTable>
      </ElCard>
    </template>

    <ElDialog
      v-model="dialogVisible"
      title="SMS Template"
      width="880px"
      destroy-on-close
      align-center
    >
      <ElForm label-position="top" :disabled="dialogDisabled">
        <ElRow :gutter="16">
          <ElCol :xs="24" :md="16">
            <ElFormItem label="Name">
              <ElInput
                v-model="templateForm.name"
                maxlength="120"
                placeholder="Enter template name"
              />
            </ElFormItem>
          </ElCol>

          <ElCol :xs="24" :md="8">
            <ElFormItem label="Enabled">
              <ElSwitch v-model="templateForm.enabled" />
            </ElFormItem>
          </ElCol>
        </ElRow>

        <ElFormItem label="Template Content">
          <ElInput
            v-model="templateForm.content"
            type="textarea"
            :rows="6"
            maxlength="5000"
            show-word-limit
            placeholder="Your verification code is {{code}}."
          />
          <div class="field-tip">
            <span v-pre>Supported variables: {{ code }} / {{ phone }} / {{ now }}</span>
          </div>
        </ElFormItem>

        <ElCollapse>
          <ElCollapseItem title="Preview" name="preview">
            <div class="test-row">
              <ElInput v-model="previewPhone" maxlength="64" placeholder="Preview phone" />
              <ElInput v-model="previewCode" maxlength="32" placeholder="Preview code" />
              <ElButton
                :loading="previewLoading"
                :disabled="dialogDisabled"
                @click="handlePreviewTemplate"
              >
                Render
              </ElButton>
            </div>
            <pre class="preview-block">{{ previewContent || 'No preview content yet.' }}</pre>
          </ElCollapseItem>

          <ElCollapseItem title="Send Test" name="test">
            <div class="test-row">
              <ElInput
                v-model="templateTestPhone"
                maxlength="500"
                placeholder="Comma-separated phone numbers"
              />
              <ElButton
                type="primary"
                :loading="templateTesting"
                :disabled="dialogDisabled"
                @click="handleTestTemplate"
              >
                Send
              </ElButton>
            </div>
          </ElCollapseItem>
        </ElCollapse>
      </ElForm>

      <template #footer>
        <div class="dialog-footer">
          <ElButton @click="dialogVisible = false">Cancel</ElButton>
          <ElButton
            type="primary"
            :loading="templateSaving"
            :disabled="dialogDisabled"
            @click="handleSaveTemplate"
          >
            Save Template
          </ElButton>
        </div>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import type { AdminPluginRecord, SmsConfigRecord, SmsTemplateRecord } from '@/api/admin'
  import {
    createSmsTemplate,
    deleteSmsTemplate,
    fetchAdminPlugins,
    fetchSmsConfig,
    fetchSmsTemplates,
    hasAdminPermission,
    previewSmsConfig,
    testSmsConfig,
    updateSmsConfig,
    updateSmsTemplate
  } from '@/api/admin'
  import { useUserStore } from '@/store/modules/user'
  import { ElMessage, ElMessageBox } from 'element-plus'

  defineOptions({ name: 'SettingsSmsPage' })

  interface SmsSettingsForm {
    enabled: boolean
    plugin_id: string
    instance_id: string
    default_template_id: string
    provider_template_id: string
  }

  interface SmsTemplateForm {
    id: number | null
    name: string
    content: string
    enabled: boolean
  }

  const loading = ref(false)
  const saving = ref(false)
  const templateLoading = ref(false)
  const templateSaving = ref(false)
  const quickTesting = ref(false)
  const previewLoading = ref(false)
  const templateTesting = ref(false)
  const initialized = ref(false)

  const userStore = useUserStore()
  const { info } = storeToRefs(userStore)

  const config = reactive<SmsSettingsForm>(createDefaultConfig())
  const templateForm = reactive<SmsTemplateForm>(createDefaultTemplateForm())

  const pluginItems = ref<AdminPluginRecord[]>([])
  const templates = ref<SmsTemplateRecord[]>([])
  const selectedBinding = ref('')
  const quickTestPhone = ref('')

  const dialogVisible = ref(false)
  const previewPhone = ref('13800138000')
  const previewCode = ref('123456')
  const previewContent = ref('')
  const templateTestPhone = ref('')

  const canView = computed(() => hasAdminPermission(info.value?.buttons, ['settings.view']))
  const canUpdate = computed(() => hasAdminPermission(info.value?.buttons, ['settings.update']))
  const formDisabled = computed(() => !canUpdate.value || loading.value || saving.value)
  const dialogDisabled = computed(() => !canUpdate.value || templateSaving.value)

  const smsPluginOptions = computed(() =>
    pluginItems.value
      .filter((item) => item.category === 'sms' && item.enabled === true && item.loaded === true)
      .map((item) => {
        const pluginId = String(item.plugin_id || '')
        const instanceId = String(item.instance_id || 'default')
        const name = String(item.name || pluginId || 'sms')

        return {
          value: `${pluginId}::${instanceId}`,
          label: `${name} (${pluginId}/${instanceId})`
        }
      })
  )

  watch(
    canView,
    (value) => {
      if (value && !initialized.value) {
        loadInitialData()
      }
    },
    { immediate: true }
  )

  function createDefaultConfig(): SmsSettingsForm {
    return {
      enabled: true,
      plugin_id: '',
      instance_id: 'default',
      default_template_id: '',
      provider_template_id: ''
    }
  }

  function createDefaultTemplateForm(): SmsTemplateForm {
    return {
      id: null,
      name: '',
      content: '',
      enabled: true
    }
  }

  function applyConfig(configData?: SmsConfigRecord) {
    Object.assign(config, createDefaultConfig(), {
      enabled: configData?.enabled !== false,
      plugin_id: String(configData?.plugin_id || ''),
      instance_id: String(configData?.instance_id || 'default'),
      default_template_id: String(configData?.default_template_id || ''),
      provider_template_id: String(configData?.provider_template_id || '')
    })

    selectedBinding.value = config.plugin_id
      ? `${config.plugin_id}::${config.instance_id || 'default'}`
      : ''
  }

  function normalizeTemplate(row?: SmsTemplateRecord | null): SmsTemplateRecord {
    return {
      id: Number(row?.id || 0) || undefined,
      name: String(row?.name || ''),
      content: String(row?.content || ''),
      enabled: row?.enabled !== false,
      created_at: row?.created_at,
      updated_at: row?.updated_at
    }
  }

  async function loadConfig() {
    const [configPayload, pluginsPayload] = await Promise.all([
      fetchSmsConfig(),
      fetchAdminPlugins()
    ])
    applyConfig(configPayload)
    pluginItems.value = (pluginsPayload.items || []) as AdminPluginRecord[]
  }

  async function loadTemplates() {
    templateLoading.value = true

    try {
      const payload = await fetchSmsTemplates()
      templates.value = (payload.items || []).map((item) => normalizeTemplate(item))
    } finally {
      templateLoading.value = false
    }
  }

  async function loadInitialData() {
    if (!canView.value) {
      return
    }

    loading.value = true

    try {
      await Promise.all([loadConfig(), loadTemplates()])
      initialized.value = true
    } finally {
      loading.value = false
    }
  }

  function handleBindingChange(value?: string) {
    if (!value) {
      config.plugin_id = ''
      config.instance_id = ''
      return
    }

    const [pluginId, instanceId] = String(value).split('::')
    config.plugin_id = pluginId || ''
    config.instance_id = instanceId || 'default'
  }

  async function handleSaveConfig() {
    saving.value = true

    try {
      await updateSmsConfig({
        enabled: config.enabled,
        plugin_id: config.plugin_id,
        instance_id: config.instance_id || 'default',
        default_template_id: config.default_template_id,
        provider_template_id: config.provider_template_id
      })

      ElMessage.success('SMS configuration saved')
    } finally {
      saving.value = false
    }
  }

  function openTemplateDialog(row?: SmsTemplateRecord) {
    Object.assign(templateForm, createDefaultTemplateForm())

    if (row) {
      Object.assign(templateForm, {
        id: Number(row.id || 0) || null,
        name: String(row.name || ''),
        content: String(row.content || ''),
        enabled: row.enabled !== false
      })
    }

    previewContent.value = ''
    templateTestPhone.value = ''
    dialogVisible.value = true
  }

  async function handleSaveTemplate() {
    if (!String(templateForm.name || '').trim()) {
      ElMessage.error('Please enter a template name')
      return
    }

    if (!String(templateForm.content || '').trim()) {
      ElMessage.error('Please enter template content')
      return
    }

    templateSaving.value = true

    try {
      const payload = {
        name: templateForm.name.trim(),
        content: templateForm.content.trim(),
        enabled: templateForm.enabled
      }

      if (templateForm.id) {
        await updateSmsTemplate(templateForm.id, payload)
      } else {
        await createSmsTemplate(payload)
      }

      ElMessage.success('Template saved')
      dialogVisible.value = false
      await loadTemplates()
    } finally {
      templateSaving.value = false
    }
  }

  async function handleDeleteTemplate(row: SmsTemplateRecord) {
    if (!row.id) {
      return
    }

    await ElMessageBox.confirm('Delete this SMS template?', 'Delete Template', {
      type: 'warning'
    })

    await deleteSmsTemplate(row.id)
    ElMessage.success('Template deleted')
    await loadTemplates()
  }

  async function handlePreviewTemplate() {
    if (!String(templateForm.content || '').trim()) {
      ElMessage.error('Please enter template content first')
      return
    }

    previewLoading.value = true

    try {
      const payload = await previewSmsConfig({
        content: templateForm.content.trim(),
        variables: {
          code: previewCode.value,
          phone: previewPhone.value
        }
      })

      previewContent.value = String(payload.content || '')
    } finally {
      previewLoading.value = false
    }
  }

  async function handleTestTemplate() {
    const phone = templateTestPhone.value.trim()
    if (!phone) {
      ElMessage.error('Please enter at least one test phone number')
      return
    }

    templateTesting.value = true

    try {
      const payload: Record<string, unknown> = {
        phone,
        plugin_id: config.plugin_id,
        instance_id: config.instance_id || 'default',
        provider_template_id: config.provider_template_id,
        variables: {
          code: previewCode.value || '123456',
          phone: previewPhone.value || phone.split(',')[0]
        }
      }

      if (templateForm.id) {
        payload.template_id = templateForm.id
      } else {
        const content = templateForm.content.trim()
        if (!content) {
          ElMessage.error('Please fill in the template content or save the template first')
          return
        }
        payload.content = content
      }

      await testSmsConfig(payload)
      ElMessage.success('Test SMS sent')
    } finally {
      templateTesting.value = false
    }
  }

  async function handleQuickTest() {
    const phone = quickTestPhone.value.trim()
    if (!phone) {
      ElMessage.error('Please enter at least one test phone number')
      return
    }

    const selectedTemplateId = Number(config.default_template_id || 0) || 0
    const fallbackTemplate = templates.value.find((item) => item.enabled !== false)
    const fallbackTemplateId = Number(fallbackTemplate?.id || 0) || 0
    const templateId = selectedTemplateId || fallbackTemplateId

    if (!templateId) {
      ElMessage.error('Choose a default template or enable at least one SMS template first')
      return
    }

    quickTesting.value = true

    try {
      await testSmsConfig({
        phone,
        template_id: templateId,
        plugin_id: config.plugin_id,
        instance_id: config.instance_id || 'default',
        provider_template_id: config.provider_template_id
      })

      ElMessage.success('Test SMS sent')
    } finally {
      quickTesting.value = false
    }
  }
</script>

<style scoped lang="scss">
  .sms-settings-page {
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
  .field-tip {
    margin-top: 6px;
    color: var(--el-text-color-secondary);
    font-size: 13px;
    line-height: 1.7;
  }

  .page-actions,
  .table-actions,
  .dialog-footer {
    display: flex;
    gap: 12px;
  }

  .section-card {
    border: none;
    border-radius: 18px;
  }

  .section-card + .section-card {
    margin-top: 16px;
  }

  .section-title {
    color: var(--el-text-color-primary);
    font-size: 16px;
    font-weight: 700;
  }

  .table-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }

  .switch-row,
  .test-row {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .test-row {
    width: 100%;
  }

  .test-row :deep(.el-input) {
    flex: 1;
  }

  .full-width {
    width: 100%;
  }

  .preview-block {
    min-height: 120px;
    margin-top: 12px;
    padding: 12px;
    overflow: auto;
    border-radius: 12px;
    background: var(--el-fill-color-light);
    color: var(--el-text-color-regular);
    line-height: 1.7;
    white-space: pre-wrap;
  }

  @media (max-width: 768px) {
    .page-header,
    .table-header,
    .test-row {
      flex-direction: column;
      align-items: stretch;
    }

    .page-actions,
    .table-actions {
      width: 100%;
    }

    .page-actions :deep(.el-button),
    .table-actions :deep(.el-button),
    .test-row :deep(.el-button) {
      width: 100%;
    }
  }
</style>
