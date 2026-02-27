import { http } from "./http";

let cachedAdminPath: string | null = null;
let isValidated: boolean = false; // 标记是否已验证成功

/**
 * 从 localStorage 读取缓存的管理端路径
 */
function loadAdminPathFromStorage(): string | null {
  try {
    return localStorage.getItem("admin_path_cache");
  } catch {
    return null;
  }
}

/**
 * 保存管理端路径到 localStorage
 */
function saveAdminPathToStorage(path: string): void {
  try {
    localStorage.setItem("admin_path_cache", path);
    localStorage.setItem("admin_path_validated", "true");
  } catch {
    // ignore
  }
}

/**
 * 检查是否已验证过
 */
function isPathValidated(): boolean {
  try {
    return localStorage.getItem("admin_path_validated") === "true";
  } catch {
    return false;
  }
}

/**
 * 检查给定路径是否是管理端路径
 */
export async function checkAdminPath(path: string): Promise<{ isAdmin: boolean; adminPath: string }> {
  // 先检查缓存：如果已经验证过且路径匹配，直接返回
  const cached = loadAdminPathFromStorage();
  if (isPathValidated() && cached && path === cached) {
    isValidated = true;
    cachedAdminPath = cached;
    return {
      isAdmin: true,
      adminPath: cached
    };
  }
  
  // 如果当前会话已验证过，直接返回
  if (isValidated && cachedAdminPath && path === cachedAdminPath) {
    return {
      isAdmin: true,
      adminPath: cachedAdminPath
    };
  }
  
  try {
    const res = await http.post<{ is_admin: boolean; admin_path: string }>("/api/v1/check-admin-path", { path });
    if (res.data) {
      cachedAdminPath = res.data.admin_path;
      
      // 如果验证成功，保存到 localStorage 并标记已验证
      if (res.data.is_admin) {
        saveAdminPathToStorage(res.data.admin_path);
        isValidated = true;
      }
      
      return {
        isAdmin: res.data.is_admin,
        adminPath: res.data.admin_path
      };
    }
  } catch (error) {
    console.error("Failed to check admin path:", error);
  }
  
  return { isAdmin: false, adminPath: cached || "admin" };
}

/**
 * 获取管理端路径（带缓存）
 */
export function getCachedAdminPath(): string {
  if (!cachedAdminPath) {
    cachedAdminPath = loadAdminPathFromStorage();
  }
  return cachedAdminPath || "admin";
}

/**
 * 清除缓存的管理端路径
 */
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

/**
 * 获取管理端路径（从后端获取）
 */
export async function fetchAdminPath(): Promise<string> {
  // 先尝试从缓存读取
  const cached = getCachedAdminPath();
  if (cached && cached !== "admin") {
    return cached;
  }
  
  try {
    // 尝试检查一个随机路径来获取真实的管理端路径
    const res = await http.post<{ is_admin: boolean; admin_path: string }>("/api/v1/check-admin-path", { path: "_probe_" });
    if (res.data && res.data.admin_path) {
      cachedAdminPath = res.data.admin_path;
      saveAdminPathToStorage(res.data.admin_path);
      return res.data.admin_path;
    }
  } catch (error) {
    console.error("Failed to fetch admin path:", error);
  }
  
  return getCachedAdminPath();
}

/**
 * 构建管理端URL
 */
export function buildAdminUrl(subPath: string = ""): string {
  const adminPath = getCachedAdminPath();
  const cleanSubPath = subPath.replace(/^\/+/, "");
  return cleanSubPath ? `/${adminPath}/${cleanSubPath}` : `/${adminPath}`;
}

/**
 * 跳转到管理端登录页
 */
export async function navigateToAdminLogin(router: any, redirect?: string): Promise<void> {
  const adminPath = await fetchAdminPath();
  const query = redirect ? { redirect } : {};
  await router.push({ path: `/${adminPath}/login`, query });
}

