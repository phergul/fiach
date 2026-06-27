import { useMemo } from 'react';

import type { DeploymentTreeNode } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import {
  filterVisibleTreeNodes,
  hasActiveDeploymentTreeFilters,
  nodeMatchesFilters,
  type DeploymentTreeFilters,
} from '@utils';

import { DeploymentFileTreeFilters } from './DeploymentFileTreeFilters/DeploymentFileTreeFilters';
import { DeploymentFileTreeNodeRow } from './DeploymentFileTreeNode/DeploymentFileTreeNode';

import './DeploymentFileTree.scss';

interface DeploymentFileTreeProps {
  expandedPaths: Record<string, boolean>;
  filters: DeploymentTreeFilters;
  treeFilters: DeploymentTreeFilters;
  gameInstallPath: string;
  gameName: string;
  getChildren: (parentPath: string) => DeploymentTreeNode[];
  isScanning: boolean;
  loadErrors: Record<string, string>;
  loadingPaths: Record<string, boolean>;
  onFiltersChange: (filters: DeploymentTreeFilters) => void;
  onSelectNode: (node: DeploymentTreeNode) => void;
  onToggleNode: (node: DeploymentTreeNode, isExpanded: boolean) => void;
  rootChildren: DeploymentTreeNode[];
  scanCapReached: boolean;
  selectedPath: string | null;
}

interface TreeBranchProps {
  depth: number;
  expandedPaths: Record<string, boolean>;
  filters: DeploymentTreeFilters;
  guideContinuations: boolean[];
  gameInstallPath: string;
  gameName: string;
  getChildren: (parentPath: string) => DeploymentTreeNode[];
  loadErrors: Record<string, string>;
  loadingPaths: Record<string, boolean>;
  nodes: DeploymentTreeNode[];
  onSelectNode: (node: DeploymentTreeNode) => void;
  onToggleNode: (node: DeploymentTreeNode, isExpanded: boolean) => void;
  selectedPath: string | null;
}

const TreeBranch = ({
  depth,
  expandedPaths,
  filters,
  guideContinuations,
  gameInstallPath,
  gameName,
  getChildren,
  loadErrors,
  loadingPaths,
  nodes,
  onSelectNode,
  onToggleNode,
  selectedPath,
}: TreeBranchProps) => {
  const visibleNodes = useMemo(() => {
    const childrenByParent: Record<string, DeploymentTreeNode[]> = {};

    const collectChildren = (parentNodes: DeploymentTreeNode[]) => {
      for (const node of parentNodes) {
        if (!node.IsDirectory) {
          continue;
        }

        childrenByParent[node.Path] = getChildren(node.Path);
        collectChildren(childrenByParent[node.Path]);
      }
    };

    collectChildren(nodes);

    return filterVisibleTreeNodes(nodes, filters, childrenByParent);
  }, [filters, getChildren, nodes]);

  return (
    <ul className="deployment-file-tree-list">
      {visibleNodes.map((node, nodeIndex) => {
        const isLastSibling = nodeIndex === visibleNodes.length - 1;
        const nodeGuideContinuations =
          depth === 0 ? [] : [...guideContinuations, !isLastSibling];
        const isExpanded = expandedPaths[node.Path] === true;
        const children = node.IsDirectory ? getChildren(node.Path) : [];
        const visibleChildren =
          hasActiveDeploymentTreeFilters(filters) && node.IsDirectory
            ? children.filter((child) => {
                if (child.IsDirectory) {
                  return true;
                }

                return nodeMatchesFilters(child, filters);
              })
            : children;

        return (
          <li
            className={
              selectedPath === node.Path
                ? 'deployment-file-tree-item deployment-file-tree-item-selected'
                : 'deployment-file-tree-item'
            }
            key={node.Path}
          >
            <DeploymentFileTreeNodeRow
              depth={depth}
              gameInstallPath={gameInstallPath}
              gameName={gameName}
              guideContinuations={nodeGuideContinuations}
              isExpanded={isExpanded}
              isLoading={loadingPaths[node.Path] === true}
              loadError={loadErrors[node.Path] ?? null}
              node={node}
              onSelect={onSelectNode}
              onToggle={onToggleNode}
            />

            {node.IsDirectory && isExpanded && visibleChildren.length > 0 && (
              <TreeBranch
                depth={depth + 1}
                expandedPaths={expandedPaths}
                filters={filters}
                guideContinuations={nodeGuideContinuations}
                gameInstallPath={gameInstallPath}
                gameName={gameName}
                getChildren={getChildren}
                loadErrors={loadErrors}
                loadingPaths={loadingPaths}
                nodes={visibleChildren}
                onSelectNode={onSelectNode}
                onToggleNode={onToggleNode}
                selectedPath={selectedPath}
              />
            )}
          </li>
        );
      })}
    </ul>
  );
};

export const DeploymentFileTree = ({
  expandedPaths,
  filters,
  treeFilters,
  gameInstallPath,
  gameName,
  getChildren,
  isScanning,
  loadErrors,
  loadingPaths,
  onFiltersChange,
  onSelectNode,
  onToggleNode,
  rootChildren,
  scanCapReached,
  selectedPath,
}: DeploymentFileTreeProps) => {
  const hasVisibleNodes = rootChildren.length > 0;

  return (
    <section className="deployment-file-tree" aria-label="Deployment file tree">
      <DeploymentFileTreeFilters
        filters={filters}
        isScanning={isScanning}
        onFiltersChange={onFiltersChange}
        scanCapReached={scanCapReached}
      />

      <div className="deployment-file-tree-body">
        {!hasVisibleNodes && (
          <p className="deployment-file-tree-empty">This profile has no deployment paths to review.</p>
        )}

        {hasVisibleNodes && (
          <TreeBranch
            depth={0}
            expandedPaths={expandedPaths}
            filters={treeFilters}
            guideContinuations={[]}
            gameInstallPath={gameInstallPath}
            gameName={gameName}
            getChildren={getChildren}
            loadErrors={loadErrors}
            loadingPaths={loadingPaths}
            nodes={rootChildren}
            onSelectNode={onSelectNode}
            onToggleNode={onToggleNode}
            selectedPath={selectedPath}
          />
        )}
      </div>
    </section>
  );
};
