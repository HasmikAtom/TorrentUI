import { authClient } from "./lib/auth-client";

export async function apiFetch(input: string, init: RequestInit = {}) {
  const res = await fetch(input, { ...init, credentials: "include" });
  if (res.status === 401) {
    await authClient.signOut().catch(() => {});
    // session store flips to null; <App> re-renders to LoginScreen
  }
  return res;
}
