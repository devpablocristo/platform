import { HttpError, request, requestResponse } from "../src/fetch";

describe("core-http fetch", () => {
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
});
