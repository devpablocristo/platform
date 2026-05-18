import type { EventInput } from "@fullcalendar/core";

export type CalendarEventLike = {
  start?: EventInput["start"];
};

export type TimeGridViewport = {
  scrollTime: string;
  slotMinTime: string;
};

function toDate(value: EventInput["start"]): Date | null {
  if (value instanceof Date) {
    return value;
  }
  if (typeof value === "string" || typeof value === "number") {
    const parsed = new Date(value);
    return Number.isNaN(parsed.getTime()) ? null : parsed;
  }
  return null;
}

function toMinutesOfDay(date: Date): number {
  return date.getHours() * 60 + date.getMinutes();
}

function formatMinutesOfDay(totalMinutes: number): string {
  const normalized = Math.max(0, Math.min(totalMinutes, 24 * 60 - 1));
  const hours = String(Math.floor(normalized / 60)).padStart(2, "0");
  const minutes = String(normalized % 60).padStart(2, "0");
  return `${hours}:${minutes}:00`;
}

export function resolveInitialTimeGridViewport({
  events,
  rangeStart,
  rangeEnd,
  fallbackHour = 8,
  marginMinutes = 30,
}: {
  events: readonly CalendarEventLike[];
  rangeStart: Date;
  rangeEnd: Date;
  fallbackHour?: number;
  marginMinutes?: number;
}): TimeGridViewport {
  const earliestVisibleEvent = events
    .map((event) => toDate(event.start))
    .filter((date): date is Date => date != null && date >= rangeStart && date < rangeEnd)
    .sort((a, b) => a.getTime() - b.getTime())[0];

  const fallbackStartMinutes = Math.max(0, fallbackHour * 60 - marginMinutes);
  const defaultSlotMinMinutes = Math.max(0, (fallbackHour - 1) * 60);

  if (!earliestVisibleEvent) {
    return {
      scrollTime: formatMinutesOfDay(fallbackStartMinutes),
      slotMinTime: formatMinutesOfDay(defaultSlotMinMinutes),
    };
  }

  const earliestMinutes = toMinutesOfDay(earliestVisibleEvent);
  const earliestWithMarginMinutes = Math.max(0, earliestMinutes - marginMinutes);
  const scrollMinutes = Math.min(fallbackStartMinutes, earliestWithMarginMinutes);
  const slotMinMinutes = Math.min(defaultSlotMinMinutes, scrollMinutes);

  return {
    scrollTime: formatMinutesOfDay(scrollMinutes),
    slotMinTime: formatMinutesOfDay(slotMinMinutes),
  };
}

export function resolveInitialTimeGridScrollTime({
  events,
  rangeStart,
  rangeEnd,
  fallbackHour = 8,
}: {
  events: readonly CalendarEventLike[];
  rangeStart: Date;
  rangeEnd: Date;
  fallbackHour?: number;
}): string {
  return resolveInitialTimeGridViewport({
    events,
    rangeStart,
    rangeEnd,
    fallbackHour,
  }).scrollTime;
}
