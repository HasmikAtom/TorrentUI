import { betterAuth } from "better-auth";
import { admin } from "better-auth/plugins";
import { APIError, createAuthMiddleware } from "better-auth/api";
import { db, bootstrapAdmins } from "./db.js";

export async function beforeUserCreate(user: {
  id: string;
  email: string;
  name: string;
  image?: string | null;
  createdAt: Date;
  updatedAt: Date;
  emailVerified: boolean;
  [key: string]: unknown;
}) {
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

  hooks: {
    before: createAuthMiddleware(async (ctx) => {
      // Only target admin-plugin endpoints
      if (!ctx.path.startsWith("/admin/")) return;

      const session = ctx.context.session;
      if (!session) return; // unauthenticated requests are rejected elsewhere

      const targetId =
        (ctx.body as { userId?: string } | undefined)?.userId ??
        (ctx.params as { id?: string } | undefined)?.id ??
        null;

      if (targetId && targetId === session.user.id) {
        throw new APIError("BAD_REQUEST", { message: "cannot modify yourself" });
      }
    }),
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
