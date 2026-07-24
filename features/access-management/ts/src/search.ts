export function normalizeAccessSearch(value: string): string {
  return value.trim().toLocaleLowerCase();
}

export function matchesAccessSearch(query: string, values: unknown[]): boolean {
  if (!query) return true;
  return values.some((value) => String(value ?? "").toLocaleLowerCase().includes(query));
}
