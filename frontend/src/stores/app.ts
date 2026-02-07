import { defineStore } from "pinia";

const API_KEY_STORAGE = "admin_api_key";
const THEME_MODE_STORAGE = "console_theme_mode";

export const useAppStore = defineStore("app", {
  state: () => ({
    adminApiKey: localStorage.getItem(API_KEY_STORAGE) || "",
    consoleThemeMode: (localStorage.getItem(THEME_MODE_STORAGE) as 'light' | 'dark') || 'light'
  }),
  getters: {
    isDarkMode: (state) => state.consoleThemeMode === 'dark'
  },
  actions: {
    setAdminApiKey(key: string) {
      this.adminApiKey = key;
      if (key) {
        localStorage.setItem(API_KEY_STORAGE, key);
      } else {
        localStorage.removeItem(API_KEY_STORAGE);
      }
    },
    setConsoleThemeMode(mode: 'light' | 'dark') {
      this.consoleThemeMode = mode;
      localStorage.setItem(THEME_MODE_STORAGE, mode);
      if (mode === 'dark') {
        document.documentElement.classList.add('console-dark');
      } else {
        document.documentElement.classList.remove('console-dark');
      }
    },
    toggleConsoleTheme() {
      this.setConsoleThemeMode(this.consoleThemeMode === 'light' ? 'dark' : 'light');
    },
    initConsoleTheme() {
      const savedMode = localStorage.getItem(THEME_MODE_STORAGE) as 'light' | 'dark' | null;
      const mode = savedMode || 'light';
      if (mode === 'dark') {
        document.documentElement.classList.add('console-dark');
      }
      this.consoleThemeMode = mode;
    }
  }
});
