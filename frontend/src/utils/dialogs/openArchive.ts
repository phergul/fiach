import { Dialogs } from '@wailsio/runtime';

import { isDialogCancelError } from './dialogErrors';
import {
  normalizeMultipleDialogSelection,
  normalizeSingleDialogSelection,
} from './normalizeDialogSelection';

interface OpenArchiveOptions {
  buttonText: string;
  title: string;
}

export const openArchive = async ({ buttonText, title }: OpenArchiveOptions) => {
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
          Pattern:
            // '*.zip;*.7z;*.rar;*.tar;*.tar.gz;*.tgz;*.tar.bz2;*.tbz2;*.tar.xz;*.txz;*.tar.zst;*.tzst',
            '*.zip;*.7z;*.rar;*.tar;*.tgz;*.tbz2;*.txz;*.tzst',
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

  return normalizeSingleDialogSelection(selectedPath);
};

export const openArchives = async ({ buttonText, title }: OpenArchiveOptions) => {
  let selectedPath: string | string[] | null;

  try {
    selectedPath = await Dialogs.OpenFile({
      AllowsMultipleSelection: true,
      ButtonText: buttonText,
      CanChooseDirectories: false,
      CanChooseFiles: true,
      Filters: [
        {
          DisplayName: 'Mod Archives',
          Pattern:
            // '*.zip;*.7z;*.rar;*.tar;*.tar.gz;*.tgz;*.tar.bz2;*.tbz2;*.tar.xz;*.txz;*.tar.zst;*.tzst',
            '*.zip;*.7z;*.rar;*.tar;*.tgz;*.tbz2;*.txz;*.tzst',
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

  return normalizeMultipleDialogSelection(selectedPath);
};
