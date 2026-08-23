export const percent = (value: number) => `${(value * 100).toFixed(1)}%`
export const dateTime = (value?: string) => value ? new Intl.DateTimeFormat('zh-CN', { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value)) : '—'
export const sourceTypeLabel: Record<string, string> = { supplier_statement: '供应商声明', formulation: '配方依据', laboratory: '实验室资料', internal_review: '内部复核' }
export const roleLabel: Record<string, string> = { quality_analyst: '质量分析员', reviewer: '复核人', admin: '管理员' }
