<template>
  <ElDialog
    v-model="dialogVisible"
    title="批量生成套餐"
    width="980px"
    destroy-on-close
    align-center
  >
    <ElForm label-position="top">
      <ElDivider content-position="left">基础配置</ElDivider>

      <ElRow :gutter="12">
        <ElCol :span="8">
          <ElFormItem label="线路">
            <ElSelect
              v-model="formModel.plan_group_id"
              filterable
              placeholder="选择线路"
              class="full-width"
            >
              <ElOption
                v-for="item in lineOptions"
                :key="item.value"
                :label="item.label"
                :value="item.value"
              />
            </ElSelect>
          </ElFormItem>
        </ElCol>
        <ElCol :span="8">
          <ElFormItem label="定价倍率">
            <ElInputNumber
              v-model="formModel.price_multiplier"
              :min="0.1"
              :step="0.1"
              class="full-width"
            />
          </ElFormItem>
        </ElCol>
        <ElCol :span="8">
          <ElFormItem label="端口数">
            <ElInputNumber v-model="formModel.port_num" :min="0" class="full-width" />
          </ElFormItem>
        </ElCol>
      </ElRow>

      <ElRow :gutter="12">
        <ElCol :span="6">
          <ElFormItem label="线路总成本">
            <ElInputNumber v-model="formModel.total_cost" :min="0" class="full-width" />
          </ElFormItem>
        </ElCol>
        <ElCol :span="6">
          <ElFormItem label="CPU 型号">
            <ElInput v-model.trim="formModel.cpu_model" />
          </ElFormItem>
        </ElCol>
        <ElCol :span="6">
          <ElFormItem label="启用">
            <ElSwitch v-model="formModel.active" />
          </ElFormItem>
        </ElCol>
        <ElCol :span="6">
          <ElFormItem label="可见">
            <ElSwitch v-model="formModel.visible" />
          </ElFormItem>
        </ElCol>
      </ElRow>

      <ElDivider content-position="left">资源规则</ElDivider>

      <ElRow :gutter="12">
        <ElCol :span="8"
          ><ElFormItem label="CPU 最小"
            ><ElInputNumber v-model="formModel.cpu_min" :min="1" class="full-width" /></ElFormItem
        ></ElCol>
        <ElCol :span="8"
          ><ElFormItem label="CPU 最大"
            ><ElInputNumber v-model="formModel.cpu_max" :min="1" class="full-width" /></ElFormItem
        ></ElCol>
        <ElCol :span="8"
          ><ElFormItem label="CPU 步进"
            ><ElInputNumber v-model="formModel.cpu_step" :min="1" class="full-width" /></ElFormItem
        ></ElCol>
      </ElRow>

      <ElRow :gutter="12">
        <ElCol :span="8"
          ><ElFormItem label="内存比率 (GB/核)"
            ><ElInputNumber
              v-model="formModel.memory_ratio"
              :min="0.5"
              :step="0.5"
              class="full-width" /></ElFormItem
        ></ElCol>
        <ElCol :span="8"
          ><ElFormItem label="内存最小"
            ><ElInputNumber
              v-model="formModel.memory_min"
              :min="1"
              class="full-width" /></ElFormItem
        ></ElCol>
        <ElCol :span="8"
          ><ElFormItem label="内存最大"
            ><ElInputNumber
              v-model="formModel.memory_max"
              :min="1"
              class="full-width" /></ElFormItem
        ></ElCol>
      </ElRow>

      <ElRow :gutter="12">
        <ElCol :span="8"
          ><ElFormItem label="存储最小 (GB)"
            ><ElInputNumber v-model="formModel.disk_min" :min="1" class="full-width" /></ElFormItem
        ></ElCol>
        <ElCol :span="8"
          ><ElFormItem label="存储最大 (GB)"
            ><ElInputNumber v-model="formModel.disk_max" :min="1" class="full-width" /></ElFormItem
        ></ElCol>
        <ElCol :span="8"
          ><ElFormItem label="存储步进 (GB)"
            ><ElInputNumber v-model="formModel.disk_step" :min="1" class="full-width" /></ElFormItem
        ></ElCol>
      </ElRow>

      <ElRow :gutter="12">
        <ElCol :span="8"
          ><ElFormItem label="带宽最小 (Mbps)"
            ><ElInputNumber v-model="formModel.bw_min" :min="1" class="full-width" /></ElFormItem
        ></ElCol>
        <ElCol :span="8"
          ><ElFormItem label="带宽最大 (Mbps)"
            ><ElInputNumber v-model="formModel.bw_max" :min="1" class="full-width" /></ElFormItem
        ></ElCol>
        <ElCol :span="8"
          ><ElFormItem label="带宽步进 (Mbps)"
            ><ElInputNumber v-model="formModel.bw_step" :min="1" class="full-width" /></ElFormItem
        ></ElCol>
      </ElRow>

      <ElDivider content-position="left">容量计算</ElDivider>

      <ElRow :gutter="12">
        <ElCol :span="6"
          ><ElFormItem label="CPU 总量"
            ><ElInputNumber
              v-model="formModel.total_cores"
              :min="0"
              class="full-width" /></ElFormItem
        ></ElCol>
        <ElCol :span="6"
          ><ElFormItem label="内存总量 (GB)"
            ><ElInputNumber v-model="formModel.total_mem" :min="0" class="full-width" /></ElFormItem
        ></ElCol>
        <ElCol :span="6"
          ><ElFormItem label="存储总量 (GB)"
            ><ElInputNumber
              v-model="formModel.total_disk"
              :min="0"
              class="full-width" /></ElFormItem
        ></ElCol>
        <ElCol :span="6"
          ><ElFormItem label="带宽总量 (Mbps)"
            ><ElInputNumber v-model="formModel.total_bw" :min="0" class="full-width" /></ElFormItem
        ></ElCol>
      </ElRow>

      <ElRow :gutter="12">
        <ElCol :span="8">
          <ElFormItem label="是否超开">
            <ElSwitch v-model="formModel.overcommit_enabled" />
          </ElFormItem>
        </ElCol>
        <ElCol :span="8">
          <ElFormItem label="超开倍率">
            <ElInputNumber
              v-model="formModel.overcommit_ratio"
              :min="1"
              :step="0.1"
              :disabled="!formModel.overcommit_enabled"
              class="full-width"
            />
          </ElFormItem>
        </ElCol>
        <ElCol :span="8">
          <div class="capacity-tip">用于计算余量，留空表示不限制。</div>
        </ElCol>
      </ElRow>

      <div class="preview-actions">
        <ElButton type="primary" @click="emit('generate')">生成套餐</ElButton>
        <ElButton @click="emit('clear')">清空预览</ElButton>
      </div>

      <ElTable :data="generatedRows" border row-key="key" size="small" max-height="320">
        <ElTableColumn prop="name" label="名称" min-width="160" />
        <ElTableColumn prop="cores" label="CPU" width="80" />
        <ElTableColumn prop="memory_gb" label="内存" width="90" />
        <ElTableColumn prop="disk_gb" label="存储" width="90" />
        <ElTableColumn prop="bandwidth_mbps" label="带宽" width="90" />
        <ElTableColumn prop="monthly_price" label="月费" width="100" />
        <ElTableColumn label="余量" width="100">
          <template #default="{ row }">
            {{ formatCapacity(row.capacity_remaining) }}
          </template>
        </ElTableColumn>
      </ElTable>
    </ElForm>

    <template #footer>
      <div class="dialog-footer">
        <ElButton @click="dialogVisible = false">关闭</ElButton>
        <ElButton type="primary" :disabled="!generatedRows.length" @click="emit('apply')">
          应用生成
        </ElButton>
      </div>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  defineOptions({ name: 'CatalogPackageBatchDialog' })

  interface BatchFormValue {
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

  interface GeneratedPackageRow {
    key: string
    name: string
    cores: number
    memory_gb: number
    disk_gb: number
    bandwidth_mbps: number
    monthly_price: number
    capacity_remaining: number
  }

  interface Props {
    visible: boolean
    modelValue: BatchFormValue
    lineOptions?: Array<{ label: string; value: number }>
    generatedRows?: GeneratedPackageRow[]
  }

  interface Emits {
    (e: 'update:visible', value: boolean): void
    (e: 'update:modelValue', value: BatchFormValue): void
    (e: 'generate'): void
    (e: 'clear'): void
    (e: 'apply'): void
  }

  const props = withDefaults(defineProps<Props>(), {
    lineOptions: () => [],
    generatedRows: () => []
  })
  const emit = defineEmits<Emits>()

  const dialogVisible = computed({
    get: () => props.visible,
    set: (value) => emit('update:visible', value)
  })

  const formModel = computed({
    get: () => props.modelValue,
    set: (value) => emit('update:modelValue', value)
  })

  function formatCapacity(value?: number | null) {
    const amount = Number(value)
    if (!Number.isFinite(amount)) return '-'
    if (amount < 0) return '不限'
    if (amount === 0) return '售罄'
    return String(amount)
  }
</script>

<style scoped lang="scss">
  .full-width {
    width: 100%;
  }

  .capacity-tip {
    padding-top: 28px;
    color: var(--el-text-color-secondary);
    font-size: 13px;
    line-height: 1.6;
  }

  .preview-actions,
  .dialog-footer {
    display: flex;
    gap: 12px;
  }

  .preview-actions {
    margin-bottom: 16px;
  }

  .dialog-footer {
    justify-content: flex-end;
  }
</style>
