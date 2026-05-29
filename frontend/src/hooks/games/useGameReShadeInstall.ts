import { useState } from 'react';

import { DownloadAndOpenReShadeInstaller } from '@bindings/github.com/phergul/mod-manager/internal/services/reshadeservice';
import {
  ReShadeDetectionStatus,
  type StoredGame,
} from '@bindings/github.com/phergul/mod-manager/internal/services/dto/models';
import { useToast } from '@components/Common/Toast/Toast';
import { getErrorMessage } from '@utils';

import type { UseGameReShadeDetectionResult } from './useGameReShadeDetection';

interface UseGameReShadeInstallInput {
  game: StoredGame | undefined;
  onMenuClose: () => void;
  reShadeDetection: UseGameReShadeDetectionResult;
}

export const useGameReShadeInstall = ({
  game,
  onMenuClose,
  reShadeDetection,
}: UseGameReShadeInstallInput) => {
  const { addToast } = useToast();
  const [isLaunchingInstaller, setIsLaunchingInstaller] = useState(false);
  const canInstallReShade = game !== undefined
    && reShadeDetection.result?.Status === ReShadeDetectionStatus.ReShadeDetectionStatusNotInstalled;

  const downloadAndOpenInstaller = async () => {
    if (game === undefined || !canInstallReShade || isLaunchingInstaller) {
      return;
    }

    onMenuClose();
    setIsLaunchingInstaller(true);

    try {
      const result = await DownloadAndOpenReShadeInstaller();
      addToast({
        message: `ReShade ${result.Version} installer opened.`,
        tone: 'success',
      });
    } catch (error) {
      addToast({
        message: getErrorMessage(error),
        tone: 'error',
      });
    } finally {
      setIsLaunchingInstaller(false);
    }
  };

  return {
    canInstallReShade,
    downloadAndOpenInstaller,
    isLaunchingInstaller,
  };
};

export type UseGameReShadeInstallResult = ReturnType<typeof useGameReShadeInstall>;
