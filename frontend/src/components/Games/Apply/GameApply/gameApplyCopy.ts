export const getApplyDisabledTitle = (
  isSameProfileApplied: boolean,
  isAnotherProfileApplied: boolean,
  appliedProfileName: string | null,
  canApply: boolean,
  isIncrementalApply: boolean,
  isAppliedProfileLoading: boolean,
  appliedProfileLoadError: string | null,
  isPreviewLoading: boolean,
  isApplyPending: boolean,
  previewAvailable: boolean,
) => {
  if (isApplyPending) {
    return 'Apply is already in progress.';
  }
  if (isSameProfileApplied && isIncrementalApply && !canApply) {
    return 'Resolve blocking drift or deployment issues before re-applying this profile.';
  }
  if (isSameProfileApplied && !isIncrementalApply) {
    return 'Incremental deployment preview is loading.';
  }
  if (isAnotherProfileApplied && appliedProfileName !== null) {
    return `${appliedProfileName} is applied. Restore vanilla before applying another profile.`;
  }
  if (isAppliedProfileLoading) {
    return 'Applied profile state is loading.';
  }
  if (appliedProfileLoadError !== null) {
    return 'Applied profile state could not be loaded.';
  }
  if (isPreviewLoading) {
    return 'Deployment preview is loading.';
  }
  if (!previewAvailable) {
    return 'Deployment preview is not ready yet.';
  }
  if (!canApply) {
    return 'Resolve blocking issues before applying this profile.';
  }

  return 'Confirm before applying this profile.';
};

export const getDeploymentReviewDescription = (
  isSameProfileApplied: boolean,
  isAnotherProfileApplied: boolean,
  appliedProfileName: string | null,
) => {
  if (isSameProfileApplied) {
    return 'Review drift and profile changes since the last apply.';
  }

  if (isAnotherProfileApplied && appliedProfileName !== null) {
    return `Restore vanilla before applying another profile. ${appliedProfileName} is currently applied.`;
  }

  return 'Review planned file changes in the deployment tree.';
};
