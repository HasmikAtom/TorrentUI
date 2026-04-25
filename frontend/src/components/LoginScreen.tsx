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
