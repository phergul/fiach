import { renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { BuildDeploymentReviewPreview } from '@bindings/github.com/phergul/fiach/internal/services/deploymentreviewservice';
import type { DeploymentReviewPreview } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';

import {
  deploymentPreviewResource,
  invalidateDeploymentPreview,
  useDeploymentReviewPreview,
} from './useDeploymentReviewPreview';

vi.mock('@bindings/github.com/phergul/fiach/internal/services/deploymentreviewservice', () => ({
  BuildDeploymentReviewPreview: vi.fn(),
}));

const profileID = 44;
const firstPreview = {
  PreviewHash: 'preview-v1',
  Root: { Children: [] },
  Summary: { CanApply: true, PlanMode: 'full' },
} as unknown as DeploymentReviewPreview;
const secondPreview = {
  PreviewHash: 'preview-v2',
  Root: { Children: [] },
  Summary: { CanApply: true, PlanMode: 'full' },
} as unknown as DeploymentReviewPreview;

describe('useDeploymentReviewPreview', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    deploymentPreviewResource.clear();
    vi.mocked(BuildDeploymentReviewPreview).mockResolvedValue(firstPreview);
  });

  it('returns a cached preview immediately for the same profile', async () => {
    const firstHook = renderHook(() => useDeploymentReviewPreview(profileID));

    await waitFor(() => {
      expect(firstHook.result.current.preview).toEqual(firstPreview);
    });
    firstHook.unmount();

    vi.mocked(BuildDeploymentReviewPreview).mockResolvedValue(secondPreview);

    const secondHook = renderHook(() => useDeploymentReviewPreview(profileID));

    expect(secondHook.result.current.preview).toEqual(firstPreview);
    expect(secondHook.result.current.isInitialLoading).toBe(false);
    expect(secondHook.result.current.isRefreshing).toBe(true);

    await waitFor(() => {
      expect(secondHook.result.current.preview).toEqual(secondPreview);
    });
    expect(BuildDeploymentReviewPreview).toHaveBeenCalledTimes(2);
  });

  it('refetches when refreshPreview is called even with a cached preview', async () => {
    const { result } = renderHook(() => useDeploymentReviewPreview(profileID));

    await waitFor(() => {
      expect(result.current.preview).toEqual(firstPreview);
    });
    expect(BuildDeploymentReviewPreview).toHaveBeenCalledOnce();

    vi.mocked(BuildDeploymentReviewPreview).mockResolvedValue(secondPreview);

    await result.current.refreshPreview();

    await waitFor(() => {
      expect(result.current.preview).toEqual(secondPreview);
    });
    expect(BuildDeploymentReviewPreview).toHaveBeenCalledTimes(2);
  });

  it('drops cached preview after invalidation', async () => {
    const firstHook = renderHook(() => useDeploymentReviewPreview(profileID));

    await waitFor(() => {
      expect(firstHook.result.current.preview).toEqual(firstPreview);
    });
    firstHook.unmount();

    invalidateDeploymentPreview(profileID);
    vi.mocked(BuildDeploymentReviewPreview).mockResolvedValue(secondPreview);

    const secondHook = renderHook(() => useDeploymentReviewPreview(profileID));

    expect(secondHook.result.current.preview).toBeNull();
    expect(secondHook.result.current.isInitialLoading).toBe(true);

    await waitFor(() => {
      expect(secondHook.result.current.preview).toEqual(secondPreview);
    });
  });
});
