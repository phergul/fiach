import type { ComponentProps } from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';

import { SetDeploymentConflictRule } from '@bindings/github.com/phergul/fiach/internal/services/deploymentreviewservice';
import type {
  DeploymentFileDetail,
  DeploymentReviewPreview,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import {
  DeploymentFileDetail as DeploymentFileDetailModel,
  DeploymentReviewPreview as DeploymentReviewPreviewModel,
  DeploymentSummary,
  WriterEntryDTO,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';

import { DeploymentConflictDecision } from './DeploymentConflictDecision';

vi.mock('@bindings/github.com/phergul/fiach/internal/services/deploymentreviewservice', () => ({
  SetDeploymentConflictRule: vi.fn(),
}));

const buildDetail = (overrides: Partial<DeploymentFileDetail> = {}): DeploymentFileDetail => {
  return new DeploymentFileDetailModel({
    RelativePath: 'Shared/plugin.txt',
    ConflictCategory: 'ambiguous_overwrite',
    ConflictAvailableActions: ['set_per_file_winner:1', 'set_per_file_winner:2'],
    ProfileModsURL: '/library/42',
    WriterStack: [
      new WriterEntryDTO({
        Order: 1,
        SourceKind: 'mod',
        SourceID: 'mod:1',
        ModID: 1,
        ModName: 'Alpha',
        LoadOrder: 0,
      }),
      new WriterEntryDTO({
        Order: 2,
        SourceKind: 'mod',
        SourceID: 'mod:2',
        ModID: 2,
        ModName: 'Beta',
        LoadOrder: 0,
      }),
    ],
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
      GameID: 42,
      PlanMode: 'first_apply',
      PreviewHash: 'updated-hash',
      PreviousApplyAt: null,
      ProfileID: 1,
      ProfileName: 'Default',
      StatusCounts: { replaced: 1 },
      WarningCount: 0,
    }),
  });
};

const renderConflictDecision = (props: ComponentProps<typeof DeploymentConflictDecision>) => {
  return render(
    <MemoryRouter>
      <DeploymentConflictDecision {...props} />
    </MemoryRouter>,
  );
};

describe('DeploymentConflictDecision', () => {
  it('renders per-mod conflict actions and profile link', () => {
    renderConflictDecision({
      detail: buildDetail(),
      onPreviewUpdated: vi.fn(),
      previewHash: 'preview-hash',
      profileID: 1,
    });

    expect(screen.getByRole('button', { name: /Use Alpha for this file/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Use Beta for this file/i })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /Open profile mods/i })).toHaveAttribute(
      'href',
      '/library/42',
    );
  });

  it('saves a per-file winner and refreshes preview state', async () => {
    const onPreviewUpdated = vi.fn();
    vi.mocked(SetDeploymentConflictRule).mockResolvedValue(buildPreview());

    renderConflictDecision({
      detail: buildDetail(),
      onPreviewUpdated,
      previewHash: 'preview-hash',
      profileID: 1,
    });

    fireEvent.click(screen.getByRole('button', { name: /Use Alpha for this file/i }));

    await waitFor(() => {
      expect(SetDeploymentConflictRule).toHaveBeenCalledWith(
        1,
        'preview-hash',
        'Shared/plugin.txt',
        'set_per_file_winner:1',
      );
    });
    expect(onPreviewUpdated).toHaveBeenCalled();
  });

  it('shows clear action for saved per-file rules', () => {
    renderConflictDecision({
      detail: buildDetail({
        ConflictCategory: 'expected_overwrite',
        ConflictAvailableActions: [
          'set_per_file_winner:1',
          'set_per_file_winner:2',
          'clear_conflict_rule',
        ],
        SavedConflictRuleModID: 1,
        SavedConflictRuleModName: 'Alpha',
      }),
      onPreviewUpdated: vi.fn(),
      previewHash: 'preview-hash',
      profileID: 1,
    });

    expect(screen.getByText('Saved per-file rule')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Clear per-file rule/i })).toBeInTheDocument();
  });
});
