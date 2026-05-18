/**
 * Combina dos mapas por clave string: se conservan primero las entradas de `base`
 * cuya clave no existe en `override`; luego se añaden (o reemplazan) todas las de `override`.
 * El orden de iteración queda alineado con el merge histórico de catálogos estáticos + CRUD.
 */
export function mergeRecordsPreferOverride<T>(
  base: Record<string, T>,
  override: Record<string, T>,
): Record<string, T> {
  const out: Record<string, T> = {};
  for (const key of Object.keys(base)) {
    if (!Object.prototype.hasOwnProperty.call(override, key)) {
      out[key] = base[key] as T;
    }
  }
  for (const key of Object.keys(override)) {
    out[key] = override[key] as T;
  }
  return out;
}
