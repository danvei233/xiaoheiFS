const REDIRECT_BASE_URL = 'https://xiaohei.invalid'

function normalizeAdminPath(adminPath: string): string {
  return String(adminPath || '')
    .trim()
    .replace(/^\/+|\/+$/g, '')
}

function containsControlCharacter(value: string): boolean {
  return Array.from(value).some((character) => {
    const code = character.charCodeAt(0)
    return code <= 31 || code === 127
  })
}

export function getAdminDashboardPath(adminPath: string): string {
  const normalized = normalizeAdminPath(adminPath) || 'admin'
  return `/${normalized}/dashboard/console`
}

/** Resolve an untrusted redirect to a path inside the current admin realm. */
export function resolveAdminRedirect(redirect: unknown, adminPath: string): string {
  const fallback = getAdminDashboardPath(adminPath)
  const normalizedAdminPath = normalizeAdminPath(adminPath)

  if (
    !normalizedAdminPath ||
    typeof redirect !== 'string' ||
    !redirect.startsWith('/') ||
    redirect.startsWith('//') ||
    redirect.includes('\\') ||
    containsControlCharacter(redirect)
  ) {
    return fallback
  }

  try {
    const target = new URL(redirect, REDIRECT_BASE_URL)
    const prefix = `/${normalizedAdminPath}`
    const decodedPath = decodeURIComponent(target.pathname)
    const isInsideAdmin = (path: string) => path === prefix || path.startsWith(`${prefix}/`)

    if (
      target.origin !== REDIRECT_BASE_URL ||
      !isInsideAdmin(target.pathname) ||
      !isInsideAdmin(decodedPath)
    ) {
      return fallback
    }

    return `${target.pathname}${target.search}${target.hash}`
  } catch {
    return fallback
  }
}

/**
 * Resolve an untrusted post-login redirect for the user realm.
 *
 * 管理域已有 `resolveAdminRedirect` 白名单校验；用户登录页此前把
 * `route.query.redirect` 裸交给 `router.replace`。vue-router 4 的 base 解析
 * 会把任意串拼成同源路径（不会真正跳外域），但为防将来改成
 * `window.location.href = redirect` / 升级路由时变成开放重定向，这里同样
 * 收敛到白名单（web_auth_surface F4）。只允许同源相对路径且限定在
 * console / 公共页，不满足一律回退 `/console`。
 */
export function resolveUserRedirect(redirect: unknown): string {
  const fallback = '/console'
  if (
    typeof redirect !== 'string' ||
    !redirect.startsWith('/') ||
    redirect.startsWith('//') ||
    redirect.includes('\\') ||
    containsControlCharacter(redirect)
  ) {
    return fallback
  }
  try {
    const target = new URL(redirect, REDIRECT_BASE_URL)
    const decodedPath = decodeURIComponent(target.pathname)
    const allowed = (path: string): boolean =>
      path === '/console' ||
      path.startsWith('/console/') ||
      path === '/login' ||
      path === '/register' ||
      path === '/cart' ||
      path === '/buy'
    if (target.origin !== REDIRECT_BASE_URL || !allowed(target.pathname) || !allowed(decodedPath)) {
      return fallback
    }
    return `${target.pathname}${target.search}${target.hash}`
  } catch {
    return fallback
  }
}

/** Check a button permission against the current identity's permission list. */
export function hasButtonPermission(
  buttons: readonly string[] | null | undefined,
  permission: string
): boolean {
  if (!permission || !Array.isArray(buttons)) return false
  return buttons.includes('*') || buttons.includes(permission)
}
