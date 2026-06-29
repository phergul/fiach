import { renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { createSingletonCachedResource } from './createSingletonCachedResource';

const firstValue = [{ id: 1, name: 'first' }];
const secondValue = [{ id: 1, name: 'second' }];
const emptyValue: { id: number; name: string }[] = [];

const createTestResource = () => {
  const fetch = vi.fn(async () => firstValue);

  const resource = createSingletonCachedResource({
    emptyValue,
    fetch,
  });

  return { fetch, resource };
};

describe('createSingletonCachedResource', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('fetch always calls the loader and updates the store', async () => {
    const { fetch, resource } = createTestResource();

    const loaded = await resource.fetch();

    expect(loaded).toEqual(firstValue);
    expect(fetch).toHaveBeenCalledOnce();
    expect(resource.getCached()).toEqual(firstValue);
  });

  it('preload returns cached data without calling the loader on a hit', async () => {
    const { fetch, resource } = createTestResource();

    await resource.fetch();
    fetch.mockClear();

    const preloaded = await resource.preload();

    expect(preloaded).toEqual(firstValue);
    expect(fetch).not.toHaveBeenCalled();
  });

  it('refresh always calls the loader even when cached', async () => {
    const { fetch, resource } = createTestResource();

    const { result } = renderHook(() => resource.useCached());

    await waitFor(() => {
      expect(result.current.data).toEqual(firstValue);
    });
    expect(fetch).toHaveBeenCalledOnce();

    fetch.mockResolvedValue(secondValue);

    await result.current.refresh();

    await waitFor(() => {
      expect(result.current.data).toEqual(secondValue);
    });
    expect(fetch).toHaveBeenCalledTimes(2);
  });

  it('invalidate clears the entry so the next mount is initial loading', async () => {
    const { fetch, resource } = createTestResource();

    const firstHook = renderHook(() => resource.useCached());

    await waitFor(() => {
      expect(firstHook.result.current.data).toEqual(firstValue);
    });
    firstHook.unmount();

    resource.invalidate();
    fetch.mockResolvedValue(secondValue);

    const secondHook = renderHook(() => resource.useCached());

    expect(secondHook.result.current.data).toEqual([]);
    expect(secondHook.result.current.isInitialLoading).toBe(true);

    await waitFor(() => {
      expect(secondHook.result.current.data).toEqual(secondValue);
    });
  });

  it('setCached patches the store without calling the loader', async () => {
    const { fetch, resource } = createTestResource();
    const patchedValue = [{ id: 2, name: 'patched' }];

    resource.setCached(patchedValue);

    expect(resource.getCached()).toEqual(patchedValue);
    expect(fetch).not.toHaveBeenCalled();
  });

  it('remount shows cached data immediately then refreshes', async () => {
    const { fetch, resource } = createTestResource();

    const firstHook = renderHook(() => resource.useCached());

    await waitFor(() => {
      expect(firstHook.result.current.data).toEqual(firstValue);
    });
    firstHook.unmount();

    fetch.mockResolvedValue(secondValue);

    const secondHook = renderHook(() => resource.useCached());

    expect(secondHook.result.current.data).toEqual(firstValue);
    expect(secondHook.result.current.isInitialLoading).toBe(false);
    expect(secondHook.result.current.isRefreshing).toBe(true);

    await waitFor(() => {
      expect(secondHook.result.current.data).toEqual(secondValue);
    });
    expect(fetch).toHaveBeenCalledTimes(2);
  });

  it('clear resets the entire store', async () => {
    const { fetch, resource } = createTestResource();

    await resource.fetch();
    resource.clear();
    fetch.mockClear();

    await resource.preload();

    expect(fetch).toHaveBeenCalledOnce();
  });
});
