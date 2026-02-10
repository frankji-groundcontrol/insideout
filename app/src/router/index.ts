import { createRouter, createWebHistory } from 'vue-router'
import HelloWorld from '../components/HelloWorld.vue'
import LoginPage from '@/views/auth/LoginPage.vue'
import DashboardView from '@/views/dashboard/DashboardView.vue'
import WorkshopDetailView from '@/views/workshop/WorkshopDetailView.vue'
import ExportPreview from '@/views/workshop/ExportPreview.vue'
import UserProfile from '@/views/profile/UserProfile.vue'
import { useUserStore } from '@/stores/user'

const routes = [
  {
    path: '/',
    name: 'Home',
    component: HelloWorld,
    meta: { public: true },
  },
  {
    path: '/dashboard',
    name: 'Dashboard',
    component: DashboardView,
  },
  {
    path: '/workshop/:id',
    name: 'WorkshopDetail',
    component: WorkshopDetailView,
  },
  {
    path: '/workshop/:id/export',
    name: 'WorkshopExport',
    component: ExportPreview,
  },
  {
    path: '/profile',
    name: 'Profile',
    component: UserProfile,
  },
  {
    path: '/login',
    name: 'Login',
    component: LoginPage,
    meta: { layout: 'empty', public: true },
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

// 导航守卫：未认证用户只能访问公开页面
router.beforeEach((to) => {
  const userStore = useUserStore()

  if (to.path === '/login' && userStore.isAuthenticated) {
    return '/dashboard'
  }

  if (!to.meta.public && !userStore.isAuthenticated) {
    return '/login'
  }
})

export default router
