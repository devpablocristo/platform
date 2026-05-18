type ListEnvelope<T> = {
  items?: T[] | null | unknown;
  data?: unknown;
  hasMore?: boolean;
  has_more?: boolean;
  nextCursor?: string;
  next_cursor?: string;
};

export type PaginatedList<T> = {
  items: T[];
  hasMore: boolean;
  nextCursor: string;
};

/**
 * Normaliza respuestas de listado tipo arreglo, `{ items }` o envelopes BFF anidados con `data`.
 * Go suele serializar slices nil como `null`; cualquier rama sin arreglo termina en lista vacía.
 */
export function parseListItemsFromResponse<T>(input: unknown): T[] {
  const queue: unknown[] = [input];
  const seen = new Set<unknown>();

  while (queue.length > 0) {
    const current = queue.shift();
    if (Array.isArray(current)) {
      return current as T[];
    }
    if (current == null || typeof current !== "object" || seen.has(current)) {
      continue;
    }

    seen.add(current);
    const envelope = current as ListEnvelope<T>;

    if (Array.isArray(envelope.items)) {
      return envelope.items;
    }

    if ("data" in envelope) {
      queue.push(envelope.data);
    }
    if ("items" in envelope) {
      queue.push(envelope.items);
    }
  }

  return [];
}

/**
 * Como `parseListItemsFromResponse` pero conserva `has_more` y `next_cursor` del backend.
 */
export function parsePaginatedResponse<T>(input: unknown): PaginatedList<T> {
  const items = parseListItemsFromResponse<T>(input);
  let hasMore = false;
  let nextCursor = "";
  const envelope = findEnvelope(input);
  if (envelope) {
    hasMore = Boolean(envelope.hasMore ?? envelope.has_more);
    nextCursor = String(envelope.nextCursor ?? envelope.next_cursor ?? "");
  } else if (input != null && typeof input === "object" && !Array.isArray(input)) {
    const envelope = input as ListEnvelope<T>;
    hasMore = Boolean(envelope.hasMore ?? envelope.has_more);
    nextCursor = String(envelope.nextCursor ?? envelope.next_cursor ?? "");
  }
  return { items, hasMore, nextCursor };
}

function findEnvelope<T>(input: unknown): ListEnvelope<T> | null {
  const queue: unknown[] = [input];
  const seen = new Set<unknown>();

  while (queue.length > 0) {
    const current = queue.shift();
    if (current == null || typeof current !== "object" || Array.isArray(current) || seen.has(current)) {
      continue;
    }
    seen.add(current);
    const envelope = current as ListEnvelope<T>;
    if (
      "hasMore" in envelope ||
      "has_more" in envelope ||
      "nextCursor" in envelope ||
      "next_cursor" in envelope
    ) {
      return envelope;
    }
    if ("data" in envelope) queue.push(envelope.data);
    if ("items" in envelope) queue.push(envelope.items);
  }

  return null;
}
