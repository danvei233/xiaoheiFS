import { defineStore } from "pinia";
import { listNotifications, getUnreadCount, markNotificationRead, markAllNotificationsRead } from "@/services/user";

export const useNotificationsStore = defineStore("notifications", {
  state: () => ({
    items: [] as any[],
    unreadCount: 0,
    loading: false
  }),

  actions: {
    async fetchNotifications(params?: any) {
      this.loading = true;
      try {
        const res = await listNotifications(params);
        this.items = res.data?.items || [];
      } finally {
        this.loading = false;
      }
    },

    async fetchUnreadCount() {
      try {
        const res = await getUnreadCount();
        this.unreadCount = res.data?.unread || 0;
      } catch (error) {
        console.error("Failed to fetch unread count:", error);
      }
    },

    async markAsRead(id: number | string) {
      await markNotificationRead(id);
      const notification = this.items.find(item => item.id === id);
      if (notification) {
        notification.read_at = new Date().toISOString();
      }
      this.unreadCount = Math.max(0, this.unreadCount - 1);
    },

    async markAllAsRead() {
      await markAllNotificationsRead();
      this.items.forEach(item => {
        if (!item.read_at) {
          item.read_at = new Date().toISOString();
        }
      });
      this.unreadCount = 0;
    }
  }
});
