/**
 * Generación de trigramas y similitud (Dice coefficient).
 *
 * Funciones puras — no tienen conocimiento de UI ni de búsqueda.
 */

import { normalize } from "./normalize";

/** Extrae trigramas de un string (normalizado internamente). */
export function trigrams(text: string): Set<string> {
  const norm = normalize(text);
  const padded = `  ${norm} `;
  const result = new Set<string>();
  for (let i = 0; i <= padded.length - 3; i++) {
    result.add(padded.slice(i, i + 3));
  }
  return result;
}

/** Coeficiente de Dice: 2 * |A ∩ B| / (|A| + |B|). Rango [0, 1]. */
export function similarity(a: string, b: string): number {
  const ta = trigrams(a);
  const tb = trigrams(b);
  if (ta.size === 0 && tb.size === 0) return 1;
  if (ta.size === 0 || tb.size === 0) return 0;
  let intersection = 0;
  for (const t of ta) {
    if (tb.has(t)) intersection++;
  }
  return (2 * intersection) / (ta.size + tb.size);
}
