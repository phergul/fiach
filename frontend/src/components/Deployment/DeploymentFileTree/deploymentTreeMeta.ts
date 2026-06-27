import type { DeploymentTreeNode } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';

import { deploymentPlannedActionLabel, deploymentStatusLabel } from '../deploymentLabels';

export const treeNodeShowsStatus = (node: DeploymentTreeNode) => {
  return node.Status === 'blocked' || node.Status === 'conflict';
};

export const formatTreeNodeMeta = (node: DeploymentTreeNode) => {
  if (node.IsDirectory) {
    if (node.ChildCount <= 0) {
      return '';
    }

    return node.ChildCount === 1 ? '1 item' : `${node.ChildCount} items`;
  }

  return deploymentPlannedActionLabel[node.PlannedAction] ?? node.PlannedAction;
};

export const formatTreeNodeStatus = (node: DeploymentTreeNode) => {
  return deploymentStatusLabel[node.Status] ?? node.Status;
};
