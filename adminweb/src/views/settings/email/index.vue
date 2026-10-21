<template>
  <div v-loading="loading" class="email-settings-page art-full-height">
    <div class="page-header">
      <div>
        <div class="page-kicker">Messaging</div>
        <div class="page-title">Email Settings</div>
        <div class="page-subtitle"
          >Manage SMTP, email switches, default templates, preview, and test sends.</div
        >
      </div>

      <div class="page-actions">
        <ElButton v-if="canView" :disabled="loading || saving" @click="loadInitialData">
          Refresh
        </ElButton>
        <ElButton v-if="canUpdate" type="primary" :loading="saving" @click="handleSaveSettings">
          Save Settings
        </ElButton>
      </div>
    </div>

    <ElEmpty v-if="!canView" description="You do not have permission to view email settings." />

    <template v-else>
      <ElCard shadow="never" class="section-card">
        <template #header>
          <div class="section-title">SMTP and Delivery</div>
        </template>

        <ElForm label-position="top" :disabled="formDisabled">
          <ElRow :gutter="16">
            <ElCol :xs="24" :md="12">
              <ElFormItem label="SMTP Host">
                <ElInput v-model="form.smtp_host" maxlength="255" placeholder="smtp.example.com" />
              </ElFormItem>
            </ElCol>

            <ElCol :xs="24" :md="12">
              <ElFormItem label="SMTP Port">
                <ElInput v-model="form.smtp_port" maxlength="32" placeholder="587" />
              </ElFormItem>
            </ElCol>
          </ElRow>

          <ElRow :gutter="16">
            <ElCol :xs="24" :md="12">
              <ElFormItem label="SMTP User">
                <ElInput
                  v-model="form.smtp_user"
                  maxlength="255"
                  placeholder="mailer@example.com"
                />
              </ElFormItem>
            </ElCol>

            <ElCol :xs="24" :md="12">
              <ElFormItem label="SMTP Password">
                <ElInput
                  v-model="form.smtp_pass"
                  type="password"
                  show-password
                  maxlength="255"
                  placeholder="SMTP password"
                />
              </ElFormItem>
            </ElCol>
          </ElRow>

          <ElFormItem label="SMTP From">
            <ElInput v-model="form.smtp_from" maxlength="255" placeholder="noreply@example.com" />
          </ElFormItem>

          <ElRow :gutter="16">
            <ElCol :xs="24" :md="8">
              <ElFormItem label="Enable SMTP">
                <div class="switch-row">
                  <ElSwitch v-model="form.smtp_enabled" />
                  <span class="field-tip"
                    >Used by test sending and all template-based delivery.</span
                  >
                </div>
              </ElFormItem>
            </ElCol>

            <ElCol :xs="24" :md="8">
              <ElFormItem label="Enable Email">
                <div class="switch-row">
                  <ElSwitch v-model="form.email_enabled" />
                  <span class="field-tip"
                    >Controls whether email notifications are allowed globally.</span
                  >
                </div>
              </ElFormItem>
            </ElCol>

            <ElCol :xs="24" :md="8">
              <ElFormItem label="Expiration Reminder Email">
                <div class="switch-row">
                  <ElSwitch v-model="form.email_expire_enabled" />
                  <span class="field-tip">Sends advance reminders before service expiry.</span>
                </div>
              </ElFormItem>
            </ElCol>
          </ElRow>

          <ElRow :gutter="16">
            <ElCol :xs="24" :md="8">
              <ElFormItem label="Reminder Days">
                <ElInputNumber
                  v-model="form.expire_reminder_days"
                  :min="1"
                  :max="90"
                  :step="1"
                  class="full-width"
                />
              </ElFormItem>
            </ElCol>

            <ElCol :xs="24" :md="16">
              <ElFormItem label="SMTP Test">
                <div class="test-row">
                  <ElInput
                    v-model="smtpTestTo"
                    maxlength="255"
                    placeholder="Recipient email address"
                  />
                  <ElButton
                    type="primary"
                    :loading="smtpTesting"
                    :disabled="formDisabled"
                    @click="handleSmtpTest"
                  >
                    Send Test
                  </ElButton>
                </div>
                <div class="field-tip">
                  If an enabled template exists, the first enabled template will be used for SMTP
                  test.
                </div>
              </ElFormItem>
            </ElCol>
          </ElRow>
        </ElForm>
      </ElCard>

      <ElCard shadow="never" class="section-card">
        <template #header>
          <div class="table-header">
            <div class="section-title">Email Templates</div>
            <div class="table-actions">
              <ElButton :disabled="loading || templateLoading" @click="loadTemplates"
                >Refresh List</ElButton
              >
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
          <ElTableColumn prop="subject" label="Subject" min-width="260" show-overflow-tooltip />
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
      title="Email Template"
      width="1080px"
      destroy-on-close
      align-center
    >
      <ElForm label-position="top" :disabled="dialogDisabled">
        <div class="template-toolbar">
          <div class="toolbar-left">
            <ElSelect
              v-model="defaultTemplateKey"
              clearable
              placeholder="Choose a default template"
              style="width: 240px"
            >
              <ElOption
                v-for="item in defaultTemplateOptions"
                :key="item.key"
                :label="item.label"
                :value="item.key"
              />
            </ElSelect>
            <ElButton :disabled="dialogDisabled" @click="applyDefaultTemplate"
              >Fill Default</ElButton
            >
          </div>

          <div class="toolbar-right">
            <ElButton :disabled="dialogDisabled" @click="openPreview">Preview</ElButton>
            <ElButton :disabled="dialogDisabled" @click="helpVisible = true">Help</ElButton>
          </div>
        </div>

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

        <ElFormItem label="Subject">
          <ElInput v-model="templateForm.subject" maxlength="255" placeholder="Email subject" />
        </ElFormItem>

        <ElFormItem label="Body">
          <div class="editor-hint">
            Use the editor for HTML content, or type plain text directly. Template variables will be
            rendered with mock data in preview and tests.
          </div>
          <ArtWangEditor v-model="templateForm.body" height="380px" />
        </ElFormItem>

        <ElCollapse>
          <ElCollapseItem title="Template Variables and Mock Data" name="vars">
            <ElAlert type="info" :closable="false" show-icon>
              <template #title>Variable Examples</template>
              <template #default>
                <div class="template-vars">
                  <div
                    ><code v-pre>{{ .user.id }}</code> : {{ mockData.user.id }}</div
                  >
                  <div
                    ><code v-pre>{{ .user.username }}</code> : {{ mockData.user.username }}</div
                  >
                  <div
                    ><code v-pre>{{ .user.email }}</code> : {{ mockData.user.email }}</div
                  >
                  <div
                    ><code v-pre>{{ .user.qq }}</code> : {{ mockData.user.qq }}</div
                  >
                  <div
                    ><code v-pre>{{ .order.no }}</code> : {{ mockData.order.no }}</div
                  >
                  <div
                    ><code v-pre>{{ .vps.name }}</code> : {{ mockData.vps.name }}</div
                  >
                  <div
                    ><code v-pre>{{ .vps.expire_at }}</code> : {{ mockData.vps.expire_at }}</div
                  >
                  <div
                    ><code v-pre>{{ .message }}</code> : {{ mockData.message }}</div
                  >
                </div>
              </template>
            </ElAlert>

            <div class="mock-block">
              <div class="mock-title">Mock Payload</div>
              <pre class="mock-data">{{ mockDataJson }}</pre>
            </div>
          </ElCollapseItem>

          <ElCollapseItem title="Template Test" name="test">
            <div class="test-row">
              <ElInput v-model="templateTestTo" maxlength="255" placeholder="test@example.com" />
              <ElButton
                type="primary"
                :loading="templateTesting"
                :disabled="dialogDisabled"
                @click="handleTemplateTest"
              >
                Send Test
              </ElButton>
            </div>
            <div class="field-tip">Uses the current subject and body together with mock data.</div>
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

    <ElDialog
      v-model="previewVisible"
      title="Template Preview"
      width="760px"
      destroy-on-close
      align-center
    >
      <div class="preview-block">
        <div class="preview-title">Subject</div>
        <div class="preview-subject">{{ previewSubject || '(No subject)' }}</div>
      </div>

      <div class="preview-block">
        <div class="preview-title">Body</div>
        <div v-if="previewIsHtml" class="preview-body" v-html="previewBody" />
        <pre v-else class="preview-text">{{ previewBody || '(No content)' }}</pre>
      </div>

      <div class="preview-block">
        <div class="preview-title">Mock Data</div>
        <pre class="mock-data">{{ mockDataJson }}</pre>
      </div>
    </ElDialog>

    <ElDialog v-model="helpVisible" title="Editor Help" width="640px" destroy-on-close align-center>
      <div class="help-content">
        <p
          >Use the rich-text toolbar to format content or switch to source-like editing where
          needed.</p
        >
        <p
          >Insert variables such as <code v-pre>{{ .user.username }}</code> or
          <code v-pre>{{ .order.no }}</code
          >.</p
        >
        <p>Preview renders the current subject and body with mock data locally before sending.</p>
        <p
          >Template test calls the SMTP test endpoint with the current template body and mock
          payload.</p
        >
      </div>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import type { EmailTemplateRecord, SettingItemRecord, SmtpConfigRecord } from '@/api/admin'
  import ArtWangEditor from '@/components/core/forms/art-wang-editor/index.vue'
  import {
    createEmailTemplate,
    deleteEmailTemplate,
    fetchAdminSettings,
    fetchEmailTemplates,
    fetchSmtpConfig,
    hasAdminPermission,
    testSmtpConfig,
    updateAdminSettings,
    updateEmailTemplate,
    updateSmtpConfig
  } from '@/api/admin'
  import { useUserStore } from '@/store/modules/user'
  import { ElMessage, ElMessageBox } from 'element-plus'

  defineOptions({ name: 'SettingsEmailPage' })

  interface EmailSettingsForm {
    smtp_host: string
    smtp_port: string
    smtp_user: string
    smtp_pass: string
    smtp_from: string
    smtp_enabled: boolean
    email_enabled: boolean
    email_expire_enabled: boolean
    expire_reminder_days: number
  }

  interface EmailTemplateForm {
    id: number | null
    name: string
    subject: string
    body: string
    enabled: boolean
  }

  interface DefaultTemplateOption {
    key: string
    label: string
    subject: string
    body: string
  }

  const DEFAULT_TEMPLATES: DefaultTemplateOption[] = [
    {
      key: 'provision_success',
      label: 'Provision Success',
      subject: 'VPS Provisioned: Order {{.order.no}}',
      body: '<div><h2>VPS Provisioned</h2><p>Hello {{.user.username}}, your order <strong>{{.order.no}}</strong> is now active.</p><p>Log in to the console to manage <strong>{{.vps.name}}</strong>.</p></div>'
    },
    {
      key: 'expire_reminder',
      label: 'Expiration Reminder',
      subject: 'VPS Expiration Reminder: {{.vps.name}}',
      body: '<div><h2>Expiration Reminder</h2><p>Hello {{.user.username}}, your VPS <strong>{{.vps.name}}</strong> will expire on <strong>{{.vps.expire_at}}</strong>.</p><p>Please renew in time to avoid interruption.</p></div>'
    },
    {
      key: 'order_approved',
      label: 'Order Approved',
      subject: 'Order Approved: {{.order.no}}',
      body: '<div><h2>Order Approved</h2><p>Hello {{.user.username}}, your order <strong>{{.order.no}}</strong> has been approved.</p><p>{{.message}}</p></div>'
    },
    {
      key: 'order_rejected',
      label: 'Order Rejected',
      subject: 'Order Rejected: {{.order.no}}',
      body: '<div><h2>Order Rejected</h2><p>Hello {{.user.username}}, your order <strong>{{.order.no}}</strong> has been rejected.</p><p>Reason: {{.message}}</p></div>'
    }
  ]

  const loading = ref(false)
  const saving = ref(false)
  const templateLoading = ref(false)
  const templateSaving = ref(false)
  const smtpTesting = ref(false)
  const templateTesting = ref(false)
  const initialized = ref(false)

  const dialogVisible = ref(false)
  const previewVisible = ref(false)
  const helpVisible = ref(false)

  const defaultTemplateKey = ref<string | null>(null)
  const templateTestTo = ref('')
  const smtpTestTo = ref('')
  const previewSubject = ref('')
  const previewBody = ref('')
  const previewIsHtml = ref(false)

  const userStore = useUserStore()
  const { info } = storeToRefs(userStore)

  const form = reactive<EmailSettingsForm>(createDefaultForm())
  const templateForm = reactive<EmailTemplateForm>(createDefaultTemplateForm())
  const templates = ref<EmailTemplateRecord[]>([])

  const mockData = reactive({
    user: { id: 1001, username: 'demo_user', email: 'demo@example.com', qq: '123456' },
    order: { no: 'ORD-20240501-0001' },
    vps: { name: 'vps-001', expire_at: '2024-12-31' },
    message: 'This is a mock message.',
    now: ''
  })

  const canView = computed(() => hasAdminPermission(info.value?.buttons, ['settings.view']))
  const canUpdate = computed(() => hasAdminPermission(info.value?.buttons, ['settings.update']))
  const formDisabled = computed(() => !canUpdate.value || loading.value || saving.value)
  const dialogDisabled = computed(() => !canUpdate.value || templateSaving.value)
  const defaultTemplateOptions = computed(() =>
    DEFAULT_TEMPLATES.map((item) => ({ key: item.key, label: item.label }))
  )
  const mockDataJson = computed(() => JSON.stringify(mockData, null, 2))

  watch(
    canView,
    (value) => {
      if (value && !initialized.value) {
        loadInitialData()
      }
    },
    { immediate: true }
  )

  function createDefaultForm(): EmailSettingsForm {
    return {
      smtp_host: '',
      smtp_port: '',
      smtp_user: '',
      smtp_pass: '',
      smtp_from: '',
      smtp_enabled: false,
      email_enabled: false,
      email_expire_enabled: false,
      expire_reminder_days: 7
    }
  }

  function createDefaultTemplateForm(): EmailTemplateForm {
    return {
      id: null,
      name: '',
      subject: '',
      body: '',
      enabled: true
    }
  }

  function createSettingsMap(items: SettingItemRecord[]) {
    const map = new Map<string, unknown>()
    items.forEach((item) => {
      if (item.key) {
        map.set(String(item.key), item.value ?? '')
      }
    })
    return map
  }

  function parseBool(value: unknown, defaultValue = false) {
    if (value === undefined || value === null || value === '') {
      return defaultValue
    }

    return value === true || value === 'true' || value === '1' || value === 1
  }

  function refreshMockData() {
    mockData.now = new Date().toISOString()
  }

  function isHtmlContent(value: unknown) {
    return /<\/?[a-z][\s\S]*>/i.test(String(value || ''))
  }

  function resolvePath(source: Record<string, any>, path: string) {
    return path
      .split('.')
      .reduce((acc, key) => (acc && acc[key] !== undefined ? acc[key] : undefined), source)
  }

  function renderWithMock(input: unknown) {
    if (!input) {
      return ''
    }

    const text = String(input)
    return text.replace(/{{\s*\.([a-zA-Z0-9_.]+)\s*}}/g, (_match, key) => {
      const value = resolvePath(mockData as Record<string, any>, key)
      return value === undefined || value === null ? '' : String(value)
    })
  }

  function normalizeTemplate(row?: EmailTemplateRecord | null): EmailTemplateRecord {
    return {
      id: Number(row?.id || 0) || undefined,
      name: String(row?.name || ''),
      subject: String(row?.subject || ''),
      body: String(row?.body || ''),
      enabled: row?.enabled !== false,
      created_at: row?.created_at,
      updated_at: row?.updated_at
    }
  }

  function applySettings(settingsItems: SettingItemRecord[], smtpConfig?: SmtpConfigRecord) {
    const settingsMap = createSettingsMap(settingsItems)

    form.smtp_host = String(smtpConfig?.host ?? settingsMap.get('smtp_host') ?? '')
    form.smtp_port = String(smtpConfig?.port ?? settingsMap.get('smtp_port') ?? '')
    form.smtp_user = String(smtpConfig?.user ?? settingsMap.get('smtp_user') ?? '')
    form.smtp_pass = String(smtpConfig?.pass ?? settingsMap.get('smtp_pass') ?? '')
    form.smtp_from = String(smtpConfig?.from ?? settingsMap.get('smtp_from') ?? '')
    form.smtp_enabled =
      smtpConfig?.enabled !== undefined && smtpConfig?.enabled !== null
        ? Boolean(smtpConfig.enabled)
        : parseBool(settingsMap.get('smtp_enabled'), false)
    form.email_enabled = parseBool(settingsMap.get('email_enabled'), false)
    form.email_expire_enabled = parseBool(settingsMap.get('email_expire_enabled'), false)
    form.expire_reminder_days = Number(settingsMap.get('expire_reminder_days') || 7) || 7
  }

  async function loadSettings() {
    const [settingsPayload, smtpPayload] = await Promise.all([
      fetchAdminSettings(),
      fetchSmtpConfig()
    ])
    applySettings(settingsPayload.items || [], smtpPayload)
  }

  async function loadTemplates() {
    templateLoading.value = true

    try {
      const payload = await fetchEmailTemplates()
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
      await Promise.all([loadSettings(), loadTemplates()])
      initialized.value = true
    } finally {
      loading.value = false
    }
  }

  async function handleSaveSettings() {
    saving.value = true

    try {
      await updateSmtpConfig({
        host: form.smtp_host,
        port: form.smtp_port,
        user: form.smtp_user,
        pass: form.smtp_pass,
        from: form.smtp_from,
        enabled: form.smtp_enabled
      })

      await updateAdminSettings({
        items: [
          { key: 'email_enabled', value: String(form.email_enabled) },
          { key: 'email_expire_enabled', value: String(form.email_expire_enabled) },
          { key: 'expire_reminder_days', value: String(form.expire_reminder_days) }
        ]
      })

      ElMessage.success('Email settings saved')
    } finally {
      saving.value = false
    }
  }

  function openTemplateDialog(row?: EmailTemplateRecord) {
    Object.assign(templateForm, createDefaultTemplateForm())

    if (row) {
      Object.assign(templateForm, {
        id: Number(row.id || 0) || null,
        name: String(row.name || ''),
        subject: String(row.subject || ''),
        body: String(row.body || ''),
        enabled: row.enabled !== false
      })
    }

    defaultTemplateKey.value = null
    templateTestTo.value = ''
    dialogVisible.value = true
  }

  function applyDefaultTemplate() {
    const matched = DEFAULT_TEMPLATES.find((item) => item.key === defaultTemplateKey.value)
    if (!matched) {
      ElMessage.warning('Please choose a default template first')
      return
    }

    if (!templateForm.name.trim()) {
      templateForm.name = matched.key
    }
    templateForm.subject = matched.subject
    templateForm.body = matched.body
  }

  function openPreview() {
    refreshMockData()
    previewSubject.value = renderWithMock(templateForm.subject) || '(No subject)'
    previewBody.value = renderWithMock(templateForm.body) || '(No content)'
    previewIsHtml.value = isHtmlContent(previewBody.value)
    previewVisible.value = true
  }

  async function handleTemplateTest() {
    if (!templateTestTo.value.trim()) {
      ElMessage.error('Please enter a test recipient')
      return
    }

    if (!templateForm.body.trim()) {
      ElMessage.error('Template body cannot be empty')
      return
    }

    refreshMockData()
    templateTesting.value = true

    try {
      await testSmtpConfig({
        to: templateTestTo.value.trim(),
        subject: templateForm.subject,
        body: templateForm.body,
        variables: JSON.parse(JSON.stringify(mockData)),
        html: isHtmlContent(templateForm.body)
      })

      ElMessage.success('Test email sent')
    } finally {
      templateTesting.value = false
    }
  }

  async function handleSmtpTest() {
    if (!smtpTestTo.value.trim()) {
      ElMessage.error('Please enter a recipient email')
      return
    }

    refreshMockData()
    smtpTesting.value = true

    try {
      const enabledTemplate = templates.value.find((item) => item.enabled !== false)
      if (enabledTemplate?.name) {
        await testSmtpConfig({
          to: smtpTestTo.value.trim(),
          template_name: enabledTemplate.name,
          variables: JSON.parse(JSON.stringify(mockData))
        })
      } else {
        await testSmtpConfig({
          to: smtpTestTo.value.trim(),
          subject: 'SMTP Test',
          body: 'If you received this email, SMTP is configured correctly.',
          variables: JSON.parse(JSON.stringify(mockData)),
          html: false
        })
      }

      ElMessage.success('SMTP test sent')
    } finally {
      smtpTesting.value = false
    }
  }

  async function handleSaveTemplate() {
    if (!templateForm.name.trim()) {
      ElMessage.error('Please enter a template name')
      return
    }

    if (!templateForm.subject.trim()) {
      ElMessage.error('Please enter a subject')
      return
    }

    if (!templateForm.body.trim()) {
      ElMessage.error('Please enter template content')
      return
    }

    templateSaving.value = true

    try {
      const payload = {
        name: templateForm.name.trim(),
        subject: templateForm.subject.trim(),
        body: templateForm.body,
        enabled: templateForm.enabled
      }

      if (templateForm.id) {
        await updateEmailTemplate(templateForm.id, payload)
      } else {
        await createEmailTemplate(payload)
      }

      ElMessage.success('Template saved')
      dialogVisible.value = false
      await loadTemplates()
    } finally {
      templateSaving.value = false
    }
  }

  async function handleDeleteTemplate(row: EmailTemplateRecord) {
    if (!row.id) {
      return
    }

    await ElMessageBox.confirm('Delete this email template?', 'Delete Template', {
      type: 'warning'
    })

    await deleteEmailTemplate(row.id)
    ElMessage.success('Template deleted')
    await loadTemplates()
  }
</script>

<style scoped lang="scss">
  .email-settings-page {
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
  .field-tip,
  .help-content {
    margin-top: 6px;
    color: var(--el-text-color-secondary);
    font-size: 13px;
    line-height: 1.7;
  }

  .page-actions,
  .table-actions,
  .dialog-footer,
  .toolbar-left,
  .toolbar-right {
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

  .table-header,
  .template-toolbar {
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

  .editor-hint {
    margin-bottom: 10px;
    padding: 10px 12px;
    border: 1px solid var(--el-color-info-light-5);
    border-radius: 10px;
    background: var(--el-color-info-light-9);
    color: var(--el-color-info-dark-2);
    font-size: 13px;
    line-height: 1.7;
  }

  .template-vars {
    display: grid;
    gap: 6px;
  }

  .mock-block,
  .preview-block {
    margin-top: 12px;
  }

  .mock-title,
  .preview-title {
    margin-bottom: 8px;
    color: var(--el-text-color-primary);
    font-weight: 700;
  }

  .preview-subject,
  .preview-body,
  .preview-text,
  .mock-data {
    padding: 12px;
    border-radius: 12px;
    background: var(--el-fill-color-light);
    line-height: 1.7;
    white-space: pre-wrap;
  }

  @media (max-width: 768px) {
    .page-header,
    .table-header,
    .template-toolbar,
    .test-row {
      flex-direction: column;
      align-items: stretch;
    }

    .page-actions,
    .table-actions,
    .toolbar-left,
    .toolbar-right {
      width: 100%;
    }

    .page-actions :deep(.el-button),
    .table-actions :deep(.el-button),
    .toolbar-left :deep(.el-button),
    .toolbar-right :deep(.el-button),
    .test-row :deep(.el-button) {
      width: 100%;
    }
  }
</style>
