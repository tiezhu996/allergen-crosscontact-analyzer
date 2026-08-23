import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { assessmentApi } from '@/api/domain'
import type { AssessmentRun, AssessmentStatus } from '@/types/assessment'
import type { RiskLevel } from '@/types/risk'

export const useAssessmentStore = defineStore('assessments', () => {
  const runs = ref<AssessmentRun[]>([])
  const loading = ref(false)
  const statusCounts = computed<Record<AssessmentStatus, number>>(() => {
    const counts: Record<AssessmentStatus, number> = { queued: 0, calculating: 0, pending_review: 0, accepted: 0, rejected: 0, stale: 0 }
    runs.value.forEach((run) => counts[run.assessment_status]++)
    return counts
  })
  const riskCounts = computed<Record<RiskLevel, number>>(() => {
    const counts: Record<RiskLevel, number> = { low: 0, medium: 0, high: 0, critical: 0 }
    runs.value.forEach((run) => counts[run.highest_risk_level]++)
    return counts
  })
  async function load() { loading.value = true; try { runs.value = (await assessmentApi.list()).items } finally { loading.value = false } }
  async function create(routeId: number) { const run = await assessmentApi.create(routeId); await load(); return run }
  async function execute(id: number) { const run = await assessmentApi.run(id); await load(); return run }
  async function review(id: number, decision: 'accepted' | 'rejected', reason: string) { const run = await assessmentApi.review(id, decision, reason); await load(); return run }
  return { runs, loading, statusCounts, riskCounts, load, create, execute, review }
})
