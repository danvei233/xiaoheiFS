<template>
  <ArtSearchBar
    ref="searchBarRef"
    v-model="formData"
    :items="formItems"
    :rules="rules"
    :showExpand="false"
    @reset="handleReset"
    @search="handleSearch"
  />
</template>

<script setup lang="ts">
  import { useI18n } from 'vue-i18n'

  interface SystemImageSearchForm {
    keyword: string
    status?: string
  }

  interface Props {
    modelValue: SystemImageSearchForm
  }

  interface Emits {
    (e: 'update:modelValue', value: SystemImageSearchForm): void
    (e: 'search', params: SystemImageSearchForm): void
    (e: 'reset'): void
  }

  defineOptions({ name: 'SystemImageSearch' })

  const props = defineProps<Props>()
  const emit = defineEmits<Emits>()

  const searchBarRef = ref()
  const rules = {}
  const { t } = useI18n()

  const formData = computed({
    get: () => props.modelValue,
    set: (value) => emit('update:modelValue', value)
  })

  const formItems = computed(() => [
    {
      label: t('systemImage.search.keyword'),
      key: 'keyword',
      type: 'input',
      props: {
        clearable: true,
        placeholder: t('systemImage.search.keywordPlaceholder')
      }
    },
    {
      label: t('systemImage.search.status'),
      key: 'status',
      type: 'select',
      props: {
        clearable: true,
        placeholder: t('systemImage.search.statusPlaceholder'),
        options: [
          { label: t('systemImage.status.enabled'), value: 'enabled' },
          { label: t('systemImage.status.disabled'), value: 'disabled' }
        ]
      }
    }
  ])

  function handleReset() {
    emit('reset')
  }

  async function handleSearch(params: Record<string, any>) {
    await searchBarRef.value.validate()
    emit('search', params as SystemImageSearchForm)
  }
</script>
