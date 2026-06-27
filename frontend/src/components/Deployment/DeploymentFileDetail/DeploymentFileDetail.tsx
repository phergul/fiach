import { FileSearch } from 'lucide-react';

import type { DeploymentFileDetail } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { StateBlock } from '@components/Common/StateBlock/StateBlock';
import { deploymentPathBaseName, formatDeploymentDisplayPath } from '@utils';

import {
  deploymentConflictCategoryLabel,
  deploymentPlannedActionLabel,
  deploymentPlannedActionTone,
  deploymentRiskLabel,
  deploymentRiskTone,
  deploymentStatusLabel,
  deploymentStatusTone,
} from '../deploymentLabels';
import { DeploymentToneChip } from '../DeploymentToneChip/DeploymentToneChip';
import { DeploymentFourStateView } from './DeploymentFourStateView/DeploymentFourStateView';
import { DeploymentWriterStack } from './DeploymentWriterStack/DeploymentWriterStack';

import './DeploymentFileDetail.scss';

interface DeploymentFileDetailPanelProps {
  detail: DeploymentFileDetail | null;
  gameInstallPath: string;
  gameName: string;
  isLoading: boolean;
  loadError: string | null;
  onRetry: () => void;
  selectedPath: string | null;
}

export const DeploymentFileDetailPanel = ({
  detail,
  gameInstallPath,
  gameName,
  isLoading,
  loadError,
  onRetry,
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
  const statusTone = deploymentStatusTone[detail.FileStatus] ?? 'replace';
  const riskTone = deploymentRiskTone[detail.RiskLevel] ?? 'default';
  const plannedActionTone = deploymentPlannedActionTone[detail.PlannedAction] ?? 'default';

  return (
    <section className="deployment-file-detail" aria-label="File detail">
      <header className="deployment-file-detail-header">
        <div className="deployment-file-detail-header-copy">
          <h3 className="deployment-file-detail-title">{fileName}</h3>
          <p className="deployment-file-detail-path">{displayPath}</p>
        </div>
        <div className="deployment-file-detail-badges">
          <DeploymentToneChip
            label={deploymentStatusLabel[detail.FileStatus] ?? detail.FileStatus}
            tone={statusTone}
          />
          <DeploymentToneChip
            label={deploymentRiskLabel[detail.RiskLevel] ?? detail.RiskLevel}
            tone={riskTone}
          />
          <DeploymentToneChip
            label={deploymentPlannedActionLabel[detail.PlannedAction] ?? detail.PlannedAction}
            tone={plannedActionTone}
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

      <DeploymentFourStateView
        applied={detail.States.Applied}
        baseline={detail.States.Baseline}
        current={detail.States.Current}
        desired={detail.States.Desired}
      />

      <section className="deployment-file-detail-section" aria-label="Writer stack">
        <h4 className="deployment-file-detail-section-title">Writer stack</h4>
        <DeploymentWriterStack writers={detail.WriterStack} />
      </section>
    </section>
  );
};
