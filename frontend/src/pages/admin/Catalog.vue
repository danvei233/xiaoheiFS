<template>
  <div class="page">
    <div class="page-header">
      <div>
        <div class="page-title">售卖配置</div>
        <div class="subtle">地区、线路、套餐与计费策略维护</div>
      </div>
      <div class="page-header-actions" style="justify-content: flex-end">
        <a-space>
          <a-select
            v-model:value="goodsTypeId"
            :options="goodsTypeOptions"
            placeholder="选择商品类型"
            style="width: 260px"
            allow-clear
          />
          <a-button :disabled="!goodsTypeId" @click="syncCurrentGoodsType">同步当前类型（merge）</a-button>
        </a-space>
      </div>
    </div>

    <a-tabs>
      <a-tab-pane key="goods-types" tab="商品类型">
        <a-card class="card">
          <div class="page-header-actions" style="justify-content: flex-end; margin-bottom: 12px">
            <a-space>
              <a-button type="primary" @click="openGoodsType()">新增商品类型</a-button>
            </a-space>
          </div>
          <a-table :columns="goodsTypeColumns" :data-source="goodsTypes" row-key="id" :pagination="false">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'active'">
                <a-tag :color="record.active ? 'green' : 'red'">{{ record.active ? '启用' : '停用' }}</a-tag>
              </template>
              <template v-else-if="column.key === 'action'">
                <a-space>
                  <a-button size="small" @click="openGoodsType(record)">编辑</a-button>
                  <a-button size="small" @click="syncGoodsType(record)">同步</a-button>
                  <a-button size="small" danger @click="removeGoodsType(record)">删除</a-button>
                </a-space>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-tab-pane>
      <a-tab-pane key="regions" tab="地区">
        <a-card class="card">
          <div class="page-header-actions" style="justify-content: flex-end; margin-bottom: 12px">
            <a-space>
              <a-tag v-if="isOpenIDCPlugin" color="blue">OpenIDCS 插件：地区由同步生成，只读</a-tag>
              <a-button danger :disabled="!selectedRegionKeys.length" @click="bulkRemoveRegions">批量删除</a-button>
              <a-button v-if="!isOpenIDCPlugin" type="primary" @click="openRegion()">新增地区</a-button>
            </a-space>
          </div>
          <a-table
            :columns="regionColumns"
            :data-source="regions"
            row-key="id"
            :pagination="false"
            :row-selection="regionSelection"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'active'">
                <a-tag :color="record.active ? 'green' : 'red'">{{ record.active ? '启用' : '停用' }}</a-tag>
              </template>
              <template v-else-if="column.key === 'action'">
                <a-space>
                  <a-button v-if="!isOpenIDCPlugin" size="small" @click="openRegion(record)">编辑</a-button>
                  <a-button size="small" danger @click="removeRegion(record)">删除</a-button>
                </a-space>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-tab-pane>

      <a-tab-pane key="lines" tab="线路/附加项">
        <a-card class="card">
          <div class="page-header-actions" style="justify-content: flex-end; margin-bottom: 12px">
            <a-space>
              <a-tag v-if="isOpenIDCPlugin" color="blue">OpenIDCS 插件：线路由同步生成，只允许启用/禁用</a-tag>
              <a-button v-if="!isOpenIDCPlugin" danger :disabled="!selectedLineKeys.length" @click="bulkRemoveLines">批量删除</a-button>
              <a-button v-if="!isOpenIDCPlugin" type="primary" @click="openLine()">新增线路</a-button>
            </a-space>
          </div>
          <a-table
            :columns="lineColumns"
            :data-source="lines"
            row-key="id"
            :pagination="false"
            :row-selection="isOpenIDCPlugin ? undefined : lineSelection"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'region_id'">
                {{ regionNameById(record.region_id) }}
              </template>
              <template v-else-if="column.key === 'active'">
                <a-tag :color="record.active ? 'green' : 'red'">{{ record.active ? '启用' : '停用' }}</a-tag>
              </template>
              <template v-else-if="column.key === 'visible'">
                <a-tag :color="record.visible ? 'green' : 'default'">{{ record.visible ? '可见' : '隐藏' }}</a-tag>
              </template>
              <template v-else-if="column.key === 'capacity_remaining'">
                <a-tag :color="capacityTagColor(record.capacity_remaining)">{{ formatCapacity(record.capacity_remaining) }}</a-tag>
              </template>
              <template v-else-if="column.key === 'action'">
                <a-space>
                  <!-- openidc 插件：只允许启用/禁用 -->
                  <template v-if="isOpenIDCPlugin">
                    <a-button
                      size="small"
                      :type="record.active ? 'default' : 'primary'"
                      @click="toggleLineActive(record)"
                    >{{ record.active ? '禁用' : '启用' }}</a-button>
                  </template>
                  <!-- 普通插件：完整编辑 -->
                  <template v-else>
                    <a-button size="small" @click="openLine(record)">编辑</a-button>
                    <a-button size="small" danger @click="removeLine(record)">删除</a-button>
                  </template>
                </a-space>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-tab-pane>

      <a-tab-pane key="packages" tab="套餐">
        <a-card class="card">
          <div class="page-header-actions" style="justify-content: space-between; margin-bottom: 12px">
            <a-space>
              <a-select v-model:value="packageLineId" allow-clear placeholder="筛选线路" style="width: 200px">
                <a-select-option value="all">全部线路</a-select-option>
                <a-select-option v-for="line in lines" :key="line.id" :value="line.id">
                  {{ line.name }}
                </a-select-option>
              </a-select>
            </a-space>
            <a-space>
              <a-button danger :disabled="!selectedPackageKeys.length" @click="bulkRemovePackages">批量删除</a-button>
              <a-button @click="openBatch">批量生成</a-button>
              <a-button type="primary" @click="openPackage()">新增套餐</a-button>
            </a-space>
          </div>
          <a-table
            :columns="packageColumns"
            :data-source="filteredPackages"
            row-key="id"
            :pagination="false"
            :row-selection="packageSelection"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'plan_group_id'">
                {{ lineNameById(record.plan_group_id) }}
              </template>
              <template v-else-if="column.key === 'active'">
                <a-tag :color="record.active ? 'green' : 'red'">{{ record.active ? '启用' : '停用' }}</a-tag>
              </template>
              <template v-else-if="column.key === 'visible'">
                <a-tag :color="record.visible ? 'green' : 'default'">{{ record.visible ? '可见' : '隐藏' }}</a-tag>
              </template>
              <template v-else-if="column.key === 'capacity_remaining'">
                <a-tag :color="capacityTagColor(record.capacity_remaining)">{{ formatCapacity(record.capacity_remaining) }}</a-tag>
              </template>
              <template v-else-if="column.key === 'action'">
                <a-space>
                  <a-button size="small" @click="openPackage(record)">编辑</a-button>
                  <a-button size="small" danger @click="removePackage(record)">删除</a-button>
                </a-space>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-tab-pane>

      <a-tab-pane key="images" tab="系统镜像">
        <a-card class="card">
          <div class="page-header-actions" style="justify-content: space-between; margin-bottom: 12px">
            <a-space>
              <a-select v-model:value="imageLineId" allow-clear placeholder="选择同步线路" style="width: 200px">
                <a-select-option v-for="line in lines" :key="line.id" :value="line.id">
                  {{ `${line.name} (${lineImageCountByLineId(line.id)})` }}
                </a-select-option>
              </a-select>
              <span class="subtle">同步会更新线路可用镜像</span>
            </a-space>
            <a-space>
              <a-button danger :disabled="!selectedImageKeys.length" @click="bulkRemoveImages">批量删除</a-button>
              <a-button @click="syncImages">同步镜像</a-button>
              <a-button type="primary" @click="openImage()">新增镜像</a-button>
            </a-space>
          </div>
          <a-table
            :columns="imageColumns"
            :data-source="filteredImages"
            row-key="id"
            :pagination="false"
            :row-selection="imageSelection"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'type'">
                <a-tag :color="typeTagColor(record.type)">
                  <WindowsOutlined v-if="isWindowsType(record.type)" />
                  <CodeOutlined v-else-if="isLinuxType(record.type)" />
                  <span style="margin-left: 6px">{{ formatImageType(record.type) }}</span>
                </a-tag>
              </template>
              <template v-else-if="column.key === 'enabled'">
                <a-tag :color="record.enabled ? 'green' : 'red'">{{ record.enabled ? '启用' : '停用' }}</a-tag>
              </template>
              <template v-else-if="column.key === 'action'">
                <a-space>
                  <a-button size="small" @click="openImage(record)">编辑</a-button>
                  <a-button size="small" danger @click="removeImage(record)">删除</a-button>
                </a-space>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-tab-pane>

      <a-tab-pane key="billing" tab="计费周期">
        <a-card class="card">
          <div class="page-header-actions" style="justify-content: flex-end; margin-bottom: 12px">
            <a-space>
              <a-button danger :disabled="!selectedCycleKeys.length" @click="bulkRemoveCycles">批量删除</a-button>
              <a-button type="primary" @click="openCycle()">新增周期</a-button>
            </a-space>
          </div>
          <a-table
            :columns="cycleColumns"
            :data-source="billingCycles"
            row-key="id"
            :pagination="false"
            :row-selection="cycleSelection"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'action'">
                <a-space>
                  <a-button size="small" @click="openCycle(record)">编辑</a-button>
                  <a-button size="small" danger @click="removeCycle(record)">删除</a-button>
                </a-space>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-tab-pane>
    </a-tabs>

    <a-modal v-model:open="goodsTypeOpen" title="商品类型" :confirm-loading="goodsTypeSaving" @ok="submitGoodsType">
      <a-form layout="vertical">
        <a-form-item label="名称"><a-input v-model:value="goodsTypeForm.name" /></a-form-item>
        <a-form-item label="代码"><a-input v-model:value="goodsTypeForm.code" /></a-form-item>
        <a-form-item label="排序"><a-input-number v-model:value="goodsTypeForm.sort_order" :min="0" style="width: 100%" /></a-form-item>
        <a-form-item label="自动化实例">
          <a-select
            v-model:value="selectedAutomationRef"
            :options="automationOptions"
            :loading="automationLoading"
            placeholder="选择 automation 插件实例"
            show-search
            option-filter-prop="label"
            allow-clear
          />
        </a-form-item>
        <a-alert
          v-if="goodsTypeForm.automation_plugin_id && goodsTypeForm.automation_instance_id"
          type="info"
          show-icon
          :message="`绑定实例: ${goodsTypeForm.automation_plugin_id}/${goodsTypeForm.automation_instance_id}`"
          style="margin-bottom: 12px"
        />
        <a-card v-if="goodsTypeForm.automation_plugin_id && goodsTypeForm.automation_instance_id" size="small" title="自动化实例配置（插件模板）">
          <a-spin :spinning="automationConfigLoading">
            <a-alert v-if="automationConfigError" type="error" :message="automationConfigError" show-icon style="margin-bottom: 10px" />
            <JsonSchemaForm
              v-if="automationConfigSchema"
              v-model:modelValue="automationConfigModel"
              :schema="automationConfigSchema"
              :uiSchema="automationConfigUI"
            />
          </a-spin>
        </a-card>
        <a-row :gutter="12">
          <a-col :span="12"><a-form-item label="允许升降配"><a-switch v-model:checked="goodsTypeForm.resize_enabled" /></a-form-item></a-col>
          <a-col :span="12"><a-form-item label="允许退款"><a-switch v-model:checked="goodsTypeForm.refund_enabled" /></a-form-item></a-col>
        </a-row>
        <a-form-item label="启用"><a-switch v-model:checked="goodsTypeForm.active" /></a-form-item>
      </a-form>
    </a-modal>

    <a-drawer v-model:open="regionOpen" title="地区" width="420" @close="resetRegion">
      <a-form layout="vertical">
        <a-form-item label="名称"><a-input v-model:value="regionForm.name" /></a-form-item>
        <a-form-item label="代码"><a-input v-model:value="regionForm.code" /></a-form-item>
        <a-form-item label="启用"><a-switch v-model:checked="regionForm.active" /></a-form-item>
        <a-space>
          <a-button type="primary" @click="submitRegion">保存</a-button>
          <a-button @click="regionOpen = false">取消</a-button>
        </a-space>
      </a-form>
    </a-drawer>

    <a-drawer v-model:open="lineOpen" title="线路" width="520" @close="resetLine">
      <a-form layout="vertical">
        <a-form-item label="地区"><a-select v-model:value="lineForm.region_id">
            <a-select-option v-for="region in regions" :key="region.id" :value="region.id">{{ region.name }}</a-select-option>
          </a-select></a-form-item>
        <a-form-item label="线路名称"><a-input v-model:value="lineForm.name" /></a-form-item>
        <a-form-item label="云线路 ID"><a-input v-model:value="lineForm.line_id" /></a-form-item>
        <a-row :gutter="12">
          <a-col :span="12"><a-form-item label="CPU 单价"><a-input-number v-model:value="lineForm.unit_core" :min="0" style="width: 100%" /></a-form-item></a-col>
          <a-col :span="12"><a-form-item label="内存单价"><a-input-number v-model:value="lineForm.unit_mem" :min="0" style="width: 100%" /></a-form-item></a-col>
        </a-row>
        <a-row :gutter="12">
          <a-col :span="12"><a-form-item label="磁盘单价"><a-input-number v-model:value="lineForm.unit_disk" :min="0" style="width: 100%" /></a-form-item></a-col>
          <a-col :span="12"><a-form-item label="带宽单价"><a-input-number v-model:value="lineForm.unit_bw" :min="0" style="width: 100%" /></a-form-item></a-col>
        </a-row>
        <a-divider />
        <div class="section-title">附加项范围</div>
        <div class="subtle" style="margin-bottom: 8px">规则：-1 表示禁用该附加项，0 表示不限制上限/下限。</div>
        <a-row :gutter="12">
          <a-col :span="8"><a-form-item label="CPU最小"><a-input-number v-model:value="lineForm.add_core_min" :min="-1" style="width: 100%" /></a-form-item></a-col>
          <a-col :span="8"><a-form-item label="CPU最大"><a-input-number v-model:value="lineForm.add_core_max" :min="0" style="width: 100%" /></a-form-item></a-col>
          <a-col :span="8"><a-form-item label="CPU步进"><a-input-number v-model:value="lineForm.add_core_step" :min="1" style="width: 100%" /></a-form-item></a-col>
        </a-row>
        <a-row :gutter="12">
          <a-col :span="8"><a-form-item label="内存最小"><a-input-number v-model:value="lineForm.add_mem_min" :min="-1" style="width: 100%" /></a-form-item></a-col>
          <a-col :span="8"><a-form-item label="内存最大"><a-input-number v-model:value="lineForm.add_mem_max" :min="0" style="width: 100%" /></a-form-item></a-col>
          <a-col :span="8"><a-form-item label="内存步进"><a-input-number v-model:value="lineForm.add_mem_step" :min="1" style="width: 100%" /></a-form-item></a-col>
        </a-row>
        <a-row :gutter="12">
          <a-col :span="8"><a-form-item label="磁盘最小"><a-input-number v-model:value="lineForm.add_disk_min" :min="-1" style="width: 100%" /></a-form-item></a-col>
          <a-col :span="8"><a-form-item label="磁盘最大"><a-input-number v-model:value="lineForm.add_disk_max" :min="0" style="width: 100%" /></a-form-item></a-col>
          <a-col :span="8"><a-form-item label="磁盘步进"><a-input-number v-model:value="lineForm.add_disk_step" :min="1" style="width: 100%" /></a-form-item></a-col>
        </a-row>
        <a-row :gutter="12">
          <a-col :span="8"><a-form-item label="带宽最小"><a-input-number v-model:value="lineForm.add_bw_min" :min="-1" style="width: 100%" /></a-form-item></a-col>
          <a-col :span="8"><a-form-item label="带宽最大"><a-input-number v-model:value="lineForm.add_bw_max" :min="0" style="width: 100%" /></a-form-item></a-col>
          <a-col :span="8"><a-form-item label="带宽步进"><a-input-number v-model:value="lineForm.add_bw_step" :min="1" style="width: 100%" /></a-form-item></a-col>
        </a-row>
        <a-form-item label="启用"><a-switch v-model:checked="lineForm.active" /></a-form-item>
        <a-form-item label="可见"><a-switch v-model:checked="lineForm.visible" /></a-form-item>
        <a-form-item label="余量">
          <a-input-number v-model:value="lineForm.capacity_remaining" :min="-1" style="width: 100%" />
          <div class="subtle" style="margin-top: 6px">负数表示不限，0 表示售罄</div>
        </a-form-item>
        <a-form-item label="可用镜像">
          <a-select v-model:value="lineForm.image_ids" mode="multiple" placeholder="选择该线路可用镜像">
            <a-select-option v-for="img in systemImages" :key="img.id" :value="img.id">
              {{ img.name }} ({{ img.type || "-" }})
            </a-select-option>
          </a-select>
          <div class="subtle" style="margin-top: 6px">同步会更新该线路可用镜像</div>
        </a-form-item>
        <a-space>
          <a-button type="primary" @click="submitLine">保存</a-button>
          <a-button @click="lineOpen = false">取消</a-button>
        </a-space>
      </a-form>
    </a-drawer>

    <a-drawer v-model:open="packageOpen" title="套餐" width="420" @close="resetPackage">
      <a-form layout="vertical">
        <a-form-item label="名称"><a-input v-model:value="packageForm.name" /></a-form-item>
        <a-form-item label="线路">
          <a-select v-model:value="packageForm.plan_group_id" placeholder="选择线路">
            <a-select-option v-for="line in lines" :key="line.id" :value="line.id">
              {{ line.name }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-row :gutter="12">
          <a-col :span="12"><a-form-item label="CPU"><a-input-number v-model:value="packageForm.cores" :min="1" style="width: 100%" /></a-form-item></a-col>
          <a-col :span="12"><a-form-item label="内存(GB)"><a-input-number v-model:value="packageForm.memory_gb" :min="1" style="width: 100%" /></a-form-item></a-col>
        </a-row>
        <a-row :gutter="12">
          <a-col :span="12"><a-form-item label="磁盘(GB)"><a-input-number v-model:value="packageForm.disk_gb" :min="10" style="width: 100%" /></a-form-item></a-col>
          <a-col :span="12"><a-form-item label="带宽(Mbps)"><a-input-number v-model:value="packageForm.bandwidth_mbps" :min="1" style="width: 100%" /></a-form-item></a-col>
        </a-row>
        <div class="subtle" style="margin-top: -8px; margin-bottom: 8px">实践建议：套餐磁盘大小尽量统一，可降低升降配中的容量兼容问题。</div>
        <a-row :gutter="12">
          <a-col :span="12"><a-form-item label="CPU 型号"><a-input v-model:value="packageForm.cpu_model" /></a-form-item></a-col>
          <a-col :span="12"><a-form-item label="端口数"><a-input-number v-model:value="packageForm.port_num" :min="0" style="width: 100%" /></a-form-item></a-col>
        </a-row>
        <a-form-item label="月费"><a-input-number v-model:value="packageForm.monthly_price" :min="0" style="width: 100%" /></a-form-item>
        <a-form-item label="启用"><a-switch v-model:checked="packageForm.active" /></a-form-item>
        <a-form-item label="可见"><a-switch v-model:checked="packageForm.visible" /></a-form-item>
        <a-form-item label="余量">
          <a-input-number v-model:value="packageForm.capacity_remaining" :min="-1" style="width: 100%" />
          <div class="subtle" style="margin-top: 6px">负数表示不限，0 表示售罄</div>
        </a-form-item>
        <a-space>
          <a-button type="primary" @click="submitPackage">保存</a-button>
          <a-button @click="packageOpen = false">取消</a-button>
        </a-space>
      </a-form>
    </a-drawer>

    <a-drawer v-model:open="imageOpen" title="系统镜像" width="420" @close="resetImage">
      <a-form layout="vertical">
        <a-form-item label="镜像 ID">
          <a-input-number v-model:value="imageForm.image_id" :min="1" :precision="0" style="width: 100%" />
        </a-form-item>
        <a-form-item label="名称"><a-input v-model:value="imageForm.name" /></a-form-item>
        <a-form-item label="类型">
          <a-select v-model:value="imageForm.type" :options="IMAGE_TYPE_OPTIONS" placeholder="请选择镜像类型" />
        </a-form-item>
        <a-form-item label="启用"><a-switch v-model:checked="imageForm.enabled" /></a-form-item>
        <a-space>
          <a-button type="primary" @click="submitImage">保存</a-button>
          <a-button @click="imageOpen = false">取消</a-button>
        </a-space>
      </a-form>
    </a-drawer>

    <a-drawer v-model:open="cycleOpen" title="计费周期" width="420" @close="resetCycle">
      <a-form layout="vertical">
        <a-form-item label="名称"><a-input v-model:value="cycleForm.name" /></a-form-item>
        <a-form-item label="月数"><a-input-number v-model:value="cycleForm.months" :min="1" style="width: 100%" /></a-form-item>
        <a-form-item label="倍率"><a-input-number v-model:value="cycleForm.multiplier" :min="0" style="width: 100%" /></a-form-item>
        <a-row :gutter="12">
          <a-col :span="12"><a-form-item label="最小数量"><a-input-number v-model:value="cycleForm.min_qty" :min="1" style="width: 100%" /></a-form-item></a-col>
          <a-col :span="12"><a-form-item label="最大数量"><a-input-number v-model:value="cycleForm.max_qty" :min="1" style="width: 100%" /></a-form-item></a-col>
        </a-row>
        <a-form-item label="启用"><a-switch v-model:checked="cycleForm.active" /></a-form-item>
        <a-space>
          <a-button type="primary" @click="submitCycle">保存</a-button>
          <a-button @click="cycleOpen = false">取消</a-button>
        </a-space>
      </a-form>
    </a-drawer>

    <a-modal v-model:open="batchOpen" title="批量生成套餐" width="920" :footer="null" @cancel="closeBatch">
      <a-form layout="vertical">
        <a-divider>基础配置</a-divider>
        <a-row :gutter="12">
          <a-col :span="8">
            <a-form-item label="线路">
              <a-select v-model:value="batchForm.plan_group_id" placeholder="选择线路">
                <a-select-option v-for="line in lines" :key="line.id" :value="line.id">
                  {{ line.name }}
                </a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="定价倍率">
              <a-input-number v-model:value="batchForm.price_multiplier" :min="0.1" :step="0.1" style="width: 100%" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="端口数">
              <a-input-number v-model:value="batchForm.port_num" :min="0" style="width: 100%" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="12">
          <a-col :span="6">
            <a-form-item label="线路总成本">
              <a-input-number v-model:value="batchForm.total_cost" :min="0" style="width: 100%" />
            </a-form-item>
          </a-col>
          <a-col :span="6"><a-form-item label="CPU 型号"><a-input v-model:value="batchForm.cpu_model" /></a-form-item></a-col>
          <a-col :span="6"><a-form-item label="启用"><a-switch v-model:checked="batchForm.active" /></a-form-item></a-col>
          <a-col :span="6"><a-form-item label="可见"><a-switch v-model:checked="batchForm.visible" /></a-form-item></a-col>
        </a-row>
        <a-divider>资源规则</a-divider>
        <a-row :gutter="12">
          <a-col :span="8"><a-form-item label="CPU 最小"><a-input-number v-model:value="batchForm.cpu_min" :min="1" style="width: 100%" /></a-form-item></a-col>
          <a-col :span="8"><a-form-item label="CPU 最大"><a-input-number v-model:value="batchForm.cpu_max" :min="1" style="width: 100%" /></a-form-item></a-col>
          <a-col :span="8"><a-form-item label="CPU 步进"><a-input-number v-model:value="batchForm.cpu_step" :min="1" style="width: 100%" /></a-form-item></a-col>
        </a-row>
        <a-row :gutter="12">
          <a-col :span="8"><a-form-item label="内存比率(GB/核)"><a-input-number v-model:value="batchForm.memory_ratio" :min="0.5" :step="0.5" style="width: 100%" /></a-form-item></a-col>
          <a-col :span="8"><a-form-item label="内存最小"><a-input-number v-model:value="batchForm.memory_min" :min="1" style="width: 100%" /></a-form-item></a-col>
          <a-col :span="8"><a-form-item label="内存最大"><a-input-number v-model:value="batchForm.memory_max" :min="1" style="width: 100%" /></a-form-item></a-col>
        </a-row>
        <a-row :gutter="12">
          <a-col :span="8"><a-form-item label="存储最小(GB)"><a-input-number v-model:value="batchForm.disk_min" :min="1" style="width: 100%" /></a-form-item></a-col>
          <a-col :span="8"><a-form-item label="存储最大(GB)"><a-input-number v-model:value="batchForm.disk_max" :min="1" style="width: 100%" /></a-form-item></a-col>
          <a-col :span="8"><a-form-item label="存储步进(GB)"><a-input-number v-model:value="batchForm.disk_step" :min="1" style="width: 100%" /></a-form-item></a-col>
        </a-row>
        <a-row :gutter="12">
          <a-col :span="8"><a-form-item label="带宽最小(Mbps)"><a-input-number v-model:value="batchForm.bw_min" :min="1" style="width: 100%" /></a-form-item></a-col>
          <a-col :span="8"><a-form-item label="带宽最大(Mbps)"><a-input-number v-model:value="batchForm.bw_max" :min="1" style="width: 100%" /></a-form-item></a-col>
          <a-col :span="8"><a-form-item label="带宽步进(Mbps)"><a-input-number v-model:value="batchForm.bw_step" :min="1" style="width: 100%" /></a-form-item></a-col>
        </a-row>

        <a-divider>容量计算</a-divider>
        <a-row :gutter="12">
          <a-col :span="6"><a-form-item label="CPU 总量"><a-input-number v-model:value="batchForm.total_cores" :min="0" style="width: 100%" /></a-form-item></a-col>
          <a-col :span="6"><a-form-item label="内存总量(GB)"><a-input-number v-model:value="batchForm.total_mem" :min="0" style="width: 100%" /></a-form-item></a-col>
          <a-col :span="6"><a-form-item label="存储总量(GB)"><a-input-number v-model:value="batchForm.total_disk" :min="0" style="width: 100%" /></a-form-item></a-col>
          <a-col :span="6"><a-form-item label="带宽总量(Mbps)"><a-input-number v-model:value="batchForm.total_bw" :min="0" style="width: 100%" /></a-form-item></a-col>
        </a-row>
        <a-row :gutter="12">
          <a-col :span="8"><a-form-item label="是否超开"><a-switch v-model:checked="batchForm.overcommit_enabled" /></a-form-item></a-col>
          <a-col :span="8"><a-form-item label="超开倍率"><a-input-number v-model:value="batchForm.overcommit_ratio" :min="1" :step="0.1" style="width: 100%" :disabled="!batchForm.overcommit_enabled" /></a-form-item></a-col>
          <a-col :span="8">
            <div class="subtle" style="margin-top: 30px">用于计算余量，留空表示不限制</div>
          </a-col>
        </a-row>

        <a-space style="margin-bottom: 16px">
          <a-button type="primary" @click="generatePackages">生成套餐</a-button>
          <a-button @click="clearGenerated">清空预览</a-button>
        </a-space>

        <a-table :columns="batchColumns" :data-source="generatedPackages" row-key="key" :pagination="false" size="small">
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'capacity_remaining'">
              {{ formatCapacity(record.capacity_remaining) }}
            </template>
          </template>
        </a-table>

        <div style="margin-top: 16px; text-align: right">
          <a-space>
            <a-button :disabled="!generatedPackages.length" type="primary" @click="applyGenerated">应用生成</a-button>
            <a-button @click="closeBatch">关闭</a-button>
          </a-space>
        </div>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from "vue";
import { CodeOutlined, WindowsOutlined } from "@ant-design/icons-vue";
import JsonSchemaForm from "@/components/forms/JsonSchemaForm.vue";
import {
  listRegions,
  createRegion,
  updateRegion,
  deleteRegion,
  bulkDeleteRegions,
  listLines,
  createLine,
  updateLine,
  deleteLine,
  bulkDeleteLines,
  listPackages,
  createPackage,
  updatePackage,
  deletePackage,
  bulkDeletePackages,
  listSystemImages,
  createSystemImage,
  updateSystemImage,
  deleteSystemImage,
  bulkDeleteSystemImages,
  setLineSystemImages,
  syncSystemImages,
  listBillingCycles,
  createBillingCycle,
  updateBillingCycle,
  deleteBillingCycle,
  bulkDeleteBillingCycles,
  listGoodsTypes,
  syncGoodsTypeAutomation,
  getGoodsTypeAutomationOptions,
  getGoodsTypeCapabilities,
  updateGoodsTypeCapabilities,
  createGoodsType,
  updateGoodsType,
  deleteGoodsType,
  listAdminPlugins,
  getAdminPluginInstanceConfigSchema,
  getAdminPluginInstanceConfig,
  updateAdminPluginInstanceConfig
} from "@/services/admin";
import { message, Modal } from "ant-design-vue";

const regions = ref([]);
const lines = ref([]);
const packages = ref([]);
const systemImages = ref([]);
const billingCycles = ref([]);

const goodsTypes = ref<any[]>([]);
const goodsTypeId = ref<any>(null);
const goodsTypeOptions = computed(() =>
  (goodsTypes.value || [])
    .filter((gt) => gt.active !== false)
    .sort((a, b) => {
      const sa = Number(a.sort_order ?? 0);
      const sb = Number(b.sort_order ?? 0);
      if (sa !== sb) return sa - sb;
      return Number(a.id ?? 0) - Number(b.id ?? 0);
    })
    .map((gt) => ({ label: gt.name, value: gt.id }))
);

// 判断当前选中商品类型是否为 openidc 插件（地区/线路只读）
const isOpenIDCPlugin = computed(() => {
  if (!goodsTypeId.value) return false;
  const gt = goodsTypes.value.find((item) => Number(item.id) === Number(goodsTypeId.value));
  if (!gt) return false;
  const pluginId = String(gt.automation_plugin_id || "").toLowerCase();
  return pluginId === "openidc_default" || pluginId.includes("openidc");
});

const goodsTypeColumns = [
  { title: "ID", dataIndex: "id", key: "id" },
  { title: "名称", dataIndex: "name", key: "name" },
  { title: "代码", dataIndex: "code", key: "code" },
  { title: "automation_plugin_id", dataIndex: "automation_plugin_id", key: "automation_plugin_id" },
  { title: "automation_instance_id", dataIndex: "automation_instance_id", key: "automation_instance_id" },
  { title: "排序", dataIndex: "sort_order", key: "sort_order" },
  { title: "启用", dataIndex: "active", key: "active" },
  { title: "操作", key: "action" }
];

const selectedRegionKeys = ref([]);
const selectedLineKeys = ref([]);
const selectedPackageKeys = ref([]);
const selectedImageKeys = ref([]);
const selectedCycleKeys = ref([]);

const packageLineId = ref(null);
const imageLineId = ref(null);
const lineScopedImages = ref([]);
const lineImageCountMap = ref<Record<number, number>>({});
const batchOpen = ref(false);
const generatedPackages = ref([]);

const filteredPackages = computed(() => {
  if (!packageLineId.value || packageLineId.value === "all") return packages.value;
  const target = Number(packageLineId.value);
  return packages.value.filter((item) => Number(item.plan_group_id) === target);
});

const filteredImages = computed(() => {
  if (!imageLineId.value) return systemImages.value;
  return lineScopedImages.value;
});

const regionSelection = computed(() => ({
  selectedRowKeys: selectedRegionKeys.value,
  onChange: (keys) => {
    selectedRegionKeys.value = keys;
  }
}));
const lineSelection = computed(() => ({
  selectedRowKeys: selectedLineKeys.value,
  onChange: (keys) => {
    selectedLineKeys.value = keys;
  }
}));
const packageSelection = computed(() => ({
  selectedRowKeys: selectedPackageKeys.value,
  onChange: (keys) => {
    selectedPackageKeys.value = keys;
  }
}));
const imageSelection = computed(() => ({
  selectedRowKeys: selectedImageKeys.value,
  onChange: (keys) => {
    selectedImageKeys.value = keys;
  }
}));
const cycleSelection = computed(() => ({
  selectedRowKeys: selectedCycleKeys.value,
  onChange: (keys) => {
    selectedCycleKeys.value = keys;
  }
}));

const regionOpen = ref(false);
const lineOpen = ref(false);
const packageOpen = ref(false);
const imageOpen = ref(false);
const cycleOpen = ref(false);

const goodsTypeOpen = ref(false);
const goodsTypeSaving = ref(false);
const automationLoading = ref(false);
const automationOptions = ref<{ label: string; value: string }[]>([]);
const selectedAutomationRef = ref<string>("");
const automationConfigLoading = ref(false);
const automationConfigError = ref("");
const automationConfigSchema = ref<any>(null);
const automationConfigUI = ref<any>({});
const automationConfigModel = ref<Record<string, any>>({});

const goodsTypeForm = reactive({
  id: null,
  code: "",
  name: "",
  active: true,
  sort_order: 0,
  automation_plugin_id: "",
  automation_instance_id: "",
  resize_enabled: true,
  refund_enabled: true
});

const safeJson = (s: string) => {
  try {
    return JSON.parse(String(s || "{}"));
  } catch {
    return null;
  }
};

const toOptionValue = (value: any) => {
  const num = Number(value);
  if (Number.isFinite(num) && Number.isInteger(num)) return num;
  return String(value ?? "").trim();
};

const applyEnumOptions = (schemaObj: any, uiObj: any, field: string, options: Array<{ value: any; label: string }>) => {
  if (!schemaObj || !schemaObj.properties || !schemaObj.properties[field] || !Array.isArray(options) || options.length === 0) {
    return;
  }
  const prop = schemaObj.properties[field];
  prop.enum = options.map((item) => item.value);
  prop.enumNames = options.map((item) => item.label);
  uiObj[field] = {
    ...(uiObj[field] || {}),
    "ui:widget": "select"
  };
};

const regionForm = reactive({ id: null, goods_type_id: null, name: "", code: "", active: true });
const lineForm = reactive({
  id: null,
  region_id: null,
  name: "",
  line_id: "",
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
  image_ids: []
});
const packageForm = reactive({
  id: null,
  name: "",
  plan_group_id: null,
  cores: 1,
  memory_gb: 1,
  disk_gb: 20,
  bandwidth_mbps: 1,
  cpu_model: "",
  port_num: 30,
  monthly_price: 0,
  active: true,
  visible: true,
  capacity_remaining: -1
});
const imageForm = reactive({ id: null, image_id: null, name: "", type: "linux", enabled: true });
const cycleForm = reactive({ id: null, name: "", months: 1, multiplier: 1, min_qty: 1, max_qty: 12, active: true });
const IMAGE_TYPE_OPTIONS = [
  { label: "Linux", value: "linux" },
  { label: "Windows", value: "windows" }
];

const formatCapacity = (value) => {
  const num = Number(value);
  if (!Number.isFinite(num)) return "-";
  if (num < 0) return "不限";
  if (num === 0) return "售罄";
  return String(num);
};

const capacityTagColor = (value) => {
  const num = Number(value);
  if (!Number.isFinite(num)) return "default";
  if (num < 0) return "green";
  if (num === 0) return "red";
  return "blue";
};

const lineNameById = (id) => {
  const match = lines.value.find((item) => Number(item.id) === Number(id));
  return match?.name || String(id || "-");
};

const regionNameById = (id) => {
  const match = regions.value.find((item) => Number(item.id) === Number(id));
  return match?.name || String(id || "-");
};

const toPositiveInt = (value) => {
  const num = Number(value);
  if (!Number.isFinite(num) || num <= 0) return null;
  return Math.trunc(num);
};

const resolveCloudLineId = (value) => {
  const numeric = toPositiveInt(value);
  if (!numeric) return null;
  const byId = lines.value.find((item) => Number(item.id) === numeric);
  const byIdLine = toPositiveInt(byId?.line_id);
  if (byIdLine) return byIdLine;
  const byLineId = lines.value.find((item) => Number(item.line_id) === numeric);
  const mapped = toPositiveInt(byLineId?.line_id);
  if (mapped) return mapped;
  return null;
};

const lineImageCountByLineId = (lineId) => {
  const id = Number(lineId);
  if (!Number.isFinite(id) || id <= 0) return 0;
  return Number(lineImageCountMap.value[id] || 0);
};

const LINE_NUMERIC_FIELDS = new Set([
  "region_id",
  "line_id",
  "unit_core",
  "unit_mem",
  "unit_disk",
  "unit_bw",
  "add_core_min",
  "add_core_max",
  "add_core_step",
  "add_mem_min",
  "add_mem_max",
  "add_mem_step",
  "add_disk_min",
  "add_disk_max",
  "add_disk_step",
  "add_bw_min",
  "add_bw_max",
  "add_bw_step",
  "capacity_remaining",
  "sort_order"
]);

const normalizeLinePayload = (source) => {
  const input = (source || {}) as Record<string, any>;
  const payload: Record<string, any> = {};
  Object.keys(input).forEach((key) => {
    if (key === "id" || key === "image_ids") return;
    const value = input[key];
    if (value === undefined || value === null || value === "") return;
    if (LINE_NUMERIC_FIELDS.has(key)) {
      const num = Number(value);
      if (Number.isFinite(num)) payload[key] = num;
      return;
    }
    payload[key] = value;
  });
  return payload;
};

const isWindowsType = (value) => String(value || "").toLowerCase().includes("win");
const isLinuxType = (value) => String(value || "").toLowerCase().includes("linux");
const formatImageType = (value) => (value ? String(value) : "-");
const typeTagColor = (value) => {
  if (isWindowsType(value)) return "blue";
  if (isLinuxType(value)) return "green";
  return "default";
};

const sortByString = (key) => (a, b) => String(a[key] ?? "").localeCompare(String(b[key] ?? ""));
const sortByNumber = (key) => (a, b) => Number(a[key] ?? 0) - Number(b[key] ?? 0);

const loadLineImages = async (lineId) => {
  const cloudLineId = resolveCloudLineId(lineId);
  if (!cloudLineId) {
    lineForm.image_ids = [];
    return;
  }
  const res = await listSystemImages({ line_id: cloudLineId });
  const items = res.data?.items || [];
  lineForm.image_ids = items.map((row) => row.id ?? row.ID).filter(Boolean);
};

const mapSystemImageRow = (row) => ({
  id: row.id ?? row.ID,
  image_id: row.image_id ?? row.ImageID,
  name: row.name ?? row.Name,
  type: row.type ?? row.Type,
  enabled: row.enabled ?? row.Enabled
});

const loadScopedImages = async () => {
  if (!imageLineId.value) {
    lineScopedImages.value = [];
    return;
  }
  const cloudLineId = resolveCloudLineId(imageLineId.value);
  if (!cloudLineId) {
    lineScopedImages.value = [];
    return;
  }
  const res = await listSystemImages({ line_id: cloudLineId });
  lineScopedImages.value = (res.data?.items || []).map(mapSystemImageRow);
};

const loadLineImageCounts = async () => {
  const map: Record<number, number> = {};
  const tasks = (lines.value || []).map(async (line) => {
    const localLineId = Number(line?.id || 0);
    if (!Number.isFinite(localLineId) || localLineId <= 0) return;
    const cloudLineId = resolveCloudLineId(localLineId);
    if (!cloudLineId) {
      map[localLineId] = 0;
      return;
    }
    const res = await listSystemImages({ line_id: cloudLineId });
    map[localLineId] = Array.isArray(res.data?.items) ? res.data.items.length : 0;
  });
  await Promise.all(tasks);
  lineImageCountMap.value = map;
};

const regionColumns = [
  { title: "ID", dataIndex: "id", key: "id", sorter: sortByNumber("id") },
  { title: "名称", dataIndex: "name", key: "name", sorter: sortByString("name") },
  { title: "代码", dataIndex: "code", key: "code", sorter: sortByString("code") },
  { title: "状态", dataIndex: "active", key: "active", sorter: sortByNumber("active") },
  { title: "操作", key: "action" }
];

const lineColumns = [
  { title: "ID", dataIndex: "id", key: "id", sorter: sortByNumber("id") },
  {
    title: "地区",
    dataIndex: "region_id",
    key: "region_id",
    sorter: (a, b) => regionNameById(a.region_id).localeCompare(regionNameById(b.region_id))
  },
  { title: "名称", dataIndex: "name", key: "name", sorter: sortByString("name") },
  { title: "云线路 ID", dataIndex: "line_id", key: "line_id", sorter: sortByNumber("line_id") },
  { title: "CPU单价", dataIndex: "unit_core", key: "unit_core", sorter: sortByNumber("unit_core") },
  { title: "内存单价", dataIndex: "unit_mem", key: "unit_mem", sorter: sortByNumber("unit_mem") },
  { title: "磁盘单价", dataIndex: "unit_disk", key: "unit_disk", sorter: sortByNumber("unit_disk") },
  { title: "带宽单价", dataIndex: "unit_bw", key: "unit_bw", sorter: sortByNumber("unit_bw") },
  { title: "可见", dataIndex: "visible", key: "visible", sorter: sortByNumber("visible") },
  { title: "余量", dataIndex: "capacity_remaining", key: "capacity_remaining", sorter: sortByNumber("capacity_remaining") },
  { title: "启用", dataIndex: "active", key: "active", sorter: sortByNumber("active") },
  { title: "操作", key: "action" }
];

const packageColumns = [
  { title: "ID", dataIndex: "id", key: "id", sorter: sortByNumber("id") },
  { title: "名称", dataIndex: "name", key: "name", sorter: sortByString("name") },
  {
    title: "线路",
    dataIndex: "plan_group_id",
    key: "plan_group_id",
    sorter: (a, b) => lineNameById(a.plan_group_id).localeCompare(lineNameById(b.plan_group_id))
  },
  { title: "月费", dataIndex: "monthly_price", key: "monthly_price", sorter: sortByNumber("monthly_price") },
  { title: "端口数", dataIndex: "port_num", key: "port_num", sorter: sortByNumber("port_num") },
  { title: "可见", dataIndex: "visible", key: "visible", sorter: sortByNumber("visible") },
  { title: "余量", dataIndex: "capacity_remaining", key: "capacity_remaining", sorter: sortByNumber("capacity_remaining") },
  { title: "启用", dataIndex: "active", key: "active", sorter: sortByNumber("active") },
  { title: "操作", key: "action" }
];

const imageColumns = [
  { title: "ID", dataIndex: "id", key: "id", sorter: sortByNumber("id") },
  { title: "镜像 ID", dataIndex: "image_id", key: "image_id", sorter: sortByString("image_id") },
  { title: "名称", dataIndex: "name", key: "name", sorter: sortByString("name") },
  { title: "类型", dataIndex: "type", key: "type", sorter: sortByString("type") },
  { title: "启用", dataIndex: "enabled", key: "enabled", sorter: sortByNumber("enabled") },
  { title: "操作", key: "action" }
];

const cycleColumns = [
  { title: "ID", dataIndex: "id", key: "id", sorter: sortByNumber("id") },
  { title: "名称", dataIndex: "name", key: "name", sorter: sortByString("name") },
  { title: "月数", dataIndex: "months", key: "months", sorter: sortByNumber("months") },
  { title: "倍率", dataIndex: "multiplier", key: "multiplier", sorter: sortByNumber("multiplier") },
  { title: "最小数量", dataIndex: "min_qty", key: "min_qty", sorter: sortByNumber("min_qty") },
  { title: "最大数量", dataIndex: "max_qty", key: "max_qty", sorter: sortByNumber("max_qty") },
  { title: "操作", key: "action" }
];

const load = async () => {
  if (!goodsTypeId.value) {
    regions.value = [];
    lines.value = [];
    packages.value = [];
    return;
  }
  const goodsTypeParam = goodsTypeId.value ? { goods_type_id: goodsTypeId.value } : undefined;
  const [regionRes, lineRes, packageRes, imageRes, cycleRes] = await Promise.all([
    listRegions(goodsTypeParam),
    listLines(goodsTypeParam),
    listPackages(goodsTypeParam),
    listSystemImages(),
    listBillingCycles()
  ]);
  regions.value = (regionRes.data?.items || []).map((row) => ({
    id: row.id ?? row.ID,
    goods_type_id: row.goods_type_id ?? row.GoodsTypeID,
    name: row.name ?? row.Name,
    code: row.code ?? row.Code,
    active: row.active ?? row.Active
  }));
  lines.value = (lineRes.data?.items || []).map((row) => ({
    id: row.id ?? row.ID,
    goods_type_id: row.goods_type_id ?? row.GoodsTypeID,
    region_id: row.region_id ?? row.RegionID,
    name: row.name ?? row.Name ?? row.line_name ?? row.LineName,
    line_id: row.line_id ?? row.LineID,
    unit_core: row.unit_core ?? row.UnitCore,
    unit_mem: row.unit_mem ?? row.UnitMem,
    unit_disk: row.unit_disk ?? row.UnitDisk,
    unit_bw: row.unit_bw ?? row.UnitBW,
    add_core_min: row.add_core_min ?? row.AddCoreMin,
    add_core_max: row.add_core_max ?? row.AddCoreMax,
    add_core_step: row.add_core_step ?? row.AddCoreStep,
    add_mem_min: row.add_mem_min ?? row.AddMemMin,
    add_mem_max: row.add_mem_max ?? row.AddMemMax,
    add_mem_step: row.add_mem_step ?? row.AddMemStep,
    add_disk_min: row.add_disk_min ?? row.AddDiskMin,
    add_disk_max: row.add_disk_max ?? row.AddDiskMax,
    add_disk_step: row.add_disk_step ?? row.AddDiskStep,
    add_bw_min: row.add_bw_min ?? row.AddBwMin,
    add_bw_max: row.add_bw_max ?? row.AddBwMax,
    add_bw_step: row.add_bw_step ?? row.AddBwStep,
    active: row.active ?? row.Active,
    visible: row.visible ?? row.Visible,
    capacity_remaining: row.capacity_remaining ?? row.CapacityRemaining
  }));
  packages.value = (packageRes.data?.items || []).map((row) => ({
    id: row.id ?? row.ID,
    name: row.name ?? row.Name,
    goods_type_id: row.goods_type_id ?? row.GoodsTypeID,
    plan_group_id: row.plan_group_id ?? row.PlanGroupID,
    cores: row.cores ?? row.Cores,
    memory_gb: row.memory_gb ?? row.MemoryGB,
    disk_gb: row.disk_gb ?? row.DiskGB,
    bandwidth_mbps: row.bandwidth_mbps ?? row.BandwidthMB,
    cpu_model: row.cpu_model ?? row.CPUModel,
    port_num: row.port_num ?? row.PortNum,
    monthly_price: row.monthly_price ?? row.Monthly,
    active: row.active ?? row.Active,
    visible: row.visible ?? row.Visible,
    capacity_remaining: row.capacity_remaining ?? row.CapacityRemaining
  }));
  systemImages.value = (imageRes.data?.items || []).map((row) => ({
    id: row.id ?? row.ID,
    image_id: row.image_id ?? row.ImageID,
    name: row.name ?? row.Name,
    type: row.type ?? row.Type,
    enabled: row.enabled ?? row.Enabled
  }));
  billingCycles.value = (cycleRes.data?.items || []).map((row) => ({
    id: row.id ?? row.ID,
    name: row.name ?? row.Name,
    months: row.months ?? row.Months,
    multiplier: row.multiplier ?? row.Multiplier,
    min_qty: row.min_qty ?? row.MinQty,
    max_qty: row.max_qty ?? row.MaxQty
  }));
  await loadLineImageCounts();
  await loadScopedImages();
};

const loadGoodsTypeList = async () => {
  const res = await listGoodsTypes();
  goodsTypes.value = (res.data?.items || []).map((row: any) => ({
    id: row.id ?? row.ID,
    code: row.code ?? row.Code,
    name: row.name ?? row.Name,
    active: row.active ?? row.Active,
    sort_order: row.sort_order ?? row.SortOrder,
    automation_plugin_id: row.automation_plugin_id ?? row.AutomationPluginID,
    automation_instance_id: row.automation_instance_id ?? row.AutomationInstanceID
  }));
  if (!goodsTypeId.value && goodsTypeOptions.value.length) {
    goodsTypeId.value = goodsTypeOptions.value[0].value;
  }
};

const loadAutomationInstances = async () => {
  automationLoading.value = true;
  try {
    const res = await listAdminPlugins();
    const items = (res.data?.items || []).filter((item: any) => String(item.category || "") === "automation");
    automationOptions.value = items.map((item: any) => {
      const pluginID = String(item.plugin_id || "");
      const instanceID = String(item.instance_id || "default");
      const enabled = !!(item.enabled ?? item.Enabled);
      return {
        value: `${pluginID}:${instanceID}`,
        label: `${pluginID}/${instanceID}${enabled ? "（启用）" : "（未启用）"}`
      };
    });
    if (!selectedAutomationRef.value && automationOptions.value.length > 0) {
      selectedAutomationRef.value = automationOptions.value[0].value;
    }
  } finally {
    automationLoading.value = false;
  }
};

const loadAutomationConfigTemplate = async () => {
  automationConfigSchema.value = null;
  automationConfigUI.value = {};
  automationConfigModel.value = {};
  automationConfigError.value = "";
  const pluginID = String(goodsTypeForm.automation_plugin_id || "").trim();
  const instanceID = String(goodsTypeForm.automation_instance_id || "").trim();
  if (!pluginID || !instanceID) return;
  automationConfigLoading.value = true;
  try {
    const [schemaRes, cfgRes] = await Promise.all([
      getAdminPluginInstanceConfigSchema("automation", pluginID, instanceID),
      getAdminPluginInstanceConfig("automation", pluginID, instanceID)
    ]);
    const schemaObj = safeJson(schemaRes.data?.json_schema || "{}");
    const uiObj = safeJson(schemaRes.data?.ui_schema || "{}") || {};
    const cfgObj = safeJson(cfgRes.data?.config_json || "{}") || {};
    if (!schemaObj || String(schemaObj.type || "") !== "object") {
      automationConfigError.value = "插件未返回有效 object schema";
      return;
    }
    let properties = (schemaObj && schemaObj.properties) || {};
    const pluginIDNorm = String(pluginID || "").trim().toLowerCase();
    const isXiaoheiProxy = pluginIDNorm === "xiaohei_proxy" || pluginIDNorm.endsWith("/xiaohei_proxy");
    // Hard fallback: xiaohei_proxy must expose goods_type_id in UI even if plugin schema is stale.
    if (isXiaoheiProxy && (!properties || !properties.goods_type_id)) {
      schemaObj.properties = schemaObj.properties || {};
      schemaObj.properties.goods_type_id = {
        type: "integer",
        title: "Upstream Goods Type ID",
        description: "上游商品类型ID",
        minimum: 1
      };
      const req = Array.isArray(schemaObj.required) ? schemaObj.required.map(String) : [];
      if (!req.includes("goods_type_id")) req.push("goods_type_id");
      schemaObj.required = req;
      properties = schemaObj.properties || {};
    }
    if (isXiaoheiProxy && properties && properties.goods_type_id) {
      properties.goods_type_id.description = "填写上游系统的 goods_type_id（不是本地 ID）";
      properties.goods_type_id.enum = undefined;
      properties.goods_type_id.enumNames = undefined;
      uiObj.goods_type_id = {
        ...(uiObj.goods_type_id || {}),
        "ui:widget": "updown"
      };
    }
    const currentGoodsTypeID = Number(goodsTypeForm.id || 0);
    if (Number.isFinite(currentGoodsTypeID) && currentGoodsTypeID > 0) {
      try {
        const optionsRes = await getGoodsTypeAutomationOptions(currentGoodsTypeID);
        const lineItems = optionsRes.data?.line_items || [];
        const productTypeItems = optionsRes.data?.product_type_items || lineItems;
        const packageItems = optionsRes.data?.package_items || [];
        const productItems = optionsRes.data?.product_items || packageItems;
        const billingCycleItems = optionsRes.data?.billing_cycle_items || [];
        const cancelTypeItems = optionsRes.data?.cancel_type_items || [];

        const lineOptions = lineItems
          .map((it) => {
            const id = toOptionValue(it?.id);
            if (id === "" || id === null || id === undefined) return null;
            return { value: id, label: `${id} - ${String(it?.name || "").trim() || "Line"}` };
          })
          .filter(Boolean) as Array<{ value: any; label: string }>;

        const productTypeOptions = productTypeItems
          .map((it) => {
            const id = toOptionValue(it?.id);
            if (id === "" || id === null || id === undefined) return null;
            return { value: id, label: `${id} - ${String(it?.name || "").trim() || "ProductType"}` };
          })
          .filter(Boolean) as Array<{ value: any; label: string }>;

        const packageOptions = packageItems
          .map((it) => {
            const id = toOptionValue(it?.id);
            if (id === "" || id === null || id === undefined) return null;
            const lineID = toOptionValue(it?.line_id);
            const linePrefix = lineID !== "" && lineID !== null && lineID !== undefined ? `Line ${lineID} / ` : "";
            return { value: id, label: `${linePrefix}${id} - ${String(it?.name || "").trim() || "Package"}` };
          })
          .filter(Boolean) as Array<{ value: any; label: string }>;

        const productOptions = productItems
          .map((it) => {
            const id = toOptionValue(it?.id);
            if (id === "" || id === null || id === undefined) return null;
            const lineID = toOptionValue(it?.line_id);
            const linePrefix = lineID !== "" && lineID !== null && lineID !== undefined ? `Line ${lineID} / ` : "";
            return { value: id, label: `${linePrefix}${id} - ${String(it?.name || "").trim() || "Product"}` };
          })
          .filter(Boolean) as Array<{ value: any; label: string }>;

        const billingCycleOptions = billingCycleItems
          .map((it) => {
            const value = String(it?.value || "").trim();
            if (!value) return null;
            return { value, label: String(it?.label || value) };
          })
          .filter(Boolean) as Array<{ value: any; label: string }>;

        const cancelTypeOptions = cancelTypeItems
          .map((it) => {
            const value = String(it?.value || "").trim();
            if (!value) return null;
            return { value, label: String(it?.label || value) };
          })
          .filter(Boolean) as Array<{ value: any; label: string }>;

        applyEnumOptions(schemaObj, uiObj, "line_id", lineOptions);
        applyEnumOptions(schemaObj, uiObj, "product_type_id", productTypeOptions);
        applyEnumOptions(schemaObj, uiObj, "goods_type_id", productTypeOptions);
        applyEnumOptions(schemaObj, uiObj, "upstream_goods_type_id", productTypeOptions);
        applyEnumOptions(schemaObj, uiObj, "package_id", packageOptions);
        applyEnumOptions(schemaObj, uiObj, "product_id", productOptions);
        applyEnumOptions(schemaObj, uiObj, "billing_cycle", billingCycleOptions);
        applyEnumOptions(schemaObj, uiObj, "cancel_type", cancelTypeOptions);
      } catch {
        // Keep manual input available when dynamic options cannot be loaded.
      }
    }
    automationConfigSchema.value = schemaObj;
    automationConfigUI.value = uiObj;
    automationConfigModel.value = cfgObj;
  } catch (e: any) {
    const data = e?.response?.data || {};
    const missing = Array.isArray(data?.missing_fields) ? data.missing_fields.filter(Boolean) : [];
    if (String(data?.code || "") === "missing_required_config" && missing.length > 0) {
      automationConfigError.value = `${String(data?.error || "自动化实例配置不完整")}。缺少: ${missing.join(", ")}`;
    } else {
      automationConfigError.value = data?.error || "加载自动化实例配置模板失败";
    }
  } finally {
    automationConfigLoading.value = false;
  }
};

const syncCurrentGoodsType = async () => {
  if (!goodsTypeId.value) return;
  await syncGoodsTypeAutomation(goodsTypeId.value, "merge");
  message.success("OK");
  await load();
};

const openGoodsType = async (record?: any) => {
  if (record) {
    Object.assign(goodsTypeForm, { resize_enabled: true, refund_enabled: true }, record);
    try {
      const capRes = await getGoodsTypeCapabilities(record.id);
      const caps = capRes.data || {};
      goodsTypeForm.resize_enabled = !!caps.resize_enabled;
      goodsTypeForm.refund_enabled = !!caps.refund_enabled;
    } catch {
      goodsTypeForm.resize_enabled = true;
      goodsTypeForm.refund_enabled = true;
    }
  } else {
    Object.assign(goodsTypeForm, {
      id: null,
      code: "",
      name: "",
      active: true,
      sort_order: 0,
      automation_plugin_id: "lightboat",
      automation_instance_id: "default",
      resize_enabled: true,
      refund_enabled: true
    });
  }
  selectedAutomationRef.value = goodsTypeForm.automation_plugin_id && goodsTypeForm.automation_instance_id
    ? `${goodsTypeForm.automation_plugin_id}:${goodsTypeForm.automation_instance_id}`
    : "";
  loadAutomationInstances();
  loadAutomationConfigTemplate();
  goodsTypeOpen.value = true;
};

const submitGoodsType = async () => {
  goodsTypeSaving.value = true;
  try {
    const payload = { ...goodsTypeForm } as Record<string, any>;
    const resizeEnabled = !!goodsTypeForm.resize_enabled;
    const refundEnabled = !!goodsTypeForm.refund_enabled;
    delete payload.resize_enabled;
    delete payload.refund_enabled;
    let currentGoodsTypeID = Number(payload.id || 0);
    if (payload.id) {
      await updateGoodsType(payload.id, payload);
    } else {
      const res = await createGoodsType(payload);
      currentGoodsTypeID = Number(res?.data?.id || res?.data?.ID || 0);
    }
    if (currentGoodsTypeID > 0) {
      await updateGoodsTypeCapabilities(currentGoodsTypeID, {
        resize_enabled: resizeEnabled,
        refund_enabled: refundEnabled
      });
    }
    if (automationConfigSchema.value && payload.automation_plugin_id && payload.automation_instance_id) {
      await updateAdminPluginInstanceConfig(
        "automation",
        String(payload.automation_plugin_id),
        String(payload.automation_instance_id),
        JSON.stringify(automationConfigModel.value || {})
      );
    }
    message.success("OK");
    goodsTypeOpen.value = false;
    await loadGoodsTypeList();
  } finally {
    goodsTypeSaving.value = false;
  }
};

const removeGoodsType = (record: any) => {
  Modal.confirm({
    title: "确认删除该商品类型?",
    onOk: async () => {
      await deleteGoodsType(record.id);
      message.success("已删除");
      await loadGoodsTypeList();
    }
  });
};

const syncGoodsType = async (record: any) => {
  await syncGoodsTypeAutomation(record.id, "merge");
  message.success("OK");
  await load();
};

watch(goodsTypeId, async () => {
  resetRegion();
  await load();
});

watch(imageLineId, async () => {
  await loadScopedImages();
});

watch(selectedAutomationRef, async (value) => {
  const raw = String(value || "").trim();
  if (!raw) {
    goodsTypeForm.automation_plugin_id = "";
    goodsTypeForm.automation_instance_id = "";
    automationConfigSchema.value = null;
    automationConfigUI.value = {};
    automationConfigModel.value = {};
    automationConfigError.value = "";
    return;
  }
  const idx = raw.indexOf(":");
  if (idx <= 0) return;
  goodsTypeForm.automation_plugin_id = raw.slice(0, idx);
  goodsTypeForm.automation_instance_id = raw.slice(idx + 1);
  await loadAutomationConfigTemplate();
});

const openRegion = (record) => {
  if (record) Object.assign(regionForm, record);
  else Object.assign(regionForm, { id: null, goods_type_id: goodsTypeId.value || null, name: "", code: "", active: true });
  regionOpen.value = true;
};

const submitRegion = async () => {
  if (!regionForm.goods_type_id) {
    regionForm.goods_type_id = goodsTypeId.value || null;
  }
  if (regionForm.id) {
    await updateRegion(regionForm.id, regionForm);
  } else {
    await createRegion(regionForm);
  }
  message.success("已保存地区");
  regionOpen.value = false;
  load();
};

const resetRegion = () => Object.assign(regionForm, { id: null, goods_type_id: goodsTypeId.value || null, name: "", code: "", active: true });

const removeRegion = (record) => {
  Modal.confirm({
    title: "确认删除该地区?",
    onOk: async () => {
      await deleteRegion(record.id);
      message.success("已删除");
      load();
    }
  });
};

const bulkRemoveRegions = () => {
  Modal.confirm({
    title: `确认删除所选 ${selectedRegionKeys.value.length} 个地区?`,
    onOk: async () => {
      await bulkDeleteRegions(selectedRegionKeys.value);
      selectedRegionKeys.value = [];
      message.success("已批量删除");
      load();
    }
  });
};

const openLine = async (record) => {
  if (record) {
    Object.assign(lineForm, record);
  } else {
    resetLine();
  }
  lineOpen.value = true;
  if (record?.id) {
    await loadLineImages(record.id ?? record.ID);
  } else {
    lineForm.image_ids = [];
  }
};

const resetLine = () =>
  Object.assign(lineForm, {
    id: null,
    region_id: null,
    name: "",
    line_id: "",
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
    image_ids: []
  });

const submitLine = async () => {
  const imageIDs = Array.isArray(lineForm.image_ids) ? lineForm.image_ids : [];
  const payload = normalizeLinePayload(lineForm);
  let res;
  if (lineForm.id) {
    res = await updateLine(lineForm.id, payload);
  } else {
    res = await createLine(payload);
  }
  const lineId = lineForm.id ?? res?.data?.id ?? res?.data?.ID;
  if (lineId) {
    await setLineSystemImages(lineId, { image_ids: imageIDs });
  }
  message.success("已保存线路");
  lineOpen.value = false;
  load();
};

// openidc 插件专用：切换线路启用/禁用状态
const toggleLineActive = async (record) => {
  const newActive = !record.active;
  await updateLine(record.id, { active: newActive });
  message.success(newActive ? "已启用" : "已禁用");
  load();
};

const removeLine = (record) => {
  Modal.confirm({
    title: "确认删除该线路?",
    onOk: async () => {
      await deleteLine(record.id);
      message.success("已删除");
      load();
    }
  });
};

const bulkRemoveLines = () => {
  Modal.confirm({
    title: `确认删除所选 ${selectedLineKeys.value.length} 条线路?`,
    onOk: async () => {
      await bulkDeleteLines(selectedLineKeys.value);
      selectedLineKeys.value = [];
      message.success("已批量删除");
      load();
    }
  });
};

const openPackage = async (record) => {
  if (record) Object.assign(packageForm, record);
  else {
    resetPackage();
    if (packageLineId.value && packageLineId.value !== "all") {
      packageForm.plan_group_id = packageLineId.value;
    }
  }
  packageOpen.value = true;
};

const resetPackage = () =>
  Object.assign(packageForm, {
    id: null,
    name: "",
    plan_group_id: null,
    cores: 1,
    memory_gb: 1,
    disk_gb: 20,
    bandwidth_mbps: 1,
    cpu_model: "",
    port_num: 30,
    monthly_price: 0,
    active: true,
    visible: true,
    capacity_remaining: -1
  });

const submitPackage = async () => {
  if (packageForm.id) {
    await updatePackage(packageForm.id, packageForm);
  } else {
    await createPackage(packageForm);
  }
  message.success("已保存套餐");
  packageOpen.value = false;
  load();
};

const removePackage = (record) => {
  Modal.confirm({
    title: "确认删除该套餐?",
    onOk: async () => {
      await deletePackage(record.id);
      message.success("已删除");
      load();
    }
  });
};

const bulkRemovePackages = () => {
  Modal.confirm({
    title: `确认删除所选 ${selectedPackageKeys.value.length} 个套餐?`,
    onOk: async () => {
      await bulkDeletePackages(selectedPackageKeys.value);
      selectedPackageKeys.value = [];
      message.success("已批量删除");
      load();
    }
  });
};

const openImage = (record) => {
  if (record) {
    Object.assign(imageForm, {
      id: record.id ?? null,
      image_id: Number(record.image_id ?? 0) || null,
      name: record.name ?? "",
      type: String(record.type || "").toLowerCase() || "linux",
      enabled: !!record.enabled
    });
  }
  else resetImage();
  imageOpen.value = true;
};

const resetImage = () => Object.assign(imageForm, { id: null, image_id: null, name: "", type: "linux", enabled: true });

const submitImage = async () => {
  const imageID = Number(imageForm.image_id || 0);
  if (!Number.isInteger(imageID) || imageID <= 0) {
    message.error("镜像 ID 必须是正整数");
    return;
  }
  const imageType = String(imageForm.type || "").trim().toLowerCase();
  if (!["linux", "windows"].includes(imageType)) {
    message.error("请选择镜像类型");
    return;
  }
  const payload = {
    id: imageForm.id,
    image_id: imageID,
    name: String(imageForm.name || "").trim(),
    type: imageType,
    enabled: !!imageForm.enabled
  };
  if (imageForm.id) {
    await updateSystemImage(imageForm.id, payload);
  } else {
    await createSystemImage(payload);
  }
  message.success("已保存镜像");
  imageOpen.value = false;
  load();
};

const removeImage = (record) => {
  Modal.confirm({
    title: "确认删除该镜像?",
    onOk: async () => {
      await deleteSystemImage(record.id);
      message.success("已删除");
      load();
    }
  });
};

const bulkRemoveImages = () => {
  Modal.confirm({
    title: `确认删除所选 ${selectedImageKeys.value.length} 个镜像?`,
    onOk: async () => {
      await bulkDeleteSystemImages(selectedImageKeys.value);
      selectedImageKeys.value = [];
      message.success("已批量删除");
      load();
    }
  });
};

const syncImages = async () => {
  if (!imageLineId.value) {
    message.error("请先选择线路再同步镜像");
    return;
  }
  const cloudLineId = resolveCloudLineId(imageLineId.value);
  if (!cloudLineId) {
    message.error("无法解析线路 ID");
    return;
  }
  await syncSystemImages({ line_id: cloudLineId, plan_group_id: imageLineId.value });
  message.success("已触发同步");
  await load();
};

const batchForm = reactive({
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
  cpu_model: "",
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
});

const batchColumns = [
  { title: "名称", dataIndex: "name", key: "name" },
  { title: "CPU", dataIndex: "cores", key: "cores" },
  { title: "内存", dataIndex: "memory_gb", key: "memory_gb" },
  { title: "存储", dataIndex: "disk_gb", key: "disk_gb" },
  { title: "带宽", dataIndex: "bandwidth_mbps", key: "bandwidth_mbps" },
  { title: "月费", dataIndex: "monthly_price", key: "monthly_price" },
  { title: "余量", dataIndex: "capacity_remaining", key: "capacity_remaining" }
];

const openBatch = () => {
  if (packageLineId.value && packageLineId.value !== "all") {
    batchForm.plan_group_id = packageLineId.value;
  }
  batchOpen.value = true;
};

const closeBatch = () => {
  batchOpen.value = false;
};

const clearGenerated = () => {
  generatedPackages.value = [];
};

const calcCapacity = (cores, memory, disk, bandwidth) => {
  const multiplier = batchForm.overcommit_enabled ? Number(batchForm.overcommit_ratio || 1) : 1;
  const totals = {
    cores: Number(batchForm.total_cores || 0) * multiplier,
    mem: Number(batchForm.total_mem || 0) * multiplier,
    disk: Number(batchForm.total_disk || 0) * multiplier,
    bw: Number(batchForm.total_bw || 0) * multiplier
  };
  const candidates = [];
  if (totals.cores > 0 && cores > 0) candidates.push(Math.floor(totals.cores / cores));
  if (totals.mem > 0 && memory > 0) candidates.push(Math.floor(totals.mem / memory));
  if (totals.disk > 0 && disk > 0) candidates.push(Math.floor(totals.disk / disk));
  if (totals.bw > 0 && bandwidth > 0) candidates.push(Math.floor(totals.bw / bandwidth));
  if (!candidates.length) return -1;
  return Math.max(0, Math.min(...candidates));
};

const generatePackages = () => {
  if (!batchForm.plan_group_id) {
    message.error("请选择线路");
    return;
  }
  const line = lines.value.find((item) => Number(item.id) === Number(batchForm.plan_group_id));
  if (!line) {
    message.error("线路信息未加载");
    return;
  }
  const cpuMin = Number(batchForm.cpu_min || 0);
  const cpuMax = Number(batchForm.cpu_max || 0);
  const cpuStep = Math.max(1, Number(batchForm.cpu_step || 1));
  const diskMin = Number(batchForm.disk_min || 0);
  const diskMax = Number(batchForm.disk_max || 0);
  const diskStep = Math.max(1, Number(batchForm.disk_step || 1));
  const bwMin = Number(batchForm.bw_min || 0);
  const bwMax = Number(batchForm.bw_max || 0);
  const bwStep = Math.max(1, Number(batchForm.bw_step || 1));
  const memoryRatio = Number(batchForm.memory_ratio || 0);
  const memoryMin = Number(batchForm.memory_min || 0);
  const memoryMax = Number(batchForm.memory_max || 0);
  if (!cpuMin || !cpuMax || cpuMax < cpuMin) {
    message.error("CPU 范围不正确");
    return;
  }
  if (!diskMin || !diskMax || diskMax < diskMin) {
    message.error("存储范围不正确");
    return;
  }
  if (!bwMin || !bwMax || bwMax < bwMin) {
    message.error("带宽范围不正确");
    return;
  }
  if (!memoryRatio) {
    message.error("内存比率需大于 0");
    return;
  }
  const items = [];
  let priceMultiplier = Number(batchForm.price_multiplier || 1);
  const totalCost = Number(batchForm.total_cost || 0);
  if (totalCost > 0) {
    const baseCost =
      Number(line.unit_core || 0) * Number(batchForm.total_cores || 0) +
      Number(line.unit_mem || 0) * Number(batchForm.total_mem || 0) +
      Number(line.unit_disk || 0) * Number(batchForm.total_disk || 0) +
      Number(line.unit_bw || 0) * Number(batchForm.total_bw || 0);
    if (baseCost > 0) {
      priceMultiplier = totalCost / baseCost;
    }
  }
  for (let cpu = cpuMin; cpu <= cpuMax; cpu += cpuStep) {
    let memory = Math.round(cpu * memoryRatio);
    if (memoryMin && memory < memoryMin) memory = memoryMin;
    if (memoryMax && memory > memoryMax) continue;
    for (let disk = diskMin; disk <= diskMax; disk += diskStep) {
      for (let bw = bwMin; bw <= bwMax; bw += bwStep) {
        const monthlyBase =
          Number(line.unit_core || 0) * cpu +
          Number(line.unit_mem || 0) * memory +
          Number(line.unit_disk || 0) * disk +
          Number(line.unit_bw || 0) * bw;
        const monthlyPrice = Number((monthlyBase * priceMultiplier).toFixed(2));
        const capacityRemaining = calcCapacity(cpu, memory, disk, bw);
        items.push({
          key: `${cpu}-${memory}-${disk}-${bw}`,
          name: `${cpu}C${memory}G ${disk}G ${bw}M`,
          plan_group_id: batchForm.plan_group_id,
          cores: cpu,
          memory_gb: memory,
          disk_gb: disk,
          bandwidth_mbps: bw,
          cpu_model: batchForm.cpu_model,
          port_num: batchForm.port_num,
          monthly_price: monthlyPrice,
          active: batchForm.active,
          visible: batchForm.visible,
          capacity_remaining: capacityRemaining
        });
      }
    }
  }
  if (!items.length) {
    message.warning("未生成任何套餐，请检查条件");
    generatedPackages.value = [];
    return;
  }
  const maxRows = 200;
  if (items.length > maxRows) {
    message.warning(`生成数量过多，已截断至 ${maxRows} 条`);
  }
  generatedPackages.value = items.slice(0, maxRows);
};

const applyGenerated = async () => {
  if (!generatedPackages.value.length) return;
  Modal.confirm({
    title: `确认批量创建 ${generatedPackages.value.length} 个套餐?`,
    onOk: async () => {
      for (const item of generatedPackages.value) {
        const payload = { ...item };
        delete payload.key;
        await createPackage(payload);
      }
      message.success("已批量创建套餐");
      generatedPackages.value = [];
      batchOpen.value = false;
      load();
    }
  });
};

const openCycle = (record) => {
  if (record) Object.assign(cycleForm, record);
  else resetCycle();
  cycleOpen.value = true;
};

const resetCycle = () => Object.assign(cycleForm, { id: null, name: "", months: 1, multiplier: 1, min_qty: 1, max_qty: 12, active: true });

const submitCycle = async () => {
  if (cycleForm.id) {
    await updateBillingCycle(cycleForm.id, cycleForm);
  } else {
    await createBillingCycle(cycleForm);
  }
  message.success("已保存周期");
  cycleOpen.value = false;
  load();
};

const removeCycle = (record) => {
  Modal.confirm({
    title: "确认删除该周期?",
    onOk: async () => {
      await deleteBillingCycle(record.id);
      message.success("已删除");
      load();
    }
  });
};

const bulkRemoveCycles = () => {
  Modal.confirm({
    title: `确认删除所选 ${selectedCycleKeys.value.length} 个周期?`,
    onOk: async () => {
      await bulkDeleteBillingCycles(selectedCycleKeys.value);
      selectedCycleKeys.value = [];
      message.success("已批量删除");
      load();
    }
  });
};

onMounted(async () => {
  await loadGoodsTypeList();
  await load();
});
</script>
