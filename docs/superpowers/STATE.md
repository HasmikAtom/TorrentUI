# Better Auth Integration — Session Handoff State

**Last session:** 2026-04-25
**Branch:** `better-auth`
**Plan:** [`docs/superpowers/plans/2026-04-25-better-auth.md`](plans/2026-04-25-better-auth.md)
**Spec:** [`docs/superpowers/specs/2026-04-25-better-auth-design.md`](specs/2026-04-25-better-auth-design.md)

## Where to pick up

**Next task: Task 5** — TDD allowlist hook (write `auth-service/src/__tests__/hooks.test.ts`).

The plan was updated mid-session as we hit reality. Read the plan top-to-bottom before starting Task 5 — several tasks have been corrected from the original draft (see "Plan corrections baked in" below).

## Tasks complete (4/26)

| # | Task | Commit | Notes |
|---|---|---|---|
| 1 | Initialize auth-service package | `74009673` + fix `a0c99a6` | Initial implementer mistakenly created `src/server.ts` and `.env.example` was silently gitignored by root `.env.*` rule. Both fixed in `a0c99a6`. |
| 2 | Database layer (`db.ts`) | `0fa5d3b` | Clean — exports `db`, `bootstrapAdmins`, `runOwnedMigrations`, `reconcileBootstrapAdmins`. |
| 3 | Better Auth config (`auth.ts`) | `3e24f89` | Implementer had to widen `beforeUserCreate` parameter type to match Better Auth v1.6.9's `User & Record<string, unknown>` shape (full `User` schema includes `createdAt: Date`, `updatedAt: Date`, `emailVerified: boolean`). Plan's Task 5 test code was patched in `22f0e59` to use a `makeUser` factory that produces the full shape. |
| 4 | Programmatic migrator smoke test | (no commit — runs against gitignored `data/auth.sqlite`) | Verified `getMigrations(auth.options).runMigrations()` creates `user`, `session`, `account`, `verification` tables. No CLI involved. |

## Plan corrections baked in (commits `4420b1c`, `9de3768`)

Three real errors were discovered and fixed in the plan:

1. **`@better-auth/cli` is frozen at v1.4.x** and incompatible with `better-auth@1.6.x` (peer-dep skew on `better-call`'s `kAPIErrorHeaderSymbol` export). Plan now drops the CLI entirely. Server boot calls `getMigrations(auth.options).runMigrations()` programmatically (Task 10).

2. **Version floors were too low.** Updated to match Better Auth's own e2e fixture:
   - `better-sqlite3@^12.6.2` (was 11.x — peer of better-auth>=1.4 is ^12)
   - `hono@^4.12.12` (was 4.6 — below floor)
   - `@hono/node-server@^1.19.14` (was 1.13 — below floor)
   - `tsx@^4.21.0`, `typescript@^5.9.3`

3. **pnpm blocks postinstall scripts by default.** `package.json` now has `pnpm.onlyBuiltDependencies = ["better-sqlite3", "esbuild"]` so installs work without manual `pnpm approve-builds`.

## Local dev state to be aware of

- `auth-service/.env` exists locally (gitignored) with a real `BETTER_AUTH_SECRET`. Don't regenerate unless you want to invalidate any sessions you've created during testing.
- `data/auth.sqlite` exists locally with the four Better Auth tables created. `invited_emails` shows up only after the server runs (Task 10).
- `auth-service/node_modules` has the corrected dep versions installed. If you `rm -rf` it, `pnpm install` will re-fetch the same combo (lockfile is checked in).

## Plan tasks remaining

Phase 2 finish: 5
Phase 3: 6, 7, 8, 9, 10 (all backend HTTP layer)
Phase 4: 11, 12 (Go side)
Phase 5: 13–20 (frontend)
Phase 6: 21, 22, 23 (Docker / Vite / Makefile)
Phase 7: 24, 25, 26 (CLAUDE.md, E2E, cutover)

## Recommended next-session approach

The remaining tasks have been validated against reality enough that the implementer-loop should run smoothly:
- TDD security-critical paths (Tasks 5, 7, 8, 11) — keep the full implementer + spec reviewer + code quality reviewer loop.
- Pure config/scaffold tasks (13, 21–24) — direct execution + visual verification, skip the dual reviewers.
- Frontend component tasks (16–20) — implementer + a single visual/typecheck verification.

Use the subagent-driven-development skill to dispatch each task. Plan code blocks are now reliable enough to paste into implementer prompts verbatim.
