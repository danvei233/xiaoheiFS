<template>
  <div v-loading="loading" class="fcm-settings-page art-full-height">
    <div class="page-header">
      <div>
        <div class="page-kicker">Push</div>
        <div class="page-title">FCM Settings</div>
        <div class="page-subtitle">
          Configure Firebase Cloud Messaging for admin push notifications.
        </div>
      </div>

      <div class="page-actions">
        <ElButton v-if="canView" :disabled="loading || saving" @click="fetchData">Refresh</ElButton>
        <ElButton v-if="canUpdate" type="primary" :loading="saving" @click="handleSave">
          Save Settings
        </ElButton>
      </div>
    </div>

    <ElEmpty v-if="!canView" description="You do not have permission to view FCM settings." />

    <template v-else>
      <ElRow :gutter="16">
        <ElCol :xs="24" :xl="16">
          <ElCard shadow="never" class="section-card">
            <template #header>
              <div class="section-title">Firebase Cloud Messaging</div>
            </template>

            <ElForm label-position="top" :disabled="formDisabled">
              <ElFormItem label="Enable FCM">
                <div class="switch-row">
                  <ElSwitch v-model="form.fcm_enabled" />
                  <span class="field-tip">
                    Disable this to stop sending push notifications to registered admin devices.
                  </span>
                </div>
              </ElFormItem>

              <ElFormItem label="Legacy Server Key">
                <ElInput
                  v-model="form.fcm_server_key"
                  type="textarea"
                  :rows="3"
                  placeholder="Optional legacy server key"
                />
                <div class="field-tip">
                  Keep this only if you still need the legacy FCM sending path.
                </div>
              </ElFormItem>

              <ElFormItem label="Project ID">
                <ElInput
                  v-model="form.fcm_project_id"
                  maxlength="255"
                  placeholder="your-firebase-project-id"
                />
              </ElFormItem>

              <ElFormItem label="Service Account JSON">
                <ElInput
                  v-model="form.fcm_service_account_json"
                  type="textarea"
                  :rows="10"
                  placeholder='{"type":"service_account","project_id":"...","private_key":"..."}'
                />
              </ElFormItem>
            </ElForm>
          </ElCard>
        </ElCol>

        <ElCol :xs="24" :xl="8">
          <ElCard shadow="never" class="section-card section-card--accent">
            <template #header>
              <div class="section-title">Notes</div>
            </template>

            <ElAlert
              type="info"
              :closable="false"
              show-icon
              title="Preferred Configuration"
              description="Use Project ID plus Service Account JSON for HTTP v1. Admin devices still need to register tokens through /admin/api/v1/push-tokens."
            />
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

  defineOptions({ name: 'SettingsFcmPage' })

  interface FcmSettingForm {
    fcm_enabled: boolean
    fcm_server_key: string
    fcm_project_id: string
    fcm_service_account_json: string
  }

  const loading = ref(false)
  const saving = ref(false)
  const initialized = ref(false)

  const userStore = useUserStore()
  const { info } = storeToRefs(userStore)

  const form = reactive<FcmSettingForm>(createDefaultForm())

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

  function createDefaultForm(): FcmSettingForm {
    return {
      fcm_enabled: false,
      fcm_server_key: '',
      fcm_project_id: '',
      fcm_service_account_json: ''
    }
  }

  function parseBool(value: unknown, defaultValue = false) {
    if (value === undefined || value === null || value === '') {
      return defaultValue
    }

    return value === true || value === 'true' || value === '1' || value === 1
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

    form.fcm_enabled = parseBool(settingsMap.get('fcm_enabled'), false)
    form.fcm_server_key = String(settingsMap.get('fcm_server_key') || '')
    form.fcm_project_id = String(settingsMap.get('fcm_project_id') || '')
    form.fcm_service_account_json = String(settingsMap.get('fcm_service_account_json') || '')
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
          { key: 'fcm_enabled', value: form.fcm_enabled ? 'true' : 'false' },
          { key: 'fcm_server_key', value: String(form.fcm_server_key || '') },
          { key: 'fcm_project_id', value: String(form.fcm_project_id || '') },
          {
            key: 'fcm_service_account_json',
            value: String(form.fcm_service_account_json || '')
          }
        ]
      })

      ElMessage.success('Settings saved successfully')
    } finally {
      saving.value = false
    }
  }
</script>

<style scoped lang="scss">
  .fcm-settings-page {
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

  .section-card--accent {
    background: linear-gradient(145deg, rgb(59 130 246 / 6%), rgb(16 185 129 / 6%));
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
