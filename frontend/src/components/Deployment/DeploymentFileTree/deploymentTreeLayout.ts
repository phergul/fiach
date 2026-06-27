const TREE_INDENT_REM = 0.875;
const TREE_TOGGLE_REM = 1.75;

export const deploymentTreeRowPaddingRem = (depth: number, isDirectory: boolean) => {
  if (isDirectory) {
    return depth * TREE_INDENT_REM;
  }

  if (depth === 0) {
    return TREE_TOGGLE_REM;
  }

  return (depth - 1) * TREE_INDENT_REM + TREE_TOGGLE_REM;
};
