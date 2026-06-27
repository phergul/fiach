import { describe, expect, it } from 'vitest';

import type { DeploymentTreeNode } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';

import { formatTreeNodeMeta, treeNodeShowsStatus } from './deploymentTreeMeta';

const node = (overrides: Partial<DeploymentTreeNode> = {}): DeploymentTreeNode =>
  ({
    ChildCount: 0,
    Children: [],
    HasChildren: false,
    IsDirectory: false,
    Name: 'mod.dll',
    Path: 'mod.dll',
    PlannedAction: 'create',
    RiskLevel: 'none',
    Status: 'added',
    ...overrides,
  }) as DeploymentTreeNode;

describe('deploymentTreeMeta', () => {
  it('shows planned action for files and item counts for directories', () => {
    expect(formatTreeNodeMeta(node())).toBe('Create');
    expect(formatTreeNodeMeta(node({ IsDirectory: true, HasChildren: true, ChildCount: 11 }))).toBe(
      '11 items',
    );
    expect(formatTreeNodeMeta(node({ IsDirectory: true, HasChildren: true, ChildCount: 1 }))).toBe(
      '1 item',
    );
  });

  it('only surfaces status text for blocking states', () => {
    expect(treeNodeShowsStatus(node())).toBe(false);
    expect(treeNodeShowsStatus(node({ Status: 'blocked' }))).toBe(true);
    expect(treeNodeShowsStatus(node({ Status: 'conflict' }))).toBe(true);
  });
});
