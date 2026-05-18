import type { FlatMessages, LanguageCode, TranslationVariables } from "./types";

/**
 * Soporta `{var}` y `{{var}}` (compat con plantillas tipo i18next).
 */
export function interpolate(template: string, variables?: TranslationVariables): string {
  const normalized = template.replace(/\{\{(\w+)\}\}/g, "{$1}");
  if (!variables) {
    return normalized;
  }
  return normalized.replace(/\{(\w+)\}/g, (_match, key: string) => String(variables[key] ?? ""));
}

export function formatMessage(
  messages: Record<LanguageCode, FlatMessages>,
  language: LanguageCode,
  defaultLanguage: LanguageCode,
  key: string,
  variables?: TranslationVariables,
): string {
  const template = messages[language]?.[key] ?? messages[defaultLanguage]?.[key] ?? key;
  return interpolate(template, variables);
}
