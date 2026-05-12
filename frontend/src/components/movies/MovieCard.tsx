import { useState } from "react";
import { imageUrl } from "./api";
import type { Movie } from "./types";

type Props = {
  movie: Movie;
  onClick: (ratingKey: string) => void;
};

export function MovieCard({ movie, onClick }: Props) {
  const [imgFailed, setImgFailed] = useState(false);

  return (
    <button
      type="button"
      onClick={() => onClick(movie.ratingKey)}
      className="group flex flex-col text-left focus:outline-none focus:ring-2 focus:ring-ring rounded-md"
    >
      <div className="relative aspect-[2/3] w-full overflow-hidden rounded-md bg-muted shadow-sm">
        {!imgFailed && movie.thumb ? (
          <img
            src={imageUrl(movie.thumb)}
            alt={movie.title}
            loading="lazy"
            onError={() => setImgFailed(true)}
            className="h-full w-full object-cover transition-transform group-hover:scale-[1.03]"
          />
        ) : (
          <div className="flex h-full w-full items-center justify-center p-2 text-center text-xs text-muted-foreground">
            {movie.title}
          </div>
        )}
      </div>
      <div className="mt-2 line-clamp-1 text-sm font-medium" title={movie.title}>
        {movie.title}
      </div>
      {movie.year > 0 && (
        <div className="text-xs text-muted-foreground">{movie.year}</div>
      )}
    </button>
  );
}
