<template>
  <textarea
    ref="textareaRef"
    v-model="codeValue"
    class="editor-code"
    :rows="rows"
    @input="handleInput"
    @keydown="handleKeydown"
  />
</template>

<script setup lang="ts">
import { ref, watch, nextTick } from 'vue';

const props = defineProps<{
  modelValue: string;
  rows?: number;
}>();

const emit = defineEmits<{
  'update:modelValue': [value: string];
}>();

const textareaRef = ref<HTMLTextAreaElement | null>(null);
const codeValue = ref(props.modelValue);

watch(() => props.modelValue, (newValue) => {
  if (newValue !== codeValue.value) {
    codeValue.value = newValue;
  }
}, { immediate: true });

const handleInput = () => {
  emit('update:modelValue', codeValue.value);
};

const handleKeydown = (e: KeyboardEvent) => {
  // Tab 键插入两个空格
  if (e.key === 'Tab') {
    e.preventDefault();
    const textarea = textareaRef.value;
    if (!textarea) return;

    const start = textarea.selectionStart;
    const end = textarea.selectionEnd;
    const value = codeValue.value;

    codeValue.value = value.substring(0, start) + '  ' + value.substring(end);
    emit('update:modelValue', codeValue.value);

    nextTick(() => {
      textarea.selectionStart = textarea.selectionEnd = start + 2;
    });
  }
};

defineExpose({
  focus: () => textareaRef.value?.focus()
});
</script>

<style scoped>
.editor-code {
  width: 100%;
  min-height: 300px;
  padding: 16px;
  font-family: 'Courier New', Courier, monospace;
  font-size: 14px;
  line-height: 1.6;
  border: none;
  outline: none;
  resize: vertical;
  background: transparent;
  color: inherit;
}

.editor-code::placeholder {
  color: #999;
}
</style>