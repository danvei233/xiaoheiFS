<template>
  <div class="editor-toolbar">
    <!-- 撤销/重做 -->
    <div class="toolbar-group">
      <button
        :disabled="!canUndo"
        @click="$emit('undo')"
        title="撤销 (Ctrl+Z)"
        :class="{ disabled: !canUndo }"
      >
        <span v-html="IconUndo()"></span>
      </button>
      <button
        :disabled="!canRedo"
        @click="$emit('redo')"
        title="重做 (Ctrl+Y)"
        :class="{ disabled: !canRedo }"
      >
        <span v-html="IconRedo()"></span>
      </button>
    </div>

    <!-- 块级元素 -->
    <div class="toolbar-group">
      <select @change="$emit('formatBlock', ($event.target as HTMLSelectElement).value)">
        <option value="p">段落</option>
        <option value="h1">标题 1</option>
        <option value="h2">标题 2</option>
        <option value="h3">标题 3</option>
        <option value="blockquote">引用</option>
      </select>
    </div>

    <!-- 文本样式 -->
    <div class="toolbar-group">
      <button
        @click="$emit('formatInline', 'bold')"
        title="加粗 (Ctrl+B)"
        :class="{ active: isBold }"
      >
        <span v-html="IconBold()"></span>
      </button>
      <button
        @click="$emit('formatInline', 'italic')"
        title="斜体 (Ctrl+I)"
        :class="{ active: isItalic }"
      >
        <span v-html="IconItalic()"></span>
      </button>
      <button
        @click="$emit('formatInline', 'underline')"
        title="下划线 (Ctrl+U)"
        :class="{ active: isUnderline }"
      >
        <span v-html="IconUnderline()"></span>
      </button>
      <button
        @click="$emit('formatInline', 'strikeThrough')"
        title="删除线"
        :class="{ active: isStrikeThrough }"
      >
        <span v-html="IconStrikethrough()"></span>
      </button>
    </div>

    <!-- 对齐方式 -->
    <div class="toolbar-group">
      <button
        @click="$emit('align', 'left')"
        title="左对齐"
        :class="{ active: align === 'left' }"
      >
        <span v-html="IconAlignLeft()"></span>
      </button>
      <button
        @click="$emit('align', 'center')"
        title="居中"
        :class="{ active: align === 'center' }"
      >
        <span v-html="IconAlignCenter()"></span>
      </button>
      <button
        @click="$emit('align', 'right')"
        title="右对齐"
        :class="{ active: align === 'right' }"
      >
        <span v-html="IconAlignRight()"></span>
      </button>
      <button
        @click="$emit('align', 'justify')"
        title="两端对齐"
        :class="{ active: align === 'justify' }"
      >
        <span v-html="IconAlignJustify()"></span>
      </button>
    </div>

    <!-- 列表 -->
    <div class="toolbar-group">
      <button
        @click="$emit('list', 'unordered')"
        title="无序列表"
        :class="{ active: isUnorderedList }"
      >
        <span v-html="IconListUnordered()"></span>
      </button>
      <button
        @click="$emit('list', 'ordered')"
        title="有序列表"
        :class="{ active: isOrderedList }"
      >
        <span v-html="IconListOrdered()"></span>
      </button>
    </div>

    <!-- 缩进 -->
    <div class="toolbar-group">
      <button @click="$emit('outdent')" title="减少缩进">
        <span v-html="IconOutdent()"></span>
      </button>
      <button @click="$emit('indent')" title="增加缩进">
        <span v-html="IconIndent()"></span>
      </button>
    </div>

    <!-- 链接和图片 -->
    <div class="toolbar-group">
      <button @click="$emit('insertLink')" title="插入链接">
        <span v-html="IconLink()"></span>
      </button>
      <button @click="$emit('insertImage')" title="插入图片">
        <span v-html="IconImage()"></span>
      </button>
    </div>

    <!-- 表格 -->
    <div class="toolbar-group">
      <button @click="$emit('insertTable')" title="插入表格">
        <span v-html="IconTable()"></span>
      </button>
    </div>

    <!-- 清除格式 -->
    <div class="toolbar-group">
      <button @click="$emit('removeFormat')" title="清除格式">
        <span v-html="IconRemoveFormat()"></span>
      </button>
    </div>

    <!-- 模式切换 -->
    <div class="toolbar-group">
      <button
        @click="$emit('toggleMode')"
        title="切换编辑模式"
        :class="{ active: mode === 'code' }"
      >
        <span v-html="IconCode()"></span>
      </button>
    </div>

    <!-- 全屏 -->
    <div class="toolbar-group" style="margin-left: auto; border-right: none;">
      <button @click="$emit('toggleFullscreen')" title="全屏">
        <span v-html="IconFullscreen()"></span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue';
import {
  IconUndo,
  IconRedo,
  IconBold,
  IconItalic,
  IconUnderline,
  IconStrikethrough,
  IconAlignLeft,
  IconAlignCenter,
  IconAlignRight,
  IconAlignJustify,
  IconListUnordered,
  IconListOrdered,
  IconIndent,
  IconOutdent,
  IconLink,
  IconImage,
  IconTable,
  IconCode,
  IconFullscreen,
  IconRemoveFormat
} from './icons';

const props = defineProps<{
  canUndo: boolean;
  canRedo: boolean;
  mode: 'rich' | 'code';
}>();

defineEmits<{
  undo: [];
  redo: [];
  formatInline: [command: string];
  formatBlock: [tag: string];
  align: [align: 'left' | 'center' | 'right' | 'justify'];
  list: [type: 'ordered' | 'unordered'];
  indent: [];
  outdent: [];
  insertLink: [];
  insertImage: [];
  insertTable: [];
  removeFormat: [];
  toggleMode: [];
  toggleFullscreen: [];
}>();

// 格式状态
const isBold = ref(false);
const isItalic = ref(false);
const isUnderline = ref(false);
const isStrikeThrough = ref(false);
const align = ref<'left' | 'center' | 'right' | 'justify'>('left');
const isUnorderedList = ref(false);
const isOrderedList = ref(false);

// 更新格式状态
const updateFormatState = () => {
  isBold.value = document.queryCommandState('bold');
  isItalic.value = document.queryCommandState('italic');
  isUnderline.value = document.queryCommandState('underline');
  isStrikeThrough.value = document.queryCommandState('strikeThrough');

  if (document.queryCommandState('justifyLeft')) align.value = 'left';
  else if (document.queryCommandState('justifyCenter')) align.value = 'center';
  else if (document.queryCommandState('justifyRight')) align.value = 'right';
  else if (document.queryCommandState('justifyFull')) align.value = 'justify';

  isUnorderedList.value = document.queryCommandState('insertUnorderedList');
  isOrderedList.value = document.queryCommandState('insertOrderedList');
};

// 监听选区变化
const handleSelectionChange = () => {
  updateFormatState();
};

onMounted(() => {
  document.addEventListener('selectionchange', handleSelectionChange);
  updateFormatState();
});

onUnmounted(() => {
  document.removeEventListener('selectionchange', handleSelectionChange);
});
</script>

<style scoped>
.editor-toolbar {
  display: flex;
  gap: 4px;
  padding: 8px;
  border-bottom: 1px solid var(--border, #e6e8ec);
  background: var(--bg-soft, #f5f6f8);
  flex-wrap: wrap;
  align-items: center;
}

.toolbar-group {
  display: flex;
  gap: 2px;
  padding-right: 8px;
  margin-right: 8px;
  border-right: 1px solid var(--border, #e6e8ec);
}

.toolbar-group:last-child {
  border-right: none;
}

.toolbar-group button {
  width: 32px;
  height: 32px;
  border: 1px solid transparent;
  background: transparent;
  border-radius: 4px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text2, #666);
  transition: all 0.2s;
  padding: 0;
}

.toolbar-group button:hover:not(.disabled) {
  background: var(--primary, #1890ff);
  color: white;
  border-color: var(--primary, #1890ff);
}

.toolbar-group button.active {
  background: var(--primary, #1890ff);
  color: white;
  border-color: var(--primary, #1890ff);
}

.toolbar-group button.disabled {
  opacity: 0.3;
  cursor: not-allowed;
}

.toolbar-group button :deep(svg) {
  width: 16px;
  height: 16px;
}

.toolbar-group select {
  height: 32px;
  padding: 0 8px;
  border: 1px solid var(--border, #e6e8ec);
  border-radius: 4px;
  background: white;
  color: var(--text1, #1f2329);
  cursor: pointer;
  font-size: 14px;
}

.toolbar-group select:hover {
  border-color: var(--primary, #1890ff);
}

.toolbar-group select:focus {
  outline: none;
  border-color: var(--primary, #1890ff);
}
</style>