import { createApp } from 'vue'
import App from './App.vue'
import router from './router'


import ElementPlus from 'element-plus';
import 'element-plus/lib/theme-chalk/index.css';

createApp(App).use(router).use(ElementPlus,{ size: 'mini', zIndex: 2000 }).mount('#app');
