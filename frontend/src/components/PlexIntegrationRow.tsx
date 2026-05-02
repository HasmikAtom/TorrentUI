import { useState } from "react";
import { Plug } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import { apiFetch } from "@/services";

type Props = {
  enabled: boolean;
  hasToken: boolean;
  onUpdate: () => void;
};

export function PlexIntegrationRow({ enabled, hasToken, onUpdate }: Props) {
  const [expanded, setExpanded] = useState(false);
  const [token, setToken] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const connected = hasToken;

  async function handleSave() {
    setBusy(true);
    setError("");
    try {
      const res = await apiFetch("/api/integrations/plex", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ token }),
      });
      if (!res.ok) {
        const data = await res.json();
        setError(data.error || "Failed to save");
        return;
      }
      setToken("");
      setExpanded(false);
      onUpdate();
    } finally {
      setBusy(false);
    }
  }

  async function handleToggle(checked: boolean) {
    setBusy(true);
    try {
      await apiFetch("/api/integrations/plex", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ enabled: checked }),
      });
      onUpdate();
    } finally {
      setBusy(false);
    }
  }

  async function handleDisconnect() {
    setBusy(true);
    setError("");
    try {
      await apiFetch("/api/integrations/plex", { method: "DELETE" });
      setExpanded(false);
      onUpdate();
    } finally {
      setBusy(false);
    }
  }

  return (
    <div>
      <div
        className="flex items-center p-4 gap-3 cursor-pointer hover:bg-accent/50 transition-colors"
        onClick={() => setExpanded(!expanded)}
      >
        <div
          className={`flex size-8 items-center justify-center rounded-md ${
            connected ? "bg-[#e5a00d] text-black" : "bg-[#e5a00d]/20 text-[#e5a00d]"
          }`}
        >
          <Plug className="size-4" />
        </div>
        <div className="flex-1 min-w-0">
          <div className="font-semibold text-sm">Plex</div>
          <div className="text-xs text-muted-foreground">Media server integration</div>
        </div>
        {connected ? (
          <div className="flex items-center gap-3">
            <span className="text-xs text-green-500">● Connected</span>
            <Switch
              checked={enabled}
              onCheckedChange={handleToggle}
              disabled={busy}
              onClick={(e) => e.stopPropagation()}
            />
          </div>
        ) : (
          <Button
            size="sm"
            variant="outline"
            onClick={(e) => {
              e.stopPropagation();
              setExpanded(true);
            }}
          >
            Connect
          </Button>
        )}
      </div>

      {expanded && (
        <div className="px-4 pb-4 pl-[60px] border-t bg-muted/30">
          <div className="pt-4 space-y-3">
            <div>
              <label className="text-xs text-muted-foreground block mb-1.5">
                Plex Token
              </label>
              <div className="flex gap-2">
                <Input
                  type="password"
                  placeholder={connected ? "••••••••••••••••" : "Paste your Plex token"}
                  value={token}
                  onChange={(e) => setToken(e.target.value)}
                  disabled={busy}
                />
                <Button onClick={handleSave} disabled={busy || !token} size="sm">
                  {connected ? "Update" : "Save"}
                </Button>
                {connected ? (
                  <Button
                    variant="destructive"
                    size="sm"
                    onClick={handleDisconnect}
                    disabled={busy}
                  >
                    Disconnect
                  </Button>
                ) : (
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => {
                      setExpanded(false);
                      setToken("");
                      setError("");
                    }}
                  >
                    Cancel
                  </Button>
                )}
              </div>
              {error && (
                <p className="text-xs text-destructive mt-1.5">{error}</p>
              )}
            </div>
            <p className="text-[11px] text-muted-foreground">
              Find your token at plex.tv — go to Settings → Devices, click a
              device, and look for the token in the XML URL.
            </p>
          </div>
        </div>
      )}
    </div>
  );
}
