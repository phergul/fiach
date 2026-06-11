import { Dialogs } from '@wailsio/runtime';

import { isDialogCancelError } from './dialogErrors';

interface OpenArchiveOptions {
  buttonText: string;
  title: string;
}

export const openArchive = async ({
  buttonText,
  title,
}: OpenArchiveOptions) => {
  let selectedPath: string | string[] | null;

  try {
    selectedPath = await Dialogs.OpenFile({
      AllowsMultipleSelection: false,
      ButtonText: buttonText,
      CanChooseDirectories: false,
      CanChooseFiles: true,
      Filters: [
        {
          DisplayName: 'Mod Archives',
          Pattern: '*.zip;*.7z;*.rar;*.tar;*.tar.gz;*.tgz;*.tar.bz2;*.tbz2;*.tar.xz;*.txz;*.tar.zst;*.tzst',
        },
      ],
      Title: title,
    });
  } catch (error) {
    if (isDialogCancelError(error)) {
      return null;
    }

    throw error;
  }

  if (Array.isArray(selectedPath)) {
    const firstSelectedPath = selectedPath[0];
    return firstSelectedPath && firstSelectedPath.trim() !== '' ? firstSelectedPath : null;
  }

  if (typeof selectedPath !== 'string') {
    return null;
  }

  return selectedPath.trim() === '' ? null : selectedPath;
};
