import { useEffect, useState } from "react";
import { apiFetch } from "@/services";

export type IntegrationState = {
  plexEnabled: boolean;
  plexHasToken: boolean;
};

const defaultState: IntegrationState = {
  plexEnabled: false,
  plexHasToken: false,
};

let cached: IntegrationState | null = null;
let inFlight: Promise<IntegrationState> | null = null;
const subscribers = new Set<(s: IntegrationState) => void>();

async function load(): Promise<IntegrationState> {
  if (cached) return cached;
  if (!inFlight) {
    inFlight = apiFetch("/api/integrations").then(async (res) => {
      if (!res.ok) return defaultState;
      const data = (await res.json()) as IntegrationState;
      cached = data;
      subscribers.forEach((cb) => cb(data));
      return data;
    });
  }
  return inFlight;
}

export function refreshIntegrations() {
  cached = null;
  inFlight = null;
  load();
}

export function useIntegrations(): {
  state: IntegrationState;
  loading: boolean;
  refresh: () => void;
} {
  const [state, setState] = useState<IntegrationState>(cached ?? defaultState);
  const [loading, setLoading] = useState(cached === null);

  useEffect(() => {
    let alive = true;
    const sub = (s: IntegrationState) => {
      if (alive) setState(s);
    };
    subscribers.add(sub);
    if (cached) {
      setState(cached);
      setLoading(false);
    } else {
      load().then((s) => {
        if (alive) {
          setState(s);
          setLoading(false);
        }
      });
    }
    return () => {
      alive = false;
      subscribers.delete(sub);
    };
  }, []);

  return { state, loading, refresh: refreshIntegrations };
}
