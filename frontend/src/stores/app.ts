import { defineStore } from "pinia";
import { watch } from "vue";
import { getThemeAppearance, isThemeMode, isThemePreset, type ThemeMode, type ThemePreset } from "@/theme";

const API_KEY_STORAGE = "admin_api_key";
const THEME_MODE_STORAGE = "console_theme_mode";
const THEME_PRESET_STORAGE = "console_theme_preset";
const THEME_ROUNDED_STORAGE = "console_theme_rounded";
const THEME_SURFACE_STORAGE = "console_theme_transparent";
const THEME_FONT_STORAGE = "console_font_style";

export type FontStyle = "mixed" | "mono";
/** One-time marker: executive becomes the default preset. */
const THEME_DEFAULT_MIGRATION = "console_theme_default_v3";

function readPreference(key: string): string | null {
  try { return localStorage.getItem(key); } catch { return null; }
}
function savePreference(key: string, value: string) {
  try { localStorage.setItem(key, value); } catch { /* Appearance remains usable without browser storage. */ }
}
function readFlag(key: string, fallback: boolean) {
  const value = readPreference(key);
  return value === null ? fallback : value === "1";
}
function readFontStyle(): FontStyle {
  return readPreference(THEME_FONT_STORAGE) === "mono" ? "mono" : "mixed";
}
function systemDark() {
  return typeof window !== "undefined" && typeof window.matchMedia === "function"
    ? window.matchMedia("(prefers-color-scheme: dark)").matches : false;
}
const themeSessions = new WeakMap<object, { users: number; dispose: () => void }>();

export const useAppStore = defineStore("app", {
  state: () => {
    const mode = readPreference(THEME_MODE_STORAGE);
    const preset = readPreference(THEME_PRESET_STORAGE);
    return {
      adminApiKey: readPreference(API_KEY_STORAGE) || "",
      consoleThemeMode: (isThemeMode(mode) ? mode : "auto") as ThemeMode,
      themePreset: (isThemePreset(preset) ? preset : "executive") as ThemePreset,
      roundedCorners: readFlag(THEME_ROUNDED_STORAGE, true),
      transparentSurface: readFlag(THEME_SURFACE_STORAGE, false),
      fontStyle: readFontStyle(),
      systemPrefersDark: systemDark()
    };
  },
  getters: {
    isDarkMode: (state) => state.consoleThemeMode === "dark" || (state.consoleThemeMode === "auto" && state.systemPrefersDark),
    themeTokens(): ReturnType<typeof getThemeAppearance>["token"] {
      return getThemeAppearance(this.themePreset, this.isDarkMode, {
        rounded: this.roundedCorners, transparent: this.transparentSurface
      }).token;
    }
  },
  actions: {
    setAdminApiKey(key: string) {
      this.adminApiKey = key;
      if (key) localStorage.setItem(API_KEY_STORAGE, key);
      else localStorage.removeItem(API_KEY_STORAGE);
    },
    setConsoleThemeMode(mode: ThemeMode) {
      if (!isThemeMode(mode)) return;
      this.systemPrefersDark = systemDark();
      this.consoleThemeMode = mode;
      savePreference(THEME_MODE_STORAGE, mode);
    },
    setThemePreset(preset: ThemePreset) {
      if (!isThemePreset(preset)) return;
      this.themePreset = preset;
      savePreference(THEME_PRESET_STORAGE, preset);
    },
    setRoundedCorners(value: boolean) {
      this.roundedCorners = value;
      savePreference(THEME_ROUNDED_STORAGE, value ? "1" : "0");
    },
    setTransparentSurface(value: boolean) {
      this.transparentSurface = value;
      savePreference(THEME_SURFACE_STORAGE, value ? "1" : "0");
    },
    setFontStyle(value: FontStyle) {
      this.fontStyle = value === "mono" ? "mono" : "mixed";
      savePreference(THEME_FONT_STORAGE, this.fontStyle);
    },
    toggleConsoleTheme() {
      this.setConsoleThemeMode(this.isDarkMode ? "light" : "dark");
    },
    /** Activate console appearance; the returned idempotent cleanup releases its listeners and DOM styles. */
    initConsoleTheme(): () => void {
      if (typeof document === "undefined") return () => {};
      let session = themeSessions.get(this);
      if (!session) {
        // 沉稳大气 replaced the old default; adopt it once for browsers that still store the previous default.
        if (!readPreference(THEME_DEFAULT_MIGRATION)) {
          const stored = readPreference(THEME_PRESET_STORAGE);
          if (!stored || stored === "minimal") {
            this.themePreset = "executive";
            savePreference(THEME_PRESET_STORAGE, "executive");
          }
          savePreference(THEME_DEFAULT_MIGRATION, "1");
        }
        const root = document.documentElement;
        const previousDark = root.classList.contains("console-dark");
        const previousPreset = root.getAttribute("data-console-theme");
        const previousRounded = root.getAttribute("data-console-rounded");
        const previousSurface = root.getAttribute("data-console-surface");
        const previousFont = root.getAttribute("data-console-font");
        const previousScheme = root.style.colorScheme;
        const previousVariables = new Map<string, { value: string; priority: string }>();
        const media = typeof window.matchMedia === "function" ? window.matchMedia("(prefers-color-scheme: dark)") : null;
        this.systemPrefersDark = media?.matches ?? false;
        const onChange = (event: MediaQueryListEvent) => { this.systemPrefersDark = event.matches; };
        if (media?.addEventListener) media.addEventListener("change", onChange);
        else media?.addListener(onChange);
        const stop = watch(() => [this.themePreset, this.isDarkMode, this.roundedCorners, this.transparentSurface, this.fontStyle] as const, () => {
          root.classList.toggle("console-dark", this.isDarkMode);
          root.setAttribute("data-console-theme", this.themePreset);
          root.setAttribute("data-console-rounded", this.roundedCorners ? "on" : "off");
          root.setAttribute("data-console-surface", this.transparentSurface ? "transparent" : "solid");
          root.setAttribute("data-console-font", this.fontStyle);
          root.style.colorScheme = this.isDarkMode ? "dark" : "light";
          const appearance = getThemeAppearance(this.themePreset, this.isDarkMode, {
            rounded: this.roundedCorners, transparent: this.transparentSurface
          });
          for (const [name, value] of Object.entries(appearance.variables)) {
            if (!previousVariables.has(name)) previousVariables.set(name, { value: root.style.getPropertyValue(name), priority: root.style.getPropertyPriority(name) });
            root.style.setProperty(name, value);
          }
        }, { immediate: true, flush: "sync" });
        session = { users: 0, dispose: () => {
          stop();
          if (media?.removeEventListener) media.removeEventListener("change", onChange);
          else media?.removeListener(onChange);
          root.classList.toggle("console-dark", previousDark);
          if (previousPreset === null) root.removeAttribute("data-console-theme");
          else root.setAttribute("data-console-theme", previousPreset);
          if (previousRounded === null) root.removeAttribute("data-console-rounded");
          else root.setAttribute("data-console-rounded", previousRounded);
          if (previousSurface === null) root.removeAttribute("data-console-surface");
          else root.setAttribute("data-console-surface", previousSurface);
          if (previousFont === null) root.removeAttribute("data-console-font");
          else root.setAttribute("data-console-font", previousFont);
          root.style.colorScheme = previousScheme;
          previousVariables.forEach(({ value, priority }, name) => {
            if (value) root.style.setProperty(name, value, priority);
            else root.style.removeProperty(name);
          });
        } };
        themeSessions.set(this, session);
      }
      session.users++;
      const activeSession = session;
      let released = false;
      return () => {
        if (released) return;
        released = true;
        if (--activeSession.users === 0) {
          activeSession.dispose();
          themeSessions.delete(this);
        }
      };
    }
  }
});
