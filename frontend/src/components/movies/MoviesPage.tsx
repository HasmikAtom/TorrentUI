import { useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { Plug } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useIntegrations } from "@/hooks/useIntegrations";
import { MovieGrid } from "./MovieGrid";
import { SortDropdown } from "./SortDropdown";
import { MovieDetail } from "./MovieDetail";
import { useMoviesQuery } from "./useMoviesQuery";

function NotConnectedCTA() {
  return (
    <div className="mx-auto mt-16 max-w-md rounded-lg border bg-card p-8 text-center">
      <Plug className="mx-auto mb-3 size-8 text-muted-foreground" />
      <h2 className="text-lg font-semibold">Connect Plex to see your movies</h2>
      <p className="mt-2 text-sm text-muted-foreground">
        Add your Plex token in Integrations and we'll show your library here.
      </p>
      <Button asChild className="mt-4">
        <Link to="/integrations">Open Integrations</Link>
      </Button>
    </div>
  );
}

export function MoviesPage() {
  const { state: integrations, loading: integrationsLoading } = useIntegrations();
  const query = useMoviesQuery();
  const [selected, setSelected] = useState<string | null>(null);
  const sentinelRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const el = sentinelRef.current;
    if (!el) return;
    const observer = new IntersectionObserver((entries) => {
      if (entries[0]?.isIntersecting) query.loadMore();
    });
    observer.observe(el);
    return () => observer.disconnect();
  }, [query]);

  if (integrationsLoading) {
    return <div className="p-6 text-muted-foreground">Loading…</div>;
  }
  if (!integrations.plexEnabled || !integrations.plexHasToken) {
    return <NotConnectedCTA />;
  }
  if (query.error?.kind === "not_configured") {
    return <NotConnectedCTA />;
  }

  return (
    <div className="space-y-4 py-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Movies</h1>
        <SortDropdown value={query.sort} onChange={query.setSort} />
      </div>

      {query.error && query.items.length === 0 ? (
        <div className="mx-auto mt-12 max-w-md rounded-lg border bg-card p-6 text-center">
          <p className="text-sm">
            {query.error.kind === "unauthorized"
              ? "Your Plex token is invalid. Reconnect in Integrations."
              : "Couldn't reach your Plex server."}
          </p>
          <Button onClick={query.retry} variant="outline" size="sm" className="mt-3">
            Retry
          </Button>
        </div>
      ) : (
        <MovieGrid
          items={query.items}
          loading={query.loading}
          done={query.done}
          sentinelRef={sentinelRef}
          onCardClick={setSelected}
        />
      )}

      <MovieDetail ratingKey={selected} onClose={() => setSelected(null)} />
    </div>
  );
}
