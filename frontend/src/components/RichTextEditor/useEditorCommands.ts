import { ref, type Ref } from 'vue';

// 历史记录项
interface HistoryItem {
  html: string;
  selection?: SelectionRange;
}

interface SelectionRange {
  startOffset: number;
  endOffset: number;
  startContainerPath: number[];
  endContainerPath: number[];
}

// 编辑器命令封装
export function useEditorCommands(editorRef: Ref<HTMLElement | null>) {
  const history = ref<HistoryItem[]>([]);
  const historyIndex = ref(-1);
  const maxHistorySize = 50;

  // 保存当前状态到历史记录
  const saveToHistory = () => {
    if (!editorRef.value) return;

    const html = editorRef.value.innerHTML;
    const selection = saveSelection();

    // 如果当前不在历史记录末尾，删除后面的记录
    if (historyIndex.value < history.value.length - 1) {
      history.value = history.value.slice(0, historyIndex.value + 1);
    }

    // 避免重复保存相同内容
    if (history.value.length > 0 && history.value[historyIndex.value].html === html) {
      return;
    }

    history.value.push({ html, selection });
    if (history.value.length > maxHistorySize) {
      history.value.shift();
    } else {
      historyIndex.value++;
    }
  };

  // 保存选区
  const saveSelection = (): SelectionRange | undefined => {
    const selection = window.getSelection();
    if (!selection || selection.rangeCount === 0) return undefined;

    const range = selection.getRangeAt(0);
    const startPath = getNodePath(range.startContainer);
    const endPath = getNodePath(range.endContainer);

    return {
      startOffset: range.startOffset,
      endOffset: range.endOffset,
      startContainerPath: startPath,
      endContainerPath: endPath
    };
  };

  // 恢复选区
  const restoreSelection = (selectionRange: SelectionRange) => {
    if (!editorRef.value) return;

    const selection = window.getSelection();
    if (!selection) return;

    const startContainer = getNodeByPath(editorRef.value, selectionRange.startContainerPath);
    const endContainer = getNodeByPath(editorRef.value, selectionRange.endContainerPath);

    if (!startContainer || !endContainer) return;

    const range = document.createRange();
    range.setStart(startContainer, selectionRange.startOffset);
    range.setEnd(endContainer, selectionRange.endOffset);

    selection.removeAllRanges();
    selection.addRange(range);
  };

  // 获取节点路径
  const getNodePath = (node: Node): number[] => {
    const path: number[] = [];
    let current: Node | null = node;

    while (current && current !== editorRef.value) {
      const parent = current.parentNode;
      if (!parent) break;

      const siblings = Array.from(parent.childNodes);
      const index = siblings.indexOf(current);
      path.unshift(index);

      current = parent;
    }

    return path;
  };

  // 根据路径获取节点
  const getNodeByPath = (root: HTMLElement, path: number[]): Node | null => {
    let current: Node = root;

    for (const index of path) {
      if (current.childNodes[index]) {
        current = current.childNodes[index];
      } else {
        return null;
      }
    }

    return current;
  };

  // 执行命令
  const executeCommand = (command: string, value: any = null) => {
    document.execCommand(command, false, value);
    editorRef.value?.focus();
  };

  // 格式化内联文本
  const formatInline = (format: string) => {
    saveToHistory();
    executeCommand(format);
  };

  // 格式化块级元素
  const formatBlock = (tag: string) => {
    saveToHistory();
    executeCommand('formatBlock', tag);
  };

  // 插入列表
  const insertList = (type: 'ordered' | 'unordered') => {
    saveToHistory();
    executeCommand(type === 'ordered' ? 'insertOrderedList' : 'insertUnorderedList');
  };

  // 改变对齐方式
  const changeAlignment = (align: 'left' | 'center' | 'right' | 'justify') => {
    saveToHistory();
    executeCommand(`justify${align.charAt(0).toUpperCase() + align.slice(1)}`);
  };

  // 缩进
  const indent = () => {
    saveToHistory();
    executeCommand('indent');
  };

  // 减少缩进
  const outdent = () => {
    saveToHistory();
    executeCommand('outdent');
  };

  // 插入链接
  const insertLink = (url: string, text?: string) => {
    saveToHistory();
    if (text) {
      executeCommand('insertHTML', `<a href="${url}">${text}</a>`);
    } else {
      executeCommand('createLink', url);
    }
  };

  // 插入图片
  const insertImage = (src: string, alt: string = '') => {
    saveToHistory();
    executeCommand('insertImage', src);
    if (alt) {
      const img = editorRef.value?.querySelector('img:last-child') as HTMLImageElement;
      if (img) img.alt = alt;
    }
  };

  // 插入表格
  const insertTable = (rows: number, cols: number) => {
    saveToHistory();
    let tableHtml = '<table style="border-collapse: collapse; width: 100%;">';

    for (let i = 0; i < rows; i++) {
      tableHtml += '<tr>';
      for (let j = 0; j < cols; j++) {
        const tag = i === 0 ? 'th' : 'td';
        const style = 'border: 1px solid #ccc; padding: 8px; text-align: left;';
        tableHtml += `<${tag} style="${style}">${i === 0 ? '' : ''}</${tag}>`;
      }
      tableHtml += '</tr>';
    }

    tableHtml += '</table><p><br></p>';
    executeCommand('insertHTML', tableHtml);
  };

  // 清除格式
  const removeFormat = () => {
    saveToHistory();
    executeCommand('removeFormat');
  };

  // 撤销
  const undo = () => {
    if (historyIndex.value > 0) {
      historyIndex.value--;
      const item = history.value[historyIndex.value];
      if (editorRef.value) {
        editorRef.value.innerHTML = item.html;
        if (item.selection) {
          restoreSelection(item.selection);
        }
      }
    }
  };

  // 重做
  const redo = () => {
    if (historyIndex.value < history.value.length - 1) {
      historyIndex.value++;
      const item = history.value[historyIndex.value];
      if (editorRef.value) {
        editorRef.value.innerHTML = item.html;
        if (item.selection) {
          restoreSelection(item.selection);
        }
      }
    }
  };

  // 初始化历史记录
  const initHistory = () => {
    if (!editorRef.value) return;
    saveToHistory();
  };

  // 检查命令状态
  const queryCommandState = (command: string): boolean => {
    return document.queryCommandState(command);
  };

  return {
    // 命令方法
    formatInline,
    formatBlock,
    insertList,
    changeAlignment,
    indent,
    outdent,
    insertLink,
    insertImage,
    insertTable,
    removeFormat,
    undo,
    redo,
    queryCommandState,

    // 历史记录
    initHistory,
    saveToHistory,

    // 状态
    canUndo: () => historyIndex.value > 0,
    canRedo: () => historyIndex.value < history.value.length - 1
  };
}