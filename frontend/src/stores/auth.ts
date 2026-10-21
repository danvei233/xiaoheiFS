import { defineStore } from "pinia";
import { userLogin, getMe, updateMe } from "@/services/user";

const STORAGE_KEY = "user_token";

export const useAuthStore = defineStore("auth", {
  state: () => ({
    token: localStorage.getItem(STORAGE_KEY) || "",
    loading: false,
    profile: null
  }),
  actions: {
    async login(payload) {
      this.loading = true;
      try {
        const res = await userLogin(payload);
        const token = res.data?.access_token || "";
        this.profile = res.data?.user || null;
        this.token = token;
        if (token) {
          localStorage.setItem(STORAGE_KEY, token);
        }
        return token;
      } catch {
        return "";
      } finally {
        this.loading = false;
      }
    },
    async fetchMe() {
      const res = await getMe();
      this.profile = res.data || null;
    },
    async updateProfile(payload) {
      const res = await updateMe(payload);
      this.profile = res.data || null;
    },
    logout() {
      this.token = "";
      this.profile = null;
      localStorage.removeItem(STORAGE_KEY);
    }
  }
});
