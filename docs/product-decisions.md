# Product decisions (locked)

> 与 `AGENTS.md` § Product decisions 同步。变更需单独讨论，勿在实现 PR 中静默推翻。

| Topic | Decision | Rationale |
| --- | --- | --- |
| Public API Key | **Not implemented** | Mutating APIs require session JWT + RBAC; public read is rate-limited. A dedicated security model would be required before introducing long-lived keys. |
| OpenAPI generator | **Deferred** | Contract is README endpoint tables, Echo routes, and `docs/api/*`. Auto-gen is docs tooling, not a release gate. |
| OAuth / OIDC | **Reserved only** | `AuthProvider` interface + settings keys exist; no provider login UI until a dedicated phase. |
| Dark mode | **Out of scope** (Phase 12) | Light theme + `prefers-reduced-motion` only. |
| Case / Submission / Identifier | **Compat until 2026-12-31** | Event-first APIs are canonical; legacy routes send Deprecation/Sunset headers. No new features on Case paths. |
| Captcha runtime | **Demo only** | No third-party siteverify in runtime; catalog may retain future adapter keys. |
| Dev email code | **Fixed `123456`** | Development skips SMTP; production without SMTP fails closed for mail-dependent flows. |

## Compatibility retirement checklist (before Sunset 2026-12-31)

Do **not** drop tables or remove Case handlers until each step has evidence. Target window: **2026-10 → 2026-12**.

### Already done (as of merge PR #1 / main@0bf2c1f+)

- [x] Deprecation + Sunset headers on legacy Case routes (`docs/api/case-event-migration.md`)
- [x] Event-first public/management APIs and publish path
- [x] Frontend `/cases/[id]` soft-redirects to `/events/[id]`
- [x] Admin UI labels legacy Submission/Case as 兼容窗口
- [x] README API table documents Event-first + legacy Case note
- [x] Product decisions locked (this file + `AGENTS.md`)

### Before removing Case read aliases (recommend ≥30 days notice)

1. [ ] Inventory external callers of `/api/v1/cases/*` and `/api/cases/*` (access logs / support tickets).
2. [ ] Publish migration notice (changelog / README banner) with Sunset date.
3. [ ] Confirm successor fields consumed: Event `details`, `occurred_*`, public Subject + events list.
4. [ ] Keep dual-read smoke tests green for one release cycle after notice.

### Code removal (post-notice, single dedicated PR)

1. [ ] Remove frontend admin Submission/Case tables (or gate behind feature flag already off).
2. [ ] Delete `/cases/[id]` redirect only after no inbound links (or keep permanent redirect).
3. [ ] Remove Case handler routes and deprecation middleware once traffic is zero.
4. [ ] Archive or migrate remaining Case rows into Event (if any dual-write leftovers).
5. [ ] Drop or rename legacy tables **only after** dual-read verification + backup.
6. [ ] Update gap ledgers + tag release with deletion commit SHA.

### Explicitly out of scope for “optional follow-ups” now

- Implementing OAuth provider login UI
- Full dark mode
- OpenAPI code generation
- Public API Key product
