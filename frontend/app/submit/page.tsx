'use client'

import Link from 'next/link'
import { useState, type FormEvent } from 'react'
import { useAuth } from '@/app/providers'
import { DemoCaptcha } from '@/components/auth/demo-captcha'
import { apiRequest } from '@/lib/api'
import { ApiError } from '@/lib/api-error'
import { Alert } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Panel } from '@/components/ui/panel'
import { LoadingState } from '@/components/ui/loading-state'

type Account = { platform: string; username: string; account_id: string }
type EventInput = {
  title: string
  details: string
  severity: number
  text_evidence: string
  link_url: string
  link_title: string
  files: File[]
}

export default function SubmitPage() {
  const { status, user } = useAuth()
  const [accounts, setAccounts] = useState<Account[]>([{ platform: 'qq', username: '', account_id: '' }])
  const [events, setEvents] = useState<EventInput[]>([
    { title: '', details: '', severity: 1, text_evidence: '', link_url: '', link_title: '', files: [] },
  ])
  const [displayName, setDisplayName] = useState('')
  const [verificationCode, setVerificationCode] = useState('')
  const [captchaToken, setCaptchaToken] = useState('')
  const [error, setError] = useState('')
  const [result, setResult] = useState<{ public_id: string } | null>(null)
  const [loading, setLoading] = useState(false)

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

  async function publish(e: FormEvent) {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      const textEvidence = events
        .map((event, event_index) => ({
          event_index,
          title: '提交文本证据',
          text: event.text_evidence.trim(),
        }))
        .filter((item) => item.text.length > 0)

      const linkEvidence = events
        .map((event, event_index) => ({
          event_index,
          title: event.link_title.trim() || '链接证据',
          description: '',
          url: event.link_url.trim(),
        }))
        .filter((item) => item.url.length > 0)

      const fileEvidenceMeta: Array<{ event_index: number; title: string; filename: string; field: string }> = []
      const form = new FormData()
      events.forEach((event, eventIndex) => {
        event.files.forEach((file, fileIndex) => {
          const field = `file_${eventIndex}_${fileIndex}`
          form.append(field, file, file.name)
          fileEvidenceMeta.push({
            event_index: eventIndex,
            title: file.name,
            filename: file.name,
            field,
          })
        })
      })

      const payload = {
        display_name: displayName,
        accounts,
        events: events.map(({ title, details, severity }) => ({ title, details, severity })),
        text_evidence: textEvidence,
        link_evidence: linkEvidence,
        file_evidence: fileEvidenceMeta,
        verification_code: verificationCode,
        captcha_token: captchaToken,
      }

      let data: { public_id: string }
      if (fileEvidenceMeta.length > 0) {
        form.append('payload', JSON.stringify(payload))
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
      setError(err instanceof ApiError ? err.message : '发布失败')
    } finally {
      setLoading(false)
    }
  }

  async function sendCode() {
    try {
      const email = user?.email
      if (!email) {
        setError('无法读取当前用户邮箱')
        return
      }
      await apiRequest('/api/auth/send-verification-code', {
        auth: true,
        json: { email, purpose: 'submission' },
      })
      setError('')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : '发送验证码失败')
    }
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
      {error && (
        <div className="mt-4">
          <Alert>{error}</Alert>
        </div>
      )}
      <form onSubmit={publish} className="mt-6 space-y-6">
        <Panel>
          <h2 className="font-semibold">1. 对象</h2>
          <label htmlFor="display-name" className="mt-3 block text-sm">
            通用名
          </label>
          <input
            id="display-name"
            className="mt-1 min-h-touch w-full rounded border border-border p-2"
            placeholder="留空时使用第一条账号用户名"
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
          />
        </Panel>
        <Panel>
          <h2 className="font-semibold">2. 账号</h2>
          {accounts.map((a, i) => (
            <div className="mt-3 flex flex-col gap-2 md:flex-row" key={i}>
              <input
                className="min-h-touch rounded border border-border p-2 md:w-28"
                value={a.platform}
                onChange={(e) => setAccounts(accounts.map((x, j) => (j === i ? { ...x, platform: e.target.value } : x)))}
                aria-label={`平台 ${i + 1}`}
              />
              <input
                className="min-h-touch flex-1 rounded border border-border p-2"
                placeholder="用户名"
                value={a.username}
                onChange={(e) => setAccounts(accounts.map((x, j) => (j === i ? { ...x, username: e.target.value } : x)))}
              />
              <input
                className="min-h-touch flex-1 rounded border border-border p-2"
                placeholder="账号 ID"
                value={a.account_id}
                onChange={(e) => setAccounts(accounts.map((x, j) => (j === i ? { ...x, account_id: e.target.value } : x)))}
              />
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
        </Panel>
        <Panel>
          <h2 className="font-semibold">3. 事件与证据</h2>
          {events.map((v, i) => (
            <div className="mt-3 space-y-2 border-t border-border pt-3 first:border-t-0 first:pt-0" key={i}>
              <input
                required
                className="min-h-touch w-full rounded border border-border p-2"
                placeholder="事件标题"
                value={v.title}
                onChange={(e) => setEvents(events.map((x, j) => (j === i ? { ...x, title: e.target.value } : x)))}
              />
              <textarea
                required
                className="w-full rounded border border-border p-2"
                placeholder="事件详情"
                value={v.details}
                onChange={(e) => setEvents(events.map((x, j) => (j === i ? { ...x, details: e.target.value } : x)))}
              />
              <textarea
                className="w-full rounded border border-border p-2"
                placeholder="可选文本证据（保存为 .txt，最大 200KiB）"
                value={v.text_evidence}
                onChange={(e) => setEvents(events.map((x, j) => (j === i ? { ...x, text_evidence: e.target.value } : x)))}
              />
              <div className="grid gap-2 md:grid-cols-2">
                <input
                  className="min-h-touch rounded border border-border p-2"
                  placeholder="可选链接标题"
                  value={v.link_title}
                  onChange={(e) => setEvents(events.map((x, j) => (j === i ? { ...x, link_title: e.target.value } : x)))}
                />
                <input
                  className="min-h-touch rounded border border-border p-2"
                  placeholder="可选链接 URL (https://...)"
                  value={v.link_url}
                  onChange={(e) => setEvents(events.map((x, j) => (j === i ? { ...x, link_url: e.target.value } : x)))}
                />
              </div>
              <input
                type="file"
                multiple
                className="block w-full text-sm"
                onChange={(e) =>
                  setEvents(
                    events.map((x, j) => (j === i ? { ...x, files: e.target.files ? Array.from(e.target.files) : [] } : x)),
                  )
                }
              />
              {v.files.length > 0 && <p className="text-xs text-muted">已选 {v.files.length} 个文件</p>}
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
        </Panel>
        <Panel>
          <h2 className="font-semibold">4. 验证与发布</h2>
          <div className="mt-3 flex flex-col gap-2 sm:flex-row">
            <input
              className="min-h-touch min-w-0 flex-1 rounded border border-border p-2"
              placeholder="邮箱验证码（开发环境 123456）"
              value={verificationCode}
              onChange={(e) => setVerificationCode(e.target.value)}
            />
            <Button type="button" variant="secondary" onClick={sendCode}>
              发送验证码
            </Button>
          </div>
          <div className="mt-3">
            <DemoCaptcha value={captchaToken} onChange={setCaptchaToken} purpose="submission" />
          </div>
          <Button type="submit" variant="danger" className="mt-4 w-full" disabled={loading}>
            {loading ? '发布中...' : '确认并公开发布'}
          </Button>
        </Panel>
      </form>
    </main>
  )
}
