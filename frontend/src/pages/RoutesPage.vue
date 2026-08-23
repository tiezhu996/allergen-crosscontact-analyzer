<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { Edit3, GitCompareArrows, Plus, RefreshCw, Route, Trash2 } from 'lucide-vue-next'
import { ElMessage } from 'element-plus'
import { edgeApi, profileApi, routeApi } from '@/api/domain'
import VersionDiffDialog from '@/components/common/VersionDiffDialog.vue'
import { useAuth } from '@/hooks/useAuth'
import type { AllergenProfile, ContactEdge, ProcessRoute, RouteStep } from '@/types/domain'
import { percent } from '@/utils/format'

const auth = useAuth()
const routes = ref<ProcessRoute[]>([])
const profiles = ref<AllergenProfile[]>([])
const edges = ref<ContactEdge[]>([])
const loading = ref(false)
const edgeLoading = ref(false)
const selected = ref<ProcessRoute>()
const routeDialog = ref(false)
const edgeDialog = ref(false)
const saving = ref(false)
const editingRoute = ref<ProcessRoute>()
const editingEdge = ref<ContactEdge>()
const diffOpen = ref(false)
const routeForm = reactive<{ route_code: string; product_name: string; route_status: string; declared: string; steps: RouteStep[] }>({ route_code: '', product_name: '', route_status: 'active', declared: '', steps: [] })
const edgeForm = reactive({ from_step_code: '', to_step_code: '', contact_type: 'shared_line', shared_equipment: '', cleaning_factor: 0.5, carryover_probability: 0.5, evidence_note: '', enabled: true })
const activeProfiles = computed(() => profiles.value.filter((item) => item.profile_status === 'active' || editingRoute.value))

async function loadRoutes() {
  loading.value = true
  try { routes.value = (await routeApi.list({ page_size: 100 })).items; if (!selected.value && routes.value.length) await choose(routes.value[0]); else if (selected.value) selected.value = routes.value.find((item) => item.id === selected.value?.id) }
  finally { loading.value = false }
}
async function loadProfiles() { profiles.value = (await profileApi.list({ page_size: 100 })).items }
async function choose(route: ProcessRoute) { selected.value = route; edgeLoading.value = true; try { edges.value = (await edgeApi.list({ route_id: route.id, page_size: 100 })).items } finally { edgeLoading.value = false } }
function newStep(): RouteStep { return { step_code: '', step_name: '', profile_id: profiles.value[0]?.id || 0 } }
function openCreateRoute() { editingRoute.value = undefined; Object.assign(routeForm, { route_code: '', product_name: '', route_status: 'active', declared: '', steps: [newStep(), newStep()] }); routeDialog.value = true }
function openEditRoute(route: ProcessRoute) { editingRoute.value = route; Object.assign(routeForm, { route_code: route.route_code, product_name: route.product_name, route_status: route.route_status, declared: route.declared_allergens_json.join(', '), steps: route.ordered_steps_json.map((step) => ({ ...step })) }); routeDialog.value = true }
async function saveRoute() {
  const declared_allergens = routeForm.declared.split(/[,，]/).map((item) => item.trim()).filter(Boolean)
  if (!routeForm.product_name || routeForm.steps.length < 2 || routeForm.steps.some((step) => !step.step_code || !step.step_name || !step.profile_id)) return
  saving.value = true
  try {
    const payload = { product_name: routeForm.product_name, ordered_steps: routeForm.steps, declared_allergens, route_status: routeForm.route_status }
    if (editingRoute.value) await routeApi.update(editingRoute.value.id, { ...payload, expected_version: editingRoute.value.version })
    else await routeApi.create({ ...payload, route_code: routeForm.route_code })
    ElMessage.success(editingRoute.value ? '路线已生成新版本' : '路线已创建'); routeDialog.value = false; await loadRoutes()
  } finally { saving.value = false }
}
function openCreateEdge() { if (!selected.value) return; editingEdge.value = undefined; const steps = selected.value.ordered_steps_json; Object.assign(edgeForm, { from_step_code: steps[0]?.step_code || '', to_step_code: steps[1]?.step_code || '', contact_type: 'shared_line', shared_equipment: '', cleaning_factor: .5, carryover_probability: .5, evidence_note: '', enabled: true }); edgeDialog.value = true }
function openEditEdge(edge: ContactEdge) { editingEdge.value = edge; Object.assign(edgeForm, { from_step_code: edge.from_step_code, to_step_code: edge.to_step_code, contact_type: edge.contact_type, shared_equipment: edge.shared_equipment, cleaning_factor: edge.cleaning_factor, carryover_probability: edge.carryover_probability, evidence_note: edge.evidence_note, enabled: edge.enabled }); edgeDialog.value = true }
async function saveEdge() {
  if (!selected.value || !edgeForm.shared_equipment || !edgeForm.evidence_note) return
  saving.value = true
  try {
    const payload = { contact_type: edgeForm.contact_type, shared_equipment: edgeForm.shared_equipment, cleaning_factor: edgeForm.cleaning_factor, carryover_probability: edgeForm.carryover_probability, evidence_note: edgeForm.evidence_note, enabled: edgeForm.enabled }
    if (editingEdge.value) await edgeApi.update(editingEdge.value.id, { ...payload, expected_version: editingEdge.value.version })
    else await edgeApi.create({ ...payload, route_id: selected.value.id, from_step_code: edgeForm.from_step_code, to_step_code: edgeForm.to_step_code })
    ElMessage.success(editingEdge.value ? '接触边已更新' : '接触边已创建'); edgeDialog.value = false; await choose(selected.value); await loadRoutes()
  } finally { saving.value = false }
}
onMounted(async () => { await Promise.all([loadProfiles(), loadRoutes()]) })
</script>

<template>
  <header class="page-header"><div class="page-header-copy"><h1>工艺路线</h1><p>步骤顺序、声明谱与共享接触关系</p></div><div class="page-header-actions"><el-button v-if="auth.canEdit.value" type="primary" :icon="Plus" @click="openCreateRoute">新建路线</el-button></div></header>
  <section class="content-band stack">
    <div class="data-surface">
      <div class="surface-header"><Route :size="18" /><h2>路线版本</h2><span>{{ routes.length }} 条</span><el-tooltip content="刷新"><el-button class="toolbar-spacer" :icon="RefreshCw" text circle aria-label="刷新" @click="loadRoutes" /></el-tooltip></div>
      <el-table v-loading="loading" :data="routes" row-key="id" highlight-current-row @row-click="choose">
        <el-table-column label="路线代码" min-width="150"><template #default="{ row }"><button class="link-button mono" @click.stop="choose(row)">{{ row.route_code }}</button></template></el-table-column>
        <el-table-column prop="product_name" label="产品 / 分析对象" min-width="200" />
        <el-table-column label="步骤" width="90"><template #default="{ row }">{{ row.ordered_steps_json.length }}</template></el-table-column>
        <el-table-column label="声明过敏原" min-width="180"><template #default="{ row }"><div class="allergen-tags"><span v-for="item in row.declared_allergens_json" :key="item" class="allergen-tag">{{ item }}</span><span v-if="!row.declared_allergens_json.length" class="muted">无</span></div></template></el-table-column>
        <el-table-column label="状态 / 版本" width="140"><template #default="{ row }"><span class="status-pill" :class="row.route_status">{{ row.route_status }}</span><span class="mono muted" style="margin-left: 7px">v{{ row.version }}</span></template></el-table-column>
        <el-table-column label="" width="104" fixed="right"><template #default="{ row }"><el-tooltip content="版本摘要"><el-button :icon="GitCompareArrows" text circle aria-label="版本摘要" @click.stop="selected = row; diffOpen = true" /></el-tooltip><el-tooltip v-if="auth.canEdit.value" content="编辑"><el-button :icon="Edit3" text circle aria-label="编辑" @click.stop="openEditRoute(row)" /></el-tooltip></template></el-table-column>
        <template #empty><div class="empty-state"><div><strong>暂无工艺路线</strong><span>创建路线后维护步骤和接触关系</span></div></div></template>
      </el-table>
    </div>

    <div class="section-grid">
      <div class="data-surface">
        <div class="surface-header"><h2>接触边</h2><span v-if="selected">{{ selected.route_code }}</span><el-button v-if="auth.canEdit.value && selected" class="toolbar-spacer" :icon="Plus" text @click="openCreateEdge">新增接触边</el-button></div>
        <el-table v-loading="edgeLoading" :data="edges" row-key="id" size="small">
          <el-table-column label="路径" min-width="140"><template #default="{ row }"><span class="mono">{{ row.from_step_code }} → {{ row.to_step_code }}</span></template></el-table-column>
          <el-table-column prop="shared_equipment" label="共享设备" min-width="150" />
          <el-table-column label="清洗 / 带入" width="135"><template #default="{ row }"><span class="score">{{ percent(row.cleaning_factor) }}</span><span class="muted"> / {{ percent(row.carryover_probability) }}</span></template></el-table-column>
          <el-table-column label="启用" width="72"><template #default="{ row }"><span class="status-pill" :class="row.enabled ? 'active' : 'retired'">{{ row.enabled ? '是' : '否' }}</span></template></el-table-column>
          <el-table-column v-if="auth.canEdit.value" label="" width="52"><template #default="{ row }"><el-button :icon="Edit3" text circle aria-label="编辑接触边" @click="openEditEdge(row)" /></template></el-table-column>
          <template #empty><div class="empty-state"><div><strong>{{ selected ? '暂无接触边' : '请选择路线' }}</strong><span v-if="selected">添加真实接触关系后可运行矩阵</span></div></div></template>
        </el-table>
      </div>
      <div class="data-surface step-surface">
        <div class="surface-header"><h2>步骤序列</h2><span v-if="selected">v{{ selected.version }}</span></div>
        <div v-if="selected" class="step-list"><div v-for="(step, index) in selected.ordered_steps_json" :key="step.step_code" class="step-row"><span>{{ index + 1 }}</span><div><strong>{{ step.step_code }}</strong><small>{{ step.step_name }}</small></div><code>谱 #{{ step.profile_id }}</code></div></div>
        <div v-else class="empty-state"><span>请选择路线</span></div>
      </div>
    </div>
  </section>

  <el-dialog v-model="routeDialog" :title="editingRoute ? '编辑工艺路线' : '新建工艺路线'" width="780px" destroy-on-close>
    <el-form label-position="top">
      <div class="route-form-grid"><el-form-item label="路线代码" required><el-input v-model="routeForm.route_code" :disabled="Boolean(editingRoute)" placeholder="RT-COOKIE-01" /></el-form-item><el-form-item label="产品 / 分析对象" required><el-input v-model="routeForm.product_name" /></el-form-item><el-form-item label="状态" required><el-select v-model="routeForm.route_status" style="width: 100%"><el-option label="草稿" value="draft" /><el-option label="生效" value="active" /><el-option label="停用" value="retired" /></el-select></el-form-item></div>
      <el-form-item label="声明过敏原"><el-input v-model="routeForm.declared" placeholder="多个项目用逗号分隔" /></el-form-item>
      <div class="steps-editor"><div class="editor-title"><strong>有序步骤</strong><el-button :icon="Plus" text @click="routeForm.steps.push(newStep())">添加步骤</el-button></div><div v-for="(step, index) in routeForm.steps" :key="index" class="step-editor-row"><span class="step-index">{{ index + 1 }}</span><el-input v-model="step.step_code" placeholder="步骤代码" /><el-input v-model="step.step_name" placeholder="步骤名称" /><el-select v-model="step.profile_id" filterable placeholder="过敏原谱"><el-option v-for="profile in activeProfiles" :key="profile.id" :label="`${profile.profile_code} · ${profile.material_name}`" :value="profile.id" /></el-select><el-tooltip content="删除步骤"><el-button :icon="Trash2" text circle :disabled="routeForm.steps.length <= 2" aria-label="删除步骤" @click="routeForm.steps.splice(index, 1)" /></el-tooltip></div></div>
    </el-form>
    <template #footer><el-button @click="routeDialog = false">取消</el-button><el-button type="primary" :loading="saving" @click="saveRoute">保存</el-button></template>
  </el-dialog>

  <el-dialog v-model="edgeDialog" :title="editingEdge ? '编辑接触边' : '新增接触边'" width="680px" destroy-on-close>
    <el-form label-position="top"><div class="edge-grid"><el-form-item label="来源步骤" required><el-select v-model="edgeForm.from_step_code" :disabled="Boolean(editingEdge)" style="width: 100%"><el-option v-for="step in selected?.ordered_steps_json" :key="step.step_code" :label="`${step.step_code} · ${step.step_name}`" :value="step.step_code" /></el-select></el-form-item><el-form-item label="目标步骤" required><el-select v-model="edgeForm.to_step_code" :disabled="Boolean(editingEdge)" style="width: 100%"><el-option v-for="step in selected?.ordered_steps_json" :key="step.step_code" :label="`${step.step_code} · ${step.step_name}`" :value="step.step_code" /></el-select></el-form-item></div><div class="edge-grid"><el-form-item label="接触类型" required><el-select v-model="edgeForm.contact_type" style="width: 100%"><el-option label="顺序接触" value="sequence" /><el-option label="共享产线" value="shared_line" /><el-option label="返工" value="rework" /><el-option label="空气传播" value="airborne" /><el-option label="人工转移" value="manual_transfer" /></el-select></el-form-item><el-form-item label="共享设备" required><el-input v-model="edgeForm.shared_equipment" /></el-form-item></div><div class="slider-row"><label><span>清洗衰减 <strong>{{ percent(edgeForm.cleaning_factor) }}</strong></span><el-slider v-model="edgeForm.cleaning_factor" :min="0" :max="1" :step="0.05" /></label><label><span>带入概率 <strong>{{ percent(edgeForm.carryover_probability) }}</strong></span><el-slider v-model="edgeForm.carryover_probability" :min="0" :max="1" :step="0.05" /></label></div><el-form-item label="清洗与证据记录" required><el-input v-model="edgeForm.evidence_note" type="textarea" :rows="3" maxlength="1000" show-word-limit /></el-form-item><el-form-item label="参与计算"><el-switch v-model="edgeForm.enabled" active-text="启用" inactive-text="停用" /></el-form-item></el-form>
    <template #footer><el-button @click="edgeDialog = false">取消</el-button><el-button type="primary" :loading="saving" @click="saveEdge">保存</el-button></template>
  </el-dialog>

  <VersionDiffDialog v-if="selected" v-model="diffOpen" entity-type="process_route" :entity-id="selected.id" :version="selected.version" />
</template>

<style scoped>
.step-list { display: grid; }.step-row { min-height: 58px; display: grid; grid-template-columns: 26px 1fr auto; align-items: center; gap: 9px; padding: 8px 13px; border-bottom: 1px solid var(--line); }.step-row:last-child { border-bottom: 0; }.step-row > span { width: 23px; height: 23px; display: grid; place-items: center; color: var(--muted); border: 1px solid var(--line-strong); font-size: 11px; }.step-row strong, .step-row small { display: block; }.step-row strong { font: 12px "SFMono-Regular", Consolas, monospace; }.step-row small { margin-top: 3px; color: var(--muted); }.step-row code { color: var(--muted); font-size: 11px; }.route-form-grid { display: grid; grid-template-columns: 190px 1fr 130px; gap: 12px; }.steps-editor { border: 1px solid var(--line); }.editor-title { min-height: 46px; display: flex; align-items: center; justify-content: space-between; padding: 7px 12px; background: var(--surface-muted); border-bottom: 1px solid var(--line); }.editor-title strong { font-size: 13px; }.step-editor-row { display: grid; grid-template-columns: 28px 140px 1fr minmax(190px, 1.2fr) 36px; align-items: center; gap: 8px; padding: 9px; border-bottom: 1px solid var(--line); }.step-editor-row:last-child { border-bottom: 0; }.step-index { color: var(--muted); text-align: center; font-size: 11px; }.edge-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 14px; }.slider-row { display: grid; grid-template-columns: 1fr 1fr; gap: 26px; margin-bottom: 18px; padding: 12px 18px; background: var(--surface-muted); border: 1px solid var(--line); }.slider-row label > span { display: flex; justify-content: space-between; margin-bottom: 4px; font-size: 12px; }.slider-row strong { font-variant-numeric: tabular-nums; }
@media (max-width: 700px) { .route-form-grid, .edge-grid, .slider-row { grid-template-columns: 1fr; gap: 0; }.step-editor-row { grid-template-columns: 28px 1fr 36px; }.step-editor-row .el-input:nth-of-type(2), .step-editor-row .el-select { grid-column: 2; }.step-editor-row .el-button { grid-column: 3; grid-row: 1 / 4; } }
</style>
