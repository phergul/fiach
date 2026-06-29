import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { SetDeploymentDriftDecision } from '@bindings/github.com/phergul/fiach/internal/services/deploymentreviewservice';
import type {
  DeploymentFileDetail,
  DeploymentReviewPreview,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import {
  DeploymentFileDetail as DeploymentFileDetailModel,
  DeploymentReviewPreview as DeploymentReviewPreviewModel,
  DeploymentSummary,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';

import { DeploymentDriftDecision } from './DeploymentDriftDecision';

vi.mock('@bindings/github.com/phergul/fiach/internal/services/deploymentreviewservice', () => ({
  SetDeploymentDriftDecision: vi.fn(),
}));

const buildDetail = (overrides: Partial<DeploymentFileDetail> = {}): DeploymentFileDetail => {
  return new DeploymentFileDetailModel({
    RelativePath: 'Mods/SkyUI/Data/SkyUI.esp',
    PlannedAction: 'require_decision',
    FileStatus: 'drifted',
    DriftKind: 'modified',
    AvailableActions: ['backup_and_apply', 'keep_external', 'skipped'],
    UserDecision: '',
    UserDecisionLabel: '',
    ...overrides,
  });
};

const buildPreview = (): DeploymentReviewPreview => {
  return new DeploymentReviewPreviewModel({
    PreviewHash: 'updated-hash',
    Summary: new DeploymentSummary({
      AppliedAt: null,
      BlockingCount: 0,
      CanApply: true,
      GameID: 1,
      PlanMode: 'incremental',
      PreviewHash: 'updated-hash',
      PreviousApplyAt: null,
      ProfileID: 1,
      ProfileName: 'Default',
      StatusCounts: { external: 1 },
      WarningCount: 0,
    }),
  });
};

describe('DeploymentDriftDecision', () => {
  it('renders available drift actions for unresolved drift', () => {
    render(
      <DeploymentDriftDecision
        detail={buildDetail()}
        onPreviewUpdated={vi.fn()}
        planMode="incremental"
        previewHash="preview-hash"
        profileID={1}
      />,
    );

    expect(screen.getByRole('button', { name: /Backup and apply/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Keep external/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Skip/i })).toBeInTheDocument();
  });

  it('labels missing drift apply action without backup wording', () => {
    render(
      <DeploymentDriftDecision
        detail={buildDetail({
          DriftKind: 'missing',
          AvailableActions: ['skipped', 'backup_and_apply'],
        })}
        onPreviewUpdated={vi.fn()}
        planMode="incremental"
        previewHash="preview-hash"
        profileID={1}
      />,
    );

    expect(screen.getByRole('button', { name: /Apply mod version/i })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /Keep external/i })).not.toBeInTheDocument();
  });

  it('saves a decision and refreshes preview state', async () => {
    const onPreviewUpdated = vi.fn();
    vi.mocked(SetDeploymentDriftDecision).mockResolvedValue(buildPreview());

    render(
      <DeploymentDriftDecision
        detail={buildDetail()}
        onPreviewUpdated={onPreviewUpdated}
        planMode="incremental"
        previewHash="preview-hash"
        profileID={1}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: /Keep external/i }));

    await waitFor(() => {
      expect(SetDeploymentDriftDecision).toHaveBeenCalledWith(
        1,
        'preview-hash',
        'Mods/SkyUI/Data/SkyUI.esp',
        'keep_external',
      );
    });
    expect(onPreviewUpdated).toHaveBeenCalled();
  });

  it('shows clear action for saved external decisions', () => {
    render(
      <DeploymentDriftDecision
        detail={buildDetail({
          PlannedAction: 'noop',
          FileStatus: 'external',
          UserDecision: 'keep_external',
          UserDecisionLabel: 'Keep external',
          AvailableActions: ['clear'],
        })}
        onPreviewUpdated={vi.fn()}
        planMode="incremental"
        previewHash="preview-hash"
        profileID={1}
      />,
    );

    expect(screen.getByText('Saved decision')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Clear decision/i })).toBeInTheDocument();
  });
});
