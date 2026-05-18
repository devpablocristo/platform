// @vitest-environment node

import { describe, expect, it } from 'vitest';
import { resolveInitialTimeGridScrollTime, resolveInitialTimeGridViewport } from './time';

describe('resolveInitialTimeGridScrollTime', () => {
  it('keeps the default 07:30 margin when the earliest event starts after 08:00', () => {
    expect(
      resolveInitialTimeGridScrollTime({
        events: [
          { start: new Date(2099, 3, 5, 14, 0, 0) },
          { start: new Date(2099, 3, 5, 9, 30, 0) },
          { start: new Date(2099, 3, 6, 8, 0, 0) },
        ],
        rangeStart: new Date(2099, 3, 5, 0, 0, 0),
        rangeEnd: new Date(2099, 3, 6, 0, 0, 0),
      }),
    ).toBe('07:30:00');
  });

  it('falls back to the provided hour with a 30 minute margin when there are no visible events', () => {
    expect(
      resolveInitialTimeGridScrollTime({
        events: [{ start: 'invalid-date' }],
        rangeStart: new Date('2099-04-05T00:00:00Z'),
        rangeEnd: new Date('2099-04-06T00:00:00Z'),
        fallbackHour: 8,
      }),
    ).toBe('07:30:00');
  });
});

describe('resolveInitialTimeGridViewport', () => {
  it('opens the viewport 30 minutes before early events', () => {
    expect(
      resolveInitialTimeGridViewport({
        events: [{ start: new Date(2099, 3, 5, 7, 0, 0) }],
        rangeStart: new Date(2099, 3, 5, 0, 0, 0),
        rangeEnd: new Date(2099, 3, 6, 0, 0, 0),
        fallbackHour: 8,
      }),
    ).toEqual({
      scrollTime: '06:30:00',
      slotMinTime: '06:30:00',
    });
  });

  it('keeps the standard slot minimum when no earlier opening is needed', () => {
    expect(
      resolveInitialTimeGridViewport({
        events: [{ start: new Date(2099, 3, 5, 10, 0, 0) }],
        rangeStart: new Date(2099, 3, 5, 0, 0, 0),
        rangeEnd: new Date(2099, 3, 6, 0, 0, 0),
        fallbackHour: 8,
      }),
    ).toEqual({
      scrollTime: '07:30:00',
      slotMinTime: '07:00:00',
    });
  });
});
