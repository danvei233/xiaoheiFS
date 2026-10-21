<template>
  <div v-loading="loading" class="pricing-settings-page art-full-height">
    <div class="page-header">
      <div>
        <div class="page-kicker">Pricing</div>
        <div class="page-title">Pricing And Refund Settings</div>
        <div class="page-subtitle">
          Configure refund windows, refund curves, and resize pricing behavior.
        </div>
      </div>

      <div class="page-actions">
        <ElButton v-if="canView" :disabled="loading || saving" @click="fetchData">Refresh</ElButton>
        <ElButton v-if="canUpdate" type="primary" :loading="saving" @click="handleSave">
          Save Settings
        </ElButton>
      </div>
    </div>

    <ElEmpty v-if="!canView" description="You do not have permission to view pricing settings." />

    <template v-else>
      <ElRow :gutter="16">
        <ElCol :xs="24" :xl="12">
          <ElCard shadow="never" class="section-card">
            <template #header>
              <div class="section-title">Refund Windows</div>
            </template>

            <ElForm label-position="top" :disabled="formDisabled">
              <ElRow :gutter="12">
                <ElCol :xs="24" :md="12">
                  <ElFormItem label="Full Refund Days">
                    <ElInputNumber
                      v-model="form.refund_full_days"
                      :min="0"
                      :step="1"
                      class="full-width"
                    />
                  </ElFormItem>
                </ElCol>

                <ElCol :xs="24" :md="12">
                  <ElFormItem label="Prorated Refund Days">
                    <ElInputNumber
                      v-model="form.refund_prorate_days"
                      :min="0"
                      :step="1"
                      class="full-width"
                    />
                  </ElFormItem>
                </ElCol>
              </ElRow>

              <ElFormItem label="No Refund After Days">
                <ElInputNumber
                  v-model="form.refund_no_refund_days"
                  :min="0"
                  :step="1"
                  class="full-width"
                />
                <div class="field-tip">
                  After this threshold, users can no longer request a refund.
                </div>
              </ElFormItem>

              <ElDivider>Hour Override</ElDivider>

              <ElRow :gutter="12">
                <ElCol :xs="24" :md="8">
                  <ElFormItem label="Full Refund Hours">
                    <ElInputNumber
                      v-model="form.refund_full_hours"
                      :min="0"
                      :step="1"
                      class="full-width"
                    />
                  </ElFormItem>
                </ElCol>

                <ElCol :xs="24" :md="8">
                  <ElFormItem label="Prorated Refund Hours">
                    <ElInputNumber
                      v-model="form.refund_prorate_hours"
                      :min="0"
                      :step="1"
                      class="full-width"
                    />
                  </ElFormItem>
                </ElCol>

                <ElCol :xs="24" :md="8">
                  <ElFormItem label="No Refund After Hours">
                    <ElInputNumber
                      v-model="form.refund_no_refund_hours"
                      :min="0"
                      :step="1"
                      class="full-width"
                    />
                  </ElFormItem>
                </ElCol>
              </ElRow>

              <ElDivider />

              <ElFormItem label="Approval Required">
                <div class="switch-row">
                  <ElSwitch v-model="form.refund_requires_approval" />
                  <span class="field-tip">
                    When enabled, refund requests enter a manual review queue first.
                  </span>
                </div>
              </ElFormItem>

              <ElFormItem label="Refund On Admin Delete">
                <div class="switch-row">
                  <ElSwitch v-model="form.refund_on_admin_delete" />
                  <span class="field-tip">
                    Automatically calculate and process refunds when an admin deletes a service.
                  </span>
                </div>
              </ElFormItem>
            </ElForm>
          </ElCard>
        </ElCol>

        <ElCol :xs="24" :xl="12">
          <ElCard shadow="never" class="section-card section-card--accent">
            <template #header>
              <div class="section-title">Refund Curve JSON</div>
            </template>

            <ElForm label-position="top" :disabled="formDisabled">
              <ElAlert
                type="info"
                :closable="false"
                show-icon
                title="Curve Priority"
                description="If a refund curve is configured, it takes priority over the day and hour windows on the left."
              />

              <ElFormItem label="Curve Points" class="curve-field">
                <ElInput
                  v-model="form.refund_curve_json"
                  type="textarea"
                  :rows="14"
                  placeholder='[
  { "percent": 0, "ratio": 1 },
  { "percent": 10, "ratio": 0.9 },
  { "percent": 100, "ratio": 0 }
]'
                />
                <div class="field-tip">
                  `percent` is usage percentage from `0-100`, and `ratio` is refund ratio from
                  `0-1`.
                </div>
              </ElFormItem>
            </ElForm>
          </ElCard>
        </ElCol>
      </ElRow>

      <ElCard shadow="never" class="section-card resize-card">
        <template #header>
          <div class="section-title">Resize Pricing</div>
        </template>

        <ElForm label-position="top" :disabled="formDisabled">
          <ElRow :gutter="16">
            <ElCol :xs="24" :md="8">
              <ElFormItem label="Resize Price Mode">
                <ElSelect v-model="form.resize_price_mode" class="full-width">
                  <ElOption label="Remaining Period Ratio" value="remaining" />
                </ElSelect>
                <div class="field-tip">
                  The backend currently supports only the remaining-period pricing mode.
                </div>
              </ElFormItem>
            </ElCol>

            <ElCol :xs="24" :md="8">
              <ElFormItem label="Refund Ratio">
                <ElInputNumber
                  v-model="form.resize_refund_ratio"
                  :min="0"
                  :max="1"
                  :step="0.01"
                  class="full-width"
                />
              </ElFormItem>
            </ElCol>

            <ElCol :xs="24" :md="8">
              <ElFormItem label="Rounding Mode">
                <ElSelect v-model="form.resize_rounding" class="full-width">
                  <ElOption label="Round" value="round" />
                  <ElOption label="Ceil" value="ceil" />
                  <ElOption label="Floor" value="floor" />
                </ElSelect>
              </ElFormItem>
            </ElCol>
          </ElRow>

          <ElRow :gutter="16">
            <ElCol :xs="24" :md="8">
              <ElFormItem label="Minimum Charge">
                <ElInputNumber
                  v-model="form.resize_min_charge"
                  :min="0"
                  :step="0.01"
                  class="full-width"
                />
              </ElFormItem>
            </ElCol>

            <ElCol :xs="24" :md="8">
              <ElFormItem label="Minimum Refund">
                <ElInputNumber
                  v-model="form.resize_min_refund"
                  :min="0"
                  :step="0.01"
                  class="full-width"
                />
              </ElFormItem>
            </ElCol>

            <ElCol :xs="24" :md="8">
              <ElFormItem label="Refund To Wallet">
                <div class="switch-row switch-row--compact">
                  <ElSwitch v-model="form.resize_refund_to_wallet" />
                  <span class="field-tip">Route resize refunds to the user wallet.</span>
                </div>
              </ElFormItem>
            </ElCol>
          </ElRow>
        </ElForm>
      </ElCard>
    </template>
  </div>
</template>

<script setup lang="ts">
  import type { SettingItemRecord } from '@/api/admin'
  import { fetchAdminSettings, hasAdminPermission, updateAdminSettings } from '@/api/admin'
  import { useUserStore } from '@/store/modules/user'

  defineOptions({ name: 'SettingsPricingPage' })

  type ResizeRounding = 'round' | 'ceil' | 'floor'

  interface PricingSettingForm {
    refund_full_days: number
    refund_prorate_days: number
    refund_no_refund_days: number
    refund_full_hours: number
    refund_prorate_hours: number
    refund_no_refund_hours: number
    refund_requires_approval: boolean
    refund_on_admin_delete: boolean
    refund_curve_json: string
    resize_price_mode: string
    resize_refund_ratio: number
    resize_rounding: ResizeRounding
    resize_min_charge: number
    resize_min_refund: number
    resize_refund_to_wallet: boolean
  }

  interface RefundCurvePoint {
    percent: number
    ratio: number
  }

  const loading = ref(false)
  const saving = ref(false)
  const initialized = ref(false)

  const userStore = useUserStore()
  const { info } = storeToRefs(userStore)

  const form = reactive<PricingSettingForm>(createDefaultForm())

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

  function createDefaultForm(): PricingSettingForm {
    return {
      refund_full_days: 1,
      refund_prorate_days: 7,
      refund_no_refund_days: 30,
      refund_full_hours: 0,
      refund_prorate_hours: 0,
      refund_no_refund_hours: 0,
      refund_requires_approval: true,
      refund_on_admin_delete: true,
      refund_curve_json: '[]',
      resize_price_mode: 'remaining',
      resize_refund_ratio: 1,
      resize_rounding: 'round',
      resize_min_charge: 0,
      resize_min_refund: 0,
      resize_refund_to_wallet: true
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

    const parsed = Number.parseInt(String(value), 10)
    return Number.isFinite(parsed) ? parsed : defaultValue
  }

  function parseFloatValue(value: unknown, defaultValue = 0) {
    if (value === undefined || value === null || value === '') {
      return defaultValue
    }

    const parsed = Number.parseFloat(String(value))
    return Number.isFinite(parsed) ? parsed : defaultValue
  }

  function normalizeRounding(value: unknown, defaultValue: ResizeRounding) {
    const normalized = String(value || '')
      .trim()
      .toLowerCase()

    if (normalized === 'ceil' || normalized === 'floor' || normalized === 'round') {
      return normalized as ResizeRounding
    }

    return defaultValue
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

    form.refund_full_days = Math.max(0, parseIntValue(settingsMap.get('refund_full_days'), 1))
    form.refund_prorate_days = Math.max(0, parseIntValue(settingsMap.get('refund_prorate_days'), 7))
    form.refund_no_refund_days = Math.max(
      0,
      parseIntValue(settingsMap.get('refund_no_refund_days'), 30)
    )
    form.refund_full_hours = Math.max(0, parseIntValue(settingsMap.get('refund_full_hours'), 0))
    form.refund_prorate_hours = Math.max(
      0,
      parseIntValue(settingsMap.get('refund_prorate_hours'), 0)
    )
    form.refund_no_refund_hours = Math.max(
      0,
      parseIntValue(settingsMap.get('refund_no_refund_hours'), 0)
    )
    form.refund_requires_approval = parseBool(settingsMap.get('refund_requires_approval'), true)
    form.refund_on_admin_delete = parseBool(settingsMap.get('refund_on_admin_delete'), true)
    form.refund_curve_json = String(settingsMap.get('refund_curve_json') || '[]').trim() || '[]'
    form.resize_price_mode =
      String(settingsMap.get('resize_price_mode') || 'remaining').trim() || 'remaining'
    form.resize_refund_ratio = parseFloatValue(settingsMap.get('resize_refund_ratio'), 1)
    form.resize_rounding = normalizeRounding(settingsMap.get('resize_rounding'), 'round')
    form.resize_min_charge = parseFloatValue(settingsMap.get('resize_min_charge'), 0)
    form.resize_min_refund = parseFloatValue(settingsMap.get('resize_min_refund'), 0)
    form.resize_refund_to_wallet = parseBool(settingsMap.get('resize_refund_to_wallet'), true)
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

  function normalizeCurveJson(value: string) {
    const text = String(value || '').trim()
    if (!text) {
      return '[]'
    }

    let parsed: unknown

    try {
      parsed = JSON.parse(text)
    } catch {
      throw new Error('Refund curve must be valid JSON')
    }

    if (!Array.isArray(parsed)) {
      throw new Error('Refund curve must be a JSON array')
    }

    const normalized: RefundCurvePoint[] = parsed.map((item, index) => {
      const record = item as Record<string, unknown>
      const percent = Number(record?.percent ?? record?.hours)
      const ratio = Number(record?.ratio)

      if (!Number.isFinite(percent) || percent < 0 || percent > 100) {
        throw new Error(`Refund curve item ${index + 1} has an invalid percent value`)
      }

      if (!Number.isFinite(ratio) || ratio < 0 || ratio > 1) {
        throw new Error(`Refund curve item ${index + 1} has an invalid ratio value`)
      }

      return {
        percent: Math.round(percent),
        ratio
      }
    })

    normalized.sort((left, right) => left.percent - right.percent)

    return JSON.stringify(normalized, null, 2)
  }

  async function handleSave() {
    saving.value = true

    try {
      form.refund_curve_json = normalizeCurveJson(form.refund_curve_json)

      await updateAdminSettings({
        items: [
          { key: 'refund_full_days', value: String(Math.max(0, form.refund_full_days)) },
          { key: 'refund_prorate_days', value: String(Math.max(0, form.refund_prorate_days)) },
          {
            key: 'refund_no_refund_days',
            value: String(Math.max(0, form.refund_no_refund_days))
          },
          { key: 'refund_full_hours', value: String(Math.max(0, form.refund_full_hours)) },
          {
            key: 'refund_prorate_hours',
            value: String(Math.max(0, form.refund_prorate_hours))
          },
          {
            key: 'refund_no_refund_hours',
            value: String(Math.max(0, form.refund_no_refund_hours))
          },
          {
            key: 'refund_requires_approval',
            value: String(form.refund_requires_approval)
          },
          {
            key: 'refund_on_admin_delete',
            value: String(form.refund_on_admin_delete)
          },
          { key: 'refund_curve_json', value: form.refund_curve_json },
          { key: 'resize_price_mode', value: String(form.resize_price_mode || 'remaining') },
          {
            key: 'resize_refund_ratio',
            value: String(Math.min(1, Math.max(0, form.resize_refund_ratio)))
          },
          { key: 'resize_rounding', value: normalizeRounding(form.resize_rounding, 'round') },
          { key: 'resize_min_charge', value: String(Math.max(0, form.resize_min_charge)) },
          { key: 'resize_min_refund', value: String(Math.max(0, form.resize_min_refund)) },
          {
            key: 'resize_refund_to_wallet',
            value: String(form.resize_refund_to_wallet)
          }
        ]
      })

      ElMessage.success('Settings saved successfully')
    } catch (error) {
      ElMessage.error(error instanceof Error ? error.message : 'Failed to save settings')
    } finally {
      saving.value = false
    }
  }
</script>

<style scoped lang="scss">
  .pricing-settings-page {
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
    background: linear-gradient(145deg, rgb(245 158 11 / 6%), rgb(59 130 246 / 6%));
  }

  .resize-card {
    margin-top: 16px;
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

  .switch-row--compact {
    min-height: 32px;
  }

  .curve-field {
    margin-top: 16px;
    margin-bottom: 0;
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
