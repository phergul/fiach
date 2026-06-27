import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import type { DeploymentTreeNode } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';

import { DeploymentFileTree } from './DeploymentFileTree';

const rootChildren: DeploymentTreeNode[] = [
  {
    ChildCount: 1,
    Children: [],
    HasChildren: true,
    IsDirectory: true,
    Name: 'BepInEx',
    Path: 'BepInEx',
    PlannedAction: 'create',
    RiskLevel: 'none',
    Status: 'added',
  } as DeploymentTreeNode,
];

describe('DeploymentFileTree', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders status filters and calls toggle handlers', async () => {
    const user = userEvent.setup();
    const onToggleNode = vi.fn().mockResolvedValue(undefined);
    const onSelectNode = vi.fn();

    render(
      <DeploymentFileTree
        expandedPaths={{}}
        filters={{ risks: [], searchQuery: '', statuses: [] }}
        treeFilters={{ risks: [], searchQuery: '', statuses: [] }}
        gameInstallPath="/Games/Test"
        gameName="Test"
        getChildren={() => []}
        isScanning={false}
        loadErrors={{}}
        loadingPaths={{}}
        onFiltersChange={vi.fn()}
        onSelectNode={onSelectNode}
        onToggleNode={onToggleNode}
        rootChildren={rootChildren}
        scanCapReached={false}
        selectedPath={null}
      />,
    );

    expect(screen.getByRole('button', { name: 'Status' })).toBeInTheDocument();
    expect(screen.getByText('BepInEx')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Expand BepInEx' }));
    expect(onToggleNode).toHaveBeenCalled();
  });

  it('shows scanning status when filters are active', async () => {
    render(
      <DeploymentFileTree
        expandedPaths={{}}
        filters={{ risks: [], searchQuery: 'dll', statuses: ['added'] }}
        treeFilters={{ risks: [], searchQuery: 'dll', statuses: ['added'] }}
        gameInstallPath="/Games/Test"
        gameName="Test"
        getChildren={() => []}
        isScanning={true}
        loadErrors={{}}
        loadingPaths={{}}
        onFiltersChange={vi.fn()}
        onSelectNode={vi.fn()}
        onToggleNode={vi.fn()}
        rootChildren={rootChildren}
        scanCapReached={false}
        selectedPath={null}
      />,
    );

    await waitFor(() => {
      expect(screen.getByText('Scanning files…')).toBeInTheDocument();
    });
  });
});
