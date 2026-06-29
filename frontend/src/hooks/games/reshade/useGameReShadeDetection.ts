import { useCallback, useEffect, useState } from 'react';

import { DetectGameReShade } from '@bindings/github.com/phergul/fiach/internal/services/reshadeservice';
import type { ReShadeDetectionResult } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { getErrorMessage } from '@utils';

import { useRuntime } from '../../runtime/useRuntime';

export const useGameReShadeDetection = (gameID: number | null) => {
  const { isWindows } = useRuntime();
  const [result, setResult] = useState<ReShadeDetectionResult | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    if (gameID === null || !isWindows) {
      setResult(null);
      setIsLoading(false);
      setLoadError(null);
      return;
    }

    setIsLoading(true);
    setLoadError(null);

    try {
      setResult(await DetectGameReShade(gameID));
    } catch (error) {
      setResult(null);
      setLoadError(getErrorMessage(error));
      throw error;
    } finally {
      setIsLoading(false);
    }
  }, [gameID, isWindows]);

  useEffect(() => {
    void refresh().catch(() => undefined);
  }, [refresh]);

  return {
    isLoading,
    loadError,
    refresh,
    result,
  };
};

export type UseGameReShadeDetectionResult = ReturnType<typeof useGameReShadeDetection>;
