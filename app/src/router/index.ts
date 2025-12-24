import { createRouter, createWebHistory } from 'vue-router'
import HelloWorld from '../components/HelloWorld.vue'
import LoginPage from '@/views/auth/LoginPage.vue'
import DashboardView from '@/views/dashboard/DashboardView.vue'
import WorkshopDetailView from '@/views/workshop/WorkshopDetailView.vue'

const routes = [
  {
    path: '/',
    name: 'Home',
    component: HelloWorld
  },
  {
    path: '/dashboard',
    name: 'Dashboard',
    component: DashboardView
  },
  {
    path: '/workshop/:id',
    name: 'WorkshopDetail',
    component: WorkshopDetailView
  },
  {
    path: '/login',
    name: 'Login',
    component: LoginPage,
    meta: { layout: 'empty' }
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router
