export const DEPLOYMENT_DRIFT_DECISIONS = [
  'backup_and_apply',
  'keep_external',
  'skipped',
  'clear',
] as const;

export type DeploymentDriftDecisionValue = (typeof DEPLOYMENT_DRIFT_DECISIONS)[number];

export const deploymentDriftDecisionLabel: Record<DeploymentDriftDecisionValue, string> = {
  backup_and_apply: 'Backup and apply',
  keep_external: 'Keep external',
  skipped: 'Skip',
  clear: 'Clear decision',
};

export const deploymentDriftDecisionDescription: Record<
  Exclude<DeploymentDriftDecisionValue, 'clear'>,
  string
> = {
  backup_and_apply: 'Archive the current file, then apply the mod version on confirm.',
  keep_external: 'Leave the external file in place and stop managing this path.',
  skipped: 'Leave this path unchanged for now and keep the decision across sessions.',
};

export const resolveDriftDecisionLabel = (
  decision: string,
  driftKind = '',
): string => {
  if (decision === 'backup_and_apply' && driftKind === 'missing') {
    return 'Apply mod version';
  }

  if (decision in deploymentDriftDecisionLabel) {
    return deploymentDriftDecisionLabel[decision as DeploymentDriftDecisionValue];
  }

  return decision;
};

export const shouldShowDriftDecisionPanel = (
  planMode: string,
  plannedAction: string,
  fileStatus: string,
  availableActions: string[],
  userDecision: string,
) => {
  if (planMode !== 'incremental') {
    return false;
  }

  if (plannedAction === 'require_decision') {
    return true;
  }

  if (userDecision !== '' && (fileStatus === 'external' || fileStatus === 'skipped')) {
    return true;
  }

  return availableActions.length > 0;
};
