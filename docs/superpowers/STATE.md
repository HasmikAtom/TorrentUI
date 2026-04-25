# Better Auth Integration — Session Handoff State

**Last session:** 2026-04-25 (continuation)
**Branch:** `better-auth`
**Plan:** [`docs/superpowers/plans/2026-04-25-better-auth.md`](plans/2026-04-25-better-auth.md)
**Spec:** [`docs/superpowers/specs/2026-04-25-better-auth-design.md`](specs/2026-04-25-better-auth-design.md)

## Where to pick up

**Tasks 5–25 complete.** Stack runs end-to-end via `make dev-build`; Google OAuth + bootstrap admin verified live in browser. Only **Task 26** (production cutover) remains and it's an operator runbook with no code changes.

If picking up for additional cleanup, tackle the open follow-ups (see bottom).

## Tasks complete (25/26)

Tasks 1–4 from the previous session are still on commits `74009673`, `0fa5d3b`, `3e24f89`, plus `a0c99a6` (scaffold fix), `22f0e59` (test-shape patch), `4420b1c` (dep-version patch), `9de3768` (drop @better-auth/cli).

This session added 26 commits for Tasks 5–25:

| Task | Commit | Notes |
|---|---|---|
| 5 | `31d9db3` | TDD allowlist hook tests, 3/3 pass on first run |
| 6 | `ae983d5` | requireAuth + requireAdmin Hono middleware |
| 7 | `43cd8f3` | proxyToGo with header stripping (TDD) |
| 8 | `be638da` | /api/admin/invites endpoints (TDD) |
| 9 | `52de652` + `08c4628` | Self-modify guard. **Plan bug found:** `ctx.context.session` is null at global `hooks.before` lifecycle in Better Auth 1.6.9. Fixed by `await getSessionFromCtx(ctx)` (mirrors what `adminMiddleware` does). |
| 10 | `0342ae1` | Hono server.ts: programmatic migrations on boot, route wiring, conditional static SPA. Smoke test: `/health` returns ok, all 5 SQLite tables present. |
| 11 | `ad067e0` | Go RequireUser middleware with TDD |
| 12 | `5d38f34` | Wrap Go routes with RequireUser group |
| 13 | `529f7f6` | Frontend on pnpm + better-auth + react-router-dom@6 |
| 14 | `58c4b0e` | Better Auth React client with adminClient plugin |
| 15 | `128cf77` | apiFetch centralization (16 fetch sites in 5 components — scope expansion approved by user since plan assumed services.tsx already centralized) |
| 16 | `9e4e67b` | LoginScreen with Google sign-in + allowlist error banner |
| 17 | `53717ba` | AppShell layout |
| 18 | `bbd99cd` | App.tsx router refactor + Home.tsx extract + AdminPage stub |
| 19 | `914b4a2` | AdminPage Allowlist section + shadcn alert-dialog primitive |
| 20 | `c7c1118` | AdminPage Users section |
| 21 | `b2a2aa0` | Multi-stage prod Dockerfile + Dockerfile.dev. **Plan deviation:** added `auth-service/tsconfig.build.json` excluding test files so prod `tsc` succeeds (test files have known TS2769 from Tasks 7/8). `pnpm typecheck` still surfaces the gap. |
| 22 | `3ca9233` | docker-compose: auth-service public, backend internal-only |
| 23 | `9d6ff9b` | Vite proxy → :3000 + Makefile auth targets |
| 24 | `f081f49` | CLAUDE.md updated with new architecture |

## Mid-stream bug fixes (after the bulk of tasks)

Six real bugs caught during E2E that the plan didn't anticipate:

| Commit | Issue |
|---|---|
| `a76a3bf` | Vite proxy stripped `/api` (correct for old Go target, wrong for auth-service which expects /api preserved). Also re-added `frontend-dev` to dev compose for one-command dev. Migrated `frontend/Dockerfile.dev` from npm to pnpm (was still referencing deleted `package-lock.json`). |
| `c255ab1` | `auth.ts` had hardcoded `secure: true` cookies → would block cookies over HTTP in dev. Now follows `NODE_ENV === "production"`. |
| `dc48e3e` | Auth-service in dev compose was publishing host port :3000 (collided with user's other services). Switched to `expose:` only — browser hits Vite at :5173 anyway. |
| `70837f9` | macOS `better-sqlite3` binary leaking into Linux containers via `COPY auth-service/`. Added `.dockerignore` (root + frontend) excluding `node_modules`. |
| `319a8f1` | `auth-client.ts` used relative `baseURL: "/api/auth"` — Better Auth's React client throws `BetterAuthError: Invalid base URL` at module init, crashing the SPA before mount. Fixed to `${window.location.origin}/api/auth`. Caught by chrome-devtools-live MCP E2E agent. |

## How to run

```bash
cd /Users/disayan/code/personal/TorrentUI
make dev-build
# wait for all 3 containers to settle, then open:
open http://localhost:5173
```

Compose auto-loads `/Users/disayan/code/personal/TorrentUI/.env` (gitignored) which has `BETTER_AUTH_SECRET`, `BETTER_AUTH_URL=http://localhost:5173`, `GOOGLE_CLIENT_ID/SECRET`, `BOOTSTRAP_ADMIN_EMAILS=d.isayan@gmail.com,hasmikatomyan@gmail.com`. The same secrets also live in `auth-service/.env` for host-only `pnpm dev` of auth-service alone.

For production, the Google OAuth client has `http://localhost:5173/api/auth/callback/google` (dev) and `http://localhost:3000/api/auth/callback/google` (legacy/host) registered. For prod cutover (Task 26) the operator must add the public domain's redirect URI.

A 1Password item `TorrentUI Google OAuth (dev)` (id `6iurap2yneuvk65y4jz23s25ge`, vault Private) holds the client_id + client_secret with the bootstrap admin context in notes.

## E2E verification done (Task 25)

All four paths from the plan verified end-to-end via chrome-devtools-live MCP + manual sign-in:

- LoginScreen renders with correct copy and "Sign in with Google" button.
- `?error=not-allowlisted` shows the rejection banner verbatim.
- Sign-in flow drives to Google with correct `client_id`, `redirect_uri=http://localhost:5173/api/auth/callback/google`, `scope=openid email profile`, `prompt=select_account`.
- After Google callback, browser lands at `/`, AppShell shows the user's name, "Admin" link visible (bootstrap admin reconciled to role=admin).
- `/admin` while unauthenticated renders LoginScreen (URL stays at `/admin` — see follow-up below).
- API gating: `GET /api/torrents` and `/api/admin/invites` both 401 when no cookie.

Manual: signed in successfully as `d.isayan@gmail.com`, AppShell + admin link rendered correctly.

## Open follow-ups (none blocking)

- **TS2769 in auth-service tests:** `c.set("user", ...)` against untyped `Hono()` in `proxy.test.ts` and `admin-routes.test.ts`. Worked around in Task 21 by separate `tsconfig.build.json` for emit. Root cause needs a `declare module "hono"` augmentation or per-instance typed Variables.
- **Router under session gate:** `BrowserRouter` only mounts after `session != null` in `App.tsx`. URL stays at `/admin` instead of redirecting to `/` when unauthenticated. Hoist the router above the gate to fix.
- **Lost UI elements:** the hexagon logo and `<ThemeToggle />` from the old page header are gone. AppShell provides a new header with text brand, admin link, and user dropdown — but no logo and no theme toggle. Add to AppShell if desired.
- **Sign-out is fire-and-forget:** `onSelect={() => signOut()}` swallows errors. AdminPage `add()` and `remove()` similarly don't surface non-401 failures.
- **Go middleware trust boundary:** `RequireUser()` reads `X-User-Id`/`X-User-Email` from headers without verifying request provenance. Relies on Docker network isolation (Go's port not exposed). Defense-in-depth: shared internal secret header.
- **Unused frontend prod Dockerfiles:** `frontend/Dockerfile` and `frontend/Dockerfile.dev` (the latter still in use by dev compose; the former unused in prod since auth-service serves the SPA) — `frontend/Dockerfile` is a candidate for deletion as cleanup.
- **`backend-dev` host port `:8081` dropped** by Task 22. Re-add `ports: - "8081:8080"` if you want direct host-side debugging during dev.
- **Tasks 7 + 8 reviewer notes:** plan-prescribed `(ctx.body as any)` and `Hono<any>` in admin-routes — typed alternatives noted but not applied to keep plan-verbatim.

## Task 26 (operator runbook — only thing left)

1. Generate prod `BETTER_AUTH_SECRET`: `openssl rand -hex 32`. Store in your secret manager.
2. Update Google OAuth client: add `https://<your-public-domain>` as authorized origin and `https://<your-public-domain>/api/auth/callback/google` as redirect URI.
3. `make prod-build-deploy`. Migrations run automatically on first boot; logs should show no errors.
4. Repoint Cloudflare Tunnel from the previous service to `http://auth-service:3000`. Restart cloudflared.
5. Verify: hit your public URL signed-out → LoginScreen → sign in with `d.isayan@gmail.com` → app loads.
6. Once verified, remove the Cloudflare Access policy in front of the public hostname.
