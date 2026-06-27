import { describe, expect, it } from 'vitest';

import { formatAppliedAt, formatAppliedAtFromDate } from './formatAppliedAt';

describe('formatAppliedAt', () => {
  it('formats RFC3339 timestamps', () => {
    expect(formatAppliedAt('2026-06-27T12:00:00Z')).toMatch(/^Applied /);
  });

  it('returns unknown for invalid timestamps', () => {
    expect(formatAppliedAt('')).toBe('Applied time unknown');
    expect(formatAppliedAt('not-a-date')).toBe('Applied time unknown');
  });

  it('formats Date values', () => {
    expect(formatAppliedAtFromDate(new Date('2026-06-27T12:00:00Z'))).toMatch(/^Applied /);
  });

  it('formats RFC3339 strings from Wails time bindings', () => {
    expect(formatAppliedAtFromDate('2026-06-27T12:00:00Z')).toMatch(/^Applied /);
  });

  it('returns unknown for invalid timestamp values', () => {
    expect(formatAppliedAtFromDate(null)).toBe('Applied time unknown');
    expect(formatAppliedAtFromDate({})).toBe('Applied time unknown');
  });
});
