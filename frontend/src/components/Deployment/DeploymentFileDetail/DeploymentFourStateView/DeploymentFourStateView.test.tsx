import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { StateComparison } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';

import { DeploymentFourStateView } from './DeploymentFourStateView';

const populatedState = {
  Exists: true,
  Label: 'Current game install',
  SHA256: 'abc',
  SizeBytes: 12,
};

describe('DeploymentFourStateView', () => {
  it('highlights drift from comparison booleans and shows explanation', () => {
    render(
      <DeploymentFourStateView
        applied={populatedState}
        baseline={null}
        comparison={
          new StateComparison({
            AppliedMatchesCurrent: false,
            AppliedMatchesDesired: true,
            CurrentMatchesDesired: false,
          })
        }
        current={populatedState}
        desired={populatedState}
        driftExplanation="This file was modified on disk since the last apply."
        driftKind="modified"
        planMode="incremental"
      />,
    );

    expect(
      screen.getByText('This file was modified on disk since the last apply.'),
    ).toBeInTheDocument();
    expect(document.querySelector('.deployment-four-state-view-column-drift')).toBeTruthy();
  });

  it('shows not available yet for first apply baseline and applied columns', () => {
    render(
      <DeploymentFourStateView
        applied={null}
        baseline={null}
        comparison={
          new StateComparison({
            AppliedMatchesCurrent: false,
            AppliedMatchesDesired: false,
            CurrentMatchesDesired: false,
          })
        }
        current={{ Exists: false, Label: '', SHA256: '', SizeBytes: 0 }}
        desired={populatedState}
        driftExplanation=""
        driftKind="none"
        planMode="first_apply"
      />,
    );

    expect(screen.getAllByText('Not available yet')).toHaveLength(2);
  });
});
