import type {
  DeploymentReviewPreview,
  DeploymentSummary,
  DeploymentTreeNode,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';

import {
  deploymentPreviewResource,
  fetchDeploymentReviewPreview,
  invalidateDeploymentPreview,
  preloadDeploymentReviewPreview,
} from './deploymentPreviewResource';

export {
  deploymentPreviewResource,
  fetchDeploymentReviewPreview,
  invalidateDeploymentPreview,
  preloadDeploymentReviewPreview,
};

export const useDeploymentReviewPreview = (profileID: number | null) => {
  const cached = deploymentPreviewResource.useCached(profileID);
  const preview = cached.data;

  const summary: DeploymentSummary | null = preview?.Summary ?? null;
  const rootChildren: DeploymentTreeNode[] = preview?.Root.Children ?? [];
  const previewHash = preview?.PreviewHash ?? '';

  return {
    applyPreview: (updatedPreview: DeploymentReviewPreview) => {
      cached.setCached(updatedPreview);
    },
    isInitialLoading: cached.isInitialLoading,
    isLoading: cached.isLoading,
    isRefreshing: cached.isRefreshing,
    loadError: cached.loadError,
    preview,
    previewHash,
    refreshPreview: cached.refresh,
    rootChildren,
    summary,
  };
};

export type UseDeploymentReviewPreviewResult = ReturnType<typeof useDeploymentReviewPreview>;
