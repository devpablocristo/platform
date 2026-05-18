import {
  closestCorners,
  pointerWithin,
  rectIntersection,
  type CollisionDetection,
} from "@dnd-kit/core";

/**
 * Collision detection para kanban.
 * Devuelve columnas y tarjetas — el handler resuelve la posición.
 */
export const kanbanCollisionDetection: CollisionDetection = (args) => {
  const activeId = args.active?.id;

  const pointer = pointerWithin(args).filter((c) => activeId == null || c.id !== activeId);
  if (pointer.length > 0) return pointer;

  const rect = rectIntersection(args).filter((c) => activeId == null || c.id !== activeId);
  if (rect.length > 0) return rect;

  return closestCorners(args).filter((c) => activeId == null || c.id !== activeId);
};
