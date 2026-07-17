export type HttpErrorKind =
  | "validation"
  | "authentication"
  | "authorization"
  | "not_found"
  | "conflict"
  | "rate_limit"
  | "network"
  | "timeout"
  | "server"
  | "unknown";

export type NormalizedHttpError = {
  kind: HttpErrorKind;
  status?: number;
  code?: string;
  message?: string;
  details?: string;
  fieldErrors?: Record<string, string[]>;
  requestId?: string;
  retryable: boolean;
  cause: unknown;
};

export type NormalizedHttpErrorPatch = Partial<Omit<NormalizedHttpError, "cause">>;

export type HttpErrorBodyAdapter = (
  body: unknown,
  error: unknown,
) => NormalizedHttpErrorPatch | undefined;

export type NormalizeHttpErrorOptions = {
  fallbackMessage?: string;
  bodyAdapter?: HttpErrorBodyAdapter;
};

type UnknownRecord = Record<string, unknown>;

function isRecord(value: unknown): value is UnknownRecord {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function nonEmptyString(value: unknown): string | undefined {
  if (typeof value !== "string") return undefined;
  const trimmed = value.trim();
  return trimmed.length > 0 ? trimmed : undefined;
}

function statusNumber(value: unknown): number | undefined {
  return typeof value === "number" && Number.isFinite(value) ? value : undefined;
}

function codeString(value: unknown): string | undefined {
  if (typeof value === "number" && Number.isFinite(value)) return String(value);
  return nonEmptyString(value);
}

function parseJSON(value: unknown): unknown {
  if (typeof value !== "string") return value;
  const trimmed = value.trim();
  if (!trimmed.startsWith("{") && !trimmed.startsWith("[")) return value;
  try {
    return JSON.parse(trimmed) as unknown;
  } catch {
    return value;
  }
}

function responseOf(error: unknown): UnknownRecord | undefined {
  if (!isRecord(error) || !isRecord(error.response)) return undefined;
  return error.response;
}

function bodyOf(error: unknown): unknown {
  const response = responseOf(error);
  if (response && "data" in response) return parseJSON(response.data);
  if (isRecord(error) && "body" in error) return parseJSON(error.body);
  return parseJSON(error);
}

function nestedErrorOf(body: unknown): UnknownRecord | undefined {
  if (!isRecord(body) || !isRecord(body.error)) return undefined;
  return body.error;
}

function statusOf(error: unknown, body: unknown): number | undefined {
  const response = responseOf(error);
  const nested = nestedErrorOf(body);
  return (
    statusNumber(response?.status) ??
    (isRecord(error) ? statusNumber(error.status) : undefined) ??
    statusNumber(nested?.status) ??
    (isRecord(body) ? statusNumber(body.status) : undefined)
  );
}

function headerValue(headers: unknown, key: string): string | undefined {
  if (!headers) return undefined;
  if (isRecord(headers)) {
    const direct = nonEmptyString(headers[key]) ?? nonEmptyString(headers[key.toLowerCase()]);
    if (direct) return direct;
  }
  if (
    typeof headers === "object" &&
    headers !== null &&
    "get" in headers &&
    typeof headers.get === "function"
  ) {
    return nonEmptyString(headers.get(key));
  }
  return undefined;
}

function requestIdOf(error: unknown, body: unknown): string | undefined {
  const response = responseOf(error);
  const nested = nestedErrorOf(body);
  return (
    (isRecord(body)
      ? nonEmptyString(body.requestId) ??
        nonEmptyString(body.request_id) ??
        nonEmptyString(body.trace_id)
      : undefined) ??
    nonEmptyString(nested?.requestId) ??
    nonEmptyString(nested?.request_id) ??
    headerValue(response?.headers, "x-request-id")
  );
}

function fieldErrorsOf(value: unknown): Record<string, string[]> | undefined {
  if (!isRecord(value)) return undefined;
  const entries: Array<[string, string[]]> = [];
  for (const [field, messages] of Object.entries(value)) {
    if (Array.isArray(messages)) {
      const normalized = messages.map(nonEmptyString).filter((message): message is string => Boolean(message));
      if (normalized.length > 0) entries.push([field, normalized]);
      continue;
    }
    const message = nonEmptyString(messages);
    if (message) entries.push([field, [message]]);
  }
  return entries.length > 0 ? Object.fromEntries(entries) : undefined;
}

function genericBodyPatch(body: unknown): NormalizedHttpErrorPatch {
  if (typeof body === "string") {
    return { message: nonEmptyString(body) };
  }
  if (!isRecord(body)) return {};

  const nested = nestedErrorOf(body);
  const nestedString = typeof body.error === "string" ? nonEmptyString(body.error) : undefined;
  return {
    status: statusNumber(nested?.status) ?? statusNumber(body.status),
    code: codeString(nested?.code) ?? codeString(body.code),
    message:
      nonEmptyString(nested?.message) ??
      nestedString ??
      nonEmptyString(body.message) ??
      nonEmptyString(body.error_message),
    details:
      nonEmptyString(nested?.details) ??
      nonEmptyString(nested?.detail) ??
      nonEmptyString(body.details) ??
      nonEmptyString(body.detail),
    fieldErrors:
      fieldErrorsOf(nested?.fieldErrors) ??
      fieldErrorsOf(nested?.field_errors) ??
      fieldErrorsOf(body.fieldErrors) ??
      fieldErrorsOf(body.field_errors) ??
      fieldErrorsOf(body.errors),
    requestId: requestIdOf({}, body),
  };
}

function errorMessageOf(error: unknown): string | undefined {
  if (error instanceof Error) return nonEmptyString(error.message);
  if (isRecord(error)) return nonEmptyString(error.message);
  return typeof error === "string" ? nonEmptyString(error) : undefined;
}

function errorCodeOf(error: unknown): string | undefined {
  return isRecord(error) ? codeString(error.code) : undefined;
}

function isTimeout(error: unknown, message: string | undefined): boolean {
  const code = errorCodeOf(error)?.toUpperCase();
  return (
    code === "ECONNABORTED" ||
    code === "ETIMEDOUT" ||
    code === "ERR_TIMEOUT" ||
    /\btimeout\b|timed out|tiempo de espera/i.test(message ?? "")
  );
}

function isNetworkError(error: unknown, message: string | undefined): boolean {
  const code = errorCodeOf(error)?.toUpperCase();
  return (
    code === "ERR_NETWORK" ||
    code === "ENETUNREACH" ||
    code === "ECONNREFUSED" ||
    /failed to fetch|networkerror|network request failed|load failed|network error/i.test(
      message ?? "",
    )
  );
}

function kindFor(status: number | undefined, error: unknown, message: string | undefined): HttpErrorKind {
  if (isTimeout(error, message)) return "timeout";
  if (status === undefined && isNetworkError(error, message)) return "network";
  if (status === 400 || status === 422) return "validation";
  if (status === 401) return "authentication";
  if (status === 403) return "authorization";
  if (status === 404) return "not_found";
  if (status === 409) return "conflict";
  if (status === 429) return "rate_limit";
  if (status !== undefined && status >= 500) return "server";
  return "unknown";
}

function isRetryable(kind: HttpErrorKind): boolean {
  return kind === "network" || kind === "timeout" || kind === "rate_limit" || kind === "server";
}

/**
 * Convierte errores de transporte desconocidos en un contrato estable.
 *
 * No depende de Axios: reconoce su shape estructuralmente. Los productos pueden
 * aportar un bodyAdapter para envelopes propios sin filtrar reglas de negocio a
 * platform.
 */
export function normalizeHttpError(
  error: unknown,
  options: NormalizeHttpErrorOptions = {},
): NormalizedHttpError {
  const body = bodyOf(error);
  const generic = genericBodyPatch(body);
  const adapted = options.bodyAdapter?.(body, error) ?? {};
  const directMessage = errorMessageOf(error);
  const status = adapted.status ?? generic.status ?? statusOf(error, body);
  const message =
    adapted.message ??
    generic.message ??
    directMessage ??
    nonEmptyString(options.fallbackMessage);
  const kind = adapted.kind ?? kindFor(status, error, message);

  return {
    kind,
    status,
    code: adapted.code ?? generic.code ?? errorCodeOf(error),
    message,
    details: adapted.details ?? generic.details,
    fieldErrors: adapted.fieldErrors ?? generic.fieldErrors,
    requestId: adapted.requestId ?? generic.requestId ?? requestIdOf(error, body),
    retryable: adapted.retryable ?? isRetryable(kind),
    cause: error,
  };
}

/**
 * Convierte errores de fetch en texto entendible.
 * Detecta errores de red (Failed to fetch, NetworkError, etc.) y los reemplaza
 * por un mensaje descriptivo.
 */
export function formatFetchErrorForUser(err: unknown, unreachableMessage: string): string {
  const normalized = normalizeHttpError(err);
  if (normalized.kind === "network") {
    return unreachableMessage;
  }
  return stripHttpErrorPrefix(normalized.details ?? normalized.message ?? String(err));
}

/** Quita el prefijo "HttpError: " que añade el cliente HTTP. */
export function stripHttpErrorPrefix(message: string): string {
  return message.replace(/^HttpError:\s*/i, "").trim();
}
