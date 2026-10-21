import { computed, reactive, watch } from 'vue'

export interface SystemImageDialogFormValue {
  id: number | null
  image_id: number | null
  name: string
  type: string
  enabled: boolean
}

interface SystemImageDialogBindingProps {
  visible: boolean
  formData: SystemImageDialogFormValue
}

export function createDefaultSystemImageDialogForm(): SystemImageDialogFormValue {
  return {
    id: null,
    image_id: null,
    name: '',
    type: 'linux',
    enabled: true
  }
}

export function useSystemImageDialogBinding(
  props: SystemImageDialogBindingProps,
  updateVisible: (value: boolean) => void
) {
  const localForm = reactive<SystemImageDialogFormValue>(createDefaultSystemImageDialogForm())

  const dialogVisible = computed({
    get: () => props.visible,
    set: (value: boolean) => updateVisible(value)
  })

  watch(
    () => [props.visible, props.formData] as const,
    ([visible]) => {
      if (!visible) {
        return
      }

      Object.assign(localForm, createDefaultSystemImageDialogForm(), props.formData)
    },
    { immediate: true, deep: true }
  )

  return {
    dialogVisible,
    localForm
  }
}
