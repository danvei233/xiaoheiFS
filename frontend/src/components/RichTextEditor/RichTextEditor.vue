<template>
  <div class="rich-text-editor" :class="{ fullscreen: isFullscreen }">
    <!-- 工具栏 - rich 模式 -->
    <div v-if="mode === 'rich'" class="editor-toolbar">
      <div class="toolbar-group">
        <button
          :disabled="!commands.canUndo()"
          @click="commands.undo"
          title="撤销 (Ctrl+Z)"
          :class="{ disabled: !commands.canUndo() }"
        >
          <span v-html="IconUndo()"></span>
        </button>
        <button
          :disabled="!commands.canRedo()"
          @click="commands.redo"
          title="重做 (Ctrl+Y)"
          :class="{ disabled: !commands.canRedo() }"
        >
          <span v-html="IconRedo()"></span>
        </button>
      </div>

      <div class="toolbar-group">
        <select @change="commands.formatBlock(($event.target as HTMLSelectElement).value)">
          <option value="p">段落</option>
          <option value="h1">标题 1</option>
          <option value="h2">标题 2</option>
          <option value="h3">标题 3</option>
          <option value="blockquote">引用</option>
        </select>
      </div>

      <div class="toolbar-group">
        <button
          @click="commands.formatInline('bold')"
          title="加粗 (Ctrl+B)"
          :class="{ active: isBold }"
        >
          <span v-html="IconBold()"></span>
        </button>
        <button
          @click="commands.formatInline('italic')"
          title="斜体 (Ctrl+I)"
          :class="{ active: isItalic }"
        >
          <span v-html="IconItalic()"></span>
        </button>
        <button
          @click="commands.formatInline('underline')"
          title="下划线 (Ctrl+U)"
          :class="{ active: isUnderline }"
        >
          <span v-html="IconUnderline()"></span>
        </button>
        <button
          @click="commands.formatInline('strikeThrough')"
          title="删除线"
          :class="{ active: isStrikeThrough }"
        >
          <span v-html="IconStrikethrough()"></span>
        </button>
      </div>

      <div class="toolbar-group">
        <button
          @click="commands.changeAlignment('left')"
          title="左对齐"
          :class="{ active: align === 'left' }"
        >
          <span v-html="IconAlignLeft()"></span>
        </button>
        <button
          @click="commands.changeAlignment('center')"
          title="居中"
          :class="{ active: align === 'center' }"
        >
          <span v-html="IconAlignCenter()"></span>
        </button>
        <button
          @click="commands.changeAlignment('right')"
          title="右对齐"
          :class="{ active: align === 'right' }"
        >
          <span v-html="IconAlignRight()"></span>
        </button>
        <button
          @click="commands.changeAlignment('justify')"
          title="两端对齐"
          :class="{ active: align === 'justify' }"
        >
          <span v-html="IconAlignJustify()"></span>
        </button>
      </div>

      <div class="toolbar-group">
        <button
          @click="commands.insertList('unordered')"
          title="无序列表"
          :class="{ active: isUnorderedList }"
        >
          <span v-html="IconListUnordered()"></span>
        </button>
        <button
          @click="commands.insertList('ordered')"
          title="有序列表"
          :class="{ active: isOrderedList }"
        >
          <span v-html="IconListOrdered()"></span>
        </button>
      </div>

      <div class="toolbar-group">
        <button @click="commands.outdent()" title="减少缩进">
          <span v-html="IconOutdent()"></span>
        </button>
        <button @click="commands.indent()" title="增加缩进">
          <span v-html="IconIndent()"></span>
        </button>
      </div>

      <div class="toolbar-group">
        <button @click="handleInsertLink" title="插入链接">
          <span v-html="IconLink()"></span>
        </button>
        <button @click="handleInsertImage" title="插入图片">
          <span v-html="IconImage()"></span>
        </button>
      </div>

      <div class="toolbar-group">
        <button @click="handleInsertTable" title="插入表格">
          <span v-html="IconTable()"></span>
        </button>
      </div>

      <div class="toolbar-group">
        <button @click="commands.removeFormat()" title="清除格式">
          <span v-html="IconRemoveFormat()"></span>
        </button>
      </div>

      <!-- 变量选择器 -->
      <div class="toolbar-group">
        <VariableSelector @insert="handleInsertVariable" />
      </div>

      <div class="toolbar-group" style="margin-left: auto; border-right: none;">
        <button
          @click="mode = 'code'"
          title="切换到 HTML 模式"
          :class="{ active: mode === 'code' }"
        >
          <span v-html="IconCode()"></span>
        </button>
        <button @click="isFullscreen = !isFullscreen" title="全屏">
          <span v-html="IconFullscreen()"></span>
        </button>
      </div>
    </div>

    <!-- 工具栏 - code 模式 -->
    <div v-else class="editor-toolbar">
      <div class="toolbar-group" style="margin-left: auto; border-right: none;">
        <button
          @click="mode = 'rich'"
          title="切换到可视化模式"
          :class="{ active: mode === 'code' }"
        >
          <span v-html="IconCode()"></span>
        </button>
        <button @click="isFullscreen = !isFullscreen" title="全屏">
          <span v-html="IconFullscreen()"></span>
        </button>
      </div>
    </div>

    <div class="editor-container">
      <EditorContent
        v-if="mode === 'rich'"
        ref="contentRef"
        v-model="internalValue"
        placeholder="在此输入内容..."
        :can-undo="commands.canUndo()"
        :can-redo="commands.canRedo()"
        @undo="commands.undo"
        @redo="commands.redo"
        @remove-format="commands.removeFormat"
      />
      <EditorCodeView
        v-else
        v-model="internalValue"
        :rows="rows"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted } from 'vue';
import EditorContent from './EditorContent.vue';
import EditorCodeView from './EditorCodeView.vue';
import VariableSelector from './VariableSelector.vue';
import { useEditorCommands } from './useEditorCommands';
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
  modelValue: string;
  height?: number;
  rows?: number;
}>();

const emit = defineEmits<{
  'update:modelValue': [value: string];
}>();

const contentRef = ref<InstanceType<typeof EditorContent> | null>(null);
const editorRef = ref<HTMLElement | null>(null);
const isFullscreen = ref(false);
const mode = ref<'rich' | 'code'>('rich');

// 格式状态
const isBold = ref(false);
const isItalic = ref(false);
const isUnderline = ref(false);
const isStrikeThrough = ref(false);
const align = ref<'left' | 'center' | 'right' | 'justify'>('left');
const isUnorderedList = ref(false);
const isOrderedList = ref(false);

const internalValue = ref(props.modelValue);

// 获取编辑器元素
const getEditorElement = (): HTMLElement | null => {
  return contentRef.value?.getEditor() || null;
};

// 初始化命令
const commands = useEditorCommands(editorRef);

watch(() => props.modelValue, (newValue) => {
  internalValue.value = newValue;
});

watch(internalValue, (newValue) => {
  emit('update:modelValue', newValue);
});

// 自动判断编辑器模式
const autoDetectMode = () => {
  const isHtmlContent = /<\/?[a-z][\s\S]*>/i.test(String(internalValue.value || ''));
  mode.value = isHtmlContent ? 'rich' : 'rich';
};

// 插入链接
const handleInsertLink = () => {
  const url = prompt('请输入链接地址：');
  if (url) {
    const text = prompt('请输入链接文本（可选）：', url);
    commands.insertLink(url, text || undefined);
  }
};

// 插入图片
const handleInsertImage = () => {
  const src = prompt('请输入图片地址：');
  if (src) {
    const alt = prompt('请输入图片描述（可选）：') || '';
    commands.insertImage(src, alt);
  }
};

// 插入表格
const handleInsertTable = () => {
  const rows = parseInt(prompt('请输入行数：', '3') || '3', 10);
  const cols = parseInt(prompt('请输入列数：', '3') || '3', 10);

  if (rows > 0 && cols > 0 && rows <= 20 && cols <= 20) {
    commands.insertTable(rows, cols);
  } else {
    alert('请输入有效的行数和列数（1-20）');
  }
};

// 插入变量
const handleInsertVariable = (variable: string) => {
  contentRef.value?.insertVariable(variable);
};

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
  if (mode.value === 'rich') {
    updateFormatState();
  }
};

// 全屏快捷键
const handleKeydown = (e: KeyboardEvent) => {
  if (e.key === 'Escape' && isFullscreen.value) {
    isFullscreen.value = false;
  }

  if ((e.ctrlKey || e.metaKey) && e.shiftKey && e.key === 'S') {
    e.preventDefault();
    mode.value = mode.value === 'rich' ? 'code' : 'rich';
  }
};

onMounted(() => {
  autoDetectMode();

  // 延迟初始化编辑器命令
  setTimeout(() => {
    editorRef.value = getEditorElement();
    if (editorRef.value) {
      commands.initHistory();
    }
  }, 100);

  document.addEventListener('keydown', handleKeydown);
  document.addEventListener('selectionchange', handleSelectionChange);
  updateFormatState();
});

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown);
  document.removeEventListener('selectionchange', handleSelectionChange);
});

// 暴露方法
defineExpose({
  focus: () => {
    if (contentRef.value) {
      contentRef.value.focus();
    }
  },
  getContent: () => contentRef.value?.getContent() || '',
  setContent: (html: string) => contentRef.value?.setContent(html)
});
</script>

<style scoped>
.rich-text-editor {
  border: 1px solid var(--border, #e6e8ec);
  border-radius: var(--radius-sm, 6px);
  background: var(--card, #ffffff);
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

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
  align-items: center;
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

.editor-container {
  display: flex;
  flex-direction: column;
  flex: 1;
  overflow: hidden;
}

/* 全屏模式 */
.rich-text-editor.fullscreen {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 9999;
  border-radius: 0;
}

.rich-text-editor.fullscreen .editor-container {
  flex: 1;
  overflow: hidden;
}
</style>