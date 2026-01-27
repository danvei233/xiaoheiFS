<template>
  <div
    class="editable-region"
    :class="{ 'is-editing': isEditing, 'is-hovering': isHovering }"
    @mouseenter="isHovering = true"
    @mouseleave="isHovering = false"
  >
    <!-- 编辑图标（悬浮时显示） -->
    <div v-if="!isEditing && (isHovering || alwaysShowControls)" class="edit-icon">
      <svg width="14" height="14" viewBox="0 0 16 16" fill="none">
        <path
          d="M11.5 3.5L12.5 4.5M3 13L12.5 3.5L13.5 4.5L4 14L3 13Z"
          stroke="currentColor"
          stroke-width="1.5"
          stroke-linecap="round"
          stroke-linejoin="round"
        />
      </svg>
    </div>

    <!-- 数组项控制按钮 -->
    <div
      v-if="isArrayItem && !isEditing && (isHovering || alwaysShowControls)"
      class="array-item-controls"
    >
      <button
        v-if="canRemove"
        class="array-btn remove"
        @click.stop="handleRemove"
        :title="'删除'"
      >
        <svg width="12" height="12" viewBox="0 0 16 16" fill="none">
          <path
            d="M3 5H13M5 5V13M11 5V13M5 8H11"
            stroke="currentColor"
            stroke-width="1.5"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
        </svg>
      </button>
      <button
        v-if="canAdd"
        class="array-btn add"
        @click.stop="handleAdd"
        :title="'添加'"
      >
        <svg width="12" height="12" viewBox="0 0 16 16" fill="none">
          <path
            d="M8 3V13M3 8H13"
            stroke="currentColor"
            stroke-width="1.5"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
        </svg>
      </button>
    </div>

    <!-- 编辑模式：Popover 编辑器 -->
    <a-popover
      v-if="!isEditing"
      v-model:open="popoverVisible"
      :title="label || '编辑'"
      trigger="click"
      placement="topLeft"
      @openChange="handlePopoverChange"
    >
      <template #content>
        <div class="inline-edit-content">
          <!-- 文本输入 -->
          <a-input
            v-if="editType === 'text' || editType === 'url'"
            :model-value="modelValue"
            @update:model-value="handleUpdate"
            :placeholder="placeholder"
            @keydown.enter="handleSave"
            @keydown.esc="handleCancel"
            size="small"
            ref="inputRef"
            autofocus
          />
          <!-- 多行文本 -->
          <a-textarea
            v-else-if="editType === 'textarea'"
            :model-value="modelValue"
            @update:model-value="handleUpdate"
            :placeholder="placeholder"
            :rows="rows || 3"
            size="small"
            ref="inputRef"
            autofocus
          />
          <!-- 数字输入 -->
          <a-input-number
            v-else-if="editType === 'number'"
            :model-value="modelValue"
            @update:model-value="handleUpdate"
            :placeholder="placeholder"
            size="small"
            ref="inputRef"
            autofocus
            style="width: 100%"
          />
          <!-- 操作按钮 -->
          <div class="inline-edit-actions">
            <a-button size="small" @click="handleCancel">取消</a-button>
            <a-button size="small" type="primary" @click="handleSave">确定</a-button>
          </div>
        </div>
      </template>
      <div class="editable-content" @click="handleClick">
        <slot>{{ displayValue }}</slot>
      </div>
    </a-popover>

    <!-- 编辑中：内联输入 -->
    <div v-else class="inline-editing">
      <a-input
        v-if="editType === 'text' || editType === 'url'"
        :model-value="modelValue"
        @update:model-value="handleUpdate"
        :placeholder="placeholder"
        @keydown.enter="handleSave"
        @keydown.esc="handleCancel"
        size="small"
        ref="inputRef"
        autofocus
      />
      <a-textarea
        v-else-if="editType === 'textarea'"
        :model-value="modelValue"
        @update:model-value="handleUpdate"
        :placeholder="placeholder"
        :rows="rows || 3"
        size="small"
        ref="inputRef"
        autofocus
      />
      <a-input-number
        v-else-if="editType === 'number'"
        :model-value="modelValue"
        @update:model-value="handleUpdate"
        :placeholder="placeholder"
        size="small"
        ref="inputRef"
        autofocus
        style="width: 100%"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, nextTick, watch, onMounted } from 'vue'

interface Props {
  fieldPath?: string
  modelValue: any
  editType?: 'text' | 'textarea' | 'number' | 'url'
  isArrayItem?: boolean
  canAdd?: boolean
  canRemove?: boolean
  label?: string
  placeholder?: string
  rows?: number
  alwaysShowControls?: boolean
}

interface Emits {
  (e: 'update:modelValue', value: any): void
  (e: 'edit'): void
  (e: 'save'): void
  (e: 'cancel'): void
  (e: 'addItem'): void
  (e: 'removeItem'): void
}

const props = withDefaults(defineProps<Props>(), {
  editType: 'text',
  isArrayItem: false,
  canAdd: false,
  canRemove: false,
  alwaysShowControls: false,
  rows: 3
})

const emit = defineEmits<Emits>()

// Debug: Log when component mounts
onMounted(() => {
  console.log('[InlineEdit] Component mounted!', {
    fieldPath: props.fieldPath,
    modelValue: props.modelValue,
    isArrayItem: props.isArrayItem
  })
})

const isHovering = ref(false)
const isEditing = ref(false)
const popoverVisible = ref(false)
const inputRef = ref<any>(null)

const displayValue = computed(() => {
  if (props.modelValue === null || props.modelValue === undefined) {
    return props.placeholder || '点击编辑'
  }
  return props.modelValue
})

const handleClick = () => {
  emit('edit')
}

const handlePopoverChange = (open: boolean) => {
  if (open) {
    nextTick(() => {
      inputRef.value?.focus()
    })
  }
}

const handleUpdate = (value: any) => {
  emit('update:modelValue', value)
}

const handleSave = () => {
  isEditing.value = false
  popoverVisible.value = false
  emit('save')
}

const handleCancel = () => {
  isEditing.value = false
  popoverVisible.value = false
  emit('cancel')
}

const handleAdd = () => {
  emit('addItem')
}

const handleRemove = () => {
  emit('removeItem')
}

// 暴露方法供外部调用
defineExpose({
  startEdit: () => {
    isEditing.value = true
    nextTick(() => {
      inputRef.value?.focus()
    })
  },
  endEdit: () => {
    isEditing.value = false
  }
})
</script>

<style scoped>
.editable-region {
  position: relative;
  display: inline-block;
  transition: all 0.2s ease;
  border-radius: 4px;
  padding: 4px 8px;
  min-width: 20px;
  min-height: 20px;
  cursor: pointer;
  border: 1px dashed transparent;
}

.editable-region:hover {
  border: 1px dashed rgba(24, 144, 255, 0.6);
  background: rgba(24, 144, 255, 0.05);
  border-radius: 4px;
}

.editable-region.is-editing {
  border: 1px solid #1677ff;
  background: white;
  box-shadow: 0 2px 8px rgba(24, 144, 255, 0.15);
  z-index: 10;
  cursor: default;
}

.edit-icon {
  position: absolute;
  top: -8px;
  right: -8px;
  width: 20px;
  height: 20px;
  background: #1677ff;
  color: white;
  border-radius: 50%;
  border: 2px solid white;
  display: flex;
  align-items: center;
  justify-content: center;
  opacity: 0;
  transition: opacity 0.2s;
  z-index: 10;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.15);
}

.editable-region:hover .edit-icon {
  opacity: 1;
}

.array-item-controls {
  position: absolute;
  top: -10px;
  right: -10px;
  display: flex;
  gap: 4px;
  opacity: 0;
  transition: opacity 0.2s;
  z-index: 10;
}

.editable-region:hover .array-item-controls {
  opacity: 1;
}

.array-btn {
  width: 20px;
  height: 20px;
  border: 2px solid white;
  border-radius: 4px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  color: white;
  z-index: 10;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.2);
  transition: transform 0.2s;
}

.array-btn.add {
  background: #52c41a;
}

.array-btn.remove {
  background: #ff4d4f;
}

.array-btn:hover {
  transform: scale(1.1);
}

.editable-content {
  display: inline-block;
  width: 100%;
}

.inline-edit-content {
  min-width: 200px;
}

.inline-edit-actions {
  display: flex;
  gap: 8px;
  margin-top: 12px;
  justify-content: flex-end;
}

.inline-editing {
  display: block;
  width: 100%;
}
</style>
