import { useState } from 'react';

import { SetDeploymentDriftDecision } from '@bindings/github.com/phergul/fiach/internal/services/deploymentreviewservice';
import type {
  DeploymentFileDetail,
  DeploymentReviewPreview,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { DeploymentToneChip } from '@components/Deployment/DeploymentToneChip/DeploymentToneChip';
import { getErrorMessage } from '@utils';

import {
  deploymentDriftDecisionDescription,
  resolveDriftDecisionLabel,
  shouldShowDriftDecisionPanel,
} from '../deploymentDriftDecisionLabels';

import './DeploymentDriftDecision.scss';

interface DeploymentDriftDecisionProps {
  detail: DeploymentFileDetail;
  onPreviewUpdated: (preview: DeploymentReviewPreview) => void;
  planMode: string;
  previewHash: string;
  profileID: number;
}

export const DeploymentDriftDecision = ({
  detail,
  onPreviewUpdated,
  planMode,
  previewHash,
  profileID,
}: DeploymentDriftDecisionProps) => {
  const [isSaving, setIsSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);

  if (
    !shouldShowDriftDecisionPanel(
      planMode,
      detail.PlannedAction,
      detail.FileStatus,
      detail.AvailableActions,
      detail.UserDecision,
    )
  ) {
    return null;
  }

  const handleDecision = async (decision: string) => {
    if (isSaving) {
      return;
    }

    setIsSaving(true);
    setSaveError(null);

    try {
      const preview = await SetDeploymentDriftDecision(
        profileID,
        previewHash,
        detail.RelativePath,
        decision,
      );
      onPreviewUpdated(preview);
    } catch (error) {
      setSaveError(getErrorMessage(error));
    } finally {
      setIsSaving(false);
    }
  };

  const showSavedDecision =
    detail.UserDecision !== '' && detail.PlannedAction !== 'require_decision';

  return (
    <section className="deployment-drift-decision" aria-label="Drift decision">
      {detail.PlannedAction === 'require_decision' && (
        <p className="deployment-drift-decision-note">
          Choose how to handle this drifted file before re-applying.
        </p>
      )}

      {showSavedDecision && (
        <div className="deployment-drift-decision-current">
          <span className="deployment-drift-decision-current-label">Saved decision</span>
          <DeploymentToneChip
            label={detail.UserDecisionLabel || detail.UserDecision}
            tone={detail.FileStatus === 'skipped' ? 'warning' : 'info'}
          />
        </div>
      )}

      {saveError !== null && <p className="deployment-drift-decision-error">{saveError}</p>}

      <div className="deployment-drift-decision-actions">
        {detail.AvailableActions.map((decision) => {
          const isClearAction = decision === 'clear';
          const description =
            !isClearAction && decision in deploymentDriftDecisionDescription
              ? deploymentDriftDecisionDescription[
                  decision as keyof typeof deploymentDriftDecisionDescription
                ]
              : null;

          return (
            <button
              key={decision}
              className={
                isClearAction
                  ? 'deployment-drift-decision-action deployment-drift-decision-action-clear'
                  : 'deployment-drift-decision-action'
              }
              disabled={isSaving}
              onClick={() => {
                handleDecision(decision).catch(() => undefined);
              }}
              type="button"
            >
              <span className="deployment-drift-decision-action-label">
                {resolveDriftDecisionLabel(decision, detail.DriftKind)}
              </span>
              {description !== null && (
                <span className="deployment-drift-decision-action-description">{description}</span>
              )}
            </button>
          );
        })}
      </div>
    </section>
  );
};
