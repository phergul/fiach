import { useState } from 'react';
import { Link } from 'react-router-dom';

import { SetDeploymentConflictRule } from '@bindings/github.com/phergul/fiach/internal/services/deploymentreviewservice';
import type {
  DeploymentFileDetail,
  DeploymentReviewPreview,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { DeploymentToneChip } from '@components/Deployment/DeploymentToneChip/DeploymentToneChip';
import { getErrorMessage } from '@utils';

import {
  conflictDecisionGuidance,
  DEPLOYMENT_CONFLICT_CLEAR_ACTION,
  resolveConflictActionLabel,
  shouldShowConflictDecisionPanel,
} from '../deploymentConflictDecisionLabels';

import './DeploymentConflictDecision.scss';

interface DeploymentConflictDecisionProps {
  detail: DeploymentFileDetail;
  onPreviewUpdated: (preview: DeploymentReviewPreview) => void;
  previewHash: string;
  profileID: number;
}

export const DeploymentConflictDecision = ({
  detail,
  onPreviewUpdated,
  previewHash,
  profileID,
}: DeploymentConflictDecisionProps) => {
  const [isSaving, setIsSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);

  if (!shouldShowConflictDecisionPanel(detail)) {
    return null;
  }

  const handleAction = async (action: string) => {
    if (isSaving) {
      return;
    }

    setIsSaving(true);
    setSaveError(null);

    try {
      const preview = await SetDeploymentConflictRule(
        profileID,
        previewHash,
        detail.RelativePath,
        action,
      );
      onPreviewUpdated(preview);
    } catch (error) {
      setSaveError(getErrorMessage(error));
    } finally {
      setIsSaving(false);
    }
  };

  const showSavedRule =
    detail.SavedConflictRuleModID !== null && detail.SavedConflictRuleModName.trim() !== '';

  return (
    <section className="deployment-conflict-decision" aria-label="Conflict resolution">
      {detail.ConflictCategory === 'ambiguous_overwrite' && (
        <p className="deployment-conflict-decision-note">
          Choose which mod should provide this file before applying.
        </p>
      )}

      {showSavedRule && (
        <div className="deployment-conflict-decision-current">
          <span className="deployment-conflict-decision-current-label">Saved per-file rule</span>
          <DeploymentToneChip label={detail.SavedConflictRuleModName} tone="info" />
        </div>
      )}

      {saveError !== null && <p className="deployment-conflict-decision-error">{saveError}</p>}

      <div className="deployment-conflict-decision-actions">
        {detail.ConflictAvailableActions.map((action) => {
          const isClearAction = action === DEPLOYMENT_CONFLICT_CLEAR_ACTION;

          return (
            <button
              key={action}
              className={
                isClearAction
                  ? 'deployment-conflict-decision-action deployment-conflict-decision-action-clear'
                  : 'deployment-conflict-decision-action'
              }
              disabled={isSaving}
              onClick={() => {
                handleAction(action).catch(() => undefined);
              }}
              type="button"
            >
              <span className="deployment-conflict-decision-action-label">
                {resolveConflictActionLabel(action, detail)}
              </span>
            </button>
          );
        })}
      </div>

      {detail.ProfileModsURL !== '' && (
        <p className="deployment-conflict-decision-guidance">
          {conflictDecisionGuidance}{' '}
          <Link className="deployment-conflict-decision-guidance-link" to={detail.ProfileModsURL}>
            Open profile mods
          </Link>
        </p>
      )}
    </section>
  );
};
