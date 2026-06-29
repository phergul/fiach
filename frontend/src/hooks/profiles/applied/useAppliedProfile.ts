import { useCallback, useState } from 'react';

import { RestoreVanillaState } from '@bindings/github.com/phergul/fiach/internal/services/profileservice';
import type { RestoreResult } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { useToast } from '@components/Common/Toast/Toast';

import {
  appliedProfileResource,
  fetchAppliedProfile,
  invalidateAppliedProfile,
  preloadAppliedProfile,
} from './appliedProfileResource';

export {
  appliedProfileResource,
  fetchAppliedProfile,
  invalidateAppliedProfile,
  preloadAppliedProfile,
};

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
  const {
    data: appliedProfile,
    isInitialLoading,
    isLoading,
    isRefreshing,
    loadError,
    refresh,
  } = appliedProfileResource.useCached(gameID);
  const [pendingAction, setPendingAction] = useState<AppliedProfileAction | null>(null);

  const restoreVanilla = useCallback(async () => {
    if (gameID === null) {
      return Promise.reject(new Error('game is not selected'));
    }

    setPendingAction('restore');

    try {
      const result = await RestoreVanillaState(gameID);
      if (result.Success) {
        await refresh();
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
  }, [addErrorToast, addToast, gameID, refresh]);

  return {
    appliedProfile,
    isInitialLoading,
    isLoading,
    isRefreshing,
    loadError,
    pendingAction,
    refreshAppliedProfile: refresh,
    restoreVanilla,
  };
};

export type UseAppliedProfileResult = ReturnType<typeof useAppliedProfile>;
