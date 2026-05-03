import { createPinia } from 'pinia'
import naive from 'naive-ui'
import { createApp } from 'vue'
import App from '@/app/App.vue'
import { router } from '@/app/router'
import '@/styles/main.css'
createApp(App).use(createPinia()).use(naive).use(router).mount('#app')
