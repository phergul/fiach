import { useEffect, useState } from 'react';

import { DetectGameReShade } from '@bindings/github.com/phergul/fiach/internal/services/reshadeservice';
import type { ReShadeDetectionResult } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { getErrorMessage } from '@utils';

export const useGameReShadeDetection = (gameID: number | null) => {
  const [result, setResult] = useState<ReShadeDetectionResult | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);

  useEffect(() => {
    let isMounted = true;

    const detectReShade = async () => {
      if (gameID === null) {
        setResult(null);
        setIsLoading(false);
        setLoadError(null);
        return;
      }

      setIsLoading(true);
      setLoadError(null);

      try {
        const detectionResult = await DetectGameReShade(gameID);
        if (isMounted) {
          setResult(detectionResult);
        }
      } catch (error) {
        if (isMounted) {
          setResult(null);
          setLoadError(getErrorMessage(error));
        }
      } finally {
        if (isMounted) {
          setIsLoading(false);
        }
      }
    };

    detectReShade();

    return () => {
      isMounted = false;
    };
  }, [gameID]);

  return {
    isLoading,
    loadError,
    result,
  };
};

export type UseGameReShadeDetectionResult = ReturnType<typeof useGameReShadeDetection>;
