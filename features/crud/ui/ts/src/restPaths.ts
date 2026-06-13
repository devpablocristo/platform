import type { CrudLifecycleView } from "./types";

/**
 * Segmentos de URL para el modo `basePath` de CrudPage.
 * Deben coincidir con `platform/lifecycle/go/paths`.
 */
export const CrudPathSegment = {
  archived: "archived",
  archive: "archive",
  unarchive: "unarchive",
  trash: "trash",
  restore: "restore",
  purge: "purge",
} as const;

export type CrudItemAction = "archive" | "unarchive" | "trash" | "restore" | "purge";

function trimTrailingSlash(path: string): string {
  return path.replace(/\/+$/, "");
}

export function crudListPath(basePath: string, view: CrudLifecycleView): string {
  const base = trimTrailingSlash(basePath);
  if (view === "archived") return `${base}/${CrudPathSegment.archived}`;
  if (view === "trash") return `${base}/${CrudPathSegment.trash}`;
  return base;
}

export function crudItemPath(basePath: string, id: string, action?: CrudItemAction): string {
  const base = trimTrailingSlash(basePath);
  if (!action) return `${base}/${id}`;
  return `${base}/${id}/${CrudPathSegment[action]}`;
}
