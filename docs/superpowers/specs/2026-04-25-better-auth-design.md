# Better Auth Integration — Design

**Date:** 2026-04-25
**Status:** Spec — pending implementation
**Branch:** `better-auth`

## Goal

Add user-facing authentication to TorrentUI so the app can be safely exposed to the internet without relying on Cloudflare Access. Auth is for a small group (household + close friends). Login is **Google OAuth only**, gated by an **email allowlist** that admins manage in-app.

## Constraints

- Backend is Go (Gin); no DB today.
- Frontend is plain Vite + React (no router, no SSR).
- Better Auth is JS-only — must run in Node, not Go.
- App is hosted on a personal server behind a Cloudflare Tunnel.
- Two bootstrap admins: `d.isayan@gmail.com`, `hasmikatomyan@gmail.com`.

## Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Where Better Auth runs | **Node sidecar** (Hono on Node 20+) alongside the Go backend | Better Auth is JS-only; Hono is the lightest framework with first-class Better Auth support. |
| How Go validates sessions | **Reverse-proxy through the sidecar.** Sidecar authenticates, attaches `X-User-Id/Email/Role` headers, forwards to Go on the internal Docker network. | Single public surface, Go stays auth-agnostic, no JWT/cookie machinery in Go, SSE streams pass through cleanly. |
| Invite mechanism | **Email allowlist.** Admin inserts an email into `invited_emails`; user signs in with Google. If Google's email matches, account created on first sign-in. Otherwise `FORBIDDEN`. | Zero machinery — no SMTP, no codes, no expiring tokens. Inviting = adding a row. |
| Admin model | **Env-var bootstrap admins + in-app `/admin` page.** `BOOTSTRAP_ADMIN_EMAILS` env auto-promotes on first sign-in and reconciles `role='admin'` on every service start. Admins manage invites and users via UI. | No chicken-and-egg, no SSH-to-edit-DB after launch, recoverable from misconfiguration. |
| Cloudflare Access | **Remove it** once Better Auth is verified. App becomes publicly reachable; allowlist is the gate. | Better Auth + allowlist is sufficient. CFA was a stopgap. |
| Per-user data | **None.** All signed-in users share the same view of torrents, storage, etc. `X-User-*` headers are for audit logging only. | Household app — torrents land in shared `/mediastorage`. Per-user ownership is out of scope. |
| Frontend routing | **Add `react-router-dom`.** `/` = existing app, `/admin` = new full page. | A separate page is cleaner UX than a tab once invite/user management grows. |
| Frontend deploy | **Multi-stage Dockerfile** in `auth-service` builds the React bundle and serves the static output. | `docker compose up` is the single source of truth for deploy. |
| DB | **SQLite via `better-sqlite3`**, file at `/data/auth.sqlite`, Docker volume. | Right-sized for a household. Better Auth ships a built-in SQLite adapter. |

## Architecture

```
Internet
   │
   ▼  (Cloudflare Tunnel — no CFA gate)
┌─────────────────────────────────────────────────┐
│ auth-service  (Node + Hono + Better Auth)       │
│   :3000  — sole public surface                  │
│                                                 │
│   - serves React static bundle                  │
│   - /api/auth/*  → Better Auth handler          │
│   - /api/admin/* → in-process admin routes      │
│   - /api/*       → reverse-proxy to Go backend  │
│                   (rejects unauthenticated;     │
│                    on auth, attaches            │
│                    X-User-Id/Email/Role)        │
│                                                 │
│   - SQLite at /data/auth.sqlite (Docker volume) │
└─────────────────────────────────────────────────┘
                    │ HTTP (private Docker network)
                    ▼
┌─────────────────────────────────────────────────┐
│ backend  (Go + Gin) — :8080 internal only       │
│   - trusts X-User-* headers                     │
│   - no public port                              │
└─────────────────────────────────────────────────┘
                    │
                    ▼
       Transmission RPC, /mediastorage, etc.
```

## Stack & Versions

- `better-auth@^1.6.9` (April 2026 stable; avoid 1.7 beta)
- `hono` + `@hono/node-server`
- `better-sqlite3`
- `@better-auth/cli` (devDep, for migrations)
- `react-router-dom@^6` (frontend)
- Node 20 LTS
- Use **pnpm** for the auth-service package manager.

## Repo Layout

```
backend/             # Go (existing) — port no longer published
frontend/            # React (existing) — built static files served by auth-service
auth-service/        # NEW
  src/
    auth.ts          # betterAuth() config
    server.ts        # Hono app
    db.ts            # better-sqlite3 + invited_emails migration + bootstrap reconciliation
    middleware.ts    # requireAuth, requireAdmin
    proxy.ts         # proxyToGo
    admin-routes.ts  # /api/admin/* handlers
  public/            # frontend build output (multi-stage produces this)
  package.json
  tsconfig.json
  Dockerfile         # multi-stage: build frontend, build auth-service, runtime
  Dockerfile.dev
data/                # NEW — auth.sqlite, mounted as volume
docker-compose.yml
docker-compose.dev.yml
docs/superpowers/specs/2026-04-25-better-auth-design.md
```

## Data Model

### Better Auth tables (created by `@better-auth/cli migrate`)
- `user` — `id`, `email`, `name`, `image`, `emailVerified`, `createdAt`, `updatedAt`, plus admin-plugin columns: `role`, `banned`, `banReason`, `banExpires`, `impersonatedBy`
- `session` — `id`, `userId`, `expiresAt`, `token`, `ipAddress`, `userAgent`, `impersonatedBy`
- `account` — OAuth account links
- `verification` — Better Auth internal

### Owned table (managed in `db.ts`, not by Better Auth's schema)
```sql
CREATE TABLE IF NOT EXISTS invited_emails (
  email      TEXT PRIMARY KEY,
  invited_by TEXT,
  created_at INTEGER NOT NULL DEFAULT (unixepoch())
);
```

Rows are **kept** after the user signs up (single source of truth for "who's allowed in").

## Better Auth Config (auth.ts)

```ts
import { betterAuth } from "better-auth";
import { admin } from "better-auth/plugins";
import { APIError } from "better-auth/api";
import Database from "better-sqlite3";
import { db } from "./db";

const bootstrapAdmins = (process.env.BOOTSTRAP_ADMIN_EMAILS ?? "")
  .split(",").map(s => s.trim().toLowerCase()).filter(Boolean);

export const auth = betterAuth({
  database: db,
  baseURL: process.env.BETTER_AUTH_URL!,
  secret: process.env.BETTER_AUTH_SECRET!,
  trustedOrigins: [process.env.BETTER_AUTH_URL!],
  useSecureCookies: process.env.NODE_ENV === "production",

  socialProviders: {
    google: {
      clientId: process.env.GOOGLE_CLIENT_ID!,
      clientSecret: process.env.GOOGLE_CLIENT_SECRET!,
      prompt: "select_account",
      scopes: ["openid", "email", "profile"],
      mapProfileToUser: (profile) => {
        if (!profile.email) {
          throw new APIError("BAD_REQUEST", { message: "Google profile missing email" });
        }
        return { email: profile.email, name: profile.name, image: profile.picture };
      },
    },
  },

  databaseHooks: {
    user: {
      create: {
        before: async (user) => {
          const email = user.email.toLowerCase();
          const isBootstrap = bootstrapAdmins.includes(email);
          const isInvited = !!db.prepare(
            "SELECT 1 FROM invited_emails WHERE lower(email) = ?"
          ).get(email);

          if (!isBootstrap && !isInvited) {
            throw new APIError("FORBIDDEN", { message: "Email not on allowlist" });
          }
          return { data: { ...user, role: isBootstrap ? "admin" : "user" } };
        },
      },
    },
  },

  session: {
    expiresIn: 60 * 60 * 24 * 30,           // 30 days
    updateAge: 60 * 60 * 24,                // refresh once a day
    cookieCache: { enabled: true, maxAge: 300 }, // 5-min lookup cache
  },

  advanced: {
    defaultCookieAttributes: { sameSite: "lax", httpOnly: true, secure: true },
  },

  plugins: [admin({ defaultRole: "user", adminRoles: ["admin"] })],
});
```

## Bootstrap-Admin Reconciliation (db.ts startup)

Runs after CLI migrations and `invited_emails` table creation, on every service boot:

```ts
const stmt = db.prepare("UPDATE user SET role = 'admin' WHERE lower(email) = ?");
for (const email of bootstrapAdmins) stmt.run(email);
```

Guarantees env-listed emails are always admins. Demoting a bootstrap admin requires removing them from env and manually flipping `role` in DB (rare, household-scale operation).

## Hono Server (server.ts)

```ts
const app = new Hono();

// 1. Better Auth
app.on(["GET", "POST"], "/api/auth/*", (c) => auth.handler(c.req.raw));

// 2. Custom admin endpoints — only invite management (user mgmt
//    uses Better Auth's built-in /api/auth/admin/* via the admin
//    plugin, called from the frontend through authClient.admin.*)
app.use("/api/admin/*", requireAuth, requireAdmin);
app.get("/api/admin/invites", listInvites);
app.post("/api/admin/invites", addInvite);
app.delete("/api/admin/invites/:email", removeInvite);   // ?revokeSessions=true also revokes active sessions for that email

// 3. Everything else — proxy to Go
app.use("/api/*", requireAuth);
app.all("/api/*", proxyToGo);

// 4. Static SPA + history-mode fallback
app.get("/health", (c) => c.json({ status: "ok" }));
app.use("/*", serveStatic({ root: "./public" }));
app.notFound((c) => c.html(readFileSync("./public/index.html", "utf8")));

serve({ fetch: app.fetch, port: Number(process.env.PORT ?? 3000) });
```

## requireAuth / requireAdmin (middleware.ts)

```ts
export const requireAuth = createMiddleware(async (c, next) => {
  const session = await auth.api.getSession({ headers: c.req.raw.headers });
  if (!session) return c.json({ error: "unauthorized" }, 401);
  c.set("user", session.user);
  c.set("session", session.session);
  await next();
});

export const requireAdmin = createMiddleware(async (c, next) => {
  const user = c.get("user");
  if (user.role !== "admin") return c.json({ error: "forbidden" }, 403);
  await next();
});
```

## proxyToGo (proxy.ts)

```ts
const GO_BACKEND = process.env.GO_BACKEND_URL ?? "http://backend:8080";

export const proxyToGo = async (c: Context) => {
  const user = c.get("user") as User;
  const url = new URL(c.req.url);
  const target = GO_BACKEND + url.pathname.replace(/^\/api/, "") + url.search;

  const headers = new Headers(c.req.raw.headers);
  headers.delete("cookie");
  headers.delete("x-user-id");
  headers.delete("x-user-email");
  headers.delete("x-user-role");
  headers.set("x-user-id", user.id);
  headers.set("x-user-email", user.email);
  headers.set("x-user-role", user.role ?? "user");

  return fetch(target, {
    method: c.req.method,
    headers,
    body: c.req.method === "GET" || c.req.method === "HEAD"
      ? undefined
      : c.req.raw.body,
    duplex: "half",
  });
};
```

**Header-spoof defense:** strips inbound `x-user-*` and `cookie` before re-attaching trusted values. Go has no public port; spoofing requires Docker-network access.

## Admin Endpoints — Self-Modification Guard

Two layers:

1. **Custom invite routes** (`/api/admin/invites/*`): trivially safe — these target an email string, not a user record.

2. **Better Auth admin plugin endpoints** (`/api/auth/admin/*`): wrap the admin plugin with a `before` hook that rejects self-targeting calls.

```ts
admin({
  defaultRole: "user",
  adminRoles: ["admin"],
  // applied via the plugin's hooks layer, not the global one
  // — pseudocode; see Better Auth docs for exact admin-plugin hook API
  before: async (ctx) => {
    const targetId = ctx.body?.userId ?? ctx.params?.id;
    if (targetId && targetId === ctx.session?.user.id) {
      throw new APIError("BAD_REQUEST", { message: "cannot modify yourself" });
    }
  },
});
```

If the admin plugin doesn't expose a per-endpoint hook, fall back to a top-level `hooks.before` (`createAuthMiddleware`) gated on `ctx.path.startsWith("/admin/")`. **The implementation plan must verify the exact hook surface against Better Auth's current admin-plugin docs and pick the cleanest one.**

Applies to: `setRole`, `banUser`, `removeUser`, `revokeUserSessions`. Last-resort recovery is the env-var bootstrap reconciliation on restart.

## Go Backend Changes

New file `backend/middleware/auth.go`:

```go
func RequireUser() gin.HandlerFunc {
    return func(c *gin.Context) {
        id := c.GetHeader("X-User-Id")
        email := c.GetHeader("X-User-Email")
        if id == "" || email == "" {
            c.AbortWithStatusJSON(401, gin.H{"error": "no user header"})
            return
        }
        c.Set("userId", id)
        c.Set("userEmail", email)
        c.Set("userRole", c.GetHeader("X-User-Role"))
        c.Next()
    }
}
```

In `main.go`, wrap all existing routes except `/health` in a `RequireUser()`-protected group.

## Frontend Changes

### Auth client (frontend/src/lib/auth-client.ts)
```ts
import { createAuthClient } from "better-auth/react";
import { adminClient } from "better-auth/client/plugins";

export const authClient = createAuthClient({
  baseURL: "/api/auth",
  plugins: [adminClient()],
});
export const { signIn, signOut, useSession } = authClient;
```

### App root (frontend/src/App.tsx)
```tsx
function App() {
  const { data: session, isPending } = useSession();
  if (isPending) return <SplashSpinner />;
  if (!session)  return <LoginScreen />;

  return (
    <BrowserRouter>
      <AppShell user={session.user}>
        <Routes>
          <Route path="/"      element={<Home />} />
          <Route path="/admin" element={
            session.user.role === "admin"
              ? <AdminPage />
              : <Navigate to="/" replace />
          } />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </AppShell>
    </BrowserRouter>
  );
}
```

### `<LoginScreen>`
Single-button page; reads `?error=` for "not on allowlist" messaging.
```tsx
<Button onClick={() => signIn.social({
  provider: "google",
  callbackURL: "/",
  errorCallbackURL: "/?error=not-allowlisted",
})}>Sign in with Google</Button>
```

### `<AppShell>`
Layout component: header (avatar, name, sign-out, "Admin" link if `role === "admin"`) + `{children}`.

### `<Home>`
The current `App.tsx` content (Download / PirateBay / RuTracker / Storage tabs), unchanged.

### `<AdminPage>`
Full-page view with two stacked sections:
- **Allowlist** — table of `invited_emails` (custom `/api/admin/invites` endpoints); "Add email" form; per-row delete with a confirm dialog containing a checkbox "Also revoke active sessions for this email" (passes `?revokeSessions=true`).
- **Users** — uses Better Auth's admin client (`authClient.admin.listUsers`, `setRole`, `banUser`, `unbanUser`, `revokeUserSessions`, `removeUser`) directly. No custom `/api/admin/users/*` routes — the admin plugin already exposes `/api/auth/admin/*`.

Self-modification guard lives both in the UI (disable the row's destructive actions when `target.id === currentUser.id`) **and** in a custom `before` hook on the admin plugin endpoints (defense in depth — UI gating is not authoritative).

Built with existing shadcn/Radix primitives.

### services.tsx
- Default `credentials: "include"` on all fetches
- Centralize 401 handler → `signOut()` + re-render to `<LoginScreen>` + toast "Session expired"

## Configuration

### auth-service/.env
```bash
BETTER_AUTH_SECRET=<openssl rand -hex 32>
BETTER_AUTH_URL=https://your-public-domain.tld
GOOGLE_CLIENT_ID=...apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=...
BOOTSTRAP_ADMIN_EMAILS=d.isayan@gmail.com,hasmikatomyan@gmail.com
GO_BACKEND_URL=http://backend:8080
DATABASE_PATH=/data/auth.sqlite
NODE_ENV=production
PORT=3000
```

### Google Cloud OAuth Client (one-time, manual)
1. Console → APIs & Services → Credentials → Create OAuth 2.0 Client ID
2. Type: Web application
3. Authorized JS origins: `https://your-public-domain.tld`, `http://localhost:3000` (dev)
4. Redirect URIs: `https://your-public-domain.tld/api/auth/callback/google`, `http://localhost:3000/api/auth/callback/google`
5. Copy ClientID/Secret to `.env`
6. OAuth consent screen: External, Testing mode (under 100 testers cap forever)

### docker-compose.yml (prod)
- `auth-service`: build `./auth-service`, expose `3000:3000`, mount `./data:/data`
- `backend`: drop `ports:`, replace with `expose: ["8080"]`
- Cloudflare Tunnel: repoint hostname from `backend:8080` → `auth-service:3000`

### Vite dev proxy
Update `frontend/vite.config.ts` `VITE_API_TARGET` default from `http://localhost:8085` → `http://localhost:3000`. Auth-service in dev proxies to Go on `:8085`.

### Migrations
- `pnpm --filter auth-service exec @better-auth/cli migrate --config ./src/auth.ts` — creates Better Auth tables
- `invited_emails` table — created idempotently in `db.ts` on every startup (`CREATE TABLE IF NOT EXISTS`)
- Bootstrap-admin reconciliation — runs after both, on every startup

### Makefile additions
```
make auth-dev          # cd auth-service && pnpm dev
make auth-migrate      # run @better-auth/cli migrate
make auth-shell-db     # sqlite3 ./data/auth.sqlite
```

## Edge Cases & Behavior

| Case | Behavior |
|---|---|
| Email not on allowlist | `before` hook throws `FORBIDDEN` → redirect to `/?error=not-allowlisted` → LoginScreen shows clear message |
| Admin removes invite *after* signup | User stays signed in up to `cookieCache.maxAge` (5 min) or `expiresIn` (30 days). UI provides one-click "Revoke sessions" alongside delete. |
| OAuth profile missing email | `mapProfileToUser` throws BAD_REQUEST early |
| Session expires mid-request | 401 → frontend signs out, shows toast |
| Go backend down | Proxy `fetch` rejects → 502 → frontend toast |
| SSE streams (`/scrape/*/stream`) | Same-origin EventSource carries cookies; Hono streams `Response` body through; no special handling |
| Admin self-modify | 400 from admin handler |
| All admins lost | Restart service → bootstrap reconciliation re-promotes env-listed admins |
| Banned user with cached cookie | Up to 5-min staleness; "Revoke sessions" bypasses cache |

## CSRF Posture

- Better Auth's `/api/auth/*` ships built-in CSRF protection.
- `/api/admin/*` and proxied `/api/*`: same-origin only, `HttpOnly` + `SameSite=Lax` cookies. Sufficient for this threat model.

## Logging

- Auth-service `info` events: sign-in success (email + role), sign-in rejected (email + reason), session revoked (target + by), invite added/removed (target + by), role change.
- No request bodies, no tokens.
- Go backend: `request_id + user_email` per request.
- Stdout only (Docker log driver).

## Testing

Project has no test suite today. Adding minimum viable for security-critical code:

| Layer | Tests | Tool |
|---|---|---|
| `databaseHooks.user.create.before` | allowlisted passes; non-allowlisted throws FORBIDDEN; bootstrap email gets `role:'admin'` | `node:test` + ephemeral SQLite |
| `proxyToGo` header handling | inbound `x-user-*` and `cookie` stripped; session-derived values attached | `node:test` |
| `/api/admin/*` | non-admin → 403; admin → 200; self-modify → 400 | `node:test` integration with seeded SQLite |
| Go `RequireUser` middleware | missing headers → 401; valid → context populated | `go test` |
| End-to-end | manual checklist (4 paths: signed-out, signed-in non-admin, signed-in admin, allowlisted-but-not-yet-signed-up) | manual |

No frontend tests. Real SQLite per test (file-per-test) — no mocks for the DB.

## Out of Scope

- Multi-tenant / per-user data ownership
- Magic-link or email-OTP login
- Other OAuth providers (GitHub, etc.)
- 2FA / passkeys
- Self-service signup (allowlist is the signup gate)
- Rate limiting on auth endpoints (Better Auth's defaults are fine for this audience size)
- SMTP — no transactional email

## Rollout

1. Implement auth-service end-to-end against a dev SQLite + Google OAuth test client
2. Wire Go middleware + drop public port in compose
3. Wire frontend (router, login, admin page)
4. Deploy alongside CFA still active — verify full flow works
5. Repoint Cloudflare Tunnel from `backend:8080` to `auth-service:3000`
6. Remove CFA gate on the Cloudflare hostname
7. Done

A separate implementation plan will break this into ordered, individually-testable steps.
