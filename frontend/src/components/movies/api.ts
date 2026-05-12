import { apiFetch } from "@/services";
import type { ListMoviesResult, MovieDetail, SortKey } from "./types";

export type FetchError =
  | { kind: "not_configured" }
  | { kind: "unauthorized" }
  | { kind: "unreachable" }
  | { kind: "unknown"; status: number };

async function asFetchError(res: Response): Promise<FetchError> {
  let body: { error?: string } = {};
  try {
    body = await res.json();
  } catch {
    // ignore
  }
  if (res.status === 412 || body.error === "plex_not_configured") return { kind: "not_configured" };
  if (res.status === 401 || body.error === "plex_unauthorized") return { kind: "unauthorized" };
  if (res.status === 502 || body.error === "plex_server_unreachable") return { kind: "unreachable" };
  return { kind: "unknown", status: res.status };
}

export async function fetchMovies(start: number, size: number, sort: SortKey): Promise<
  { ok: true; data: ListMoviesResult } | { ok: false; error: FetchError }
> {
  const qs = new URLSearchParams({ start: String(start), size: String(size), sort });
  const res = await apiFetch(`/api/plex/movies?${qs.toString()}`);
  if (!res.ok) return { ok: false, error: await asFetchError(res) };
  const data = (await res.json()) as ListMoviesResult;
  return { ok: true, data };
}

export async function fetchMovieDetail(ratingKey: string): Promise<
  { ok: true; data: MovieDetail } | { ok: false; error: FetchError }
> {
  const res = await apiFetch(`/api/plex/movies/${encodeURIComponent(ratingKey)}`);
  if (!res.ok) return { ok: false, error: await asFetchError(res) };
  const data = (await res.json()) as MovieDetail;
  return { ok: true, data };
}

export function imageUrl(path: string): string {
  return `/api/plex/image?path=${encodeURIComponent(path)}`;
}
