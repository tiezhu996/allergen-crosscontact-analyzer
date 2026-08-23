import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', name: 'login', component: () => import('@/pages/LoginPage.vue'), meta: { public: true } },
    { path: '/', redirect: '/profiles' },
    { path: '/profiles', name: 'profiles', component: () => import('@/pages/ProfilesPage.vue') },
    { path: '/routes', name: 'routes', component: () => import('@/pages/RoutesPage.vue') },
    { path: '/matrix', name: 'matrix', component: () => import('@/pages/MatrixPage.vue') },
    { path: '/assessments', name: 'assessments', component: () => import('@/pages/AssessmentsPage.vue') },
    { path: '/audit', name: 'audit', component: () => import('@/pages/AuditPage.vue'), meta: { reviewOnly: true } },
    { path: '/:pathMatch(.*)*', redirect: '/profiles' },
  ],
})

router.beforeEach((to) => {
  const auth = useAuthStore()
  if (!to.meta.public && !auth.authenticated) return { name: 'login', query: { redirect: to.fullPath } }
  if (to.name === 'login' && auth.authenticated) return { name: 'profiles' }
  if (to.meta.reviewOnly && auth.user?.role === 'quality_analyst') return { name: 'assessments' }
  return true
})

export default router
