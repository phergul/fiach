import { useEffect, useMemo, useState } from 'react';

import type { DeploymentReviewPreview, DeploymentTreeNode } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import {
  useDeploymentFileDetail,
  useDeploymentTree,
  type UseDeploymentTreeResult,
} from '@hooks';
import { emptyDeploymentTreeFilters, type DeploymentTreeFilters } from '@utils';

import { DeploymentFileDetailPanel } from '../DeploymentFileDetail/DeploymentFileDetail';
import { DeploymentFileTree } from '../DeploymentFileTree/DeploymentFileTree';

import './DeploymentReview.scss';

interface DeploymentReviewProps {
  gameInstallPath: string;
  gameName: string;
  onPreviewRefreshNeeded: () => void;
  onPreviewUpdated: (preview: DeploymentReviewPreview) => void;
  planMode: string;
  previewHash: string;
  profileID: number | null;
  rootChildren: DeploymentTreeNode[];
}

const SEARCH_DEBOUNCE_MS = 250;

export const DeploymentReview = ({
  gameInstallPath,
  gameName,
  onPreviewRefreshNeeded,
  onPreviewUpdated,
  planMode,
  previewHash,
  profileID,
  rootChildren,
}: DeploymentReviewProps) => {
  const [filters, setFilters] = useState<DeploymentTreeFilters>(emptyDeploymentTreeFilters());
  const [debouncedSearchQuery, setDebouncedSearchQuery] = useState('');
  const [selectedPath, setSelectedPath] = useState<string | null>(null);

  useEffect(() => {
    const timeoutID = window.setTimeout(() => {
      setDebouncedSearchQuery(filters.searchQuery);
    }, SEARCH_DEBOUNCE_MS);

    return () => {
      window.clearTimeout(timeoutID);
    };
  }, [filters.searchQuery]);

  const treeFilters = useMemo(
    () => ({
      ...filters,
      searchQuery: debouncedSearchQuery,
    }),
    [debouncedSearchQuery, filters.risks, filters.statuses],
  );

  const treeManager: UseDeploymentTreeResult = useDeploymentTree(
    previewHash,
    rootChildren,
    treeFilters,
  );

  const { detail, isLoading, loadError, refreshDetail } = useDeploymentFileDetail(
    previewHash,
    selectedPath,
  );

  useEffect(() => {
    setSelectedPath(null);
  }, [profileID]);

  useEffect(() => {
    if (loadError !== null && loadError.includes('preview is no longer available')) {
      onPreviewRefreshNeeded();
    }
  }, [loadError, onPreviewRefreshNeeded]);

  const handlePreviewUpdated = (preview: DeploymentReviewPreview) => {
    onPreviewUpdated(preview);
  };

  const handleSelectNode = (node: DeploymentTreeNode) => {
    if (node.IsDirectory) {
      return;
    }

    setSelectedPath(node.Path);
  };

  const handleToggleNode = async (node: DeploymentTreeNode, isExpanded: boolean) => {
    await treeManager.toggleNode(node.Path, isExpanded);
  };

  return (
    <section className="deployment-review" aria-label="Deployment review">
      <div className="deployment-review-tree-pane">
        <DeploymentFileTree
          expandedPaths={treeManager.expandedPaths}
          filters={filters}
          treeFilters={treeFilters}
          gameInstallPath={gameInstallPath}
          gameName={gameName}
          getChildren={treeManager.getChildren}
          isScanning={treeManager.isScanning}
          loadErrors={treeManager.loadErrors}
          loadingPaths={treeManager.loadingPaths}
          onFiltersChange={setFilters}
          onSelectNode={handleSelectNode}
          onToggleNode={handleToggleNode}
          rootChildren={rootChildren}
          scanCapReached={treeManager.scanCapReached}
          selectedPath={selectedPath}
        />
      </div>

      <div className="deployment-review-detail-pane">
        <DeploymentFileDetailPanel
          detail={detail}
          gameInstallPath={gameInstallPath}
          gameName={gameName}
          isLoading={isLoading}
          loadError={loadError}
          onPreviewUpdated={handlePreviewUpdated}
          onRetry={() => {
            refreshDetail().catch(() => undefined);
          }}
          planMode={planMode}
          previewHash={previewHash}
          profileID={profileID}
          selectedPath={selectedPath}
        />
      </div>
    </section>
  );
};
