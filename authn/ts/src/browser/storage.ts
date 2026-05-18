import { createBrowserStorageNamespace } from "@devpablocristo/core-browser/storage";

export interface TokenPair {
  access_token: string;
  refresh_token?: string | null;
}

export interface BrowserTokenStorageOptions {
  namespace: string;
  storage?: Storage;
  hostAware?: boolean;
  accessTokenKey?: string;
  refreshTokenKey?: string;
  legacyKeys?: string[];
}

export interface BrowserTokenStorage {
  key(name: string): string;
  getAccessToken(): string | null;
  getRefreshToken(): string | null;
  setAccessToken(token: string): void;
  setRefreshToken(token: string): void;
  setTokens(tokens: TokenPair): void;
  clear(): void;
}

export function createBrowserTokenStorage(options: BrowserTokenStorageOptions): BrowserTokenStorage {
  const accessTokenKey = options.accessTokenKey ?? "access_token";
  const refreshTokenKey = options.refreshTokenKey ?? "refresh_token";
  const storage = createBrowserStorageNamespace({
    namespace: options.namespace,
    storage: options.storage,
    hostAware: options.hostAware,
    legacyKeys: options.legacyKeys ?? [accessTokenKey, refreshTokenKey],
  });

  return {
    key: storage.key,
    getAccessToken: () => storage.getString(accessTokenKey),
    getRefreshToken: () => storage.getString(refreshTokenKey),
    setAccessToken: (token: string) => storage.setString(accessTokenKey, token),
    setRefreshToken: (token: string) => storage.setString(refreshTokenKey, token),
    setTokens: (tokens: TokenPair) => {
      storage.setString(accessTokenKey, tokens.access_token);
      if (tokens.refresh_token) {
        storage.setString(refreshTokenKey, tokens.refresh_token);
      }
    },
    clear: storage.clear,
  };
}
