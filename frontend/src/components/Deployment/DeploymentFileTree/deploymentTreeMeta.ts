import type { DeploymentTreeNode } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';

import { resolveDeploymentActionLabel, resolveDeploymentActionTone } from '../deploymentLabels';

export const formatTreeNodeMeta = (node: DeploymentTreeNode) => {
  if (node.IsDirectory) {
    if (node.ChildCount <= 0) {
      return '';
    }

    return node.ChildCount === 1 ? '1 item' : `${node.ChildCount} items`;
  }

  return resolveDeploymentActionLabel(node.Status, node.PlannedAction);
};

export const formatTreeNodeActionTone = (node: DeploymentTreeNode) => {
  return resolveDeploymentActionTone(node.Status, node.PlannedAction);
};
