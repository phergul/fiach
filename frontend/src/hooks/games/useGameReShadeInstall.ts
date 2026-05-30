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
  const reShadeStatus = reShadeDetection.result?.Status;
  const reShadeInstallerActionLabel = (() => {
    if (game === undefined) {
      return null;
    }

    switch (reShadeStatus) {
      case ReShadeDetectionStatus.ReShadeDetectionStatusInstalled:
        return 'Update/Reinstall ReShade';
      case ReShadeDetectionStatus.ReShadeDetectionStatusNotInstalled:
        return 'Install ReShade';
      default:
        return null;
    }
  })();
  const canLaunchInstaller = reShadeInstallerActionLabel !== null;

  const downloadAndOpenInstaller = async () => {
    if (game === undefined || !canLaunchInstaller || isLaunchingInstaller) {
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
    downloadAndOpenInstaller,
    isLaunchingInstaller,
    reShadeInstallerActionLabel,
  };
};

export type UseGameReShadeInstallResult = ReturnType<typeof useGameReShadeInstall>;
