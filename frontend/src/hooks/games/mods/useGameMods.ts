import { useCallback, useEffect, useState } from 'react';

import {
  GetGameManagedModStorageUsage,
  ListGameTags,
  ListMods,
} from '@bindings/github.com/phergul/fiach/internal/services/modservice';
import type { Mod, Tag } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { getErrorMessage } from '@utils';

interface CachedGameMods {
  gameTags: Tag[];
  mods: Mod[];
  storageUsedBytes: number | null;
}

const cachedGameModsByGameID = new Map<number, CachedGameMods>();

export const useGameMods = (gameID: number | null) => {
  const cachedGameMods = gameID === null ? undefined : cachedGameModsByGameID.get(gameID);
  const [mods, setMods] = useState<Mod[]>(cachedGameMods?.mods ?? []);
  const [gameTags, setGameTags] = useState<Tag[]>(cachedGameMods?.gameTags ?? []);
  const [isLoading, setIsLoading] = useState(cachedGameMods === undefined);
  const [isStorageUsageLoading, setIsStorageUsageLoading] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [storageUsedBytes, setStorageUsedBytes] = useState<number | null>(
    cachedGameMods?.storageUsedBytes ?? null,
  );

  const loadStorageUsage = useCallback(async () => {
    if (gameID === null) {
      setStorageUsedBytes(null);
      setIsStorageUsageLoading(false);
      return null;
    }

    setIsStorageUsageLoading(true);

    try {
      const bytes = await GetGameManagedModStorageUsage(gameID);
      const cachedMods = cachedGameModsByGameID.get(gameID);
      if (cachedMods !== undefined) {
        cachedGameModsByGameID.set(gameID, {
          ...cachedMods,
          storageUsedBytes: bytes,
        });
      }
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
      const [loadedMods, loadedTags, loadedStorageUsedBytes] = await Promise.all([
        ListMods(gameID),
        ListGameTags(gameID),
        loadStorageUsage(),
      ]);
      cachedGameModsByGameID.set(gameID, {
        gameTags: loadedTags,
        mods: loadedMods,
        storageUsedBytes: loadedStorageUsedBytes,
      });
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

      const cachedInitialMods = cachedGameModsByGameID.get(gameID);
      if (cachedInitialMods === undefined) {
        setMods([]);
        setGameTags([]);
        setStorageUsedBytes(null);
      } else {
        setMods(cachedInitialMods.mods);
        setGameTags(cachedInitialMods.gameTags);
        setStorageUsedBytes(cachedInitialMods.storageUsedBytes);
      }
      setIsLoading(true);
      setIsStorageUsageLoading(true);
      setLoadError(null);
      let loadedStorageUsedBytes: number | null = cachedInitialMods?.storageUsedBytes ?? null;

      try {
        const storageUsagePromise = GetGameManagedModStorageUsage(gameID)
          .then((bytes) => {
            loadedStorageUsedBytes = bytes;
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
        cachedGameModsByGameID.set(gameID, {
          gameTags: loadedTags,
          mods: loadedMods,
          storageUsedBytes: loadedStorageUsedBytes,
        });
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
    isInitialLoading: isLoading && mods.length === 0,
    isLoading,
    isRefreshing: isLoading && mods.length > 0,
    isStorageUsageLoading,
    loadError,
    mods,
    gameTags,
    refreshMods: loadMods,
    storageUsedBytes,
  };
};

export type UseGameModsResult = ReturnType<typeof useGameMods>;
