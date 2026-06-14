import { useState } from 'react';

import {
  DownloadAndOpenReShadeAddonInstaller,
  DownloadAndOpenReShadeInstaller,
  PreflightReShadeInstaller,
} from '@bindings/github.com/phergul/fiach/internal/services/reshadeservice';
import {
  type ReShadeInstallerPreflight,
  ReShadeDetectionStatus,
  type StoredGame,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { ReShadeInstallerVariant } from '@bindings/github.com/phergul/fiach/internal/optiscaler/models';
import { useToast } from '@components/Common/Toast/Toast';

import type { UseGameReShadeDetectionResult } from './useGameReShadeDetection';

interface UseGameReShadeInstallInput {
  game: StoredGame | undefined;
  onCoordinate: (preflight: ReShadeInstallerPreflight) => void;
  onMenuClose: () => void;
  reShadeDetection: UseGameReShadeDetectionResult;
}

export const useGameReShadeInstall = ({
  game,
  onCoordinate,
  onMenuClose,
  reShadeDetection,
}: UseGameReShadeInstallInput) => {
  const { addErrorToast, addToast } = useToast();
  const [isLaunchingInstaller, setIsLaunchingInstaller] = useState(false);
  const [isCompletionPromptOpen, setIsCompletionPromptOpen] = useState(false);
  const [isRefreshingDetection, setIsRefreshingDetection] = useState(false);
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

  const launchInstaller = async (variant: ReShadeInstallerVariant) => {
    if (game === undefined || !canLaunchInstaller || isLaunchingInstaller) {
      return;
    }

    onMenuClose();
    setIsLaunchingInstaller(true);

    try {
      const preflight = await PreflightReShadeInstaller(game.ID, variant);
      if (preflight.Disposition === 'blocked') {
        addToast({ message: preflight.Message, tone: 'error' });
        return;
      }
      if (preflight.Disposition === 'coordinated') {
        onCoordinate(preflight);
        return;
      }
      const result = variant === ReShadeInstallerVariant.ReShadeInstallerVariantAddon
        ? await DownloadAndOpenReShadeAddonInstaller()
        : await DownloadAndOpenReShadeInstaller();
      addToast({
        message: `ReShade ${result.Version}${variant === ReShadeInstallerVariant.ReShadeInstallerVariantAddon
          ? ' add-on'
          : ''} installer opened.`,
        tone: 'success',
      });
      setIsCompletionPromptOpen(true);
    } catch (error) {
      addErrorToast(error);
    } finally {
      setIsLaunchingInstaller(false);
    }
  };

  const downloadAndOpenInstaller = () =>
    launchInstaller(ReShadeInstallerVariant.ReShadeInstallerVariantStandard);
  const downloadAndOpenAddonInstaller = () =>
    launchInstaller(ReShadeInstallerVariant.ReShadeInstallerVariantAddon);

  const cancelCompletionPrompt = () => {
    if (!isRefreshingDetection) {
      setIsCompletionPromptOpen(false);
    }
  };

  const confirmInstallerFinished = async () => {
    if (isRefreshingDetection) {
      return;
    }
    setIsRefreshingDetection(true);
    try {
      await reShadeDetection.refresh();
      setIsCompletionPromptOpen(false);
      addToast({
        message: 'ReShade status refreshed.',
        tone: 'success',
      });
    } catch (error) {
      addErrorToast(error);
    } finally {
      setIsRefreshingDetection(false);
    }
  };

  return {
    cancelCompletionPrompt,
    confirmInstallerFinished,
    downloadAndOpenAddonInstaller,
    downloadAndOpenInstaller,
    isCompletionPromptOpen,
    isLaunchingInstaller,
    isRefreshingDetection,
    reShadeAddonInstallerActionLabel: reShadeInstallerActionLabels.addon,
    reShadeInstallerActionLabel: reShadeInstallerActionLabels.standard,
  };
};

export type UseGameReShadeInstallResult = ReturnType<typeof useGameReShadeInstall>;
