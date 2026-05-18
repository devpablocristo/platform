export type RequestOptions = {
  method?: string;
  body?: unknown;
  rawBody?: BodyInit | null;
  headers?: Record<string, string>;
  skipJSONContentType?: boolean;
  baseURLs?: string[];
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

export async function requestResponse(path: string, options: RequestOptions = {}): Promise<Response> {
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

  const requestBody =
    options.rawBody ??
    (options.body !== undefined ? JSON.stringify(options.body) : undefined);

  let lastError: unknown = null;
  for (const baseURL of normalizeBaseURLs(options)) {
    try {
      const response = await fetch(joinURL(baseURL, path), {
        method: options.method ?? "GET",
        headers,
        body: requestBody,
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

export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const response = await requestResponse(path, options);
  if (response.status === 204) {
    return undefined as T;
  }
  const contentType = response.headers.get("content-type") ?? "";
  if (contentType.includes("application/json")) {
    return (await response.json()) as T;
  }
  return (await response.text()) as T;
}
