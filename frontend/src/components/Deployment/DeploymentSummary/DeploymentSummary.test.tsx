import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import type { DeploymentSummary } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';

import { DeploymentSummaryBar } from './DeploymentSummary';

const buildSummary = (overrides: Partial<DeploymentSummary> = {}): DeploymentSummary =>
  ({
    AppliedAt: null,
    BlockingCount: 0,
    CanApply: false,
    GameID: 1,
    PlanMode: 'incremental',
    PreviewHash: 'hash',
    PreviousApplyAt: null,
    ProfileID: 2,
    ProfileName: 'Default',
    StatusCounts: {
      drifted: 2,
      added: 1,
    },
    WarningCount: 0,
    ...overrides,
  }) as DeploymentSummary;

describe('DeploymentSummaryBar', () => {
  it('renders drifted count and last applied timestamp in incremental mode', () => {
    render(
      <DeploymentSummaryBar
        summary={buildSummary({
          AppliedAt: '2026-06-27T12:00:00Z',
        })}
      />,
    );

    expect(screen.getByText('Drifted 2')).toBeInTheDocument();
    expect(screen.getByText('Last applied')).toBeInTheDocument();
    expect(screen.getByText('Can apply')).toBeInTheDocument();
    expect(screen.getByText('No')).toBeInTheDocument();
  });
});
