import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { authApi } from '@/api/domain'
import type { User } from '@/types/domain'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('allergen_token') || '')
  const user = ref<User | null>(JSON.parse(localStorage.getItem('allergen_user') || 'null'))
  const authenticated = computed(() => Boolean(token.value && user.value))

  async function login(username: string, password: string) {
    const response = await authApi.login(username, password)
    token.value = response.token; user.value = response.user
    localStorage.setItem('allergen_token', response.token); localStorage.setItem('allergen_user', JSON.stringify(response.user))
  }
  async function refresh() { user.value = await authApi.me(); localStorage.setItem('allergen_user', JSON.stringify(user.value)) }
  function logout() { token.value = ''; user.value = null; localStorage.removeItem('allergen_token'); localStorage.removeItem('allergen_user') }
  return { token, user, authenticated, login, refresh, logout }
})
