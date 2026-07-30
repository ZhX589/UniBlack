# Product Experience Deepening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use compose:subagent (recommended) or compose:execute to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver a traceable public profile, a faster appeal workbench, and a guided single-page publishing flow without replacing UniBlack's existing domain or API contracts.

**Architecture:** Stabilize a small shared presentation layer for Subject/Event/Appeal status language, then implement the three approved product slices in order. Only Event detail receives a backend change: expose the existing Event-to-Evidence relationship as read-only JSON; governance stays frontend-only, and publishing keeps the current JSON/multipart endpoint with page-local validation helpers.

**Tech Stack:** Go 1.25, Echo, GORM read/write, PostgreSQL, React 18, Next.js 14 App Router, TypeScript, Tailwind CSS, Vitest, Testing Library, Playwright.

## Global Constraints

- Project name remains `UniBlack`; never introduce `CloudBan`.
- Preserve Subject → Account → Event → Evidence as the canonical model.
- Keep Case/Submission/Identifier compatibility until 2026-12-31.
- Pages use `lib/api.ts`; they must not read tokens, construct Bearer headers, or handle 401 independently.
- Do not add OAuth, dark mode, OpenAPI generation, public API keys, a global state library, or a query-cache library.
- Do not add database migrations for this work; the Event evidence change exposes an existing relationship only.
- Do not invent evidence URLs, timestamps, submitter identities, reviewer identities, or verification claims absent from API responses.
- Keep publishing single-page and preserve both existing JSON and multipart request shapes.
- Preserve single-item, prompt-based appeal resolution; do not add batch adjudication.
- New status, risk, and severity presentation must include text and must not rely on color alone.
- All visual work must follow the existing frosted-glass tokens and UI primitives; do not add a component library.
- Respect `prefers-reduced-motion`, keyboard focus, semantic headings, labels, and 44px interaction targets.
- Feature implementation must occur on a `feature/` branch, not directly on `main`.
- Do not create commits unless the user explicitly asks for commits.

---

## File Structure

### Shared trustworthy-profile slice

- Create `frontend/lib/trustworthy-profile.ts`: pure presentation mappings shared by public profile pages and the governance queue.
- Create `frontend/lib/trustworthy-profile.test.ts`: exhaustive enum, clamp, account, and evidence-action tests.
- Modify `frontend/lib/types.ts`: add only Evidence metadata already returned by the backend model.
- Modify `backend/internal/models/models.go`: declare the existing Event-to-Evidence GORM/JSON association.
- Modify `backend/internal/repository/event.go`: preload ordered evidence only for Event detail reads.
- Modify `backend/internal/integration/governance_test.go`: prove Event detail returns only its ordered evidence.

### Public profile slice

- Modify `frontend/app/subjects/[id]/page.tsx`: archive-like Subject identity, account cluster, and Event trail.
- Modify `frontend/app/events/[id]/page.tsx`: separate facts, evidence, and correction/appeal layers.
- Create `frontend/e2e/trustworthy-profile.spec.ts`: deterministic mocked public profile scenarios at all configured viewports.

### Governance slice

- Modify `frontend/app/admin/appeals/page.tsx`: responsive workbench header, counts, filters, list, and explicit feedback state.
- Create `frontend/app/admin/appeals/appeal-presentations.test.ts`: queue-specific target and audit-label tests if they do not fit the shared helper.
- Modify `frontend/e2e/appeals-and-compat.spec.ts`: deterministic queue scan, filter, resolve, cancel, and recovery scenarios.

### Guided publishing slice

- Create `frontend/app/submit/publishing.ts`: page-domain draft types, completion, validation, payload, file listing, and error-section mapping.
- Create `frontend/app/submit/publishing.test.ts`: pure validation and request-shape tests.
- Modify `frontend/app/submit/page.tsx`: three-section UI, completion states, severity input, error focus, and protocol-preserving submit.
- Modify `frontend/components/auth/demo-captcha.tsx`: optional error callback and loading disable behavior.
- Create `frontend/components/auth/demo-captcha.test.tsx`: captcha success/failure/loading tests.
- Create `frontend/e2e/guided-publishing.spec.ts`: anonymous, completion, validation, JSON failure, multipart success, and captcha failure scenarios.

### Final evidence

- Modify `docs/frontend-gap-analysis.md`: record only verified status and commands after all gates pass.

---

### Task 1: Shared Trust Presentation And Event Evidence Read

**Covers:** [S1, S2, S3, S4, S5, S8, S9, S10, S11, S12]

**Files:**
- Create: `frontend/lib/trustworthy-profile.ts`
- Create: `frontend/lib/trustworthy-profile.test.ts`
- Modify: `frontend/lib/types.ts:20-37`
- Modify: `backend/internal/models/models.go:95-109`
- Modify: `backend/internal/repository/event.go:29-35`
- Modify: `backend/internal/integration/governance_test.go:28-90`

**Interfaces:**
- Consumes: existing `Account`, `Evidence`, and `Badge` tone values.
- Produces:

```ts
export type PresentationTone = 'neutral' | 'danger' | 'success' | 'warning'

export type StatusPresentation = {
  label: string
  auditLabel: string
  tone: PresentationTone
}

export function subjectStatusPresentation(status: string): StatusPresentation
export function eventStatusPresentation(status: string): StatusPresentation
export function appealStatusPresentation(status: string): StatusPresentation
export function appealOutcomeLabel(outcome?: string | null, audit?: boolean): string | null
export function riskPresentation(level?: number | null): { value: number; label: string }
export function severityPresentation(level?: number | null): { value: number; label: string }
export function evidenceTypeLabel(type: string): string
export function accountDisplayValue(account: Pick<Account, 'username' | 'account_id'>): string | null
export function evidenceAction(evidence: Pick<Evidence, 'type' | 'url'>): { href: string; label: string } | null
```

- Produces backend JSON field `event.evidence`, ordered by `created_at ASC` for Event detail only.

- [ ] **Step 1: Create the feature branch before implementation**

Run from repository root:

```bash
git switch -c feature/product-experience-deepening
```

Expected: current branch becomes `feature/product-experience-deepening`; existing untracked `.mimocode/` and approved docs remain untouched.

- [ ] **Step 2: Write failing presentation-helper tests**

Create `frontend/lib/trustworthy-profile.test.ts` with focused cases such as:

```ts
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
  it('maps known statuses and preserves unknown audit values', () => {
    expect(subjectStatusPresentation('active')).toEqual({
      label: '黑名单记录', auditLabel: '黑名单记录（active）', tone: 'danger',
    })
    expect(eventStatusPresentation('corrected').label).toBe('已更正')
    expect(appealStatusPresentation('escalated').auditLabel).toBe('未知状态（escalated）')
  })

  it('maps appeal outcomes without losing the raw audit value', () => {
    expect(appealOutcomeLabel('malicious_submission', true)).toBe('认定为恶意提交（malicious_submission）')
    expect(appealOutcomeLabel(undefined)).toBeNull()
  })

  it('clamps risk and severity to their display ranges', () => {
    expect(riskPresentation(9).value).toBe(5)
    expect(riskPresentation(-1).value).toBe(0)
    expect(severityPresentation(0).value).toBe(1)
  })

  it('uses username, then account id, and rejects empty accounts', () => {
    expect(accountDisplayValue({ username: 'alice', account_id: '42' })).toBe('alice')
    expect(accountDisplayValue({ username: ' ', account_id: '42' })).toBe('42')
    expect(accountDisplayValue({ username: null, account_id: null })).toBeNull()
  })

  it('only offers evidence actions for real URLs', () => {
    expect(evidenceAction({ type: 'file', url: null })).toBeNull()
    expect(evidenceAction({ type: 'link', url: 'https://example.test/report' })).toEqual({
      href: 'https://example.test/report', label: '打开来源链接',
    })
    expect(evidenceTypeLabel('file')).toBe('文件')
  })
})
```

- [ ] **Step 3: Run the helper test and verify failure**

Run from `frontend/`:

```bash
npm run test:run -- lib/trustworthy-profile.test.ts
```

Expected: FAIL because `lib/trustworthy-profile.ts` does not exist.

- [ ] **Step 4: Implement the minimal pure presentation module**

Create `frontend/lib/trustworthy-profile.ts` with explicit maps and deterministic fallbacks. Use clamping and trimming; do not inspect browser or API state. Unknown values must use `neutral` and preserve the raw value in `auditLabel`.

Required mappings:

```ts
const subjectStatuses = {
  active: ['黑名单记录', 'danger'],
  cleared: ['已清除', 'success'],
  archived: ['已归档', 'neutral'],
} as const

const eventStatuses = {
  published: ['已公开', 'neutral'],
  corrected: ['已更正', 'warning'],
  withdrawn: ['已撤回', 'neutral'],
} as const

const appealStatuses = {
  pending: ['待处理', 'warning'],
  approved: ['已通过', 'success'],
  rejected: ['已驳回', 'neutral'],
} as const
```

Risk labels: `0 未评级`, `1 较低`, `2 低`, `3 中等`, `4 较高`, `5 高`.

Severity labels: `1 轻微`, `2 较低`, `3 中等`, `4 较高`, `5 严重`.

- [ ] **Step 5: Extend the frontend Evidence type only with existing model fields**

Add to `Evidence` in `frontend/lib/types.ts`:

```ts
original_filename?: string | null
file_size?: number | null
mime_type?: string | null
created_at?: string
```

Do not add reviewer, submitter, or generated download fields.

- [ ] **Step 6: Run helper tests and typecheck**

Run from `frontend/`:

```bash
npm run test:run -- lib/trustworthy-profile.test.ts
npm run typecheck
```

Expected: helper tests PASS and TypeScript reports no errors.

- [ ] **Step 7: Write the failing Event evidence integration subtest**

Add a subtest to `TestEventGovernanceIntegration` in `backend/internal/integration/governance_test.go`:

```go
t.Run("event detail reads its evidence in creation order", func(t *testing.T) {
	resetDatabase(t, database)
	ctx := context.Background()
	submitter := createUser(t, database, "event-detail-evidence")
	_, event := createSubjectEvent(t, database, submitter.ID, "event-detail")
	firstTitle := "First source"
	firstURL := "https://example.test/first"
	secondTitle := "Stored file"
	fileName := "proof.pdf"
	first := models.Evidence{EventID: &event.ID, Type: "link", Title: &firstTitle, URL: &firstURL}
	second := models.Evidence{EventID: &event.ID, Type: "file", Title: &secondTitle, OriginalFilename: &fileName}
	if err := database.Create(&first).Error; err != nil { t.Fatal(err) }
	if err := database.Create(&second).Error; err != nil { t.Fatal(err) }

	otherSubject, otherEvent := createSubjectEvent(t, database, submitter.ID, "event-detail-other")
	_ = otherSubject
	otherTitle := "Other event source"
	if err := database.Create(&models.Evidence{EventID: &otherEvent.ID, Type: "text", Title: &otherTitle}).Error; err != nil { t.Fatal(err) }

	got, err := repository.NewEventRepository(database).GetByID(ctx, event.ID)
	if err != nil { t.Fatalf("GetByID() error = %v", err) }
	if len(got.Evidence) != 2 { t.Fatalf("evidence = %#v", got.Evidence) }
	if value(got.Evidence[0].Title) != firstTitle || value(got.Evidence[1].Title) != secondTitle {
		t.Fatalf("evidence order = %#v", got.Evidence)
	}
})
```

Use the repository's existing test helper names exactly; adjust only the unused local assignment if `createSubjectEvent` already discards Subject cleanly.

- [ ] **Step 8: Run the integration test and verify failure**

Run from `backend/` with the repository's integration database environment:

```bash
```

Expected: FAIL because `models.Event` has no `Evidence` field or `GetByID` does not preload it.

- [ ] **Step 9: Add the existing read-only association and ordered preload**

In `backend/internal/models/models.go`, add to `Event`:

```go
Evidence []Evidence `gorm:"foreignKey:EventID" json:"evidence,omitempty"`
```

In `backend/internal/repository/event.go`, change `GetByID` to:

```go
err := r.db.WithContext(ctx).
	Preload("Evidence", func(db *gorm.DB) *gorm.DB {
		return db.Order("created_at ASC")
	}).
	Where("id = ?", id).
	First(&event).Error
```

Do not preload Evidence in `ListBySubject` or Subject timeline reads.

- [ ] **Step 10: Verify Task 1**

Run:

```bash
# backend/

# frontend/
npm run typecheck
```

Expected: all commands PASS. Review `git diff --check`; do not commit without an explicit user request.

---

### Task 2: Trustworthy Subject And Event Profiles

**Covers:** [S1, S2, S4, S5, S8, S9, S10, S11, S12]

**Files:**
- Modify: `frontend/app/subjects/[id]/page.tsx`
- Modify: `frontend/app/events/[id]/page.tsx`
- Create: `frontend/e2e/trustworthy-profile.spec.ts`

**Interfaces:**
- Consumes: all Task 1 presentation helpers and `EventItem.evidence`.
- Produces: stable headings/labels used by Playwright: `对象档案`, `关联账号`, `公开事件轨迹`, `事实描述`, `证据材料`, `申诉与纠错`.

- [ ] **Step 1: Write deterministic failing Playwright scenarios**

Create `frontend/e2e/trustworthy-profile.spec.ts`. Intercept exact public routes and return stable payloads. Include these tests:

```ts
import { expect, test } from '@playwright/test'
import { expectNoHorizontalOverflow } from './helpers'

test('subject profile exposes identity accounts and event trail', async ({ page }) => {
  await page.route('**/api/v1/subjects/UBS_PROFILE_TEST', (route) => route.fulfill({
    contentType: 'application/json',
    body: JSON.stringify({
      id: 'subject-1', public_id: 'UBS_PROFILE_TEST', display_name: '档案对象', status: 'active', risk_level: 4,
      accounts: [
        { platform: 'discord', username: 'primary-user', account_id: '1001' },
        { platform: 'telegram', username: 'secondary-user', account_id: null },
      ],
      identifiers: [{ platform: 'qq', value: '123456' }],
      events: [
        { id: 'event-1', title: '第一条公开事件', details: '可核验摘要', status: 'published', severity: 3 },
        { id: 'event-2', title: '已更正事件', details: '更正后的摘要', status: 'corrected', severity: 2 },
      ],
    }),
  }))
  await page.goto('/subjects/UBS_PROFILE_TEST')
  await expect(page.getByRole('heading', { name: '对象档案' })).toBeVisible()
  await expect(page.getByText('UBS_PROFILE_TEST')).toBeVisible()
  await expect(page.getByText(/较高.*4\/5/)).toBeVisible()
  await expect(page.getByText('主标识')).toBeVisible()
  await expect(page.getByRole('link', { name: /查看事件与证据/ }).first()).toHaveAttribute('href', '/events/event-1')
  await expectNoHorizontalOverflow(page)
})
```

Add separate tests for:

- empty Subject events: include “不代表对该对象作出绝对安全判断” and exclude “安全对象”.
- Event with link, file-with-URL, file-without-URL, unknown evidence, and anonymous appeal entry.
- Event with `evidence: []`: show “当前公开响应中没有可展示的证据材料”.
- Event POST appeal 400: preserve textarea value and show danger feedback.
- Subject/Event GET 500: render recoverable ErrorState; Subject retains “返回列表”.

Each test calls `expectNoHorizontalOverflow(page)`; Playwright projects supply 375/768/1280.

- [ ] **Step 2: Run the new spec and verify failure**

Run from `frontend/`:

```bash
npm run test:e2e -- e2e/trustworthy-profile.spec.ts --project=desktop-chrome
```

Expected: FAIL because the approved headings, hierarchy, empty-state copy, and evidence actions are absent.

- [ ] **Step 3: Rebuild Subject detail as an archive-like profile**

Modify `frontend/app/subjects/[id]/page.tsx` with this semantic structure:

```tsx
<main className="mx-auto max-w-4xl py-8">
  <Link href="/subjects">← 返回列表</Link>
  <header className="mt-4 border-b border-border pb-6">
    <p className="text-sm font-mono text-muted">公开对象档案</p>
    <h1 className="mt-2 text-3xl font-bold">对象档案</h1>
    {/* display name, public id, translated status, risk text + n/5 */}
  </header>
  <section aria-labelledby="subject-accounts-heading">...</section>
  <section aria-labelledby="subject-events-heading">...</section>
</main>
```

Requirements:

- Filter Accounts whose `accountDisplayValue` is null.
- Show Accounts before legacy Identifiers.
- Mark the first displayed Account as `主标识`; if no Account exists, mark the first compatible Identifier.
- Label legacy rows `兼容账号记录`; never say “已验证”.
- Event list is a semantic `<ol>` with a left relationship line or border, not a grid of equal cards.
- Each Event shows translated status, translated severity plus `n/5`, summary, and `查看事件与证据`.
- Empty copy explicitly avoids an absolute safety conclusion.

- [ ] **Step 4: Rebuild Event detail into four explicit layers**

Modify `frontend/app/events/[id]/page.tsx`:

- Event header: title, translated status, event ID, translated severity plus `n/5`.
- `事实描述`: always render; preserve whitespace; empty details use a restrained empty state.
- `证据材料`: render type, title, description, optional filename/size; use `evidenceAction` only for real URLs.
- External evidence links retain `target="_blank" rel="noreferrer"`.
- `申诉与纠错`: retain authenticated request flow, but replace raw enum copy with shared Chinese labels.
- Replace `message.includes('失败')` with explicit feedback:

```ts
type Feedback = { tone: 'success' | 'danger'; message: string } | null
```

- On appeal failure, do not clear `reason`; on success, clear it and reload history.
- Do not invent a Subject backlink from `subject_id`.

- [ ] **Step 5: Run targeted profile verification**

Run from `frontend/`:

```bash
npm run test:run -- lib/trustworthy-profile.test.ts
npm run lint
npm run typecheck
npm run test:e2e -- e2e/trustworthy-profile.spec.ts --project=desktop-chrome --project=mobile
```

Expected: all PASS, including no horizontal overflow on desktop and mobile.

- [ ] **Step 6: Run the frontend build gate for the first product slice**

Run:

```bash
npm run build
git diff --check
```

Expected: production build succeeds and diff check is clean. Do not commit without explicit instruction.

---

### Task 3: Responsive Governance Workbench

**Covers:** [S1, S2, S4, S6, S8, S9, S10, S11, S12]

**Files:**
- Modify: `frontend/app/admin/appeals/page.tsx`
- Create: `frontend/app/admin/appeals/appeal-presentations.test.ts`
- Modify: `frontend/e2e/appeals-and-compat.spec.ts`

**Interfaces:**
- Consumes: `appealStatusPresentation` and `appealOutcomeLabel` from Task 1.
- Produces page-local:

```ts
type AppealListResponse = { appeals?: Appeal[]; total?: number; page?: number; page_size?: number }
type AppealTarget =
  | { kind: 'event'; id: string; href: string }
  | { kind: 'legacy-case'; id: string; href: null }
  | { kind: 'missing'; id: null; href: null }

export function appealTarget(appeal: Pick<Appeal, 'event_id' | 'case_id'>): AppealTarget
```

- [ ] **Step 1: Write failing target/audit tests**

Create `frontend/app/admin/appeals/appeal-presentations.test.ts`:

```ts
import { describe, expect, it } from 'vitest'
import { appealTarget } from './page'

describe('appeal workbench presentation', () => {
  it('prefers Event targets over legacy Case targets', () => {
    expect(appealTarget({ event_id: 'event-1', case_id: 'case-1' })).toEqual({
      kind: 'event', id: 'event-1', href: '/events/event-1',
    })
  })

  it('keeps legacy Case targets visible but unlinked', () => {
    expect(appealTarget({ event_id: null, case_id: 'case-1' })).toEqual({
      kind: 'legacy-case', id: 'case-1', href: null,
    })
  })
})
```

If importing a Next page causes test setup side effects, move only `appealTarget` to `frontend/app/admin/appeals/presentation.ts` and test that file. Do not create a global governance module.

- [ ] **Step 2: Extend failing Playwright workbench scenarios**

In `frontend/e2e/appeals-and-compat.spec.ts`, retain existing compatibility tests and add route-mocked scenarios for:

- `事件申诉工作台`, `待处理 N 项`, filter, refresh, Event link, legacy target, complete reason and resolution text.
- Absence of checkbox, `全选`, and `批量裁决` controls.
- approved filter with empty state while pending count remains sourced from a `status=pending&page_size=1` request.
- prompt resolve: accept `事实字段有误`, assert body `{ outcome: 'corrected', reason: '事实字段有误' }`, then return an adjudicated item and remove action buttons.
- prompt dismiss for `malicious_submission`: assert no resolve request.
- first list request 500, then refresh succeeds.
- no horizontal overflow at all configured viewports.

- [ ] **Step 3: Run targeted tests and verify failure**

Run from `frontend/`:

```bash
npm run test:run -- app/admin/appeals/appeal-presentations.test.ts
npm run test:e2e -- e2e/appeals-and-compat.spec.ts --project=desktop-chrome
```

Expected: FAIL because the target helper, new heading, pending count, responsive list, and refresh path are absent.

- [ ] **Step 4: Implement the workbench data flow**

Modify `frontend/app/admin/appeals/page.tsx`:

- Track list `total`, pending `total`, explicit feedback, and request freshness.
- For `filter === 'pending'`, use the list response `total` as pending total.
- For other filters, request `/api/appeals?page=1&page_size=1&status=pending` for count only.
- Prevent stale filter responses from replacing current results using an `AbortController` in the effect-driven load or a monotonically increasing request id.
- Keep exact resolve request semantics:

```ts
const reason = window.prompt(`裁决说明（outcome=${outcome}）`)
if (reason === null) return
await apiRequest(`/api/appeals/${id}/review/resolve`, {
  auth: true,
  json: { outcome, reason: reason.trim() },
})
```

- Preserve empty-string reasons because the existing endpoint accepts them.
- Reload server state after success; do not optimistically infer Event or sanction consequences.

- [ ] **Step 5: Replace the horizontal table with one responsive semantic list**

Required order per `<li>`:

1. Event ID link, legacy Case compatibility label, or unavailable target.
2. translated appeal status with raw enum in `auditLabel`.
3. full `whitespace-pre-wrap` reason.
4. translated outcome with raw enum and optional `resolution_reason`.
5. four single-item actions only when pending.

Page header/tooling:

- Heading `事件申诉工作台`.
- Text: “逐项核对申诉理由和关联事件，不进行批量裁决。”
- `待处理 — 项` before a count is known; never show a misleading initial zero.
- Filter labels: `待处理（pending）`, `已通过（approved）`, `已驳回（rejected）`, `全部状态`.
- `刷新队列` button remains available after errors and says `刷新中…` while loading.

- [ ] **Step 6: Verify Task 3**

Run from `frontend/`:

```bash
npm run test:run -- app/admin/appeals/appeal-presentations.test.ts lib/trustworthy-profile.test.ts
npm run lint
npm run typecheck
npm run test:e2e -- e2e/appeals-and-compat.spec.ts --project=desktop-chrome --project=tablet --project=mobile
npm run build
git diff --check
```

Expected: all PASS; mobile does not rely on horizontal table scrolling. Do not commit without explicit instruction.

---

### Task 4: Guided Publishing Domain Logic

**Covers:** [S1, S2, S3, S4, S7, S8, S9, S10, S11, S12]

**Files:**
- Create: `frontend/app/submit/publishing.ts`
- Create: `frontend/app/submit/publishing.test.ts`
- Modify: `frontend/lib/types.ts`
- Modify: `frontend/components/auth/demo-captcha.tsx`
- Create: `frontend/components/auth/demo-captcha.test.tsx`

**Interfaces:**
- Produces:

```ts
export type PublishSectionId = 'accounts' | 'facts' | 'evidence-confirmation'
export type PublishRequirements = { emailVerification: boolean; captcha: boolean }
export type PublishEventDraft = PublishEventInput & {
  text_evidence: string
  link_url: string
  link_title: string
  files: File[]
}
export type PublishDraft = {
  displayName: string
  accounts: PublishAccountInput[]
  events: PublishEventDraft[]
  verificationCode: string
  captchaToken: string
}
export type PublishIssue = { section: PublishSectionId; field?: string; message: string }

export function validatePublishDraft(draft: PublishDraft, requirements: PublishRequirements): PublishIssue[]
export function getPublishSectionCompletion(draft: PublishDraft, requirements: PublishRequirements): Record<PublishSectionId, boolean>
export function mapPublishErrorToSection(error: unknown): PublishSectionId
export function buildPublishPayload(draft: PublishDraft): PublishSubjectPayload
export function listPublishFiles(draft: PublishDraft): Array<{ field: string; file: File }>
```

- Extends `DemoCaptcha` with optional `onError?: (message: string) => void`.

- [ ] **Step 1: Add publish API boundary types**

Add to `frontend/lib/types.ts`:

```ts
export type PublishAccountInput = { platform: string; username: string; account_id: string }
export type PublishEventInput = { title: string; details: string; severity: number }
export type PublishTextEvidenceInput = { event_index: number; title: string; text: string }
export type PublishLinkEvidenceInput = { event_index: number; title: string; description: string; url: string }
export type PublishFileEvidenceInput = { event_index: number; title: string; filename: string; field: string }
export type PublishSubjectPayload = {
  display_name: string
  accounts: PublishAccountInput[]
  events: PublishEventInput[]
  text_evidence: PublishTextEvidenceInput[]
  link_evidence: PublishLinkEvidenceInput[]
  file_evidence: PublishFileEvidenceInput[]
  verification_code: string
  captcha_token: string
}
export type PublishSubjectResponse = { public_id: string }
```

- [ ] **Step 2: Write failing publishing-domain tests**

Create `frontend/app/submit/publishing.test.ts`. Cover these exact behavioral groups:

- empty draft: all sections incomplete when both verification requirements are enabled.
- platform + username completes accounts; account ID alone requires explicit display name.
- every added Account row must be valid; normalized duplicate platform/value fails.
- every Event requires trimmed title/details and severity 1–5.
- no evidence is allowed when enabled verification requirements are satisfied.
- partially filled links fail; only absolute `http:`/`https:` URLs with a host pass.
- UTF-8 text evidence passes at 200 KiB and fails one byte over using `TextEncoder`.
- files fail over 20 MiB or outside `.png,.jpg,.jpeg,.gif,.webp,.pdf,.txt,.bin,.zip,.webm,.mp4,.md`.
- error keywords map deterministically to the three sections; unknown errors map to `evidence-confirmation`.
- payload preserves Account/Event order, trims/filter evidence, preserves `event_index`, and emits file fields `file_0_0`, `file_0_1`, `file_1_0`.

Representative test:

```ts
it('keeps the existing multipart file field contract', () => {
  const draft = validDraft()
  draft.events[0].files = [new File(['a'], 'a.pdf'), new File(['b'], 'b.txt')]
  draft.events.push({ ...validEvent(), files: [new File(['c'], 'c.png')] })
  expect(listPublishFiles(draft).map((item) => item.field)).toEqual(['file_0_0', 'file_0_1', 'file_1_0'])
  expect(buildPublishPayload(draft).file_evidence.map((item) => item.field)).toEqual(['file_0_0', 'file_0_1', 'file_1_0'])
})
```

- [ ] **Step 3: Run publishing tests and verify failure**

Run from `frontend/`:

```bash
npm run test:run -- app/submit/publishing.test.ts
```

Expected: FAIL because `publishing.ts` is absent.

- [ ] **Step 4: Implement the minimal publishing-domain module**

Create `frontend/app/submit/publishing.ts` with the interfaces above.

Validation rules must match existing backend behavior only:

- account normalization: lower-case trimmed platform; use account ID when present, otherwise username.
- display name may be empty only when at least one username can resolve it.
- evidence is recommended, not required.
- text limit uses UTF-8 byte length.
- links use `new URL`, then require `http:` or `https:` and non-empty hostname.
- files are metadata validation only; do not mutate or read file contents.
- `buildPublishPayload` keeps `display_name` unchanged; the server remains responsible for default display-name resolution.

- [ ] **Step 5: Write failing DemoCaptcha tests**

Create `frontend/components/auth/demo-captcha.test.tsx` using Testing Library and mocked `apiRequest`. Cover:

- initial and verified `aria-pressed` states.
- submission path `/api/verification/demo/submission`, `auth: true`.
- success calls `onChange(token)`.
- verified click clears with `onChange('')` and makes no request.
- pending request disables button and shows `验证中...`.
- `ApiError` calls `onError(error.message)`.
- unknown error calls `onError('人机验证失败，请重试')`.
- failed request does not call `onChange('')`.

- [ ] **Step 6: Implement DemoCaptcha's optional error boundary**

Update its props:

```ts
onError?: (message: string) => void
```

Catch request errors, call `onError`, preserve the existing value, and add `disabled={loading}`. Existing register callers require no changes because the prop is optional.

- [ ] **Step 7: Verify Task 4 pure logic**

Run from `frontend/`:

```bash
npm run test:run -- app/submit/publishing.test.ts components/auth/demo-captcha.test.tsx
npm run lint
npm run typecheck
```

Expected: all PASS. Do not commit without explicit instruction.

---

### Task 5: Guided Publishing Page And Browser Contract

**Covers:** [S1, S2, S4, S7, S8, S9, S10, S11, S12]

**Files:**
- Modify: `frontend/app/submit/page.tsx`
- Create: `frontend/e2e/guided-publishing.spec.ts`

**Interfaces:**
- Consumes all Task 4 helpers and types.
- Keeps endpoint contract:

```ts
POST /api/subjects/publish
// no files: { auth: true, json: PublishSubjectPayload }
// files: { method: 'POST', auth: true, body: FormData(payload + file_* fields) }
```

- [ ] **Step 1: Write failing guided-publishing Playwright scenarios**

Create `frontend/e2e/guided-publishing.spec.ts` with local route mocks. Cover:

1. anonymous `/submit`: login guidance, no active publish/file controls, no overflow.
2. authenticated task flow: three headings, initial `待补充`, completed states after inputs, severity 3 selection, no-evidence recommendation.
3. client validation: invalid account prevents publish request, focuses `publish-section-accounts`, preserves display name/Event/code.
4. JSON server failure `invalid verification code`: request is JSON, error maps/focuses confirmation section, every input remains.
5. multipart success using in-memory `setInputFiles`: request includes `payload` and `file_0_0`; success shows public ID and `/subjects/:publicID` link.
6. captcha endpoint failure: third section error, earlier values remain, captcha button re-enables.
7. `expectNoHorizontalOverflow` for every scenario.

Use an injected valid JWT or the existing `loginAs` helper according to the current E2E setup. Mock `/api/settings/public` in this spec so `security.email_verification` and `security.captcha_enabled` are deterministic; do not create permanent objects in the database.

- [ ] **Step 2: Run the new E2E spec and verify failure**

Run from `frontend/`:

```bash
E2E_ALLOW_DEFAULT_USERS=1 npm run test:e2e -- e2e/guided-publishing.spec.ts --project=desktop-chrome
```

Expected: FAIL because the three-section flow, completion language, severity control, and focus behavior are absent.

- [ ] **Step 3: Load public verification requirements through the existing API boundary**

In `frontend/app/submit/page.tsx`, load `PublicSettings` as the register page does:

```ts
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
```

Use `settingBool`. Do not add these security values to `SiteContext` solely for this page. A failed settings request must not clear draft state.

- [ ] **Step 4: Recompose the form into three stable focusable sections**

Use one `<form noValidate>` and these exact containers:

```tsx
<Panel id="publish-section-accounts" tabIndex={-1}>...</Panel>
<Panel id="publish-section-facts" tabIndex={-1}>...</Panel>
<Panel id="publish-section-evidence-confirmation" tabIndex={-1}>...</Panel>
```

Each section includes a visible number, approved title, concise explanation, and text completion Badge (`已完成`/`待补充`).

- Accounts section: display name plus Account rows; explain first username resolution.
- Facts section: each Event has labels for title/details and a `Select` severity 1–5 with Chinese labels.
- Evidence/confirmation section: evidence inputs grouped under their Event title, selected file names/count, no-evidence recommendation, verification controls, and final checklist.
- Do not add page navigation, persistence, Account deletion, Event deletion, or a generic Step component.

- [ ] **Step 5: Implement validation, focus, and protocol-preserving submit**

Before sending:

```ts
const issues = validatePublishDraft(draft, requirements)
if (issues.length > 0) {
  setIssues(issues)
  focusSection(issues[0].section)
  return
}
```

Use:

```ts
function focusSection(section: PublishSectionId) {
  requestAnimationFrame(() => {
    document.getElementById(`publish-section-${section}`)?.focus()
  })
}
```

On API failure:

- preserve all React state, including File arrays.
- keep a top-level Alert and a section-level issue.
- map section using `mapPublishErrorToSection`.
- restore button from `发布中...`.

For request construction, use only `buildPublishPayload` and `listPublishFiles`; retain the exact JSON/multipart branches. Use `PublishSubjectResponse` for success.

- [ ] **Step 6: Connect verification errors to the final section**

- `sendCode` failures set feedback/issue for `evidence-confirmation` and focus it.
- Pass `onError` to `DemoCaptcha` with the same behavior.
- Show email code and captcha only when corresponding public settings enable them.
- On settings fallback, show a warning and conservatively require both, preserving current secure behavior.

- [ ] **Step 7: Verify Task 5**

Run from `frontend/`:

```bash
npm run test:run -- app/submit/publishing.test.ts components/auth/demo-captcha.test.tsx
npm run lint
npm run typecheck
npm run test:e2e -- e2e/guided-publishing.spec.ts --project=desktop-chrome --project=tablet --project=mobile
npm run build
git diff --check
```

Expected: all PASS; JSON and multipart request assertions remain compatible. Do not commit without explicit instruction.

---

### Task 6: Full Verification, Independent Review, And Progress Evidence

**Covers:** [S1, S2, S3, S4, S5, S6, S7, S8, S9, S10, S11, S12]

**Files:**
- Modify: `docs/frontend-gap-analysis.md`
- Review: every file changed by Tasks 1–5

**Interfaces:**
- Consumes: completed and individually green slices.
- Produces: verified ledger evidence and a review-ready diff; no new product behavior.

- [ ] **Step 1: Run complete backend verification**

Run from `backend/`:

```bash
gofmt -w internal/models/models.go internal/repository/event.go internal/integration/governance_test.go
```

Expected: all tests PASS and the server binary builds at `/tmp/uniblack`.

- [ ] **Step 2: Run complete frontend verification**

Run from `frontend/`:

```bash
npm run test:run
npm run lint
npm run typecheck
npm run build
E2E_ALLOW_DEFAULT_USERS=1 npm run test:e2e
```

Expected: all unit tests, lint, typecheck, production build, and all desktop/tablet/mobile Playwright projects PASS.

If system Chrome is unavailable, rerun E2E with:

```bash
PLAYWRIGHT_USE_BUNDLED=1 E2E_ALLOW_DEFAULT_USERS=1 npm run test:e2e
```

- [ ] **Step 3: Run an independent read-only review**

Dispatch a fresh reviewer against the approved spec and final diff. Require findings first and ask it to verify:

- no invented evidence actions or audit metadata.
- no direct token/fetch boundary violations.
- no Case/Submission retirement regression.
- Event evidence is preloaded only for detail.
- prompt-based single resolution remains intact.
- publishing JSON/multipart shapes and input preservation remain intact.
- all `[S1]`–`[S12]` requirements have implementation or test evidence.

Fix any real findings and rerun the affected targeted tests plus the full gate when behavior changes.

- [ ] **Step 4: Update only factual progress ledger rows**

In `docs/frontend-gap-analysis.md`:

- Update `/subjects/[id]`, `/events/[id]`, `/admin/appeals`, and `/submit` rows with the implemented experience.
- Update shared status/presentation and responsive evidence only when commands actually passed.
- Record exact observed test/build counts from Step 2; do not predict counts.
- Do not rewrite roadmap Goal bodies or old compose specs/plans.

- [ ] **Step 5: Final workspace and diff audit**

Run from repository root:

```bash
git status --short --branch
git diff --check
git diff --stat
```

Expected:

- branch is `feature/product-experience-deepening`.
- no accidental `.env`, credentials, generated browser artifacts, or `.mimocode/` files are staged/tracked.
- diff check is clean.
- no commit, merge, push, or tag is performed unless the user explicitly requests it.

---

## Plan Self-Review

- Spec coverage: Tasks 1–6 collectively cover every real anchor `[S1]` through `[S12]`; every `Covers` ID exists in the approved spec.
- Scope: only Event detail receives a backend read change; governance and publishing do not expand APIs or persistence.
- Type consistency: `EventItem.evidence`, backend JSON `evidence`, shared status helper names, publishing draft names, and `PublishSubjectPayload` remain consistent across tasks.
- Placeholder scan: no TBD, TODO, “similar to”, or unspecified test step remains.
- Delivery order: shared presentation and Event evidence → public profile → governance → publishing logic → publishing UI → full verification.
