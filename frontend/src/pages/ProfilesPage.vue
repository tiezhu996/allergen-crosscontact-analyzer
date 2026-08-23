<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { Edit3, Plus, RefreshCw, Search, Waypoints } from 'lucide-vue-next'
import { ElMessage } from 'element-plus'
import { profileApi } from '@/api/domain'
import { useAuth } from '@/hooks/useAuth'
import type { AllergenProfile } from '@/types/domain'
import { dateTime, sourceTypeLabel } from '@/utils/format'

const auth = useAuth()
const loading = ref(false)
const profiles = ref<AllergenProfile[]>([])
const total = ref(0)
const query = reactive({ search: '', status: '' })
const dialog = ref(false)
const saving = ref(false)
const editing = ref<AllergenProfile>()
const usageOpen = ref(false)
const usageLoading = ref(false)
const usage = ref<Array<{ route_id: number; route_code: string; product_name: string; route_version: number }>>([])
const form = reactive({ profile_code: '', material_name: '', allergens: '', source_type: 'supplier_statement', supplier_statement_date: '', profile_status: 'active' })

async function load() { loading.value = true; try { const result = await profileApi.list({ search: query.search || undefined, status: query.status || undefined, page_size: 100 }); profiles.value = result.items; total.value = result.total } finally { loading.value = false } }
function openCreate() { editing.value = undefined; Object.assign(form, { profile_code: '', material_name: '', allergens: '', source_type: 'supplier_statement', supplier_statement_date: '', profile_status: 'active' }); dialog.value = true }
function openEdit(item: AllergenProfile) { editing.value = item; Object.assign(form, { profile_code: item.profile_code, material_name: item.material_name, allergens: item.allergens_json.join(', '), source_type: item.source_type, supplier_statement_date: item.supplier_statement_date?.slice(0, 10) || '', profile_status: item.profile_status }); dialog.value = true }
async function save() {
  const allergens = form.allergens.split(/[,，]/).map((item) => item.trim()).filter(Boolean)
  if (!form.material_name.trim() || !allergens.length) return
  saving.value = true
  try {
    const payload = { material_name: form.material_name, allergens, source_type: form.source_type, supplier_statement_date: form.supplier_statement_date, profile_status: form.profile_status }
    if (editing.value) await profileApi.update(editing.value.id, { ...payload, expected_version: editing.value.version })
    else await profileApi.create({ ...payload, profile_code: form.profile_code })
    ElMessage.success(editing.value ? '过敏原谱已更新' : '过敏原谱已创建'); dialog.value = false; await load()
  } finally { saving.value = false }
}
async function showUsage(item: AllergenProfile) { usageOpen.value = true; usageLoading.value = true; try { usage.value = (await profileApi.detail(item.id)).used_by_routes } finally { usageLoading.value = false } }
onMounted(load)
</script>

<template>
  <header class="page-header"><div class="page-header-copy"><h1>过敏原谱</h1><p>材料输入、证据来源与路线引用</p></div><div class="page-header-actions"><el-button v-if="auth.canEdit.value" type="primary" :icon="Plus" @click="openCreate">新建谱</el-button></div></header>
  <section class="content-band">
    <div class="toolbar"><el-input v-model="query.search" clearable placeholder="谱代码或材料名称" :prefix-icon="Search" style="width: 250px" @keyup.enter="load" /><el-select v-model="query.status" clearable placeholder="全部状态" style="width: 140px" @change="load"><el-option label="草稿" value="draft" /><el-option label="生效" value="active" /><el-option label="停用" value="retired" /></el-select><el-tooltip content="刷新"><el-button :icon="RefreshCw" circle aria-label="刷新" @click="load" /></el-tooltip><span class="toolbar-spacer subtle-count">{{ total }} 条谱记录</span></div>
    <div class="data-surface">
      <el-table v-loading="loading" :data="profiles" row-key="id">
        <el-table-column label="谱代码" min-width="150"><template #default="{ row }"><button class="link-button mono" @click="showUsage(row)">{{ row.profile_code }}</button></template></el-table-column>
        <el-table-column prop="material_name" label="材料名称" min-width="190" />
        <el-table-column label="过敏原" min-width="180"><template #default="{ row }"><div class="allergen-tags"><span v-for="item in row.allergens_json" :key="item" class="allergen-tag">{{ item }}</span></div></template></el-table-column>
        <el-table-column label="证据来源" min-width="130"><template #default="{ row }">{{ sourceTypeLabel[row.source_type] || row.source_type }}</template></el-table-column>
        <el-table-column label="声明日期" width="130"><template #default="{ row }">{{ row.supplier_statement_date?.slice(0, 10) || '—' }}</template></el-table-column>
        <el-table-column label="状态 / 版本" width="130"><template #default="{ row }"><span class="status-pill" :class="row.profile_status">{{ row.profile_status }}</span><span class="mono muted" style="margin-left: 7px">v{{ row.version }}</span></template></el-table-column>
        <el-table-column label="更新时间" width="160"><template #default="{ row }">{{ dateTime(row.updated_at) }}</template></el-table-column>
        <el-table-column v-if="auth.canEdit.value" label="" width="58" fixed="right"><template #default="{ row }"><el-tooltip content="编辑"><el-button :icon="Edit3" text circle aria-label="编辑" @click="openEdit(row)" /></el-tooltip></template></el-table-column>
        <template #empty><div class="empty-state"><div><strong>暂无匹配的过敏原谱</strong><span>调整筛选条件或创建第一条记录</span></div></div></template>
      </el-table>
    </div>
  </section>

  <el-dialog v-model="dialog" :title="editing ? '编辑过敏原谱' : '新建过敏原谱'" width="620px" destroy-on-close>
    <el-form label-position="top">
      <div class="form-grid"><el-form-item label="谱代码" required><el-input v-model="form.profile_code" :disabled="Boolean(editing)" placeholder="MAT-SESAME" /></el-form-item><el-form-item label="状态" required><el-select v-model="form.profile_status" style="width: 100%"><el-option label="草稿" value="draft" /><el-option label="生效" value="active" /><el-option label="停用" value="retired" /></el-select></el-form-item></div>
      <el-form-item label="材料名称" required><el-input v-model="form.material_name" placeholder="材料或配方输入名称" /></el-form-item>
      <el-form-item label="过敏原" required><el-input v-model="form.allergens" placeholder="多个过敏原用逗号分隔" /></el-form-item>
      <div class="form-grid"><el-form-item label="证据来源" required><el-select v-model="form.source_type" style="width: 100%"><el-option label="供应商声明" value="supplier_statement" /><el-option label="配方依据" value="formulation" /><el-option label="实验室资料" value="laboratory" /><el-option label="内部复核" value="internal_review" /></el-select></el-form-item><el-form-item label="声明日期"><el-date-picker v-model="form.supplier_statement_date" type="date" value-format="YYYY-MM-DD" style="width: 100%" /></el-form-item></div>
    </el-form>
    <template #footer><el-button @click="dialog = false">取消</el-button><el-button type="primary" :loading="saving" @click="save">保存</el-button></template>
  </el-dialog>

  <el-drawer v-model="usageOpen" title="路线引用" size="440px"><div v-loading="usageLoading"><div v-if="!usage.length" class="empty-state"><div><Waypoints :size="22" /><strong>暂无路线引用</strong></div></div><div v-else class="usage-list"><div v-for="item in usage" :key="item.route_id"><strong>{{ item.route_code }}</strong><span>{{ item.product_name }}</span><code>v{{ item.route_version }}</code></div></div></div></el-drawer>
</template>

<style scoped>
.form-grid { display: grid; grid-template-columns: 1fr 180px; gap: 14px; }.usage-list { display: grid; border-top: 1px solid var(--line); }.usage-list > div { display: grid; grid-template-columns: 1fr auto; gap: 4px 10px; padding: 13px 4px; border-bottom: 1px solid var(--line); }.usage-list strong { font: 12px "SFMono-Regular", Consolas, monospace; }.usage-list span { grid-column: 1; color: var(--muted); font-size: 12px; }.usage-list code { grid-column: 2; grid-row: 1 / 3; align-self: center; color: var(--muted); }
@media (max-width: 540px) { .form-grid { grid-template-columns: 1fr; gap: 0; } }
</style>
