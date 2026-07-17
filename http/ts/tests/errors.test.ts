import { HttpError } from "../src/fetch";
import { normalizeHttpError } from "../src/errors";

describe("normalizeHttpError", () => {
  it("normalizes axios-like validation errors", () => {
    const cause = {
      response: {
        status: 422,
        headers: { "x-request-id": "req-422" },
        data: {
          error: {
            code: "invalid_input",
            message: "Revisá los datos",
            field_errors: { name: ["Es obligatorio"] },
          },
        },
      },
    };

    expect(normalizeHttpError(cause)).toMatchObject({
      kind: "validation",
      status: 422,
      code: "invalid_input",
      message: "Revisá los datos",
      fieldErrors: { name: ["Es obligatorio"] },
      requestId: "req-422",
      retryable: false,
      cause,
    });
  });

  it("normalizes direct structured error envelopes", () => {
    expect(
      normalizeHttpError({
        success: false,
        message: "No se pudo guardar",
        error: { status: 409, code: 17, details: "El registro ya existe" },
      }),
    ).toMatchObject({
      kind: "conflict",
      status: 409,
      code: "17",
      message: "No se pudo guardar",
      details: "El registro ya existe",
      retryable: false,
    });
  });

  it("parses JSON bodies carried by HttpError", () => {
    const cause = new HttpError(
      "upstream failed",
      503,
      JSON.stringify({ message: "Servicio temporalmente no disponible", request_id: "req-503" }),
    );

    expect(normalizeHttpError(cause)).toMatchObject({
      kind: "server",
      status: 503,
      message: "Servicio temporalmente no disponible",
      requestId: "req-503",
      retryable: true,
    });
  });

  it.each([
    [{ code: "ERR_NETWORK", message: "Network Error" }, "network"],
    [{ code: "ECONNABORTED", message: "timeout of 1000ms exceeded" }, "timeout"],
  ])("classifies transport failures", (cause, kind) => {
    expect(normalizeHttpError(cause)).toMatchObject({ kind, retryable: true });
  });

  it("lets a product adapter enrich a body", () => {
    const cause = { response: { status: 400, data: { product_reason: "duplicate" } } };
    const normalized = normalizeHttpError(cause, {
      bodyAdapter: (body) =>
        typeof body === "object" && body !== null && "product_reason" in body
          ? { kind: "conflict", code: String(body.product_reason), details: "Dato repetido" }
          : undefined,
    });

    expect(normalized).toMatchObject({
      kind: "conflict",
      status: 400,
      code: "duplicate",
      details: "Dato repetido",
      retryable: false,
    });
  });

  it("uses a safe fallback for malformed values", () => {
    expect(normalizeHttpError(null, { fallbackMessage: "No se pudo completar la solicitud" })).toMatchObject({
      kind: "unknown",
      message: "No se pudo completar la solicitud",
      retryable: false,
      cause: null,
    });
  });
});
