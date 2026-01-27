import axios from "axios";
import { message, notification, Modal } from "ant-design-vue";
import { useAuthStore } from "@/stores/auth";
import { useAdminAuthStore } from "@/stores/adminAuth";
import { useAppStore } from "@/stores/app";

const apiBase = import.meta.env.VITE_API_BASE || "";

export const http = axios.create({
  baseURL: apiBase,
  timeout: 20000
});

let realnameModalOpen = false;

http.interceptors.request.use((config) => {
  const user = useAuthStore();
  const admin = useAdminAuthStore();
  const app = useAppStore();
  if (config.url?.startsWith("/api") && user.token) {
    config.headers = config.headers || {};
    config.headers.Authorization = `Bearer ${user.token}`;
  }
  if (config.url?.startsWith("/admin/api") && admin.token) {
    config.headers = config.headers || {};
    config.headers.Authorization = `Bearer ${admin.token}`;
  }
  if (config.headers && "X-Use-Api-Key" in config.headers) {
    if (app.adminApiKey) {
      config.headers["X-API-Key"] = app.adminApiKey;
    }
    delete config.headers["X-Use-Api-Key"];
  }
  return config;
});

http.interceptors.response.use(
  (res) => res,
  (error) => {
    const status = error?.response?.status;
    const msg = error?.response?.data?.error || error?.response?.data?.message || error?.message || "Request failed";
    const url = error?.config?.url || "";
    if (status === 401) {
      if (url.startsWith("/admin/")) {
        const admin = useAdminAuthStore();
        admin.logout();
        if (window.location.pathname.startsWith("/admin")) {
          window.location.href = "/admin/login";
        }
      } else {
        const user = useAuthStore();
        user.logout();
        if (!window.location.pathname.startsWith("/admin")) {
          window.location.href = "/";
        }
      }
      message.error("鉴权失败，请重新登录");
    } else if (status === 403 && url.startsWith("/api") && msg.toLowerCase().includes("real name required")) {
      if (!realnameModalOpen) {
        realnameModalOpen = true;
        Modal.confirm({
          title: "需要实名认证",
          content: "该操作需要完成实名认证，是否前往认证页面？",
          okText: "去认证",
          cancelText: "稍后再说",
          onOk: () => {
            window.location.href = "/console/realname";
          },
          onCancel: () => {
            realnameModalOpen = false;
          },
          afterClose: () => {
            realnameModalOpen = false;
          }
        });
      }
    } else if (status >= 500) {
      notification.error({ message: "服务端错误", description: msg });
    } else {
      message.error(msg);
    }
    return Promise.reject(error);
  }
);

export const withApiKey = (headers: Record<string, string> = {}) => ({
  headers: {
    ...headers,
    "X-Use-Api-Key": "1"
  }
});
