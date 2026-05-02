import { useEffect, useState } from "react";
import { apiFetch } from "@/services";
import { PlexIntegrationRow } from "./PlexIntegrationRow";

type IntegrationState = {
  plexEnabled: boolean;
  plexHasToken: boolean;
};

export function IntegrationsPage() {
  const [state, setState] = useState<IntegrationState | null>(null);
  const [loading, setLoading] = useState(true);

  async function load() {
    const res = await apiFetch("/api/integrations");
    if (res.ok) {
      setState(await res.json());
    }
    setLoading(false);
  }

  useEffect(() => {
    load();
  }, []);

  if (loading) {
    return (
      <div className="p-6 max-w-2xl mx-auto">
        <div className="text-muted-foreground">Loading…</div>
      </div>
    );
  }

  return (
    <div className="p-6 max-w-2xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Integrations</h1>
        <p className="text-sm text-muted-foreground mt-1">
          Connect external services to your account.
        </p>
      </div>
      <div className="border rounded-lg overflow-hidden">
        <PlexIntegrationRow
          enabled={state?.plexEnabled ?? false}
          hasToken={state?.plexHasToken ?? false}
          onUpdate={load}
        />
      </div>
    </div>
  );
}
