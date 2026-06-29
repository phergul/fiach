import { renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { createKeyedCachedResource } from './createKeyedCachedResource';

type TestData = { id: number; value: string } | null;

const emptyValue = null;
const firstValue = { id: 1, value: 'first' };
const secondValue = { id: 1, value: 'second' };
const key = 7;

const createTestResource = (presence: 'hasKey' | 'hasValue' = 'hasValue') => {
  const fetch = vi.fn(async () => firstValue);

  const resource = createKeyedCachedResource<TestData, number>({
    emptyValue,
    fetch,
    presence,
  });

  return { fetch, resource };
};

describe('createKeyedCachedResource', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('fetch always calls the loader and updates the store', async () => {
    const { fetch, resource } = createTestResource();

    const loaded = await resource.fetch(key);

    expect(loaded).toEqual(firstValue);
    expect(fetch).toHaveBeenCalledOnce();
    expect(resource.getCached(key)).toEqual(firstValue);
  });

  it('preload returns cached data without calling the loader on a hit', async () => {
    const { fetch, resource } = createTestResource();

    await resource.fetch(key);
    fetch.mockClear();

    const preloaded = await resource.preload(key);

    expect(preloaded).toEqual(firstValue);
    expect(fetch).not.toHaveBeenCalled();
  });

  it('refresh always calls the loader even when cached', async () => {
    const { fetch, resource } = createTestResource();

    const { result } = renderHook(() => resource.useCached(key));

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

    const firstHook = renderHook(() => resource.useCached(key));

    await waitFor(() => {
      expect(firstHook.result.current.data).toEqual(firstValue);
    });
    firstHook.unmount();

    resource.invalidate(key);
    fetch.mockResolvedValue(secondValue);

    const secondHook = renderHook(() => resource.useCached(key));

    expect(secondHook.result.current.data).toBeNull();
    expect(secondHook.result.current.isInitialLoading).toBe(true);

    await waitFor(() => {
      expect(secondHook.result.current.data).toEqual(secondValue);
    });
  });

  it('supports hasKey presence when null is a valid cached value', async () => {
    const fetch = vi.fn(async () => null);
    const resource = createKeyedCachedResource<TestData, number>({
      emptyValue,
      fetch,
      presence: 'hasKey',
    });

    await resource.fetch(key);
    fetch.mockClear();

    const preloaded = await resource.preload(key);

    expect(preloaded).toBeNull();
    expect(fetch).not.toHaveBeenCalled();
  });

  it('supports hasValue presence when an absent key is a miss', async () => {
    const { fetch, resource } = createTestResource('hasValue');

    fetch.mockClear();

    await resource.preload(key);

    expect(fetch).toHaveBeenCalledOnce();
  });

  it('setCached patches the store without calling the loader', async () => {
    const { fetch, resource } = createTestResource();
    const patchedValue = { id: 1, value: 'patched' };

    resource.setCached(key, patchedValue);

    expect(resource.getCached(key)).toEqual(patchedValue);
    expect(fetch).not.toHaveBeenCalled();
  });

  it('useCached with a null key does not fetch', async () => {
    const { fetch, resource } = createTestResource();

    const { result } = renderHook(() => resource.useCached(null));

    expect(result.current.data).toBeNull();
    expect(result.current.isLoading).toBe(false);
    expect(result.current.loadError).toBeNull();
    expect(fetch).not.toHaveBeenCalled();
  });

  it('remount shows cached data immediately then refreshes', async () => {
    const { fetch, resource } = createTestResource();

    const firstHook = renderHook(() => resource.useCached(key));

    await waitFor(() => {
      expect(firstHook.result.current.data).toEqual(firstValue);
    });
    firstHook.unmount();

    fetch.mockResolvedValue(secondValue);

    const secondHook = renderHook(() => resource.useCached(key));

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

    await resource.fetch(key);
    resource.clear();
    fetch.mockClear();

    await resource.preload(key);

    expect(fetch).toHaveBeenCalledOnce();
  });

  it('deduplicates concurrent fetch calls for the same key', async () => {
    const { fetch, resource } = createTestResource();
    let resolveFetch: (value: TestData) => void = () => undefined;
    const pendingFetch = new Promise<TestData>((resolve) => {
      resolveFetch = resolve;
    });
    fetch.mockReturnValue(pendingFetch as ReturnType<typeof fetch>);

    const firstPromise = resource.fetch(key);
    const secondPromise = resource.fetch(key);

    resolveFetch(firstValue);

    await expect(Promise.all([firstPromise, secondPromise])).resolves.toEqual([
      firstValue,
      firstValue,
    ]);
    expect(fetch).toHaveBeenCalledOnce();
  });
});
