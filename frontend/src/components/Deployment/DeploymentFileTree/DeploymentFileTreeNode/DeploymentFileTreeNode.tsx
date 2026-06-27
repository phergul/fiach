import type { CSSProperties, MouseEvent } from 'react';

import { ChevronRight, File, Folder } from 'lucide-react';

import type { DeploymentTreeNode } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { formatDeploymentDisplayPath } from '@utils';

import { DeploymentToneChip } from '../../DeploymentToneChip/DeploymentToneChip';
import { deploymentTreeRowPaddingRem } from '../deploymentTreeLayout';
import { formatTreeNodeActionTone, formatTreeNodeMeta } from '../deploymentTreeMeta';

import './DeploymentFileTreeNode.scss';

interface DeploymentFileTreeNodeProps {
  depth: number;
  gameInstallPath: string;
  gameName: string;
  isExpanded: boolean;
  isLoading: boolean;
  loadError: string | null;
  node: DeploymentTreeNode;
  onSelect: (node: DeploymentTreeNode) => void;
  onToggle: (node: DeploymentTreeNode, isExpanded: boolean) => void;
}

export const DeploymentFileTreeNodeRow = ({
  depth,
  gameInstallPath,
  gameName,
  isExpanded,
  isLoading,
  loadError,
  node,
  onSelect,
  onToggle,
}: DeploymentFileTreeNodeProps) => {
  const displayPath = formatDeploymentDisplayPath(node.Path, gameInstallPath, gameName);
  const actionLabel = formatTreeNodeMeta(node);
  const actionTone = formatTreeNodeActionTone(node);
  const canExpand = node.IsDirectory && node.HasChildren;
  const showActionChip = !node.IsDirectory && actionLabel !== '';
  const rowPaddingRem = deploymentTreeRowPaddingRem(depth, node.IsDirectory);
  const rowStyle = {
    '--deployment-file-tree-row-padding': `${rowPaddingRem}rem`,
  } as CSSProperties;

  const handleRowClick = () => {
    if (canExpand) {
      onToggle(node, isExpanded);
      return;
    }

    onSelect(node);
  };

  const handleToggleClick = (event: MouseEvent<HTMLButtonElement>) => {
    event.stopPropagation();
    onToggle(node, isExpanded);
  };

  return (
    <div className="deployment-file-tree-node">
      <div
        className="deployment-file-tree-node-row"
        onClick={handleRowClick}
        onKeyDown={(event) => {
          if (event.key === 'Enter' || event.key === ' ') {
            event.preventDefault();
            handleRowClick();
          }
        }}
        role="button"
        style={rowStyle}
        tabIndex={0}
      >
        {canExpand ? (
          <button
            aria-expanded={isExpanded}
            aria-label={`${isExpanded ? 'Collapse' : 'Expand'} ${node.Name}`}
            className="deployment-file-tree-node-toggle"
            onClick={handleToggleClick}
            type="button"
          >
            <ChevronRight
              className={
                isExpanded
                  ? 'deployment-file-tree-node-toggle-icon deployment-file-tree-node-toggle-icon-expanded'
                  : 'deployment-file-tree-node-toggle-icon'
              }
              aria-hidden="true"
            />
          </button>
        ) : null}

        <span className="deployment-file-tree-node-content">
          {node.IsDirectory ? (
            <Folder className="deployment-file-tree-node-icon" aria-hidden="true" />
          ) : (
            <File className="deployment-file-tree-node-icon" aria-hidden="true" />
          )}
          <span
            className={
              node.IsDirectory
                ? 'deployment-file-tree-node-name deployment-file-tree-node-name-dir'
                : 'deployment-file-tree-node-name deployment-file-tree-node-name-file'
            }
            title={displayPath}
          >
            {node.Name}
          </span>
        </span>

        <div className="deployment-file-tree-node-meta">
          {showActionChip && <DeploymentToneChip label={actionLabel} tone={actionTone} />}
          {node.IsDirectory && actionLabel !== '' && (
            <span className="deployment-file-tree-node-meta-label">{actionLabel}</span>
          )}
        </div>
      </div>

      {isLoading && <p className="deployment-file-tree-node-message">Loading…</p>}
      {loadError !== null && (
        <p className="deployment-file-tree-node-message deployment-file-tree-node-message-error">
          {loadError}
        </p>
      )}
    </div>
  );
};
