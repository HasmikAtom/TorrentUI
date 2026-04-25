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
