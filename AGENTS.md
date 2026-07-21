# AGENTS.md

Guidance for OpenCode / MiMoCode sessions working in UniBlack.

## Project

UniBlack is a reusable "cloud blacklist" system (community-maintained list of untrusted accounts with evidence, appeals, and audit trails).

**Current state (2026-07-21):** monorepo with working backend, frontend, migrations, CI, and production Compose smoke evidence on `main`. Progress ledgers:

- `docs/implementation-gap-analysis.md` — full-stack progress vs design goals
- `docs/frontend-gap-analysis.md` — Phase 12 frontend ledger
- `docs/roadmap.md` — phase goals (Goal bodies are planning; Status sections track progress)
- `docs/api/case-event-migration.md` — Case → Event deprecation (Sunset 2026-12-31)

- Backend: Go + Echo, GORM (read/write only), PostgreSQL, golang-migrate
- Frontend: React + Next.js (App Router), Tailwind CSS, minimal shared UI
- Auth: JWT + Refresh Token; OAuth2/OIDC reserved via `AuthProvider` (not implemented)
- Storage: `Storage` interface — LocalStorage (dev) / MinIO S3-compatible (prod)
- Docs: `README.md`, `docs/roadmap.md`, gap ledgers under `docs/`

## Branch / Git conventions (from README)

Use simplified Git Flow — **do not commit directly to `main`**:

- features → branch under `feature/`
- fixes → branch under `fix/`
- docs → branch under `docs/`
- workflow: develop → open PR → merge to main → delete branch → tag

## Development principles (from docs/roadmap.md)

- Order matters: stabilize project structure first, then database, then auth, then domain model, then features, then frontend. Avoid redesigning early layers.
- **Use database migrations (golang-migrate), not GORM auto-create tables.** Migrations must be repeatable and support rollback.
- Core domain entity is **`Subject`** (a person/account). Canonical publish model is **Account + Event** (Phase 13). Legacy `Case` / `Identifier` / `Submission` remain only inside the compatibility window ending **2026-12-31**.
- A Subject has many **Accounts** (platform, username, account_id, custom attributes). Identifiers are legacy compatibility reads.
- Evidence metadata lives in DB; binary files go through `Storage` (Local or MinIO/S3).
- Local-first development; Docker Compose is for production packaging and full-stack smoke.

## Locked conventions (do not change without discussion)

See `docs/roadmap.md` § "Conventions & Decisions" for full detail. Key points:

- **Naming**: project is `UniBlack` everywhere (code, modules, env, images). Never `CloudBan`.
- **Migration tool**: golang-migrate; SQL files in `backend/internal/migrations/` (`*.up.sql` / `*.down.sql`).
- **Auth**: define an `AuthProvider` interface; JWT is the first impl. `User` keeps `auth_provider` + `external_id` for future OAuth/OIDC. Don't hardcode login to JWT only.
- **Layering** (backend): `handler -> service -> repository -> db/models`, single-direction; external deps (DB, storage, auth) injected via interfaces.
- **Structure**: `backend/` (Go/Echo, layered) and `frontend/` (Next.js App Router) as monorepo roots.
- **Frontend API boundary**: pages must not read tokens, set Bearer headers, or call `fetch` directly; use `lib/api.ts` + Auth/Settings providers.
- **Captcha / email**: runtime uses demo captcha only; development email code is fixed `123456`; production without SMTP must fail closed for mail-dependent flows.

## Product decisions (locked 2026-07-21)

| Topic | Decision |
| --- | --- |
| Public API Key (historical Phase 8) | **Not implemented.** Public read is rate-limited JWT-optional endpoints; mutating APIs require session JWT + RBAC. Revisit only with a dedicated security spec. |
| OpenAPI generator (historical Phase 8) | **Deferred.** README + handler routes + `docs/api/*` are the current contract; auto-gen is a future docs tooling task, not a release blocker. |
| OAuth (Phase 10) | **Reserved only** (`AuthProvider` + settings keys). No provider login UI/flow until a dedicated phase. |
| Dark mode | **Out of scope** for Phase 12; keep light theme + `prefers-reduced-motion`. |
| Legacy Case/Submission UI | **Compatibility window** until Sunset 2026-12-31; no new features on Case paths. |

## Notes for agents

- Code, tests, Docker, and CI **do exist** — prefer reading gap ledgers and `main` over bootstrap assumptions.
- `.env` is gitignored; `.env.example` exists under repo root, `backend/`, and `frontend/` as committed templates.
- Never rewrite compose **Goal** bodies when updating progress; only Status / gap ledgers / checkboxes with evidence.
- Build/run backend from `backend/` (`go build -o /tmp/uniblack ./cmd/server`); monorepo root is not a Go module.
