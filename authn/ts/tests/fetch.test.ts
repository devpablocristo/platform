import { HttpError, registerTokenProvider, request, requestResponse, resetTokenProvider } from "../src/http/fetch";

describe("fetch auth helpers", () => {
  beforeEach(() => {
    resetTokenProvider();
    vi.restoreAllMocks();
  });

  it("sends bearer token when provider exists", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ ok: true }), {
        status: 200,
        headers: { "content-type": "application/json" },
      }),
    );
    vi.stubGlobal("fetch", fetchMock);
    registerTokenProvider(async () => "token-123");

    await request("/v1/test");

    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining("/v1/test"),
      expect.objectContaining({
        headers: expect.objectContaining({
          Authorization: "Bearer token-123",
        }),
      }),
    );
  });

  it("throws HttpError on non-2xx responses", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response("denied", {
          status: 401,
          headers: { "content-type": "text/plain" },
        }),
      ),
    );

    await expect(requestResponse("/v1/test")).rejects.toBeInstanceOf(HttpError);
  });
});
