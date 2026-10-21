<template>
  <ElDialog
    v-model="dialogVisible"
    :title="dialogTitle"
    width="560px"
    destroy-on-close
    align-center
  >
    <ElForm ref="formRef" :model="localForm" :rules="rules" label-width="110px">
      <ElAlert class="dialog-alert" type="info" :closable="false">
        <template #title>{{ task?.description || '暂无任务描述' }}</template>
      </ElAlert>

      <ElFormItem label="任务键名">
        <ElInput :model-value="task?.key || '-'" disabled />
      </ElFormItem>

      <ElFormItem label="启用状态" prop="enabled">
        <ElSwitch v-model="localForm.enabled" />
      </ElFormItem>

      <ElFormItem label="执行策略" prop="strategy">
        <ElSelect v-model="localForm.strategy" placeholder="请选择执行策略">
          <ElOption label="间隔执行" value="interval" />
          <ElOption label="每日执行" value="daily" />
        </ElSelect>
      </ElFormItem>

      <ElFormItem v-if="localForm.strategy === 'interval'" label="执行间隔" prop="interval_sec">
        <ElInputNumber
          v-model="localForm.interval_sec"
          :min="1"
          :max="86400"
          :step="10"
          style="width: 100%"
        />
        <div class="field-tip">单位为秒，最小值为 1 秒。</div>
      </ElFormItem>

      <ElFormItem v-else label="执行时间" prop="daily_at">
        <ElTimePicker
          v-model="localForm.daily_at"
          format="HH:mm"
          value-format="HH:mm"
          placeholder="请选择每日执行时间"
          :clearable="false"
          style="width: 100%"
        />
        <div class="field-tip">使用 24 小时制时间。</div>
      </ElFormItem>
    </ElForm>

    <template #footer>
      <div class="dialog-footer">
        <ElButton @click="dialogVisible = false">取消</ElButton>
        <ElButton type="primary" :loading="submitting" @click="handleSubmit">保存配置</ElButton>
      </div>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  import type { ScheduledTaskRecord, ScheduledTaskStrategy } from '@/api/admin'
  import type { FormInstance, FormRules } from 'element-plus'

  defineOptions({ name: 'TaskConfigDialog' })

  interface TaskConfigFormValue {
    enabled: boolean
    strategy: ScheduledTaskStrategy
    interval_sec: number
    daily_at: string
  }

  interface Props {
    visible: boolean
    task?: ScheduledTaskRecord | null
    submitting?: boolean
  }

  interface Emits {
    (e: 'update:visible', value: boolean): void
    (e: 'submit', value: TaskConfigFormValue): void
  }

  const props = withDefaults(defineProps<Props>(), {
    task: null,
    submitting: false
  })

  const emit = defineEmits<Emits>()
  const formRef = ref<FormInstance>()

  const dialogVisible = computed({
    get: () => props.visible,
    set: (value) => emit('update:visible', value)
  })

  const dialogTitle = computed(() => {
    const name = String(props.task?.name || '')
    return name ? `配置任务 · ${name}` : '配置任务'
  })

  const localForm = reactive<TaskConfigFormValue>(createDefaultForm())

  const rules = computed<FormRules>(() => ({
    strategy: [{ required: true, message: '请选择执行策略', trigger: 'change' }],
    interval_sec: [
      {
        validator: (_rule, value, callback) => {
          if (localForm.strategy !== 'interval') {
            callback()
            return
          }

          if (!Number.isFinite(Number(value)) || Number(value) < 1) {
            callback(new Error('请输入大于等于 1 的执行间隔'))
            return
          }

          callback()
        },
        trigger: 'change'
      }
    ],
    daily_at: [
      {
        validator: (_rule, value, callback) => {
          if (localForm.strategy !== 'daily') {
            callback()
            return
          }

          if (!String(value || '').trim()) {
            callback(new Error('请选择每日执行时间'))
            return
          }

          callback()
        },
        trigger: 'change'
      }
    ]
  }))

  watch(
    () => [props.visible, props.task] as const,
    ([visible]) => {
      if (!visible) {
        return
      }

      applyTask(props.task)
      nextTick(() => formRef.value?.clearValidate())
    },
    { immediate: true, deep: true }
  )

  watch(
    () => localForm.strategy,
    () => {
      nextTick(() => formRef.value?.clearValidate(['interval_sec', 'daily_at']))
    }
  )

  function createDefaultForm(): TaskConfigFormValue {
    return {
      enabled: false,
      strategy: 'interval',
      interval_sec: 300,
      daily_at: '09:00'
    }
  }

  function applyTask(task?: ScheduledTaskRecord | null) {
    Object.assign(localForm, createDefaultForm(), {
      enabled: Boolean(task?.enabled),
      strategy: task?.strategy === 'daily' ? 'daily' : 'interval',
      interval_sec: Number(task?.interval_sec || 300),
      daily_at: String(task?.daily_at || '09:00')
    })
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
      enabled: localForm.enabled,
      strategy: localForm.strategy,
      interval_sec: Number(localForm.interval_sec || 0),
      daily_at: String(localForm.daily_at || '')
    })
  }
</script>

<style scoped lang="scss">
  .dialog-alert {
    margin-bottom: 18px;
  }

  .dialog-footer {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
  }

  .field-tip {
    margin-top: 8px;
    color: var(--el-text-color-secondary);
    font-size: 12px;
    line-height: 1.5;
  }
</style>
