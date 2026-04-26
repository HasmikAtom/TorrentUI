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
