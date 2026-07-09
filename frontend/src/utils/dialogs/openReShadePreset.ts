import { Dialogs } from '@wailsio/runtime';

import { isDialogCancelError } from './dialogErrors';
import { normalizeSingleDialogSelection } from './normalizeDialogSelection';

interface OpenReShadePresetOptions {
  buttonText: string;
  title: string;
}

export const openReShadePreset = async ({ buttonText, title }: OpenReShadePresetOptions) => {
  let selectedPath: string | string[] | null;

  try {
    selectedPath = await Dialogs.OpenFile({
      AllowsMultipleSelection: false,
      ButtonText: buttonText,
      CanChooseDirectories: false,
      CanChooseFiles: true,
      Filters: [
        {
          DisplayName: 'ReShade Presets',
          Pattern: '*.ini',
        },
        {
          DisplayName: 'All Files',
          Pattern: '*.*',
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
