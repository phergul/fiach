// Use createSingletonCachedResource for global keyed reads (no cache key).
// Do not use for:
// - useGameMods / OptiScaler / ReShade: multi-field fetches and extra loading flags

import { useCallback, useEffect, useState } from 'react';

import { getErrorMessage } from '@utils';

import type {
  CachedHookResult,
  SingletonCachedResourceConfig,
  SingletonCachedResourceHandle,
} from './types';

export const createSingletonCachedResource = <TData>(
  config: SingletonCachedResourceConfig<TData>,
): SingletonCachedResourceHandle<TData> => {
  let cachedData: TData | null = null;
  let inFlight: Promise<TData> | null = null;

  const hasCachedEntry = (): boolean => {
    if (config.hasCachedEntry !== undefined) {
      return config.hasCachedEntry();
    }

    return cachedData !== null;
  };

  const getCached = (): TData | undefined => {
    if (!hasCachedEntry()) {
      return undefined;
    }

    return cachedData as TData;
  };

  const setCached = (data: TData) => {
    cachedData = data;
  };

  const invalidate = () => {
    cachedData = null;
    inFlight = null;
  };

  const clear = () => {
    invalidate();
  };

  const fetch = async (): Promise<TData> => {
    if (inFlight !== null) {
      return inFlight;
    }

    const pendingFetch = config
      .fetch()
      .then((data) => {
        cachedData = data;
        inFlight = null;
        return data;
      })
      .catch((error: unknown) => {
        inFlight = null;
        throw error;
      });

    inFlight = pendingFetch;
    return pendingFetch;
  };

  const preload = async (): Promise<TData> => {
    if (hasCachedEntry()) {
      return cachedData as TData;
    }

    return fetch();
  };

  const useCached = (): CachedHookResult<TData> => {
    const cachedValue = getCached();
    const [data, setData] = useState<TData>(cachedValue ?? config.emptyValue);
    const [isLoading, setIsLoading] = useState(!hasCachedEntry());
    const [loadError, setLoadError] = useState<string | null>(null);

    const refresh = useCallback(async () => {
      setIsLoading(true);
      setLoadError(null);

      try {
        const loadedData = await fetch();
        setData(loadedData);
        return loadedData;
      } catch (error) {
        const message = getErrorMessage(error);
        setLoadError(message);
        throw error;
      } finally {
        setIsLoading(false);
      }
    }, []);

    useEffect(() => {
      let isMounted = true;

      const loadInitial = async () => {
        const cachedInitial = getCached();
        setData(cachedInitial ?? config.emptyValue);
        setIsLoading(true);
        setLoadError(null);

        try {
          const loadedData = await fetch();
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
    }, []);

    const hasEntry = hasCachedEntry();
    const isInitialLoading = isLoading && !hasEntry;
    const isRefreshing = isLoading && hasEntry;

    return {
      data,
      isInitialLoading,
      isLoading,
      isRefreshing,
      loadError,
      refresh,
      setCached: (nextData: TData) => {
        setCached(nextData);
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
