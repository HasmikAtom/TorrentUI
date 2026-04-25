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
