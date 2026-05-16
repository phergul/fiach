import { Dialogs } from '@wailsio/runtime';

interface OpenZipArchiveOptions {
  buttonText: string;
  title: string;
}

export const openZipArchive = async ({
  buttonText,
  title,
}: OpenZipArchiveOptions) => {
  const selectedPath = await Dialogs.OpenFile({
    AllowsMultipleSelection: false,
    ButtonText: buttonText,
    CanChooseDirectories: false,
    CanChooseFiles: true,
    Filters: [
      {
        DisplayName: 'ZIP Archives',
        Pattern: '*.zip',
      },
    ],
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
