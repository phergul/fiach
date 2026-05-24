import { useState } from 'react';

import { SetGameModStoragePathOverride } from '@bindings/github.com/phergul/mod-manager/internal/services/settingsservice';
import type { StoredGame } from '@bindings/github.com/phergul/mod-manager/internal/services/dto/models';
import { useToast } from '@components/Common/Toast/Toast';
import { getErrorMessage, openDirectory } from '@utils';

interface PendingStorageOverride {
  confirmLabel: string;
  path: string;
  successMessage: string;
  title: string;
}

interface UseGameStorageOverrideInput {
  game: StoredGame | undefined;
  onMenuClose: () => void;
  updateStoredGame: (game: StoredGame) => void;
}

export const useGameStorageOverride = ({
  game,
  onMenuClose,
  updateStoredGame,
}: UseGameStorageOverrideInput) => {
  const { addToast } = useToast();
  const [pendingStorageOverride, setPendingStorageOverride] = useState<PendingStorageOverride | null>(null);
  const [isApplyingStorageOverride, setIsApplyingStorageOverride] = useState(false);

  const requestSetStorageOverride = async () => {
    if (game === undefined || isApplyingStorageOverride) {
      return;
    }

    onMenuClose();

    try {
      const path = await openDirectory({
        buttonText: 'Use Folder',
        canCreateDirectories: true,
        title: 'Select mod storage override',
      });
      if (path === null) {
        return;
      }

      setPendingStorageOverride({
        confirmLabel: 'Set Override',
        path,
        successMessage: 'Mod storage override set.',
        title: 'Set Storage Override?',
      });
    } catch (error) {
      const message = getErrorMessage(error);
      addToast({
        message,
        tone: 'error',
      });
    }
  };

  const requestClearStorageOverride = () => {
    if (game === undefined || isApplyingStorageOverride) {
      return;
    }

    onMenuClose();
    setPendingStorageOverride({
      confirmLabel: 'Clear Override',
      path: '',
      successMessage: 'Mod storage override cleared.',
      title: 'Clear Storage Override?',
    });
  };

  const cancelStorageOverride = () => {
    if (!isApplyingStorageOverride) {
      setPendingStorageOverride(null);
    }
  };

  const applyStorageOverride = async () => {
    if (game === undefined || pendingStorageOverride === null || isApplyingStorageOverride) {
      return;
    }

    setIsApplyingStorageOverride(true);

    try {
      const updatedGame: StoredGame = await SetGameModStoragePathOverride(game.ID, pendingStorageOverride.path);
      updateStoredGame(updatedGame);
      addToast({
        message: pendingStorageOverride.successMessage,
        tone: 'success',
      });
      setPendingStorageOverride(null);
    } catch (error) {
      const message = getErrorMessage(error);
      addToast({
        message,
        tone: 'error',
      });
    } finally {
      setIsApplyingStorageOverride(false);
    }
  };

  return {
    applyStorageOverride,
    cancelStorageOverride,
    isApplyingStorageOverride,
    pendingStorageOverride,
    requestClearStorageOverride,
    requestSetStorageOverride,
  };
};

export type UseGameStorageOverrideResult = ReturnType<typeof useGameStorageOverride>;
