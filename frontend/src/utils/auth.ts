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

/** Check a button permission against the current identity's permission list. */
export function hasButtonPermission(
  buttons: readonly string[] | null | undefined,
  permission: string
): boolean {
  if (!permission || !Array.isArray(buttons)) return false
  return buttons.includes('*') || buttons.includes(permission)
}
