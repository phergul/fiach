export const DEPLOYMENT_FILE_STATUSES = ['added', 'replaced', 'blocked', 'conflict'] as const;

export const DEPLOYMENT_RISK_LEVELS = ['none', 'info', 'error'] as const;

export type DeploymentFileStatus = (typeof DEPLOYMENT_FILE_STATUSES)[number];

export type DeploymentRiskLevel = (typeof DEPLOYMENT_RISK_LEVELS)[number];

const deploymentStatusToAction: Record<DeploymentFileStatus, string> = {
  added: 'create',
  replaced: 'replace',
  blocked: 'block',
  conflict: 'conflict',
};

export const deploymentActionLabel: Record<string, string> = {
  create: 'Create',
  replace: 'Replace',
  block: 'Block',
  conflict: 'Conflict',
};

export const deploymentRiskLabel: Record<string, string> = {
  none: 'None',
  info: 'Info',
  error: 'Error',
};

export const deploymentConflictCategoryLabel: Record<string, string> = {
  expected_overwrite: 'Expected overwrite',
  ambiguous_overwrite: 'Ambiguous overwrite',
  destructive_file_directory: 'Destructive file vs directory',
};

export const DEPLOYMENT_TONE_CHIP_TONES = [
  'add',
  'replace',
  'blocked',
  'conflict',
  'warning',
  'info',
  'error',
  'default',
] as const;

export type DeploymentToneChipTone = (typeof DEPLOYMENT_TONE_CHIP_TONES)[number];

export const deploymentActionTone: Record<string, DeploymentToneChipTone> = {
  create: 'add',
  replace: 'replace',
  block: 'blocked',
  conflict: 'conflict',
};

export const deploymentRiskTone: Record<string, 'default' | 'info' | 'error'> = {
  none: 'default',
  info: 'info',
  error: 'error',
};

const deploymentSummaryOnlyTone: Record<string, DeploymentToneChipTone> = {
  blocking: 'blocked',
  warnings: 'warning',
};

const resolveDeploymentActionKey = (status: string, plannedAction = '') => {
  if (status === 'conflict') {
    return 'conflict';
  }

  if (plannedAction !== '') {
    return plannedAction;
  }

  return deploymentStatusToAction[status as DeploymentFileStatus] ?? status;
};

export const resolveDeploymentActionLabel = (status: string, plannedAction = '') => {
  const key = resolveDeploymentActionKey(status, plannedAction);
  return deploymentActionLabel[key] ?? key;
};

export const resolveDeploymentActionTone = (
  status: string,
  plannedAction = '',
): DeploymentToneChipTone => {
  const key = resolveDeploymentActionKey(status, plannedAction);
  return deploymentActionTone[key] ?? 'default';
};

export const resolveDeploymentSummaryTone = (key: string): DeploymentToneChipTone => {
  if (key in deploymentSummaryOnlyTone) {
    return deploymentSummaryOnlyTone[key];
  }

  return resolveDeploymentActionTone(key);
};
