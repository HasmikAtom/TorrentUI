export type Movie = {
  ratingKey: string;
  title: string;
  year: number;
  thumb: string;
  art: string;
  rating: number;
  audienceRating: number;
  duration: number;
  addedAt: number;
  summary: string;
};

export type MovieDetail = Movie & {
  contentRating: string;
  studio: string;
  originallyAvailableAt: string;
  genres: string[];
  directors: string[];
  writers: string[];
  cast: string[];
};

export type ListMoviesResult = {
  items: Movie[];
  total: number;
  start: number;
  size: number;
};

export type SortKey =
  | "addedAt:desc"
  | "titleSort:asc"
  | "year:desc"
  | "rating:desc";

export const SORT_OPTIONS: { key: SortKey; label: string }[] = [
  { key: "addedAt:desc", label: "Recently Added" },
  { key: "titleSort:asc", label: "Title A–Z" },
  { key: "year:desc", label: "Year (Newest)" },
  { key: "rating:desc", label: "Rating" },
];
