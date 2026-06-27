import { describe, expect, it } from 'vitest';

import { deploymentTreeRowPaddingRem } from './deploymentTreeLayout';

describe('deploymentTreeLayout', () => {
  it('indents directories by depth and aligns file icons with parent folder icons', () => {
    expect(deploymentTreeRowPaddingRem(0, true)).toBe(0);
    expect(deploymentTreeRowPaddingRem(1, true)).toBe(0.875);
    expect(deploymentTreeRowPaddingRem(0, false)).toBe(1.75);
    expect(deploymentTreeRowPaddingRem(1, false)).toBe(1.75);
    expect(deploymentTreeRowPaddingRem(2, false)).toBe(2.625);
  });
});
