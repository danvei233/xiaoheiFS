import { createApp } from "vue";
import { createPinia } from "pinia";
import Antd from "ant-design-vue";
import "ant-design-vue/dist/reset.css";
import App from "./App.vue";
import router from "./router";
import "./styles/theme.css";
import "./styles/admin.css";
import "./theme.css";
import { t } from "./locales";

const app = createApp(App);
app.use(createPinia());
app.use(router);
app.use(Antd);
app.config.globalProperties.$t = t;
app.mount("#app");
