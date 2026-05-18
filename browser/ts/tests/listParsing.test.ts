import { describe, expect, it } from "vitest";
import { parseListItemsFromResponse, parsePaginatedResponse } from "../src/crud/listParsing";

describe("parseListItemsFromResponse", () => {
  it("returns array as-is", () => {
    expect(parseListItemsFromResponse([{ id: "a" }])).toEqual([{ id: "a" }]);
  });

  it("reads items from envelope", () => {
    expect(parseListItemsFromResponse({ items: [{ id: "x" }] })).toEqual([{ id: "x" }]);
  });

  it("treats null items as empty", () => {
    const envelope: { items: null; total: number } = { items: null, total: 0 };
    expect(parseListItemsFromResponse(envelope)).toEqual([]);
  });

  it("treats missing items as empty", () => {
    expect(parseListItemsFromResponse({})).toEqual([]);
  });

  it("unwraps nested bff envelopes", () => {
    expect(
      parseListItemsFromResponse({
        data: {
          data: {
            items: [{ id: "nested" }],
          },
        },
      }),
    ).toEqual([{ id: "nested" }]);
  });

  it("returns empty when nested envelopes have null items", () => {
    expect(
      parseListItemsFromResponse({
        data: {
          data: {
            items: null,
          },
        },
      }),
    ).toEqual([]);
  });
});

describe("parsePaginatedResponse", () => {
  it("accepts camelCase pagination metadata", () => {
    expect(
      parsePaginatedResponse({
        items: [{ id: "a" }],
        hasMore: true,
        nextCursor: "cursor-1",
      }),
    ).toEqual({ items: [{ id: "a" }], hasMore: true, nextCursor: "cursor-1" });
  });

  it("accepts snake_case pagination metadata", () => {
    expect(
      parsePaginatedResponse({
        items: [{ id: "b" }],
        has_more: true,
        next_cursor: "cursor-2",
      }),
    ).toEqual({ items: [{ id: "b" }], hasMore: true, nextCursor: "cursor-2" });
  });

  it("keeps array legacy responses compatible", () => {
    expect(parsePaginatedResponse([{ id: "legacy" }])).toEqual({
      items: [{ id: "legacy" }],
      hasMore: false,
      nextCursor: "",
    });
  });

  it("reads nested pagination envelopes", () => {
    expect(
      parsePaginatedResponse({
        data: {
          items: [{ id: "nested-page" }],
          hasMore: true,
          nextCursor: "cursor-3",
        },
      }),
    ).toEqual({ items: [{ id: "nested-page" }], hasMore: true, nextCursor: "cursor-3" });
  });
});
