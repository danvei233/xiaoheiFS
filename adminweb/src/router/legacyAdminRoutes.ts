import type { AppRouteRecordRaw } from '@/utils/router'

const legacyAdminRedirectMap = [
  ['/console', '/dashboard/console'],
  ['/revenue-analytics', '/dashboard/revenue-analytics'],
  ['/orders', '/order/review'],
  ['/users', '/system/user'],
  ['/user-tiers', '/system/user-tiers'],
  ['/coupons', '/marketing/coupons'],
  ['/admins', '/system/admins'],
  ['/permission-groups', '/system/permission-groups'],
  ['/profile', '/system/user-center']
] as const

function createLegacyRouteName(path: string): string {
  return `LegacyAdmin${path
    .split('/')
    .filter(Boolean)
    .map((part) =>
      part
        .split('-')
        .map((segment) => segment.charAt(0).toUpperCase() + segment.slice(1))
        .join('')
    )
    .join('')}Redirect`
}

export const legacyAdminRedirectRoutes: AppRouteRecordRaw[] = legacyAdminRedirectMap.map(
  ([path, target]) => ({
    path,
    name: createLegacyRouteName(path),
    redirect: (to) => ({
      path: target,
      query: to.query,
      hash: to.hash
    }),
    meta: { title: 'Legacy Admin Redirect', isHideTab: true }
  })
)
