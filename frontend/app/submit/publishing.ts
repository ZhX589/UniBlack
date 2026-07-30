// Page-scoped pure validation / completion / payload helpers for `/submit`.
//
// Implements the Task 4 "Publishing rules":
// - Validate the in-memory PublishDraft against the backend's publish rules so
//   the UI can show per-section issues without first making a server round-trip.
// - Derive per-section completion flags (red → green only when no issue targets
//   the section; purely a function of the current form values, no persistence).
// - Map a server/transport ApiError into the section that should scroll into
//   focus so the user's typed input is never discarded on failure.
// - Build the multipart-publish JSON payload plus the file manifest that ships
//   through FormData alongside the JSON body.
//
// NOTE: this module is intentionally side-effect free; it does not call the
// API and does not touch any store/state. Inputs are treated as immutable.

import type {
  PublishAccountInput,
  PublishEventInput,
  PublishFileEvidenceInput,
  PublishLinkEvidenceInput,
  PublishSubjectPayload,
  PublishTextEvidenceInput,
} from '@/lib/types'

export type PublishSectionId = 'accounts' | 'facts' | 'evidence-confirmation'

/** Verification requirements the runtime expects from the submitter. */
export type PublishRequirements = {
  emailVerification: boolean
  captcha: boolean
}

/**
 * Per-event form draft. The file list is intentionally part of the event row so
 * the section UI can render evidence counts per event without a separate map.
 */
export type PublishEventDraft = {
  title: string
  details: string
  severity: number
  text_evidence: string
  link_url: string
  link_title: string
  files: File[]
}

/** Whole-form draft backing the three-task publish page. */
export type PublishDraft = {
  displayName: string
  accounts: PublishAccountInput[]
  events: PublishEventDraft[]
  verificationCode: string
  captchaToken: string
}

/** A single validation problem scoped to a section and (optionally) a field. */
export type PublishIssue = {
  section: PublishSectionId
  field?: string
  message: string
}

// --- Backend-mirrored constants --------------------------------------------
// These mirror backend/internal/service/evidence.go and event.go so the
// frontend produces the same outcome without a server round-trip.
const MAX_TEXT_EVIDENCE_BYTES = 200 * 1024 // 200 KiB
const MAX_FILE_BYTES = 20 << 20 // 20 MiB
const ALLOWED_FILE_EXTENSIONS = new Set([
  '.png', '.jpg', '.jpeg', '.gif', '.webp',
  '.pdf', '.txt', '.bin', '.zip', '.webm', '.mp4', '.md',
])
// Default link / text evidence titles used when the publisher leaves the field
// blank — matches `app/submit/page.tsx` historical defaults.
const DEFAULT_LINK_TITLE = '链接证据'
const DEFAULT_TEXT_TITLE = '提交文本证据'

/** Normalize an account row the way the backend does: trim + lower platform. */
function normalizePlatform(input: string): string {
  return input.trim().toLowerCase()
}

/** Dedup key mirrors backend domain.AccountDedupKey (account_id preferred). */
function accountDedupKey(platform: string, username: string, accountId: string): string {
  const p = platform.trim().toLowerCase()
  if (accountId.trim() !== '') {
    return `${p}:${accountId.trim().toLowerCase()}`
  }
  return `${p}:${username.trim().toLowerCase()}`
}

/** Trim and return UTF-8 byte length of `text` (matches `len(text)` in Go). */
function utf8ByteLength(text: string): number {
  return new TextEncoder().encode(text).byteLength
}

/**
 * Validate the whole draft, returning every issue found (no short-circuit).
 *
 * Order of checks within a section does not matter for completion — the UI
 * surfaces all issues together — but field identifiers are kept stable so the
 * Task 5 UI can attach messages to specific inputs.
 */
export function validatePublishDraft(draft: PublishDraft, requirements: PublishRequirements): PublishIssue[] {
  const issues: PublishIssue[] = []

  // --- accounts section -----------------------------------------------------
  const trimmedDisplay = draft.displayName.trim()
  const firstResolvableUsername = draft.accounts.find((a) => a.username.trim() !== '')
  if (trimmedDisplay === '' && !firstResolvableUsername) {
    issues.push({
      section: 'accounts',
      field: 'display_name',
      message: '请填写通用名，或在第一行账号填入用户名作为通用名来源。',
    })
  }

  if (draft.accounts.length === 0) {
    issues.push({
      section: 'accounts',
      field: 'accounts',
      message: '至少需要一行账号。',
    })
  }

  const seen = new Set<string>()
  draft.accounts.forEach((account, index) => {
    const platform = normalizePlatform(account.platform)
    const username = account.username.trim()
    const accountId = account.account_id.trim()
    if (platform === '') {
      issues.push({
        section: 'accounts',
        field: `accounts.${index}.platform`,
        message: '账号平台不能为空。',
      })
    }
    if (username === '' && accountId === '') {
      issues.push({
        section: 'accounts',
        field: `accounts.${index}`,
        message: '用户名和账号 ID 至少填写一项。',
      })
    }
    // Dedup check only when the row is otherwise structurally valid.
    if (platform !== '' && (username !== '' || accountId !== '')) {
      const key = accountDedupKey(platform, username, accountId)
      if (seen.has(key)) {
        issues.push({
          section: 'accounts',
          field: `accounts.${index}`,
          message: '该账号与已有账号重复。',
        })
      } else {
        seen.add(key)
      }
    }
  })

  // --- facts section --------------------------------------------------------
  if (draft.events.length === 0) {
    issues.push({
      section: 'facts',
      field: 'events',
      message: '至少需要一个事件。',
    })
  }
  draft.events.forEach((event, index) => {
    if (event.title.trim() === '') {
      issues.push({
        section: 'facts',
        field: `events.${index}.title`,
        message: '事件标题不能为空。',
      })
    }
    if (event.details.trim() === '') {
      issues.push({
        section: 'facts',
        field: `events.${index}.details`,
        message: '事件详情不能为空。',
      })
    }
    const severity = Number(event.severity)
    if (!Number.isInteger(severity) || severity < 1 || severity > 5) {
      issues.push({
        section: 'facts',
        field: `events.${index}.severity`,
        message: '严重级别必须为 1-5 的整数。',
      })
    }
  })

  // --- evidence-confirmation section ---------------------------------------
  // Evidence is recommended, not required. The only evidence-side failures are
  // partially filled links, non-http(s) URLs, oversized text, oversized files,
  // and disallowed file extensions.
  draft.events.forEach((event, index) => {
    const linkUrl = event.link_url.trim()
    const linkTitle = event.link_title.trim()
    const linkTouched = linkUrl !== '' || linkTitle !== ''
    if (linkTouched) {
      if (linkTitle === '') {
        issues.push({
          section: 'evidence-confirmation',
          field: `events.${index}.link_title`,
          message: '链接证据必须同时填写标题与 URL。',
        })
      }
      if (linkUrl === '' || !isAbsoluteHttpURL(linkUrl)) {
        issues.push({
          section: 'evidence-confirmation',
          field: `events.${index}.link_url`,
          message: '链接 URL 必须为带主机的 http(s) 绝对地址。',
        })
      }
    }

    const text = event.text_evidence
    if (text.trim() !== '') {
      const byteLen = utf8ByteLength(text)
      if (byteLen > MAX_TEXT_EVIDENCE_BYTES) {
        issues.push({
          section: 'evidence-confirmation',
          field: `events.${index}.text_evidence`,
          message: `文本证据不得超过 ${MAX_TEXT_EVIDENCE_BYTES} 字节（UTF-8）。`,
        })
      }
    }

    event.files.forEach((file, fileIndex) => {
      if (file.size > MAX_FILE_BYTES) {
        issues.push({
          section: 'evidence-confirmation',
          field: `events.${index}.files`,
          message: `文件 "${file.name}" 超过 20 MiB 上限。`,
        })
      }
      const ext = fileExtension(file.name)
      if (ext === '' || !ALLOWED_FILE_EXTENSIONS.has(ext)) {
        issues.push({
          section: 'evidence-confirmation',
          field: `events.${index}.files`,
          message: `文件 "${file.name}" 的扩展名不被允许。`,
        })
      }
    })
  })

  // Verification requirements (email + captcha). These come from runtime
  // settings (captcha disabled at build time / or session-level email already
  // verified). See backend/internal/service/auth.go for the source of truth.
  if (requirements.emailVerification && draft.verificationCode.trim() === '') {
    issues.push({
      section: 'evidence-confirmation',
      field: 'verification_code',
      message: '请填写邮箱验证码。',
    })
  }
  if (requirements.captcha && draft.captchaToken.trim() === '') {
    issues.push({
      section: 'evidence-confirmation',
      field: 'captcha_token',
      message: '请完成人机验证。',
    })
  }

  return issues
}

/**
 * Return a section → completeness map derived from {@link validatePublishDraft}.
 * A section is "complete" iff no detected issue targets it.
 *
 * Completion is recomputed on every render from the current form values; it is
 * NOT persisted and is purely a view-layer affordance. (See [S3] Non-Goals.)
 */
export function getPublishSectionCompletion(
  draft: PublishDraft,
  requirements: PublishRequirements,
): Record<PublishSectionId, boolean> {
  const issues = validatePublishDraft(draft, requirements)
  return {
    accounts: !issues.some((i) => i.section === 'accounts'),
    facts: !issues.some((i) => i.section === 'facts'),
    'evidence-confirmation': !issues.some((i) => i.section === 'evidence-confirmation'),
  }
}

/**
 * Map an API/transport error to the section whose state the user should review
 * so that any partial input is preserved. The function never throws; it always
 * returns a valid section. The classification reflects what the backend reports
 * today — adjust only if backend message keys change.
 *
 * Keyword precedence (first match wins):
 *   account / duplicate / display_name              → accounts
 *   evidence / file / verification / captcha /
 *     submission                                    → evidence-confirmation
 *   event / title / details / severity             → facts
 *   (no match) / unknown / network                 → evidence-confirmation
 *
 * Evidence-related keywords are checked BEFORE facts keywords because the
 * backend phrase "invalid evidence event index" (filed by the file/text/link
 * evidence validators) contains the bare word "event" yet describes an
 * evidence error. Giving `evidence` priority over `event` keeps that string
 * — and any "evidence requires title" message — on the evidence-confirmation
 * section so the user's typed title/evidence stays intact.
 */
export function mapPublishErrorToSection(error: unknown): PublishSectionId {
  const text = extractMessageText(error).toLowerCase()
  if (text.includes('account') || text.includes('duplicate') || text.includes('display_name')) {
    return 'accounts'
  }
  if (
    text.includes('evidence') ||
    text.includes('file') ||
    text.includes('verification') ||
    text.includes('captcha') ||
    text.includes('submission')
  ) {
    return 'evidence-confirmation'
  }
  if (text.includes('event') || text.includes('title') || text.includes('details') || text.includes('severity')) {
    return 'facts'
  }
  // Default bucket — unknown, network, and everything else — points the user
  // at the evidence-and-confirmation panel where the submit/verification
  // affordances live. The draft's already-typed values are never discarded.
  return 'evidence-confirmation'
}

function extractMessageText(error: unknown): string {
  if (error === null || error === undefined) return ''
  if (typeof error === 'string') return error
  if (error instanceof Error) return error.message
  // Generic object with `.message` (e.g. ApiError): use it. Fallback to '' so
  // classifiers fall through to the default section rather than throwing.
  if (typeof error === 'object' && error !== null) {
    const obj = error as { message?: unknown }
    if (typeof obj.message === 'string') return obj.message
  }
  return ''
}

function fileExtension(name: string): string {
  const i = name.lastIndexOf('.')
  if (i === -1) return ''
  return name.slice(i).toLowerCase()
}

/**
 * Matches backend domain.ValidateAbsoluteHTTPURL: absolute http(s) URL with a
 * host. Requires the strict `scheme://` shape up front so the URL constructor
 * does not normalise protocol-relative or path-only inputs into accepted URLs.
 */
function isAbsoluteHttpURL(raw: string): boolean {
  try {
    const trimmed = raw.trim()
    if (!/^https?:\/\//i.test(trimmed)) return false
    const url = new URL(trimmed)
    if (url.host === '') return false
    const scheme = url.protocol.toLowerCase()
    return scheme === 'http:' || scheme === 'https:'
  } catch {
    return false
  }
}

/**
 * Build the JSON payload accepted by `POST /api/subjects/publish`.
 *
 * `display_name` is preserved verbatim (the server trims and resolves the
 * fallback per ResolveDisplayName). Evidence rows are trimmed and filtered so
 * the server never receives empty rows. File objects are excluded from the
 * JSON payload — the multipart handler receives them via FormData, addressed
 * by the `field` key returned by {@link listPublishFiles}.
 */
export function buildPublishPayload(draft: PublishDraft): PublishSubjectPayload {
  const textEvidence: PublishTextEvidenceInput[] = []
  draft.events.forEach((event, eventIndex) => {
    const text = event.text_evidence.trim()
    if (text !== '') {
      textEvidence.push({
        event_index: eventIndex,
        title: DEFAULT_TEXT_TITLE,
        text,
      })
    }
  })

  const linkEvidence: PublishLinkEvidenceInput[] = []
  draft.events.forEach((event, eventIndex) => {
    const url = event.link_url.trim()
    if (url !== '') {
      const title = event.link_title.trim() || DEFAULT_LINK_TITLE
      linkEvidence.push({
        event_index: eventIndex,
        title,
        description: '',
        url,
      })
    }
  })

  const fileEvidence: PublishFileEvidenceInput[] = []
  draft.events.forEach((event, eventIndex) => {
    event.files.forEach((file, fileIndex) => {
      const field = `file_${eventIndex}_${fileIndex}`
      fileEvidence.push({
        event_index: eventIndex,
        title: file.name,
        filename: file.name,
        field,
      })
    })
  })

  const events: PublishEventInput[] = draft.events.map((event) => ({
    title: event.title,
    details: event.details,
    severity: event.severity,
  }))

  return {
    display_name: draft.displayName,
    accounts: draft.accounts.map((a) => ({
      platform: a.platform,
      username: a.username,
      account_id: a.account_id,
    })),
    events,
    text_evidence: textEvidence,
    link_evidence: linkEvidence,
    file_evidence: fileEvidence,
    verification_code: draft.verificationCode,
    captcha_token: draft.captchaToken,
  }
}

/**
 * Return every file across all event rows paired with the multipart field name
 * that addresses it in the publishing FormData. Order is stable: outer index
 * is eventIndex, inner index is fileIndex within the row.
 */
export function listPublishFiles(draft: PublishDraft): Array<{ field: string; file: File }> {
  const out: Array<{ field: string; file: File }> = []
  draft.events.forEach((event, eventIndex) => {
    event.files.forEach((file, fileIndex) => {
      out.push({ field: `file_${eventIndex}_${fileIndex}`, file })
    })
  })
  return out
}
