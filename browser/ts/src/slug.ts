/**
 * camelCase ↔ kebab-case helpers for resource slugs in URLs.
 *
 * Internal resource identifiers (e.g. config map keys) are typically
 * camelCase because they double as TypeScript object property names; URLs
 * that end users see are typically kebab-case for readability.
 *
 * This module is purely algorithmic. Consumers that have non-trivial
 * irregular plurals or capitalization quirks (e.g. `bikeWorkOrders` →
 * `bike-work-orders`) supply a custom map via `applySlugMap`.
 */

/** camelCase → kebab-case via algorithmic transformation. */
export function camelToKebab(input: string): string {
  return input
    .replace(/([a-z\d])([A-Z])/g, "$1-$2")
    .replace(/([A-Z])([A-Z][a-z])/g, "$1-$2")
    .toLowerCase();
}

/** kebab-case → camelCase via algorithmic transformation. */
export function kebabToCamel(input: string): string {
  return input.replace(/-([a-zA-Z])/g, (_, ch: string) => ch.toUpperCase());
}

/**
 * Apply a custom mapping with algorithmic fallback. If `resourceId` is a
 * key of `customMap`, return its value; otherwise transform with
 * `camelToKebab`.
 *
 *   const SLUG: Record<string, string> = { bikeWorkOrders: 'bike-work-orders' };
 *   applySlugMap('bikeWorkOrders', SLUG);    // 'bike-work-orders'
 *   applySlugMap('priceLists', SLUG);        // 'price-lists' (algorithmic)
 */
export function applySlugMap(
  resourceId: string,
  customMap: Readonly<Record<string, string>> = {},
): string {
  if (resourceId in customMap) return customMap[resourceId];
  return camelToKebab(resourceId);
}

/**
 * Reverse direction: a URL slug → camelCase resourceId. Looks up
 * `customMap`'s VALUES (slug → camel direction); falls back to
 * `kebabToCamel` when no entry matches.
 */
export function unapplySlugMap(
  slug: string,
  customMap: Readonly<Record<string, string>> = {},
): string {
  for (const [camel, mappedSlug] of Object.entries(customMap)) {
    if (mappedSlug === slug) return camel;
  }
  return kebabToCamel(slug);
}
