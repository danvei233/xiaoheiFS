<template>
  <ElDialog
    v-model="dialogVisible"
    :title="localForm.id ? '编辑周期' : '新增周期'"
    width="460px"
    destroy-on-close
    align-center
  >
    <ElForm label-position="top">
      <ElFormItem label="名称">
        <ElInput v-model.trim="localForm.name" maxlength="80" placeholder="请输入名称" />
      </ElFormItem>
      <ElRow :gutter="12">
        <ElCol :span="12"
          ><ElFormItem label="月数"
            ><ElInputNumber v-model="localForm.months" :min="1" class="full-width" /></ElFormItem
        ></ElCol>
        <ElCol :span="12"
          ><ElFormItem label="倍率"
            ><ElInputNumber
              v-model="localForm.multiplier"
              :min="0"
              :step="0.01"
              class="full-width" /></ElFormItem
        ></ElCol>
      </ElRow>
      <ElRow :gutter="12">
        <ElCol :span="12"
          ><ElFormItem label="最小数量"
            ><ElInputNumber v-model="localForm.min_qty" :min="1" class="full-width" /></ElFormItem
        ></ElCol>
        <ElCol :span="12"
          ><ElFormItem label="最大数量"
            ><ElInputNumber v-model="localForm.max_qty" :min="1" class="full-width" /></ElFormItem
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
          ><ElFormItem label="启用"><ElSwitch v-model="localForm.active" /></ElFormItem
        ></ElCol>
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
  defineOptions({ name: 'CatalogBillingCycleDialog' })

  interface BillingCycleDialogFormValue {
    id: number | null
    name: string
    months: number
    multiplier: number
    min_qty: number
    max_qty: number
    active: boolean
    sort_order: number
  }

  interface Props {
    visible: boolean
    formData: BillingCycleDialogFormValue
    submitting?: boolean
  }

  interface Emits {
    (e: 'update:visible', value: boolean): void
    (e: 'submit', value: BillingCycleDialogFormValue): void
  }

  const props = withDefaults(defineProps<Props>(), {
    submitting: false
  })
  const emit = defineEmits<Emits>()

  const localForm = reactive<BillingCycleDialogFormValue>(createDefaultForm())

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

  function createDefaultForm(): BillingCycleDialogFormValue {
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
</script>

<style scoped lang="scss">
  .dialog-footer {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
  }

  .full-width {
    width: 100%;
  }
</style>
