<template>
  <div v-loading="loading" class="webhook-settings-page art-full-height">
    <div class="page-header">
      <div>
        <div class="page-kicker">Notifications</div>
        <div class="page-title">Webhook Settings</div>
        <div class="page-subtitle">
          Configure multiple webhook endpoints for robot notifications and run a broadcast test.
        </div>
      </div>

      <div class="page-actions">
        <ElButton v-if="canUpdate" :disabled="saving" @click="addWebhook">Add Webhook</ElButton>
        <ElButton
          v-if="canView"
          :loading="testLoading"
          :disabled="loading"
          @click="handleTestWebhook"
        >
          Send Test
        </ElButton>
        <ElButton
          v-if="canUpdate"
          type="primary"
          :loading="saving"
          :disabled="loading"
          @click="handleSave"
        >
          Save
        </ElButton>
      </div>
    </div>

    <ElEmpty v-if="!canView" description="You do not have permission to view webhook settings." />

    <template v-else>
      <ElAlert
        type="info"
        :closable="false"
        show-icon
        title="Event behavior"
        description="Leaving the events field empty means the webhook receives all supported events."
      />

      <ElCard shadow="never" class="section-card">
        <template #header>
          <div class="section-title">Webhook List</div>
        </template>

        <ElTable :data="webhooks" border row-key="_key">
          <ElTableColumn label="Name" min-width="180">
            <template #default="{ row }">
              <ElInput
                v-model="row.name"
                :disabled="!canUpdate"
                maxlength="120"
                placeholder="Webhook name"
              />
            </template>
          </ElTableColumn>

          <ElTableColumn label="Webhook URL" min-width="320">
            <template #default="{ row }">
              <ElInput
                v-model="row.url"
                :disabled="!canUpdate"
                maxlength="2048"
                placeholder="https://..."
              />
            </template>
          </ElTableColumn>

          <ElTableColumn label="Secret" min-width="220">
            <template #default="{ row }">
              <ElInput
                v-model="row.secret"
                :disabled="!canUpdate"
                maxlength="255"
                placeholder="Optional signing secret"
              />
            </template>
          </ElTableColumn>

          <ElTableColumn label="Events" min-width="260">
            <template #default="{ row }">
              <ElSelect
                v-model="row.events"
                :disabled="!canUpdate"
                class="full-width"
                multiple
                filterable
                allow-create
                default-first-option
                collapse-tags
                collapse-tags-tooltip
                placeholder="Empty means all events"
              >
                <ElOption
                  v-for="item in eventOptions"
                  :key="item.value"
                  :label="item.label"
                  :value="item.value"
                />
              </ElSelect>
            </template>
          </ElTableColumn>

          <ElTableColumn label="Enabled" width="100">
            <template #default="{ row }">
              <ElSwitch v-model="row.enabled" :disabled="!canUpdate" />
            </template>
          </ElTableColumn>

          <ElTableColumn label="Actions" width="110" fixed="right">
            <template #default="{ $index }">
              <ElButton link type="danger" :disabled="!canUpdate" @click="removeWebhook($index)">
                Remove
              </ElButton>
            </template>
          </ElTableColumn>
        </ElTable>

        <div v-if="!webhooks.length" class="empty-state">
          <ElEmpty description="No webhook configured yet." />
        </div>
      </ElCard>
    </template>
  </div>
</template>

<script setup lang="ts">
  import type { RobotConfigRecord, RobotWebhookRecord } from '@/api/admin'
  import {
    fetchRobotConfig,
    hasAdminPermission,
    testRobotWebhook,
    updateRobotConfig
  } from '@/api/admin'
  import { useUserStore } from '@/store/modules/user'
  import { ElMessage } from 'element-plus'

  defineOptions({ name: 'SettingsWebhookPage' })

  interface LocalWebhookRecord {
    _key: string
    name: string
    url: string
    secret: string
    enabled: boolean
    events: string[]
  }

  interface EventOption {
    label: string
    value: string
  }

  const eventOptions: EventOption[] = [
    { label: 'Order pending payment', value: 'order.pending_payment' },
    { label: 'Order pending review', value: 'order.pending_review' },
    { label: 'Order approved', value: 'order.approved' },
    { label: 'Order rejected', value: 'order.rejected' },
    { label: 'Order canceled', value: 'order.canceled' },
    { label: 'Order provisioning', value: 'order.provisioning' },
    { label: 'Order completed', value: 'order.completed' },
    { label: 'Order item active', value: 'order.item.active' },
    { label: 'Order item failed', value: 'order.item.failed' },
    { label: 'Payment created', value: 'payment.created' },
    { label: 'Payment confirmed', value: 'payment.confirmed' },
    { label: 'Payment approved', value: 'payment.approved' },
    { label: 'Webhook test', value: 'webhook.test' }
  ]

  const loading = ref(false)
  const saving = ref(false)
  const testLoading = ref(false)
  const initialized = ref(false)
  const webhooks = ref<LocalWebhookRecord[]>([])

  const userStore = useUserStore()
  const { info } = storeToRefs(userStore)

  const canView = computed(() => hasAdminPermission(info.value?.buttons, ['settings.view']))
  const canUpdate = computed(() => hasAdminPermission(info.value?.buttons, ['settings.update']))

  watch(
    canView,
    (value) => {
      if (value && !initialized.value) {
        loadData()
      }
    },
    { immediate: true }
  )

  function createKey() {
    return `${Date.now()}-${Math.random().toString(16).slice(2)}`
  }

  function normalizeWebhooks(items?: RobotWebhookRecord[]) {
    return (items || []).map((item) => ({
      _key: createKey(),
      name: String(item.name || 'Webhook'),
      url: String(item.url || ''),
      secret: String(item.secret || ''),
      enabled: item.enabled !== false,
      events: Array.isArray(item.events) ? item.events.map((event) => String(event)) : []
    }))
  }

  async function loadData() {
    if (!canView.value) {
      return
    }

    loading.value = true

    try {
      const payload = (await fetchRobotConfig()) as RobotConfigRecord
      webhooks.value = normalizeWebhooks(payload.webhooks)
      initialized.value = true
    } finally {
      loading.value = false
    }
  }

  function addWebhook() {
    webhooks.value.push({
      _key: createKey(),
      name: `Webhook ${webhooks.value.length + 1}`,
      url: '',
      secret: '',
      enabled: true,
      events: []
    })
  }

  function removeWebhook(index: number) {
    webhooks.value.splice(index, 1)
  }

  async function handleSave() {
    saving.value = true

    try {
      await updateRobotConfig({
        webhooks: webhooks.value.map((item) => ({
          name: String(item.name || '').trim() || 'Webhook',
          url: String(item.url || '').trim(),
          secret: String(item.secret || ''),
          enabled: Boolean(item.enabled),
          events: Array.isArray(item.events)
            ? item.events.map((event) => String(event).trim()).filter(Boolean)
            : []
        }))
      })

      ElMessage.success('Webhook settings saved')
    } catch (error: any) {
      ElMessage.error(error?.response?.data?.error || 'Save failed')
    } finally {
      saving.value = false
    }
  }

  async function handleTestWebhook() {
    testLoading.value = true

    try {
      await testRobotWebhook({
        event: 'webhook.test',
        data: {
          text: 'Test Webhook',
          sender: 'console',
          timestamp: Math.floor(Date.now() / 1000)
        }
      })

      ElMessage.success('Test request sent')
    } catch (error: any) {
      ElMessage.error(error?.response?.data?.error || 'Test request failed')
    } finally {
      testLoading.value = false
    }
  }
</script>

<style scoped lang="scss">
  .webhook-settings-page {
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

  .page-subtitle {
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

  .section-title {
    color: var(--el-text-color-primary);
    font-size: 16px;
    font-weight: 700;
  }

  .full-width {
    width: 100%;
  }

  .empty-state {
    padding-top: 16px;
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
