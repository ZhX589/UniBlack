# Subject Event Governance Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use compose:subagent (recommended) or compose:execute to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 UniBlack 从 Case 导向的举报流程升级为对象、账号、事件、证据归档、申诉处罚和统一控制台的可治理系统。

**Architecture:** 数据库保存对象、账号、事件、证据索引、处罚和审计关系；对象目录中的 `manifest.json` 与 `README.txt` 是可导出、可校验的快照。对外逐步采用 Subject/Event 术语和 `UBS_<ULID>`，以可回滚迁移和兼容读取路径替代破坏性重建。

**Tech Stack:** Go、Echo、GORM（仅运行期读写）、PostgreSQL、golang-migrate、对象存储接口、Next.js 14、React 18、Tailwind CSS。

## Global Constraints

- 所有功能代码在 `feature/subject-event-governance` 分支实现，不直接提交 `main`。
- 建表和数据迁移仅使用 `backend/internal/migrations/*.up.sql` / `*.down.sql`。
- 对外使用对象、账号、事件、证据；新代码不得新增 Case 术语。
- 公开 ID 固定为不可编辑的 `UBS_<ULID>`；内部关系继续使用 UUID。
- 数据库保存检索、关系、治理和审计索引；JSON 包不是唯一真源。
- 文本证据为 UTF-8 `.txt`，最大 200 KiB。
- 注册和提交需要邮箱验证与内置演示 captcha；不得调用第三方 captcha 端点。
- `APP_ENV=development` 只接受邮箱码 `123456`；非开发环境无 SMTP 必须失败。
- 菜单隐藏不能替代后端 RBAC；所有写入和治理动作必须审计。

---

### Task 1: 对象公开 ID、账号与事件的可回滚数据迁移

**Covers:** S1, S2, S3, S4, S11

**Files:**
- Create: `backend/internal/migrations/000006_subject_events.up.sql`
- Create: `backend/internal/migrations/000006_subject_events.down.sql`
- Modify: `backend/internal/models/models.go`
- Test: `backend/internal/service/subject_event_test.go`

**Interfaces:**
- Produces `Subject.PublicID string`, `Account`, `Event`; `GeneratePublicID() string` returns `UBS_` plus a ULID.

- [ ] **Step 1: Write failing service tests**

```go
func TestCreateSubjectUsesGeneratedPublicID(t *testing.T) {
    subject := newSubject("", []AccountInput{{Platform: "qq", Username: "alice"}})
    require.Regexp(t, `^UBS_[0-9A-HJKMNP-TV-Z]{26}$`, subject.PublicID)
    require.Equal(t, "alice", subject.DisplayName)
}

func TestAccountIDWinsDuplicateKey(t *testing.T) {
    require.Equal(t, "telegram:123", accountDedupKey("telegram", "Alice", "123"))
}
```

- [ ] **Step 2: Run the focused test to verify failure**

Run: `cd backend && go test ./internal/service -run 'TestCreateSubjectUsesGeneratedPublicID|TestAccountIDWinsDuplicateKey' -count=1`

Expected: FAIL because `AccountInput`, `GeneratePublicID`, and `accountDedupKey` do not exist.

- [ ] **Step 3: Add migration and minimal domain types**

```sql
ALTER TABLE subjects ADD COLUMN public_id VARCHAR(30);
UPDATE subjects SET public_id = 'UBS_' || replace(id::text, '-', '') WHERE public_id IS NULL;
ALTER TABLE subjects ALTER COLUMN public_id SET NOT NULL;
CREATE UNIQUE INDEX idx_subjects_public_id ON subjects(public_id);

CREATE TABLE accounts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  subject_id UUID NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
  platform VARCHAR(50) NOT NULL,
  platform_label VARCHAR(100),
  account_type VARCHAR(20) NOT NULL,
  username VARCHAR(255),
  account_id VARCHAR(255),
  custom_attributes JSONB NOT NULL DEFAULT '{}',
  is_primary BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (username IS NOT NULL OR account_id IS NOT NULL)
);
CREATE UNIQUE INDEX idx_accounts_platform_id ON accounts(platform, account_id) WHERE account_id IS NOT NULL;
CREATE UNIQUE INDEX idx_accounts_platform_username ON accounts(platform, username) WHERE account_id IS NULL AND username IS NOT NULL;

CREATE TABLE events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  subject_id UUID NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
  title VARCHAR(255) NOT NULL,
  occurred_from TIMESTAMPTZ,
  occurred_to TIMESTAMPTZ,
  details TEXT NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'published',
  severity SMALLINT NOT NULL DEFAULT 1 CHECK (severity BETWEEN 1 AND 5),
  submitted_by UUID REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (occurred_to IS NULL OR occurred_from IS NULL OR occurred_to >= occurred_from)
);
```

Use a matching down migration that drops only the new indexes/tables/column after documenting that migrated public IDs are not reversible to absence.

- [ ] **Step 4: Implement subject/account/event model methods and pass tests**

```go
func accountDedupKey(platform, username, accountID string) string {
    platform = strings.ToLower(strings.TrimSpace(platform))
    if accountID != "" { return platform + ":" + strings.TrimSpace(accountID) }
    return platform + ":" + strings.TrimSpace(username)
}
```

Run: `cd backend && go test ./internal/service -run 'TestCreateSubjectUsesGeneratedPublicID|TestAccountIDWinsDuplicateKey' -count=1`

Expected: PASS.

- [ ] **Step 5: Verify migration round trip and commit**

Run: `cd backend && go test ./... && go build ./cmd/server`

Expected: exit 0.

Commit: `git commit -am "feat: add subject public ids accounts and events"`

### Task 2: 实现真实对象存储与证据归档索引

**Covers:** S4, S5, S10, S11

**Files:**
- Modify: `backend/internal/storage/storage.go`
- Create: `backend/internal/storage/local.go`
- Modify: `backend/internal/models/models.go`
- Modify: `backend/internal/service/evidence.go`
- Modify: `backend/internal/repository/evidence.go`
- Create: `backend/internal/service/evidence_archive_test.go`

**Interfaces:**
- Consumes `Subject.PublicID`, Event ID, `storage.Storage`.
- Produces `BuildEvidenceKey(publicID string, eventNumber, evidenceNumber int, ext string) string` and a persisted `Evidence.StorageKey`.

- [ ] **Step 1: Write failing archive tests**

```go
func TestBuildEvidenceKey(t *testing.T) {
    require.Equal(t, "subjects/UBS_01ABC/evidence/UBS_01ABC_E001_T002.txt",
        BuildEvidenceKey("UBS_01ABC", 1, 2, ".txt"))
}

func TestTextEvidenceRejectsOver200KiB(t *testing.T) {
    _, err := service.CreateTextEvidence(ctx, input(strings.Repeat("x", 200*1024+1)))
    require.ErrorIs(t, err, ErrTextEvidenceTooLarge)
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `cd backend && go test ./internal/service -run 'TestBuildEvidenceKey|TestTextEvidenceRejectsOver200KiB' -count=1`

Expected: FAIL because archive helpers do not exist.

- [ ] **Step 3: Implement local storage writes and evidence keying**

```go
const maxTextEvidenceBytes = 200 * 1024

func BuildEvidenceKey(publicID string, eventNumber, evidenceNumber int, ext string) string {
    return fmt.Sprintf("subjects/%s/evidence/%s_E%03d_%s%03d%s",
        publicID, publicID, eventNumber, map[bool]string{true: "T", false: "F"}[ext == ".txt"], evidenceNumber, ext)
}
```

Store uploaded bytes under the key, persist `storage_key`, original filename, SHA-256, MIME, size and event relation. Text evidence uses `strings.NewReader(text)` and extension `.txt`; reject invalid UTF-8 and oversized content.

- [ ] **Step 4: Run tests and file integrity checks**

Run: `cd backend && go test ./internal/service ./internal/storage -count=1`

Expected: PASS, including a test that reads stored bytes and compares SHA-256.

- [ ] **Step 5: Commit**

Commit: `git commit -am "feat: archive evidence under subject public ids"`

### Task 3: 默认发布提交和对象事件事务

**Covers:** S2, S3, S4, S8, S10

**Files:**
- Modify: `backend/internal/service/submission.go`
- Modify: `backend/internal/repository/submission.go`
- Modify: `backend/internal/handler/submission.go`
- Create: `backend/internal/service/submission_publish_test.go`
- Modify: `frontend/app/submit/page.tsx`

**Interfaces:**
- Produces `CreatePublishedSubmission(ctx, req, userID) (*PublishedSubmission, error)`.
- Request contains `display_name`, `accounts`, `events`, `email_code`, `demo_token`.

- [ ] **Step 1: Write failing transaction tests**

```go
func TestPublishedSubmissionCreatesSubjectAccountsEventsAndAudit(t *testing.T) {
    result, err := svc.CreatePublishedSubmission(ctx, requestWithOneEvent(), userID)
    require.NoError(t, err)
    require.Equal(t, "published", result.Events[0].Status)
    require.Len(t, result.Subject.Accounts, 1)
    require.NotEmpty(t, result.Subject.PublicID)
}

func TestSubmissionRollsBackWhenAccountDuplicates(t *testing.T) {
    _, err := svc.CreatePublishedSubmission(ctx, duplicateAccountRequest(), userID)
    require.ErrorIs(t, err, ErrDuplicateAccount)
    require.Zero(t, countCreatedSubjects(t))
}
```

- [ ] **Step 2: Run the tests to verify failure**

Run: `cd backend && go test ./internal/service -run 'TestPublishedSubmission' -count=1`

Expected: FAIL because the publishing service does not exist.

- [ ] **Step 3: Implement a single database transaction**

```go
return s.db.Transaction(func(tx *gorm.DB) error {
    subject := makeSubject(req.DisplayName, req.Accounts)
    if err := subjectRepo.With(tx).Create(ctx, subject); err != nil { return err }
    if err := accountRepo.With(tx).CreateAll(ctx, subject.ID, req.Accounts); err != nil { return err }
    if err := eventRepo.With(tx).CreateAll(ctx, subject.ID, req.Events, userID); err != nil { return err }
    return auditRepo.With(tx).Create(ctx, publishedAudit(subject, userID))
})
```

Before this transaction, reject active submission sanctions and require validated email/demo tokens. Preserve old Submission review endpoints during the migration window; route the new form to the new endpoint.

- [ ] **Step 4: Implement the disabled unauthenticated submit state**

```tsx
{auth.status !== 'authenticated' ? (
  <DisabledSubmitNotice loginHref={`/login?next=${encodeURIComponent('/submit')}`} />
) : (
  <SubjectEventSubmissionForm />
)}
```

The form has Object, Accounts, Events, Evidence, Verification/Publish sections; errors link to their section IDs.

- [ ] **Step 5: Run service and frontend checks, then commit**

Run: `cd backend && go test ./internal/service -count=1 && go build ./cmd/server && cd ../frontend && npm run build`

Expected: exit 0.

Commit: `git commit -am "feat: publish subject events from verified submissions"`

### Task 4: 演示 captcha 与分环境邮箱验证

**Covers:** S7, S8, S10, S11

**Files:**
- Modify: `backend/internal/captcha/provider.go`
- Create: `backend/internal/captcha/demo.go`
- Modify: `backend/internal/service/auth.go`
- Modify: `backend/internal/repository/verification.go`
- Modify: `backend/internal/setting/options.go`
- Modify: `frontend/app/register/page.tsx`
- Create: `frontend/components/auth/demo-captcha.tsx`
- Create: `backend/internal/service/verification_test.go`

**Interfaces:**
- Produces `IssueDemoToken(ctx, purpose, sessionID)`, `VerifyDemoToken(ctx, token, purpose, sessionID)` and `VerifyEmailCode(ctx, email, purpose, code)`.

- [ ] **Step 1: Write failing verification tests**

```go
func TestDevelopmentEmailCodeIsFixed(t *testing.T) {
    svc := newAuthService("development")
    require.NoError(t, svc.VerifyEmailCode(ctx, "a@example.com", "register", "123456"))
    require.Error(t, svc.VerifyEmailCode(ctx, "a@example.com", "register", "654321"))
}

func TestDemoCaptchaDoesNotCallExternalHTTP(t *testing.T) {
    token := issueDemoToken(t, "submission", "session-1")
    require.NoError(t, verifyDemoToken(ctx, token, "submission", "session-1"))
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `cd backend && go test ./internal/service ./internal/captcha -run 'TestDevelopmentEmailCodeIsFixed|TestDemoCaptchaDoesNotCallExternalHTTP' -count=1`

Expected: FAIL because demo token and environment split do not exist.

- [ ] **Step 3: Implement only the demo provider**

```go
func (s *DemoStore) Issue(ctx context.Context, purpose, sessionID string) (string, error) {
    return s.sign(DemoClaims{Purpose: purpose, SessionID: sessionID, ExpiresAt: time.Now().Add(5 * time.Minute)})
}
```

Delete runtime selection of real HTTP captcha providers. Keep Catalog fields as configuration metadata and expose `security.captcha_mode = "demo"` publicly. In production, SMTP absence returns `ErrSMTPRequired`; in development, `123456` is accepted without mail send/database random code.

- [ ] **Step 4: Add accessible DemoCaptcha UI and verify tests**

```tsx
<button type="button" aria-pressed={verified} onClick={verify}>
  {verified ? '验证已完成' : '我不是自动程序'}
</button>
```

Use it in registration and submission; no third-party `<script>` tags may remain.

Run: `cd backend && go test ./internal/service ./internal/captcha -count=1 && cd ../frontend && npm run build`

Expected: exit 0.

- [ ] **Step 5: Commit**

Commit: `git commit -am "feat: add demo captcha and environment email verification"`

### Task 5: 申诉结论、分级处罚与审计

**Covers:** S4, S6, S10, S11

**Files:**
- Create: `backend/internal/migrations/000007_sanctions.up.sql`
- Create: `backend/internal/migrations/000007_sanctions.down.sql`
- Create: `backend/internal/models/sanction.go`
- Create: `backend/internal/repository/sanction.go`
- Create: `backend/internal/service/sanction.go`
- Modify: `backend/internal/service/appeal.go`
- Modify: `backend/internal/handler/appeal.go`
- Create: `backend/internal/service/sanction_test.go`

**Interfaces:**
- Produces `SanctionService.HasActiveSubmissionRestriction(ctx, userID) (bool, error)` and `ResolveAppeal(ctx, appealID, resolution, actorID)`.

- [ ] **Step 1: Write failing sanction tests**

```go
func TestSuspensionBlocksSubmissionUntilExpiry(t *testing.T) {
    createSanction(t, userID, "submission_suspension", time.Now().Add(time.Hour))
    blocked, err := svc.HasActiveSubmissionRestriction(ctx, userID)
    require.NoError(t, err)
    require.True(t, blocked)
}

func TestWithdrawnEventKeepsAuditRecord(t *testing.T) {
    err := svc.ResolveAppeal(ctx, appealID, Resolution{Outcome: "withdrawn", Reason: "证据不足"}, adminID)
    require.NoError(t, err)
    require.NotEmpty(t, listAuditForEvent(t, eventID))
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `cd backend && go test ./internal/service -run 'TestSuspensionBlocksSubmissionUntilExpiry|TestWithdrawnEventKeepsAuditRecord' -count=1`

Expected: FAIL because sanction service and resolution outcomes do not exist.

- [ ] **Step 3: Add sanctions and resolution implementation**

```sql
CREATE TABLE sanctions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id),
  type VARCHAR(32) NOT NULL CHECK (type IN ('warning','submission_suspension','submission_ban')),
  reason TEXT NOT NULL,
  related_event_id UUID REFERENCES events(id),
  related_appeal_id UUID REFERENCES appeals(id),
  starts_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  ends_at TIMESTAMPTZ,
  imposed_by UUID NOT NULL REFERENCES users(id),
  revoked_at TIMESTAMPTZ,
  revoked_by UUID REFERENCES users(id),
  revoke_reason TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Require admin for create/revoke. Only `submission_suspension` before `ends_at` and non-revoked `submission_ban` block publishing. `warning` never blocks.

- [ ] **Step 4: Run tests and authorization checks**

Run: `cd backend && go test ./internal/service ./internal/handler -count=1 && go build ./cmd/server`

Expected: exit 0.

- [ ] **Step 5: Commit**

Commit: `git commit -am "feat: add appeal outcomes and submission sanctions"`

### Task 6: 对象 JSON 包导出、预览导入和管理端

**Covers:** S5, S6, S9, S10, S11

**Files:**
- Create: `backend/internal/export/subject_archive.go`
- Create: `backend/internal/export/subject_archive_test.go`
- Create: `backend/internal/handler/export.go`
- Modify: `backend/cmd/server/main.go`
- Create: `frontend/app/admin/governance/page.tsx`
- Create: `frontend/app/admin/sanctions/page.tsx`
- Create: `frontend/app/admin/archives/page.tsx`
- Modify: `frontend/lib/navigation.ts`
- Modify: `docs/configuration.md`
- Modify: `docs/frontend-gap-analysis.md`

**Interfaces:**
- Produces `BuildSubjectArchive(ctx, publicID) (Archive, error)`, `ValidateImport(r io.Reader) (ImportPreview, error)`.

- [ ] **Step 1: Write failing archive tests**

```go
func TestArchiveManifestReferencesHashedEvidence(t *testing.T) {
    archive, err := exporter.BuildSubjectArchive(ctx, publicID)
    require.NoError(t, err)
    require.Equal(t, 1, archive.Manifest.SchemaVersion)
    require.Equal(t, "UBS_01ABC_E001_T001.txt", archive.Manifest.Events[0].Evidence[0].FileName)
    require.NotEmpty(t, archive.Manifest.Events[0].Evidence[0].SHA256)
}

func TestImportPreviewRejectsExistingPublicID(t *testing.T) {
    preview, err := importer.ValidateImport(existingArchive(t))
    require.NoError(t, err)
    require.Contains(t, preview.Conflicts, publicID)
}
```

- [ ] **Step 2: Run archive tests to verify failure**

Run: `cd backend && go test ./internal/export -run 'TestArchiveManifestReferencesHashedEvidence|TestImportPreviewRejectsExistingPublicID' -count=1`

Expected: FAIL because the export package does not exist.

- [ ] **Step 3: Implement zip archive and preview-only import**

```go
type Manifest struct {
    SchemaVersion int `json:"schema_version"`
    Subject       SubjectManifest `json:"subject"`
    Events        []EventManifest `json:"events"`
    ExportedAt    time.Time `json:"exported_at"`
}
```

Write `manifest.json`, Chinese `README.txt`, and evidence bytes to a zip response. Import endpoint performs validation and conflict preview only; a separate explicit confirm endpoint applies a validated import and never overwrites an existing `public_id`.

- [ ] **Step 4: Add unified admin entries and verify UI build**

Add Content & Governance entries for Objects, Events, Appeals, Sanctions, Audit; add Site & Configuration entries for Brand, Registration & Verification, SMTP, Demo Captcha, Import/Export. Reuse the existing Admin Shell and role filters.

Run: `cd backend && go test ./internal/export ./... && go build ./cmd/server && cd ../frontend && npm run build`

Expected: exit 0.

- [ ] **Step 5: Perform end-to-end archive smoke test and commit**

Run: `curl -f -H "Authorization: Bearer $ADMIN_TOKEN" http://localhost:8080/api/admin/exports/subjects/$PUBLIC_ID -o /tmp/$PUBLIC_ID.zip && unzip -t /tmp/$PUBLIC_ID.zip && unzip -p /tmp/$PUBLIC_ID.zip manifest.json | jq '.schema_version == 1'`

Expected: `No errors detected` and `true`.

Commit: `git commit -am "feat: export subject event archives and governance console"`

## Final Review Gate

- [ ] Run `cd backend && go test ./... && go build ./cmd/server`; expected exit 0.
- [ ] Run `cd frontend && npm run lint && npm run typecheck && npm run build`; expected exit 0.
- [ ] Verify no runtime source calls `challenges.cloudflare.com`, `google.com/recaptcha`, or `hcaptcha.com`.
- [ ] Verify development accepts only `123456`; production without SMTP returns a clear failure.
- [ ] Export an archive, validate zip and SHA-256, then verify conflicting import stays preview-only.
- [ ] Verify anonymous submit is disabled, valid user submits published event, suspended user is blocked, and admin can resolve appeal/revoke sanction.
- [ ] Update `docs/roadmap.md`, `docs/configuration.md`, and `docs/frontend-gap-analysis.md` only with fresh verification evidence.
