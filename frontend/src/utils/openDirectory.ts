import { Dialogs } from '@wailsio/runtime';

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
  const selectedPath = await Dialogs.OpenFile({
    AllowsMultipleSelection: false,
    ButtonText: buttonText,
    CanChooseDirectories: true,
    CanChooseFiles: false,
    CanCreateDirectories: canCreateDirectories,
    Title: title,
  });

  if (Array.isArray(selectedPath)) {
    const firstSelectedPath = selectedPath[0];
    return firstSelectedPath && firstSelectedPath.trim() !== '' ? firstSelectedPath : null;
  }

  if (typeof selectedPath !== 'string') {
    return null;
  }

  return selectedPath.trim() === '' ? null : selectedPath;
};
