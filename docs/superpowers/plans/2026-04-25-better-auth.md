# Better Auth Integration — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add Google OAuth authentication and an email-allowlist invite model to TorrentUI by introducing a Node sidecar that runs Better Auth, reverse-proxies to the existing Go backend, and serves the React frontend.

**Architecture:** Node + Hono + Better Auth sidecar (`auth-service`) becomes the sole public surface on `:3000`. It serves the static React bundle, owns `/api/auth/*` and `/api/admin/*`, and reverse-proxies the rest of `/api/*` to the Go backend after attaching `X-User-Id/Email/Role` headers. The Go backend's port is no longer published — it's reachable only on the internal Docker network. SQLite stores Better Auth's tables plus our custom `invited_emails` allowlist.

**Tech Stack:** `better-auth@^1.6.9`, `hono` + `@hono/node-server`, `better-sqlite3`, `react-router-dom@^6`, Node 20, `pnpm` for the new service, existing Go/Gin backend, existing Vite/React frontend.

**Spec:** `docs/superpowers/specs/2026-04-25-better-auth-design.md`

---

## File Structure

**New (`auth-service/`):**
- `auth-service/package.json` — pnpm-managed Node service
- `auth-service/tsconfig.json` — TypeScript config (strict, ESM)
- `auth-service/src/db.ts` — better-sqlite3 instance, `invited_emails` migration, bootstrap-admin reconciliation
- `auth-service/src/auth.ts` — `betterAuth({...})` config including allowlist hook + admin plugin
- `auth-service/src/middleware.ts` — `requireAuth`, `requireAdmin`
- `auth-service/src/proxy.ts` — `proxyToGo` reverse-proxy handler
- `auth-service/src/admin-routes.ts` — `/api/admin/invites/*` handlers
- `auth-service/src/server.ts` — Hono app entry point
- `auth-service/src/__tests__/hooks.test.ts` — allowlist hook unit tests
- `auth-service/src/__tests__/proxy.test.ts` — proxy header-handling unit tests
- `auth-service/src/__tests__/admin-routes.test.ts` — admin endpoint integration tests
- `auth-service/Dockerfile` — multi-stage: build frontend, build TS, runtime
- `auth-service/Dockerfile.dev` — dev image (tsx watch)
- `auth-service/.env.example`
- `auth-service/.gitignore`

**New (`backend/`):**
- `backend/middleware/auth.go` — `RequireUser()` Gin middleware
- `backend/middleware/auth_test.go` — unit tests for the middleware

**Modified:**
- `backend/main.go` — wrap routes in `RequireUser()` group; `/health` stays public
- `frontend/package.json` — add `better-auth`, `react-router-dom`
- `frontend/src/lib/auth-client.ts` — NEW
- `frontend/src/App.tsx` — replaced with router + session gating
- `frontend/src/components/LoginScreen.tsx` — NEW
- `frontend/src/components/AppShell.tsx` — NEW
- `frontend/src/components/Home.tsx` — extracted from old `App.tsx`
- `frontend/src/components/AdminPage.tsx` — NEW
- `frontend/src/services.tsx` — default `credentials: "include"`, central 401 handler
- `frontend/vite.config.ts` — `VITE_API_TARGET` default flips to `http://localhost:3000`
- `docker-compose.yml` — add `auth-service`, drop `backend`'s public port, add `./data` volume
- `docker-compose.dev.yml` — corresponding dev wiring
- `Makefile` — add `auth-dev`, `auth-migrate`, `auth-shell-db`
- `CLAUDE.md` — document new auth-service component (one paragraph)

**New (`data/`):**
- `data/.gitkeep` — directory committed; `auth.sqlite` is gitignored

---

## Phase 1 — auth-service skeleton

### Task 1: Initialize the auth-service package

**Files:**
- Create: `auth-service/package.json`
- Create: `auth-service/tsconfig.json`
- Create: `auth-service/.gitignore`
- Create: `auth-service/.env.example`

- [ ] **Step 1: Create the package directory and `package.json`**

```bash
mkdir -p auth-service/src/__tests__
cd auth-service
```

`auth-service/package.json`:
```json
{
  "name": "torrentui-auth-service",
  "version": "0.1.0",
  "private": true,
  "type": "module",
  "scripts": {
    "dev": "tsx watch src/server.ts",
    "build": "tsc",
    "start": "node dist/server.js",
    "test": "node --test --import tsx src/__tests__/*.test.ts",
    "migrate": "better-auth migrate --config src/auth.ts",
    "typecheck": "tsc --noEmit"
  },
  "dependencies": {
    "better-auth": "^1.6.9",
    "better-sqlite3": "^11.5.0",
    "hono": "^4.6.0",
    "@hono/node-server": "^1.13.0"
  },
  "devDependencies": {
    "@better-auth/cli": "^1.6.9",
    "@types/better-sqlite3": "^7.6.0",
    "@types/node": "^20.0.0",
    "tsx": "^4.19.0",
    "typescript": "^5.6.0"
  },
  "engines": { "node": ">=20" }
}
```

- [ ] **Step 2: Create `tsconfig.json`**

`auth-service/tsconfig.json`:
```json
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "ESNext",
    "moduleResolution": "Bundler",
    "outDir": "dist",
    "rootDir": "src",
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "forceConsistentCasingInFileNames": true,
    "resolveJsonModule": true,
    "isolatedModules": true
  },
  "include": ["src/**/*"]
}
```

- [ ] **Step 3: Create `.gitignore`**

`auth-service/.gitignore`:
```
node_modules
dist
.env
*.log
```

- [ ] **Step 4: Create `.env.example`**

`auth-service/.env.example`:
```bash
BETTER_AUTH_SECRET=<openssl rand -hex 32>
BETTER_AUTH_URL=http://localhost:3000
GOOGLE_CLIENT_ID=
GOOGLE_CLIENT_SECRET=
BOOTSTRAP_ADMIN_EMAILS=d.isayan@gmail.com,hasmikatomyan@gmail.com
GO_BACKEND_URL=http://localhost:8085
DATABASE_PATH=./data/auth.sqlite
NODE_ENV=development
PORT=3000
```

- [ ] **Step 5: Install dependencies**

Run: `cd auth-service && pnpm install`
Expected: `Done in <Xs>` with no peer-dep errors.

- [ ] **Step 6: Verify TypeScript compiles**

Run: `pnpm --filter auth-service typecheck` (or `cd auth-service && pnpm typecheck`)
Expected: no errors (the project has no source files yet, but `tsc` should still exit 0).

- [ ] **Step 7: Commit**

```bash
git add auth-service/package.json auth-service/tsconfig.json auth-service/.gitignore auth-service/.env.example auth-service/pnpm-lock.yaml
git commit -m "scaffold auth-service package"
```

---

### Task 2: Database layer (`db.ts`)

**Files:**
- Create: `auth-service/src/db.ts`
- Create: `data/.gitkeep`
- Modify: `.gitignore` (root) — add `data/auth.sqlite*`

- [ ] **Step 1: Create the `data/` directory and gitkeep**

```bash
mkdir -p data
touch data/.gitkeep
```

- [ ] **Step 2: Update root `.gitignore` to ignore SQLite files**

Append to `.gitignore`:
```
data/auth.sqlite
data/auth.sqlite-journal
data/auth.sqlite-shm
data/auth.sqlite-wal
```

- [ ] **Step 3: Create `db.ts` with bootstrap reconciliation**

`auth-service/src/db.ts`:
```ts
import Database from "better-sqlite3";
import { mkdirSync } from "node:fs";
import { dirname } from "node:path";

const dbPath = process.env.DATABASE_PATH ?? "./data/auth.sqlite";

mkdirSync(dirname(dbPath), { recursive: true });

export const db = new Database(dbPath);
db.pragma("journal_mode = WAL");
db.pragma("foreign_keys = ON");

export const bootstrapAdmins = (process.env.BOOTSTRAP_ADMIN_EMAILS ?? "")
  .split(",")
  .map((s) => s.trim().toLowerCase())
  .filter(Boolean);

export function runOwnedMigrations() {
  db.exec(`
    CREATE TABLE IF NOT EXISTS invited_emails (
      email      TEXT PRIMARY KEY,
      invited_by TEXT,
      created_at INTEGER NOT NULL DEFAULT (unixepoch())
    );
  `);
}

export function reconcileBootstrapAdmins() {
  const stmt = db.prepare(
    "UPDATE user SET role = 'admin' WHERE lower(email) = ?"
  );
  for (const email of bootstrapAdmins) {
    try {
      stmt.run(email);
    } catch {
      // user table may not exist yet on very first boot before migrate ran
    }
  }
}
```

- [ ] **Step 4: Commit**

```bash
git add auth-service/src/db.ts data/.gitkeep .gitignore
git commit -m "add auth-service db module with invited_emails migration and admin reconciliation"
```

---

## Phase 2 — Better Auth config

### Task 3: Better Auth config (`auth.ts`)

**Files:**
- Create: `auth-service/src/auth.ts`

- [ ] **Step 1: Create `auth.ts`**

`auth-service/src/auth.ts`:
```ts
import { betterAuth } from "better-auth";
import { admin } from "better-auth/plugins";
import { APIError } from "better-auth/api";
import { db, bootstrapAdmins } from "./db.js";

export async function beforeUserCreate(user: { id: string; email: string; name?: string }) {
  const email = user.email.toLowerCase();
  const isBootstrap = bootstrapAdmins.includes(email);
  const isInvited = !!db
    .prepare("SELECT 1 FROM invited_emails WHERE lower(email) = ?")
    .get(email);

  if (!isBootstrap && !isInvited) {
    throw new APIError("FORBIDDEN", { message: "Email not on allowlist" });
  }
  return { data: { ...user, role: isBootstrap ? "admin" : "user" } };
}

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
    user: { create: { before: beforeUserCreate } },
  },

  session: {
    expiresIn: 60 * 60 * 24 * 30,
    updateAge: 60 * 60 * 24,
    cookieCache: { enabled: true, maxAge: 300 },
  },

  advanced: {
    defaultCookieAttributes: { sameSite: "lax", httpOnly: true, secure: true },
  },

  plugins: [admin({ defaultRole: "user", adminRoles: ["admin"] })],
});

export type AuthUser = NonNullable<
  Awaited<ReturnType<typeof auth.api.getSession>>
>["user"];
```

- [ ] **Step 2: Verify it typechecks**

Run: `cd auth-service && pnpm typecheck`
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add auth-service/src/auth.ts
git commit -m "configure Better Auth with Google OAuth, allowlist hook, admin plugin"
```

---

### Task 4: Generate Better Auth schema

**Files:** none — runs the CLI to mutate the SQLite file.

- [ ] **Step 1: Set up an env file for local dev**

```bash
cp auth-service/.env.example auth-service/.env
# Set BETTER_AUTH_SECRET to a real value:
sed -i.bak "s|<openssl rand -hex 32>|$(openssl rand -hex 32)|" auth-service/.env
rm auth-service/.env.bak
```

(Leave `GOOGLE_CLIENT_ID` and `GOOGLE_CLIENT_SECRET` empty for now — they're only needed for the actual sign-in flow.)

- [ ] **Step 2: Run the migrate CLI**

Run:
```bash
cd auth-service
pnpm exec @better-auth/cli@latest migrate --config src/auth.ts --y
```
Expected: prints "Migration complete." and creates `user`, `session`, `account`, `verification` tables in `data/auth.sqlite`.

- [ ] **Step 3: Verify tables exist**

Run: `sqlite3 data/auth.sqlite ".tables"`
Expected output: `account  invited_emails  session  user  verification`
(`invited_emails` won't exist yet — it's created by `runOwnedMigrations()` at server startup, not by the CLI. Re-check after Task 10 when the server boots.)

- [ ] **Step 4: Commit (no file changes — just confirms the migrate workflow)**

This task creates no committed artifacts. Verify the workflow works, then proceed.

---

### Task 5: TDD — allowlist hook

**Files:**
- Create: `auth-service/src/__tests__/hooks.test.ts`

- [ ] **Step 1: Write the failing tests**

`auth-service/src/__tests__/hooks.test.ts`:
```ts
import { test, before, after } from "node:test";
import assert from "node:assert/strict";
import { mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

let tmpDir: string;

before(() => {
  tmpDir = mkdtempSync(join(tmpdir(), "ba-test-"));
  process.env.DATABASE_PATH = join(tmpDir, "auth.sqlite");
  process.env.BETTER_AUTH_SECRET = "test-secret-test-secret-test-secret-test";
  process.env.BETTER_AUTH_URL = "http://localhost:3000";
  process.env.GOOGLE_CLIENT_ID = "test-id";
  process.env.GOOGLE_CLIENT_SECRET = "test-secret";
  process.env.BOOTSTRAP_ADMIN_EMAILS = "boss@example.com";
});

after(() => rmSync(tmpDir, { recursive: true, force: true }));

test("allowlisted email is permitted, role=user", async () => {
  const { db, runOwnedMigrations } = await import("../db.js");
  const { beforeUserCreate } = await import("../auth.js");
  runOwnedMigrations();
  db.prepare("INSERT OR IGNORE INTO invited_emails (email) VALUES (?)").run("alice@example.com");

  const result = await beforeUserCreate({
    id: "u1", email: "alice@example.com", name: "Alice",
  });
  assert.equal(result.data.role, "user");
});

test("non-allowlisted email throws FORBIDDEN", async () => {
  const { beforeUserCreate } = await import("../auth.js");
  await assert.rejects(
    () => beforeUserCreate({ id: "u2", email: "stranger@example.com", name: "Eve" }),
    /Email not on allowlist/
  );
});

test("bootstrap admin email gets role=admin", async () => {
  const { beforeUserCreate } = await import("../auth.js");
  const result = await beforeUserCreate({
    id: "u3", email: "BOSS@example.com", name: "Boss",
  });
  assert.equal(result.data.role, "admin");
});
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd auth-service && pnpm test`
Expected: tests fail (tables don't exist yet, or hook signature mismatch). Note exact failure mode.

- [ ] **Step 3: Make tests pass**

The test setup creates the `user` table inline so the bootstrap admin reconciliation can run. The hook is already implemented from Task 3. Adjust imports/typing in tests if needed; do not change `auth.ts` to make tests pass.

Run: `pnpm test`
Expected: 3 passed.

- [ ] **Step 4: Commit**

```bash
git add auth-service/src/__tests__/hooks.test.ts
git commit -m "test allowlist hook: user, admin, and rejection paths"
```

---

## Phase 3 — HTTP layer

### Task 6: Auth middleware

**Files:**
- Create: `auth-service/src/middleware.ts`

- [ ] **Step 1: Create `middleware.ts`**

`auth-service/src/middleware.ts`:
```ts
import { createMiddleware } from "hono/factory";
import { auth, type AuthUser } from "./auth.js";

type Vars = {
  user: AuthUser;
  session: { id: string; userId: string; expiresAt: Date };
};

export const requireAuth = createMiddleware<{ Variables: Vars }>(
  async (c, next) => {
    const session = await auth.api.getSession({ headers: c.req.raw.headers });
    if (!session) return c.json({ error: "unauthorized" }, 401);
    c.set("user", session.user);
    c.set("session", session.session as any);
    await next();
  }
);

export const requireAdmin = createMiddleware<{ Variables: Vars }>(
  async (c, next) => {
    const user = c.get("user");
    if (user.role !== "admin") return c.json({ error: "forbidden" }, 403);
    await next();
  }
);
```

- [ ] **Step 2: Typecheck**

Run: `cd auth-service && pnpm typecheck`
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add auth-service/src/middleware.ts
git commit -m "add requireAuth and requireAdmin Hono middleware"
```

---

### Task 7: TDD — reverse proxy with header stripping

**Files:**
- Create: `auth-service/src/proxy.ts`
- Create: `auth-service/src/__tests__/proxy.test.ts`

- [ ] **Step 1: Write the failing tests**

`auth-service/src/__tests__/proxy.test.ts`:
```ts
import { test } from "node:test";
import assert from "node:assert/strict";
import { Hono } from "hono";
import { proxyToGo } from "../proxy.js";

function makeApp(captured: { headers?: Headers; url?: string }) {
  // mock Go backend with a simple fetch shim
  const originalFetch = globalThis.fetch;
  globalThis.fetch = (async (input: any, init: any) => {
    captured.url = typeof input === "string" ? input : input.url;
    captured.headers = new Headers(init?.headers);
    return new Response(JSON.stringify({ ok: true }), {
      headers: { "content-type": "application/json" },
    });
  }) as any;
  process.env.GO_BACKEND_URL = "http://backend.test";

  const app = new Hono();
  app.use("*", async (c, next) => {
    c.set("user", { id: "u1", email: "a@x.com", role: "user" } as any);
    await next();
  });
  app.all("/api/*", proxyToGo);

  return { app, restore: () => (globalThis.fetch = originalFetch) };
}

test("proxy strips inbound x-user-* and cookie headers, attaches trusted ones", async () => {
  const captured: any = {};
  const { app, restore } = makeApp(captured);

  const res = await app.request("/api/torrents", {
    method: "GET",
    headers: {
      "cookie": "session=stolen",
      "x-user-id": "spoofed",
      "x-user-email": "spoof@x.com",
      "x-user-role": "admin",
    },
  });

  restore();
  assert.equal(res.status, 200);
  assert.equal(captured.url, "http://backend.test/torrents");
  assert.equal(captured.headers.get("cookie"), null);
  assert.equal(captured.headers.get("x-user-id"), "u1");
  assert.equal(captured.headers.get("x-user-email"), "a@x.com");
  assert.equal(captured.headers.get("x-user-role"), "user");
});

test("proxy strips /api prefix from forwarded path", async () => {
  const captured: any = {};
  const { app, restore } = makeApp(captured);

  await app.request("/api/scrape/piratebay/foo");
  restore();
  assert.equal(captured.url, "http://backend.test/scrape/piratebay/foo");
});

test("proxy preserves query string", async () => {
  const captured: any = {};
  const { app, restore } = makeApp(captured);

  await app.request("/api/torrents?limit=10");
  restore();
  assert.equal(captured.url, "http://backend.test/torrents?limit=10");
});
```

- [ ] **Step 2: Run tests, verify they fail**

Run: `cd auth-service && pnpm test`
Expected: failure — `proxy.ts` doesn't exist.

- [ ] **Step 3: Implement `proxy.ts`**

`auth-service/src/proxy.ts`:
```ts
import type { Context } from "hono";
import type { AuthUser } from "./auth.js";

const GO_BACKEND = () => process.env.GO_BACKEND_URL ?? "http://backend:8080";

export const proxyToGo = async (c: Context) => {
  const user = c.get("user") as AuthUser;
  const url = new URL(c.req.url);
  const target = GO_BACKEND() + url.pathname.replace(/^\/api/, "") + url.search;

  const headers = new Headers(c.req.raw.headers);
  headers.delete("cookie");
  headers.delete("x-user-id");
  headers.delete("x-user-email");
  headers.delete("x-user-role");
  headers.set("x-user-id", user.id);
  headers.set("x-user-email", user.email);
  headers.set("x-user-role", (user as any).role ?? "user");

  const init: RequestInit = {
    method: c.req.method,
    headers,
    body:
      c.req.method === "GET" || c.req.method === "HEAD"
        ? undefined
        : c.req.raw.body,
    // @ts-expect-error duplex required for streaming bodies in Node fetch
    duplex: "half",
    redirect: "manual",
  };

  return fetch(target, init);
};
```

- [ ] **Step 4: Run tests, verify they pass**

Run: `pnpm test`
Expected: all proxy tests pass (plus the earlier hook tests).

- [ ] **Step 5: Commit**

```bash
git add auth-service/src/proxy.ts auth-service/src/__tests__/proxy.test.ts
git commit -m "add proxyToGo with header stripping and TDD coverage"
```

---

### Task 8: TDD — admin invite routes

**Files:**
- Create: `auth-service/src/admin-routes.ts`
- Create: `auth-service/src/__tests__/admin-routes.test.ts`

- [ ] **Step 1: Write the failing tests**

`auth-service/src/__tests__/admin-routes.test.ts`:
```ts
import { test, before } from "node:test";
import assert from "node:assert/strict";
import { mkdtempSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { Hono } from "hono";

let app: Hono;
let dbRef: any;

before(async () => {
  const dir = mkdtempSync(join(tmpdir(), "ba-admin-"));
  process.env.DATABASE_PATH = join(dir, "auth.sqlite");
  process.env.BETTER_AUTH_SECRET = "x".repeat(40);
  process.env.BETTER_AUTH_URL = "http://localhost:3000";
  process.env.GOOGLE_CLIENT_ID = "x";
  process.env.GOOGLE_CLIENT_SECRET = "x";
  process.env.BOOTSTRAP_ADMIN_EMAILS = "boss@example.com";

  const { db, runOwnedMigrations } = await import("../db.js");
  runOwnedMigrations();
  dbRef = db;
  db.exec(`CREATE TABLE IF NOT EXISTS user (
    id TEXT PRIMARY KEY, email TEXT, role TEXT
  )`);
  db.exec(`CREATE TABLE IF NOT EXISTS session (
    id TEXT PRIMARY KEY, userId TEXT, token TEXT, expiresAt INTEGER
  )`);

  const { mountAdminRoutes } = await import("../admin-routes.js");
  app = new Hono();
  app.use("*", async (c, next) => {
    c.set("user", { id: "admin-1", email: "boss@example.com", role: "admin" });
    await next();
  });
  mountAdminRoutes(app);
});

test("POST /api/admin/invites adds an email", async () => {
  const res = await app.request("/api/admin/invites", {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ email: "alice@example.com" }),
  });
  assert.equal(res.status, 201);
  const row = dbRef.prepare("SELECT * FROM invited_emails WHERE email = ?").get("alice@example.com");
  assert.ok(row);
  assert.equal(row.invited_by, "admin-1");
});

test("GET /api/admin/invites lists rows", async () => {
  const res = await app.request("/api/admin/invites");
  assert.equal(res.status, 200);
  const body = await res.json() as { invites: { email: string }[] };
  assert.ok(body.invites.some((i) => i.email === "alice@example.com"));
});

test("DELETE /api/admin/invites/:email removes the row", async () => {
  const res = await app.request("/api/admin/invites/alice@example.com", { method: "DELETE" });
  assert.equal(res.status, 204);
  const row = dbRef.prepare("SELECT * FROM invited_emails WHERE email = ?").get("alice@example.com");
  assert.equal(row, undefined);
});

test("POST rejects invalid email", async () => {
  const res = await app.request("/api/admin/invites", {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ email: "not-an-email" }),
  });
  assert.equal(res.status, 400);
});
```

- [ ] **Step 2: Run tests, verify they fail**

Run: `cd auth-service && pnpm test`
Expected: failure — `admin-routes.ts` doesn't exist.

- [ ] **Step 3: Implement `admin-routes.ts`**

`auth-service/src/admin-routes.ts`:
```ts
import type { Hono } from "hono";
import { db } from "./db.js";
import { auth, type AuthUser } from "./auth.js";

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export function mountAdminRoutes(app: Hono<any>) {
  app.get("/api/admin/invites", (c) => {
    const invites = db
      .prepare("SELECT email, invited_by, created_at FROM invited_emails ORDER BY created_at DESC")
      .all();
    return c.json({ invites });
  });

  app.post("/api/admin/invites", async (c) => {
    const user = c.get("user") as AuthUser;
    const body = (await c.req.json().catch(() => ({}))) as { email?: string };
    const email = (body.email ?? "").trim().toLowerCase();
    if (!EMAIL_RE.test(email)) {
      return c.json({ error: "invalid email" }, 400);
    }
    db.prepare(
      "INSERT OR IGNORE INTO invited_emails (email, invited_by) VALUES (?, ?)"
    ).run(email, user.id);
    return c.json({ email }, 201);
  });

  app.delete("/api/admin/invites/:email", async (c) => {
    const email = decodeURIComponent(c.req.param("email")).toLowerCase();
    const revoke = c.req.query("revokeSessions") === "true";

    db.prepare("DELETE FROM invited_emails WHERE email = ?").run(email);

    if (revoke) {
      const userRow = db
        .prepare("SELECT id FROM user WHERE lower(email) = ?")
        .get(email) as { id: string } | undefined;
      if (userRow) {
        await auth.api.revokeUserSessions({
          body: { userId: userRow.id },
          headers: c.req.raw.headers,
        });
      }
    }
    return c.body(null, 204);
  });
}
```

- [ ] **Step 4: Run tests, verify they pass**

Run: `pnpm test`
Expected: all admin-routes tests pass (plus all earlier tests).

- [ ] **Step 5: Commit**

```bash
git add auth-service/src/admin-routes.ts auth-service/src/__tests__/admin-routes.test.ts
git commit -m "add /api/admin/invites endpoints with TDD coverage"
```

---

### Task 9: Self-modify guard for Better Auth admin endpoints

**Files:**
- Modify: `auth-service/src/auth.ts`

- [ ] **Step 1: Add the global `before` hook to the Better Auth config**

In `auth-service/src/auth.ts`, add this block alongside the existing `databaseHooks`:

```ts
import { createAuthMiddleware } from "better-auth/api";

// inside betterAuth({ ... }):
hooks: {
  before: createAuthMiddleware(async (ctx) => {
    // Only target admin-plugin endpoints
    if (!ctx.path.startsWith("/admin/")) return;

    const session = ctx.context.session;
    if (!session) return; // unauthenticated requests are rejected elsewhere

    const targetId =
      (ctx.body as any)?.userId ?? (ctx.params as any)?.id ?? null;

    if (targetId && targetId === session.user.id) {
      throw new APIError("BAD_REQUEST", { message: "cannot modify yourself" });
    }
  }),
},
```

The full updated `auth.ts` config now has both `databaseHooks` (signup gate) and `hooks` (admin self-modify guard).

- [ ] **Step 2: Verify Better Auth's hook surface**

Run: `cd auth-service && pnpm typecheck`
Expected: no errors. If `ctx.path`, `ctx.body`, or `ctx.params` shape is different in the installed version, adjust to match — consult `node_modules/better-auth/dist/api/index.d.ts`.

- [ ] **Step 3: Commit**

```bash
git add auth-service/src/auth.ts
git commit -m "add self-modify guard hook on admin plugin endpoints"
```

---

### Task 10: Hono server (`server.ts`)

**Files:**
- Create: `auth-service/src/server.ts`

- [ ] **Step 1: Create `server.ts` wiring everything together**

`auth-service/src/server.ts`:
```ts
import { Hono } from "hono";
import { serve } from "@hono/node-server";
import { serveStatic } from "@hono/node-server/serve-static";
import { readFileSync, existsSync } from "node:fs";
import { join } from "node:path";

import { auth } from "./auth.js";
import { runOwnedMigrations, reconcileBootstrapAdmins } from "./db.js";
import { requireAuth, requireAdmin } from "./middleware.js";
import { proxyToGo } from "./proxy.js";
import { mountAdminRoutes } from "./admin-routes.js";

runOwnedMigrations();
reconcileBootstrapAdmins();

const app = new Hono();

app.get("/health", (c) => c.json({ status: "ok" }));

app.on(["GET", "POST"], "/api/auth/*", (c) => auth.handler(c.req.raw));

app.use("/api/admin/*", requireAuth, requireAdmin);
mountAdminRoutes(app);

app.use("/api/*", requireAuth);
app.all("/api/*", proxyToGo);

const PUBLIC_DIR = join(process.cwd(), "public");
const indexHtmlPath = join(PUBLIC_DIR, "index.html");

if (existsSync(PUBLIC_DIR)) {
  app.use("/*", serveStatic({ root: "./public" }));
  app.notFound((c) => {
    if (existsSync(indexHtmlPath)) {
      return c.html(readFileSync(indexHtmlPath, "utf8"));
    }
    return c.json({ error: "not found" }, 404);
  });
}

const port = Number(process.env.PORT ?? 3000);
serve({ fetch: app.fetch, port });
console.log(`auth-service listening on :${port}`);
```

- [ ] **Step 2: Smoke-test the server boots**

```bash
cd auth-service
pnpm dev
```
Expected: `auth-service listening on :3000` printed; no crashes.

- [ ] **Step 3: Smoke-test `/health`**

In another terminal: `curl http://localhost:3000/health`
Expected: `{"status":"ok"}`

- [ ] **Step 4: Smoke-test that `invited_emails` was created**

Run: `sqlite3 data/auth.sqlite ".tables"`
Expected: includes `invited_emails`.

- [ ] **Step 5: Stop the dev server (Ctrl+C) and commit**

```bash
git add auth-service/src/server.ts
git commit -m "wire Hono server: Better Auth, admin routes, proxy, static SPA"
```

---

## Phase 4 — Go backend

### Task 11: TDD — Go `RequireUser` middleware

**Files:**
- Create: `backend/middleware/auth.go`
- Create: `backend/middleware/auth_test.go`

- [ ] **Step 1: Write the failing test**

`backend/middleware/auth_test.go`:
```go
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequireUser())
	r.GET("/echo", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"id":    c.GetString("userId"),
			"email": c.GetString("userEmail"),
			"role":  c.GetString("userRole"),
		})
	})
	return r
}

func TestRequireUser_RejectsMissingHeaders(t *testing.T) {
	r := setupRouter()
	req := httptest.NewRequest(http.MethodGet, "/echo", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRequireUser_PopulatesContextFromHeaders(t *testing.T) {
	r := setupRouter()
	req := httptest.NewRequest(http.MethodGet, "/echo", nil)
	req.Header.Set("X-User-Id", "abc")
	req.Header.Set("X-User-Email", "a@x.com")
	req.Header.Set("X-User-Role", "admin")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	expected := `{"email":"a@x.com","id":"abc","role":"admin"}`
	if w.Body.String() != expected {
		t.Fatalf("expected %s, got %s", expected, w.Body.String())
	}
}
```

- [ ] **Step 2: Run, verify it fails**

Run: `cd backend && go test ./middleware/...`
Expected: build error (`middleware/auth.go` doesn't exist).

- [ ] **Step 3: Implement the middleware**

`backend/middleware/auth.go`:
```go
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func RequireUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-User-Id")
		email := c.GetHeader("X-User-Email")
		if id == "" || email == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "no user header"})
			return
		}
		c.Set("userId", id)
		c.Set("userEmail", email)
		c.Set("userRole", c.GetHeader("X-User-Role"))
		c.Next()
	}
}
```

- [ ] **Step 4: Run tests, verify they pass**

Run: `go test ./middleware/...`
Expected: 2 passed.

- [ ] **Step 5: Commit**

```bash
git add backend/middleware/auth.go backend/middleware/auth_test.go
git commit -m "add Go RequireUser middleware with tests"
```

---

### Task 12: Wrap Go routes with `RequireUser`

**Files:**
- Modify: `backend/main.go`

- [ ] **Step 1: Update `main.go` to group all non-health routes under `RequireUser()`**

In `backend/main.go`, replace the route registration block. Add the import:
```go
import "github.com/hasmikatom/torrent/middleware"
```

Replace the existing route block (lines around 56–86) with:
```go
r.GET("/health", func(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{"status": "ok"})
})

api := r.Group("/", middleware.RequireUser())
{
    api.POST("/download", handleDownload)
    api.POST("/download/file", handleFileDownload)
    api.POST("/download/batch", handleBatchDownload)
    api.POST("/download/file/batch", handleBatchFileDownload)

    api.POST("/download/prepare", handlePrepareDownload)
    api.POST("/download/file/prepare", handleFilePrepareDownload)
    api.POST("/download/prepare/batch", handleBatchPrepareDownload)
    api.POST("/download/file/prepare/batch", handleBatchFilePrepareDownload)
    api.GET("/download/prepare/status/:id", handlePrepareStatus)
    api.POST("/download/finalize", handleFinalizeDownload)
    api.POST("/download/cancel", handleCancelDownload)
    api.GET("/status/:id", getTorrentStatus)
    api.GET("/torrents", listTorrents)
    api.DELETE("/torrents/:id", deleteTorrent)
    api.PUT("/torrents/:id/rename", renameTorrent)
    api.GET("/storage", getStorageInfo)

    api.POST("/scrape/piratebay/:name", scrapePirateBay)
    api.POST("/scrape/rutracker/:name", scrapeRuTracker)
    api.GET("/scrape/piratebay/:name/stream", scrapePirateBaySSE)
    api.GET("/scrape/rutracker/:name/stream", scrapeRuTrackerSSE)
    api.GET("/scrape/sources", getScraperSources)
}
```

- [ ] **Step 2: Verify Go still builds**

Run: `cd backend && go build ./...`
Expected: no errors.

- [ ] **Step 3: Verify the existing tests (if any) still pass**

Run: `go test ./...`
Expected: pass (only middleware tests exist currently).

- [ ] **Step 4: Commit**

```bash
git add backend/main.go
git commit -m "require auth headers on all Go routes except /health"
```

---

## Phase 5 — Frontend

### Task 13: Add frontend dependencies

**Files:**
- Modify: `frontend/package.json`

- [ ] **Step 1: Switch the frontend to pnpm if it isn't already**

If `frontend/package-lock.json` exists, delete it and switch to pnpm:
```bash
cd frontend
rm -f package-lock.json
rm -rf node_modules
pnpm install
```
Otherwise just ensure pnpm is in use:
```bash
cd frontend && pnpm install
```

- [ ] **Step 2: Install `better-auth` and `react-router-dom`**

```bash
cd frontend
pnpm add better-auth react-router-dom
pnpm add -D @types/react-router-dom
```

- [ ] **Step 3: Verify install**

Run: `pnpm ls better-auth react-router-dom`
Expected: both listed at expected versions, no peer-dep warnings.

- [ ] **Step 4: Commit**

```bash
git add frontend/package.json frontend/pnpm-lock.yaml
git rm -f frontend/package-lock.json 2>/dev/null || true
git commit -m "switch frontend to pnpm and add better-auth + react-router-dom"
```

---

### Task 14: Auth client (`auth-client.ts`)

**Files:**
- Create: `frontend/src/lib/auth-client.ts`

- [ ] **Step 1: Create the auth client**

`frontend/src/lib/auth-client.ts`:
```ts
import { createAuthClient } from "better-auth/react";
import { adminClient } from "better-auth/client/plugins";

export const authClient = createAuthClient({
  baseURL: "/api/auth",
  plugins: [adminClient()],
});

export const { signIn, signOut, useSession } = authClient;
```

- [ ] **Step 2: Typecheck**

Run: `cd frontend && pnpm exec tsc --noEmit`
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/lib/auth-client.ts
git commit -m "add Better Auth React client with admin plugin"
```

---

### Task 15: Update `services.tsx` for credentials and 401 handling

**Files:**
- Modify: `frontend/src/services.tsx`

- [ ] **Step 1: Read current state**

Run: `cat frontend/src/services.tsx`
Note the current `fetch` call sites — every call needs `credentials: "include"`, and 401 responses need to trigger sign-out.

- [ ] **Step 2: Add a centralized `apiFetch` helper at the top of the file**

At the top of `frontend/src/services.tsx`, add:
```ts
import { authClient } from "./lib/auth-client";

export async function apiFetch(input: string, init: RequestInit = {}) {
  const res = await fetch(input, { ...init, credentials: "include" });
  if (res.status === 401) {
    await authClient.signOut().catch(() => {});
    // session store flips to null; <App> re-renders to LoginScreen
  }
  return res;
}
```

- [ ] **Step 3: Replace every `fetch(` call in `services.tsx` with `apiFetch(`**

Use search-replace within the file for all bare `fetch(` calls that go to `/api/...`. Leave any non-API fetches alone (there should be none in `services.tsx`).

- [ ] **Step 4: Typecheck**

Run: `cd frontend && pnpm exec tsc --noEmit`
Expected: no errors.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/services.tsx
git commit -m "centralize API fetch with credentials and 401 sign-out handling"
```

---

### Task 16: `LoginScreen` component

**Files:**
- Create: `frontend/src/components/LoginScreen.tsx`

- [ ] **Step 1: Create the component**

`frontend/src/components/LoginScreen.tsx`:
```tsx
import { signIn } from "@/lib/auth-client";
import { Button } from "@/components/ui/button";

export function LoginScreen() {
  const params = new URLSearchParams(window.location.search);
  const error = params.get("error");

  return (
    <div className="min-h-screen flex items-center justify-center bg-background p-4">
      <div className="w-full max-w-sm space-y-6 text-center">
        <div>
          <h1 className="text-2xl font-semibold">TorrentUI</h1>
          <p className="text-muted-foreground text-sm mt-2">Sign in to continue.</p>
        </div>

        {error === "not-allowlisted" && (
          <div className="rounded-md border border-destructive/50 bg-destructive/10 px-4 py-3 text-sm text-destructive text-left">
            This email isn't on the allowlist. Ask an admin to add you.
          </div>
        )}

        <Button
          className="w-full"
          onClick={() =>
            signIn.social({
              provider: "google",
              callbackURL: "/",
              errorCallbackURL: "/?error=not-allowlisted",
            })
          }
        >
          Sign in with Google
        </Button>
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Typecheck**

Run: `cd frontend && pnpm exec tsc --noEmit`
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/LoginScreen.tsx
git commit -m "add LoginScreen with Google sign-in and allowlist error message"
```

---

### Task 17: `AppShell` component

**Files:**
- Create: `frontend/src/components/AppShell.tsx`

- [ ] **Step 1: Create the component**

`frontend/src/components/AppShell.tsx`:
```tsx
import { Link } from "react-router-dom";
import { signOut } from "@/lib/auth-client";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

type User = {
  id: string;
  email: string;
  name?: string | null;
  image?: string | null;
  role?: string | null;
};

export function AppShell({ user, children }: { user: User; children: React.ReactNode }) {
  return (
    <div className="min-h-screen flex flex-col">
      <header className="border-b px-4 py-3 flex items-center justify-between">
        <Link to="/" className="font-semibold text-lg">TorrentUI</Link>

        <div className="flex items-center gap-4">
          {user.role === "admin" && (
            <Link to="/admin" className="text-sm text-muted-foreground hover:text-foreground">
              Admin
            </Link>
          )}

          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="sm" className="gap-2">
                {user.image && (
                  <img src={user.image} alt="" className="h-6 w-6 rounded-full" />
                )}
                <span>{user.name ?? user.email}</span>
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onSelect={() => signOut()}>Sign out</DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </header>

      <main className="flex-1">{children}</main>
    </div>
  );
}
```

- [ ] **Step 2: Typecheck**

Run: `cd frontend && pnpm exec tsc --noEmit`
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/AppShell.tsx
git commit -m "add AppShell layout with header and sign-out menu"
```

---

### Task 18: Refactor `App.tsx` and extract `Home`

**Files:**
- Create: `frontend/src/components/Home.tsx`
- Modify: `frontend/src/App.tsx`

- [ ] **Step 1: Read current `App.tsx`**

Run: `cat frontend/src/App.tsx`
Note: this is the existing tabbed UI (Download / PirateBay / RuTracker / Storage + active torrents).

- [ ] **Step 2: Move the entire current App body into `Home.tsx`**

Create `frontend/src/components/Home.tsx`. Paste the current contents of `App.tsx` into it, renaming the exported component from `App` to `Home`. Update imports as needed (e.g., remove the top-level layout wrapper if any was present).

- [ ] **Step 3: Replace `App.tsx` with router + session gating**

`frontend/src/App.tsx`:
```tsx
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { useSession } from "@/lib/auth-client";
import { LoginScreen } from "@/components/LoginScreen";
import { AppShell } from "@/components/AppShell";
import { Home } from "@/components/Home";
import { AdminPage } from "@/components/AdminPage";

export default function App() {
  const { data: session, isPending } = useSession();

  if (isPending) {
    return (
      <div className="min-h-screen flex items-center justify-center text-muted-foreground">
        Loading…
      </div>
    );
  }
  if (!session) return <LoginScreen />;

  return (
    <BrowserRouter>
      <AppShell user={session.user as any}>
        <Routes>
          <Route path="/" element={<Home />} />
          <Route
            path="/admin"
            element={
              (session.user as any).role === "admin"
                ? <AdminPage />
                : <Navigate to="/" replace />
            }
          />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </AppShell>
    </BrowserRouter>
  );
}
```

- [ ] **Step 4: Stub `AdminPage` so the imports resolve**

Create `frontend/src/components/AdminPage.tsx` with a placeholder (real implementation in Task 19/20):
```tsx
export function AdminPage() {
  return <div className="p-6">Admin page (coming soon)</div>;
}
```

- [ ] **Step 5: Typecheck and dev-build**

Run: `cd frontend && pnpm exec tsc --noEmit && pnpm build`
Expected: no errors.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/App.tsx frontend/src/components/Home.tsx frontend/src/components/AdminPage.tsx
git commit -m "refactor App.tsx into router with Home/Admin routes and session gating"
```

---

### Task 19: `AdminPage` — Allowlist section

**Files:**
- Modify: `frontend/src/components/AdminPage.tsx`

- [ ] **Step 1: Replace the `AdminPage` stub with the allowlist section**

`frontend/src/components/AdminPage.tsx`:
```tsx
import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent,
  AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { apiFetch } from "@/services";

type Invite = { email: string; invited_by: string | null; created_at: number };

function AllowlistSection() {
  const [invites, setInvites] = useState<Invite[]>([]);
  const [email, setEmail] = useState("");
  const [busy, setBusy] = useState(false);

  async function load() {
    const res = await apiFetch("/api/admin/invites");
    if (res.ok) {
      const data = (await res.json()) as { invites: Invite[] };
      setInvites(data.invites);
    }
  }

  useEffect(() => { load(); }, []);

  async function add() {
    setBusy(true);
    try {
      await apiFetch("/api/admin/invites", {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({ email }),
      });
      setEmail("");
      await load();
    } finally {
      setBusy(false);
    }
  }

  async function remove(targetEmail: string, revoke: boolean) {
    const url = `/api/admin/invites/${encodeURIComponent(targetEmail)}` +
      (revoke ? "?revokeSessions=true" : "");
    await apiFetch(url, { method: "DELETE" });
    await load();
  }

  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold">Allowlist</h2>

      <div className="flex gap-2">
        <Input
          type="email"
          placeholder="email@example.com"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
        />
        <Button onClick={add} disabled={busy || !email}>Add</Button>
      </div>

      <table className="w-full text-sm">
        <thead className="text-left text-muted-foreground">
          <tr><th className="py-2">Email</th><th>Invited by</th><th>Added</th><th></th></tr>
        </thead>
        <tbody>
          {invites.map((i) => (
            <tr key={i.email} className="border-t">
              <td className="py-2">{i.email}</td>
              <td>{i.invited_by ?? "—"}</td>
              <td>{new Date(i.created_at * 1000).toLocaleDateString()}</td>
              <td className="text-right">
                <RemoveDialog email={i.email} onConfirm={remove} />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </section>
  );
}

function RemoveDialog({
  email,
  onConfirm,
}: {
  email: string;
  onConfirm: (email: string, revoke: boolean) => void;
}) {
  const [revoke, setRevoke] = useState(false);
  return (
    <AlertDialog>
      <AlertDialogTrigger asChild>
        <Button variant="ghost" size="sm">Remove</Button>
      </AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Remove {email}?</AlertDialogTitle>
          <AlertDialogDescription>
            They won't be able to sign in again.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <label className="flex items-center gap-2 text-sm">
          <input type="checkbox" checked={revoke} onChange={(e) => setRevoke(e.target.checked)} />
          Also revoke active sessions for this email
        </label>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction onClick={() => onConfirm(email, revoke)}>Remove</AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

export function AdminPage() {
  return (
    <div className="p-6 max-w-4xl mx-auto space-y-8">
      <h1 className="text-2xl font-bold">Admin</h1>
      <AllowlistSection />
    </div>
  );
}
```

- [ ] **Step 2: Confirm `Input` and `AlertDialog` primitives exist**

Run: `ls frontend/src/components/ui/ | grep -E '^(input|alert-dialog)'`
Expected: both files listed. If `input.tsx` is missing, add it via shadcn convention (or use a plain `<input>` in this section). The current repo has `button.tsx`, `dropdown-menu.tsx`, `dialog.tsx` — not `alert-dialog.tsx` or `input.tsx`.

If missing, add them via shadcn: `cd frontend && npx shadcn@latest add input alert-dialog`. If shadcn isn't initialized, hand-write minimal Radix-based versions following the pattern of `dialog.tsx`.

- [ ] **Step 3: Build the frontend**

Run: `cd frontend && pnpm build`
Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/AdminPage.tsx frontend/src/components/ui/
git commit -m "add admin allowlist section with add/remove and revoke-sessions option"
```

---

### Task 20: `AdminPage` — Users section

**Files:**
- Modify: `frontend/src/components/AdminPage.tsx`

- [ ] **Step 1: Add `UsersSection` and render it below `AllowlistSection`**

In `frontend/src/components/AdminPage.tsx`, append:
```tsx
import { authClient } from "@/lib/auth-client";

type AdminUser = {
  id: string;
  email: string;
  name?: string | null;
  role?: string | null;
  banned?: boolean | null;
  createdAt: string | Date;
};

function UsersSection({ currentUserId }: { currentUserId: string }) {
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [busyId, setBusyId] = useState<string | null>(null);

  async function load() {
    const res = await authClient.admin.listUsers({ query: { limit: 100 } });
    if (res.data) setUsers(res.data.users as AdminUser[]);
  }

  useEffect(() => { load(); }, []);

  async function setRole(userId: string, role: "admin" | "user") {
    setBusyId(userId);
    try { await authClient.admin.setRole({ userId, role }); await load(); }
    finally { setBusyId(null); }
  }

  async function ban(userId: string) {
    setBusyId(userId);
    try { await authClient.admin.banUser({ userId }); await load(); }
    finally { setBusyId(null); }
  }

  async function unban(userId: string) {
    setBusyId(userId);
    try { await authClient.admin.unbanUser({ userId }); await load(); }
    finally { setBusyId(null); }
  }

  async function revoke(userId: string) {
    setBusyId(userId);
    try { await authClient.admin.revokeUserSessions({ userId }); }
    finally { setBusyId(null); }
  }

  async function remove(userId: string) {
    setBusyId(userId);
    try { await authClient.admin.removeUser({ userId }); await load(); }
    finally { setBusyId(null); }
  }

  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold">Users</h2>
      <table className="w-full text-sm">
        <thead className="text-left text-muted-foreground">
          <tr>
            <th className="py-2">Email</th><th>Name</th><th>Role</th>
            <th>Banned?</th><th>Created</th><th></th>
          </tr>
        </thead>
        <tbody>
          {users.map((u) => {
            const self = u.id === currentUserId;
            return (
              <tr key={u.id} className="border-t">
                <td className="py-2">{u.email}</td>
                <td>{u.name ?? "—"}</td>
                <td>{u.role ?? "user"}</td>
                <td>{u.banned ? "yes" : "no"}</td>
                <td>{new Date(u.createdAt).toLocaleDateString()}</td>
                <td className="text-right space-x-2">
                  {!self && u.role !== "admin" && (
                    <Button size="sm" variant="ghost"
                      onClick={() => setRole(u.id, "admin")} disabled={busyId === u.id}>
                      Promote
                    </Button>
                  )}
                  {!self && u.role === "admin" && (
                    <Button size="sm" variant="ghost"
                      onClick={() => setRole(u.id, "user")} disabled={busyId === u.id}>
                      Demote
                    </Button>
                  )}
                  {!self && !u.banned && (
                    <Button size="sm" variant="ghost"
                      onClick={() => ban(u.id)} disabled={busyId === u.id}>Ban</Button>
                  )}
                  {!self && u.banned && (
                    <Button size="sm" variant="ghost"
                      onClick={() => unban(u.id)} disabled={busyId === u.id}>Unban</Button>
                  )}
                  {!self && (
                    <>
                      <Button size="sm" variant="ghost"
                        onClick={() => revoke(u.id)} disabled={busyId === u.id}>
                        Revoke sessions
                      </Button>
                      <Button size="sm" variant="ghost"
                        onClick={() => remove(u.id)} disabled={busyId === u.id}>
                        Delete
                      </Button>
                    </>
                  )}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </section>
  );
}
```

- [ ] **Step 2: Update `AdminPage` to render `UsersSection` and pass current user id**

Replace the `AdminPage` export with:
```tsx
export function AdminPage() {
  const { data: session } = useSession();
  if (!session) return null;
  return (
    <div className="p-6 max-w-4xl mx-auto space-y-8">
      <h1 className="text-2xl font-bold">Admin</h1>
      <AllowlistSection />
      <UsersSection currentUserId={session.user.id} />
    </div>
  );
}
```
Add the import at the top: `import { useSession } from "@/lib/auth-client";`

- [ ] **Step 3: Build**

Run: `cd frontend && pnpm build`
Expected: no errors. If `authClient.admin.*` types aren't recognized, ensure the `adminClient()` plugin is registered in `auth-client.ts` (Task 14).

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/AdminPage.tsx
git commit -m "add admin users section with role/ban/revoke/delete actions"
```

---

## Phase 6 — Containerization & deploy

### Task 21: auth-service Dockerfile (multi-stage)

**Files:**
- Create: `auth-service/Dockerfile`
- Create: `auth-service/Dockerfile.dev`

- [ ] **Step 1: Create the multi-stage prod Dockerfile**

`auth-service/Dockerfile`:
```dockerfile
# Stage 1: build the React frontend
FROM node:20-alpine AS frontend-build
WORKDIR /app
RUN corepack enable
COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY frontend/ ./
RUN pnpm build
# output: /app/dist

# Stage 2: build the auth-service TS
FROM node:20-alpine AS service-build
WORKDIR /app
RUN corepack enable
COPY auth-service/package.json auth-service/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY auth-service/ ./
RUN pnpm build

# Stage 3: runtime
FROM node:20-alpine
WORKDIR /app
RUN corepack enable
COPY auth-service/package.json auth-service/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile --prod
COPY --from=service-build /app/dist ./dist
COPY --from=frontend-build /app/dist ./public

EXPOSE 3000
CMD ["node", "dist/server.js"]
```

(Note: this Dockerfile is built from the *repo root* so it can copy `frontend/` and `auth-service/` siblings. Compose will reflect that.)

- [ ] **Step 2: Create the dev Dockerfile**

`auth-service/Dockerfile.dev`:
```dockerfile
FROM node:20-alpine
WORKDIR /app
RUN corepack enable
COPY auth-service/package.json auth-service/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY auth-service/ ./
EXPOSE 3000
CMD ["pnpm", "dev"]
```

- [ ] **Step 3: Build the prod image to verify**

Run: `docker build -f auth-service/Dockerfile -t torrentui-auth:test .`
Expected: build succeeds; image is created.

- [ ] **Step 4: Commit**

```bash
git add auth-service/Dockerfile auth-service/Dockerfile.dev
git commit -m "add multi-stage Dockerfile and dev Dockerfile for auth-service"
```

---

### Task 22: docker-compose updates

**Files:**
- Modify: `docker-compose.yml`
- Modify: `docker-compose.dev.yml` (if present — otherwise create equivalent dev override)

- [ ] **Step 1: Read current compose files**

Run: `ls docker-compose*.yml && cat docker-compose.yml`
Note the current `backend` service definition to preserve it.

- [ ] **Step 2: Add `auth-service` and remove `backend`'s public port in `docker-compose.yml`**

Add as a new service alongside `backend` (or wherever services are defined):
```yaml
services:
  auth-service:
    build:
      context: .
      dockerfile: auth-service/Dockerfile
    ports:
      - "3000:3000"
    environment:
      - BETTER_AUTH_SECRET=${BETTER_AUTH_SECRET}
      - BETTER_AUTH_URL=${BETTER_AUTH_URL}
      - GOOGLE_CLIENT_ID=${GOOGLE_CLIENT_ID}
      - GOOGLE_CLIENT_SECRET=${GOOGLE_CLIENT_SECRET}
      - BOOTSTRAP_ADMIN_EMAILS=${BOOTSTRAP_ADMIN_EMAILS}
      - GO_BACKEND_URL=http://backend:8080
      - DATABASE_PATH=/data/auth.sqlite
      - NODE_ENV=production
      - PORT=3000
    volumes:
      - ./data:/data
    depends_on:
      - backend
    restart: unless-stopped

  backend:
    # remove "ports:" — replace with:
    expose:
      - "8080"
    # ... preserve the rest of the existing backend config (build, environment, volumes)
```

- [ ] **Step 3: Update `docker-compose.dev.yml` similarly**

Use `auth-service/Dockerfile.dev`, mount `./auth-service:/app` and `/app/node_modules` for hot reload, set `NODE_ENV=development` and `BETTER_AUTH_URL=http://localhost:3000`. Keep the dev frontend running outside Docker (Vite at :5173) — it proxies to auth-service at :3000.

- [ ] **Step 4: Validate compose files**

Run: `docker compose -f docker-compose.yml config > /dev/null` and same for the dev file.
Expected: no errors.

- [ ] **Step 5: Commit**

```bash
git add docker-compose.yml docker-compose.dev.yml
git commit -m "add auth-service to compose; drop backend public port"
```

---

### Task 23: Vite proxy & Makefile updates

**Files:**
- Modify: `frontend/vite.config.ts`
- Modify: `Makefile`

- [ ] **Step 1: Read current Vite config**

Run: `cat frontend/vite.config.ts`
Note where `VITE_API_TARGET` is referenced.

- [ ] **Step 2: Flip the default target**

In `frontend/vite.config.ts`, change the line that defaults `VITE_API_TARGET` from `http://localhost:8085` to `http://localhost:3000`. The proxy now points at the auth-service in dev; auth-service in turn proxies to Go on `:8085`.

- [ ] **Step 3: Add Makefile targets**

Append to `Makefile`:
```makefile
.PHONY: auth-dev auth-migrate auth-shell-db

auth-dev:
	cd auth-service && pnpm dev

auth-migrate:
	cd auth-service && pnpm exec @better-auth/cli@latest migrate --config src/auth.ts --y

auth-shell-db:
	sqlite3 ./data/auth.sqlite
```

- [ ] **Step 4: Commit**

```bash
git add frontend/vite.config.ts Makefile
git commit -m "point Vite proxy to auth-service and add auth-service Makefile targets"
```

---

## Phase 7 — Verification & cutover

### Task 24: Update `CLAUDE.md`

**Files:**
- Modify: `CLAUDE.md`

- [ ] **Step 1: Add an `auth-service` paragraph to the Architecture and Project Structure sections**

In the architecture diagram, add `auth-service` as the public surface in front of the Go backend. In the project structure section, add:
```
auth-service/
  src/
    auth.ts          # betterAuth() config
    db.ts            # better-sqlite3 + invited_emails + bootstrap reconciliation
    middleware.ts    # requireAuth, requireAdmin
    proxy.ts         # proxyToGo
    admin-routes.ts  # /api/admin/invites/*
    server.ts        # Hono app entry
data/                # SQLite volume
```

Update the Hosting & Authentication section to read: "Authentication is Google OAuth via Better Auth, gated by an admin-managed email allowlist. Cloudflare Tunnel still terminates TLS but Cloudflare Access is no longer in front of the app."

- [ ] **Step 2: Commit**

```bash
git add CLAUDE.md
git commit -m "document auth-service in CLAUDE.md"
```

---

### Task 25: End-to-end verification

**Files:** none — manual checklist.

- [ ] **Step 1: Set up a real Google OAuth client**

Follow the steps in the spec (Configuration → Google Cloud OAuth Client). Put real `GOOGLE_CLIENT_ID` and `GOOGLE_CLIENT_SECRET` values into `auth-service/.env`. Add `http://localhost:3000` to authorized origins and `http://localhost:3000/api/auth/callback/google` to redirect URIs.

- [ ] **Step 2: Start everything in dev**

Three terminals:
1. `cd backend && air`
2. `cd auth-service && pnpm dev`
3. `cd frontend && pnpm dev`

- [ ] **Step 3: Run the four E2E paths**

| Path | Expected |
|---|---|
| Open `http://localhost:5173` signed-out | `<LoginScreen>` shown |
| Click "Sign in with Google" as `d.isayan@gmail.com` (bootstrap admin) | Redirects to Google → back to `/`; header shows your name; "Admin" link visible |
| Open in incognito, sign in with a Google email NOT in the allowlist | Redirects to `/?error=not-allowlisted`; LoginScreen shows the error |
| As admin, add `someone@example.com` to allowlist; have them sign in | They're admitted; header shows their name; no "Admin" link |

- [ ] **Step 4: Test API gating**

With dev tools open, in an authenticated session: `fetch('/api/torrents')` → 200 with data. Then sign out and try the same: → 401.

Without auth-service running, `curl http://localhost:8085/torrents` (Go directly): 401 if Go is started fresh under the new code.

- [ ] **Step 5: Test admin operations**

In the `/admin` page:
- Add an email, see it in the list.
- Delete it (without revoke), confirm row disappears.
- Promote a non-admin user, refresh, confirm `role` updated.
- Try to demote yourself: button disabled in UI; if attempted via console, server returns 400.

- [ ] **Step 6: Test SSE still works**

Open the PirateBay or RuTracker tab and run a search. The streaming progress UI should work — confirms cookie-based auth doesn't break EventSource.

- [ ] **Step 7: Document any issues found**

If anything fails, file a follow-up task; don't paper over with hacks.

---

### Task 26: Production cutover (operator runbook)

**Files:** none — operator actions outside the repo.

- [ ] **Step 1: Generate a real `BETTER_AUTH_SECRET` for prod**

`openssl rand -hex 32` — store in your secret-management of choice and inject as the `BETTER_AUTH_SECRET` env var.

- [ ] **Step 2: Update Google OAuth client for prod URLs**

Add `https://<your-public-domain>` and `https://<your-public-domain>/api/auth/callback/google` to authorized origins/redirect URIs.

- [ ] **Step 3: Build and deploy**

```bash
make prod-build-deploy
```

- [ ] **Step 4: Migrate prod DB**

```bash
docker exec -it torrentui-auth-service-1 \
  pnpm exec @better-auth/cli@latest migrate --config src/auth.ts --y
```

- [ ] **Step 5: Repoint Cloudflare Tunnel**

In `cloudflared` config, change the public hostname's service from `http://backend:8080` (or wherever) to `http://auth-service:3000`. Restart `cloudflared`.

- [ ] **Step 6: Verify prod signs you in**

Hit your public URL signed-out → LoginScreen → sign in with `d.isayan@gmail.com` → see the app.

- [ ] **Step 7: Remove Cloudflare Access**

Once you've confirmed end-to-end auth works in prod, remove the CFA application/policy gating the public hostname. The app is now public-facing, gated by Better Auth + the allowlist.

- [ ] **Step 8: Final commit on the branch**

If there are any tweaks discovered during cutover, commit them. Otherwise, the branch is ready to merge.

---

## Done

The `better-auth` branch should now contain:
- A new `auth-service/` with Better Auth + Hono + SQLite + tests
- A protected Go backend with no public port
- A frontend with router, login, and admin page
- Containerization that builds and deploys as a single `docker compose up`

Open a PR (assigned to `@me`, draft) and merge after the prod cutover step succeeds.
