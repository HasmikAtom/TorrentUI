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

function makeUser(over: { email: string; name?: string; id?: string }) {
  const now = new Date();
  return {
    id: over.id ?? "u-test",
    email: over.email,
    name: over.name ?? "Test",
    emailVerified: false,
    createdAt: now,
    updatedAt: now,
  } as Parameters<typeof import("../auth.js").beforeUserCreate>[0];
}

test("allowlisted email is permitted, role=user", async () => {
  const { db, runOwnedMigrations } = await import("../db.js");
  const { beforeUserCreate } = await import("../auth.js");
  runOwnedMigrations();
  db.prepare("INSERT OR IGNORE INTO invited_emails (email) VALUES (?)").run("alice@example.com");

  const result = await beforeUserCreate(makeUser({ id: "u1", email: "alice@example.com", name: "Alice" }));
  assert.equal((result as any).data.role, "user");
});

test("non-allowlisted email throws FORBIDDEN", async () => {
  const { beforeUserCreate } = await import("../auth.js");
  await assert.rejects(
    () => beforeUserCreate(makeUser({ id: "u2", email: "stranger@example.com", name: "Eve" })),
    /Email not on allowlist/
  );
});

test("bootstrap admin email gets role=admin", async () => {
  const { beforeUserCreate } = await import("../auth.js");
  const result = await beforeUserCreate(makeUser({ id: "u3", email: "BOSS@example.com", name: "Boss" }));
  assert.equal((result as any).data.role, "admin");
});
