import { ModSourceType } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';

const archiveExtensionPattern = /\.(?:tar\.(?:bz2|gz|xz|zst)|tbz2|tgz|txz|tzst|7z|rar|tar|zip)$/i;

export const isArchiveImportPath = (path: string) => archiveExtensionPattern.test(path.trim());

export const inferImportSourceType = (path: string): ModSourceType =>
  isArchiveImportPath(path)
    ? ModSourceType.ModSourceTypeArchive
    : ModSourceType.ModSourceTypeFolder;
