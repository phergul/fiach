import { createContext, ReactNode, useCallback, useContext, useMemo, useState } from 'react';

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
  const [releasePromise, setReleasePromise] = useState<Promise<OptiScalerRelease | null> | null>(null);

  const loadRelease = useCallback(() => {
    if (release !== null) {
      return Promise.resolve(release);
    }
    if (releasePromise !== null) {
      return releasePromise;
    }

    setIsReleaseLoading(true);
    setReleaseError(null);
    const request = GetOptiScalerReleaseStatus()
      .then((result) => {
        setRelease(result);
        return result;
      })
      .catch((error) => {
        setReleaseError(getErrorMessage(error));
        return null;
      })
      .finally(() => {
        setIsReleaseLoading(false);
      });
    setReleasePromise(request);
    return request;
  }, [release, releasePromise]);

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
