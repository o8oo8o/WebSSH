import { createApp } from "vue";
import { createPinia } from "pinia";

import App from "./App.vue";
import router from "./router";
import ElementPlus from "element-plus";
import "element-plus/dist/index.css";
import axios from "axios";
import piniaPluginPersistedstate from "pinia-plugin-persistedstate";

import { useGlobalStore } from "./stores/store";

const app = createApp(App);

const pinia = createPinia();
pinia.use(piniaPluginPersistedstate);

app.use(pinia);
app.use(ElementPlus, { size: "small", zIndex: 2000 });
app.use(router);

let globalStore = useGlobalStore();

// 导航守卫 配置
// 使用 router.beforeEach 注册一个全局前置守卫，判断用户是否登陆
router.beforeEach((to, from) => {
    if ((!globalStore.isInit) && to.name === "SysInit") {
        return true;
    }

    if (to.name === "Login") {
        return true;
    }

    var local_auth = localStorage.getItem("auth");
    if (local_auth === "yes" && globalStore.isLogin) {
        return true
    }

    router.push({ "name": "Login" });
    return false;
});


//////////////
//   拦截器
//////////////

axios.interceptors.request.use(
    (req) => {
        // 在发送请求之前加token
        req.headers.Time = String(new Date().getTime());
        req.headers.Authorization = localStorage.getItem("token");
        return req;
    },
    (err) => {
        return Promise.reject(err);
    }
);

// 添加响应拦截器
axios.interceptors.response.use(
    (res) => {
        let newToken = res.headers["newtoken"];
        if (newToken) {
            // 若有newtoken则刷新token
            localStorage.setItem("token", newToken);
        }
        return res;
    },
    (err) => {
        if (err.response && (err.response.status === 401)) {
            router.replace({ "name": "Login" });
        }
        return Promise.reject(err);
    }
)

app.mount("#app");
