export type PresentationTone = 'neutral' | 'danger' | 'success' | 'warning'

export type StatusPresentation = {
  label: string
  auditLabel: string
  tone: PresentationTone
}

const subjectStatuses: Record<string, [string, PresentationTone]> = {
  active: ['黑名单记录', 'danger'],
  cleared: ['已清除', 'success'],
  archived: ['已归档', 'neutral'],
}

const eventStatuses: Record<string, [string, PresentationTone]> = {
  published: ['已公开', 'neutral'],
  corrected: ['已更正', 'warning'],
  withdrawn: ['已撤回', 'neutral'],
}

const appealStatuses: Record<string, [string, PresentationTone]> = {
  pending: ['待处理', 'warning'],
  approved: ['已通过', 'success'],
  rejected: ['已驳回', 'neutral'],
}

function makeStatusPresentation(status: string, map: Record<string, [string, PresentationTone]>, fallbackKey: string): StatusPresentation {
  const entry = map[status]
  if (entry) {
    return { label: entry[0], auditLabel: `${entry[0]}（${status}）`, tone: entry[1] }
  }
  const label = `${fallbackKey}（${status}）`
  return { label, auditLabel: label, tone: 'neutral' }
}

export function subjectStatusPresentation(status: string): StatusPresentation {
  return makeStatusPresentation(status, subjectStatuses, '未知状态')
}

export function eventStatusPresentation(status: string): StatusPresentation {
  return makeStatusPresentation(status, eventStatuses, '未知状态')
}

export function appealStatusPresentation(status: string): StatusPresentation {
  return makeStatusPresentation(status, appealStatuses, '未知状态')
}

const appealOutcomes: Record<string, string> = {
  upheld: '维持原记录',
  corrected: '已更正记录',
  withdrawn: '已撤回记录',
  malicious_submission: '认定为恶意提交',
}

export function appealOutcomeLabel(outcome?: string | null, audit?: boolean): string | null {
  if (!outcome) return null
  const label = appealOutcomes[outcome]
  if (audit) return `${label ?? outcome}（${outcome}）`
  return label ?? outcome
}

const riskLabels = ['未评级', '较低', '低', '中等', '较高', '高']

export function riskPresentation(level?: number | null): { value: number; label: string } {
  const clamped = clamp(toInt(level, 0), 0, 5)
  return { value: clamped, label: riskLabels[clamped] }
}

const severityLabels = ['', '轻微', '较低', '中等', '较高', '严重']

export function severityPresentation(level?: number | null): { value: number; label: string } {
  const clamped = clamp(toInt(level, 1), 1, 5)
  return { value: clamped, label: severityLabels[clamped] }
}

const evidenceTypeLabels: Record<string, string> = {
  text: '文本证据',
  link: '来源链接',
  file: '文件材料',
  image: '图片材料',
}

export function evidenceTypeLabel(type: string): string {
  return evidenceTypeLabels[type] ?? `其他材料（${type}）`
}

export function accountDisplayValue(account: { username?: string | null; account_id?: string | null }): string | null {
  if (account.username && account.username.trim()) return account.username.trim()
  if (account.account_id && account.account_id.trim()) return account.account_id.trim()
  return null
}

export function evidenceAction(
  evidence: { type: string; url?: string | null },
): { href: string; label: string } | null {
  if (evidence.type !== 'link') return null
  if (!evidence.url) return null
  return { href: evidence.url, label: '打开来源链接' }
}

function toInt(n: unknown, fallback: number): number {
  if (n === null || n === undefined) return fallback
  const num = Number(n)
  return Number.isFinite(num) ? num : fallback
}

function clamp(value: number, min: number, max: number): number {
  return Math.max(min, Math.min(max, value))
}
