<template>
  <div v-loading="loading" class="captcha-settings-page art-full-height">
    <div class="page-header">
      <div>
        <div class="page-kicker">Security</div>
        <div class="page-title">验证码设置</div>
        <div class="page-subtitle">统一管理登录/注册验证码开关、图形验证码策略与极验参数。</div>
      </div>

      <div class="page-actions">
        <ElButton v-if="canView" :disabled="loading || saving" @click="fetchData">刷新</ElButton>
        <ElButton v-if="canUpdate" type="primary" :loading="saving" @click="handleSave">
          保存设置
        </ElButton>
      </div>
    </div>

    <ElEmpty v-if="!canView" description="当前账号没有查看验证码设置的权限" />

    <template v-else>
      <ElRow :gutter="16">
        <ElCol :xs="24" :xl="12">
          <ElCard shadow="never" class="section-card">
            <template #header>
              <div class="section-title">基础开关</div>
            </template>

            <ElForm label-position="top" :disabled="formDisabled">
              <ElFormItem label="注册启用验证码">
                <div class="switch-row">
                  <ElSwitch v-model="form.register_captcha_enabled" />
                  <span class="field-tip">启用后，用户注册前需要先完成验证码验证。</span>
                </div>
              </ElFormItem>

              <ElFormItem label="登录启用验证码">
                <div class="switch-row">
                  <ElSwitch v-model="form.login_captcha_enabled" />
                  <span class="field-tip">启用后，后台登录页会要求先通过验证码校验。</span>
                </div>
              </ElFormItem>

              <ElFormItem label="验证码方案">
                <ElRadioGroup v-model="form.auth_captcha_provider">
                  <ElRadio value="image">图形验证码</ElRadio>
                  <ElRadio value="geetest">极验（GeeTest）</ElRadio>
                </ElRadioGroup>
              </ElFormItem>

              <ElAlert
                type="info"
                :closable="false"
                show-icon
                title="方案说明"
                description="切换到极验后，登录和注册验证码会改为极验行为验证；图形验证码参数仍会保留，切回时继续使用。"
              />
            </ElForm>
          </ElCard>
        </ElCol>

        <ElCol :xs="24" :xl="12">
          <ElCard shadow="never" class="section-card">
            <template #header>
              <div class="section-title">图形验证码参数</div>
            </template>

            <ElForm label-position="top" :disabled="formDisabled">
              <ElRow :gutter="16">
                <ElCol :xs="24" :md="12">
                  <ElFormItem label="长度">
                    <ElInputNumber
                      v-model="form.auth_captcha_code_len"
                      :min="4"
                      :max="12"
                      :step="1"
                      class="full-width"
                    />
                  </ElFormItem>
                </ElCol>

                <ElCol :xs="24" :md="12">
                  <ElFormItem label="复杂度">
                    <ElSelect v-model="form.auth_captcha_code_complexity" class="full-width">
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
            </ElForm>
          </ElCard>

          <ElCard shadow="never" class="section-card section-card--accent">
            <template #header>
              <div class="section-title">极验参数</div>
            </template>

            <ElForm label-position="top" :disabled="formDisabled">
              <ElFormItem label="Captcha ID">
                <ElInput
                  v-model="form.auth_geetest_captcha_id"
                  maxlength="255"
                  placeholder="请输入极验 captcha_id"
                />
              </ElFormItem>

              <ElFormItem label="Captcha Key">
                <ElInput
                  v-model="form.auth_geetest_captcha_key"
                  type="password"
                  show-password
                  maxlength="255"
                  placeholder="请输入极验 captcha_key"
                />
              </ElFormItem>

              <ElFormItem label="API Server">
                <ElInput
                  v-model="form.auth_geetest_api_server"
                  maxlength="1024"
                  placeholder="https://gcaptcha4.geetest.com"
                />
              </ElFormItem>
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

  defineOptions({ name: 'SettingsCaptchaPage' })

  interface CaptchaSettingsForm {
    register_captcha_enabled: boolean
    login_captcha_enabled: boolean
    auth_captcha_provider: 'image' | 'geetest'
    auth_captcha_code_len: number
    auth_captcha_code_complexity: 'digits' | 'letters' | 'alnum'
    auth_geetest_captcha_id: string
    auth_geetest_captcha_key: string
    auth_geetest_api_server: string
  }

  interface OptionItem {
    label: string
    value: 'digits' | 'letters' | 'alnum'
  }

  const loading = ref(false)
  const saving = ref(false)
  const initialized = ref(false)

  const userStore = useUserStore()
  const { info } = storeToRefs(userStore)

  const form = reactive<CaptchaSettingsForm>(createDefaultForm())

  const codeComplexityOptions: OptionItem[] = [
    { label: '纯数字', value: 'digits' },
    { label: '纯字母', value: 'letters' },
    { label: '字母 + 数字', value: 'alnum' }
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

  function createDefaultForm(): CaptchaSettingsForm {
    return {
      register_captcha_enabled: true,
      login_captcha_enabled: false,
      auth_captcha_provider: 'image',
      auth_captcha_code_len: 5,
      auth_captcha_code_complexity: 'alnum',
      auth_geetest_captcha_id: '',
      auth_geetest_captcha_key: '',
      auth_geetest_api_server: 'https://gcaptcha4.geetest.com'
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

  function normalizeComplexity(
    value: unknown,
    defaultValue: CaptchaSettingsForm['auth_captcha_code_complexity']
  ) {
    const normalized = String(value || '')
      .trim()
      .toLowerCase()

    if (normalized === 'digits' || normalized === 'letters' || normalized === 'alnum') {
      return normalized as CaptchaSettingsForm['auth_captcha_code_complexity']
    }

    return defaultValue
  }

  function normalizeProvider(value: unknown) {
    return String(value || '')
      .trim()
      .toLowerCase() === 'geetest'
      ? 'geetest'
      : 'image'
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
    Object.assign(form, createDefaultForm())

    const settingsMap = createSettingsMap(items)

    form.register_captcha_enabled = parseBool(
      settingsMap.get('auth_register_captcha_enabled'),
      true
    )
    form.login_captcha_enabled = parseBool(settingsMap.get('auth_login_captcha_enabled'), false)
    form.auth_captcha_provider = normalizeProvider(settingsMap.get('auth_captcha_provider'))
    form.auth_captcha_code_len = parseIntValue(settingsMap.get('auth_captcha_code_len'), 5)
    form.auth_captcha_code_complexity = normalizeComplexity(
      settingsMap.get('auth_captcha_code_complexity'),
      'alnum'
    )
    form.auth_geetest_captcha_id = String(settingsMap.get('auth_geetest_captcha_id') || '')
    form.auth_geetest_captcha_key = String(settingsMap.get('auth_geetest_captcha_key') || '')
    form.auth_geetest_api_server = String(
      settingsMap.get('auth_geetest_api_server') || 'https://gcaptcha4.geetest.com'
    )
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

  async function handleSave() {
    saving.value = true

    try {
      await updateAdminSettings({
        items: [
          {
            key: 'auth_register_captcha_enabled',
            value: form.register_captcha_enabled ? 'true' : 'false'
          },
          {
            key: 'auth_login_captcha_enabled',
            value: form.login_captcha_enabled ? 'true' : 'false'
          },
          {
            key: 'auth_captcha_provider',
            value: normalizeProvider(form.auth_captcha_provider)
          },
          {
            key: 'auth_captcha_code_len',
            value: String(form.auth_captcha_code_len || 5)
          },
          {
            key: 'auth_captcha_code_complexity',
            value: normalizeComplexity(form.auth_captcha_code_complexity, 'alnum')
          },
          {
            key: 'auth_geetest_captcha_id',
            value: String(form.auth_geetest_captcha_id || '').trim()
          },
          {
            key: 'auth_geetest_captcha_key',
            value: String(form.auth_geetest_captcha_key || '').trim()
          },
          {
            key: 'auth_geetest_api_server',
            value: String(form.auth_geetest_api_server || 'https://gcaptcha4.geetest.com').trim()
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
  .captcha-settings-page {
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
    background: linear-gradient(145deg, rgb(245 158 11 / 6%), rgb(14 165 233 / 6%));
  }

  .section-title {
    color: var(--el-text-color-primary);
    font-size: 16px;
    font-weight: 700;
  }

  .switch-row {
    display: flex;
    align-items: center;
    gap: 12px;
    flex-wrap: wrap;
  }

  .full-width {
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
  }
</style>
