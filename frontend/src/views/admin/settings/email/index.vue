<template>
  <div class="email-settings-page pb-5">
    <ElCard v-loading="configLoading" class="art-card-xs config-card">
      <template #header>
        <div class="page-header">
          <div>
            <h2 class="page-title">邮件与模板</h2>
            <p class="page-subtitle">SMTP 配置、发送测试与邮件模板管理</p>
          </div>
          <ElSpace wrap>
            <ElButton :disabled="configLoading || configSaving" @click="loadConfig">
              <ArtSvgIcon icon="ri:refresh-line" />
              刷新配置
            </ElButton>
            <ElButton
              v-if="canUpdateSettings || canUpdateSmtp"
              type="primary"
              :loading="configSaving"
              @click="saveConfig"
            >
              <ArtSvgIcon icon="ri:save-line" />
              保存配置
            </ElButton>
          </ElSpace>
        </div>
      </template>

      <ArtForm
        v-model="configForm"
        :items="configItems"
        :disabled="configLoading || configSaving"
        :show-reset="false"
        :show-submit="false"
        :span="12"
        :gutter="20"
        label-position="top"
      >
        <template #smtp_test>
          <div class="inline-action">
            <ElInput v-model="smtpTestTo" placeholder="接收人邮箱" />
            <ElButton type="primary" :disabled="!canTestSmtp" @click="sendSmtpTest">
              <ArtSvgIcon icon="ri:send-plane-line" />
              测试发送
            </ElButton>
          </div>
          <div class="field-help">
            未启用模板时将发送默认文案；如有启用模板，将优先发送列表中的第一个启用模板。
          </div>
        </template>
      </ArtForm>
    </ElCard>

    <ElCard v-if="canViewTemplates" class="templates-card">
      <ArtTableHeader v-model:columns="columnChecks" :loading="loading" @refresh="refreshData">
        <template #left>
          <ElButton v-if="canUpdateTemplates" type="primary" @click="openTemplate()">
            <ArtSvgIcon icon="ri:add-line" />
            新增模板
          </ElButton>
        </template>
      </ArtTableHeader>

      <ArtTable row-key="id" :loading="loading" :data="data" :columns="columns">
        <template #enabled="{ row }">
          <ElTag :type="row.enabled ? 'success' : 'info'">
            {{ row.enabled ? '启用' : '停用' }}
          </ElTag>
        </template>
        <template #operation="{ row }">
          <ElSpace>
            <ElButton v-if="canUpdateTemplates" link type="primary" @click="openTemplate(row)">
              编辑
            </ElButton>
            <ElButton v-if="canDeleteTemplates" link type="danger" @click="removeTemplate(row)">
              删除
            </ElButton>
          </ElSpace>
        </template>
      </ArtTable>
    </ElCard>

    <ElDialog v-model="templateVisible" title="邮件模板" width="min(1000px, 92vw)" destroy-on-close>
      <div class="dialog-toolbar">
        <ElSpace wrap>
          <ElSelect v-model="defaultTemplateKey" clearable placeholder="选择默认模板">
            <ElOption
              v-for="item in defaultTemplates"
              :key="item.key"
              :label="item.label"
              :value="item.key"
            />
          </ElSelect>
          <ElButton @click="applyDefaultTemplate">填充默认</ElButton>
        </ElSpace>
        <ElSpace>
          <ElButton @click="openPreview">
            <ArtSvgIcon icon="ri:eye-line" />
            预览
          </ElButton>
          <ElTooltip content="模板变量说明">
            <ElButton circle aria-label="模板变量说明" @click="showHelp">
              <ArtSvgIcon icon="ri:question-line" />
            </ElButton>
          </ElTooltip>
        </ElSpace>
      </div>

      <ArtForm
        v-model="templateForm"
        :items="templateItems"
        :disabled="!canUpdateTemplates"
        :show-reset="false"
        :show-submit="false"
        :span="12"
        label-position="top"
      >
        <template #body>
          <EmailTemplateEditor v-model="templateForm.body" height="400px" />
        </template>
        <template #template_test>
          <div class="inline-action">
            <ElInput v-model="templateTestTo" placeholder="test@example.com" />
            <ElButton type="primary" :disabled="!canTestSmtp" @click="sendTemplateTest">
              发送测试
            </ElButton>
          </div>
        </template>
      </ArtForm>

      <ElCollapse class="template-details">
        <ElCollapseItem title="模板变量与 Mock 数据" name="variables">
          <div class="template-vars">
            <div v-for="item in templateVariables" :key="item.code">
              <code>{{ item.code }}</code
              >：{{ item.value }}
            </div>
          </div>
          <div class="field-help">渲染数据：</div>
          <pre class="mock-data">{{ mockDataJson }}</pre>
        </ElCollapseItem>
      </ElCollapse>

      <template #footer>
        <ElButton @click="templateVisible = false">取消</ElButton>
        <ElButton
          v-if="canUpdateTemplates"
          type="primary"
          :loading="templateSaving"
          @click="saveTemplate"
        >
          保存模板
        </ElButton>
      </template>
    </ElDialog>

    <ElDialog v-model="previewVisible" title="模板预览" width="min(760px, 92vw)">
      <ElDescriptions :column="1" border>
        <ElDescriptionsItem label="主题">{{ previewSubject }}</ElDescriptionsItem>
        <ElDescriptionsItem label="正文">
          <!-- 存储型 XSS 防护（P2-77 / m062 H1）：邮件模板 HTML 来自数据库/管理员
               输入，不能 v-html 进主文档。用无 sandbox 权限的 iframe srcdoc 渲染，
               脚本/事件处理器被惰性化，同源能力被移除。 -->
          <iframe
            v-if="previewIsHtml"
            class="preview-frame"
            sandbox=""
            :srcdoc="previewBody"
            title="邮件模板预览"
          />
          <pre v-else class="preview-text">{{ previewBody }}</pre>
        </ElDescriptionsItem>
        <ElDescriptionsItem label="Mock 数据">
          <pre class="mock-data">{{ mockDataJson }}</pre>
        </ElDescriptionsItem>
      </ElDescriptions>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import {
    deleteEmailTemplate,
    getSmtpConfig,
    listEmailTemplates,
    testSmtpConfig,
    updateEmailTemplate,
    updateSmtpConfig,
    upsertEmailTemplate
  } from '@/services/admin'
  import { useTable } from '@/hooks/core/useTable'
  import {
    booleanSetting,
    fetchSettingMap,
    saveSettingItems,
    settingBoolean,
    settingInteger,
    settingString,
    stringSetting,
    useAdminPermissions
  } from '../_shared/settings'

  defineOptions({ name: 'AdminSettingsEmail' })

  // web_bundle_mobile F4：wangeditor 831kB/280kB gzip 只在真正打开模板编辑器
  // 时按需拉取，不让"进入邮件设置页"就整块加载（EmailTemplateEditor 只在
  // 模板编辑弹窗里渲染）。
  const EmailTemplateEditor = defineAsyncComponent(
    () => import('@/components/business/email-template-editor/index.vue')
  )

  interface EmailTemplateRow {
    id: number | null
    name: string
    subject: string
    body: string
    enabled: boolean
  }

  interface TemplateForm extends Record<string, unknown> {
    id: number | null
    name: string
    subject: string
    body: string
    enabled: boolean
  }

  const configForm = reactive({
    smtp_host: '',
    smtp_port: '',
    smtp_user: '',
    smtp_pass: '',
    smtp_from: '',
    smtp_enabled: false,
    email_enabled: false,
    email_expire_enabled: false,
    expire_reminder_days: 7
  })
  const configLoading = ref(false)
  const configSaving = ref(false)
  const smtpTestTo = ref('')
  const templateVisible = ref(false)
  const templateSaving = ref(false)
  const templateForm = reactive<TemplateForm>(createTemplateForm())
  const defaultTemplateKey = ref<string>()
  const templateTestTo = ref('')
  const previewVisible = ref(false)
  const previewSubject = ref('')
  const previewBody = ref('')
  const previewIsHtml = ref(false)
  const { hasPermission } = useAdminPermissions()
  const canViewSettings = hasPermission('settings.view')
  const canUpdateSettings = hasPermission('settings.update')
  const canViewSmtp = hasPermission('smtp.view')
  const canUpdateSmtp = hasPermission('smtp.update')
  const canTestSmtp = hasPermission('smtp.test')
  const canViewTemplates = hasPermission('email_template.list')
  const canUpdateTemplates = hasPermission('email_template.update')
  const canDeleteTemplates = hasPermission('email_template.delete')

  const mockData = {
    user: { id: 1001, username: 'demo_user', email: 'demo@example.com', qq: '123456' },
    order: { no: 'ORD-20240501-0001', amount: '299.00' },
    vps: { name: 'vps-001', ip: '192.0.2.10', expire_at: '2024-12-31' },
    message: 'This is a mock message.',
    now: ''
  }
  const mockDataJson = computed(() => JSON.stringify(mockData, null, 2))
  const templateVariables = computed(() => [
    { code: '{{ .user.id }}', value: mockData.user.id },
    { code: '{{ .user.username }}', value: mockData.user.username },
    { code: '{{ .user.email }}', value: mockData.user.email },
    { code: '{{ .user.qq }}', value: mockData.user.qq },
    { code: '{{ .order.no }}', value: mockData.order.no },
    { code: '{{ .order.amount }}', value: mockData.order.amount },
    { code: '{{ .vps.name }}', value: mockData.vps.name },
    { code: '{{ .vps.ip }}', value: mockData.vps.ip },
    { code: '{{ .vps.expire_at }}', value: mockData.vps.expire_at },
    { code: '{{ .message }}', value: mockData.message },
    { code: '{{ .now }}', value: mockData.now }
  ])

  const defaultTemplates = [
    {
      key: 'provision_success',
      label: '开通成功 (provision_success)',
      subject: 'VPS Provisioned: Order {{.order.no}}',
      body: `<!DOCTYPE html>
<html>
<body style="margin:0; padding:24px; background:#f4f6fb; font-family: Arial, sans-serif; color:#1f2329;">
  <div style="max-width:640px; margin:0 auto;">
    <div style="font-size:12px; color:#6b7280;">Provision Notice</div>
    <div style="font-size:20px; font-weight:700;">VPS Provisioned</div>
    <div style="height:12px;"></div>
    <div style="background:#ffffff; border-radius:12px; box-shadow:0 8px 20px rgba(15,23,42,0.08); padding:24px;">
      <div style="display:inline-block; padding:6px 10px; background:#eef2ff; color:#4338ca; border-radius:999px; font-size:12px; font-weight:600;">Active</div>
      <h2 style="margin:12px 0 8px; font-size:18px;">Hi {{.user.username}},</h2>
      <p style="margin:0 0 12px;">Your VPS for order <strong>{{.order.no}}</strong> is now active.</p>
      <div style="background:#f8fafc; border-radius:10px; padding:12px;">
        <div style="font-size:12px; color:#6b7280;">Next step</div>
        <div style="font-size:14px; font-weight:600; padding-top:4px;">Log in to the console to manage your instance.</div>
      </div>
      <p style="margin:16px 0 0; font-size:13px; color:#6b7280;">If you have any questions, reply to this email.</p>
    </div>
    <div style="padding-top:12px; font-size:12px; color:#94a3b8;">This is an automated message.</div>
  </div>
</body>
</html>`
    },
    {
      key: 'expire_reminder',
      label: '到期提醒 (expire_reminder)',
      subject: 'VPS Expiration Reminder: {{.vps.name}}',
      body: `<!DOCTYPE html>
<html>
<body style="margin:0; padding:24px; background:#fff7ed; font-family: Arial, sans-serif; color:#1f2329;">
  <div style="max-width:640px; margin:0 auto;">
    <div style="font-size:12px; color:#9a3412;">Reminder</div>
    <div style="font-size:20px; font-weight:700;">VPS Expiration Alert</div>
    <div style="height:12px;"></div>
    <div style="background:#ffffff; border-radius:12px; box-shadow:0 8px 20px rgba(180,83,9,0.08); padding:24px;">
      <div style="display:inline-block; padding:6px 10px; background:#ffedd5; color:#9a3412; border-radius:999px; font-size:12px; font-weight:600;">Action Required</div>
      <h2 style="margin:12px 0 8px; font-size:18px;">Hi {{.user.username}},</h2>
      <p style="margin:0 0 12px;">Your VPS <strong>{{.vps.name}}</strong> will expire on <strong>{{.vps.expire_at}}</strong>.</p>
      <div style="background:#fff7ed; border-radius:10px; padding:12px;">
        <div style="font-size:12px; color:#9a3412;">Recommendation</div>
        <div style="font-size:14px; font-weight:600; padding-top:4px;">Renew early to avoid service interruption.</div>
      </div>
      <p style="margin:16px 0 0; font-size:13px; color:#9a3412;">If you have questions, contact support.</p>
    </div>
    <div style="padding-top:12px; font-size:12px; color:#c2410c;">This is an automated message.</div>
  </div>
</body>
</html>`
    },
    {
      key: 'order_approved',
      label: '订单通过 (order_approved)',
      subject: 'Order Approved: {{.order.no}}',
      body: `<!DOCTYPE html>
<html>
<body style="margin:0; padding:24px; background:#ecfeff; font-family: Arial, sans-serif; color:#1f2329;">
  <div style="max-width:640px; margin:0 auto;">
    <div style="font-size:12px; color:#0e7490;">Order Update</div>
    <div style="font-size:20px; font-weight:700;">Order Approved</div>
    <div style="height:12px;"></div>
    <div style="background:#ffffff; border-radius:12px; box-shadow:0 8px 20px rgba(14,116,144,0.08); padding:24px;">
      <div style="display:inline-block; padding:6px 10px; background:#cffafe; color:#0e7490; border-radius:999px; font-size:12px; font-weight:600;">Approved</div>
      <h2 style="margin:12px 0 8px; font-size:18px;">Hi {{.user.username}},</h2>
      <p style="margin:0 0 12px;">Your order <strong>{{.order.no}}</strong> has been approved.</p>
      <div style="background:#f0fdfa; border-radius:10px; padding:12px; font-size:14px;">
        {{.message}}
      </div>
      <p style="margin:16px 0 0; font-size:13px; color:#0e7490;">You will receive another email when provisioning is complete.</p>
    </div>
    <div style="padding-top:12px; font-size:12px; color:#0891b2;">This is an automated message.</div>
  </div>
</body>
</html>`
    },
    {
      key: 'order_rejected',
      label: '订单驳回 (order_rejected)',
      subject: 'Order Rejected: {{.order.no}}',
      body: `<!DOCTYPE html>
<html>
<body style="margin:0; padding:24px; background:#fef2f2; font-family: Arial, sans-serif; color:#1f2329;">
  <div style="max-width:640px; margin:0 auto;">
    <div style="font-size:12px; color:#b91c1c;">Order Update</div>
    <div style="font-size:20px; font-weight:700;">Order Rejected</div>
    <div style="height:12px;"></div>
    <div style="background:#ffffff; border-radius:12px; box-shadow:0 8px 20px rgba(185,28,28,0.08); padding:24px;">
      <div style="display:inline-block; padding:6px 10px; background:#fee2e2; color:#b91c1c; border-radius:999px; font-size:12px; font-weight:600;">Rejected</div>
      <h2 style="margin:12px 0 8px; font-size:18px;">Hi {{.user.username}},</h2>
      <p style="margin:0 0 12px;">Your order <strong>{{.order.no}}</strong> has been rejected.</p>
      <div style="background:#fef2f2; border-radius:10px; padding:12px; font-size:14px;">
        Reason: {{.message}}
      </div>
      <p style="margin:16px 0 0; font-size:13px; color:#b91c1c;">You can reply to this email if you need help.</p>
    </div>
    <div style="padding-top:12px; font-size:12px; color:#ef4444;">This is an automated message.</div>
  </div>
</body>
</html>`
    }
  ]

  const configItems = computed(() => [
    {
      key: 'smtp_host',
      label: 'SMTP Host',
      type: 'input',
      props: { disabled: !canUpdateSmtp.value }
    },
    {
      key: 'smtp_port',
      label: 'SMTP Port',
      type: 'input',
      props: { disabled: !canUpdateSmtp.value }
    },
    {
      key: 'smtp_user',
      label: 'SMTP User',
      type: 'input',
      props: { disabled: !canUpdateSmtp.value }
    },
    {
      key: 'smtp_pass',
      label: 'SMTP Password',
      type: 'input',
      props: { type: 'password', showPassword: true, disabled: !canUpdateSmtp.value }
    },
    {
      key: 'smtp_from',
      label: 'SMTP From',
      type: 'input',
      span: 24,
      props: { disabled: !canUpdateSmtp.value }
    },
    {
      key: 'smtp_enabled',
      label: '启用 SMTP',
      type: 'switch',
      props: { disabled: !canUpdateSmtp.value }
    },
    {
      key: 'email_enabled',
      label: '启用邮件',
      type: 'switch',
      props: { disabled: !canUpdateSettings.value }
    },
    {
      key: 'email_expire_enabled',
      label: '发送到期提醒邮件',
      type: 'switch',
      props: { disabled: !canUpdateSettings.value }
    },
    {
      key: 'expire_reminder_days',
      label: '到期提醒天数',
      type: 'number',
      props: { min: 1, class: 'full-width', disabled: !canUpdateSettings.value }
    },
    { key: 'smtp_test', label: 'SMTP 测试', span: 24 }
  ])
  const templateItems = computed(() => [
    { key: 'name', label: '名称', type: 'input', span: 16 },
    { key: 'enabled', label: '启用', type: 'switch', span: 8 },
    { key: 'subject', label: '主题', type: 'input', span: 24 },
    { key: 'body', label: '内容', span: 24 },
    { key: 'template_test', label: '模板测试', span: 24 }
  ])

  const fetchTemplates = async () => {
    if (!canViewTemplates.value) return { records: [], total: 0 }
    const response = await listEmailTemplates()
    const rows = (response.data?.items ?? []).map(normalizeTemplate)
    return { records: rows, total: rows.length }
  }

  const {
    columns,
    columnChecks,
    data,
    loading,
    refreshData,
    refreshCreate,
    refreshUpdate,
    refreshRemove
  } = useTable({
    core: {
      apiFn: fetchTemplates,
      columnsFactory: () => [
        { prop: 'id', label: 'ID', width: 80 },
        { prop: 'name', label: '名称', minWidth: 180, showOverflowTooltip: true },
        { prop: 'subject', label: '主题', minWidth: 260, showOverflowTooltip: true },
        { prop: 'enabled', label: '状态', width: 100, useSlot: true },
        { prop: 'operation', label: '操作', width: 140, fixed: 'right', useSlot: true }
      ]
    }
  })

  onMounted(loadConfig)

  function createTemplateForm(): TemplateForm {
    return { id: null, name: '', subject: '', body: '', enabled: true }
  }

  function normalizeTemplate(record: Record<string, unknown>): EmailTemplateRow {
    const id = record.id ?? record.ID
    return {
      id: id == null ? null : Number(id),
      name: String(record.name ?? record.Name ?? ''),
      subject: String(record.subject ?? record.Subject ?? ''),
      body: String(record.body ?? record.Body ?? record.content ?? record.Content ?? ''),
      enabled: Boolean(record.enabled ?? record.Enabled ?? true)
    }
  }

  async function loadConfig(): Promise<void> {
    configLoading.value = true
    try {
      const map = canViewSettings.value ? await fetchSettingMap() : new Map<string, unknown>()
      const smtp = canViewSmtp.value ? (await getSmtpConfig()).data : undefined
      configForm.smtp_host = String(smtp?.host ?? settingString(map, 'smtp_host'))
      configForm.smtp_port = String(smtp?.port ?? settingString(map, 'smtp_port'))
      configForm.smtp_user = String(smtp?.user ?? settingString(map, 'smtp_user'))
      configForm.smtp_pass = String(smtp?.pass ?? settingString(map, 'smtp_pass'))
      configForm.smtp_from = String(smtp?.from ?? settingString(map, 'smtp_from'))
      configForm.smtp_enabled = smtp?.enabled ?? settingBoolean(map, 'smtp_enabled')
      configForm.email_enabled = settingBoolean(map, 'email_enabled')
      configForm.email_expire_enabled = settingBoolean(map, 'email_expire_enabled')
      configForm.expire_reminder_days = settingInteger(map, 'expire_reminder_days', 7)
    } finally {
      configLoading.value = false
    }
  }

  async function saveConfig(): Promise<void> {
    if (!canUpdateSettings.value && !canUpdateSmtp.value) return
    // fail-closed：只有 smtp.update 而没有 smtp.view 时，前端拿不到现有密码；
    // 后端 AdminSMTPConfigUpdate 会无条件把 smtp_pass 写成本次提交值。如果这里
    // 留空提交，会把现有 SMTP 密码静默清空（verify-r2 R-6 / m062 M1）。因此要求
    // 这类管理员保存 SMTP 时必须显式填写新密码，否则阻止保存。
    if (canUpdateSmtp.value && !canViewSmtp.value && !configForm.smtp_pass) {
      ElMessage.error(
        '你没有 smtp.view 权限，保存 SMTP 配置前必须填写新的 SMTP 密码（否则现有密码会被清空）'
      )
      return
    }
    configSaving.value = true
    try {
      if (canUpdateSmtp.value) {
        await updateSmtpConfig({
          host: configForm.smtp_host,
          port: configForm.smtp_port,
          user: configForm.smtp_user,
          pass: configForm.smtp_pass,
          from: configForm.smtp_from,
          enabled: configForm.smtp_enabled
        })
      }
      if (canUpdateSettings.value) {
        await saveSettingItems([
          booleanSetting('email_enabled', configForm.email_enabled),
          booleanSetting('email_expire_enabled', configForm.email_expire_enabled),
          stringSetting('expire_reminder_days', configForm.expire_reminder_days)
        ])
      }
      ElMessage.success('邮件配置已保存')
    } finally {
      configSaving.value = false
    }
  }

  function variables(): Record<string, unknown> {
    mockData.now = new Date().toISOString()
    return { ...mockData }
  }

  function resolvePath(root: Record<string, unknown>, path: string): unknown {
    return path.split('.').reduce<unknown>((value, key) => {
      if (!value || typeof value !== 'object') return undefined
      return (value as Record<string, unknown>)[key]
    }, root)
  }

  function renderMock(value: string): string {
    const data = variables()
    return value.replace(/{{\s*\.([a-zA-Z0-9_.]+)\s*}}/g, (_match, path: string) =>
      String(resolvePath(data, path) ?? '')
    )
  }

  function isHtml(value: string): boolean {
    return /<\/?[a-z][\s\S]*>/i.test(value)
  }

  async function sendSmtpTest(): Promise<void> {
    if (!canTestSmtp.value) return
    if (!smtpTestTo.value.trim()) {
      ElMessage.error('请输入接收人邮箱')
      return
    }
    let enabledTemplate: EmailTemplateRow | undefined
    if (canViewTemplates.value) {
      const templateResponse = await listEmailTemplates()
      enabledTemplate = (templateResponse.data?.items ?? [])
        .map(normalizeTemplate)
        .find((item) => item.enabled)
    }
    await testSmtpConfig(
      enabledTemplate
        ? {
            to: smtpTestTo.value.trim(),
            template_name: enabledTemplate.name,
            variables: variables()
          }
        : {
            to: smtpTestTo.value.trim(),
            subject: 'SMTP Test',
            body: '如果您收到此邮件，则说明邮箱成功配置。',
            variables: variables(),
            html: false
          }
    )
    ElMessage.success('测试邮件已发送')
  }

  function openTemplate(row?: EmailTemplateRow): void {
    if (!canUpdateTemplates.value) return
    Object.assign(templateForm, row ?? createTemplateForm())
    defaultTemplateKey.value = undefined
    templateTestTo.value = ''
    templateVisible.value = true
  }

  function applyDefaultTemplate(): void {
    const template = defaultTemplates.find((item) => item.key === defaultTemplateKey.value)
    if (!template) {
      ElMessage.warning('请选择默认模板')
      return
    }
    templateForm.name ||= template.key
    templateForm.subject = template.subject
    templateForm.body = template.body
  }

  function openPreview(): void {
    previewSubject.value = renderMock(templateForm.subject) || '(无主题)'
    previewBody.value = renderMock(templateForm.body) || '(无内容)'
    previewIsHtml.value = isHtml(previewBody.value)
    previewVisible.value = true
  }

  function showHelp(): void {
    ElMessageBox.alert(
      '编辑模式切换：点击工具栏右上角的 </> 按钮可在可视化模式和 HTML 源码模式之间切换。\n\n插入模板变量：点击工具栏的“插入变量”按钮，从下拉菜单中选择要插入的变量（如 {{.user.username}}）。\n\n模板变量保护：插入的变量会被自动保护，整体可删除但不可修改内容。变量会以渐变色背景显示。\n\n快捷键：Ctrl+B 加粗；Ctrl+I 斜体；Ctrl+U 下划线；Ctrl+Z 撤销；Ctrl+Y 重做；Ctrl+Shift+S 切换编辑模式；Esc 退出全屏。\n\n右键菜单：在编辑器中右键可快速访问撤销、重做、剪切、复制、粘贴和清除格式等功能。\n\n发送预览和测试邮件时会使用模拟数据替换这些变量。',
      '编辑器使用帮助',
      { confirmButtonText: '知道了' }
    )
  }

  async function sendTemplateTest(): Promise<void> {
    if (!canTestSmtp.value) return
    if (!templateTestTo.value.trim()) {
      ElMessage.error('请输入测试收件人')
      return
    }
    if (!templateForm.body) {
      ElMessage.error('模板内容不能为空')
      return
    }
    await testSmtpConfig({
      to: templateTestTo.value.trim(),
      subject: templateForm.subject,
      body: templateForm.body,
      variables: variables(),
      html: isHtml(templateForm.body)
    })
    ElMessage.success('测试邮件已发送')
  }

  async function saveTemplate(): Promise<void> {
    if (!canUpdateTemplates.value) return
    templateSaving.value = true
    try {
      const payload = {
        name: templateForm.name.trim(),
        subject: templateForm.subject,
        body: templateForm.body,
        enabled: templateForm.enabled
      }
      if (templateForm.id !== null) await updateEmailTemplate(templateForm.id, payload)
      else await upsertEmailTemplate(payload)
      templateVisible.value = false
      ElMessage.success('邮件模板已保存')
      if (templateForm.id !== null) await refreshUpdate()
      else await refreshCreate()
    } finally {
      templateSaving.value = false
    }
  }

  async function removeTemplate(row: EmailTemplateRow): Promise<void> {
    if (!canDeleteTemplates.value) return
    if (row.id === null) return
    await ElMessageBox.confirm('确认删除该邮件模板吗？', '删除模板', { type: 'warning' })
    await deleteEmailTemplate(row.id)
    ElMessage.success('邮件模板已删除')
    await refreshRemove()
  }
</script>

<style lang="scss" scoped>
  .config-card {
    margin-bottom: 16px;
  }

  .page-header,
  .dialog-toolbar,
  .inline-action {
    display: flex;
    gap: 12px;
    align-items: center;
  }

  .page-header,
  .dialog-toolbar {
    justify-content: space-between;
  }

  .page-title {
    margin: 0;
    font-size: 18px;
    font-weight: 600;
    color: var(--art-gray-900);
  }

  .page-subtitle {
    margin: 4px 0 0;
    font-size: 13px;
    color: var(--art-gray-600);
  }

  .templates-card {
    min-height: 480px;
    margin-top: 12px;
  }

  .dialog-toolbar {
    margin-bottom: 12px;
  }

  .inline-action :is(.el-input) {
    flex: 1;
  }

  .field-help {
    margin-top: 6px;
    font-size: 12px;
    color: var(--art-gray-600);
  }

  .template-details {
    margin-top: 12px;
  }

  .template-vars {
    display: grid;
    gap: 4px;
    margin-bottom: 12px;
    color: var(--art-gray-800);
  }

  .mock-data {
    max-height: 320px;
    padding: 12px;
    margin: 6px 0 0;
    overflow: auto;
    color: var(--art-gray-800);
    white-space: pre-wrap;
    background: var(--art-hover-color);
    border: 1px solid var(--art-card-border);
    border-radius: calc(var(--custom-radius) / 2 + 2px);
  }

  .preview-body,
  .preview-text {
    max-height: 480px;
    margin: 0;
    overflow: auto;
    color: var(--art-gray-800);
    white-space: pre-wrap;
  }

  .full-width {
    width: 100%;
  }

  @media (width <= 640px) {
    .page-header,
    .dialog-toolbar,
    .inline-action {
      flex-direction: column;
      align-items: stretch;
    }
  }
</style>
