'use client'

import Link from 'next/link'
import { useEffect, useState, type FormEvent } from 'react'
import { useAuth } from '@/app/providers'
import { DemoCaptcha } from '@/components/auth/demo-captcha'
import { apiRequest } from '@/lib/api'
import { ApiError } from '@/lib/api-error'
import { settingBool } from '@/lib/settings'
import type { PublicSettings } from '@/lib/types'
import { Alert } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input, Textarea, Select } from '@/components/ui/input'
import { Panel } from '@/components/ui/panel'
import { LoadingState } from '@/components/ui/loading-state'
import {
  type PublishDraft,
  type PublishEventDraft,
  type PublishRequirements,
  type PublishSectionId,
  type PublishIssue,
  validatePublishDraft,
  getPublishSectionCompletion,
  mapPublishErrorToSection,
  buildPublishPayload,
  listPublishFiles,
} from './publishing'

// Severity labels in Chinese — matches backend textual severity mapping.
const SEVERITY_LABELS: Record<number, string> = {
  1: '1 — 轻微',
  2: '2 — 一般',
  3: '3 — 较严重',
  4: '4 — 严重',
  5: '5 — 极严重',
}

// Local account input shape (plain object, matches PublishAccountInput).
type AccountRow = { platform: string; username: string; account_id: string }

// Focus a section panel by its stable DOM id.
function focusSection(section: PublishSectionId) {
  requestAnimationFrame(() => {
    document.getElementById(`publish-section-${section}`)?.focus()
  })
}

export default function SubmitPage() {
  const { status, user } = useAuth()

  // ── Settings / verification requirements ──────────────────────────
  const [settings, setSettings] = useState<PublicSettings | null>(null)
  const [settingsWarning, setSettingsWarning] = useState('')

  useEffect(() => {
    apiRequest<PublicSettings>('/api/settings/public')
      .then(setSettings)
      .catch(() => {
        setSettings({
          'security.email_verification': true,
          'security.captcha_enabled': true,
        })
        setSettingsWarning('无法读取验证配置，已采用保守验证要求。')
      })
  }, [])

  const requirements: PublishRequirements = {
    emailVerification: settings ? settingBool(settings, 'security.email_verification', true) : false,
    captcha: settings ? settingBool(settings, 'security.captcha_enabled', true) : false,
  }
  // While settings haven't loaded, don't enforce verification.
  const requirementsReady = settings !== null

  // ── Draft state ──────────────────────────────────────────────────
  const [accounts, setAccounts] = useState<AccountRow[]>([{ platform: 'qq', username: '', account_id: '' }])
  const [events, setEvents] = useState<PublishEventDraft[]>([
    { title: '', details: '', severity: 1, text_evidence: '', link_url: '', link_title: '', files: [] },
  ])
  const [displayName, setDisplayName] = useState('')
  const [verificationCode, setVerificationCode] = useState('')
  const [captchaToken, setCaptchaToken] = useState('')
  const [loading, setLoading] = useState(false)
  const [result, setResult] = useState<{ public_id: string } | null>(null)

  // Validation / errors
  const [globalError, setGlobalError] = useState('')
  const [captchaError, setCaptchaError] = useState('')
  const [codeError, setCodeError] = useState('')
  const [validationIssues, setValidationIssues] = useState<PublishIssue[]>([])
  const [submitFailed, setSubmitFailed] = useState(false) // track whether last submit attempt failed

  function clearErrors() {
    setGlobalError('')
    setCaptchaError('')
    setCodeError('')
    setValidationIssues([])
    setSubmitFailed(false)
  }

  function draft(): PublishDraft {
    return { displayName, accounts, events, verificationCode, captchaToken }
  }

  // ── Section completion (recomputed every render) ─────────────────
  const completion = requirementsReady
    ? getPublishSectionCompletion(draft(), requirements)
    : { accounts: false, facts: false, 'evidence-confirmation': false }

  function sectionIssues(section: PublishSectionId): PublishIssue[] {
    return validationIssues.filter((i) => i.section === section)
  }

  // Filter global error into the section-level area.
  function getSectionGlobalIssues(section: PublishSectionId): string[] {
    if (!submitFailed || !globalError) return []
    const mapped = mapPublishErrorToSection(new Error(globalError))
    if (mapped === 'evidence-confirmation' && section !== mapped) return []
    if (mapped === 'evidence-confirmation') return [globalError]
    return mapped === section ? [globalError] : []
  }

  const sectionGlobalIssues: Record<PublishSectionId, string[]> = {
    accounts: getSectionGlobalIssues('accounts'),
    facts: getSectionGlobalIssues('facts'),
    'evidence-confirmation': getSectionGlobalIssues('evidence-confirmation'),
  }

  // evidence-confirmation also gets captcha + code errors.
  const evidenceErrors = [
    ...sectionGlobalIssues['evidence-confirmation'],
    ...(captchaError ? [captchaError] : []),
    ...(codeError ? [codeError] : []),
  ]

  const hasEvidenceEvidence = events.some(
      (e) => e.files.length > 0 || e.text_evidence.trim() !== '' || e.link_url.trim() !== '',
    )

  // ── Submit ──────────────────────────────────────────────────────
  async function publish(e: FormEvent) {
    e.preventDefault()
    clearErrors()

    const req = requirementsReady ? requirements : { emailVerification: true, captcha: true }
    const d = draft()
    const issues = validatePublishDraft(d, req)
    if (issues.length > 0) {
      setValidationIssues(issues)
      focusSection(issues[0].section)
      return
    }

    setLoading(true)
    try {
      const payload = buildPublishPayload(d)
      const files = listPublishFiles(d)

      let data: { public_id: string }
      if (files.length > 0) {
        const form = new FormData()
        form.append('payload', JSON.stringify(payload))
        for (const { field, file } of files) {
          form.append(field, file, file.name)
        }
        data = await apiRequest<{ public_id: string }>('/api/subjects/publish', {
          method: 'POST',
          auth: true,
          body: form,
        })
      } else {
        data = await apiRequest<{ public_id: string }>('/api/subjects/publish', {
          auth: true,
          json: payload,
        })
      }
      setResult(data)
    } catch (err) {
      const message = err instanceof ApiError ? err.message : '发布失败'
      setGlobalError(message)
      setSubmitFailed(true)
      const section = mapPublishErrorToSection(err)
      focusSection(section)
    } finally {
      setLoading(false)
    }
  }

  async function sendCode() {
    try {
      const email = user?.email
      if (!email) {
        setCodeError('无法读取当前用户邮箱')
        return
      }
      await apiRequest('/api/auth/send-verification-code', {
        auth: true,
        json: { email, purpose: 'submission' },
      })
      setCodeError('')
    } catch (err) {
      setCodeError(err instanceof ApiError ? err.message : '发送验证码失败')
    }
  }

  function handleCaptchaError(message: string) {
    setCaptchaError(message)
    focusSection('evidence-confirmation')
  }

  // ── Rendering ─────────────────────────────────────────────────────

  if (status === 'loading') return <LoadingState message="正在确认登录状态..." />

  if (status !== 'authenticated') {
    return (
      <main className="py-8">
        <h1 className="text-3xl font-bold">提交对象与事件</h1>
        <p className="mt-3 text-muted">
          登录并完成邮箱验证后可发布对象、账号和事件。提交后默认公开，错误或恶意内容可被申诉、修正、撤销。
        </p>
        <Panel className="mt-6 border-dashed">
          <p className="text-muted">账号、事件、证据上传和发布操作在登录后可用。</p>
          <Link href="/login?next=/submit" className="mt-4 inline-block">
            <Button>登录后提交</Button>
          </Link>
        </Panel>
      </main>
    )
  }

  if (result) {
    return (
      <main className="py-8">
        <h1 className="text-2xl font-bold">对象与事件已发布</h1>
        <p className="mt-2">
          公开对象 ID：<code>{result.public_id}</code>
        </p>
        <Link href={`/subjects/${result.public_id}`} className="mt-4 inline-block text-primary hover:underline">
          查看对象档案
        </Link>
      </main>
    )
  }

  return (
    <main className="mx-auto max-w-3xl py-8">
      <h1 className="text-3xl font-bold">提交对象与事件</h1>
      <p className="mt-2 text-muted">
        分段填写对象、账号、事件、文本/链接/文件证据与验证信息。开发环境邮箱验证码为 123456。
      </p>

      {settingsWarning && (
        <div className="mt-4">
          <Alert tone="warning">{settingsWarning}</Alert>
        </div>
      )}

      <form noValidate onSubmit={publish} className="mt-6 space-y-6">
        {/* ── 1. 对象账号 ──────────────────────────────────────────── */}
        <Panel id="publish-section-accounts" tabIndex={-1}>
          <div className="flex items-center justify-between">
            <h2 className="font-semibold">1 对象账号</h2>
            <Badge tone={completion.accounts ? 'success' : 'neutral'}>
              {completion.accounts ? '已完成' : '待补充'}
            </Badge>
          </div>
          <p className="mt-1 text-sm text-muted">
            填写对象通用名和关联的在线账号。至少需要一个可分辨的账号（平台 + 用户名或账号 ID）。
          </p>

          <label htmlFor="publish-display-name" className="mt-3 block text-sm font-medium">
            通用名
          </label>
          <Input
            id="publish-display-name"
            placeholder="留空时使用第一条账号用户名"
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
          />

          {accounts.map((a, i) => (
            <div className="mt-3 flex flex-col gap-2 md:flex-row" key={i}>
              <div className="md:w-28">
                <label htmlFor={`publish-account-platform-${i}`} className="block text-sm font-medium">
                  平台 {i + 1}
                </label>
                <Input
                  id={`publish-account-platform-${i}`}
                  value={a.platform}
                  onChange={(e) =>
                    setAccounts(accounts.map((x, j) => (j === i ? { ...x, platform: e.target.value } : x)))
                  }
                  aria-label={`平台 ${i + 1}`}
                />
              </div>
              <div className="flex-1">
                <label htmlFor={`publish-account-username-${i}`} className="block text-sm font-medium">
                  用户名 {i + 1}
                </label>
                <Input
                  id={`publish-account-username-${i}`}
                  placeholder="用户名"
                  value={a.username}
                  onChange={(e) =>
                    setAccounts(accounts.map((x, j) => (j === i ? { ...x, username: e.target.value } : x)))
                  }
                />
              </div>
              <div className="flex-1">
                <label htmlFor={`publish-account-id-${i}`} className="block text-sm font-medium">
                  账号 ID {i + 1}
                </label>
                <Input
                  id={`publish-account-id-${i}`}
                  placeholder="账号 ID"
                  value={a.account_id}
                  onChange={(e) =>
                    setAccounts(accounts.map((x, j) => (j === i ? { ...x, account_id: e.target.value } : x)))
                  }
                />
              </div>
            </div>
          ))}
          <Button
            type="button"
            variant="ghost"
            className="mt-3"
            onClick={() => setAccounts([...accounts, { platform: 'custom', username: '', account_id: '' }])}
          >
            添加账号
          </Button>

          {/* Account-level issues */}
          {sectionGlobalIssues.accounts.map((msg, idx) => (
            <Alert key={`g-${idx}`} tone="danger">
              {msg}
            </Alert>
          ))}
          {sectionIssues('accounts').map((issue, idx) => (
            <Alert key={`a-${idx}`} tone="danger">
              {issue.message}
            </Alert>
          ))}
        </Panel>

        {/* ── 2. 事件事实 ──────────────────────────────────────────── */}
        <Panel id="publish-section-facts" tabIndex={-1}>
          <div className="flex items-center justify-between">
            <h2 className="font-semibold">2 事件事实</h2>
            <Badge tone={completion.facts ? 'success' : 'neutral'}>
              {completion.facts ? '已完成' : '待补充'}
            </Badge>
          </div>
          <p className="mt-1 text-sm text-muted">
            记录该对象在平台上的特定事件，每个事件包含标题、详情和严重级别。至少需要一个事件。
          </p>

          {events.map((v, i) => (
            <div className="mt-3 space-y-2 border-t border-border pt-3 first:border-t-0 first:pt-0" key={i}>
              <label htmlFor={`publish-event-title-${i}`} className="block text-sm font-medium">
                事件标题 {i + 1}
              </label>
              <Input
                id={`publish-event-title-${i}`}
                required
                placeholder="事件标题"
                value={v.title}
                onChange={(e) =>
                  setEvents(events.map((x, j) => (j === i ? { ...x, title: e.target.value } : x)))
                }
              />
              <label htmlFor={`publish-event-details-${i}`} className="block text-sm font-medium">
                事件详情 {i + 1}
              </label>
              <Textarea
                id={`publish-event-details-${i}`}
                required
                placeholder="事件详情"
                value={v.details}
                onChange={(e) =>
                  setEvents(events.map((x, j) => (j === i ? { ...x, details: e.target.value } : x)))
                }
              />
              <div>
                <label htmlFor={`publish-severity-${i}`} className="block text-sm font-medium">
                  严重级别 {i + 1}
                </label>
                <Select
                  id={`publish-severity-${i}`}
                  value={v.severity}
                  onChange={(e) =>
                    setEvents(events.map((x, j) => (j === i ? { ...x, severity: Number(e.target.value) } : x)))
                  }
                >
                  {Array.from({ length: 5 }, (_, n) => n + 1).map((n) => (
                    <option key={n} value={n}>
                      {SEVERITY_LABELS[n]}
                    </option>
                  ))}
                </Select>
              </div>
            </div>
          ))}
          <Button
            type="button"
            variant="ghost"
            className="mt-3"
            onClick={() =>
              setEvents([
                ...events,
                { title: '', details: '', severity: 1, text_evidence: '', link_url: '', link_title: '', files: [] },
              ])
            }
          >
            添加事件
          </Button>

          {/* Fact-level issues */}
          {sectionGlobalIssues.facts.map((msg, idx) => (
            <Alert key={`g-${idx}`} tone="danger">
              {msg}
            </Alert>
          ))}
          {sectionIssues('facts').map((issue, idx) => (
            <Alert key={`f-${idx}`} tone="danger">
              {issue.message}
            </Alert>
          ))}
        </Panel>

        {/* ── 3. 证据与确认 ────────────────────────────────────────── */}
        <Panel id="publish-section-evidence-confirmation" tabIndex={-1}>
          <div className="flex items-center justify-between">
            <h2 className="font-semibold">3 证据与确认</h2>
            <Badge tone={completion['evidence-confirmation'] ? 'success' : 'neutral'}>
              {completion['evidence-confirmation'] ? '已完成' : '待补充'}
            </Badge>
          </div>
          <p className="mt-1 text-sm text-muted">
            按事件分组提交文本、链接或文件证据。证据非必填，但可核验材料有助于访客判断记录可信度。
          </p>

          {/* Evidence per event */}
          {events.map((v, i) => (
            <div className="mt-3 border-t border-border pt-3 first:border-t-0 first:pt-0" key={i}>
              <h3 className="text-sm font-medium">
                事件 {i + 1}：{v.title.trim() || '（未标题）'}
              </h3>

              <label htmlFor={`publish-text-${i}`} className="mt-2 block text-sm">
                文本证据 {i + 1}
              </label>
              <Textarea
                id={`publish-text-${i}`}
                placeholder="可选文本证据（保存为 .txt，最大 200 KiB）"
                value={v.text_evidence}
                onChange={(e) =>
                  setEvents(events.map((x, j) => (j === i ? { ...x, text_evidence: e.target.value } : x)))
                }
              />

              <div className="mt-2 grid gap-2 md:grid-cols-2">
                <div>
                  <label htmlFor={`link-title-${i}`} className="block text-sm">
                    链接标题 {i + 1}
                  </label>
                  <Input
                    id={`link-title-${i}`}
                    placeholder="可选链接标题"
                    value={v.link_title}
                    onChange={(e) =>
                      setEvents(events.map((x, j) => (j === i ? { ...x, link_title: e.target.value } : x)))
                    }
                  />
                </div>
                <div>
                  <label htmlFor={`link-url-${i}`} className="block text-sm">
                    链接 URL {i + 1}
                  </label>
                  <Input
                    id={`link-url-${i}`}
                    placeholder="可选链接 URL (https://...)"
                    value={v.link_url}
                    onChange={(e) =>
                      setEvents(events.map((x, j) => (j === i ? { ...x, link_url: e.target.value } : x)))
                    }
                  />
                </div>
              </div>

              <div className="mt-2">
                <label htmlFor={`file-${i}`} className="block text-sm">
                  文件证据 {i + 1}
                </label>
                <Input
                  id={`file-${i}`}
                  type="file"
                  multiple
                  className="block"
                  onChange={(e) =>
                    setEvents(
                      events.map((x, j) =>
                        j === i ? { ...x, files: e.target.files ? Array.from(e.target.files) : [] } : x,
                      ),
                    )
                  }
                />
                {v.files.length > 0 && <p className="text-xs text-muted">已选 {v.files.length} 个文件</p>}
              </div>
            </div>
          ))}

          {/* No-evidence recommendation */}
          {!hasEvidenceEvidence && (
            <p className="mt-3 text-sm text-muted">
              当前未附证据；系统允许继续发布，但可核验材料可以有助于访客理解记录。
            </p>
          )}

          {/* Verification controls — only when settings demand them */}
          {(requirementsReady && requirements.emailVerification) && (
            <div className="mt-3 flex flex-col gap-2 sm:flex-row">
              <div className="min-w-0 flex-1">
                <label htmlFor="publish-verification-code" className="block text-sm font-medium">
                  邮箱验证码
                </label>
                <Input
                  id="publish-verification-code"
                  placeholder="邮箱验证码（开发环境 123456）"
                  value={verificationCode}
                  onChange={(e) => setVerificationCode(e.target.value)}
                />
              </div>
              <Button type="button" variant="secondary" onClick={sendCode}>
                发送验证码
              </Button>
            </div>
          )}
          {codeError && <p className="mt-1 text-sm text-danger">{codeError}</p>}

          {(requirementsReady && requirements.captcha) && (
            <div className="mt-3">
              <DemoCaptcha
                value={captchaToken}
                onChange={setCaptchaToken}
                onError={handleCaptchaError}
                purpose="submission"
              />
            </div>
          )}
          {captchaError && <p className="mt-1 text-sm text-danger">{captchaError}</p>}

          {/* Section-level issues */}
          {evidenceErrors.map((msg, idx) => (
            <Alert key={`ge-${idx}`} tone="danger">
              {msg}
            </Alert>
          ))}
          {sectionIssues('evidence-confirmation').map((issue, idx) => (
            <Alert key={`ec-${idx}`} tone="danger">
              {issue.message}
            </Alert>
          ))}

          {/* Publish action */}
          <p className="mt-3 text-sm text-muted">发布后默认公开，错误或恶意内容可被申诉、修正、撤销。</p>
          <Button type="submit" variant="danger" className="mt-4 w-full" disabled={loading}>
            {loading ? '发布中...' : '确认并公开发布'}
          </Button>
        </Panel>
      </form>
    </main>
  )
}