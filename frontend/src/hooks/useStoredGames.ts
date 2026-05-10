import { useCallback, useEffect, useState } from 'react';

import { useToast } from '../components/Common/Toast/Toast';
import { getErrorMessage } from '../utils/getErrorMessage';
import {
  GetStoredGames,
  ScanAndSaveSteamGames,
} from '../../bindings/github.com/phergul/mod-manager/internal/services/steamservice';
import type {
  SteamScanResult,
  StoredGame,
} from '../../bindings/github.com/phergul/mod-manager/internal/storage/models';

let hasRunInitialScan = false;
let initialScanPromise: Promise<SteamScanResult> | null = null;

const ensureInitialScan = () => {
  if (hasRunInitialScan) {
    return Promise.resolve(null);
  }

  if (initialScanPromise === null) {
    initialScanPromise = ScanAndSaveSteamGames().finally(() => {
      hasRunInitialScan = true;
      initialScanPromise = null;
    });
  }

  return initialScanPromise;
};

export const useStoredGames = () => {
  const { addToast, removeToast } = useToast();
  const [games, setGames] = useState<StoredGame[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isScanning, setIsScanning] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [scanError, setScanError] = useState<string | null>(null);

  const loadGames = useCallback(async () => {
    setIsLoading(true);
    setLoadError(null);

    try {
      const storedGames = await GetStoredGames();
      setGames(storedGames);
      return storedGames;
    } catch (error) {
      const message = getErrorMessage(error);
      setLoadError(message);
      throw error;
    } finally {
      setIsLoading(false);
    }
  }, []);

  const refreshGames = useCallback(async () => {
    if (isScanning) {
      return;
    }

    setIsScanning(true);
    setScanError(null);

    const scanningToastID = addToast({
      duration: 0,
      message: 'Scanning Steam library...',
    });

    try {
      const result = await ScanAndSaveSteamGames();
      await loadGames();
      addToast({
        message: `Scan complete. ${result.Games.length} games found.`,
        tone: 'success',
      });
    } catch (error) {
      const message = getErrorMessage(error);
      setScanError(message);
      addToast({
        message,
        tone: 'error',
      });
    } finally {
      removeToast(scanningToastID);
      setIsScanning(false);
    }
  }, [addToast, isScanning, loadGames, removeToast]);

  useEffect(() => {
    let isMounted = true;

    const loadAndScan = async () => {
      let cachedGames: StoredGame[] = [];

      try {
        cachedGames = await loadGames();
      } catch {
        cachedGames = [];
      }

      if (hasRunInitialScan) {
        return;
      }

      setIsScanning(true);
      setScanError(null);

      try {
        await ensureInitialScan();
        if (isMounted) {
          await loadGames();
        }
      } catch (error) {
        const message = getErrorMessage(error);
        if (isMounted) {
          setScanError(message);
          if (cachedGames.length > 0) {
            addToast({
              message,
              tone: 'error',
            });
          }
        }
      } finally {
        if (isMounted) {
          setIsScanning(false);
        }
      }
    };

    loadAndScan();

    return () => {
      isMounted = false;
    };
  }, [addToast, loadGames]);

  return {
    games,
    isLoading,
    isScanning,
    loadError,
    refreshGames,
    retryLoadGames: loadGames,
    scanError,
  };
};
