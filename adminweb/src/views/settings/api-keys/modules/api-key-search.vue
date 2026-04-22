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
  interface ApiKeySearchForm {
    keyword: string
    status?: string
  }

  interface Props {
    modelValue: ApiKeySearchForm
  }

  interface Emits {
    (e: 'update:modelValue', value: ApiKeySearchForm): void
    (e: 'search', value: ApiKeySearchForm): void
    (e: 'reset'): void
  }

  defineOptions({ name: 'ApiKeySearch' })

  const props = defineProps<Props>()
  const emit = defineEmits<Emits>()

  const searchBarRef = ref()

  const formData = computed({
    get: () => props.modelValue,
    set: (value) => emit('update:modelValue', value)
  })

  const formItems = computed(() => [
    {
      label: 'Keyword',
      key: 'keyword',
      type: 'input',
      props: {
        clearable: true,
        placeholder: 'Search key hash'
      }
    },
    {
      label: 'Status',
      key: 'status',
      type: 'select',
      props: {
        clearable: true,
        placeholder: 'Select status',
        options: [
          { label: 'active', value: 'active' },
          { label: 'disabled', value: 'disabled' }
        ]
      }
    }
  ])

  function handleReset() {
    emit('reset')
  }

  async function handleSearch(params: Record<string, any>) {
    await searchBarRef.value?.validate?.()
    emit('search', params as ApiKeySearchForm)
  }
</script>
