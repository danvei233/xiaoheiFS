<template>
  <div v-loading="loading" class="lifecycle-settings-page art-full-height">
    <div class="page-header">
      <div>
        <div class="page-kicker">Lifecycle</div>
        <div class="page-title">Lifecycle Settings</div>
        <div class="page-subtitle">
          Manage expiration reminders, automatic cleanup, and emergency renew behavior.
        </div>
      </div>

      <div class="page-actions">
        <ElButton v-if="canView" :disabled="loading || saving" @click="fetchData">Refresh</ElButton>
        <ElButton v-if="canUpdate" type="primary" :loading="saving" @click="handleSave">
          Save Settings
        </ElButton>
      </div>
    </div>

    <ElEmpty v-if="!canView" description="You do not have permission to view lifecycle settings." />

    <template v-else>
      <ElRow :gutter="16">
        <ElCol :xs="24" :xl="12">
          <ElCard shadow="never" class="section-card">
            <template #header>
              <div class="section-title">Expiry Reminder</div>
            </template>

            <ElForm label-position="top" :disabled="formDisabled">
              <ElFormItem label="Email Reminder">
                <div class="switch-row">
                  <ElSwitch v-model="form.email_expire_enabled" />
                  <span class="field-tip">
                    Send a reminder before a VPS instance reaches its expiration time.
                  </span>
                </div>
              </ElFormItem>

              <ElFormItem label="Reminder Days Before Expiry">
                <ElInputNumber
                  v-model="form.expire_reminder_days"
                  :min="0"
                  :step="1"
                  class="full-width"
                />
                <div class="field-tip">
                  Example: `7` means the reminder is sent 7 days before expiry.
                </div>
              </ElFormItem>
            </ElForm>
          </ElCard>

          <ElCard shadow="never" class="section-card">
            <template #header>
              <div class="section-title">Automatic Cleanup</div>
            </template>

            <ElForm label-position="top" :disabled="formDisabled">
              <ElFormItem label="Auto Delete Expired VPS">
                <div class="switch-row">
                  <ElSwitch v-model="form.auto_delete_enabled" />
                  <span class="field-tip">
                    Reclaim expired VPS instances automatically after the configured delay.
                  </span>
                </div>
              </ElFormItem>

              <ElFormItem label="Days After Expiry">
                <ElInputNumber
                  v-model="form.auto_delete_days"
                  :min="0"
                  :step="1"
                  :disabled="!form.auto_delete_enabled || formDisabled"
                  class="full-width"
                />
                <div class="field-tip">
                  The scheduled task will delete expired instances after this many days.
                </div>
              </ElFormItem>
            </ElForm>
          </ElCard>
        </ElCol>

        <ElCol :xs="24" :xl="12">
          <ElCard shadow="never" class="section-card section-card--accent">
            <template #header>
              <div class="section-title">Emergency Renew</div>
            </template>

            <ElForm label-position="top" :disabled="formDisabled">
              <ElFormItem label="Enable Emergency Renew">
                <div class="switch-row">
                  <ElSwitch v-model="form.emergency_renew_enabled" />
                  <span class="field-tip">
                    Allow users to extend expiry within a limited time window.
                  </span>
                </div>
              </ElFormItem>

              <ElFormItem label="Window Days Before Expiry">
                <ElInputNumber
                  v-model="form.emergency_renew_window_days"
                  :min="0"
                  :step="1"
                  :disabled="!form.emergency_renew_enabled || formDisabled"
                  class="full-width"
                />
                <div class="field-tip">
                  `0` means emergency renew is allowed whenever the service has not yet expired.
                </div>
              </ElFormItem>

              <ElFormItem label="Days Added Per Renew">
                <ElInputNumber
                  v-model="form.emergency_renew_days"
                  :min="1"
                  :step="1"
                  :disabled="!form.emergency_renew_enabled || formDisabled"
                  class="full-width"
                />
                <div class="field-tip">
                  Each successful emergency renew extends the expiry by this many days.
                </div>
              </ElFormItem>

              <ElFormItem label="Minimum Interval Hours">
                <ElInputNumber
                  v-model="form.emergency_renew_interval_hours"
                  :min="1"
                  :step="1"
                  :disabled="!form.emergency_renew_enabled || formDisabled"
                  class="full-width"
                />
                <div class="field-tip">
                  Minimum time between two emergency renew actions for the same service.
                </div>
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

  defineOptions({ name: 'SettingsLifecyclePage' })

  interface LifecycleSettingForm {
    email_expire_enabled: boolean
    expire_reminder_days: number
    auto_delete_enabled: boolean
    auto_delete_days: number
    emergency_renew_enabled: boolean
    emergency_renew_window_days: number
    emergency_renew_days: number
    emergency_renew_interval_hours: number
  }

  const loading = ref(false)
  const saving = ref(false)
  const initialized = ref(false)

  const userStore = useUserStore()
  const { info } = storeToRefs(userStore)

  const form = reactive<LifecycleSettingForm>(createDefaultForm())

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

  function createDefaultForm(): LifecycleSettingForm {
    return {
      email_expire_enabled: false,
      expire_reminder_days: 7,
      auto_delete_enabled: false,
      auto_delete_days: 7,
      emergency_renew_enabled: true,
      emergency_renew_window_days: 7,
      emergency_renew_days: 1,
      emergency_renew_interval_hours: 720
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

    const firstValue = String(value)
      .split(',')
      .map((item) => item.trim())
      .find(Boolean)
    const parsed = Number.parseInt(firstValue || '', 10)

    return Number.isFinite(parsed) ? parsed : defaultValue
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

    form.email_expire_enabled = parseBool(settingsMap.get('email_expire_enabled'), false)
    form.expire_reminder_days = Math.max(
      0,
      parseIntValue(settingsMap.get('expire_reminder_days'), 7)
    )
    form.auto_delete_enabled = parseBool(settingsMap.get('auto_delete_enabled'), false)
    form.auto_delete_days = Math.max(0, parseIntValue(settingsMap.get('auto_delete_days'), 7))
    form.emergency_renew_enabled = parseBool(settingsMap.get('emergency_renew_enabled'), true)
    form.emergency_renew_window_days = Math.max(
      0,
      parseIntValue(settingsMap.get('emergency_renew_window_days'), 7)
    )
    form.emergency_renew_days = Math.max(
      1,
      parseIntValue(settingsMap.get('emergency_renew_days'), 1)
    )
    form.emergency_renew_interval_hours = Math.max(
      1,
      parseIntValue(settingsMap.get('emergency_renew_interval_hours'), 720)
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
          { key: 'email_expire_enabled', value: String(form.email_expire_enabled) },
          { key: 'expire_reminder_days', value: String(Math.max(0, form.expire_reminder_days)) },
          { key: 'auto_delete_enabled', value: String(form.auto_delete_enabled) },
          { key: 'auto_delete_days', value: String(Math.max(0, form.auto_delete_days)) },
          { key: 'emergency_renew_enabled', value: String(form.emergency_renew_enabled) },
          {
            key: 'emergency_renew_window_days',
            value: String(Math.max(0, form.emergency_renew_window_days))
          },
          { key: 'emergency_renew_days', value: String(Math.max(1, form.emergency_renew_days)) },
          {
            key: 'emergency_renew_interval_hours',
            value: String(Math.max(1, form.emergency_renew_interval_hours))
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
  .lifecycle-settings-page {
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
    background: linear-gradient(145deg, rgb(16 185 129 / 6%), rgb(59 130 246 / 6%));
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
