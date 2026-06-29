import { GetAppliedProfileSummary } from '@bindings/github.com/phergul/fiach/internal/services/profileservice';
import type { AppliedProfileSummary } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';

import { createKeyedCachedResource } from '../../cache/createKeyedCachedResource';

export const appliedProfileResource = createKeyedCachedResource<
  AppliedProfileSummary | null,
  number
>({
  emptyValue: null,
  fetch: (gameID) => GetAppliedProfileSummary(gameID),
  presence: 'hasKey',
});

export const fetchAppliedProfile = appliedProfileResource.fetch;
export const preloadAppliedProfile = appliedProfileResource.preload;
export const invalidateAppliedProfile = appliedProfileResource.invalidate;
