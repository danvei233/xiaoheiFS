import request from '@/utils/http'

const DEFAULT_ADMIN_PATH = 'admin'
const ADMIN_PATH_CACHE_KEY = 'admin_path_cache'
const ADMIN_PATH_VALIDATED_KEY = 'admin_path_validated'

let inMemoryValidatedPath = ''
let currentPathCheckKey = ''
let currentPathCheckPromise: Promise<AdminPathState> | null = null

export interface AdminPathState {
  status: 'standalone' | 'validated' | 'invalid'
  currentPath: string | null
  cachedPath: string | null
  adminPath: string
}

interface AdminPathCheckResponse {
  is_admin?: boolean
}

export function normalizeAdminPath(path: string | null | undefined): string {
  return String(path || '')
    .trim()
    .replace(/^\/+|\/+$/g, '')
}

function loadCachedAdminPath(): string | null {
  try {
    const cachedPath = normalizeAdminPath(localStorage.getItem(ADMIN_PATH_CACHE_KEY))
    return cachedPath || null
  } catch {
    return null
  }
}

export function getCachedAdminPath(): string {
  return loadCachedAdminPath() || DEFAULT_ADMIN_PATH
}

export function cacheAdminPath(path: string): string {
  const normalizedPath = normalizeAdminPath(path) || DEFAULT_ADMIN_PATH

  try {
    localStorage.setItem(ADMIN_PATH_CACHE_KEY, normalizedPath)
    localStorage.setItem(ADMIN_PATH_VALIDATED_KEY, 'true')
  } catch {
    // ignore storage failures
  }

  inMemoryValidatedPath = normalizedPath
  return normalizedPath
}

export function clearAdminPathCache(): void {
  try {
    localStorage.removeItem(ADMIN_PATH_CACHE_KEY)
    localStorage.removeItem(ADMIN_PATH_VALIDATED_KEY)
  } catch {
    // ignore storage failures
  }

  inMemoryValidatedPath = ''
}

export function getCurrentAdminPath(): string | null {
  if (typeof window === 'undefined') {
    return null
  }

  const segments = window.location.pathname.split('/').filter(Boolean)
  const currentPath = normalizeAdminPath(segments[0] || '')
  return currentPath || null
}

export async function checkAdminPath(
  path: string
): Promise<{ isAdmin: boolean; adminPath: string }> {
  const normalizedPath = normalizeAdminPath(path)
  const cachedPath = loadCachedAdminPath()

  if (!normalizedPath) {
    return {
      isAdmin: false,
      adminPath: cachedPath || DEFAULT_ADMIN_PATH
    }
  }

  if (inMemoryValidatedPath && inMemoryValidatedPath === normalizedPath) {
    return {
      isAdmin: true,
      adminPath: normalizedPath
    }
  }

  try {
    const response = await request.post<AdminPathCheckResponse>({
      url: '/api/v1/check-admin-path',
      data: { path: normalizedPath },
      showErrorMessage: false
    })

    if (response?.is_admin) {
      cacheAdminPath(normalizedPath)
      return {
        isAdmin: true,
        adminPath: normalizedPath
      }
    }
  } catch {
    return {
      isAdmin: false,
      adminPath: cachedPath || DEFAULT_ADMIN_PATH
    }
  }

  if (cachedPath && cachedPath === normalizedPath) {
    clearAdminPathCache()
  }

  return {
    isAdmin: false,
    adminPath: loadCachedAdminPath() || DEFAULT_ADMIN_PATH
  }
}

export async function ensureCurrentAdminPath(): Promise<AdminPathState> {
  const currentPath = getCurrentAdminPath()
  const cachedPath = loadCachedAdminPath()

  if (!currentPath) {
    return {
      status: 'standalone',
      currentPath: null,
      cachedPath,
      adminPath: cachedPath || DEFAULT_ADMIN_PATH
    }
  }

  if (currentPathCheckPromise && currentPathCheckKey === currentPath) {
    return currentPathCheckPromise
  }

  currentPathCheckKey = currentPath
  currentPathCheckPromise = checkAdminPath(currentPath).then((result) => ({
    status: result.isAdmin ? 'validated' : 'invalid',
    currentPath,
    cachedPath: loadCachedAdminPath(),
    adminPath: result.adminPath
  }))

  return currentPathCheckPromise
}

export async function resolveLoginAdminPath(): Promise<string> {
  const currentPathState = await ensureCurrentAdminPath()

  if (currentPathState.status === 'validated' && currentPathState.currentPath) {
    return currentPathState.currentPath
  }

  return currentPathState.adminPath || DEFAULT_ADMIN_PATH
}

export function buildAdminHashUrl(adminPath: string, routePath: string = ''): string {
  const normalizedAdminPath = normalizeAdminPath(adminPath) || DEFAULT_ADMIN_PATH
  const normalizedRoutePath = String(routePath || '')
    .trim()
    .replace(/^#/, '')

  if (!normalizedRoutePath || normalizedRoutePath === '/') {
    return `/${normalizedAdminPath}/`
  }

  const withLeadingSlash = normalizedRoutePath.startsWith('/')
    ? normalizedRoutePath
    : `/${normalizedRoutePath}`

  return `/${normalizedAdminPath}/#${withLeadingSlash}`
}
