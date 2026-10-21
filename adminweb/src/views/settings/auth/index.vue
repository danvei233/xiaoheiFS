<template>
  <div v-loading="loading" class="auth-settings-page art-full-height">
    <div class="page-header">
      <div>
        <div class="page-kicker">Account Security</div>
        <div class="page-title">认证设置</div>
        <div class="page-subtitle">
          统一管理注册入口、验证码策略、找回密码、绑定校验和登录保护。
        </div>
      </div>

      <div class="page-actions">
        <ElButton v-if="canView" :disabled="loading || saving" @click="fetchData">刷新</ElButton>
        <ElButton v-if="canUpdate" type="primary" :loading="saving" @click="handleSave">
          保存设置
        </ElButton>
      </div>
    </div>

    <ElEmpty v-if="!canView" description="当前账号没有查看认证设置的权限" />

    <template v-else>
      <ElRow :gutter="16">
        <ElCol :xs="24" :xl="14">
          <ElCard shadow="never" class="section-card">
            <template #header>
              <div class="section-title">注册入口</div>
            </template>

            <ElForm label-position="top" :disabled="formDisabled">
              <ElFormItem label="启用前台注册">
                <div class="switch-row">
                  <ElSwitch v-model="form.register_enabled" />
                  <span class="field-tip">关闭后，前台注册入口会提示暂时停止开放注册。</span>
                </div>
              </ElFormItem>

              <ElFormItem label="必填字段">
                <ElCheckboxGroup v-model="form.register_required_fields">
                  <ElCheckbox
                    v-for="item in requiredFieldOptions"
                    :key="item.value"
                    :label="item.value"
                    :disabled="item.disabled || formDisabled"
                  >
                    {{ item.label }}
                  </ElCheckbox>
                </ElCheckboxGroup>
                <div class="field-tip">
                  用户名和密码始终为系统必填项；邮箱是否必填请使用单独开关控制。
                </div>
              </ElFormItem>

              <ElFormItem label="邮箱必填">
                <div class="switch-row">
                  <ElSwitch v-model="form.register_email_required" />
                  <span class="field-tip">启用后，前台注册时必须填写邮箱地址。</span>
                </div>
              </ElFormItem>
            </ElForm>
          </ElCard>

          <ElCard shadow="never" class="section-card">
            <template #header>
              <div class="section-title">密码规则</div>
            </template>

            <ElForm label-position="top" :disabled="formDisabled">
              <ElRow :gutter="16">
                <ElCol :xs="24" :md="12">
                  <ElFormItem label="最小长度">
                    <ElInputNumber
                      v-model="form.password_min_len"
                      :min="6"
                      :max="64"
                      :step="1"
                      class="full-width"
                    />
                  </ElFormItem>
                </ElCol>

                <ElCol :xs="24" :md="12">
                  <ElFormItem label="必须包含">
                    <div class="switch-grid">
                      <label class="switch-item">
                        <span>大写字母</span>
                        <ElSwitch v-model="form.password_require_upper" />
                      </label>
                      <label class="switch-item">
                        <span>小写字母</span>
                        <ElSwitch v-model="form.password_require_lower" />
                      </label>
                      <label class="switch-item">
                        <span>数字</span>
                        <ElSwitch v-model="form.password_require_number" />
                      </label>
                      <label class="switch-item">
                        <span>符号</span>
                        <ElSwitch v-model="form.password_require_symbol" />
                      </label>
                    </div>
                  </ElFormItem>
                </ElCol>
              </ElRow>
            </ElForm>
          </ElCard>
        </ElCol>

        <ElCol :xs="24" :xl="10">
          <ElCard shadow="never" class="section-card section-card--accent">
            <template #header>
              <div class="section-title">注册验证</div>
            </template>

            <ElForm label-position="top" :disabled="formDisabled">
              <ElFormItem label="验证渠道">
                <ElCheckboxGroup v-model="form.register_verify_channels">
                  <ElCheckbox
                    v-for="item in verifyChannelOptions"
                    :key="item.value"
                    :label="item.value"
                  >
                    {{ item.label }}
                  </ElCheckbox>
                </ElCheckboxGroup>
              </ElFormItem>

              <ElFormItem label="验证码有效期（秒）">
                <ElInputNumber
                  v-model="form.register_verify_ttl_sec"
                  :min="60"
                  :max="3600"
                  :step="30"
                  class="full-width"
                />
              </ElFormItem>

              <ElDivider>验证码策略</ElDivider>

              <ElRow :gutter="12">
                <ElCol :xs="24" :md="12">
                  <ElFormItem label="短信长度">
                    <ElInputNumber
                      v-model="form.auth_sms_code_len"
                      :min="4"
                      :max="12"
                      :step="1"
                      class="full-width"
                    />
                  </ElFormItem>
                </ElCol>

                <ElCol :xs="24" :md="12">
                  <ElFormItem label="短信复杂度">
                    <ElSelect v-model="form.auth_sms_code_complexity" class="full-width">
                      <ElOption
                        v-for="item in codeComplexityOptions"
                        :key="item.value"
                        :label="item.label"
                        :value="item.value"
                      />
                    </ElSelect>
                  </ElFormItem>
                </ElCol>
              </ElRow>

              <ElRow :gutter="12">
                <ElCol :xs="24" :md="12">
                  <ElFormItem label="邮件长度">
                    <ElInputNumber
                      v-model="form.auth_email_code_len"
                      :min="4"
                      :max="12"
                      :step="1"
                      class="full-width"
                    />
                  </ElFormItem>
                </ElCol>

                <ElCol :xs="24" :md="12">
                  <ElFormItem label="邮件复杂度">
                    <ElSelect v-model="form.auth_email_code_complexity" class="full-width">
                      <ElOption
                        v-for="item in codeComplexityOptions"
                        :key="item.value"
                        :label="item.label"
                        :value="item.value"
                      />
                    </ElSelect>
                  </ElFormItem>
                </ElCol>
              </ElRow>

              <ElAlert
                type="info"
                :closable="false"
                show-icon
                title="模板维护入口"
                description="邮件模板请在系统设置的邮件相关页面配置，短信模板请在短信设置页面维护。"
              />
            </ElForm>
          </ElCard>

          <ElCard shadow="never" class="section-card">
            <template #header>
              <div class="section-title">登录与账号保护</div>
            </template>

            <ElForm label-position="top" :disabled="formDisabled">
              <ElFormItem label="登录提醒">
                <div class="stack-switches">
                  <label class="switch-item switch-item--inline">
                    <span>启用登录提醒</span>
                    <ElSwitch v-model="form.auth_login_notify_enabled" />
                  </label>
                </div>
              </ElFormItem>

              <ElFormItem label="提醒触发条件">
                <ElCheckboxGroup v-model="form.auth_login_notify_events">
                  <ElCheckbox
                    v-for="item in loginNotifyEventOptions"
                    :key="item.value"
                    :label="item.value"
                  >
                    {{ item.label }}
                  </ElCheckbox>
                </ElCheckboxGroup>
              </ElFormItem>

              <ElFormItem label="提醒渠道">
                <ElCheckboxGroup v-model="form.auth_login_notify_channels">
                  <ElCheckbox
                    v-for="item in verifyChannelOptions"
                    :key="item.value"
                    :label="item.value"
                  >
                    {{ item.label }}
                  </ElCheckbox>
                </ElCheckboxGroup>
              </ElFormItem>

              <ElDivider />

              <ElFormItem label="找回密码">
                <div class="stack-switches">
                  <label class="switch-item switch-item--inline">
                    <span>启用找回密码</span>
                    <ElSwitch v-model="form.auth_password_reset_enabled" />
                  </label>
                </div>
              </ElFormItem>

              <ElFormItem label="找回渠道">
                <ElCheckboxGroup v-model="form.auth_password_reset_channels">
                  <ElCheckbox
                    v-for="item in verifyChannelOptions"
                    :key="item.value"
                    :label="item.value"
                  >
                    {{ item.label }}
                  </ElCheckbox>
                </ElCheckboxGroup>
              </ElFormItem>

              <ElFormItem label="找回验证码有效期（秒）">
                <ElInputNumber
                  v-model="form.auth_password_reset_verify_ttl_sec"
                  :min="60"
                  :max="3600"
                  :step="30"
                  class="full-width"
                />
              </ElFormItem>

              <ElDivider />

              <ElFormItem label="联系方式绑定">
                <div class="stack-switches">
                  <label class="switch-item switch-item--inline">
                    <span>启用邮箱绑定</span>
                    <ElSwitch v-model="form.auth_email_bind_enabled" />
                  </label>
                  <label class="switch-item switch-item--inline">
                    <span>启用手机绑定</span>
                    <ElSwitch v-model="form.auth_phone_bind_enabled" />
                  </label>
                  <label class="switch-item switch-item--inline">
                    <span>换绑后通知旧联系方式</span>
                    <ElSwitch v-model="form.auth_contact_change_notify_old_enabled" />
                  </label>
                </div>
              </ElFormItem>

              <ElFormItem label="绑定验证码有效期（秒）">
                <ElInputNumber
                  v-model="form.auth_contact_bind_verify_ttl_sec"
                  :min="60"
                  :max="3600"
                  :step="30"
                  class="full-width"
                />
              </ElFormItem>

              <ElFormItem label="未开启 2FA 时的密码要求">
                <div class="stack-switches">
                  <label class="switch-item switch-item--inline">
                    <span>首次绑定需要密码</span>
                    <ElSwitch v-model="form.auth_bind_require_password_when_no_2fa" />
                  </label>
                  <label class="switch-item switch-item--inline">
                    <span>换绑需要密码</span>
                    <ElSwitch v-model="form.auth_rebind_require_password_when_no_2fa" />
                  </label>
                </div>
              </ElFormItem>

              <ElFormItem label="两步验证（2FA）">
                <div class="stack-switches">
                  <label class="switch-item switch-item--inline">
                    <span>启用 2FA 总开关</span>
                    <ElSwitch v-model="form.auth_2fa_enabled" />
                  </label>
                  <label class="switch-item switch-item--inline">
                    <span>启用 2FA 绑定流程</span>
                    <ElSwitch v-model="form.auth_2fa_bind_enabled" />
                  </label>
                  <label class="switch-item switch-item--inline">
                    <span>启用 2FA 换绑流程</span>
                    <ElSwitch v-model="form.auth_2fa_rebind_enabled" />
                  </label>
                </div>
              </ElFormItem>

              <ElDivider />

              <ElFormItem label="登录频率限制">
                <div class="stack-switches">
                  <label class="switch-item switch-item--inline">
                    <span>启用登录频率限制</span>
                    <ElSwitch v-model="form.login_rate_limit_enabled" />
                  </label>
                </div>
              </ElFormItem>

              <ElRow :gutter="16">
                <ElCol :xs="24" :md="12">
                  <ElFormItem label="窗口秒数">
                    <ElInputNumber
                      v-model="form.login_rate_limit_window_sec"
                      :min="60"
                      :max="3600"
                      :step="30"
                      class="full-width"
                    />
                  </ElFormItem>
                </ElCol>

                <ElCol :xs="24" :md="12">
                  <ElFormItem label="最大次数">
                    <ElInputNumber
                      v-model="form.login_rate_limit_max_attempts"
                      :min="3"
                      :max="30"
                      :step="1"
                      class="full-width"
                    />
                  </ElFormItem>
                </ElCol>
              </ElRow>
            </ElForm>
          </ElCard>
        </ElCol>
      </ElRow>
    </template>
  </div>
</template>

<script setup lang="ts">
  import type { SettingItemRecord } from '@/api/admin'
  import { fetchAdminSettings, hasAdminPermission, updateAdminSettings } from '@/api/admin'
  import { useUserStore } from '@/store/modules/user'

  defineOptions({ name: 'SettingsAuthPage' })

  interface AuthSettingForm {
    register_enabled: boolean
    register_required_fields: string[]
    register_email_required: boolean
    password_min_len: number
    password_require_upper: boolean
    password_require_lower: boolean
    password_require_number: boolean
    password_require_symbol: boolean
    register_verify_type: string
    register_verify_channels: string[]
    register_verify_ttl_sec: number
    login_rate_limit_enabled: boolean
    login_rate_limit_window_sec: number
    login_rate_limit_max_attempts: number
    auth_login_notify_enabled: boolean
    auth_login_notify_events: string[]
    auth_login_notify_channels: string[]
    auth_password_reset_enabled: boolean
    auth_password_reset_channels: string[]
    auth_password_reset_verify_ttl_sec: number
    auth_sms_code_len: number
    auth_sms_code_complexity: string
    auth_email_code_len: number
    auth_email_code_complexity: string
    auth_email_bind_enabled: boolean
    auth_phone_bind_enabled: boolean
    auth_contact_change_notify_old_enabled: boolean
    auth_contact_bind_verify_ttl_sec: number
    auth_bind_require_password_when_no_2fa: boolean
    auth_rebind_require_password_when_no_2fa: boolean
    auth_2fa_enabled: boolean
    auth_2fa_bind_enabled: boolean
    auth_2fa_rebind_enabled: boolean
  }

  interface OptionItem {
    label: string
    value: string
    disabled?: boolean
  }

  const loading = ref(false)
  const saving = ref(false)
  const initialized = ref(false)

  const userStore = useUserStore()
  const { info } = storeToRefs(userStore)

  const form = reactive<AuthSettingForm>(createDefaultForm())

  const requiredFieldOptions: OptionItem[] = [
    { label: '用户名', value: 'username', disabled: true },
    { label: '密码', value: 'password', disabled: true },
    { label: '手机号', value: 'phone' },
    { label: 'QQ', value: 'qq' }
  ]

  const verifyChannelOptions: OptionItem[] = [
    { label: '邮箱', value: 'email' },
    { label: '短信', value: 'sms' }
  ]

  const codeComplexityOptions: OptionItem[] = [
    { label: '纯数字', value: 'digits' },
    { label: '纯字母', value: 'letters' },
    { label: '字母 + 数字', value: 'alnum' }
  ]

  const loginNotifyEventOptions: OptionItem[] = [
    { label: '首次登录', value: 'first' },
    { label: 'IP 变化', value: 'ip_change' }
  ]

  const canView = computed(() => hasAdminPermission(info.value?.buttons, ['settings.view']))
  const canUpdate = computed(() => hasAdminPermission(info.value?.buttons, ['settings.update']))
  const formDisabled = computed(() => !canUpdate.value || loading.value || saving.value)

  watch(
    canView,
    (value) => {
      if (value && !initialized.value) {
        fetchData()
      }
    },
    { immediate: true }
  )

  function createDefaultForm(): AuthSettingForm {
    return {
      register_enabled: true,
      register_required_fields: ['username', 'password'],
      register_email_required: true,
      password_min_len: 6,
      password_require_upper: false,
      password_require_lower: false,
      password_require_number: false,
      password_require_symbol: false,
      register_verify_type: 'none',
      register_verify_channels: ['email'],
      register_verify_ttl_sec: 600,
      login_rate_limit_enabled: true,
      login_rate_limit_window_sec: 300,
      login_rate_limit_max_attempts: 5,
      auth_login_notify_enabled: true,
      auth_login_notify_events: ['first', 'ip_change'],
      auth_login_notify_channels: ['email'],
      auth_password_reset_enabled: true,
      auth_password_reset_channels: ['email'],
      auth_password_reset_verify_ttl_sec: 600,
      auth_sms_code_len: 6,
      auth_sms_code_complexity: 'digits',
      auth_email_code_len: 6,
      auth_email_code_complexity: 'alnum',
      auth_email_bind_enabled: true,
      auth_phone_bind_enabled: true,
      auth_contact_change_notify_old_enabled: true,
      auth_contact_bind_verify_ttl_sec: 600,
      auth_bind_require_password_when_no_2fa: false,
      auth_rebind_require_password_when_no_2fa: true,
      auth_2fa_enabled: true,
      auth_2fa_bind_enabled: true,
      auth_2fa_rebind_enabled: true
    }
  }

  function parseBool(value: unknown, defaultValue = false) {
    if (value === undefined || value === null || value === '') {
      return defaultValue
    }

    return value === true || value === 'true' || value === '1' || value === 1
  }

  function parseIntValue(value: unknown, defaultValue = 0) {
    if (value === undefined || value === null || value === '') {
      return defaultValue
    }

    const parsed = Number(value)
    return Number.isFinite(parsed) ? Math.floor(parsed) : defaultValue
  }

  function parseList(value: unknown, defaultValue: string[]) {
    if (!value) {
      return [...defaultValue]
    }

    try {
      const parsed = typeof value === 'string' ? JSON.parse(value) : value
      return Array.isArray(parsed) ? parsed.map((item) => String(item)) : [...defaultValue]
    } catch {
      return [...defaultValue]
    }
  }

  function normalizeComplexity(value: unknown, defaultValue: string) {
    const normalized = String(value || '')
      .trim()
      .toLowerCase()

    if (normalized === 'digits' || normalized === 'letters' || normalized === 'alnum') {
      return normalized
    }

    return defaultValue
  }

  function normalizeRequiredFields() {
    const normalized = new Set(
      form.register_required_fields.map((item) => String(item).trim().toLowerCase())
    )

    normalized.add('username')
    normalized.add('password')
    normalized.delete('email')

    form.register_required_fields = Array.from(normalized)
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

  function applySettings(items: SettingItemRecord[]) {
    const defaults = createDefaultForm()
    Object.assign(form, defaults)

    const settingsMap = createSettingsMap(items)

    form.register_enabled = parseBool(settingsMap.get('auth_register_enabled'), true)
    form.register_required_fields = parseList(settingsMap.get('auth_register_required_fields'), [
      'username',
      'password'
    ])
    form.register_email_required = parseBool(settingsMap.get('auth_register_email_required'), true)
    form.password_min_len = parseIntValue(settingsMap.get('auth_password_min_len'), 6)
    form.password_require_upper = parseBool(settingsMap.get('auth_password_require_upper'), false)
    form.password_require_lower = parseBool(settingsMap.get('auth_password_require_lower'), false)
    form.password_require_number = parseBool(settingsMap.get('auth_password_require_number'), false)
    form.password_require_symbol = parseBool(settingsMap.get('auth_password_require_symbol'), false)
    form.register_verify_type = String(settingsMap.get('auth_register_verify_type') || 'none')
    form.register_verify_channels = parseList(
      settingsMap.get('auth_register_verify_channels'),
      form.register_verify_type === 'none' ? [] : [form.register_verify_type]
    )
    form.register_verify_ttl_sec = parseIntValue(
      settingsMap.get('auth_register_verify_ttl_sec'),
      600
    )
    form.login_rate_limit_enabled = parseBool(
      settingsMap.get('auth_login_rate_limit_enabled'),
      true
    )
    form.login_rate_limit_window_sec = parseIntValue(
      settingsMap.get('auth_login_rate_limit_window_sec'),
      300
    )
    form.login_rate_limit_max_attempts = parseIntValue(
      settingsMap.get('auth_login_rate_limit_max_attempts'),
      5
    )
    form.auth_login_notify_enabled = parseBool(settingsMap.get('auth_login_notify_enabled'), true)
    form.auth_login_notify_channels = parseList(settingsMap.get('auth_login_notify_channels'), [
      'email'
    ])
    form.auth_login_notify_events = []
    if (parseBool(settingsMap.get('auth_login_notify_on_first_login'), true)) {
      form.auth_login_notify_events.push('first')
    }
    if (parseBool(settingsMap.get('auth_login_notify_on_ip_change'), true)) {
      form.auth_login_notify_events.push('ip_change')
    }
    form.auth_password_reset_enabled = parseBool(
      settingsMap.get('auth_password_reset_enabled'),
      true
    )
    form.auth_password_reset_channels = parseList(settingsMap.get('auth_password_reset_channels'), [
      'email'
    ])
    form.auth_password_reset_verify_ttl_sec = parseIntValue(
      settingsMap.get('auth_password_reset_verify_ttl_sec'),
      600
    )
    form.auth_sms_code_len = parseIntValue(settingsMap.get('auth_sms_code_len'), 6)
    form.auth_sms_code_complexity = normalizeComplexity(
      settingsMap.get('auth_sms_code_complexity'),
      'digits'
    )
    form.auth_email_code_len = parseIntValue(settingsMap.get('auth_email_code_len'), 6)
    form.auth_email_code_complexity = normalizeComplexity(
      settingsMap.get('auth_email_code_complexity'),
      'alnum'
    )
    form.auth_email_bind_enabled = parseBool(settingsMap.get('auth_email_bind_enabled'), true)
    form.auth_phone_bind_enabled = parseBool(settingsMap.get('auth_phone_bind_enabled'), true)
    form.auth_contact_change_notify_old_enabled = parseBool(
      settingsMap.get('auth_contact_change_notify_old_enabled'),
      true
    )
    form.auth_contact_bind_verify_ttl_sec = parseIntValue(
      settingsMap.get('auth_contact_bind_verify_ttl_sec'),
      600
    )
    form.auth_bind_require_password_when_no_2fa = parseBool(
      settingsMap.get('auth_bind_require_password_when_no_2fa'),
      false
    )
    form.auth_rebind_require_password_when_no_2fa = parseBool(
      settingsMap.get('auth_rebind_require_password_when_no_2fa'),
      true
    )
    form.auth_2fa_enabled = parseBool(settingsMap.get('auth_2fa_enabled'), true)
    form.auth_2fa_bind_enabled = parseBool(settingsMap.get('auth_2fa_bind_enabled'), true)
    form.auth_2fa_rebind_enabled = parseBool(settingsMap.get('auth_2fa_rebind_enabled'), true)

    normalizeRequiredFields()
  }

  async function fetchData() {
    if (!canView.value) {
      return
    }

    loading.value = true

    try {
      const payload = await fetchAdminSettings()
      applySettings(payload.items || [])
      initialized.value = true
    } finally {
      loading.value = false
    }
  }

  function toStringValue(value: boolean) {
    return value ? 'true' : 'false'
  }

  function normalizeVerifyType() {
    if (
      !Array.isArray(form.register_verify_channels) ||
      form.register_verify_channels.length === 0
    ) {
      form.register_verify_type = 'none'
      return
    }

    if (form.register_verify_channels.includes('email')) {
      form.register_verify_type = 'email'
      return
    }

    form.register_verify_type = 'sms'
  }

  async function handleSave() {
    normalizeRequiredFields()
    normalizeVerifyType()

    saving.value = true

    try {
      await updateAdminSettings({
        items: [
          { key: 'auth_register_enabled', value: toStringValue(form.register_enabled) },
          {
            key: 'auth_register_required_fields',
            value: JSON.stringify(form.register_required_fields)
          },
          {
            key: 'auth_register_email_required',
            value: toStringValue(form.register_email_required)
          },
          { key: 'auth_password_min_len', value: String(form.password_min_len ?? 6) },
          {
            key: 'auth_password_require_upper',
            value: toStringValue(form.password_require_upper)
          },
          {
            key: 'auth_password_require_lower',
            value: toStringValue(form.password_require_lower)
          },
          {
            key: 'auth_password_require_number',
            value: toStringValue(form.password_require_number)
          },
          {
            key: 'auth_password_require_symbol',
            value: toStringValue(form.password_require_symbol)
          },
          { key: 'auth_register_verify_type', value: form.register_verify_type },
          {
            key: 'auth_register_verify_channels',
            value: JSON.stringify(form.register_verify_channels || [])
          },
          {
            key: 'auth_register_verify_ttl_sec',
            value: String(form.register_verify_ttl_sec ?? 600)
          },
          {
            key: 'auth_login_rate_limit_enabled',
            value: toStringValue(form.login_rate_limit_enabled)
          },
          {
            key: 'auth_login_rate_limit_window_sec',
            value: String(form.login_rate_limit_window_sec ?? 300)
          },
          {
            key: 'auth_login_rate_limit_max_attempts',
            value: String(form.login_rate_limit_max_attempts ?? 5)
          },
          {
            key: 'auth_login_notify_enabled',
            value: toStringValue(form.auth_login_notify_enabled)
          },
          {
            key: 'auth_login_notify_channels',
            value: JSON.stringify(form.auth_login_notify_channels || [])
          },
          {
            key: 'auth_login_notify_on_first_login',
            value: toStringValue(form.auth_login_notify_events.includes('first'))
          },
          {
            key: 'auth_login_notify_on_ip_change',
            value: toStringValue(form.auth_login_notify_events.includes('ip_change'))
          },
          {
            key: 'auth_password_reset_enabled',
            value: toStringValue(form.auth_password_reset_enabled)
          },
          {
            key: 'auth_password_reset_channels',
            value: JSON.stringify(form.auth_password_reset_channels || [])
          },
          {
            key: 'auth_password_reset_verify_ttl_sec',
            value: String(form.auth_password_reset_verify_ttl_sec ?? 600)
          },
          { key: 'auth_sms_code_len', value: String(form.auth_sms_code_len ?? 6) },
          {
            key: 'auth_sms_code_complexity',
            value: normalizeComplexity(form.auth_sms_code_complexity, 'digits')
          },
          { key: 'auth_email_code_len', value: String(form.auth_email_code_len ?? 6) },
          {
            key: 'auth_email_code_complexity',
            value: normalizeComplexity(form.auth_email_code_complexity, 'alnum')
          },
          {
            key: 'auth_email_bind_enabled',
            value: toStringValue(form.auth_email_bind_enabled)
          },
          {
            key: 'auth_phone_bind_enabled',
            value: toStringValue(form.auth_phone_bind_enabled)
          },
          {
            key: 'auth_contact_change_notify_old_enabled',
            value: toStringValue(form.auth_contact_change_notify_old_enabled)
          },
          {
            key: 'auth_contact_bind_verify_ttl_sec',
            value: String(form.auth_contact_bind_verify_ttl_sec ?? 600)
          },
          {
            key: 'auth_bind_require_password_when_no_2fa',
            value: toStringValue(form.auth_bind_require_password_when_no_2fa)
          },
          {
            key: 'auth_rebind_require_password_when_no_2fa',
            value: toStringValue(form.auth_rebind_require_password_when_no_2fa)
          },
          { key: 'auth_2fa_enabled', value: toStringValue(form.auth_2fa_enabled) },
          { key: 'auth_2fa_bind_enabled', value: toStringValue(form.auth_2fa_bind_enabled) },
          {
            key: 'auth_2fa_rebind_enabled',
            value: toStringValue(form.auth_2fa_rebind_enabled)
          }
        ]
      })

      ElMessage.success('保存成功')
    } finally {
      saving.value = false
    }
  }
</script>

<style scoped lang="scss">
  .auth-settings-page {
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

  .page-actions {
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

  .section-card--accent {
    background: linear-gradient(145deg, rgb(14 165 233 / 6%), rgb(16 185 129 / 6%));
  }

  .section-title {
    color: var(--el-text-color-primary);
    font-size: 16px;
    font-weight: 700;
  }

  .full-width {
    width: 100%;
  }

  .switch-row {
    display: flex;
    align-items: center;
    gap: 12px;
    flex-wrap: wrap;
  }

  .switch-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 12px;
  }

  .stack-switches {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .switch-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    padding: 12px 14px;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 14px;
    background: var(--el-bg-color-page);
  }

  .switch-item--inline {
    width: 100%;
  }

  @media (max-width: 768px) {
    .page-header {
      flex-direction: column;
      align-items: stretch;
    }

    .page-actions {
      width: 100%;
    }

    .page-actions :deep(.el-button) {
      flex: 1;
    }

    .switch-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
