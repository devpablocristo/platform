import { describe, expect, it } from 'vitest';
import {
  formatSchedulingClock,
  formatSchedulingCompactClock,
  formatSchedulingDateOnly,
  formatSchedulingDateTime,
  formatSchedulingWeekdayNarrow,
  parseSchedulingDateOnlyLatinInput,
  resolveSchedulingCopyLocale,
} from './locale';

describe('scheduling locale helpers', () => {
  it('resolves regional locales to the supported copy presets', () => {
    expect(resolveSchedulingCopyLocale('es-AR')).toBe('es');
    expect(resolveSchedulingCopyLocale('en-US')).toBe('en');
    expect(resolveSchedulingCopyLocale(undefined)).toBe('en');
  });

  it('formats date and time values with the provided locale', () => {
    expect(formatSchedulingClock('2099-12-01T10:00:00Z', 'es-AR')).not.toBe(formatSchedulingClock('2099-12-01T10:00:00Z', 'en-US'));
    expect(formatSchedulingCompactClock('2099-12-01T10:00:00Z', 'es-AR')).toMatch(/\d{2}:\d{2}/);
    expect(formatSchedulingDateTime('2099-12-01T10:00:00Z', 'es-AR')).toMatch(/\d{1,2}\/\d{1,2}\/\d{4}/);
    expect(formatSchedulingDateTime('2099-12-01T10:00:00Z', 'en-US')).toMatch(/dec/i);
    expect(formatSchedulingDateTime('invalid-date', 'es-AR')).toBe('invalid-date');
  });

  it('formats YYYY-MM-DD for display without exposing raw ISO date', () => {
    expect(formatSchedulingDateOnly('2099-04-08', 'es-AR')).toMatch(/\d{1,2}\/\d{1,2}\/\d{4}/);
    expect(formatSchedulingDateOnly('2099-04-08', 'es-AR')).not.toContain('2099-04');
    expect(formatSchedulingDateOnly('not-a-date', 'es-AR')).toBe('not-a-date');
  });

  it('parses Latin day-first and ISO paste to YYYY-MM-DD', () => {
    expect(parseSchedulingDateOnlyLatinInput('04/09/2026')).toBe('2026-09-04');
    expect(parseSchedulingDateOnlyLatinInput('4.9.2026')).toBe('2026-09-04');
    expect(parseSchedulingDateOnlyLatinInput('2026-09-04')).toBe('2026-09-04');
    expect(parseSchedulingDateOnlyLatinInput('32/01/2026')).toBeNull();
    expect(parseSchedulingDateOnlyLatinInput('04/13/2026')).toBeNull();
    expect(parseSchedulingDateOnlyLatinInput('')).toBeNull();
  });

  it('formats weekday picks with the provided locale', () => {
    expect(formatSchedulingWeekdayNarrow(1, 'es-AR')).not.toBe('');
    expect(formatSchedulingWeekdayNarrow(1, 'en-US')).not.toBe('');
  });
});
