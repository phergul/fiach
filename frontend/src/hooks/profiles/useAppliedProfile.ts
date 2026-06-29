import { useCallback, useEffect, useState } from 'react';

import {
  GetAppliedProfileSummary,
  RestoreVanillaState,
} from '@bindings/github.com/phergul/fiach/internal/services/profileservice';
import type { AppliedProfileSummary } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import type { RestoreResult } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { useToast } from '@components/Common/Toast/Toast';
import { getErrorMessage } from '@utils';

type AppliedProfileAction = 'restore';

const buildRestoreSuccessMessage = (result: RestoreResult) => {
  if (result.CompletedCount === 0) {
    return 'No restore operations were needed.';
  }
  if (result.CompletedCount === 1) {
    return 'Restored 1 operation.';
  }

  return `Restored ${result.CompletedCount} operations.`;
};

const buildRestoreFailureMessage = (result: RestoreResult) => {
  const failedResult = result.Results.find((operationResult) => operationResult.Error !== null);
  const failure = failedResult?.Error ?? 'Restore stopped before all operations completed.';

  return `Restore stopped: ${failure} Completed ${result.CompletedCount}, skipped ${result.SkippedCount}.`;
};

export const useAppliedProfile = (gameID: number | null) => {
  const { addErrorToast, addToast } = useToast();
  const [appliedProfile, setAppliedProfile] = useState<AppliedProfileSummary | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [pendingAction, setPendingAction] = useState<AppliedProfileAction | null>(null);

  const loadAppliedProfile = useCallback(
    async (isCurrent: () => boolean = () => true) => {
      if (gameID === null) {
        if (isCurrent()) {
          setAppliedProfile(null);
          setIsLoading(false);
          setLoadError(null);
        }
        return null;
      }

      if (isCurrent()) {
        setIsLoading(true);
        setLoadError(null);
      }

      try {
        const loadedAppliedProfile = await GetAppliedProfileSummary(gameID);
        if (isCurrent()) {
          setAppliedProfile(loadedAppliedProfile);
        }
        return loadedAppliedProfile;
      } catch (error) {
        const message = getErrorMessage(error);
        if (isCurrent()) {
          setLoadError(message);
        }
        throw error;
      } finally {
        if (isCurrent()) {
          setIsLoading(false);
        }
      }
    },
    [gameID],
  );

  const refreshAppliedProfile = useCallback(() => loadAppliedProfile(), [loadAppliedProfile]);

  const restoreVanilla = useCallback(async () => {
    if (gameID === null) {
      return Promise.reject(new Error('game is not selected'));
    }

    setPendingAction('restore');

    try {
      const result = await RestoreVanillaState(gameID);
      if (result.Success) {
        await loadAppliedProfile();
      }
      addToast({
        message: result.Success
          ? buildRestoreSuccessMessage(result)
          : buildRestoreFailureMessage(result),
        tone: result.Success ? 'success' : 'error',
      });
      return result;
    } catch (error) {
      addErrorToast(error);
      throw error;
    } finally {
      setPendingAction(null);
    }
  }, [addErrorToast, addToast, gameID, loadAppliedProfile]);

  useEffect(() => {
    let isMounted = true;

    loadAppliedProfile(() => isMounted).catch(() => {
      // Load errors are stored in hook state for the caller to render.
    });

    return () => {
      isMounted = false;
    };
  }, [loadAppliedProfile]);

  return {
    appliedProfile,
    isLoading,
    loadError,
    pendingAction,
    refreshAppliedProfile,
    restoreVanilla,
  };
};

export type UseAppliedProfileResult = ReturnType<typeof useAppliedProfile>;
