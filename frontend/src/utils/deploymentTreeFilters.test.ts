import { describe, expect, it } from 'vitest';

import type { DeploymentTreeNode } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import {
  emptyDeploymentTreeFilters,
  filterVisibleTreeNodes,
  hasActiveDeploymentTreeFilters,
  matchesSearch,
  matchesStatus,
  nodeMatchesFilters,
} from './deploymentTreeFilters';

const fileNode = (overrides: Partial<DeploymentTreeNode> = {}): DeploymentTreeNode =>
  ({
    ChildCount: 0,
    Children: [],
    HasChildren: false,
    IsDirectory: false,
    Name: 'mod.dll',
    Path: 'BepInEx/plugins/mod.dll',
    PlannedAction: 'create',
    RiskLevel: 'none',
    Status: 'added',
    ...overrides,
  }) as DeploymentTreeNode;

const directoryNode = (overrides: Partial<DeploymentTreeNode> = {}): DeploymentTreeNode =>
  ({
    ChildCount: 1,
    Children: [],
    HasChildren: true,
    IsDirectory: true,
    Name: 'plugins',
    Path: 'BepInEx/plugins',
    PlannedAction: 'create',
    RiskLevel: 'none',
    Status: 'added',
    ...overrides,
  }) as DeploymentTreeNode;

describe('deploymentTreeFilters', () => {
  it('treats empty filters as inactive', () => {
    expect(hasActiveDeploymentTreeFilters(emptyDeploymentTreeFilters())).toBe(false);
  });

  it('matches status, search, and combined filters', () => {
    const node = fileNode();

    expect(matchesStatus(node, ['added'])).toBe(true);
    expect(matchesStatus(node, ['blocked'])).toBe(false);
    expect(matchesSearch(node, 'mod.dll')).toBe(true);
    expect(matchesSearch(node, 'missing')).toBe(false);
    expect(
      nodeMatchesFilters(node, {
        risks: [],
        searchQuery: 'plugins',
        statuses: ['added'],
      }),
    ).toBe(true);
  });

  it('returns all nodes when filters are inactive', () => {
    const nodes = [directoryNode(), fileNode()];

    expect(filterVisibleTreeNodes(nodes, emptyDeploymentTreeFilters(), {})).toEqual(nodes);
  });

  it('keeps ancestor directories when a descendant matches', () => {
    const directory = directoryNode();
    const child = fileNode({ Status: 'blocked', Path: 'BepInEx/plugins/blocked.dll', Name: 'blocked.dll' });
    const filters = {
      risks: [],
      searchQuery: '',
      statuses: ['blocked'],
    };

    expect(
      filterVisibleTreeNodes([directory], filters, {
        [directory.Path]: [child],
      }),
    ).toEqual([directory]);
  });
});
