export interface EnvLike {
  [key: string]: string | undefined;
}

export interface ClerkBrowserConfig {
  publishableKey: string;
  enabled: boolean;
}

function readDefaultEnv(): EnvLike {
  return ((import.meta as ImportMeta & { env?: EnvLike }).env ?? {}) as EnvLike;
}

export function resolveClerkBrowserConfig(env: EnvLike = readDefaultEnv()): ClerkBrowserConfig {
  const publishableKey = env.VITE_CLERK_PUBLISHABLE_KEY?.trim() ?? "";
  return {
    publishableKey,
    enabled: publishableKey.length > 0,
  };
}

export function createClerkTokenProvider(
  getToken: () => Promise<string | null | undefined>,
): () => Promise<string | null> {
  return async () => (await getToken()) ?? null;
}
