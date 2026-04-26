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
