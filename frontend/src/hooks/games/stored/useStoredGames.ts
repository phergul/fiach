import { useCallback, useEffect, useState } from 'react';

import { ScanAndSaveGames } from '@bindings/github.com/phergul/fiach/internal/services/gamesservice';
import type {
  SourceScanResult,
  StoredGame,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { useToast } from '@components/Common/Toast/Toast';
import { getErrorMessage } from '@utils';

import {
  fetchStoredGames,
  invalidateStoredGames,
  preloadStoredGames,
  storedGamesResource,
} from './storedGamesResource';

export { fetchStoredGames, invalidateStoredGames, preloadStoredGames, storedGamesResource };

let hasRunInitialScan = false;
let initialScanPromise: Promise<SourceScanResult> | null = null;

const ensureInitialScan = () => {
  if (hasRunInitialScan) {
    return Promise.resolve(null);
  }

  if (initialScanPromise === null) {
    initialScanPromise = ScanAndSaveGames().finally(() => {
      hasRunInitialScan = true;
      initialScanPromise = null;
    });
  }

  return initialScanPromise;
};

export const useStoredGames = () => {
  const { addErrorToast, addToast, removeToast } = useToast();
  const {
    data: games,
    isInitialLoading,
    isLoading,
    isRefreshing,
    loadError,
    refresh,
    setCached: setGames,
  } = storedGamesResource.useCached();
  const [isScanning, setIsScanning] = useState(false);
  const [scanError, setScanError] = useState<string | null>(null);

  const updateStoredGame = useCallback(
    (updatedGame: StoredGame) => {
      const cachedGames = storedGamesResource.getCached() ?? games;
      const nextGames = cachedGames.map((game) =>
        game.ID === updatedGame.ID ? updatedGame : game,
      );
      setGames(nextGames);
    },
    [games, setGames],
  );

  const loadGames = useCallback(() => refresh(), [refresh]);

  const refreshGames = useCallback(async () => {
    if (isScanning) {
      return;
    }

    setIsScanning(true);
    setScanError(null);

    const scanningToastID = addToast({
      duration: 0,
      message: 'Scanning game libraries...',
    });

    try {
      const result = await ScanAndSaveGames();
      await refresh();
      addToast({
        message: `Scan complete. ${result.Games.length} games found.`,
        tone: 'success',
      });
    } catch (error) {
      const message = getErrorMessage(error);
      setScanError(message);
      addErrorToast(error);
    } finally {
      removeToast(scanningToastID);
      setIsScanning(false);
    }
  }, [addErrorToast, addToast, isScanning, refresh, removeToast]);

  useEffect(() => {
    let isMounted = true;

    const runInitialScan = async () => {
      if (hasRunInitialScan) {
        return;
      }

      setIsScanning(true);
      setScanError(null);

      try {
        await ensureInitialScan();
        if (isMounted) {
          await refresh();
        }
      } catch (error) {
        const message = getErrorMessage(error);
        if (isMounted) {
          setScanError(message);
          const cachedGames = storedGamesResource.getCached() ?? [];
          if (cachedGames.length > 0) {
            addErrorToast(error);
          }
        }
      } finally {
        if (isMounted) {
          setIsScanning(false);
        }
      }
    };

    runInitialScan();
  }, [addErrorToast, refresh]);

  return {
    games,
    isInitialLoading,
    isLoading,
    isRefreshing,
    isScanning,
    loadError,
    refreshGames,
    retryLoadGames: loadGames,
    scanError,
    updateStoredGame,
  };
};
