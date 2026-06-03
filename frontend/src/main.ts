import { createPinia } from "pinia";
import { createApp } from "vue";
import naive from "naive-ui";
import App from "@/app/App.vue";
import { router } from "@/app/router";
import "@/styles/main.css";

const app = createApp(App);
app.use(createPinia());
// Naive UI full import: 27+ distinct components are used across the codebase
// (NConfigProvider, NButton, NCard, NDataTable, NForm, NSelect, NSpace, NGrid,
// NIcon, NTag, NDrawer, NModal, NAlert, NSpin, NSwitch, NCheckbox, NInput,
// NInputNumber, NEmpty, NResult, NText, NTabs, NTabPane, NPopconfirm, NLayout,
// NLayoutSider, NLayoutContent, etc.). Tree-shaking via selective imports would
// save ~100-200KB gzipped but adds significant maintenance burden — every new
// component requires updating the plugin file. Full import is the right tradeoff
// at this component count.
// See: https://www.naiveui.com/en-US/os-theme/docs/import-on-demand
app.use(naive);
app.use(router);
app.mount("#app");
