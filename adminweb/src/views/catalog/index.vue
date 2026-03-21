<template>
  <div v-loading="loading" class="catalog-page art-full-height">
    <ElCard shadow="never">
      <div class="page-header">
        <div>
          <div class="page-title">Catalog</div>
          <div class="page-subtitle">管理商品类型、地区、线路、套餐、镜像和计费周期。</div>
        </div>

        <div class="page-actions">
          <ElSelect
            v-model="goodsTypeId"
            clearable
            filterable
            placeholder="选择商品类型"
            class="goods-type-select"
            @change="handleGoodsTypeChange"
          >
            <ElOption
              v-for="item in goodsTypeOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </ElSelect>
          <ElButton :disabled="!goodsTypeId" @click="syncCurrentGoodsType">同步当前类型</ElButton>
          <ElButton @click="initializePage">刷新</ElButton>
        </div>
      </div>

      <ElEmpty v-if="!canView" description="You do not have permission to view catalog." />

      <template v-else>
        <ElAlert
          v-if="selectedGoodsType"
          type="info"
          show-icon
          :closable="false"
          :title="`当前商品类型：${selectedGoodsType.name || '-'} / ${selectedGoodsType.code || '-'}`"
          class="page-alert"
        />

        <ElTabs v-model="activeTab">
          <ElTabPane name="goods-types" label="商品类型">
            <div class="toolbar">
              <div class="toolbar-actions">
                <ElButton
                  v-if="hasPermission('goods_type.create')"
                  type="primary"
                  @click="openGoodsTypeDialog()"
                >
                  新增商品类型
                </ElButton>
              </div>
            </div>

            <ElTable :data="goodsTypes" border row-key="id">
              <ElTableColumn prop="id" label="ID" width="90" />
              <ElTableColumn prop="name" label="名称" min-width="180" />
              <ElTableColumn prop="code" label="代码" min-width="150" />
              <ElTableColumn label="自动化实例" min-width="220">
                <template #default="{ row }">{{ formatAutomationBinding(row) }}</template>
              </ElTableColumn>
              <ElTableColumn prop="sort_order" label="排序" width="90" />
              <ElTableColumn label="状态" width="100">
                <template #default="{ row }">
                  <ElTag :type="row.active ? 'success' : 'danger'">{{
                    row.active ? '启用' : '停用'
                  }}</ElTag>
                </template>
              </ElTableColumn>
              <ElTableColumn label="操作" width="220" fixed="right">
                <template #default="{ row }">
                  <div class="table-actions">
                    <ElButton
                      v-if="hasPermission('goods_type.update')"
                      link
                      type="primary"
                      @click="openGoodsTypeDialog(row)"
                      >编辑</ElButton
                    >
                    <ElButton
                      v-if="hasPermission('goods_type.sync')"
                      link
                      type="primary"
                      @click="syncGoodsType(row)"
                      >同步</ElButton
                    >
                    <ElButton
                      v-if="hasPermission('goods_type.delete')"
                      link
                      type="danger"
                      @click="removeGoodsType(row)"
                      >删除</ElButton
                    >
                  </div>
                </template>
              </ElTableColumn>
            </ElTable>
          </ElTabPane>

          <ElTabPane name="regions" label="地区">
            <ElAlert
              v-if="isCatalogReadonly"
              type="info"
              show-icon
              :closable="false"
              title="当前自动化插件声明目录只读，地区由插件同步，不允许手动增删改。"
              class="tab-alert"
            />

            <div class="toolbar">
              <div class="toolbar-actions">
                <ElButton
                  v-if="
                    !isCatalogReadonly && hasPermission(['region.delete', 'region.bulk_delete'])
                  "
                  :disabled="!selectedRegionIds.length"
                  type="danger"
                  @click="removeSelectedRegions"
                >
                  批量删除
                </ElButton>
                <ElButton
                  v-if="!isCatalogReadonly && hasPermission('region.create')"
                  type="primary"
                  :disabled="!goodsTypeId"
                  @click="openRegionDialog()"
                >
                  新增地区
                </ElButton>
              </div>
            </div>

            <ElTable
              :data="regions"
              border
              row-key="id"
              @selection-change="onRegionSelectionChange"
            >
              <ElTableColumn
                v-if="!isCatalogReadonly && hasPermission(['region.delete', 'region.bulk_delete'])"
                type="selection"
                width="48"
              />
              <ElTableColumn prop="id" label="ID" width="90" />
              <ElTableColumn prop="name" label="名称" min-width="180" />
              <ElTableColumn prop="code" label="代码" min-width="140" />
              <ElTableColumn label="状态" width="100">
                <template #default="{ row }">
                  <ElTag :type="row.active ? 'success' : 'danger'">{{
                    row.active ? '启用' : '停用'
                  }}</ElTag>
                </template>
              </ElTableColumn>
              <ElTableColumn label="操作" width="150" fixed="right">
                <template #default="{ row }">
                  <div class="table-actions">
                    <ElButton
                      v-if="!isCatalogReadonly && hasPermission('region.update')"
                      link
                      type="primary"
                      @click="openRegionDialog(row)"
                      >编辑</ElButton
                    >
                    <ElButton
                      v-if="!isCatalogReadonly && hasPermission('region.delete')"
                      link
                      type="danger"
                      @click="removeRegion(row)"
                      >删除</ElButton
                    >
                  </div>
                </template>
              </ElTableColumn>
            </ElTable>
          </ElTabPane>

          <ElTabPane name="plan-groups" label="线路 / 附加项">
            <ElAlert
              v-if="isCatalogReadonly"
              type="info"
              show-icon
              :closable="false"
              title="当前自动化插件声明目录只读，线路由插件同步，仅允许启用或停用。"
              class="tab-alert"
            />

            <div class="toolbar">
              <div class="toolbar-actions">
                <ElButton
                  v-if="
                    !isCatalogReadonly &&
                    hasPermission(['plan_group.delete', 'plan_group.bulk_delete'])
                  "
                  :disabled="!selectedPlanGroupIds.length"
                  type="danger"
                  @click="removeSelectedPlanGroups"
                >
                  批量删除
                </ElButton>
                <ElButton
                  v-if="!isCatalogReadonly && hasPermission('plan_group.create')"
                  type="primary"
                  :disabled="!goodsTypeId"
                  @click="openPlanGroupDialog()"
                >
                  新增线路
                </ElButton>
              </div>
            </div>

            <ElTable
              :data="planGroups"
              border
              row-key="id"
              @selection-change="onPlanGroupSelectionChange"
            >
              <ElTableColumn
                v-if="
                  !isCatalogReadonly &&
                  hasPermission(['plan_group.delete', 'plan_group.bulk_delete'])
                "
                type="selection"
                width="48"
              />
              <ElTableColumn prop="id" label="ID" width="90" />
              <ElTableColumn label="地区" min-width="160">
                <template #default="{ row }">{{ getRegionName(row.region_id) }}</template>
              </ElTableColumn>
              <ElTableColumn prop="name" label="名称" min-width="180" />
              <ElTableColumn prop="line_id" label="云线路 ID" width="120" />
              <ElTableColumn label="可见" width="90">
                <template #default="{ row }"
                  ><ElTag :type="row.visible ? 'success' : 'info'">{{
                    row.visible ? '可见' : '隐藏'
                  }}</ElTag></template
                >
              </ElTableColumn>
              <ElTableColumn label="余量" width="100">
                <template #default="{ row }"
                  ><ElTag :type="capacityTagType(row.capacity_remaining)">{{
                    formatCapacity(row.capacity_remaining)
                  }}</ElTag></template
                >
              </ElTableColumn>
              <ElTableColumn label="状态" width="100">
                <template #default="{ row }"
                  ><ElTag :type="row.active ? 'success' : 'danger'">{{
                    row.active ? '启用' : '停用'
                  }}</ElTag></template
                >
              </ElTableColumn>
              <ElTableColumn label="操作" width="220" fixed="right">
                <template #default="{ row }">
                  <div class="table-actions">
                    <ElButton
                      v-if="
                        isCatalogReadonly &&
                        hasPermission(['plan_group.update', 'plan_group.set_system_images'])
                      "
                      link
                      :type="row.active ? 'danger' : 'primary'"
                      @click="togglePlanGroupActive(row)"
                    >
                      {{ row.active ? '禁用' : '启用' }}
                    </ElButton>
                    <template v-else>
                      <ElButton
                        v-if="hasPermission(['plan_group.update', 'plan_group.set_system_images'])"
                        link
                        type="primary"
                        @click="openPlanGroupDialog(row)"
                        >编辑</ElButton
                      >
                      <ElButton
                        v-if="hasPermission('plan_group.delete')"
                        link
                        type="danger"
                        @click="removePlanGroup(row)"
                        >删除</ElButton
                      >
                    </template>
                  </div>
                </template>
              </ElTableColumn>
            </ElTable>
          </ElTabPane>

          <ElTabPane name="packages" label="套餐">
            <div class="toolbar">
              <div class="toolbar-filters">
                <ElSelect
                  v-model="packagePlanGroupFilter"
                  clearable
                  placeholder="筛选线路"
                  class="inline-select"
                >
                  <ElOption label="全部线路" value="all" />
                  <ElOption
                    v-for="item in planGroupOptions"
                    :key="item.value"
                    :label="item.label"
                    :value="item.value"
                  />
                </ElSelect>
              </div>
              <div class="toolbar-actions">
                <ElButton
                  v-if="hasPermission(['package.delete', 'package.bulk_delete'])"
                  :disabled="!selectedPackageIds.length"
                  type="danger"
                  @click="removeSelectedPackages"
                  >批量删除</ElButton
                >
                <ElButton
                  v-if="hasPermission('package.create')"
                  type="primary"
                  :disabled="!planGroups.length"
                  @click="openPackageDialog()"
                  >新增套餐</ElButton
                >
                <ElButton
                  v-if="hasPermission('package.create')"
                  :disabled="!planGroups.length"
                  @click="openPackageBatchDialog"
                >
                  批量生成
                </ElButton>
              </div>
            </div>

            <ElTable
              :data="filteredPackages"
              border
              row-key="id"
              @selection-change="onPackageSelectionChange"
            >
              <ElTableColumn
                v-if="hasPermission(['package.delete', 'package.bulk_delete'])"
                type="selection"
                width="48"
              />
              <ElTableColumn prop="id" label="ID" width="90" />
              <ElTableColumn prop="name" label="名称" min-width="180" />
              <ElTableColumn label="线路" min-width="160"
                ><template #default="{ row }">{{
                  getPlanGroupName(row.plan_group_id)
                }}</template></ElTableColumn
              >
              <ElTableColumn label="规格" min-width="220"
                ><template #default="{ row }"
                  >{{ row.cores }}C / {{ row.memory_gb }}G / {{ row.disk_gb }}G /
                  {{ row.bandwidth_mbps }}M</template
                ></ElTableColumn
              >
              <ElTableColumn prop="monthly_price" label="月费" width="120" />
              <ElTableColumn prop="port_num" label="端口数" width="90" />
              <ElTableColumn label="状态" width="100"
                ><template #default="{ row }"
                  ><ElTag :type="row.active ? 'success' : 'danger'">{{
                    row.active ? '启用' : '停用'
                  }}</ElTag></template
                ></ElTableColumn
              >
              <ElTableColumn label="操作" width="150" fixed="right">
                <template #default="{ row }">
                  <div class="table-actions">
                    <ElButton
                      v-if="hasPermission('package.update')"
                      link
                      type="primary"
                      @click="openPackageDialog(row)"
                      >编辑</ElButton
                    >
                    <ElButton
                      v-if="hasPermission('package.delete')"
                      link
                      type="danger"
                      @click="removePackage(row)"
                      >删除</ElButton
                    >
                  </div>
                </template>
              </ElTableColumn>
            </ElTable>
          </ElTabPane>

          <ElTabPane name="images" label="系统镜像">
            <div class="toolbar">
              <div class="toolbar-filters">
                <ElSelect
                  v-model="imagePlanGroupFilter"
                  clearable
                  placeholder="选择同步线路"
                  class="inline-select"
                  @change="refreshScopedImages"
                >
                  <ElOption
                    v-for="item in planGroupOptions"
                    :key="item.value"
                    :label="item.label"
                    :value="item.value"
                  />
                </ElSelect>
              </div>
              <div class="toolbar-actions">
                <ElButton
                  v-if="hasPermission(['system_image.delete', 'system_image.bulk_delete'])"
                  :disabled="!selectedSystemImageIds.length"
                  type="danger"
                  @click="removeSelectedSystemImages"
                  >批量删除</ElButton
                >
                <ElButton
                  v-if="hasPermission('system_image.sync')"
                  :disabled="!imagePlanGroupFilter"
                  @click="syncSystemImages"
                  >同步镜像</ElButton
                >
                <ElButton
                  v-if="hasPermission('system_image.create')"
                  type="primary"
                  @click="openSystemImageDialog()"
                  >新增镜像</ElButton
                >
              </div>
            </div>

            <ElTable
              :data="displaySystemImages"
              border
              row-key="id"
              @selection-change="onSystemImageSelectionChange"
            >
              <ElTableColumn
                v-if="hasPermission(['system_image.delete', 'system_image.bulk_delete'])"
                type="selection"
                width="48"
              />
              <ElTableColumn prop="id" label="ID" width="90" />
              <ElTableColumn prop="image_id" label="镜像 ID" width="120" />
              <ElTableColumn prop="name" label="名称" min-width="220" />
              <ElTableColumn label="类型" width="120"
                ><template #default="{ row }"
                  ><ElTag :type="imageTypeTagType(row.type)">{{
                    formatImageType(row.type)
                  }}</ElTag></template
                ></ElTableColumn
              >
              <ElTableColumn label="状态" width="100"
                ><template #default="{ row }"
                  ><ElTag :type="row.enabled ? 'success' : 'danger'">{{
                    row.enabled ? '启用' : '停用'
                  }}</ElTag></template
                ></ElTableColumn
              >
              <ElTableColumn label="操作" width="150" fixed="right">
                <template #default="{ row }">
                  <div class="table-actions">
                    <ElButton
                      v-if="hasPermission('system_image.update')"
                      link
                      type="primary"
                      @click="openSystemImageDialog(row)"
                      >编辑</ElButton
                    >
                    <ElButton
                      v-if="hasPermission('system_image.delete')"
                      link
                      type="danger"
                      @click="removeSystemImage(row)"
                      >删除</ElButton
                    >
                  </div>
                </template>
              </ElTableColumn>
            </ElTable>
          </ElTabPane>

          <ElTabPane name="billing-cycles" label="计费周期">
            <div class="toolbar">
              <div class="toolbar-actions">
                <ElButton
                  v-if="hasPermission(['billing_cycle.delete', 'billing_cycle.bulk_delete'])"
                  :disabled="!selectedBillingCycleIds.length"
                  type="danger"
                  @click="removeSelectedBillingCycles"
                  >批量删除</ElButton
                >
                <ElButton
                  v-if="hasPermission('billing_cycle.create')"
                  type="primary"
                  @click="openBillingCycleDialog()"
                  >新增周期</ElButton
                >
              </div>
            </div>

            <ElTable
              :data="billingCycles"
              border
              row-key="id"
              @selection-change="onBillingCycleSelectionChange"
            >
              <ElTableColumn
                v-if="hasPermission(['billing_cycle.delete', 'billing_cycle.bulk_delete'])"
                type="selection"
                width="48"
              />
              <ElTableColumn prop="id" label="ID" width="90" />
              <ElTableColumn prop="name" label="名称" min-width="180" />
              <ElTableColumn prop="months" label="月数" width="100" />
              <ElTableColumn prop="multiplier" label="倍率" width="100" />
              <ElTableColumn prop="min_qty" label="最小数量" width="100" />
              <ElTableColumn prop="max_qty" label="最大数量" width="100" />
              <ElTableColumn label="状态" width="100"
                ><template #default="{ row }"
                  ><ElTag :type="row.active ? 'success' : 'info'">{{
                    row.active ? '启用' : '停用'
                  }}</ElTag></template
                ></ElTableColumn
              >
              <ElTableColumn label="操作" width="150" fixed="right">
                <template #default="{ row }">
                  <div class="table-actions">
                    <ElButton
                      v-if="hasPermission('billing_cycle.update')"
                      link
                      type="primary"
                      @click="openBillingCycleDialog(row)"
                      >编辑</ElButton
                    >
                    <ElButton
                      v-if="hasPermission('billing_cycle.delete')"
                      link
                      type="danger"
                      @click="removeBillingCycle(row)"
                      >删除</ElButton
                    >
                  </div>
                </template>
              </ElTableColumn>
            </ElTable>
          </ElTabPane>
        </ElTabs>
      </template>
    </ElCard>

    <GoodsTypeDialog
      v-model:visible="goodsTypeDialogVisible"
      :form-data="goodsTypeForm"
      :automation-options="automationOptions"
      :schema-json="automationConfigSchemaJson"
      :config-json="automationConfigJson"
      :config-error="automationConfigError"
      :config-loading="automationConfigLoading"
      :submitting="goodsTypeSubmitting"
      @update:configJson="handleAutomationConfigJsonChange"
      @automation-change="handleGoodsTypeAutomationChange"
      @submit="submitGoodsType"
    />
    <RegionDialog
      v-model:visible="regionDialogVisible"
      :form-data="regionForm"
      :submitting="regionSubmitting"
      @submit="submitRegion"
    />
    <PlanGroupDrawer
      v-model:visible="planGroupDrawerVisible"
      :form-data="planGroupForm"
      :region-options="regionOptions"
      :system-image-options="systemImageOptions"
      :submitting="planGroupSubmitting"
      @submit="submitPlanGroup"
    />
    <PackageDialog
      v-model:visible="packageDialogVisible"
      :form-data="packageForm"
      :plan-group-options="planGroupOptions"
      :submitting="packageSubmitting"
      @submit="submitPackage"
    />
    <PackageBatchDialog
      v-model:visible="packageBatchDialogVisible"
      :model-value="batchForm"
      :line-options="planGroupOptions"
      :generated-rows="generatedPackages"
      @update:modelValue="handleBatchFormUpdate"
      @generate="generateBatchPackages"
      @clear="clearGeneratedPackages"
      @apply="applyGeneratedPackages"
    />
    <SystemImageDialog
      v-model:visible="systemImageDialogVisible"
      :form-data="systemImageForm"
      :submitting="systemImageSubmitting"
      @submit="submitSystemImage"
    />
    <BillingCycleDialog
      v-model:visible="billingCycleDialogVisible"
      :form-data="billingCycleForm"
      :submitting="billingCycleSubmitting"
      @submit="submitBillingCycle"
    />
  </div>
</template>

<script setup lang="ts">
  import type { AdminPluginRecord, CatalogGoodsType, CatalogSystemImage } from '@/api/admin'
  import {
    createDefaultSystemImageDialogForm
  } from '@/components/business/system-image-dialog/model'
  import type {
    SystemImageDialogFormValue as SystemImageFormValue
  } from '@/components/business/system-image-dialog/model'
  import {
    bulkDeleteAdminBillingCycles,
    bulkDeleteAdminPackages,
    bulkDeleteAdminPlanGroups,
    bulkDeleteAdminRegions,
    bulkDeleteAdminSystemImages,
    createAdminBillingCycle,
    createAdminGoodsType,
    createAdminPackage,
    createAdminPlanGroup,
    createAdminRegion,
    createAdminSystemImage,
    deleteAdminBillingCycle,
    deleteAdminGoodsType,
    deleteAdminPackage,
    deleteAdminPlanGroup,
    deleteAdminRegion,
    deleteAdminSystemImage,
    fetchAdminBillingCycles,
    fetchAdminGoodsTypeCapabilities,
    fetchAdminGoodsTypes,
    fetchAdminPackages,
    fetchAdminPluginInstanceConfig,
    fetchAdminPluginInstanceConfigSchema,
    fetchAdminPlanGroups,
    fetchAdminPlugins,
    fetchAdminRegions,
    fetchAdminSystemImages,
    hasAdminPermission,
    setAdminPlanGroupSystemImages,
    syncAdminGoodsTypeAutomation,
    syncAdminSystemImages,
    updateAdminBillingCycle,
    updateAdminGoodsType,
    updateAdminGoodsTypeCapabilities,
    updateAdminPluginInstanceConfig,
    updateAdminPackage,
    updateAdminPlanGroup,
    updateAdminRegion,
    updateAdminSystemImage
  } from '@/api/admin'
  import { useUserStore } from '@/store/modules/user'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import BillingCycleDialog from './modules/billing-cycle-dialog.vue'
  import GoodsTypeDialog from './modules/goods-type-dialog.vue'
  import PackageBatchDialog from './modules/package-batch-dialog.vue'
  import PackageDialog from './modules/package-dialog.vue'
  import PlanGroupDrawer from './modules/plan-group-drawer.vue'
  import RegionDialog from './modules/region-dialog.vue'
  import SystemImageDialog from './modules/system-image-dialog.vue'

  defineOptions({ name: 'CatalogPage' })

  interface GoodsTypeRow {
    id: number | null
    code: string
    name: string
    active: boolean
    sort_order: number
    automation_plugin_id: string
    automation_instance_id: string
  }

  interface RegionRow {
    id: number | null
    goods_type_id: number | null
    name: string
    code: string
    active: boolean
  }

  interface PlanGroupRow {
    id: number | null
    goods_type_id: number | null
    region_id: number | null
    line_id: number | null
    name: string
    unit_core: number
    unit_mem: number
    unit_disk: number
    unit_bw: number
    add_core_min: number
    add_core_max: number
    add_core_step: number
    add_mem_min: number
    add_mem_max: number
    add_mem_step: number
    add_disk_min: number
    add_disk_max: number
    add_disk_step: number
    add_bw_min: number
    add_bw_max: number
    add_bw_step: number
    active: boolean
    visible: boolean
    capacity_remaining: number
    sort_order: number
  }

  interface PackageRow {
    id: number | null
    plan_group_id: number | null
    name: string
    cores: number
    memory_gb: number
    disk_gb: number
    bandwidth_mbps: number
    cpu_model: string
    port_num: number
    monthly_price: number
    active: boolean
    visible: boolean
    capacity_remaining: number
    sort_order: number
  }

  interface SystemImageRow {
    id: number | null
    image_id: number | null
    name: string
    type: string
    enabled: boolean
  }

  interface BillingCycleRow {
    id: number | null
    name: string
    months: number
    multiplier: number
    min_qty: number
    max_qty: number
    active: boolean
    sort_order: number
  }

  type GoodsTypeFormValue = {
    id: number | null
    code: string
    name: string
    active: boolean
    sort_order: number
    automation_plugin_id: string
    automation_instance_id: string
    resize_enabled: boolean
    refund_enabled: boolean
  }

  type RegionFormValue = {
    id: number | null
    goods_type_id: number | null
    name: string
    code: string
    active: boolean
  }

  type PlanGroupFormValue = {
    id: number | null
    region_id: number | null
    name: string
    line_id: number | null
    unit_core: number
    unit_mem: number
    unit_disk: number
    unit_bw: number
    add_core_min: number
    add_core_max: number
    add_core_step: number
    add_mem_min: number
    add_mem_max: number
    add_mem_step: number
    add_disk_min: number
    add_disk_max: number
    add_disk_step: number
    add_bw_min: number
    add_bw_max: number
    add_bw_step: number
    active: boolean
    visible: boolean
    capacity_remaining: number
    sort_order: number
    image_ids: number[]
  }

  type PackageFormValue = {
    id: number | null
    plan_group_id: number | null
    name: string
    cores: number
    memory_gb: number
    disk_gb: number
    bandwidth_mbps: number
    cpu_model: string
    port_num: number
    monthly_price: number
    active: boolean
    visible: boolean
    capacity_remaining: number
    sort_order: number
  }

  type PackageBatchFormValue = {
    plan_group_id: number | null
    cpu_min: number
    cpu_max: number
    cpu_step: number
    memory_ratio: number
    memory_min: number
    memory_max: number
    disk_min: number
    disk_max: number
    disk_step: number
    bw_min: number
    bw_max: number
    bw_step: number
    port_num: number
    cpu_model: string
    price_multiplier: number
    total_cost: number
    active: boolean
    visible: boolean
    total_cores: number
    total_mem: number
    total_disk: number
    total_bw: number
    overcommit_enabled: boolean
    overcommit_ratio: number
  }

  type GeneratedPackageRow = {
    key: string
    plan_group_id: number | null
    name: string
    cores: number
    memory_gb: number
    disk_gb: number
    bandwidth_mbps: number
    cpu_model: string
    port_num: number
    monthly_price: number
    active: boolean
    visible: boolean
    capacity_remaining: number
  }

  type AutomationBindingValue = {
    pluginId: string
    instanceId: string
  }

  type BillingCycleFormValue = {
    id: number | null
    name: string
    months: number
    multiplier: number
    min_qty: number
    max_qty: number
    active: boolean
    sort_order: number
  }

  type SelectFilterValue = number | 'all' | null

  const userStore = useUserStore()
  const { info } = storeToRefs(userStore)

  const loading = ref(false)
  const activeTab = ref('goods-types')
  const goodsTypeId = ref<number | null>(null)
  const packagePlanGroupFilter = ref<SelectFilterValue>('all')
  const imagePlanGroupFilter = ref<number | null>(null)

  const goodsTypes = ref<GoodsTypeRow[]>([])
  const regions = ref<RegionRow[]>([])
  const planGroups = ref<PlanGroupRow[]>([])
  const packages = ref<PackageRow[]>([])
  const systemImages = ref<SystemImageRow[]>([])
  const scopedSystemImages = ref<SystemImageRow[]>([])
  const billingCycles = ref<BillingCycleRow[]>([])
  const automationPlugins = ref<AdminPluginRecord[]>([])

  const selectedRegionIds = ref<number[]>([])
  const selectedPlanGroupIds = ref<number[]>([])
  const selectedPackageIds = ref<number[]>([])
  const selectedSystemImageIds = ref<number[]>([])
  const selectedBillingCycleIds = ref<number[]>([])

  const goodsTypeDialogVisible = ref(false)
  const regionDialogVisible = ref(false)
  const planGroupDrawerVisible = ref(false)
  const packageDialogVisible = ref(false)
  const packageBatchDialogVisible = ref(false)
  const systemImageDialogVisible = ref(false)
  const billingCycleDialogVisible = ref(false)

  const goodsTypeSubmitting = ref(false)
  const regionSubmitting = ref(false)
  const planGroupSubmitting = ref(false)
  const packageSubmitting = ref(false)
  const systemImageSubmitting = ref(false)
  const billingCycleSubmitting = ref(false)

  const goodsTypeForm = ref<GoodsTypeFormValue>(createDefaultGoodsTypeForm())
  const regionForm = ref<RegionFormValue>(createDefaultRegionForm())
  const planGroupForm = ref<PlanGroupFormValue>(createDefaultPlanGroupForm())
  const packageForm = ref<PackageFormValue>(createDefaultPackageForm())
  const batchForm = ref<PackageBatchFormValue>(createDefaultBatchForm())
  const systemImageForm = ref<SystemImageFormValue>(createDefaultSystemImageDialogForm())
  const billingCycleForm = ref<BillingCycleFormValue>(createDefaultBillingCycleForm())
  const generatedPackages = ref<GeneratedPackageRow[]>([])

  const automationConfigSchemaJson = ref('{}')
  const automationConfigJson = ref('{}')
  const automationConfigError = ref('')
  const automationConfigLoading = ref(false)

  let automationConfigRequestId = 0

  const canView = computed(() =>
    hasAdminPermission(info.value?.buttons, [
      'goods_type.list',
      'region.list',
      'plan_group.list',
      'package.list',
      'system_image.list',
      'billing_cycle.list'
    ])
  )

  const goodsTypeOptions = computed(() =>
    goodsTypes.value.map((item) => ({
      value: Number(item.id),
      label: item.code ? `${item.name} (${item.code})` : item.name
    }))
  )

  const regionOptions = computed(() =>
    regions.value
      .filter((item) => item.id !== null)
      .map((item) => ({
        value: Number(item.id),
        label: item.code ? `${item.name} (${item.code})` : item.name
      }))
  )

  const planGroupOptions = computed(() =>
    planGroups.value
      .filter((item) => item.id !== null)
      .map((item) => ({
        value: Number(item.id),
        label: `${item.name}${item.line_id ? ` / ${item.line_id}` : ''}`
      }))
  )

  const automationOptions = computed(() =>
    automationPlugins.value.map((item) => {
      const pluginId = String(item.plugin_id || '').trim()
      const instanceId = String(item.instance_id || 'default').trim() || 'default'
      return {
        value: `${pluginId}:${instanceId}`,
        label: `${pluginId}/${instanceId}${item.enabled ? '（启用）' : '（未启用）'}`
      }
    })
  )

  const selectedGoodsType = computed(
    () => goodsTypes.value.find((item) => Number(item.id) === Number(goodsTypeId.value)) || null
  )

  const isCatalogReadonly = computed(() => {
    const current = selectedGoodsType.value
    if (!current) return false
    const targetRef = formatAutomationBinding(current, true)
    const plugin = automationPlugins.value.find(
      (item) =>
        `${String(item.plugin_id || '')}:${String(item.instance_id || 'default')}` === targetRef
    )
    return Boolean(plugin?.manifest?.capabilities?.automation?.catalog_readonly)
  })

  const filteredPackages = computed(() => {
    if (!packagePlanGroupFilter.value || packagePlanGroupFilter.value === 'all')
      return packages.value
    return packages.value.filter(
      (item) => Number(item.plan_group_id) === Number(packagePlanGroupFilter.value)
    )
  })

  const displaySystemImages = computed(() =>
    imagePlanGroupFilter.value ? scopedSystemImages.value : systemImages.value
  )

  const systemImageOptions = computed(() =>
    systemImages.value
      .filter((item) => item.id !== null)
      .map((item) => ({
        value: Number(item.id),
        label: `${item.name}${item.type ? ` (${formatImageType(item.type)})` : ''}`
      }))
  )

  onMounted(() => {
    initializePage()
  })

  function hasPermission(required: string | string[]) {
    return hasAdminPermission(info.value?.buttons, required)
  }

  function createDefaultGoodsTypeForm(): GoodsTypeFormValue {
    return {
      id: null,
      code: '',
      name: '',
      active: true,
      sort_order: 0,
      automation_plugin_id: '',
      automation_instance_id: '',
      resize_enabled: true,
      refund_enabled: true
    }
  }

  function createDefaultRegionForm(): RegionFormValue {
    return { id: null, goods_type_id: goodsTypeId.value, name: '', code: '', active: true }
  }

  function createDefaultPlanGroupForm(): PlanGroupFormValue {
    return {
      id: null,
      region_id: null,
      name: '',
      line_id: null,
      unit_core: 0,
      unit_mem: 0,
      unit_disk: 0,
      unit_bw: 0,
      add_core_min: -1,
      add_core_max: 0,
      add_core_step: 1,
      add_mem_min: -1,
      add_mem_max: 0,
      add_mem_step: 1,
      add_disk_min: -1,
      add_disk_max: 0,
      add_disk_step: 10,
      add_bw_min: -1,
      add_bw_max: 0,
      add_bw_step: 10,
      active: true,
      visible: true,
      capacity_remaining: -1,
      sort_order: 0,
      image_ids: []
    }
  }

  function createDefaultPackageForm(): PackageFormValue {
    return {
      id: null,
      plan_group_id: null,
      name: '',
      cores: 1,
      memory_gb: 1,
      disk_gb: 20,
      bandwidth_mbps: 1,
      cpu_model: '',
      port_num: 30,
      monthly_price: 0,
      active: true,
      visible: true,
      capacity_remaining: -1,
      sort_order: 0
    }
  }

  function createDefaultBillingCycleForm(): BillingCycleFormValue {
    return {
      id: null,
      name: '',
      months: 1,
      multiplier: 1,
      min_qty: 1,
      max_qty: 12,
      active: true,
      sort_order: 0
    }
  }

  function createDefaultBatchForm(): PackageBatchFormValue {
    return {
      plan_group_id: null,
      cpu_min: 1,
      cpu_max: 8,
      cpu_step: 1,
      memory_ratio: 2,
      memory_min: 1,
      memory_max: 128,
      disk_min: 20,
      disk_max: 200,
      disk_step: 20,
      bw_min: 1,
      bw_max: 100,
      bw_step: 5,
      port_num: 30,
      cpu_model: '',
      price_multiplier: 1,
      total_cost: 0,
      active: true,
      visible: true,
      total_cores: 0,
      total_mem: 0,
      total_disk: 0,
      total_bw: 0,
      overcommit_enabled: false,
      overcommit_ratio: 1
    }
  }

  function toNullableNumber(value: unknown) {
    if (value === '' || value === null || value === undefined) return null
    const parsed = Number(value)
    return Number.isFinite(parsed) ? parsed : null
  }

  function toNumber(value: unknown, fallback = 0) {
    const parsed = Number(value)
    return Number.isFinite(parsed) ? parsed : fallback
  }

  function toBoolean(value: unknown, fallback = false) {
    if (typeof value === 'boolean') return value
    if (value === 'true') return true
    if (value === 'false') return false
    return fallback
  }

  function normalizeGoodsType(item: any): GoodsTypeRow {
    return {
      id: toNullableNumber(item?.id ?? item?.ID),
      code: String(item?.code ?? item?.Code ?? ''),
      name: String(item?.name ?? item?.Name ?? ''),
      active: toBoolean(item?.active ?? item?.Active, true),
      sort_order: toNumber(item?.sort_order ?? item?.SortOrder, 0),
      automation_plugin_id: String(item?.automation_plugin_id ?? item?.AutomationPluginID ?? ''),
      automation_instance_id: String(
        item?.automation_instance_id ?? item?.AutomationInstanceID ?? ''
      )
    }
  }

  function normalizeRegion(item: any): RegionRow {
    return {
      id: toNullableNumber(item?.id ?? item?.ID),
      goods_type_id: toNullableNumber(item?.goods_type_id ?? item?.GoodsTypeID),
      name: String(item?.name ?? item?.Name ?? ''),
      code: String(item?.code ?? item?.Code ?? ''),
      active: toBoolean(item?.active ?? item?.Active, true)
    }
  }

  function normalizePlanGroup(item: any): PlanGroupRow {
    return {
      id: toNullableNumber(item?.id ?? item?.ID),
      goods_type_id: toNullableNumber(item?.goods_type_id ?? item?.GoodsTypeID),
      region_id: toNullableNumber(item?.region_id ?? item?.RegionID),
      line_id: toNullableNumber(item?.line_id ?? item?.LineID),
      name: String(item?.name ?? item?.Name ?? ''),
      unit_core: toNumber(item?.unit_core ?? item?.UnitCore, 0),
      unit_mem: toNumber(item?.unit_mem ?? item?.UnitMem, 0),
      unit_disk: toNumber(item?.unit_disk ?? item?.UnitDisk, 0),
      unit_bw: toNumber(item?.unit_bw ?? item?.UnitBW, 0),
      add_core_min: toNumber(item?.add_core_min ?? item?.AddCoreMin, -1),
      add_core_max: toNumber(item?.add_core_max ?? item?.AddCoreMax, 0),
      add_core_step: toNumber(item?.add_core_step ?? item?.AddCoreStep, 1),
      add_mem_min: toNumber(item?.add_mem_min ?? item?.AddMemMin, -1),
      add_mem_max: toNumber(item?.add_mem_max ?? item?.AddMemMax, 0),
      add_mem_step: toNumber(item?.add_mem_step ?? item?.AddMemStep, 1),
      add_disk_min: toNumber(item?.add_disk_min ?? item?.AddDiskMin, -1),
      add_disk_max: toNumber(item?.add_disk_max ?? item?.AddDiskMax, 0),
      add_disk_step: toNumber(item?.add_disk_step ?? item?.AddDiskStep, 10),
      add_bw_min: toNumber(item?.add_bw_min ?? item?.AddBwMin, -1),
      add_bw_max: toNumber(item?.add_bw_max ?? item?.AddBwMax, 0),
      add_bw_step: toNumber(item?.add_bw_step ?? item?.AddBwStep, 10),
      active: toBoolean(item?.active ?? item?.Active, true),
      visible: toBoolean(item?.visible ?? item?.Visible, true),
      capacity_remaining: toNumber(item?.capacity_remaining ?? item?.CapacityRemaining, -1),
      sort_order: toNumber(item?.sort_order ?? item?.SortOrder, 0)
    }
  }

  function normalizePackage(item: any): PackageRow {
    return {
      id: toNullableNumber(item?.id ?? item?.ID),
      plan_group_id: toNullableNumber(item?.plan_group_id ?? item?.PlanGroupID),
      name: String(item?.name ?? item?.Name ?? ''),
      cores: toNumber(item?.cores ?? item?.Cores, 1),
      memory_gb: toNumber(item?.memory_gb ?? item?.MemoryGB, 1),
      disk_gb: toNumber(item?.disk_gb ?? item?.DiskGB, 20),
      bandwidth_mbps: toNumber(item?.bandwidth_mbps ?? item?.BandwidthMB, 1),
      cpu_model: String(item?.cpu_model ?? item?.CPUModel ?? ''),
      port_num: toNumber(item?.port_num ?? item?.PortNum, 30),
      monthly_price: toNumber(item?.monthly_price ?? item?.Monthly ?? 0, 0),
      active: toBoolean(item?.active ?? item?.Active, true),
      visible: toBoolean(item?.visible ?? item?.Visible, true),
      capacity_remaining: toNumber(item?.capacity_remaining ?? item?.CapacityRemaining, -1),
      sort_order: toNumber(item?.sort_order ?? item?.SortOrder, 0)
    }
  }

  function normalizeSystemImage(item: any): SystemImageRow {
    return {
      id: toNullableNumber(item?.id ?? item?.ID),
      image_id: toNullableNumber(item?.image_id ?? item?.ImageID),
      name: String(item?.name ?? item?.Name ?? ''),
      type: String(item?.type ?? item?.Type ?? 'linux').toLowerCase(),
      enabled: toBoolean(item?.enabled ?? item?.Enabled, true)
    }
  }

  function normalizeBillingCycle(item: any): BillingCycleRow {
    return {
      id: toNullableNumber(item?.id ?? item?.ID),
      name: String(item?.name ?? item?.Name ?? ''),
      months: toNumber(item?.months ?? item?.Months, 1),
      multiplier: toNumber(item?.multiplier ?? item?.Multiplier, 1),
      min_qty: toNumber(item?.min_qty ?? item?.MinQty, 1),
      max_qty: toNumber(item?.max_qty ?? item?.MaxQty, 12),
      active: toBoolean(item?.active ?? item?.Active, true),
      sort_order: toNumber(item?.sort_order ?? item?.SortOrder, 0)
    }
  }

  function formatAutomationBinding(row: GoodsTypeRow, raw = false) {
    const pluginId = String(row.automation_plugin_id || '').trim()
    const instanceId = String(row.automation_instance_id || '').trim()
    if (!pluginId || !instanceId) return raw ? '' : '-'
    return raw ? `${pluginId}:${instanceId}` : `${pluginId}/${instanceId}`
  }

  function formatCapacity(value?: number | null) {
    const n = Number(value)
    if (!Number.isFinite(n)) return '-'
    if (n < 0) return '不限'
    if (n === 0) return '售罄'
    return String(n)
  }

  function capacityTagType(value?: number | null) {
    const n = Number(value)
    if (!Number.isFinite(n)) return 'info' as const
    if (n < 0) return 'success' as const
    if (n === 0) return 'danger' as const
    return 'primary' as const
  }

  function formatImageType(value?: string) {
    const type = String(value || '').toLowerCase()
    if (type.includes('win')) return 'Windows'
    if (type.includes('linux')) return 'Linux'
    return value || '-'
  }

  function imageTypeTagType(value?: string) {
    const type = String(value || '').toLowerCase()
    if (type.includes('win')) return 'primary' as const
    if (type.includes('linux')) return 'success' as const
    return 'info' as const
  }

  function getRegionName(regionId?: number | null) {
    return regions.value.find((item) => Number(item.id) === Number(regionId))?.name || '-'
  }

  function getPlanGroupName(planGroupId?: number | null) {
    return planGroups.value.find((item) => Number(item.id) === Number(planGroupId))?.name || '-'
  }

  function normalizePrettyJson(value?: string) {
    const text = String(value || '').trim()
    if (!text) {
      return '{}'
    }

    try {
      return JSON.stringify(JSON.parse(text), null, 2)
    } catch {
      return text
    }
  }

  function parseJsonText(value?: string) {
    const text = String(value || '').trim()
    if (!text) {
      return {}
    }

    return JSON.parse(text)
  }

  function resetAutomationConfigState() {
    automationConfigSchemaJson.value = '{}'
    automationConfigJson.value = '{}'
    automationConfigError.value = ''
    automationConfigLoading.value = false
  }

  function clearCatalogData() {
    regions.value = []
    planGroups.value = []
    packages.value = []
    systemImages.value = []
    scopedSystemImages.value = []
    billingCycles.value = []
    selectedRegionIds.value = []
    selectedPlanGroupIds.value = []
    selectedPackageIds.value = []
    selectedSystemImageIds.value = []
    selectedBillingCycleIds.value = []
    generatedPackages.value = []
  }

  async function initializePage() {
    loading.value = true
    try {
      await Promise.all([loadGoodsTypes(), loadAutomationPlugins()])
      await ensureSelectedGoodsType()
      await loadCatalogData()
    } finally {
      loading.value = false
    }
  }

  async function loadGoodsTypes() {
    const payload = await fetchAdminGoodsTypes().catch(() => ({ items: [] as CatalogGoodsType[] }))
    goodsTypes.value = (payload.items || []).map((item) => normalizeGoodsType(item))
  }

  async function loadAutomationPlugins() {
    try {
      const payload = await fetchAdminPlugins()
      automationPlugins.value = (payload.items || []).filter(
        (item) => String(item.category || '').trim() === 'automation'
      )
    } catch {
      automationPlugins.value = []
    }
  }

  async function ensureSelectedGoodsType(preferredId?: number | null) {
    const preferred = preferredId ?? goodsTypeId.value
    if (preferred && goodsTypes.value.some((item) => Number(item.id) === Number(preferred))) {
      goodsTypeId.value = Number(preferred)
      return
    }
    goodsTypeId.value = goodsTypes.value[0]?.id ? Number(goodsTypes.value[0].id) : null
  }

  async function loadCatalogData() {
    if (!goodsTypeId.value) {
      clearCatalogData()
      return
    }

    const params = { goods_type_id: goodsTypeId.value }
    const [regionsResult, planGroupsResult, packagesResult, imagesResult, cyclesResult] =
      await Promise.allSettled([
        fetchAdminRegions(params),
        fetchAdminPlanGroups(params),
        fetchAdminPackages(params),
        fetchAdminSystemImages(),
        fetchAdminBillingCycles()
      ])

    regions.value =
      regionsResult.status === 'fulfilled'
        ? (regionsResult.value.items || []).map(normalizeRegion)
        : []
    planGroups.value =
      planGroupsResult.status === 'fulfilled'
        ? (planGroupsResult.value.items || []).map(normalizePlanGroup)
        : []
    packages.value =
      packagesResult.status === 'fulfilled'
        ? (packagesResult.value.items || []).map(normalizePackage)
        : []
    systemImages.value =
      imagesResult.status === 'fulfilled'
        ? (imagesResult.value.items || []).map(normalizeSystemImage)
        : []
    billingCycles.value =
      cyclesResult.status === 'fulfilled'
        ? (cyclesResult.value.items || []).map(normalizeBillingCycle)
        : []

    if (
      packagePlanGroupFilter.value !== 'all' &&
      packagePlanGroupFilter.value !== null &&
      !planGroups.value.some((item) => Number(item.id) === Number(packagePlanGroupFilter.value))
    ) {
      packagePlanGroupFilter.value = 'all'
    }

    if (
      imagePlanGroupFilter.value &&
      !planGroups.value.some((item) => Number(item.id) === Number(imagePlanGroupFilter.value))
    ) {
      imagePlanGroupFilter.value = null
    }

    if (
      batchForm.value.plan_group_id &&
      !planGroups.value.some((item) => Number(item.id) === Number(batchForm.value.plan_group_id))
    ) {
      batchForm.value.plan_group_id = planGroups.value[0]?.id
        ? Number(planGroups.value[0].id)
        : null
      generatedPackages.value = []
    }

    await refreshScopedImages()
  }

  async function refreshScopedImages() {
    if (!imagePlanGroupFilter.value) {
      scopedSystemImages.value = []
      return
    }
    const payload = await fetchAdminSystemImages({
      plan_group_id: imagePlanGroupFilter.value
    }).catch(() => ({ items: [] as CatalogSystemImage[] }))
    scopedSystemImages.value = (payload.items || []).map((item) => normalizeSystemImage(item))
  }

  async function handleGoodsTypeChange() {
    packagePlanGroupFilter.value = 'all'
    imagePlanGroupFilter.value = null
    await loadCatalogData()
  }

  async function openGoodsTypeDialog(row?: GoodsTypeRow) {
    resetAutomationConfigState()
    goodsTypeForm.value = row
      ? {
          id: row.id,
          code: row.code,
          name: row.name,
          active: row.active,
          sort_order: row.sort_order,
          automation_plugin_id: row.automation_plugin_id,
          automation_instance_id: row.automation_instance_id,
          resize_enabled: true,
          refund_enabled: true
        }
      : createDefaultGoodsTypeForm()

    if (row?.id) {
      try {
        const caps = await fetchAdminGoodsTypeCapabilities(row.id)
        goodsTypeForm.value.resize_enabled = toBoolean(caps.resize_enabled, true)
        goodsTypeForm.value.refund_enabled = toBoolean(caps.refund_enabled, true)
      } catch {
        goodsTypeForm.value.resize_enabled = true
        goodsTypeForm.value.refund_enabled = true
      }
    } else if (automationOptions.value[0]?.value) {
      const [pluginId, instanceId] = automationOptions.value[0].value.split(':')
      goodsTypeForm.value.automation_plugin_id = pluginId || ''
      goodsTypeForm.value.automation_instance_id = instanceId || ''
    }

    goodsTypeDialogVisible.value = true
  }

  async function handleGoodsTypeAutomationChange(value: AutomationBindingValue) {
    const pluginId = String(value.pluginId || '').trim()
    const instanceId = String(value.instanceId || '').trim()
    const requestId = ++automationConfigRequestId

    resetAutomationConfigState()

    if (!pluginId || !instanceId) {
      return
    }

    automationConfigLoading.value = true

    try {
      const [schemaResponse, configResponse] = await Promise.all([
        fetchAdminPluginInstanceConfigSchema('automation', pluginId, instanceId),
        fetchAdminPluginInstanceConfig('automation', pluginId, instanceId)
      ])

      if (requestId !== automationConfigRequestId) {
        return
      }

      automationConfigSchemaJson.value = normalizePrettyJson(schemaResponse.json_schema)
      automationConfigJson.value = normalizePrettyJson(configResponse.config_json)
    } catch (error: any) {
      if (requestId !== automationConfigRequestId) {
        return
      }

      automationConfigError.value = String(
        error?.response?.data?.error || error?.message || '加载自动化配置失败'
      )
      automationConfigSchemaJson.value = '{}'
      automationConfigJson.value = '{}'
    } finally {
      if (requestId === automationConfigRequestId) {
        automationConfigLoading.value = false
      }
    }
  }

  function handleAutomationConfigJsonChange(value: string) {
    automationConfigJson.value = value
  }

  async function submitGoodsType(form: GoodsTypeFormValue) {
    if (!form.name.trim()) return ElMessage.error('请输入商品类型名称')
    if (!form.code.trim()) return ElMessage.error('请输入商品类型代码')
    const pluginId = form.automation_plugin_id.trim()
    const instanceId = form.automation_instance_id.trim()
    let configJson = ''

    if (pluginId && instanceId) {
      try {
        configJson = JSON.stringify(parseJsonText(automationConfigJson.value))
      } catch {
        automationConfigError.value = '配置 JSON 格式不正确'
        return ElMessage.error('请修正自动化配置 JSON')
      }
    }

    goodsTypeSubmitting.value = true
    try {
      const payload = {
        code: form.code.trim(),
        name: form.name.trim(),
        active: form.active,
        sort_order: form.sort_order,
        automation_plugin_id: pluginId,
        automation_instance_id: instanceId
      }
      let currentId = form.id
      if (form.id) await updateAdminGoodsType(form.id, payload)
      else currentId = toNullableNumber(((await createAdminGoodsType(payload)) as any)?.id)
      if (currentId) {
        await updateAdminGoodsTypeCapabilities(currentId, {
          resize_enabled: form.resize_enabled,
          refund_enabled: form.refund_enabled
        })
      }
      if (pluginId && instanceId) {
        await updateAdminPluginInstanceConfig(
          'automation',
          pluginId,
          instanceId,
          configJson || '{}'
        )
      }
      goodsTypeDialogVisible.value = false
      ElMessage.success('商品类型已保存')
      await loadGoodsTypes()
      await ensureSelectedGoodsType(currentId)
      await loadCatalogData()
    } finally {
      goodsTypeSubmitting.value = false
    }
  }

  async function removeGoodsType(row: GoodsTypeRow) {
    if (!row.id || !(await confirmAction('确认删除该商品类型？'))) return
    await deleteAdminGoodsType(row.id)
    ElMessage.success('已删除商品类型')
    await loadGoodsTypes()
    await ensureSelectedGoodsType(goodsTypeId.value === row.id ? null : goodsTypeId.value)
    await loadCatalogData()
  }

  async function syncGoodsType(row: GoodsTypeRow) {
    if (!row.id) return
    await syncAdminGoodsTypeAutomation(row.id, 'merge')
    ElMessage.success('已触发同步')
    if (Number(goodsTypeId.value) === Number(row.id)) await loadCatalogData()
  }

  async function syncCurrentGoodsType() {
    if (!goodsTypeId.value) return ElMessage.error('请先选择商品类型')
    await syncAdminGoodsTypeAutomation(goodsTypeId.value, 'merge')
    ElMessage.success('已触发同步')
    await loadCatalogData()
  }

  function openRegionDialog(row?: RegionRow) {
    regionForm.value = row ? { ...row } : createDefaultRegionForm()
    regionDialogVisible.value = true
  }

  async function submitRegion(form: RegionFormValue) {
    if (!goodsTypeId.value) return ElMessage.error('请先选择商品类型')
    if (!form.name.trim()) return ElMessage.error('请输入地区名称')
    regionSubmitting.value = true
    try {
      const payload = {
        goods_type_id: goodsTypeId.value,
        name: form.name.trim(),
        code: form.code.trim(),
        active: form.active
      }
      if (form.id) await updateAdminRegion(form.id, payload)
      else await createAdminRegion(payload)
      regionDialogVisible.value = false
      ElMessage.success('地区已保存')
      await loadCatalogData()
    } finally {
      regionSubmitting.value = false
    }
  }

  async function removeRegion(row: RegionRow) {
    if (!row.id || !(await confirmAction('确认删除该地区？'))) return
    await deleteAdminRegion(row.id)
    ElMessage.success('已删除地区')
    await loadCatalogData()
  }

  async function removeSelectedRegions() {
    if (
      !selectedRegionIds.value.length ||
      !(await confirmAction(`确认删除所选 ${selectedRegionIds.value.length} 个地区？`))
    )
      return
    await bulkDeleteAdminRegions(selectedRegionIds.value)
    ElMessage.success('已批量删除地区')
    await loadCatalogData()
  }

  async function loadPlanGroupImages(planGroupId: number) {
    const payload = await fetchAdminSystemImages({ plan_group_id: planGroupId }).catch(() => ({
      items: [] as CatalogSystemImage[]
    }))
    return (payload.items || [])
      .map((item) => normalizeSystemImage(item).id)
      .filter((item): item is number => item !== null)
  }

  async function openPlanGroupDialog(row?: PlanGroupRow) {
    planGroupForm.value = row ? { ...row, image_ids: [] } : createDefaultPlanGroupForm()
    if (row?.id) planGroupForm.value.image_ids = await loadPlanGroupImages(row.id)
    planGroupDrawerVisible.value = true
  }

  async function submitPlanGroup(form: PlanGroupFormValue) {
    if (!form.region_id) return ElMessage.error('请选择地区')
    if (!form.name.trim()) return ElMessage.error('请输入线路名称')
    planGroupSubmitting.value = true
    try {
      const payload = { ...form, name: form.name.trim() }
      delete (payload as any).id
      delete (payload as any).image_ids
      let currentId = form.id
      if (form.id) await updateAdminPlanGroup(form.id, payload)
      else currentId = toNullableNumber(((await createAdminPlanGroup(payload)) as any)?.id)
      if (currentId) await setAdminPlanGroupSystemImages(currentId, { image_ids: form.image_ids })
      planGroupDrawerVisible.value = false
      ElMessage.success('线路已保存')
      await loadCatalogData()
    } finally {
      planGroupSubmitting.value = false
    }
  }

  async function togglePlanGroupActive(row: PlanGroupRow) {
    if (!row.id) return
    await updateAdminPlanGroup(row.id, { active: !row.active })
    ElMessage.success(row.active ? '已禁用' : '已启用')
    await loadCatalogData()
  }

  async function removePlanGroup(row: PlanGroupRow) {
    if (!row.id || !(await confirmAction('确认删除该线路？'))) return
    await deleteAdminPlanGroup(row.id)
    ElMessage.success('已删除线路')
    await loadCatalogData()
  }

  async function removeSelectedPlanGroups() {
    if (
      !selectedPlanGroupIds.value.length ||
      !(await confirmAction(`确认删除所选 ${selectedPlanGroupIds.value.length} 条线路？`))
    )
      return
    await bulkDeleteAdminPlanGroups(selectedPlanGroupIds.value)
    ElMessage.success('已批量删除线路')
    await loadCatalogData()
  }

  function openPackageDialog(row?: PackageRow) {
    packageForm.value = row ? { ...row } : createDefaultPackageForm()
    if (!row && packagePlanGroupFilter.value && packagePlanGroupFilter.value !== 'all') {
      packageForm.value.plan_group_id = Number(packagePlanGroupFilter.value)
    }
    packageDialogVisible.value = true
  }

  function handleBatchFormUpdate(value: PackageBatchFormValue) {
    batchForm.value = value
  }

  function openPackageBatchDialog() {
    if (!planGroups.value.length) {
      ElMessage.error('暂无可用线路')
      return
    }

    if (packagePlanGroupFilter.value && packagePlanGroupFilter.value !== 'all') {
      batchForm.value.plan_group_id = Number(packagePlanGroupFilter.value)
    } else if (!batchForm.value.plan_group_id) {
      batchForm.value.plan_group_id = Number(planGroups.value[0]?.id || 0) || null
    }

    packageBatchDialogVisible.value = true
  }

  function clearGeneratedPackages() {
    generatedPackages.value = []
  }

  function calculateBatchCapacity(cores: number, memory: number, disk: number, bandwidth: number) {
    const multiplier = batchForm.value.overcommit_enabled
      ? Number(batchForm.value.overcommit_ratio || 1)
      : 1

    const totals = {
      cores: Number(batchForm.value.total_cores || 0) * multiplier,
      mem: Number(batchForm.value.total_mem || 0) * multiplier,
      disk: Number(batchForm.value.total_disk || 0) * multiplier,
      bw: Number(batchForm.value.total_bw || 0) * multiplier
    }

    const candidates: number[] = []

    if (totals.cores > 0 && cores > 0) candidates.push(Math.floor(totals.cores / cores))
    if (totals.mem > 0 && memory > 0) candidates.push(Math.floor(totals.mem / memory))
    if (totals.disk > 0 && disk > 0) candidates.push(Math.floor(totals.disk / disk))
    if (totals.bw > 0 && bandwidth > 0) candidates.push(Math.floor(totals.bw / bandwidth))

    if (!candidates.length) {
      return -1
    }

    return Math.max(0, Math.min(...candidates))
  }

  function generateBatchPackages() {
    if (!batchForm.value.plan_group_id) {
      return ElMessage.error('请选择线路')
    }

    const planGroup = planGroups.value.find(
      (item) => Number(item.id) === Number(batchForm.value.plan_group_id)
    )

    if (!planGroup) {
      return ElMessage.error('线路信息未加载')
    }

    const cpuMin = Number(batchForm.value.cpu_min || 0)
    const cpuMax = Number(batchForm.value.cpu_max || 0)
    const cpuStep = Math.max(1, Number(batchForm.value.cpu_step || 1))
    const diskMin = Number(batchForm.value.disk_min || 0)
    const diskMax = Number(batchForm.value.disk_max || 0)
    const diskStep = Math.max(1, Number(batchForm.value.disk_step || 1))
    const bwMin = Number(batchForm.value.bw_min || 0)
    const bwMax = Number(batchForm.value.bw_max || 0)
    const bwStep = Math.max(1, Number(batchForm.value.bw_step || 1))
    const memoryRatio = Number(batchForm.value.memory_ratio || 0)
    const memoryMin = Number(batchForm.value.memory_min || 0)
    const memoryMax = Number(batchForm.value.memory_max || 0)

    if (!cpuMin || !cpuMax || cpuMax < cpuMin) {
      return ElMessage.error('CPU 范围不正确')
    }

    if (!diskMin || !diskMax || diskMax < diskMin) {
      return ElMessage.error('存储范围不正确')
    }

    if (!bwMin || !bwMax || bwMax < bwMin) {
      return ElMessage.error('带宽范围不正确')
    }

    if (!memoryRatio) {
      return ElMessage.error('内存比率需大于 0')
    }

    let priceMultiplier = Number(batchForm.value.price_multiplier || 1)
    const totalCost = Number(batchForm.value.total_cost || 0)

    if (totalCost > 0) {
      const baseCost =
        Number(planGroup.unit_core || 0) * Number(batchForm.value.total_cores || 0) +
        Number(planGroup.unit_mem || 0) * Number(batchForm.value.total_mem || 0) +
        Number(planGroup.unit_disk || 0) * Number(batchForm.value.total_disk || 0) +
        Number(planGroup.unit_bw || 0) * Number(batchForm.value.total_bw || 0)

      if (baseCost > 0) {
        priceMultiplier = totalCost / baseCost
      }
    }

    const items: GeneratedPackageRow[] = []

    for (let cpu = cpuMin; cpu <= cpuMax; cpu += cpuStep) {
      let memory = Math.round(cpu * memoryRatio)
      if (memoryMin && memory < memoryMin) memory = memoryMin
      if (memoryMax && memory > memoryMax) continue

      for (let disk = diskMin; disk <= diskMax; disk += diskStep) {
        for (let bandwidth = bwMin; bandwidth <= bwMax; bandwidth += bwStep) {
          const monthlyBase =
            Number(planGroup.unit_core || 0) * cpu +
            Number(planGroup.unit_mem || 0) * memory +
            Number(planGroup.unit_disk || 0) * disk +
            Number(planGroup.unit_bw || 0) * bandwidth

          items.push({
            key: `${cpu}-${memory}-${disk}-${bandwidth}`,
            plan_group_id: batchForm.value.plan_group_id,
            name: `${cpu}C${memory}G ${disk}G ${bandwidth}M`,
            cores: cpu,
            memory_gb: memory,
            disk_gb: disk,
            bandwidth_mbps: bandwidth,
            cpu_model: batchForm.value.cpu_model,
            port_num: batchForm.value.port_num,
            monthly_price: Number((monthlyBase * priceMultiplier).toFixed(2)),
            active: batchForm.value.active,
            visible: batchForm.value.visible,
            capacity_remaining: calculateBatchCapacity(cpu, memory, disk, bandwidth)
          })
        }
      }
    }

    if (!items.length) {
      generatedPackages.value = []
      return ElMessage.warning('未生成任何套餐，请检查条件')
    }

    const maxRows = 200
    if (items.length > maxRows) {
      ElMessage.warning(`生成数量过多，已截断至 ${maxRows} 条`)
    }

    generatedPackages.value = items.slice(0, maxRows)
  }

  async function applyGeneratedPackages() {
    if (!generatedPackages.value.length) {
      return
    }

    const confirmed = await confirmAction(`确认批量创建 ${generatedPackages.value.length} 个套餐？`)
    if (!confirmed) {
      return
    }

    for (const item of generatedPackages.value) {
      const payload = { ...item }
      delete (payload as { key?: string }).key
      await createAdminPackage(payload)
    }

    generatedPackages.value = []
    packageBatchDialogVisible.value = false
    ElMessage.success('已批量创建套餐')
    await loadCatalogData()
  }

  async function submitPackage(form: PackageFormValue) {
    if (!form.plan_group_id) return ElMessage.error('请选择线路')
    if (!form.name.trim()) return ElMessage.error('请输入套餐名称')
    packageSubmitting.value = true
    try {
      const payload = { ...form, name: form.name.trim() }
      delete (payload as any).id
      if (form.id) await updateAdminPackage(form.id, payload)
      else await createAdminPackage(payload)
      packageDialogVisible.value = false
      ElMessage.success('套餐已保存')
      await loadCatalogData()
    } finally {
      packageSubmitting.value = false
    }
  }

  async function removePackage(row: PackageRow) {
    if (!row.id || !(await confirmAction('确认删除该套餐？'))) return
    await deleteAdminPackage(row.id)
    ElMessage.success('已删除套餐')
    await loadCatalogData()
  }

  async function removeSelectedPackages() {
    if (
      !selectedPackageIds.value.length ||
      !(await confirmAction(`确认删除所选 ${selectedPackageIds.value.length} 个套餐？`))
    )
      return
    await bulkDeleteAdminPackages(selectedPackageIds.value)
    ElMessage.success('已批量删除套餐')
    await loadCatalogData()
  }

  function openSystemImageDialog(row?: SystemImageRow) {
    systemImageForm.value = row ? { ...row } : createDefaultSystemImageDialogForm()
    systemImageDialogVisible.value = true
  }

  async function submitSystemImage(form: SystemImageFormValue) {
    if (!form.image_id || form.image_id <= 0) return ElMessage.error('镜像 ID 必须是正整数')
    if (!form.name.trim()) return ElMessage.error('请输入镜像名称')
    systemImageSubmitting.value = true
    try {
      const payload = {
        image_id: form.image_id,
        name: form.name.trim(),
        type: form.type,
        enabled: form.enabled
      }
      if (form.id) await updateAdminSystemImage(form.id, payload)
      else await createAdminSystemImage(payload)
      systemImageDialogVisible.value = false
      ElMessage.success('镜像已保存')
      await loadCatalogData()
    } finally {
      systemImageSubmitting.value = false
    }
  }

  async function removeSystemImage(row: SystemImageRow) {
    if (!row.id || !(await confirmAction('确认删除该镜像？'))) return
    await deleteAdminSystemImage(row.id)
    ElMessage.success('已删除镜像')
    await loadCatalogData()
  }

  async function removeSelectedSystemImages() {
    if (
      !selectedSystemImageIds.value.length ||
      !(await confirmAction(`确认删除所选 ${selectedSystemImageIds.value.length} 个镜像？`))
    )
      return
    await bulkDeleteAdminSystemImages(selectedSystemImageIds.value)
    ElMessage.success('已批量删除镜像')
    await loadCatalogData()
  }

  async function syncSystemImages() {
    if (!imagePlanGroupFilter.value) return ElMessage.error('请先选择线路')
    await syncAdminSystemImages({ plan_group_id: imagePlanGroupFilter.value })
    ElMessage.success('已触发镜像同步')
    await loadCatalogData()
  }

  function openBillingCycleDialog(row?: BillingCycleRow) {
    billingCycleForm.value = row ? { ...row } : createDefaultBillingCycleForm()
    billingCycleDialogVisible.value = true
  }

  async function submitBillingCycle(form: BillingCycleFormValue) {
    if (!form.name.trim()) return ElMessage.error('请输入周期名称')
    billingCycleSubmitting.value = true
    try {
      const payload = { ...form, name: form.name.trim() }
      delete (payload as any).id
      if (form.id) await updateAdminBillingCycle(form.id, payload)
      else await createAdminBillingCycle(payload)
      billingCycleDialogVisible.value = false
      ElMessage.success('计费周期已保存')
      await loadCatalogData()
    } finally {
      billingCycleSubmitting.value = false
    }
  }

  async function removeBillingCycle(row: BillingCycleRow) {
    if (!row.id || !(await confirmAction('确认删除该计费周期？'))) return
    await deleteAdminBillingCycle(row.id)
    ElMessage.success('已删除计费周期')
    await loadCatalogData()
  }

  async function removeSelectedBillingCycles() {
    if (
      !selectedBillingCycleIds.value.length ||
      !(await confirmAction(`确认删除所选 ${selectedBillingCycleIds.value.length} 个计费周期？`))
    )
      return
    await bulkDeleteAdminBillingCycles(selectedBillingCycleIds.value)
    ElMessage.success('已批量删除计费周期')
    await loadCatalogData()
  }

  async function confirmAction(message: string) {
    try {
      await ElMessageBox.confirm(message, '提示', {
        type: 'warning',
        confirmButtonText: '确认',
        cancelButtonText: '取消'
      })
      return true
    } catch {
      return false
    }
  }

  function onRegionSelectionChange(rows: RegionRow[]) {
    selectedRegionIds.value = rows
      .map((item) => item.id)
      .filter((item): item is number => item !== null)
  }

  function onPlanGroupSelectionChange(rows: PlanGroupRow[]) {
    selectedPlanGroupIds.value = rows
      .map((item) => item.id)
      .filter((item): item is number => item !== null)
  }

  function onPackageSelectionChange(rows: PackageRow[]) {
    selectedPackageIds.value = rows
      .map((item) => item.id)
      .filter((item): item is number => item !== null)
  }

  function onSystemImageSelectionChange(rows: SystemImageRow[]) {
    selectedSystemImageIds.value = rows
      .map((item) => item.id)
      .filter((item): item is number => item !== null)
  }

  function onBillingCycleSelectionChange(rows: BillingCycleRow[]) {
    selectedBillingCycleIds.value = rows
      .map((item) => item.id)
      .filter((item): item is number => item !== null)
  }
</script>

<style scoped lang="scss">
  .catalog-page {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .page-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 16px;
    margin-bottom: 12px;
  }

  .page-title {
    color: var(--el-text-color-primary);
    font-size: 28px;
    font-weight: 700;
    line-height: 1.1;
  }

  .page-subtitle {
    margin-top: 8px;
    color: var(--el-text-color-secondary);
    font-size: 14px;
    line-height: 1.6;
  }

  .page-actions,
  .toolbar-actions,
  .table-actions {
    display: flex;
    flex-wrap: wrap;
    gap: 12px;
  }

  .goods-type-select {
    width: 260px;
  }

  .page-alert,
  .tab-alert {
    margin-bottom: 16px;
  }

  .toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    margin-bottom: 14px;
  }

  .toolbar-filters {
    display: flex;
    align-items: center;
    gap: 12px;
    flex-wrap: wrap;
  }

  .inline-select {
    width: 220px;
  }

  @media (max-width: 768px) {
    .page-header,
    .toolbar {
      flex-direction: column;
      align-items: stretch;
    }

    .goods-type-select,
    .inline-select {
      width: 100%;
    }
  }
</style>
