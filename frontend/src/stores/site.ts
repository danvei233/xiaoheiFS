import { defineStore } from "pinia";
import { getCmsBlocks, getCmsPosts, getSiteSettings } from "@/services/user";

interface SiteState {
  siteName: string;
  logoUrl: string;
  faviconUrl: string;
  language: string;
  maintenanceMode: boolean;
  maintenanceMessage: string;
  blocks: Record<string, any[]>;
  posts: Record<string, any[]>;
  settings: Record<string, any>;
}

export interface SiteNavItem {
  label: string;
  url: string;
  target?: "_self" | "_blank";
  lang?: string;
  enabled?: boolean;
}

export const useSiteStore = defineStore("site", {
  state: (): SiteState => ({
    siteName: "小黑云控制台",
    logoUrl: "",
    faviconUrl: "",
    language: "zh-CN",
    maintenanceMode: false,
    maintenanceMessage: "",
    blocks: {},
    posts: {},
    settings: {}
  }),

  getters: {
    currentLang(): string {
      return this.language;
    },

    headerNavItems(): SiteNavItem[] {
      const raw = this.settings?.site_nav_items;
      const fallback: SiteNavItem[] =
        this.language === "en-US"
          ? [
              { label: "Products", url: "/products", target: "_self" },
              { label: "Docs", url: "/docs", target: "_self" },
              { label: "Help", url: "/help", target: "_self" }
            ]
          : [
              { label: "产品", url: "/products", target: "_self" },
              { label: "文档", url: "/docs", target: "_self" },
              { label: "帮助", url: "/help", target: "_self" }
            ];
      if (!raw) return fallback;
      try {
        const parsed = typeof raw === "string" ? JSON.parse(raw) : raw;
        const arr = Array.isArray(parsed) ? parsed : [];
        const normalized: SiteNavItem[] = arr
          .map((x: any) => ({
            label: String(x?.label || "").trim(),
            url: String(x?.url || "").trim(),
            target: (String(x?.target || "_self") as SiteNavItem["target"]) === "_blank" ? "_blank" : "_self",
            lang: x?.lang ? String(x.lang).trim() : undefined,
            enabled: x?.enabled === false ? false : true
          }))
          .filter((x) => (x.label || x.url) && x.enabled !== false);
        const filtered = normalized.filter((x) => !x.lang || x.lang === this.language);
        return filtered.length > 0 ? filtered : fallback;
      } catch {
        return fallback;
      }
    }
  },

  actions: {
    setLang(lang?: string) {
      this.language = lang || navigator.language || "zh-CN";
    },

    async fetchSettings() {
      try {
        const res = await getSiteSettings();
        const items = (res.data as any)?.items || [];
        this.settings = {};
        items.forEach((item: any) => {
          this.settings[item.key] = item.value;
        });
        const resolveValue = (key: string, legacyKey?: string) =>
          this.settings[key] || (legacyKey ? this.settings[legacyKey] : "");
        this.siteName = resolveValue("site_name") || "小黑云控制台";
        this.logoUrl = resolveValue("logo_url", "site_logo");
        this.faviconUrl = resolveValue("favicon_url", "site_favicon");
        const maintenanceRaw = resolveValue("maintenance_mode", "site_maintenance_mode");
        this.maintenanceMode = maintenanceRaw === "true" || maintenanceRaw === true;
        this.maintenanceMessage =
          resolveValue("maintenance_message", "site_maintenance_message") ||
          "系统维护中，请稍后再试";
      } catch (error) {
        console.error("Failed to fetch site settings:", error);
      }
    },

    async fetchBlocks(page?: string) {
      try {
        const res = await getCmsBlocks({ page: page || "home", lang: this.language });
        this.blocks[page || "home"] = res.data?.items || [];
      } catch (error) {
        console.error("Failed to fetch CMS blocks:", error);
      }
    },

    async fetchPosts(categoryKey?: string) {
      try {
        const res = await getCmsPosts({ category_key: categoryKey, lang: this.language, limit: 100, offset: 0 });
        this.posts[categoryKey || "all"] = res.data?.items || [];
      } catch (error) {
        console.error("Failed to fetch CMS posts:", error);
      }
    }
  }
});




