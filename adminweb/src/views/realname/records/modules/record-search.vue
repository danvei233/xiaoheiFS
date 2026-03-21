<template>
  <ArtSearchBar
    ref="searchBarRef"
    v-model="formData"
    :items="formItems"
    :showExpand="false"
    @reset="handleReset"
    @search="handleSearch"
  />
</template>

<script setup lang="ts">
  interface RecordSearchForm {
    user_id: string
    status?: string
  }

  interface Props {
    modelValue: RecordSearchForm
  }

  interface Emits {
    (e: 'update:modelValue', value: RecordSearchForm): void
    (e: 'search', value: RecordSearchForm): void
    (e: 'reset'): void
  }

  defineOptions({ name: 'RealnameRecordSearch' })

  const props = defineProps<Props>()
  const emit = defineEmits<Emits>()

  const searchBarRef = ref()

  const formData = computed({
    get: () => props.modelValue,
    set: (value) => emit('update:modelValue', value)
  })

  const formItems = computed(() => [
    {
      label: '用户 ID',
      key: 'user_id',
      type: 'input',
      props: {
        clearable: true,
        placeholder: '按用户 ID 搜索'
      }
    },
    {
      label: '状态',
      key: 'status',
      type: 'select',
      props: {
        clearable: true,
        placeholder: '请选择状态',
        options: [
          { label: '待审核', value: 'pending' },
          { label: '已通过', value: 'verified' },
          { label: '未通过', value: 'failed' }
        ]
      }
    }
  ])

  function handleReset() {
    emit('reset')
  }

  async function handleSearch(params: Record<string, unknown>) {
    await searchBarRef.value?.validate?.()
    emit('search', {
      user_id: String(params.user_id || ''),
      status: params.status ? String(params.status) : undefined
    })
  }
</script>
