<template>
  <ElDrawer
    v-model="drawerVisible"
    :title="localForm.id ? '编辑线路' : '新增线路'"
    size="760px"
    destroy-on-close
  >
    <ElForm label-position="top" class="drawer-form">
      <ElFormItem label="地区">
        <ElSelect
          v-model="localForm.region_id"
          filterable
          placeholder="请选择地区"
          class="full-width"
        >
          <ElOption
            v-for="item in regionOptions"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </ElSelect>
      </ElFormItem>
      <ElRow :gutter="12">
        <ElCol :span="12">
          <ElFormItem label="线路名称">
            <ElInput v-model.trim="localForm.name" maxlength="120" placeholder="请输入名称" />
          </ElFormItem>
        </ElCol>
        <ElCol :span="12">
          <ElFormItem label="云线路 ID">
            <ElInputNumber v-model="localForm.line_id" :min="0" class="full-width" />
          </ElFormItem>
        </ElCol>
      </ElRow>
      <ElRow :gutter="12">
        <ElCol :span="12">
          <ElFormItem label="CPU 单价">
            <ElInputNumber v-model="localForm.unit_core" :min="0" :step="0.01" class="full-width" />
          </ElFormItem>
        </ElCol>
        <ElCol :span="12">
          <ElFormItem label="内存单价">
            <ElInputNumber v-model="localForm.unit_mem" :min="0" :step="0.01" class="full-width" />
          </ElFormItem>
        </ElCol>
      </ElRow>
      <ElRow :gutter="12">
        <ElCol :span="12">
          <ElFormItem label="磁盘单价">
            <ElInputNumber v-model="localForm.unit_disk" :min="0" :step="0.01" class="full-width" />
          </ElFormItem>
        </ElCol>
        <ElCol :span="12">
          <ElFormItem label="带宽单价">
            <ElInputNumber v-model="localForm.unit_bw" :min="0" :step="0.01" class="full-width" />
          </ElFormItem>
        </ElCol>
      </ElRow>

      <ElDivider content-position="left">附加项范围</ElDivider>
      <div class="drawer-hint">规则：`-1` 表示禁用该附加项，`0` 表示不限制上下限。</div>

      <ElRow :gutter="12">
        <ElCol :span="8"
          ><ElFormItem label="CPU 最小"
            ><ElInputNumber
              v-model="localForm.add_core_min"
              :min="-1"
              class="full-width" /></ElFormItem
        ></ElCol>
        <ElCol :span="8"
          ><ElFormItem label="CPU 最大"
            ><ElInputNumber
              v-model="localForm.add_core_max"
              :min="0"
              class="full-width" /></ElFormItem
        ></ElCol>
        <ElCol :span="8"
          ><ElFormItem label="CPU 步进"
            ><ElInputNumber
              v-model="localForm.add_core_step"
              :min="1"
              class="full-width" /></ElFormItem
        ></ElCol>
      </ElRow>
      <ElRow :gutter="12">
        <ElCol :span="8"
          ><ElFormItem label="内存最小"
            ><ElInputNumber
              v-model="localForm.add_mem_min"
              :min="-1"
              class="full-width" /></ElFormItem
        ></ElCol>
        <ElCol :span="8"
          ><ElFormItem label="内存最大"
            ><ElInputNumber
              v-model="localForm.add_mem_max"
              :min="0"
              class="full-width" /></ElFormItem
        ></ElCol>
        <ElCol :span="8"
          ><ElFormItem label="内存步进"
            ><ElInputNumber
              v-model="localForm.add_mem_step"
              :min="1"
              class="full-width" /></ElFormItem
        ></ElCol>
      </ElRow>
      <ElRow :gutter="12">
        <ElCol :span="8"
          ><ElFormItem label="磁盘最小"
            ><ElInputNumber
              v-model="localForm.add_disk_min"
              :min="-1"
              class="full-width" /></ElFormItem
        ></ElCol>
        <ElCol :span="8"
          ><ElFormItem label="磁盘最大"
            ><ElInputNumber
              v-model="localForm.add_disk_max"
              :min="0"
              class="full-width" /></ElFormItem
        ></ElCol>
        <ElCol :span="8"
          ><ElFormItem label="磁盘步进"
            ><ElInputNumber
              v-model="localForm.add_disk_step"
              :min="1"
              class="full-width" /></ElFormItem
        ></ElCol>
      </ElRow>
      <ElRow :gutter="12">
        <ElCol :span="8"
          ><ElFormItem label="带宽最小"
            ><ElInputNumber
              v-model="localForm.add_bw_min"
              :min="-1"
              class="full-width" /></ElFormItem
        ></ElCol>
        <ElCol :span="8"
          ><ElFormItem label="带宽最大"
            ><ElInputNumber
              v-model="localForm.add_bw_max"
              :min="0"
              class="full-width" /></ElFormItem
        ></ElCol>
        <ElCol :span="8"
          ><ElFormItem label="带宽步进"
            ><ElInputNumber
              v-model="localForm.add_bw_step"
              :min="1"
              class="full-width" /></ElFormItem
        ></ElCol>
      </ElRow>

      <ElRow :gutter="12">
        <ElCol :span="8">
          <ElFormItem label="排序">
            <ElInputNumber v-model="localForm.sort_order" :min="0" class="full-width" />
          </ElFormItem>
        </ElCol>
        <ElCol :span="8">
          <ElFormItem label="余量">
            <ElInputNumber v-model="localForm.capacity_remaining" :min="-1" class="full-width" />
          </ElFormItem>
        </ElCol>
        <ElCol :span="8">
          <ElFormItem label="状态">
            <div class="switch-group">
              <ElSwitch v-model="localForm.active" />
              <span>启用</span>
              <ElSwitch v-model="localForm.visible" />
              <span>可见</span>
            </div>
          </ElFormItem>
        </ElCol>
      </ElRow>

      <ElFormItem label="可用镜像">
        <ElSelect
          v-model="localForm.image_ids"
          multiple
          filterable
          collapse-tags
          collapse-tags-tooltip
          placeholder="选择该线路可用镜像"
          class="full-width"
        >
          <ElOption
            v-for="item in systemImageOptions"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </ElSelect>
      </ElFormItem>
    </ElForm>

    <template #footer>
      <div class="dialog-footer">
        <ElButton @click="drawerVisible = false">取消</ElButton>
        <ElButton
          type="primary"
          :loading="submitting"
          @click="emit('submit', { ...localForm, image_ids: [...localForm.image_ids] })"
        >
          保存
        </ElButton>
      </div>
    </template>
  </ElDrawer>
</template>

<script setup lang="ts">
  defineOptions({ name: 'CatalogPlanGroupDrawer' })

  interface PlanGroupDialogFormValue {
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

  interface Props {
    visible: boolean
    formData: PlanGroupDialogFormValue
    regionOptions?: Array<{ label: string; value: number }>
    systemImageOptions?: Array<{ label: string; value: number }>
    submitting?: boolean
  }

  interface Emits {
    (e: 'update:visible', value: boolean): void
    (e: 'submit', value: PlanGroupDialogFormValue): void
  }

  const props = withDefaults(defineProps<Props>(), {
    regionOptions: () => [],
    systemImageOptions: () => [],
    submitting: false
  })
  const emit = defineEmits<Emits>()

  const localForm = reactive<PlanGroupDialogFormValue>(createDefaultForm())

  const drawerVisible = computed({
    get: () => props.visible,
    set: (value) => emit('update:visible', value)
  })

  watch(
    () => [props.visible, props.formData] as const,
    ([visible]) => {
      if (!visible) return
      Object.assign(localForm, createDefaultForm(), props.formData, {
        image_ids: Array.isArray(props.formData.image_ids) ? [...props.formData.image_ids] : []
      })
    },
    { immediate: true, deep: true }
  )

  function createDefaultForm(): PlanGroupDialogFormValue {
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
</script>

<style scoped lang="scss">
  .drawer-form {
    padding-right: 8px;
  }

  .drawer-hint {
    margin-bottom: 12px;
    color: var(--el-text-color-secondary);
    font-size: 13px;
    line-height: 1.6;
  }

  .dialog-footer {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
  }

  .switch-group {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;
    min-height: 32px;
  }

  .full-width {
    width: 100%;
  }
</style>
