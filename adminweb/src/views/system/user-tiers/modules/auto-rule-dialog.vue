<template>
  <ElDialog
    v-model="dialogVisible"
    :title="dialogTitle"
    width="720px"
    destroy-on-close
    align-center
  >
    <ElForm ref="formRef" :model="localForm" :rules="rules" label-position="top">
      <ElRow :gutter="12">
        <ElCol :span="12">
          <ElFormItem label="时长(天，-1 不限)" prop="duration_days">
            <ElInputNumber v-model="localForm.duration_days" :min="-1" class="full-width" />
          </ElFormItem>
        </ElCol>

        <ElCol :span="12">
          <ElFormItem label="排序" prop="sort_order">
            <ElInputNumber v-model="localForm.sort_order" :min="0" class="full-width" />
          </ElFormItem>
        </ElCol>
      </ElRow>

      <ElFormItem label="审批条件" prop="conditions_json">
        <div class="auto-hint">同一条规则内为 AND；不同规则记录之间为 OR。</div>

        <div class="condition-list">
          <div v-for="(item, index) in conditionRows" :key="index" class="condition-row">
            <ElSelect v-model="item.metric" placeholder="条件项" class="condition-control">
              <ElOption
                v-for="option in metricOptions"
                :key="option.value"
                :label="option.label"
                :value="option.value"
              />
            </ElSelect>

            <ElSelect v-model="item.operator" placeholder="运算符" class="condition-operator">
              <ElOption
                v-for="option in operatorOptions"
                :key="option.value"
                :label="option.label"
                :value="option.value"
              />
            </ElSelect>

            <ElInputNumber v-model="item.value" placeholder="目标值" class="condition-control" />

            <ElButton text type="danger" @click="removeCondition(index)">删除</ElButton>
          </div>

          <ElButton @click="addCondition">添加条件</ElButton>
        </div>
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

  defineOptions({ name: 'UserTierAutoRuleDialog' })

  interface AutoCondition {
    metric: string
    operator: string
    value: number
  }

  export interface UserTierAutoRuleFormValue {
    id: number | null
    duration_days: number
    sort_order: number
    conditions_json: string
  }

  interface Props {
    visible: boolean
    mode: 'create' | 'edit'
    formData: UserTierAutoRuleFormValue
    submitting?: boolean
  }

  interface Emits {
    (e: 'update:visible', value: boolean): void
    (e: 'submit', value: UserTierAutoRuleFormValue): void
  }

  const metricOptions = [
    { label: '注册时长(月)', value: 'register_months' },
    { label: '用户钱包余额(元)', value: 'wallet_balance' }
  ]

  const operatorOptions = [
    { label: '大于', value: 'gt' },
    { label: '小于', value: 'lt' },
    { label: '等于', value: 'eq' }
  ]

  const props = withDefaults(defineProps<Props>(), {
    submitting: false
  })
  const emit = defineEmits<Emits>()

  const formRef = ref<FormInstance>()
  const conditionRows = ref<AutoCondition[]>([])

  const dialogVisible = computed({
    get: () => props.visible,
    set: (value) => emit('update:visible', value)
  })

  const dialogTitle = computed(() =>
    props.mode === 'edit' ? '编辑自动审批条件' : '新增自动审批条件'
  )

  const localForm = reactive<UserTierAutoRuleFormValue>(createDefaultForm())

  const rules = computed<FormRules>(() => ({
    sort_order: [{ required: true, message: '请输入排序', trigger: 'change' }]
  }))

  watch(
    () => props.visible,
    (visible) => {
      if (!visible) {
        return
      }

      Object.assign(localForm, createDefaultForm(), props.formData)
      conditionRows.value = parseConditionsJSON(props.formData.conditions_json)
      if (!conditionRows.value.length) {
        conditionRows.value = [createDefaultCondition()]
      }
      nextTick(() => formRef.value?.clearValidate())
    },
    { immediate: true }
  )

  function createDefaultForm(): UserTierAutoRuleFormValue {
    return {
      id: null,
      duration_days: -1,
      sort_order: 10,
      conditions_json: '[]'
    }
  }

  function createDefaultCondition(): AutoCondition {
    return {
      metric: 'register_months',
      operator: 'gt',
      value: 0
    }
  }

  function parseConditionsJSON(raw?: string) {
    const text = String(raw || '').trim()
    if (!text) {
      return []
    }

    try {
      const parsed = JSON.parse(text)
      if (!Array.isArray(parsed)) {
        return []
      }

      return parsed
        .map((item) => ({
          metric: String(item?.metric || ''),
          operator: String(item?.operator || ''),
          value: Number(item?.value || 0)
        }))
        .filter(
          (item) =>
            metricOptions.some((option) => option.value === item.metric) &&
            operatorOptions.some((option) => option.value === item.operator) &&
            Number.isFinite(item.value)
        )
    } catch {
      return []
    }
  }

  function addCondition() {
    conditionRows.value.push(createDefaultCondition())
  }

  function removeCondition(index: number) {
    conditionRows.value.splice(index, 1)
  }

  async function handleSubmit() {
    if (!formRef.value) {
      return
    }

    const valid = await formRef.value.validate().catch(() => false)
    if (!valid) {
      return
    }

    const normalized = conditionRows.value.map((item) => ({
      metric: String(item.metric || '').trim(),
      operator: String(item.operator || '').trim(),
      value: Number(item.value)
    }))

    const invalid = normalized.some(
      (item) =>
        !metricOptions.some((option) => option.value === item.metric) ||
        !operatorOptions.some((option) => option.value === item.operator) ||
        !Number.isFinite(item.value)
    )

    if (invalid) {
      ElMessage.error('审批条件存在无效项')
      return
    }

    emit('submit', {
      ...localForm,
      conditions_json: JSON.stringify(normalized)
    })
  }
</script>

<style scoped lang="scss">
  .dialog-footer {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
  }

  .auto-hint {
    margin-bottom: 8px;
    color: var(--el-text-color-secondary);
    font-size: 13px;
  }

  .condition-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .condition-row {
    display: grid;
    grid-template-columns: minmax(0, 1fr) 140px minmax(0, 1fr) auto;
    gap: 8px;
    align-items: center;
  }

  .condition-control,
  .full-width {
    width: 100%;
  }

  .condition-operator {
    width: 140px;
  }

  @media (max-width: 768px) {
    .condition-row {
      grid-template-columns: 1fr;
    }

    .condition-operator {
      width: 100%;
    }
  }
</style>
