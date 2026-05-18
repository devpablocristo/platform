import { describe, expect, it, vi } from "vitest";

const pointerWithinMock = vi.hoisted(() => vi.fn());
const rectIntersectionMock = vi.hoisted(() => vi.fn());
const closestCornersMock = vi.hoisted(() => vi.fn());

vi.mock("@dnd-kit/core", () => ({
  pointerWithin: (...args: unknown[]) => pointerWithinMock(...args),
  rectIntersection: (...args: unknown[]) => rectIntersectionMock(...args),
  closestCorners: (...args: unknown[]) => closestCornersMock(...args),
}));

import { kanbanCollisionDetection } from "../src/collision";

describe("kanbanCollisionDetection", () => {
  it("prefers pointer collisions before any fallback", () => {
    const pointer = [{ id: "col-todo" }];
    pointerWithinMock.mockReturnValue(pointer);
    rectIntersectionMock.mockReturnValue([{ id: "col-doing" }]);
    closestCornersMock.mockReturnValue([{ id: "col-done" }]);

    expect(kanbanCollisionDetection({} as never)).toStrictEqual(pointer);
    expect(rectIntersectionMock).not.toHaveBeenCalled();
    expect(closestCornersMock).not.toHaveBeenCalled();
  });

  it("falls back to rect intersection and then closest corners", () => {
    const rect = [{ id: "col-doing" }];
    const corners = [{ id: "col-done" }];

    pointerWithinMock.mockReturnValue([]);
    rectIntersectionMock.mockReturnValue(rect);
    closestCornersMock.mockReturnValue(corners);
    expect(kanbanCollisionDetection({} as never)).toStrictEqual(rect);

    rectIntersectionMock.mockReturnValue([]);
    expect(kanbanCollisionDetection({} as never)).toStrictEqual(corners);
  });
});
