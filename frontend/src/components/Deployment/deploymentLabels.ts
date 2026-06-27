export const DEPLOYMENT_FILE_STATUSES = ['added', 'replaced', 'blocked', 'conflict'] as const;

export const DEPLOYMENT_RISK_LEVELS = ['none', 'info', 'error'] as const;

export type DeploymentFileStatus = (typeof DEPLOYMENT_FILE_STATUSES)[number];

export type DeploymentRiskLevel = (typeof DEPLOYMENT_RISK_LEVELS)[number];

export const deploymentStatusLabel: Record<string, string> = {
  added: 'Added',
  replaced: 'Replaced',
  blocked: 'Blocked',
  conflict: 'Conflict',
};

export const deploymentRiskLabel: Record<string, string> = {
  none: 'None',
  info: 'Info',
  error: 'Error',
};

export const deploymentPlannedActionLabel: Record<string, string> = {
  create: 'Create',
  replace: 'Replace',
  block: 'Block',
};

export const deploymentConflictCategoryLabel: Record<string, string> = {
  expected_overwrite: 'Expected overwrite',
  ambiguous_overwrite: 'Ambiguous overwrite',
  destructive_file_directory: 'Destructive file vs directory',
};

export const deploymentStatusTone: Record<string, 'add' | 'blocked' | 'conflict' | 'replace'> = {
  added: 'add',
  replaced: 'replace',
  blocked: 'blocked',
  conflict: 'conflict',
};

export const deploymentRiskTone: Record<string, 'default' | 'info' | 'error'> = {
  none: 'default',
  info: 'info',
  error: 'error',
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

export const deploymentPlannedActionTone: Record<string, DeploymentToneChipTone> = {
  create: 'add',
  replace: 'replace',
  block: 'blocked',
};

export const deploymentSummaryTone: Record<string, DeploymentToneChipTone> = {
  added: 'add',
  replaced: 'replace',
  blocked: 'blocked',
  conflict: 'conflict',
  blocking: 'blocked',
  warnings: 'warning',
};
