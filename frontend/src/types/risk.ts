export type RiskLevel = 'low' | 'medium' | 'high' | 'critical'

export const riskLabels: Record<RiskLevel, string> = {
  low: '低',
  medium: '中',
  high: '高',
  critical: '关键',
}
