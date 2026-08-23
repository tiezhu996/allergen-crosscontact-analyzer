import { computed } from 'vue'
import { useAuthStore } from '@/stores/auth'

export function useAuth() {
  const store = useAuthStore()
  const canEdit = computed(() => store.user?.role === 'quality_analyst' || store.user?.role === 'admin')
  const canReview = computed(() => store.user?.role === 'reviewer' || store.user?.role === 'admin')
  return { user: computed(() => store.user), authenticated: computed(() => store.authenticated), canEdit, canReview, login: store.login, refresh: store.refresh, logout: store.logout }
}
