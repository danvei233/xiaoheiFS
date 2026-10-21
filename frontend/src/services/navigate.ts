import type { Router } from "vue-router";

let routerPromise: Promise<Router> | null = null;

async function getRouter(): Promise<Router> {
  if (!routerPromise) {
    routerPromise = import("@/router").then((m) => m.default);
  }
  return routerPromise;
}

export async function navigateReplace(path: string) {
  const router = await getRouter();
  try {
    await router.replace(path);
  } catch {
    // ignore navigation duplication errors
  }
}

export async function navigatePush(path: string) {
  const router = await getRouter();
  try {
    await router.push(path);
  } catch {
    // ignore navigation duplication errors
  }
}
