import { defineStore } from "pinia";
import { adminLogin, getAdminProfile } from "@/services/admin";

const STORAGE_KEY = "admin_token";

export const useAdminAuthStore = defineStore("adminAuth", {
  state: () => ({
    token: localStorage.getItem(STORAGE_KEY) || "",
    loading: false,
    profile: null,
    totpEnabled: false,
    mfaRequired: false,
    mfaBindRequired: false,
    mfaUnlocked: false
  }),
  actions: {
    async login(payload) {
      this.loading = true;
      try {
        const res = await adminLogin(payload);
        const token = res.data?.access_token || "";
        this.setToken(token);
        this.profile = res.data?.user || res.data?.admin || this.profile;
        this.totpEnabled = !!res.data?.totp_enabled;
        this.mfaRequired = !!res.data?.mfa_required;
        this.mfaBindRequired = !!res.data?.mfa_bind_required;
        this.mfaUnlocked = !!res.data?.mfa_unlocked;
        return token;
      } catch {
        return "";
      } finally {
        this.loading = false;
      }
    },
    async fetchProfile() {
      const res = await getAdminProfile();
      this.profile = res.data || null;
    },
    logout() {
      this.setToken("");
      this.profile = null;
      this.totpEnabled = false;
      this.mfaRequired = false;
      this.mfaBindRequired = false;
      this.mfaUnlocked = false;
    },
    setToken(token) {
      this.token = token || "";
      if (this.token) {
        localStorage.setItem(STORAGE_KEY, this.token);
      } else {
        localStorage.removeItem(STORAGE_KEY);
      }
    },
    setMfaGateState(payload) {
      this.mfaRequired = !!payload?.mfaRequired;
      this.mfaBindRequired = !!payload?.mfaBindRequired;
      if (typeof payload?.mfaUnlocked === "boolean") {
        this.mfaUnlocked = payload.mfaUnlocked;
      }
      if (typeof payload?.totpEnabled === "boolean") {
        this.totpEnabled = payload.totpEnabled;
      }
    }
  }
});
