import { describe, expect, it } from 'vitest';

import {
  deploymentDriftDecisionLabel,
  resolveDriftDecisionLabel,
  shouldShowDriftDecisionPanel,
} from './deploymentDriftDecisionLabels';

describe('deploymentDriftDecisionLabels', () => {
  it('shows decision panel for unresolved drift and saved external/skipped paths', () => {
    expect(
      shouldShowDriftDecisionPanel('incremental', 'require_decision', 'drifted', ['skipped'], ''),
    ).toBe(true);
    expect(
      shouldShowDriftDecisionPanel('incremental', 'noop', 'external', ['clear'], 'keep_external'),
    ).toBe(true);
    expect(
      shouldShowDriftDecisionPanel('first_apply', 'require_decision', 'drifted', ['skipped'], ''),
    ).toBe(false);
  });

  it('labels missing drift apply action differently from backup and apply', () => {
    expect(resolveDriftDecisionLabel('backup_and_apply', 'missing')).toBe('Apply mod version');
    expect(resolveDriftDecisionLabel('backup_and_apply', 'modified')).toBe(
      deploymentDriftDecisionLabel.backup_and_apply,
    );
  });
});
