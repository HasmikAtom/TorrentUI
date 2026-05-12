import { Skeleton } from "@/components/ui/skeleton";
import { MovieCard } from "./MovieCard";
import type { Movie } from "./types";

type Props = {
  items: Movie[];
  loading: boolean;
  done: boolean;
  onCardClick: (ratingKey: string) => void;
  sentinelRef: React.RefObject<HTMLDivElement>;
};

export function MovieGrid({ items, loading, done, onCardClick, sentinelRef }: Props) {
  if (loading && items.length === 0) {
    return (
      <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6">
        {Array.from({ length: 12 }).map((_, i) => (
          <div key={i} className="flex flex-col gap-2">
            <Skeleton className="aspect-[2/3] w-full rounded-md" />
            <Skeleton className="h-4 w-3/4" />
            <Skeleton className="h-3 w-1/4" />
          </div>
        ))}
      </div>
    );
  }

  if (!loading && items.length === 0) {
    return (
      <div className="flex h-48 items-center justify-center text-muted-foreground">
        No movies found in your Plex libraries.
      </div>
    );
  }

  return (
    <>
      <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6">
        {items.map((m) => (
          <MovieCard key={m.ratingKey} movie={m} onClick={onCardClick} />
        ))}
      </div>
      <div ref={sentinelRef} className="flex h-12 items-center justify-center">
        {loading && (
          <div className="text-sm text-muted-foreground">Loading…</div>
        )}
        {done && (
          <div className="text-xs text-muted-foreground">End of library</div>
        )}
      </div>
    </>
  );
}
