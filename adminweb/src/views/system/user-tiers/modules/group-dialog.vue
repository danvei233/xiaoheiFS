<template>
  <ElDialog
    v-model="dialogVisible"
    :title="dialogTitle"
    width="520px"
    destroy-on-close
    align-center
  >
    <ElForm ref="formRef" :model="localForm" :rules="rules" label-position="top">
      <ElFormItem label="名称" prop="name">
        <ElInput v-model="localForm.name" maxlength="120" placeholder="请输入用户组名称" />
      </ElFormItem>

      <ElFormItem label="颜色" prop="color">
        <div class="color-row">
          <ElColorPicker v-model="localForm.color" />
          <ElTag>{{ localForm.color || '#1677ff' }}</ElTag>
        </div>
      </ElFormItem>

      <ElFormItem label="图标" prop="icon">
        <ElSelect v-model="localForm.icon" placeholder="请选择图标" class="full-width">
          <ElOption
            v-for="option in iconOptions"
            :key="option.value"
            :label="option.label"
            :value="option.value"
          />
        </ElSelect>
      </ElFormItem>

      <ElFormItem label="优先级" prop="priority">
        <ElInputNumber v-model="localForm.priority" :min="1" :max="999" class="full-width" />
      </ElFormItem>

      <ElFormItem label="自动审批开关" prop="auto_approve_enabled">
        <ElSwitch v-model="localForm.auto_approve_enabled" />
      </ElFormItem>
    </ElForm>

    <template #footer>
      <div class="dialog-footer">
        <ElButton @click="dialogVisible = false">取消</ElButton>
        <ElButton type="primary" :loading="submitting" @click="handleSubmit">提交</ElButton>
      </div>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  import type { FormInstance, FormRules } from 'element-plus'

  defineOptions({ name: 'UserTierGroupDialog' })

  export interface UserTierGroupFormValue {
    id: number | null
    name: string
    color: string
    icon: string
    priority: number
    auto_approve_enabled: boolean
    is_default: boolean
  }

  interface Props {
    visible: boolean
    mode: 'create' | 'edit'
    formData: UserTierGroupFormValue
    submitting?: boolean
  }

  interface Emits {
    (e: 'update:visible', value: boolean): void
    (e: 'submit', value: UserTierGroupFormValue): void
  }

  const iconOptions = [
    { value: 'badge', label: '认证徽章' },
    { value: 'star', label: '星标' },
    { value: 'crown', label: '皇冠' },
    { value: 'rocket', label: '火箭' },
    { value: 'trophy', label: '奖杯' },
    { value: 'fire', label: '火焰' },
    { value: 'thunder', label: '闪电' },
    { value: 'gift', label: '礼物' },
    { value: 'heart', label: '爱心' }
  ]

  const props = withDefaults(defineProps<Props>(), {
    submitting: false
  })
  const emit = defineEmits<Emits>()

  const formRef = ref<FormInstance>()

  const dialogVisible = computed({
    get: () => props.visible,
    set: (value) => emit('update:visible', value)
  })

  const dialogTitle = computed(() => (props.mode === 'edit' ? '编辑用户组' : '创建用户组'))

  const localForm = reactive<UserTierGroupFormValue>(createDefaultForm())

  const rules = computed<FormRules>(() => ({
    name: [{ required: true, message: '请输入用户组名称', trigger: 'blur' }],
    priority: [{ required: true, message: '请输入优先级', trigger: 'change' }]
  }))

  watch(
    () => props.visible,
    (visible) => {
      if (!visible) {
        return
      }

      Object.assign(localForm, createDefaultForm(), props.formData)
      nextTick(() => formRef.value?.clearValidate())
    },
    { immediate: true }
  )

  function createDefaultForm(): UserTierGroupFormValue {
    return {
      id: null,
      name: '',
      color: '#1677ff',
      icon: 'badge',
      priority: 10,
      auto_approve_enabled: true,
      is_default: false
    }
  }

  async function handleSubmit() {
    if (!formRef.value) {
      return
    }

    const valid = await formRef.value.validate().catch(() => false)
    if (!valid) {
      return
    }

    emit('submit', {
      ...localForm,
      name: localForm.name.trim(),
      color: String(localForm.color || '#1677ff'),
      icon: String(localForm.icon || 'badge')
    })
  }
</script>

<style scoped lang="scss">
  .dialog-footer {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
  }

  .color-row {
    display: inline-flex;
    align-items: center;
    gap: 12px;
  }

  .full-width {
    width: 100%;
  }
</style>
