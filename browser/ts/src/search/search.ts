/**
 * Función principal de búsqueda fuzzy client-side.
 *
 * Combina word-level prefix/substring match + trigram similarity
 * para dar resultados relevantes con tolerancia a typos y acentos.
 */

import { normalize } from "./normalize";
import { similarity } from "./trigram";

export type SearchEntry<T> = {
  item: T;
  text: string;
};

export type SearchResult<T> = {
  item: T;
  score: number;
};

export type SearchOptions = {
  /** Umbral mínimo para resultados trigram-only (sin prefix/substring). Default: 0.55 */
  trigramThreshold?: number;
  /** Máximo resultados devueltos. Default: sin límite. */
  limit?: number;
};

const DEFAULT_TRIGRAM_THRESHOLD = 0.55;
const PREFIX_BOOST = 0.5;
const SUBSTRING_BOOST = 0.35;

/**
 * Busca entries por similitud con el query.
 *
 * Estrategia de scoring (por entry):
 * 1. Si alguna palabra del texto empieza con el query → baseSim + 0.5
 * 2. Si el query es substring del texto → baseSim + 0.35
 * 3. Solo trigramas → requiere similitud ≥ trigramThreshold (default 0.55)
 *
 * baseSim = max(similitud vs texto completo, mejor similitud vs palabra individual)
 *
 * Resultados ordenados por score descendente.
 */
export function search<T>(
  query: string,
  entries: SearchEntry<T>[],
  options?: SearchOptions,
): SearchResult<T>[] {
  const q = normalize(query);
  if (q.length === 0) return [];

  const threshold = options?.trigramThreshold ?? DEFAULT_TRIGRAM_THRESHOLD;
  const results: SearchResult<T>[] = [];

  for (const entry of entries) {
    const norm = normalize(entry.text);
    const words = norm.split(/\s+/).filter(Boolean);

    const hasSubstring = norm.includes(q);
    const hasWordPrefix = words.some((w) => w.startsWith(q));

    let bestWordSim = 0;
    for (const w of words) {
      const s = similarity(q, w);
      if (s > bestWordSim) bestWordSim = s;
    }

    const fullSim = similarity(query, entry.text);
    const baseSim = Math.max(fullSim, bestWordSim);

    if (hasWordPrefix || hasSubstring) {
      const boost = hasWordPrefix ? PREFIX_BOOST : SUBSTRING_BOOST;
      results.push({ item: entry.item, score: Math.min(1, baseSim + boost) });
    } else if (baseSim >= threshold) {
      results.push({ item: entry.item, score: baseSim });
    }
  }

  results.sort((a, b) => b.score - a.score);

  if (options?.limit && results.length > options.limit) {
    return results.slice(0, options.limit);
  }
  return results;
}
