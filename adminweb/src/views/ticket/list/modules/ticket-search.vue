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
  interface TicketSearchForm {
    status?: string
    user_id: string
    q: string
  }

  interface Props {
    modelValue: TicketSearchForm
  }

  interface Emits {
    (e: 'update:modelValue', value: TicketSearchForm): void
    (e: 'search', value: TicketSearchForm): void
    (e: 'reset'): void
  }

  defineOptions({ name: 'TicketSearch' })

  const props = defineProps<Props>()
  const emit = defineEmits<Emits>()

  const searchBarRef = ref()

  const formData = computed({
    get: () => props.modelValue,
    set: (value) => emit('update:modelValue', value)
  })

  const formItems = computed(() => [
    {
      label: '状态',
      key: 'status',
      type: 'select',
      props: {
        clearable: true,
        placeholder: '请选择状态',
        options: [
          { label: '待处理', value: 'open' },
          { label: '等待回复', value: 'waiting_user' },
          { label: '处理中', value: 'waiting_admin' },
          { label: '已关闭', value: 'closed' }
        ]
      }
    },
    {
      label: '用户 ID',
      key: 'user_id',
      type: 'input',
      props: {
        clearable: true,
        placeholder: '请输入用户 ID'
      }
    },
    {
      label: '关键词',
      key: 'q',
      type: 'input',
      props: {
        clearable: true,
        placeholder: '搜索标题或内容'
      }
    }
  ])

  function handleReset() {
    emit('reset')
  }

  async function handleSearch(params: Record<string, any>) {
    await searchBarRef.value?.validate?.()
    emit('search', params as TicketSearchForm)
  }
</script>
