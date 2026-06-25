import { describe, expect, test } from 'bun:test';

import { getErrorMessage, getRawErrorMessage } from '../src/utils/getErrorMessage';

const genericFallback = 'Something went wrong. Check the logs for details.';

describe('getErrorMessage', () => {
  test('formats a plain Error', () => {
    expect(getErrorMessage(new Error('disk full'))).toBe('Disk full.');
  });

  test('formats a string error', () => {
    expect(getErrorMessage('permission denied')).toBe('Permission denied.');
  });

  test('passes through a backend friendly message', () => {
    expect(getErrorMessage('A profile with this name already exists for this game.')).toBe(
      'A profile with this name already exists for this game.',
    );
  });

  test('uses generic fallback for chained Go errors', () => {
    expect(getErrorMessage('import mod: import mod source: read source folder: permission denied')).toBe(
      genericFallback,
    );
  });

  test('uses generic fallback for technical sqlite errors', () => {
    expect(
      getErrorMessage(
        'create profile: insert profile row: constraint failed: UNIQUE constraint failed: profiles.game_id, profiles.name (2067)',
      ),
    ).toBe(genericFallback);
  });

  test('uses generic fallback for Wails-style chained message objects', () => {
    expect(getErrorMessage({ message: 'open logs window: application is not configured' })).toBe(
      genericFallback,
    );
  });

  test('uses generic fallback for a RuntimeError cause chain', () => {
    const error = new Error('RuntimeError', {
      cause: 'pre validate import: prepare folder import source: source folder "/mods/Empty" is empty',
    });
    error.name = 'RuntimeError';

    expect(getErrorMessage(error)).toBe(genericFallback);
  });

  test('unwraps a serialized RuntimeError cause', () => {
    const error = new Error('{"message":"RuntimeError","cause":{"message":"pre validate import: source folder is empty"}}');
    error.name = 'RuntimeError';

    expect(getErrorMessage(error)).toBe(genericFallback);
  });

  test('does not format a bare serialized RuntimeError envelope as a chain', () => {
    const error = new Error('{"message":"RuntimeError"}');
    error.name = 'RuntimeError';

    expect(getErrorMessage(error)).toBe('Something went wrong.');
  });

  test('falls back for unknown objects', () => {
    expect(getErrorMessage({ detail: 'hidden' })).toBe('Something went wrong.');
  });
});

describe('getRawErrorMessage', () => {
  test('preserves the raw chain for diagnostics', () => {
    expect(getRawErrorMessage(new Error('import mod: read source folder: permission denied')))
      .toBe('import mod: read source folder: permission denied');
  });
});
