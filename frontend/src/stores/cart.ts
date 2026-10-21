import { defineStore } from "pinia";
import { listCart, addCartItem, updateCartItem, deleteCartItem, clearCart } from "@/services/user";

const parseSpec = (spec) => {
  if (!spec) return {};
  if (typeof spec === "string") {
    try {
      return JSON.parse(spec);
    } catch {
      return {};
    }
  }
  return spec;
};

export const useCartStore = defineStore("cart", {
  state: () => ({
    items: [],
    loading: false
  }),
  actions: {
    async fetchCart() {
      this.loading = true;
      try {
        const res = await listCart();
        const items = res.data?.items || [];
        this.items = items.map((row) => ({
          id: row.id ?? row.ID,
          package_id: row.package_id ?? row.PackageID,
          system_id: row.system_id ?? row.SystemID,
          spec: parseSpec(row.spec ?? row.Spec ?? row.spec_json ?? row.SpecJSON),
          qty: row.qty ?? row.Qty,
          amount: row.amount ?? row.Amount
        }));
      } finally {
        this.loading = false;
      }
    },
    async addItem(payload) {
      await addCartItem(payload);
      await this.fetchCart();
    },
    async updateItem(id, payload) {
      await updateCartItem(id, payload);
      await this.fetchCart();
    },
    async removeItem(id) {
      await deleteCartItem(id);
      await this.fetchCart();
    },
    async clearAll() {
      await clearCart();
      this.items = [];
    },
    clear() {
      this.items = [];
    }
  }
});
