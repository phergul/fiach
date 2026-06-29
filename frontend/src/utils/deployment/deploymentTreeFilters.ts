import type { DeploymentTreeNode } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';

export interface DeploymentTreeFilters {
  risks: string[];
  searchQuery: string;
  statuses: string[];
}

export const emptyDeploymentTreeFilters = (): DeploymentTreeFilters => ({
  risks: [],
  searchQuery: '',
  statuses: [],
});

export const hasActiveDeploymentTreeFilters = (filters: DeploymentTreeFilters): boolean => {
  return (
    filters.statuses.length > 0 || filters.risks.length > 0 || filters.searchQuery.trim().length > 0
  );
};

export const matchesStatus = (node: DeploymentTreeNode, selectedStatuses: string[]): boolean => {
  if (selectedStatuses.length === 0) {
    return true;
  }

  return selectedStatuses.includes(node.Status);
};

export const matchesRisk = (node: DeploymentTreeNode, selectedRisks: string[]): boolean => {
  if (selectedRisks.length === 0) {
    return true;
  }

  return selectedRisks.includes(node.RiskLevel);
};

export const matchesSearch = (node: DeploymentTreeNode, query: string): boolean => {
  const normalizedQuery = query.trim().toLowerCase();
  if (normalizedQuery === '') {
    return true;
  }

  return (
    node.Name.toLowerCase().includes(normalizedQuery) ||
    node.Path.toLowerCase().includes(normalizedQuery)
  );
};

export const nodeMatchesFilters = (
  node: DeploymentTreeNode,
  filters: DeploymentTreeFilters,
): boolean => {
  return (
    matchesStatus(node, filters.statuses) &&
    matchesRisk(node, filters.risks) &&
    matchesSearch(node, filters.searchQuery)
  );
};

const directoryHasMatchingDescendant = (
  node: DeploymentTreeNode,
  filters: DeploymentTreeFilters,
  childrenByParent: Record<string, DeploymentTreeNode[]>,
): boolean => {
  if (!node.IsDirectory) {
    return nodeMatchesFilters(node, filters);
  }

  const children = childrenByParent[node.Path];
  if (children === undefined) {
    return nodeMatchesFilters(node, filters);
  }

  for (const child of children) {
    if (child.IsDirectory) {
      if (directoryHasMatchingDescendant(child, filters, childrenByParent)) {
        return true;
      }
      continue;
    }

    if (nodeMatchesFilters(child, filters)) {
      return true;
    }
  }

  return nodeMatchesFilters(node, filters);
};

export const filterVisibleTreeNodes = (
  nodes: DeploymentTreeNode[],
  filters: DeploymentTreeFilters,
  childrenByParent: Record<string, DeploymentTreeNode[]>,
): DeploymentTreeNode[] => {
  if (!hasActiveDeploymentTreeFilters(filters)) {
    return nodes;
  }

  return nodes.filter((node) => {
    if (!node.IsDirectory) {
      return nodeMatchesFilters(node, filters);
    }

    return directoryHasMatchingDescendant(node, filters, childrenByParent);
  });
};

export const collectDirectoryPaths = (
  nodes: DeploymentTreeNode[],
  childrenByParent: Record<string, DeploymentTreeNode[]>,
): string[] => {
  const paths: string[] = [];

  const visit = (currentNodes: DeploymentTreeNode[]) => {
    for (const node of currentNodes) {
      if (!node.IsDirectory || !node.HasChildren) {
        continue;
      }

      paths.push(node.Path);
      const children = childrenByParent[node.Path];
      if (children !== undefined) {
        visit(children);
      }
    }
  };

  visit(nodes);
  return paths;
};
