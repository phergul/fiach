import { useState } from 'react';

import {
  DownloadAndOpenReShadeAddonInstaller,
  DownloadAndOpenReShadeInstaller,
} from '@bindings/github.com/phergul/fiach/internal/services/reshadeservice';
import {
  ReShadeDetectionStatus,
  type StoredGame,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { useToast } from '@components/Common/Toast/Toast';

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
  const { addErrorToast, addToast } = useToast();
  const [isLaunchingInstaller, setIsLaunchingInstaller] = useState(false);
  const reShadeStatus = reShadeDetection.result?.Status;
  const reShadeInstallerActionLabels = (() => {
    if (game === undefined) {
      return {
        addon: null,
        standard: null,
      };
    }

    switch (reShadeStatus) {
      case ReShadeDetectionStatus.ReShadeDetectionStatusInstalled:
        return {
          addon: 'Update/Reinstall ReShade with Add-on Support',
          standard: 'Update/Reinstall ReShade',
        };
      case ReShadeDetectionStatus.ReShadeDetectionStatusNotInstalled:
        return {
          addon: 'Install ReShade with Add-on Support',
          standard: 'Install ReShade',
        };
      default:
        return {
          addon: null,
          standard: null,
        };
    }
  })();
  const canLaunchInstaller = reShadeInstallerActionLabels.standard !== null;

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
      addErrorToast(error);
    } finally {
      setIsLaunchingInstaller(false);
    }
  };

  const downloadAndOpenAddonInstaller = async () => {
    if (game === undefined || !canLaunchInstaller || isLaunchingInstaller) {
      return;
    }

    onMenuClose();
    setIsLaunchingInstaller(true);

    try {
      const result = await DownloadAndOpenReShadeAddonInstaller();
      addToast({
        message: `ReShade ${result.Version} add-on installer opened.`,
        tone: 'success',
      });
    } catch (error) {
      addErrorToast(error);
    } finally {
      setIsLaunchingInstaller(false);
    }
  };

  return {
    downloadAndOpenAddonInstaller,
    downloadAndOpenInstaller,
    isLaunchingInstaller,
    reShadeAddonInstallerActionLabel: reShadeInstallerActionLabels.addon,
    reShadeInstallerActionLabel: reShadeInstallerActionLabels.standard,
  };
};

export type UseGameReShadeInstallResult = ReturnType<typeof useGameReShadeInstall>;
