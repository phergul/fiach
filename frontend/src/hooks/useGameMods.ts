import { useCallback, useEffect, useState } from 'react';

import {
  GetGameManagedModStorageUsage,
  ListMods,
} from '@bindings/github.com/phergul/mod-manager/internal/services/modservice';
import type { Mod } from '@bindings/github.com/phergul/mod-manager/internal/storage/models';
import { getErrorMessage } from '@utils';

export const useGameMods = (gameID: number | null) => {
  const [mods, setMods] = useState<Mod[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isStorageUsageLoading, setIsStorageUsageLoading] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [storageUsedBytes, setStorageUsedBytes] = useState<number | null>(null);

  const loadStorageUsage = useCallback(async () => {
    if (gameID === null) {
      setStorageUsedBytes(null);
      setIsStorageUsageLoading(false);
      return null;
    }

    setIsStorageUsageLoading(true);

    try {
      const bytes = await GetGameManagedModStorageUsage(gameID);
      setStorageUsedBytes(bytes);
      return bytes;
    } catch {
      setStorageUsedBytes(null);
      return null;
    } finally {
      setIsStorageUsageLoading(false);
    }
  }, [gameID]);

  const loadMods = useCallback(async () => {
    if (gameID === null) {
      setMods([]);
      setIsLoading(false);
      setLoadError(null);
      setStorageUsedBytes(null);
      setIsStorageUsageLoading(false);
      return [];
    }

    setIsLoading(true);
    setLoadError(null);

    try {
      const [loadedMods] = await Promise.all([ListMods(gameID), loadStorageUsage()]);
      setMods(loadedMods);
      return loadedMods;
    } catch (error) {
      const message = getErrorMessage(error);
      setLoadError(message);
      throw error;
    } finally {
      setIsLoading(false);
    }
  }, [gameID, loadStorageUsage]);

  useEffect(() => {
    let isMounted = true;

    const loadInitialMods = async () => {
      if (gameID === null) {
        setMods([]);
        setLoadError(null);
        setIsLoading(false);
        setStorageUsedBytes(null);
        setIsStorageUsageLoading(false);
        return;
      }

      setIsLoading(true);
      setIsStorageUsageLoading(true);
      setLoadError(null);

      try {
        const storageUsagePromise = GetGameManagedModStorageUsage(gameID)
          .then((bytes) => {
            if (isMounted) {
              setStorageUsedBytes(bytes);
            }
          })
          .catch(() => {
            if (isMounted) {
              setStorageUsedBytes(null);
            }
          })
          .finally(() => {
            if (isMounted) {
              setIsStorageUsageLoading(false);
            }
          });

        const [loadedMods] = await Promise.all([ListMods(gameID), storageUsagePromise]);
        if (isMounted) {
          setMods(loadedMods);
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

    loadInitialMods();

    return () => {
      isMounted = false;
    };
  }, [gameID]);

  return {
    isLoading,
    isStorageUsageLoading,
    loadError,
    mods,
    refreshMods: loadMods,
    storageUsedBytes,
  };
};

export type UseGameModsResult = ReturnType<typeof useGameMods>;
