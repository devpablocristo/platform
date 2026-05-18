/* eslint-disable react-refresh/only-export-components -- provider + hook en mismo archivo */
import { createContext, useContext, useEffect, useMemo, useState, type PropsWithChildren } from 'react';
import { createBrowserStorageNamespace } from '../storage';
import { formatMessage } from './formatMessage';
import { localeTagForLanguage } from './localeTag';
import { toSentenceCase } from './sentenceCase';
import type { FlatMessages, LanguageCode, TranslationVariables, TranslationsByLanguage } from './types';

export type I18nConfig = {
  /** Namespace para persistir el idioma (e.g. 'pymes-ui') */
  namespace: string;
  /** Key de storage (e.g. 'app:language') */
  storageKey?: string;
  /** Idioma default (default: 'es') */
  defaultLanguage?: LanguageCode;
  /** Claves absolutas en localStorage (sin namespace) a leer una vez y migrar al storage con namespace */
  legacyLanguageKeys?: string[];
  /** Idiomas soportados con label keys */
  supportedLanguages?: Array<{ code: LanguageCode; labelKey: string }>;
  /** Mensajes agrupados por idioma */
  messages: Record<LanguageCode, FlatMessages>;
  /** Función opcional para localizar texto (e.g. vocabulary replacement) */
  localizeText?: (text: string) => string;
};

export type I18nContextValue = {
  language: LanguageCode;
  setLanguage: (language: LanguageCode) => void;
  t: (key: string, variables?: TranslationVariables) => string;
  localizeText: (text: string) => string;
  sentenceCase: (text: string) => string;
  localizeUiText: (text: string) => string;
  options: Array<{ code: LanguageCode; labelKey: string }>;
};

/**
 * Merge de múltiples fuentes de traducciones.
 */
export function mergeMessages(...sources: TranslationsByLanguage[]): Record<LanguageCode, FlatMessages> {
  return {
    es: Object.assign({}, ...sources.map((s) => s.es)),
    en: Object.assign({}, ...sources.map((s) => s.en)),
  };
}

/**
 * Crea un provider de i18n con storage persistente.
 * Retorna { Provider, useI18n } para uso en la app.
 */
export function createI18nProvider(config: I18nConfig) {
  const defaultLanguage = config.defaultLanguage ?? 'es';
  const storageKey = config.storageKey ?? 'language';
  const supportedLanguages = config.supportedLanguages ?? [
    { code: 'es', labelKey: 'common.language.es' },
    { code: 'en', labelKey: 'common.language.en' },
  ];
  const storage = createBrowserStorageNamespace({ namespace: config.namespace, hostAware: false });
  const identity = (text: string) => text;
  const localizeText = config.localizeText ?? identity;

  function getMessage(language: LanguageCode, key: string, variables?: TranslationVariables): string {
    return formatMessage(config.messages, language, defaultLanguage, key, variables);
  }

  function readStoredLanguage(): LanguageCode {
    if (typeof window === 'undefined') return defaultLanguage;
    for (const legacyKey of config.legacyLanguageKeys ?? []) {
      try {
        const raw = window.localStorage.getItem(legacyKey);
        if (raw === 'en' || raw === 'es') {
          window.localStorage.removeItem(legacyKey);
          storage.setString(storageKey, raw);
          return raw;
        }
      } catch {
        /* private mode */
      }
    }
    const stored = storage.getString(storageKey);
    return stored === 'en' || stored === 'es' ? stored : defaultLanguage;
  }

  const defaultContext: I18nContextValue = {
    language: defaultLanguage,
    setLanguage: () => undefined,
    t: (key, variables) => formatMessage(config.messages, defaultLanguage, defaultLanguage, key, variables),
    localizeText,
    sentenceCase: toSentenceCase,
    localizeUiText: (text) => toSentenceCase(localizeText(text)),
    options: supportedLanguages,
  };

  const I18nContext = createContext<I18nContextValue>(defaultContext);

  function Provider({ children, initialLanguage }: PropsWithChildren<{ initialLanguage?: LanguageCode }>) {
    const [language, setLanguageState] = useState<LanguageCode>(() => initialLanguage ?? readStoredLanguage());

    useEffect(() => {
      storage.setString(storageKey, language);
      if (typeof document !== 'undefined') {
        document.documentElement.lang = localeTagForLanguage(language);
      }
    }, [language]);

    const value = useMemo<I18nContextValue>(
      () => ({
        language,
        setLanguage: setLanguageState,
        t: (key, variables) => getMessage(language, key, variables),
        localizeText,
        sentenceCase: toSentenceCase,
        localizeUiText: (text) => toSentenceCase(localizeText(text)),
        options: supportedLanguages,
      }),
      [language],
    );

    return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
  }

  function useI18n(): I18nContextValue {
    return useContext(I18nContext);
  }

  return { Provider, useI18n, toSentenceCase };
}

export { toSentenceCase };
