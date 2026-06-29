import { BuildDeploymentReviewPreview } from '@bindings/github.com/phergul/fiach/internal/services/deploymentreviewservice';
import type { DeploymentReviewPreview } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';

import { createKeyedCachedResource } from '../../cache/createKeyedCachedResource';

export const deploymentPreviewResource = createKeyedCachedResource<
  DeploymentReviewPreview | null,
  number
>({
  emptyValue: null,
  fetch: (profileID) => BuildDeploymentReviewPreview(profileID),
  presence: 'hasValue',
});

export const fetchDeploymentReviewPreview = deploymentPreviewResource.fetch;
export const preloadDeploymentReviewPreview = deploymentPreviewResource.preload;
export const invalidateDeploymentPreview = deploymentPreviewResource.invalidate;
