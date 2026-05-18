import type { FlatMessages } from "./types";

/**
 * Aplana un árbol de mensajes anidados a claves con punto (p. ej. `common.loading`).
 */
export function flattenNestedMessages(obj: Record<string, unknown>, prefix = ""): FlatMessages {
  const out: FlatMessages = {};
  for (const [k, v] of Object.entries(obj)) {
    const key = prefix ? `${prefix}.${k}` : k;
    if (v !== null && typeof v === "object" && !Array.isArray(v)) {
      Object.assign(out, flattenNestedMessages(v as Record<string, unknown>, key));
    } else if (typeof v === "string") {
      out[key] = v;
    }
  }
  return out;
}
