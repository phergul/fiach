const archiveExtensionPattern = /\.(?:tar\.(?:bz2|gz|xz|zst)|tbz2|tgz|txz|tzst|7z|rar|tar|zip)$/i;

export const getFolderImportName = (path: string) => {
  const trimmedPath = path.trim().replace(/[\\/]+$/, '');
  const folderName = trimmedPath.split(/[\\/]/).pop();

  return folderName && folderName.trim() !== '' ? folderName : 'Imported Mod';
};

export const getArchiveImportName = (path: string) => {
  const fileName = getFolderImportName(path);
  const archiveName = fileName.replace(archiveExtensionPattern, '');

  return archiveName.trim() === '' ? 'Imported Mod' : archiveName;
};

export const getImportSourceLabel = (sourceType: 'folder' | 'archive') =>
  sourceType === 'folder' ? 'Source folder' : 'Source archive';
