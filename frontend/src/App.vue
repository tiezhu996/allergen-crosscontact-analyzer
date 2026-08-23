<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ClipboardCheck, FlaskConical, LogOut, Menu, Network, ScrollText, ShieldCheck, TableProperties, X } from 'lucide-vue-next'
import { useAuth } from '@/hooks/useAuth'
import { roleLabel } from '@/utils/format'

const route = useRoute()
const router = useRouter()
const auth = useAuth()
const navOpen = ref(false)
const isLogin = computed(() => route.name === 'login')
const nav = computed(() => [
  { to: '/profiles', label: '过敏原谱', icon: FlaskConical, show: true },
  { to: '/routes', label: '工艺路线', icon: Network, show: true },
  { to: '/matrix', label: '接触矩阵', icon: TableProperties, show: true },
  { to: '/assessments', label: '评估工作台', icon: ClipboardCheck, show: true },
  { to: '/audit', label: '审计检索', icon: ScrollText, show: auth.canReview.value },
].filter((item) => item.show))

function leave() { auth.logout(); router.push('/login') }
function expired() { leave() }
onMounted(() => window.addEventListener('auth:expired', expired))
onBeforeUnmount(() => window.removeEventListener('auth:expired', expired))
</script>

<template>
  <router-view v-if="isLogin" />
  <div v-else class="app-frame">
    <header class="mobile-bar">
      <button class="plain-icon" :aria-label="navOpen ? '关闭导航' : '打开导航'" @click="navOpen = !navOpen"><X v-if="navOpen" :size="20" /><Menu v-else :size="20" /></button>
      <strong>Allergen Trace</strong><span>内部分析</span>
    </header>
    <aside class="sidebar" :class="{ open: navOpen }">
      <div class="brand"><div class="brand-mark"><ShieldCheck :size="22" /></div><div><strong>Allergen Trace</strong><span>CROSS-CONTACT DESK</span></div></div>
      <nav aria-label="主导航">
        <router-link v-for="item in nav" :key="item.to" :to="item.to" @click="navOpen = false"><component :is="item.icon" :size="18" /><span>{{ item.label }}</span></router-link>
      </nav>
      <div class="boundary"><span class="boundary-mark" />内部风险分析</div>
      <div class="account">
        <div class="avatar">{{ auth.user.value?.display_name.slice(0, 1) }}</div>
        <div class="account-copy"><strong>{{ auth.user.value?.display_name }}</strong><span>{{ roleLabel[auth.user.value?.role || ''] }}</span></div>
        <el-tooltip content="退出登录"><button class="plain-icon" aria-label="退出登录" @click="leave"><LogOut :size="17" /></button></el-tooltip>
      </div>
    </aside>
    <main class="main-content"><router-view /></main>
  </div>
</template>
