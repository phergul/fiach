import { useCallback, useEffect, useState } from 'react';

import { GetDeploymentFileInspection } from '@bindings/github.com/phergul/fiach/internal/services/deploymentreviewservice';
import type { DeploymentFileInspection } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { getErrorMessage } from '@utils';

export const useDeploymentFileInspection = (previewHash: string, selectedPath: string | null) => {
  const [inspection, setInspection] = useState<DeploymentFileInspection | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);

  const loadInspection = useCallback(async () => {
    if (previewHash === '' || selectedPath === null) {
      setInspection(null);
      setIsLoading(false);
      setLoadError(null);
      return null;
    }

    setIsLoading(true);
    setLoadError(null);

    try {
      const loadedInspection = await GetDeploymentFileInspection(previewHash, selectedPath);
      setInspection(loadedInspection);
      return loadedInspection;
    } catch (error) {
      const message = getErrorMessage(error);
      setLoadError(message);
      setInspection(null);
      throw error;
    } finally {
      setIsLoading(false);
    }
  }, [previewHash, selectedPath]);

  useEffect(() => {
    let isMounted = true;

    const loadInitialInspection = async () => {
      if (previewHash === '' || selectedPath === null) {
        setInspection(null);
        setIsLoading(false);
        setLoadError(null);
        return;
      }

      setIsLoading(true);
      setLoadError(null);

      try {
        const loadedInspection = await GetDeploymentFileInspection(previewHash, selectedPath);
        if (isMounted) {
          setInspection(loadedInspection);
        }
      } catch (error) {
        if (isMounted) {
          setLoadError(getErrorMessage(error));
          setInspection(null);
        }
      } finally {
        if (isMounted) {
          setIsLoading(false);
        }
      }
    };

    loadInitialInspection();

    return () => {
      isMounted = false;
    };
  }, [previewHash, selectedPath]);

  return {
    inspection,
    isLoading,
    loadError,
    refreshInspection: loadInspection,
  };
};

export type UseDeploymentFileInspectionResult = ReturnType<typeof useDeploymentFileInspection>;
