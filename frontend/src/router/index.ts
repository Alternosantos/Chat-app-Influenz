import { createRouter, createWebHashHistory, RouteRecordRaw } from 'vue-router'
import HomeChat from '../components/HomeChat.vue'

const routes: RouteRecordRaw[] = [
  { path: '/', name: 'Home', component: HomeChat},
]

const router = createRouter({
  history: createWebHashHistory(),
  routes
});

export default router
