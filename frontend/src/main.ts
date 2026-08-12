import { createPinia } from "pinia";
import { createApp } from "vue";
import App from "@/app/App.vue";
import { router } from "@/app/router";
import { setupI18n } from "@/shared/i18n";
import { initTheme } from "@/shared/theme/theme";
import "@/shared/styles/main.css";

async function bootstrap(): Promise<void> {
  const app = createApp(App);
  const pinia = createPinia();

  app.use(pinia);
  app.use(router);
  await setupI18n(app);

// Theme store (preference/density/skin) needs Pinia installed first — wires
// the prefers-color-scheme listener, writes the initial data-theme/
// data-density attributes, and (via its own skinId watcher) loads the
// active skin's tokens.css. NConfigProvider's theme/themeOverrides are
// derived from this same store inside App.vue via useNaiveTheme().
  initTheme(app);

// Naive UI itself needs no global registration — its components are
// imported directly where used, and NConfigProvider (theme + overrides) is
// mounted in App.vue's template. Global feedback (toasts/receipt tray) is
// provided by <FeedbackProvider> inside App.vue as well.

  app.mount("#app");
}

void bootstrap();
