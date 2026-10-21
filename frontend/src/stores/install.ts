import { defineStore } from "pinia";
import { getInstallStatus } from "@/services/user";

interface InstallState {
  loaded: boolean;
  installed: boolean;
}

export const useInstallStore = defineStore("install", {
  state: (): InstallState => ({
    loaded: false,
    installed: true
  }),

  actions: {
    async fetchStatus() {
      try {
        const res = await getInstallStatus();
        this.installed = !!res.data?.installed;
      } catch {
        // If status API is unavailable, do not hard-block navigation.
        this.installed = true;
      } finally {
        this.loaded = true;
      }
    }
  }
});

