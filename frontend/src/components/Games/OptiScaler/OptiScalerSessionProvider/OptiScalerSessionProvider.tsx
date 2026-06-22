import { createContext, ReactNode, useCallback, useContext, useMemo, useRef, useState } from 'react';

import type { OptiScalerRelease } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { GetOptiScalerReleaseStatus } from '@bindings/github.com/phergul/fiach/internal/services/optiscalerservice';
import { getErrorMessage } from '@utils';

interface OptiScalerSessionContextValue {
  isReleaseLoading: boolean;
  loadRelease: () => Promise<OptiScalerRelease | null>;
  release: OptiScalerRelease | null;
  releaseError: string | null;
}

interface OptiScalerSessionProviderProps {
  children: ReactNode;
}

const OptiScalerSessionContext = createContext<OptiScalerSessionContextValue | null>(null);

export const OptiScalerSessionProvider = ({ children }: OptiScalerSessionProviderProps) => {
  const [release, setRelease] = useState<OptiScalerRelease | null>(null);
  const [releaseError, setReleaseError] = useState<string | null>(null);
  const [isReleaseLoading, setIsReleaseLoading] = useState(false);
  const releasePromiseRef = useRef<Promise<OptiScalerRelease | null> | null>(null);

  const loadRelease = useCallback(() => {
    if (release !== null) {
      return Promise.resolve(release);
    }
    if (releasePromiseRef.current !== null) {
      return releasePromiseRef.current;
    }

    setIsReleaseLoading(true);
    setReleaseError(null);
    let shouldCacheRequest = false;
    const request = GetOptiScalerReleaseStatus()
      .then((result) => {
        const releaseError = result.error ?? '';
        if (releaseError !== '') {
          setRelease(null);
          setReleaseError(releaseError);
          return null;
        }
        shouldCacheRequest = true;
        setRelease(result);
        return result;
      })
      .catch((error) => {
        setReleaseError(getErrorMessage(error));
        return null;
      })
      .finally(() => {
        setIsReleaseLoading(false);
        if (!shouldCacheRequest && releasePromiseRef.current === request) {
          releasePromiseRef.current = null;
        }
      });
    releasePromiseRef.current = request;
    return request;
  }, [release]);

  const value = useMemo(() => ({
    isReleaseLoading,
    loadRelease,
    release,
    releaseError,
  }), [isReleaseLoading, loadRelease, release, releaseError]);

  return (
    <OptiScalerSessionContext.Provider value={value}>
      {children}
    </OptiScalerSessionContext.Provider>
  );
};

export const useOptiScalerSession = () => {
  const context = useContext(OptiScalerSessionContext);
  if (context === null) {
    throw new Error('useOptiScalerSession must be used inside OptiScalerSessionProvider');
  }
  return context;
};
