import { createBrowserStorageNamespace } from './storage';

export type Theme = 'light' | 'dark';

export type ThemeConfig = {
  /** Namespace para el storage (e.g. 'pymes-ui') */
  namespace: string;
  /** Key dentro del namespace (e.g. 'app:theme') */
  storageKey?: string;
  /** Atributo HTML donde se aplica el tema (default: 'data-theme') */
  attribute?: string;
};

export type ThemeManager = {
  get: () => Theme;
  toggle: () => Theme;
  apply: (theme?: Theme) => void;
};

/**
 * Crea un theme manager con storage persistente y detección de preferencia del sistema.
 */
export function createThemeManager(config: ThemeConfig): ThemeManager {
  const storageKey = config.storageKey ?? 'theme';
  const attribute = config.attribute ?? 'data-theme';
  const storage = createBrowserStorageNamespace({ namespace: config.namespace, hostAware: false });

  function get(): Theme {
    const stored = storage.getString(storageKey);
    if (stored === 'dark' || stored === 'light') return stored;
    return typeof window !== 'undefined' && window.matchMedia('(prefers-color-scheme: dark)').matches
      ? 'dark'
      : 'light';
  }

  function apply(theme?: Theme): void {
    const t = theme ?? get();
    if (typeof document !== 'undefined') {
      document.documentElement.setAttribute(attribute, t);
    }
  }

  function toggle(): Theme {
    const next = get() === 'dark' ? 'light' : 'dark';
    storage.setString(storageKey, next);
    apply(next);
    return next;
  }

  return { get, toggle, apply };
}
