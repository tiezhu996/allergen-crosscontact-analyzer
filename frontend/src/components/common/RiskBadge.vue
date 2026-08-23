<script setup lang="ts">
import { computed } from 'vue'
import { riskLabels, type RiskLevel } from '@/types/risk'

const props = defineProps<{ level: RiskLevel; score?: number }>()
const label = computed(() => props.score === undefined ? riskLabels[props.level] : `${riskLabels[props.level]} ${(props.score * 100).toFixed(1)}%`)
</script>

<template><span class="risk-badge" :class="level"><span class="risk-dot" />{{ label }}</span></template>

<style scoped>
.risk-badge { display: inline-flex; align-items: center; gap: 6px; min-height: 25px; padding: 2px 8px; border: 1px solid; border-radius: 3px; font-size: 12px; font-weight: 800; white-space: nowrap; }.risk-dot { width: 7px; height: 7px; border-radius: 50%; background: currentColor; }
.low { color: #24654f; background: #e8f3ee; border-color: #a5cdbd; }.medium { color: #7d5a0b; background: #fff5dc; border-color: #dfc377; }.high { color: #a04f1d; background: #fff0e6; border-color: #e1b89d; }.critical { color: #9b342e; background: #fae9e7; border-color: #dfaaa6; }
</style>
