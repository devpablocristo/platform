export type RequestOptions = {
  method?: string;
  body?: unknown;
  rawBody?: BodyInit | null;
  headers?: Record<string, string>;
  skipJSONContentType?: boolean;
  signal?: AbortSignal;
  /**
   * Compatibility option for the module-level request helpers. Instance clients
   * always use the base URL supplied to createHttpClient.
   */
  baseURLs?: string[];
};

export type FetchImplementation = (
  input: RequestInfo | URL,
  init?: RequestInit,
) => Promise<Response>;

export type ResolveHeadersContext = Readonly<{
  path: string;
  url: string;
  method: string;
  signal?: AbortSignal;
}>;

export type ResolveHeaders = (
  context: ResolveHeadersContext,
) => HeadersInit | undefined | Promise<HeadersInit | undefined>;

export type HttpClientOptions = {
  baseURL: string;
  fetch?: FetchImplementation;
  resolveHeaders?: ResolveHeaders;
};

export type HttpClient = {
  requestResponse(path: string, options?: RequestOptions): Promise<Response>;
  request<T>(path: string, options?: RequestOptions): Promise<T>;
};

export class HttpError extends Error {
  constructor(
    message: string,
    readonly status?: number,
    readonly body?: string,
  ) {
    super(message);
    this.name = "HttpError";
  }
}

function normalizeBaseURLs(options: RequestOptions): string[] {
  const explicit = (options.baseURLs ?? []).map((value) => value.trim()).filter(Boolean);
  if (explicit.length > 0) {
    return [...new Set(explicit)];
  }
  return [""];
}

function joinURL(baseURL: string, path: string): string {
  if (!baseURL) {
    return path;
  }
  if (/^https?:\/\//i.test(path)) {
    return path;
  }
  const cleanBase = baseURL.endsWith("/") ? baseURL.slice(0, -1) : baseURL;
  const cleanPath = path.startsWith("/") ? path : `/${path}`;
  return `${cleanBase}${cleanPath}`;
}

async function readError(response: Response): Promise<HttpError> {
  const text = await response.text().catch(() => response.statusText);
  let message = text || response.statusText || `HTTP ${response.status}`;

  if (text) {
    try {
      const body = JSON.parse(text) as
        | { error?: string | { message?: string; code?: string }; message?: string }
        | undefined;
      if (body?.error && typeof body.error === "object") {
        message = body.error.message || body.error.code || message;
      } else if (typeof body?.error === "string") {
        message = body.error;
      } else if (body?.message) {
        message = body.message;
      }
    } catch {
      // keep text as-is
    }
  }

  return new HttpError(message, response.status, text);
}

type RequestTransport = {
  baseURLs: string[];
  fetch: FetchImplementation;
  resolveHeaders?: ResolveHeaders;
};

async function buildHeaders(
  path: string,
  url: string,
  options: RequestOptions,
  resolveHeaders?: ResolveHeaders,
): Promise<Headers> {
  const method = options.method ?? "GET";
  const headers = new Headers(
    await resolveHeaders?.({
      path,
      url,
      method,
      signal: options.signal,
    }),
  );

  new Headers(options.headers).forEach((value, key) => {
    headers.set(key, value);
  });

  if (
    !options.skipJSONContentType &&
    !headers.has("Content-Type") &&
    !(typeof FormData !== "undefined" && options.rawBody instanceof FormData)
  ) {
    headers.set("Content-Type", "application/json");
  }

  return headers;
}

async function requestResponseWithTransport(
  path: string,
  options: RequestOptions,
  transport: RequestTransport,
): Promise<Response> {
  const requestBody =
    options.rawBody ??
    (options.body !== undefined ? JSON.stringify(options.body) : undefined);

  let lastError: unknown = null;
  for (const baseURL of transport.baseURLs) {
    const url = joinURL(baseURL, path);
    try {
      const headers = await buildHeaders(path, url, options, transport.resolveHeaders);
      const response = await transport.fetch(url, {
        method: options.method ?? "GET",
        headers,
        body: requestBody,
        signal: options.signal,
      });

      if (!response.ok) {
        throw await readError(response);
      }
      return response;
    } catch (error) {
      lastError = error;
      if (error instanceof HttpError) {
        throw error;
      }
    }
  }

  if (lastError instanceof Error) {
    throw lastError;
  }
  throw new Error("No se pudo completar la solicitud");
}

async function requestWithTransport<T>(
  path: string,
  options: RequestOptions,
  transport: RequestTransport,
): Promise<T> {
  const response = await requestResponseWithTransport(path, options, transport);
  if (response.status === 204) {
    return undefined as T;
  }
  const contentType = response.headers.get("content-type") ?? "";
  if (contentType.includes("application/json")) {
    return (await response.json()) as T;
  }
  return (await response.text()) as T;
}

function defaultFetch(input: RequestInfo | URL, init?: RequestInit): Promise<Response> {
  return globalThis.fetch(input, init);
}

/**
 * Creates an isolated HTTP client. Configuration is captured by this instance;
 * no credentials or tenant headers are read from module-level state.
 */
export function createHttpClient(options: HttpClientOptions): HttpClient {
  const transport: RequestTransport = {
    baseURLs: [options.baseURL.trim()],
    fetch: options.fetch ?? defaultFetch,
    resolveHeaders: options.resolveHeaders,
  };

  return {
    requestResponse(path, requestOptions = {}) {
      return requestResponseWithTransport(path, requestOptions, transport);
    },
    request<T>(path: string, requestOptions: RequestOptions = {}) {
      return requestWithTransport<T>(path, requestOptions, transport);
    },
  };
}

/**
 * Compatibility helper backed by globalThis.fetch.
 */
export async function requestResponse(
  path: string,
  options: RequestOptions = {},
): Promise<Response> {
  return requestResponseWithTransport(path, options, {
    baseURLs: normalizeBaseURLs(options),
    fetch: defaultFetch,
  });
}

/**
 * Compatibility helper backed by globalThis.fetch.
 */
export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  return requestWithTransport<T>(path, options, {
    baseURLs: normalizeBaseURLs(options),
    fetch: defaultFetch,
  });
}
