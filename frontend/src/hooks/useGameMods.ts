import { useCallback, useEffect, useState } from 'react';

import {
  GetGameManagedModStorageUsage,
  ListGameTags,
  ListMods,
} from '@bindings/github.com/phergul/fiach/internal/services/modservice';
import type { Mod, Tag } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { getErrorMessage } from '@utils';

export const useGameMods = (gameID: number | null) => {
  const [mods, setMods] = useState<Mod[]>([]);
  const [gameTags, setGameTags] = useState<Tag[]>([]);
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
      setGameTags([]);
      setIsLoading(false);
      setLoadError(null);
      setStorageUsedBytes(null);
      setIsStorageUsageLoading(false);
      return [];
    }

    setIsLoading(true);
    setLoadError(null);

    try {
      const [loadedMods, loadedTags] = await Promise.all([
        ListMods(gameID),
        ListGameTags(gameID),
        loadStorageUsage(),
      ]);
      setMods(loadedMods);
      setGameTags(loadedTags);
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
        setGameTags([]);
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

        const [loadedMods, loadedTags] = await Promise.all([
          ListMods(gameID),
          ListGameTags(gameID),
          storageUsagePromise,
        ]);
        if (isMounted) {
          setMods(loadedMods);
          setGameTags(loadedTags);
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
    gameTags,
    refreshMods: loadMods,
    storageUsedBytes,
  };
};

export type UseGameModsResult = ReturnType<typeof useGameMods>;
