import {
  HttpError,
  request as coreRequest,
  requestResponse as coreRequestResponse,
  type RequestOptions as CoreRequestOptions,
} from "@devpablocristo/core-http/fetch";

export { HttpError } from "@devpablocristo/core-http/fetch";

export type TokenProvider = () => Promise<string | null>;

let tokenProvider: TokenProvider | null = null;

export function registerTokenProvider(provider: TokenProvider): void {
  tokenProvider = provider;
}

export function resetTokenProvider(): void {
  tokenProvider = null;
}

export type RequestOptions = CoreRequestOptions & {
  orgId?: string;
};

function readEnv(name: string): string | undefined {
  const env = (import.meta as ImportMeta & { env?: Record<string, string | undefined> }).env;
  return env?.[name];
}

function isLocalhost(): boolean {
  if (typeof window === "undefined") {
    return true;
  }

  return ["localhost", "127.0.0.1"].includes(window.location.hostname);
}

function resolveDefaultBaseURL(): string {
  const configured = readEnv("VITE_API_URL")?.trim();
  if (configured) {
    return configured;
  }

  if (typeof window === "undefined") {
    return "http://localhost:8100";
  }

  const protocol = window.location.protocol || "http:";
  const hostname = window.location.hostname || "localhost";
  return `${protocol}//${hostname}:8100`;
}

function resolveBaseURLs(options: RequestOptions): string[] {
  const explicit = (options.baseURLs ?? []).map((value) => value.trim()).filter(Boolean);
  if (explicit.length > 0) {
    return [...new Set(explicit)];
  }
  return [resolveDefaultBaseURL()];
}

function resolveLocalAPIKeyFallback(): string | null {
  if (!isLocalhost()) {
    return null;
  }

  return "psk_local_admin";
}

async function buildHeaders(options: RequestOptions): Promise<Record<string, string>> {
  const headers: Record<string, string> = {
    ...(options.headers ?? {}),
  };

  if (
    !options.skipJSONContentType &&
    !("Content-Type" in headers) &&
    !(typeof FormData !== "undefined" && options.rawBody instanceof FormData)
  ) {
    headers["Content-Type"] = "application/json";
  }

  const token = tokenProvider ? await tokenProvider() : null;
  const apiKey = readEnv("VITE_API_KEY")?.trim() || resolveLocalAPIKeyFallback();
  // Do not send X-API-KEY with Bearer by default: middleware would fall back to service identity and hide the human session.
  const allowDualAuth = readEnv("VITE_DEV_ALLOW_API_KEY_WITH_CLERK_BEARER") === "true";

  if (token) {
    headers.Authorization = `Bearer ${token}`;
    if (allowDualAuth && apiKey) {
      headers["X-API-KEY"] = apiKey;
    }
  } else if (apiKey) {
    headers["X-API-KEY"] = apiKey;
  }

  if (options.orgId) {
    headers["X-Org-ID"] = options.orgId;
  }

  return headers;
}

export async function requestResponse(path: string, options: RequestOptions = {}): Promise<Response> {
  return coreRequestResponse(path, {
    ...options,
    headers: await buildHeaders(options),
    baseURLs: resolveBaseURLs(options),
  });
}

export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  return coreRequest<T>(path, {
    ...options,
    headers: await buildHeaders(options),
    baseURLs: resolveBaseURLs(options),
  });
}
