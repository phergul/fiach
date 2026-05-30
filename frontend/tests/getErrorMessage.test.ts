import { describe, expect, test } from 'bun:test';

import { getErrorMessage, getRawErrorMessage } from '../src/utils/getErrorMessage';

describe('getErrorMessage', () => {
  test('formats a plain Error', () => {
    expect(getErrorMessage(new Error('disk full'))).toBe('Disk full.');
  });

  test('formats a string error', () => {
    expect(getErrorMessage('permission denied')).toBe('Permission denied.');
  });

  test('formats a Wails-style message object', () => {
    expect(getErrorMessage({ message: 'open logs window: application is not configured' }))
      .toBe('Open logs window failed: application is not configured.');
  });

  test('uses a RuntimeError cause instead of the runtime envelope', () => {
    const error = new Error('RuntimeError', {
      cause: 'pre validate import: prepare folder import source: source folder "/mods/Empty" is empty',
    });
    error.name = 'RuntimeError';

    expect(getErrorMessage(error)).toBe('Pre validate import failed: source folder "/mods/Empty" is empty.');
  });

  test('unwraps a serialized RuntimeError cause', () => {
    const error = new Error('{"message":"RuntimeError","cause":{"message":"pre validate import: source folder is empty"}}');
    error.name = 'RuntimeError';

    expect(getErrorMessage(error)).toBe('Pre validate import failed: source folder is empty.');
  });

  test('does not format a bare serialized RuntimeError envelope as a chain', () => {
    const error = new Error('{"message":"RuntimeError"}');
    error.name = 'RuntimeError';

    expect(getErrorMessage(error)).toBe('Something went wrong.');
  });

  test('summarizes a chained Go error', () => {
    expect(getErrorMessage('import mod: import mod source: read source folder: permission denied'))
      .toBe('Import mod failed: permission denied.');
  });

  test('deduplicates repeated chain segments', () => {
    expect(getErrorMessage('import mod: import mod: permission denied'))
      .toBe('Import mod failed: permission denied.');
  });

  test('does not split Windows drive paths as chain separators', () => {
    expect(getErrorMessage('open installer: C:\\Games\\ReShade.exe: access denied'))
      .toBe('Open installer failed: access denied.');
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
