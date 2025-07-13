import { createRouter, createWebHistory, createWebHashHistory } from "vue-router"
import Home from "../views/Home.vue"
import Login from "../views/Login.vue"

let basePath = "";
let history = createWebHashHistory(basePath);
if (import.meta.env.VITE_ROUTE_MODE === "WebHistory") {
  if (import.meta.env.VITE_WEB_BASE_DIR) {
    basePath = `${import.meta.env.VITE_WEB_BASE_DIR}`;
  }
  history = createWebHistory(basePath);
}

const router = createRouter({
  history: history,
  routes: [
    {
      path: "/",
      name: "Home",
      component: Home
    },
    {
      path: "/app",
      name: "Home",
      component: Home
    },
    {
      path: "/login",
      name: "Login",
      component: Login
    },
    {
      path: "/manage",
      name: "Manage",
      component: () => import(/* webpackChunkName: "manage" */ '../views/Manage.vue')
    },

    {
      path: "/sys-init",
      name: "SysInit",
      component: () => import("../views/SysInit.vue")
    },
    {
      path: "/about",
      name: "About",
      component: () => import("../views/About.vue")
    },
    {
      path: "/:all(.*)",
      name: "NotFound",
      component: () => import("../views/NotFound.vue")
    },

  ]
})

export default router;
