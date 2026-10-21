<template>
  <ElDialog
    v-model="dialogVisible"
    :title="localForm.id ? '编辑地区' : '新增地区'"
    width="460px"
    destroy-on-close
    align-center
  >
    <ElForm label-position="top">
      <ElFormItem label="名称">
        <ElInput v-model.trim="localForm.name" maxlength="80" placeholder="请输入名称" />
      </ElFormItem>
      <ElFormItem label="代码">
        <ElInput v-model.trim="localForm.code" maxlength="80" placeholder="请输入代码" />
      </ElFormItem>
      <ElFormItem label="启用">
        <ElSwitch v-model="localForm.active" />
      </ElFormItem>
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
  defineOptions({ name: 'CatalogRegionDialog' })

  interface RegionDialogFormValue {
    id: number | null
    goods_type_id: number | null
    name: string
    code: string
    active: boolean
  }

  interface Props {
    visible: boolean
    formData: RegionDialogFormValue
    submitting?: boolean
  }

  interface Emits {
    (e: 'update:visible', value: boolean): void
    (e: 'submit', value: RegionDialogFormValue): void
  }

  const props = withDefaults(defineProps<Props>(), {
    submitting: false
  })
  const emit = defineEmits<Emits>()

  const localForm = reactive<RegionDialogFormValue>(createDefaultForm())

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

  function createDefaultForm(): RegionDialogFormValue {
    return {
      id: null,
      goods_type_id: null,
      name: '',
      code: '',
      active: true
    }
  }
</script>

<style scoped lang="scss">
  .dialog-footer {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
  }
</style>
