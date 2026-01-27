import { defineStore } from "pinia";
import { adminLogin, getAdminProfile } from "@/services/admin";

const STORAGE_KEY = "admin_token";

export const useAdminAuthStore = defineStore("adminAuth", {
  state: () => ({
    token: localStorage.getItem(STORAGE_KEY) || "",
    loading: false,
    profile: null
  }),
  actions: {
    async login(payload) {
      this.loading = true;
      try {
        const res = await adminLogin(payload);
        const token = res.data?.access_token || "";
        this.token = token;
        this.profile = res.data?.user || res.data?.admin || this.profile;
        if (token) {
          localStorage.setItem(STORAGE_KEY, token);
        }
        return token;
      } finally {
        this.loading = false;
      }
    },
    async fetchProfile() {
      const res = await getAdminProfile();
      this.profile = res.data || null;
    },
    logout() {
      this.token = "";
      this.profile = null;
      localStorage.removeItem(STORAGE_KEY);
    }
  }
});
