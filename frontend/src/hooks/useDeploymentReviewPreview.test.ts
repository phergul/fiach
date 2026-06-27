import { renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { BuildDeploymentReviewPreview } from '@bindings/github.com/phergul/fiach/internal/services/deploymentreviewservice';
import {
  DeploymentReviewPreview,
  DeploymentSummary,
  DeploymentTreeNode,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';

import { useDeploymentReviewPreview } from './useDeploymentReviewPreview';

vi.mock('@bindings/github.com/phergul/fiach/internal/services/deploymentreviewservice', () => ({
  BuildDeploymentReviewPreview: vi.fn(),
}));

const buildPreview = () => {
  const summary = new DeploymentSummary({
    BlockingCount: 0,
    CanApply: true,
    ProfileID: 7,
    ProfileName: 'Default',
    StatusCounts: { added: 2 },
    WarningCount: 0,
  });
  const root = new DeploymentTreeNode({
    Children: [
      new DeploymentTreeNode({
        IsDirectory: false,
        Name: 'mod.dll',
        Path: 'mod.dll',
        Status: 'added',
      }),
    ],
    IsDirectory: true,
    Name: '',
    Path: '',
  });

  return new DeploymentReviewPreview({
    PreviewHash: 'preview-hash',
    Root: root,
    Summary: summary,
  });
};

describe('useDeploymentReviewPreview', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('loads preview data for a profile', async () => {
    vi.mocked(BuildDeploymentReviewPreview).mockResolvedValue(buildPreview());

    const { result } = renderHook(() => useDeploymentReviewPreview(7));

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.previewHash).toBe('preview-hash');
    expect(result.current.summary?.CanApply).toBe(true);
    expect(result.current.rootChildren).toHaveLength(1);
  });

  it('maps load errors', async () => {
    vi.mocked(BuildDeploymentReviewPreview).mockRejectedValue(new Error('preview failed'));

    const { result } = renderHook(() => useDeploymentReviewPreview(7));

    await waitFor(() => {
      expect(result.current.loadError).toBe('Preview failed.');
    });
  });
});
