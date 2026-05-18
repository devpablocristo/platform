/**
 * Extrae un mensaje legible de un error de API de autenticación (Clerk u otros).
 * Busca el patrón { errors: [{ message: string }] } que usan Clerk y otros providers.
 */
export function formatAuthAPIUserMessage(err: unknown, fallback: string): string {
  if (err != null && typeof err === 'object' && 'errors' in err) {
    const errors = (err as { errors?: unknown[] }).errors;
    if (Array.isArray(errors) && errors.length > 0) {
      const first = errors[0];
      if (first && typeof first === 'object' && 'message' in first) {
        const msg = String((first as { message: unknown }).message).trim();
        if (msg) return msg;
      }
    }
  }
  if (err instanceof Error && err.message.trim()) {
    return err.message.trim();
  }
  return fallback;
}
