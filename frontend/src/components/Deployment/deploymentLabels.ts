export const DEPLOYMENT_FILE_STATUSES = ['added', 'replaced', 'blocked', 'conflict'] as const;

export const DEPLOYMENT_INCREMENTAL_STATUSES = [
  'drifted',
  'external',
  'skipped',
  'unchanged',
  'deleted',
  'restored',
] as const;

export const DEPLOYMENT_SUMMARY_STATUSES = [
  ...DEPLOYMENT_FILE_STATUSES,
  ...DEPLOYMENT_INCREMENTAL_STATUSES,
] as const;

export const DEPLOYMENT_FILTER_STATUSES = [
  ...DEPLOYMENT_FILE_STATUSES,
  ...DEPLOYMENT_INCREMENTAL_STATUSES,
] as const;

export const DEPLOYMENT_RISK_LEVELS = ['none', 'info', 'error'] as const;

export type DeploymentFileStatus = (typeof DEPLOYMENT_FILE_STATUSES)[number];

export type DeploymentRiskLevel = (typeof DEPLOYMENT_RISK_LEVELS)[number];

const deploymentStatusToAction: Record<DeploymentFileStatus, string> = {
  added: 'create',
  replaced: 'replace',
  blocked: 'block',
  conflict: 'conflict',
};

export const deploymentStatusLabel: Record<string, string> = {
  added: 'Added',
  replaced: 'Replaced',
  blocked: 'Blocked',
  conflict: 'Conflict',
  drifted: 'Drifted',
  external: 'External',
  skipped: 'Skipped',
  unchanged: 'Unchanged',
  deleted: 'Deleted',
  restored: 'Restored',
};

export const deploymentActionLabel: Record<string, string> = {
  create: 'Create',
  replace: 'Replace',
  delete: 'Delete',
  restore_baseline: 'Restore baseline',
  repair: 'Repair',
  block: 'Block',
  conflict: 'Conflict',
  require_decision: 'Decision required',
  backup_then_replace: 'Backup and apply',
  backup_then_delete: 'Backup and delete',
  backup_then_restore: 'Backup and restore',
  noop: 'No change',
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
  delete: 'replace',
  restore_baseline: 'replace',
  repair: 'warning',
  block: 'blocked',
  conflict: 'conflict',
  require_decision: 'warning',
  backup_then_replace: 'replace',
  backup_then_delete: 'replace',
  backup_then_restore: 'replace',
  noop: 'default',
  drifted: 'warning',
  external: 'info',
  skipped: 'warning',
  unchanged: 'default',
  deleted: 'replace',
  restored: 'replace',
};

export const deploymentRiskTone: Record<string, 'default' | 'info' | 'error'> = {
  none: 'default',
  info: 'info',
  error: 'error',
};

const deploymentSummaryOnlyTone: Record<string, DeploymentToneChipTone> = {
  blocking: 'blocked',
  warnings: 'warning',
  drifted: 'warning',
  external: 'info',
  skipped: 'warning',
  unchanged: 'default',
  deleted: 'replace',
  restored: 'replace',
};

const resolveDeploymentActionKey = (status: string, plannedAction = '') => {
  if (status === 'conflict') {
    return 'conflict';
  }

  if (
    status === 'drifted' ||
    status === 'external' ||
    status === 'skipped' ||
    status === 'unchanged'
  ) {
    return status;
  }

  if (plannedAction !== '') {
    return plannedAction;
  }

  return deploymentStatusToAction[status as DeploymentFileStatus] ?? status;
};

export const resolveDeploymentStatusLabel = (status: string) => {
  return deploymentStatusLabel[status] ?? status;
};

export const resolveDeploymentActionLabel = (status: string, plannedAction = '') => {
  const key = resolveDeploymentActionKey(status, plannedAction);
  if (key in deploymentStatusLabel) {
    return deploymentStatusLabel[key];
  }

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
