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
  interface WalletOrderSearchForm {
    status?: string
    user_id: string
  }

  interface Props {
    modelValue: WalletOrderSearchForm
  }

  interface Emits {
    (e: 'update:modelValue', value: WalletOrderSearchForm): void
    (e: 'search', value: WalletOrderSearchForm): void
    (e: 'reset'): void
  }

  defineOptions({ name: 'WalletOrderSearch' })

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
          { label: '待审核', value: 'pending_review' },
          { label: '已通过', value: 'approved' },
          { label: '已驳回', value: 'rejected' }
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
    }
  ])

  function handleReset() {
    emit('reset')
  }

  async function handleSearch(params: Record<string, any>) {
    await searchBarRef.value?.validate?.()
    emit('search', params as WalletOrderSearchForm)
  }
</script>
