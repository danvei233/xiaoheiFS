import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import path from "node:path";

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "src")
    }
  },
  build: {
    chunkSizeWarningLimit: 1600,
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes("node_modules")) return;

          if (id.includes("/echarts/")) return "vendor-echarts";
          if (id.includes("/ant-design-vue/") || id.includes("/@ant-design/")) return "vendor-antd";
          if (id.includes("/vue/") || id.includes("/vue-router/") || id.includes("/pinia/")) return "vendor-vue";
          if (id.includes("/tinymce/") || id.includes("/@tinymce/")) return "vendor-tinymce";
          if (id.includes("/ckeditor5/") || id.includes("/@ckeditor/")) return "vendor-ckeditor";
          if (id.includes("/quill/") || id.includes("/@vueup/")) return "vendor-quill";

          return "vendor";
        }
      }
    }
  },
  server: {
    port: 5173,
    proxy: {
      "/api": {
        target: "http://localhost:8080",
        changeOrigin: true
      },
      "/admin/api": {
        target: "http://localhost:8080",
        changeOrigin: true
      },
      "/sdk": {
        target: "http://localhost:8080",
        changeOrigin: true
      },
      "/uploads": {
        target: "http://localhost:8080",
        changeOrigin: true
      }
    }
  }
});
