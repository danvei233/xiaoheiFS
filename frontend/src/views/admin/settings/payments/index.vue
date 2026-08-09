<template>
  <div class="payments-page art-full-height">
    <ElCard class="art-table-card">
      <ArtTableHeader v-model:columns="columnChecks" :loading="loading" @refresh="refreshData" />

      <ArtTable row-key="key" :loading="loading" :data="data" :columns="columns">
        <template #type="{ row }">
          <ElTag :type="row.type === 'plugin' ? 'primary' : 'info'">
            {{ row.type === 'plugin' ? '插件' : '内置' }}
          </ElTag>
        </template>
        <template #order_enabled="{ row }">
          <ElSwitch
            :model-value="row.order_enabled"
            :loading="row.busy_order"
            :disabled="!canUpdate"
            @change="(value: boolean) => toggle(row, 'order', value)"
          />
        </template>
        <template #wallet_enabled="{ row }">
          <ElSwitch
            :model-value="row.wallet_enabled"
            :loading="row.busy_wallet"
            :disabled="!canUpdate"
            @change="(value: boolean) => toggle(row, 'wallet', value)"
          />
        </template>
      </ArtTable>
    </ElCard>
  </div>
</template>

<script setup lang="ts">
  import {
    listAdminPaymentProviders,
    listAdminPlugins,
    updateAdminPaymentProvider
  } from '@/services/admin'
  import type { PaymentProvider, PluginListItem } from '@/services/types'
  import { useTable } from '@/hooks/core/useTable'
  import { useAdminPermissions } from '../_shared/settings'

  defineOptions({ name: 'AdminSettingsPayments' })

  interface PaymentRow {
    type: 'builtin' | 'plugin'
    key: string
    provider_key: string
    name: string
    order_enabled: boolean
    wallet_enabled: boolean
    busy_order: boolean
    busy_wallet: boolean
  }

  const { hasPermission } = useAdminPermissions()
  const canUpdate = hasPermission('payment.update')
  const canListPlugins = hasPermission('plugin.list')

  const fetchPaymentRows = async () => {
    const rows = await buildRows()
    return { records: rows, total: rows.length }
  }

  const { columns, columnChecks, data, loading, refreshData, refreshUpdate } = useTable({
    core: {
      apiFn: fetchPaymentRows,
      columnsFactory: () => [
        { prop: 'type', label: '类型', width: 100, useSlot: true },
        { prop: 'name', label: '名称', minWidth: 220, showOverflowTooltip: true },
        { prop: 'key', label: 'Key', minWidth: 240, showOverflowTooltip: true },
        { prop: 'order_enabled', label: '订单支付', width: 120, useSlot: true },
        { prop: 'wallet_enabled', label: '钱包充值', width: 120, useSlot: true }
      ]
    }
  })

  function manifestMethods(plugin: PluginListItem): string[] {
    return [...new Set(plugin.manifest?.capabilities?.payment?.methods?.map(String) ?? [])].filter(
      Boolean
    )
  }

  function responseStatus(error: unknown): number | undefined {
    if (!error || typeof error !== 'object' || !('response' in error)) return undefined
    const response = error.response
    if (!response || typeof response !== 'object' || !('status' in response)) return undefined
    const status = Number(response.status)
    return Number.isFinite(status) ? status : undefined
  }

  async function loadPlugins(): Promise<PluginListItem[]> {
    if (!canListPlugins.value) return []
    try {
      const response = await listAdminPlugins()
      return response.data?.items ?? []
    } catch (error) {
      if (responseStatus(error) === 403) return []
      throw error
    }
  }

  async function buildRows(): Promise<PaymentRow[]> {
    const [providerResponse, plugins] = await Promise.all([
      listAdminPaymentProviders({ include_disabled: true, include_legacy: false }),
      loadPlugins()
    ])
    const providers = providerResponse.data?.items ?? []
    const stateMap = new Map<string, PaymentProvider>()
    providers.forEach((provider) => {
      const key = String(provider.key ?? '')
      if (key) stateMap.set(key, provider)
    })

    const builtinRows: PaymentRow[] = providers
      .filter((provider) => {
        const key = String(provider.key ?? '').toLowerCase()
        return Boolean(key) && key !== 'yipay' && key !== 'custom' && !key.includes('.')
      })
      .map((provider) => ({
        type: 'builtin',
        key: String(provider.key),
        provider_key: String(provider.key),
        name: String(provider.name ?? provider.key ?? ''),
        order_enabled: provider.order_enabled !== false,
        wallet_enabled: provider.wallet_enabled !== false,
        busy_order: false,
        busy_wallet: false
      }))

    const paymentPlugins = plugins.filter(
      (plugin) =>
        plugin.enabled &&
        plugin.loaded &&
        plugin.category === 'payment' &&
        manifestMethods(plugin).length > 0
    )
    const pluginRows: PaymentRow[] = []
    paymentPlugins.forEach((plugin) => {
      const pluginId = String(plugin.plugin_id ?? '')
      const instanceId = String(plugin.instance_id ?? 'default')
      manifestMethods(plugin).forEach((method) => {
        const providerKey = `${pluginId}.${method}`
        const provider = stateMap.get(providerKey)
        pluginRows.push({
          type: 'plugin',
          key: `${pluginId}.${instanceId}.${method}`,
          provider_key: providerKey,
          name: `${String(plugin.name ?? pluginId)} / ${method}`,
          order_enabled: provider?.order_enabled !== false,
          wallet_enabled: provider?.wallet_enabled !== false,
          busy_order: false,
          busy_wallet: false
        })
      })
    })
    return [...builtinRows, ...pluginRows].sort((left, right) => left.key.localeCompare(right.key))
  }

  async function toggle(
    row: PaymentRow,
    scene: 'order' | 'wallet',
    enabled: boolean
  ): Promise<void> {
    if (scene === 'order') row.busy_order = true
    else row.busy_wallet = true
    try {
      await updateAdminPaymentProvider(row.provider_key, { scene, enabled })
      ElMessage.success('支付场景状态已更新')
      await refreshUpdate()
    } finally {
      row.busy_order = false
      row.busy_wallet = false
    }
  }
</script>

<style lang="scss" scoped>
  .payments-page {
    min-width: 0;
  }
</style>
