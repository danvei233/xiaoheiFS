<template>
  <ElDialog
    v-model="dialogVisible"
    :title="localForm.id ? '编辑套餐' : '新增套餐'"
    width="560px"
    destroy-on-close
    align-center
  >
    <ElForm label-position="top">
      <ElFormItem label="名称">
        <ElInput v-model.trim="localForm.name" maxlength="120" placeholder="请输入名称" />
      </ElFormItem>
      <ElFormItem label="线路">
        <ElSelect
          v-model="localForm.plan_group_id"
          filterable
          placeholder="请选择线路"
          class="full-width"
        >
          <ElOption
            v-for="item in planGroupOptions"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </ElSelect>
      </ElFormItem>
      <ElRow :gutter="12">
        <ElCol :span="12"
          ><ElFormItem label="CPU"
            ><ElInputNumber v-model="localForm.cores" :min="1" class="full-width" /></ElFormItem
        ></ElCol>
        <ElCol :span="12"
          ><ElFormItem label="内存 (GB)"
            ><ElInputNumber v-model="localForm.memory_gb" :min="1" class="full-width" /></ElFormItem
        ></ElCol>
      </ElRow>
      <ElRow :gutter="12">
        <ElCol :span="12"
          ><ElFormItem label="磁盘 (GB)"
            ><ElInputNumber v-model="localForm.disk_gb" :min="1" class="full-width" /></ElFormItem
        ></ElCol>
        <ElCol :span="12"
          ><ElFormItem label="带宽 (Mbps)"
            ><ElInputNumber
              v-model="localForm.bandwidth_mbps"
              :min="1"
              class="full-width" /></ElFormItem
        ></ElCol>
      </ElRow>
      <ElRow :gutter="12">
        <ElCol :span="12"
          ><ElFormItem label="CPU 型号"
            ><ElInput
              v-model.trim="localForm.cpu_model"
              maxlength="120"
              placeholder="可选" /></ElFormItem
        ></ElCol>
        <ElCol :span="12"
          ><ElFormItem label="端口数"
            ><ElInputNumber v-model="localForm.port_num" :min="0" class="full-width" /></ElFormItem
        ></ElCol>
      </ElRow>
      <ElRow :gutter="12">
        <ElCol :span="12"
          ><ElFormItem label="月费"
            ><ElInputNumber
              v-model="localForm.monthly_price"
              :min="0"
              :step="0.01"
              class="full-width" /></ElFormItem
        ></ElCol>
        <ElCol :span="12"
          ><ElFormItem label="余量"
            ><ElInputNumber
              v-model="localForm.capacity_remaining"
              :min="-1"
              class="full-width" /></ElFormItem
        ></ElCol>
      </ElRow>
      <ElRow :gutter="12">
        <ElCol :span="12"
          ><ElFormItem label="排序"
            ><ElInputNumber
              v-model="localForm.sort_order"
              :min="0"
              class="full-width" /></ElFormItem
        ></ElCol>
        <ElCol :span="12"
          ><ElFormItem label="状态"
            ><div class="switch-group"
              ><ElSwitch v-model="localForm.active" /><span>启用</span
              ><ElSwitch v-model="localForm.visible" /><span>可见</span></div
            ></ElFormItem
          ></ElCol
        >
      </ElRow>
    </ElForm>

    <template #footer>
      <div class="dialog-footer">
        <ElButton @click="dialogVisible = false">取消</ElButton>
        <ElButton type="primary" :loading="submitting" @click="emit('submit', { ...localForm })">
          保存
        </ElButton>
      </div>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  defineOptions({ name: 'CatalogPackageDialog' })

  interface PackageDialogFormValue {
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

  interface Props {
    visible: boolean
    formData: PackageDialogFormValue
    planGroupOptions?: Array<{ label: string; value: number }>
    submitting?: boolean
  }

  interface Emits {
    (e: 'update:visible', value: boolean): void
    (e: 'submit', value: PackageDialogFormValue): void
  }

  const props = withDefaults(defineProps<Props>(), {
    planGroupOptions: () => [],
    submitting: false
  })
  const emit = defineEmits<Emits>()

  const localForm = reactive<PackageDialogFormValue>(createDefaultForm())

  const dialogVisible = computed({
    get: () => props.visible,
    set: (value) => emit('update:visible', value)
  })

  watch(
    () => [props.visible, props.formData] as const,
    ([visible]) => {
      if (!visible) return
      Object.assign(localForm, createDefaultForm(), props.formData)
    },
    { immediate: true, deep: true }
  )

  function createDefaultForm(): PackageDialogFormValue {
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
</script>

<style scoped lang="scss">
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
