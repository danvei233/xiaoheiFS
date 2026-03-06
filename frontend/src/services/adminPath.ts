import { http } from "./http";

let cachedAdminPath: string | null = null;
let isValidated = false;

function normalizePath(path: string): string {
  return String(path || "").trim().replace(/^\/+|\/+$/g, "");
}

function loadAdminPathFromStorage(): string | null {
  try {
    const path = normalizePath(localStorage.getItem("admin_path_cache") || "");
    return path || null;
  } catch {
    return null;
  }
}

function saveAdminPathToStorage(path: string): void {
  const normalized = normalizePath(path);
  if (!normalized) return;
  try {
    localStorage.setItem("admin_path_cache", normalized);
    localStorage.setItem("admin_path_validated", "true");
  } catch {
    // ignore
  }
}

function isPathValidated(): boolean {
  try {
    return localStorage.getItem("admin_path_validated") === "true";
  } catch {
    return false;
  }
}

export async function checkAdminPath(path: string): Promise<{ isAdmin: boolean; adminPath: string }> {
  const normalized = normalizePath(path);
  const cached = loadAdminPathFromStorage();

  if (isPathValidated() && cached && normalized === cached) {
    isValidated = true;
    cachedAdminPath = cached;
    return { isAdmin: true, adminPath: cached };
  }

  if (isValidated && cachedAdminPath && normalized === cachedAdminPath) {
    return { isAdmin: true, adminPath: cachedAdminPath };
  }

  if (!normalized) {
    return { isAdmin: false, adminPath: cached || "admin" };
  }

  try {
    const res = await http.post<{ is_admin: boolean }>("/api/v1/check-admin-path", { path: normalized });
    if (res.data?.is_admin) {
      cachedAdminPath = normalized;
      saveAdminPathToStorage(normalized);
      isValidated = true;
      return { isAdmin: true, adminPath: normalized };
    }
  } catch (error) {
    console.error("Failed to check admin path:", error);
  }

  return { isAdmin: false, adminPath: cached || "admin" };
}

export function getCachedAdminPath(): string {
  if (!cachedAdminPath) {
    cachedAdminPath = loadAdminPathFromStorage();
  }
  return cachedAdminPath || "admin";
}

export function clearAdminPathCache(): void {
  cachedAdminPath = null;
  isValidated = false;
  try {
    localStorage.removeItem("admin_path_cache");
    localStorage.removeItem("admin_path_validated");
  } catch {
    // ignore
  }
}

export async function fetchAdminPath(): Promise<string> {
  return getCachedAdminPath();
}

export function buildAdminUrl(subPath: string = ""): string {
  const adminPath = getCachedAdminPath();
  const cleanSubPath = subPath.replace(/^\/+/, "");
  return cleanSubPath ? `/${adminPath}/${cleanSubPath}` : `/${adminPath}`;
}

export async function navigateToAdminLogin(router: any, redirect?: string): Promise<void> {
  const adminPath = await fetchAdminPath();
  const query = redirect ? { redirect } : {};
  await router.push({ path: `/${adminPath}/login`, query });
}
