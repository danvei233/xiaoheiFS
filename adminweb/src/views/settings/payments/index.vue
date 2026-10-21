<template>
  <div v-loading="loading" class="payments-settings-page art-full-height">
    <div class="page-header">
      <div>
        <div class="page-kicker">Billing</div>
        <div class="page-title">Payment Settings</div>
        <div class="page-subtitle">
          Configure which payment providers can be used for order checkout and wallet top-up.
        </div>
      </div>

      <div class="page-actions">
        <ElButton v-if="canView" :disabled="loading" @click="fetchData">Refresh</ElButton>
      </div>
    </div>

    <ElEmpty v-if="!canView" description="You do not have permission to view payment settings." />

    <template v-else>
      <ElAlert
        type="info"
        :closable="false"
        show-icon
        title="How this page works"
        description="Built-in providers and payment plugin methods are shown together here. Each scene switch is managed independently."
      />

      <ElCard shadow="never" class="section-card">
        <template #header>
          <div class="section-title">Provider Matrix</div>
        </template>

        <ElTable :data="rows" border row-key="key">
          <ElTableColumn label="Type" width="110">
            <template #default="{ row }">
              <ElTag :type="row.type === 'plugin' ? 'warning' : 'info'">
                {{ row.type === 'plugin' ? 'Plugin' : 'Built-in' }}
              </ElTag>
            </template>
          </ElTableColumn>

          <ElTableColumn prop="name" label="Name" min-width="240" />
          <ElTableColumn prop="key" label="Key" min-width="240" show-overflow-tooltip />

          <ElTableColumn label="Order Checkout" width="150">
            <template #default="{ row }">
              <ElSwitch
                :model-value="row.order_enabled"
                :loading="row.busy_order"
                :disabled="!canUpdate"
                @change="handleToggle(row, 'order', $event)"
              />
            </template>
          </ElTableColumn>

          <ElTableColumn label="Wallet Top-up" width="150">
            <template #default="{ row }">
              <ElSwitch
                :model-value="row.wallet_enabled"
                :loading="row.busy_wallet"
                :disabled="!canUpdate"
                @change="handleToggle(row, 'wallet', $event)"
              />
            </template>
          </ElTableColumn>
        </ElTable>
      </ElCard>
    </template>
  </div>
</template>

<script setup lang="ts">
  import type {
    AdminPluginRecord,
    PaymentProviderRecord,
    PluginPaymentMethodRecord
  } from '@/api/admin'
  import {
    fetchAdminPaymentProviders,
    fetchAdminPluginPaymentMethods,
    fetchAdminPlugins,
    hasAdminPermission,
    updateAdminPaymentProvider
  } from '@/api/admin'
  import { useUserStore } from '@/store/modules/user'
  import { ElMessage } from 'element-plus'

  defineOptions({ name: 'SettingsPaymentsPage' })

  type ToggleScene = 'order' | 'wallet'

  interface BaseRow {
    type: 'builtin' | 'plugin'
    key: string
    provider_key: string
    name: string
    enabled: boolean
    order_enabled: boolean
    wallet_enabled: boolean
    busy_order?: boolean
    busy_wallet?: boolean
  }

  interface PluginRow extends BaseRow {
    type: 'plugin'
    plugin_id: string
    instance_id: string
    method: string
  }

  type RowItem = BaseRow | PluginRow

  const loading = ref(false)
  const rows = ref<RowItem[]>([])

  const userStore = useUserStore()
  const { info } = storeToRefs(userStore)

  const canView = computed(() => hasAdminPermission(info.value?.buttons, ['settings.view']))
  const canUpdate = computed(() => hasAdminPermission(info.value?.buttons, ['settings.update']))

  watch(
    canView,
    (value) => {
      if (value && !rows.value.length) {
        fetchData()
      }
    },
    { immediate: true }
  )

  async function pluginMethodsMap(plugin: AdminPluginRecord) {
    const category = String(plugin.category || 'payment').trim() || 'payment'
    const pluginId = String(plugin.plugin_id || '').trim()
    const instanceId = String(plugin.instance_id || 'default').trim() || 'default'

    if (!pluginId) {
      return new Map<string, boolean>()
    }

    const payload = await fetchAdminPluginPaymentMethods({
      category,
      plugin_id: pluginId,
      instance_id: instanceId
    })

    const out = new Map<string, boolean>()
    ;(payload.items || []).forEach((item: PluginPaymentMethodRecord) => {
      const method = String(item.method || '').trim()
      if (!method) {
        return
      }
      out.set(method, Boolean(item.enabled))
    })

    return out
  }

  function pluginMethodsFromManifest(plugin: AdminPluginRecord) {
    const methods = plugin.manifest?.capabilities?.payment?.methods || []
    const uniqueMethods = new Set<string>()

    methods.forEach((method) => {
      const value = String(method || '').trim()
      if (value) {
        uniqueMethods.add(value)
      }
    })

    return Array.from(uniqueMethods)
  }

  async function buildRows() {
    const [providersPayload, pluginsPayload] = await Promise.all([
      fetchAdminPaymentProviders({ include_disabled: true, include_legacy: false }),
      fetchAdminPlugins()
    ])

    const providers = (providersPayload.items || []) as PaymentProviderRecord[]
    const plugins = (pluginsPayload.items || []) as AdminPluginRecord[]

    const providerStateMap = new Map<
      string,
      { enabled: boolean; order_enabled: boolean; wallet_enabled: boolean }
    >()

    providers.forEach((provider) => {
      const key = String(provider.key || '').trim()
      if (!key) {
        return
      }

      providerStateMap.set(key, {
        enabled: Boolean(provider.enabled),
        order_enabled: provider.order_enabled !== false,
        wallet_enabled: provider.wallet_enabled !== false
      })
    })

    const builtinRows: RowItem[] = providers
      .filter((provider) => {
        const key = String(provider.key || '')
          .trim()
          .toLowerCase()

        if (!key || key === 'yipay' || key === 'custom') {
          return false
        }

        return !key.includes('.')
      })
      .map((provider) => ({
        type: 'builtin',
        key: String(provider.key || ''),
        provider_key: String(provider.key || ''),
        name: String(provider.name || provider.key || ''),
        enabled: Boolean(provider.enabled),
        order_enabled: provider.order_enabled !== false,
        wallet_enabled: provider.wallet_enabled !== false,
        busy_order: false,
        busy_wallet: false
      }))

    const enabledPaymentPlugins = plugins.filter((plugin) => {
      if (!plugin.enabled || !plugin.loaded) {
        return false
      }

      if (String(plugin.category || '').trim() !== 'payment') {
        return false
      }

      return pluginMethodsFromManifest(plugin).length > 0
    })

    const methodStateList = await Promise.all(
      enabledPaymentPlugins.map((plugin) => pluginMethodsMap(plugin))
    )

    const pluginRows: RowItem[] = []

    enabledPaymentPlugins.forEach((plugin, index) => {
      const pluginId = String(plugin.plugin_id || '').trim()
      const instanceId = String(plugin.instance_id || 'default').trim() || 'default'
      const methods = pluginMethodsFromManifest(plugin)
      const enabledMap = methodStateList[index]

      methods.forEach((method) => {
        pluginRows.push({
          type: 'plugin',
          key: `${pluginId}.${instanceId}.${method}`,
          provider_key: `${pluginId}.${method}`,
          name: `${String(plugin.name || pluginId)} / ${method}`,
          plugin_id: pluginId,
          instance_id: instanceId,
          method,
          enabled: enabledMap.has(method) ? Boolean(enabledMap.get(method)) : true,
          order_enabled: providerStateMap.get(`${pluginId}.${method}`)?.order_enabled ?? true,
          wallet_enabled: providerStateMap.get(`${pluginId}.${method}`)?.wallet_enabled ?? true,
          busy_order: false,
          busy_wallet: false
        })
      })
    })

    rows.value = [...builtinRows, ...pluginRows].sort((left, right) =>
      left.key.localeCompare(right.key)
    )
  }

  async function fetchData() {
    if (!canView.value) {
      return
    }

    loading.value = true

    try {
      await buildRows()
    } finally {
      loading.value = false
    }
  }

  async function handleToggle(
    row: RowItem,
    scene: ToggleScene,
    checked: boolean | string | number
  ) {
    const enabled = Boolean(checked)
    if (scene === 'order') {
      row.busy_order = true
    } else {
      row.busy_wallet = true
    }

    try {
      await updateAdminPaymentProvider(row.provider_key, {
        scene,
        enabled
      })

      if (scene === 'order') {
        row.order_enabled = enabled
      } else {
        row.wallet_enabled = enabled
      }

      ElMessage.success('Updated successfully')
    } catch (error: any) {
      ElMessage.error(error?.response?.data?.error || 'Update failed')
    } finally {
      if (scene === 'order') {
        row.busy_order = false
      } else {
        row.busy_wallet = false
      }
    }
  }
</script>

<style scoped lang="scss">
  .payments-settings-page {
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
