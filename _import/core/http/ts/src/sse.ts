import { requestResponse, type RequestOptions } from "./fetch";

export type JSONEventStreamOptions<TProgress, TResult> = RequestOptions & {
  progressEvent?: string;
  resultEvent?: string;
  errorEvent?: string;
  onProgress?: (event: TProgress) => void;
};

export async function requestJSONEventStream<TProgress = unknown, TResult = unknown>(
  path: string,
  options: JSONEventStreamOptions<TProgress, TResult> = {},
): Promise<TResult> {
  const response = await requestResponse(path, {
    ...options,
    headers: {
      Accept: "text/event-stream",
      ...(options.headers ?? {}),
    },
  });

  const reader = response.body?.getReader();
  if (!reader) {
    throw new Error("No response body");
  }

  const progressEvent = options.progressEvent ?? "progress";
  const resultEvent = options.resultEvent ?? "result";
  const errorEvent = options.errorEvent ?? "error";
  const decoder = new TextDecoder();
  let buffer = "";

  return new Promise<TResult>((resolve, reject) => {
    const pump = (): void => {
      reader.read().then(({ done, value }) => {
        if (done) {
          reject(new Error("Stream ended without result"));
          return;
        }

        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split("\n");
        buffer = lines.pop() ?? "";

        let currentEvent = "";
        for (const line of lines) {
          if (line.startsWith("event: ")) {
            currentEvent = line.slice(7).trim();
            continue;
          }
          if (!line.startsWith("data: ")) {
            continue;
          }

          const data = line.slice(6);
          try {
            const parsed = JSON.parse(data) as TProgress & TResult & { error?: string };
            if (currentEvent === progressEvent) {
              options.onProgress?.(parsed as TProgress);
            } else if (currentEvent === resultEvent) {
              resolve(parsed as TResult);
              return;
            } else if (currentEvent === errorEvent) {
              reject(new Error(parsed.error ?? "Stream failed"));
              return;
            }
          } catch {
            // ignore malformed frame
          }
        }

        pump();
      }).catch(reject);
    };

    pump();
  });
}
