export interface BrowserStorageNamespaceOptions {
  namespace: string;
  storage?: Storage;
  hostAware?: boolean;
  legacyKeys?: string[];
}

export interface BrowserStorageNamespace {
  key(name: string): string;
  getString(name: string): string | null;
  setString(name: string, value: string): void;
  remove(name: string): void;
  getJSON<T>(name: string): T | null;
  setJSON(name: string, value: unknown | null | undefined): void;
  clear(): void;
}

function resolveStorage(custom?: Storage): Storage | null {
  if (custom) {
    return custom;
  }
  if (typeof window === "undefined") {
    return null;
  }
  return window.localStorage;
}

function resolvePrefix(namespace: string, hostAware: boolean): string {
  const cleanNamespace = namespace.trim();
  if (!hostAware || typeof window === "undefined") {
    return `${cleanNamespace}:`;
  }
  return `${cleanNamespace}:${window.location.host}:`;
}

export function createBrowserStorageNamespace(
  options: BrowserStorageNamespaceOptions,
): BrowserStorageNamespace {
  const storage = resolveStorage(options.storage);
  const hostAware = options.hostAware ?? true;
  const legacyKeys = options.legacyKeys ?? [];

  function prefix(): string {
    return resolvePrefix(options.namespace, hostAware);
  }

  function key(name: string): string {
    return `${prefix()}${name}`;
  }

  function getString(name: string): string | null {
    return storage?.getItem(key(name)) ?? null;
  }

  function setString(name: string, value: string): void {
    storage?.setItem(key(name), value);
  }

  function remove(name: string): void {
    storage?.removeItem(key(name));
  }

  function getJSON<T>(name: string): T | null {
    const raw = getString(name);
    if (!raw) {
      return null;
    }
    try {
      return JSON.parse(raw) as T;
    } catch {
      return null;
    }
  }

  function setJSON(name: string, value: unknown | null | undefined): void {
    if (value === null || value === undefined) {
      remove(name);
      return;
    }
    setString(name, JSON.stringify(value));
  }

  function clear(): void {
    if (!storage) {
      return;
    }

    legacyKeys.forEach((legacyKey) => storage.removeItem(legacyKey));

    const prefixValue = prefix();
    const toRemove: string[] = [];
    for (let index = 0; index < storage.length; index += 1) {
      const itemKey = storage.key(index);
      if (itemKey && itemKey.startsWith(prefixValue)) {
        toRemove.push(itemKey);
      }
    }
    toRemove.forEach((itemKey) => storage.removeItem(itemKey));
  }

  return {
    key,
    getString,
    setString,
    remove,
    getJSON,
    setJSON,
    clear,
  };
}
