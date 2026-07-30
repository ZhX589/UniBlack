import { describe, expect, it } from 'vitest'
import {
  accountDisplayValue,
  appealOutcomeLabel,
  appealStatusPresentation,
  eventStatusPresentation,
  evidenceAction,
  evidenceTypeLabel,
  riskPresentation,
  severityPresentation,
  subjectStatusPresentation,
} from './trustworthy-profile'

describe('trustworthy profile presentation', () => {
  it('maps known statuses with label, auditLabel, and tone', () => {
    expect(subjectStatusPresentation('active')).toEqual({
      label: '黑名单记录', auditLabel: '黑名单记录（active）', tone: 'danger',
    })
    expect(subjectStatusPresentation('cleared')).toEqual({
      label: '已清除', auditLabel: '已清除（cleared）', tone: 'success',
    })
    expect(subjectStatusPresentation('archived')).toEqual({
      label: '已归档', auditLabel: '已归档（archived）', tone: 'neutral',
    })
    expect(eventStatusPresentation('published')).toEqual({
      label: '已公开', auditLabel: '已公开（published）', tone: 'neutral',
    })
    expect(eventStatusPresentation('corrected').label).toBe('已更正')
    expect(eventStatusPresentation('corrected').tone).toBe('warning')
    expect(eventStatusPresentation('withdrawn').label).toBe('已撤回')
    expect(appealStatusPresentation('pending').label).toBe('待处理')
    expect(appealStatusPresentation('approved').label).toBe('已通过')
    expect(appealStatusPresentation('rejected').label).toBe('已驳回')
  })

  it('preserves raw enum in auditLabel for unknown values with neutral tone', () => {
    const unknown = subjectStatusPresentation('something_never_seen')
    expect(unknown.label).toBe('未知状态（something_never_seen）')
    expect(unknown.auditLabel).toBe('未知状态（something_never_seen）')
    expect(unknown.tone).toBe('neutral')

    expect(appealStatusPresentation('escalated').auditLabel).toBe('未知状态（escalated）')
    expect(appealStatusPresentation('escalated').tone).toBe('neutral')
  })

  it('maps all appeal outcomes with optional audit suffix', () => {
    expect(appealOutcomeLabel('malicious_submission', true)).toBe('认定为恶意提交（malicious_submission）')
    expect(appealOutcomeLabel('malicious_submission')).toBe('认定为恶意提交')
    expect(appealOutcomeLabel('upheld', true)).toBe('维持原记录（upheld）')
    expect(appealOutcomeLabel('upheld')).toBe('维持原记录')
    expect(appealOutcomeLabel('corrected', true)).toBe('已更正记录（corrected）')
    expect(appealOutcomeLabel('corrected')).toBe('已更正记录')
    expect(appealOutcomeLabel('withdrawn', true)).toBe('已撤回记录（withdrawn）')
    expect(appealOutcomeLabel('withdrawn')).toBe('已撤回记录')
  })

  it('returns null for nil/empty appeal outcomes', () => {
    expect(appealOutcomeLabel(undefined)).toBeNull()
    expect(appealOutcomeLabel(null)).toBeNull()
    expect(appealOutcomeLabel('')).toBeNull()
  })

  it('clamps risk to 0-5 and severity to 1-5', () => {
    expect(riskPresentation(9).value).toBe(5)
    expect(riskPresentation(9).label).toBe('高')
    expect(riskPresentation(-1).value).toBe(0)
    expect(riskPresentation(-1).label).toBe('未评级')
    expect(riskPresentation(undefined).value).toBe(0)
    expect(riskPresentation(null).value).toBe(0)

    expect(severityPresentation(0).value).toBe(1)
    expect(severityPresentation(0).label).toBe('轻微')
    expect(severityPresentation(6).value).toBe(5)
    expect(severityPresentation(6).label).toBe('严重')
    expect(severityPresentation(undefined).value).toBe(1)
    expect(severityPresentation(null).value).toBe(1)
  })

  it('returns trimmed username, then trimmed account_id, else null', () => {
    expect(accountDisplayValue({ username: 'alice', account_id: '42' })).toBe('alice')
    expect(accountDisplayValue({ username: '  bob  ', account_id: '99' })).toBe('bob')
    expect(accountDisplayValue({ username: '', account_id: '42' })).toBe('42')
    expect(accountDisplayValue({ username: ' ', account_id: '42' })).toBe('42')
    expect(accountDisplayValue({ username: null, account_id: '  chars  ' })).toBe('chars')
    expect(accountDisplayValue({ username: null, account_id: null })).toBeNull()
    expect(accountDisplayValue({ username: undefined, account_id: null })).toBeNull()
  })

  it('labels every evidence type as specified', () => {
    expect(evidenceTypeLabel('text')).toBe('文本证据')
    expect(evidenceTypeLabel('link')).toBe('来源链接')
    expect(evidenceTypeLabel('file')).toBe('文件材料')
    expect(evidenceTypeLabel('image')).toBe('图片材料')
    expect(evidenceTypeLabel('unknown')).toBe('其他材料（unknown）')
    expect(evidenceTypeLabel('custom_type')).toBe('其他材料（custom_type）')
  })

  it('returns evidence action only for link type with non-empty url', () => {
    expect(evidenceAction({ type: 'file', url: null })).toBeNull()
    expect(evidenceAction({ type: 'file', url: 'https://example.test/doc.pdf' })).toBeNull()
    expect(evidenceAction({ type: 'text', url: 'https://example.test/note' })).toBeNull()
    expect(evidenceAction({ type: 'link', url: '' })).toBeNull()
    expect(evidenceAction({ type: 'link', url: 'https://example.test/report' })).toEqual({
      href: 'https://example.test/report', label: '打开来源链接',
    })
  })
})
