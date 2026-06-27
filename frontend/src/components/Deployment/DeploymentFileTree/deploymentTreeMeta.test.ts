import { describe, expect, it } from 'vitest';

import type { DeploymentTreeNode } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';

import { formatTreeNodeMeta } from './deploymentTreeMeta';

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

  it('uses conflict label when status is conflict', () => {
    expect(formatTreeNodeMeta(node({ Status: 'conflict', PlannedAction: 'block' }))).toBe('Conflict');
  });
});
