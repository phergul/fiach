import { useState } from 'react';

import type { StrategyType } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { ImportMod, PreValidateImport } from '@bindings/github.com/phergul/fiach/internal/services/modservice';
import { ModSourceType } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { ResolveGameModStoragePath } from '@bindings/github.com/phergul/fiach/internal/services/settingsservice';
import { useToast } from '@components/Common/Toast/Toast';
import { getErrorMessage, openDirectory, openZipArchive } from '@utils';

interface ImportReviewState {
  initialName: string;
  sourceLabel: string;
  sourcePath: string;
  sourceType: ModSourceType;
  targetPath: string;
}

interface ImportWizardSubmitInput {
  name: string;
  strategyType: StrategyType;
  targetRelativePath: string;
}

interface UseGameModImportFlowInput {
  gameID: number | null;
  refreshMods: () => Promise<unknown>;
}

const getFolderName = (path: string) => {
  const trimmedPath = path.trim().replace(/[\\/]+$/, '');
  const folderName = trimmedPath.split(/[\\/]/).pop();

  return folderName && folderName.trim() !== '' ? folderName : 'Imported Mod';
};

const getArchiveName = (path: string) => {
  const fileName = getFolderName(path);
  const archiveName = fileName.replace(/\.zip$/i, '');

  return archiveName.trim() === '' ? 'Imported Mod' : archiveName;
};

export const useGameModImportFlow = ({
  gameID,
  refreshMods,
}: UseGameModImportFlowInput) => {
  const { addErrorToast, addToast } = useToast();
  const [importWizard, setImportWizard] = useState<ImportReviewState | null>(null);
  const [importError, setImportError] = useState<string | null>(null);
  const [isImporting, setIsImporting] = useState(false);
  const [isImportMenuOpen, setIsImportMenuOpen] = useState(false);

  const startImportFlow = async ({
    buttonText,
    initialNameForPath,
    sourceLabel,
    sourceType,
    title,
    selectPath,
  }: {
    buttonText: string;
    initialNameForPath: (path: string) => string;
    selectPath: (options: { buttonText: string; title: string }) => Promise<string | null>;
    sourceLabel: string;
    sourceType: ModSourceType;
    title: string;
  }) => {
    if (gameID === null || isImporting) {
      return;
    }

    setIsImportMenuOpen(false);
    setImportError(null);

    try {
      const sourcePath = await selectPath({
        buttonText,
        title,
      });
      if (sourcePath === null) {
        return;
      }

      await PreValidateImport({
        SourceType: sourceType,
        SourcePath: sourcePath,
      });

      const targetPath = await ResolveGameModStoragePath(gameID);
      setImportWizard({
        initialName: initialNameForPath(sourcePath),
        sourceLabel,
        sourcePath,
        sourceType,
        targetPath,
      });
    } catch (error) {
      addErrorToast(error);
    }
  };

  const startFolderImportFlow = () => startImportFlow({
    buttonText: 'Configure Import',
    initialNameForPath: getFolderName,
    selectPath: openDirectory,
    sourceLabel: 'Source folder',
    sourceType: ModSourceType.ModSourceTypeFolder,
    title: 'Select mod folder',
  });

  const startArchiveImportFlow = () => startImportFlow({
    buttonText: 'Configure Import',
    initialNameForPath: getArchiveName,
    selectPath: openZipArchive,
    sourceLabel: 'Source archive',
    sourceType: ModSourceType.ModSourceTypeArchive,
    title: 'Select mod archive',
  });

  const closeImportReview = () => {
    if (isImporting) {
      return;
    }

    setImportWizard(null);
    setImportError(null);
  };

  const importWizardMod = async ({
    name,
    strategyType,
    targetRelativePath,
  }: ImportWizardSubmitInput) => {
    if (gameID === null || importWizard === null || isImporting) {
      return;
    }

    setIsImporting(true);
    setImportError(null);

    try {
      const importResult = await ImportMod({
        GameID: gameID,
        Name: name,
        SourceType: importWizard.sourceType,
        SourcePath: importWizard.sourcePath,
        StrategyType: strategyType,
        TargetRelativePath: targetRelativePath,
      });
      setImportWizard(null);

      try {
        await refreshMods();
      } catch (refreshError) {
        addErrorToast(refreshError);
      }

      addToast({
        message: `Imported ${importResult.Mod.Name}.`,
        tone: 'success',
      });
    } catch (error) {
      const message = getErrorMessage(error);
      setImportError(message);
      addErrorToast(error);
    } finally {
      setIsImporting(false);
    }
  };

  return {
    closeImportReview,
    importError,
    importWizard,
    importWizardMod,
    isImporting,
    isImportMenuOpen,
    setIsImportMenuOpen,
    startArchiveImportFlow,
    startFolderImportFlow,
  };
};

export type UseGameModImportFlowResult = ReturnType<typeof useGameModImportFlow>;
