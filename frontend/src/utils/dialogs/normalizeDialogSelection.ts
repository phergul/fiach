export const normalizeSingleDialogSelection = (
  selectedPath: string | string[] | null,
): string | null => {
  if (Array.isArray(selectedPath)) {
    const firstSelectedPath = selectedPath[0];
    return firstSelectedPath && firstSelectedPath.trim() !== '' ? firstSelectedPath : null;
  }

  if (typeof selectedPath !== 'string') {
    return null;
  }

  return selectedPath.trim() === '' ? null : selectedPath;
};

export const normalizeMultipleDialogSelection = (
  selectedPath: string | string[] | null,
): string[] | null => {
  if (selectedPath === null) {
    return null;
  }

  const selectedPaths = Array.isArray(selectedPath) ? selectedPath : [selectedPath];
  const normalizedPaths = selectedPaths.map((path) => path.trim()).filter((path) => path !== '');

  return normalizedPaths;
};
