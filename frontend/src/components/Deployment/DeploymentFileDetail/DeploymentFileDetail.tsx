import { FileSearch } from 'lucide-react';

import type { DeploymentFileDetail, DeploymentFileInspection, DeploymentReviewPreview } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { StateBlock } from '@components/Common/StateBlock/StateBlock';
import { deploymentPathBaseName, formatAppliedAtFromDate, formatDeploymentDisplayPath } from '@utils';

import {
  deploymentConflictCategoryLabel,
  deploymentRiskLabel,
  deploymentRiskTone,
  resolveDeploymentActionLabel,
  resolveDeploymentActionTone,
} from '../deploymentLabels';
import { DeploymentToneChip } from '../DeploymentToneChip/DeploymentToneChip';
import { DeploymentFourStateView } from './DeploymentFourStateView/DeploymentFourStateView';
import { DeploymentConflictDecision } from './DeploymentConflictDecision/DeploymentConflictDecision';
import { DeploymentDriftDecision } from './DeploymentDriftDecision/DeploymentDriftDecision';
import { DeploymentFileInspector } from './DeploymentFileInspector/DeploymentFileInspector';
import { DeploymentWriterStack } from './DeploymentWriterStack/DeploymentWriterStack';

import './DeploymentFileDetail.scss';

interface DeploymentFileDetailPanelProps {
  detail: DeploymentFileDetail | null;
  gameInstallPath: string;
  gameName: string;
  inspection: DeploymentFileInspection | null;
  inspectionError: string | null;
  isInspectionLoading: boolean;
  isLoading: boolean;
  loadError: string | null;
  onPreviewUpdated: (preview: DeploymentReviewPreview) => void;
  onRetry: () => void;
  onRetryInspection: () => void;
  planMode: string;
  previewHash: string;
  profileID: number | null;
  selectedPath: string | null;
}

export const DeploymentFileDetailPanel = ({
  detail,
  gameInstallPath,
  gameName,
  inspection,
  inspectionError,
  isInspectionLoading,
  isLoading,
  loadError,
  onPreviewUpdated,
  onRetry,
  onRetryInspection,
  planMode,
  previewHash,
  profileID,
  selectedPath,
}: DeploymentFileDetailPanelProps) => {
  if (selectedPath === null) {
    return (
      <section className="deployment-file-detail deployment-file-detail-empty" aria-label="File detail">
        <FileSearch className="deployment-file-detail-empty-icon" aria-hidden="true" />
        <StateBlock className="deployment-file-detail-empty-copy" title="No file selected" />
      </section>
    );
  }

  if (isLoading) {
    return (
      <section className="deployment-file-detail deployment-file-detail-empty" aria-label="File detail">
        <StateBlock message="Loading file detail…" title="Loading" />
      </section>
    );
  }

  if (loadError !== null) {
    return (
      <section className="deployment-file-detail deployment-file-detail-empty" aria-label="File detail">
        <StateBlock message={loadError} title="Could not load file detail" />
        <button className="deployment-file-detail-retry" onClick={onRetry} type="button">
          Retry
        </button>
      </section>
    );
  }

  if (detail === null) {
    return (
      <section className="deployment-file-detail deployment-file-detail-empty" aria-label="File detail">
        <StateBlock message="File detail is not available for this path." title="No detail" />
      </section>
    );
  }

  const displayPath = formatDeploymentDisplayPath(detail.RelativePath, gameInstallPath, gameName);
  const fileName = deploymentPathBaseName(detail.RelativePath);
  const actionLabel = resolveDeploymentActionLabel(detail.FileStatus, detail.PlannedAction);
  const actionTone = resolveDeploymentActionTone(detail.FileStatus, detail.PlannedAction);
  const riskTone = deploymentRiskTone[detail.RiskLevel] ?? 'default';

  return (
    <section className="deployment-file-detail" aria-label="File detail">
      <div className="deployment-file-detail-header-area">
        <header className="deployment-file-detail-header">
          <div className="deployment-file-detail-header-copy">
            <h3 className="deployment-file-detail-title">{fileName}</h3>
            <p className="deployment-file-detail-path">{displayPath}</p>
          </div>
          <div className="deployment-file-detail-badges">
            <DeploymentToneChip label={actionLabel} tone={actionTone} />
            <DeploymentToneChip
              label={deploymentRiskLabel[detail.RiskLevel] ?? detail.RiskLevel}
              tone={riskTone}
            />
          </div>
        </header>

        {detail.ConflictCategory !== '' && (
          <p className="deployment-file-detail-category">
            {deploymentConflictCategoryLabel[detail.ConflictCategory] ?? detail.ConflictCategory}
          </p>
        )}

        {detail.Explanation !== '' && (
          <p className="deployment-file-detail-explanation">{detail.Explanation}</p>
        )}

        {detail.LastAppliedAt !== null && (
          <p className="deployment-file-detail-applied-at">
            {formatAppliedAtFromDate(detail.LastAppliedAt)}
          </p>
        )}

        {detail.PlannedAction === 'require_decision' && detail.AvailableActions.length === 0 && (
          <p className="deployment-file-detail-decision-note">
            A decision will be required before re-applying this file.
          </p>
        )}

        {detail !== null && profileID !== null && (
          <DeploymentConflictDecision
            detail={detail}
            onPreviewUpdated={onPreviewUpdated}
            previewHash={previewHash}
            profileID={profileID}
          />
        )}

        {detail !== null && profileID !== null && (
          <DeploymentDriftDecision
            detail={detail}
            onPreviewUpdated={onPreviewUpdated}
            planMode={planMode}
            previewHash={previewHash}
            profileID={profileID}
          />
        )}
      </div>

      <div className="deployment-file-detail-body">
        <section className="deployment-file-detail-section" aria-label="File versions">
          <h4 className="deployment-file-detail-section-title">File versions</h4>
          <DeploymentFourStateView
            applied={detail.States.Applied}
            baseline={detail.States.Baseline}
            comparison={detail.Comparison}
            current={detail.States.Current}
            desired={detail.States.Desired}
            driftExplanation={detail.DriftExplanation}
            driftKind={detail.DriftKind}
            planMode={planMode}
          />
        </section>

        <section className="deployment-file-detail-section" aria-label="File comparison">
          <h4 className="deployment-file-detail-section-title">File comparison</h4>
          <DeploymentFileInspector
            inspection={inspection}
            isLoading={isInspectionLoading}
            loadError={inspectionError}
            onRetry={() => {
              onRetryInspection();
            }}
          />
        </section>

        <section className="deployment-file-detail-section" aria-label="Writer stack">
          <h4 className="deployment-file-detail-section-title">Writer stack</h4>
          <DeploymentWriterStack writers={detail.WriterStack} />
        </section>
      </div>
    </section>
  );
};
