import { useCallback, useEffect, useState } from 'react';

import { GetDeploymentFileDetail } from '@bindings/github.com/phergul/fiach/internal/services/deploymentreviewservice';
import type { DeploymentFileDetail } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { getErrorMessage } from '@utils';

export const useDeploymentFileDetail = (previewHash: string, selectedPath: string | null) => {
  const [detail, setDetail] = useState<DeploymentFileDetail | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);

  const loadDetail = useCallback(async () => {
    if (previewHash === '' || selectedPath === null) {
      setDetail(null);
      setIsLoading(false);
      setLoadError(null);
      return null;
    }

    setIsLoading(true);
    setLoadError(null);

    try {
      const loadedDetail = await GetDeploymentFileDetail(previewHash, selectedPath);
      setDetail(loadedDetail);
      return loadedDetail;
    } catch (error) {
      const message = getErrorMessage(error);
      setLoadError(message);
      setDetail(null);
      throw error;
    } finally {
      setIsLoading(false);
    }
  }, [previewHash, selectedPath]);

  useEffect(() => {
    let isMounted = true;

    const loadInitialDetail = async () => {
      if (previewHash === '' || selectedPath === null) {
        setDetail(null);
        setIsLoading(false);
        setLoadError(null);
        return;
      }

      setIsLoading(true);
      setLoadError(null);

      try {
        const loadedDetail = await GetDeploymentFileDetail(previewHash, selectedPath);
        if (isMounted) {
          setDetail(loadedDetail);
        }
      } catch (error) {
        if (isMounted) {
          setLoadError(getErrorMessage(error));
          setDetail(null);
        }
      } finally {
        if (isMounted) {
          setIsLoading(false);
        }
      }
    };

    loadInitialDetail();

    return () => {
      isMounted = false;
    };
  }, [previewHash, selectedPath]);

  return {
    detail,
    isLoading,
    loadError,
    refreshDetail: loadDetail,
  };
};

export type UseDeploymentFileDetailResult = ReturnType<typeof useDeploymentFileDetail>;
