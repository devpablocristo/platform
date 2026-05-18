import { requestJSONEventStream } from "../src/sse";

function streamFromChunks(chunks: string[]): ReadableStream<Uint8Array> {
  const encoder = new TextEncoder();
  return new ReadableStream<Uint8Array>({
    start(controller) {
      chunks.forEach((chunk) => controller.enqueue(encoder.encode(chunk)));
      controller.close();
    },
  });
}

describe("requestJSONEventStream", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("emits progress and resolves result", async () => {
    const progress = vi.fn();
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(
        streamFromChunks([
          'event: progress\ndata: {"step":1}\n',
          'event: result\ndata: {"ok":true}\n',
        ]),
        {
          status: 200,
          headers: { "content-type": "text/event-stream" },
        },
      ),
    );

    await expect(
      requestJSONEventStream<{ step: number }, { ok: boolean }>("/stream", {
        onProgress: progress,
      }),
    ).resolves.toEqual({ ok: true });

    expect(progress).toHaveBeenCalledWith({ step: 1 });
  });
});
