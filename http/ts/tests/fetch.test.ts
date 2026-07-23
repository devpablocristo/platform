import {
  createHttpClient,
  HttpError,
  request,
  requestResponse,
  type FetchImplementation,
  type ResolveHeadersContext,
} from "../src/fetch";

describe("platform-http fetch", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("requests JSON payloads", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(JSON.stringify({ ok: true }), {
        status: 200,
        headers: { "content-type": "application/json" },
      }),
    );

    await expect(request<{ ok: boolean }>("/v1/test")).resolves.toEqual({ ok: true });
  });

  it("throws parsed http errors", async () => {
    vi.spyOn(globalThis, "fetch").mockImplementation(async () =>
      new Response(JSON.stringify({ error: { message: "forbidden" } }), {
        status: 403,
        headers: { "content-type": "application/json" },
      }),
    );

    await expect(request("/v1/test")).rejects.toEqual(expect.any(HttpError));
    await expect(request("/v1/test")).rejects.toMatchObject({ message: "forbidden", status: 403 });
  });

  it("joins explicit base urls", async () => {
    const fetchSpy = vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(JSON.stringify({ ok: true }), {
        status: 200,
        headers: { "content-type": "application/json" },
      }),
    );

    await requestResponse("/v1/test", { baseURLs: ["http://localhost:9999"] });

    expect(fetchSpy).toHaveBeenCalledWith(
      "http://localhost:9999/v1/test",
      expect.objectContaining({ method: "GET" }),
    );
  });

  it("creates isolated clients with their own base urls and fetch implementations", async () => {
    const firstFetch = vi.fn(async () => Response.json({ client: "first" }));
    const secondFetch = vi.fn(async () => Response.json({ client: "second" }));
    const globalFetch = vi.spyOn(globalThis, "fetch");

    const first = createHttpClient({
      baseURL: "https://first.example.test/",
      fetch: firstFetch,
    });
    const second = createHttpClient({
      baseURL: "https://second.example.test",
      fetch: secondFetch,
    });

    await expect(first.request<{ client: string }>("/v1/session")).resolves.toEqual({
      client: "first",
    });
    await expect(second.request<{ client: string }>("v1/session")).resolves.toEqual({
      client: "second",
    });

    expect(firstFetch).toHaveBeenCalledWith(
      "https://first.example.test/v1/session",
      expect.any(Object),
    );
    expect(secondFetch).toHaveBeenCalledWith(
      "https://second.example.test/v1/session",
      expect.any(Object),
    );
    expect(globalFetch).not.toHaveBeenCalled();
  });

  it("resolves headers for every request and lets request headers override defaults", async () => {
    const contexts: ResolveHeadersContext[] = [];
    const fetchImplementation = vi.fn<FetchImplementation>(
      async () => new Response(null, { status: 204 }),
    );
    const client = createHttpClient({
      baseURL: "https://api.example.test",
      fetch: fetchImplementation,
      resolveHeaders: async (context) => {
        contexts.push(context);
        return {
          Authorization: "Bearer dynamic-token",
          "X-Request-Source": "resolver",
        };
      },
    });

    await client.requestResponse("/v1/session", {
      method: "POST",
      headers: {
        "X-Request-Source": "caller",
      },
    });
    await client.requestResponse("/v1/session");

    expect(contexts).toEqual([
      {
        path: "/v1/session",
        url: "https://api.example.test/v1/session",
        method: "POST",
        signal: undefined,
      },
      {
        path: "/v1/session",
        url: "https://api.example.test/v1/session",
        method: "GET",
        signal: undefined,
      },
    ]);

    const firstInit = fetchImplementation.mock.calls[0]?.[1];
    const firstHeaders = new Headers(firstInit?.headers);
    expect(firstHeaders.get("authorization")).toBe("Bearer dynamic-token");
    expect(firstHeaders.get("x-request-source")).toBe("caller");
    expect(firstHeaders.get("content-type")).toBe("application/json");
  });

  it("forwards AbortSignal to both the resolver and fetch", async () => {
    const controller = new AbortController();
    const resolveHeaders = vi.fn(() => undefined);
    const fetchImplementation = vi.fn<FetchImplementation>(
      async () => new Response(null, { status: 204 }),
    );
    const client = createHttpClient({
      baseURL: "",
      fetch: fetchImplementation,
      resolveHeaders,
    });

    await client.requestResponse("/v1/session", { signal: controller.signal });

    expect(resolveHeaders).toHaveBeenCalledWith(
      expect.objectContaining({ signal: controller.signal }),
    );
    expect(fetchImplementation).toHaveBeenCalledWith(
      "/v1/session",
      expect.objectContaining({ signal: controller.signal }),
    );
  });

  it("uses the instance base url even when legacy baseURLs are supplied per request", async () => {
    const fetchImplementation = vi.fn(async () => new Response(null, { status: 204 }));
    const client = createHttpClient({
      baseURL: "https://api.example.test",
      fetch: fetchImplementation,
    });

    await client.requestResponse("/v1/session", {
      baseURLs: ["https://override.example.test"],
    });

    expect(fetchImplementation).toHaveBeenCalledWith(
      "https://api.example.test/v1/session",
      expect.any(Object),
    );
  });

  it("preserves absolute request urls", async () => {
    const fetchImplementation = vi.fn(async () => new Response(null, { status: 204 }));
    const client = createHttpClient({
      baseURL: "https://api.example.test",
      fetch: fetchImplementation,
    });

    await client.requestResponse("https://uploads.example.test/object");

    expect(fetchImplementation).toHaveBeenCalledWith(
      "https://uploads.example.test/object",
      expect.any(Object),
    );
  });

  it("does not set JSON content type for FormData bodies", async () => {
    const fetchImplementation = vi.fn<FetchImplementation>(
      async () => new Response(null, { status: 204 }),
    );
    const client = createHttpClient({
      baseURL: "https://api.example.test",
      fetch: fetchImplementation,
    });

    await client.requestResponse("/upload", {
      method: "POST",
      rawBody: new FormData(),
    });

    const init = fetchImplementation.mock.calls[0]?.[1];
    expect(new Headers(init?.headers).has("content-type")).toBe(false);
  });

  it("returns undefined for 204 and text for non-JSON responses", async () => {
    const fetchImplementation = vi
      .fn()
      .mockResolvedValueOnce(new Response(null, { status: 204 }))
      .mockResolvedValueOnce(
        new Response("ready", {
          status: 200,
          headers: { "content-type": "text/plain" },
        }),
      );
    const client = createHttpClient({
      baseURL: "https://api.example.test",
      fetch: fetchImplementation,
    });

    await expect(client.request<void>("/empty")).resolves.toBeUndefined();
    await expect(client.request<string>("/text")).resolves.toBe("ready");
  });
});
