import { createApp } from "vue";
import App from "./App.vue";
import router from "./router";

// import "element-plus/dist/index.css";
import ElementPlus from "element-plus";

createApp(App)
  .use(router)
  .use(ElementPlus, { size: "small", zIndex: 2000 })
  .mount("#app");
