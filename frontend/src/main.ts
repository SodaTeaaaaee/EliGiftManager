import { createPinia } from "pinia";
import { createApp } from "vue";
import naive from "naive-ui";
import App from "@/app/App.vue";
import { router } from "@/app/router";
import "@/styles/main.css";

const app = createApp(App);
app.use(createPinia());
app.use(naive);
app.use(router);
app.mount("#app");
