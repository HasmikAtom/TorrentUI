import { useEffect, useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Skeleton } from "@/components/ui/skeleton";
import { fetchMovieDetail, imageUrl, type FetchError } from "./api";
import type { MovieDetail as MovieDetailT } from "./types";

type Props = {
  ratingKey: string | null;
  onClose: () => void;
};

function formatRuntime(ms: number): string {
  if (!ms) return "";
  const totalMin = Math.round(ms / 60000);
  const h = Math.floor(totalMin / 60);
  const m = totalMin % 60;
  if (h === 0) return `${m}m`;
  return `${h}h ${m}m`;
}

export function MovieDetail({ ratingKey, onClose }: Props) {
  const [data, setData] = useState<MovieDetailT | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<FetchError | null>(null);

  useEffect(() => {
    if (!ratingKey) return;
    setData(null);
    setError(null);
    setLoading(true);
    let alive = true;
    fetchMovieDetail(ratingKey).then((res) => {
      if (!alive) return;
      setLoading(false);
      if (res.ok) setData(res.data);
      else setError(res.error);
    });
    return () => {
      alive = false;
    };
  }, [ratingKey]);

  const open = ratingKey !== null;

  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="max-w-3xl">
        {loading && (
          <>
            <DialogHeader>
              <DialogTitle><Skeleton className="h-6 w-48" /></DialogTitle>
              <DialogDescription className="sr-only">Loading movie details</DialogDescription>
            </DialogHeader>
            <div className="grid grid-cols-1 gap-4 md:grid-cols-[200px_1fr]">
              <Skeleton className="aspect-[2/3] w-full rounded-md" />
              <div className="space-y-2">
                <Skeleton className="h-4 w-full" />
                <Skeleton className="h-4 w-full" />
                <Skeleton className="h-4 w-3/4" />
              </div>
            </div>
          </>
        )}
        {error && (
          <>
            <DialogHeader>
              <DialogTitle>Couldn't load movie</DialogTitle>
              <DialogDescription>
                {error.kind === "unauthorized"
                  ? "Your Plex token is invalid. Reconnect in Integrations."
                  : "Could not reach your Plex server."}
              </DialogDescription>
            </DialogHeader>
          </>
        )}
        {data && (
          <>
            <DialogHeader>
              <DialogTitle>
                {data.title}
                {data.year > 0 && <span className="ml-2 text-muted-foreground font-normal">({data.year})</span>}
              </DialogTitle>
              <DialogDescription className="sr-only">Movie details</DialogDescription>
            </DialogHeader>
            <div className="grid grid-cols-1 gap-4 md:grid-cols-[200px_1fr]">
              {data.thumb ? (
                <img
                  src={imageUrl(data.thumb)}
                  alt={data.title}
                  className="aspect-[2/3] w-full rounded-md object-cover"
                />
              ) : (
                <div className="aspect-[2/3] w-full rounded-md bg-muted" />
              )}
              <div className="space-y-3 text-sm">
                <div className="flex flex-wrap gap-x-3 gap-y-1 text-muted-foreground">
                  {formatRuntime(data.duration) && <span>{formatRuntime(data.duration)}</span>}
                  {data.contentRating && <span>{data.contentRating}</span>}
                  {data.audienceRating > 0 && <span>★ {data.audienceRating.toFixed(1)}</span>}
                </div>
                {data.genres.length > 0 && (
                  <div className="flex flex-wrap gap-1.5">
                    {data.genres.map((g) => (
                      <span key={g} className="rounded-full bg-muted px-2 py-0.5 text-xs">{g}</span>
                    ))}
                  </div>
                )}
                {data.summary && <p className="leading-relaxed">{data.summary}</p>}
                {data.directors.length > 0 && (
                  <div>
                    <span className="font-medium">Directed by:</span>{" "}
                    <span className="text-muted-foreground">{data.directors.join(", ")}</span>
                  </div>
                )}
                {data.cast.length > 0 && (
                  <div>
                    <span className="font-medium">Cast:</span>{" "}
                    <span className="text-muted-foreground">{data.cast.join(", ")}</span>
                  </div>
                )}
              </div>
            </div>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}
