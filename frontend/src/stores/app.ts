import { defineStore } from "pinia";

const API_KEY_STORAGE = "admin_api_key";

export const useAppStore = defineStore("app", {
  state: () => ({
    adminApiKey: localStorage.getItem(API_KEY_STORAGE) || ""
  }),
  actions: {
    setAdminApiKey(key: string) {
      this.adminApiKey = key;
      if (key) {
        localStorage.setItem(API_KEY_STORAGE, key);
      } else {
        localStorage.removeItem(API_KEY_STORAGE);
      }
    }
  }
});
