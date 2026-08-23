import type { AssessmentRun, MatrixCell, RiskItem } from './assessment'
import type { RiskLevel } from './risk'

export type Role = 'quality_analyst' | 'reviewer' | 'admin'
export interface User { id: number; username: string; display_name: string; role: Role }
export interface RouteStep { step_code: string; step_name: string; profile_id: number }

export interface AllergenProfile {
  id: number; profile_code: string; material_name: string; allergens_json: string[]; source_type: string
  supplier_statement_date?: string; profile_status: 'draft' | 'active' | 'retired'; version: number
  reviewed_by?: number; created_by: number; created_at: string; updated_at: string
}

export interface ProcessRoute {
  id: number; route_code: string; product_name: string; ordered_steps_json: RouteStep[]
  declared_allergens_json: string[]; route_status: 'draft' | 'active' | 'retired'; version: number
  owner_id: number; created_at: string; updated_at: string
}

export interface ContactEdge {
  id: number; route_id: number; from_step_code: string; to_step_code: string; contact_type: string
  shared_equipment: string; cleaning_factor: number; carryover_probability: number; evidence_note: string
  enabled: boolean; version: number; created_by: number; created_at: string; updated_at: string
}

export interface MatrixResult {
  risk_items: RiskItem[]; matrix: MatrixCell[]; declared_allergens: string[]; propagated_allergens: string[]
  highest_risk_level: RiskLevel; thresholds: { medium: number; high: number; critical: number; version: string }
  cycles: string[][]; cycle_edges_skipped: number; depth_limit_reached: number; max_depth: number
}

export interface AuditEvent {
  id: number; request_id: string; actor_id: number; actor_name: string; action: string; entity_type: string
  entity_id: number; before_summary: string; after_summary: string; metadata_json: Record<string, unknown>; created_at: string
}

export interface PageResult<T> { data: T[]; total: number; page: number; pageSize: number }
export interface VersionDiff { entity_type: string; entity_id: number; version: number; latest_audit: Record<string, unknown> }
export type { AssessmentRun }
