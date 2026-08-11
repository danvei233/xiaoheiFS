/**
 * 域（realm）解析
 *
 * 本应用一个 SPA 承载三个域，鉴权体系与菜单来源各自独立：
 *
 * | realm   | 路径              | token                       | 菜单来源                     |
 * |---------|-------------------|-----------------------------|------------------------------|
 * | public  | /、/products …    | 无（登录前购物车也在这里）  | 无侧边栏                     |
 * | console | /console/*        | stores/auth（user_token）   | consoleMenus（静态）         |
 * | admin   | /<admin_path>/*   | stores/adminAuth            | asyncRoutes（按权限裁剪）    |
 *
 * @module router/realm
 */
import type { RouteLocationNormalized } from 'vue-router'
import { getCachedAdminPath } from '@/services/adminPath'

export type Realm = 'public' | 'console' | 'admin'

/** 控制台路径前缀 */
export const CONSOLE_PREFIX = '/console'

/**
 * 判断目标路由属于哪个域。
 *
 * 优先看 meta.realm（静态路由显式声明），其次看路径前缀。
 * 管理后台路径是运行时决定的，所以要拿当前缓存的 admin_path 比对。
 */
export function resolveRealm(to: RouteLocationNormalized): Realm {
  const declared = to.meta?.realm as Realm | undefined
  if (declared) return declared

  const path = to.path
  if (path === CONSOLE_PREFIX || path.startsWith(`${CONSOLE_PREFIX}/`)) {
    return 'console'
  }

  const adminPath = getCachedAdminPath()
  if (adminPath && (path === `/${adminPath}` || path.startsWith(`/${adminPath}/`))) {
    return 'admin'
  }

  return 'public'
}

/**
 * 给一组菜单路由的 path 统一加上管理后台前缀。
 *
 * ADP 的 RouteTransformer 要求一级菜单的 path 是绝对路径，而管理后台挂载点
 * 直到运行时拿到 admin_path 才确定，因此在注册前做一次前缀改写。
 *
 * `meta.activePath` 也必须一起改写：隐藏的详情页（探针详情、工单详情这类
 * `isHide: true` 的页面）靠它指回父级菜单来保持高亮，只改 path 会让侧边栏
 * 在动态后台路径下失去高亮。
 */
export function prefixRoutePaths<
  T extends { path?: string; meta?: Record<string, any>; children?: T[] }
>(routes: T[], prefix: string): T[] {
  const clean = `/${prefix.replace(/^\/+|\/+$/g, '')}`
  const withPrefix = (p: string) => (p.startsWith('/') ? `${clean}${p}` : p)

  /** 把相对子路径按父级目录拼成完整路径；绝对路径原样返回 */
  const resolvePath = (p: string, parentPath: string): string => {
    if (p.startsWith('/')) return p
    return parentPath ? `${parentPath}/${p}` : `/${p}`
  }

  const walk = (list: T[], parentPath = ''): T[] =>
    list.map((route) => {
      const next = { ...route } as T

      if (typeof route.path === 'string') {
        // 子级相对路径（如 'console'）按父目录拼成完整路径再统一加前缀。
        // 否则写进菜单树的子路径是相对路径，getFirstMenuPath 推导首页路径
        // 会得到 /console 这类错误结果，侧边栏跳转也解析不到正确地址。
        next.path = withPrefix(resolvePath(route.path, parentPath))
      }

      if (typeof route.meta?.activePath === 'string') {
        next.meta = {
          ...route.meta,
          activePath: withPrefix(resolvePath(route.meta.activePath, parentPath))
        }
      }

      if (route.children?.length) {
        next.children = walk(route.children, resolvePath(route.path, parentPath))
      }

      return next
    })

  return walk(routes)
}
