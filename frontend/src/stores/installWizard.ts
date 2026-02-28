import { defineStore } from "pinia";

type DBType = "sqlite" | "mysql";

interface InstallWizardState {
  dbType: DBType;
  sqlitePath: string;
  mysql: {
    host: string;
    port: number;
    user: string;
    pass: string;
    dbName: string;
    params: string;
  };
  dbChecked: boolean;
  dbCheckError: string;

  siteName: string;
  siteUrl: string;

  adminUser: string;
  adminPass: string;
  adminPath: string;
}

const STORAGE_KEY = "install_wizard_v1";

function safeLoad(): Partial<InstallWizardState> | null {
  try {
    const raw = sessionStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    return JSON.parse(raw) as Partial<InstallWizardState>;
  } catch {
    return null;
  }
}

export const useInstallWizardStore = defineStore("installWizard", {
  state: (): InstallWizardState => {
    const saved = safeLoad() || {};
    return {
      dbType: saved.dbType === "sqlite" ? "sqlite" : "mysql",
      sqlitePath: saved.sqlitePath || "./data/app.db",
      mysql: {
        host: saved.mysql?.host || "127.0.0.1",
        port: typeof saved.mysql?.port === "number" ? saved.mysql.port : 3306,
        user: saved.mysql?.user || "root",
        pass: saved.mysql?.pass || "",
        dbName: saved.mysql?.dbName || "",
        params: saved.mysql?.params || "charset=utf8mb4&parseTime=True&loc=Local"
      },
      dbChecked: false,
      dbCheckError: "",

      siteName: saved.siteName || "",
      siteUrl: saved.siteUrl || "",

      adminUser: saved.adminUser || "admin",
      adminPass: "",
      adminPath: saved.adminPath || ""
    };
  },

  getters: {
    mysqlDSN(state): string {
      const u = encodeURIComponent(state.mysql.user || "");
      const p = encodeURIComponent(state.mysql.pass || "");
      const host = state.mysql.host || "127.0.0.1";
      const port = Number(state.mysql.port || 3306);
      const dbName = state.mysql.dbName || "";
      const params = state.mysql.params ? `?${state.mysql.params}` : "";
      return `${u}:${p}@tcp(${host}:${port})/${dbName}${params}`;
    }
  },

  actions: {
    touchDB() {
      this.dbChecked = false;
      this.dbCheckError = "";
      this.persist();
    },
    markDBChecked(ok: boolean, errMsg = "") {
      this.dbChecked = ok;
      this.dbCheckError = ok ? "" : errMsg;
      this.persist();
    },
    persist() {
      try {
        sessionStorage.setItem(
          STORAGE_KEY,
          JSON.stringify({
            dbType: this.dbType,
            sqlitePath: this.sqlitePath,
            mysql: this.mysql,
            siteName: this.siteName,
            siteUrl: this.siteUrl,
            adminUser: this.adminUser,
            adminPath: this.adminPath
          })
        );
      } catch {
        // ignore
      }
    },
    reset() {
      sessionStorage.removeItem(STORAGE_KEY);
      this.$reset();
    }
  }
});
