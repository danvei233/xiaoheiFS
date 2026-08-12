/**
 * 路由工具函数
 *
 * 提供路由相关的工具函数
 *
 * @module utils/router
 */
import { RouteLocationNormalized, RouteRecordRaw } from 'vue-router'
import AppConfig from '@/config'
import NProgress from 'nprogress'
import 'nprogress/nprogress.css'
import i18n, { $t } from '@/locales'
import { useSiteStore } from '@/stores/site'

/** 扩展的路由配置类型 */
export type AppRouteRecordRaw = RouteRecordRaw & {
  hidden?: boolean
}

/** 顶部进度条配置 */
export const configureNProgress = () => {
  NProgress.configure({
    easing: 'ease',
    speed: 600,
    showSpinner: false,
    parent: 'body'
  })
}

/** 当前页标题（不含品牌后缀），供站点品牌加载完成后重拼 document.title */
let currentPageTitle = ''
/** 单一可清理的标题写入 timer，避免快速导航时多个 timer 竞态残留（H7） */
let titleTimer: ReturnType<typeof setTimeout> | null = null

/**
 * 拼接"页名 - 品牌"。品牌本身可能含字面 ` - `（如"小黑云 - 高性能云"）。
 * 这里总是整段重写（不迭代替换），重复调用幂等，不会像 m065 R-5 的
 * MutationObserver 那样逐次把品牌再拼一遍导致无限循环。
 */
export function composeDocumentTitle(pageTitle: string, brand: string): string {
  return pageTitle ? `${pageTitle} - ${brand}` : brand
}

/** 站点品牌：后端设置驱动的 site.siteName，回退到构建期 AppConfig.systemInfo.name */
function resolveBrandName(): string {
  try {
    const name = useSiteStore().siteName?.trim()
    if (name) return name
  } catch {
    // pinia 尚未就绪时（守卫早期）回退
  }
  return AppConfig.systemInfo.name
}

/** 把"页名 - 品牌"写进 document.title。品牌变化时由 App.vue 调 refreshDocumentTitle。 */
function applyDocumentTitle(): void {
  document.title = composeDocumentTitle(currentPageTitle, resolveBrandName())
}

/**
 * 站点品牌加载完成后重新拼一次标题。不要在包含字面 ` - ` 的品牌上反复
 * 拼接（m065 R-5：MutationObserver 逐次替换会导致无限循环），这里每次
 * 都用 currentPageTitle + 当前品牌整段重写，天然幂等。
 */
export function refreshDocumentTitle(): void {
  applyDocumentTitle()
}

/**
 * 设置页面标题，根据路由元信息和系统信息拼接标题
 * @param to 当前路由对象
 */
export const setPageTitle = (to: RouteLocationNormalized): void => {
  // 防御：Vue Router 通常保证 meta 存在，但守卫早期/测试桩里可能缺。
  const { title } = to.meta ?? {}
  if (title) {
    currentPageTitle = formatMenuTitle(String(title))
  } else {
    // H7：无 meta.title 时立即回退品牌，不能只清变量不写 document.title——
    // 否则上一个页面标题会一直残留。
    currentPageTitle = ''
    if (titleTimer) clearTimeout(titleTimer)
    titleTimer = null
    applyDocumentTitle()
    return
  }
  // 延后一拍，等路由切换完成、DOM 更新后再写标题。用**同一个**可清理的
  // timer：快速连续导航时先清掉上一次，避免旧页标题晚到覆盖新页。
  if (titleTimer) clearTimeout(titleTimer)
  titleTimer = setTimeout(() => {
    applyDocumentTitle()
    titleTimer = null
  }, 150)
}

/**
 * 格式化菜单标题
 * @param title 菜单标题，可以是 i18n 的 key，也可以是字符串
 * @returns 格式化后的菜单标题
 */
export const formatMenuTitle = (title: string): string => {
  if (title) {
    if (title.startsWith('menus.')) {
      // 使用 te() 方法检查翻译键值是否存在，避免控制台警告
      if (i18n.global.te(title)) {
        return $t(title)
      } else {
        // 如果翻译不存在，返回键值的最后部分作为fallback
        return title.split('.').pop() || title
      }
    }
    return title
  }
  return ''
}
