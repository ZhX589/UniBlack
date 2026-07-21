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

## Compatibility retirement checklist (before Sunset)

1. Confirm no external clients depend on `/api/v1/cases/*` without successor headers.  
2. Remove or redirect frontend `/cases/[id]` and admin Submission/Case tables.  
3. Drop or archive legacy tables only after dual-read verification.  
4. Update README public API table to Event-only.  
5. Bump gap ledgers with deletion commit evidence.
