<script setup lang="ts">
import { computed } from 'vue'
import { ArrowRight, Beaker, Wrench } from 'lucide-vue-next'
import RiskBadge from './RiskBadge.vue'
import { percent } from '@/utils/format'
import type { RiskItem } from '@/types/assessment'

const props = defineProps<{ item?: RiskItem }>()
const title = computed(() => props.item ? `${props.item.allergen} · ${props.item.source_profile_code}` : '传播证据')
</script>

<template>
  <section class="evidence-panel" aria-live="polite">
    <div class="evidence-head"><div><span class="eyebrow">传播证据</span><h3>{{ title }}</h3></div><RiskBadge v-if="item" :level="item.risk_level" :score="item.raw_score" /></div>
    <div v-if="!item" class="evidence-empty">选择矩阵或风险条目查看完整路径</div>
    <template v-else>
      <div class="evidence-source"><Beaker :size="17" /><div><strong>{{ item.source_material }}</strong><span>{{ item.source_step_code }} · 谱 #{{ item.source_profile_id }}</span></div></div>
      <div class="path-line"><template v-for="(step, index) in item.path" :key="`${step}-${index}`"><span class="path-step">{{ step }}</span><ArrowRight v-if="index < item.path.length - 1" :size="14" /></template></div>
      <dl class="evidence-facts"><div><dt>目标步骤</dt><dd>{{ item.target_step_name }}</dd></div><div><dt>阈值版本</dt><dd>{{ item.threshold_version }}</dd></div><div><dt>声明状态</dt><dd>{{ item.declared ? '已声明' : '传播项' }}</dd></div></dl>
      <div class="edge-list">
        <article v-for="edge in item.cleaning_evidence" :key="edge.edge_id" class="edge-row">
          <Wrench :size="15" /><div class="edge-copy"><div><strong>{{ edge.from }} → {{ edge.to }}</strong><span>{{ edge.shared_equipment }}</span></div><p>{{ edge.evidence_note }}</p><div class="edge-metrics"><span>清洗 {{ percent(edge.cleaning_factor) }}</span><span>带入 {{ percent(edge.carryover_probability) }}</span><span>边权 {{ percent(edge.edge_weight) }}</span><span>v{{ edge.edge_version }}</span></div></div>
        </article>
      </div>
    </template>
  </section>
</template>

<style scoped>
.evidence-panel { min-height: 330px; background: var(--surface); border: 1px solid var(--line); }.evidence-head { min-height: 62px; display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 12px 14px; border-bottom: 1px solid var(--line); }.eyebrow { color: var(--muted); font-size: 10px; font-weight: 800; }.evidence-head h3 { margin: 3px 0 0; font-size: 15px; letter-spacing: 0; }.evidence-empty { min-height: 265px; display: grid; place-items: center; padding: 24px; color: var(--muted); text-align: center; }.evidence-source { display: flex; gap: 10px; margin: 14px; padding: 10px; background: #f0f3f0; border-left: 3px solid var(--accent); }.evidence-source strong, .evidence-source span { display: block; }.evidence-source strong { font-size: 13px; }.evidence-source span { margin-top: 3px; color: var(--muted); font-size: 11px; }.path-line { display: flex; align-items: center; flex-wrap: wrap; gap: 5px; padding: 0 14px 14px; color: var(--muted); }.path-step { padding: 4px 7px; color: var(--text); background: #fff; border: 1px solid var(--line-strong); font: 11px "SFMono-Regular", Consolas, monospace; }.evidence-facts { display: grid; grid-template-columns: repeat(3, 1fr); margin: 0; border-block: 1px solid var(--line); }.evidence-facts div { padding: 9px 12px; border-right: 1px solid var(--line); }.evidence-facts div:last-child { border-right: 0; }.evidence-facts dt { color: var(--muted); font-size: 10px; }.evidence-facts dd { margin: 3px 0 0; font-size: 12px; font-weight: 700; }.edge-list { display: grid; }.edge-row { display: flex; gap: 9px; padding: 12px 14px; border-bottom: 1px solid var(--line); }.edge-row:last-child { border-bottom: 0; }.edge-copy { min-width: 0; flex: 1; }.edge-copy > div:first-child { display: flex; justify-content: space-between; gap: 8px; }.edge-copy strong { font-size: 12px; }.edge-copy span { color: var(--muted); font-size: 11px; }.edge-copy p { margin: 6px 0; color: #4c5752; font-size: 11px; line-height: 1.45; }.edge-metrics { display: flex; flex-wrap: wrap; gap: 8px; }.edge-metrics span { color: #53625c; font-variant-numeric: tabular-nums; }
@media (max-width: 520px) { .evidence-facts { grid-template-columns: 1fr; }.evidence-facts div { border-right: 0; border-bottom: 1px solid var(--line); }.evidence-facts div:last-child { border-bottom: 0; } }
</style>
