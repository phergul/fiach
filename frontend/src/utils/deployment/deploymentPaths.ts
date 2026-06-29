export const normalizeDeploymentPath = (path: string) => {
  return path.trim().replace(/\\/g, '/').replace(/\/+$/g, '');
};

export const deploymentPathBaseName = (path: string) => {
  const normalizedPath = normalizeDeploymentPath(path);
  const pathParts = normalizedPath.split('/').filter(Boolean);

  return pathParts[pathParts.length - 1] ?? '';
};

export const formatDeploymentDisplayPath = (
  targetPath: string,
  gameInstallPath: string,
  gameName: string,
) => {
  const normalizedTargetPath = normalizeDeploymentPath(targetPath);
  const normalizedInstallPath = normalizeDeploymentPath(gameInstallPath);
  const installFolderName = deploymentPathBaseName(gameInstallPath) || gameName;

  if (normalizedInstallPath === '') {
    return normalizedTargetPath;
  }

  const targetPathLower = normalizedTargetPath.toLowerCase();
  const installPathLower = normalizedInstallPath.toLowerCase();

  if (targetPathLower === installPathLower) {
    return installFolderName;
  }
  if (targetPathLower.startsWith(`${installPathLower}/`)) {
    return `${installFolderName}/${normalizedTargetPath.slice(normalizedInstallPath.length + 1)}`;
  }

  return normalizedTargetPath;
};

export const formatDeploymentBytes = (bytes: number) => {
  if (bytes <= 0) {
    return '0 B';
  }

  const units = ['B', 'KB', 'MB', 'GB'];
  let value = bytes;
  let unitIndex = 0;

  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024;
    unitIndex += 1;
  }

  return `${value.toFixed(value >= 10 || unitIndex === 0 ? 0 : 1)} ${units[unitIndex]}`;
};

export const truncateDeploymentHash = (hash: string) => {
  if (hash.length <= 12) {
    return hash;
  }

  return `${hash.slice(0, 8)}…${hash.slice(-4)}`;
};
