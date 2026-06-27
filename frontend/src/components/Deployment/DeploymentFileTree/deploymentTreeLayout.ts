export const DEPLOYMENT_TREE_INDENT_REM = 0.875;
const TREE_TOGGLE_REM = 1.75;

export interface DeploymentTreeNodeGuideLayout {
  ancestorContinuations: boolean[];
  leafContinuation: boolean | null;
}

export const deploymentTreeNodeGuideLayout = (
  guideContinuations: boolean[],
): DeploymentTreeNodeGuideLayout => {
  if (guideContinuations.length === 0) {
    return {
      ancestorContinuations: [],
      leafContinuation: null,
    };
  }

  return {
    ancestorContinuations: guideContinuations.slice(0, -1),
    leafContinuation: guideContinuations[guideContinuations.length - 1] ?? null,
  };
};

export const deploymentTreeRowPaddingRem = (depth: number, isDirectory: boolean) => {
  if (isDirectory) {
    return depth * DEPLOYMENT_TREE_INDENT_REM;
  }

  if (depth === 0) {
    return TREE_TOGGLE_REM;
  }

  return (depth - 1) * DEPLOYMENT_TREE_INDENT_REM + TREE_TOGGLE_REM;
};
