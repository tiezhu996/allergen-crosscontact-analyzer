<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { Eye, GitCompareArrows, RefreshCw, Search } from 'lucide-vue-next'
import { auditApi } from '@/api/domain'
import VersionDiffDialog from '@/components/common/VersionDiffDialog.vue'
import type { AuditEvent } from '@/types/domain'
import { dateTime } from '@/utils/format'

const loading = ref(false)
const events = ref<AuditEvent[]>([])
const total = ref(0)
const query = reactive({ action: '', entity_type: '', request_id: '' })
const selected = ref<AuditEvent>()
const detailOpen = ref(false)
const diffOpen = ref(false)
const selectedVersion = computed(() => Number(selected.value?.metadata_json?.to_version || selected.value?.metadata_json?.version || 1))

async function load() { loading.value = true; try { const result = await auditApi.list({ ...query, action: query.action || undefined, entity_type: query.entity_type || undefined, request_id: query.request_id || undefined, page_size: 100 }); events.value = result.items; total.value = result.total } finally { loading.value = false } }
function inspect(event: AuditEvent) { selected.value = event; detailOpen.value = true }
function compare(event: AuditEvent) { selected.value = event; diffOpen.value = true }
onMounted(load)
</script>

<template>
  <header class="page-header"><div class="page-header-copy"><h1>审计检索</h1><p>输入版本、阈值、评估与复核事件</p></div></header>
  <section class="content-band">
    <div class="toolbar"><el-select v-model="query.entity_type" clearable placeholder="全部实体" style="width: 170px" @change="load"><el-option label="过敏原谱" value="allergen_profile" /><el-option label="工艺路线" value="process_route" /><el-option label="接触边" value="contact_edge" /><el-option label="评估" value="assessment_run" /></el-select><el-input v-model="query.action" clearable placeholder="动作代码" :prefix-icon="Search" style="width: 210px" @keyup.enter="load" /><el-input v-model="query.request_id" clearable placeholder="Request ID" class="request-search" @keyup.enter="load" /><el-tooltip content="刷新"><el-button :icon="RefreshCw" circle aria-label="刷新" @click="load" /></el-tooltip><span class="toolbar-spacer subtle-count">{{ total }} 条事件</span></div>
    <div class="data-surface">
      <el-table v-loading="loading" :data="events" row-key="id">
        <el-table-column label="时间" width="165"><template #default="{ row }">{{ dateTime(row.created_at) }}</template></el-table-column>
        <el-table-column label="操作者" min-width="130"><template #default="{ row }"><strong>{{ row.actor_name }}</strong><div class="muted actor-sub">ID {{ row.actor_id }}</div></template></el-table-column>
        <el-table-column prop="action" label="动作" min-width="190"><template #default="{ row }"><span class="mono">{{ row.action }}</span></template></el-table-column>
        <el-table-column label="实体" width="160"><template #default="{ row }">{{ row.entity_type }} <span class="mono">#{{ row.entity_id }}</span></template></el-table-column>
        <el-table-column label="Request ID" min-width="230"><template #default="{ row }"><span class="mono muted">{{ row.request_id }}</span></template></el-table-column>
        <el-table-column label="" width="96" fixed="right"><template #default="{ row }"><el-tooltip content="查看摘要"><el-button :icon="Eye" text circle aria-label="查看摘要" @click="inspect(row)" /></el-tooltip><el-tooltip content="版本对比"><el-button :icon="GitCompareArrows" text circle aria-label="版本对比" @click="compare(row)" /></el-tooltip></template></el-table-column>
        <template #empty><div class="empty-state"><div><strong>没有匹配的审计事件</strong><span>调整实体、动作或 Request ID</span></div></div></template>
      </el-table>
    </div>
  </section>

  <el-drawer v-model="detailOpen" title="审计事件" size="min(620px, 94vw)"><template v-if="selected"><div class="audit-meta"><div><span>事件</span><strong class="mono">{{ selected.action }}</strong></div><div><span>实体</span><strong>{{ selected.entity_type }} #{{ selected.entity_id }}</strong></div><div><span>请求</span><strong class="mono">{{ selected.request_id }}</strong></div></div><section class="summary-block before"><span>变更前</span><pre>{{ selected.before_summary || '首次创建' }}</pre></section><section class="summary-block after"><span>变更后</span><pre>{{ selected.after_summary }}</pre></section><section class="metadata-block"><span>元数据</span><pre>{{ JSON.stringify(selected.metadata_json, null, 2) }}</pre></section></template></el-drawer>
  <VersionDiffDialog v-if="selected" v-model="diffOpen" :entity-type="selected.entity_type" :entity-id="selected.entity_id" :version="selectedVersion" />
</template>

<style scoped>
.request-search { width: 250px; }.actor-sub { margin-top: 3px; font-size: 10px; }.audit-meta { display: grid; gap: 1px; background: var(--line); border: 1px solid var(--line); }.audit-meta > div { padding: 10px 12px; background: var(--surface); }.audit-meta span, .audit-meta strong { display: block; }.audit-meta span { color: var(--muted); font-size: 10px; }.audit-meta strong { margin-top: 4px; font-size: 12px; word-break: break-all; }.summary-block, .metadata-block { margin-top: 14px; border: 1px solid var(--line); }.summary-block > span, .metadata-block > span { display: block; padding: 6px 10px; background: var(--surface-muted); border-bottom: 1px solid var(--line); font-size: 11px; font-weight: 800; }.summary-block.before > span { color: #853f39; background: #faecea; }.summary-block.after > span { color: #28604d; background: #e9f3ee; }.summary-block pre, .metadata-block pre { max-height: 220px; overflow: auto; margin: 0; padding: 12px; white-space: pre-wrap; word-break: break-word; font: 11px/1.55 "SFMono-Regular", Consolas, monospace; }
@media (max-width: 600px) { .request-search { width: 100%; } }
</style>
