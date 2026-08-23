import type { RiskLevel } from './risk'

export type AssessmentStatus = 'queued' | 'calculating' | 'pending_review' | 'accepted' | 'rejected' | 'stale'

export const assessmentLabels: Record<AssessmentStatus, string> = {
  queued: '待计算',
  calculating: '计算中',
  pending_review: '待复核',
  accepted: '已接受',
  rejected: '已拒绝',
  stale: '已过期',
}

export interface AssessmentRun {
  id: number
  route_id: number
  assessment_status: AssessmentStatus
  input_snapshot_json: Record<string, unknown>
  matrix_json: MatrixCell[]
  risk_items_json: RiskItem[]
  highest_risk_level: RiskLevel
  algorithm_version: string
  created_by: number
  reviewed_by?: number
  review_reason: string
  completed_at?: string
  reviewed_at?: string
  created_at: string
}

export interface EdgeEvidence {
  edge_id: number
  from: string
  to: string
  contact_type: string
  shared_equipment: string
  cleaning_factor: number
  carryover_probability: number
  edge_weight: number
  evidence_note: string
  edge_version: number
}

export interface RiskItem {
  allergen: string
  source_profile_id: number
  source_profile_code: string
  source_material: string
  source_step_code: string
  target_step_code: string
  target_step_name: string
  path: string[]
  raw_score: number
  risk_level: RiskLevel
  declared: boolean
  critical_edge?: EdgeEvidence
  cleaning_evidence: EdgeEvidence[]
  threshold_version: string
}

export interface MatrixCell {
  target_step_code: string
  target_step_name: string
  allergen: string
  max_raw_score: number
  risk_level: RiskLevel
  path_count: number
  declared: boolean
}
