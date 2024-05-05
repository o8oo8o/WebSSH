import { createRouter, createWebHistory } from "vue-router"
import Home from "../views/Home.vue"
import Login from "../views/Login.vue"

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
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
      component: () => import("../views/Manage.vue")
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

export default router
