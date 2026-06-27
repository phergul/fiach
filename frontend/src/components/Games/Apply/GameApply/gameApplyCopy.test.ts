import { describe, expect, it } from 'vitest';

import { getApplyDisabledTitle, getDeploymentReviewDescription } from './gameApplyCopy';

describe('gameApplyCopy', () => {
  it('describes same-profile incremental review', () => {
    expect(getDeploymentReviewDescription(true, false, 'Default')).toBe(
      'Review drift and profile changes since the last apply.',
    );
    expect(
      getApplyDisabledTitle(true, false, 'Default', false, false, null, false, false, false, true),
    ).toContain('read-only');
  });

  it('describes another applied profile', () => {
    expect(getDeploymentReviewDescription(false, true, 'Other')).toBe(
      'Restore vanilla before applying another profile. Other is currently applied.',
    );
  });
});
