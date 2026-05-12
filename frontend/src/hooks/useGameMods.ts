import { useCallback, useEffect, useState } from 'react';

import { ListMods } from '@bindings/github.com/phergul/mod-manager/internal/services/modservice';
import type { Mod } from '@bindings/github.com/phergul/mod-manager/internal/storage/models';
import { getErrorMessage } from '@utils';

export const useGameMods = (gameID: number | null) => {
  const [mods, setMods] = useState<Mod[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);

  const loadMods = useCallback(async () => {
    if (gameID === null) {
      setMods([]);
      setIsLoading(false);
      setLoadError(null);
      return [];
    }

    setIsLoading(true);
    setLoadError(null);

    try {
      const loadedMods = await ListMods(gameID);
      setMods(loadedMods);
      return loadedMods;
    } catch (error) {
      const message = getErrorMessage(error);
      setLoadError(message);
      throw error;
    } finally {
      setIsLoading(false);
    }
  }, [gameID]);

  useEffect(() => {
    let isMounted = true;

    const loadInitialMods = async () => {
      if (gameID === null) {
        setMods([]);
        setLoadError(null);
        setIsLoading(false);
        return;
      }

      setIsLoading(true);
      setLoadError(null);

      try {
        const loadedMods = await ListMods(gameID);
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
    loadError,
    mods,
    refreshMods: loadMods,
  };
};

export type UseGameModsResult = ReturnType<typeof useGameMods>;
