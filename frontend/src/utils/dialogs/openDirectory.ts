import { Dialogs } from '@wailsio/runtime';

import { isDialogCancelError } from './dialogErrors';
import {
  normalizeMultipleDialogSelection,
  normalizeSingleDialogSelection,
} from './normalizeDialogSelection';

interface OpenDirectoryOptions {
  buttonText: string;
  canCreateDirectories?: boolean;
  title: string;
}

export const openDirectory = async ({
  buttonText,
  canCreateDirectories = false,
  title,
}: OpenDirectoryOptions) => {
  let selectedPath: string | string[] | null;

  try {
    selectedPath = await Dialogs.OpenFile({
      AllowsMultipleSelection: false,
      ButtonText: buttonText,
      CanChooseDirectories: true,
      CanChooseFiles: false,
      CanCreateDirectories: canCreateDirectories,
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

export const openDirectories = async ({
  buttonText,
  canCreateDirectories = false,
  title,
}: OpenDirectoryOptions) => {
  let selectedPath: string | string[] | null;

  try {
    selectedPath = await Dialogs.OpenFile({
      AllowsMultipleSelection: true,
      ButtonText: buttonText,
      CanChooseDirectories: true,
      CanChooseFiles: false,
      CanCreateDirectories: canCreateDirectories,
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
