<script setup lang="ts">
import { ref, watch } from 'vue'
import { auditApi } from '@/api/domain'
import type { VersionDiff } from '@/types/domain'
import { dateTime } from '@/utils/format'

const props = defineProps<{ modelValue: boolean; entityType: string; entityId: number; version: number }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()
const loading = ref(false)
const diff = ref<VersionDiff>()
const error = ref('')
let requestSequence = 0

watch(() => [props.modelValue, props.entityType, props.entityId, props.version] as const, async ([open]) => {
  if (!open || !props.entityId) return
  const sequence = ++requestSequence
  loading.value = true
  error.value = ''
  diff.value = undefined
  try {
    const response = await auditApi.version(props.entityType, props.entityId, props.version)
    if (sequence === requestSequence) diff.value = response
  } catch {
    if (sequence === requestSequence) error.value = '版本摘要加载失败，请关闭后重试。'
  } finally {
    if (sequence === requestSequence) loading.value = false
  }
}, { immediate: true })
</script>

<template>
  <el-dialog :model-value="modelValue" title="版本变更摘要" width="680px" @update:model-value="emit('update:modelValue', $event)">
    <div v-loading="loading" class="version-content">
      <div v-if="error" class="version-state error">{{ error }}</div>
      <template v-else-if="diff">
        <div class="version-meta"><span>{{ diff.entity_type }} #{{ diff.entity_id }}</span><strong>当前 v{{ diff.version }}</strong><span>{{ dateTime(String(diff.latest_audit.created_at || '')) }}</span></div>
        <div class="diff-block"><span>变更前</span><pre>{{ diff.latest_audit.before || '首次创建' }}</pre></div>
        <div class="diff-block after"><span>变更后</span><pre>{{ diff.latest_audit.after }}</pre></div>
        <div class="version-foot">{{ diff.latest_audit.actor }} · {{ diff.latest_audit.action }} · {{ diff.latest_audit.request_id }}</div>
      </template>
      <div v-else-if="!loading" class="version-state">暂无可比较的版本摘要</div>
    </div>
    <template #footer><el-button @click="emit('update:modelValue', false)">关闭</el-button></template>
  </el-dialog>
</template>

<style scoped>
.version-content { min-height: 220px; }.version-state { min-height: 220px; display: grid; place-items: center; padding: 24px; color: var(--muted); text-align: center; }.version-state.error { color: var(--red); }.version-meta { display: grid; grid-template-columns: 1fr auto 1fr; align-items: center; gap: 12px; padding: 10px 12px; background: var(--surface-muted); border: 1px solid var(--line); }.version-meta span:last-child { text-align: right; color: var(--muted); font-size: 11px; }.diff-block { margin-top: 12px; border: 1px solid #dfbab7; }.diff-block > span { display: block; padding: 5px 9px; color: #843d37; background: #faecea; font-size: 11px; font-weight: 800; }.diff-block.after { border-color: #afd0c1; }.diff-block.after > span { color: #27604c; background: #eaf4ef; }.diff-block pre { max-height: 130px; overflow: auto; margin: 0; padding: 10px; color: #3d4743; white-space: pre-wrap; word-break: break-word; font: 11px/1.5 "SFMono-Regular", Consolas, monospace; }.version-foot { margin-top: 12px; color: var(--muted); font-size: 11px; }
@media (max-width: 540px) { .version-meta { grid-template-columns: 1fr; }.version-meta span:last-child { text-align: left; } }
</style>
