<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { LockKeyhole, ShieldCheck, UserRound } from 'lucide-vue-next'
import { useAuth } from '@/hooks/useAuth'

const auth = useAuth()
const router = useRouter()
const route = useRoute()
const loading = ref(false)
const form = reactive({ username: '', password: '' })

async function submit() {
  if (!form.username || !form.password) return
  loading.value = true
  try { await auth.login(form.username, form.password); await router.push(String(route.query.redirect || '/profiles')) } finally { loading.value = false }
}
</script>

<template>
  <main class="login-shell">
    <section class="login-context">
      <div class="login-brand"><ShieldCheck :size="27" /><div><strong>Allergen Trace</strong><span>CROSS-CONTACT ANALYSIS</span></div></div>
      <div class="matrix-visual" aria-hidden="true">
        <div v-for="index in 30" :key="index" :class="{ active: [3, 9, 14, 18, 24].includes(index), critical: index === 18 }" />
      </div>
      <div class="context-label"><span>证据路径</span><span>版本快照</span><span>人工复核</span></div>
      <p>内部食品安全分析环境</p>
    </section>
    <section class="login-form-wrap">
      <form class="login-form" @submit.prevent="submit">
        <div class="form-heading"><span>受控访问</span><h1>登录分析台</h1><p>使用分配的质量、复核或管理账号</p></div>
        <label><span>用户名</span><el-input v-model="form.username" size="large" autocomplete="username" placeholder="请输入用户名" :prefix-icon="UserRound" /></label>
        <label><span>密码</span><el-input v-model="form.password" size="large" type="password" autocomplete="current-password" show-password placeholder="请输入密码" :prefix-icon="LockKeyhole" /></label>
        <el-button native-type="submit" type="primary" size="large" :loading="loading" :disabled="!form.username || !form.password">登录</el-button>
        <div class="login-boundary"><ShieldCheck :size="15" />仅用于内部风险分析与复核记录</div>
      </form>
    </section>
  </main>
</template>

<style scoped>
.login-shell { min-height: 100vh; display: grid; grid-template-columns: minmax(330px, 43%) 1fr; background: #f4f5f2; }.login-context { min-height: 100vh; display: flex; flex-direction: column; padding: clamp(28px, 5vw, 72px); color: #eaf2ee; background: #24332e; border-right: 1px solid #3f514a; }.login-brand { display: flex; align-items: center; gap: 11px; }.login-brand strong, .login-brand span { display: block; }.login-brand strong { font-size: 19px; }.login-brand span { margin-top: 3px; color: #9eb2aa; font-size: 9px; font-weight: 800; }.matrix-visual { width: min(100%, 470px); display: grid; grid-template-columns: repeat(6, 1fr); aspect-ratio: 6 / 5; margin: auto 0 28px; border: 1px solid #60756c; }.matrix-visual div { border-right: 1px solid #40534b; border-bottom: 1px solid #40534b; background: #293b34; }.matrix-visual div:nth-child(6n) { border-right: 0; }.matrix-visual div:nth-child(n+25) { border-bottom: 0; }.matrix-visual div.active { background: #527f6e; box-shadow: inset 0 0 0 5px #2c4139; }.matrix-visual div.critical { background: #a3594d; }.context-label { display: flex; flex-wrap: wrap; gap: 8px; }.context-label span { padding: 4px 7px; color: #c8d7d1; border: 1px solid #597066; font-size: 11px; }.login-context > p { margin: 16px 0 0; color: #99ada5; font-size: 12px; }.login-form-wrap { min-height: 100vh; display: grid; place-items: center; padding: 32px; }.login-form { width: min(100%, 390px); display: grid; gap: 19px; }.form-heading { margin-bottom: 10px; }.form-heading > span { color: var(--accent); font-size: 11px; font-weight: 800; }.form-heading h1 { margin: 6px 0 7px; font-size: 27px; letter-spacing: 0; }.form-heading p { margin: 0; color: var(--muted); font-size: 13px; }.login-form label { display: grid; gap: 7px; }.login-form label > span { font-size: 12px; font-weight: 800; }.login-form .el-button { width: 100%; margin-top: 5px; }.login-boundary { display: flex; align-items: center; gap: 7px; margin-top: 8px; padding-top: 16px; color: var(--muted); border-top: 1px solid var(--line); font-size: 11px; }
@media (max-width: 760px) { .login-shell { grid-template-columns: 1fr; }.login-context { min-height: 180px; padding: 24px; }.matrix-visual { display: none; }.context-label { margin-top: auto; padding-top: 32px; }.login-context > p { margin-top: 12px; }.login-form-wrap { min-height: calc(100vh - 180px); padding: 32px 20px; } }
</style>
