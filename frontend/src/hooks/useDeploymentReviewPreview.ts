import { useCallback, useEffect, useState } from 'react';

import { BuildDeploymentReviewPreview } from '@bindings/github.com/phergul/fiach/internal/services/deploymentreviewservice';
import type {
  DeploymentReviewPreview,
  DeploymentSummary,
  DeploymentTreeNode,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { getErrorMessage } from '@utils';

export const useDeploymentReviewPreview = (profileID: number | null) => {
  const [preview, setPreview] = useState<DeploymentReviewPreview | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);

  const loadPreview = useCallback(async () => {
    if (profileID === null) {
      setPreview(null);
      setIsLoading(false);
      setLoadError(null);
      return null;
    }

    setIsLoading(true);
    setLoadError(null);

    try {
      const loadedPreview = await BuildDeploymentReviewPreview(profileID);
      setPreview(loadedPreview);
      return loadedPreview;
    } catch (error) {
      const message = getErrorMessage(error);
      setLoadError(message);
      throw error;
    } finally {
      setIsLoading(false);
    }
  }, [profileID]);

  useEffect(() => {
    let isMounted = true;

    const loadInitialPreview = async () => {
      if (profileID === null) {
        setPreview(null);
        setIsLoading(false);
        setLoadError(null);
        return;
      }

      setIsLoading(true);
      setLoadError(null);

      try {
        const loadedPreview = await BuildDeploymentReviewPreview(profileID);
        if (isMounted) {
          setPreview(loadedPreview);
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

    loadInitialPreview();

    return () => {
      isMounted = false;
    };
  }, [profileID]);

  const summary: DeploymentSummary | null = preview?.Summary ?? null;
  const rootChildren: DeploymentTreeNode[] = preview?.Root.Children ?? [];
  const previewHash = preview?.PreviewHash ?? '';

  return {
    isLoading,
    loadError,
    preview,
    previewHash,
    refreshPreview: loadPreview,
    rootChildren,
    summary,
  };
};

export type UseDeploymentReviewPreviewResult = ReturnType<typeof useDeploymentReviewPreview>;
