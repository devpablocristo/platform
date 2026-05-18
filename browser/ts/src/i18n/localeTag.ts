import type { LanguageCode } from "./types";

/** Tag BCP-47 recomendado para `toLocaleString` según idioma de UI. */
export function localeTagForLanguage(code: LanguageCode): string {
  return code === "es" ? "es-AR" : "en-US";
}
