import { createRouter, createWebHashHistory, RouteRecordRaw } from 'vue-router'
import Home from '../views/Home.vue'

const routes: Array<RouteRecordRaw> = [
  {
    path: '/',
    name: 'Home',
    component: Home
  },
  {
    path: '/manage',
    name: 'Manage',
    component: () => import(/* webpackChunkName: "manage" */ '../views/Manage.vue')
  },
  {
    path: '/:all(.*)',
    name: 'NotFound',
    component: () => import(/* webpackChunkName: "not_found" */ '../views/NotFound.vue')
  },
]

const router = createRouter({
  history: createWebHashHistory(),
  routes
})

export default router
