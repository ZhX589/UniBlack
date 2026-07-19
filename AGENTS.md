# AGENTS.md

Guidance for OpenCode sessions working in UniBlack.

## Project

UniBlack is a reusable "cloud blacklist" system (community-maintained list of untrusted accounts with evidence, appeals, and audit trails). Currently in bootstrap phase — only docs exist; no backend/frontend code yet.

- Backend: Go + Echo, GORM, PostgreSQL
- Frontend: React + Next.js, Tailwind CSS, shadcn/ui
- Auth: JWT + Refresh Token, OAuth2 (planned)
- Docs: `README.md`, `docs/roadmap.md`

## Branch / Git conventions (from README)

Use simplified Git Flow — **do not commit directly to `main`**:
- features → branch under `feature/`
- fixes → branch under `fix/`
- docs → branch under `docs/`
- workflow: develop → open PR → merge to main → delete branch → tag

## Development principles (from docs/roadmap.md)

- Order matters: stabilize project structure first, then database, then auth, then domain model, then features, then frontend. Avoid redesigning early layers.
- **Use database migrations (golang-migrate), not GORM auto-create tables.** Migrations must be repeatable and support rollback.
- Core domain entity is `Subject` (a person/account), not `Case`. Cases, submissions, appeals all attach to a Subject.
- A Subject has many `Identifier`s (QQ, Discord, Telegram, Minecraft UUID, Steam, Email). Identifiers must be unique.
- Evidence is stored separately from Cases; DB holds only metadata, files go to MinIO (S3-compatible).
- Goal: one command (`docker compose up`) starts the whole dev environment (backend + Next.js + PostgreSQL + MinIO).

## Locked conventions (do not change without discussion)

See `docs/roadmap.md` § "Conventions & Decisions" for full detail. Key points:

- **Naming**: project is `UniBlack` everywhere (code, modules, env, images). Never `CloudBan`.
- **Migration tool**: golang-migrate; SQL files in `backend/internal/migrations/` (`*.up.sql` / `*.down.sql`).
- **Auth**: define an `AuthProvider` interface; JWT is the first impl. `User` keeps `auth_provider` + `external_id` for future OAuth/OIDC. Don't hardcode login to JWT only.
- **Layering** (backend): `handler -> service -> repository -> db/models`, single-direction; external deps (DB, storage, auth) injected via interfaces.
- **Structure**: `backend/` (Go/Echo, layered) and `frontend/` (Next.js App Router) as monorepo roots.

## Notes for agents

- No code, tests, or build config exist yet — when scaffolding, follow the stack, directory layout, and phase order in `docs/roadmap.md`.
- `.env` is gitignored; `.env.example` exists under `backend/` and `frontend/` as committed templates.
- Docs are the current source of truth until code lands; trust `docs/roadmap.md` over speculative structure.
