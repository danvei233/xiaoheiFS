<template>
  <ElDialog
    v-model="dialogVisible"
    :title="
      mode === 'config'
        ? t('systemImage.lineDialog.configTitle')
        : t('systemImage.lineDialog.syncTitle')
    "
    width="560px"
    destroy-on-close
    align-center
  >
    <ElForm label-position="top">
      <ElFormItem :label="t('systemImage.lineDialog.line')">
        <ElSelect
          v-model="lineModel"
          :placeholder="t('systemImage.lineDialog.linePlaceholder')"
          filterable
          class="full-width"
        >
          <ElOption
            v-for="line in safeLines"
            :key="line.id"
            :label="formatLineLabel(line)"
            :value="line.id"
          />
        </ElSelect>
      </ElFormItem>

      <ElFormItem v-if="mode === 'config'" :label="t('systemImage.lineDialog.enabledImages')">
        <ElSelect
          v-model="imageIdsModel"
          class="full-width"
          multiple
          filterable
          :placeholder="t('systemImage.lineDialog.enabledImagesPlaceholder')"
        >
          <ElOption
            v-for="item in imageOptions"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </ElSelect>
      </ElFormItem>

      <ElAlert
        v-if="mode === 'sync'"
        type="info"
        show-icon
        :closable="false"
        :title="t('systemImage.lineDialog.syncAlert')"
      />
    </ElForm>

    <template #footer>
      <div class="dialog-footer">
        <ElButton @click="dialogVisible = false">{{ t('common.cancel') }}</ElButton>
        <ElButton type="primary" :loading="submitting" @click="emit('submit')">
          {{
            mode === 'config'
              ? t('systemImage.lineDialog.save')
              : t('systemImage.lineDialog.startSync')
          }}
        </ElButton>
      </div>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  import { useI18n } from 'vue-i18n'

  interface LineRecord {
    id: number | null
    name: string
  }

  interface SafeLineRecord {
    id: number
    name: string
  }

  interface ImageOption {
    label: string
    value: number
  }

  interface Props {
    visible: boolean
    mode: 'config' | 'sync'
    lineId: number | null
    imageIds: number[]
    lines: LineRecord[]
    imageOptions?: ImageOption[]
    lineImageCountMap?: Record<number, number>
    submitting?: boolean
  }

  interface Emits {
    (e: 'update:visible', value: boolean): void
    (e: 'update:lineId', value: number | null): void
    (e: 'update:imageIds', value: number[]): void
    (e: 'submit'): void
  }

  defineOptions({ name: 'LineImageDialog' })

  const props = withDefaults(defineProps<Props>(), {
    imageOptions: () => [],
    lineImageCountMap: () => ({}),
    submitting: false
  })
  const emit = defineEmits<Emits>()
  const { t } = useI18n()

  const dialogVisible = computed({
    get: () => props.visible,
    set: (value) => emit('update:visible', value)
  })

  const lineModel = computed({
    get: () => props.lineId,
    set: (value) => emit('update:lineId', value ? Number(value) : null)
  })

  const imageIdsModel = computed({
    get: () => props.imageIds,
    set: (value) =>
      emit(
        'update:imageIds',
        value.map((item) => Number(item))
      )
  })

  const safeLines = computed<SafeLineRecord[]>(() =>
    props.lines
      .filter((line): line is LineRecord & { id: number } => line.id !== null)
      .map((line) => ({
        id: Number(line.id),
        name: line.name
      }))
  )

  function formatLineLabel(line: LineRecord | SafeLineRecord) {
    const count = Number(props.lineImageCountMap?.[Number(line.id || 0)] || 0)
    return `${line.name || '-'} (${count})`
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
