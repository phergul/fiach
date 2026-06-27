import { describe, expect, it } from 'vitest';

import {
  deploymentTreeNodeGuideLayout,
  deploymentTreeRowPaddingRem,
} from './deploymentTreeLayout';

describe('deploymentTreeLayout', () => {
  it('indents directories by depth and aligns file icons with parent folder icons', () => {
    expect(deploymentTreeRowPaddingRem(0, true)).toBe(0);
    expect(deploymentTreeRowPaddingRem(1, true)).toBe(0.875);
    expect(deploymentTreeRowPaddingRem(0, false)).toBe(1.75);
    expect(deploymentTreeRowPaddingRem(1, false)).toBe(1.75);
    expect(deploymentTreeRowPaddingRem(2, false)).toBe(2.625);
  });

  it('splits ancestor and leaf guide continuations for each depth', () => {
    expect(deploymentTreeNodeGuideLayout([])).toEqual({
      ancestorContinuations: [],
      leafContinuation: null,
    });
    expect(deploymentTreeNodeGuideLayout([true])).toEqual({
      ancestorContinuations: [],
      leafContinuation: true,
    });
    expect(deploymentTreeNodeGuideLayout([true, false])).toEqual({
      ancestorContinuations: [true],
      leafContinuation: false,
    });
  });
});
