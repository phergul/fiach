// Use createKeyedCachedResource for simple keyed reads (fetch/preload/invalidate/remount).
// Do not use for:
// - useGameMods / OptiScaler / ReShade: multi-field fetches and extra loading flags

import { useCallback, useEffect, useState } from 'react';

import { getErrorMessage } from '@utils';

import type {
  CachedHookResult,
  CachedKey,
  CachedResourceHandle,
  KeyedCachedResourceConfig,
} from './types';

export const createKeyedCachedResource = <TData, TKey extends CachedKey>(
  config: KeyedCachedResourceConfig<TData, TKey>,
): CachedResourceHandle<TData, TKey> => {
  const presence = config.presence ?? 'hasValue';
  const store = new Map<TKey, TData>();
  const inFlightByKey = new Map<TKey, Promise<TData>>();
  const isEmpty = config.isEmpty ?? ((data: TData) => data === config.emptyValue);

  const hasCachedEntry = (key: TKey): boolean => {
    if (config.hasCachedEntry !== undefined) {
      return config.hasCachedEntry(key);
    }

    return presence === 'hasKey' ? store.has(key) : store.get(key) !== undefined;
  };

  const getCached = (key: TKey): TData | undefined => {
    if (!hasCachedEntry(key)) {
      return undefined;
    }

    return store.get(key);
  };

  const setCached = (key: TKey, data: TData) => {
    store.set(key, data);
  };

  const invalidate = (key: TKey) => {
    store.delete(key);
    inFlightByKey.delete(key);
  };

  const clear = () => {
    store.clear();
    inFlightByKey.clear();
  };

  const fetch = async (key: TKey): Promise<TData> => {
    const inFlight = inFlightByKey.get(key);
    if (inFlight !== undefined) {
      return inFlight;
    }

    const pendingFetch = config
      .fetch(key)
      .then((data) => {
        store.set(key, data);
        inFlightByKey.delete(key);
        return data;
      })
      .catch((error: unknown) => {
        inFlightByKey.delete(key);
        throw error;
      });

    inFlightByKey.set(key, pendingFetch);
    return pendingFetch;
  };

  const preload = async (key: TKey): Promise<TData> => {
    if (hasCachedEntry(key)) {
      return store.get(key) as TData;
    }

    return fetch(key);
  };

  const useCached = (key: TKey | null): CachedHookResult<TData> => {
    const cachedValue = key === null ? undefined : getCached(key);
    const [data, setData] = useState<TData>(
      key === null ? config.emptyValue : (cachedValue ?? config.emptyValue),
    );
    const [isLoading, setIsLoading] = useState(key !== null && !hasCachedEntry(key));
    const [loadError, setLoadError] = useState<string | null>(null);

    const refresh = useCallback(async () => {
      if (key === null) {
        setData(config.emptyValue);
        setIsLoading(false);
        setLoadError(null);
        return null;
      }

      setIsLoading(true);
      setLoadError(null);

      try {
        const loadedData = await fetch(key);
        setData(loadedData);
        return loadedData;
      } catch (error) {
        const message = getErrorMessage(error);
        setLoadError(message);
        throw error;
      } finally {
        setIsLoading(false);
      }
    }, [key]);

    useEffect(() => {
      let isMounted = true;

      const loadInitial = async () => {
        if (key === null) {
          setData(config.emptyValue);
          setIsLoading(false);
          setLoadError(null);
          return;
        }

        const cachedInitial = getCached(key);
        setData(cachedInitial ?? config.emptyValue);
        setIsLoading(true);
        setLoadError(null);

        try {
          const loadedData = await fetch(key);
          if (isMounted) {
            setData(loadedData);
          }
        } catch (error) {
          if (isMounted) {
            setLoadError(getErrorMessage(error));
          }
        } finally {
          if (isMounted) {
            setIsLoading(false);
          }
        }
      };

      loadInitial();

      return () => {
        isMounted = false;
      };
    }, [key]);

    const hasEntry = key !== null && hasCachedEntry(key);
    const isInitialLoading =
      presence === 'hasKey' ? isLoading && !hasEntry : isLoading && isEmpty(data);
    const isRefreshing =
      presence === 'hasKey' ? isLoading && hasEntry : isLoading && !isEmpty(data);

    return {
      data,
      isInitialLoading,
      isLoading,
      isRefreshing,
      loadError,
      refresh,
      setCached: (nextData: TData) => {
        if (key !== null) {
          setCached(key, nextData);
        }
        setData(nextData);
      },
    };
  };

  return {
    clear,
    fetch,
    getCached,
    invalidate,
    preload,
    setCached,
    useCached,
  };
};
