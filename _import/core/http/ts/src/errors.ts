/**
 * Convierte errores de fetch en texto entendible.
 * Detecta errores de red (Failed to fetch, NetworkError, etc.) y los reemplaza
 * por un mensaje descriptivo.
 */
export function formatFetchErrorForUser(err: unknown, unreachableMessage: string): string {
  const msg = err instanceof Error ? err.message : String(err);
  if (/failed to fetch|networkerror|network request failed|load failed/i.test(msg)) {
    return unreachableMessage;
  }
  return stripHttpErrorPrefix(msg);
}

/** Quita el prefijo "HttpError: " que añade el cliente HTTP. */
export function stripHttpErrorPrefix(message: string): string {
  return message.replace(/^HttpError:\s*/i, "").trim();
}
