/**
 * v-highlight 代码高亮指令
 *
 * 为代码块提供语法高亮、行号显示和一键复制功能。
 * 基于 highlight.js 实现，支持多种编程语言的语法高亮。
 *
 * ## 主要功能
 *
 * - 语法高亮 - 使用 highlight.js 自动识别并高亮代码
 * - 行号显示 - 自动为每行代码添加行号
 * - 一键复制 - 提供复制按钮，点击即可复制代码（自动过滤行号）
 * - 性能优化 - 批量处理代码块，避免阻塞渲染
 * - 动态监听 - 使用 MutationObserver 监听新增代码块
 * - 防重复处理 - 自动标记已处理的代码块，避免重复处理
 *
 * ## 使用示例
 *
 * ```vue
 * <template>
 *   <!-- 基础用法 -->
 *   <div v-highlight v-html="codeContent"></div>
 *
 *   <!-- 配合 Markdown 渲染 -->
 *   <div v-highlight>
 *     <pre><code class="language-javascript">
 *       const hello = 'world';
 *       console.log(hello);
 *     </code></pre>
 *   </div>
 * </template>
 * ```
 *
 * ## 性能优化
 *
 * - 批量处理：每次处理 10 个代码块，避免长时间阻塞
 * - 延迟处理：使用 requestAnimationFrame 分批处理
 * - 重试机制：自动重试处理失败的代码块
 * - 智能监听：只在有新代码块时才触发处理
 *
 * @module directives/highlight
 * @author Art Design Pro Team
 */

import { App, Directive } from 'vue'
import type { LanguageFn } from 'highlight.js'

export type HighlightDirective = Directive<HTMLElement>

const processingBlocks = new WeakSet<HTMLElement>()

// web_bundle_mobile F3：只引入 highlight.js core + 白名单语言。
// `import('highlight.js')` 会带全量 ~190 种语言（963kB/307kB gzip 单块），
// 且无法被 tree-shake。改用 `highlight.js/lib/core` 并只注册实际会用到的
// 语言，未注册的语言降级为纯文本。
const IMPORTED_LANGUAGES = [
  ['bash', () => import('highlight.js/lib/languages/bash')],
  ['shell', () => import('highlight.js/lib/languages/shell')],
  ['javascript', () => import('highlight.js/lib/languages/javascript')],
  ['typescript', () => import('highlight.js/lib/languages/typescript')],
  ['go', () => import('highlight.js/lib/languages/go')],
  ['json', () => import('highlight.js/lib/languages/json')],
  ['yaml', () => import('highlight.js/lib/languages/yaml')],
  ['sql', () => import('highlight.js/lib/languages/sql')],
  ['xml', () => import('highlight.js/lib/languages/xml')],
  ['html', () => import('highlight.js/lib/languages/xml')],
  ['python', () => import('highlight.js/lib/languages/python')],
  ['css', () => import('highlight.js/lib/languages/css')]
] as const

const importHighlighter = () =>
  import('highlight.js/lib/core').then(async (module) => {
    const core = module.default
    for (const [name, loader] of IMPORTED_LANGUAGES) {
      try {
        const langModule = (await loader()) as { default?: LanguageFn }
        const lang = (langModule.default || langModule) as LanguageFn
        core.registerLanguage(name, lang)
      } catch {
        // 语言模块缺失时跳过，该语言降级为纯文本。
      }
    }
    return core
  })
let highlighterRequest: ReturnType<typeof importHighlighter> | null = null

const loadHighlighter = () => {
  highlighterRequest ??= importHighlighter().catch((error) => {
    highlighterRequest = null
    throw error
  })
  return highlighterRequest
}

async function highlightCode(block: HTMLElement) {
  const hljs = await loadHighlighter()
  hljs.highlightElement(block)
}

// 插入行号
function insertLineNumbers(block: HTMLElement) {
  const lines = block.innerHTML.split('\n')
  const numberedLines = lines
    .map((line, index) => {
      return `<span class="line-number">${index + 1}</span> ${line}`
    })
    .join('\n')
  block.innerHTML = numberedLines
}

// 添加复制按钮：调整 DOM 结构，将代码部分包裹在 .code-wrapper 内
function addCopyButton(block: HTMLElement) {
  const copyButton = document.createElement('i')
  copyButton.className = 'copy-button'
  copyButton.innerHTML =
    '<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24"><path fill="currentColor" d="M7 6V3a1 1 0 0 1 1-1h12a1 1 0 0 1 1 1v14a1 1 0 0 1-1 1h-3v3c0 .552-.45 1-1.007 1H4.007A1 1 0 0 1 3 21l.003-14c0-.552.45-1 1.006-1zM5.002 8L5 20h10V8zM9 6h8v10h2V4H9z"/></svg>'
  copyButton.onclick = () => {
    // 过滤掉行号，只复制代码内容
    const codeContent = block.innerText.replace(/^\d+\s+/gm, '')
    navigator.clipboard.writeText(codeContent).then(() => {
      ElMessage.success('复制成功')
    })
  }

  const preElement = block.parentElement
  if (preElement) {
    let codeWrapper: HTMLElement
    // 如果代码块还没有被包裹，则创建包裹容器
    if (!block.parentElement.classList.contains('code-wrapper')) {
      codeWrapper = document.createElement('div')
      codeWrapper.className = 'code-wrapper'
      preElement.replaceChild(codeWrapper, block)
      codeWrapper.appendChild(block)
    } else {
      codeWrapper = block.parentElement
    }
    // 将复制按钮添加到 pre 元素（而非 codeWrapper 内），这样它不会随滚动条滚动
    preElement.appendChild(copyButton)
  }
}

// 检查代码块是否已经被处理过
function isBlockProcessed(block: HTMLElement): boolean {
  return (
    block.hasAttribute('data-highlighted') ||
    !!block.querySelector('.line-number') ||
    !!block.parentElement?.querySelector('.copy-button')
  )
}

// 标记代码块为已处理
function markBlockAsProcessed(block: HTMLElement) {
  block.setAttribute('data-highlighted', 'true')
}

// 处理单个代码块
async function processBlock(block: HTMLElement) {
  if (isBlockProcessed(block) || processingBlocks.has(block)) {
    return
  }

  processingBlocks.add(block)
  try {
    await highlightCode(block)
    if (!block.isConnected || isBlockProcessed(block)) return
    insertLineNumbers(block)
    addCopyButton(block)
    markBlockAsProcessed(block)
  } catch (error) {
    console.warn('处理代码块时出错:', error)
  } finally {
    processingBlocks.delete(block)
  }
}

// 查找并处理所有代码块
function processAllCodeBlocks(el: HTMLElement) {
  const blocks = Array.from(el.querySelectorAll<HTMLElement>('pre code'))
  const unprocessedBlocks = blocks.filter((block) => !isBlockProcessed(block))

  if (unprocessedBlocks.length === 0) {
    return
  }

  if (unprocessedBlocks.length <= 10) {
    // 如果代码块数量少于等于10，直接处理所有代码块
    unprocessedBlocks.forEach((block) => void processBlock(block))
  } else {
    // 定义每次处理的代码块数
    const batchSize = 10
    let currentIndex = 0

    const processBatch = () => {
      const batch = unprocessedBlocks.slice(currentIndex, currentIndex + batchSize)

      batch.forEach((block) => {
        void processBlock(block)
      })

      // 更新索引并继续处理下一批
      currentIndex += batchSize
      if (currentIndex < unprocessedBlocks.length) {
        // 使用 requestAnimationFrame 确保下一帧再处理
        requestAnimationFrame(processBatch)
      }
    }

    // 开始处理第一批代码块
    processBatch()
  }
}

// 重试处理函数
function retryProcessing(el: HTMLElement, maxRetries: number = 3, delay: number = 200) {
  let retryCount = 0

  const tryProcess = () => {
    processAllCodeBlocks(el)

    // 检查是否还有未处理的代码块
    const remainingBlocks = Array.from(el.querySelectorAll<HTMLElement>('pre code')).filter(
      (block) => !isBlockProcessed(block)
    )

    if (remainingBlocks.length > 0 && retryCount < maxRetries) {
      retryCount++
      setTimeout(tryProcess, delay * retryCount) // 递增延迟
    }
  }

  tryProcess()
}

// 代码高亮、插入行号、复制按钮
const highlightDirective: HighlightDirective = {
  mounted(el: HTMLElement) {
    // 立即尝试处理一次
    processAllCodeBlocks(el)

    // 延迟处理，确保 v-html 内容已经渲染
    setTimeout(() => {
      retryProcessing(el)
    }, 100)

    // 使用 MutationObserver 监听 DOM 变化
    const observer = new MutationObserver((mutations) => {
      let hasNewCodeBlocks = false

      mutations.forEach((mutation) => {
        if (mutation.type === 'childList') {
          mutation.addedNodes.forEach((node) => {
            if (node.nodeType === Node.ELEMENT_NODE) {
              const element = node as HTMLElement
              // 检查新添加的节点是否包含代码块
              if (element.tagName === 'PRE' || element.querySelector('pre code')) {
                hasNewCodeBlocks = true
              }
            }
          })
        }
      })

      if (hasNewCodeBlocks) {
        // 延迟处理新添加的代码块
        setTimeout(() => {
          processAllCodeBlocks(el)
        }, 50)
      }
    })

    // 开始观察
    observer.observe(el, {
      childList: true,
      subtree: true
    })

    // 将 observer 存储到元素上，以便在 unmounted 时清理
    ;(el as any)._highlightObserver = observer
  },

  updated(el: HTMLElement) {
    // 当组件更新时，重新处理代码块
    setTimeout(() => {
      processAllCodeBlocks(el)
    }, 50)
  },

  unmounted(el: HTMLElement) {
    // 清理 MutationObserver
    const observer = (el as any)._highlightObserver
    if (observer) {
      observer.disconnect()
      delete (el as any)._highlightObserver
    }
  }
}

export function setupHighlightDirective(app: App) {
  app.directive('highlight', highlightDirective)
}
