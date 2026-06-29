import { useEffect } from 'react';

import { Events } from '@wailsio/runtime';

const filesDroppedEventName = 'files-dropped';

interface FilesDroppedEventData {
  files?: string[];
}

interface UseModImportFileDropInput {
  enabled: boolean;
  onFilesDropped: (files: string[]) => void;
}

export const useModImportFileDrop = ({ enabled, onFilesDropped }: UseModImportFileDropInput) => {
  useEffect(() => {
    if (!enabled) {
      return;
    }

    return Events.On(filesDroppedEventName, (event) => {
      const data = event.data as FilesDroppedEventData | string[] | null;
      const files = Array.isArray(data) ? data : Array.isArray(data?.files) ? data.files : [];

      const normalizedFiles = files
        .filter((file): file is string => typeof file === 'string')
        .map((file) => file.trim())
        .filter((file) => file !== '');

      if (normalizedFiles.length > 0) {
        onFilesDropped(normalizedFiles);
      }
    });
  }, [enabled, onFilesDropped]);
};
