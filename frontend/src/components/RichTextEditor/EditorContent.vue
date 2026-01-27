<template>
  <div class="editor-wrapper">
    <div
      ref="editorRef"
      class="editor-content"
      contenteditable="true"
      :data-placeholder="placeholder"
      @input="handleInput"
      @keydown="handleKeydown"
      @paste="handlePaste"
      @click="handleClick"
      @blur="handleBlur"
      @focus="handleFocus"
    ></div>

    <!-- 右键菜单 -->
    <ContextMenu
      :visible="contextMenuVisible"
      :x="contextMenuX"
      :y="contextMenuY"
      :menu-items="contextMenuItems"
      @close="contextMenuVisible = false"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted } from 'vue';
import ContextMenu from './ContextMenu.vue';
import {
  IconUndo,
  IconRedo,
  IconCut,
  IconCopy,
  IconPaste,
  IconRemoveFormat
} from './icons';

const props = defineProps<{
  modelValue: string;
  placeholder?: string;
  canUndo?: boolean;
  canRedo?: boolean;
}>();

const emit = defineEmits<{
  'update:modelValue': [value: string];
  undo: [];
  redo: [];
  removeFormat: [];
}>();

const editorRef = ref<HTMLElement | null>(null);
let mutationObserver: MutationObserver | null = null;

// 保存的光标位置
let savedCaret: { node: Node; offset: number } | null = null;

// 右键菜单
const contextMenuVisible = ref(false);
const contextMenuX = ref(0);
const contextMenuY = ref(0);

const contextMenuItems = ref([
  { key: 'undo', label: '撤销', icon: IconUndo(), shortcut: 'Ctrl+Z', disabled: !props.canUndo, action: () => emit('undo') },
  { key: 'redo', label: '重做', icon: IconRedo(), shortcut: 'Ctrl+Y', disabled: !props.canRedo, action: () => emit('redo') },
  { key: 'sep1', label: '', separator: true },
  { key: 'cut', label: '剪切', icon: IconCut(), shortcut: 'Ctrl+X', action: () => document.execCommand('cut') },
  { key: 'copy', label: '复制', icon: IconCopy(), shortcut: 'Ctrl+C', action: () => document.execCommand('copy') },
  { key: 'paste', label: '粘贴', icon: IconPaste(), shortcut: 'Ctrl+V', action: () => document.execCommand('paste') },
  { key: 'sep2', label: '', separator: true },
  { key: 'removeFormat', label: '清除格式', icon: IconRemoveFormat(), action: () => emit('removeFormat') },
]);

// 保存光标位置
const saveCaret = () => {
  const selection = window.getSelection();
  if (!selection || selection.rangeCount === 0 || !editorRef.value) return;

  const range = selection.getRangeAt(0);
  // 检查选区是否在编辑器内
  if (!editorRef.value.contains(range.commonAncestorContainer)) {
    savedCaret = null;
    return;
  }

  savedCaret = {
    node: range.startContainer,
    offset: range.startOffset
  };
};

// 恢复光标位置
const restoreCaret = () => {
  if (!savedCaret || !editorRef.value) return;

  // 检查保存的节点是否还在编辑器内
  if (!editorRef.value.contains(savedCaret.node)) {
    savedCaret = null;
    return;
  }

  try {
    const selection = window.getSelection();
    if (!selection) return;

    const range = document.createRange();
    range.setStart(savedCaret.node, savedCaret.offset);
    range.collapse(true);
    selection.removeAllRanges();
    selection.addRange(range);
  } catch (e) {
    // 如果恢复失败，清除保存的位置
    savedCaret = null;
  }
};

// 处理右键菜单
const handleContextMenu = (e: MouseEvent) => {
  e.preventDefault();
  contextMenuX.value = e.clientX;
  contextMenuY.value = e.clientY;

  // 更新撤销/重做状态
  contextMenuItems.value[0].disabled = !props.canUndo;
  contextMenuItems.value[1].disabled = !props.canRedo;

  contextMenuVisible.value = true;
};

// 插入变量
const insertVariable = (variable: string) => {
  if (!editorRef.value) return;

  // 聚焦编辑器
  editorRef.value.focus();

  // 恢复之前保存的光标位置
  restoreCaret();

  const selection = window.getSelection();
  if (!selection) return;

  // 如果没有选区，在末尾创建一个
  let range: Range;
  if (selection.rangeCount === 0) {
    range = document.createRange();
    const lastChild = editorRef.value.lastChild;
    if (lastChild) {
      range.setStartAfter(lastChild);
    } else {
      range.setStart(editorRef.value, 0);
    }
    range.collapse(true);
    selection.addRange(range);
  } else {
    range = selection.getRangeAt(0);
  }

  // 创建变量元素
  const span = document.createElement('span');
  span.className = 'template-variable';
  span.contentEditable = 'false';
  span.dataset.variable = variable;
  span.textContent = variable;

  range.deleteContents();
  range.insertNode(span);

  // 移动光标到变量后面
  const newRange = document.createRange();
  newRange.setStartAfter(span);
  newRange.collapse(true);
  selection.removeAllRanges();
  selection.addRange(newRange);

  emit('update:modelValue', getContent());
};

// 模板变量正则
const VARIABLE_REGEX = /(\{\{\s*\.[a-zA-Z0-9_.]+\s*\}\})/g;

// 保护模板变量（包装为不可编辑元素）
const protectVariables = (html: string): string => {
  return html.replace(VARIABLE_REGEX, '<span class="template-variable" contenteditable="false" data-variable="$1">$1</span>');
};

// 移除模板变量保护
const unprotectVariables = (html: string): string => {
  return html.replace(/<span class="template-variable"[^>]*>(.*?)<\/span>/g, '$1');
};

// 设置编辑器内容
const setContent = (html: string) => {
  if (!editorRef.value) return;
  editorRef.value.innerHTML = protectVariables(html);
};

// 获取编辑器内容（移除保护）
const getContent = (): string => {
  if (!editorRef.value) return '';
  return unprotectVariables(editorRef.value.innerHTML);
};

// 设置模板变量保护（通过 MutationObserver）
const setupVariableProtection = () => {
  if (!editorRef.value) return;

  mutationObserver = new MutationObserver((mutations) => {
    mutations.forEach((mutation) => {
      mutation.addedNodes.forEach((node) => {
        if (node.nodeType === Node.ELEMENT_NODE) {
          const variables = (node as Element).querySelectorAll('.template-variable');
          variables.forEach((span) => {
            const original = span.dataset.variable;
            if (original && span.textContent !== original) {
              span.textContent = original;
            }
          });
        }
      });
    });
  });

  mutationObserver.observe(editorRef.value, {
    childList: true,
    subtree: true,
    characterData: true
  });
};

watch(() => props.modelValue, (newValue) => {
  if (newValue !== getContent()) {
    setContent(newValue);
  }
}, { immediate: true });

const handleInput = () => {
  emit('update:modelValue', getContent());
};

// 获取光标前的文本
const getTextBeforeCursor = (): string => {
  const selection = window.getSelection();
  if (!selection || selection.rangeCount === 0) return '';

  const range = selection.getRangeAt(0);
  let text = '';

  // 获取光标前的文本
  let current = range.startContainer;
  let offset = range.startOffset;

  // 如果是文本节点
  if (current.nodeType === Node.TEXT_NODE) {
    text = current.textContent?.slice(0, offset) || '';

    // 遍历前面的兄弟节点
    let sibling = current.previousSibling;
    while (sibling) {
      if (sibling.nodeType === Node.TEXT_NODE) {
        text = sibling.textContent + text;
      }
      sibling = sibling.previousSibling;
    }

    // 遍历父节点
    let parent = current.parentElement;
    while (parent && parent !== editorRef.value) {
      let prevSibling = parent.previousSibling;
      while (prevSibling) {
        text += prevSibling.textContent || '';
        prevSibling = prevSibling.previousSibling;
      }
      parent = parent.parentElement;
    }
  } else {
    // 如果是元素节点，获取前面的文本
    const walker = document.createTreeWalker(
      editorRef.value!,
      NodeFilter.SHOW_TEXT,
      null
    );

    let node;
    while ((node = walker.nextNode())) {
      if (node === current || current.contains(node)) {
        break;
      }
      text += node.textContent || '';
    }

    if (current.nodeType === Node.TEXT_NODE && current === range.startContainer) {
      text += current.textContent?.slice(0, offset) || '';
    }
  }

  return text;
};

// 处理 Tab 键触发模板变量
const handleTabKey = (): boolean => {
  const beforeCursor = getTextBeforeCursor();

  // 检查光标前是否有 {{
  if (beforeCursor.endsWith('{{')) {
    const selection = window.getSelection();
    if (!selection || selection.rangeCount === 0) return false;

    const range = selection.getRangeAt(0);

    // 删除 {{
    range.setStart(range.startContainer, range.startOffset - 2);
    range.deleteContents();

    // 提示用户输入变量名
    const variableName = prompt('请输入变量名（如 .user.username）：');
    if (variableName) {
      const variable = `{{ ${variableName} }}`;
      insertVariable(variable);
    }
    return true;
  }

  return false;
};

const handleKeydown = (e: KeyboardEvent) => {
  // 处理 Tab 键
  if (e.key === 'Tab') {
    e.preventDefault();
    if (handleTabKey()) {
      return;
    }
    // 如果不是模板触发，插入制表符
    document.execCommand('insertText', false, '  ');
    return;
  }

  // 处理快捷键
  if (e.ctrlKey || e.metaKey) {
    switch (e.key.toLowerCase()) {
      case 'b':
        e.preventDefault();
        document.execCommand('bold', false);
        break;
      case 'i':
        e.preventDefault();
        document.execCommand('italic', false);
        break;
      case 'u':
        e.preventDefault();
        document.execCommand('underline', false);
        break;
    }
  }

  // Enter 键创建新段落
  if (e.key === 'Enter' && !e.shiftKey) {
    const selection = window.getSelection();
    if (selection && selection.rangeCount > 0) {
      const range = selection.getRangeAt(0);
      const container = range.startContainer;

      // 如果在 pre 或 code 标签内，不处理
      if (container.parentNode instanceof HTMLElement) {
        const parent = container.parentNode;
        if (parent.tagName === 'PRE' || parent.tagName === 'CODE') {
          return;
        }
      }

      // 创建新段落
      e.preventDefault();
      const p = document.createElement('p');
      p.innerHTML = '<br>';
      range.deleteContents();
      range.insertNode(p);

      // 移动光标到新段落
      const newRange = document.createRange();
      newRange.setStart(p, 0);
      newRange.collapse(true);
      selection.removeAllRanges();
      selection.addRange(newRange);
    }
  }
};

const handlePaste = (e: ClipboardEvent) => {
  e.preventDefault();
  const text = e.clipboardData?.getData('text/plain') || '';
  document.execCommand('insertText', false, text);
};

const handleClick = () => {
  editorRef.value?.focus();
};

const handleBlur = () => {
  saveCaret();
};

const handleFocus = () => {
  editorRef.value?.focus();
};

onMounted(() => {
  if (editorRef.value) {
    setContent(props.modelValue);
    setupVariableProtection();

    // 添加右键菜单事件监听
    editorRef.value.addEventListener('contextmenu', handleContextMenu);
  }
});

onUnmounted(() => {
  if (mutationObserver) {
    mutationObserver.disconnect();
  }
  if (editorRef.value) {
    editorRef.value.removeEventListener('contextmenu', handleContextMenu);
  }
});

defineExpose({
  getContent,
  setContent,
  insertVariable,
  focus: () => editorRef.value?.focus(),
  getEditor: () => editorRef.value
});
</script>

<style>
/* 全局样式 - 不使用 scoped */
.editor-wrapper {
  position: relative;
}

.editor-content {
  min-height: 300px;
  padding: 16px;
  outline: none;
  overflow-y: auto;
  word-wrap: break-word;
  line-height: 1.6;
}

.editor-content:empty:before {
  content: attr(data-placeholder);
  color: #999;
  pointer-events: none;
  position: absolute;
  cursor: text;
}

/* 模板变量样式 - 淡灰色背景 */
.editor-content .template-variable {
  display: inline-flex !important;
  align-items: center !important;
  background: #e5e7eb !important;
  color: #374151 !important;
  padding: 2px 6px !important;
  border-radius: 4px !important;
  font-family: 'Fira Code', 'Consolas', 'Monaco', monospace !important;
  font-size: 13px !important;
  font-weight: 500 !important;
  user-select: all !important;
  -webkit-user-select: all !important;
  cursor: default !important;
  border: 1px solid #d1d5db !important;
  white-space: nowrap !important;
  box-shadow: none !important;
  transition: background 0.2s !important;
}

.editor-content .template-variable:hover {
  background: #d1d5db !important;
}

/* 基础样式 */
.editor-content p {
  margin: 0.5em 0;
}

.editor-content h1,
.editor-content h2,
.editor-content h3 {
  margin: 0.5em 0;
  font-weight: 600;
}

.editor-content h1 {
  font-size: 2em;
}

.editor-content h2 {
  font-size: 1.5em;
}

.editor-content h3 {
  font-size: 1.25em;
}

.editor-content ul,
.editor-content ol {
  margin: 0.5em 0;
  padding-left: 2em;
}

.editor-content li {
  margin: 0.25em 0;
}

.editor-content blockquote {
  margin: 0.5em 0;
  padding-left: 1em;
  border-left: 4px solid #ccc;
  color: #666;
}

.editor-content a {
  color: #1890ff;
  text-decoration: underline;
  cursor: pointer;
}

.editor-content img {
  max-width: 100%;
  height: auto;
}

.editor-content table {
  border-collapse: collapse;
  width: 100%;
  margin: 0.5em 0;
}

.editor-content th,
.editor-content td {
  border: 1px solid #ccc;
  padding: 8px;
  text-align: left;
}

.editor-content th {
  background: #f5f5f5;
  font-weight: 600;
}

.editor-content pre {
  background: #f5f5f5;
  padding: 12px;
  border-radius: 4px;
  overflow-x: auto;
}

.editor-content code {
  font-family: 'Courier New', Courier, monospace;
  background: #f5f5f5;
  padding: 2px 4px;
  border-radius: 3px;
}

.editor-content pre code {
  background: transparent;
  padding: 0;
}
</style>