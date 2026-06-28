import type { DeploymentFileDetail } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';

export const DEPLOYMENT_CONFLICT_CLEAR_ACTION = 'clear_conflict_rule';
export const DEPLOYMENT_CONFLICT_WINNER_PREFIX = 'set_per_file_winner:';

export const resolveConflictActionLabel = (
  action: string,
  detail: DeploymentFileDetail,
): string => {
  if (action === DEPLOYMENT_CONFLICT_CLEAR_ACTION) {
    return 'Clear per-file rule';
  }

  if (!action.startsWith(DEPLOYMENT_CONFLICT_WINNER_PREFIX)) {
    return action;
  }

  const modID = Number.parseInt(action.slice(DEPLOYMENT_CONFLICT_WINNER_PREFIX.length), 10);
  const writer = detail.WriterStack.find((entry) => entry.ModID === modID);
  if (writer !== undefined && writer.ModName.trim() !== '') {
    return `Use ${writer.ModName} for this file`;
  }

  return 'Use selected mod for this file';
};

export const countModWriters = (detail: DeploymentFileDetail): number => {
  return detail.WriterStack.filter((writer) => writer.SourceKind === 'mod').length;
};

export const shouldShowConflictDecisionPanel = (detail: DeploymentFileDetail): boolean => {
  if (detail.ConflictAvailableActions.length > 0) {
    return true;
  }

  if (detail.SavedConflictRuleModID !== null && detail.SavedConflictRuleModName !== '') {
    return true;
  }

  const modWriterCount = countModWriters(detail);
  if (modWriterCount < 2) {
    return false;
  }

  return (
    detail.ConflictCategory === 'ambiguous_overwrite' ||
    detail.ConflictCategory === 'expected_overwrite'
  );
};

export const conflictDecisionGuidance =
  'To change load order for all files, edit mod order in the profile. Per-file rules only affect this path.';
