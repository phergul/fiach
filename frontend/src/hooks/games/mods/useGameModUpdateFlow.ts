import { useState } from 'react';

import {
  ModSourceType,
  type Mod,
  type UpdateModResult,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import {
  PreviewUpdateMod,
  UpdateMod,
} from '@bindings/github.com/phergul/fiach/internal/services/modservice';
import { useToast } from '@components/Common/Toast/Toast';
import { getErrorMessage, openArchive, openDirectory } from '@utils';

interface UpdateReviewState {
  mod: Mod;
  preview: UpdateModResult;
  sourcePath: string;
  sourceType: ModSourceType;
}

interface UseGameModUpdateFlowInput {
  refreshAfterUpdate: () => Promise<unknown>;
}

export const useGameModUpdateFlow = ({ refreshAfterUpdate }: UseGameModUpdateFlowInput) => {
  const { addErrorToast, addToast } = useToast();
  const [updateReview, setUpdateReview] = useState<UpdateReviewState | null>(null);
  const [updateError, setUpdateError] = useState<string | null>(null);
  const [isPreviewingUpdate, setIsPreviewingUpdate] = useState(false);
  const [isUpdatingMod, setIsUpdatingMod] = useState(false);

  const startUpdateFlow = async ({
    buttonText,
    mod,
    selectPath,
    sourceType,
    title,
  }: {
    buttonText: string;
    mod: Mod;
    selectPath: (options: { buttonText: string; title: string }) => Promise<string | null>;
    sourceType: ModSourceType;
    title: string;
  }) => {
    if (isPreviewingUpdate || isUpdatingMod) {
      return;
    }

    setUpdateError(null);

    try {
      const sourcePath = await selectPath({
        buttonText,
        title,
      });
      if (sourcePath === null) {
        return;
      }

      setIsPreviewingUpdate(true);
      const preview = await PreviewUpdateMod({
        ModID: mod.ID,
        SourcePath: sourcePath,
        SourceType: sourceType,
      });
      setUpdateReview({
        mod,
        preview,
        sourcePath,
        sourceType,
      });
    } catch (error) {
      addErrorToast(error);
    } finally {
      setIsPreviewingUpdate(false);
    }
  };

  const startFolderUpdateFlow = (mod: Mod) =>
    startUpdateFlow({
      buttonText: 'Review Update',
      mod,
      selectPath: openDirectory,
      sourceType: ModSourceType.ModSourceTypeFolder,
      title: `Select replacement folder for ${mod.Name}`,
    });

  const startArchiveUpdateFlow = (mod: Mod) =>
    startUpdateFlow({
      buttonText: 'Review Update',
      mod,
      selectPath: openArchive,
      sourceType: ModSourceType.ModSourceTypeArchive,
      title: `Select replacement archive for ${mod.Name}`,
    });

  const closeUpdateReview = () => {
    if (isUpdatingMod) {
      return;
    }

    setUpdateReview(null);
    setUpdateError(null);
  };

  const confirmUpdateMod = async () => {
    if (updateReview === null || isUpdatingMod) {
      return;
    }

    setIsUpdatingMod(true);
    setUpdateError(null);

    try {
      const result = await UpdateMod({
        ModID: updateReview.mod.ID,
        SourcePath: updateReview.sourcePath,
        SourceType: updateReview.sourceType,
      });
      setUpdateReview(null);

      try {
        await refreshAfterUpdate();
      } catch (refreshError) {
        addErrorToast(refreshError);
      }

      addToast({
        message: result.RequiresReapply
          ? `Updated ${result.Mod.Name}. Reapply the current profile when ready.`
          : `Updated ${result.Mod.Name}.`,
        tone: 'success',
      });
      result.Warnings.forEach((warning) => {
        addToast({
          message: warning,
          tone: 'info',
        });
      });
    } catch (error) {
      const message = getErrorMessage(error);
      setUpdateError(message);
      addErrorToast(error);
    } finally {
      setIsUpdatingMod(false);
    }
  };

  return {
    closeUpdateReview,
    confirmUpdateMod,
    isBusy: isPreviewingUpdate || isUpdatingMod,
    isPreviewingUpdate,
    isUpdatingMod,
    startArchiveUpdateFlow,
    startFolderUpdateFlow,
    updateError,
    updateReview,
  };
};

export type UseGameModUpdateFlowResult = ReturnType<typeof useGameModUpdateFlow>;
